package command

import (
	"fmt"
	"os"

	"github.com/1126buri/mangrove"
	"github.com/spf13/cobra"
)

var (
	// Version is set at build time.
	Version = "dev"

	// profileFlag is the global --profile flag.
	profileFlag string

	// cfg holds the loaded configuration.
	cfg *mangrove.Config
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:     "mgv",
	Short:   "mangrove - Multi-repo worktree manager",
	Long:    "mangrove (mgv) manages workspaces across multiple git repositories using git worktree.",
	Version: Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for completion commands
		if cmd.Name() == "completion" || cmd.Name() == "help" {
			return nil
		}

		var err error
		cfg, err = mangrove.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&profileFlag, "profile", "p", "", "profile name (overrides default_profile)")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// resolveProfile resolves the profile from the flag or config default.
func resolveProfile(interactive bool) (*mangrove.Profile, string, error) {
	if profileFlag != "" {
		return cfg.GetProfile(profileFlag)
	}

	if interactive && profileFlag == "" && cfg.DefaultProfile == "" {
		names := cfg.ProfileNames()
		if len(names) == 0 {
			return nil, "", fmt.Errorf("no profiles defined in config")
		}
		if len(names) == 1 {
			return cfg.GetProfile(names[0])
		}
		selected, err := mangrove.SelectProfile(names)
		if err != nil {
			return nil, "", err
		}
		return cfg.GetProfile(selected)
	}

	return cfg.GetProfile(profileFlag)
}
