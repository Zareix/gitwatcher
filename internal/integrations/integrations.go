package integrations

import (
	"gitwatcher/internal/config"
	"log/slog"
)

func TriggerAllIntegrations(cfg config.Config) {
	slog.Info("Triggering integrations")

	if cfg.IntegrationArcaneUrl != "" && cfg.IntegrationArcaneToken != "" && cfg.IntegrationArcaneEnvId != "" {
		if err := RunArcaneIntegration(cfg.IntegrationArcaneUrl, cfg.IntegrationArcaneToken, cfg.IntegrationArcaneEnvId, cfg.IntegrationArcaneSkipNames); err != nil {
			slog.Error("Arcane integration failed", "error", err)
		}
	}

	if cfg.IntegrationWebhookUrl != "" && cfg.IntegrationWebhookToken != "" {
		if err := RunWebhookIntegration(cfg.IntegrationWebhookUrl, cfg.IntegrationWebhookToken); err != nil {
			slog.Error("Webhook integration failed", "error", err)
		}
	}
}
