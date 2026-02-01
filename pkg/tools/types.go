package tools

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/invopop/jsonschema"
	"github.com/jmuk/sylvan/pkg/chat/parts"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// ToolError is a type of error that should be reported to agent.
type ToolError struct {
	err error
}

// Error implements error type.
func (e *ToolError) Error() string {
	return e.err.Error()
}

// Unwrap implements unwrapping.
func (e *ToolError) Unwrap() error {
	return e.err
}

// ToolDefinition is a definition of a tool.
type ToolDefinition interface {
	// The name of the tool.
	Name() string
	// The description of the tool.
	Description() string
	// Schema of the request object.
	RequestSchema() *jsonschema.Schema
	// Schema of the response object.
	ResponseSchema() *jsonschema.Schema
	process(ctx context.Context, in map[string]any) (any, []*parts.Part, error)
}

type toolDefinition[Req any, Resp any] struct {
	name        string
	description string
	proc        func(ctx context.Context, req Req) (Resp, error)

	respName        string
	respDescription string
}

func (d toolDefinition[Req, Resp]) Name() string {
	return d.name
}

func (d toolDefinition[Req, Resp]) Description() string {
	return d.description
}

func (d *toolDefinition[Req, Resp]) RequestSchema() *jsonschema.Schema {
	var t Req
	return (&jsonschema.Reflector{
		DoNotReference: true,
	}).Reflect(&t)
}

func (d *toolDefinition[Req, Resp]) ResponseSchema() *jsonschema.Schema {
	var t Resp
	var schema *jsonschema.Schema
	if reflect.TypeFor[Resp]().ConvertibleTo(reflect.TypeFor[string]()) {
		schema = &jsonschema.Schema{
			Version: jsonschema.Version,
			Type:    "object",
			Properties: orderedmap.New[string, *jsonschema.Schema](
				orderedmap.WithInitialData[string, *jsonschema.Schema](
					orderedmap.Pair[string, *jsonschema.Schema]{
						Key: d.respName,
						Value: &jsonschema.Schema{
							Type:        "string",
							Description: d.respDescription,
						},
					},
				)),
		}
	} else {
		schema = (&jsonschema.Reflector{
			DoNotReference: true,
		}).Reflect(&t)
	}
	schema.Properties.AddPairs(orderedmap.Pair[string, *jsonschema.Schema]{
		Key: "error",
		Value: &jsonschema.Schema{
			Type:        "string",
			Description: "the error message when the command failed, or empty",
		},
	})
	return schema
}

func (d *toolDefinition[Req, Resp]) process(ctx context.Context, in map[string]any) (any, []*parts.Part, error) {
	// Might not be ideal as it copies the data.
	logger := getLogger(ctx)
	jsonIn, err := json.Marshal(in)
	if err != nil {
		logger.Error("Failed to marshal input", "error", err)
		return nil, nil, err
	}
	var req Req
	if err := json.Unmarshal(jsonIn, &req); err != nil {
		logger.Error("Failed to unmarshal input", "error", err)
		return nil, nil, err
	}
	resp, err := d.proc(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	if v := reflect.ValueOf(resp); v.CanConvert(reflect.TypeFor[string]()) {
		return map[string]any{
			d.respName: v.Convert(reflect.TypeFor[string]()).String(),
		}, nil, nil
	}
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		logger.Error("Failed to marshal output", "error", err)
		return nil, nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(jsonResp, &out); err != nil {
		logger.Error("Failed to unmarshal output", "error", err)
		return nil, nil, err
	}
	return out, nil, nil
}
