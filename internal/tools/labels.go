package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/lexfrei/mcp-loki/internal/loki"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	resultTypeLabelNames  = "label_names"
	resultTypeLabelValues = "label_values"
)

// LabelsParams defines the parameters for the loki_labels tool.
type LabelsParams struct {
	Name  string `json:"name,omitempty"  jsonschema:"Label name to get values for. If omitted returns all label names"`
	Start string `json:"start,omitempty" jsonschema:"Start time (RFC3339 or relative like 1h)"`
	End   string `json:"end,omitempty"   jsonschema:"End time (RFC3339 or now)"`
}

// LabelsResult is the output of the loki_labels tool.
type LabelsResult struct {
	Type   string   `json:"type"`
	Count  int      `json:"count"`
	Labels []string `json:"labels"`
}

// NewLabelsHandler creates a handler for the loki_labels tool.
func NewLabelsHandler(client *loki.Client) mcp.ToolHandlerFor[LabelsParams, LabelsResult] {
	return func(
		ctx context.Context,
		_ *mcp.CallToolRequest,
		params LabelsParams,
	) (*mcp.CallToolResult, LabelsResult, error) {
		start, err := parseTimeOrDefault(params.Start, time.Now().Add(-time.Hour))
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, LabelsResult{}, errors.Wrap(err, "invalid start time")
		}

		end, err := parseTimeOrDefault(params.End, time.Now())
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, LabelsResult{}, errors.Wrap(err, "invalid end time")
		}

		var resp *loki.LabelsResponse

		var resultType string

		if params.Name == "" {
			resp, err = client.Labels(ctx, start, end)
			resultType = resultTypeLabelNames
		} else {
			resp, err = client.LabelValues(ctx, params.Name, start, end)
			resultType = resultTypeLabelValues
		}

		if err != nil {
			return &mcp.CallToolResult{IsError: true}, LabelsResult{}, errors.Wrap(err, "labels request failed")
		}

		result := LabelsResult{
			Type:   resultType,
			Count:  len(resp.Data),
			Labels: resp.Data,
		}

		return nil, result, nil
	}
}

// LabelsTool returns the MCP tool definition for loki_labels.
func LabelsTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "loki_labels",
		Description: "Get label names or values from Loki. Without 'name' parameter returns all label names; with 'name' returns values for that label",
	}
}

// FormatLabelsResult formats the labels result for human-readable output.
func FormatLabelsResult(result *LabelsResult) string {
	if result.Count == 0 {
		return "No labels found."
	}

	var builder strings.Builder

	if result.Type == resultTypeLabelNames {
		builder.WriteString(fmt.Sprintf("Found %d label names:\n", result.Count))
	} else {
		builder.WriteString(fmt.Sprintf("Found %d values:\n", result.Count))
	}

	for _, label := range result.Labels {
		builder.WriteString("  - ")
		builder.WriteString(label)
		builder.WriteString("\n")
	}

	return builder.String()
}
