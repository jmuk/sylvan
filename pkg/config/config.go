package config

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Backends    []map[string]any `toml:"backends"`
	MCP         []MCPConfig      `toml:"mcp"`
	BackendName string           `toml:"backend_name"`
	ModelName   string           `toml:"model_name"`
	LogLevel    slog.Level       `toml:"log_level"`
}

func ConfigFile(basePath string) string {
	return filepath.Join(basePath, "config.toml")
}

func DefaultConfigFile() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return ConfigFile(filepath.Join(userConfigDir, "sylvan")), nil
}

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

func LoadConfigFile(configFile string) (*Config, error) {
	config := &Config{}
	if err := loadConfigFile(configFile, config); err != nil {
		return nil, err
	}
	return config, nil
}

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
