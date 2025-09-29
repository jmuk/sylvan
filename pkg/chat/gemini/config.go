package gemini

import (
	"context"

	"github.com/jmuk/sylvan/pkg/chat"
	"github.com/jmuk/sylvan/pkg/tools"
	"google.golang.org/genai"
)

type Config struct {
	ConfigName      string `toml:"name"`
	ModelName       string `toml:"model_name"`
	APIKey          string `toml:"api_key,omitempty"`
	Backend         string `toml:"backend,omitempty"`
	Project         string `toml:"project,omitempty"`
	Location        string `toml:"location,omitempty"`
	excludeThoughts bool   `toml:"excludeThoughts,omitempty"`
}

func (gc *Config) Name() string {
	return gc.ConfigName
}

func (gc *Config) NewAgent(
	ctx context.Context,
	historyFile string,
	toolDefs []tools.ToolDefinition,
) (chat.Agent, error) {
	backend := genai.BackendUnspecified
	if gc.Backend == genai.BackendGeminiAPI.String() {
		backend = genai.BackendGeminiAPI
	} else if gc.Backend == genai.BackendVertexAI.String() {
		backend = genai.BackendVertexAI
	}
	return New(ctx, gc.ModelName, &genai.ClientConfig{
		APIKey:   gc.APIKey,
		Backend:  backend,
		Project:  gc.Project,
		Location: gc.Location,
	}, toolDefs, !gc.excludeThoughts)
}
