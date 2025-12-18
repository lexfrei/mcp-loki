package tools_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lexfrei/mcp-loki/internal/loki"
	"github.com/lexfrei/mcp-loki/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestStatsHandler_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/index/stats" {
			t.Errorf("expected path /loki/api/v1/index/stats, got %s", r.URL.Path)
		}

		query := r.URL.Query().Get("query")
		if query == "" {
			t.Error("expected query parameter")
		}

		resp := loki.StatsResponse{
			Status: "success",
			Data: loki.StatsData{
				Streams: 100,
				Chunks:  5000,
				Bytes:   1048576,
				Entries: 50000,
			},
		}
		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewStatsHandler(client)

	params := tools.StatsParams{
		Query: `{app="nginx"}`,
	}

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if result != nil && result.IsError {
		t.Error("expected success, got error")
	}

	if output.Streams != 100 {
		t.Errorf("expected 100 streams, got %d", output.Streams)
	}

	if output.Chunks != 5000 {
		t.Errorf("expected 5000 chunks, got %d", output.Chunks)
	}

	if output.Bytes != 1048576 {
		t.Errorf("expected 1048576 bytes, got %d", output.Bytes)
	}

	if output.Entries != 50000 {
		t.Errorf("expected 50000 entries, got %d", output.Entries)
	}
}

func TestStatsHandler_WithTimeRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")

		if start == "" || end == "" {
			t.Error("expected start and end parameters")
		}

		resp := loki.StatsResponse{
			Status: "success",
			Data:   loki.StatsData{},
		}
		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewStatsHandler(client)

	params := tools.StatsParams{
		Query: `{app="nginx"}`,
		Start: "24h",
		End:   "now",
	}

	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}
}

func TestStatsHandler_MissingQuery(t *testing.T) {
	client := loki.NewClient("http://localhost:3100", "", "", "", "")
	handler := tools.NewStatsHandler(client)

	params := tools.StatsParams{
		Query: "",
	}

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err == nil && (result == nil || !result.IsError) {
		t.Error("expected error for missing query")
	}
}
