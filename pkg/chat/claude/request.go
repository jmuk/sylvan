package claude

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/invopop/jsonschema"
)

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

func (a *Agent) buildRequestBody() ([]byte, error) {
	body := bodyData{
		Model:     a.modelName,
		MaxTokens: a.config.MaxTokens,
		Stream:    true,
		System:    a.systemPrompt,
		Thinking: &thinkingConfig{
			BudgetTokens: 8192,
			Type:         "enabled",
		},
		Tools: a.tools,
	}
	for _, hc := range a.history {
		imsg, err := hc.toInput()
		if err != nil {
			return nil, err
		}
		body.Messages = append(body.Messages, imsg)
	}
	return json.Marshal(body)
}

func (a *Agent) request() (io.ReadCloser, error) {
	body, err := a.buildRequestBody()
	if err != nil {
		return nil, err
	}

	rheaders := http.Header{}
	rheaders.Add("x-api-key", a.apiKey)
	rheaders.Add("anthropic-version", a.config.AnthropicVersion)
	rheaders.Add("content-type", "application/json")

	req := &http.Request{
		Method:        http.MethodPost,
		URL:           a.url,
		Header:        rheaders,
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return nil, errors.New(string(data))
	}
	return resp.Body, nil
}
