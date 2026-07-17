// Package tools provides MCP tool handlers for Loki operations.
package tools

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/lexfrei/mcp-loki/internal/loki"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultLimit       = 100
	defaultDirection   = "backward"
	defaultMetricStep  = "1m"
	hoursPerDay        = 24
	relativeTimeGroups = 3

	queryTypeRange   = "range"
	queryTypeInstant = "instant"
	queryTypeAuto    = "auto"
)

// ErrQueryRequired is returned when the query parameter is missing.
var ErrQueryRequired = errors.New("query parameter is required")

// ErrInvalidQueryType is returned when queryType is not supported.
var ErrInvalidQueryType = errors.New("invalid queryType")

// ErrEmptyTimeString is returned when an empty time string is provided.
var ErrEmptyTimeString = errors.New("empty time string")

// ErrInvalidTimeFormat is returned when a time string cannot be parsed.
var ErrInvalidTimeFormat = errors.New("invalid time format")

var metricQueryRE = regexp.MustCompile(`(?i)\b(?:sum|avg|min|max|count|stddev|stdvar|topk|bottomk|quantile|count_over_time|rate|bytes_rate|bytes_over_time|sum_over_time|avg_over_time|min_over_time|max_over_time|first_over_time|last_over_time|stdvar_over_time|stddev_over_time|quantile_over_time|absent_over_time|present_over_time|deriv|predict_linear|label_replace|label_join|vector|scalar)\b`)

// QueryParams defines the parameters for the loki_query tool.
type QueryParams struct {
	Query     string `json:"query"               jsonschema:"LogQL query string"`
	Start     string `json:"start,omitempty"     jsonschema:"Start time (RFC3339 or relative like 1h)"`
	End       string `json:"end,omitempty"       jsonschema:"End time (RFC3339 or now)"`
	Limit     int    `json:"limit,omitempty"     jsonschema:"Maximum entries to return (default 100)"`
	Direction string `json:"direction,omitempty" jsonschema:"Log order: forward or backward (default backward)"`
	Step      string `json:"step,omitempty"      jsonschema:"Resolution for metric range queries (e.g. 1m)"`
	QueryType string `json:"queryType,omitempty" jsonschema:"Query mode: range, instant, or auto (default auto)"`
}

// QueryResult is the output of the loki_query tool.
type QueryResult struct {
	ResultType string              `json:"resultType"`
	Count      int                 `json:"count"`
	Truncated  bool                `json:"truncated,omitempty"`
	Streams    []loki.LogEntry     `json:"streams,omitempty"`
	Series     []loki.MetricSeries `json:"series,omitempty"`
	Output     string              `json:"output,omitempty"`
}

// NewQueryHandler creates a handler for the loki_query tool.
func NewQueryHandler(client *loki.Client) mcp.ToolHandlerFor[QueryParams, QueryResult] {
	return func(
		ctx context.Context,
		_ *mcp.CallToolRequest,
		params QueryParams,
	) (*mcp.CallToolResult, QueryResult, error) {
		if params.Query == "" {
			return nil, QueryResult{}, validationErr(ErrQueryRequired)
		}

		queryType, err := normalizeQueryType(params.QueryType)
		if err != nil {
			return nil, QueryResult{}, validationErr(err)
		}

		limit := params.Limit
		if limit <= 0 {
			limit = defaultLimit
		}

		direction := params.Direction
		if direction == "" {
			direction = defaultDirection
		}

		resp, truncated, err := executeQuery(ctx, client, params, queryType, limit, direction)
		if err != nil {
			return nil, QueryResult{}, lokiErr("query failed", err)
		}

		parsed := loki.ParseQueryResponse(resp)
		result := QueryResult{
			ResultType: parsed.ResultType,
			Count:      parsed.Count,
			Truncated:  truncated,
			Streams:    parsed.Streams,
			Series:     parsed.Series,
			Output:     loki.FormatQueryResult(resp),
		}

		return nil, result, nil
	}
}

// QueryTool returns the MCP tool definition for loki_query.
func QueryTool() *mcp.Tool {
	return &mcp.Tool{
		Name: "loki_query",
		Description: "Execute a LogQL query against Loki to search and analyze logs. " +
			"Returns structured streams[] for log selectors and series[] for metric queries. " +
			"Use queryType=instant for single-point metric queries, queryType=range with step for time-series metrics, " +
			"or queryType=auto to route log selectors to range queries and metric expressions to instant/range as appropriate.",
	}
}

func executeQuery(
	ctx context.Context,
	client *loki.Client,
	params QueryParams, //nolint:gocritic // mirrors MCP handler signature
	queryType string,
	limit int,
	direction string,
) (*loki.QueryResponse, bool, error) {
	mode := resolveQueryMode(params.Query, queryType, params.Start, params.End)

	switch mode {
	case queryTypeInstant:
		evalTime, err := resolveInstantTime(params.End, params.Start)
		if err != nil {
			return nil, false, validationErr(errors.Wrap(err, "invalid instant time"))
		}

		resp, err := client.QueryInstant(ctx, params.Query, evalTime, limit)
		if err != nil {
			return nil, false, errors.Wrap(err, "instant query failed")
		}

		return resp, false, nil
	default:
		start, end, err := resolveRangeTimes(params.Start, params.End)
		if err != nil {
			return nil, false, validationErr(err)
		}

		step := params.Step
		if step == "" && isMetricQuery(params.Query) {
			step = defaultMetricStep
		}

		resp, err := client.QueryRange(ctx, params.Query, start, end, limit, direction, step)
		if err != nil {
			return nil, false, errors.Wrap(err, "range query failed")
		}

		truncated := resp.Data.ResultType == "streams" && parsedEntryCount(resp) >= limit

		return resp, truncated, nil
	}
}

func normalizeQueryType(queryType string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(queryType)) {
	case "", queryTypeAuto:
		return queryTypeAuto, nil
	case queryTypeRange, queryTypeInstant:
		return strings.ToLower(strings.TrimSpace(queryType)), nil
	default:
		return "", errors.Wrapf(ErrInvalidQueryType, "%q (use range, instant, or auto)", queryType)
	}
}

func resolveQueryMode(query, queryType, start, end string) string {
	if queryType == queryTypeRange {
		return queryTypeRange
	}

	if queryType == queryTypeInstant {
		return queryTypeInstant
	}

	if isMetricQuery(query) {
		if start == "" && end == "" {
			return queryTypeInstant
		}

		return queryTypeRange
	}

	return queryTypeRange
}

func isMetricQuery(query string) bool {
	return metricQueryRE.MatchString(query)
}

func resolveRangeTimes(startParam, endParam string) (time.Time, time.Time, error) {
	start, err := parseTimeOrDefault(startParam, time.Now().Add(-time.Hour))
	if err != nil {
		return time.Time{}, time.Time{}, errors.Wrap(err, "invalid start time")
	}

	end, err := parseTimeOrDefault(endParam, time.Now())
	if err != nil {
		return time.Time{}, time.Time{}, errors.Wrap(err, "invalid end time")
	}

	return start, end, nil
}

func resolveInstantTime(endParam, startParam string) (time.Time, error) {
	switch {
	case endParam != "":
		return ParseTime(endParam)
	case startParam != "":
		return ParseTime(startParam)
	default:
		return time.Now(), nil
	}
}

func parsedEntryCount(resp *loki.QueryResponse) int {
	return loki.ParseQueryResponse(resp).Count
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
