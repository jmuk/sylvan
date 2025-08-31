package main

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/jmuk/sylvan/pkg/ai"
	"github.com/jmuk/sylvan/pkg/tools"
	"github.com/manifoldco/promptui"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()

	ft, err := tools.NewFiles(".")
	if err != nil {
		log.Fatal(err)
	}

	toolDefs := append([]tools.ToolDefinition{tools.ExecCommandDef}, ft.ToolDefs()...)

	trun, err := tools.NewToolRunner(toolDefs)
	if err != nil {
		log.Fatal(err)
	}

	chat, err := ai.NewGemini(ctx, "gemini-2.5-flash", toolDefs)
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
		if err != nil { // io.EOF
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		msgs := []genai.Part{*genai.NewPartFromText(line)}
		for {
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
			fmt.Println()
			if len(nextMsgs) == 0 {
				break
			}
			msgs = nextMsgs
		}
	}
}
