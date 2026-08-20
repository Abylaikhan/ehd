package postgres

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// New открывает подключение GORM к основной БД ЕХД.
func New(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
		// кросс-схемные FK создаём вручную в моделях по мере надобности;
		// на AutoMigrate отключаем, чтобы не зависеть от порядка таблиц.
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// EnsureSchemas создаёт схемы модулей перед AutoMigrate.
// Имена берутся из внутренних констант, не из пользовательского ввода.
func EnsureSchemas(db *gorm.DB, schemas ...string) error {
	for _, s := range schemas {
		if err := db.Exec("CREATE SCHEMA IF NOT EXISTS " + s).Error; err != nil {
			return err
		}
	}
	return nil
}
