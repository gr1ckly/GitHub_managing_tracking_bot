package coder_service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"coder_manager/internal/coder_client"
	"coder_manager/internal/file_storage"
	"coder_manager/internal/notifier"
	"coder_manager/internal/repo"
	dao "coder_manager/pkg/dao"
)

type CreateEditorSessionRequest struct {
	Repo       string
	Branch     string
	Path       string
	ChatID     string
	TTLSeconds int64
}

type CreateEditorSessionResponse struct {
	OneTimeURL string
	SessionID  int64
	ExpiresAt  *time.Time
}

var ErrInvalidRequest = errors.New("invalid request")

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
	if strings.TrimSpace(req.Repo) == "" || strings.TrimSpace(req.Path) == "" || strings.TrimSpace(req.ChatID) == "" {
		return nil, fmt.Errorf("%w: repo, path and chat_id are required", ErrInvalidRequest)
	}
	if req.TTLSeconds <= 0 {
		return nil, fmt.Errorf("%w: ttl_seconds must be positive", ErrInvalidRequest)
	}

	owner, name, repoURL, err := parseRepo(req.Repo)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRequest, err.Error())
	}

	token, err := s.repo.GetUserToken(ctx, req.ChatID)
	if err != nil {
		return nil, err
	}

	fileBytes, err := downloadGitHubFile(ctx, token, owner, name, req.Branch, req.Path)
	if err != nil {
		return nil, err
	}

	workspaceID, err := s.coderClient.CreateWorkspace(ctx, coder_client.CreateWorkspaceRequest{
		Name: fmt.Sprintf("edit-%s-%s", owner, name),
	})
	if err != nil {
		return nil, err
	}

	if err := s.coderClient.UploadFile(ctx, coder_client.UploadFileRequest{
		WorkspaceID: workspaceID,
		Path:        req.Path,
		Content:     bytes.NewReader(fileBytes),
		Size:        int64(len(fileBytes)),
	}); err != nil {
		return nil, err
	}

	editorURL, err := s.coderClient.GetEditorURL(ctx, workspaceID)
	if err != nil {
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
		Path:         req.Path,
		UserChatID:   req.ChatID,
		SessionURL:   editorURL,
		WorkspaceID:  workspaceID,
		ExpiresAt:    &expiresAt,
		OneTimeToken: oneTimeToken,
	})
	if err != nil {
		return nil, err
	}

	oneTimeURL := fmt.Sprintf("%s/edit/%s", s.proxyBaseURL, oneTimeToken)
	return &CreateEditorSessionResponse{
		OneTimeURL: oneTimeURL,
		SessionID:  session.ID,
		ExpiresAt:  &expiresAt,
	}, nil
}

func (s *Service) HandleExpiredSessions(ctx context.Context, now time.Time, limit int) error {
	sessions, err := s.repo.ListExpiredUnsavedSessions(ctx, now, limit)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if session == nil || session.File.ID == 0 {
			continue
		}
		content, err := s.coderClient.DownloadFile(ctx, coder_client.DownloadFileRequest{
			WorkspaceID: session.WorkspaceID,
			Path:        session.File.Path,
		})
		if err != nil {
			continue
		}
		storageKey := buildStorageKey(session)
		if err := s.storage.SaveFile(ctx, file_storage.SaveFileRequest{
			Key:     storageKey,
			Content: bytes.NewReader(content),
			Size:    int64(len(content)),
		}); err != nil {
			continue
		}
		if err := s.repo.MarkSessionSaved(ctx, session.ID, now, storageKey); err != nil {
			continue
		}
		userChatID := ""
		if session.User != nil {
			userChatID = session.User.ChatID
		}
		_ = s.notifier.NotifyFileEdited(ctx, notifier.FileEditNotification{
			UserChatID: userChatID,
			Repo:       session.File.Repo.URL,
			Branch:     session.Branch,
			Path:       session.File.Path,
			S3Key:      storageKey,
		})
	}
	return nil
}

func downloadGitHubFile(ctx context.Context, token string, owner string, repoName string, branch string, filePath string) ([]byte, error) {
	if strings.TrimSpace(branch) == "" {
		branch = "main"
	}
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repoName, branch, strings.TrimLeft(filePath, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("github download failed: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
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
