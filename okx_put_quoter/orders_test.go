package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func discardLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func TestFetchOpenPutSellOrders_FiltersCorrectly(t *testing.T) {
	body := `{"code":"0","msg":"","data":[
		{"instId":"ETH-USD-260810-1700-P","ordId":"1","side":"sell","optType":"P","px":"0.10","sz":"5","accFillSz":"0"},
		{"instId":"ETH-USD-260810-1700-C","ordId":"2","side":"sell","optType":"C","px":"0.20","sz":"5","accFillSz":"0"},
		{"instId":"ETH-USD-260810-1700-P","ordId":"3","side":"buy","optType":"P","px":"0.05","sz":"5","accFillSz":"0"},
		{"instId":"BTC-USD-260810-30000-P","ordId":"4","side":"sell","optType":"P","px":"1.0","sz":"1","accFillSz":"0"}
	]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	orders, err := FetchOpenPutSellOrders(c, discardLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Order 1 (ETH put sell) and order 4 (BTC put sell) both match now that
	// the underlying restriction is gone; order 2 (call) and order 3 (buy)
	// are still filtered out.
	if len(orders) != 2 {
		t.Fatalf("expected 2 matching orders, got %d: %+v", len(orders), orders)
	}
	gotIds := map[string]bool{orders[0].OrdId: true, orders[1].OrdId: true}
	if !gotIds["1"] || !gotIds["4"] {
		t.Fatalf("expected orders 1 (ETH) and 4 (BTC) to match, got: %+v", orders)
	}
}

func TestFetchOpenPutSellOrders_DerivesOptTypeFromInstIdWhenFieldIsEmpty(t *testing.T) {
	// Reproduces a real OKX response: instType=OPTION rows can come back with
	// optType=="" even though the instId itself encodes C/P in its last
	// segment. The filter must not silently drop every row in that case.
	body := `{"code":"0","msg":"","data":[
		{"instId":"ETH-USD-260810-1700-P","ordId":"1","side":"sell","optType":"","px":"0.10","sz":"5","accFillSz":"0"},
		{"instId":"ETH-USD-260810-1700-C","ordId":"2","side":"sell","optType":"","px":"0.20","sz":"5","accFillSz":"0"},
		{"instId":"BTC-USD-260810-30000-P","ordId":"3","side":"buy","optType":"","px":"1.0","sz":"1","accFillSz":"0"}
	]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	orders, err := FetchOpenPutSellOrders(c, discardLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 matching order (ETH put sell), got %d: %+v", len(orders), orders)
	}
	if orders[0].OrdId != "1" {
		t.Fatalf("expected order 1 to match, got: %+v", orders[0])
	}
}

func TestOptTypeFromInstId(t *testing.T) {
	cases := map[string]string{
		"ETH-USD-260810-1700-P":   "P",
		"ETH-USD-260810-1700-C":   "C",
		"BTC-USD-260810-30000-P":  "P",
		"malformed-no-dash-here-": "",
		"":                        "",
	}
	for instId, want := range cases {
		if got := optTypeFromInstId(instId); got != want {
			t.Errorf("optTypeFromInstId(%q) = %q, want %q", instId, got, want)
		}
	}
}

func TestFetchOpenPutSellOrders_Paginates(t *testing.T) {
	var afterParams []string
	calls := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		afterParams = append(afterParams, r.URL.Query().Get("after"))
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Errorf("expected limit=100, got %q", got)
		}

		var rows []string
		if calls == 1 {
			// A full page of 100 raw rows; all of them match the filter.
			for i := 0; i < ordersPendingPageSize; i++ {
				rows = append(rows, fmt.Sprintf(
					`{"instId":"ETH-USD-260810-1700-P","ordId":"%d","side":"sell","optType":"P","px":"0.10","sz":"5","accFillSz":"0"}`, i+1))
			}
		} else {
			rows = append(rows,
				`{"instId":"ETH-USD-260810-1800-P","ordId":"101","side":"sell","optType":"P","px":"0.11","sz":"5","accFillSz":"0"}`)
		}
		w.Write([]byte(`{"code":"0","msg":"","data":[` + strings.Join(rows, ",") + `]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	orders, err := FetchOpenPutSellOrders(c, discardLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 paginated calls, got %d", calls)
	}
	if afterParams[0] != "" {
		t.Fatalf("expected no after param on the first call, got %q", afterParams[0])
	}
	if afterParams[1] != "100" {
		t.Fatalf("expected after=100 (last ordId of page 1) on the second call, got %q", afterParams[1])
	}
	if len(orders) != ordersPendingPageSize+1 {
		t.Fatalf("expected %d orders across both pages, got %d", ordersPendingPageSize+1, len(orders))
	}
	if orders[len(orders)-1].OrdId != "101" {
		t.Fatalf("expected the page-2 order to be included, got %+v", orders[len(orders)-1])
	}
}

func TestFetchOpenPutSellOrders_StopsOnEmptyPage(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var rows []string
		if calls == 1 {
			// A full page of non-matching rows (buy side), so this test
			// exercises the pagination-stop mechanics independently of the
			// side/optType filter.
			for i := 0; i < ordersPendingPageSize; i++ {
				rows = append(rows, fmt.Sprintf(
					`{"instId":"BTC-USD-260810-30000-P","ordId":"%d","side":"buy","optType":"P","px":"1","sz":"1","accFillSz":"0"}`, i+1))
			}
		}
		w.Write([]byte(`{"code":"0","msg":"","data":[` + strings.Join(rows, ",") + `]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	orders, err := FetchOpenPutSellOrders(c, discardLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected the loop to stop after the empty second page, got %d calls", calls)
	}
	if len(orders) != 0 {
		t.Fatalf("expected no matching put sells, got %d", len(orders))
	}
}

func TestPendingOrder_RemainingSz(t *testing.T) {
	o := PendingOrder{Sz: decimal.RequireFromString("5"), AccFillSz: decimal.RequireFromString("2")}
	if !o.RemainingSz().Equal(decimal.RequireFromString("3")) {
		t.Fatalf("expected remaining 3, got %v", o.RemainingSz())
	}
}

func TestGroupByInstId(t *testing.T) {
	orders := []PendingOrder{
		{InstId: "A", OrdId: "1"},
		{InstId: "A", OrdId: "2"},
		{InstId: "B", OrdId: "3"},
	}
	grouped := GroupByInstId(orders)
	if len(grouped["A"]) != 2 || len(grouped["B"]) != 1 {
		t.Fatalf("unexpected grouping: %+v", grouped)
	}
}
