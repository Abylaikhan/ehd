// Package app — composition root ЕХД: wiring платформы и модулей,
// AutoMigrate, запуск HTTP-сервера и graceful shutdown.
package app

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"ehd-api/config"
	authapp "ehd-api/internal/modules/auth/application"
	"ehd-api/internal/modules/auth/eds"
	authrepo "ehd-api/internal/modules/auth/repository"
	authhttp "ehd-api/internal/modules/auth/transport/http"
	evgaapp "ehd-api/internal/modules/evga/application"
	evgarepo "ehd-api/internal/modules/evga/repository"
	evgahttp "ehd-api/internal/modules/evga/transport/http"
	reporterapp "ehd-api/internal/modules/reporter/application"
	reporterch "ehd-api/internal/modules/reporter/chsource"
	reporterrepo "ehd-api/internal/modules/reporter/repository"
	reporterhttp "ehd-api/internal/modules/reporter/transport/http"
	"ehd-api/pkg/clickhouse"
	"ehd-api/pkg/crypto"
	"ehd-api/pkg/httpserver"
	"ehd-api/pkg/logger"
	"ehd-api/pkg/postgres"
)

// evgaReadOnlyDSN добавляет к DSN obm_evga сессионный параметр
// default_transaction_read_only=on (pgx передаёт его как runtime-параметр стартапа) —
// модуль ЕВГА работает с внешней БД строго в режиме чтения (спека 007 FR-2).
func evgaReadOnlyDSN(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil || u.Scheme == "" {
		return "", fmt.Errorf("EVGA_PG_DSN: ожидается URL postgres://…: %w", err)
	}
	q := u.Query()
	q.Set("default_transaction_read_only", "on")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func Run(cfg *config.Config) error {
	log, err := logger.New(cfg.Log.Level)
	if err != nil {
		return err
	}
	defer func() { _ = log.Sync() }()

	// --- PostgreSQL + AutoMigrate ---
	db, err := postgres.New(cfg.PG.DSN)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if err := authrepo.Migrate(db); err != nil {
		return err
	}
	if err := reporterrepo.Migrate(db); err != nil {
		return err
	}
	log.Info("automigrate complete")

	if err := authrepo.SeedReference(db); err != nil {
		return err
	}
	if cfg.Admin.Password != "" {
		if err := authrepo.SeedAdmin(db, cfg.Admin.Login, cfg.Admin.Email, cfg.Admin.Password); err != nil {
			return err
		}
		log.Info("admin seeded", zap.String("login", cfg.Admin.Login))
	}

	// --- Auth Module (crypto + repos + service) ---
	cipher, err := crypto.New(cfg.Auth.EncKey, cfg.Auth.HMACKey)
	if err != nil {
		return err
	}

	// Выбор верификатора ЭЦП: stub (sandbox) | ncalayer (ddulesov, RSA) | ncanode (Kalkan-сайдкар, RSA+ГОСТ РК).
	var edsVerifier eds.Verifier
	switch cfg.EDS.Mode {
	case config.EDSModeNCANode:
		edsVerifier = eds.NewNCANodeVerifier(cfg.EDS.NCANodeURL, log)
		log.Info("eds mode: ncanode", zap.String("url", cfg.EDS.NCANodeURL))
	case config.EDSModeNCALayer:
		v, err := eds.NewNCALayerVerifier(cfg.EDS.TrustDir, cfg.EDS.OCSPEnabled, log)
		if err != nil {
			return err
		}
		edsVerifier = v
		log.Info("eds mode: ncalayer", zap.String("trust_dir", cfg.EDS.TrustDir), zap.Bool("ocsp", cfg.EDS.OCSPEnabled))
	default:
		edsVerifier = eds.StubVerifier{}
		log.Info("eds mode: stub (sandbox)")
	}

	authService := authapp.NewService(
		authrepo.NewUserRepo(db),
		authrepo.NewRoleRepo(db),
		authrepo.NewSessionRepo(db),
		authrepo.NewReferenceRepo(db),
		cipher,
		edsVerifier,
		authapp.Settings{
			SessionTTL:        cfg.Auth.SessionTTL,
			TempPasswordTTL:   cfg.Auth.TempPasswordTTL,
			MaxFailedAttempts: cfg.Auth.MaxFailedAttempts,
		},
	)
	authHandler := authhttp.NewHandler(authService, cfg.Auth.CookieSecure)

	// --- Reporter Module (источник ClickHouse + интроспекция) ---
	reporterService := reporterapp.NewService(
		reporterrepo.NewDataSourceRepo(db),
		cipher,
		reporterch.New(),
		reporterapp.Config{SystemDBDenylist: cfg.Reporter.SystemDBDenylist},
		log,
	)
	dataViewRepo := reporterrepo.NewDataViewRepo(db)
	reporterViewService := reporterapp.NewViewService(
		dataViewRepo,
		reporterService, // SourceInspector: проверка активного источника + интроспекция колонок
		log,
	)
	reporterQueryService := reporterapp.NewQueryService(dataViewRepo, reporterService, log)
	reporterMenuService := reporterapp.NewMenuService(reporterrepo.NewMenuRepo(db))
	reporterHandler := reporterhttp.NewHandler(reporterService, reporterViewService, reporterQueryService, reporterMenuService)
	reporterGuard := reporterhttp.NewGuard(authService) // RBAC через auth/contract (без сети)

	// --- ClickHouse (read-only проверка готовности источника по умолчанию) ---
	ch, err := clickhouse.New(cfg.CH)
	if err != nil {
		return err
	}
	defer ch.Close()

	// --- Модуль ОБМ ЕВГА (внешняя PostgreSQL obm_evga; спека 007, режим записи — спека 008 FR-1/2) ---
	var evgaService *evgaapp.Service
	var evgaStatusService *evgaapp.StatusService
	var evgaNoticeService *evgaapp.NoticeService
	var evgaApprovalService *evgaapp.ApprovalService
	var evgaOutgoingService *evgaapp.OutgoingService
	if cfg.EVGA.Enabled() {
		evgaDSN := cfg.EVGA.DSN
		if !cfg.EVGA.WriteEnabled {
			// без явного разрешения записи сессии внешней БД принудительно read-only
			if evgaDSN, err = evgaReadOnlyDSN(evgaDSN); err != nil {
				return err
			}
		}
		evgaDB, err := postgres.New(evgaDSN)
		if err != nil {
			return err
		}
		evgaSQL, err := evgaDB.DB()
		if err != nil {
			return err
		}
		// внешняя прод-БД: скромный пул
		evgaSQL.SetMaxOpenConns(5)
		evgaSQL.SetMaxIdleConns(2)
		defer evgaSQL.Close()

		evgaRepo := evgarepo.NewRegistryRepo(evgaDB)
		if mode, err := evgaRepo.TransactionReadOnly(context.Background()); err != nil {
			log.Warn("evga: не удалось проверить режим read-only", zap.Error(err))
		} else {
			log.Info("evga: подключение к obm_evga установлено",
				zap.String("default_transaction_read_only", mode),
				zap.Bool("write_enabled", cfg.EVGA.WriteEnabled))
		}
		evgaService = evgaapp.NewService(evgaRepo, authService, log)
		evgaStatusRepo := evgarepo.NewStatusRepo(evgaDB)
		evgaStatusService = evgaapp.NewStatusService(evgaStatusRepo, evgaService, cfg.EVGA.WriteEnabled, log)
		evgaNoticeService = evgaapp.NewNoticeService(
			evgarepo.NewNoticeRepo(evgaDB), evgaStatusRepo, evgaService, cfg.EVGA.WriteEnabled, log)

		// маршрут согласования живёт в БД ЕХД (спека 010 ADR), не в obm_evga
		if err := evgarepo.MigrateRoute(db); err != nil {
			return err
		}
		evgaApprovalRepo := evgarepo.NewApprovalRepo(evgaDB)
		evgaRouteRepo := evgarepo.NewRouteRepo(db)
		evgaApprovalService = evgaapp.NewApprovalService(
			evgaApprovalRepo, evgaRouteRepo,
			evgaStatusRepo, evgaService, cfg.EVGA.WriteEnabled, log)
		evgaOutgoingService = evgaapp.NewOutgoingService(
			evgarepo.NewOutgoingRepo(evgaDB), evgaApprovalRepo, evgaRouteRepo,
			evgaStatusRepo, evgaService, cfg.EVGA.WriteEnabled, log)
	} else {
		log.Info("evga: модуль выключен (EVGA_PG_DSN не задан)")
	}

	// --- HTTP (fiber) ---
	app := httpserver.New(cfg, log)

	// liveness — без внешних зависимостей; readiness — с проверкой критических подключений
	app.Get("/livez", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/readyz", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
		defer cancel()

		checks := fiber.Map{"postgres": "ok", "clickhouse": "ok"}
		status := fiber.StatusOK
		if err := sqlDB.PingContext(ctx); err != nil {
			checks["postgres"] = "unavailable"
			status = fiber.StatusServiceUnavailable
		}
		if err := ch.Ping(ctx); err != nil {
			checks["clickhouse"] = "unavailable"
			status = fiber.StatusServiceUnavailable
		}
		if evgaService != nil {
			checks["evga"] = "ok"
			if err := evgaService.Ping(ctx); err != nil {
				checks["evga"] = "unavailable"
				status = fiber.StatusServiceUnavailable
			}
		}
		return c.Status(status).JSON(checks)
	})

	api := app.Group("/api/v1")
	authhttp.Register(api.Group("/auth"), authHandler)
	reporterhttp.Register(api.Group("/reporter"), reporterHandler, reporterGuard)
	if evgaService != nil {
		evgahttp.Register(api.Group("/evga"),
			evgahttp.NewHandler(evgaService, evgaStatusService, evgaNoticeService, evgaApprovalService, evgaOutgoingService),
			evgahttp.NewGuard(authService))
	}

	// вотчер регистрации исходящих (спека 011 FR-5): платформа проставляет doc_num —
	// мы выполняем транзакцию §11.3; останавливается вместе с сервером
	watchCtx, watchCancel := context.WithCancel(context.Background())
	defer watchCancel()
	if evgaOutgoingService != nil {
		go evgaOutgoingService.Watch(watchCtx, cfg.EVGA.WatchInterval)
	}

	// --- запуск + graceful shutdown ---
	errCh := make(chan error, 1)
	go func() {
		log.Info("server started", zap.String("port", cfg.HTTP.Port), zap.String("env", cfg.App.Env))
		errCh <- app.Listen(":" + cfg.HTTP.Port)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-stop:
		log.Info("shutting down", zap.String("signal", sig.String()))
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		// ShutdownWithContext закрывает listener и ждёт завершения активных соединений
		if err := app.ShutdownWithContext(ctx); err != nil {
			return err
		}
		log.Info("shutdown complete")
	}

	return nil
}
