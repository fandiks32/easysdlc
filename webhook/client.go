package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Error indicates a failed webhook delivery.
type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	return fmt.Sprintf("webhook error (HTTP %d): %s", e.StatusCode, e.Message)
}

// Client sends messages to a Google Chat incoming webhook.
type Client struct {
	httpClient *http.Client
	webhookURL string
}

// NewClient creates a webhook client for the given URL.
func NewClient(webhookURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		webhookURL: webhookURL,
	}
}

// Send posts a text message to the Google Chat webhook.
func (c *Client) Send(ctx context.Context, text string) error {
	payload := map[string]string{"text": text}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return &Error{StatusCode: resp.StatusCode, Message: string(body)}
	}

	return nil
}
