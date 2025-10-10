package config

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/jmuk/sylvan/pkg/chat"
	"github.com/jmuk/sylvan/pkg/chat/claude"
	"github.com/jmuk/sylvan/pkg/chat/gemini"
	"github.com/jmuk/sylvan/pkg/tools"
)

type ModelType string

const (
	ModelTypeGemini ModelType = "gemini"
	ModelTypeClaude ModelType = "claude"
)

type ModelConfig interface {
	Name() string
	// TODO: add common interface for chat
	NewAgent(
		ctx context.Context,
		tools []tools.ToolDefinition,
	) (chat.Agent, error)
}

type Config struct {
	ModelConfigs []map[string]any `toml:"model_configs"`
	MCP          []MCPConfig      `toml:"mcp"`
	ModelName    string           `toml:"model_name"`
	LogLevel     slog.Level       `toml:"log_level"`
}

func modelConfigFrom(m map[string]any) (ModelConfig, error) {
	mtData, ok := m["type"]
	if !ok {
		return nil, fmt.Errorf("missing field type for model config")
	}
	mtStr, ok := mtData.(string)
	if !ok {
		return nil, fmt.Errorf("type mismatch for type field: want string got %T", mtData)
	}
	marshaled, err := toml.Marshal(m)
	if err != nil {
		return nil, err
	}
	switch ModelType(mtStr) {
	case ModelTypeGemini:
		geminiConfig := &gemini.Config{}
		if err := toml.Unmarshal(marshaled, geminiConfig); err != nil {
			return nil, err
		}
		return geminiConfig, nil
	case ModelTypeClaude:
		return claude.ParseConfig(marshaled)
	}
	return nil, fmt.Errorf("unknown model type %s", mtStr)
}

func (c *Config) NewAgent(
	ctx context.Context,
	toolDefs []tools.ToolDefinition,
) (chat.Agent, error) {
	for _, modelConfig := range c.ModelConfigs {
		cfg, err := modelConfigFrom(modelConfig)
		if err != nil {
			log.Printf("Failed to parse model config: %s", err)
			continue
		}
		if cfg.Name() == c.ModelName {
			return cfg.NewAgent(ctx, toolDefs)
		}
	}
	return nil, errors.New("model config not found")
}

func LoadConfig() (*Config, error) {
	defaultModel := &gemini.Config{
		ConfigName: "gemini",
		ModelName:  "gemini-2.5-flash",
	}
	marshaled, err := toml.Marshal(defaultModel)
	if err != nil {
		return nil, err
	}
	data := map[string]any{}
	if err := toml.Unmarshal(marshaled, &data); err != nil {
		return nil, err
	}
	config := &Config{
		ModelConfigs: []map[string]any{data},
		ModelName:    "gemini",
		LogLevel:     slog.LevelInfo,
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	configDir := filepath.Join(userConfigDir, "sylvan")
	if _, err := os.Stat(configDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	configFile := filepath.Join(configDir, "config.toml")
	if _, err := os.Stat(configFile); err != nil {
		if os.IsNotExist(err) {
			data, err := toml.Marshal(config)
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(configFile, data, 0644); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, err
		}
		loadedConfig := &Config{}
		if err := toml.Unmarshal(data, loadedConfig); err != nil {
			return nil, err
		}
		config = loadedConfig
	}

	// TODO: load the local config file.
	return config, nil
}
