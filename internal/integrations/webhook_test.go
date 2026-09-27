package integrations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitwatcher/internal/config"
)

func TestRunWebhookIntegrationSuccess(t *testing.T) {
	var gotAuth string
	var gotMethod string
	var gotPath string
	called := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		close(called)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := RunWebhookIntegration(context.Background(), server.URL, "secret-token")
	if err != nil {
		t.Fatalf("run webhook: %v", err)
	}

	select {
	case <-called:
	default:
		t.Fatal("webhook server was not called")
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/" {
		t.Fatalf("path = %q, want /", gotPath)
	}
	if gotAuth != "Bearer secret-token" {
		t.Fatalf("authorization = %q, want %q", gotAuth, "Bearer secret-token")
	}
}

func TestRunWebhookIntegrationHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	err := RunWebhookIntegration(context.Background(), server.URL, "token")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestRunWebhookIntegrationHonoursContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	if err := RunWebhookIntegration(ctx, server.URL, "token"); err == nil {
		t.Fatal("expected error when context is cancelled")
	}
}

func TestTriggerOnPullSkipsUnconfiguredIntegrations(t *testing.T) {
	TriggerOnPull(context.Background(), config.Config{})
}
