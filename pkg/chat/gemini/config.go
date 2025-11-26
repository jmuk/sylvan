package gemini

import (
	"context"
	"strings"

	"github.com/jmuk/sylvan/pkg/chat/agent"
	"github.com/jmuk/sylvan/pkg/session"
	"github.com/jmuk/sylvan/pkg/tools"
	"google.golang.org/genai"
)

type Config struct {
	ConfigName      string `toml:"name"`
	APIKey          string `toml:"api_key,omitempty"`
	Backend         string `toml:"backend,omitempty"`
	Project         string `toml:"project,omitempty"`
	Location        string `toml:"location,omitempty"`
	excludeThoughts bool   `toml:"excludeThoughts,omitempty"`
}

func (gc *Config) Name() string {
	return gc.ConfigName
}

func (gc *Config) clientConfig() *genai.ClientConfig {
	backend := genai.BackendUnspecified
	if gc.Backend == genai.BackendGeminiAPI.String() {
		backend = genai.BackendGeminiAPI
	} else if gc.Backend == genai.BackendVertexAI.String() {
		backend = genai.BackendVertexAI
	}
	return &genai.ClientConfig{
		APIKey:   gc.APIKey,
		Backend:  backend,
		Project:  gc.Project,
		Location: gc.Location,
	}
}

func (gc *Config) NewAgent(
	ctx context.Context,
	modelName string,
	systemPrompt string,
	toolDefs []tools.ToolDefinition,
) (agent.Agent, error) {
	return New(ctx, modelName, gc.clientConfig(), systemPrompt, toolDefs, !gc.excludeThoughts)
}

func (gc *Config) Models(ctx context.Context) ([]string, error) {
	l, err := session.LoggerFromContext(ctx, "gemini")
	if err != nil {
		return nil, err
	}
	client, err := genai.NewClient(ctx, gc.clientConfig())
	if err != nil {
		return nil, err
	}
	var ms []string
	for m, err := range client.Models.All(ctx) {
		if err != nil {
			return nil, err
		}
		supported := false
		for _, act := range m.SupportedActions {
			if act == "generateContent" {
				supported = true
				break
			}
		}
		l.Debug("model", "modelName", m, "supported", supported)
		if !supported {
			continue
		}
		ms = append(ms, strings.TrimPrefix(m.Name, "models/"))
	}
	return ms, nil
}
