package tools

import (
	"context"

	"github.com/jmuk/sylvan/pkg/config"
)

type Manager interface {
	ToolDefs(ctx context.Context) ([]ToolDefinition, error)
	Close() error
}

func NewManagers(cwd string, c *config.Config) []Manager {
	mgrs := []Manager{
		NewFiles(cwd),
		NewExecTool(),
	}
	for _, mcpc := range c.MCP {
		mgrs = append(mgrs, NewMCP(mcpc))
	}
	return mgrs
}
