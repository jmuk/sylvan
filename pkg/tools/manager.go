package tools

import (
	"context"
	"sort"

	"github.com/jmuk/sylvan/pkg/config"
)

type Manager interface {
	ToolDefs(ctx context.Context) ([]ToolDefinition, error)
	Close() error
}

func NewManagers(cwd string, c *config.Config) []Manager {
	mcpManagers := map[string]Manager{}
	for _, mcpc := range c.MCP {
		mcpManagers[mcpc.Name] = NewMCP(mcpc)
	}
	var keys []string
	for k := range mcpManagers {
		if mcpManagers[k] != nil {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	mgrs := make([]Manager, 0, len(keys)+2)
	for _, k := range keys {
		mgrs = append(mgrs, mcpManagers[k])
	}

	// Append files and exec at the end so that that implementation will
	// always be used even if some MCP tool names happen to conflict.
	return append(mgrs, NewFiles(cwd), NewExecTool())
}
