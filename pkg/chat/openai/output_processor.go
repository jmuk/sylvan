package openai

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/responses"
)

type outputProcessor struct {
	logger     *slog.Logger
	fc         *parts.FunctionCall
	param      string
	responseID string
}

func (p *outputProcessor) process(ev responses.ResponseStreamEventUnion) (*parts.Part, error) {
	p.logger.Debug("Received event", "event", ev)
	switch variant := ev.AsAny().(type) {
	case responses.ResponseCreatedEvent:
		p.responseID = variant.Response.ID
	case responses.ResponseErrorEvent:
		return nil, fmt.Errorf("failed: %s %s %s", variant.Code, variant.Message, variant.Param)
	case responses.ResponseTextDeltaEvent:
		return &parts.Part{Text: variant.Delta}, nil
	case responses.ResponseTextDoneEvent:
		return &parts.Part{Text: variant.Text}, nil
	case responses.ResponseReasoningTextDeltaEvent:
		return &parts.Part{Text: variant.Delta, Thought: true}, nil
	case responses.ResponseReasoningTextDoneEvent:
		return &parts.Part{Text: variant.Text, Thought: true}, nil
	case responses.ResponseOutputItemAddedEvent:
		if variant.Item.Type == "function_call" {
			call := variant.Item.AsFunctionCall()
			p.fc = &parts.FunctionCall{
				Name: call.Name,
				ID:   call.CallID,
			}
			p.param = call.Arguments
		}
		return nil, nil
	case responses.ResponseFunctionCallArgumentsDeltaEvent:
		p.param += variant.Delta
		return nil, nil
	case responses.ResponseFunctionCallArgumentsDoneEvent:
		if p.fc == nil {
			p.logger.Error("missing function call event")
			return nil, nil
		}
		p.fc.Args = map[string]any{}
		defer func() {
			p.fc = nil
			p.param = ""
		}()
		if err := json.Unmarshal([]byte(p.param), &p.fc.Args); err != nil {
			return nil, err
		}
		return &parts.Part{FunctionCall: p.fc}, nil
	default:
		p.logger.Error("unknown response")
	}
	return nil, nil
}

func (p *outputProcessor) processStream(st *ssestream.Stream[responses.ResponseStreamEventUnion], yield func(*parts.Part, error) bool) {
	for st.Next() {
		part, err := p.process(st.Current())
		if err != nil {
			if !yield(nil, err) {
				return
			}
		}
		if part != nil {
			if !yield(part, nil) {
				return
			}
		}
	}
}
