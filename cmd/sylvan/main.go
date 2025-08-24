package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/manifoldco/promptui"
	"google.golang.org/genai"
)

const systemPrompt = `
You are a professional software engineer.  You are tasked to write computer programs.
From what you are asked, make a plan, write code, verify it with tests, and repeat it
until the end result satisfies the request.
`

func createFile(filename string, content string) (ok bool, err error) {
	// TODO: ask the user to go or not.
	fmt.Println("Creating a new file", filename)
	fmt.Println("With the following content: ", content)
	err = os.WriteFile(filename, []byte(content), 0644)
	return err == nil, err
}

func main() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	funcs := []*genai.FunctionDeclaration{
		{
			Behavior:    genai.BehaviorBlocking,
			Description: "Create a new file with the given name and the given content",
			Name:        "createFile",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"filename": {
						Type:        genai.TypeString,
						Description: "The name of the file",
					},
					"content": {
						Type:        genai.TypeString,
						Description: "The content of the new file",
					},
				},
				Required: []string{"filename", "content"},
			},
			Response: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"ok": {
						Type:        genai.TypeBoolean,
						Description: "Success or not",
					},
					"error": {
						Type:        genai.TypeString,
						Description: "The error message if it fails.",
					},
				},
				Required: []string{"ok"},
			},
		},
	}

	chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(
			systemPrompt,
			genai.RoleUser,
		),
		Tools: []*genai.Tool{{FunctionDeclarations: funcs}},
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
						fmt.Println(part.Text)
					}
					if call := part.FunctionCall; call != nil {
						switch call.Name {
						case "createFile":
							filename, ok := call.Args["filename"]
							if !ok {
								fmt.Println("filename is missing")
								break
							}
							content, ok := call.Args["content"]
							if !ok {
								fmt.Println("content is missing")
								break
							}
							ok, err := createFile(filename.(string), content.(string))
							response := map[string]any{"ok": ok}
							if !ok {
								response["error"] = err.Error()
							}
							nextMsgs = append(nextMsgs, genai.Part{
								FunctionResponse: &genai.FunctionResponse{
									ID:       call.ID,
									Name:     call.Name,
									Response: response,
								}})
						default:
							fmt.Println("Unknown function: ", call.Name)
						}
					}
				}
			}
			if len(nextMsgs) == 0 {
				break
			}
			msgs = nextMsgs
		}
	}
}
