package git

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	gitwatcherconfig "gitwatcher/internal/config"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/client"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
)

const askPassScript = "#!/bin/sh\ncase \"$1\" in\nUsername*) printf '%s' \"$GITWATCHER_AUTH_USER\" ;;\nPassword*) printf '%s' \"$GITWATCHER_AUTH_PASSWORD\" ;;\nesac\n"

var (
	askPassOnce sync.Once
	askPassPath string
	askPassErr  error
)

// ensureAskPassScript writes a GIT_ASKPASS helper once per process so credentials
// travel via env vars instead of argv (avoids leaking secrets through `ps`) or on-disk git config.
func ensureAskPassScript() (string, error) {
	askPassOnce.Do(func() {
		f, err := os.CreateTemp("", "gitwatcher-askpass-*.sh")
		if err != nil {
			askPassErr = err
			return
		}
		defer func() { _ = f.Close() }()
		if _, err := f.WriteString(askPassScript); err != nil {
			askPassErr = err
			return
		}
		if err := os.Chmod(f.Name(), 0o700); err != nil {
			askPassErr = err
			return
		}
		askPassPath = f.Name()
	})
	return askPassPath, askPassErr
}

func CloseRepo(repo *git.Repository) {
	if repo == nil {
		return
	}
	if err := repo.Close(); err != nil {
		slog.Error("Could not close repository", "error", err)
	}
}

func ValidateHTTPCredentials(cfg gitwatcherconfig.Config) error {
	if cfg.AuthUser == "" || cfg.AuthPassword == "" {
		return fmt.Errorf("AUTH_TYPE=HTTP requires AUTH_USER and AUTH_PASSWORD")
	}
	return nil
}

func BuildAuthOptions(cfg gitwatcherconfig.Config) ([]client.Option, error) {
	switch strings.ToLower(cfg.AuthType) {
	case "", strings.ToLower(gitwatcherconfig.AuthTypeNone):
		return nil, nil
	case strings.ToLower(gitwatcherconfig.AuthTypeHTTP):
		if err := ValidateHTTPCredentials(cfg); err != nil {
			return nil, err
		}
		return []client.Option{client.WithHTTPAuth(&githttp.BasicAuth{Username: cfg.AuthUser, Password: cfg.AuthPassword})}, nil
	default:
		return nil, fmt.Errorf("unsupported AUTH_TYPE %q, expected %q or %q", cfg.AuthType, gitwatcherconfig.AuthTypeNone, gitwatcherconfig.AuthTypeHTTP)
	}
}

func RebaseBranchOnOrigin(ctx context.Context, cfg gitwatcherconfig.Config, repositoryPath, branchName string) error {
	authEnv, err := gitAuthEnv(cfg)
	if err != nil {
		return fmt.Errorf("prepare git auth: %w", err)
	}
	if _, err := runGitCommand(ctx, repositoryPath, cfg, authEnv, "pull", "--rebase", "origin", branchName); err != nil {
		if _, abortErr := runGitCommand(ctx, repositoryPath, cfg, authEnv, "rebase", "--abort"); abortErr != nil {
			slog.Error("Failed to abort rebase after pull --rebase failure; repository may be mid-rebase", "repository", repositoryPath, "branch", branchName, "error", abortErr)
		}
		return fmt.Errorf("rebase local branch %q on origin failed: %w", branchName, err)
	}
	return nil
}

func gitAuthEnv(cfg gitwatcherconfig.Config) ([]string, error) {
	if !strings.EqualFold(cfg.AuthType, gitwatcherconfig.AuthTypeHTTP) {
		return nil, nil
	}
	if err := ValidateHTTPCredentials(cfg); err != nil {
		return nil, err
	}
	askPass, err := ensureAskPassScript()
	if err != nil {
		return nil, err
	}
	return []string{
		"GIT_ASKPASS=" + askPass,
		"GIT_TERMINAL_PROMPT=0",
		"GITWATCHER_AUTH_USER=" + cfg.AuthUser,
		"GITWATCHER_AUTH_PASSWORD=" + cfg.AuthPassword,
	}, nil
}

func runGitCommand(ctx context.Context, repositoryPath string, cfg gitwatcherconfig.Config, extraEnv []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repositoryPath
	cmd.Env = append(cmd.Environ(),
		"GIT_AUTHOR_NAME="+cfg.CommitName,
		"GIT_AUTHOR_EMAIL="+cfg.CommitEmail,
		"GIT_COMMITTER_NAME="+cfg.CommitName,
		"GIT_COMMITTER_EMAIL="+cfg.CommitEmail,
	)
	cmd.Env = append(cmd.Env, extraEnv...)

	output, err := cmd.CombinedOutput()
	outputText := strings.TrimSpace(string(output))
	if err != nil {
		if outputText == "" {
			return "", fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, outputText)
	}

	return outputText, nil
}
