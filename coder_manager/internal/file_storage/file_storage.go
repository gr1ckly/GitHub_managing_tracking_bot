package file_storage

import (
	"context"
	"io"
)

type SaveFileRequest struct {
	Key     string
	Content io.Reader
	Size    *int64
}

type DownloadFileRequest struct {
	Key string
}

type FileStorage interface {
	SaveFile(ctx context.Context, req SaveFileRequest) error
	DownloadFile(ctx context.Context, req DownloadFileRequest) (io.ReadCloser, *int64, error)
}
