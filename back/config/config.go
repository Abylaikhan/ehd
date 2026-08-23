package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// splitList разбивает список, заданный через запятую, в срез непустых значений.
func splitList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

type Config struct {
	App      App
	HTTP     HTTP
	Log      Log
	PG       PG
	CH       CH
	Admin    Admin
	Auth     Auth
	EDS      EDS
	Reporter Reporter
}

type App struct {
	Name string
	Env  string
}

type HTTP struct {
	Port string
}

type Log struct {
	Level string
}

type PG struct {
	DSN string
}

// CH — read-only источник ClickHouse для Reporter Module.
type CH struct {
	Addr     string
	Database string
	User     string
	Password string
}

// Admin — данные seed-администратора (создаётся при старте, если задан пароль).
type Admin struct {
	Login    string
	Password string
	Email    string
}

// Auth — параметры Auth Module. Ключи шифрования/HMAC хранятся вне БД (env).
type Auth struct {
	EncKey            string // hex AES-256 (64 hex-символа = 32 байта)
	HMACKey           string
	SessionTTL        time.Duration
	TempPasswordTTL   time.Duration
	MaxFailedAttempts int
	CookieSecure      bool
}

// devEncKey — фиксированный dev-ключ (только для local); в non-local обязателен AUTH_ENC_KEY.
const devEncKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

// EDS — параметры проверки ЭЦП (NCALayer).
// Mode: "stub" (sandbox, без NCALayer) или "ncalayer" (реальная проверка CMS).
type EDS struct {
	Mode        string
	TrustDir    string // каталог с корневыми/промежуточными сертификатами НУЦ РК (PEM/DER)
	OCSPEnabled bool
	NCANodeURL  string // URL сайдкара NCANode (для режима ncanode)
}

const (
	EDSModeStub     = "stub"
	EDSModeNCALayer = "ncalayer" // ddulesov/pkcs7: RSA (ГОСТ РК не поддерживается)
	EDSModeNCANode  = "ncanode"  // сайдкар NCANode (Kalkan): RSA + ГОСТ РК
)

// Reporter — параметры Reporter Module.
type Reporter struct {
	// SystemDBDenylist — базы ClickHouse, скрываемые из интроспекции (REP-FR «Просмотр структуры», п.2).
	SystemDBDenylist []string
}

// New читает конфигурацию из переменных окружения через viper.
func New() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()

	v.SetDefault("APP_NAME", "ehd-api")
	v.SetDefault("APP_ENV", "local")
	v.SetDefault("HTTP_PORT", "8080")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("CH_DB", "ehd_src")
	v.SetDefault("ADMIN_LOGIN", "admin")
	v.SetDefault("SESSION_TTL", "3h")
	v.SetDefault("TEMP_PASSWORD_TTL", "72h")
	v.SetDefault("MAX_FAILED_ATTEMPTS", 3)
	v.SetDefault("COOKIE_SECURE", false)
	v.SetDefault("EDS_MODE", EDSModeStub)
	v.SetDefault("EDS_TRUST_DIR", "/app/eds-trust")
	v.SetDefault("EDS_OCSP_ENABLED", true)
	v.SetDefault("NCANODE_URL", "http://ncanode:14579")
	v.SetDefault("REPORTER_SYSTEM_DB_DENYLIST", "system,INFORMATION_SCHEMA,information_schema")

	cfg := &Config{
		App:  App{Name: v.GetString("APP_NAME"), Env: v.GetString("APP_ENV")},
		HTTP: HTTP{Port: v.GetString("HTTP_PORT")},
		Log:  Log{Level: v.GetString("LOG_LEVEL")},
		PG:   PG{DSN: v.GetString("PG_DSN")},
		CH: CH{
			Addr:     v.GetString("CH_ADDR"),
			Database: v.GetString("CH_DB"),
			User:     v.GetString("CH_USER"),
			Password: v.GetString("CH_PASSWORD"),
		},
		Admin: Admin{
			Login:    v.GetString("ADMIN_LOGIN"),
			Password: v.GetString("ADMIN_PASSWORD"),
			Email:    v.GetString("ADMIN_EMAIL"),
		},
		Auth: Auth{
			EncKey:            v.GetString("AUTH_ENC_KEY"),
			HMACKey:           v.GetString("AUTH_HMAC_KEY"),
			SessionTTL:        v.GetDuration("SESSION_TTL"),
			TempPasswordTTL:   v.GetDuration("TEMP_PASSWORD_TTL"),
			MaxFailedAttempts: v.GetInt("MAX_FAILED_ATTEMPTS"),
			CookieSecure:      v.GetBool("COOKIE_SECURE"),
		},
		EDS: EDS{
			Mode:        v.GetString("EDS_MODE"),
			TrustDir:    v.GetString("EDS_TRUST_DIR"),
			OCSPEnabled: v.GetBool("EDS_OCSP_ENABLED"),
			NCANodeURL:  v.GetString("NCANODE_URL"),
		},
		Reporter: Reporter{
			SystemDBDenylist: splitList(v.GetString("REPORTER_SYSTEM_DB_DENYLIST")),
		},
	}

	// local: дефолтные ключи для удобства; в non-local обязательны.
	if cfg.App.Env == "local" {
		if cfg.Auth.EncKey == "" {
			cfg.Auth.EncKey = devEncKey
		}
		if cfg.Auth.HMACKey == "" {
			cfg.Auth.HMACKey = "ehd-local-hmac-key"
		}
	}

	for name, val := range map[string]string{
		"PG_DSN":        cfg.PG.DSN,
		"CH_ADDR":       cfg.CH.Addr,
		"CH_USER":       cfg.CH.User,
		"CH_PASSWORD":   cfg.CH.Password,
		"AUTH_ENC_KEY":  cfg.Auth.EncKey,
		"AUTH_HMAC_KEY": cfg.Auth.HMACKey,
	} {
		if val == "" {
			return nil, fmt.Errorf("required env %s is not set", name)
		}
	}

	return cfg, nil
}
