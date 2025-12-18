// Package loki provides HTTP client for Grafana Loki API.
package loki

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
)

const httpClientTimeout = 30 * time.Second

// ErrLokiAPI represents an error returned by the Loki API.
var ErrLokiAPI = errors.New("loki API error")

// Client is an HTTP client for the Loki API.
type Client struct {
	baseURL  string
	username string
	password string
	token    string
	orgID    string
	client   *http.Client
}

// NewClient creates a new Loki API client.
func NewClient(baseURL, username, password, token, orgID string) *Client {
	return &Client{
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		username: username,
		password: password,
		token:    token,
		orgID:    orgID,
		client:   &http.Client{Timeout: httpClientTimeout},
	}
}

// QueryRange executes a LogQL range query.
func (c *Client) QueryRange(
	ctx context.Context,
	query string,
	start, end time.Time,
	limit int,
	direction string,
) (*QueryResponse, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	params.Set("end", strconv.FormatInt(end.UnixNano(), 10))
	params.Set("limit", strconv.Itoa(limit))
	params.Set("direction", direction)

	var resp QueryResponse

	err := c.doRequest(ctx, "/loki/api/v1/query_range", params, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Labels returns the list of known label names.
func (c *Client) Labels(ctx context.Context, start, end time.Time) (*LabelsResponse, error) {
	params := url.Values{}
	params.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	params.Set("end", strconv.FormatInt(end.UnixNano(), 10))

	var resp LabelsResponse

	err := c.doRequest(ctx, "/loki/api/v1/labels", params, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// LabelValues returns the known values for a given label.
func (c *Client) LabelValues(ctx context.Context, labelName string, start, end time.Time) (*LabelsResponse, error) {
	params := url.Values{}
	params.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	params.Set("end", strconv.FormatInt(end.UnixNano(), 10))

	var resp LabelsResponse

	err := c.doRequest(ctx, "/loki/api/v1/label/"+labelName+"/values", params, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Series returns the list of time series that match a certain label set.
func (c *Client) Series(ctx context.Context, match []string, start, end time.Time) (*SeriesResponse, error) {
	params := url.Values{}
	params.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	params.Set("end", strconv.FormatInt(end.UnixNano(), 10))

	for _, m := range match {
		params.Add("match[]", m)
	}

	var resp SeriesResponse

	err := c.doRequest(ctx, "/loki/api/v1/series", params, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Stats returns index statistics for a given query.
func (c *Client) Stats(ctx context.Context, query string, start, end time.Time) (*StatsResponse, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	params.Set("end", strconv.FormatInt(end.UnixNano(), 10))

	var resp StatsResponse

	err := c.doRequest(ctx, "/loki/api/v1/index/stats", params, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) doRequest(ctx context.Context, path string, params url.Values, result any) error {
	reqURL := c.baseURL + path + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	c.setAuthHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "request failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode >= http.StatusBadRequest {
		var errResp ErrorResponse

		unmarshalErr := json.Unmarshal(body, &errResp)
		if unmarshalErr == nil && errResp.Error != "" {
			return errors.Wrapf(ErrLokiAPI, "%s: %s", errResp.ErrorType, errResp.Error)
		}

		return errors.Wrapf(ErrLokiAPI, "status %d: %s", resp.StatusCode, string(body))
	}

	unmarshalErr := json.Unmarshal(body, result)
	if unmarshalErr != nil {
		return errors.Wrap(unmarshalErr, "failed to decode response")
	}

	return nil
}

func (c *Client) setAuthHeaders(req *http.Request) {
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	} else if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	if c.orgID != "" {
		// Use direct assignment to preserve exact header case required by Loki
		req.Header["X-Scope-OrgID"] = []string{c.orgID}
	}
}
