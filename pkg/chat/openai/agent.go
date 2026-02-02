// package openai implements Agent Using OpenAI responses API.
package openai

import (
	"context"
	"iter"
	"log/slog"
	"os"

	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/jmuk/sylvan/pkg/session"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

// Agent is an implementation of agent using OpenAI responses API.
type Agent struct {
	client      responses.ResponseService
	historyFile string

	model        shared.ResponsesModel
	systemPrompt string
	tools        []responses.ToolUnionParam

	previousResponseID param.Opt[string]
}

func (a *Agent) updateHistory(responseID string) error {
	if a.historyFile == "" {
		return nil
	}

	if _, err := os.Stat(a.historyFile); err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(a.historyFile, []byte(responseID+"\n"), 0600)
		}
		return err
	}

	f, err := os.OpenFile(a.historyFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(responseID + "\n")
	return err
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

		var input responses.ResponseNewParamsInputUnion
		if len(ps) == 1 && ps[0].Text != "" {
			input = responses.ResponseNewParamsInputUnion{
				OfString: param.NewOpt(ps[0].Text),
			}
			logger.Debug("input", "input", input)
		} else {
			for _, p := range ps {
				msg, ok, err := partToInput(p, logger)
				if err != nil {
					if !yield(nil, err) {
						return
					}
				} else if ok {
					input.OfInputItemList = append(input.OfInputItemList, msg)
				}
			}
		}
		logger.Debug("sending", "input", input)
		st := a.client.NewStreaming(ctx, responses.ResponseNewParams{
			Instructions:       param.NewOpt(a.systemPrompt),
			PreviousResponseID: a.previousResponseID,
			Input:              input,
			Model:              a.model,
			Tools:              a.tools,
		})

		proc := &outputProcessor{
			logger: logger,
		}

		proc.processStream(st, yield)
		if proc.responseID != "" {
			a.previousResponseID = param.NewOpt(proc.responseID)
			if err := a.updateHistory(proc.responseID); err != nil {
				if !yield(nil, err) {
					return
				}
			}
		}

		if st.Err() != nil {
			yield(nil, st.Err())
		}
	}
}
