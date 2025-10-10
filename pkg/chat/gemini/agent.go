package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"os"

	"github.com/invopop/jsonschema"
	"github.com/jmuk/sylvan/pkg/chat"
	"github.com/jmuk/sylvan/pkg/session"
	"github.com/jmuk/sylvan/pkg/tools"
	"google.golang.org/genai"
)

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

type Agent struct {
	chat        *genai.Chat
	historyFile string
}

func (g *Agent) saveContent(c *genai.Content) error {
	if g.historyFile == "" {
		return nil
	}
	data, err := os.ReadFile(g.historyFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	hasContent := len(data) > 0
	w := bytes.NewBuffer(data)
	if hasContent {
		if _, err := w.WriteRune('\n'); err != nil {
			return err
		}
	}
	if err := json.NewEncoder(w).Encode(c); err != nil {
		return err
	}
	return os.WriteFile(g.historyFile, w.Bytes(), 0600)
}

func (g *Agent) SendMessageStream(ctx context.Context, parts []chat.Part) iter.Seq2[*chat.Part, error] {
	return func(yield func(*chat.Part, error) bool) {
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
		if err := g.saveContent(&genai.Content{
			Parts: inputParts,
			Role:  genai.RoleUser,
		}); err != nil {
			if !yield(nil, err) {
				return
			}
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
				p := &chat.Part{}
				if part.FunctionCall != nil {
					p.FunctionCall = &chat.FunctionCall{
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
			if err := g.saveContent(result.Candidates[0].Content); err != nil {
				if !yield(nil, err) {
					return
				}
			}
		}
	}
}

func loadHistory(historyFile string) ([]*genai.Content, error) {
	f, err := os.Open(historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	var contents []*genai.Content
	for s.Scan() {
		l := s.Text()
		c := &genai.Content{}
		if err := json.Unmarshal([]byte(l), c); err != nil {
			return nil, err
		}
		contents = append(contents, c)
	}
	return contents, nil
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

	s, sok := session.FromContext(ctx)
	var historyFile string
	if sok {
		historyFile = s.HistoryFile()
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

	var history []*genai.Content
	if historyFile != "" {
		history, err = loadHistory(historyFile)
		if err != nil {
			return nil, err
		}
	}

	chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(
			chat.SystemPrompt,
			genai.RoleUser,
		),
		Tools: []*genai.Tool{{FunctionDeclarations: funcs}},
		ThinkingConfig: &genai.ThinkingConfig{
			IncludeThoughts: includeThoughts,
		},
	}, history)
	if err != nil {
		return nil, err
	}
	return &Agent{chat: chat, historyFile: historyFile}, nil
}
