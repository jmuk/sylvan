package claude

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/jmuk/sylvan/pkg/session"
)

type modelInfo struct {
	ID          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

type modelResponse struct {
	Data    []modelInfo `json:"data"`
	FirstID string      `json:"first_id"`
	HasMore bool        `json:"has_more"`
	LastID  string      `json:"last_id"`
}

func (c *Config) Models(ctx context.Context) ([]string, error) {
	logger, err := session.LoggerFromContext(ctx, "claude")
	if err != nil {
		return nil, err
	}
	apiKey, err := c.apiKey()
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}
	u = u.JoinPath("v1", "models")

	rheaders := http.Header{}
	rheaders.Add("x-api-key", apiKey)
	rheaders.Add("anthropic-version", c.AnthropicVersion)
	rheaders.Add("content-type", "application/json")

	client := http.Client{}
	var results []string
	var pageToken string
	for {
		if pageToken != "" {
			u.RawQuery = url.Values(map[string][]string{
				"after_id": {pageToken},
			}).Encode()
		}
		resp, err := client.Do(&http.Request{
			Method: http.MethodGet,
			URL:    u,
			Header: rheaders,
		})
		if err != nil {
			return nil, err
		}
		parsedResponse := &modelResponse{}
		if err := json.NewDecoder(resp.Body).Decode(parsedResponse); err != nil {
			return nil, err
		}
		for _, m := range parsedResponse.Data {
			logger.Debug("model", "model", m)
			results = append(results, m.ID)
		}
		if parsedResponse.HasMore {
			pageToken = parsedResponse.LastID
		} else {
			break
		}
	}
	return results, nil
}
