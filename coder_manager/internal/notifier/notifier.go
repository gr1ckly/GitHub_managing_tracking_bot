package notifier

import "context"

type FileEditNotification struct {
	UserChatID string
	Repo       string
	Branch     string
	Path       string
	S3Key      string
}

type Notifier interface {
	NotifyFileEdited(ctx context.Context, notification FileEditNotification) error
}
