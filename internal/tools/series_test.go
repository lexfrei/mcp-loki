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

func TestSeriesHandler_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/series" {
			t.Errorf("expected path /loki/api/v1/series, got %s", r.URL.Path)
		}

		match := r.URL.Query()["match[]"]
		if len(match) == 0 {
			t.Error("expected match[] parameter")
		}

		resp := loki.SeriesResponse{
			Status: "success",
			Data: []map[string]string{
				{"app": "nginx", "env": "prod"},
				{"app": "nginx", "env": "staging"},
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
	handler := tools.NewSeriesHandler(client)

	params := tools.SeriesParams{
		Match: []string{`{app="nginx"}`},
	}

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if result != nil && result.IsError {
		t.Error("expected success, got error")
	}

	if output.Count != 2 {
		t.Errorf("expected 2 series, got %d", output.Count)
	}
}

func TestSeriesHandler_MultipleMatchers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		matches := r.URL.Query()["match[]"]
		if len(matches) != 2 {
			t.Errorf("expected 2 match[] parameters, got %d", len(matches))
		}

		resp := loki.SeriesResponse{Status: "success", Data: []map[string]string{}}
		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewSeriesHandler(client)

	params := tools.SeriesParams{
		Match: []string{`{app="nginx"}`, `{app="redis"}`},
	}

	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}
}

func TestSeriesHandler_MissingMatch(t *testing.T) {
	client := loki.NewClient("http://localhost:3100", "", "", "", "")
	handler := tools.NewSeriesHandler(client)

	params := tools.SeriesParams{
		Match: []string{},
	}

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err == nil && (result == nil || !result.IsError) {
		t.Error("expected error for missing match")
	}
}
