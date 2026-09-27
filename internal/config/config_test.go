package config

import (
	"testing"

	"github.com/google/uuid"
)

func TestLoadConfigDefaults(t *testing.T) {
	for _, key := range []string{"REPOSITORY_PATH", "PORT", "CRON", "AUTH_TYPE", "COMMIT_NAME", "COMMIT_EMAIL", "COMMIT_MESSAGE", "DIVERGENCE_POLICY", "JOB_UUID", "LOG_JSON"} {
		t.Setenv(key, "")
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.RepositoryPath != "./output" {
		t.Fatalf("RepositoryPath = %q, want %q", cfg.RepositoryPath, "./output")
	}
	if cfg.Port != 8080 {
		t.Fatalf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.WatcherJobCron != "0 */1 * * * *" {
		t.Fatalf("WatcherJobCron = %q, want %q", cfg.WatcherJobCron, "0 */1 * * * *")
	}
	if cfg.AuthType != AuthTypeNone {
		t.Fatalf("AuthType = %q, want %q", cfg.AuthType, AuthTypeNone)
	}
	if cfg.CommitName != "gitwatcher" {
		t.Fatalf("CommitName = %q, want %q", cfg.CommitName, "gitwatcher")
	}
	if cfg.CommitEmail != "gitwatcher@local" {
		t.Fatalf("CommitEmail = %q, want %q", cfg.CommitEmail, "gitwatcher@local")
	}
	if cfg.CommitMessage != "chore: sync changes from gitwatcher" {
		t.Fatalf("CommitMessage = %q, want %q", cfg.CommitMessage, "chore: sync changes from gitwatcher")
	}
	if cfg.DivergencePolicy != DivergencePolicyManual {
		t.Fatalf("DivergencePolicy = %q, want %q", cfg.DivergencePolicy, DivergencePolicyManual)
	}
	if cfg.WatcherJobUUID == uuid.Nil {
		t.Fatal("WatcherJobUUID should be randomly generated when JOB_UUID is unset")
	}
}

func TestLoadConfigInvalidPort(t *testing.T) {
	t.Setenv("PORT", "not-a-number")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected error for invalid PORT")
	}
}

func TestLoadConfigInvalidJobUUID(t *testing.T) {
	t.Setenv("JOB_UUID", "not-a-uuid")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected error for invalid JOB_UUID")
	}
}

func TestLoadConfigValidJobUUID(t *testing.T) {
	t.Setenv("JOB_UUID", "8eaac5c3-375e-4975-bddb-a5c85228eaa0")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.WatcherJobUUID.String() != "8eaac5c3-375e-4975-bddb-a5c85228eaa0" {
		t.Fatalf("WatcherJobUUID = %q", cfg.WatcherJobUUID)
	}
}

func TestLoadConfigInvalidDivergencePolicy(t *testing.T) {
	t.Setenv("DIVERGENCE_POLICY", "force")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected error for invalid DIVERGENCE_POLICY")
	}
}

func TestLoadConfigDivergencePolicyCaseInsensitive(t *testing.T) {
	t.Setenv("DIVERGENCE_POLICY", "Rebase")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DivergencePolicy != DivergencePolicyRebase {
		t.Fatalf("DivergencePolicy = %q, want %q", cfg.DivergencePolicy, DivergencePolicyRebase)
	}
}
