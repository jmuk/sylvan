package tools

import (
	"context"
	"fmt"
	"log/slog"
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
