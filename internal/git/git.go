package git

import (
	"context"
	"fmt"
	"gitwatcher/internal/config"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/transport"
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
		slog.Error("Could not close repository", "repo", repo)
	}
}

func BuildAuthMethod(cfg config.Config) (transport.AuthMethod, error) {
	switch strings.ToLower(cfg.AuthType) {
	case "", strings.ToLower(config.AuthTypeNone):
		return nil, nil
	case strings.ToLower(config.AuthTypeHTTP):
		if cfg.AuthUser == "" || cfg.AuthPassword == "" {
			return nil, fmt.Errorf("AUTH_TYPE=HTTP requires AUTH_USER and AUTH_PASSWORD")
		}
		return &githttp.BasicAuth{Username: cfg.AuthUser, Password: cfg.AuthPassword}, nil
	default:
		return nil, fmt.Errorf("unsupported AUTH_TYPE %q, expected %q or %q", cfg.AuthType, config.AuthTypeNone, config.AuthTypeHTTP)
	}
}

func RebaseBranchOnOrigin(ctx context.Context, repositoryPath, branchName, userName, userEmail string, cfg config.Config) error {
	authEnv, err := gitAuthEnv(cfg)
	if err != nil {
		return fmt.Errorf("prepare git auth: %w", err)
	}
	if _, err := runGitCommand(ctx, repositoryPath, userName, userEmail, authEnv, "pull", "--rebase", "origin", branchName); err != nil {
		_, _ = runGitCommand(ctx, repositoryPath, userName, userEmail, authEnv, "rebase", "--abort")
		return fmt.Errorf("rebase local branch %q on origin failed: %w", branchName, err)
	}
	return nil
}

func gitAuthEnv(cfg config.Config) ([]string, error) {
	if !strings.EqualFold(cfg.AuthType, config.AuthTypeHTTP) {
		return nil, nil
	}
	if cfg.AuthUser == "" || cfg.AuthPassword == "" {
		return nil, fmt.Errorf("AUTH_TYPE=HTTP requires AUTH_USER and AUTH_PASSWORD")
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

func runGitCommand(ctx context.Context, repositoryPath, userName, userEmail string, extraEnv []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repositoryPath
	cmd.Env = append(cmd.Environ(),
		"GIT_AUTHOR_NAME="+userName,
		"GIT_AUTHOR_EMAIL="+userEmail,
		"GIT_COMMITTER_NAME="+userName,
		"GIT_COMMITTER_EMAIL="+userEmail,
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
