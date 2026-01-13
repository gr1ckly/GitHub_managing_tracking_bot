package notifier

import (
	"context"

	internalnotifier "coder_manager/internal/notifier"
)

type StubNotifier struct{}

func NewStubNotifier() *StubNotifier {
	return &StubNotifier{}
}

func (n *StubNotifier) NotifyFileEdited(ctx context.Context, notification internalnotifier.FileEditNotification) error {
	_ = ctx
	_ = notification
	return nil
}
