package claude

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"os"
	"path"

	"github.com/jmuk/sylvan/pkg/ai"
	"github.com/jmuk/sylvan/pkg/tools"
)

type Agent struct {
	history      []message
	systemPrompt string

	url    *url.URL
	apiKey string

	config *Config

	tools []tool
}

func (a *Agent) SendMessageStream(ctx context.Context, messages []ai.Part) iter.Seq2[*ai.Part, error] {
	return func(yield func(*ai.Part, error) bool) {
		for _, msg := range messages {
			a.history = append(a.history, message{
				Part: msg,
				Role: ai.RoleUser,
			})
		}
		respBody, err := a.request()
		if err != nil {
			yield(nil, err)
			return
		}
		defer respBody.Close()
		ep := newEventProcessor(respBody, a)
		for part, err := range ep.processEvents() {
			if !yield(part, err) {
				break
			}
		}
	}
}

func New(config *Config, toolDefs []tools.ToolDefinition) (*Agent, error) {
	agent := &Agent{
		systemPrompt: ai.SystemPrompt,
		config:       config,
	}
	for _, toolDef := range toolDefs {
		agent.tools = append(agent.tools, tool{
			Name:        toolDef.Name(),
			Description: toolDef.Description(),
			InputSchema: toolDef.RequestSchema(),
		})
	}
	if config.APIKey == "" {
		if config.APIKeyFromEnv == "" {
			return nil, errors.New("either api-key or api-key-from-env must be specified")
		}
		agent.apiKey = os.Getenv(config.APIKeyFromEnv)
		if agent.apiKey == "" {
			return nil, fmt.Errorf("env variable %s not defined", config.APIKeyFromEnv)
		}
	} else {
		agent.apiKey = config.APIKey
	}
	var err error
	agent.url, err = url.Parse(config.BaseURL)
	if err != nil {
		return nil, err
	}
	agent.url.Path = path.Join(agent.url.Path, "/v1/messages")
	return agent, nil
}
