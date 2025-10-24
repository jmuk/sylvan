package config

import (
	"fmt"
	"strings"
)

// MCPConfig defines a configuration to connect to a MCP server.
type MCPConfig struct {
	Name           string            `toml:"name"`
	Command        []string          `toml:"command,omitempty"`
	Endpoint       string            `toml:"endpoint,omitempty"`
	RequestHeaders map[string]string `toml:"request_headers,omitempty"`
}

func (c MCPConfig) String() string {
	if c.Endpoint != "" {
		return fmt.Sprintf("%s: %s", c.Name, c.Endpoint)
	}
	return fmt.Sprintf("%s: %s", c.Name, strings.Join(c.Command, " "))
}
