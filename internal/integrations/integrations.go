package integrations

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"gitwatcher/internal/config"
)

func TriggerOnPull(ctx context.Context, cfg config.Config) {
	slog.Info("Triggering integrations after pull")

	var errs []error

	if cfg.IntegrationWebhookUrl != "" && cfg.IntegrationWebhookToken != "" {
		if err := RunWebhookIntegration(ctx, cfg.IntegrationWebhookUrl, cfg.IntegrationWebhookToken); err != nil {
			errs = append(errs, fmt.Errorf("webhook: %w", err))
		}
	}

	if err := errors.Join(errs...); err != nil {
		slog.Error("One or more integrations failed", "error", err)
	}
}
