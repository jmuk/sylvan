package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jmuk/sylvan/pkg/ai"
	"github.com/jmuk/sylvan/pkg/chat"
	"github.com/jmuk/sylvan/pkg/tools"
)

var logtostderr bool

func newLogHandler(tempdir, path string) (slog.Handler, func(), error) {
	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}
	if logtostderr {
		return slog.NewJSONHandler(os.Stderr, opts), func() {}, nil
	}

	file, err := os.OpenFile(filepath.Join(tempdir, path), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}
	return slog.NewJSONHandler(file, opts), func() { file.Close() }, nil
}

func main() {
	flag.BoolVar(&logtostderr, "logtostderr", false, "Print out log messages to stderr")
	flag.Parse()

	ctx := context.Background()

	var tempdir string
	if logtostderr {
		tempdir = ""
	} else {
		var err error
		if tempdir, err = os.MkdirTemp("", "sylvan"); err != nil {
			log.Fatal(err)
		}
		log.Printf("Logs are stored into %s", tempdir)
	}

	toolsHandler, toolsCleanup, err := newLogHandler(tempdir, "tools.log")
	if err != nil {
		log.Fatal(err)
	}
	defer toolsCleanup()

	aiHandler, aiCleanup, err := newLogHandler(tempdir, "ai.log")
	if err != nil {
		log.Fatal(err)
	}
	defer aiCleanup()

	ft, err := tools.NewFiles(".")
	if err != nil {
		log.Fatal(err)
	}
	et := tools.NewExecTool()

	toolDefs := append([]tools.ToolDefinition{}, ft.ToolDefs()...)
	toolDefs = append(toolDefs, et.ToolDefs()...)

	trun, err := tools.NewToolRunner(toolsHandler, toolDefs)
	if err != nil {
		log.Fatal(err)
	}

	aiChat, err := ai.NewGemini(ctx, "gemini-2.5-flash", toolDefs, aiHandler)
	if err != nil {
		log.Fatal(err)
	}

	c := chat.New(aiChat, aiHandler, trun)
	if err := c.RunLoop(ctx); err != nil {
		log.Fatal(err)
	}
}
