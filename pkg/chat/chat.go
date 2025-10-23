package chat

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/jmuk/sylvan/pkg/chat/agent"
	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/jmuk/sylvan/pkg/session"
	"github.com/jmuk/sylvan/pkg/tools"
	"github.com/manifoldco/promptui"
)

type chatSession struct {
	s      *session.Session
	ag     agent.Agent
	mgrs   []tools.Manager
	runner *tools.ToolRunner
}

func (cs *chatSession) maybeInit(ctx context.Context, cwd string) error {
	if cs.ag != nil {
		return nil
	}
	ctx = cs.With(ctx)
	cfg, err := cs.s.LoadConfig()
	if err != nil {
		return err
	}
	cs.mgrs = tools.NewManagers(cwd, cfg)
	var toolDefs []tools.ToolDefinition
	for _, mgr := range cs.mgrs {
		dfs, err := mgr.ToolDefs(ctx)
		if err != nil {
			return err
		}
		toolDefs = append(toolDefs, dfs...)
	}
	cs.runner, err = tools.NewToolRunner(toolDefs)
	if err != nil {
		return err
	}
	cs.ag, err = newAgent(ctx, cfg, SystemPrompt, toolDefs)
	if err != nil {
		return err
	}
	return nil
}

func (cs *chatSession) Close() error {
	var errs error
	for _, mgr := range cs.mgrs {
		errs = errors.Join(errs, mgr.Close())
	}
	errs = errors.Join(errs, cs.s.Close())
	return errs
}

func (cs *chatSession) With(ctx context.Context) context.Context {
	return cs.s.With(ctx)
}

type Chat struct {
	rl  *readline.Instance
	cs  *chatSession
	cwd string
}

func New(ctx context.Context, cwd string) (*Chat, error) {
	s, err := session.New(cwd)
	if err != nil {
		return nil, err
	}
	rl, err := readline.NewEx(&readline.Config{
		Prompt:       "> ",
		HistoryLimit: -1,
	})
	if err != nil {
		return nil, err
	}
	return &Chat{
		rl:  rl,
		cs:  &chatSession{s: s},
		cwd: cwd,
	}, nil
}

func (c *Chat) Close() error {
	if c.cs != nil {
		return c.cs.Close()
	}
	return nil
}

type command int

const (
	commandNone command = iota
	commandQuit
	commandSession
	commandList
)

func (c *Chat) parseCommand(line string) (command, []string) {
	line = strings.TrimSpace(line)
	if line[0] != '/' {
		return commandNone, nil
	}
	words := strings.Fields(line[1:])
	if len(words) == 0 {
		return commandNone, nil
	}
	command := words[0]
	switch strings.ToLower(command) {
	case "q", "quit":
		return commandQuit, words[1:]
	case "session":
		return commandSession, words[1:]
	case "commands", "help", "list-commands":
		return commandList, words[1:]
	default:
		fmt.Printf("Unknown command %s, ignoring...\n", command)
		return commandNone, nil
	}
}

func (c *Chat) chooseNewSession() (*session.Session, error) {
	// Choose a new session.
	sessions, err := session.ListSessions(c.cwd)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		fmt.Println("No sessions found to select")
		return nil, nil
	}
	sort.Slice(sessions, func(i, j int) bool {
		t1 := sessions[i].Timestamp()
		t2 := sessions[j].Timestamp()
		// Newer one comes earlier.
		return t1.After(t2)
	})
	var foundExisting bool
	for _, s := range sessions {
		if s.ID() == c.cs.s.ID() {
			foundExisting = true
			break
		}
	}
	if !foundExisting {
		sessions = append([]*session.Session{c.cs.s}, sessions...)
	}
	items := make([]string, 0, len(sessions))
	var cursorPos int
	for i, s := range sessions {
		item := fmt.Sprintf("%s at %s", s.ID(), s.Timestamp().Format(time.RFC1123Z))
		if s.ID() == c.cs.s.ID() {
			item += " (current session)"
			cursorPos = i
		}
		items = append(items, item)
	}
	sel := promptui.Select{
		Label:     "Select the session to switch",
		Items:     items,
		CursorPos: cursorPos,
	}
	idx, _, err := sel.Run()
	if err != nil {
		return nil, err
	}
	if sessions[idx].ID() == c.cs.s.ID() {
		return nil, nil
	}
	return sessions[idx], nil
}

func (c *Chat) handleSessionCommands(args []string) (bool, error) {
	var newSession *session.Session
	if len(args) == 0 {
		var err error
		newSession, err = c.chooseNewSession()
		if err != nil {
			return false, err
		}
	} else {
		sessionID := args[0]
		if sessionID == "last" {
			// Choose the last session.
			sessions, err := session.ListSessions(c.cwd)
			if err != nil {
				return false, err
			}
			if sessions[0].ID() != c.cs.s.ID() {
				newSession = sessions[0]
			}
		} else {
			var err error
			newSession, err = session.NewFromID(sessionID)
			if err != nil {
				return false, err
			}
		}
	}
	if newSession == nil {
		return false, nil
	}
	if err := c.cs.Close(); err != nil {
		return false, err
	}
	c.cs = &chatSession{s: newSession}
	fmt.Printf("Session is updated to %s\n", c.cs.s.ID())
	return true, nil
}

func (c *Chat) handleListCommand() {
	fmt.Println(`List of possible commands:
- list, commands, help, or ?: this command -- show the list of commands.
- session: choose a new session.
- q, quit: quit this program.
	`)
}

func (c *Chat) RunLoop(ctx context.Context) error {
	ctx = c.cs.With(ctx)
	for {
		line, err := c.rl.Readline()
		if err != nil {
			if errors.Is(err, promptui.ErrEOF) || errors.Is(err, promptui.ErrAbort) {
				return nil
			}
			return err
		}
		command, args := c.parseCommand(line)
		switch command {
		case commandQuit:
			return nil
		case commandSession:
			sessionUpdated, err := c.handleSessionCommands(args)
			if err != nil {
				return err
			}
			if sessionUpdated {
				ctx = c.cs.With(ctx)
			}
			continue
		case commandList:
			c.handleListCommand()
			continue
		}

		if err := c.cs.maybeInit(ctx, c.cwd); err != nil {
			return err
		}
		if err := c.HandleMessage(ctx, line); err != nil {
			return err
		}
	}
}

func (c *Chat) HandleMessage(ctx context.Context, input string) error {
	l, err := c.cs.s.GetLogger("chat")
	if err != nil {
		return err
	}
	msgs := []parts.Part{{Text: input}}
	for {
		printed := false
		var nextMsgs []parts.Part
		l.Debug("Sending", "messages", msgs)
		for part, err := range c.cs.ag.SendMessageStream(ctx, msgs) {
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
				resp, ps, err := c.cs.runner.Run(commandCtx, call.Name, call.Args)
				cancel()
				if err != nil {
					var toolErr *tools.ToolError
					if !errors.As(err, &toolErr) {
						return err
					}
					err = toolErr.Unwrap()
				}
				nextMsgs = append(nextMsgs, parts.Part{
					FunctionResponse: &parts.FunctionResponse{
						ID:       part.FunctionCall.ID,
						Name:     part.FunctionCall.Name,
						Response: resp,
						Parts:    ps,
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
