package util

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger(logLevel string) {
	config := zap.NewProductionConfig()
	level, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		level = zapcore.InfoLevel
	}
	config.Level.SetLevel(level)
	Logger, _ = config.Build()
}

// Error 返回一个 zap.Field，用于记录错误
func Error(err error) zap.Field {
	return zap.Error(err)
}

// Int 返回一个 zap.Field，用于记录整数
func Int(key string, value int) zap.Field {
	return zap.Int(key, value)
}
