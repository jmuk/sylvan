package claude

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/jmuk/sylvan/pkg/chat/parts"
)

type contentType string

const (
	contentTypeThinking   contentType = "thinking"
	contentTypeToolUse    contentType = "tool_use"
	contentTypeToolResult contentType = "tool_result"
	contentTypeText       contentType = "text"
	contentTypeImage      contentType = "image"
	contentTypeDocument   contentType = "document"
)

type thinkingContent struct {
	Thinking  string      `json:"thinking"`
	Type      contentType `json:"type"`
	Signature string      `json:"signature"`
}

// toolUseContent si the type of content for the tool use.
type toolUseContent struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
	Type  contentType    `json:"type"`
}

type textContent struct {
	Text string      `json:"text"`
	Type contentType `json:"type"`
}

type imageContent struct {
	MediaType string      `json:"media_type"`
	Data      string      `json:"data"`
	Type      contentType `json:"type"`
}

func toImageContent(b *parts.Blob) *imageContent {
	return &imageContent{
		MediaType: b.MimeType,
		Data:      base64.StdEncoding.EncodeToString(b.Data),
		Type:      contentTypeImage,
	}
}

type documentSourceType string

const (
	documentSourceTypeText    documentSourceType = "text"
	documentSourceTypeContent documentSourceType = "content"
)

type documentSource struct {
	Data      string             `json:"data"`
	MediaType string             `json:"media_type"`
	Type      documentSourceType `json:"type"`
}

type cacheControlType string

const cacheControlTypeEphemeral cacheControlType = "ephemeral"

type documentCacheControl struct {
	Type cacheControlType `json:"type"`
	TTL  string           `json:"ttl,omitempty"`
}

type documentContent struct {
	Source       documentSource        `json:"source"`
	Type         contentType           `json:"type"`
	CacheControl *documentCacheControl `json:"cache_control,omitempty"`
}

func toDocumentContent(b *parts.Blob) *documentContent {
	return &documentContent{
		Source: documentSource{
			Data:      string(b.Data),
			MediaType: b.MimeType,
			Type:      documentSourceTypeContent,
		},
		Type: contentTypeDocument,
	}
}

// toolResultContent is the type of content for the tool use result.
type toolResultContent struct {
	ToolUseID string      `json:"tool_use_id"`
	Type      contentType `json:"type"`
	Content   any         `json:"content"`
	IsError   bool        `json:"is_error"`
}

type inputMessage struct {
	Content any        `json:"content"`
	Role    parts.Role `json:"role"`
}

// message keeps the part with the role, used to keep the
// conversation history.
type message struct {
	Part *parts.Part `json:"part"`
	Role parts.Role  `json:"role"`
}

func (m message) toInput() (inputMessage, error) {
	msg := inputMessage{Role: m.Role}
	if m.Part.Text != "" {
		// a text message or a thought.
		if m.Part.Thought {
			msg.Content = []thinkingContent{
				{
					Thinking:  m.Part.Text,
					Type:      contentTypeThinking,
					Signature: m.Part.ThinkingSignature,
				},
			}
		} else {
			msg.Content = m.Part.Text
		}
	} else if fc := m.Part.FunctionCall; fc != nil {
		msg.Content = []toolUseContent{
			{
				ID:    fc.ID,
				Name:  fc.Name,
				Input: fc.Args,
				Type:  contentTypeToolUse,
			},
		}
	} else if fr := m.Part.FunctionResponse; fr != nil {
		c := toolResultContent{
			ToolUseID: fr.ID,
			Type:      contentTypeToolResult,
			IsError:   fr.Error != nil,
		}
		if fr.Error != nil {
			c.Content = fr.Error.Error()
		} else {
			if len(fr.Parts) == 0 {
				encoded, err := json.Marshal(fr.Response)
				if err != nil {
					return inputMessage{}, err
				}
				c.Content = string(encoded)
			} else {
				var contents []any
				if fr.Response != nil {
					encoded, err := json.Marshal(fr.Response)
					if err != nil {
						return inputMessage{}, err
					}
					contents = append(contents, textContent{
						Text: string(encoded),
						Type: contentTypeText,
					})
				}
				for _, p := range fr.Parts {
					if p.Image != nil {
						contents = append(contents, toImageContent(p.Image))
					}
					if p.Text != "" {
						contents = append(contents, textContent{Text: p.Text, Type: contentTypeText})
					}
				}
				c.Content = contents
			}
		}
		msg.Content = []toolResultContent{c}
	} else if f := m.Part.File; f != nil {
		msg.Content = toDocumentContent(f)
	} else {
		return inputMessage{}, fmt.Errorf("unknown type of part: %v", m.Part)
	}
	return msg, nil
}
