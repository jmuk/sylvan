package ai

import (
	"context"
	"iter"
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
	Thought          bool
	Text             string
	FunctionCall     *FunctionCall
	FunctionResponse *FunctionResponse
}

type Agent interface {
	SendMessageStream(ctx context.Context, messages []Part) iter.Seq2[*Part, error]
}
