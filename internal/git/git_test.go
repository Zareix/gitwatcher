package git

import (
	"testing"

	gitwatcherconfig "gitwatcher/internal/config"
)

func TestBuildAuthOptionsNone(t *testing.T) {
	cfg := gitwatcherconfig.Config{AuthType: gitwatcherconfig.AuthTypeNone}

	opts, err := BuildAuthOptions(cfg)
	if err != nil {
		t.Fatalf("build auth options: %v", err)
	}
	if len(opts) != 0 {
		t.Fatalf("opts = %v, want empty", opts)
	}
}

func TestBuildAuthOptionsEmptyTypeDefaultsToNone(t *testing.T) {
	cfg := gitwatcherconfig.Config{}

	opts, err := BuildAuthOptions(cfg)
	if err != nil {
		t.Fatalf("build auth options: %v", err)
	}
	if len(opts) != 0 {
		t.Fatalf("opts = %v, want empty", opts)
	}
}

func TestBuildAuthOptionsHTTP(t *testing.T) {
	cfg := gitwatcherconfig.Config{
		AuthType:     gitwatcherconfig.AuthTypeHTTP,
		AuthUser:     "user",
		AuthPassword: "pass",
	}

	opts, err := BuildAuthOptions(cfg)
	if err != nil {
		t.Fatalf("build auth options: %v", err)
	}
	if len(opts) != 1 {
		t.Fatalf("len(opts) = %d, want 1", len(opts))
	}
}

func TestBuildAuthOptionsHTTPMissingCredentials(t *testing.T) {
	tests := map[string]gitwatcherconfig.Config{
		"missing both":     {AuthType: gitwatcherconfig.AuthTypeHTTP},
		"missing password": {AuthType: gitwatcherconfig.AuthTypeHTTP, AuthUser: "user"},
		"missing user":     {AuthType: gitwatcherconfig.AuthTypeHTTP, AuthPassword: "pass"},
	}

	for name, cfg := range tests {
		if _, err := BuildAuthOptions(cfg); err == nil {
			t.Fatalf("%s: expected error for missing credentials", name)
		}
	}
}

func TestBuildAuthOptionsUnsupported(t *testing.T) {
	cfg := gitwatcherconfig.Config{AuthType: "ssh"}

	if _, err := BuildAuthOptions(cfg); err == nil {
		t.Fatal("expected error for unsupported auth type")
	}
}

func TestValidateHTTPCredentials(t *testing.T) {
	if err := ValidateHTTPCredentials(gitwatcherconfig.Config{AuthUser: "u", AuthPassword: "p"}); err != nil {
		t.Fatalf("validate valid credentials: %v", err)
	}
	if err := ValidateHTTPCredentials(gitwatcherconfig.Config{AuthUser: "u"}); err == nil {
		t.Fatal("expected error when password missing")
	}
	if err := ValidateHTTPCredentials(gitwatcherconfig.Config{AuthPassword: "p"}); err == nil {
		t.Fatal("expected error when user missing")
	}
}
