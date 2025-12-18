// Package config provides configuration loading from environment variables.
package config

import "os"

// Config holds the application configuration loaded from environment variables.
type Config struct {
	LokiURL  string
	Username string
	Password string
	Token    string
	OrgID    string
	HTTPPort string
}

// Load reads configuration from environment variables and returns a Config.
func Load() *Config {
	lokiURL := os.Getenv("LOKI_URL")
	if lokiURL == "" {
		lokiURL = "http://localhost:3100"
	}

	return &Config{
		LokiURL:  lokiURL,
		Username: os.Getenv("LOKI_USERNAME"),
		Password: os.Getenv("LOKI_PASSWORD"),
		Token:    os.Getenv("LOKI_TOKEN"),
		OrgID:    os.Getenv("LOKI_ORG_ID"),
		HTTPPort: os.Getenv("MCP_HTTP_PORT"),
	}
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
