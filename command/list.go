package command

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/1126buri/mangrove"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all workspaces",
	Long:    "List all workspaces with their status (clean/changed).",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaces, err := mangrove.ListWorkspaces(cfg, profileFlag)
		if err != nil {
			return err
		}

		if len(workspaces) == 0 {
			fmt.Fprintln(os.Stderr, "No workspaces found.")
			return nil
		}

		// Group by profile
		grouped := make(map[string][]mangrove.WorkspaceInfo)
		for _, ws := range workspaces {
			grouped[ws.ProfileName] = append(grouped[ws.ProfileName], ws)
		}

		// Sort profile names
		profileNames := make([]string, 0, len(grouped))
		for name := range grouped {
			profileNames = append(profileNames, name)
		}
		sort.Strings(profileNames)

		for _, pName := range profileNames {
			wsList := grouped[pName]

			// Sort workspaces by name
			sort.Slice(wsList, func(i, j int) bool {
				return wsList[i].WorkspaceName < wsList[j].WorkspaceName
			})

			fmt.Fprintf(os.Stderr, "\n%s:\n", mangrove.ProfileNameStyle.Render(pName))

			for _, ws := range wsList {
				name := fmt.Sprintf("  %-20s", ws.WorkspaceName)
				var statuses []string
				for _, rs := range ws.RepoStatuses {
					if !rs.Exists {
						statuses = append(statuses, fmt.Sprintf("[%s: missing]", rs.RepoName))
						continue
					}
					statuses = append(statuses, mangrove.FormatRepoStatusCompact(rs.RepoName, rs.ChangedCount))
				}
				fmt.Fprintf(os.Stderr, "%s %s\n", name, strings.Join(statuses, " "))
			}
		}

		fmt.Fprintln(os.Stderr)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
