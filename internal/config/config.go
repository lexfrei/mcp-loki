// Package config provides configuration loading from environment variables.
package config

import (
	"os"
	"time"

	"github.com/cockroachdb/errors"
)

// defaultSessionTimeout bounds how long an idle HTTP session is kept. Without
// it, clients that never send DELETE leak one session each until the process
// runs out of memory.
const defaultSessionTimeout = 30 * time.Minute

// ErrInvalidSessionTimeout is returned when MCP_SESSION_TIMEOUT is not a
// non-negative Go duration.
var ErrInvalidSessionTimeout = errors.New("invalid MCP_SESSION_TIMEOUT")

// Config holds the application configuration loaded from environment variables.
type Config struct {
	LokiURL        string
	Username       string
	Password       string
	Token          string
	OrgID          string
	HTTPPort       string
	SessionTimeout time.Duration
}

// Load reads configuration from environment variables and returns a Config.
func Load() (*Config, error) {
	lokiURL := os.Getenv("LOKI_URL")
	if lokiURL == "" {
		lokiURL = "http://localhost:3100"
	}

	sessionTimeout, err := parseSessionTimeout(os.Getenv("MCP_SESSION_TIMEOUT"))
	if err != nil {
		return nil, err
	}

	return &Config{
		LokiURL:        lokiURL,
		Username:       os.Getenv("LOKI_USERNAME"),
		Password:       os.Getenv("LOKI_PASSWORD"),
		Token:          os.Getenv("LOKI_TOKEN"),
		OrgID:          os.Getenv("LOKI_ORG_ID"),
		HTTPPort:       os.Getenv("MCP_HTTP_PORT"),
		SessionTimeout: sessionTimeout,
	}, nil
}

func parseSessionTimeout(raw string) (time.Duration, error) {
	if raw == "" {
		return defaultSessionTimeout, nil
	}

	timeout, err := time.ParseDuration(raw)
	if err != nil {
		return 0, errors.Wrapf(ErrInvalidSessionTimeout, "%q: %v", raw, err)
	}

	if timeout < 0 {
		return 0, errors.Wrapf(ErrInvalidSessionTimeout, "%q: must not be negative", raw)
	}

	return timeout, nil
}

// HasBasicAuth returns true if both username and password are set.
func (c *Config) HasBasicAuth() bool {
	return c.Username != "" && c.Password != ""
}

// HasBearerToken returns true if a bearer token is set.
func (c *Config) HasBearerToken() bool {
	return c.Token != ""
}

// HTTPEnabled returns true if HTTP transport should be enabled.
func (c *Config) HTTPEnabled() bool {
	return c.HTTPPort != ""
}
