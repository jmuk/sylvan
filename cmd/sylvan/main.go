package main

import (
	"context"
	"encoding/json"
	"io"
	"log"

	"github.com/manifoldco/promptui"
	"google.golang.org/genai"
)

const systemPrompt = `
You are a professional software engineer.  You are tasked to write computer programs.
From what you are asked, make a plan, write code, verify it with tests, and repeat it
until the end result satisfies the request.
`

func main() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(
			systemPrompt,
			genai.RoleUser,
		),
	}, nil)
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
		for result, err := range chat.SendMessageStream(ctx, *genai.NewPartFromText(line)) {
			if err != nil {
				log.Fatal(err)
			}
			if len(result.Candidates) == 0 {
				continue
			}
			cand := result.Candidates[0]
			for _, part := range cand.Content.Parts {
				if part.Text != "" {
					print(part.Text)
				} else {
					d, err := json.MarshalIndent(part, "", "  ")
					if err != nil {
						log.Fatal(err)
					}
					println(string(d))
				}
			}
		}
		println()
	}
}
