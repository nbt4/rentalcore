package jev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestChooseValidatesAndReturnsDecision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header = %q", r.Header.Get("Authorization"))
		}
		var request decisionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != "typesafe/jev-1.13" || request.Questions["match"].Type != "choice" {
			t.Fatalf("unexpected request: %#v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"typesafe/jev-1.13-20260917","answers":{"match":{"type":"choice","choice":"product_42","confidence":0.93,"probabilities":{"product_42":0.96,"no_match":0.04}}}}`))
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "test-key", Endpoint: server.URL, Model: "typesafe/jev-1.13", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := client.Choose(context.Background(), map[string]string{"line": "QLXD"}, "Select the same product", map[string]string{
		"product_42": "Shure QLXD", "no_match": "No candidate matches",
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Choice != "product_42" || decision.Confidence != 0.93 || decision.Model == "" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}

func TestChooseRejectsUnknownChoice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"answers":{"match":{"type":"choice","choice":"invented","confidence":0.9}}}`))
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "test-key", Endpoint: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Choose(context.Background(), "state", "choose", map[string]string{"a": "A", "b": "B"})
	if err == nil {
		t.Fatal("expected unknown choice error")
	}
}

func TestChooseRejectsOversizedRequestBeforeNetwork(t *testing.T) {
	client, err := New(Config{APIKey: "test-key", Endpoint: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Choose(context.Background(), strings.Repeat("x", maxRequestSize), "choose", map[string]string{"a": "A", "b": "B"})
	if err == nil || !strings.Contains(err.Error(), "size limit") {
		t.Fatalf("Choose() error = %v, want request size error", err)
	}
}

func TestFromEnvCanBeDisabled(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	t.Setenv("JEV_ENABLED", "false")
	if client := FromEnv(); client != nil {
		t.Fatal("FromEnv() returned a client while disabled")
	}
}

func TestChooseUsesShortCircuitAfterFailure(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "test-key", Endpoint: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	criteria := map[string]string{"a": "A", "b": "B"}
	_, _ = client.Choose(context.Background(), "state", "choose", criteria)
	_, _ = client.Choose(context.Background(), "state", "choose", criteria)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("network calls = %d, want 1 during cooldown", got)
	}
}
