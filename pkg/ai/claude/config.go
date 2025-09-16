package claude

import (
	"context"

	"github.com/jmuk/sylvan/pkg/ai"
	"github.com/jmuk/sylvan/pkg/tools"
)

type Config struct {
	BaseURL          string `toml:"base_url"`
	APIKey           string `toml:"api_key"`
	APIKeyFromEnv    string `toml:"api_key_env"`
	AnthropicVersion string `toml:"anthropic_version"`
	Model            string `toml:"model"`
	MaxTokens        int    `toml:"max_tokens"`
}

func (c *Config) Name() string {
	return "claude"
}

func (c *Config) NewChat(ctx context.Context, toolDefs []tools.ToolDefinition) (ai.Agent, error) {
	return New(c, toolDefs)
}

func DefaultConfig() *Config {
	return &Config{
		BaseURL:          "https://api.anthropic.com/",
		APIKeyFromEnv:    "ANTHROPIC_API_KEY",
		AnthropicVersion: "2023-06-01",
		// TODO -- possibly use models API?
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 65536,
	}
}
