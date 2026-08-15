package tools_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/lexfrei/mcp-loki/internal/loki"
	"github.com/lexfrei/mcp-loki/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	queryInstantPath = "/loki/api/v1/query"
	queryRangePath   = "/loki/api/v1/query_range"

	valueAPI          = "api"
	metricSumOverTime = `sum by (namespace) (count_over_time({namespace=~".+"}[1h]))`
)

func TestQueryHandler_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := loki.QueryResponse{
			Status: statusSuccess,
			Data: loki.QueryData{
				ResultType: resultTypeValue,
				Result: []loki.StreamResult{
					{
						Stream: map[string]string{argApp: "test"},
						Values: [][]json.RawMessage{
							{json.RawMessage(`"1609459200000000000"`), json.RawMessage(`"test log line"`)},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewQueryHandler(client)

	params := tools.QueryParams{
		Query: selectorTest,
	}

	_, output, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if output.ResultType != resultTypeValue {
		t.Errorf("expected resultType streams, got %s", output.ResultType)
	}

	if len(output.Streams) != 1 {
		t.Fatalf("expected 1 structured stream entry, got %d", len(output.Streams))
	}

	if output.Streams[0].Line != "test log line" {
		t.Errorf("expected structured line, got %q", output.Streams[0].Line)
	}

	if output.Output == "" {
		t.Error("expected deprecated output field for compatibility")
	}
}

func TestQueryHandler_AutoRoutesMetricRangeWithStep(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != queryRangePath {
			t.Fatalf("expected query_range, got %s", r.URL.Path)
		}

		if r.URL.Query().Get("step") != "1m" {
			t.Fatalf("expected default step 1m, got %s", r.URL.Query().Get("step"))
		}

		resp := loki.QueryResponse{
			Status: statusSuccess,
			Data: loki.QueryData{
				ResultType: resultTypeMatrix,
				Result: []loki.StreamResult{
					{
						Metric: map[string]string{labelNamespace: namespaceKube},
						Values: [][]json.RawMessage{
							{json.RawMessage(`1609459200`), json.RawMessage(`"5"`)},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewQueryHandler(client)

	_, output, err := handler(context.Background(), &mcp.CallToolRequest{}, tools.QueryParams{
		Query: metricSumOverTime,
		Start: timerange1h,
	})
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if output.ResultType != resultTypeMatrix {
		t.Fatalf("expected matrix, got %s", output.ResultType)
	}

	if len(output.Series) != 1 || output.Series[0].Labels[labelNamespace] != namespaceKube {
		t.Fatalf("expected labeled series, got %+v", output.Series)
	}
}

func TestQueryHandler_AutoRoutesMetricInstant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != queryInstantPath {
			t.Fatalf("expected query, got %s", r.URL.Path)
		}

		resp := loki.QueryResponse{
			Status: statusSuccess,
			Data: loki.QueryData{
				ResultType: resultTypeVector,
				Result: []loki.StreamResult{
					{
						Metric: map[string]string{labelNamespace: namespaceFoo},
						Value: []json.RawMessage{
							json.RawMessage(`1609459200`),
							json.RawMessage(`"7"`),
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewQueryHandler(client)

	_, output, err := handler(context.Background(), &mcp.CallToolRequest{}, tools.QueryParams{
		Query: `count_over_time({namespace="foo"}[1h])`,
	})
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if output.ResultType != resultTypeVector {
		t.Fatalf("expected vector, got %s", output.ResultType)
	}

	if output.Series[0].Value == nil || output.Series[0].Value.Value != "7" {
		t.Fatalf("expected instant vector value, got %+v", output.Series)
	}
}

func TestQueryHandler_AutoRoutesLogSelectorsToRange(t *testing.T) {
	queries := []string{
		`{app="api"} |= "rate"`,
		`{app="rate-limiter"}`,
		`{job="web"} |= "count"`,
		`{app="max-pods"}`,
		`{app="api"} |= "vector clock"`,
		`{app="api"} |= "error"`,
		"{app=\"api\"} |= `rate limited`",
		"{app=\"api\"} |~ `error|rate`",
		`{app="api"} | json | count=5`,
		`{app="api"} | logfmt | rate=0.5`,
		`{app="api"} | json # count the errors`,
		"# count the errors\n{app=\"api\"}",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			var seenPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path

				if r.URL.Path == queryInstantPath {
					http.Error(w, `{"status":"error","errorType":"bad_data","error":"log queries require query_range"}`, http.StatusBadRequest)

					return
				}

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(loki.QueryResponse{
					Status: statusSuccess,
					Data: loki.QueryData{
						ResultType: resultTypeValue,
						Result: []loki.StreamResult{{
							Stream: map[string]string{argApp: valueAPI},
							Values: [][]json.RawMessage{{
								json.RawMessage(`"1609459200000000000"`),
								json.RawMessage(`"hello"`),
							}},
						}},
					},
				})
			}))
			defer server.Close()

			handler := tools.NewQueryHandler(loki.NewClient(server.URL, "", "", "", ""))
			_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, tools.QueryParams{Query: query})
			if err != nil {
				t.Fatalf("query %q failed through %s: %v", query, seenPath, err)
			}

			if seenPath != queryRangePath {
				t.Fatalf("query %q used %s, want query_range", query, seenPath)
			}
		})
	}
}

func TestQueryHandler_AutoRoutesMetricExpressionsToInstant(t *testing.T) {
	queries := []string{
		`rate_counter({app="api"} | unwrap bytes [5m])`,
		"sum(count_over_time({app=\"api\"} |= `rate limited` [5m]))",
		"# hourly errors\ncount_over_time({app=\"api\"}[1h])",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			var seenPath string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(loki.QueryResponse{
					Status: statusSuccess,
					Data: loki.QueryData{
						ResultType: resultTypeVector,
						Result: []loki.StreamResult{{
							Metric: map[string]string{labelNamespace: namespaceFoo},
							Value: []json.RawMessage{
								json.RawMessage(`1609459200`),
								json.RawMessage(`"7"`),
							},
						}},
					},
				})
			}))
			defer server.Close()

			handler := tools.NewQueryHandler(loki.NewClient(server.URL, "", "", "", ""))

			_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, tools.QueryParams{Query: query})
			if err != nil {
				t.Fatalf("query %q failed: %v", query, err)
			}

			if seenPath != queryInstantPath {
				t.Fatalf("query %q used %s, want %s", query, seenPath, queryInstantPath)
			}
		})
	}
}

func TestQueryHandler_ExplicitQueryTypeOverridesAutoRouting(t *testing.T) {
	metricQuery := `count_over_time({app="test"}[1h])`

	testCases := []struct {
		name      string
		queryType string
		wantPath  string
	}{
		{
			name:      "range keeps a metric without time bounds on query_range",
			queryType: "range",
			wantPath:  queryRangePath,
		},
		{
			name:      "instant uses the instant endpoint",
			queryType: "instant",
			wantPath:  queryInstantPath,
		},
		{
			name:      "surrounding space and casing are accepted",
			queryType: " Instant ",
			wantPath:  queryInstantPath,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var seenPath string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(loki.QueryResponse{
					Status: statusSuccess,
					Data:   loki.QueryData{ResultType: resultTypeVector},
				})
			}))
			defer server.Close()

			handler := tools.NewQueryHandler(loki.NewClient(server.URL, "", "", "", ""))

			_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, tools.QueryParams{
				Query:     metricQuery,
				QueryType: testCase.queryType,
			})
			if err != nil {
				t.Fatalf("handler failed: %v", err)
			}

			if seenPath != testCase.wantPath {
				t.Errorf("queryType %q used %s, want %s", testCase.queryType, seenPath, testCase.wantPath)
			}
		})
	}
}

func TestQueryHandler_TruncatedReflectsLimit(t *testing.T) {
	matrixResponse := loki.QueryResponse{
		Status: statusSuccess,
		Data: loki.QueryData{
			ResultType: resultTypeMatrix,
			Result: []loki.StreamResult{{
				Metric: map[string]string{labelNamespace: namespaceKube},
				Values: [][]json.RawMessage{{json.RawMessage(`1609459200`), json.RawMessage(`"5"`)}},
			}},
		},
	}

	testCases := []struct {
		name      string
		query     string
		limit     int
		response  loki.QueryResponse
		truncated bool
	}{
		{
			name:      "stream entries reach the limit",
			query:     selectorTest,
			limit:     2,
			response:  streamResponseWithEntries(2),
			truncated: true,
		},
		{
			name:      "stream entries stay below the limit",
			query:     selectorTest,
			limit:     3,
			response:  streamResponseWithEntries(2),
			truncated: false,
		},
		{
			name:      "metric series are never reported as truncated",
			query:     metricSumOverTime,
			limit:     1,
			response:  matrixResponse,
			truncated: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(testCase.response)
			}))
			defer server.Close()

			handler := tools.NewQueryHandler(loki.NewClient(server.URL, "", "", "", ""))

			_, output, err := handler(context.Background(), &mcp.CallToolRequest{}, tools.QueryParams{
				Query: testCase.query,
				Limit: testCase.limit,
				Start: timerange1h,
			})
			if err != nil {
				t.Fatalf("handler failed: %v", err)
			}

			if output.Truncated != testCase.truncated {
				t.Errorf("truncated = %v, want %v", output.Truncated, testCase.truncated)
			}
		})
	}
}

func streamResponseWithEntries(entries int) loki.QueryResponse {
	values := make([][]json.RawMessage, 0, entries)

	for range entries {
		values = append(values, []json.RawMessage{
			json.RawMessage(`"1609459200000000000"`),
			json.RawMessage(`"hello"`),
		})
	}

	return loki.QueryResponse{
		Status: statusSuccess,
		Data: loki.QueryData{
			ResultType: resultTypeValue,
			Result: []loki.StreamResult{{
				Stream: map[string]string{argApp: valueAPI},
				Values: values,
			}},
		},
	}
}

func TestQueryHandler_InvalidQueryType(t *testing.T) {
	client := loki.NewClient("http://localhost:3100", "", "", "", "")
	handler := tools.NewQueryHandler(client)

	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, tools.QueryParams{
		Query:     selectorTest,
		QueryType: "invalid",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}

	if !errors.Is(err, tools.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestQueryHandler_WithTimeRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")

		if start == "" || end == "" {
			t.Error("expected start and end parameters")
		}

		resp := loki.QueryResponse{
			Status: statusSuccess,
			Data:   loki.QueryData{ResultType: resultTypeValue, Result: []loki.StreamResult{}},
		}
		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")
	handler := tools.NewQueryHandler(client)

	params := tools.QueryParams{
		Query: selectorTest,
		Start: timerange1h,
		End:   timeNow,
	}

	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}
}

func TestQueryHandler_RelativeTime(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"hours", timerange1h, true},
		{"minutes", timerange30m, true},
		{"days", "7d", true},
		{timeNow, timeNow, true},
		{"rfc3339", timeRFC3339Sample, true},
		{errorTypeData, errorTypeData, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tools.ParseTime(tt.input)
			if tt.valid && err != nil {
				t.Errorf("expected valid time, got error: %v", err)
			}

			if !tt.valid && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestQueryHandler_MissingQuery(t *testing.T) {
	client := loki.NewClient("http://localhost:3100", "", "", "", "")
	handler := tools.NewQueryHandler(client)

	params := tools.QueryParams{
		Query: "",
	}

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err == nil {
		t.Error("expected error for missing query")
	}

	if result != nil {
		t.Error("expected nil CallToolResult on error path")
	}

	if !errors.Is(err, tools.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestQueryHandler_InvalidStartTime(t *testing.T) {
	client := loki.NewClient("http://localhost:3100", "", "", "", "")
	handler := tools.NewQueryHandler(client)

	params := tools.QueryParams{
		Query: selectorTest,
		Start: timeNotParsable,
	}

	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err == nil {
		t.Fatal("expected error for invalid start time")
	}

	if !errors.Is(err, tools.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestQueryHandler_LokiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":"error","error":"internal"}`))
	}))
	defer srv.Close()

	client := loki.NewClient(srv.URL, "", "", "", "")
	handler := tools.NewQueryHandler(client)

	params := tools.QueryParams{
		Query: selectorTest,
	}

	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, params)
	if err == nil {
		t.Fatal("expected error for Loki failure")
	}

	if !errors.Is(err, tools.ErrLokiRequest) {
		t.Errorf("expected ErrLokiRequest, got: %v", err)
	}
}
