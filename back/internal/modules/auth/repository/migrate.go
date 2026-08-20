package repository

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"ehd-api/internal/modules/auth/domain"
)

// Migrate создаёт/обновляет таблицы схемы auth через GORM AutoMigrate.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&UserModel{},
		&RoleModel{},
		&UserRoleModel{},
		&SessionModel{},
	)
}

// SeedAdmin идемпотентно создаёт первого администратора и его роль.
// Повторный вызов обновляет пароль и статус существующей записи.
func SeedAdmin(db *gorm.DB, login, email, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hashStr := string(hash)

	return db.Transaction(func(tx *gorm.DB) error {
		role := RoleModel{Code: domain.RoleAdminCode}
		if err := tx.Where(RoleModel{Code: domain.RoleAdminCode}).
			Attrs(RoleModel{NameRu: "Администратор", NameKk: "Әкімші", Status: "active"}).
			FirstOrCreate(&role).Error; err != nil {
			return err
		}

		var user UserModel
		err := tx.Where("login = ?", login).First(&user).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			user = UserModel{
				Login:        login,
				Email:        email,
				Status:       domain.UserStatusActive,
				IINVerified:  true,
				PasswordHash: &hashStr,
			}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			user.PasswordHash = &hashStr
			user.Status = domain.UserStatusActive
			if err := tx.Save(&user).Error; err != nil {
				return err
			}
		}

		link := UserRoleModel{UserID: user.ID, RoleID: role.ID}
		return tx.Where(link).FirstOrCreate(&link).Error
	})
}
