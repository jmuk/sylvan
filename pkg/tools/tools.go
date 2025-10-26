package tools

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/jmuk/sylvan/pkg/session"
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
}

func NewToolRunner(defs []ToolDefinition) (*ToolRunner, error) {
	m := make(map[string]ToolDefinition, len(defs))
	for _, d := range defs {
		if _, ok := m[d.Name()]; ok {
			log.Printf("name %s duplicated, overwritten", d.Name())
		}
		m[d.Name()] = d
	}
	return &ToolRunner{defsMap: m}, nil
}

func (r *ToolRunner) Run(ctx context.Context, name string, in map[string]any) (any, []*parts.Part, error) {
	p, ok := r.defsMap[name]
	if !ok {
		return nil, nil, fmt.Errorf("unknown tool %s", name)
	}
	s, ok := session.FromContext(ctx)
	if !ok {
		return nil, nil, fmt.Errorf("session not found")
	}
	l, err := s.GetLogger("tool")
	if err != nil {
		return nil, nil, err
	}

	ctx = context.WithValue(ctx, loggerKey, l.With("tool_name", name, "request", in))
	fmt.Println()
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
	return confirmWith(true)
}

func confirmWith(canEdit bool) (confirmationResult, error) {
	items := []string{"Yes", "No"}
	if canEdit {
		items = append(items, "No / edit by myself")
	}
	sel := promptui.Select{
		Label:     "Is this okay",
		Items:     items,
		CursorPos: 1,
	}
	idx, _, err := sel.Run()
	if err != nil {
		return confirmationNoAnswer, err
	}
	return confirmationResult(idx), nil
}
