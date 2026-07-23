package loki

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	emptyLabels     = "{}"
	minValuesLength = 2

	resultTypeStreams = "streams"
	resultTypeMatrix  = "matrix"
	resultTypeVector  = "vector"
)

var ansiEscapeRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// QueryResponse represents the response from Loki query endpoints.
type QueryResponse struct {
	Status string    `json:"status"`
	Data   QueryData `json:"data"`
}

// QueryData contains the result of a Loki query.
type QueryData struct {
	ResultType string         `json:"resultType"`
	Result     []StreamResult `json:"result"`
}

// StreamResult represents a single stream or metric series in the query result.
type StreamResult struct {
	Stream map[string]string   `json:"stream,omitempty"`
	Metric map[string]string   `json:"metric,omitempty"`
	Values [][]json.RawMessage `json:"values,omitempty"`
	Value  []json.RawMessage   `json:"value,omitempty"`
}

// LogEntry is a single log line with labels and a normalized timestamp.
type LogEntry struct {
	Timestamp string            `json:"timestamp"`
	Labels    map[string]string `json:"labels"`
	Line      string            `json:"line"`
}

// MetricSeries is a labeled metric result from matrix or vector queries.
type MetricSeries struct {
	Labels map[string]string `json:"labels"`
	Values []MetricPoint     `json:"values,omitempty"`
	Value  *MetricPoint      `json:"value,omitempty"`
}

// MetricPoint is a single metric sample.
type MetricPoint struct {
	Timestamp string `json:"timestamp"`
	Value     string `json:"value"`
}

// ParsedQueryResult holds structured query output.
type ParsedQueryResult struct {
	ResultType string
	Streams    []LogEntry
	Series     []MetricSeries
	Count      int
}

// GetValues returns the values as string pairs (timestamp, value).
func (s *StreamResult) GetValues() [][2]string {
	result := make([][2]string, 0, len(s.Values))

	for _, entry := range s.Values {
		if len(entry) >= minValuesLength {
			timestamp := decodeTimestamp(entry[0])
			value := decodeString(entry[1])

			result = append(result, [2]string{timestamp, value})
		}
	}

	return result
}

// LabelsResponse represents the response from /loki/api/v1/labels endpoint.
type LabelsResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

// SeriesResponse represents the response from /loki/api/v1/series endpoint.
type SeriesResponse struct {
	Status string              `json:"status"`
	Data   []map[string]string `json:"data"`
}

// StatsResponse represents the response from /loki/api/v1/index/stats endpoint.
type StatsResponse struct {
	Status string    `json:"status"`
	Data   StatsData `json:"data"`
}

// StatsData contains index statistics.
type StatsData struct {
	Streams int64 `json:"streams"`
	Chunks  int64 `json:"chunks"`
	Bytes   int64 `json:"bytes"`
	Entries int64 `json:"entries"`
}

// ErrorResponse represents an error response from Loki.
type ErrorResponse struct {
	Status    string `json:"status"`
	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
}

// ParseQueryResponse converts a Loki query response into structured streams or series.
func ParseQueryResponse(resp *QueryResponse) ParsedQueryResult {
	if resp == nil {
		return ParsedQueryResult{}
	}

	if len(resp.Data.Result) == 0 {
		return ParsedQueryResult{ResultType: resp.Data.ResultType}
	}

	switch resp.Data.ResultType {
	case resultTypeStreams:
		return parseStreamResults(resp.Data.Result)
	case resultTypeMatrix:
		return parseMatrixResults(resp.Data.Result)
	case resultTypeVector:
		return parseVectorResults(resp.Data.Result)
	default:
		return ParsedQueryResult{ResultType: resp.Data.ResultType}
	}
}

func parseStreamResults(results []StreamResult) ParsedQueryResult {
	streams := make([]LogEntry, 0)

	for _, stream := range results {
		labels := streamLabels(stream)
		for _, value := range stream.GetValues() {
			streams = append(streams, LogEntry{
				Timestamp: formatTimestamp(value[0], true),
				Labels:    labels,
				Line:      stripANSI(value[1]),
			})
		}
	}

	return ParsedQueryResult{
		ResultType: resultTypeStreams,
		Streams:    streams,
		Count:      len(streams),
	}
}

func parseMatrixResults(results []StreamResult) ParsedQueryResult {
	series := make([]MetricSeries, 0, len(results))

	for _, result := range results {
		points := make([]MetricPoint, 0, len(result.Values))
		for _, entry := range result.Values {
			if len(entry) < minValuesLength {
				continue
			}

			points = append(points, MetricPoint{
				Timestamp: formatTimestamp(decodeTimestamp(entry[0]), false),
				Value:     decodeString(entry[1]),
			})
		}

		series = append(series, MetricSeries{
			Labels: metricLabels(result),
			Values: points,
		})
	}

	return ParsedQueryResult{
		ResultType: resultTypeMatrix,
		Series:     series,
		Count:      len(series),
	}
}

func parseVectorResults(results []StreamResult) ParsedQueryResult {
	series := make([]MetricSeries, 0, len(results))

	for _, result := range results {
		metricSeries := MetricSeries{Labels: metricLabels(result)}
		if len(result.Value) >= minValuesLength {
			metricSeries.Value = &MetricPoint{
				Timestamp: formatTimestamp(decodeTimestamp(result.Value[0]), false),
				Value:     decodeString(result.Value[1]),
			}
		}

		series = append(series, metricSeries)
	}

	return ParsedQueryResult{
		ResultType: resultTypeVector,
		Series:     series,
		Count:      len(series),
	}
}

// FormatQueryResult formats the query result for human-readable output.
func FormatQueryResult(resp *QueryResponse) string {
	if resp == nil || len(resp.Data.Result) == 0 {
		return "No results found."
	}

	var builder strings.Builder

	for _, stream := range resp.Data.Result {
		labels := streamLabels(stream)

		builder.WriteString("Stream: ")
		builder.WriteString(formatLabels(labels))
		builder.WriteString("\n")

		switch resp.Data.ResultType {
		case resultTypeVector:
			if len(stream.Value) >= minValuesLength {
				writeFormattedPoint(&builder, decodeTimestamp(stream.Value[0]), decodeString(stream.Value[1]))
			}
		default:
			for _, value := range stream.GetValues() {
				writeFormattedPoint(&builder, value[0], value[1])
			}
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

func writeFormattedPoint(builder *strings.Builder, timestamp, value string) {
	builder.WriteString("  ")
	builder.WriteString(timestamp)
	builder.WriteString(" | ")
	builder.WriteString(value)
	builder.WriteString("\n")
}

func streamLabels(stream StreamResult) map[string]string {
	if len(stream.Stream) > 0 {
		return stream.Stream
	}

	return map[string]string{}
}

func metricLabels(stream StreamResult) map[string]string {
	if len(stream.Metric) > 0 {
		return stream.Metric
	}

	return map[string]string{}
}

func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return emptyLabels
	}

	serialized, err := json.Marshal(labels)
	if err != nil {
		return emptyLabels
	}

	return string(serialized)
}

func decodeTimestamp(raw json.RawMessage) string {
	var asString string

	err := json.Unmarshal(raw, &asString)
	if err == nil {
		return asString
	}

	var asFloat float64

	err = json.Unmarshal(raw, &asFloat)
	if err == nil {
		return strconv.FormatInt(int64(asFloat), 10)
	}

	return ""
}

func decodeString(raw json.RawMessage) string {
	var value string

	_ = json.Unmarshal(raw, &value)

	return value
}

func formatTimestamp(raw string, nanoseconds bool) string {
	if raw == "" {
		return ""
	}

	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err == nil {
		return parsed.UTC().Format(time.RFC3339Nano)
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return raw
	}

	if nanoseconds {
		sec := value / int64(time.Second)
		nsec := value % int64(time.Second)

		return time.Unix(sec, nsec).UTC().Format(time.RFC3339Nano)
	}

	return time.Unix(value, 0).UTC().Format(time.RFC3339Nano)
}

func stripANSI(value string) string {
	return ansiEscapeRE.ReplaceAllString(value, "")
}
