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

func TestLabelsHandler_GetAllLabels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/labels" {
			t.Errorf("expected path /loki/api/v1/labels, got %s", r.URL.Path)
		}

		resp := loki.LabelsResponse{
			Status: "success",
			Data:   []string{"app", "env", "host", "level"},
		}
		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewLabelsHandler(client)

	params := tools.LabelsParams{}

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if result != nil && result.IsError {
		t.Error("expected success, got error")
	}

	if output.Count != 4 {
		t.Errorf("expected 4 labels, got %d", output.Count)
	}
}

func TestLabelsHandler_GetLabelValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/label/app/values" {
			t.Errorf("expected path /loki/api/v1/label/app/values, got %s", r.URL.Path)
		}

		resp := loki.LabelsResponse{
			Status: "success",
			Data:   []string{"nginx", "redis", "postgres"},
		}
		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewLabelsHandler(client)

	params := tools.LabelsParams{
		Name: "app",
	}

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if result != nil && result.IsError {
		t.Error("expected success, got error")
	}

	if output.Count != 3 {
		t.Errorf("expected 3 values, got %d", output.Count)
	}
}

func TestLabelsHandler_WithTimeRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")

		if start == "" || end == "" {
			t.Error("expected start and end parameters")
		}

		resp := loki.LabelsResponse{Status: "success", Data: []string{}}
		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewLabelsHandler(client)

	params := tools.LabelsParams{
		Start: "1h",
		End:   "now",
	}

	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}
}
