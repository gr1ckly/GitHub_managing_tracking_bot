package notifier

import (
	"context"
	"time"
)

type FileEditNotification struct {
	FileID  int
	SavedAt time.Time
}

type Notifier interface {
	NotifyFileEdited(ctx context.Context, notification FileEditNotification) error
}
