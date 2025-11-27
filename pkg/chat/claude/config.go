package claude

import (
	"context"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/jmuk/sylvan/pkg/chat/agent"
	"github.com/jmuk/sylvan/pkg/tools"
)

type Config struct {
	ConfigName       string `toml:"name"`
	BaseURL          string `toml:"base_url"`
	APIKey           string `toml:"api_key"`
	APIKeyFromEnv    string `toml:"api_key_env"`
	AnthropicVersion string `toml:"anthropic_version"`
	MaxTokens        int    `toml:"max_tokens"`
}

func (c *Config) Name() string {
	return c.ConfigName
}

func (c *Config) apiKey() (string, error) {
	if c.APIKeyFromEnv != "" {
		apiKey := os.Getenv(c.APIKeyFromEnv)
		if apiKey == "" {
			return "", fmt.Errorf("env variable %s not defined", c.APIKeyFromEnv)
		}
		return apiKey, nil
	}
	if c.APIKey == "" {
		return "", fmt.Errorf("either api_key or api_key_env msut be specified")
	}
	return c.APIKey, nil
}

func (c *Config) NewAgent(ctx context.Context, modelName string, systemPrompt string, toolDefs []tools.ToolDefinition) (agent.Agent, error) {
	return New(ctx, c, modelName, systemPrompt, toolDefs)
}

func ParseConfig(data []byte) (*Config, error) {
	config := *DefaultConfig()
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func DefaultConfig() *Config {
	return &Config{
		BaseURL:          "https://api.anthropic.com/",
		APIKeyFromEnv:    "ANTHROPIC_API_KEY",
		AnthropicVersion: "2023-06-01",
		MaxTokens:        32768,
	}
}
