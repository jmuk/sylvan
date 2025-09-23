package claude

import (
	"encoding/json"
	"fmt"

	"github.com/jmuk/sylvan/pkg/chat"
)

type contentType string

const (
	contentTypeThinking   contentType = "thinking"
	contentTypeToolUse    contentType = "tool_use"
	contentTypeToolResult contentType = "tool_result"
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

// toolResultContent is the type of content for the tool use result.
type toolResultContent struct {
	ToolUseID string      `json:"tool_use_id"`
	Type      contentType `json:"type"`
	Content   string      `json:"content"`
	IsError   bool        `json:"is_error"`
}

type inputMessage struct {
	Content any       `json:"content"`
	Role    chat.Role `json:"role"`
}

// message keeps the part with the role, used to keep the
// conversation history.
type message struct {
	Part chat.Part `json:"part"`
	Role chat.Role `json:"role"`
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
				return inputMessage{}, err
			}
			body = string(encoded)
		}
		msg.Content = []toolResultContent{
			{
				ToolUseID: fr.ID,
				Type:      contentTypeToolResult,
				Content:   body,
				IsError:   fr.Error != nil,
			},
		}
	} else {
		return inputMessage{}, fmt.Errorf("unknown type of part: %v", m.Part)
	}
	return msg, nil
}
