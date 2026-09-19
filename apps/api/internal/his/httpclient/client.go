// Package httpclient is the HTTP adapter for the his.Client port, talking to
// a Mock HIS or any service implementing the mock-his OpenAPI contract.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/platform/apperr"
)

// Client implements his.Client over HTTP.
type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &Client{baseURL: strings.TrimSuffix(baseURL, "/"), http: httpClient}
}

func (c *Client) GetVisit(ctx context.Context, visitID string) (his.Visit, error) {
	var visit his.Visit

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/v1/visits/%s", c.baseURL, url.PathEscape(visitID)), nil)
	if err != nil {
		return visit, apperr.Wrapf(apperr.KindUpstream, err, "build request failed")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return visit, apperr.Wrapf(apperr.KindUpstream, err, "HIS unavailable")
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return visit, his.ErrVisitNotFound
	case resp.StatusCode != http.StatusOK:
		return visit, apperr.Wrapf(apperr.KindUpstream, fmt.Errorf("status %d", resp.StatusCode), "HIS returned error")
	}
	if err := json.NewDecoder(resp.Body).Decode(&visit); err != nil {
		return visit, apperr.Wrapf(apperr.KindUpstream, err, "invalid HIS response")
	}
	return visit, nil
}
