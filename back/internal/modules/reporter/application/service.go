// Package application — сценарии Reporter Module: управление источником ClickHouse
// и интроспекция его структуры. Пароль источника шифруется здесь и наружу не отдаётся.
package application

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	"ehd-api/internal/modules/reporter/domain"
	"ehd-api/pkg/crypto"
)

// SourceRepo — хранилище источника в PostgreSQL.
type SourceRepo interface {
	Create(ctx context.Context, ds domain.DataSource, passwordEnc []byte) (domain.DataSource, error)
	Get(ctx context.Context, id string) (domain.DataSource, error)
	GetActive(ctx context.Context) (domain.DataSource, error)
	List(ctx context.Context) ([]domain.DataSource, error)
	Update(ctx context.Context, ds domain.DataSource, passwordEnc []byte) (domain.DataSource, error)
	SetStatus(ctx context.Context, id, status string) error
	Count(ctx context.Context) (int64, error)
	Secret(ctx context.Context, id string) ([]byte, error)
}

// ConnParams — параметры подключения к источнику (порт приложения к ClickHouse-адаптеру).
type ConnParams struct {
	Host          string
	Port          int
	Protocol      string
	TLSEnabled    bool
	TLSSkipVerify bool
	Username      string
	Password      string
	Database      string
}

// SourceConn — открытое подключение к источнику для проверки, интроспекции и выполнения запросов.
type SourceConn interface {
	Ping(ctx context.Context) error
	Databases(ctx context.Context) ([]string, error)
	Tables(ctx context.Context, db string) ([]domain.Table, error)
	Columns(ctx context.Context, db, table string) ([]domain.Column, error)
	SortingKey(ctx context.Context, db, table string) ([]string, error)
	Query(ctx context.Context, sql string, args ...any) ([]map[string]any, error)
	ScalarUint64(ctx context.Context, sql string, args ...any) (uint64, error)
	Close() error
}

// Connector открывает подключение к ClickHouse по параметрам источника (driven-порт).
type Connector interface {
	Open(ctx context.Context, p ConnParams) (SourceConn, error)
}

// Config — настройки Reporter из окружения.
type Config struct {
	SystemDBDenylist []string
}

// Service — сценарии управления источником и интроспекции.
type Service struct {
	repo   SourceRepo
	cipher *crypto.Cipher
	conn   Connector
	cfg    Config
	log    *zap.Logger
}

func NewService(repo SourceRepo, cipher *crypto.Cipher, conn Connector, cfg Config, log *zap.Logger) *Service {
	return &Service{repo: repo, cipher: cipher, conn: conn, cfg: cfg, log: log}
}

// CreateSourceInput — входные данные создания источника.
type CreateSourceInput struct {
	Code          string
	Name          string
	Host          string
	Port          int
	Protocol      string
	TLSEnabled    bool
	TLSSkipVerify bool
	Username      string
	Password      string
}

// CreateSource создаёт единственный источник (REP-FR-010, REP-BR-001).
func (s *Service) CreateSource(ctx context.Context, in CreateSourceInput) (domain.DataSource, error) {
	if in.Protocol == "" {
		in.Protocol = domain.ProtocolNative
	}
	if err := validateCreate(in); err != nil {
		return domain.DataSource{}, err
	}

	n, err := s.repo.Count(ctx)
	if err != nil {
		return domain.DataSource{}, err
	}
	if n > 0 {
		return domain.DataSource{}, domain.ErrSourceAlreadyExists
	}

	enc, err := s.cipher.EncryptString(in.Password)
	if err != nil {
		return domain.DataSource{}, err
	}
	ds := domain.DataSource{
		Code:          in.Code,
		Name:          in.Name,
		Host:          in.Host,
		Port:          in.Port,
		Protocol:      in.Protocol,
		TLSEnabled:    in.TLSEnabled,
		TLSSkipVerify: in.TLSSkipVerify,
		Username:      in.Username,
		Status:        domain.SourceStatusInactive, // активируется админом после успешного теста
	}
	return s.repo.Create(ctx, ds, enc)
}

// UpdateSourceInput — обновление источника; Password пустой не меняет секрет.
type UpdateSourceInput struct {
	CreateSourceInput
}

// UpdateSource обновляет поля источника; пустой пароль оставляет секрет прежним.
func (s *Service) UpdateSource(ctx context.Context, id string, in UpdateSourceInput) (domain.DataSource, error) {
	if in.Protocol == "" {
		in.Protocol = domain.ProtocolNative
	}
	if err := validateUpdate(in); err != nil {
		return domain.DataSource{}, err
	}
	if _, err := s.repo.Get(ctx, id); err != nil {
		return domain.DataSource{}, err
	}

	var enc []byte
	if in.Password != "" {
		e, err := s.cipher.EncryptString(in.Password)
		if err != nil {
			return domain.DataSource{}, err
		}
		enc = e
	}
	ds := domain.DataSource{
		ID:            id,
		Code:          in.Code,
		Name:          in.Name,
		Host:          in.Host,
		Port:          in.Port,
		Protocol:      in.Protocol,
		TLSEnabled:    in.TLSEnabled,
		TLSSkipVerify: in.TLSSkipVerify,
		Username:      in.Username,
	}
	return s.repo.Update(ctx, ds, enc)
}

// GetSource возвращает источник по id (без пароля).
func (s *Service) GetSource(ctx context.Context, id string) (domain.DataSource, error) {
	return s.repo.Get(ctx, id)
}

// ListSources возвращает все источники (без пароля).
func (s *Service) ListSources(ctx context.Context) ([]domain.DataSource, error) {
	return s.repo.List(ctx)
}

// Activate/Deactivate меняют статус источника (REP-FR-012).
func (s *Service) Activate(ctx context.Context, id string) error {
	return s.setStatus(ctx, id, domain.SourceStatusActive)
}

func (s *Service) Deactivate(ctx context.Context, id string) error {
	return s.setStatus(ctx, id, domain.SourceStatusInactive)
}

func (s *Service) setStatus(ctx context.Context, id, status string) error {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return err
	}
	return s.repo.SetStatus(ctx, id, status)
}

// TestParams проверяет подключение по переданным параметрам без сохранения секрета (REP-FR-011).
func (s *Service) TestParams(ctx context.Context, p ConnParams) error {
	if p.Protocol == "" {
		p.Protocol = domain.ProtocolNative
	}
	conn, err := s.conn.Open(ctx, p)
	if err != nil {
		return domain.ErrConnectionFailed
	}
	defer conn.Close()

	tctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := conn.Ping(tctx); err != nil {
		// Логируем без секрета: только адрес и ошибку драйвера.
		s.log.Warn("source connection test failed",
			zap.String("host", p.Host), zap.Int("port", p.Port), zap.Error(err))
		return domain.ErrConnectionFailed
	}
	return nil
}

// TestSource проверяет подключение сохранённого источника (REP-FR-011).
func (s *Service) TestSource(ctx context.Context, id string) error {
	p, err := s.connParams(ctx, id, "")
	if err != nil {
		return err
	}
	return s.TestParams(ctx, p)
}

// Databases возвращает базы источника без системных (REP-FR «Просмотр структуры», п.2).
func (s *Service) Databases(ctx context.Context, id string) ([]domain.Database, error) {
	conn, err := s.open(ctx, id, "")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	names, err := conn.Databases(ctx)
	if err != nil {
		return nil, domain.ErrConnectionFailed
	}
	names = filterDatabases(names, s.cfg.SystemDBDenylist)
	out := make([]domain.Database, len(names))
	for i, n := range names {
		out[i] = domain.Database{Name: n}
	}
	return out, nil
}

// Tables возвращает таблицы/представления выбранной базы (REP-FR «Просмотр структуры», п.3).
func (s *Service) Tables(ctx context.Context, id, db string) ([]domain.Table, error) {
	conn, err := s.open(ctx, id, "")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	tables, err := conn.Tables(ctx, db)
	if err != nil {
		return nil, domain.ErrConnectionFailed
	}
	return tables, nil
}

// Columns возвращает колонки таблицы (REP-FR «Просмотр структуры», п.4–5).
func (s *Service) Columns(ctx context.Context, id, db, table string) ([]domain.Column, error) {
	conn, err := s.open(ctx, id, db)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	cols, err := conn.Columns(ctx, db, table)
	if err != nil {
		return nil, domain.ErrConnectionFailed
	}
	return cols, nil
}

// SortingKey возвращает колонки сортировочного ключа таблицы (для keyset-пагинации).
func (s *Service) SortingKey(ctx context.Context, id, db, table string) ([]string, error) {
	conn, err := s.open(ctx, id, db)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	key, err := conn.SortingKey(ctx, db, table)
	if err != nil {
		return nil, domain.ErrConnectionFailed
	}
	return key, nil
}

// RunQuery выполняет проверенный SELECT на подключении к источнику (для Query Engine).
func (s *Service) RunQuery(ctx context.Context, sourceID, database, sql string, args []any) ([]map[string]any, error) {
	conn, err := s.open(ctx, sourceID, database)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return conn.Query(ctx, sql, args...)
}

// ScalarCount выполняет COUNT-запрос и возвращает целое (для total_count).
func (s *Service) ScalarCount(ctx context.Context, sourceID, database, sql string, args []any) (uint64, error) {
	conn, err := s.open(ctx, sourceID, database)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return conn.ScalarUint64(ctx, sql, args...)
}

// open открывает подключение к сохранённому источнику (с расшифровкой секрета).
func (s *Service) open(ctx context.Context, id, database string) (SourceConn, error) {
	p, err := s.connParams(ctx, id, database)
	if err != nil {
		return nil, err
	}
	conn, err := s.conn.Open(ctx, p)
	if err != nil {
		return nil, domain.ErrConnectionFailed
	}
	return conn, nil
}

// connParams собирает параметры подключения из сохранённого источника (секрет расшифровывается).
func (s *Service) connParams(ctx context.Context, id, database string) (ConnParams, error) {
	ds, err := s.repo.Get(ctx, id)
	if err != nil {
		return ConnParams{}, err
	}
	enc, err := s.repo.Secret(ctx, id)
	if err != nil {
		return ConnParams{}, err
	}
	pass, err := s.cipher.DecryptString(enc)
	if err != nil {
		return ConnParams{}, err
	}
	return ConnParams{
		Host:          ds.Host,
		Port:          ds.Port,
		Protocol:      ds.Protocol,
		TLSEnabled:    ds.TLSEnabled,
		TLSSkipVerify: ds.TLSSkipVerify,
		Username:      ds.Username,
		Password:      pass,
		Database:      database,
	}, nil
}

// filterDatabases убирает системные базы из списка (без учёта регистра).
func filterDatabases(names, denylist []string) []string {
	deny := make(map[string]struct{}, len(denylist))
	for _, d := range denylist {
		deny[strings.ToLower(strings.TrimSpace(d))] = struct{}{}
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		if _, bad := deny[strings.ToLower(n)]; bad {
			continue
		}
		out = append(out, n)
	}
	return out
}

func validateCreate(in CreateSourceInput) error {
	if strings.TrimSpace(in.Code) == "" ||
		strings.TrimSpace(in.Name) == "" ||
		strings.TrimSpace(in.Host) == "" ||
		strings.TrimSpace(in.Username) == "" {
		return domain.ErrValidation
	}
	if in.Port <= 0 || in.Port > 65535 {
		return domain.ErrValidation
	}
	if !domain.ValidProtocol(in.Protocol) {
		return domain.ErrValidation
	}
	if in.Password == "" {
		return domain.ErrValidation
	}
	return nil
}

func validateUpdate(in UpdateSourceInput) error {
	// как create, но пароль опционален
	if strings.TrimSpace(in.Code) == "" ||
		strings.TrimSpace(in.Name) == "" ||
		strings.TrimSpace(in.Host) == "" ||
		strings.TrimSpace(in.Username) == "" {
		return domain.ErrValidation
	}
	if in.Port <= 0 || in.Port > 65535 {
		return domain.ErrValidation
	}
	if !domain.ValidProtocol(in.Protocol) {
		return domain.ErrValidation
	}
	return nil
}
