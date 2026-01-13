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

type FileStorage interface {
	SaveFile(ctx context.Context, req SaveFileRequest) error
}
