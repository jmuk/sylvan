package claude

import (
	"context"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/jmuk/sylvan/pkg/chat/agent"
	"github.com/jmuk/sylvan/pkg/tools"
)

// Config is the configuration for the claude client.
type Config struct {
	// The name of the config.
	ConfigName string `toml:"name"`

	// The URL to connect to.
	BaseURL string `toml:"base_url"`

	// The API key.
	APIKey string `toml:"api_key"`

	// The name of the environment variable that stores the API key.
	APIKeyFromEnv string `toml:"api_key_env"`

	// Anthropic version.
	AnthropicVersion string `toml:"anthropic_version"`

	// Max number of tokens.
	MaxTokens int `toml:"max_tokens"`
}

// Name implements chat.BackendConfig interface.
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

// NewAgent implements chat.BackendConfig interface.
func (c *Config) NewAgent(ctx context.Context, modelName string, systemPrompt string, toolDefs []tools.ToolDefinition) (agent.Agent, error) {
	return New(ctx, c, modelName, systemPrompt, toolDefs)
}

// ParseConfig parses the map data.
func ParseConfig(data []byte) (*Config, error) {
	config := *DefaultConfig()
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// DefaultConfig creates a default config.
func DefaultConfig() *Config {
	return &Config{
		BaseURL:          "https://api.anthropic.com/",
		APIKeyFromEnv:    "ANTHROPIC_API_KEY",
		AnthropicVersion: "2023-06-01",
		MaxTokens:        32768,
	}
}
