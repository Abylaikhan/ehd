package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	App   App
	HTTP  HTTP
	Log   Log
	PG    PG
	CH    CH
	Admin Admin
}

type App struct {
	Name string
	Env  string
}

type HTTP struct {
	Port string
}

type Log struct {
	Level string
}

type PG struct {
	DSN string
}

// CH — read-only источник ClickHouse для Reporter Module.
type CH struct {
	Addr     string
	Database string
	User     string
	Password string
}

// Admin — данные seed-администратора (создаётся при старте, если задан пароль).
type Admin struct {
	Login    string
	Password string
	Email    string
}

// New читает конфигурацию из переменных окружения через viper.
func New() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()

	v.SetDefault("APP_NAME", "ehd-api")
	v.SetDefault("APP_ENV", "local")
	v.SetDefault("HTTP_PORT", "8080")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("CH_DB", "ehd_src")
	v.SetDefault("ADMIN_LOGIN", "admin")

	cfg := &Config{
		App:  App{Name: v.GetString("APP_NAME"), Env: v.GetString("APP_ENV")},
		HTTP: HTTP{Port: v.GetString("HTTP_PORT")},
		Log:  Log{Level: v.GetString("LOG_LEVEL")},
		PG:   PG{DSN: v.GetString("PG_DSN")},
		CH: CH{
			Addr:     v.GetString("CH_ADDR"),
			Database: v.GetString("CH_DB"),
			User:     v.GetString("CH_USER"),
			Password: v.GetString("CH_PASSWORD"),
		},
		Admin: Admin{
			Login:    v.GetString("ADMIN_LOGIN"),
			Password: v.GetString("ADMIN_PASSWORD"),
			Email:    v.GetString("ADMIN_EMAIL"),
		},
	}

	for name, val := range map[string]string{
		"PG_DSN":      cfg.PG.DSN,
		"CH_ADDR":     cfg.CH.Addr,
		"CH_USER":     cfg.CH.User,
		"CH_PASSWORD": cfg.CH.Password,
	} {
		if val == "" {
			return nil, fmt.Errorf("required env %s is not set", name)
		}
	}

	return cfg, nil
}
