package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"coder_manager/internal/coder_service"
	"coder_manager/internal/grpc_server"
	"coder_manager/internal/tasks"
	"coder_manager/pkg/coder_client"
	dao "coder_manager/pkg/dao"
	"coder_manager/pkg/file_storage"
	"coder_manager/pkg/notifier"
	"coder_manager/pkg/proto"
	"coder_manager/pkg/repo"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	logger, err := initLogger()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	dbDSN, err := requireEnv("DB_DSN")
	if err != nil {
		zap.S().Fatalw("config load failed", "error", err)
	}
	proxyBaseURL, err := requireEnv("PROXY_BASE_URL")
	if err != nil {
		zap.S().Fatalw("config load failed", "error", err)
	}
	coderAccessToken := os.Getenv("CODER_ACCESS_TOKEN")
	grpcPort := envOrDefault("GRPC_PORT", "9090")
	sessionSaverPeriod, err := durationFromEnv("SESSION_SAVER_PERIOD", 30*time.Second)
	if err != nil {
		zap.S().Fatalw("config load failed", "error", err)
	}
	sessionSaverLimit, err := intFromEnv("SESSION_SAVER_LIMIT", 100)
	if err != nil {
		zap.S().Fatalw("config load failed", "error", err)
	}
	coderCfg, err := loadCoderConfig()
	if err != nil {
		zap.S().Fatalw("config load failed", "error", err)
	}
	s3Cfg, err := loadS3Config()
	if err != nil {
		zap.S().Fatalw("config load failed", "error", err)
	}

	db, err := gorm.Open(postgres.Open(dbDSN), &gorm.Config{})
	if err != nil {
		zap.S().Fatalw("db connection failed", "error", err)
	}
	if err := db.AutoMigrate(&dao.EditorSession{}, &dao.File{}, &dao.Repo{}, &dao.User{}, &dao.Token{}); err != nil {
		zap.S().Fatalw("db migrate failed", "error", err)
	}

	repoStore := repo.NewGormRepo(db)
	storage, err := file_storage.NewS3Storage(s3Cfg)
	if err != nil {
		zap.S().Fatalw("s3 init failed", "error", err)
	}
	notifyClient := notifier.NewStubNotifier()

	coderClient, err := coder_client.NewSDKClient(coderCfg)
	if err != nil {
		zap.S().Fatalw("coder client init failed", "error", err)
	}
	service := coder_service.NewService(repoStore, coderClient, storage, notifyClient, proxyBaseURL, coderAccessToken)

	saver := tasks.NewSessionSaver(service, sessionSaverPeriod, sessionSaverLimit)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go saver.Run(ctx)

	port := grpcPort
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		zap.S().Fatalw("listen error", "error", err)
	}

	server := grpc.NewServer()
	proto.RegisterCoderManagerServiceServer(server, grpc_server.NewCoderManagerServer(service))
	reflection.Register(server)

	zap.S().Infow("coder manager grpc listening", "port", port)
	if err := server.Serve(listener); err != nil {
		zap.S().Fatalw("server error", "error", err)
	}
}

func initLogger() (*zap.SugaredLogger, error) {
	level := strings.ToLower(os.Getenv("LOG_LEVEL"))
	if level == "" {
		level = "info"
	}
	encoding := strings.ToLower(os.Getenv("LOG_ENCODING"))
	if encoding == "" {
		encoding = "json"
	}
	cfg := zap.NewProductionConfig()
	if encoding != "json" {
		cfg.Encoding = "console"
	}
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.Level = zap.NewAtomicLevelAt(parseLogLevel(level))
	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	zap.ReplaceGlobals(logger)
	return logger.Sugar(), nil
}

func parseLogLevel(raw string) zapcore.Level {
	switch strings.ToLower(raw) {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
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

func loadCoderConfig() (coder_client.Config, error) {
	workspaceReadyTimeout := time.Duration(0)
	if raw := os.Getenv("CODER_WORKSPACE_READY_TIMEOUT"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return coder_client.Config{}, err
		}
		workspaceReadyTimeout = parsed
	}
	return coder_client.Config{
		URL:                     os.Getenv("CODER_URL"),
		AccessToken:             os.Getenv("CODER_ACCESS_TOKEN"),
		TemplateID:              os.Getenv("CODER_TEMPLATE_ID"),
		TemplateVersionID:       os.Getenv("CODER_TEMPLATE_VERSION_ID"),
		TemplateVersionPresetID: os.Getenv("CODER_TEMPLATE_VERSION_PRESET_ID"),
		User:                    os.Getenv("CODER_USER"),
		EditorAppSlug:           os.Getenv("CODER_EDITOR_APP_SLUG"),
		AgentName:               os.Getenv("CODER_AGENT_NAME"),
		WorkspaceReadyTimeout:   workspaceReadyTimeout,
	}, nil
}

func loadS3Config() (file_storage.Config, error) {
	forcePathStyle, err := boolFromEnv("S3_FORCE_PATH_STYLE", false)
	if err != nil {
		return file_storage.Config{}, err
	}
	maxSize, err := int64FromEnv("S3_MAX_SIZE_BYTES", 1<<30)
	if err != nil {
		return file_storage.Config{}, err
	}
	partSize, err := int64FromEnv("S3_PART_SIZE_BYTES", 0)
	if err != nil {
		return file_storage.Config{}, err
	}
	concurrency, err := intFromEnv("S3_UPLOAD_CONCURRENCY", 0)
	if err != nil {
		return file_storage.Config{}, err
	}
	return file_storage.Config{
		Endpoint:       os.Getenv("S3_ENDPOINT"),
		AccessKey:      os.Getenv("S3_ACCESS_KEY"),
		SecretKey:      os.Getenv("S3_SECRET_KEY"),
		Region:         os.Getenv("S3_REGION"),
		Bucket:         os.Getenv("S3_BUCKET"),
		ForcePathStyle: forcePathStyle,
		MaxSizeBytes:   maxSize,
		PartSizeBytes:  partSize,
		Concurrency:    concurrency,
	}, nil
}

func requireEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", errors.New(key + " is required")
	}
	return value, nil
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fallback, err
	}
	return parsed, nil
}

func intFromEnv(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback, err
	}
	return parsed, nil
}

func boolFromEnv(key string, fallback bool) (bool, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback, err
	}
	return parsed, nil
}

func int64FromEnv(key string, fallback int64) (int64, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback, err
	}
	return parsed, nil
}
