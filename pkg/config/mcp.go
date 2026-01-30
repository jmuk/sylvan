package config

import (
	"fmt"
	"strings"
)

// MCPConfig defines a configuration to connect to a MCP server.
//
// It should specify either of the Command or Endpoint, not both.
// RequestHeaders is an optional field only used for Endpoint.
type MCPConfig struct {
	// The name of the MCP Server/command.
	Name string `toml:"name"`

	// Command is the command-line to invoke the MCP.
	Command []string `toml:"command,omitempty"`

	// The HTTP endpoint for the MCP server.
	Endpoint string `toml:"endpoint,omitempty"`
	// Additional HTTP request headers when a request is sent to the
	// endpoint.
	RequestHeaders map[string]string `toml:"request_headers,omitempty"`
}

// String implements Stringer interface.
func (c MCPConfig) String() string {
	if c.Endpoint != "" {
		return fmt.Sprintf("%s: %s", c.Name, c.Endpoint)
	}
	return fmt.Sprintf("%s: %s", c.Name, strings.Join(c.Command, " "))
}
