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

func TestQueryHandler_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := loki.QueryResponse{
			Status: "success",
			Data: loki.QueryData{
				ResultType: "streams",
				Result: []loki.StreamResult{
					{
						Stream: map[string]string{"app": "test"},
						Values: [][]json.RawMessage{
							{json.RawMessage(`"1609459200000000000"`), json.RawMessage(`"test log line"`)},
						},
					},
				},
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
	handler := tools.NewQueryHandler(client)

	params := tools.QueryParams{
		Query: `{app="test"}`,
	}

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if result != nil && result.IsError {
		t.Errorf("expected success, got error")
	}
}

func TestQueryHandler_WithTimeRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")

		if start == "" || end == "" {
			t.Error("expected start and end parameters")
		}

		resp := loki.QueryResponse{
			Status: "success",
			Data:   loki.QueryData{ResultType: "streams", Result: []loki.StreamResult{}},
		}
		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewQueryHandler(client)

	params := tools.QueryParams{
		Query: `{app="test"}`,
		Start: "1h",
		End:   "now",
	}

	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}
}

func TestQueryHandler_RelativeTime(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"hours", "1h", true},
		{"minutes", "30m", true},
		{"days", "7d", true},
		{"now", "now", true},
		{"rfc3339", "2024-01-01T00:00:00Z", true},
		{"invalid", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tools.ParseTime(tt.input)
			if tt.valid && err != nil {
				t.Errorf("expected valid time, got error: %v", err)
			}

			if !tt.valid && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestQueryHandler_MissingQuery(t *testing.T) {
	client := loki.NewClient("http://localhost:3100", "", "", "", "")
	handler := tools.NewQueryHandler(client)

	params := tools.QueryParams{
		Query: "",
	}

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err == nil && (result == nil || !result.IsError) {
		t.Error("expected error for missing query")
	}
}
