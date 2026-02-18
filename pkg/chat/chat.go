// package chat provides the chat UI and the event loop.
package chat

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/chzyer/readline"
	"github.com/jmuk/sylvan/pkg/chat/agent"
	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/jmuk/sylvan/pkg/config"
	"github.com/jmuk/sylvan/pkg/session"
	"github.com/jmuk/sylvan/pkg/tools"
	"github.com/manifoldco/promptui"
)

type chatSession struct {
	cfg    *config.Config
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
	var err error
	cs.cfg, err = cs.s.LoadConfig()
	if err != nil {
		return err
	}
	cs.mgrs = tools.NewManagers(cwd, cs.cfg)
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
	prompt, err := SystemPrompt(cwd, cs.cfg.AgentsFile)
	if err != nil {
		return err
	}
	cs.ag, err = newAgent(ctx, cs.cfg, prompt, toolDefs)
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

// Chat keeps the current chat session.
type Chat struct {
	rl   *readline.Instance
	cs   *chatSession
	cwd  string
	root *os.Root
}

// New creates a new Chat.
func New(ctx context.Context, cwd string) (*Chat, error) {
	s, err := session.New(cwd)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(cwd)
	if err != nil {
		return nil, err
	}
	completer := newCombinedCompleter(root)
	rl, err := readline.NewEx(&readline.Config{
		Prompt:       "> ",
		HistoryLimit: -1,
		AutoComplete: completer,
	})
	if err != nil {
		return nil, err
	}
	return &Chat{
		rl:   rl,
		cs:   &chatSession{s: s},
		cwd:  cwd,
		root: root,
	}, nil
}

// Close cleans up the states of the chat.
func (c *Chat) Close() error {
	if c.cs != nil {
		return c.cs.Close()
	}
	return nil
}

// RunLoop starts a run loop of REPL.
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
		case commandMCP:
			if err := c.handleMCPCommand(ctx, args); err != nil {
				return err
			}
			continue
		case commandModels:
			if err := c.handleModelsCommand(ctx, args); err != nil {
				return err
			}
			continue
		case commandBackends:
			if err := c.handleBackendsCommand(ctx, args); err != nil {
				return err
			}
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

// parseInput parses the input text and hopefully find the pattern of
// file names prefixed by the '@'.
// Note:
//   - '\@' isn't the pattern for the input.
//   - '@' within the backquotes (`@foo` or ```@foo```) don't count.
func (c *Chat) parseInput(input string) ([]string, error) {
	quote := ""
	atPos := -1
	var files []string
	var cand []byte
	for i := 0; i < len(input); i++ {
		ch := input[i]
		if ch == '\\' && len(quote) <= 1 {
			// Skip a character after the backslash.
			i++
			continue
		}
		if len(quote) > 0 && strings.HasPrefix(input[i:], quote) {
			i += len(quote) - 1
			quote = ""
			continue
		}
		if atPos < 0 {
			if ch == '`' {
				if strings.HasPrefix(input[i:], "```") {
					quote = "```"
					i += 2
				} else {
					quote = "`"
				}
				continue
			}
			if ch == '@' {
				atPos = i
			}
		} else {
			if unicode.IsSpace(rune(ch)) {
				candStr := string(cand)
				if fi, err := c.root.Stat(candStr); err != nil {
					// Safe to ignore not-found error, that turns out not a file.
					if !os.IsNotExist(err) {
						return nil, err
					}
				} else if fi.Mode().IsRegular() {
					files = append(files, candStr)
				}
				atPos = -1
				cand = nil
			} else {
				cand = append(cand, ch)
			}
		}
	}
	if atPos >= 0 {
		candStr := string(cand)
		if fi, err := c.root.Stat(candStr); err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		} else if fi.Mode().IsRegular() {
			files = append(files, candStr)
		}
	}
	return files, nil
}

// HandleMessage handles a input message, sends to an agent, processes the response.
func (c *Chat) HandleMessage(ctx context.Context, input string) error {
	l, err := c.cs.s.GetLogger("chat")
	if err != nil {
		return err
	}

	files, err := c.parseInput(input)
	if err != nil {
		return err
	}
	msgs := []parts.Part{{Text: input}}
	for _, file := range files {
		content, err := c.root.ReadFile(file)
		if err != nil {
			if !os.IsNotExist(err) && !errors.Is(err, fs.ErrPermission) {
				return err
			}
		} else {
			msgs = append(msgs, parts.Part{
				File: &parts.Blob{
					Data:     content,
					MimeType: "text/plain",
					Filename: file,
				},
			})
		}
	}
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
