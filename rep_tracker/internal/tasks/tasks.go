package tasks

import (
	"context"
	"errors"
	"rep_tracker/internal/notification"
	"rep_tracker/internal/repo"
	"rep_tracker/pkg/dto"
	"rep_tracker/pkg/errs"
	"rep_tracker/pkg/github"
	"sync"
	"time"

	"go.uber.org/zap"
)

func GetCheckCommitsFunc(batchSize int, repo repo.SchedulerRepo, tokenRepo repo.TokenRepo, ghClient *github.GithubClient, writer notification.NotificationWriter) func(ctx context.Context) {
	return func(ctx context.Context) {
		repoCount, err := repo.GetCountTrackingRepos(ctx)
		if err != nil {
			zap.S().Warn("repo count tracking repos failed", zap.Error(err))
			return
		}
		offset := 0
		var wg sync.WaitGroup
		for offset < repoCount {
			localOffset := offset
			wg.Add(1)
			go func() {
				localCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				defer wg.Done()
				currRepos, currErr := repo.GetTrackingRepos(localCtx, localOffset, batchSize)
				if currErr != nil {
					zap.S().Warnf("get tracking repos failed (offset - %v, limit - %v): %v", localOffset, batchSize, currErr)
				}
				for _, currRepo := range currRepos {
					token, currErr := tokenRepo.GetToken(localCtx, currRepo.User.ChatID)
					if currErr != nil {
						zap.S().Warnf("get token for user (user_is: %v) failed: %v", currRepo.User.ID, currErr)
						continue
					}
					exists, currErr := ghClient.CheckRepo(localCtx, token, currRepo.Repo.URL)
					if currErr != nil {
						if errors.Is(currErr, errs.ErrInvalidToken) {
							disableErr := repo.DisableTrackingForUser(localCtx, currRepo.User.ID)
							if disableErr != nil {
								zap.S().Warnf("disable tracking for user (user_id: %v) failed: %v", currRepo.User.ID, disableErr)
							}
							notifyErr := writer.WriteNotification(ctx, currRepo.User.ChatID, &dto.ChangingDTO{
								Link:      currRepo.Repo.URL,
								Author:    "system",
								Title:     "Invalid PAT token. Tracking disabled until you. Refresh your token.",
								UpdatedAt: time.Now().UTC(),
							})
							if notifyErr != nil {
								zap.S().Warnf("write invalid token notification for user (user_id: %v) failed: %v", currRepo.User.ID, notifyErr)
							}
							continue
						}
						zap.S().Warnf("check repo - %v failed: %v", currRepo.Repo.URL, currErr)
						continue
					}
					if !exists {
						disableErr := repo.DisableTracking(localCtx, currRepo.ID)
						if disableErr != nil {
							zap.S().Warnf("disable tracking for repo - %v failed: %v", currRepo.Repo.URL, disableErr)
						}
						notifyErr := writer.WriteNotification(ctx, currRepo.User.ChatID, &dto.ChangingDTO{
							Link:      currRepo.Repo.URL,
							Author:    "system",
							Title:     "Repository deleted or access lost. Tracking disabled.",
							UpdatedAt: time.Now().UTC(),
						})
						if notifyErr != nil {
							zap.S().Warnf("write deletion notification for repo - %v failed: %v", currRepo.Repo.URL, notifyErr)
						}
						continue
					}
					var lastCommitTime time.Time
					if currRepo.LastCommitEntity != nil {
						lastCommitTime = currRepo.LastCommitEntity.CreatedAt
					} else {
						lastCommitTime = currRepo.CreatedAt
					}
					newCommits, currErr := ghClient.GetCommitsSince(localCtx, token, currRepo.Repo.URL, lastCommitTime)
					if currErr != nil {
						if errors.Is(currErr, errs.ErrInvalidToken) {
							disableErr := repo.DisableTrackingForUser(localCtx, currRepo.User.ID)
							if disableErr != nil {
								zap.S().Warnf("disable tracking for user (user_id: %v) failed: %v", currRepo.User.ID, disableErr)
							}
							notifyErr := writer.WriteNotification(ctx, currRepo.User.ChatID, &dto.ChangingDTO{
								Link:      currRepo.Repo.URL,
								Author:    "system",
								Title:     "Invalid PAT token. Tracking disabled until you обновите токен.",
								UpdatedAt: time.Now().UTC(),
							})
							if notifyErr != nil {
								zap.S().Warnf("write invalid token notification for user (user_id: %v) failed: %v", currRepo.User.ID, notifyErr)
							}
							continue
						}
						zap.S().Warnf("get commits for repo - %v since (%v) failed: %v", currRepo.Repo.URL, lastCommitTime, currErr)
					}
					zap.S().Infof("get commits for repo - %v since (%v): %v", currRepo.Repo.URL, lastCommitTime, len(newCommits))
					if len(newCommits) > 0 {
						err = repo.SaveCommitsAndUpdateNotification(ctx, newCommits...)
						if err != nil {
							zap.S().Warnf("save commits failed: %v", err)
						}
						for _, newCommit := range newCommits {
							zap.L().Info("Sending notification to user", 
								zap.String("chat_id", currRepo.User.ChatID),
								zap.String("commit_url", newCommit.GetCommit().GetURL()),
								zap.String("commit_sha", newCommit.GetCommit().GetSHA()),
								zap.String("repo_url", currRepo.Repo.URL))
							
							currErr = writer.WriteNotification(ctx, currRepo.User.ChatID, dto.ConvertRepositoryCommitToDTO(newCommit))
							if currErr != nil {
								zap.L().Error("Failed to send notification about commit", 
									zap.String("commit_url", newCommit.GetCommit().GetURL()),
									zap.String("commit_sha", newCommit.GetCommit().GetSHA()),
									zap.String("chat_id", currRepo.User.ChatID),
									zap.Error(currErr))
							} else {
								zap.L().Info("Successfully sent notification", 
									zap.String("commit_url", newCommit.GetCommit().GetURL()),
									zap.String("chat_id", currRepo.User.ChatID))
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
