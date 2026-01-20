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

	"github.com/google/go-github/v62/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type CreateEditorSessionRequest struct {
	Repo       string
	Branch     string
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

const maxGitHubFileSize = 1 << 30 // 1GB

type Service struct {
	repo             repo.CoderRepo
	coderClient      coder_client.CoderClient
	storage          file_storage.FileStorage
	notifier         notifier.Notifier
	proxyBaseURL     string
	coderAccessToken string
}

func NewService(repo repo.CoderRepo, coderClient coder_client.CoderClient, storage file_storage.FileStorage, notifier notifier.Notifier, proxyBaseURL string, coderAccessToken string) *Service {
	return &Service{
		repo:             repo,
		coderClient:      coderClient,
		storage:          storage,
		notifier:         notifier,
		proxyBaseURL:     strings.TrimRight(proxyBaseURL, "/"),
		coderAccessToken: coderAccessToken,
	}
}

func (s *Service) CreateEditorSession(ctx context.Context, req CreateEditorSessionRequest) (*CreateEditorSessionResponse, error) {
	pathValue := strings.TrimSpace(req.Path)
	chatID := strings.TrimSpace(req.ChatID)
	hasS3 := strings.TrimSpace(req.S3Key) != ""
	if chatID == "" {
		return nil, fmt.Errorf("%w: chat_id is required", ErrInvalidRequest)
	}
	if pathValue == "" && !hasS3 {
		return nil, fmt.Errorf("%w: path is required", ErrInvalidRequest)
	}
	if strings.TrimSpace(req.Repo) == "" && !hasS3 {
		return nil, fmt.Errorf("%w: repo or s3_key is required", ErrInvalidRequest)
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
	if strings.TrimSpace(req.S3Key) != "" {
		var err error
		storageKey, repoURL, err = normalizeS3Location(req.S3Key)
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
	} else {
		var err error
		owner, name, repoURL, err = parseRepo(req.Repo)
		if err != nil {
			zap.S().Errorw("parse repo failed", "repo", req.Repo, "error", err)
			return nil, fmt.Errorf("%w: %s", ErrInvalidRequest, err.Error())
		}

		token, err := s.repo.GetUserToken(ctx, req.ChatID)
		if err != nil {
			zap.S().Errorw("get user token failed", "chat_id", req.ChatID, "error", err)
			return nil, err
		}

		if err := ensureRepoAccess(ctx, token, owner, name); err != nil {
			zap.S().Errorw("repo access denied", "owner", owner, "repo", name, "error", err)
			return nil, err
		}

		fileReader, size, err = downloadGitHubFile(ctx, token, owner, name, req.Branch, pathValue)
		if err != nil {
			zap.S().Errorw("github download failed", "repo", req.Repo, "path", req.Path, "error", err)
			return nil, err
		}
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
		Branch:       req.Branch,
		Path:         pathValue,
		StorageKey:   storageKey,
		UserChatID:   chatID,
		SessionURL:   editorURL,
		WorkspaceID:  workspaceID,
		ExpiresAt:    &expiresAt,
		OneTimeToken: oneTimeToken,
	})
	if err != nil {
		zap.S().Errorw("create editor session failed", "repo", req.Repo, "path", req.Path, "error", err)
		return nil, err
	}

	oneTimeURL := fmt.Sprintf("%s/edit/%s", s.proxyBaseURL, oneTimeToken)
	return &CreateEditorSessionResponse{
		OneTimeURL: oneTimeURL,
		SessionID:  session.ID,
		ExpiresAt:  &expiresAt,
	}, nil
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

func downloadGitHubFile(ctx context.Context, token string, owner string, repoName string, branch string, filePath string) (io.ReadCloser, *int64, error) {
	if strings.TrimSpace(branch) == "" {
		branch = "main"
	}
	client := newGitHubClient(ctx, token)
	ref := branch
	path := strings.TrimLeft(filePath, "/")
	file, _, _, err := client.Repositories.GetContents(ctx, owner, repoName, path, &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return nil, nil, err
	}
	if file == nil {
		return nil, nil, errors.New("path does not point to a file")
	}
	if file.GetType() == "dir" {
		return nil, nil, errors.New("path points to a directory")
	}
	var sizePtr *int64
	if size := file.GetSize(); size > 0 {
		sizeValue := int64(size)
		if sizeValue > maxGitHubFileSize {
			return nil, nil, ErrFileTooLarge
		}
		sizePtr = &sizeValue
	}
	reader, _, err := client.Repositories.DownloadContents(ctx, owner, repoName, path, &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return nil, nil, err
	}
	if sizePtr == nil {
		return limitReadCloser(reader, maxGitHubFileSize), nil, nil
	}
	return reader, sizePtr, nil
}

func newGitHubClient(ctx context.Context, token string) *github.Client {
	if strings.TrimSpace(token) == "" {
		return github.NewClient(nil)
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return github.NewClient(oauth2.NewClient(ctx, ts))
}

func ensureRepoAccess(ctx context.Context, token string, owner string, repoName string) error {
	client := newGitHubClient(ctx, token)
	_, _, err := client.Repositories.Get(ctx, owner, repoName)
	return err
}

func limitReadCloser(reader io.ReadCloser, limit int64) io.ReadCloser {
	return &limitedReadCloser{
		reader: reader,
		limit:  limit,
	}
}

type limitedReadCloser struct {
	reader io.ReadCloser
	limit  int64
	read   int64
}

func (l *limitedReadCloser) Read(p []byte) (int, error) {
	if l.read >= l.limit {
		return 0, ErrFileTooLarge
	}
	if int64(len(p)) > l.limit-l.read {
		p = p[:l.limit-l.read]
	}
	n, err := l.reader.Read(p)
	l.read += int64(n)
	if err == io.EOF {
		return n, err
	}
	if l.read >= l.limit {
		return n, ErrFileTooLarge
	}
	return n, err
}

func (l *limitedReadCloser) Close() error {
	return l.reader.Close()
}

func sizeOrUnknown(size *int64) int64 {
	if size == nil {
		return -1
	}
	return *size
}

func parseRepo(input string) (string, string, string, error) {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		parsed, err := url.Parse(input)
		if err != nil {
			return "", "", "", err
		}
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(parts) < 2 {
			return "", "", "", errors.New("invalid repo URL")
		}
		return parts[0], strings.TrimSuffix(parts[1], ".git"), fmt.Sprintf("%s://%s/%s/%s", parsed.Scheme, parsed.Host, parts[0], strings.TrimSuffix(parts[1], ".git")), nil
	}
	parts := strings.Split(strings.Trim(input, "/"), "/")
	if len(parts) != 2 {
		return "", "", "", errors.New("repo must be in owner/name or url form")
	}
	return parts[0], parts[1], fmt.Sprintf("https://github.com/%s/%s", parts[0], parts[1]), nil
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
