package scheduler

import (
	"context"
	"rep_tracker/internal/notification"
	"rep_tracker/internal/repo"
	"rep_tracker/pkg/github"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type Scheduler struct {
	repo               repo.GlobalRepo
	ghClient           *github.GithubClient
	notificationWriter notification.NotificationWriter
}

func NewScheduler(repo repo.GlobalRepo, notificationWriter notification.NotificationWriter, ghClient *github.GithubClient) *Scheduler {
	return &Scheduler{
		repo:               repo,
		ghClient:           ghClient,
		notificationWriter: notificationWriter,
	}
}

func (scheduler *Scheduler) Run(ctx context.Context, duration time.Duration, task func(ctx context.Context, repo repo.GlobalRepo, ghClient *github.GithubClient, writer notification.NotificationWriter) error) {
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
				err := task(localCtx, scheduler.repo, scheduler.ghClient, scheduler.notificationWriter)
				if err != nil {
					zap.S().Error("task failed", zap.Error(err))
				}
				running.Store(false)
			}()
		}
	}
}

func GetCheckCommitsTask(batchSize int) func(ctx context.Context, repo repo.GlobalRepo, ghClient *github.GithubClient, writer notification.NotificationWriter) {
	return func(ctx context.Context, repo repo.GlobalRepo, ghClient *github.GithubClient, writer notification.NotificationWriter) {
		repoCount, err := repo.GetCountTrackingRepos(ctx)
		if err != nil {
			zap.S().Warn("repo count tracking repos failed", zap.Error(err))
			return
		}
		offset := 0
		var wg sync.WaitGroup
		for offset < repoCount {
			go func() {
				localCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				wg.Add(1)
				defer wg.Done()
				currRepos, currErr := repo.GetTrackingRepos(localCtx, offset, batchSize)
				if currErr != nil {
					zap.S().Warnf("get tracking repos failed (offset - %v, limit - %v): %v", offset, batchSize, currErr)
				}
				for _, currRepo := range currRepos {
					token, currErr := repo.GetToken(localCtx, currRepo.User.ID)
					if currErr != nil {
						zap.S().Warnf("get token for user (user_is: %v) failed: %v", currRepo.User.ID, currErr)
						continue
					}
					newCommits, currErr := ghClient.GetCommitsSince(localCtx, token, currRepo.Repo.URL, currRepo.LastCommitEntity.CreatedAt)
					if currErr != nil {
						zap.S().Warnf("get commits for repo - %v since (%v) failed: %v", currRepo.Repo.URL, currRepo.LastCommitEntity.CreatedAt, currErr)
					}
					zap.S().Infof("get commits for repo - %v since (%v): %v", currRepo.Repo.URL, currRepo.LastCommitEntity.CreatedAt, len(newCommits))
					if len(newCommits) > 0 {
						err = repo.SaveCommitsAndUpdateNotification(ctx, newCommits...)
						if err != nil {
							zap.S().Warnf("save commits failed: %v", err)
						}
					}
				}
			}()
			offset += batchSize
		}
		wg.Wait()
	}
}
