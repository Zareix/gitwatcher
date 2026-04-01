package pusher

import (
	"context"
	"errors"
	"fmt"
	"gitwatcher/internal/config"
	gitInternal "gitwatcher/internal/git"
	"log"
	"time"

	git "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
)

func PushRepo(ctx context.Context, cfg config.Config) error {
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

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("get repository worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("get repository status: %w", err)
	}
	if status.IsClean() {
		log.Println("No local changes detected, skipping push for", cfg.RepositoryPath)
		return nil
	}

	if err := worktree.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return fmt.Errorf("stage changes: %w", err)
	}

	if _, err := worktree.Commit(cfg.CommitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  cfg.CommitName,
			Email: cfg.CommitEmail,
			When:  time.Now(),
		},
	}); err != nil {
		return fmt.Errorf("commit changes: %w", err)
	}
	log.Println("Committed local changes for", cfg.RepositoryPath)

	if err := repo.PushContext(ctx, &git.PushOptions{RemoteName: "origin", Auth: authMethod}); err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			log.Println("Remote already up to date, skipping push for", cfg.RepositoryPath)
			return nil
		}
		return fmt.Errorf("push to origin failed: %w", err)
	}
	log.Println("Pushed repository at", cfg.RepositoryPath)

	return nil
}
