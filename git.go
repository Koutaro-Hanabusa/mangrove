package mangrove

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// WorktreeAdd creates a new worktree with a new branch.
// Equivalent to: git -C <repoPath> worktree add <worktreePath> -b <branch> <base>
func WorktreeAdd(repoPath, worktreePath, branch, base string) error {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "add", worktreePath, "-b", branch, base)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// WorktreeRemove removes an existing worktree.
// Equivalent to: git -C <repoPath> worktree remove <worktreePath>
func WorktreeRemove(repoPath, worktreePath string, force bool) error {
	args := []string{"-C", repoPath, "worktree", "remove", worktreePath}
	if force {
		args = append(args, "--force")
	}
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// WorktreeEntry represents a single worktree from porcelain output.
type WorktreeEntry struct {
	Worktree string
	HEAD     string
	Branch   string
	Bare     bool
	Detached bool
}

// WorktreeList lists worktrees for a repository in porcelain format.
// Equivalent to: git -C <repoPath> worktree list --porcelain
func WorktreeList(repoPath string) ([]WorktreeEntry, error) {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list failed: %w", err)
	}

	var entries []WorktreeEntry
	var current WorktreeEntry

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "worktree "):
			if current.Worktree != "" {
				entries = append(entries, current)
			}
			current = WorktreeEntry{Worktree: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "HEAD "):
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimPrefix(line, "branch ")
		case line == "bare":
			current.Bare = true
		case line == "detached":
			current.Detached = true
		case line == "":
			if current.Worktree != "" {
				entries = append(entries, current)
				current = WorktreeEntry{}
			}
		}
	}
	if current.Worktree != "" {
		entries = append(entries, current)
	}

	return entries, nil
}

// BranchList returns the list of local branch names for a repository.
// Equivalent to: git -C <repoPath> branch --list --format=%(refname:short)
func BranchList(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoPath, "branch", "--list", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git branch list failed: %w", err)
	}
	return parseLines(string(output)), nil
}

// RemoteBranchList returns the list of remote branch names for a repository.
// Equivalent to: git -C <repoPath> branch -r --format=%(refname:short)
func RemoteBranchList(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoPath, "branch", "-r", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git remote branch list failed: %w", err)
	}
	return parseLines(string(output)), nil
}

// StatusPorcelain returns the porcelain status output for a given path.
// Equivalent to: git -C <path> status --porcelain
func StatusPorcelain(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git status failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// StatusChangedCount returns the number of changed files in a worktree.
func StatusChangedCount(path string) (int, error) {
	status, err := StatusPorcelain(path)
	if err != nil {
		return 0, err
	}
	if status == "" {
		return 0, nil
	}
	return len(strings.Split(status, "\n")), nil
}

// AheadBehind returns the number of commits ahead and behind between branch and base.
// Equivalent to: git -C <repoPath> rev-list --count --left-right <base>...<branch>
func AheadBehind(repoPath, base, branch string) (ahead int, behind int, err error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-list", "--count", "--left-right", base+"..."+branch)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("git rev-list failed: %w", err)
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output: %q", string(output))
	}

	behind, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse behind count: %w", err)
	}
	ahead, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse ahead count: %w", err)
	}

	return ahead, behind, nil
}

// BranchDelete deletes a local branch.
// Equivalent to: git -C <repoPath> branch -d <branch>
func BranchDelete(repoPath, branch string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	cmd := exec.Command("git", "-C", repoPath, "branch", flag, branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git branch delete failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// FetchAll fetches from all remotes.
// Equivalent to: git -C <repoPath> fetch --all
func FetchAll(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "fetch", "--all")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch --all failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// CurrentBranch returns the current branch name of a worktree or repo.
func CurrentBranch(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// StashPush は変更をstashに退避する（未追跡ファイルも含む）。
// 実行コマンド: git -C <path> stash push --include-untracked -m <message>
func StashPush(path, message string) error {
	cmd := exec.Command("git", "-C", path, "stash", "push", "--include-untracked", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git stash push failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// StashPop は最新のstashエントリを適用して削除する。
// 実行コマンド: git -C <path> stash pop
func StashPop(path string) error {
	cmd := exec.Command("git", "-C", path, "stash", "pop")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git stash pop failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// StashRef は最新のstashエントリのコミットSHAを返す。
// worktree間でstashを共有するために使用する（reflogはworktreeごとだがオブジェクトは共有）。
func StashRef(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "stash@{0}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse stash@{0} failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// StashApply はstashのコミットSHAを指定して変更を適用する（stashは削除しない）。
// 実行コマンド: git -C <path> stash apply <ref>
func StashApply(path, ref string) error {
	cmd := exec.Command("git", "-C", path, "stash", "apply", ref)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git stash apply failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// StashDrop は最新のstashエントリを削除する。
// 実行コマンド: git -C <path> stash drop
func StashDrop(path string) error {
	cmd := exec.Command("git", "-C", path, "stash", "drop")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git stash drop failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// CheckoutBranch は既存のブランチに切り替える。
// 実行コマンド: git -C <path> checkout <branch>
func CheckoutBranch(path, branch string) error {
	cmd := exec.Command("git", "-C", path, "checkout", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// CheckoutNewBranch はベースブランチから新しいブランチを作成して切り替える。
// 実行コマンド: git -C <path> checkout -b <newBranch> <base>
func CheckoutNewBranch(path, newBranch, base string) error {
	cmd := exec.Command("git", "-C", path, "checkout", "-b", newBranch, base)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout -b failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// MergeAbort は進行中のマージを中断する。
// 実行コマンド: git -C <path> merge --abort
func MergeAbort(path string) error {
	cmd := exec.Command("git", "-C", path, "merge", "--abort")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git merge --abort failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// Merge は指定ブランチを現在のブランチにマージする。
// 実行コマンド: git -C <path> merge <branch>
func Merge(path, branch string) error {
	cmd := exec.Command("git", "-C", path, "merge", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git merge failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// parseLines splits output by newlines and returns non-empty trimmed lines.
func parseLines(output string) []string {
	var lines []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
