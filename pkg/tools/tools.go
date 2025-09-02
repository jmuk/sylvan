package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/manifoldco/promptui"
)

type loggerKeyType struct{}

var loggerKey loggerKeyType = loggerKeyType{}

func getLogger(ctx context.Context) *slog.Logger {
	logger := ctx.Value(loggerKey)
	if logger == nil {
		return nil
	}
	return logger.(*slog.Logger)
}

type ToolRunner struct {
	defsMap map[string]ToolDefinition
	logger  *slog.Logger
}

func NewToolRunner(h slog.Handler, defs []ToolDefinition) (*ToolRunner, error) {
	m := make(map[string]ToolDefinition, len(defs))
	for _, d := range defs {
		if _, ok := m[d.Name()]; ok {
			return nil, fmt.Errorf("duplicated tool name %s", d.Name())
		}
		m[d.Name()] = d
	}
	return &ToolRunner{defsMap: m, logger: slog.New(h)}, nil
}

func (r *ToolRunner) Run(ctx context.Context, name string, in map[string]any) (map[string]any, error) {
	p, ok := r.defsMap[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool %s", name)
	}

	ctx = context.WithValue(ctx, loggerKey, r.logger.With("tool_name", name, "request", in))
	return p.process(ctx, in)
}

type confirmationResult int

const (
	confirmationYes confirmationResult = iota
	confirmationNo
	confirmationEdit
	confirmationNoAnswer confirmationResult = -1
)

func confirm() (confirmationResult, error) {
	sel := promptui.Select{
		Label: "Is this okay",
		Items: []string{
			"Yes",
			"No",
			// "I change it by myself",
		},
		CursorPos: 1,
	}
	idx, _, err := sel.Run()
	if err != nil {
		return confirmationNoAnswer, err
	}
	return confirmationResult(idx), nil
}
