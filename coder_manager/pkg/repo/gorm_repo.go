package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	internalrepo "coder_manager/internal/repo"
	dao "coder_manager/pkg/dao"

	"gorm.io/gorm"
)

type GormRepo struct {
	db *gorm.DB
}

func NewGormRepo(db *gorm.DB) *GormRepo {
	return &GormRepo{db: db}
}

func (r *GormRepo) GetUserToken(ctx context.Context, chatID string) (string, error) {
	var user dao.User
	if err := r.db.WithContext(ctx).Select("id").Where("chat_id = ?", chatID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", internalrepo.ErrUserNotFound
		}
		return "", err
	}
	var token dao.Token
	if err := r.db.WithContext(ctx).Where("user_id = ?", user.ID).Order("created_at desc, id desc").First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", internalrepo.ErrTokenNotFound
		}
		return "", err
	}
	return token.Token, nil
}

func (r *GormRepo) CreateEditorSession(ctx context.Context, params internalrepo.CreateSessionParams) (*dao.EditorSession, error) {
	var session *dao.EditorSession
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		created, err := r.createEditorSessionTx(ctx, tx, params)
		if err != nil {
			return err
		}
		session = created
		return nil
	})
	return session, err
}

func (r *GormRepo) createEditorSessionTx(ctx context.Context, tx *gorm.DB, params internalrepo.CreateSessionParams) (*dao.EditorSession, error) {
	var user dao.User
	if err := tx.WithContext(ctx).Where("chat_id = ?", params.UserChatID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, internalrepo.ErrUserNotFound
		}
		return nil, err
	}

	var repo dao.Repo
	if err := tx.WithContext(ctx).Where("url = ?", params.RepoURL).First(&repo).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			repo = dao.Repo{
				URL:   params.RepoURL,
				Owner: nullableString(params.RepoOwner),
				Name:  nullableString(params.RepoName),
			}
			if err := tx.WithContext(ctx).Create(&repo).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	var file dao.File
	if err := tx.WithContext(ctx).Where("repo_id = ? AND path = ?", repo.ID, params.Path).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			file = dao.File{
				RepoID: repo.ID,
				State:  dao.FileStateModified,
				Path:   params.Path,
			}
			if err := tx.WithContext(ctx).Create(&file).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	oneTimeToken := strings.TrimSpace(params.OneTimeToken)
	if oneTimeToken == "" {
		return nil, errors.New("one-time token is required")
	}

	session := dao.EditorSession{
		FileID:       file.ID,
		SessionURL:   params.SessionURL,
		WorkspaceID:  params.WorkspaceID,
		OneTimeToken: &oneTimeToken,
		Branch:       params.Branch,
		ForUser:      &user.ID,
		ExpiresAt:    params.ExpiresAt,
	}
	if err := tx.WithContext(ctx).Create(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *GormRepo) GetSessionByToken(ctx context.Context, token string) (*dao.EditorSession, error) {
	var session dao.EditorSession
	if err := r.db.WithContext(ctx).Preload("File").Preload("User").Where("one_time_token = ?", token).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, internalrepo.ErrSessionNotFound
		}
		return nil, err
	}
	return &session, nil
}

func (r *GormRepo) MarkSessionConsumed(ctx context.Context, sessionID int64, consumedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&dao.EditorSession{}).
		Where("id = ? AND consumed_at IS NULL", sessionID).
		Update("consumed_at", consumedAt).Error
}

func (r *GormRepo) ListExpiredUnsavedSessions(ctx context.Context, now time.Time, limit int) ([]*dao.EditorSession, error) {
	var sessions []*dao.EditorSession
	query := r.db.WithContext(ctx).Preload("File").Preload("File.Repo").Preload("User").
		Where("expires_at IS NOT NULL AND expires_at <= ? AND saved_at IS NULL", now).
		Order("expires_at ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *GormRepo) MarkSessionSaved(ctx context.Context, sessionID int64, savedAt time.Time, storageKey string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&dao.EditorSession{}).
			Where("id = ?", sessionID).
			Update("saved_at", savedAt).Error; err != nil {
			return err
		}
		if storageKey == "" {
			return nil
		}
		return tx.Model(&dao.File{}).
			Where("id = (SELECT file_id FROM editor_sessions WHERE id = ?)", sessionID).
			Update("storage_key", storageKey).Error
	})
}

func nullableString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	val := value
	return &val
}
