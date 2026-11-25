package config_test

import (
	"errors"
	"testing"
	"time"

	"github.com/lexfrei/mcp-loki/internal/config"
)

const (
	testUsername = "user"
	testPassword = "pass"

	wantDefaultSessionTimeout = 30 * time.Minute
	wantCustomSessionTimeout  = 5 * time.Minute
	wantCompoundTimeout       = 90 * time.Minute
	wantSubSecondTimeout      = 500 * time.Millisecond
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("LOKI_URL", "")
	t.Setenv("LOKI_USERNAME", "")
	t.Setenv("LOKI_PASSWORD", "")
	t.Setenv("LOKI_TOKEN", "")
	t.Setenv("LOKI_ORG_ID", "")
	t.Setenv("MCP_HTTP_PORT", "")
	t.Setenv("MCP_SESSION_TIMEOUT", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

	if cfg.SessionTimeout != wantDefaultSessionTimeout {
		t.Errorf("expected default SessionTimeout 30m, got %s", cfg.SessionTimeout)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("LOKI_URL", "http://loki.example.com:3100")
	t.Setenv("LOKI_USERNAME", "admin")
	t.Setenv("LOKI_PASSWORD", "secret")
	t.Setenv("LOKI_TOKEN", "bearer-token-123")
	t.Setenv("LOKI_ORG_ID", "tenant-1")
	t.Setenv("MCP_HTTP_PORT", "8080")
	t.Setenv("MCP_SESSION_TIMEOUT", "5m")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

	if cfg.SessionTimeout != wantCustomSessionTimeout {
		t.Errorf("expected SessionTimeout 5m, got %s", cfg.SessionTimeout)
	}
}

func TestLoad_SessionTimeout(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    time.Duration
		wantErr bool
	}{
		{"zero disables timeout", "0", 0, false},
		{"zero with unit", "0s", 0, false},
		{"compound duration", "1h30m", wantCompoundTimeout, false},
		{"sub-second", "500ms", wantSubSecondTimeout, false},
		{"negative", "-1m", 0, true},
		{"missing unit", "30", 0, true},
		{"not a duration", "forever", 0, true},
		{"whitespace", " 5m", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MCP_SESSION_TIMEOUT", tt.value)

			cfg, err := config.Load()

			if tt.wantErr {
				if !errors.Is(err, config.ErrInvalidSessionTimeout) {
					t.Fatalf("expected ErrInvalidSessionTimeout, got %v", err)
				}

				if cfg != nil {
					t.Errorf("expected nil config on error, got %+v", cfg)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.SessionTimeout != tt.want {
				t.Errorf("SessionTimeout = %s, want %s", cfg.SessionTimeout, tt.want)
			}
		})
	}
}

func TestConfig_HasBasicAuth(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		want     bool
	}{
		{"both set", testUsername, testPassword, true},
		{"only username", testUsername, "", false},
		{"only password", "", testPassword, false},
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
