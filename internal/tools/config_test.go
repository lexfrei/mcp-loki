package tools_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/lexfrei/mcp-loki/internal/loki"
	"github.com/lexfrei/mcp-loki/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestConfigHandler_SummaryOnly(t *testing.T) {
	configYAML := `auth_enabled: true
ruler:
  enable_api: true
limits_config:
  retention_period: 168h
  ingestion_rate_mb: 4
  ingestion_burst_size_mb: 6
  max_query_length: 721h
  max_query_parallelism: 32
  max_entries_limit_per_query: 5000
compactor:
  retention_enabled: true`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(configYAML))
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewConfigHandler(client)

	_, output, err := handler(context.Background(), &mcp.CallToolRequest{}, tools.ConfigParams{SummaryOnly: true})
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if output.Config != "" {
		t.Error("expected empty full config in summary mode")
	}

	if output.Summary == nil {
		t.Fatal("expected summary")
	}

	if !output.Summary.AuthEnabled {
		t.Error("expected auth_enabled true")
	}

	if !output.Summary.Ruler.Enabled {
		t.Error("expected ruler enabled")
	}

	if output.Summary.Retention.Period != "168h" {
		t.Errorf("expected retention period 168h, got %s", output.Summary.Retention.Period)
	}

	if output.Summary.Limits.MaxQueryLength != "721h" {
		t.Errorf("expected max query length 721h, got %s", output.Summary.Limits.MaxQueryLength)
	}

	if output.Summary.Limits.MaxEntriesLimit != 5000 {
		t.Errorf("expected max entries limit 5000, got %d", output.Summary.Limits.MaxEntriesLimit)
	}
}

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

	if err == nil {
		t.Error("expected error for failed config request")
	}

	if result != nil {
		t.Error("expected nil CallToolResult on error path")
	}

	if !errors.Is(err, tools.ErrLokiRequest) {
		t.Errorf("expected ErrLokiRequest, got: %v", err)
	}
}

func TestConfigHandler_ConnectionError(t *testing.T) {
	// Non-existent server
	client := loki.NewClient("http://localhost:59999", "", "", "", "")
	handler := tools.NewConfigHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, tools.ConfigParams{})

	if err == nil {
		t.Error("expected error for connection error")
	}

	if result != nil {
		t.Error("expected nil CallToolResult on error path")
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
