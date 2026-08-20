package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/auth/domain"
	"ehd-api/internal/modules/auth/eds"
)

// Service — реализация contract.Provider и всех use case Auth Module.
var _ contract.Provider = (*Service)(nil)

type Service struct {
	users      UserRepo
	roles      RoleRepo
	sessions   SessionRepo
	ref        ReferenceRepo
	cipher     Cipher
	eds        eds.Verifier
	cfg        Settings
	now        func() time.Time
	challenges *challengeStore
}

func NewService(users UserRepo, roles RoleRepo, sessions SessionRepo, ref ReferenceRepo, cipher Cipher, verifier eds.Verifier, cfg Settings) *Service {
	return &Service{
		users:      users,
		roles:      roles,
		sessions:   sessions,
		ref:        ref,
		cipher:     cipher,
		eds:        verifier,
		cfg:        cfg,
		now:        time.Now,
		challenges: newChallengeStore(),
	}
}

// RegisterInput — данные самостоятельной регистрации.
type RegisterInput struct {
	Login    string
	Password string
	IIN      string
	FullName string
	Email    string
	Phone    string
}

// LoginResult — результат успешной аутентификации.
type LoginResult struct {
	Token                  string
	ExpiresAt              time.Time
	UserID                 string
	Login                  string
	PasswordChangeRequired bool
}

// Register — самостоятельная регистрация; статус pending, ИИН не подтверждён (FR-1).
func (s *Service) Register(ctx context.Context, in RegisterInput) (string, error) {
	if err := domain.ValidateIIN(in.IIN); err != nil {
		return "", err
	}
	if err := domain.ValidatePassword(in.Password); err != nil {
		return "", err
	}

	hmac := s.cipher.HMAC(in.IIN)
	if _, err := s.users.GetByIINHmac(ctx, hmac); err == nil {
		return "", domain.ErrIINTaken
	} else if !errors.Is(err, domain.ErrNotFound) {
		return "", err
	}
	if _, err := s.users.GetByLogin(ctx, in.Login); err == nil {
		return "", domain.ErrLoginTaken
	} else if !errors.Is(err, domain.ErrNotFound) {
		return "", err
	}

	iinEnc, err := s.cipher.EncryptString(in.IIN)
	if err != nil {
		return "", err
	}
	nameEnc, err := s.cipher.EncryptString(in.FullName)
	if err != nil {
		return "", err
	}
	phoneEnc, err := s.cipher.EncryptString(in.Phone)
	if err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	hs := string(hash)

	u := &domain.User{
		Login:        in.Login,
		Email:        in.Email,
		IINEnc:       iinEnc,
		IINHmac:      hmac,
		FullNameEnc:  nameEnc,
		PhoneEnc:     phoneEnc,
		IINVerified:  false,
		Status:       domain.UserStatusPending,
		PasswordHash: &hs,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return "", err
	}
	return u.ID, nil
}

// Login — вход по логину/паролю с блокировкой после N неудачных попыток (FR-2,5,7,8).
func (s *Service) Login(ctx context.Context, login, password string) (*LoginResult, error) {
	u, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}
	if u.Status == domain.UserStatusBlocked {
		return nil, domain.ErrUserBlocked
	}
	if u.PasswordHash == nil {
		return nil, domain.ErrInvalidCredentials
	}
	if u.TempPasswordExpiresAt != nil && s.now().After(*u.TempPasswordExpiresAt) {
		return nil, domain.ErrTempPasswordExpired
	}

	if bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)) != nil {
		u.FailedAttempts++
		if u.FailedAttempts >= s.cfg.MaxFailedAttempts {
			u.Status = domain.UserStatusBlocked
			_ = s.users.Update(ctx, u)
			_ = s.sessions.RevokeAllByUser(ctx, u.ID, s.now())
			return nil, domain.ErrUserBlocked
		}
		_ = s.users.Update(ctx, u)
		return nil, domain.ErrInvalidCredentials
	}

	if u.FailedAttempts != 0 {
		u.FailedAttempts = 0
		if err := s.users.Update(ctx, u); err != nil {
			return nil, err
		}
	}
	return s.startSession(ctx, u)
}

func (s *Service) startSession(ctx context.Context, u *domain.User) (*LoginResult, error) {
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	exp := s.now().Add(s.cfg.SessionTTL)
	sess := &domain.Session{UserID: u.ID, TokenHash: hashToken(token), ExpiresAt: exp}
	if err := s.sessions.Create(ctx, sess); err != nil {
		return nil, err
	}
	return &LoginResult{
		Token:                  token,
		ExpiresAt:              exp,
		UserID:                 u.ID,
		Login:                  u.Login,
		PasswordChangeRequired: u.PasswordChangeRequired,
	}, nil
}

// Logout завершает сессию по токену.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.sessions.RevokeByTokenHash(ctx, hashToken(token), s.now())
}

// ChangePassword — обязательная смена временного/текущего пароля (FR-6).
func (s *Service) ChangePassword(ctx context.Context, userID, oldPw, newPw string) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(oldPw)) != nil {
		return domain.ErrInvalidCredentials
	}
	if err := domain.ValidatePassword(newPw); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPw), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hs := string(hash)
	u.PasswordHash = &hs
	u.PasswordChangeRequired = false
	u.TempPasswordExpiresAt = nil
	return s.users.Update(ctx, u)
}

// CurrentUser реализует contract.Provider: по токену сессии возвращает Identity (FR-4,12).
func (s *Service) CurrentUser(ctx context.Context, token string) (contract.Identity, error) {
	sess, err := s.sessions.GetActiveByTokenHash(ctx, hashToken(token), s.now())
	if err != nil {
		return contract.Identity{}, err
	}
	u, err := s.users.GetByID(ctx, sess.UserID)
	if err != nil {
		return contract.Identity{}, domain.ErrSessionInvalid
	}
	if u.Status == domain.UserStatusBlocked {
		return contract.Identity{}, domain.ErrUserBlocked
	}
	roleCodes, err := s.users.RoleCodesByUser(ctx, u.ID)
	if err != nil {
		return contract.Identity{}, err
	}
	regionCodes, err := s.users.RegionCodesByUser(ctx, u.ID)
	if err != nil {
		return contract.Identity{}, err
	}
	deptCodes, err := s.users.DepartmentCodesByUser(ctx, u.ID)
	if err != nil {
		return contract.Identity{}, err
	}
	return contract.Identity{
		UserID:          u.ID,
		Login:           u.Login,
		IsAdmin:         contains(roleCodes, domain.RoleAdminCode),
		RoleCodes:       roleCodes,
		RegionCodes:     regionCodes,
		DepartmentCodes: deptCodes,
	}, nil
}

// --- ЭЦП (NCALayer): challenge + verify ---

func (s *Service) EDSChallenge() (string, error) {
	nonce, err := randomToken()
	if err != nil {
		return "", err
	}
	s.challenges.put(nonce, s.now().Add(5*time.Minute))
	return nonce, nil
}

// EDSVerify — вход/регистрация по ЭЦП; ИИН считается подтверждённым (FR-3).
func (s *Service) EDSVerify(ctx context.Context, challenge, signedData string) (*LoginResult, error) {
	if !s.challenges.consume(challenge, s.now()) {
		return nil, domain.ErrChallengeInvalid
	}
	sd, err := s.eds.Verify(challenge, signedData)
	if err != nil || !sd.Valid {
		return nil, domain.ErrInvalidCredentials
	}
	if err := domain.ValidateIIN(sd.IIN); err != nil {
		return nil, err
	}

	hmac := s.cipher.HMAC(sd.IIN)
	u, err := s.users.GetByIINHmac(ctx, hmac)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		iinEnc, encErr := s.cipher.EncryptString(sd.IIN)
		if encErr != nil {
			return nil, encErr
		}
		var nameEnc []byte
		if sd.FullName != "" {
			if nameEnc, encErr = s.cipher.EncryptString(sd.FullName); encErr != nil {
				return nil, encErr
			}
		}
		u = &domain.User{
			Login:          "eds_" + sd.IIN,
			IINEnc:         iinEnc,
			IINHmac:        hmac,
			FullNameEnc:    nameEnc,
			IINVerified:    true,
			Status:         domain.UserStatusPending,
			CertificateBIN: binPtr(sd.BIN),
		}
		if err := s.users.Create(ctx, u); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if u.Status == domain.UserStatusBlocked {
			return nil, domain.ErrUserBlocked
		}
		u.IINVerified = true
		if sd.BIN != "" {
			u.CertificateBIN = &sd.BIN
		}
		if err := s.users.Update(ctx, u); err != nil {
			return nil, err
		}
	}
	return s.startSession(ctx, u)
}

func binPtr(bin string) *string {
	if bin == "" {
		return nil
	}
	return &bin
}

// challengeStore — одноразовые ЭЦП-challenge в памяти процесса.
type challengeStore struct {
	mu sync.Mutex
	m  map[string]time.Time
}

func newChallengeStore() *challengeStore { return &challengeStore{m: make(map[string]time.Time)} }

func (c *challengeStore) put(nonce string, exp time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[nonce] = exp
}

func (c *challengeStore) consume(nonce string, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	exp, ok := c.m[nonce]
	if !ok {
		return false
	}
	delete(c.m, nonce)
	return now.Before(exp)
}
