package tools_test

import (
	"context"
	"strings"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/lexfrei/mcp-loki/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	promptErrorLogs      = "error_logs"
	promptRateQuery      = "rate_query"
	promptTopLabelValues = "top_label_values"
)

func newPromptRequest(name string, args map[string]string) *mcp.GetPromptRequest {
	return &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      name,
			Arguments: args,
		},
	}
}

func TestErrorLogsPrompt_Definition(t *testing.T) {
	prompt := tools.ErrorLogsPrompt()

	if prompt.Name != promptErrorLogs {
		t.Errorf("expected name error_logs, got %s", prompt.Name)
	}

	if prompt.Description == "" {
		t.Error("expected non-empty description")
	}

	if len(prompt.Arguments) < 1 {
		t.Fatal("expected at least 1 argument")
	}

	var hasApp bool

	for _, arg := range prompt.Arguments {
		if arg.Name == "app" {
			hasApp = true

			if !arg.Required {
				t.Error("app argument should be required")
			}
		}
	}

	if !hasApp {
		t.Error("expected app argument")
	}
}

func TestErrorLogsPrompt_Handler(t *testing.T) {
	handler := tools.ErrorLogsHandler()

	req := newPromptRequest(promptErrorLogs, map[string]string{
		"app": "nginx",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if len(result.Messages) == 0 {
		t.Fatal("expected at least one message")
	}

	msg := result.Messages[0]
	if msg.Role != "user" {
		t.Errorf("expected role user, got %s", msg.Role)
	}

	textContent, ok := msg.Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", msg.Content)
	}

	if !strings.Contains(textContent.Text, `{app="nginx"}`) {
		t.Errorf("expected LogQL with app=nginx, got: %s", textContent.Text)
	}

	if !strings.Contains(textContent.Text, "error") {
		t.Errorf("expected 'error' in query, got: %s", textContent.Text)
	}
}

func TestErrorLogsPrompt_DefaultTimerange(t *testing.T) {
	handler := tools.ErrorLogsHandler()

	req := newPromptRequest(promptErrorLogs, map[string]string{
		"app": "nginx",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Messages[0].Content)
	}

	if !strings.Contains(textContent.Text, "Start: 1h") {
		t.Errorf("expected default timerange 1h, got: %s", textContent.Text)
	}
}

func TestErrorLogsPrompt_MissingApp(t *testing.T) {
	handler := tools.ErrorLogsHandler()

	req := newPromptRequest(promptErrorLogs, map[string]string{})

	_, err := handler(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing app argument")
	}

	if !errors.Is(err, tools.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestErrorLogsPrompt_CustomTimerange(t *testing.T) {
	handler := tools.ErrorLogsHandler()

	req := newPromptRequest(promptErrorLogs, map[string]string{
		"app":       "nginx",
		"timerange": "30m",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Messages[0].Content)
	}

	if !strings.Contains(textContent.Text, "Start: 30m") {
		t.Errorf("expected custom timerange 30m, got: %s", textContent.Text)
	}
}

func TestErrorLogsPrompt_InvalidTimerange(t *testing.T) {
	handler := tools.ErrorLogsHandler()

	tests := []struct {
		name      string
		timerange string
	}{
		{"random string", "banana"},
		{"RFC3339", "2024-01-01T00:00:00Z"},
		{"now keyword", "now"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newPromptRequest(promptErrorLogs, map[string]string{
				"app":       "nginx",
				"timerange": tt.timerange,
			})

			_, err := handler(context.Background(), req)
			if err == nil {
				t.Fatal("expected error for invalid timerange")
			}

			if !errors.Is(err, tools.ErrValidation) {
				t.Errorf("expected ErrValidation, got: %v", err)
			}
		})
	}
}

func TestRateQueryPrompt_Definition(t *testing.T) {
	prompt := tools.RateQueryPrompt()

	if prompt.Name != promptRateQuery {
		t.Errorf("expected name rate_query, got %s", prompt.Name)
	}

	if prompt.Description == "" {
		t.Error("expected non-empty description")
	}

	var hasSelector bool

	for _, arg := range prompt.Arguments {
		if arg.Name == "selector" {
			hasSelector = true

			if !arg.Required {
				t.Error("selector argument should be required")
			}
		}
	}

	if !hasSelector {
		t.Error("expected selector argument")
	}
}

func TestRateQueryPrompt_Handler(t *testing.T) {
	handler := tools.RateQueryHandler()

	req := newPromptRequest(promptRateQuery, map[string]string{
		"selector": `{app="nginx"}`,
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if len(result.Messages) == 0 {
		t.Fatal("expected at least one message")
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Messages[0].Content)
	}

	if !strings.Contains(textContent.Text, `rate({app="nginx"}`) {
		t.Errorf("expected rate() with selector, got: %s", textContent.Text)
	}

	if !strings.Contains(textContent.Text, "[5m]") {
		t.Errorf("expected default interval 5m, got: %s", textContent.Text)
	}
}

func TestRateQueryPrompt_CustomInterval(t *testing.T) {
	handler := tools.RateQueryHandler()

	req := newPromptRequest(promptRateQuery, map[string]string{
		"selector": `{app="nginx"}`,
		"interval": "15m",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Messages[0].Content)
	}

	if !strings.Contains(textContent.Text, "[15m]") {
		t.Errorf("expected interval 15m, got: %s", textContent.Text)
	}
}

func TestRateQueryPrompt_MissingSelector(t *testing.T) {
	handler := tools.RateQueryHandler()

	req := newPromptRequest(promptRateQuery, map[string]string{})

	_, err := handler(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing selector argument")
	}

	if !errors.Is(err, tools.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestRateQueryPrompt_InvalidInterval(t *testing.T) {
	handler := tools.RateQueryHandler()

	tests := []struct {
		name     string
		interval string
	}{
		{"random string", "foo"},
		{"RFC3339", "2024-01-01T00:00:00Z"},
		{"now keyword", "now"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newPromptRequest(promptRateQuery, map[string]string{
				"selector": `{app="nginx"}`,
				"interval": tt.interval,
			})

			_, err := handler(context.Background(), req)
			if err == nil {
				t.Fatal("expected error for invalid interval")
			}

			if !errors.Is(err, tools.ErrValidation) {
				t.Errorf("expected ErrValidation, got: %v", err)
			}
		})
	}
}

func TestTopLabelValuesPrompt_Definition(t *testing.T) {
	prompt := tools.TopLabelValuesPrompt()

	if prompt.Name != promptTopLabelValues {
		t.Errorf("expected name top_label_values, got %s", prompt.Name)
	}

	if prompt.Description == "" {
		t.Error("expected non-empty description")
	}

	requiredArgs := map[string]bool{"selector": false, "label": false}

	for _, arg := range prompt.Arguments {
		if _, ok := requiredArgs[arg.Name]; ok {
			requiredArgs[arg.Name] = true

			if !arg.Required {
				t.Errorf("%s argument should be required", arg.Name)
			}
		}
	}

	for name, found := range requiredArgs {
		if !found {
			t.Errorf("expected %s argument", name)
		}
	}
}

func TestTopLabelValuesPrompt_Handler(t *testing.T) {
	handler := tools.TopLabelValuesHandler()

	req := newPromptRequest(promptTopLabelValues, map[string]string{
		"selector": `{app="nginx"}`,
		"label":    "status",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if len(result.Messages) == 0 {
		t.Fatal("expected at least one message")
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Messages[0].Content)
	}

	if !strings.Contains(textContent.Text, "topk") {
		t.Errorf("expected topk in query, got: %s", textContent.Text)
	}

	if !strings.Contains(textContent.Text, "sum by") {
		t.Errorf("expected sum by in query, got: %s", textContent.Text)
	}

	if !strings.Contains(textContent.Text, "status") {
		t.Errorf("expected label name in query, got: %s", textContent.Text)
	}
}

func TestTopLabelValuesPrompt_CustomN(t *testing.T) {
	handler := tools.TopLabelValuesHandler()

	req := newPromptRequest(promptTopLabelValues, map[string]string{
		"selector": `{app="nginx"}`,
		"label":    "status",
		"n":        "5",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Messages[0].Content)
	}

	if !strings.Contains(textContent.Text, "topk(5") {
		t.Errorf("expected topk(5 in query, got: %s", textContent.Text)
	}
}

func TestErrorLogsPrompt_AppWithSpecialChars(t *testing.T) {
	handler := tools.ErrorLogsHandler()

	req := newPromptRequest(promptErrorLogs, map[string]string{
		"app": "my-app.v2",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Messages[0].Content)
	}

	if !strings.Contains(textContent.Text, "my-app.v2") {
		t.Errorf("expected app name in query, got: %s", textContent.Text)
	}
}

func TestErrorLogsPrompt_WeekDuration(t *testing.T) {
	handler := tools.ErrorLogsHandler()

	req := newPromptRequest(promptErrorLogs, map[string]string{
		"app":       "nginx",
		"timerange": "1w",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Messages[0].Content)
	}

	if !strings.Contains(textContent.Text, "Start: 1w") {
		t.Errorf("expected 1w timerange, got: %s", textContent.Text)
	}
}

func TestRateQueryPrompt_MalformedSelector(t *testing.T) {
	handler := tools.RateQueryHandler()

	req := newPromptRequest(promptRateQuery, map[string]string{
		"selector": `}) | evil`,
	})

	_, err := handler(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for malformed selector")
	}

	if !errors.Is(err, tools.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestTopLabelValuesPrompt_MalformedLabel(t *testing.T) {
	handler := tools.TopLabelValuesHandler()

	req := newPromptRequest(promptTopLabelValues, map[string]string{
		"selector": `{app="nginx"}`,
		"label":    `status"); drop`,
	})

	_, err := handler(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for malformed label")
	}

	if !errors.Is(err, tools.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestTopLabelValuesPrompt_MalformedSelector(t *testing.T) {
	handler := tools.TopLabelValuesHandler()

	req := newPromptRequest(promptTopLabelValues, map[string]string{
		"selector": "not-a-selector",
		"label":    "status",
	})

	_, err := handler(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for malformed selector")
	}

	if !errors.Is(err, tools.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestTopLabelValuesPrompt_InvalidN(t *testing.T) {
	handler := tools.TopLabelValuesHandler()

	tests := []struct {
		name string
		n    string
	}{
		{"negative", "-1"},
		{"zero", "0"},
		{"not a number", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newPromptRequest(promptTopLabelValues, map[string]string{
				"selector": `{app="nginx"}`,
				"label":    "status",
				"n":        tt.n,
			})

			_, err := handler(context.Background(), req)
			if err == nil {
				t.Fatal("expected error for invalid n")
			}

			if !errors.Is(err, tools.ErrValidation) {
				t.Errorf("expected ErrValidation, got: %v", err)
			}
		})
	}
}

func TestTopLabelValuesPrompt_MissingArgs(t *testing.T) {
	handler := tools.TopLabelValuesHandler()

	tests := []struct {
		name string
		args map[string]string
	}{
		{"missing both", map[string]string{}},
		{"missing label", map[string]string{"selector": `{app="nginx"}`}},
		{"missing selector", map[string]string{"label": "status"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newPromptRequest(promptTopLabelValues, tt.args)

			_, err := handler(context.Background(), req)
			if err == nil {
				t.Fatal("expected error for missing arguments")
			}

			if !errors.Is(err, tools.ErrValidation) {
				t.Errorf("expected ErrValidation, got: %v", err)
			}
		})
	}
}
