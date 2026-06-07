package integrations

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

func RunWebhookIntegration(url string, token string) error {
	slog.Info("Triggering webhook", "url", url)

	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute webhook request: %w", err)
	}
	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(res.Body)

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("webhook returned %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	slog.Info("Webhook triggered successfully", "url", url, "status", res.StatusCode)

	return nil
}
