// package claude implements the client to Claude API.
package claude

import (
	"context"
	"iter"
	"log/slog"
	"net/url"

	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/jmuk/sylvan/pkg/session"
	"github.com/jmuk/sylvan/pkg/tools"
)

// Agent is an implementation of agent.Agent using Claude.
type Agent struct {
	history      []message
	modelName    string
	systemPrompt string

	url    *url.URL
	apiKey string

	config *Config

	tools []tool

	historyFile string
	logger      *slog.Logger
}

// SendMessageStream implements agent.Agent interface.
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

// New creates a new Claude agent.
func New(ctx context.Context, config *Config, modelName string, systemPrompt string, toolDefs []tools.ToolDefinition) (*Agent, error) {
	agent := &Agent{
		modelName:    modelName,
		systemPrompt: systemPrompt,
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
	var err error
	agent.apiKey, err = config.apiKey()
	if err != nil {
		return nil, err
	}
	agent.url, err = url.Parse(config.BaseURL)
	if err != nil {
		return nil, err
	}
	agent.url = agent.url.JoinPath("v1", "messages")
	return agent, nil
}
