package loki_test

import (
	"encoding/json"
	"testing"

	"github.com/lexfrei/mcp-loki/internal/loki"
)

const appNginx = "nginx"

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

	if resp.Data.ResultType != "streams" {
		t.Errorf("expected resultType streams, got %s", resp.Data.ResultType)
	}

	if len(resp.Data.Result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Data.Result))
	}

	stream := resp.Data.Result[0]
	if stream.Stream["app"] != appNginx {
		t.Errorf("expected stream app=nginx, got %s", stream.Stream["app"])
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

	if resp.Data.ResultType != "matrix" {
		t.Errorf("expected resultType matrix, got %s", resp.Data.ResultType)
	}
}

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

	expected := []string{"app", "env", "host", "level"}
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

	if resp.Data[0]["app"] != appNginx {
		t.Errorf("expected app=nginx, got %s", resp.Data[0]["app"])
	}

	if resp.Data[1]["env"] != "staging" {
		t.Errorf("expected env=staging, got %s", resp.Data[1]["env"])
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

	if resp.Status != "error" {
		t.Errorf("expected status error, got %s", resp.Status)
	}

	if resp.ErrorType != "bad_data" {
		t.Errorf("expected errorType bad_data, got %s", resp.ErrorType)
	}

	if resp.Error != "invalid query syntax" {
		t.Errorf("expected error 'invalid query syntax', got %s", resp.Error)
	}
}
