package loki_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lexfrei/mcp-loki/internal/loki"
)

const statusSuccess = "success"

func TestClient_QueryRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/query_range" {
			t.Errorf("expected path /loki/api/v1/query_range, got %s", r.URL.Path)
		}

		if r.URL.Query().Get("query") != `{app="test"}` {
			t.Errorf("expected query {app=\"test\"}, got %s", r.URL.Query().Get("query"))
		}

		resp := loki.QueryResponse{
			Status: statusSuccess,
			Data: loki.QueryData{
				ResultType: "streams",
				Result: []loki.StreamResult{
					{
						Stream: map[string]string{"app": "test"},
						Values: [][]json.RawMessage{
							{json.RawMessage(`"1609459200000000000"`), json.RawMessage(`"test log"`)},
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

	resp, err := client.QueryRange(context.Background(), `{app="test"}`, time.Now().Add(-time.Hour), time.Now(), 100, "backward")
	if err != nil {
		t.Fatalf("QueryRange failed: %v", err)
	}

	if resp.Status != statusSuccess {
		t.Errorf("expected status success, got %s", resp.Status)
	}

	if len(resp.Data.Result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Data.Result))
	}
}

func TestClient_Labels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/labels" {
			t.Errorf("expected path /loki/api/v1/labels, got %s", r.URL.Path)
		}

		resp := loki.LabelsResponse{
			Status: statusSuccess,
			Data:   []string{"app", "env", "host"},
		}

		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")

	resp, err := client.Labels(context.Background(), time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Fatalf("Labels failed: %v", err)
	}

	if resp.Status != statusSuccess {
		t.Errorf("expected status success, got %s", resp.Status)
	}

	if len(resp.Data) != 3 {
		t.Fatalf("expected 3 labels, got %d", len(resp.Data))
	}
}

func TestClient_LabelValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/label/app/values" {
			t.Errorf("expected path /loki/api/v1/label/app/values, got %s", r.URL.Path)
		}

		resp := loki.LabelsResponse{
			Status: statusSuccess,
			Data:   []string{"nginx", "redis", "postgres"},
		}

		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")

	resp, err := client.LabelValues(context.Background(), "app", time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Fatalf("LabelValues failed: %v", err)
	}

	if len(resp.Data) != 3 {
		t.Fatalf("expected 3 values, got %d", len(resp.Data))
	}
}

func TestClient_Series(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/series" {
			t.Errorf("expected path /loki/api/v1/series, got %s", r.URL.Path)
		}

		resp := loki.SeriesResponse{
			Status: statusSuccess,
			Data: []map[string]string{
				{"app": "nginx", "env": "prod"},
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

	resp, err := client.Series(context.Background(), []string{`{app="nginx"}`}, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Fatalf("Series failed: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 series, got %d", len(resp.Data))
	}
}

func TestClient_Stats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/index/stats" {
			t.Errorf("expected path /loki/api/v1/index/stats, got %s", r.URL.Path)
		}

		resp := loki.StatsResponse{
			Status: statusSuccess,
			Data: loki.StatsData{
				Streams: 100,
				Chunks:  5000,
				Bytes:   1048576,
				Entries: 50000,
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

	resp, err := client.Stats(context.Background(), `{app="nginx"}`, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if resp.Data.Streams != 100 {
		t.Errorf("expected 100 streams, got %d", resp.Data.Streams)
	}
}

func TestClient_BasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected basic auth header")

			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		if user != "testuser" || pass != "testpass" {
			t.Errorf("expected testuser:testpass, got %s:%s", user, pass)
		}

		resp := loki.LabelsResponse{Status: statusSuccess, Data: []string{}}
		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "testuser", "testpass", "", "")

	_, err := client.Labels(context.Background(), time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Fatalf("Labels with basic auth failed: %v", err)
	}
}

func TestClient_BearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my-token" {
			t.Errorf("expected Bearer my-token, got %s", auth)
		}

		resp := loki.LabelsResponse{Status: statusSuccess, Data: []string{}}
		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "my-token", "")

	_, err := client.Labels(context.Background(), time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Fatalf("Labels with bearer token failed: %v", err)
	}
}

func TestClient_OrgID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Headers are canonicalized by net/http, so X-Scope-OrgID becomes X-Scope-Orgid
		orgID := r.Header.Get("X-Scope-Orgid")
		if orgID != "tenant-1" {
			t.Errorf("expected X-Scope-OrgID tenant-1, got %s", orgID)
		}

		resp := loki.LabelsResponse{Status: statusSuccess, Data: []string{}}
		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "tenant-1")

	_, err := client.Labels(context.Background(), time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Fatalf("Labels with org ID failed: %v", err)
	}
}

func TestClient_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)

		resp := loki.ErrorResponse{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     "invalid query",
		}

		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := loki.NewClient(server.URL, "", "", "", "")

	_, err := client.Labels(context.Background(), time.Now().Add(-time.Hour), time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
