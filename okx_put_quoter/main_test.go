package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
)

func TestInstFamilyFor(t *testing.T) {
	got := instFamilyFor("ETH-USD-260810-1700-P")
	if got != "ETH-USD" {
		t.Fatalf("expected ETH-USD, got %s", got)
	}
}

func TestAmendOrder_SendsExpectedBody(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		gotBody = string(buf)
		w.Write([]byte(`{"code":"0","msg":"","data":[{"sCode":"0","sMsg":""}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{APIKey: "k", APISecret: "s", APIPassphrase: "p"}, srv.URL)
	err := AmendOrder(c, "ETH-USD-260810-1700-P", "42", decimal.RequireFromString("0.1095"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{`"instId":"ETH-USD-260810-1700-P"`, `"ordId":"42"`, `"newPx":"0.1095"`} {
		if !contains(gotBody, want) {
			t.Fatalf("expected body to contain %s, got %s", want, gotBody)
		}
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
