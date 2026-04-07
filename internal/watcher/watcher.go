package watcher

import (
	"context"
	"errors"
	"fmt"
	"gitwatcher/internal/config"
	gitInternal "gitwatcher/internal/git"
	"gitwatcher/internal/integrations"
	"log/slog"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

type branchSyncStatus int

const (
	branchSyncUpToDate branchSyncStatus = iota
	branchSyncAhead
	branchSyncBehind
	branchSyncDiverged
)

func (s branchSyncStatus) String() string {
	switch s {
	case branchSyncUpToDate:
		return "up_to_date"
	case branchSyncAhead:
		return "ahead"
	case branchSyncBehind:
		return "behind"
	case branchSyncDiverged:
		return "diverged"
	default:
		return "unknown"
	}
}

func RunWatcher(ctx context.Context, cfg config.Config) error {
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
	slog.Debug("Opened repository", "repository", cfg.RepositoryPath)
	defer gitInternal.CloseRepo(repo)

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("get repository worktree: %w", err)
	}

	fetchErr := repo.FetchContext(ctx, &git.FetchOptions{RemoteName: "origin", Auth: authMethod})
	if fetchErr != nil && !errors.Is(fetchErr, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fetch from origin failed: %w", fetchErr)
	}

	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("get repository status: %w", err)
	}
	if status.IsClean() {
		branchName, syncStatus, err := currentBranchSyncStatus(repo, "origin")
		if err != nil {
			return err
		}
		switch syncStatus {
		case branchSyncUpToDate:
			slog.Info("Repository is up to date", "repository", cfg.RepositoryPath, "branch", branchName)
			return nil
		case branchSyncAhead:
			slog.Info("Local branch is ahead of origin, pushing changes", "repository", cfg.RepositoryPath, "branch", branchName)
			if err := repo.PushContext(ctx, &git.PushOptions{RemoteName: "origin", Auth: authMethod}); err != nil {
				if errors.Is(err, git.NoErrAlreadyUpToDate) {
					slog.Info("Remote already up to date, skipping push", "repository", cfg.RepositoryPath, "branch", branchName)
					return nil
				}
				return fmt.Errorf("push to origin failed: %w", err)
			}
			slog.Info("Pushed repository", "repository", cfg.RepositoryPath, "branch", branchName)
			return nil
		case branchSyncBehind:
			slog.Info("Remote branch is ahead of local branch, pulling changes", "repository", cfg.RepositoryPath, "branch", branchName)
			err := worktree.PullContext(ctx, &git.PullOptions{RemoteName: "origin", ReferenceName: plumbing.NewBranchReferenceName(branchName), Auth: authMethod})
			if err != nil {
				if errors.Is(err, git.NoErrAlreadyUpToDate) {
					slog.Info("No changes detected on remote, skipping pull", "repository", cfg.RepositoryPath, "branch", branchName)
					return nil
				}
				return fmt.Errorf("pull from origin failed: %w", err)
			}
			slog.Info("Pulled repository", "repository", cfg.RepositoryPath, "branch", branchName)
			integrations.TriggerAllIntegrations(cfg)
			return nil
		case branchSyncDiverged:
			switch cfg.DivergencePolicy {
			case config.DivergencePolicyRebase:
				slog.Info("Local and remote branches diverged, attempting rebase", "repository", cfg.RepositoryPath, "branch", branchName)
				if err := gitInternal.RebaseBranchOnOrigin(ctx, cfg.RepositoryPath, branchName); err != nil {
					return err
				}
				slog.Info("Rebase completed, pushing rebased commits", "repository", cfg.RepositoryPath, "branch", branchName)
				if err := repo.PushContext(ctx, &git.PushOptions{RemoteName: "origin", Auth: authMethod}); err != nil {
					if errors.Is(err, git.NoErrAlreadyUpToDate) {
						slog.Info("Remote already up to date, skipping push", "repository", cfg.RepositoryPath, "branch", branchName)
						return nil
					}
					return fmt.Errorf("push to origin failed after rebase: %w", err)
				}
				slog.Info("Pushed repository", "repository", cfg.RepositoryPath, "branch", branchName)
				return nil
			case config.DivergencePolicyManual:
				return fmt.Errorf("local branch %q and origin have diverged; resolve manually or set DIVERGENCE_POLICY=%s", branchName, config.DivergencePolicyRebase)
			default:
				return fmt.Errorf("unsupported divergence policy %q", cfg.DivergencePolicy)
			}
		default:
			return fmt.Errorf("unknown sync status for branch %q", branchName)
		}
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
	slog.Info("Committed local changes", "repository", cfg.RepositoryPath)

	if err := worktree.PullContext(ctx, &git.PullOptions{RemoteName: "origin", Auth: authMethod}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("pull from origin failed: %w", err)
	}

	if err := repo.PushContext(ctx, &git.PushOptions{RemoteName: "origin", Auth: authMethod}); err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			slog.Info("Remote already up to date, skipping push", "repository", cfg.RepositoryPath)
			return nil
		}
		return fmt.Errorf("push to origin failed: %w", err)
	}
	slog.Info("Pushed repository", "repository", cfg.RepositoryPath)

	return nil
}

func currentBranchSyncStatus(repo *git.Repository, remoteName string) (string, branchSyncStatus, error) {
	headRef, err := repo.Head()
	if err != nil {
		return "", branchSyncDiverged, fmt.Errorf("get HEAD reference: %w", err)
	}
	if !headRef.Name().IsBranch() {
		return "", branchSyncDiverged, fmt.Errorf("repository is in detached HEAD state: %s", headRef.Name())
	}

	branchName := headRef.Name().Short()
	localCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", branchSyncDiverged, fmt.Errorf("load local HEAD commit: %w", err)
	}

	remoteRefName := plumbing.NewRemoteReferenceName(remoteName, branchName)
	remoteRef, err := repo.Reference(remoteRefName, true)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return branchName, branchSyncAhead, nil
		}
		return "", branchSyncDiverged, fmt.Errorf("get remote reference %q: %w", remoteRefName, err)
	}

	remoteCommit, err := repo.CommitObject(remoteRef.Hash())
	if err != nil {
		return "", branchSyncDiverged, fmt.Errorf("load remote commit %q: %w", remoteRefName, err)
	}

	if localCommit.Hash == remoteCommit.Hash {
		return branchName, branchSyncUpToDate, nil
	}

	remoteIsAncestor, err := remoteCommit.IsAncestor(localCommit)
	if err != nil {
		return "", branchSyncDiverged, fmt.Errorf("compare remote and local commits: %w", err)
	}
	localIsAncestor, err := localCommit.IsAncestor(remoteCommit)
	if err != nil {
		return "", branchSyncDiverged, fmt.Errorf("compare local and remote commits: %w", err)
	}

	switch {
	case remoteIsAncestor:
		return branchName, branchSyncAhead, nil
	case localIsAncestor:
		return branchName, branchSyncBehind, nil
	default:
		return branchName, branchSyncDiverged, nil
	}
}
