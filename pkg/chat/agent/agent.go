package agent

import (
	"context"
	"iter"

	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/jmuk/sylvan/pkg/tools"
)

type Agent interface {
	SendMessageStream(ctx context.Context, messages []parts.Part) iter.Seq2[*parts.Part, error]
}

type Factory interface {
	NewAgent(
		ctx context.Context,
		systemPrompt string,
		toolDefs []tools.ToolDefinition,
	) (Agent, error)
}
