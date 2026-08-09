package main

import (
	"io"
	"log"
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

func TestAmendOrder_RejectedByPerItemSCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"0","msg":"","data":[{"sCode":"51503","sMsg":"order not found or completed"}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{APIKey: "k", APISecret: "s", APIPassphrase: "p"}, srv.URL)
	err := AmendOrder(c, "ETH-USD-260810-1700-P", "42", decimal.RequireFromString("0.1095"))
	if err == nil {
		t.Fatalf("expected error for rejected amend, got nil")
	}
}

func TestAmendTracker_AlreadySent(t *testing.T) {
	tr := NewAmendTracker()
	px := decimal.RequireFromString("0.1095")

	if tr.AlreadySent("42", px) {
		t.Fatalf("fresh tracker should not report anything as sent")
	}
	tr.Record("42", px)
	if !tr.AlreadySent("42", px) {
		t.Fatalf("expected recorded price to be reported as already sent")
	}
	// Same numeric value, different scale, must still match.
	if !tr.AlreadySent("42", decimal.RequireFromString("0.10950")) {
		t.Fatalf("expected numerically equal price to match")
	}
	if tr.AlreadySent("42", decimal.RequireFromString("0.11")) {
		t.Fatalf("a different price must not be treated as already sent")
	}
	if tr.AlreadySent("43", px) {
		t.Fatalf("a different ordId must not be treated as already sent")
	}
}

// Fix 6: orders-pending can still report the pre-amend price on the next poll
// while the book already shows our amended order. The tracker must stop the
// duplicate amend.
func TestRunOnce_StaleSnapshot_DoesNotReAmend(t *testing.T) {
	var amendCalls int
	pollNum := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v5/trade/orders-pending":
			// Both polls report the same stale price of 0.10.
			w.Write([]byte(`{"code":"0","msg":"","data":[{"instId":"ETH-USD-260810-1700-P","ordId":"7","side":"sell","optType":"P","px":"0.10","sz":"5","accFillSz":"0"}]}`))
		case "/api/v5/public/instrument-tick-bands":
			w.Write([]byte(`{"code":"0","msg":"","data":[{"instFamily":"ETH-USD","tickBand":[{"maxPx":"0.005","minPx":"0","tickSz":"0.0001"},{"maxPx":"10000000","minPx":"0.005","tickSz":"0.0005"}]}]}`))
		case "/api/v5/market/books":
			// Ask1 is 0.11 in both polls; after the first amend that level is
			// in fact our own (already-amended) order.
			w.Write([]byte(`{"code":"0","msg":"","data":[{"asks":[["0.11","5","0","1"],["0.13","2","0","1"]],"bids":[],"ts":"1","seqId":1}]}`))
		case "/api/v5/public/mark-price":
			w.Write([]byte(`{"code":"0","msg":"","data":[{"instId":"ETH-USD-260810-1700-P","markPx":"0.09","ts":"1"}]}`))
		case "/api/v5/trade/amend-order":
			amendCalls++
			w.Write([]byte(`{"code":"0","msg":"","data":[{"sCode":"0","sMsg":""}]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.Write([]byte(`{"code":"0","msg":"","data":[]}`))
		}
	}))
	defer srv.Close()

	c := NewClient(Config{APIKey: "k", APISecret: "s", APIPassphrase: "p"}, srv.URL)
	cache := NewTickCache()
	tracker := NewAmendTracker()
	cfg := Config{DryRun: false}
	logger := log.New(io.Discard, "", 0)

	for pollNum = 1; pollNum <= 2; pollNum++ {
		if err := runOnce(c, cache, cfg, logger, tracker); err != nil {
			t.Fatalf("poll %d: unexpected error: %v", pollNum, err)
		}
	}

	if amendCalls != 1 {
		t.Fatalf("expected exactly 1 amend across two polls, got %d", amendCalls)
	}
}

// Dry-run must never populate the tracker, since nothing was actually sent.
func TestRunOnce_DryRun_DoesNotRecordAmends(t *testing.T) {
	var amendCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v5/trade/orders-pending":
			w.Write([]byte(`{"code":"0","msg":"","data":[{"instId":"ETH-USD-260810-1700-P","ordId":"7","side":"sell","optType":"P","px":"0.10","sz":"5","accFillSz":"0"}]}`))
		case "/api/v5/public/instrument-tick-bands":
			w.Write([]byte(`{"code":"0","msg":"","data":[{"instFamily":"ETH-USD","tickBand":[{"maxPx":"10000000","minPx":"0","tickSz":"0.0005"}]}]}`))
		case "/api/v5/market/books":
			w.Write([]byte(`{"code":"0","msg":"","data":[{"asks":[["0.11","5","0","1"]],"bids":[],"ts":"1","seqId":1}]}`))
		case "/api/v5/public/mark-price":
			w.Write([]byte(`{"code":"0","msg":"","data":[{"instId":"ETH-USD-260810-1700-P","markPx":"0.09","ts":"1"}]}`))
		case "/api/v5/trade/amend-order":
			amendCalls++
			w.Write([]byte(`{"code":"0","msg":"","data":[{"sCode":"0","sMsg":""}]}`))
		}
	}))
	defer srv.Close()

	c := NewClient(Config{APIKey: "k", APISecret: "s", APIPassphrase: "p"}, srv.URL)
	tracker := NewAmendTracker()
	logger := log.New(io.Discard, "", 0)

	if err := runOnce(c, NewTickCache(), Config{DryRun: true}, logger, tracker); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if amendCalls != 0 {
		t.Fatalf("dry-run must not send amends, got %d", amendCalls)
	}
	if tracker.AlreadySent("7", decimal.RequireFromString("0.11")) {
		t.Fatalf("dry-run must not record anything in the tracker")
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
