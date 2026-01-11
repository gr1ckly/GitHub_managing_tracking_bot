package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	scheduler2 "rep_tracker/pkg/scheduler"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/postgres"
	gormio "gorm.io/gorm"

	"rep_tracker/internal/tasks"
	"rep_tracker/pkg/github"
	repgorm "rep_tracker/pkg/gorm"
	"rep_tracker/pkg/kafka"
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
		serviceName = "link_tracker"
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

	repo := repgorm.NewGormGlobalRepo(db)
	ghClient := github.NewGithubClient()
	writer, err := kafka.NewKafkaNotificationWriter(kafka.KafkaNotificationWriterConfig{
		Addr:         cfg.kafkaBrokers,
		Topic:        cfg.kafkaTopic,
		MaxAttempts:  cfg.kafkaMaxAttempts,
		BatchSize:    cfg.kafkaBatchSize,
		BatchTimeout: cfg.kafkaBatchTimeout,
		WriteTimeout: cfg.kafkaWriteTimeout,
	})
	if err != nil {
		zap.L().Fatal("kafka writer init failed", zap.Error(err))
	}
	defer writer.Close()

	checkFunc := tasks.GetCheckCommitsFunc(cfg.trackBatchSize, repo, ghClient, writer)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var scheduler scheduler2.Scheduler
	scheduler.Run(ctx, cfg.trackInterval, checkFunc)
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

type appConfig struct {
	dbDSN             string
	kafkaBrokers      []string
	kafkaTopic        string
	kafkaMaxAttempts  int
	kafkaBatchSize    int
	kafkaBatchTimeout time.Duration
	kafkaWriteTimeout time.Duration
	trackBatchSize    int
	trackInterval     time.Duration
}

func loadConfig() (appConfig, error) {
	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		return appConfig{}, fmt.Errorf("DB_DSN is required")
	}

	brokersRaw := os.Getenv("KAFKA_BROKERS")
	if brokersRaw == "" {
		return appConfig{}, fmt.Errorf("KAFKA_BROKERS is required")
	}
	brokers := splitAndTrim(brokersRaw)
	if len(brokers) == 0 {
		return appConfig{}, fmt.Errorf("KAFKA_BROKERS is empty")
	}

	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		return appConfig{}, fmt.Errorf("KAFKA_TOPIC is required")
	}

	return appConfig{
		dbDSN:             dbDSN,
		kafkaBrokers:      brokers,
		kafkaTopic:        topic,
		kafkaMaxAttempts:  getEnvInt("KAFKA_MAX_ATTEMPTS", 3),
		kafkaBatchSize:    getEnvInt("KAFKA_BATCH_SIZE", 100),
		kafkaBatchTimeout: time.Duration(getEnvInt("KAFKA_BATCH_TIMEOUT_MS", 1000)) * time.Millisecond,
		kafkaWriteTimeout: time.Duration(getEnvInt("KAFKA_WRITE_TIMEOUT_MS", 10000)) * time.Millisecond,
		trackBatchSize:    getEnvInt("TRACK_BATCH_SIZE", 100),
		trackInterval:     time.Duration(getEnvInt("TRACK_INTERVAL_SEC", 60)) * time.Second,
	}, nil
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

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
