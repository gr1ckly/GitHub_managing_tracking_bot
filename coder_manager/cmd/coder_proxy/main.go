package main

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	dao "coder_manager/pkg/dao"
	"coder_manager/pkg/proxy"
	"coder_manager/pkg/repo"

	"github.com/coder/coder/v2/codersdk"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

	cfg, err := loadProxyConfig()
	if err != nil {
		zap.S().Fatalw("config load failed", "error", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.DB), &gorm.Config{})
	if err != nil {
		zap.S().Fatalw("db connection failed", "error", err)
	}
	if err := db.AutoMigrate(&dao.EditorSession{}, &dao.File{}, &dao.Repo{}, &dao.User{}, &dao.Token{}); err != nil {
		zap.S().Fatalw("db migrate failed", "error", err)
	}

	repoStore := repo.NewGormRepo(db)

	rewriter := proxy.QueryTokenRewriter{
		Param: cfg.TokenQueryParam,
		Token: cfg.CoderAccessToken,
	}
	handler := &proxy.Handler{
		Store:      repoStore,
		Rewriter:   rewriter,
		PathPrefix: cfg.PathPrefix,
	}

	mux := http.NewServeMux()
	mux.Handle(cfg.PathPrefix, handler)

	port := cfg.Port
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	zap.S().Infow("coder proxy listening", "port", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

func loadProxyConfig() (proxy.Config, error) {
	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		return proxy.Config{}, errors.New("DB_DSN is required")
	}
	coderToken := os.Getenv("CODER_ACCESS_TOKEN")
	if coderToken == "" {
		return proxy.Config{}, errors.New("CODER_ACCESS_TOKEN is required")
	}
	port := os.Getenv("PROXY_PORT")
	if port == "" {
		port = "8082"
	}
	tokenParam := os.Getenv("CODER_TOKEN_QUERY_PARAM")
	if tokenParam == "" {
		tokenParam = codersdk.SessionTokenCookie
	}
	pathPrefix := strings.TrimSpace(os.Getenv("PROXY_PATH_PREFIX"))
	if pathPrefix == "" {
		pathPrefix = "/edit/"
	}
	if !strings.HasSuffix(pathPrefix, "/") {
		pathPrefix += "/"
	}
	return proxy.Config{
		DB:               dbDSN,
		Port:             port,
		CoderAccessToken: coderToken,
		TokenQueryParam:  tokenParam,
		PathPrefix:       pathPrefix,
	}, nil
}
