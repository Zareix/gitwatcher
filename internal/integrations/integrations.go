package integrations

import (
	"gitwatcher/internal/config"
	"log"
)

func TriggerAllIntegrations(cfg config.Config) {
	log.Println("Triggering integrations")
	
	if cfg.IntegrationArcaneUrl != "" && cfg.IntegrationArcaneToken != "" && cfg.IntegrationArcaneEnvId != "" {
		if err := RunArcaneIntegration(cfg.IntegrationArcaneUrl, cfg.IntegrationArcaneToken, cfg.IntegrationArcaneEnvId, cfg.IntegrationArcaneSkipNames); err != nil {
			log.Printf("Arcane integration failed: %v", err)
		}
	}
}
