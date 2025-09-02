package tools

import (
	"context"
	"encoding/json"

	"github.com/invopop/jsonschema"
)

type ToolDefinition interface {
	Name() string
	Description() string
	RequestSchema() *jsonschema.Schema
	ResponseSchema() *jsonschema.Schema
	process(ctx context.Context, in map[string]any) (map[string]any, error)
}

type toolDefinition[Req any, Resp any] struct {
	name        string
	description string
	proc        func(ctx context.Context, req Req) Resp
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
	return (&jsonschema.Reflector{
		DoNotReference: true,
	}).Reflect(&t)
}

func (d *toolDefinition[Req, Resp]) process(ctx context.Context, in map[string]any) (map[string]any, error) {
	// Might not be ideal as it copies the data.
	logger := getLogger(ctx)
	jsonIn, err := json.Marshal(in)
	if err != nil {
		logger.Error("Failed to marshal input", "error", err)
		return nil, err
	}
	var req Req
	if err := json.Unmarshal(jsonIn, &req); err != nil {
		logger.Error("Failed to unmarshal input", "error", err)
		return nil, err
	}
	resp := d.proc(ctx, req)
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		logger.Error("Failed to marshal output", "error", err)
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(jsonResp, &out); err != nil {
		logger.Error("Failed to unmarshal output", "error", err)
		return nil, err
	}
	return out, nil
}
