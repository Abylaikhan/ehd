package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EvgaUserDeptModel — переопределение департамента ДВГА для пользователя ЕХД (спека 016 FR-1).
// Хранится в БД ЕХД: внешняя obm_evga в боевой среде read-only, писать туда нельзя.
type EvgaUserDeptModel struct {
	UserID       string `gorm:"primaryKey;size:36"` // users.id (БД ЕХД)
	DepartmentID int64  `gorm:"not null"`           // obm_evga.its_departments.id
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (EvgaUserDeptModel) TableName() string { return "evga_user_departments" }

// MigrateUserDept — AutoMigrate таблицы привязок в БД ЕХД.
func MigrateUserDept(db *gorm.DB) error { return db.AutoMigrate(&EvgaUserDeptModel{}) }

// UserDeptRepo — привязки «пользователь ЕХД → департамент ДВГА» (БД ЕХД).
type UserDeptRepo struct{ db *gorm.DB }

func NewUserDeptRepo(db *gorm.DB) *UserDeptRepo { return &UserDeptRepo{db: db} }

// Get — департамент пользователя; nil, если переопределения нет.
func (r *UserDeptRepo) Get(ctx context.Context, userID string) (*int64, error) {
	var m EvgaUserDeptModel
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, nil
	case err != nil:
		return nil, err
	}
	id := m.DepartmentID
	return &id, nil
}

// Set — назначить/переназначить департамент (upsert).
func (r *UserDeptRepo) Set(ctx context.Context, userID string, departmentID int64) error {
	m := EvgaUserDeptModel{UserID: userID, DepartmentID: departmentID}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"department_id", "updated_at"}),
	}).Create(&m).Error
}

// Clear — снять переопределение (возврат к резолву по ИИН).
func (r *UserDeptRepo) Clear(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&EvgaUserDeptModel{}).Error
}
