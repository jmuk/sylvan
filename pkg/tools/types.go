package tools

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

type ToolDefinition[Req any, Resp any] struct {
	name         string
	description  string
	requestType  string
	responseType string
	proc         func(req Req) Resp
}

func (d ToolDefinition[Req, Resp]) Name() string {
	return d.name
}

func (d ToolDefinition[Req, Resp]) Description() string {
	return d.description
}

func (d *ToolDefinition[Req, Resp]) RequestSchema() *jsonschema.Schema {
	var t Req
	s := jsonschema.Reflect(&t)
	return s.Definitions[d.requestType]
}

func (d *ToolDefinition[Req, Resp]) ResponseSchema() *jsonschema.Schema {
	var t Resp
	s := jsonschema.Reflect(&t)
	return s.Definitions[d.responseType]
}

func (d *ToolDefinition[Req, Resp]) process(in map[string]any) (map[string]any, error) {
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
