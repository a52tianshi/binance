package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDoPrivate_SignsRequestCorrectly(t *testing.T) {
	cfg := Config{APIKey: "key123", APISecret: "secret123", APIPassphrase: "pass123"}

	var gotKey, gotSign, gotTs, gotPass, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("OK-ACCESS-KEY")
		gotSign = r.Header.Get("OK-ACCESS-SIGN")
		gotTs = r.Header.Get("OK-ACCESS-TIMESTAMP")
		gotPass = r.Header.Get("OK-ACCESS-PASSPHRASE")
		gotMethod = r.Method
		w.Write([]byte(`{"code":"0","msg":"","data":[{"ok":true}]}`))
	}))
	defer srv.Close()

	c := NewClient(cfg, srv.URL)
	var out []map[string]bool
	err := c.DoPrivate("GET", "/api/v5/trade/orders-pending", url.Values{"instType": {"OPTION"}}, nil, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 || !out[0]["ok"] {
		t.Fatalf("unexpected decoded data: %+v", out)
	}
	if gotKey != "key123" || gotPass != "pass123" || gotMethod != "GET" {
		t.Fatalf("unexpected headers: key=%s pass=%s method=%s", gotKey, gotPass, gotMethod)
	}
	if gotTs == "" || gotSign == "" {
		t.Fatalf("expected timestamp and signature headers to be set")
	}

	// Recompute expected signature the same way OKX requires and check it matches.
	requestPath := "/api/v5/trade/orders-pending?instType=OPTION"
	preHash := gotTs + "GET" + requestPath
	mac := hmac.New(sha256.New, []byte(cfg.APISecret))
	mac.Write([]byte(preHash))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if gotSign != want {
		t.Fatalf("signature mismatch: got %s want %s", gotSign, want)
	}
}

func TestDoPublic_NoAuthHeaders(t *testing.T) {
	cfg := Config{APIKey: "key123", APISecret: "secret123", APIPassphrase: "pass123"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("OK-ACCESS-KEY") != "" {
			t.Errorf("public endpoint should not send auth headers")
		}
		w.Write([]byte(`{"code":"0","msg":"","data":[{"markPx":"1.23"}]}`))
	}))
	defer srv.Close()

	c := NewClient(cfg, srv.URL)
	var out []map[string]string
	if err := c.DoPublic("GET", "/api/v5/public/mark-price", url.Values{"instType": {"OPTION"}}, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out[0]["markPx"] != "1.23" {
		t.Fatalf("unexpected data: %+v", out)
	}
}

func TestDoPrivate_APIErrorReturned(t *testing.T) {
	cfg := Config{APIKey: "k", APISecret: "s", APIPassphrase: "p"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"50001","msg":"boom","data":[]}`))
	}))
	defer srv.Close()

	c := NewClient(cfg, srv.URL)
	var out json.RawMessage
	err := c.DoPrivate("GET", "/api/v5/trade/orders-pending", nil, nil, &out)
	if err == nil {
		t.Fatalf("expected error for non-zero code")
	}
}
