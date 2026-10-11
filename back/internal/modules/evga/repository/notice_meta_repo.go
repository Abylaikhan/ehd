package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NoticeMetaModel — реквизиты уведомления, которых нет в obm_evga (спека 018 FR-2).
// Хранится в БД ЕХД: внешняя БД в боевой среде read-only, добавлять туда колонки нельзя.
type NoticeMetaModel struct {
	NoticeID  int64 `gorm:"primaryKey"` // its_risk_notice.id (obm_evga)
	Comment   string `gorm:"type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (NoticeMetaModel) TableName() string { return "evga_notice_meta" }

// MigrateNoticeMeta — AutoMigrate таблицы реквизитов в БД ЕХД.
func MigrateNoticeMeta(db *gorm.DB) error { return db.AutoMigrate(&NoticeMetaModel{}) }

// NoticeMetaRepo — комментарий к уведомлению (БД ЕХД).
type NoticeMetaRepo struct{ db *gorm.DB }

func NewNoticeMetaRepo(db *gorm.DB) *NoticeMetaRepo { return &NoticeMetaRepo{db: db} }

// Comment — текущий комментарий; пусто, если не задан.
func (r *NoticeMetaRepo) Comment(ctx context.Context, noticeID int64) (string, error) {
	var m NoticeMetaModel
	err := r.db.WithContext(ctx).Where("notice_id = ?", noticeID).First(&m).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return "", nil
	case err != nil:
		return "", err
	}
	return m.Comment, nil
}

// SetComment — сохранить комментарий (upsert).
func (r *NoticeMetaRepo) SetComment(ctx context.Context, noticeID int64, comment string) error {
	m := NoticeMetaModel{NoticeID: noticeID, Comment: comment}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "notice_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"comment", "updated_at"}),
	}).Create(&m).Error
}
