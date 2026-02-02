// package parts defines the types of messages and its parts.
package parts

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// Role is the role of a message.
type Role string

const (
	// User role.
	RoleUser Role = "user"

	// Role of the assistant / agent.
	RoleAssistant Role = "assistant"
)

// FunctionCall is the message to call a function.
type FunctionCall struct {
	// ID of the call.
	ID string `json:"id"`
	// The name of the function.
	Name string `json:"name"`
	// Arguments to the function.
	Args map[string]any `json:"args"`
}

// FunctionResponse is the message sent back to the agent as the result
// of a function call.
type FunctionResponse struct {
	// ID of the call.
	ID string `json:"id"`
	// The name of the function.
	Name string `json:"name"`
	// The response object.  Typically a string or a JSON-able dict.
	Response any `json:"response"`
	// Parts provides additional data to the response.
	Parts []*Part `json:"parts"`
	// Error stores the error content when the function call ends up with a failure.
	Error error `json:"error"`
}

// MarshalJSON implements json.Marshaler.
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

// UnmarshalJSON implements json.Unmarshaler.
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

// Blob defines a blob data object (e.g. image).
type Blob struct {
	// The content binary data.
	Data []byte `json:"data"`
	// The mime-type of the data.
	MimeType string `json:"mime_type"`
	// The name of the file when supplied.
	Filename string `json:"filename,omitempty"`
}

// DataURL returns a data URL containing the blob data.
func (b *Blob) DataURL() string {
	return fmt.Sprintf("data:%s;base64,%s", b.MimeType, base64.StdEncoding.EncodeToString(b.Data))
}

// FileRef defines a reference to a file.
type FileRef struct {
	// URL of the file.
	URL string `json:"url"`
	// The mime type of the file.
	MimeType string `json:"mime_type"`
}

// Part is a part, or a segment, of messages.
type Part struct {
	// Set to true if it is a thinking content.
	Thought bool `json:"thought,omitempty"`

	// The text content.
	Text string `json:"text,omitempty"`

	// The thinking signature.
	ThinkingSignature string `json:"thinking_signature,omitempty"`

	// The function call.
	FunctionCall *FunctionCall `json:"function_call,omitempty"`

	// The response to a function call.
	FunctionResponse *FunctionResponse `json:"function_response,omitempty"`

	// Image.
	Image *Blob `json:"image,omitempty"`

	// Audio data.
	Audio *Blob `json:"audio,omitempty"`

	// Misc file data.
	File *Blob `json:"file,omitempty"`

	// Reference to an external resource.
	FileRef *FileRef `json:"fileref,omitempty"`
}
