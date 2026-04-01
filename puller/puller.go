package puller

import (
	"context"
	"errors"
	"fmt"
	"gitwatcher/internal/config"
	"log"
	"strings"

	git "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/transport"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
)

func PullRepo(ctx context.Context, cfg config.Config) error {
	if cfg.RepositoryPath == "" {
		return fmt.Errorf("repository path is empty")
	}

	authMethod, err := buildAuthMethod(cfg)
	if err != nil {
		return err
	}

	repo, err := git.PlainOpen(cfg.RepositoryPath)
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return fmt.Errorf("repository does not exist at %q: %w", cfg.RepositoryPath, err)
		}
		return fmt.Errorf("open repository %q: %w", cfg.RepositoryPath, err)
	}
	log.Println("Opened repository at", cfg.RepositoryPath)
	defer closeRepo(repo)

	fetchErr := repo.FetchContext(ctx, &git.FetchOptions{RemoteName: "origin", Auth: authMethod})
	if fetchErr != nil {
		if errors.Is(fetchErr, git.NoErrAlreadyUpToDate) {
			log.Println("No remote changes detected, skipping pull for", cfg.RepositoryPath)
			return nil
		}
		return fmt.Errorf("fetch from origin failed: %w", fetchErr)
	}
	log.Println("Fetched new changes for", cfg.RepositoryPath)

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("get repository worktree: %w", err)
	}

	if err := worktree.PullContext(ctx, &git.PullOptions{RemoteName: "origin", Auth: authMethod}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("pull from origin failed: %w", err)
	}
	log.Println("Pulled repository at", cfg.RepositoryPath)

	return nil
}

func closeRepo(repo *git.Repository) {
	if repo == nil {
		return
	}
	if err := repo.Close(); err != nil {
		log.Println("close repository:", err)
	}
}

func buildAuthMethod(cfg config.Config) (transport.AuthMethod, error) {
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
