package coder_service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"coder_manager/internal/coder_client"
	"coder_manager/internal/file_storage"
	"coder_manager/internal/notifier"
	"coder_manager/internal/repo"
	dao "coder_manager/pkg/dao"

	"github.com/coder/coder/v2/codersdk"
	"go.uber.org/zap"
)

type CreateEditorSessionRequest struct {
	Path       string
	ChatID     string
	TTLSeconds int64
	S3Key      string
}

type CreateEditorSessionResponse struct {
	OneTimeURL string
	SessionID  int64
	ExpiresAt  *time.Time
}

type SaveEditorSessionRequest struct {
	SessionID int64
}

type SaveEditorSessionResponse struct {
	StorageKey string
	SavedAt    *time.Time
}

var ErrInvalidRequest = errors.New("invalid request")
var ErrFileTooLarge = errors.New("file exceeds size limit")

type Service struct {
	repo             repo.CoderRepo
	coderClient      coder_client.CoderClient
	storage          file_storage.FileStorage
	notifier         notifier.Notifier
	proxyBaseURL     string
	coderAccessToken string
	tokenQueryParam  string
}

func NewService(repo repo.CoderRepo, coderClient coder_client.CoderClient, storage file_storage.FileStorage, notifier notifier.Notifier, proxyBaseURL string, coderAccessToken string, tokenQueryParam string) *Service {
	if strings.TrimSpace(tokenQueryParam) == "" {
		tokenQueryParam = codersdk.SessionTokenCookie
	}
	return &Service{
		repo:             repo,
		coderClient:      coderClient,
		storage:          storage,
		notifier:         notifier,
		proxyBaseURL:     strings.TrimRight(proxyBaseURL, "/"),
		coderAccessToken: coderAccessToken,
		tokenQueryParam:  strings.TrimSpace(tokenQueryParam),
	}
}

func (s *Service) CreateEditorSession(ctx context.Context, req CreateEditorSessionRequest) (*CreateEditorSessionResponse, error) {
	pathValue := strings.TrimSpace(req.Path)
	chatID := strings.TrimSpace(req.ChatID)
	s3Key := strings.TrimSpace(req.S3Key)
	if chatID == "" {
		return nil, fmt.Errorf("%w: chat_id is required", ErrInvalidRequest)
	}
	if s3Key == "" {
		return nil, fmt.Errorf("%w: s3_key is required", ErrInvalidRequest)
	}
	if req.TTLSeconds <= 0 {
		return nil, fmt.Errorf("%w: ttl_seconds must be positive", ErrInvalidRequest)
	}

	var (
		owner      string
		name       string
		repoURL    string
		fileReader io.ReadCloser
		size       *int64
		storageKey string
	)
	var err error
	storageKey, repoURL, err = normalizeS3Location(s3Key)
	if err != nil {
		zap.S().Errorw("parse s3 location failed", "s3_key", req.S3Key, "error", err)
		return nil, fmt.Errorf("%w: %s", ErrInvalidRequest, err.Error())
	}
	if pathValue == "" {
		pathValue = path.Base(storageKey)
	}
	if strings.TrimSpace(pathValue) == "" {
		return nil, fmt.Errorf("%w: path is required", ErrInvalidRequest)
	}
	fileReader, size, err = s.storage.DownloadFile(ctx, file_storage.DownloadFileRequest{
		Key: storageKey,
	})
	if err != nil {
		zap.S().Errorw("s3 download failed", "key", storageKey, "error", err)
		return nil, err
	}
	defer fileReader.Close()

	workspaceName := fmt.Sprintf("edit-%s-%s", owner, name)
	if strings.Trim(workspaceName, "-") == "" {
		workspaceName = "edit-session"
		if storageKey != "" {
			workspaceName = fmt.Sprintf("edit-s3-%s", path.Base(storageKey))
		}
	}
	workspaceID, err := s.coderClient.CreateWorkspace(ctx, coder_client.CreateWorkspaceRequest{
		Name: workspaceName,
	})
	if err != nil {
		zap.S().Errorw("create workspace failed", "owner", owner, "repo", name, "error", err)
		return nil, err
	}

	if err := s.coderClient.UploadFile(ctx, coder_client.UploadFileRequest{
		WorkspaceID: workspaceID,
		Path:        pathValue,
		Content:     fileReader,
		Size:        sizeOrUnknown(size),
	}); err != nil {
		zap.S().Errorw("coder upload failed", "workspace_id", workspaceID, "path", req.Path, "error", err)
		return nil, err
	}

	editorURL, err := s.coderClient.GetEditorURL(ctx, workspaceID)
	if err != nil {
		zap.S().Errorw("get editor url failed", "workspace_id", workspaceID, "error", err)
		return nil, err
	}

	oneTimeToken, err := generateToken(32)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(time.Duration(req.TTLSeconds) * time.Second)
	session, err := s.repo.CreateEditorSession(ctx, repo.CreateSessionParams{
		RepoURL:      repoURL,
		RepoOwner:    owner,
		RepoName:     name,
		Branch:       "",
		Path:         pathValue,
		StorageKey:   storageKey,
		UserChatID:   chatID,
		SessionURL:   editorURL,
		WorkspaceID:  workspaceID,
		ExpiresAt:    &expiresAt,
		OneTimeToken: oneTimeToken,
	})
	if err != nil {
		zap.S().Errorw("create editor session failed", "path", pathValue, "error", err)
		return nil, err
	}

	oneTimeURL, err := s.buildEditorURL(editorURL, oneTimeToken, session.ID)
	if err != nil {
		return nil, err
	}
	return &CreateEditorSessionResponse{
		OneTimeURL: oneTimeURL,
		SessionID:  session.ID,
		ExpiresAt:  &expiresAt,
	}, nil
}

func (s *Service) buildEditorURL(editorURL string, oneTimeToken string, sessionID int64) (string, error) {
	if s.proxyBaseURL != "" {
		return fmt.Sprintf("%s/edit/%s", s.proxyBaseURL, oneTimeToken), nil
	}
	if strings.TrimSpace(editorURL) == "" {
		return "", errors.New("editor url is empty")
	}
	if strings.TrimSpace(s.coderAccessToken) == "" {
		return "", errors.New("coder access token is required to build direct editor url")
	}
	target, err := url.Parse(editorURL)
	if err != nil {
		return "", fmt.Errorf("invalid editor url: %w", err)
	}
	query := target.Query()
	query.Set(s.tokenQueryParam, s.coderAccessToken)
	target.RawQuery = query.Encode()
	if err := s.repo.MarkSessionConsumed(context.Background(), sessionID, time.Now()); err != nil {
		zap.S().Warnw("mark session consumed failed for direct editor url", "session_id", sessionID, "error", err)
	}
	return target.String(), nil
}

func (s *Service) SaveEditorSession(ctx context.Context, req SaveEditorSessionRequest) (*SaveEditorSessionResponse, error) {
	if req.SessionID <= 0 {
		return nil, fmt.Errorf("%w: session_id must be positive", ErrInvalidRequest)
	}
	session, err := s.repo.GetSessionByID(ctx, req.SessionID)
	if err != nil {
		zap.S().Errorw("get session failed", "session_id", req.SessionID, "error", err)
		return nil, err
	}
	if session == nil || session.File.ID == 0 {
		return nil, repo.ErrSessionNotFound
	}
	if session.SavedAt != nil {
		storageKey := storageKeyForSession(session)
		return &SaveEditorSessionResponse{
			StorageKey: storageKey,
			SavedAt:    session.SavedAt,
		}, nil
	}
	storageKey := storageKeyForSession(session)
	savedAt := session.CreatedAt
	go func(session *dao.EditorSession, savedAt time.Time) {
		if _, err := s.saveSessionFile(context.Background(), session, savedAt, true); err != nil {
			zap.S().Errorw("async save session failed", "session_id", session.ID, "error", err)
		}
	}(session, savedAt)
	return &SaveEditorSessionResponse{
		StorageKey: storageKey,
		SavedAt:    &savedAt,
	}, nil
}

func (s *Service) HandleExpiredSessions(ctx context.Context, now time.Time, limit int) error {
	sessions, err := s.repo.ListExpiredUnsavedSessions(ctx, now, limit)
	if err != nil {
		zap.S().Errorw("list expired sessions failed", "error", err)
		return err
	}
	if len(sessions) == 0 {
		return nil
	}
	const maxConcurrency = 4
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	for _, session := range sessions {
		if session == nil || session.File.ID == 0 {
			continue
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(session *dao.EditorSession) {
			defer wg.Done()
			defer func() { <-sem }()
			if _, err := s.saveSessionFile(ctx, session, now, false); err != nil {
				zap.S().Errorw("save session failed", "session_id", session.ID, "error", err)
			}
		}(session)
	}
	wg.Wait()
	return nil
}

func (s *Service) HandleActiveSessions(ctx context.Context, now time.Time, limit int) error {
	sessions, err := s.repo.ListActiveUnsavedSessions(ctx, now, limit)
	if err != nil {
		zap.S().Errorw("list active sessions failed", "error", err)
		return err
	}
	if len(sessions) == 0 {
		return nil
	}
	const maxConcurrency = 4
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	for _, session := range sessions {
		if session == nil || session.File.ID == 0 {
			continue
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(session *dao.EditorSession) {
			defer wg.Done()
			defer func() { <-sem }()
			if _, err := s.saveSessionFile(ctx, session, now, false); err != nil {
				zap.S().Errorw("save active session failed", "session_id", session.ID, "error", err)
			}
		}(session)
	}
	wg.Wait()
	return nil
}

func sizeOrUnknown(size *int64) int64 {
	if size == nil {
		return -1
	}
	return *size
}

func generateToken(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func buildStorageKey(session *dao.EditorSession) string {
	owner := ""
	name := ""
	if session.File.Repo.Owner != nil {
		owner = *session.File.Repo.Owner
	}
	if session.File.Repo.Name != nil {
		name = *session.File.Repo.Name
	}
	repoSegment := strings.Trim(path.Join(owner, name), "/")
	if repoSegment == "" {
		repoSegment = "unknown-repo"
	}
	return fmt.Sprintf("edited/%s/%d/%s", repoSegment, session.ID, strings.TrimLeft(session.File.Path, "/"))
}

func storageKeyForSession(session *dao.EditorSession) string {
	if session.File.StorageKey != nil && strings.TrimSpace(*session.File.StorageKey) != "" {
		return *session.File.StorageKey
	}
	return buildStorageKey(session)
}

func (s *Service) saveSessionFile(ctx context.Context, session *dao.EditorSession, savedAt time.Time, expireSession bool) (string, error) {
	reader, err := s.coderClient.DownloadFile(ctx, coder_client.DownloadFileRequest{
		WorkspaceID: session.WorkspaceID,
		Path:        session.File.Path,
	})
	if err != nil {
		zap.S().Errorw("coder download failed", "session_id", session.ID, "path", session.File.Path, "error", err)
		return "", err
	}
	defer reader.Close()
	storageKey := storageKeyForSession(session)
	if err := s.storage.SaveFile(ctx, file_storage.SaveFileRequest{
		Key:     storageKey,
		Content: reader,
		Size:    nil,
	}); err != nil {
		zap.S().Errorw("s3 save failed", "session_id", session.ID, "key", storageKey, "error", err)
		return "", err
	}
	if err := s.repo.MarkSessionSaved(ctx, session.ID, savedAt, storageKey); err != nil {
		zap.S().Errorw("mark session saved failed", "session_id", session.ID, "error", err)
		return "", err
	}
	if expireSession {
		expiresAt := time.Now()
		if err := s.repo.MarkSessionExpired(ctx, session.ID, expiresAt); err != nil {
			zap.S().Errorw("mark session expired failed", "session_id", session.ID, "error", err)
		}
	}
	completedAt := time.Now()
	if err := s.notifier.NotifyFileEdited(ctx, notifier.FileEditNotification{
		FileID:  session.File.ID,
		SavedAt: completedAt,
	}); err != nil {
		zap.S().Errorw("notify failed", "session_id", session.ID, "error", err)
	}
	return storageKey, nil
}

func normalizeS3Location(location string) (string, string, error) {
	trimmed := strings.TrimSpace(location)
	if trimmed == "" {
		return "", "", errors.New("s3 location is empty")
	}
	if strings.HasPrefix(trimmed, "s3://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return "", "", err
		}
		bucket := strings.Trim(parsed.Host, "/")
		key := strings.TrimLeft(parsed.Path, "/")
		if bucket == "" || key == "" {
			return "", "", errors.New("invalid s3 url")
		}
		return key, fmt.Sprintf("s3://%s/%s", bucket, key), nil
	}
	key := strings.TrimLeft(trimmed, "/")
	if key == "" {
		return "", "", errors.New("s3 key is empty")
	}
	return key, fmt.Sprintf("s3://%s", key), nil
}
