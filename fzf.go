package mangrove

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrCancelled is returned when the user cancels an fzf selection (Esc or Ctrl+C).
var ErrCancelled = errors.New("selection cancelled by user")

// IsFzfAvailable checks whether fzf is installed and available in PATH.
func IsFzfAvailable() bool {
	_, err := exec.LookPath("fzf")
	return err == nil
}

// SelectWithFzf presents a list of items via fzf for the user to select from.
// Returns the selected item or an error if fzf exits non-zero (e.g., user pressed Esc).
func SelectWithFzf(items []string, prompt, header string) (string, error) {
	if !IsFzfAvailable() {
		return "", fmt.Errorf("fzf is not installed. Install it with: brew install fzf")
	}

	if len(items) == 0 {
		return "", fmt.Errorf("no items to select from")
	}

	args := []string{}
	if prompt != "" {
		args = append(args, "--prompt", prompt+" ")
	}
	if header != "" {
		args = append(args, "--header", header)
	}
	args = append(args, "--height", "~40%", "--reverse")

	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n"))
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 130 || exitErr.ExitCode() == 1 {
				return "", fmt.Errorf("%w", ErrCancelled)
			}
		}
		return "", fmt.Errorf("fzf selection failed: %w", err)
	}

	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return "", fmt.Errorf("no item selected")
	}

	return selected, nil
}

// SelectDirectory lets the user pick a directory using fzf's directory walker.
// walkerRoot sets the starting directory for browsing. If empty, defaults to the user's home directory.
func SelectDirectory(prompt, walkerRoot string) (string, error) {
	if !IsFzfAvailable() {
		return "", fmt.Errorf("fzf is not installed. Install it with: brew install fzf")
	}

	if walkerRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		walkerRoot = home
	}

	args := []string{
		"--walker=dir,hidden",
		"--walker-root=" + walkerRoot,
		"--scheme=path",
		"--height", "~40%",
		"--reverse",
	}
	if prompt != "" {
		args = append(args, "--prompt", prompt+" ")
	}

	cmd := exec.Command("fzf", args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 130 || exitErr.ExitCode() == 1 {
				return "", fmt.Errorf("%w", ErrCancelled)
			}
		}
		return "", fmt.Errorf("fzf directory selection failed: %w", err)
	}

	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return "", fmt.Errorf("no directory selected")
	}

	return selected, nil
}

// skipDirs lists directory names that should be skipped during repository search.
var skipDirs = map[string]bool{
	"node_modules": true,
	".cache":       true,
	".npm":         true,
	".cargo":       true,
	"vendor":       true,
	"Library":      true,
}

// FindGitRepositories walks the directory tree rooted at root and returns
// paths that contain a .git directory (i.e., repository roots).
// It skips common non-project directories for performance and stops descending
// into a repository once .git is found.
func FindGitRepositories(root string) ([]string, error) {
	var repos []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}

		if !d.IsDir() {
			return nil
		}

		name := d.Name()

		// Skip common non-project directories
		if skipDirs[name] {
			return fs.SkipDir
		}

		// Check if this directory contains .git
		gitDir := filepath.Join(path, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			repos = append(repos, path)
			return fs.SkipDir // Don't descend into this repo
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory tree: %w", err)
	}

	return repos, nil
}

// SelectGitRepository finds git repositories under root and lets the user
// pick one via fzf.
func SelectGitRepository(prompt, root string) (string, error) {
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		root = home
	}

	repos, err := FindGitRepositories(root)
	if err != nil {
		return "", err
	}

	if len(repos) == 0 {
		return "", fmt.Errorf("no git repositories found under %s", root)
	}

	return SelectWithFzf(repos, prompt, "Select git repository")
}

// SelectBranch gets the branch list for a repo, puts defaultBranch first,
// and lets the user select via fzf.
func SelectBranch(repoPath, prompt, defaultBranch string) (string, error) {
	branches, err := BranchList(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to get branch list: %w", err)
	}

	if len(branches) == 0 {
		return "", fmt.Errorf("no branches found in %s", repoPath)
	}

	// Put default branch first if it exists
	ordered := reorderWithDefault(branches, defaultBranch)

	return SelectWithFzf(ordered, prompt, "Select base branch")
}

// SelectWorkspace lets the user select a workspace from a list of workspace labels via fzf.
func SelectWorkspace(items []string) (string, error) {
	return SelectWithFzf(items, "Select workspace:", "")
}

// SelectProfile lets the user select a profile from a list of profile names via fzf.
func SelectProfile(names []string) (string, error) {
	return SelectWithFzf(names, "Profile:", "Select profile")
}

// reorderWithDefault moves the defaultItem to the front of the list.
func reorderWithDefault(items []string, defaultItem string) []string {
	if defaultItem == "" {
		return items
	}

	ordered := make([]string, 0, len(items))
	found := false
	for _, item := range items {
		if item == defaultItem {
			found = true
			continue
		}
		ordered = append(ordered, item)
	}

	if found {
		ordered = append([]string{defaultItem}, ordered...)
	}

	return ordered
}
