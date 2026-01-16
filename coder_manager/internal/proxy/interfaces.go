package proxy

import (
	"context"
	"net/url"
	"time"

	dao "coder_manager/pkg/dao"
)

type SessionStore interface {
	GetSessionByToken(ctx context.Context, token string) (*dao.EditorSession, error)
	MarkSessionConsumed(ctx context.Context, sessionID int64, consumedAt time.Time) error
}

type URLRewriter interface {
	Rewrite(target *url.URL) error
}
