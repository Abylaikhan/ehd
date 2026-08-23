package application

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"ehd-api/internal/modules/reporter/domain"
	"ehd-api/pkg/crypto"
)

const testEncKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

func newCipher(t *testing.T) *crypto.Cipher {
	t.Helper()
	c, err := crypto.New(testEncKey, "test-hmac")
	if err != nil {
		t.Fatalf("crypto.New: %v", err)
	}
	return c
}

// --- фейки ---

type fakeRepo struct {
	count      int64
	created    domain.DataSource
	createdEnc []byte
	source     domain.DataSource
	secret     []byte
	getErr     error
}

func (r *fakeRepo) Create(_ context.Context, ds domain.DataSource, enc []byte) (domain.DataSource, error) {
	r.created = ds
	r.createdEnc = enc
	ds.ID = "11111111-1111-1111-1111-111111111111"
	return ds, nil
}
func (r *fakeRepo) Get(_ context.Context, _ string) (domain.DataSource, error) {
	return r.source, r.getErr
}
func (r *fakeRepo) GetActive(context.Context) (domain.DataSource, error) {
	return r.source, r.getErr
}
func (r *fakeRepo) List(context.Context) ([]domain.DataSource, error) { return nil, nil }
func (r *fakeRepo) Update(_ context.Context, ds domain.DataSource, _ []byte) (domain.DataSource, error) {
	return ds, nil
}
func (r *fakeRepo) SetStatus(context.Context, string, string) error { return nil }
func (r *fakeRepo) Count(context.Context) (int64, error)            { return r.count, nil }
func (r *fakeRepo) Secret(context.Context, string) ([]byte, error)  { return r.secret, nil }

type fakeConn struct {
	pingErr error
	dbs     []string
}

func (c *fakeConn) Ping(context.Context) error                  { return c.pingErr }
func (c *fakeConn) Databases(context.Context) ([]string, error) { return c.dbs, nil }
func (c *fakeConn) Tables(context.Context, string) ([]domain.Table, error) {
	return nil, nil
}
func (c *fakeConn) Columns(context.Context, string, string) ([]domain.Column, error) {
	return nil, nil
}
func (c *fakeConn) Query(context.Context, string, ...any) ([]map[string]any, error) {
	return nil, nil
}
func (c *fakeConn) ScalarUint64(context.Context, string, ...any) (uint64, error) { return 0, nil }
func (c *fakeConn) Close() error                                                 { return nil }

type fakeConnector struct {
	openErr error
	conn    *fakeConn
}

func (f fakeConnector) Open(context.Context, ConnParams) (SourceConn, error) {
	if f.openErr != nil {
		return nil, f.openErr
	}
	return f.conn, nil
}

func validInput() CreateSourceInput {
	return CreateSourceInput{
		Code: "ch-main", Name: "Основной ClickHouse", Host: "clickhouse", Port: 9000,
		Protocol: domain.ProtocolNative, Username: "reporter_ro", Password: "s3cr3t",
	}
}

func newService(repo SourceRepo, conn Connector, cipher *crypto.Cipher) *Service {
	return NewService(repo, cipher, conn, Config{
		SystemDBDenylist: []string{"system", "INFORMATION_SCHEMA", "information_schema"},
	}, zap.NewNop())
}

// --- тесты ---

func TestCreateSource_EncryptsPasswordAndDefaultsInactive(t *testing.T) {
	repo := &fakeRepo{count: 0}
	cipher := newCipher(t)
	svc := newService(repo, fakeConnector{conn: &fakeConn{}}, cipher)

	ds, err := svc.CreateSource(context.Background(), validInput())
	if err != nil {
		t.Fatalf("CreateSource: %v", err)
	}
	if ds.Status != domain.SourceStatusInactive {
		t.Errorf("новый источник должен быть inactive, получено %q", ds.Status)
	}
	// пароль в хранилище зашифрован (AC-1): не равен plaintext, но расшифровывается обратно.
	if string(repo.createdEnc) == "s3cr3t" {
		t.Error("пароль сохранён в открытом виде")
	}
	got, err := cipher.DecryptString(repo.createdEnc)
	if err != nil || got != "s3cr3t" {
		t.Errorf("расшифровка пароля: got=%q err=%v", got, err)
	}
}

func TestCreateSource_AlreadyExists(t *testing.T) {
	repo := &fakeRepo{count: 1} // источник уже есть
	svc := newService(repo, fakeConnector{conn: &fakeConn{}}, newCipher(t))

	_, err := svc.CreateSource(context.Background(), validInput())
	if !errors.Is(err, domain.ErrSourceAlreadyExists) {
		t.Fatalf("ожидалась ErrSourceAlreadyExists, получено %v", err)
	}
}

func TestCreateSource_Validation(t *testing.T) {
	cases := map[string]func(*CreateSourceInput){
		"empty_code":     func(in *CreateSourceInput) { in.Code = "" },
		"empty_name":     func(in *CreateSourceInput) { in.Name = "" },
		"empty_host":     func(in *CreateSourceInput) { in.Host = "" },
		"empty_username": func(in *CreateSourceInput) { in.Username = "" },
		"empty_password": func(in *CreateSourceInput) { in.Password = "" },
		"bad_port_zero":  func(in *CreateSourceInput) { in.Port = 0 },
		"bad_port_high":  func(in *CreateSourceInput) { in.Port = 70000 },
		"bad_protocol":   func(in *CreateSourceInput) { in.Protocol = "ftp" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &fakeRepo{count: 0}
			svc := newService(repo, fakeConnector{conn: &fakeConn{}}, newCipher(t))
			in := validInput()
			mutate(&in)
			if _, err := svc.CreateSource(context.Background(), in); !errors.Is(err, domain.ErrValidation) {
				t.Fatalf("ожидалась ErrValidation, получено %v", err)
			}
		})
	}
}

func TestCreateSource_DefaultProtocolNative(t *testing.T) {
	repo := &fakeRepo{count: 0}
	svc := newService(repo, fakeConnector{conn: &fakeConn{}}, newCipher(t))
	in := validInput()
	in.Protocol = "" // не задан → native
	ds, err := svc.CreateSource(context.Background(), in)
	if err != nil {
		t.Fatalf("CreateSource: %v", err)
	}
	if ds.Protocol != domain.ProtocolNative {
		t.Errorf("протокол по умолчанию должен быть native, получено %q", ds.Protocol)
	}
}

func TestTestParams_ConnectionFailures(t *testing.T) {
	cipher := newCipher(t)
	// не удалось открыть подключение
	svc := newService(&fakeRepo{}, fakeConnector{openErr: errors.New("dial")}, cipher)
	if err := svc.TestParams(context.Background(), ConnParams{Host: "x", Port: 9000}); !errors.Is(err, domain.ErrConnectionFailed) {
		t.Fatalf("open err → ожидалась ErrConnectionFailed, получено %v", err)
	}
	// подключение открылось, но SELECT 1 упал
	svc = newService(&fakeRepo{}, fakeConnector{conn: &fakeConn{pingErr: errors.New("auth")}}, cipher)
	if err := svc.TestParams(context.Background(), ConnParams{Host: "x", Port: 9000}); !errors.Is(err, domain.ErrConnectionFailed) {
		t.Fatalf("ping err → ожидалась ErrConnectionFailed, получено %v", err)
	}
}

func TestTestParams_OK(t *testing.T) {
	svc := newService(&fakeRepo{}, fakeConnector{conn: &fakeConn{}}, newCipher(t))
	if err := svc.TestParams(context.Background(), ConnParams{Host: "clickhouse", Port: 9000}); err != nil {
		t.Fatalf("ожидался успех, получено %v", err)
	}
}

func TestDatabases_ExcludesSystem(t *testing.T) {
	cipher := newCipher(t)
	enc, _ := cipher.EncryptString("pass")
	repo := &fakeRepo{
		source: domain.DataSource{ID: "id", Host: "clickhouse", Port: 9000, Protocol: domain.ProtocolNative, Username: "u"},
		secret: enc,
	}
	conn := &fakeConn{dbs: []string{"default", "system", "ehd_src", "INFORMATION_SCHEMA", "information_schema"}}
	svc := newService(repo, fakeConnector{conn: conn}, cipher)

	dbs, err := svc.Databases(context.Background(), "id")
	if err != nil {
		t.Fatalf("Databases: %v", err)
	}
	got := make([]string, len(dbs))
	for i, d := range dbs {
		got[i] = d.Name
	}
	want := map[string]bool{"default": true, "ehd_src": true}
	if len(got) != len(want) {
		t.Fatalf("ожидалось %v, получено %v", want, got)
	}
	for _, n := range got {
		if !want[n] {
			t.Errorf("системная база %q не должна попадать в список", n)
		}
	}
}

func TestFilterDatabases_CaseInsensitive(t *testing.T) {
	out := filterDatabases(
		[]string{"Default", "SYSTEM", "ehd_src", "Information_Schema"},
		[]string{"system", "information_schema"},
	)
	for _, n := range out {
		if n == "SYSTEM" || n == "Information_Schema" {
			t.Errorf("база %q должна быть отфильтрована без учёта регистра", n)
		}
	}
	if len(out) != 2 {
		t.Fatalf("ожидалось 2 базы, получено %v", out)
	}
}
