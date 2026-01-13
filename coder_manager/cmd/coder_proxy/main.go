package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"coder_manager/internal/repo"

	dao "coder_manager/pkg/dao"

	"github.com/coder/coder/v2/codersdk"
	"go.uber.org/zap"
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
	coderToken := os.Getenv("CODER_ACCESS_TOKEN")
	if coderToken == "" {
		logger.Fatal("CODER_ACCESS_TOKEN is required")
	}
	tokenParam := os.Getenv("CODER_TOKEN_QUERY_PARAM")
	if tokenParam == "" {
		tokenParam = codersdk.SessionTokenCookie
	}

	db, err := gorm.Open(postgres.Open(dbDSN), &gorm.Config{})
	if err != nil {
		logger.Fatal("db connection failed", zap.Error(err))
	}
	if err := db.AutoMigrate(&dao.EditorSession{}, &dao.File{}, &dao.Repo{}, &dao.User{}, &dao.Token{}); err != nil {
		logger.Fatal("db migrate failed", zap.Error(err))
	}

	repoStore := repo.NewGormRepo(db)

	mux := http.NewServeMux()
	mux.HandleFunc("/edit/", func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.URL.Path, "/edit/")
		if token == "" {
			http.Error(w, "missing token", http.StatusBadRequest)
			return
		}
		session, err := repoStore.GetSessionByToken(r.Context(), token)
		if err != nil {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		if session.ConsumedAt != nil {
			http.Error(w, "session already used", http.StatusGone)
			return
		}
		if session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt) {
			http.Error(w, "session expired", http.StatusGone)
			return
		}
		target, err := url.Parse(session.SessionURL)
		if err != nil {
			http.Error(w, "invalid session url", http.StatusInternalServerError)
			return
		}
		query := target.Query()
		query.Set(tokenParam, coderToken)
		target.RawQuery = query.Encode()

		_ = repoStore.MarkSessionConsumed(context.Background(), session.ID, time.Now())
		http.Redirect(w, r, target.String(), http.StatusTemporaryRedirect)
	})

	port := os.Getenv("PROXY_PORT")
	if port == "" {
		port = "8082"
	}
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	logger.Info("coder proxy listening", zap.String("port", port))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("server error", zap.Error(err))
	}
}
