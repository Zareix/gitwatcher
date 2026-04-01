package puller

import (
	"context"
	"errors"
	"fmt"
	"gitwatcher/internal/config"
	gitInternal "gitwatcher/internal/git"
	"log"

	git "github.com/go-git/go-git/v6"
)

func PullRepo(ctx context.Context, cfg config.Config) error {
	if cfg.RepositoryPath == "" {
		return fmt.Errorf("repository path is empty")
	}

	authMethod, err := gitInternal.BuildAuthMethod(cfg)
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
	defer gitInternal.CloseRepo(repo)

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
