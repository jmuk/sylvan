package tools

import (
	"fmt"

	"github.com/invopop/jsonschema"
)

type Processor interface {
	Name() string
	Description() string
	RequestSchema() *jsonschema.Schema
	ResponseSchema() *jsonschema.Schema
	process(in map[string]any) (map[string]any, error)
}

var Defs = []Processor{
	createFileDef,
	readFileDef,
	modifyFileRef,
}

var defsMap = map[string]Processor{}

func init() {
	for _, d := range Defs {
		defsMap[d.Name()] = d
	}
}

func Process(name string, in map[string]any) (map[string]any, error) {
	p, ok := defsMap[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool %s", name)
	}
	return p.process(in)
}
