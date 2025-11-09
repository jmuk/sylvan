package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jmuk/sylvan/pkg/chat/agent"
	"github.com/jmuk/sylvan/pkg/tools"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
)

type Config struct {
	ConfigName    string `toml:"name"`
	BaseURL       string `toml:"base_url"`
	APIKey        string `toml:"api_key"`
	APIKeyFromEnv string `toml:"api_key_env"`
	ModelName     string `toml:"model_name"`
}

func (c *Config) Name() string {
	return c.ConfigName
}

func convertToolDef(d tools.ToolDefinition) (responses.ToolUnionParam, error) {
	rsch := d.RequestSchema()
	encoded, err := json.Marshal(rsch)
	if err != nil {
		return responses.ToolUnionParam{}, err
	}
	parameters := map[string]any{}
	if err := json.Unmarshal(encoded, &parameters); err != nil {
		return responses.ToolUnionParam{}, err
	}
	return responses.ToolUnionParam{
		OfFunction: &responses.FunctionToolParam{
			Parameters:  parameters,
			Name:        d.Name(),
			Description: param.NewOpt(d.Description()),
			Type:        "function",
		},
	}, nil
}

func (c *Config) NewAgent(
	ctx context.Context,
	systemPrompt string,
	toolDefs []tools.ToolDefinition,
) (agent.Agent, error) {
	var opts []option.RequestOption
	if c.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(c.BaseURL))
	}
	if c.APIKeyFromEnv != "" {
		apikey := os.Getenv(c.APIKeyFromEnv)
		if apikey == "" {
			return nil, fmt.Errorf("environment variable %s not found", c.APIKeyFromEnv)
		}
		opts = append(opts, option.WithAPIKey(apikey))
	} else if c.APIKey != "" {
		opts = append(opts, option.WithAPIKey(c.APIKey))
	}
	var toolParams []responses.ToolUnionParam
	for _, tdef := range toolDefs {
		toolParam, err := convertToolDef(tdef)
		if err != nil {
			return nil, err
		}
		toolParams = append(toolParams, toolParam)
	}
	return &Agent{
		client:       responses.NewResponseService(opts...),
		model:        c.ModelName,
		systemPrompt: systemPrompt,
		tools:        toolParams,
	}, nil
}
