package completion

import (
	"context"
	"encoding/json"

	"github.com/jmuk/sylvan/pkg/chat/agent"
	sylvanopenai "github.com/jmuk/sylvan/pkg/chat/openai"
	"github.com/jmuk/sylvan/pkg/session"
	"github.com/jmuk/sylvan/pkg/tools"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/shared"
)

type Config sylvanopenai.Config

func convertToolDef(d tools.ToolDefinition) (openai.ChatCompletionToolUnionParam, error) {
	rsch := d.RequestSchema()
	encoded, err := json.Marshal(rsch)
	if err != nil {
		return openai.ChatCompletionToolUnionParam{}, err
	}
	parameters := map[string]any{}
	if err := json.Unmarshal(encoded, &parameters); err != nil {
		return openai.ChatCompletionToolUnionParam{}, err
	}
	return openai.ChatCompletionToolUnionParam{
		OfFunction: &openai.ChatCompletionFunctionToolParam{
			Function: shared.FunctionDefinitionParam{
				Name:        d.Name(),
				Description: param.NewOpt(d.Description()),
				Parameters:  parameters,
			},
			Type: "function",
		},
	}, nil
}

// func parseHistoryFile(filename string) (param.Opt[string], error) {
// 	content, err := os.ReadFile(filename)
// 	if err != nil {
// 		if os.IsNotExist(err) {
// 			return param.Opt[string]{}, nil
// 		}
// 		return param.Opt[string]{}, err
// 	}

// 	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
// 	return param.NewOpt(lines[len(lines)-1]), nil
// }

func (c *Config) Name() string {
	return c.ConfigName
}

func (c *Config) NewAgent(
	ctx context.Context,
	modelName string,
	systemPrompt string,
	toolDefs []tools.ToolDefinition,
) (agent.Agent, error) {
	opts, err := (*sylvanopenai.Config)(c).GetOpts()
	if err != nil {
		return nil, err
	}

	var toolParams []openai.ChatCompletionToolUnionParam
	for _, tdef := range toolDefs {
		toolParam, err := convertToolDef(tdef)
		if err != nil {
			return nil, err
		}
		toolParams = append(toolParams, toolParam)
	}

	var historyFile string

	return &Agent{
		client:       openai.NewChatCompletionService(opts...),
		historyFile:  historyFile,
		modelName:    modelName,
		systemPrompt: systemPrompt,
		tools:        toolParams,
		history: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
		},
	}, nil
}

func (c *Config) Models(ctx context.Context) ([]string, error) {
	logger, err := session.LoggerFromContext(ctx, "openai")
	if err != nil {
		return nil, err
	}
	opts, err := (*sylvanopenai.Config)(c).GetOpts()
	if err != nil {
		return nil, err
	}
	client := openai.NewModelService(opts...)
	models, err := client.List(ctx)
	if err != nil {
		return nil, err
	}
	var results []string
	for models != nil {
		for _, m := range models.Data {
			logger.Debug("model", "model", m)
			results = append(results, m.ID)
		}
		models, err = models.GetNextPage()
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}
