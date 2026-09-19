// Package httpclient is the HTTP adapter for the his.Client port, talking to
// a Mock HIS or any service implementing the mock-his OpenAPI contract.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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

// Events reads one page of the append-only canonical event feed, strictly
// after the cursor, oldest first. A non-positive limit lets the HIS pick its
// default page size.
func (c *Client) Events(ctx context.Context, after string, limit int) (his.EventPage, error) {
	var page his.EventPage

	query := url.Values{}
	if after != "" {
		query.Set("after", after)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/v1/events?%s", c.baseURL, query.Encode()), nil)
	if err != nil {
		return page, apperr.Wrapf(apperr.KindUpstream, err, "build request failed")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return page, apperr.Wrapf(apperr.KindUpstream, err, "HIS unavailable")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return page, apperr.Wrapf(apperr.KindUpstream, fmt.Errorf("status %d", resp.StatusCode), "HIS returned error")
	}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return page, apperr.Wrapf(apperr.KindUpstream, err, "invalid HIS response")
	}
	return page, nil
}
