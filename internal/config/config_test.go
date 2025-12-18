package config_test

import (
	"testing"

	"github.com/lexfrei/mcp-loki/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("LOKI_URL", "")
	t.Setenv("LOKI_USERNAME", "")
	t.Setenv("LOKI_PASSWORD", "")
	t.Setenv("LOKI_TOKEN", "")
	t.Setenv("LOKI_ORG_ID", "")
	t.Setenv("MCP_HTTP_PORT", "")

	cfg := config.Load()

	if cfg.LokiURL != "http://localhost:3100" {
		t.Errorf("expected default LokiURL http://localhost:3100, got %s", cfg.LokiURL)
	}

	if cfg.Username != "" {
		t.Errorf("expected empty Username, got %s", cfg.Username)
	}

	if cfg.Password != "" {
		t.Errorf("expected empty Password, got %s", cfg.Password)
	}

	if cfg.Token != "" {
		t.Errorf("expected empty Token, got %s", cfg.Token)
	}

	if cfg.OrgID != "" {
		t.Errorf("expected empty OrgID, got %s", cfg.OrgID)
	}

	if cfg.HTTPPort != "" {
		t.Errorf("expected empty HTTPPort, got %s", cfg.HTTPPort)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("LOKI_URL", "http://loki.example.com:3100")
	t.Setenv("LOKI_USERNAME", "admin")
	t.Setenv("LOKI_PASSWORD", "secret")
	t.Setenv("LOKI_TOKEN", "bearer-token-123")
	t.Setenv("LOKI_ORG_ID", "tenant-1")
	t.Setenv("MCP_HTTP_PORT", "8080")

	cfg := config.Load()

	if cfg.LokiURL != "http://loki.example.com:3100" {
		t.Errorf("expected LokiURL http://loki.example.com:3100, got %s", cfg.LokiURL)
	}

	if cfg.Username != "admin" {
		t.Errorf("expected Username admin, got %s", cfg.Username)
	}

	if cfg.Password != "secret" {
		t.Errorf("expected Password secret, got %s", cfg.Password)
	}

	if cfg.Token != "bearer-token-123" {
		t.Errorf("expected Token bearer-token-123, got %s", cfg.Token)
	}

	if cfg.OrgID != "tenant-1" {
		t.Errorf("expected OrgID tenant-1, got %s", cfg.OrgID)
	}

	if cfg.HTTPPort != "8080" {
		t.Errorf("expected HTTPPort 8080, got %s", cfg.HTTPPort)
	}
}

func TestConfig_HasBasicAuth(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		want     bool
	}{
		{"both set", "user", "pass", true},
		{"only username", "user", "", false},
		{"only password", "", "pass", false},
		{"neither set", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Username: tt.username,
				Password: tt.password,
			}

			if got := cfg.HasBasicAuth(); got != tt.want {
				t.Errorf("HasBasicAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_HasBearerToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{"token set", "my-token", true},
		{"token empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Token: tt.token,
			}

			if got := cfg.HasBearerToken(); got != tt.want {
				t.Errorf("HasBearerToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_HTTPEnabled(t *testing.T) {
	tests := []struct {
		name     string
		httpPort string
		want     bool
	}{
		{"port set", "8080", true},
		{"port empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				HTTPPort: tt.httpPort,
			}

			if got := cfg.HTTPEnabled(); got != tt.want {
				t.Errorf("HTTPEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
