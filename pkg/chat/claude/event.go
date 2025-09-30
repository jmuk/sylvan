package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"

	"github.com/jmuk/sylvan/pkg/chat"
	"github.com/jmuk/sylvan/pkg/sse"
)

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

type eventProcessor struct {
	scanner *sse.Scanner
	agent   *Agent

	currentBlock *contentBlock
}

func newEventProcessor(reader io.Reader, agent *Agent) *eventProcessor {
	return &eventProcessor{
		scanner: sse.NewScanner(reader),
		agent:   agent,
	}
}

func (ep *eventProcessor) processContentBlockStart(ev *sse.Event) (*chat.Part, error) {
	if ep.currentBlock != nil {
		return nil, fmt.Errorf("content block start appears before closing a previous one")
	}
	ep.currentBlock = &contentBlock{}
	if err := json.Unmarshal([]byte(ev.Data), ep.currentBlock); err != nil {
		return nil, err
	}
	cb := ep.currentBlock.ContentBlock
	if cb.Type == blockTypeText && cb.Text != "" {
		return &chat.Part{Text: cb.Text}, nil
	} else if cb.Type == blockTypeThinking && cb.Thinking != "" {
		return &chat.Part{Text: cb.Thinking, Thought: true}, nil
	}
	return nil, nil
}

func (ep *eventProcessor) processDeltaText(delta *contentBlockDelta) (*chat.Part, error) {
	cb := ep.currentBlock
	if cb.ContentBlock.Type != blockTypeText {
		return nil, fmt.Errorf("type mismatch: want %s got text", cb.Type)
	}
	cb.ContentBlock.Text += delta.Delta.Text
	return &chat.Part{Text: delta.Delta.Text}, nil
}

func (ep *eventProcessor) processDeltaJSON(delta *contentBlockDelta) (*chat.Part, error) {
	cb := ep.currentBlock
	if cb.ContentBlock.Type != blockTypeToolUse {
		return nil, fmt.Errorf("type mismatch: want %s got partial_json", cb.ContentBlock.Type)
	}
	// Use Text field to accumulate partial JSON then parse it.
	cb.ContentBlock.Text += delta.Delta.PartialJSON
	// not produce a part yet, it's still partial.
	return nil, nil
}

func (ep *eventProcessor) processDeltaThinking(delta *contentBlockDelta) (*chat.Part, error) {
	cb := ep.currentBlock
	if cb.ContentBlock.Type != blockTypeThinking {
		return nil, fmt.Errorf("type mismatch: want %s got thinking", cb.ContentBlock.Type)
	}
	cb.ContentBlock.Thinking += delta.Delta.Thinking
	return &chat.Part{Text: delta.Delta.Thinking, Thought: true}, nil
}

func (ep *eventProcessor) processDeltaSignature(delta *contentBlockDelta) (*chat.Part, error) {
	ep.currentBlock.ContentBlock.Signature += delta.Delta.Signature
	return nil, nil
}

func (ep *eventProcessor) processContentBlockDelta(ev *sse.Event) (*chat.Part, error) {
	delta := &contentBlockDelta{}
	if err := json.Unmarshal([]byte(ev.Data), delta); err != nil {
		return nil, err
	}
	if ep.currentBlock == nil {
		return nil, fmt.Errorf("missing content block start")
	}
	if ep.currentBlock.Index != delta.Index {
		return nil, fmt.Errorf("index mismatch: want %d got %d", ep.currentBlock.Index, delta.Index)
	}
	switch delta.Delta.Type {
	case deltaTypeText:
		return ep.processDeltaText(delta)
	case deltaTypeJSON:
		return ep.processDeltaJSON(delta)
	case deltaTypeThinking:
		return ep.processDeltaThinking(delta)
	case deltaTypeSignature:
		return ep.processDeltaSignature(delta)
	}
	return nil, fmt.Errorf("unknown delta type %s", delta.Delta.Type)
}

func (ep *eventProcessor) processContentBlockStop(ev *sse.Event) (*chat.Part, bool, error) {
	cb := ep.currentBlock
	if cb == nil {
		return nil, false, fmt.Errorf("content_block_stop appears without start")
	}
	defer func() {
		ep.currentBlock = nil
	}()
	switch cb.ContentBlock.Type {
	case blockTypeText:
		return &chat.Part{
			Text: cb.ContentBlock.Text,
		}, false, nil
	case blockTypeThinking:
		return &chat.Part{
			Thought:           true,
			Text:              cb.ContentBlock.Thinking,
			ThinkingSignature: cb.ContentBlock.Signature,
		}, false, nil
	case blockTypeToolUse:
		cb.ContentBlock.Input = map[string]any{}
		if err := json.Unmarshal([]byte(cb.ContentBlock.Text), &cb.ContentBlock.Input); err != nil {
			return nil, false, err
		}
		part := &chat.Part{FunctionCall: &chat.FunctionCall{
			ID:   cb.ContentBlock.ID,
			Name: cb.ContentBlock.Name,
			Args: cb.ContentBlock.Input,
		}}
		return part, true, nil
	}
	return nil, false, fmt.Errorf("unknown block type %s", cb.ContentBlock.Type)
}

func (ep *eventProcessor) processEvents() iter.Seq2[*chat.Part, error] {
	return func(yield func(*chat.Part, error) bool) {
		for {
			ev, err := ep.scanner.Scan()
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
				part, err := ep.processContentBlockStart(ev)
				if part != nil || err != nil {
					if !yield(part, err) {
						return
					}
				}
			case eventTypeContentBlockDelta:
				part, err := ep.processContentBlockDelta(ev)
				if part != nil || err != nil {
					if !yield(part, err) {
						return
					}
				}
			case eventTypeContentBlockStop:
				part, emit, err := ep.processContentBlockStop(ev)
				if emit || err != nil {
					if !yield(part, err) {
						return
					}
				}
				ep.agent.history = append(ep.agent.history, message{
					Part: part,
					Role: chat.RoleAssistant,
				})
			}
		}
	}
}
