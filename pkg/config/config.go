package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/jmuk/sylvan/pkg/ai"
	"github.com/jmuk/sylvan/pkg/ai/claude"
	"github.com/jmuk/sylvan/pkg/ai/gemini"
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
	NewChat(ctx context.Context,
		tools []tools.ToolDefinition,
	) (ai.Agent, error)
}

type Config struct {
	ModelConfigs []ModelConfig
	ModelName    string
	LogLevel     slog.Level
}

func (c *Config) MarshalTOML() ([]byte, error) {
	data := map[string]any{
		"model_name":    c.ModelName,
		"loglevel":      c.LogLevel,
		"model_configs": []map[string]any{},
	}
	for i, mc := range c.ModelConfigs {
		d, err := toml.Marshal(mc)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal %d-th config: %w", i, err)
		}
		m := map[string]any{}
		if err := toml.Unmarshal(d, &m); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %d-th config: %w", i, err)
		}
		m["type"] = ModelTypeGemini
		data["model_configs"] = append(data["model_configs"].([]map[string]any), m)
	}
	return toml.Marshal(data)
}

func (c *Config) UnmarshalTOML(input any) error {
	data, ok := input.(map[string]any)
	if !ok {
		return fmt.Errorf("type mismatched: want map[string]any got %T", input)
	}
	c.ModelName, ok = data["model_name"].(string)
	if !ok {
		return fmt.Errorf(`model_name: want string got %T`, data["model_name"])
	}
	if ll, ok := data["loglevel"]; !ok {
		c.LogLevel = slog.LevelInfo
	} else if lls, ok := ll.(string); !ok {
		return fmt.Errorf(`loglevel: want string got %T`, ll)
	} else {
		loglevel := &c.LogLevel
		if err := loglevel.UnmarshalText([]byte(lls)); err != nil {
			return fmt.Errorf(`failed to parse loglevel: %w`, err)
		}
	}

	modelsData, ok := data["model_configs"]
	if !ok {
		return nil
	}
	models, ok := modelsData.([]map[string]any)
	if !ok {
		return fmt.Errorf(`model_configs: want []map[string]any got %T`, modelsData)
	}
	for i, modelConfig := range models {
		mtData, ok := modelConfig["type"]
		if !ok {
			return fmt.Errorf("missing field type for model config")
		}
		mtStr, ok := mtData.(string)
		if !ok {
			return fmt.Errorf("type mismatch for type field: want string got %T", mtData)
		}
		marshaled, err := toml.Marshal(modelConfig)
		if err != nil {
			return fmt.Errorf(`failed to marshal %d-th config: %w`, i, err)
		}
		switch ModelType(mtStr) {
		case ModelTypeGemini:
			geminiConfig := &gemini.Config{}
			if err := toml.Unmarshal(marshaled, geminiConfig); err != nil {
				return fmt.Errorf(`failed to parse %d-th config: %w`, i, err)
			}
			c.ModelConfigs = append(c.ModelConfigs, geminiConfig)
		case ModelTypeClaude:
			claudeConfig, err := claude.ParseConfig(marshaled)
			if err != nil {
				return fmt.Errorf(`failed to aprse %d-th config: %w`, i, err)
			}
			c.ModelConfigs = append(c.ModelConfigs, claudeConfig)
		}
	}
	return nil
}

func (c *Config) NewChat(ctx context.Context,
	toolDefs []tools.ToolDefinition,
) (ai.Agent, error) {
	for _, modelConfig := range c.ModelConfigs {
		if modelConfig.Name() == c.ModelName {
			return modelConfig.NewChat(ctx, toolDefs)
		}
	}
	return nil, errors.New("model config not found")
}

func LoadConfig() (*Config, error) {
	config := &Config{
		ModelConfigs: []ModelConfig{
			&gemini.Config{
				ConfigName: "gemini",
				ModelName:  "gemini-2.5-flash",
			},
		},
		ModelName: "gemini",
		LogLevel:  slog.LevelInfo,
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
