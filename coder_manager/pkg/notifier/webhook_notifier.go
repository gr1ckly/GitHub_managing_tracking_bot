package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	internalnotifier "coder_manager/internal/notifier"
)

type WebhookNotifier struct {
	endpoint string
	client   *http.Client
}

type Notifier = internalnotifier.Notifier

type fileSavePayload struct {
	FileID  int       `json:"file_id"`
	SavedAt time.Time `json:"saved_at"`
}

func NewWebhookNotifier(endpoint string, timeout time.Duration) (*WebhookNotifier, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return nil, errors.New("notification endpoint is empty")
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &WebhookNotifier{
		endpoint: trimmed,
		client:   &http.Client{Timeout: timeout},
	}, nil
}

func (n *WebhookNotifier) NotifyFileEdited(ctx context.Context, notification internalnotifier.FileEditNotification) error {
	payload := fileSavePayload{
		FileID:  notification.FileID,
		SavedAt: notification.SavedAt,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return errors.New("notification request failed with status " + resp.Status)
	}
	return nil
}
