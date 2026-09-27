package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gitwatcher/internal/config"

	git "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
)

func TestRunWatcherCommitsAndPushesDirtyWorktree(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	remotePath := filepath.Join(rootDir, "remote.git")
	localPath := filepath.Join(rootDir, "local")

	if _, err := git.PlainInit(remotePath, true); err != nil {
		t.Fatalf("init remote repo: %v", err)
	}

	localRepo, worktree := initLocalRepo(t, localPath)

	commitFile(t, worktree, localPath, "note.txt", "first version", "initial commit")
	branchName := currentBranchName(t, localRepo)
	createRemote(t, localRepo, remotePath)
	pushBranch(t, localRepo, branchName)

	if err := os.WriteFile(filepath.Join(localPath, "dirty.txt"), []byte("dirty content"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	cfg := config.Config{
		RepositoryPath: localPath,
		CommitName:     "Test User",
		CommitEmail:    "test@example.com",
	}
	if err := RunWatcher(ctx, cfg); err != nil {
		t.Fatalf("run watcher: %v", err)
	}

	assertBranchSynced(t, remotePath, localPath, branchName)

	status, err := worktree.Status()
	if err != nil {
		t.Fatalf("read worktree status: %v", err)
	}
	if !status.IsClean() {
		t.Fatalf("worktree should be clean after sync, got: %v", status)
	}
}

func TestRunWatcherPullsChangesOnCleanBehindBranch(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	remotePath := filepath.Join(rootDir, "remote.git")
	localPath := filepath.Join(rootDir, "local")
	peerPath := filepath.Join(rootDir, "peer")

	if _, err := git.PlainInit(remotePath, true); err != nil {
		t.Fatalf("init remote repo: %v", err)
	}

	localRepo, localWorktree := initLocalRepo(t, localPath)
	commitFile(t, localWorktree, localPath, "base.txt", "base", "initial commit")
	branchName := currentBranchName(t, localRepo)
	createRemote(t, localRepo, remotePath)
	pushBranch(t, localRepo, branchName)

	peerRepo, err := git.PlainClone(peerPath, &git.CloneOptions{URL: remotePath})
	if err != nil {
		t.Fatalf("clone peer repo: %v", err)
	}
	peerWorktree, err := peerRepo.Worktree()
	if err != nil {
		t.Fatalf("get peer worktree: %v", err)
	}
	commitFile(t, peerWorktree, peerPath, "remote.txt", "remote change", "remote commit")
	pushBranchFromPeer(t, peerRepo, branchName)

	if err := localRepo.FetchContext(ctx, &git.FetchOptions{RemoteName: "origin"}); err != nil {
		t.Fatalf("fetch remote refs: %v", err)
	}

	_, syncStatus, err := currentBranchSyncStatus(localRepo, "origin")
	if err != nil {
		t.Fatalf("read sync status before run: %v", err)
	}
	if syncStatus != branchSyncBehind {
		t.Fatalf("sync status before run = %s, want %s", syncStatus, branchSyncBehind)
	}

	if err := RunWatcher(ctx, config.Config{RepositoryPath: localPath}); err != nil {
		t.Fatalf("run watcher: %v", err)
	}

	assertBranchSynced(t, remotePath, localPath, branchName)

	remoteFile := filepath.Join(localPath, "remote.txt")
	if _, err := os.Stat(remoteFile); err != nil {
		t.Fatalf("pulled file %q should exist: %v", remoteFile, err)
	}
}

func TestRunWatcherSkipsWhenUpToDate(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	remotePath := filepath.Join(rootDir, "remote.git")
	localPath := filepath.Join(rootDir, "local")

	if _, err := git.PlainInit(remotePath, true); err != nil {
		t.Fatalf("init remote repo: %v", err)
	}

	localRepo, worktree := initLocalRepo(t, localPath)
	commitFile(t, worktree, localPath, "note.txt", "first version", "initial commit")
	branchName := currentBranchName(t, localRepo)
	createRemote(t, localRepo, remotePath)
	pushBranch(t, localRepo, branchName)

	if err := RunWatcher(ctx, config.Config{RepositoryPath: localPath}); err != nil {
		t.Fatalf("run watcher: %v", err)
	}

	assertBranchSynced(t, remotePath, localPath, branchName)
}

func pushBranchFromPeer(t *testing.T, repo *git.Repository, branchName string) {
	t.Helper()

	if err := repo.PushContext(context.Background(), &git.PushOptions{RemoteName: "origin"}); err != nil {
		t.Fatalf("push peer branch: %v", err)
	}
}

func assertBranchSynced(t *testing.T, remotePath string, localPath string, branchName string) {
	t.Helper()

	remoteRepo, err := git.PlainOpen(remotePath)
	if err != nil {
		t.Fatalf("open remote repo: %v", err)
	}
	remoteRef, err := remoteRepo.Reference(plumbing.NewBranchReferenceName(branchName), true)
	if err != nil {
		t.Fatalf("read remote branch %q: %v", branchName, err)
	}

	localRepo, err := git.PlainOpen(localPath)
	if err != nil {
		t.Fatalf("open local repo: %v", err)
	}
	localHead, err := localRepo.Head()
	if err != nil {
		t.Fatalf("read local head: %v", err)
	}

	if remoteRef.Hash() != localHead.Hash() {
		t.Fatalf("remote branch hash = %s, want local head hash %s", remoteRef.Hash(), localHead.Hash())
	}
}
