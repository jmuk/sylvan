package config

import (
	"context"

	"github.com/jmuk/sylvan/pkg/ai"
	"github.com/jmuk/sylvan/pkg/tools"
	"google.golang.org/genai"
)

type GeminiConfig struct {
	ConfigName      string `toml:"name"`
	ModelName       string `toml:"model_name"`
	APIKey          string `toml:"api_key,omitempty"`
	Backend         string `toml:"backend,omitempty"`
	Project         string `toml:"project,omitempty"`
	Location        string `toml:"location,omitempty"`
	excludeThoughts bool   `toml:"excludeThoughts,omitempty"`
}

func (gc *GeminiConfig) hiddenMethod() {
}

func (gc *GeminiConfig) Name() string {
	return gc.ConfigName
}

func (gc *GeminiConfig) NewChat(
	ctx context.Context,
	toolDefs []tools.ToolDefinition,
) (*genai.Chat, error) {
	backend := genai.BackendUnspecified
	if gc.Backend == genai.BackendGeminiAPI.String() {
		backend = genai.BackendGeminiAPI
	} else if gc.Backend == genai.BackendVertexAI.String() {
		backend = genai.BackendVertexAI
	}
	return ai.NewGemini(ctx, gc.ModelName, &genai.ClientConfig{
		APIKey:   gc.APIKey,
		Backend:  backend,
		Project:  gc.Project,
		Location: gc.Location,
	}, toolDefs, !gc.excludeThoughts)
}
