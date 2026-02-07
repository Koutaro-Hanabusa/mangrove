package mangrove

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

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
			if exitErr.ExitCode() == 130 {
				return "", fmt.Errorf("selection cancelled by user")
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
