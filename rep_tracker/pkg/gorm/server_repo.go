package gorm

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"rep_tracker/internal/server_model"
	"rep_tracker/pkg/errs"
	"go.uber.org/zap"

	gormio "gorm.io/gorm"
)

type GormServerRepo struct {
	gorm *gormio.DB
}

func NewGormServerRepo(gorm *gormio.DB) *GormServerRepo {
	return &GormServerRepo{gorm: gorm}
}

func (r *GormServerRepo) AddNotificationRep(ctx context.Context, trackingRepo *server_model.TrackingRepo) error {
	if trackingRepo == nil {
		return fmt.Errorf("tracking repo is nil")
	}
	return r.gorm.WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		userID, err := resolveUserID(ctx, tx, trackingRepo.ChatID)
		if err != nil {
			return err
		}
		repoID, err := resolveRepoID(ctx, tx, trackingRepo.Link)
		if err != nil {
			return err
		}
		if err := ensureUserRepo(ctx, tx, userID, repoID); err != nil {
			return err
		}

		existing, err := gormio.G[Notification](tx).
			Where("user_id = ? AND repo_id = ?", userID, repoID).
			First(ctx)
		if err == nil {
			_, err = gormio.G[Notification](tx).
				Where("id = ?", existing.ID).
				Update(ctx, "enabled", true)
			return err
		}
		if !errors.Is(err, gormio.ErrRecordNotFound) {
			return err
		}

		newNotification := Notification{
			UserID:  userID,
			RepoID:  repoID,
			Enabled: true,
		}
		return gormio.G[Notification](tx).Create(ctx, &newNotification)
	})
}

func (r *GormServerRepo) RemoveNotificationRep(ctx context.Context, trackingRepo *server_model.TrackingRepo) error {
	if trackingRepo == nil {
		return fmt.Errorf("tracking repo is nil")
	}
	return r.gorm.WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		userID, err := resolveUserID(ctx, tx, trackingRepo.ChatID)
		if err != nil {
			return err
		}
		repoID, err := resolveRepoID(ctx, tx, trackingRepo.Link)
		if err != nil {
			return err
		}
		_, err = gormio.G[Notification](tx).
			Where("user_id = ? AND repo_id = ?", userID, repoID).
			Delete(ctx)
		return err
	})
}

func resolveUserID(ctx context.Context, tx *gormio.DB, chatID string) (int, error) {
	if chatID == "" {
		return 0, fmt.Errorf("user chat_id is required")
	}
	user, err := gormio.G[User](tx).
		Where("chat_id = ?", chatID).
		First(ctx)
	if err != nil {
		if errors.Is(err, gormio.ErrRecordNotFound) {
			return 0, errs.ErrUserNotFound
		}
		return 0, err
	}
	return user.ID, nil
}

func resolveRepoID(ctx context.Context, tx *gormio.DB, link string) (int, error) {
	if link == "" {
		return 0, fmt.Errorf("repo url is required")
	}
	
	zap.L().Debug("Looking for repository", 
		zap.String("link", link), 
		zap.String("linkWithGit", link+".git"))
	
	repo, err := gormio.G[Repo](tx).
		Where("url = ? OR url = ?", link, link+".git").
		First(ctx)
	
	if err == nil {
		zap.L().Debug("Repository found", 
			zap.String("link", link), 
			zap.Int("repoId", repo.ID))
		return repo.ID, nil
	}
	
	if !errors.Is(err, gormio.ErrRecordNotFound) {
		zap.L().Error("Database error while searching for repository", 
			zap.String("link", link), 
			zap.Error(err))
		return 0, err
	}
	
	zap.L().Warn("Repository not found, will create new one", 
		zap.String("link", link))

	owner, name := parseOwnerRepoFromLink(link)
	zap.L().Debug("Parsed owner and name", 
		zap.String("owner", owner), 
		zap.String("name", name))
	
	newRepo := Repo{
		URL:   link,
		Owner: ptrString(owner),
		Name:  ptrString(name),
	}
	
	if err := gormio.G[Repo](tx).Create(ctx, &newRepo); err != nil {
		zap.L().Error("Failed to create repository", 
			zap.String("link", link), 
			zap.Error(err))
		return 0, err
	}
	
	zap.L().Info("Repository created successfully", 
		zap.String("link", link), 
		zap.Int("repoId", newRepo.ID))
	
	return newRepo.ID, nil
}

func ensureUserRepo(ctx context.Context, tx *gormio.DB, userID int, repoID int) error {
	_, err := gormio.G[UserRepo](tx).
		Where("user_id = ? AND repo_id = ?", userID, repoID).
		First(ctx)
	if err == nil {
		return nil
	}
	if !errors.Is(err, gormio.ErrRecordNotFound) {
		return err
	}
	newLink := UserRepo{UserID: userID, RepoID: repoID}
	return gormio.G[UserRepo](tx).Create(ctx, &newLink)
}

func parseOwnerRepoFromLink(raw string) (string, string) {
	if raw == "" {
		return "", ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", ""
	}
	path := strings.Trim(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", ""
	}
	owner := parts[0]
	repo := strings.TrimSuffix(parts[1], ".git")
	return owner, repo
}
