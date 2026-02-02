package chat

import (
	"context"
	"fmt"
	"log"

	"github.com/BurntSushi/toml"
	"github.com/jmuk/sylvan/pkg/chat/agent"
	"github.com/jmuk/sylvan/pkg/chat/claude"
	"github.com/jmuk/sylvan/pkg/chat/gemini"
	"github.com/jmuk/sylvan/pkg/chat/openai"
	"github.com/jmuk/sylvan/pkg/chat/openai/completion"
	"github.com/jmuk/sylvan/pkg/config"
	"github.com/jmuk/sylvan/pkg/tools"
)

// BackendType defines the list of the backends.
type BackendType string

const (
	// Gemini.
	BackendTypeGemini BackendType = "gemini"

	// Claude.
	BackendTypeClaude BackendType = "claude"

	// OpenAI responses API.
	BackendTypeOpenAI BackendType = "openai"

	// OpenAI completion API.
	BackendTypeOpenAIComp BackendType = "openai_comp"
)

// BackendConfig defines the interface common for the backend.
type BackendConfig interface {
	// The name of the config.
	Name() string
	// Creates a new agent.
	// TODO: add common interface for chat
	NewAgent(
		ctx context.Context,
		modelName string,
		systemPrompt string,
		tools []tools.ToolDefinition,
	) (agent.Agent, error)
	// Returns the list of models in the backend.
	Models(ctx context.Context) ([]string, error)
}

func backendFrom(m map[string]any) (BackendConfig, error) {
	mtData, ok := m["type"]
	if !ok {
		return nil, fmt.Errorf("	 field type for model config")
	}
	mtStr, ok := mtData.(string)
	if !ok {
		return nil, fmt.Errorf("type mismatch for type field: want string got %T", mtData)
	}
	marshaled, err := toml.Marshal(m)
	if err != nil {
		return nil, err
	}
	switch BackendType(mtStr) {
	case BackendTypeGemini:
		geminiConfig := &gemini.Config{}
		if err := toml.Unmarshal(marshaled, geminiConfig); err != nil {
			return nil, err
		}
		return geminiConfig, nil
	case BackendTypeClaude:
		return claude.ParseConfig(marshaled)
	case BackendTypeOpenAI:
		openaiConfig := &openai.Config{}
		if err := toml.Unmarshal(marshaled, openaiConfig); err != nil {
			return nil, err
		}
		return openaiConfig, nil
	case BackendTypeOpenAIComp:
		openaiConfig := &completion.Config{}
		if err := toml.Unmarshal(marshaled, openaiConfig); err != nil {
			return nil, err
		}
		return openaiConfig, nil
	}
	return nil, fmt.Errorf("unknown model type %s", mtStr)
}

func getBackend(c *config.Config) (BackendConfig, error) {
	for _, backend := range c.Backends {
		cfg, err := backendFrom(backend)
		if err != nil {
			log.Printf("Failed to parse model config: %s", err)
			continue
		}
		if cfg.Name() == c.BackendName {
			return cfg, nil
		}
	}
	return nil, fmt.Errorf("backend %s not found", c.BackendName)
}

func newAgent(
	ctx context.Context,
	c *config.Config,
	systemPrompt string,
	toolDefs []tools.ToolDefinition,
) (agent.Agent, error) {
	cfg, err := getBackend(c)
	if err != nil {
		return nil, err
	}
	return cfg.NewAgent(ctx, c.ModelName, systemPrompt, toolDefs)
}
