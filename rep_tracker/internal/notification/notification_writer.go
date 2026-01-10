package notification

import (
	"context"
	"rep_tracker/pkg/dto"
)

type NotificationWriter interface {
	WriteNotification(ctx context.Context, dto dto.ChangingDTO) error
}
