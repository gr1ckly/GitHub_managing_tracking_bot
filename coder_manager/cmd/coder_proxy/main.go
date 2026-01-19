package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"coder_manager/internal/coder_bootstrap"
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
	bootstrapCfg, err := loadBootstrapConfig(false)
	if err != nil {
		return proxy.Config{}, err
	}
	bootstrapResult, err := coder_bootstrap.Ensure(context.Background(), bootstrapCfg)
	if err != nil {
		return proxy.Config{}, err
	}
	if bootstrapResult.AccessToken == "" {
		return proxy.Config{}, errors.New("CODER_ACCESS_TOKEN is required")
	}
	coderToken := bootstrapResult.AccessToken
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

func loadBootstrapConfig(requireTemplate bool) (coder_bootstrap.Config, error) {
	tokenLifetime, err := durationFromEnv("CODER_BOOTSTRAP_TOKEN_LIFETIME", 0)
	if err != nil {
		return coder_bootstrap.Config{}, err
	}
	waitTimeout, err := durationFromEnv("CODER_BOOTSTRAP_WAIT_TIMEOUT", 0)
	if err != nil {
		return coder_bootstrap.Config{}, err
	}
	waitInterval, err := durationFromEnv("CODER_BOOTSTRAP_WAIT_INTERVAL", 0)
	if err != nil {
		return coder_bootstrap.Config{}, err
	}
	tokenScope := codersdk.APIKeyScopeAll
	if raw := strings.TrimSpace(os.Getenv("CODER_BOOTSTRAP_TOKEN_SCOPE")); raw != "" {
		switch strings.ToLower(raw) {
		case string(codersdk.APIKeyScopeApplicationConnect):
			tokenScope = codersdk.APIKeyScopeApplicationConnect
		case string(codersdk.APIKeyScopeAll):
			tokenScope = codersdk.APIKeyScopeAll
		default:
			return coder_bootstrap.Config{}, errors.New("invalid CODER_BOOTSTRAP_TOKEN_SCOPE")
		}
	}
	return coder_bootstrap.Config{
		URL:                     os.Getenv("CODER_URL"),
		AccessToken:             os.Getenv("CODER_ACCESS_TOKEN"),
		TemplateID:              os.Getenv("CODER_TEMPLATE_ID"),
		TemplateVersionID:       os.Getenv("CODER_TEMPLATE_VERSION_ID"),
		TemplateVersionPresetID: os.Getenv("CODER_TEMPLATE_VERSION_PRESET_ID"),
		TemplateName:            os.Getenv("CODER_BOOTSTRAP_TEMPLATE_NAME"),
		TemplateExampleID:       os.Getenv("CODER_BOOTSTRAP_TEMPLATE_EXAMPLE_ID"),
		TemplateExampleName:     os.Getenv("CODER_BOOTSTRAP_TEMPLATE_EXAMPLE_NAME"),
		UserEmail:               os.Getenv("CODER_BOOTSTRAP_EMAIL"),
		Username:                os.Getenv("CODER_BOOTSTRAP_USERNAME"),
		UserPassword:            os.Getenv("CODER_BOOTSTRAP_PASSWORD"),
		UserFullName:            os.Getenv("CODER_BOOTSTRAP_NAME"),
		TokenName:               os.Getenv("CODER_BOOTSTRAP_TOKEN_NAME"),
		TokenLifetime:           tokenLifetime,
		TokenScope:              tokenScope,
		WaitTimeout:             waitTimeout,
		WaitInterval:            waitInterval,
		RequireTemplate:         requireTemplate,
	}, nil
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
