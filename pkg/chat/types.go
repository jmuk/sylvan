package chat

import (
	"context"
	"iter"

	"github.com/jmuk/sylvan/pkg/tools"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type FunctionCall struct {
	ID   string
	Name string
	Args map[string]any
}

type FunctionResponse struct {
	ID       string
	Name     string
	Response map[string]any
	Error    error
}

type Part struct {
	Thought           bool
	Text              string
	ThinkingSignature string
	FunctionCall      *FunctionCall
	FunctionResponse  *FunctionResponse
}

type Agent interface {
	SendMessageStream(ctx context.Context, messages []Part) iter.Seq2[*Part, error]
}

type AgentFactory interface {
	NewAgent(
		ctx context.Context,
		historyFile string,
		toolDefs []tools.ToolDefinition,
	) (Agent, error)
}
