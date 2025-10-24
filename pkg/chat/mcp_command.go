package chat

import (
	"context"
	"fmt"
)

func (c *Chat) handleMCPCommand(ctx context.Context, args []string) error {
	if len(args) == 0 {
		// show the list of MCP tools.
		if c.cs.cfg == nil {
			if err := c.cs.maybeInit(ctx, c.cwd); err != nil {
				return err
			}
		}
		for _, mcpc := range c.cs.cfg.MCP {
			fmt.Println(mcpc.String())
		}
		return nil
	}

	// TODO: use an LLM to turn the user input to the correct intent.
	return nil
}
