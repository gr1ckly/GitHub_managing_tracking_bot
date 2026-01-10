package repo

import (
	"context"
	"rep_tracker/pkg/gorm"

	"github.com/google/go-github/github"
)

type GlobalRepo interface {
	SaveCommitsAndUpdateNotification(ctx context.Context, commits ...*github.RepositoryCommit) error
	GetCountTrackingRepos(ctx context.Context) (int, error)
	GetTrackingRepos(ctx context.Context, offset int, limit int) ([]*gorm.Notification, error)
	GetToken(ctx context.Context, userId int) (string, error)
}
