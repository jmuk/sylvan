package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jmuk/sylvan/pkg/ai"
	"github.com/jmuk/sylvan/pkg/tools"
	"github.com/manifoldco/promptui"
	"google.golang.org/genai"
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

	chat, err := ai.NewGemini(ctx, "gemini-2.5-flash", toolDefs, aiHandler)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}

	p := promptui.Prompt{
		Label: "> ",
	}

	logger := slog.New(aiHandler)

	for {
		line, err := p.Run()
		if err != nil {
			// io.EOF
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		if line == "\\q" {
			break
		}
		msgs := []genai.Part{*genai.NewPartFromText(line)}
		for {
			printed := false
			var nextMsgs []genai.Part
			for result, err := range chat.SendMessageStream(ctx, msgs...) {
				if err != nil {
					log.Fatal(err)
				}
				logger.Debug("Received message", "result", result)
				if len(result.Candidates) == 0 {
					continue
				}
				for _, part := range result.Candidates[0].Content.Parts {
					if part.Text != "" {
						fmt.Print(part.Text)
						printed = true
					}
					if call := part.FunctionCall; call != nil {
						commandCtx, cancel := context.WithTimeout(ctx, time.Minute)
						resp, err := trun.Run(commandCtx, call.Name, call.Args)
						cancel()
						if err != nil {
							log.Fatal(err)
						}
						nextMsgs = append(nextMsgs, genai.Part{
							FunctionResponse: &genai.FunctionResponse{
								ID:       part.FunctionCall.ID,
								Name:     part.FunctionCall.Name,
								Response: resp,
							},
						})
					}
				}
			}
			if printed {
				fmt.Println()
			}
			if len(nextMsgs) == 0 {
				break
			}
			msgs = nextMsgs
		}
	}
}
