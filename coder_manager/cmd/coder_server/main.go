package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"coder_manager/internal/coder_client"
	"coder_manager/internal/coder_service"
	"coder_manager/internal/file_storage"
	"coder_manager/internal/grpc_server"
	"coder_manager/internal/notifier"
	"coder_manager/internal/repo"
	"coder_manager/internal/tasks"
	dao "coder_manager/pkg/dao"
	"coder_manager/pkg/proto"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		logger.Fatal("DB_DSN is required")
	}
	proxyBaseURL := os.Getenv("PROXY_BASE_URL")
	if proxyBaseURL == "" {
		logger.Fatal("PROXY_BASE_URL is required")
	}
	coderAccessToken := os.Getenv("CODER_ACCESS_TOKEN")

	db, err := gorm.Open(postgres.Open(dbDSN), &gorm.Config{})
	if err != nil {
		logger.Fatal("db connection failed", zap.Error(err))
	}
	if err := db.AutoMigrate(&dao.EditorSession{}, &dao.File{}, &dao.Repo{}, &dao.User{}, &dao.Token{}); err != nil {
		logger.Fatal("db migrate failed", zap.Error(err))
	}

	repoStore := repo.NewGormRepo(db)
	storage, err := file_storage.NewS3StorageFromEnv()
	if err != nil {
		logger.Fatal("s3 init failed", zap.Error(err))
	}
	notifyClient := notifier.NewStubNotifier()

	coderClient, err := coder_client.NewSDKClientFromEnv()
	if err != nil {
		logger.Fatal("coder client init failed", zap.Error(err))
	}
	service := coder_service.NewService(repoStore, coderClient, storage, notifyClient, proxyBaseURL, coderAccessToken)

	saver := tasks.NewSessionSaver(service, 30*time.Second, 100)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go saver.Run(ctx)

	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "9090"
	}
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Fatal("listen error", zap.Error(err))
	}

	server := grpc.NewServer()
	proto.RegisterCoderManagerServiceServer(server, grpc_server.NewCoderManagerServer(service))
	reflection.Register(server)

	logger.Info("coder manager grpc listening", zap.String("port", port))
	if err := server.Serve(listener); err != nil {
		logger.Fatal("server error", zap.Error(err))
	}
}
