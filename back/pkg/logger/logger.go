package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New возвращает структурированный JSON-логгер zap
// (требование ТЗ: JSON-логи с request_id, user_id, duration и пр.).
func New(level string) (*zap.Logger, error) {
	lvl := zap.NewAtomicLevel()
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl.SetLevel(zapcore.InfoLevel)
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	cfg.Encoding = "json"
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.MessageKey = "msg"

	return cfg.Build()
}
