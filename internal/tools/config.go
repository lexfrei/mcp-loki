package tools

import (
	"context"

	"github.com/cockroachdb/errors"

	"github.com/lexfrei/mcp-loki/internal/loki"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ConfigParams defines the parameters for the loki_config tool.
type ConfigParams struct{}

// ConfigResult is the output of the loki_config tool.
type ConfigResult struct {
	Config string `json:"config"`
}

// NewConfigHandler creates a handler for the loki_config tool.
func NewConfigHandler(client *loki.Client) mcp.ToolHandlerFor[ConfigParams, ConfigResult] {
	return func(
		ctx context.Context,
		_ *mcp.CallToolRequest,
		_ ConfigParams,
	) (*mcp.CallToolResult, ConfigResult, error) {
		config, err := client.Config(ctx)
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, ConfigResult{}, errors.Wrap(err, "failed to get config")
		}

		return nil, ConfigResult{
			Config: config,
		}, nil
	}
}

// ConfigTool returns the MCP tool definition for loki_config.
func ConfigTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "loki_config",
		Description: "Get Loki server configuration (YAML format)",
	}
}
