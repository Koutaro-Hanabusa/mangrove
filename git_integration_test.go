package mangrove

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo creates a temporary git repository with an initial commit on "main".
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %s: %v", args, out, err)
		}
	}
	run("init")
	run("checkout", "-b", "main")
	// create initial commit
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	run("add", ".")
	run("commit", "-m", "initial commit")
	return dir
}

func TestCurrentBranch(t *testing.T) {
	repo := initTestRepo(t)

	t.Run("デフォルトブランチはmain", func(t *testing.T) {
		branch, err := CurrentBranch(repo)
		if err != nil {
			t.Fatalf("CurrentBranch failed: %v", err)
		}
		if branch != "main" {
			t.Errorf("got %q, want %q", branch, "main")
		}
	})

	t.Run("チェックアウト後は新ブランチ名を返す", func(t *testing.T) {
		cmd := exec.Command("git", "-C", repo, "checkout", "-b", "develop")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git checkout -b develop failed: %s: %v", out, err)
		}

		branch, err := CurrentBranch(repo)
		if err != nil {
			t.Fatalf("CurrentBranch failed: %v", err)
		}
		if branch != "develop" {
			t.Errorf("got %q, want %q", branch, "develop")
		}
	})
}

func TestBranchList(t *testing.T) {
	repo := initTestRepo(t)

	t.Run("mainブランチが含まれる", func(t *testing.T) {
		branches, err := BranchList(repo)
		if err != nil {
			t.Fatalf("BranchList failed: %v", err)
		}
		if !contains(branches, "main") {
			t.Errorf("expected branches to contain %q, got %v", "main", branches)
		}
	})

	t.Run("作成したブランチが一覧に含まれる", func(t *testing.T) {
		cmd := exec.Command("git", "-C", repo, "branch", "feature")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git branch feature failed: %s: %v", out, err)
		}

		branches, err := BranchList(repo)
		if err != nil {
			t.Fatalf("BranchList failed: %v", err)
		}
		if !contains(branches, "main") {
			t.Errorf("expected branches to contain %q, got %v", "main", branches)
		}
		if !contains(branches, "feature") {
			t.Errorf("expected branches to contain %q, got %v", "feature", branches)
		}
	})
}

func TestStatusPorcelain(t *testing.T) {
	repo := initTestRepo(t)

	t.Run("クリーンなリポジトリは空文字列", func(t *testing.T) {
		status, err := StatusPorcelain(repo)
		if err != nil {
			t.Fatalf("StatusPorcelain failed: %v", err)
		}
		if status != "" {
			t.Errorf("expected empty status for clean repo, got %q", status)
		}
	})

	t.Run("未追跡ファイルは??で表示", func(t *testing.T) {
		os.WriteFile(filepath.Join(repo, "untracked.txt"), []byte("hello"), 0644)

		status, err := StatusPorcelain(repo)
		if err != nil {
			t.Fatalf("StatusPorcelain failed: %v", err)
		}
		if !strings.Contains(status, "??") {
			t.Errorf("expected status to contain %q, got %q", "??", status)
		}
	})
}

func TestStatusChangedCount(t *testing.T) {
	repo := initTestRepo(t)

	t.Run("クリーンなリポジトリは0件", func(t *testing.T) {
		count, err := StatusChangedCount(repo)
		if err != nil {
			t.Fatalf("StatusChangedCount failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0, got %d", count)
		}
	})

	t.Run("未追跡ファイル1つで1件", func(t *testing.T) {
		os.WriteFile(filepath.Join(repo, "file1.txt"), []byte("a"), 0644)

		count, err := StatusChangedCount(repo)
		if err != nil {
			t.Fatalf("StatusChangedCount failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1, got %d", count)
		}
	})

	t.Run("未追跡ファイル2つで2件", func(t *testing.T) {
		os.WriteFile(filepath.Join(repo, "file2.txt"), []byte("b"), 0644)

		count, err := StatusChangedCount(repo)
		if err != nil {
			t.Fatalf("StatusChangedCount failed: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2, got %d", count)
		}
	})
}

func TestWorktreeLifecycle(t *testing.T) {
	repo := initTestRepo(t)
	wtDir := filepath.Join(t.TempDir(), "wt")

	// Resolve symlinks so paths match git's resolved output (macOS /var -> /private/var).
	resolvedWtDir := wtDir
	if resolved, err := filepath.EvalSymlinks(filepath.Dir(wtDir)); err == nil {
		resolvedWtDir = filepath.Join(resolved, "wt")
	}

	// Step 1: Add worktree
	err := WorktreeAdd(repo, wtDir, "feature", "main")
	if err != nil {
		t.Fatalf("WorktreeAdd failed: %v", err)
	}

	// Step 2: Verify worktree directory exists
	if _, err := os.Stat(wtDir); os.IsNotExist(err) {
		t.Fatal("worktree directory was not created")
	}

	// Step 3: WorktreeList should include the new worktree
	entries, err := WorktreeList(repo)
	if err != nil {
		t.Fatalf("WorktreeList failed: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Worktree == resolvedWtDir {
			found = true
			if e.Branch != "refs/heads/feature" {
				t.Errorf("expected branch %q, got %q", "refs/heads/feature", e.Branch)
			}
			break
		}
	}
	if !found {
		t.Errorf("worktree %q not found in list: %+v", resolvedWtDir, entries)
	}

	// Step 4: CurrentBranch on worktree should return "feature"
	branch, err := CurrentBranch(wtDir)
	if err != nil {
		t.Fatalf("CurrentBranch on worktree failed: %v", err)
	}
	if branch != "feature" {
		t.Errorf("expected branch %q, got %q", "feature", branch)
	}

	// Step 5: Remove worktree
	err = WorktreeRemove(repo, wtDir, false)
	if err != nil {
		t.Fatalf("WorktreeRemove failed: %v", err)
	}

	// Step 6: WorktreeList should no longer include the removed worktree
	entries, err = WorktreeList(repo)
	if err != nil {
		t.Fatalf("WorktreeList after remove failed: %v", err)
	}
	for _, e := range entries {
		if e.Worktree == resolvedWtDir {
			t.Errorf("worktree %q should have been removed but still in list", resolvedWtDir)
		}
	}
}

// contains checks if a string slice contains a given value.
func contains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
