package notification

import (
	"context"
	"rep_tracker/pkg/dto"
)

type NotificationWriter interface {
	WriteNotification(ctx context.Context, chatId string, dto *dto.ChangingDTO) error
}
