package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jmuk/sylvan/pkg/ai"
	"github.com/jmuk/sylvan/pkg/tools"
	"github.com/manifoldco/promptui"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()

	tempdir, err := os.MkdirTemp("", "sylvan")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Logs are stored into %s", tempdir)
	toolsLog, err := os.OpenFile(filepath.Join(tempdir, "tools.log"), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer toolsLog.Close()

	toolsHandler := slog.NewJSONHandler(toolsLog, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	aiLog, err := os.OpenFile(filepath.Join(tempdir, "ai.log"), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer aiLog.Close()

	aiHandler := slog.NewJSONHandler(aiLog, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	ft, err := tools.NewFiles(".", toolsHandler)
	if err != nil {
		log.Fatal(err)
	}
	et := tools.NewExecTool(toolsHandler)

	toolDefs := append([]tools.ToolDefinition{}, ft.ToolDefs()...)
	toolDefs = append(toolDefs, et.ToolDefs()...)

	trun, err := tools.NewToolRunner(toolDefs)
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
				if len(result.Candidates) == 0 {
					continue
				}
				for _, part := range result.Candidates[0].Content.Parts {
					if part.Text != "" {
						fmt.Print(part.Text)
						printed = true
					}
					if call := part.FunctionCall; call != nil {
						resp, err := trun.Run(call.Name, call.Args)
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
