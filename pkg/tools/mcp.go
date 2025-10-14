package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/jmuk/sylvan/pkg/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type transportFactory interface {
	newTransport() mcp.Transport
}

type commandFactory struct {
	command []string
}

func (cf *commandFactory) newTransport() mcp.Transport {
	return &mcp.CommandTransport{
		Command: exec.Command(cf.command[0], cf.command[1:]...),
	}
}

type httpFactory struct {
	endpoint string
	headers  http.Header
}

type headerAddingRoundTripper struct {
	headers      http.Header
	roundTripper http.RoundTripper
}

func (rt *headerAddingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	for k, v := range rt.headers {
		if _, ok := r.Header[k]; !ok {
			r.Header[k] = v
		}
	}
	return rt.roundTripper.RoundTrip(r)
}

func (hsf *httpFactory) newTransport() mcp.Transport {
	transport := &mcp.SSEClientTransport{
		Endpoint: hsf.endpoint,
	}
	if len(hsf.headers) > 0 {
		transport.HTTPClient = &http.Client{
			Transport: &headerAddingRoundTripper{
				headers:      hsf.headers,
				roundTripper: http.DefaultTransport,
			},
		}
	}
	return transport
}

type MCPTool struct {
	name    string
	client  *mcp.Client
	factory transportFactory

	clientSession *mcp.ClientSession
}

func newMCPTool() *MCPTool {
	var mt *MCPTool
	clientOpts := &mcp.ClientOptions{
		LoggingMessageHandler: func(ctx context.Context, msg *mcp.LoggingMessageRequest) {
			mt.logMessage(ctx, msg)
		},
	}
	mt = &MCPTool{
		client: mcp.NewClient(
			&mcp.Implementation{
				Name:    "sylvan",
				Version: "v0.0.1",
			},
			clientOpts,
		),
	}
	return mt
}

func NewCommandMCP(name string, command []string) *MCPTool {
	mt := newMCPTool()
	mt.factory = &commandFactory{
		command: command,
	}
	return mt
}

func NewHTTPMCP(name, endpoint string, headers map[string]string) *MCPTool {
	var h http.Header
	if len(headers) > 0 {
		h = http.Header{}
		for k, v := range headers {
			h.Add(k, v)
		}
	}
	mt := newMCPTool()
	mt.factory = &httpFactory{
		endpoint: endpoint,
		headers:  h,
	}
	return mt
}

type mcpToolDefinition struct {
	name        string
	description string

	inSchema  *jsonschema.Schema
	outSchema *jsonschema.Schema

	mt *MCPTool
}

func (mtd *mcpToolDefinition) Name() string {
	return mtd.name
}

func (mtd *mcpToolDefinition) Description() string {
	return mtd.description
}

func (mtd *mcpToolDefinition) RequestSchema() *jsonschema.Schema {
	return mtd.inSchema
}

func (mtd *mcpToolDefinition) ResponseSchema() *jsonschema.Schema {
	return mtd.outSchema
}

func (mtd *mcpToolDefinition) process(ctx context.Context, in map[string]any) (any, []*parts.Part, error) {
	return mtd.mt.process(ctx, mtd.name, in)
}

func (mt *MCPTool) Close() error {
	var err error
	if mt.clientSession != nil {
		err = mt.clientSession.Close()
		mt.clientSession = nil
	}
	return err
}

func (mt *MCPTool) logMessage(ctx context.Context, msg *mcp.LoggingMessageRequest) {
	s, ok := session.FromContext(ctx)
	if !ok {
		// Do nothing.
		return
	}
	loggerName := "mcp"
	p := msg.Params
	if p.Logger != "" && !strings.Contains(p.Logger, "/") {
		loggerName += "-" + p.Logger
	}
	logger, err := s.GetLogger(loggerName)
	if err != nil {
		log.Printf("Failed to get the logger: %v", err)
		return
	}
	lvl := slog.LevelInfo
	for _, lvl = range []slog.Level{slog.LevelDebug, slog.LevelError, slog.LevelWarn, slog.LevelInfo} {
		if lvl.String() == string(p.Level) {
			break
		}
	}
	logger.Log(ctx, lvl, "log request", "data", p.Data)
}

func (mt *MCPTool) newSession(ctx context.Context) (*mcp.ClientSession, error) {
	transport := mt.factory.newTransport()
	if s, ok := session.FromContext(ctx); ok {
		logname := strings.Replace(mt.name, "/", "_", -1)
		if len(logname) > 64 {
			logname = logname[:64]
		}
		logFile, err := s.GetLogFile(fmt.Sprintf("mcp-%s-log.txt", logname))
		if err != nil {
			return nil, err
		}
		transport = &mcp.LoggingTransport{
			Transport: transport,
			Writer:    logFile,
		}
	}
	return mt.client.Connect(ctx, transport, nil)
}

func (mt *MCPTool) getSession(ctx context.Context) (*mcp.ClientSession, error) {
	if mt.clientSession != nil {
		return mt.clientSession, nil
	}
	cs, err := mt.newSession(ctx)
	if err != nil {
		return nil, err
	}
	mt.clientSession = cs
	return cs, nil
}

func (mt *MCPTool) process(ctx context.Context, name string, in map[string]any) (any, []*parts.Part, error) {
	sess, err := mt.getSession(ctx)
	if err != nil {
		return nil, nil, err
	}
	var logger *slog.Logger
	if s, ok := session.FromContext(ctx); ok {
		var err error
		if logger, err = s.GetLogger("mcp-tool"); err != nil {
			return nil, nil, err
		}
	}
	result, err := sess.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: in,
	})
	if err != nil {
		return nil, nil, err
	}
	if result.IsError {
		messages := &strings.Builder{}
		for _, content := range result.Content {
			if tc, ok := content.(*mcp.TextContent); ok {
				if _, err := messages.WriteString(tc.Text); err != nil {
					return nil, nil, err
				}
			}
		}
		return nil, nil, &ToolError{errors.New(messages.String())}
	}
	var ps []*parts.Part
	for _, content := range result.Content {
		p := &parts.Part{}
		if ac, ok := content.(*mcp.AudioContent); ok {
			p.Audio = &parts.Blob{
				MimeType: ac.MIMEType,
				Data:     ac.Data,
			}
		} else if ic, ok := content.(*mcp.ImageContent); ok {
			p.Image = &parts.Blob{
				MimeType: ic.MIMEType,
				Data:     ic.Data,
			}
		} else if tc, ok := content.(*mcp.TextContent); ok {
			p.Text = tc.Text
		} else {
			logger.Error("unknown content", "content", content)
			continue
		}
		ps = append(ps, p)
	}
	return result.StructuredContent, ps, nil
}

func (mt *MCPTool) ToolDefs(ctx context.Context) ([]ToolDefinition, error) {
	session, err := mt.newSession(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close()
	var cursor string
	var results []ToolDefinition
	for {
		tools, err := session.ListTools(ctx, &mcp.ListToolsParams{
			Cursor: cursor,
		})
		if err != nil {
			return nil, err
		}
		for _, t := range tools.Tools {
			inSchemaEnc, err := json.Marshal(t.InputSchema)
			if err != nil {
				return nil, err
			}
			inSchema := &jsonschema.Schema{}
			if err := json.Unmarshal(inSchemaEnc, inSchema); err != nil {
				return nil, err
			}
			var outSchema *jsonschema.Schema
			if t.OutputSchema != nil {
				outSchemaEnc, err := json.Marshal(t.OutputSchema)
				if err != nil {
					return nil, err
				}
				outSchema = &jsonschema.Schema{}
				if err := json.Unmarshal(outSchemaEnc, outSchema); err != nil {
					return nil, err
				}
			}
			results = append(results, &mcpToolDefinition{
				name:        t.Name,
				description: t.Description,
				inSchema:    inSchema,
				outSchema:   outSchema,
				mt:          mt,
			})
		}
		if tools.NextCursor == "" {
			break
		}
		cursor = tools.NextCursor
	}
	return results, nil
}
