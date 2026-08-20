package application

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"

	"ehd-api/internal/modules/auth/domain"
)

// AdminUserView — представление пользователя для админ-интерфейса (ПДн расшифрованы/замаскированы).
type AdminUserView struct {
	ID              string
	Login           string
	Email           string
	FullName        string
	IINMasked       string
	IINVerified     bool
	Status          string
	FailedAttempts  int
	Roles           []string
	RegionCodes     []string
	DepartmentCodes []string
	CreatedAt       time.Time
}

// UpdateUserInput — частичное обновление админом; nil-поля не меняются.
type UpdateUserInput struct {
	IINVerified   *bool
	Status        *string
	RoleIDs       *[]string
	RegionIDs     *[]string
	DepartmentIDs *[]string
}

func (s *Service) buildUserView(ctx context.Context, u *domain.User) (AdminUserView, error) {
	fullName, err := s.cipher.DecryptString(u.FullNameEnc)
	if err != nil {
		return AdminUserView{}, err
	}
	iin, err := s.cipher.DecryptString(u.IINEnc)
	if err != nil {
		return AdminUserView{}, err
	}
	roleCodes, err := s.users.RoleCodesByUser(ctx, u.ID)
	if err != nil {
		return AdminUserView{}, err
	}
	regionCodes, err := s.users.RegionCodesByUser(ctx, u.ID)
	if err != nil {
		return AdminUserView{}, err
	}
	deptCodes, err := s.users.DepartmentCodesByUser(ctx, u.ID)
	if err != nil {
		return AdminUserView{}, err
	}
	return AdminUserView{
		ID:              u.ID,
		Login:           u.Login,
		Email:           u.Email,
		FullName:        fullName,
		IINMasked:       maskIIN(iin),
		IINVerified:     u.IINVerified,
		Status:          u.Status,
		FailedAttempts:  u.FailedAttempts,
		Roles:           roleCodes,
		RegionCodes:     regionCodes,
		DepartmentCodes: deptCodes,
		CreatedAt:       u.CreatedAt,
	}, nil
}

// ListUsers — список пользователей с фильтром/пагинацией (FR-10).
func (s *Service) ListUsers(ctx context.Context, status, q string, limit, offset int) ([]AdminUserView, int64, error) {
	users, total, err := s.users.List(ctx, status, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	views := make([]AdminUserView, 0, len(users))
	for i := range users {
		v, err := s.buildUserView(ctx, &users[i])
		if err != nil {
			return nil, 0, err
		}
		views = append(views, v)
	}
	return views, total, nil
}

// GetUser — карточка пользователя.
func (s *Service) GetUser(ctx context.Context, id string) (AdminUserView, error) {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return AdminUserView{}, err
	}
	return s.buildUserView(ctx, u)
}

// UpdateUser — проверка ИИН, статус, назначение ролей/регионов/подразделений (FR-10),
// с защитой последнего активного администратора (FR-13).
func (s *Service) UpdateUser(ctx context.Context, id string, in UpdateUserInput) error {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}

	isAdmin, err := s.isActiveAdmin(ctx, u)
	if err != nil {
		return err
	}

	if in.Status != nil && *in.Status == domain.UserStatusBlocked && isAdmin {
		last, err := s.isLastAdmin(ctx)
		if err != nil {
			return err
		}
		if last {
			return domain.ErrLastAdmin
		}
	}

	if in.RoleIDs != nil && isAdmin {
		keepsAdmin, err := s.roleSetKeepsAdmin(ctx, *in.RoleIDs)
		if err != nil {
			return err
		}
		if !keepsAdmin {
			last, err := s.isLastAdmin(ctx)
			if err != nil {
				return err
			}
			if last {
				return domain.ErrLastAdmin
			}
		}
	}

	if in.IINVerified != nil {
		u.IINVerified = *in.IINVerified
	}
	if in.Status != nil {
		u.Status = *in.Status
	}
	if err := s.users.Update(ctx, u); err != nil {
		return err
	}
	if in.RoleIDs != nil {
		if err := s.users.SetRoles(ctx, id, *in.RoleIDs); err != nil {
			return err
		}
	}
	if in.RegionIDs != nil {
		if err := s.users.SetRegions(ctx, id, *in.RegionIDs); err != nil {
			return err
		}
	}
	if in.DepartmentIDs != nil {
		if err := s.users.SetDepartments(ctx, id, *in.DepartmentIDs); err != nil {
			return err
		}
	}
	return nil
}

// UnlockUser — разблокировка после 3 неудачных входов (FR-7).
func (s *Service) UnlockUser(ctx context.Context, id string) error {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}
	u.Status = domain.UserStatusActive
	u.FailedAttempts = 0
	return s.users.Update(ctx, u)
}

// SetTempPassword — админ создаёт временный пароль (действует TempPasswordTTL), FR-6.
// Возвращает открытый временный пароль (показывается администратору один раз).
func (s *Service) SetTempPassword(ctx context.Context, id string) (string, error) {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	temp, err := genTempPassword()
	if err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(temp), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	hs := string(hash)
	exp := s.now().Add(s.cfg.TempPasswordTTL)
	u.PasswordHash = &hs
	u.PasswordChangeRequired = true
	u.TempPasswordExpiresAt = &exp
	if err := s.users.Update(ctx, u); err != nil {
		return "", err
	}
	return temp, nil
}

func (s *Service) ListRoles(ctx context.Context) ([]domain.Role, error) {
	return s.roles.List(ctx)
}

func (s *Service) CreateRole(ctx context.Context, code, nameRu, nameKk string) (*domain.Role, error) {
	role := &domain.Role{Code: code, NameRu: nameRu, NameKk: nameKk, Status: domain.StatusActive}
	if err := s.roles.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *Service) ListRegions(ctx context.Context) ([]domain.Reference, error) {
	return s.ref.ListRegions(ctx)
}

func (s *Service) ListDepartments(ctx context.Context) ([]domain.Reference, error) {
	return s.ref.ListDepartments(ctx)
}

// --- helpers ---

func (s *Service) isActiveAdmin(ctx context.Context, u *domain.User) (bool, error) {
	if u.Status != domain.UserStatusActive {
		return false, nil
	}
	codes, err := s.users.RoleCodesByUser(ctx, u.ID)
	if err != nil {
		return false, err
	}
	return contains(codes, domain.RoleAdminCode), nil
}

func (s *Service) isLastAdmin(ctx context.Context) (bool, error) {
	n, err := s.users.CountActiveAdmins(ctx)
	if err != nil {
		return false, err
	}
	return n <= 1, nil
}

func (s *Service) roleSetKeepsAdmin(ctx context.Context, roleIDs []string) (bool, error) {
	adminRole, err := s.roles.GetByCode(ctx, domain.RoleAdminCode)
	if err != nil {
		return false, err
	}
	return contains(roleIDs, adminRole.ID), nil
}

func maskIIN(iin string) string {
	if len(iin) < 4 {
		return "****"
	}
	return "********" + iin[len(iin)-4:]
}

// genTempPassword гарантированно удовлетворяет политике (заглавная, строчная, цифра).
func genTempPassword() (string, error) {
	tok, err := randomToken()
	if err != nil {
		return "", err
	}
	return "Aa1" + tok[:9], nil
}
