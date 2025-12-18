package tools_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lexfrei/mcp-loki/internal/loki"
	"github.com/lexfrei/mcp-loki/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestReadyHandler_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
			t.Errorf("expected path /ready, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewReadyHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, tools.ReadyParams{})
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if result != nil && result.IsError {
		t.Error("expected success, got error")
	}

	if !output.Ready {
		t.Error("expected Ready=true")
	}
}

func TestReadyHandler_NotReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready"))
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewReadyHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, tools.ReadyParams{})

	// Should return error or IsError=true, but not panic
	if err == nil && (result == nil || !result.IsError) {
		if output.Ready {
			t.Error("expected Ready=false for unavailable server")
		}
	}
}

func TestReadyHandler_ConnectionError(t *testing.T) {
	// Non-existent server
	client := loki.NewClient("http://localhost:59999", "", "", "", "")
	handler := tools.NewReadyHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, tools.ReadyParams{})

	// Should handle connection error gracefully
	if err == nil && (result == nil || !result.IsError) {
		if output.Ready {
			t.Error("expected Ready=false for connection error")
		}
	}
}

func TestReadyTool_Definition(t *testing.T) {
	tool := tools.ReadyTool()

	if tool.Name != "loki_ready" {
		t.Errorf("expected name loki_ready, got %s", tool.Name)
	}

	if tool.Description == "" {
		t.Error("expected non-empty description")
	}
}
