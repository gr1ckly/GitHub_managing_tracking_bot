package main

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	logger, err := buildLogger()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	zap.ReplaceGlobals(logger)
	zap.L().Info("logger initialized", zap.String("service", "link_tracker"))
}

func buildLogger() (*zap.Logger, error) {
	env := strings.ToLower(os.Getenv("APP_ENV"))
	level := parseLogLevel(os.Getenv("LOG_LEVEL"))

	var cfg zap.Config
	if env == "dev" {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}
	cfg.Level = zap.NewAtomicLevelAt(level)
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return cfg.Build()
}

func parseLogLevel(raw string) zapcore.Level {
	switch strings.ToLower(raw) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}
