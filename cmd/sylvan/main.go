package main

import (
	"context"
	"log"
	"os"

	"github.com/jmuk/sylvan/pkg/chat"
	"github.com/jmuk/sylvan/pkg/config"
	"github.com/jmuk/sylvan/pkg/tools"
)

func main() {
	ctx := context.Background()

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	config, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	toolMgrs := []tools.Manager{
		tools.NewFiles(cwd),
		tools.NewExecTool(),
	}

	for _, mcpConfig := range config.MCP {
		var tool *tools.MCPTool
		name := mcpConfig.Name
		if mcpConfig.Endpoint != "" {
			if name == "" {
				name = mcpConfig.Endpoint
			}
			tool = tools.NewHTTPMCP(name, mcpConfig.Endpoint, mcpConfig.RequestHeaders)
		} else if len(mcpConfig.Command) > 0 {
			if name == "" {
				name = mcpConfig.Command[0]
			}
			tool = tools.NewCommandMCP(name, mcpConfig.Command)
		} else {
			log.Printf("Unrecognized mcp config %+v", mcpConfig)
			continue
		}
		toolMgrs = append(toolMgrs, tool)
	}

	c, err := chat.New(ctx, config, toolMgrs, cwd)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	if err := c.RunLoop(ctx); err != nil {
		log.Print(err)
	}
}
