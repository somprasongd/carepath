// Package line implements the production Notifier adapter (ADR-0013 §1):
// a text push through the LINE Messaging API. It is selected at startup only
// when LINE_MESSAGING_CHANNEL_TOKEN is set — the token is the operator's
// opt-in to real, billed sends.
//
// The adapter is unit-tested against a scripted fake server; it has not been
// exercised against the real API from this repository (no channel exists
// yet). Enabling it is an environment change, not a code change.
package line

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"carepath/apps/api/internal/platform/apperr"
)

// ProductionEndpoint is the Messaging API push URL. A field-level override
// exists for tests only.
const ProductionEndpoint = "https://api.line.me/v2/bot/message/push"

// Notifier pushes text messages via the LINE Messaging API.
type Notifier struct {
	token    string
	endpoint string
	http     *http.Client
}

func New(channelToken string, httpClient *http.Client) *Notifier {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &Notifier{token: channelToken, endpoint: ProductionEndpoint, http: httpClient}
}

func (n *Notifier) Channel() string { return "line" }

type pushRequest struct {
	To       string        `json:"to"`
	Messages []pushMessage `json:"messages"`
}

type pushMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (n *Notifier) Notify(ctx context.Context, recipient, text string) error {
	body, err := json.Marshal(pushRequest{
		To:       recipient,
		Messages: []pushMessage{{Type: "text", Text: text}},
	})
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "line: encode push")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.endpoint, bytes.NewReader(body))
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "line: build push request")
	}
	req.Header.Set("Authorization", "Bearer "+n.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.http.Do(req)
	if err != nil {
		return apperr.Wrapf(apperr.KindUpstream, err, "line: push unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return apperr.New(apperr.KindUpstream,
			fmt.Sprintf("line: push rejected: %s: %s", resp.Status, strings.TrimSpace(string(detail))))
	}
	return nil
}
