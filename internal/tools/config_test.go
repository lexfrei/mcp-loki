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

func TestConfigHandler_Success(t *testing.T) {
	configYAML := `server:
  http_listen_port: 3100
ingester:
  lifecycler:
    ring:
      replication_factor: 1`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/config" {
			t.Errorf("expected path /config, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "text/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(configYAML))
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewConfigHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, tools.ConfigParams{})
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if result != nil && result.IsError {
		t.Error("expected success, got error")
	}

	if output.Config == "" {
		t.Error("expected non-empty config")
	}

	if output.Config != configYAML {
		t.Errorf("config mismatch: got %s", output.Config)
	}
}

func TestConfigHandler_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewConfigHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, tools.ConfigParams{})

	// Should return error
	if err == nil && (result == nil || !result.IsError) {
		t.Error("expected error for failed config request")
	}
}

func TestConfigHandler_ConnectionError(t *testing.T) {
	// Non-existent server
	client := loki.NewClient("http://localhost:59999", "", "", "", "")
	handler := tools.NewConfigHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, tools.ConfigParams{})

	// Should handle connection error gracefully
	if err == nil && (result == nil || !result.IsError) {
		t.Error("expected error for connection error")
	}
}

func TestConfigTool_Definition(t *testing.T) {
	tool := tools.ConfigTool()

	if tool.Name != "loki_config" {
		t.Errorf("expected name loki_config, got %s", tool.Name)
	}

	if tool.Description == "" {
		t.Error("expected non-empty description")
	}
}
