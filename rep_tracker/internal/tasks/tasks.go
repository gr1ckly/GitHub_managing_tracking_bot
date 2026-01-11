package tasks

import (
	"context"
	"rep_tracker/internal/notification"
	"rep_tracker/internal/repo"
	"rep_tracker/pkg/dto"
	"rep_tracker/pkg/github"
	"sync"

	"go.uber.org/zap"
)

func GetCheckCommitsFunc(batchSize int, repo repo.GlobalRepo, ghClient *github.GithubClient, writer notification.NotificationWriter) func(ctx context.Context) {
	return func(ctx context.Context) {
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
					lastCommitTime := time.Time{}
					if currRepo.LastCommitEntity != nil {
						lastCommitTime = currRepo.LastCommitEntity.CreatedAt
					}
					newCommits, currErr := ghClient.GetCommitsSince(localCtx, token, currRepo.Repo.URL, lastCommitTime)
					if currErr != nil {
						zap.S().Warnf("get commits for repo - %v since (%v) failed: %v", currRepo.Repo.URL, lastCommitTime, currErr)
					}
					zap.S().Infof("get commits for repo - %v since (%v): %v", currRepo.Repo.URL, lastCommitTime, len(newCommits))
					if len(newCommits) > 0 {
						err = repo.SaveCommitsAndUpdateNotification(ctx, newCommits...)
						if err != nil {
							zap.S().Warnf("save commits failed: %v", err)
						}
						for _, newCommit := range newCommits {
							currErr = writer.WriteNotification(ctx, currRepo.User.ChatID, dto.ConvertRepositoryCommitToDTO(newCommit))
							if err != nil {
								zap.S().Warnf("write notification about commit (%v %v) failed: %v", newCommit.GetCommit().GetURL(), newCommit.GetCommit().GetSHA(), err)
							}
						}
					}
				}
			}()
			offset += batchSize
		}
		wg.Wait()
	}
}
