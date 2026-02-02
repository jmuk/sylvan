// package agent defines the interfaces of an LLM agent.
package agent

import (
	"context"
	"iter"

	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/jmuk/sylvan/pkg/tools"
)

// Agent is an LLM agent.
type Agent interface {
	// send messages and returns the stream of the response.
	SendMessageStream(ctx context.Context, messages []parts.Part) iter.Seq2[*parts.Part, error]
}

// Factory creates a new agent.
type Factory interface {
	// NewAgent creates a new agent instance.
	NewAgent(
		ctx context.Context,
		systemPrompt string,
		toolDefs []tools.ToolDefinition,
	) (Agent, error)
}
