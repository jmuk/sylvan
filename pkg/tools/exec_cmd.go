package tools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var commandDefaultTimeout = time.Minute

type execCommandRequest struct {
	CommandLine string `json:"command_line" jsonschema:"the full command line string; note that this does not evaluate glob patterns"`
}

type execCommandResponse struct {
	ReturnCode int    `json:"return_code" jsonschema:"description=the return code of the command; 0 means successful"`
	Output     string `json:"output" jsonschema:"the standard output of the command"`
	ErrorOut   string `json:"error_out" jsonschema:"the standard error output of the command"`
}

var whiteSpaces = regexp.MustCompile(`\s+`)

type bufferWithViewer struct {
	io.Writer
	viewer *io.PipeReader
	pipe   *io.PipeWriter
	buffer *strings.Builder
	logger *slog.Logger
}

func newBufferWithViewer(prefix string, logger *slog.Logger) *bufferWithViewer {
	viewer, pipe := io.Pipe()
	buffer := &strings.Builder{}
	go func() {
		s := bufio.NewScanner(viewer)
		for s.Scan() {
			line := s.Text()
			if prefix == "" {
				fmt.Println(line)
			} else {
				fmt.Println(prefix, line)
			}
		}
		err := s.Err()
		if err != nil && err != io.EOF && err != io.ErrClosedPipe {
			slog.Error("Failed to read", "error", err)
		}
	}()
	return &bufferWithViewer{
		Writer: io.MultiWriter(pipe, buffer),
		viewer: viewer,
		pipe:   pipe,
		buffer: buffer,
		logger: logger.With("prefix", prefix),
	}
}

func (b *bufferWithViewer) Close() error {
	return errors.Join(
		b.viewer.Close(),
		b.pipe.Close(),
	)
}

func (b *bufferWithViewer) String() string {
	return b.buffer.String()
}

type ExecTool struct {
}

func NewExecTool() *ExecTool {
	return &ExecTool{}
}

func (et *ExecTool) execCommand(ctx context.Context, req execCommandRequest) execCommandResponse {
	logger := getLogger(ctx).With("commandline", req.CommandLine)
	logger.Debug("Start execution")
	params := whiteSpaces.Split(req.CommandLine, -1)
	logger = logger.With("command", params[0])

	ctx, cancel := context.WithTimeout(context.Background(), commandDefaultTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, params[0], params[1:]...)
	stdout := newBufferWithViewer("", logger)
	stderr := newBufferWithViewer("error:", logger)
	defer stdout.Close()
	defer stderr.Close()
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil && !errors.Is(err, &exec.ExitError{}) {
		logger.Error("Failed to execute", "error", err)
		return execCommandResponse{
			ReturnCode: -1,
			ErrorOut:   err.Error(),
		}
	}
	state := cmd.ProcessState
	if state == nil {
		logger.Error("Missing process state")
		resp := execCommandResponse{
			ReturnCode: -1,
			ErrorOut:   "Unknown error",
		}
		if err != nil {
			resp.ErrorOut = err.Error()
		}
		return resp
	}

	logger.Debug("Execution completed", "return_code", state.ExitCode())

	return execCommandResponse{
		ReturnCode: state.ExitCode(),
		Output:     stdout.String(),
		ErrorOut:   stderr.String(),
	}
}

func (et *ExecTool) ToolDefs() []ToolDefinition {
	return []ToolDefinition{
		&toolDefinition[execCommandRequest, execCommandResponse]{
			name:        "exec_command",
			description: "execute a command",
			proc:        et.execCommand,
		},
	}
}
