package chat

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jmuk/sylvan/pkg/session"
	"github.com/jmuk/sylvan/pkg/tools"
	"github.com/manifoldco/promptui"
)

type Chat struct {
	chat Agent
	p    *promptui.Prompt
	trun *tools.ToolRunner
	s    *session.Session
	cwd  string
}

func New(chat Agent, trun *tools.ToolRunner, cwd string) (*Chat, error) {
	s, err := session.New(cwd)
	if err != nil {
		return nil, err
	}
	p := &promptui.Prompt{
		Label: ">",
	}
	return &Chat{
		chat: chat,
		p:    p,
		trun: trun,
		s:    s,
		cwd:  cwd,
	}, nil
}

func (c *Chat) Close() error {
	return c.s.Close()
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
	if err := c.s.Init(); err != nil {
		return err
	}
	l, err := c.s.GetLogger("chat")
	if err != nil {
		return err
	}
	ctx = c.s.With(ctx)
	msgs := []Part{{Text: input}}
	for {
		printed := false
		var nextMsgs []Part
		l.Debug("Sending", "messages", msgs)
		for part, err := range c.chat.SendMessageStream(ctx, msgs) {
			if err != nil {
				return err
			}
			l.Debug("Received message", "result", part)
			if part.Text != "" {
				fmt.Fprint(os.Stdout, part.Text)
				printed = true
			}
			if call := part.FunctionCall; call != nil {
				commandCtx, cancel := context.WithTimeout(ctx, time.Minute)
				resp, err := c.trun.Run(commandCtx, call.Name, call.Args)
				cancel()
				if err != nil {
					var toolErr *tools.ToolError
					if !errors.As(err, &toolErr) {
						return err
					}
					err = toolErr.Unwrap()
				}
				nextMsgs = append(nextMsgs, Part{
					FunctionResponse: &FunctionResponse{
						ID:       part.FunctionCall.ID,
						Name:     part.FunctionCall.Name,
						Response: resp,
						Error:    err,
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
