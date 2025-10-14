package claude

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"net/url"
	"os"
	"path"

	"github.com/jmuk/sylvan/pkg/chat"
	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/jmuk/sylvan/pkg/session"
	"github.com/jmuk/sylvan/pkg/tools"
)

type Agent struct {
	history      []message
	systemPrompt string

	url    *url.URL
	apiKey string

	config *Config

	tools []tool

	historyFile string
	logger      *slog.Logger
}

func (a *Agent) SendMessageStream(ctx context.Context, messages []parts.Part) iter.Seq2[*parts.Part, error] {
	return func(yield func(*parts.Part, error) bool) {
		histSize := len(a.history)
		for _, msg := range messages {
			a.history = append(a.history, message{
				Part: &msg,
				Role: parts.RoleUser,
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
			a.logger.Info("got part", "part", part, "err", err)
			if !yield(part, err) {
				return
			}
		}
		a.saveContent(a.history[histSize:])
	}
}

func New(ctx context.Context, config *Config, toolDefs []tools.ToolDefinition) (*Agent, error) {
	agent := &Agent{
		systemPrompt: chat.SystemPrompt,
		config:       config,
		logger:       slog.New(slog.DiscardHandler),
	}
	if s, ok := session.FromContext(ctx); ok {
		agent.historyFile = s.HistoryFile()
		if err := agent.loadHistory(); err != nil {
			return nil, err
		}
		var err error
		agent.logger, err = s.GetLogger("claude")
		if err != nil {
			return nil, err
		}
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
