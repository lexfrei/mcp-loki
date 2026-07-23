package loki_test

import (
	"encoding/json"
	"testing"

	"github.com/lexfrei/mcp-loki/internal/loki"
)

const (
	appNginx = "nginx"

	resultTypeStreams = "streams"
	resultTypeMatrix  = "matrix"
	resultTypeVector  = "vector"
	timestampSample   = "2021-01-01T00:00:00Z"
	namespaceKube     = "kube-system"
	statusError       = "error"
	errorTypeBadData  = "bad_data"

	labelApp  = "app"
	labelEnv  = "env"
	labelHost = "host"
)

func TestQueryResponse_Unmarshal_Streams(t *testing.T) {
	raw := `{
		"status": "success",
		"data": {
			"resultType": "streams",
			"result": [
				{
					"stream": {"app": "nginx", "env": "prod"},
					"values": [
						["1609459200000000000", "log line 1"],
						["1609459201000000000", "log line 2"]
					]
				}
			]
		}
	}`

	var resp loki.QueryResponse
	err := json.Unmarshal([]byte(raw), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.Status != statusSuccess {
		t.Errorf("expected status success, got %s", resp.Status)
	}

	if resp.Data.ResultType != resultTypeStreams {
		t.Errorf("expected resultType streams, got %s", resp.Data.ResultType)
	}

	if len(resp.Data.Result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Data.Result))
	}

	stream := resp.Data.Result[0]
	if stream.Stream[labelApp] != appNginx {
		t.Errorf("expected stream app=nginx, got %s", stream.Stream[labelApp])
	}

	values := stream.GetValues()
	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(values))
	}

	if values[0][1] != "log line 1" {
		t.Errorf("expected 'log line 1', got %s", values[0][1])
	}
}

func TestQueryResponse_Unmarshal_Matrix(t *testing.T) {
	raw := `{
		"status": "success",
		"data": {
			"resultType": "matrix",
			"result": [
				{
					"metric": {"__name__": "log_count"},
					"values": [
						[1609459200, "100"],
						[1609459260, "150"]
					]
				}
			]
		}
	}`

	var resp loki.QueryResponse
	err := json.Unmarshal([]byte(raw), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.Data.ResultType != resultTypeMatrix {
		t.Errorf("expected resultType matrix, got %s", resp.Data.ResultType)
	}
}

func TestQueryResponse_Unmarshal_Vector(t *testing.T) {
	raw := vectorResponseFixture

	var resp loki.QueryResponse
	err := json.Unmarshal([]byte(raw), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.Data.ResultType != resultTypeVector {
		t.Errorf("expected resultType vector, got %s", resp.Data.ResultType)
	}

	if len(resp.Data.Result[0].Value) != 2 {
		t.Fatalf("expected vector value pair, got %d elements", len(resp.Data.Result[0].Value))
	}
}

func TestParseQueryResponse_Streams(t *testing.T) {
	raw := `{
		"status": "success",
		"data": {
			"resultType": "streams",
			"result": [
				{
					"stream": {"app": "nginx", "env": "prod"},
					"values": [
						["1609459200000000000", "\u001b[31merror\u001b[0m"]
					]
				}
			]
		}
	}`

	var resp loki.QueryResponse
	err := json.Unmarshal([]byte(raw), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	parsed := loki.ParseQueryResponse(&resp)
	if parsed.ResultType != resultTypeStreams {
		t.Fatalf("expected streams, got %s", parsed.ResultType)
	}

	if len(parsed.Streams) != 1 {
		t.Fatalf("expected 1 stream entry, got %d", len(parsed.Streams))
	}

	if parsed.Streams[0].Line != "error" {
		t.Errorf("expected ANSI stripped line, got %q", parsed.Streams[0].Line)
	}

	if parsed.Streams[0].Timestamp != timestampSample {
		t.Errorf("expected RFC3339Nano timestamp, got %s", parsed.Streams[0].Timestamp)
	}

	if parsed.Streams[0].Labels[labelApp] != appNginx {
		t.Errorf("expected app=nginx, got %s", parsed.Streams[0].Labels[labelApp])
	}
}

func TestParseQueryResponse_Matrix(t *testing.T) {
	raw := `{
		"status": "success",
		"data": {
			"resultType": "matrix",
			"result": [
				{
					"metric": {"namespace": "kube-system"},
					"values": [
						[1609459200, "100"],
						[1609459260, "150"]
					]
				}
			]
		}
	}`

	var resp loki.QueryResponse
	err := json.Unmarshal([]byte(raw), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	parsed := loki.ParseQueryResponse(&resp)
	if parsed.ResultType != resultTypeMatrix {
		t.Fatalf("expected matrix, got %s", parsed.ResultType)
	}

	if len(parsed.Series) != 1 {
		t.Fatalf("expected 1 series, got %d", len(parsed.Series))
	}

	if parsed.Series[0].Labels["namespace"] != namespaceKube {
		t.Errorf("expected namespace label, got %v", parsed.Series[0].Labels)
	}

	if parsed.Series[0].Values[0].Timestamp != timestampSample {
		t.Errorf("expected RFC3339Nano timestamp, got %s", parsed.Series[0].Values[0].Timestamp)
	}
}

func TestParseQueryResponse_Vector(t *testing.T) {
	raw := vectorResponseFixture

	var resp loki.QueryResponse
	err := json.Unmarshal([]byte(raw), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	parsed := loki.ParseQueryResponse(&resp)
	if parsed.ResultType != resultTypeVector {
		t.Fatalf("expected vector, got %s", parsed.ResultType)
	}

	if parsed.Series[0].Value == nil {
		t.Fatal("expected vector value")
	}

	if parsed.Series[0].Value.Value != "42" {
		t.Errorf("expected value 42, got %s", parsed.Series[0].Value.Value)
	}
}

const vectorResponseFixture = `{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": [
				{
					"metric": {"namespace": "kube-system"},
					"value": [1609459200, "42"]
				}
			]
		}
	}`

func TestLabelsResponse_Unmarshal(t *testing.T) {
	raw := `{
		"status": "success",
		"data": ["app", "env", "host", "level"]
	}`

	var resp loki.LabelsResponse
	err := json.Unmarshal([]byte(raw), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.Status != statusSuccess {
		t.Errorf("expected status success, got %s", resp.Status)
	}

	if len(resp.Data) != 4 {
		t.Fatalf("expected 4 labels, got %d", len(resp.Data))
	}

	expected := []string{labelApp, labelEnv, labelHost, "level"}
	for i, label := range expected {
		if resp.Data[i] != label {
			t.Errorf("expected label %s at index %d, got %s", label, i, resp.Data[i])
		}
	}
}

func TestSeriesResponse_Unmarshal(t *testing.T) {
	raw := `{
		"status": "success",
		"data": [
			{"app": "nginx", "env": "prod"},
			{"app": "nginx", "env": "staging"}
		]
	}`

	var resp loki.SeriesResponse
	err := json.Unmarshal([]byte(raw), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.Status != statusSuccess {
		t.Errorf("expected status success, got %s", resp.Status)
	}

	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 series, got %d", len(resp.Data))
	}

	if resp.Data[0][labelApp] != appNginx {
		t.Errorf("expected app=nginx, got %s", resp.Data[0][labelApp])
	}

	if resp.Data[1][labelEnv] != "staging" {
		t.Errorf("expected env=staging, got %s", resp.Data[1][labelEnv])
	}
}

func TestStatsResponse_Unmarshal(t *testing.T) {
	raw := `{
		"status": "success",
		"data": {
			"streams": 100,
			"chunks": 5000,
			"bytes": 1048576,
			"entries": 50000
		}
	}`

	var resp loki.StatsResponse
	err := json.Unmarshal([]byte(raw), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.Status != statusSuccess {
		t.Errorf("expected status success, got %s", resp.Status)
	}

	if resp.Data.Streams != 100 {
		t.Errorf("expected 100 streams, got %d", resp.Data.Streams)
	}

	if resp.Data.Chunks != 5000 {
		t.Errorf("expected 5000 chunks, got %d", resp.Data.Chunks)
	}

	if resp.Data.Bytes != 1048576 {
		t.Errorf("expected 1048576 bytes, got %d", resp.Data.Bytes)
	}

	if resp.Data.Entries != 50000 {
		t.Errorf("expected 50000 entries, got %d", resp.Data.Entries)
	}
}

func TestErrorResponse_Unmarshal(t *testing.T) {
	raw := `{
		"status": "error",
		"errorType": "bad_data",
		"error": "invalid query syntax"
	}`

	var resp loki.ErrorResponse
	err := json.Unmarshal([]byte(raw), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.Status != statusError {
		t.Errorf("expected status error, got %s", resp.Status)
	}

	if resp.ErrorType != errorTypeBadData {
		t.Errorf("expected errorType bad_data, got %s", resp.ErrorType)
	}

	if resp.Error != "invalid query syntax" {
		t.Errorf("expected error 'invalid query syntax', got %s", resp.Error)
	}
}
