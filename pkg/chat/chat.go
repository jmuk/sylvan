package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jmuk/sylvan/pkg/ai"
	"github.com/jmuk/sylvan/pkg/tools"
	"github.com/manifoldco/promptui"
)

type Chat struct {
	chat   ai.Agent
	p      *promptui.Prompt
	logger *slog.Logger
	trun   *tools.ToolRunner
}

func New(chat ai.Agent, handler slog.Handler, trun *tools.ToolRunner) *Chat {
	p := &promptui.Prompt{
		Label: ">",
	}
	return &Chat{
		chat:   chat,
		p:      p,
		logger: slog.New(handler),
		trun:   trun,
	}
}

func (c *Chat) RunLoop(ctx context.Context) error {
	for {
		line, err := c.p.Run()
		if err != nil {
			if errors.Is(err, promptui.ErrEOF) || errors.Is(err, promptui.ErrAbort) {
				return nil
			}
			return err
		}
		if strings.HasPrefix(line, "/q") {
			return nil
		}
		if err := c.HandleMessage(ctx, line); err != nil {
			return err
		}
	}
}

func (c *Chat) HandleMessage(ctx context.Context, input string) error {
	msgs := []ai.Part{{Text: input}}
	for {
		printed := false
		var nextMsgs []ai.Part
		for part, err := range c.chat.SendMessageStream(ctx, msgs) {
			if err != nil {
				return err
			}
			c.logger.Debug("Received message", "result", part)
			if part.Text != "" {
				fmt.Print(part.Text)
				printed = true
			}
			if call := part.FunctionCall; call != nil {
				commandCtx, cancel := context.WithTimeout(ctx, time.Minute)
				resp, err := c.trun.Run(commandCtx, call.Name, call.Args)
				cancel()
				if err != nil {
					return err
				}
				nextMsgs = append(nextMsgs, ai.Part{
					FunctionResponse: &ai.FunctionResponse{
						ID:       part.FunctionCall.ID,
						Name:     part.FunctionCall.Name,
						Response: resp,
					},
				})
			}
		}
		if printed {
			fmt.Println()
		}
		if len(nextMsgs) == 0 {
			break
		}
		msgs = nextMsgs
	}
	return nil
}
