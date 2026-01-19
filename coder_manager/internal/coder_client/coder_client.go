package coder_client

import (
	"context"
	"io"
)

type CreateWorkspaceRequest struct {
	Name string
}

type UploadFileRequest struct {
	WorkspaceID string
	Path        string
	Content     io.Reader
	Size        int64
}

type DownloadFileRequest struct {
	WorkspaceID string
	Path        string
}

type CoderClient interface {
	CreateWorkspace(ctx context.Context, req CreateWorkspaceRequest) (string, error)
	UploadFile(ctx context.Context, req UploadFileRequest) error
	GetEditorURL(ctx context.Context, workspaceID string) (string, error)
	DownloadFile(ctx context.Context, req DownloadFileRequest) (io.ReadCloser, error)
}
