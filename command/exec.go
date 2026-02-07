package command

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/1126buri/mangrove"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec [workspace-name] -- <command> [args...]",
	Short: "Execute a command in each repo of a workspace",
	Long: `Execute a command in each repo worktree of a workspace.

Examples:
  mgv exec -- git status
  mgv exec feature-login -- git status
  mgv exec feature-login --profile project-a -- make build`,
	DisableFlagParsing: false,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Split args at "--"
		var wsNameArg string
		var cmdArgs []string

		dashIdx := cmd.ArgsLenAtDash()
		if dashIdx >= 0 {
			preArgs := args[:dashIdx]
			cmdArgs = args[dashIdx:]

			if len(preArgs) > 0 {
				wsNameArg = preArgs[0]
			}
		} else {
			// No "--" separator; treat first arg as workspace name if provided
			if len(args) > 0 {
				wsNameArg = args[0]
				cmdArgs = args[1:]
			}
		}

		if len(cmdArgs) == 0 {
			return fmt.Errorf("no command specified. Use: mgv exec [workspace] -- <command>")
		}

		var profileName, wsName string

		if wsNameArg != "" {
			wsName = wsNameArg
			_, pName, err := resolveProfile(profileFlag == "")
			if err != nil {
				return err
			}
			profileName = pName
		} else {
			// Interactive workspace selection
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

		// Execute command in each repo worktree
		for _, repo := range profile.Repos {
			repoDir := filepath.Join(wsPath, repo.Name)
			if _, err := os.Stat(repoDir); os.IsNotExist(err) {
				mangrove.PrintWarning("Skipping %s: directory not found", repo.Name)
				continue
			}

			fmt.Fprintf(os.Stderr, "\n[%s]\n", mangrove.RepoNameStyle.Render(repo.Name))

			execCmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
			execCmd.Dir = repoDir
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			execCmd.Stdin = os.Stdin

			if err := execCmd.Run(); err != nil {
				mangrove.PrintError("Command failed in %s: %v", repo.Name, err)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}
