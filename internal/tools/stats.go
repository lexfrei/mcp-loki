package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/lexfrei/mcp-loki/internal/loki"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	bytesPerKB = 1024
	bytesPerMB = bytesPerKB * 1024
	bytesPerGB = bytesPerMB * 1024
)

// StatsParams defines the parameters for the loki_stats tool.
type StatsParams struct {
	Query string `json:"query"           jsonschema:"description=LogQL selector (e.g. {app=nginx}),required"`
	Start string `json:"start,omitempty" jsonschema:"description=Start time (RFC3339 or relative like 1h)"`
	End   string `json:"end,omitempty"   jsonschema:"description=End time (RFC3339 or now)"`
}

// StatsResult is the output of the loki_stats tool.
type StatsResult struct {
	Streams int64  `json:"streams"`
	Chunks  int64  `json:"chunks"`
	Bytes   int64  `json:"bytes"`
	Entries int64  `json:"entries"`
	Output  string `json:"output"`
}

// NewStatsHandler creates a handler for the loki_stats tool.
func NewStatsHandler(client *loki.Client) mcp.ToolHandlerFor[StatsParams, StatsResult] {
	return func(
		ctx context.Context,
		_ *mcp.CallToolRequest,
		params StatsParams,
	) (*mcp.CallToolResult, StatsResult, error) {
		if params.Query == "" {
			return &mcp.CallToolResult{IsError: true}, StatsResult{}, ErrQueryRequired
		}

		start, err := parseTimeOrDefault(params.Start, time.Now().Add(-time.Hour))
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, StatsResult{}, errors.Wrap(err, "invalid start time")
		}

		end, err := parseTimeOrDefault(params.End, time.Now())
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, StatsResult{}, errors.Wrap(err, "invalid end time")
		}

		resp, err := client.Stats(ctx, params.Query, start, end)
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, StatsResult{}, errors.Wrap(err, "stats request failed")
		}

		result := StatsResult{
			Streams: resp.Data.Streams,
			Chunks:  resp.Data.Chunks,
			Bytes:   resp.Data.Bytes,
			Entries: resp.Data.Entries,
			Output:  formatStatsResult(&resp.Data),
		}

		return nil, result, nil
	}
}

// StatsTool returns the MCP tool definition for loki_stats.
func StatsTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "loki_stats",
		Description: "Get index statistics from Loki for a given log selector",
	}
}

func formatStatsResult(stats *loki.StatsData) string {
	return fmt.Sprintf(
		"Index Statistics:\n  Streams: %d\n  Chunks: %d\n  Bytes: %s\n  Entries: %d",
		stats.Streams,
		stats.Chunks,
		formatBytes(stats.Bytes),
		stats.Entries,
	)
}

func formatBytes(bytes int64) string {
	switch {
	case bytes >= bytesPerGB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(bytesPerGB))
	case bytes >= bytesPerMB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(bytesPerMB))
	case bytes >= bytesPerKB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(bytesPerKB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
