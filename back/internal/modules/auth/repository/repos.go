package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ehd-api/internal/modules/auth/domain"
)

// --- mapping helpers ---

func toDomainUser(m *UserModel) *domain.User {
	u := &domain.User{
		ID:                     m.ID.String(),
		Login:                  m.Login,
		Email:                  m.Email,
		IINEnc:                 m.IINEnc,
		FullNameEnc:            m.FullNameEnc,
		PhoneEnc:               m.PhoneEnc,
		IINVerified:            m.IINVerified,
		Status:                 m.Status,
		PasswordHash:           m.PasswordHash,
		FailedAttempts:         int(m.FailedAttempts),
		CertificateBIN:         m.CertificateBIN,
		PasswordChangeRequired: m.PasswordChangeRequired,
		TempPasswordExpiresAt:  m.TempPasswordExpiresAt,
		CreatedAt:              m.CreatedAt,
		UpdatedAt:              m.UpdatedAt,
	}
	if m.IINHmac != nil {
		u.IINHmac = *m.IINHmac
	}
	return u
}

func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func translateNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrNotFound
	}
	return err
}

// --- UserRepo ---

type UserRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	m := UserModel{
		Login:                  u.Login,
		IINEnc:                 u.IINEnc,
		IINHmac:                ptrOrNil(u.IINHmac),
		FullNameEnc:            u.FullNameEnc,
		PhoneEnc:               u.PhoneEnc,
		Email:                  u.Email,
		IINVerified:            u.IINVerified,
		Status:                 u.Status,
		PasswordHash:           u.PasswordHash,
		FailedAttempts:         int16(u.FailedAttempts),
		CertificateBIN:         u.CertificateBIN,
		PasswordChangeRequired: u.PasswordChangeRequired,
		TempPasswordExpiresAt:  u.TempPasswordExpiresAt,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return err
	}
	u.ID = m.ID.String()
	u.CreatedAt = m.CreatedAt
	u.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *UserRepo) getBy(ctx context.Context, cond string, arg any) (*domain.User, error) {
	var m UserModel
	if err := r.db.WithContext(ctx).Where(cond, arg).First(&m).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return toDomainUser(&m), nil
}

func (r *UserRepo) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	return r.getBy(ctx, "login = ?", login)
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return r.getBy(ctx, "id = ?", uid)
}

func (r *UserRepo) GetByIINHmac(ctx context.Context, hmac string) (*domain.User, error) {
	return r.getBy(ctx, "iin_hmac = ?", hmac)
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	uid, err := uuid.Parse(u.ID)
	if err != nil {
		return domain.ErrNotFound
	}
	return r.db.WithContext(ctx).Model(&UserModel{}).Where("id = ?", uid).
		Updates(map[string]any{
			"email":                    u.Email,
			"iin_verified":             u.IINVerified,
			"status":                   u.Status,
			"password_hash":            u.PasswordHash,
			"failed_attempts":          int16(u.FailedAttempts),
			"certificate_bin":          u.CertificateBIN,
			"password_change_required": u.PasswordChangeRequired,
			"temp_password_expires_at": u.TempPasswordExpiresAt,
			"updated_at":               time.Now(),
		}).Error
}

func (r *UserRepo) List(ctx context.Context, status, q string, limit, offset int) ([]domain.User, int64, error) {
	tx := r.db.WithContext(ctx).Model(&UserModel{})
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("login ILIKE ? OR email ILIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []UserModel
	if err := tx.Order("created_at DESC").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.User, 0, len(models))
	for i := range models {
		out = append(out, *toDomainUser(&models[i]))
	}
	return out, total, nil
}

func (r *UserRepo) CountActiveAdmins(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("users u").
		Joins("JOIN user_roles ur ON ur.user_id = u.id").
		Joins("JOIN roles r ON r.id = ur.role_id").
		Where("r.code = ? AND u.status = ?", domain.RoleAdminCode, domain.UserStatusActive).
		Distinct("u.id").Count(&n).Error
	return n, err
}

func (r *UserRepo) replaceLinks(ctx context.Context, table, col, userID string, ids []string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return domain.ErrNotFound
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM "+table+" WHERE user_id = ?", uid).Error; err != nil {
			return err
		}
		for _, id := range ids {
			pid, err := uuid.Parse(id)
			if err != nil {
				return err
			}
			if err := tx.Exec("INSERT INTO "+table+" (user_id, "+col+") VALUES (?, ?)", uid, pid).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *UserRepo) SetRoles(ctx context.Context, userID string, roleIDs []string) error {
	return r.replaceLinks(ctx, "user_roles", "role_id", userID, roleIDs)
}

func (r *UserRepo) SetRegions(ctx context.Context, userID string, regionIDs []string) error {
	return r.replaceLinks(ctx, "user_regions", "region_id", userID, regionIDs)
}

func (r *UserRepo) SetDepartments(ctx context.Context, userID string, departmentIDs []string) error {
	return r.replaceLinks(ctx, "user_departments", "department_id", userID, departmentIDs)
}

func (r *UserRepo) pluckCodes(ctx context.Context, refTable, linkTable, linkCol, userID string) ([]string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	var codes []string
	err = r.db.WithContext(ctx).Table(refTable+" ref").
		Joins("JOIN "+linkTable+" lnk ON lnk."+linkCol+" = ref.id").
		Where("lnk.user_id = ? AND ref.status = ?", uid, domain.StatusActive).
		Pluck("ref.code", &codes).Error
	return codes, err
}

func (r *UserRepo) RoleCodesByUser(ctx context.Context, userID string) ([]string, error) {
	return r.pluckCodes(ctx, "roles", "user_roles", "role_id", userID)
}

func (r *UserRepo) RegionCodesByUser(ctx context.Context, userID string) ([]string, error) {
	return r.pluckCodes(ctx, "regions", "user_regions", "region_id", userID)
}

func (r *UserRepo) DepartmentCodesByUser(ctx context.Context, userID string) ([]string, error) {
	return r.pluckCodes(ctx, "departments", "user_departments", "department_id", userID)
}

// --- RoleRepo ---

type RoleRepo struct{ db *gorm.DB }

func NewRoleRepo(db *gorm.DB) *RoleRepo { return &RoleRepo{db: db} }

func mapRole(m *RoleModel) domain.Role {
	return domain.Role{ID: m.ID.String(), Code: m.Code, NameRu: m.NameRu, NameKk: m.NameKk, Status: m.Status}
}

func (r *RoleRepo) List(ctx context.Context) ([]domain.Role, error) {
	var models []RoleModel
	if err := r.db.WithContext(ctx).Order("code").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Role, 0, len(models))
	for i := range models {
		out = append(out, mapRole(&models[i]))
	}
	return out, nil
}

func (r *RoleRepo) Create(ctx context.Context, role *domain.Role) error {
	m := RoleModel{Code: role.Code, NameRu: role.NameRu, NameKk: role.NameKk, Status: role.Status}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return err
	}
	role.ID = m.ID.String()
	return nil
}

func (r *RoleRepo) GetByCode(ctx context.Context, code string) (*domain.Role, error) {
	var m RoleModel
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&m).Error; err != nil {
		return nil, translateNotFound(err)
	}
	role := mapRole(&m)
	return &role, nil
}

func (r *RoleRepo) RolesByUser(ctx context.Context, userID string) ([]domain.Role, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	var models []RoleModel
	err = r.db.WithContext(ctx).Table("roles r").
		Joins("JOIN user_roles ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", uid).Order("r.code").Find(&models).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.Role, 0, len(models))
	for i := range models {
		out = append(out, mapRole(&models[i]))
	}
	return out, nil
}

// --- SessionRepo ---

type SessionRepo struct{ db *gorm.DB }

func NewSessionRepo(db *gorm.DB) *SessionRepo { return &SessionRepo{db: db} }

func (r *SessionRepo) Create(ctx context.Context, s *domain.Session) error {
	uid, err := uuid.Parse(s.UserID)
	if err != nil {
		return domain.ErrNotFound
	}
	m := SessionModel{UserID: uid, TokenHash: s.TokenHash, ExpiresAt: s.ExpiresAt}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return err
	}
	s.ID = m.ID.String()
	s.CreatedAt = m.CreatedAt
	return nil
}

func (r *SessionRepo) GetActiveByTokenHash(ctx context.Context, hash string, now time.Time) (*domain.Session, error) {
	var m SessionModel
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hash, now).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSessionInvalid
		}
		return nil, err
	}
	return &domain.Session{
		ID: m.ID.String(), UserID: m.UserID.String(), TokenHash: m.TokenHash,
		CreatedAt: m.CreatedAt, ExpiresAt: m.ExpiresAt, RevokedAt: m.RevokedAt,
	}, nil
}

func (r *SessionRepo) RevokeByTokenHash(ctx context.Context, hash string, now time.Time) error {
	return r.db.WithContext(ctx).Model(&SessionModel{}).
		Where("token_hash = ? AND revoked_at IS NULL", hash).
		Update("revoked_at", now).Error
}

func (r *SessionRepo) RevokeAllByUser(ctx context.Context, userID string, now time.Time) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return domain.ErrNotFound
	}
	return r.db.WithContext(ctx).Model(&SessionModel{}).
		Where("user_id = ? AND revoked_at IS NULL", uid).
		Update("revoked_at", now).Error
}

func (r *SessionRepo) DeleteExpired(ctx context.Context, now time.Time) error {
	return r.db.WithContext(ctx).Where("expires_at < ?", now).Delete(&SessionModel{}).Error
}

// --- ReferenceRepo ---

type ReferenceRepo struct{ db *gorm.DB }

func NewReferenceRepo(db *gorm.DB) *ReferenceRepo { return &ReferenceRepo{db: db} }

func (r *ReferenceRepo) ListRegions(ctx context.Context) ([]domain.Reference, error) {
	var models []RegionModel
	if err := r.db.WithContext(ctx).Order("code").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Reference, 0, len(models))
	for _, m := range models {
		out = append(out, domain.Reference{ID: m.ID.String(), Code: m.Code, NameRu: m.NameRu, NameKk: m.NameKk, Status: m.Status})
	}
	return out, nil
}

func (r *ReferenceRepo) ListDepartments(ctx context.Context) ([]domain.Reference, error) {
	var models []DepartmentModel
	if err := r.db.WithContext(ctx).Order("code").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Reference, 0, len(models))
	for _, m := range models {
		out = append(out, domain.Reference{ID: m.ID.String(), Code: m.Code, NameRu: m.NameRu, NameKk: m.NameKk, Status: m.Status})
	}
	return out, nil
}
