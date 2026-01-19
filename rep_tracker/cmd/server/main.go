package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	gormio "gorm.io/gorm"

	"rep_tracker/internal/grpc_server"
	"rep_tracker/internal/rep_service"
	"rep_tracker/pkg/github"
	repgorm "rep_tracker/pkg/gorm"
	"rep_tracker/pkg/proto"
)

func main() {
	logger, err := buildLogger()
	if err != nil {
		fmt.Printf("Error when building logger: %v", err)
		return
	}
	defer logger.Sync()

	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "rep_tracker_server"
	}
	logger = logger.With(zap.String("service", serviceName))
	zap.ReplaceGlobals(logger)
	zap.L().Info("logger initialized")

	cfg, err := loadConfig()
	if err != nil {
		zap.L().Fatal("invalid config", zap.Error(err))
	}

	db, err := gormio.Open(postgres.Open(cfg.dbDSN), &gormio.Config{})
	if err != nil {
		zap.L().Fatal("db connection failed", zap.Error(err))
	}

	tokenRepo := repgorm.NewGormTokenRepo(db)
	serverRepo := repgorm.NewGormServerRepo(db)
	ghClient := github.NewGithubClient()
	repService := rep_service.NewRepService(ghClient, tokenRepo, serverRepo)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	err = grpc_server.ConfigureGrpcServerAndServer(ctx, &cfg.grpc, func(s grpc.ServiceRegistrar) {
		proto.RegisterRepTrackerServiceServer(s, grpc_server.NewRepTrackerServiceServer(repService))
	})
	if err != nil {
		zap.L().Fatal("grpc server stopped with error", zap.Error(err))
	}
}

type appConfig struct {
	dbDSN string
	grpc  grpc_server.GrpcServerConfig
}

func loadConfig() (appConfig, error) {
	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		return appConfig{}, fmt.Errorf("DB_DSN is required")
	}

	grpcCfg := grpc_server.GrpcServerConfig{
		Addr:                    getEnvString("GRPC_ADDR", ":8081"),
		Transport:               getEnvString("GRPC_TRANSPORT", "tcp"),
		ConcurrentStreamsNumber: getEnvInt("GRPC_MAX_CONCURRENT_STREAMS", 0),
		MaxRcvSize:              getEnvInt("GRPC_MAX_RECV_SIZE", 0),
		MaxSendSize:             getEnvInt("GRPC_MAX_SEND_SIZE", 0),
		EnableHealthService:     getEnvBool("GRPC_ENABLE_HEALTH", true),
		EnableReflection:        getEnvBool("GRPC_ENABLE_REFLECTION", false),
		MaxConnectionIdle:       getEnvDuration("GRPC_MAX_CONN_IDLE_SEC", 0),
		MaxConnectionAge:        getEnvDuration("GRPC_MAX_CONN_AGE_SEC", 0),
		MaxConnectionAgeGrace:   getEnvDuration("GRPC_MAX_CONN_AGE_GRACE_SEC", 0),
		KeepAliveTime:           getEnvDuration("GRPC_KEEPALIVE_TIME_SEC", 0),
		KeepAliveTimeout:        getEnvDuration("GRPC_KEEPALIVE_TIMEOUT_SEC", 0),
		KeepAliveMinTime:        getEnvDuration("GRPC_KEEPALIVE_MIN_TIME_SEC", 0),
		KeepAliveWithoutStream:  getEnvBool("GRPC_KEEPALIVE_PERMIT_WITHOUT_STREAM", false),
		GracefulStopTimeout:     getEnvDuration("GRPC_GRACEFUL_STOP_TIMEOUT_SEC", 10),
	}

	return appConfig{dbDSN: dbDSN, grpc: grpcCfg}, nil
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

func getEnvString(key string, def string) string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	return raw
}

func getEnvInt(key string, def int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return val
}

func getEnvBool(key string, def bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "y":
		return true
	case "0", "false", "no", "n":
		return false
	default:
		return def
	}
}

func getEnvDuration(key string, defSec int) time.Duration {
	sec := getEnvInt(key, defSec)
	if sec <= 0 {
		return 0
	}
	return time.Duration(sec) * time.Second
}
