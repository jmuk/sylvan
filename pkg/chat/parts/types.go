package parts

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type FunctionCall struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type FunctionResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Response any     `json:"response"`
	Parts    []*Part `json:"parts"`
	Error    error   `json:"error"`
}

func (fr *FunctionResponse) MarshalJSON() ([]byte, error) {
	var m map[string]any
	if fr != nil {
		m = map[string]any{
			"id":       fr.ID,
			"name":     fr.Name,
			"response": fr.Response,
		}
		if fr.Error != nil {
			m["error"] = fr.Error.Error()
		}
	}
	return json.Marshal(m)
}

func (fr *FunctionResponse) UnmarshalJSON(data []byte) error {
	m := map[string]any{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	if id, ok := m["id"]; ok {
		if fr.ID, ok = id.(string); !ok {
			return fmt.Errorf(`field "id" must be a string`)
		}
	}
	if name, ok := m["name"]; ok {
		if fr.Name, ok = name.(string); !ok {
			return fmt.Errorf(`field "name" must be a string`)
		}
	}
	if response, ok := m["response"]; ok {
		if fr.Response, ok = response.(map[string]any); !ok {
			return fmt.Errorf(`field "response" must be a map`)
		}
	}
	if errdata, ok := m["error"]; ok {
		if errstr, ok := errdata.(string); ok {
			fr.Error = errors.New(errstr)
		} else {
			return fmt.Errorf(`field "error" must be a string`)
		}
	}
	return nil
}

type Blob struct {
	Data     []byte `json:"data"`
	MimeType string `json:"mime_type"`
	Filename string `json:"filename,omitempty"`
}

type FileRef struct {
	URL      string `json:"url"`
	MimeType string `json:"mime_type"`
}

type Part struct {
	Thought           bool              `json:"thought,omitempty"`
	Text              string            `json:"text,omitempty"`
	ThinkingSignature string            `json:"thinking_signature,omitempty"`
	FunctionCall      *FunctionCall     `json:"function_call,omitempty"`
	FunctionResponse  *FunctionResponse `json:"function_response,omitempty"`
	Image             *Blob             `json:"image,omitempty"`
	Audio             *Blob             `json:"audio,omitempty"`
	File              *Blob             `json:"file,omitempty"`
	FileRef           *FileRef          `json:"fileref,omitempty"`
}
