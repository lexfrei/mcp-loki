// Package tools provides MCP tool handlers for Loki operations.
package tools

import (
	"context"
	"regexp"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/lexfrei/mcp-loki/internal/loki"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultLimit       = 100
	defaultDirection   = "backward"
	hoursPerDay        = 24
	relativeTimeGroups = 3
)

// ErrQueryRequired is returned when the query parameter is missing.
var ErrQueryRequired = errors.New("query parameter is required")

// ErrEmptyTimeString is returned when an empty time string is provided.
var ErrEmptyTimeString = errors.New("empty time string")

// ErrInvalidTimeFormat is returned when a time string cannot be parsed.
var ErrInvalidTimeFormat = errors.New("invalid time format")

// QueryParams defines the parameters for the loki_query tool.
type QueryParams struct {
	Query     string `json:"query"               jsonschema:"description=LogQL query string,required"`
	Start     string `json:"start,omitempty"     jsonschema:"description=Start time (RFC3339 or relative like 1h)"`
	End       string `json:"end,omitempty"       jsonschema:"description=End time (RFC3339 or now)"`
	Limit     int    `json:"limit,omitempty"     jsonschema:"description=Maximum entries to return (default 100)"`
	Direction string `json:"direction,omitempty" jsonschema:"description=Log order: forward or backward (default backward)"`
}

// QueryResult is the output of the loki_query tool.
type QueryResult struct {
	ResultType string `json:"resultType"`
	Count      int    `json:"count"`
	Output     string `json:"output"`
}

// NewQueryHandler creates a handler for the loki_query tool.
func NewQueryHandler(client *loki.Client) mcp.ToolHandlerFor[QueryParams, QueryResult] {
	return func(
		ctx context.Context,
		_ *mcp.CallToolRequest,
		params QueryParams,
	) (*mcp.CallToolResult, QueryResult, error) {
		if params.Query == "" {
			return &mcp.CallToolResult{IsError: true}, QueryResult{}, ErrQueryRequired
		}

		start, err := parseTimeOrDefault(params.Start, time.Now().Add(-time.Hour))
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, QueryResult{}, errors.Wrap(err, "invalid start time")
		}

		end, err := parseTimeOrDefault(params.End, time.Now())
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, QueryResult{}, errors.Wrap(err, "invalid end time")
		}

		limit := params.Limit
		if limit <= 0 {
			limit = defaultLimit
		}

		direction := params.Direction
		if direction == "" {
			direction = defaultDirection
		}

		resp, err := client.QueryRange(ctx, params.Query, start, end, limit, direction)
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, QueryResult{}, errors.Wrap(err, "query failed")
		}

		output := loki.FormatQueryResult(resp)
		result := QueryResult{
			ResultType: resp.Data.ResultType,
			Count:      len(resp.Data.Result),
			Output:     output,
		}

		return nil, result, nil
	}
}

// QueryTool returns the MCP tool definition for loki_query.
func QueryTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "loki_query",
		Description: "Execute a LogQL query against Loki to search and analyze logs",
	}
}

// ParseTime parses a time string that can be RFC3339, "now", or relative (1h, 30m, 7d).
func ParseTime(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, ErrEmptyTimeString
	}

	if timeStr == "now" {
		return time.Now(), nil
	}

	// Try RFC3339 first
	parsedTime, parseErr := time.Parse(time.RFC3339, timeStr)
	if parseErr == nil {
		return parsedTime, nil
	}

	// Try relative time (e.g., "1h", "30m", "7d")
	re := regexp.MustCompile(`^(\d+)([smhd])$`)
	matches := re.FindStringSubmatch(timeStr)

	if len(matches) == relativeTimeGroups {
		value, _ := strconv.Atoi(matches[1])
		unit := matches[2]

		var duration time.Duration

		switch unit {
		case "s":
			duration = time.Duration(value) * time.Second
		case "m":
			duration = time.Duration(value) * time.Minute
		case "h":
			duration = time.Duration(value) * time.Hour
		case "d":
			duration = time.Duration(value) * hoursPerDay * time.Hour
		}

		return time.Now().Add(-duration), nil
	}

	return time.Time{}, errors.Wrapf(ErrInvalidTimeFormat, "%s (use RFC3339, 'now', or relative like 1h, 30m, 7d)", timeStr)
}

func parseTimeOrDefault(timeStr string, defaultTime time.Time) (time.Time, error) {
	if timeStr == "" {
		return defaultTime, nil
	}

	return ParseTime(timeStr)
}
