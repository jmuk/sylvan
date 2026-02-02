package gemini

import (
	"context"
	"strings"

	"github.com/jmuk/sylvan/pkg/chat/agent"
	"github.com/jmuk/sylvan/pkg/session"
	"github.com/jmuk/sylvan/pkg/tools"
	"google.golang.org/genai"
)

// Config is the configuration data of Gemini Agent.
type Config struct {
	// The name of the config.
	ConfigName string `toml:"name"`

	// The API key.
	APIKey string `toml:"api_key,omitempty"`

	// The backend.
	Backend string `toml:"backend,omitempty"`

	// The GCP project.
	Project string `toml:"project,omitempty"`

	// The GCP location.
	Location string `toml:"location,omitempty"`

	// Whether exclude the thoughts or not.
	ExcludeThoughts bool `toml:"excludeThoughts,omitempty"`
}

// Name implements chat.BackendConfig interface.
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

// NewAgent implements chat.BackendConfig interface.
func (gc *Config) NewAgent(
	ctx context.Context,
	modelName string,
	systemPrompt string,
	toolDefs []tools.ToolDefinition,
) (agent.Agent, error) {
	return New(ctx, modelName, gc.clientConfig(), systemPrompt, toolDefs, !gc.ExcludeThoughts)
}

// Models implements chat.BackendConfig interface.
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
