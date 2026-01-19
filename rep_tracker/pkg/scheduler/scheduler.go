package scheduler

import (
	"context"
	"sync/atomic"
	"time"
)

type ScheduledTask func(ctx context.Context)

type Scheduler struct{}

func (scheduler *Scheduler) Run(ctx context.Context, duration time.Duration, task ScheduledTask) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()
	var running atomic.Bool
	running.Store(false)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if running.Load() {
				continue
			}
			go func() {
				localCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				running.Store(true)
				task(localCtx)
				running.Store(false)
			}()
		}
	}
}
