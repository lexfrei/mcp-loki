package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	initializeBody = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{` +
		`"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`
	pingBody = `{"jsonrpc":"2.0","id":2,"method":"ping"}`

	testSessionTimeout = 50 * time.Millisecond
	expiryDeadline     = 5 * time.Second
	pollInterval       = 100 * time.Millisecond
	idleWait           = 200 * time.Millisecond
)

// postMCP returns the status code and the Mcp-Session-Id response header, so
// callers never hold the response itself.
func postMCP(t *testing.T, url, sessionID, body string) (int, string) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}

	defer func() { _ = resp.Body.Close() }()

	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, resp.Header.Get("Mcp-Session-Id")
}

func initSession(t *testing.T, sessionTimeout time.Duration) (string, string) {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	ts := httptest.NewServer(newHTTPHandler(server, sessionTimeout))
	t.Cleanup(ts.Close)

	status, sessionID := postMCP(t, ts.URL, "", initializeBody)
	if status != http.StatusOK {
		t.Fatalf("initialize: status %d", status)
	}

	if sessionID == "" {
		t.Fatal("initialize: no Mcp-Session-Id header")
	}

	return ts.URL, sessionID
}

func TestHTTPHandler_ClosesIdleSession(t *testing.T) {
	url, sessionID := initSession(t, testSessionTimeout)

	deadline := time.Now().Add(expiryDeadline)

	for {
		status, _ := postMCP(t, url, sessionID, pingBody)
		if status == http.StatusNotFound {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("idle session still alive after deadline: last status %d", status)
		}

		time.Sleep(pollInterval)
	}
}

func TestHTTPHandler_ZeroTimeoutKeepsSession(t *testing.T) {
	url, sessionID := initSession(t, 0)

	time.Sleep(idleWait)

	status, _ := postMCP(t, url, sessionID, pingBody)
	if status != http.StatusOK {
		t.Fatalf("expected session to survive with timeout disabled, got status %d", status)
	}
}
