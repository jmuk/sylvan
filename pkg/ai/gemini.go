package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"github.com/invopop/jsonschema"
	"github.com/jmuk/sylvan/pkg/tools"
	"google.golang.org/genai"
)

const systemPrompt = `
You are a seasoned software engineer.  Your task is to provide the technical solution
for what the user asked.  Please follow the steps below to pursue the goal. Please
also write your thoughts step-by-step as much as possible.

1. Plan

The request is often vague, and therefore you will have to set up a list of concrete
tasks to achieve the goal.  First, you set up the plan, the list of things you'll do,
and show it to the users.

2. Investigate the code base

Often times you are tasked to make changes on an existing code base.  Check the current
status and align the plan and your outcome with the existing code base.  Read the files,
documentations, etc. when necessary.

3. Tests

As a seasoned software engineer, you'll adopt test-driven-development (TDD) whenever
applicable. Before implementing the solution, first set up the tests, add new test
cases, or modify the tests. Then run the test scenarios and confirm that those tests
_fail_, because the actual solution hasn't been provided yet.

4. Code

Then you write the code, and make sure that the tests now _pass_. Note that the test
code must not be modified during this step.
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

type GeminiAgent struct {
	chat *genai.Chat
}

func (g *GeminiAgent) SendMessageStream(ctx context.Context, parts []Part) iter.Seq2[*Part, error] {
	return func(yield func(*Part, error) bool) {
		inputParts := make([]*genai.Part, 0, len(parts))
		for _, part := range parts {
			p := &genai.Part{}
			if part.Text != "" {
				p.Text = part.Text
			}
			if part.FunctionResponse != nil {
				p.FunctionResponse = &genai.FunctionResponse{
					ID:       part.FunctionResponse.ID,
					Name:     part.FunctionResponse.Name,
					Response: part.FunctionResponse.Response,
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
				p := &Part{}
				if part.FunctionCall != nil {
					p.FunctionCall = &FunctionCall{
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

func NewGemini(
	ctx context.Context,
	modelName string,
	clientConfig *genai.ClientConfig,
	toolDefs []tools.ToolDefinition,
	includeThoughts bool,
) (*GeminiAgent, error) {
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
			systemPrompt,
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
	return &GeminiAgent{chat: chat}, nil
}
