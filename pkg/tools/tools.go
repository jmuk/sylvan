package tools

import (
	"fmt"
)

// var Defs = []Processor{
// 	createFileDef,
// 	readFileDef,
// 	modifyFileRef,
// 	searchFilesDef,
// 	execCommandDef,
// }

type ToolRunner struct {
	defsMap map[string]ToolDefinition
}

func NewToolRunner(defs []ToolDefinition) (*ToolRunner, error) {
	m := make(map[string]ToolDefinition, len(defs))
	for _, d := range defs {
		if _, ok := m[d.Name()]; ok {
			return nil, fmt.Errorf("duplicated tool name %s", d.Name())
		}
		m[d.Name()] = d
	}
	return &ToolRunner{defsMap: m}, nil
}

func (r *ToolRunner) Run(name string, in map[string]any) (map[string]any, error) {
	p, ok := r.defsMap[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool %s", name)
	}
	return p.process(in)
}
