// Package application — use cases Auth Module. Определяет порты (интерфейсы),
// которые реализует repository; зависимости направлены внутрь.
package application

import (
	"context"
	"time"

	"ehd-api/internal/modules/auth/domain"
)

type UserRepo interface {
	Create(ctx context.Context, u *domain.User) error
	GetByLogin(ctx context.Context, login string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByIINHmac(ctx context.Context, hmac string) (*domain.User, error)
	Update(ctx context.Context, u *domain.User) error
	List(ctx context.Context, status, q string, limit, offset int) ([]domain.User, int64, error)
	CountActiveAdmins(ctx context.Context) (int64, error)
	SetRoles(ctx context.Context, userID string, roleIDs []string) error
	SetRegions(ctx context.Context, userID string, regionIDs []string) error
	SetDepartments(ctx context.Context, userID string, departmentIDs []string) error
	RoleCodesByUser(ctx context.Context, userID string) ([]string, error)
	RegionCodesByUser(ctx context.Context, userID string) ([]string, error)
	DepartmentCodesByUser(ctx context.Context, userID string) ([]string, error)
}

type RoleRepo interface {
	List(ctx context.Context) ([]domain.Role, error)
	Create(ctx context.Context, role *domain.Role) error
	GetByCode(ctx context.Context, code string) (*domain.Role, error)
	RolesByUser(ctx context.Context, userID string) ([]domain.Role, error)
}

type SessionRepo interface {
	Create(ctx context.Context, s *domain.Session) error
	GetActiveByTokenHash(ctx context.Context, hash string, now time.Time) (*domain.Session, error)
	RevokeByTokenHash(ctx context.Context, hash string, now time.Time) error
	RevokeAllByUser(ctx context.Context, userID string, now time.Time) error
}

type ReferenceRepo interface {
	ListRegions(ctx context.Context) ([]domain.Reference, error)
	ListDepartments(ctx context.Context) ([]domain.Reference, error)
}

// Cipher — шифрование PII и HMAC ИИН (реализуется pkg/crypto).
type Cipher interface {
	EncryptString(string) ([]byte, error)
	DecryptString([]byte) (string, error)
	HMAC(string) string
}

// Settings — параметры Auth из конфигурации.
type Settings struct {
	SessionTTL        time.Duration
	TempPasswordTTL   time.Duration
	MaxFailedAttempts int
}
