package tools

import (
	"context"

	"github.com/lexfrei/mcp-loki/internal/loki"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ReadyParams defines the parameters for the loki_ready tool.
type ReadyParams struct{}

// ReadyResult is the output of the loki_ready tool.
type ReadyResult struct {
	Ready   bool   `json:"ready"`
	Message string `json:"message"`
}

// NewReadyHandler creates a handler for the loki_ready tool.
func NewReadyHandler(client *loki.Client) mcp.ToolHandlerFor[ReadyParams, ReadyResult] {
	return func(
		ctx context.Context,
		_ *mcp.CallToolRequest,
		_ ReadyParams,
	) (*mcp.CallToolResult, ReadyResult, error) {
		readyErr := client.Ready(ctx)

		// Always return a result, never an error
		// This allows LLMs to check readiness without error handling
		result := ReadyResult{
			Ready:   readyErr == nil,
			Message: "Loki is ready",
		}

		if readyErr != nil {
			result.Message = readyErr.Error()
		}

		return nil, result, nil
	}
}

// ReadyTool returns the MCP tool definition for loki_ready.
func ReadyTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "loki_ready",
		Description: "Check if Loki is ready to accept requests",
	}
}
