package watcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitwatcher/internal/config"

	git "github.com/go-git/go-git/v6"
	gitConfig "github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

func TestRunWatcherPushesCleanAheadBranch(t *testing.T) {
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

	secondCommit := commitFile(t, worktree, localPath, "note.txt", "second version", "second commit")

	if err := RunWatcher(ctx, config.Config{RepositoryPath: localPath}); err != nil {
		t.Fatalf("run watcher: %v", err)
	}

	remoteRepo, err := git.PlainOpen(remotePath)
	if err != nil {
		t.Fatalf("open remote repo: %v", err)
	}
	remoteRef, err := remoteRepo.Reference(plumbing.NewBranchReferenceName(branchName), true)
	if err != nil {
		t.Fatalf("read remote branch %q: %v", branchName, err)
	}
	if remoteRef.Hash() != secondCommit {
		t.Fatalf("remote branch hash = %s, want %s", remoteRef.Hash(), secondCommit)
	}
}

func TestRunWatcherDivergedManualPolicyReturnsError(t *testing.T) {
	setup := setupDivergedRepos(t)

	cfg := config.Config{
		RepositoryPath:   setup.localPath,
		DivergencePolicy: config.DivergencePolicyManual,
	}

	err := RunWatcher(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected diverged repository to fail with manual policy")
	}
	if !strings.Contains(err.Error(), "have diverged") {
		t.Fatalf("expected diverged error message, got: %v", err)
	}
}

func TestRunWatcherDivergedRebasePolicyRebasesAndPushes(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git CLI not found in PATH")
	}

	setup := setupDivergedRepos(t)
	cfg := config.Config{
		RepositoryPath:   setup.localPath,
		DivergencePolicy: config.DivergencePolicyRebase,
	}

	if err := RunWatcher(context.Background(), cfg); err != nil {
		t.Fatalf("run watcher with rebase policy: %v", err)
	}

	localRepo, err := git.PlainOpen(setup.localPath)
	if err != nil {
		t.Fatalf("open local repo: %v", err)
	}
	branchName, syncStatus, err := currentBranchSyncStatus(localRepo, "origin")
	if err != nil {
		t.Fatalf("read branch sync status: %v", err)
	}
	if syncStatus != branchSyncUpToDate {
		t.Fatalf("sync status = %s, want %s", syncStatus, branchSyncUpToDate)
	}

	localHead, err := localRepo.Head()
	if err != nil {
		t.Fatalf("read local head: %v", err)
	}
	remoteRepo, err := git.PlainOpen(setup.remotePath)
	if err != nil {
		t.Fatalf("open remote repo: %v", err)
	}
	remoteRef, err := remoteRepo.Reference(plumbing.NewBranchReferenceName(branchName), true)
	if err != nil {
		t.Fatalf("read remote branch %q: %v", branchName, err)
	}
	if remoteRef.Hash() != localHead.Hash() {
		t.Fatalf("remote hash = %s, local hash = %s", remoteRef.Hash(), localHead.Hash())
	}
}

type divergedSetup struct {
	remotePath string
	localPath  string
}

func setupDivergedRepos(t *testing.T) divergedSetup {
	t.Helper()

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
	if err := peerRepo.PushContext(context.Background(), &git.PushOptions{RemoteName: "origin"}); err != nil {
		t.Fatalf("push peer branch: %v", err)
	}

	commitFile(t, localWorktree, localPath, "local.txt", "local change", "local commit")

	return divergedSetup{remotePath: remotePath, localPath: localPath}
}

func initLocalRepo(t *testing.T, path string) (*git.Repository, *git.Worktree) {
	t.Helper()

	repo, err := git.PlainInit(path, false)
	if err != nil {
		t.Fatalf("init local repo: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("get worktree: %v", err)
	}

	return repo, worktree
}

func currentBranchName(t *testing.T, repo *git.Repository) string {
	t.Helper()

	headRef, err := repo.Head()
	if err != nil {
		t.Fatalf("read HEAD: %v", err)
	}
	return headRef.Name().Short()
}

func createRemote(t *testing.T, localRepo *git.Repository, remotePath string) {
	t.Helper()

	if _, err := localRepo.CreateRemote(&gitConfig.RemoteConfig{Name: "origin", URLs: []string{remotePath}}); err != nil {
		t.Fatalf("create remote: %v", err)
	}
}

func pushBranch(t *testing.T, repo *git.Repository, branchName string) {
	t.Helper()

	refSpec := gitConfig.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", branchName, branchName))
	if err := repo.PushContext(context.Background(), &git.PushOptions{RemoteName: "origin", RefSpecs: []gitConfig.RefSpec{refSpec}}); err != nil {
		t.Fatalf("push branch %q: %v", branchName, err)
	}
}

func commitFile(t *testing.T, worktree *git.Worktree, repoPath string, fileName string, content string, message string) plumbing.Hash {
	t.Helper()

	if err := os.WriteFile(filepath.Join(repoPath, fileName), []byte(content), 0o644); err != nil {
		t.Fatalf("write file %q: %v", fileName, err)
	}

	if _, err := worktree.Add(fileName); err != nil {
		t.Fatalf("add file %q: %v", fileName, err)
	}

	hash, err := worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit file %q: %v", fileName, err)
	}

	return hash
}
