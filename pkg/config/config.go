package config

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config keeps the configuration.
type Config struct {
	// Backends is the list of the backend / models.
	// The actual format of each item is defined in the backend's
	// config struct.
	Backends []map[string]any `toml:"backends"`
	// MCP is the list of MCPs the agent can interact with.
	MCP []MCPConfig `toml:"mcp"`
	// The name of the backend to be used.
	BackendName string `toml:"backend_name"`
	// The name of the LLM to be used.
	ModelName string `toml:"model_name"`
	// The output log level.
	LogLevel slog.Level `toml:"log_level"`
}

// ConfigFile returns the path of the config file.
func ConfigFile(basePath string) string {
	return filepath.Join(basePath, "config.toml")
}

// DefaultConfigFile returns the config file for the user.
func DefaultConfigFile() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return ConfigFile(filepath.Join(userConfigDir, "sylvan")), nil
}

// DefaultConfig returns the default config when no config is set.
// TODO: add a startup wizard instead of this.
func DefaultConfig() (*Config, error) {
	return &Config{
		Backends:    []map[string]any{},
		BackendName: "gemini",
		ModelName:   "gemini-2.5-flash",
		LogLevel:    slog.LevelInfo,
	}, nil
}

func ensureConfigDir(configFile string) error {
	dirName := filepath.Dir(configFile)
	return os.MkdirAll(dirName, 0755)
}

func loadConfigFile(configFile string, config *Config) error {
	if err := ensureConfigDir(configFile); err != nil {
		return err
	}
	_, err := toml.DecodeFile(configFile, config)
	return err
}

// LoadConfigFile reads a file and load it into the Config struct.
func LoadConfigFile(configFile string) (*Config, error) {
	config := &Config{}
	if err := loadConfigFile(configFile, config); err != nil {
		return nil, err
	}
	return config, nil
}

// LoadConfigFiles loads the given config files in the order
// and merges the result.
func LoadConfigFiles(paths ...string) (*Config, error) {
	// First, load the default config.
	defaultPath, err := DefaultConfigFile()
	if err != nil {
		return nil, err
	}
	config, err := LoadConfigFile(defaultPath)
	if os.IsNotExist(err) {
		config, err = DefaultConfig()
		if err != nil {
			return nil, err
		}
		// Store the default config into the disk if missing.
		if err := EditConfig(defaultPath, func(*Config) (*Config, error) { return config, nil }); err != nil {
			return nil, err
		}
	}
	for _, p := range paths {
		if err := loadConfigFile(p, config); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
	}
	return config, nil
}

// EditConfig modifies the content of the config specified in the file.
func EditConfig(configFile string, edit func(c *Config) (*Config, error)) error {
	loadedConfig := &Config{}
	if _, err := toml.DecodeFile(configFile, loadedConfig); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	if err := ensureConfigDir(configFile); err != nil {
		return err
	}
	newConfig, err := edit(loadedConfig)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(configFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(newConfig); err != nil {
		return err
	}
	return nil
}
