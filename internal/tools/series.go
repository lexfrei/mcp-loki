package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/lexfrei/mcp-loki/internal/loki"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ErrMatchRequired is returned when the match parameter is missing.
var ErrMatchRequired = errors.New("match parameter is required")

// SeriesParams defines the parameters for the loki_series tool.
type SeriesParams struct {
	Match []string `json:"match"           jsonschema:"Series selectors (e.g. {app=nginx})"`
	Start string   `json:"start,omitempty" jsonschema:"Start time (RFC3339 or relative like 1h)"`
	End   string   `json:"end,omitempty"   jsonschema:"End time (RFC3339 or now)"`
}

// SeriesResult is the output of the loki_series tool.
type SeriesResult struct {
	Count  int                 `json:"count"`
	Series []map[string]string `json:"series"`
	Output string              `json:"output"`
}

// NewSeriesHandler creates a handler for the loki_series tool.
func NewSeriesHandler(client *loki.Client) mcp.ToolHandlerFor[SeriesParams, SeriesResult] {
	return func(
		ctx context.Context,
		_ *mcp.CallToolRequest,
		params SeriesParams,
	) (*mcp.CallToolResult, SeriesResult, error) {
		if len(params.Match) == 0 {
			return &mcp.CallToolResult{IsError: true}, SeriesResult{}, ErrMatchRequired
		}

		start, err := parseTimeOrDefault(params.Start, time.Now().Add(-time.Hour))
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, SeriesResult{}, errors.Wrap(err, "invalid start time")
		}

		end, err := parseTimeOrDefault(params.End, time.Now())
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, SeriesResult{}, errors.Wrap(err, "invalid end time")
		}

		resp, err := client.Series(ctx, params.Match, start, end)
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, SeriesResult{}, errors.Wrap(err, "series request failed")
		}

		result := SeriesResult{
			Count:  len(resp.Data),
			Series: resp.Data,
			Output: formatSeriesResult(resp.Data),
		}

		return nil, result, nil
	}
}

// SeriesTool returns the MCP tool definition for loki_series.
func SeriesTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "loki_series",
		Description: "Get log streams (series) from Loki that match the given label selectors",
	}
}

func formatSeriesResult(series []map[string]string) string {
	if len(series) == 0 {
		return "No series found."
	}

	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Found %d series:\n", len(series)))

	for idx, seriesItem := range series {
		serialized, _ := json.Marshal(seriesItem)
		builder.WriteString(fmt.Sprintf("  %d. %s\n", idx+1, string(serialized)))
	}

	return builder.String()
}
