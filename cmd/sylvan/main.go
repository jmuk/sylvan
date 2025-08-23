package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/chzyer/readline"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	rl, err := readline.New("> ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF
			break
		}
		for result, err := range chat.SendMessageStream(ctx, *genai.NewPartFromText(line)) {
			if err != nil {
				log.Fatal(err)
			}
			d, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				log.Fatal(err)
			}
			println(string(d))
		}
	}
}
