package line

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The adapter is exercised against a scripted fake of the Messaging API
// push endpoint: the request shape (auth header, recipient, single text
// message) is the contract. No real channel exists in this repository yet
// (ADR-0013 §1) — enabling the adapter is an environment change.
func TestNotifyPushesTextMessage(t *testing.T) {
	var gotAuth, gotContentType string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	notifier := New("test-channel-token", server.Client())
	notifier.endpoint = server.URL

	if err := notifier.Notify(context.Background(), "U123", "ใกล้ถึงคิวของคุณแล้ว"); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if gotAuth != "Bearer test-channel-token" {
		t.Fatalf("authorization = %q, want the channel token as a bearer", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Fatalf("content-type = %q", gotContentType)
	}
	if gotBody["to"] != "U123" {
		t.Fatalf("to = %v, want the recipient user id", gotBody["to"])
	}
	messages, ok := gotBody["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("messages = %v, want exactly one message", gotBody["messages"])
	}
	msg := messages[0].(map[string]any)
	if msg["type"] != "text" || msg["text"] != "ใกล้ถึงคิวของคุณแล้ว" {
		t.Fatalf("message = %v, want a text message with the given text", msg)
	}
}

func TestNotifySurfacesRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Invalid channel access token"}`))
	}))
	defer server.Close()

	notifier := New("bad-token", server.Client())
	notifier.endpoint = server.URL

	err := notifier.Notify(context.Background(), "U123", "text")
	if err == nil {
		t.Fatal("notify with a rejected token must fail")
	}
	if want := "line: push rejected"; err.Error()[:len(want)] != want {
		t.Fatalf("error = %q, want it to name the rejection", err)
	}
}

func TestChannelName(t *testing.T) {
	notifier := New("token", nil)
	if got := notifier.Channel(); got != "line" {
		t.Fatalf("channel = %q, want line", got)
	}
}
