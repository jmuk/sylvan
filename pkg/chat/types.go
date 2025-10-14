package chat

import (
	"context"
	"iter"

	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/jmuk/sylvan/pkg/tools"
)

type Agent interface {
	SendMessageStream(ctx context.Context, messages []parts.Part) iter.Seq2[*parts.Part, error]
}

type AgentFactory interface {
	NewAgent(
		ctx context.Context,
		toolDefs []tools.ToolDefinition,
	) (Agent, error)
}
