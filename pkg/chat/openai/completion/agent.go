// package completion implements agent using OpenAI completion API.
package completion

import (
	"context"
	"iter"
	"log/slog"

	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/jmuk/sylvan/pkg/session"
	"github.com/openai/openai-go/v3"
)

// Agent is an implementation of agent using OpenAI completion API.
type Agent struct {
	client       openai.ChatCompletionService
	modelName    string
	systemPrompt string

	tools []openai.ChatCompletionToolUnionParam

	history     []openai.ChatCompletionMessageParamUnion
	historyFile string
}

func (a *Agent) updateHistory() error {
	return nil
}

// SendMessageStream implements agent.Agent interface.
func (a *Agent) SendMessageStream(ctx context.Context, ps []parts.Part) iter.Seq2[*parts.Part, error] {
	return func(yield func(*parts.Part, error) bool) {
		logger := slog.New(slog.DiscardHandler)
		if s, sok := session.FromContext(ctx); sok {
			if gotLogger, err := s.GetLogger("openai"); err != nil {
				if !yield(nil, err) {
					return
				}
			} else {
				logger = gotLogger
			}
		}

		messages := append([]openai.ChatCompletionMessageParamUnion{}, a.history...)
		for _, p := range ps {
			msg, ok, err := partToMessage(p, logger)
			if err != nil {
				if !yield(nil, err) {
					return
				}
			} else if ok {
				messages = append(messages, msg)
			}
		}
		logger.Debug("sending", "messages", messages)
		st := a.client.NewStreaming(ctx, openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    a.modelName,
			Tools:    a.tools,
		})

		proc := &outputProcessor{
			logger: logger,
		}

		proc.processStream(st, yield)
		a.history = append(a.history, proc.history...)
		if err := a.updateHistory(); err != nil {
			if !yield(nil, err) {
				return
			}
		}

		if st.Err() != nil {
			yield(nil, st.Err())
		}
	}
}
