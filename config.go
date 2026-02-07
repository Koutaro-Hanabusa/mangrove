package mangrove

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Hook represents a post-create hook to run after workspace creation.
type Hook struct {
	Repo string `mapstructure:"repo" yaml:"repo"`
	Run  string `mapstructure:"run"  yaml:"run"`
}

// Hooks holds the different hook stages.
type Hooks struct {
	PostCreate []Hook `mapstructure:"post_create" yaml:"post_create"`
}

// Repo represents a single git repository within a profile.
type Repo struct {
	Name        string `mapstructure:"name"         yaml:"name"`
	Path        string `mapstructure:"path"         yaml:"path"`
	DefaultBase string `mapstructure:"default_base" yaml:"default_base"`
}

// Profile represents a named collection of repositories and their hooks.
type Profile struct {
	Repos []Repo `mapstructure:"repos" yaml:"repos"`
	Hooks Hooks  `mapstructure:"hooks" yaml:"hooks"`
}

// Config is the top-level configuration structure.
type Config struct {
	BaseDir        string             `mapstructure:"base_dir"        yaml:"base_dir"`
	DefaultProfile string             `mapstructure:"default_profile" yaml:"default_profile"`
	Profiles       map[string]Profile `mapstructure:"profiles"        yaml:"profiles"`
}

// ExpandPath expands ~ to the user's home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// LoadConfig reads the configuration from ~/.config/mgv/config.yaml.
func LoadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "mgv")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	// Set defaults
	viper.SetDefault("base_dir", "~/mgv-workspaces")
	viper.SetDefault("default_profile", "")
	viper.SetDefault("profiles", map[string]Profile{})

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("config file not found at %s/config.yaml: %w", configDir, err)
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Expand paths
	cfg.BaseDir = ExpandPath(cfg.BaseDir)
	for profileName, profile := range cfg.Profiles {
		for i := range profile.Repos {
			profile.Repos[i].Path = ExpandPath(profile.Repos[i].Path)
		}
		cfg.Profiles[profileName] = profile
	}

	return &cfg, nil
}

// GetProfile returns the named profile from the config.
// If name is empty, returns the default profile.
func (c *Config) GetProfile(name string) (*Profile, string, error) {
	if name == "" {
		name = c.DefaultProfile
	}
	if name == "" {
		return nil, "", fmt.Errorf("no profile specified and no default_profile set in config")
	}
	profile, ok := c.Profiles[name]
	if !ok {
		return nil, "", fmt.Errorf("profile %q not found in config", name)
	}
	return &profile, name, nil
}

// ProfileNames returns a sorted list of profile names.
func (c *Config) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	return names
}

// GetRepoDefaultBase returns the default base branch for a repo,
// falling back to "main" if not set.
func (r *Repo) GetDefaultBase() string {
	if r.DefaultBase != "" {
		return r.DefaultBase
	}
	return "main"
}
