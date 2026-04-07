package git

import (
	"context"
	"fmt"
	"gitwatcher/internal/config"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/transport"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
)

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

func RebaseBranchOnOrigin(ctx context.Context, repositoryPath string, branchName string) error {
	if _, err := runGitCommand(ctx, repositoryPath, "pull", "--rebase", "origin", branchName); err != nil {
		_, _ = runGitCommand(ctx, repositoryPath, "rebase", "--abort")
		return fmt.Errorf("rebase local branch %q on origin failed: %w", branchName, err)
	}
	return nil
}

func runGitCommand(ctx context.Context, repositoryPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repositoryPath

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
