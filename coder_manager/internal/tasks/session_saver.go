package tasks

import (
	"context"
	"time"

	"coder_manager/internal/coder_service"
)

type SessionSaver struct {
	service  *coder_service.Service
	interval time.Duration
	limit    int
}

func NewSessionSaver(service *coder_service.Service, interval time.Duration, limit int) *SessionSaver {
	return &SessionSaver{
		service:  service,
		interval: interval,
		limit:    limit,
	}
}

func (s *SessionSaver) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			_ = s.service.HandleExpiredSessions(ctx, now, s.limit)
		}
	}
}
