package tools

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

type ToolDefinition interface {
	Name() string
	Description() string
	RequestSchema() *jsonschema.Schema
	ResponseSchema() *jsonschema.Schema
	process(in map[string]any) (map[string]any, error)
}

type toolDefinition[Req any, Resp any] struct {
	name        string
	description string
	proc        func(req Req) Resp
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

func (d *toolDefinition[Req, Resp]) process(in map[string]any) (map[string]any, error) {
	// Might not be ideal as it copies the data.
	jsonIn, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var req Req
	if err := json.Unmarshal(jsonIn, &req); err != nil {
		return nil, err
	}
	resp := d.proc(req)
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(jsonResp, &out); err != nil {
		return nil, err
	}
	return out, nil
}
