package mangrove

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorkspaceInfo represents summary info about a workspace.
type WorkspaceInfo struct {
	ProfileName   string
	WorkspaceName string
	Path          string
	RepoStatuses  []RepoStatus
}

// RepoStatus represents the status of a single repo within a workspace.
type RepoStatus struct {
	RepoName     string
	BranchName   string
	ChangedCount int
	Ahead        int
	Behind       int
	DefaultBase  string
	Exists       bool
}

// GetWorkspacePath returns the full path for a workspace.
func GetWorkspacePath(cfg *Config, profileName, name string) string {
	return filepath.Join(cfg.BaseDir, profileName, name)
}

// CreateWorkspace creates a new workspace with worktrees for all repos in the profile.
func CreateWorkspace(cfg *Config, profile *Profile, profileName, name string, baseBranches map[string]string) error {
	wsPath := GetWorkspacePath(cfg, profileName, name)

	// Check if workspace already exists
	if _, err := os.Stat(wsPath); err == nil {
		return fmt.Errorf("workspace %q already exists at %s", name, wsPath)
	}

	// Create workspace directory
	if err := os.MkdirAll(wsPath, 0o755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nCreating workspace: %s/%s\n", profileName, name)

	// Create worktrees for each repo
	for _, repo := range profile.Repos {
		base, ok := baseBranches[repo.Name]
		if !ok {
			base = repo.GetDefaultBase()
		}

		worktreePath := filepath.Join(wsPath, repo.Name)

		if err := WorktreeAdd(repo.Path, worktreePath, name, base); err != nil {
			// Clean up on failure
			cleanupWorkspace(cfg, profile, profileName, name)
			return fmt.Errorf("failed to create worktree for %s: %w", repo.Name, err)
		}

		PrintSuccess("%s  %s \u2192 %s",
			RepoNameStyle.Render(repo.Name),
			BranchNameStyle.Render(base),
			BranchNameStyle.Render(name),
		)
	}

	// Run post_create hooks
	if len(profile.Hooks.PostCreate) > 0 {
		PrintSuccess("Running post_create hooks...")
		for _, hook := range profile.Hooks.PostCreate {
			hookDir := filepath.Join(wsPath, hook.Repo)
			if _, err := os.Stat(hookDir); os.IsNotExist(err) {
				PrintWarning("Skipping hook for %s: directory not found", hook.Repo)
				continue
			}

			cmd := exec.Command("sh", "-c", hook.Run)
			cmd.Dir = hookDir
			cmd.Stdout = os.Stderr
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				PrintWarning("Hook failed for %s (%s): %v", hook.Repo, hook.Run, err)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\nWorkspace ready: %s\n", wsPath)
	return nil
}

// RemoveWorkspace removes a workspace and optionally deletes its branches.
func RemoveWorkspace(cfg *Config, profile *Profile, profileName, name string, deleteBranch, force bool) error {
	wsPath := GetWorkspacePath(cfg, profileName, name)

	// Check if workspace exists
	if _, err := os.Stat(wsPath); os.IsNotExist(err) {
		return fmt.Errorf("workspace %q not found at %s", name, wsPath)
	}

	// If not forcing, check for uncommitted changes
	if !force {
		for _, repo := range profile.Repos {
			repoDir := filepath.Join(wsPath, repo.Name)
			if _, err := os.Stat(repoDir); os.IsNotExist(err) {
				continue
			}
			count, err := StatusChangedCount(repoDir)
			if err != nil {
				continue
			}
			if count > 0 {
				return fmt.Errorf("%s has uncommitted changes (%d files). Use --force to remove anyway", repo.Name, count)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\nRemoving workspace: %s/%s\n", profileName, name)

	for _, repo := range profile.Repos {
		repoDir := filepath.Join(wsPath, repo.Name)
		if _, err := os.Stat(repoDir); os.IsNotExist(err) {
			continue
		}

		// Remove worktree
		if err := WorktreeRemove(repo.Path, repoDir, force); err != nil {
			PrintError("%s  worktree removal failed: %v", repo.Name, err)
			continue
		}

		msg := "worktree removed"

		// Delete branch if requested
		if deleteBranch {
			if err := BranchDelete(repo.Path, name, force); err != nil {
				PrintWarning("%s  worktree removed, branch deletion failed: %v", repo.Name, err)
			} else {
				msg = "worktree removed, branch deleted"
			}
		}

		PrintSuccess("%s  %s", RepoNameStyle.Render(repo.Name), msg)
	}

	// Remove workspace directory
	if err := os.RemoveAll(wsPath); err != nil {
		return fmt.Errorf("failed to remove workspace directory: %w", err)
	}

	PrintSuccess("Directory cleaned up")
	return nil
}

// ListWorkspaces scans the base_dir for workspaces and returns their info.
// If profileName is empty, all profiles are scanned.
func ListWorkspaces(cfg *Config, profileName string) ([]WorkspaceInfo, error) {
	var workspaces []WorkspaceInfo

	profilesToScan := make(map[string]Profile)
	if profileName != "" {
		profile, ok := cfg.Profiles[profileName]
		if !ok {
			return nil, fmt.Errorf("profile %q not found", profileName)
		}
		profilesToScan[profileName] = profile
	} else {
		profilesToScan = cfg.Profiles
	}

	for pName, profile := range profilesToScan {
		profileDir := filepath.Join(cfg.BaseDir, pName)

		entries, err := os.ReadDir(profileDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read profile directory %s: %w", profileDir, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			wsName := entry.Name()
			wsPath := filepath.Join(profileDir, wsName)

			ws := WorkspaceInfo{
				ProfileName:   pName,
				WorkspaceName: wsName,
				Path:          wsPath,
			}

			for _, repo := range profile.Repos {
				repoDir := filepath.Join(wsPath, repo.Name)
				rs := RepoStatus{
					RepoName:    repo.Name,
					DefaultBase: repo.GetDefaultBase(),
				}

				if _, err := os.Stat(repoDir); os.IsNotExist(err) {
					rs.Exists = false
					ws.RepoStatuses = append(ws.RepoStatuses, rs)
					continue
				}

				rs.Exists = true

				branch, err := CurrentBranch(repoDir)
				if err == nil {
					rs.BranchName = branch
				}

				count, err := StatusChangedCount(repoDir)
				if err == nil {
					rs.ChangedCount = count
				}

				ahead, behind, err := AheadBehind(repo.Path, rs.DefaultBase, branch)
				if err == nil {
					rs.Ahead = ahead
					rs.Behind = behind
				}

				ws.RepoStatuses = append(ws.RepoStatuses, rs)
			}

			workspaces = append(workspaces, ws)
		}
	}

	return workspaces, nil
}

// WorkspaceLabels returns a list of formatted workspace labels for fzf selection.
func WorkspaceLabels(workspaces []WorkspaceInfo) []string {
	labels := make([]string, 0, len(workspaces))
	for _, ws := range workspaces {
		parts := []string{fmt.Sprintf("%s/%s", ws.ProfileName, ws.WorkspaceName)}
		for _, rs := range ws.RepoStatuses {
			if !rs.Exists {
				parts = append(parts, fmt.Sprintf("[%s: missing]", rs.RepoName))
				continue
			}
			parts = append(parts, FormatRepoStatusCompact(rs.RepoName, rs.ChangedCount))
		}
		labels = append(labels, strings.Join(parts, "     "))
	}
	return labels
}

// ParseWorkspaceLabel extracts profile and workspace name from a label.
func ParseWorkspaceLabel(label string) (profileName, workspaceName string, err error) {
	// Label format: "profile/workspace     [repo: status] ..."
	parts := strings.Fields(label)
	if len(parts) == 0 {
		return "", "", fmt.Errorf("empty label")
	}

	slashParts := strings.SplitN(parts[0], "/", 2)
	if len(slashParts) != 2 {
		return "", "", fmt.Errorf("invalid label format: %q", label)
	}

	return slashParts[0], slashParts[1], nil
}

// cleanupWorkspace attempts to clean up a partially created workspace.
func cleanupWorkspace(cfg *Config, profile *Profile, profileName, name string) {
	wsPath := GetWorkspacePath(cfg, profileName, name)
	for _, repo := range profile.Repos {
		repoDir := filepath.Join(wsPath, repo.Name)
		if _, err := os.Stat(repoDir); err == nil {
			_ = WorktreeRemove(repo.Path, repoDir, true)
		}
	}
	_ = os.RemoveAll(wsPath)
}
