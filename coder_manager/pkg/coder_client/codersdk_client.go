package coder_client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	internalclient "coder_manager/internal/coder_client"

	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/codersdk/workspacesdk"
	"github.com/google/uuid"
)

type SDKClient struct {
	client                  *codersdk.Client
	templateID              *uuid.UUID
	templateVersionID       *uuid.UUID
	templateVersionPresetID *uuid.UUID
	user                    string
	editorAppSlug           string
	agentName               string
	workspaceReadyTimeout   time.Duration
}

func NewSDKClient(cfg Config) (*SDKClient, error) {
	coderURL := strings.TrimSpace(cfg.URL)
	if coderURL == "" {
		return nil, errors.New("coder url is required")
	}
	parsedURL, err := url.Parse(coderURL)
	if err != nil {
		return nil, fmt.Errorf("invalid CODER_URL: %w", err)
	}
	token := strings.TrimSpace(cfg.AccessToken)
	if token == "" {
		return nil, errors.New("coder access token is required")
	}
	var templateID *uuid.UUID
	if raw := strings.TrimSpace(cfg.TemplateID); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid CODER_TEMPLATE_ID: %w", err)
		}
		templateID = &parsed
	}
	var templateVersionID *uuid.UUID
	if raw := strings.TrimSpace(cfg.TemplateVersionID); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid CODER_TEMPLATE_VERSION_ID: %w", err)
		}
		templateVersionID = &parsed
	}
	var templateVersionPresetID *uuid.UUID
	if raw := strings.TrimSpace(cfg.TemplateVersionPresetID); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid CODER_TEMPLATE_VERSION_PRESET_ID: %w", err)
		}
		templateVersionPresetID = &parsed
	}
	user := strings.TrimSpace(cfg.User)
	if user == "" {
		user = codersdk.Me
	}
	editorAppSlug := strings.TrimSpace(cfg.EditorAppSlug)
	agentName := strings.TrimSpace(cfg.AgentName)
	timeout := cfg.WorkspaceReadyTimeout
	if timeout == 0 {
		timeout = 2 * time.Minute
	}
	client := codersdk.New(parsedURL, codersdk.WithSessionToken(token))
	return &SDKClient{
		client:                  client,
		templateID:              templateID,
		templateVersionID:       templateVersionID,
		templateVersionPresetID: templateVersionPresetID,
		user:                    user,
		editorAppSlug:           editorAppSlug,
		agentName:               agentName,
		workspaceReadyTimeout:   timeout,
	}, nil
}

func (c *SDKClient) CreateWorkspace(ctx context.Context, req internalclient.CreateWorkspaceRequest) (string, error) {
	if c.client == nil {
		return "", errors.New("coder client is nil")
	}
	name := sanitizeWorkspaceName(req.Name)
	if name == "" {
		name = "editor"
	}
	if c.templateID == nil && c.templateVersionID == nil {
		return "", errors.New("CODER_TEMPLATE_ID or CODER_TEMPLATE_VERSION_ID must be set")
	}
	request := codersdk.CreateWorkspaceRequest{
		Name: name,
	}
	if c.templateVersionID != nil {
		request.TemplateVersionID = *c.templateVersionID
	} else if c.templateID != nil {
		request.TemplateID = *c.templateID
	}
	if c.templateVersionPresetID != nil {
		request.TemplateVersionPresetID = *c.templateVersionPresetID
	}
	workspace, err := c.client.CreateUserWorkspace(ctx, c.user, request)
	if err != nil {
		return "", err
	}
	if err := c.ensureWorkspaceReady(ctx, workspace.ID); err != nil {
		return "", err
	}
	return workspace.ID.String(), nil
}

func (c *SDKClient) UploadFile(ctx context.Context, req internalclient.UploadFileRequest) error {
	if c.client == nil {
		return errors.New("coder client is nil")
	}
	if req.Content == nil {
		return errors.New("file content is nil")
	}
	workspaceID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace id: %w", err)
	}
	agent, err := c.getWorkspaceAgent(ctx, workspaceID)
	if err != nil {
		return err
	}
	conn, err := c.dialAgent(ctx, agent.ID)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.WriteFile(ctx, req.Path, req.Content)
}

func (c *SDKClient) GetEditorURL(ctx context.Context, workspaceID string) (string, error) {
	if c.client == nil {
		return "", errors.New("coder client is nil")
	}
	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return "", fmt.Errorf("invalid workspace id: %w", err)
	}
	agent, err := c.getWorkspaceAgent(ctx, workspaceUUID)
	if err != nil {
		return "", err
	}
	if len(agent.Apps) == 0 {
		return "", errors.New("workspace agent has no apps")
	}
	for _, app := range agent.Apps {
		if app.URL == "" {
			continue
		}
		if c.editorAppSlug != "" && !strings.EqualFold(app.Slug, c.editorAppSlug) && !strings.EqualFold(app.DisplayName, c.editorAppSlug) {
			continue
		}
		return app.URL, nil
	}
	if agent.Apps[0].URL == "" {
		return "", errors.New("workspace app url is empty")
	}
	return agent.Apps[0].URL, nil
}

func (c *SDKClient) DownloadFile(ctx context.Context, req internalclient.DownloadFileRequest) (io.ReadCloser, error) {
	if c.client == nil {
		return nil, errors.New("coder client is nil")
	}
	workspaceID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace id: %w", err)
	}
	agent, err := c.getWorkspaceAgent(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	conn, err := c.dialAgent(ctx, agent.ID)
	if err != nil {
		return nil, err
	}
	reader, _, err := conn.ReadFile(ctx, req.Path, 0, -1)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &agentReader{
		reader: reader,
		conn:   conn,
	}, nil
}

type agentReader struct {
	reader io.ReadCloser
	conn   workspacesdk.AgentConn
}

func (r *agentReader) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *agentReader) Close() error {
	if r.reader != nil {
		_ = r.reader.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

func (c *SDKClient) ensureWorkspaceReady(ctx context.Context, workspaceID uuid.UUID) error {
	deadline := time.Now().Add(c.workspaceReadyTimeout)
	for {
		workspace, err := c.client.Workspace(ctx, workspaceID)
		if err != nil {
			return err
		}
		status := workspace.LatestBuild.Status
		if status == codersdk.WorkspaceStatusRunning {
			return nil
		}
		if status == codersdk.WorkspaceStatusFailed || status == codersdk.WorkspaceStatusCanceled {
			return fmt.Errorf("workspace build failed: %s", status)
		}
		if time.Now().After(deadline) {
			return errors.New("workspace did not become ready in time")
		}
		time.Sleep(2 * time.Second)
	}
}

func (c *SDKClient) getWorkspaceAgent(ctx context.Context, workspaceID uuid.UUID) (codersdk.WorkspaceAgent, error) {
	workspace, err := c.client.Workspace(ctx, workspaceID)
	if err != nil {
		return codersdk.WorkspaceAgent{}, err
	}
	if workspace.LatestBuild.Transition != codersdk.WorkspaceTransitionStart && workspace.LatestBuild.Status == codersdk.WorkspaceStatusStopped {
		if _, err := c.client.CreateWorkspaceBuild(ctx, workspaceID, codersdk.CreateWorkspaceBuildRequest{
			Transition: codersdk.WorkspaceTransitionStart,
		}); err != nil {
			return codersdk.WorkspaceAgent{}, err
		}
		if err := c.ensureWorkspaceReady(ctx, workspaceID); err != nil {
			return codersdk.WorkspaceAgent{}, err
		}
		workspace, err = c.client.Workspace(ctx, workspaceID)
		if err != nil {
			return codersdk.WorkspaceAgent{}, err
		}
	}
	var agents []codersdk.WorkspaceAgent
	for _, resource := range workspace.LatestBuild.Resources {
		agents = append(agents, resource.Agents...)
	}
	if len(agents) == 0 {
		return codersdk.WorkspaceAgent{}, errors.New("workspace has no agents")
	}
	if c.agentName != "" {
		for _, agent := range agents {
			if agent.Name == c.agentName || agent.ID.String() == c.agentName {
				return agent, nil
			}
		}
		return codersdk.WorkspaceAgent{}, fmt.Errorf("agent %q not found", c.agentName)
	}
	if len(agents) == 1 {
		return agents[0], nil
	}
	return agents[0], nil
}

func (c *SDKClient) dialAgent(ctx context.Context, agentID uuid.UUID) (workspacesdk.AgentConn, error) {
	wsClient := workspacesdk.New(c.client)
	conn, err := wsClient.DialAgent(ctx, agentID, &workspacesdk.DialAgentOptions{})
	if err != nil {
		return nil, err
	}
	if !conn.AwaitReachable(ctx) {
		conn.Close()
		return nil, errors.New("workspace agent not reachable")
	}
	return conn, nil
}

func sanitizeWorkspaceName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		if r == '_' || r == ' ' {
			return '-'
		}
		return -1
	}, name)
	name = strings.Trim(name, "-")
	if len(name) > 32 {
		name = name[:32]
	}
	return name
}
