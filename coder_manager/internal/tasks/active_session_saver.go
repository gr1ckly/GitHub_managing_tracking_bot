package tasks

import (
	"context"
	"time"

	"coder_manager/internal/coder_service"
)

type ActiveSessionSaver struct {
	service  *coder_service.Service
	interval time.Duration
	limit    int
}

func NewActiveSessionSaver(service *coder_service.Service, interval time.Duration, limit int) *ActiveSessionSaver {
	return &ActiveSessionSaver{
		service:  service,
		interval: interval,
		limit:    limit,
	}
}

func (s *ActiveSessionSaver) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			_ = s.service.HandleActiveSessions(ctx, now, s.limit)
		}
	}
}
