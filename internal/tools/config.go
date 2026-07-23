package tools

import (
	"context"
	"strings"

	"github.com/cockroachdb/errors"
	"gopkg.in/yaml.v3"

	"github.com/lexfrei/mcp-loki/internal/loki"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ConfigParams defines the parameters for the loki_config tool.
type ConfigParams struct {
	SummaryOnly bool `json:"summary_only,omitempty" jsonschema:"Return a compact summary instead of full YAML"` //nolint:tagliatelle // MCP API uses snake_case
}

// ConfigResult is the output of the loki_config tool.
type ConfigResult struct {
	Config  string         `json:"config,omitempty"`
	Summary *ConfigSummary `json:"summary,omitempty"`
}

// ConfigSummary contains actionable Loki configuration fields.
type ConfigSummary struct {
	AuthEnabled bool            `json:"auth_enabled"` //nolint:tagliatelle // matches Loki config keys
	Ruler       ConfigRuler     `json:"ruler"`
	Retention   ConfigRetention `json:"retention"`
	Limits      ConfigLimits    `json:"limits"`
}

// ConfigRuler summarizes ruler configuration.
type ConfigRuler struct {
	Enabled bool `json:"enabled"`
}

// ConfigRetention summarizes retention-related settings.
type ConfigRetention struct {
	Enabled bool   `json:"enabled"`
	Period  string `json:"period,omitempty"`
}

// ConfigLimits summarizes ingestion and query limits.
type ConfigLimits struct {
	IngestionRateMB      float64 `json:"ingestion_rate_mb,omitempty"`           //nolint:tagliatelle // matches Loki config keys
	IngestionBurstSizeMB float64 `json:"ingestion_burst_size_mb,omitempty"`     //nolint:tagliatelle // matches Loki config keys
	MaxQueryLength       string  `json:"max_query_length,omitempty"`            //nolint:tagliatelle // matches Loki config keys
	MaxQueryParallelism  int     `json:"max_query_parallelism,omitempty"`       //nolint:tagliatelle // matches Loki config keys
	MaxEntriesLimit      int     `json:"max_entries_limit_per_query,omitempty"` //nolint:tagliatelle // matches Loki config keys
}

type rawLokiConfig struct {
	AuthEnabled *bool `yaml:"auth_enabled"` //nolint:tagliatelle // matches Loki config keys
	Ruler       struct {
		EnableAPI *bool `yaml:"enable_api"` //nolint:tagliatelle // matches Loki config keys
	} `yaml:"ruler"`
	LimitsConfig struct {
		RetentionPeriod         string  `yaml:"retention_period"`            //nolint:tagliatelle // matches Loki config keys
		RejectOldSamples        *bool   `yaml:"reject_old_samples"`          //nolint:tagliatelle // matches Loki config keys
		IngestionRateMB         float64 `yaml:"ingestion_rate_mb"`           //nolint:tagliatelle // matches Loki config keys
		IngestionBurstSizeMB    float64 `yaml:"ingestion_burst_size_mb"`     //nolint:tagliatelle // matches Loki config keys
		MaxQueryLength          string  `yaml:"max_query_length"`            //nolint:tagliatelle // matches Loki config keys
		MaxQueryParallelism     int     `yaml:"max_query_parallelism"`       //nolint:tagliatelle // matches Loki config keys
		MaxEntriesLimitPerQuery int     `yaml:"max_entries_limit_per_query"` //nolint:tagliatelle // matches Loki config keys
	} `yaml:"limits_config"` //nolint:tagliatelle // matches Loki config keys
	Compactor struct {
		RetentionEnabled *bool `yaml:"retention_enabled"` //nolint:tagliatelle // matches Loki config keys
	} `yaml:"compactor"`
}

// NewConfigHandler creates a handler for the loki_config tool.
func NewConfigHandler(client *loki.Client) mcp.ToolHandlerFor[ConfigParams, ConfigResult] {
	return func(
		ctx context.Context,
		_ *mcp.CallToolRequest,
		params ConfigParams,
	) (*mcp.CallToolResult, ConfigResult, error) {
		config, err := client.Config(ctx)
		if err != nil {
			return nil, ConfigResult{}, lokiErr("failed to get config", err)
		}

		if !params.SummaryOnly {
			return nil, ConfigResult{Config: config}, nil
		}

		summary, err := parseConfigSummary(config)
		if err != nil {
			return nil, ConfigResult{}, lokiErr("failed to parse config summary", err)
		}

		return nil, ConfigResult{Summary: summary}, nil
	}
}

// ConfigTool returns the MCP tool definition for loki_config.
func ConfigTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "loki_config",
		Description: "Get Loki server configuration. Use summary_only=true for a compact JSON summary of retention, limits, ruler, and auth settings.",
	}
}

func parseConfigSummary(configYAML string) (*ConfigSummary, error) {
	var raw rawLokiConfig

	err := yaml.Unmarshal([]byte(configYAML), &raw)
	if err != nil {
		return nil, errors.Wrap(err, "invalid config YAML")
	}

	summary := &ConfigSummary{
		AuthEnabled: boolValue(raw.AuthEnabled),
		Ruler: ConfigRuler{
			Enabled: boolValue(raw.Ruler.EnableAPI),
		},
		Retention: ConfigRetention{
			Enabled: boolValue(raw.Compactor.RetentionEnabled),
			Period:  strings.TrimSpace(raw.LimitsConfig.RetentionPeriod),
		},
		Limits: ConfigLimits{
			IngestionRateMB:      raw.LimitsConfig.IngestionRateMB,
			IngestionBurstSizeMB: raw.LimitsConfig.IngestionBurstSizeMB,
			MaxQueryLength:       raw.LimitsConfig.MaxQueryLength,
			MaxQueryParallelism:  raw.LimitsConfig.MaxQueryParallelism,
			MaxEntriesLimit:      raw.LimitsConfig.MaxEntriesLimitPerQuery,
		},
	}

	return summary, nil
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}

	return *value
}
