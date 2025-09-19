package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/invopop/jsonschema"
	"github.com/jmuk/sylvan/pkg/ai"
	"github.com/jmuk/sylvan/pkg/sse"
	"github.com/jmuk/sylvan/pkg/tools"
)

type historicalContent struct {
	Part ai.Part
	role ai.Role
}

type inputMessage struct {
	Content any     `json:"content"`
	Role    ai.Role `json:"role"`
}

type toolConfiguration struct {
	AllowedTools []string `json:"allowed_tools,omitempty"`
	Enabled      bool     `json:"enabled,omitempty"`
}

type requestMCPServerURLDefinition struct {
	Name               string            `json:"name"`
	Type               string            `json:"type"`
	Url                string            `json:"url"`
	AuthorizationToken string            `json:"authorization_token,omitempty"`
	ToolConfiguration  toolConfiguration `json:"tool_configuration,omitempty"`
}

type requestMetadata struct {
	UserID string `json:"user_id"`
}

type thinkingConfig struct {
	BudgetTokens int    `json:"budget_tokens"`
	Type         string `json:"type"`
}

type toolChoice struct {
	Name                   string `json:"name,omitempty"`
	Type                   string `json:"type"`
	DisableParallelToolUse bool   `json:"disable_parallel_tool_use,omitempty"`
}

type tool struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	InputSchema *jsonschema.Schema `json:"input_schema"`
}

type bodyData struct {
	Model         string                          `json:"model"`
	Messages      []inputMessage                  `json:"messages"`
	MaxTokens     int                             `json:"max_tokens"`
	Container     string                          `json:"container,omitempty"`
	MCPServers    []requestMCPServerURLDefinition `json:"mcp_servers,omitempty"`
	Metadata      *requestMetadata                `json:"metadata,omitempty"`
	ServiceTier   string                          `json:"service_tier,omitempty"`
	StopSequences []string                        `json:"stop_sequences,omitempty"`
	Stream        bool                            `json:"stream,omitempty"`
	System        string                          `json:"system,omitempty"`
	Temperature   float32                         `json:"temperature,omitempty"`
	Thinking      *thinkingConfig                 `json:"thinking,omitempty"`
	ToolChoice    *toolChoice                     `json:"tool_choice,omitempty"`
	Tools         []tool                          `json:"tools,omitempty"`
	TopK          int                             `json:"top_k,omitempty"`
	TopP          int                             `json:"top_p,omitempty"`
}

type Agent struct {
	history      []historicalContent
	systemPrompt string

	url    *url.URL
	apiKey string

	config *Config

	tools []tool
}

type eventType string

const (
	eventTypePing              eventType = "ping"
	eventTypeError             eventType = "error"
	eventTypeMessageStart      eventType = "message_start"
	eventTypeMessageDelta      eventType = "message_delta"
	eventTypeMessageStop       eventType = "message_stop"
	eventTypeContentBlockStart eventType = "content_block_start"
	eventTypeContentBlockDelta eventType = "content_block_delta"
	eventTypeContentBlockStop  eventType = "content_block_stop"
)

type deltaType string

const (
	deltaTypeNone      deltaType = ""
	deltaTypeText      deltaType = "text_delta"
	deltaTypeJSON      deltaType = "input_json_delta"
	deltaTypeThinking  deltaType = "thinking_delta"
	deltaTypeSignature deltaType = "signature_delta"
)

type contentBlockDelta struct {
	Type  eventType `json:"type"`
	Index int       `json:"index"`
	Delta struct {
		Type        deltaType `json:"type"`
		Text        string    `json:"text"`
		PartialJSON string    `json:"partial_json"`
		Thinking    string    `json:"thinking"`
		Signature   string    `json:"signature"`
	} `json:"delta"`
}

type blockType string

const (
	blockTypeText     blockType = "text"
	blockTypeToolUse  blockType = "tool_use"
	blockTypeThinking blockType = "thinking"
)

type contentBlock struct {
	Type         eventType `json:"type"`
	Index        int       `json:"index"`
	ContentBlock struct {
		Type blockType `json:"type"`

		Text string `json:"text"`

		ID    string         `json:"id"`
		Name  string         `json:"name"`
		Input map[string]any `json:"input"`

		Thinking  string `json:"thinking"`
		Signature string `json:"signature"`
	} `json:"content_block"`
}

func (a *Agent) SendMessageStream(ctx context.Context, messages []ai.Part) iter.Seq2[*ai.Part, error] {
	return func(yield func(*ai.Part, error) bool) {
		rheaders := http.Header{}
		rheaders.Add("x-api-key", a.apiKey)
		rheaders.Add("anthropic-version", a.config.AnthropicVersion)
		rheaders.Add("content-type", "application/json")
		var reqMessages []inputMessage
		for _, msg := range messages {
			a.history = append(a.history, historicalContent{
				Part: msg,
				role: ai.RoleUser,
			})
		}
		for _, hc := range a.history {
			imsg := inputMessage{Role: hc.role}
			if hc.Part.Text != "" {
				if hc.Part.Thought {
					imsg.Content = []map[string]any{
						{
							"thinking":  hc.Part.Text,
							"type":      "thinking",
							"signature": hc.Part.ThinkingSignature,
						},
					}
				} else {
					imsg.Content = hc.Part.Text
				}
			} else if fc := hc.Part.FunctionCall; fc != nil {
				imsg.Content = []map[string]any{
					{
						"id":    fc.ID,
						"name":  fc.Name,
						"input": fc.Args,
						"type":  "tool_use",
					},
				}
			} else if fr := hc.Part.FunctionResponse; fr != nil {
				var body string
				if fr.Error != nil {
					body = fr.Error.Error()
				} else if len(fr.Response) == 1 {
					for _, v := range fr.Response {
						s, ok := v.(string)
						if ok {
							body = s
						}
					}
				}
				if body == "" {
					encoded, err := json.Marshal(fr.Response)
					if err != nil {
						yield(nil, err)
						return
					}
					body = string(encoded)
				}
				imsg.Content = []map[string]any{
					{
						"tool_use_id": fr.ID,
						"type":        "tool_result",
						"content":     body,
						"is_error":    hc.Part.FunctionResponse.Error != nil,
					},
				}
			}

			reqMessages = append(reqMessages, imsg)
		}
		body := bodyData{
			Model:     a.config.Model,
			Messages:  reqMessages,
			MaxTokens: a.config.MaxTokens,
			Stream:    true,
			System:    a.systemPrompt,
			Thinking: &thinkingConfig{
				BudgetTokens: 8192,
				Type:         "enabled",
			},
			Tools: a.tools[:1],
		}
		encoded, err := json.Marshal(body)
		if err != nil {
			yield(nil, err)
			return
		}
		req := &http.Request{
			Method:        http.MethodPost,
			URL:           a.url,
			Header:        rheaders,
			Body:          io.NopCloser(bytes.NewReader(encoded)),
			ContentLength: int64(len(encoded)),
		}
		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			yield(nil, err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			data, _ := io.ReadAll(resp.Body)
			yield(nil, errors.New(string(data)))
			return
		}
		sseScanner := sse.NewScanner(resp.Body)
		var currentBlock *contentBlock = nil
		for {
			ev, err := sseScanner.Scan()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				if !yield(nil, err) {
					break
				}
			}
			switch eventType(ev.Event) {
			case eventTypeError:
				if !yield(nil, errors.New(ev.Data)) {
					return
				}
			case eventTypeContentBlockStart:
				currentBlock = &contentBlock{}
				if err := json.Unmarshal([]byte(ev.Data), currentBlock); err != nil {
					if !yield(nil, err) {
						return
					}
				}
				if currentBlock.ContentBlock.Type == blockTypeText && currentBlock.ContentBlock.Text != "" {
					if !yield(&ai.Part{Text: currentBlock.ContentBlock.Text}, nil) {
						return
					}
				} else if currentBlock.ContentBlock.Type == blockTypeThinking && currentBlock.ContentBlock.Thinking != "" {
					if !yield(&ai.Part{Thought: true, Text: currentBlock.ContentBlock.Thinking}, nil) {
						return
					}
				}
			case eventTypeContentBlockDelta:
				cbd := &contentBlockDelta{}
				if err := json.Unmarshal([]byte(ev.Data), cbd); err != nil {
					if !yield(nil, err) {
						return
					}
				}
				if currentBlock == nil {
					if !yield(nil, fmt.Errorf("missing content block start")) {
						return
					}
				}
				if currentBlock.Index != cbd.Index {
					if !yield(nil, fmt.Errorf("index mismatch: want %d got %d", currentBlock.Index, cbd.Index)) {
						return
					}
				}
				switch cbd.Delta.Type {
				case deltaTypeText:
					if currentBlock.ContentBlock.Type != blockTypeText {
						if !yield(nil, fmt.Errorf("type mismatch: want %s got text", currentBlock.Type)) {
							return
						}
					}
					currentBlock.ContentBlock.Text += cbd.Delta.Text
					if !yield(&ai.Part{Text: cbd.Delta.Text}, nil) {
						return
					}
				case deltaTypeJSON:
					if currentBlock.ContentBlock.Type != blockTypeToolUse {
						if !yield(nil, fmt.Errorf("type mismatch: want %s got partial_json", currentBlock.ContentBlock.Type)) {
							return
						}
					}
					// Use Text field to accumulate partial JSON then parse it.
					currentBlock.ContentBlock.Text += cbd.Delta.PartialJSON
					// not yield yet.
				case deltaTypeThinking:
					if currentBlock.ContentBlock.Type != blockTypeThinking {
						if !yield(nil, fmt.Errorf("type mismatch: want %s got thinking", currentBlock.ContentBlock.Type)) {
							return
						}
					}
					currentBlock.ContentBlock.Thinking += cbd.Delta.Thinking
					if !yield(&ai.Part{Text: cbd.Delta.Thinking, Thought: true}, nil) {
						return
					}
				case deltaTypeSignature:
					currentBlock.ContentBlock.Signature += cbd.Delta.Signature
				}
			case eventTypeContentBlockStop:
				if currentBlock == nil {
					if !yield(nil, fmt.Errorf("content_block_stop appears without start")) {
						return
					}
				}
				switch currentBlock.ContentBlock.Type {
				case blockTypeText:
					a.history = append(a.history, historicalContent{
						Part: ai.Part{
							Text: currentBlock.ContentBlock.Text,
						},
						role: ai.RoleAssistant,
					})
				case blockTypeThinking:
					a.history = append(a.history, historicalContent{
						Part: ai.Part{
							Thought:           true,
							Text:              currentBlock.ContentBlock.Thinking,
							ThinkingSignature: currentBlock.ContentBlock.Signature,
						},
						role: ai.RoleAssistant,
					})
				case blockTypeToolUse:
					currentBlock.ContentBlock.Input = map[string]any{}
					if err := json.Unmarshal([]byte(currentBlock.ContentBlock.Text), &currentBlock.ContentBlock.Input); err != nil {
						if !yield(nil, err) {
							return
						}
					}
					part := &ai.Part{FunctionCall: &ai.FunctionCall{
						ID:   currentBlock.ContentBlock.ID,
						Name: currentBlock.ContentBlock.Name,
						Args: currentBlock.ContentBlock.Input,
					}}
					if !yield(part, err) {
						return
					}
					a.history = append(a.history, historicalContent{
						Part: *part,
						role: ai.RoleAssistant,
					})
				}
				currentBlock = nil
			}
		}
	}
}

func New(config *Config, toolDefs []tools.ToolDefinition) (*Agent, error) {
	agent := &Agent{
		systemPrompt: ai.SystemPrompt,
		config:       config,
	}
	for _, toolDef := range toolDefs {
		agent.tools = append(agent.tools, tool{
			Name:        toolDef.Name(),
			Description: toolDef.Description(),
			InputSchema: toolDef.RequestSchema(),
		})
	}
	if config.APIKey == "" {
		if config.APIKeyFromEnv == "" {
			return nil, errors.New("either api-key or api-key-from-env must be specified")
		}
		agent.apiKey = os.Getenv(config.APIKeyFromEnv)
		if agent.apiKey == "" {
			return nil, fmt.Errorf("env variable %s not defined", config.APIKeyFromEnv)
		}
	} else {
		agent.apiKey = config.APIKey
	}
	var err error
	agent.url, err = url.Parse(config.BaseURL)
	if err != nil {
		return nil, err
	}
	agent.url.Path = path.Join(agent.url.Path, "/v1/messages")
	return agent, nil
}
