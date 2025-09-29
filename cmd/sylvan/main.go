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

	ft, err := tools.NewFiles(cwd)
	if err != nil {
		log.Fatal(err)
	}
	et := tools.NewExecTool()

	toolDefs := append([]tools.ToolDefinition{}, ft.ToolDefs()...)
	toolDefs = append(toolDefs, et.ToolDefs()...)

	config, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	c, err := chat.New(config, toolDefs, cwd)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	if err := c.RunLoop(ctx); err != nil {
		log.Print(err)
	}
}
