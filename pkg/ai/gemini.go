package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/invopop/jsonschema"
	"github.com/jmuk/sylvan/pkg/tools"
	"google.golang.org/genai"
)

const systemPrompt = `
You are a professional software engineer.  You are tasked to write computer programs.
From what you are asked, make a plan, write code, verify it with tests, and repeat it
until the end result satisfies the request.
`

func toSchema(s *jsonschema.Schema) (*genai.Schema, error) {
	encoded, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	decoded := &genai.Schema{}
	if err := json.Unmarshal(encoded, decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func NewGemini(ctx context.Context, modelName string, toolDefs []tools.ToolDefinition, handler slog.Handler) (*genai.Chat, error) {
	logger := slog.New(handler)
	client, err := genai.NewClient(ctx, nil)
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

	logger.Debug("Tool definitions", "tools", funcs)

	return client.Chats.Create(ctx, "gemini-2.5-flash", &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(
			systemPrompt,
			genai.RoleUser,
		),
		Tools: []*genai.Tool{{FunctionDeclarations: funcs}},
	}, nil)
}
