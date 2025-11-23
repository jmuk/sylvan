package completion

import (
	"encoding/json"
	"log/slog"

	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/ssestream"
)

type outputProcessor struct {
	logger  *slog.Logger
	history []openai.ChatCompletionMessageParamUnion
}

func (p *outputProcessor) isLastHistoryText() bool {
	if len(p.history) == 0 {
		return false
	}
	last := p.history[len(p.history)-1]
	ass := last.OfAssistant
	if ass == nil {
		return false
	}
	content := ass.Content
	return len(content.OfArrayOfContentParts) > 0
}

func (p *outputProcessor) appendHistory(text string) {
	msg := openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion{
		OfText: &openai.ChatCompletionContentPartTextParam{
			Text: text,
		},
	}

	if p.isLastHistoryText() {
		p.history[len(p.history)-1].OfAssistant.Content.OfArrayOfContentParts = append(
			p.history[len(p.history)-1].OfAssistant.Content.OfArrayOfContentParts, msg,
		)
	}

	p.history = append(p.history, openai.AssistantMessage([]openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion{msg}))
}

func (p *outputProcessor) process(ev openai.ChatCompletionChunk) ([]*parts.Part, error) {
	p.logger.Debug("Received event", "event", ev)
	if len(ev.Choices) == 0 {
		return nil, nil
	}
	delta := ev.Choices[0].Delta
	if len(delta.ToolCalls) > 0 {
		var ps []*parts.Part
		var callHistory []openai.ChatCompletionMessageToolCallUnionParam
		for _, tc := range delta.ToolCalls {
			parsed := map[string]any{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &parsed); err != nil {
				return nil, err
			}
			ps = append(ps, &parts.Part{
				FunctionCall: &parts.FunctionCall{
					ID:   tc.ID,
					Name: tc.Function.Name,
					Args: parsed,
				},
			})
			callHistory = append(callHistory, openai.ChatCompletionMessageToolCallUnionParam{
				OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
					ID: tc.ID,
					Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
						Arguments: tc.Function.Arguments,
						Name:      tc.Function.Name,
					},
					Type: "function",
				},
			})
		}
		p.history = append(p.history, openai.ChatCompletionMessageParamUnion{
			OfAssistant: &openai.ChatCompletionAssistantMessageParam{
				ToolCalls: callHistory,
			},
		})
		return ps, nil
	}
	if delta.Content != "" {
		return nil, nil
	}

	p.appendHistory(delta.Content)
	return []*parts.Part{{Text: delta.Content}}, nil
}

func (p *outputProcessor) processStream(st *ssestream.Stream[openai.ChatCompletionChunk], yield func(*parts.Part, error) bool) {
	for st.Next() {
		parts, err := p.process(st.Current())
		if err != nil {
			if !yield(nil, err) {
				return
			}
		}
		for _, part := range parts {
			if !yield(part, nil) {
				return
			}
		}
	}
}
