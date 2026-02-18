package command

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Koutaro-Hanabusa/mangrove"
)

// initTestRepo はテスト用のgitリポジトリを作成してパスを返す。
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init", "-b", "main")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial commit")
	return dir
}

// gitRun はテスト内でgitコマンドを実行し出力を返す。
func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed in %s: %s: %v", args, dir, output, err)
	}
	return strings.TrimSpace(string(output))
}

// setupRepoWithWorktree はリポジトリとworktreeを作成する。
// worktreeブランチ名はwsBranchで作成される。
func setupRepoWithWorktree(t *testing.T, wsBranch string) (repoPath, wtPath string) {
	t.Helper()
	repoPath = initTestRepo(t)
	wtPath = filepath.Join(t.TempDir(), "worktree")
	gitRun(t, repoPath, "worktree", "add", wtPath, "-b", wsBranch, "main")
	return repoPath, wtPath
}

func TestApplyStash(t *testing.T) {
	repoPath, wtPath := setupRepoWithWorktree(t, "ws-test")

	// worktreeで未コミット変更を作成
	if err := os.WriteFile(filepath.Join(wtPath, "new.txt"), []byte("new file\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err := applyStash(wtPath, repoPath, "apply/test", "main", "test-repo")
	if err != nil {
		t.Fatalf("applyStash failed: %v", err)
	}

	// 元リポが新ブランチに切り替わっていること
	branch, err := mangrove.CurrentBranch(repoPath)
	if err != nil {
		t.Fatal(err)
	}
	if branch != "apply/test" {
		t.Errorf("applyStash後のブランチ = %q, want %q", branch, "apply/test")
	}

	// 元リポにnew.txtが存在すること（stash popで復元）
	if _, err := os.Stat(filepath.Join(repoPath, "new.txt")); os.IsNotExist(err) {
		t.Error("applyStash後にnew.txtが元リポに存在しない")
	}

	// worktreeがクリーンであること（stash pushで退避済み）
	status, err := mangrove.StatusPorcelain(wtPath)
	if err != nil {
		t.Fatal(err)
	}
	if status != "" {
		t.Errorf("applyStash後にworktreeが汚れている: %q", status)
	}
}

func TestApplyMerge(t *testing.T) {
	repoPath, wtPath := setupRepoWithWorktree(t, "ws-test")

	// worktreeでコミットを追加
	if err := os.WriteFile(filepath.Join(wtPath, "feature.txt"), []byte("feature\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, wtPath, "add", ".")
	gitRun(t, wtPath, "commit", "-m", "add feature in worktree")

	origBranch, _ := mangrove.CurrentBranch(repoPath)

	err := applyMerge(wtPath, repoPath, "ws-test", "apply/test", "main", "test-repo")
	if err != nil {
		t.Fatalf("applyMerge failed: %v", err)
	}

	// 元リポが元のブランチに復帰していること
	branch, err := mangrove.CurrentBranch(repoPath)
	if err != nil {
		t.Fatal(err)
	}
	if branch != origBranch {
		t.Errorf("applyMerge後のブランチ = %q, want %q (元のブランチ)", branch, origBranch)
	}

	// apply/testブランチにfeature.txtが存在すること
	gitRun(t, repoPath, "checkout", "apply/test")
	if _, err := os.Stat(filepath.Join(repoPath, "feature.txt")); os.IsNotExist(err) {
		t.Error("applyMerge後にfeature.txtがapply/testブランチに存在しない")
	}
}

func TestApplyStashRollbackOnCheckoutFailure(t *testing.T) {
	repoPath, wtPath := setupRepoWithWorktree(t, "ws-test")

	// worktreeで未コミット変更を作成
	if err := os.WriteFile(filepath.Join(wtPath, "new.txt"), []byte("new file\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// 同名ブランチを先に作って、checkout -b が失敗するようにする
	gitRun(t, repoPath, "branch", "apply/conflict")

	err := applyStash(wtPath, repoPath, "apply/conflict", "main", "test-repo")
	if err == nil {
		t.Fatal("既存ブランチ名でapplyStashがエラーにならなかった")
	}

	// ロールバック: worktreeに変更が復元されていること
	status, err := mangrove.StatusPorcelain(wtPath)
	if err != nil {
		t.Fatal(err)
	}
	if status == "" {
		t.Error("ロールバック後にworktreeの変更が復元されていない")
	}
}

func TestApplyMergeRollbackOnConflict(t *testing.T) {
	repoPath, wtPath := setupRepoWithWorktree(t, "ws-test")

	// worktreeでREADME変更してコミット
	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("# worktree change\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, wtPath, "add", ".")
	gitRun(t, wtPath, "commit", "-m", "worktree change")

	// mainでも同じファイルを変更（コンフリクト発生させる）
	gitRun(t, repoPath, "checkout", "main")
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# main conflict\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repoPath, "add", ".")
	gitRun(t, repoPath, "commit", "-m", "main conflict")

	origBranch, _ := mangrove.CurrentBranch(repoPath)

	err := applyMerge(wtPath, repoPath, "ws-test", "apply/conflict", "main", "test-repo")
	if err == nil {
		t.Fatal("コンフリクト時にapplyMergeがエラーにならなかった")
	}

	// ロールバック: 元ブランチに復帰していること
	branch, err := mangrove.CurrentBranch(repoPath)
	if err != nil {
		t.Fatal(err)
	}
	if branch != origBranch {
		t.Errorf("ロールバック後のブランチ = %q, want %q", branch, origBranch)
	}

	// apply/conflictブランチが削除されていること
	branches, err := mangrove.BranchList(repoPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range branches {
		if b == "apply/conflict" {
			t.Error("ロールバック後にapply/conflictブランチが残っている")
		}
	}
}
