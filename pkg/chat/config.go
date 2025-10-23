package chat

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/BurntSushi/toml"
	"github.com/jmuk/sylvan/pkg/chat/agent"
	"github.com/jmuk/sylvan/pkg/chat/claude"
	"github.com/jmuk/sylvan/pkg/chat/gemini"
	"github.com/jmuk/sylvan/pkg/config"
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
		systemPrompt string,
		tools []tools.ToolDefinition,
	) (agent.Agent, error)
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

func newAgent(
	ctx context.Context,
	c *config.Config,
	systemPrompt string,
	toolDefs []tools.ToolDefinition,
) (agent.Agent, error) {
	for _, modelConfig := range c.ModelConfigs {
		cfg, err := modelConfigFrom(modelConfig)
		if err != nil {
			log.Printf("Failed to parse model config: %s", err)
			continue
		}
		if cfg.Name() == c.ModelName {
			return cfg.NewAgent(ctx, systemPrompt, toolDefs)
		}
	}
	return nil, errors.New("model config not found")
}
