package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
)

func TestFetchOpenEthPutSellOrders_FiltersCorrectly(t *testing.T) {
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
	orders, err := FetchOpenEthPutSellOrders(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 matching order, got %d: %+v", len(orders), orders)
	}
	if orders[0].OrdId != "1" {
		t.Fatalf("unexpected order matched: %+v", orders[0])
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
