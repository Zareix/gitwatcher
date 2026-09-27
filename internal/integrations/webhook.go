package integrations

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const webhookTimeout = 30 * time.Second

func RunWebhookIntegration(ctx context.Context, url string, token string) error {
	slog.Info("Triggering webhook", "url", url)

	client := &http.Client{Timeout: webhookTimeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("execute webhook request: %w", err)
	}
	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(res.Body)

	body, err := io.ReadAll(io.LimitReader(res.Body, 1024))
	if err != nil {
		return fmt.Errorf("read webhook response body: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("webhook returned %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	slog.Info("Webhook triggered successfully", "url", url, "status", res.StatusCode)

	return nil
}
