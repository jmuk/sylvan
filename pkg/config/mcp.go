package config

// MCPConfig defines a configuration to connect to a MCP server.
type MCPConfig struct {
	Name           string            `toml:"name"`
	Command        []string          `toml:"command,omitempty"`
	Endpoint       string            `toml:"endpoint,omitempty"`
	RequestHeaders map[string]string `toml:"request_headers,omitempty"`
}
