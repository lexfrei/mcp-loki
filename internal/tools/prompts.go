package tools

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/cockroachdb/errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultInterval = "5m"

var (
	relativeDurationRe = regexp.MustCompile(`^\d+[smhdw]$`)
	selectorRe         = regexp.MustCompile(`^\{.+\}$`)
	labelNameRe        = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// validateRelativeDuration checks that a string is a valid relative duration (e.g. 1h, 30m, 7d, 1w).
func validateRelativeDuration(value string) error {
	if !relativeDurationRe.MatchString(value) {
		return validationErr(errors.Newf("invalid duration %q: use relative format like 1h, 30m, 7d, 1w", value))
	}

	return nil
}

// validateSelector checks that a string looks like a LogQL stream selector ({...}).
func validateSelector(value string) error {
	if !selectorRe.MatchString(value) {
		return validationErr(errors.Newf("invalid selector %q: must be a LogQL stream selector like {app=\"nginx\"}", value))
	}

	return nil
}

// validateLabelName checks that a string is a valid Prometheus/Loki label name.
func validateLabelName(value string) error {
	if !labelNameRe.MatchString(value) {
		return validationErr(errors.Newf("invalid label name %q: must match [a-zA-Z_][a-zA-Z0-9_]*", value))
	}

	return nil
}

// ErrorLogsPrompt returns the MCP prompt definition for error_logs.
func ErrorLogsPrompt() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        "error_logs",
		Description: "Find error logs for a specific application",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "app",
				Description: "Application name to search errors for",
				Required:    true,
			},
			{
				Name:        "timerange",
				Description: "Time range to search (e.g. 1h, 30m, 7d). Default: 1h",
				Required:    false,
			},
		},
	}
}

// ErrorLogsHandler returns the handler for the error_logs prompt.
func ErrorLogsHandler() mcp.PromptHandler {
	return func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		app := req.Params.Arguments["app"]
		if app == "" {
			return nil, validationErr(errors.New("app argument is required"))
		}

		timerange := req.Params.Arguments["timerange"]
		if timerange == "" {
			timerange = "1h"
		} else {
			err := validateRelativeDuration(timerange)
			if err != nil {
				return nil, err
			}
		}

		query := fmt.Sprintf(`{app=%q} |= "error"`, app)

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Error logs for %s (last %s)", app, timerange),
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{
						Text: fmt.Sprintf(
							"Use the loki_query tool to find error logs:\n\n"+
								"Query: %s\n"+
								"Start: %s\n"+
								"Direction: backward\n"+
								"Limit: 100",
							query, timerange,
						),
					},
				},
			},
		}, nil
	}
}

// RateQueryPrompt returns the MCP prompt definition for rate_query.
func RateQueryPrompt() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        "rate_query",
		Description: "Calculate the rate of log lines matching a selector",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "selector",
				Description: "LogQL stream selector (e.g. {app=\"nginx\"})",
				Required:    true,
			},
			{
				Name:        "interval",
				Description: "Rate interval (e.g. 5m, 1h). Default: 5m",
				Required:    false,
			},
		},
	}
}

// RateQueryHandler returns the handler for the rate_query prompt.
func RateQueryHandler() mcp.PromptHandler {
	return func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		selector := req.Params.Arguments["selector"]
		if selector == "" {
			return nil, validationErr(errors.New("selector argument is required"))
		}

		err := validateSelector(selector)
		if err != nil {
			return nil, err
		}

		interval := req.Params.Arguments["interval"]
		if interval == "" {
			interval = defaultInterval
		} else {
			err = validateRelativeDuration(interval)
			if err != nil {
				return nil, err
			}
		}

		query := fmt.Sprintf("rate(%s[%s])", selector, interval)

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Rate query for %s over %s intervals", selector, interval),
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{
						Text: fmt.Sprintf(
							"Use the loki_query tool to calculate log rate:\n\n"+
								"Query: %s\n"+
								"Start: 1h\n"+
								"Direction: backward",
							query,
						),
					},
				},
			},
		}, nil
	}
}

// TopLabelValuesPrompt returns the MCP prompt definition for top_label_values.
func TopLabelValuesPrompt() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        "top_label_values",
		Description: "Find top N values for a label by log volume",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "selector",
				Description: "LogQL stream selector (e.g. {app=\"nginx\"})",
				Required:    true,
			},
			{
				Name:        "label",
				Description: "Label name to group by",
				Required:    true,
			},
			{
				Name:        "n",
				Description: "Number of top values to return. Default: 10",
				Required:    false,
			},
			{
				Name:        "interval",
				Description: "Rate interval (e.g. 5m, 1h). Default: 5m",
				Required:    false,
			},
		},
	}
}

type topLabelValuesArgs struct {
	selector, label, n, interval string
}

func parseTopLabelValuesArgs(args map[string]string) (*topLabelValuesArgs, error) {
	selector := args["selector"]
	if selector == "" {
		return nil, validationErr(errors.New("selector argument is required"))
	}

	err := validateSelector(selector)
	if err != nil {
		return nil, err
	}

	label := args["label"]
	if label == "" {
		return nil, validationErr(errors.New("label argument is required"))
	}

	err = validateLabelName(label)
	if err != nil {
		return nil, err
	}

	n := args["n"]
	if n == "" {
		n = "10"
	} else {
		nVal, atoiErr := strconv.Atoi(n)
		if atoiErr != nil || nVal <= 0 {
			return nil, validationErr(
				errors.Newf("invalid n %q: must be a positive integer", n),
			)
		}
	}

	interval := args["interval"]
	if interval == "" {
		interval = defaultInterval
	} else {
		err = validateRelativeDuration(interval)
		if err != nil {
			return nil, err
		}
	}

	return &topLabelValuesArgs{selector: selector, label: label, n: n, interval: interval}, nil
}

// TopLabelValuesHandler returns the handler for the top_label_values prompt.
func TopLabelValuesHandler() mcp.PromptHandler {
	return func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		parsed, err := parseTopLabelValuesArgs(req.Params.Arguments)
		if err != nil {
			return nil, err
		}

		query := fmt.Sprintf("topk(%s, sum by (%s) (rate(%s[%s])))",
			parsed.n, parsed.label, parsed.selector, parsed.interval)

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Top %s %s values by log volume", parsed.n, parsed.label),
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{
						Text: fmt.Sprintf(
							"Use the loki_query tool to find top label values:\n\n"+
								"Query: %s\n"+
								"Start: 1h\n"+
								"Direction: backward",
							query,
						),
					},
				},
			},
		}, nil
	}
}
