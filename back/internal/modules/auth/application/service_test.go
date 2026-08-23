package application

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"ehd-api/internal/modules/auth/domain"
	"ehd-api/internal/modules/auth/eds"
)

// --- fakes ---

type fakeCipher struct{}

func (fakeCipher) EncryptString(s string) ([]byte, error) { return []byte(s), nil }
func (fakeCipher) DecryptString(b []byte) (string, error) { return string(b), nil }
func (fakeCipher) HMAC(s string) string                   { return "h:" + s }

func cloneUser(u *domain.User) *domain.User { c := *u; return &c }

type fakeUsers struct {
	byID map[string]*domain.User
	seq  int
}

func newFakeUsers() *fakeUsers { return &fakeUsers{byID: map[string]*domain.User{}} }

func (f *fakeUsers) Create(_ context.Context, u *domain.User) error {
	f.seq++
	u.ID = "u" + strconv.Itoa(f.seq)
	u.CreatedAt = time.Unix(0, 0)
	f.byID[u.ID] = cloneUser(u)
	return nil
}
func (f *fakeUsers) GetByID(_ context.Context, id string) (*domain.User, error) {
	if u, ok := f.byID[id]; ok {
		return cloneUser(u), nil
	}
	return nil, domain.ErrNotFound
}
func (f *fakeUsers) GetByLogin(_ context.Context, login string) (*domain.User, error) {
	for _, u := range f.byID {
		if u.Login == login {
			return cloneUser(u), nil
		}
	}
	return nil, domain.ErrNotFound
}
func (f *fakeUsers) GetByIINHmac(_ context.Context, hmac string) (*domain.User, error) {
	for _, u := range f.byID {
		if u.IINHmac == hmac {
			return cloneUser(u), nil
		}
	}
	return nil, domain.ErrNotFound
}
func (f *fakeUsers) Update(_ context.Context, u *domain.User) error {
	if _, ok := f.byID[u.ID]; !ok {
		return domain.ErrNotFound
	}
	f.byID[u.ID] = cloneUser(u)
	return nil
}
func (f *fakeUsers) List(context.Context, string, string, int, int) ([]domain.User, int64, error) {
	return nil, 0, nil
}
func (f *fakeUsers) CountActiveAdmins(context.Context) (int64, error)                { return 1, nil }
func (f *fakeUsers) SetRoles(context.Context, string, []string) error                { return nil }
func (f *fakeUsers) SetRegions(context.Context, string, []string) error              { return nil }
func (f *fakeUsers) SetDepartments(context.Context, string, []string) error          { return nil }
func (f *fakeUsers) RoleCodesByUser(context.Context, string) ([]string, error)       { return nil, nil }
func (f *fakeUsers) RegionCodesByUser(context.Context, string) ([]string, error)     { return nil, nil }
func (f *fakeUsers) DepartmentCodesByUser(context.Context, string) ([]string, error) { return nil, nil }

type fakeRoles struct{}

func (fakeRoles) List(context.Context) ([]domain.Role, error) { return nil, nil }
func (fakeRoles) Create(context.Context, *domain.Role) error  { return nil }
func (fakeRoles) GetByCode(_ context.Context, code string) (*domain.Role, error) {
	return &domain.Role{ID: "r-admin", Code: code}, nil
}
func (fakeRoles) RolesByUser(context.Context, string) ([]domain.Role, error) { return nil, nil }

type fakeSessions struct {
	byHash map[string]*domain.Session
}

func newFakeSessions() *fakeSessions { return &fakeSessions{byHash: map[string]*domain.Session{}} }

func (f *fakeSessions) Create(_ context.Context, s *domain.Session) error {
	s.ID = "s" + strconv.Itoa(len(f.byHash)+1)
	cp := *s
	f.byHash[s.TokenHash] = &cp
	return nil
}
func (f *fakeSessions) GetActiveByTokenHash(_ context.Context, hash string, now time.Time) (*domain.Session, error) {
	s, ok := f.byHash[hash]
	if !ok || s.RevokedAt != nil || !now.Before(s.ExpiresAt) {
		return nil, domain.ErrSessionInvalid
	}
	cp := *s
	return &cp, nil
}
func (f *fakeSessions) RevokeByTokenHash(_ context.Context, hash string, now time.Time) error {
	if s, ok := f.byHash[hash]; ok {
		s.RevokedAt = &now
	}
	return nil
}
func (f *fakeSessions) RevokeAllByUser(_ context.Context, userID string, now time.Time) error {
	for _, s := range f.byHash {
		if s.UserID == userID {
			s.RevokedAt = &now
		}
	}
	return nil
}

type fakeRef struct{}

func (fakeRef) ListRegions(context.Context) ([]domain.Reference, error)     { return nil, nil }
func (fakeRef) ListDepartments(context.Context) ([]domain.Reference, error) { return nil, nil }

// --- test harness ---

func newTestService(t *testing.T, now func() time.Time) (*Service, *fakeUsers) {
	t.Helper()
	users := newFakeUsers()
	svc := NewService(users, fakeRoles{}, newFakeSessions(), fakeRef{}, fakeCipher{}, eds.StubVerifier{},
		Settings{SessionTTL: time.Hour, TempPasswordTTL: 72 * time.Hour, MaxFailedAttempts: 3})
	svc.now = now
	return svc, users
}

func mustHash(t *testing.T, pw string) *string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	s := string(h)
	return &s
}

func TestLoginLockoutAfterThreeFailures(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	svc, users := newTestService(t, func() time.Time { return base })
	users.byID["u1"] = &domain.User{ID: "u1", Login: "ivan", Status: domain.UserStatusActive, PasswordHash: mustHash(t, "Passw0rd")}

	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := svc.Login(ctx, "ivan", "wrong"); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("attempt %d: want ErrInvalidCredentials, got %v", i+1, err)
		}
	}
	if _, err := svc.Login(ctx, "ivan", "wrong"); !errors.Is(err, domain.ErrUserBlocked) {
		t.Fatalf("3rd attempt: want ErrUserBlocked, got %v", err)
	}
	if _, err := svc.Login(ctx, "ivan", "Passw0rd"); !errors.Is(err, domain.ErrUserBlocked) {
		t.Fatalf("blocked user with correct pw: want ErrUserBlocked, got %v", err)
	}
	if users.byID["u1"].Status != domain.UserStatusBlocked {
		t.Fatalf("user status = %q, want blocked", users.byID["u1"].Status)
	}
}

func TestLoginTempPasswordExpired(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	svc, users := newTestService(t, func() time.Time { return base })
	past := base.Add(-time.Hour)
	users.byID["u1"] = &domain.User{ID: "u1", Login: "ivan", Status: domain.UserStatusActive,
		PasswordHash: mustHash(t, "Passw0rd"), TempPasswordExpiresAt: &past}

	if _, err := svc.Login(context.Background(), "ivan", "Passw0rd"); !errors.Is(err, domain.ErrTempPasswordExpired) {
		t.Fatalf("want ErrTempPasswordExpired, got %v", err)
	}
}

func TestSessionLifecycleAndExpiry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	clock := func() time.Time { return now }
	svc, users := newTestService(t, clock)
	users.byID["u1"] = &domain.User{ID: "u1", Login: "ivan", Status: domain.UserStatusActive, PasswordHash: mustHash(t, "Passw0rd")}

	ctx := context.Background()
	res, err := svc.Login(ctx, "ivan", "Passw0rd")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	id, err := svc.CurrentUser(ctx, res.Token)
	if err != nil {
		t.Fatalf("current user: %v", err)
	}
	if id.UserID != "u1" {
		t.Fatalf("identity user = %q, want u1", id.UserID)
	}

	// продвигаем время за пределы TTL сессии → сессия недействительна
	now = now.Add(2 * time.Hour)
	if _, err := svc.CurrentUser(ctx, res.Token); !errors.Is(err, domain.ErrSessionInvalid) {
		t.Fatalf("expired session: want ErrSessionInvalid, got %v", err)
	}
}

func TestChangePasswordSetsWhenNoPassword(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	svc, users := newTestService(t, func() time.Time { return base })
	users.byID["u1"] = &domain.User{ID: "u1", Login: "eds_x", Status: domain.UserStatusPending, PasswordHash: nil}
	ctx := context.Background()

	// пароля нет (вход по ЭЦП) — можно задать первый без старого
	if err := svc.ChangePassword(ctx, "u1", "", "NewPass1"); err != nil {
		t.Fatalf("set password without existing: %v", err)
	}
	if users.byID["u1"].PasswordHash == nil {
		t.Fatal("password was not set")
	}
	// теперь пароль есть — неверный старый отклоняется
	if err := svc.ChangePassword(ctx, "u1", "wrong", "NewPass2"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("with password + wrong old: want ErrInvalidCredentials, got %v", err)
	}
}

func TestRegisterRejectsDuplicateIIN(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	svc, _ := newTestService(t, func() time.Time { return base })
	ctx := context.Background()
	in := RegisterInput{Login: "ivan", Password: "Passw0rd", IIN: "990101300123", FullName: "Иван", Email: "i@x.kz"}
	if _, err := svc.Register(ctx, in); err != nil {
		t.Fatalf("first register: %v", err)
	}
	dup := in
	dup.Login = "ivan2"
	if _, err := svc.Register(ctx, dup); !errors.Is(err, domain.ErrIINTaken) {
		t.Fatalf("duplicate IIN: want ErrIINTaken, got %v", err)
	}
}
