package command

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Koutaro-Hanabusa/mangrove"
	"github.com/spf13/cobra"
)

var (
	applyYes    bool
	applyRepos  []string
	applyMethod string
	applyBase   string
	applyBranch string
)

var applyCmd = &cobra.Command{
	Use:   "apply [workspace-name]",
	Short: "Apply worktree changes to the original repo",
	Long: `Apply changes from a workspace worktree back to the original repository.

Supports two methods:
  stash  - Stash uncommitted changes in worktree, then pop them on a new branch in the original repo.
  merge  - Merge the worktree branch into a new branch in the original repo.

Examples:
  mgv apply
  mgv apply feature-login
  mgv apply feature-login --method stash --base main --branch apply/feature-login
  mgv apply feature-login --repo api --repo web
  mgv apply feature-login -y -m merge -b main`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		interactive := !applyYes

		var profileName, wsName string

		if len(args) > 0 {
			wsName = args[0]
			_, pName, err := resolveProfile(profileFlag == "")
			if err != nil {
				return err
			}
			profileName = pName
		} else {
			if !mangrove.IsFzfAvailable() {
				return fmt.Errorf("fzf is required for interactive mode. Install with: brew install fzf")
			}

			workspaces, err := mangrove.ListWorkspaces(cfg, profileFlag)
			if err != nil {
				return err
			}
			if len(workspaces) == 0 {
				return fmt.Errorf("no workspaces found")
			}

			labels := mangrove.WorkspaceLabels(workspaces)
			selected, err := mangrove.SelectWorkspace(labels)
			if err != nil {
				return err
			}

			pName, wName, err := mangrove.ParseWorkspaceLabel(selected)
			if err != nil {
				return err
			}
			profileName = pName
			wsName = wName
		}

		profile, _, err := cfg.GetProfile(profileName)
		if err != nil {
			return err
		}

		wsPath := mangrove.GetWorkspacePath(cfg, profileName, wsName)

		// Build set of target repos if --repo is specified
		repoFilter := make(map[string]bool)
		for _, r := range applyRepos {
			repoFilter[r] = true
		}

		fmt.Fprintf(os.Stderr, "\nApplying workspace: %s/%s\n",
			mangrove.ProfileNameStyle.Render(profileName),
			mangrove.RepoNameStyle.Render(wsName),
		)

		for _, repo := range profile.Repos {
			if len(repoFilter) > 0 && !repoFilter[repo.Name] {
				continue
			}

			wtDir := filepath.Join(wsPath, repo.Name)
			if _, err := os.Stat(wtDir); os.IsNotExist(err) {
				mangrove.PrintWarning("%s: worktree not found, skipping", repo.Name)
				continue
			}

			fmt.Fprintf(os.Stderr, "\n[%s]\n", mangrove.RepoNameStyle.Render(repo.Name))

			// Show status
			branch, err := mangrove.CurrentBranch(wtDir)
			if err != nil {
				mangrove.PrintError("%s: failed to get branch: %v", repo.Name, err)
				continue
			}

			changedCount, err := mangrove.StatusChangedCount(wtDir)
			if err != nil {
				mangrove.PrintError("%s: failed to get status: %v", repo.Name, err)
				continue
			}

			ahead, behind, _ := mangrove.AheadBehind(repo.Path, repo.GetDefaultBase(), branch)
			mangrove.PrintRepoStatus(repo.Name, branch, changedCount, ahead, behind, repo.GetDefaultBase())

			// Guard: check original repo for uncommitted changes
			origStatus, err := mangrove.StatusPorcelain(repo.Path)
			if err != nil {
				mangrove.PrintError("%s: failed to check original repo status: %v", repo.Name, err)
				continue
			}
			if origStatus != "" {
				mangrove.PrintError("%s: original repo has uncommitted changes. Please commit or stash first.", repo.Name)
				continue
			}

			// Select method
			method := applyMethod
			if method == "" {
				if interactive {
					if !mangrove.IsFzfAvailable() {
						return fmt.Errorf("fzf is required for interactive mode")
					}
					selected, err := mangrove.SelectMethod(repo.Name)
					if err != nil {
						return err
					}
					method = selected
				} else {
					return fmt.Errorf("--method is required in non-interactive mode")
				}
			}

			if method == "skip" {
				mangrove.PrintInfo("Skipped %s", repo.Name)
				continue
			}

			// Guard: stash requires uncommitted changes
			if method == "stash" && changedCount == 0 {
				mangrove.PrintWarning("%s: no uncommitted changes to stash, skipping", repo.Name)
				continue
			}

			// Guard: merge requires commits ahead
			if method == "merge" && ahead == 0 {
				mangrove.PrintWarning("%s: no commits ahead to merge, skipping", repo.Name)
				continue
			}

			// Select base branch
			baseBranch := applyBase
			if baseBranch == "" {
				if interactive {
					prompt := fmt.Sprintf("[%s] Base branch:", repo.Name)
					selected, err := mangrove.SelectBranch(repo.Path, prompt, repo.GetDefaultBase())
					if err != nil {
						return err
					}
					baseBranch = selected
				} else {
					baseBranch = repo.GetDefaultBase()
				}
			}

			// Determine new branch name
			newBranch := applyBranch
			if newBranch == "" {
				defaultName := fmt.Sprintf("apply/%s", wsName)
				if interactive {
					fmt.Fprintf(os.Stderr, "  ? New branch name [%s]: ", defaultName)
					reader := bufio.NewReader(os.Stdin)
					input, err := reader.ReadString('\n')
					if err != nil {
						return fmt.Errorf("failed to read branch name: %w", err)
					}
					input = strings.TrimSpace(input)
					if input != "" {
						newBranch = input
					} else {
						newBranch = defaultName
					}
				} else {
					newBranch = defaultName
				}
			}

			// Execute
			switch method {
			case "stash":
				if err := applyStash(wtDir, repo.Path, newBranch, baseBranch, repo.Name); err != nil {
					mangrove.PrintError("%s: stash apply failed: %v", repo.Name, err)
					continue
				}
			case "merge":
				if err := applyMerge(wtDir, repo.Path, branch, newBranch, baseBranch, repo.Name); err != nil {
					mangrove.PrintError("%s: merge apply failed: %v", repo.Name, err)
					continue
				}
			default:
				mangrove.PrintError("%s: unknown method %q", repo.Name, method)
				continue
			}

			mangrove.PrintSuccess("%s: applied via %s → %s (base: %s)", repo.Name, method, newBranch, baseBranch)
		}

		fmt.Fprintln(os.Stderr)
		return nil
	},
}

// applyStash applies worktree changes via stash push/pop.
func applyStash(wtDir, repoPath, newBranch, baseBranch, repoName string) error {
	// Step 1: stash push in worktree
	msg := fmt.Sprintf("mgv-apply: %s", newBranch)
	if err := mangrove.StashPush(wtDir, msg); err != nil {
		return fmt.Errorf("stash push failed: %w", err)
	}

	// Step 2: create new branch in original repo
	if err := mangrove.CheckoutNewBranch(repoPath, newBranch, baseBranch); err != nil {
		// Rollback: pop stash back in worktree
		mangrove.PrintWarning("%s: rolling back stash to worktree...", repoName)
		_ = mangrove.StashPop(wtDir)
		return fmt.Errorf("checkout -b failed: %w", err)
	}

	// Step 3: pop stash in original repo (shared .git)
	if err := mangrove.StashPop(repoPath); err != nil {
		// Rollback: go back to previous branch, delete new branch, pop stash in worktree
		mangrove.PrintWarning("%s: rolling back...", repoName)
		_ = mangrove.CheckoutBranch(repoPath, baseBranch)
		_ = mangrove.BranchDelete(repoPath, newBranch, true)
		_ = mangrove.StashPop(wtDir)
		return fmt.Errorf("stash pop failed: %w", err)
	}

	return nil
}

// applyMerge applies worktree changes via merge.
func applyMerge(wtDir, repoPath, wtBranch, newBranch, baseBranch, repoName string) error {
	// Record current branch of original repo
	origBranch, err := mangrove.CurrentBranch(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Step 1: create new branch in original repo
	if err := mangrove.CheckoutNewBranch(repoPath, newBranch, baseBranch); err != nil {
		return fmt.Errorf("checkout -b failed: %w", err)
	}

	// Step 2: merge worktree branch
	if err := mangrove.Merge(repoPath, wtBranch); err != nil {
		// Rollback: go back to original branch, delete new branch
		mangrove.PrintWarning("%s: rolling back...", repoName)
		_ = mangrove.CheckoutBranch(repoPath, origBranch)
		_ = mangrove.BranchDelete(repoPath, newBranch, true)
		return fmt.Errorf("merge failed: %w", err)
	}

	// Step 3: return to original branch
	if err := mangrove.CheckoutBranch(repoPath, origBranch); err != nil {
		mangrove.PrintWarning("%s: failed to return to %s: %v", repoName, origBranch, err)
	}

	return nil
}

func init() {
	applyCmd.Flags().BoolVarP(&applyYes, "yes", "y", false, "non-interactive mode")
	applyCmd.Flags().StringSliceVarP(&applyRepos, "repo", "r", nil, "target specific repos")
	applyCmd.Flags().StringVarP(&applyMethod, "method", "m", "", "apply method: stash or merge")
	applyCmd.Flags().StringVarP(&applyBase, "base", "b", "", "base branch for new branch")
	applyCmd.Flags().StringVar(&applyBranch, "branch", "", "new branch name")
	rootCmd.AddCommand(applyCmd)
}
