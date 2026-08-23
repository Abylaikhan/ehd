package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ehd-api/internal/modules/reporter/domain"
)

// DataSourceRepo — доступ к источнику ClickHouse в PostgreSQL.
type DataSourceRepo struct{ db *gorm.DB }

func NewDataSourceRepo(db *gorm.DB) *DataSourceRepo { return &DataSourceRepo{db: db} }

func toDomainSource(m DataSourceModel) domain.DataSource {
	return domain.DataSource{
		ID:            m.ID.String(),
		Code:          m.Code,
		Name:          m.Name,
		Host:          m.Host,
		Port:          m.Port,
		Protocol:      m.Protocol,
		TLSEnabled:    m.TLSEnabled,
		TLSSkipVerify: m.TLSSkipVerify,
		Username:      m.Username,
		Status:        m.Status,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

// Create сохраняет новый источник с уже зашифрованным паролем.
func (r *DataSourceRepo) Create(ctx context.Context, ds domain.DataSource, passwordEnc []byte) (domain.DataSource, error) {
	m := DataSourceModel{
		ID:            uuid.New(),
		Code:          ds.Code,
		Name:          ds.Name,
		Host:          ds.Host,
		Port:          ds.Port,
		Protocol:      ds.Protocol,
		TLSEnabled:    ds.TLSEnabled,
		TLSSkipVerify: ds.TLSSkipVerify,
		Username:      ds.Username,
		PasswordEnc:   passwordEnc,
		Status:        ds.Status,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return domain.DataSource{}, err
	}
	return toDomainSource(m), nil
}

// Get возвращает источник по id (без пароля).
func (r *DataSourceRepo) Get(ctx context.Context, id string) (domain.DataSource, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.DataSource{}, domain.ErrSourceNotFound
	}
	var m DataSourceModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.DataSource{}, domain.ErrSourceNotFound
		}
		return domain.DataSource{}, err
	}
	return toDomainSource(m), nil
}

// GetActive возвращает активный источник (используется при выполнении запросов).
func (r *DataSourceRepo) GetActive(ctx context.Context) (domain.DataSource, error) {
	var m DataSourceModel
	if err := r.db.WithContext(ctx).First(&m, "status = ?", domain.SourceStatusActive).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.DataSource{}, domain.ErrSourceNotFound
		}
		return domain.DataSource{}, err
	}
	return toDomainSource(m), nil
}

// List возвращает все источники (без пароля).
func (r *DataSourceRepo) List(ctx context.Context) ([]domain.DataSource, error) {
	var ms []DataSourceModel
	if err := r.db.WithContext(ctx).Order("created_at").Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]domain.DataSource, len(ms))
	for i, m := range ms {
		out[i] = toDomainSource(m)
	}
	return out, nil
}

// Update обновляет поля источника; passwordEnc == nil — пароль не меняется.
func (r *DataSourceRepo) Update(ctx context.Context, ds domain.DataSource, passwordEnc []byte) (domain.DataSource, error) {
	uid, err := uuid.Parse(ds.ID)
	if err != nil {
		return domain.DataSource{}, domain.ErrSourceNotFound
	}
	updates := map[string]any{
		"code":            ds.Code,
		"name":            ds.Name,
		"host":            ds.Host,
		"port":            ds.Port,
		"protocol":        ds.Protocol,
		"tls_enabled":     ds.TLSEnabled,
		"tls_skip_verify": ds.TLSSkipVerify,
		"username":        ds.Username,
	}
	if passwordEnc != nil {
		updates["password_enc"] = passwordEnc
	}
	res := r.db.WithContext(ctx).Model(&DataSourceModel{}).Where("id = ?", uid).Updates(updates)
	if res.Error != nil {
		return domain.DataSource{}, res.Error
	}
	if res.RowsAffected == 0 {
		return domain.DataSource{}, domain.ErrSourceNotFound
	}
	return r.Get(ctx, ds.ID)
}

// SetStatus меняет статус источника (активация/деактивация).
func (r *DataSourceRepo) SetStatus(ctx context.Context, id, status string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.ErrSourceNotFound
	}
	res := r.db.WithContext(ctx).Model(&DataSourceModel{}).Where("id = ?", uid).Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrSourceNotFound
	}
	return nil
}

// Count возвращает число источников (для инварианта единственности).
func (r *DataSourceRepo) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&DataSourceModel{}).Count(&n).Error
	return n, err
}

// Secret возвращает зашифрованный пароль источника для установления подключения.
func (r *DataSourceRepo) Secret(ctx context.Context, id string) ([]byte, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, domain.ErrSourceNotFound
	}
	var m DataSourceModel
	if err := r.db.WithContext(ctx).Select("password_enc").First(&m, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSourceNotFound
		}
		return nil, err
	}
	return m.PasswordEnc, nil
}
