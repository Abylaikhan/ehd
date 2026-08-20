// Команда seed вручную создаёт/обновляет первого администратора через GORM.
// В обычном режиме админ создаётся автоматически при старте ehd-api (AutoMigrate + seed).
package main

import (
	"log"

	"ehd-api/config"
	authrepo "ehd-api/internal/modules/auth/repository"
	"ehd-api/pkg/postgres"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.Admin.Password == "" {
		log.Fatal("ADMIN_PASSWORD не задан")
	}

	db, err := postgres.New(cfg.PG.DSN)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	if err := postgres.EnsureSchemas(db, "auth"); err != nil {
		log.Fatalf("schema: %v", err)
	}
	if err := authrepo.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := authrepo.SeedAdmin(db, cfg.Admin.Login, cfg.Admin.Email, cfg.Admin.Password); err != nil {
		log.Fatalf("seed: %v", err)
	}

	log.Printf("администратор %q создан/обновлён", cfg.Admin.Login)
}
