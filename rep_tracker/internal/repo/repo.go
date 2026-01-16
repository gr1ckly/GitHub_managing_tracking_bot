package repo

import (
	"context"
	"rep_tracker/internal/server_model"
	"rep_tracker/pkg/gorm"

	"github.com/google/go-github/github"
)

type SchedulerRepo interface {
	SaveCommitsAndUpdateNotification(ctx context.Context, commits ...*github.RepositoryCommit) error
	GetCountTrackingRepos(ctx context.Context) (int, error)
	GetTrackingRepos(ctx context.Context, offset int, limit int) ([]*gorm.Notification, error)
	DisableTracking(ctx context.Context, notificationID int) error
	DisableTrackingForUser(ctx context.Context, userID int) error
}

type TokenRepo interface {
	GetToken(ctx context.Context, chatId string) (string, error)
}

type ServerRepo interface {
	AddNotificationRep(ctx context.Context, notification *server_model.TrackingRepo) error
	RemoveNotificationRep(ctx context.Context, notification *server_model.TrackingRepo) error
}
