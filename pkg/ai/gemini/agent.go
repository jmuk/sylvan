package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"github.com/invopop/jsonschema"
	"github.com/jmuk/sylvan/pkg/ai"
	"github.com/jmuk/sylvan/pkg/tools"
	"google.golang.org/genai"
)

func toSchema(s *jsonschema.Schema) (*genai.Schema, error) {
	encoded, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(encoded))
	decoded := &genai.Schema{}
	if err := json.Unmarshal(encoded, decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

type Agent struct {
	chat *genai.Chat
}

func (g *Agent) SendMessageStream(ctx context.Context, parts []ai.Part) iter.Seq2[*ai.Part, error] {
	return func(yield func(*ai.Part, error) bool) {
		inputParts := make([]*genai.Part, 0, len(parts))
		for _, part := range parts {
			p := &genai.Part{}
			if part.Text != "" {
				p.Text = part.Text
			}
			if part.FunctionResponse != nil {
				resp := make(map[string]any, len(part.FunctionResponse.Response))
				for k, v := range part.FunctionResponse.Response {
					resp[k] = v
				}
				if part.FunctionResponse.Error != nil {
					resp["error"] = part.FunctionResponse.Error.Error()
				}
				p.FunctionResponse = &genai.FunctionResponse{
					ID:       part.FunctionResponse.ID,
					Name:     part.FunctionResponse.Name,
					Response: resp,
				}
			}
			inputParts = append(inputParts, p)
		}
		for result, err := range g.chat.SendStream(ctx, inputParts...) {
			if err != nil {
				if !yield(nil, err) {
					return
				}
			}
			if len(result.Candidates) == 0 || result.Candidates[0].Content == nil {
				continue
			}
			for _, part := range result.Candidates[0].Content.Parts {
				p := &ai.Part{}
				if part.FunctionCall != nil {
					p.FunctionCall = &ai.FunctionCall{
						ID:   part.FunctionCall.ID,
						Name: part.FunctionCall.Name,
						Args: part.FunctionCall.Args,
					}
				}
				if part.Text != "" {
					p.Text = part.Text
					p.Thought = part.Thought
				}
				if !yield(p, nil) {
					return
				}
			}
		}
	}
}

func New(
	ctx context.Context,
	modelName string,
	clientConfig *genai.ClientConfig,
	toolDefs []tools.ToolDefinition,
	includeThoughts bool,
) (*Agent, error) {
	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, err
	}

	var funcs []*genai.FunctionDeclaration
	for _, d := range toolDefs {
		params, err := toSchema(d.RequestSchema())
		if err != nil {
			return nil, fmt.Errorf("failed to encode request schema for %s: %w", d.Name(), err)
		}
		resp, err := toSchema(d.ResponseSchema())
		if err != nil {
			return nil, fmt.Errorf("failed to encode response schema for %s: %w", d.Name(), err)
		}
		funcs = append(funcs, &genai.FunctionDeclaration{
			Name:        d.Name(),
			Description: d.Description(),
			Behavior:    genai.BehaviorBlocking,
			Parameters:  params,
			Response:    resp,
		})
	}

	chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(
			ai.SystemPrompt,
			genai.RoleUser,
		),
		Tools: []*genai.Tool{{FunctionDeclarations: funcs}},
		ThinkingConfig: &genai.ThinkingConfig{
			IncludeThoughts: includeThoughts,
		},
	}, nil)
	if err != nil {
		return nil, err
	}
	return &Agent{chat: chat}, nil
}
