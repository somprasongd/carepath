// Package httpclient is the HTTP adapter for the his.Client port, talking to
// a Mock HIS or any service implementing the mock-his OpenAPI contract.
package httpclient

import (
	"bytes"
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

// TransitionStep sends the canonical step-status command to the HIS
// (ADR-0008). HIS-side validation errors keep their meaning: 400 maps to
// Invalid, 404 to NotFound, 409 to Conflict — each carrying the HIS's own
// message (e.g. which rule rejected the transition).
func (c *Client) TransitionStep(ctx context.Context, visitID string, sequence int, cmd his.TransitionCommand) (his.VisitStep, error) {
	var step his.VisitStep

	payload, err := json.Marshal(cmd)
	if err != nil {
		return step, apperr.Wrapf(apperr.KindInternal, err, "encode command failed")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/v1/visits/%s/steps/%d/transition",
			c.baseURL, url.PathEscape(visitID), sequence),
		bytes.NewReader(payload))
	if err != nil {
		return step, apperr.Wrapf(apperr.KindUpstream, err, "build request failed")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return step, apperr.Wrapf(apperr.KindUpstream, err, "HIS unavailable")
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
	case resp.StatusCode == http.StatusBadRequest:
		return step, statusError(apperr.KindInvalid, resp)
	case resp.StatusCode == http.StatusNotFound:
		return step, statusError(apperr.KindNotFound, resp)
	case resp.StatusCode == http.StatusConflict:
		return step, statusError(apperr.KindConflict, resp)
	default:
		return step, apperr.Wrapf(apperr.KindUpstream, fmt.Errorf("status %d", resp.StatusCode), "HIS returned error")
	}
	if err := json.NewDecoder(resp.Body).Decode(&step); err != nil {
		return step, apperr.Wrapf(apperr.KindUpstream, err, "invalid HIS response")
	}
	return step, nil
}

// statusError decodes the contract's {"error": msg} envelope so a rejected
// command surfaces the HIS's own reason instead of a generic one.
func statusError(kind apperr.Kind, resp *http.Response) error {
	var body struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body.Error == "" {
		return apperr.New(kind, "HIS rejected the command")
	}
	return apperr.New(kind, body.Error)
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
