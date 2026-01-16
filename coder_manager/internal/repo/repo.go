package repo

import (
	"context"
	"errors"
	"time"

	dao "coder_manager/pkg/dao"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrTokenNotFound   = errors.New("token not found")
	ErrSessionNotFound = errors.New("session not found")
)

type CreateSessionParams struct {
	RepoURL      string
	RepoOwner    string
	RepoName     string
	Branch       string
	Path         string
	StorageKey   string
	UserChatID   string
	SessionURL   string
	WorkspaceID  string
	ExpiresAt    *time.Time
	OneTimeToken string
}

type CoderRepo interface {
	GetUserToken(ctx context.Context, chatID string) (string, error)
	CreateEditorSession(ctx context.Context, params CreateSessionParams) (*dao.EditorSession, error)
	GetSessionByToken(ctx context.Context, token string) (*dao.EditorSession, error)
	MarkSessionConsumed(ctx context.Context, sessionID int64, consumedAt time.Time) error
	ListExpiredUnsavedSessions(ctx context.Context, now time.Time, limit int) ([]*dao.EditorSession, error)
	MarkSessionSaved(ctx context.Context, sessionID int64, savedAt time.Time, storageKey string) error
}
