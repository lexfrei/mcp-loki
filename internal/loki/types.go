package loki

import (
	"encoding/json"
	"strings"
)

const (
	emptyLabels     = "{}"
	minValuesLength = 2
)

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

// StreamResult represents a single stream in the query result.
type StreamResult struct {
	Stream map[string]string   `json:"stream,omitempty"`
	Metric map[string]string   `json:"metric,omitempty"`
	Values [][]json.RawMessage `json:"values"`
}

// GetValues returns the values as string pairs (timestamp, value).
func (s *StreamResult) GetValues() [][2]string {
	result := make([][2]string, 0, len(s.Values))

	for _, entry := range s.Values {
		if len(entry) >= minValuesLength {
			var timestamp, value string

			// First element can be string (nanoseconds) or number (seconds)
			_ = json.Unmarshal(entry[0], &timestamp)

			// Second element is always string (log line or metric value)
			_ = json.Unmarshal(entry[1], &value)

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

// FormatQueryResult formats the query result for human-readable output.
func FormatQueryResult(resp *QueryResponse) string {
	if len(resp.Data.Result) == 0 {
		return "No results found."
	}

	var builder strings.Builder

	for _, stream := range resp.Data.Result {
		labels := stream.Stream
		if labels == nil {
			labels = stream.Metric
		}

		builder.WriteString("Stream: ")
		builder.WriteString(formatLabels(labels))
		builder.WriteString("\n")

		for _, value := range stream.GetValues() {
			builder.WriteString("  ")
			builder.WriteString(value[0])
			builder.WriteString(" | ")
			builder.WriteString(value[1])
			builder.WriteString("\n")
		}

		builder.WriteString("\n")
	}

	return builder.String()
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
