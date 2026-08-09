package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
)

func TestFetchTickBands_ParsesLiveShapedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"0","msg":"","data":[{"instFamily":"ETH-USD","instType":"OPTION","seriesId":"","tickBand":[{"maxPx":"0.005","minPx":"0","tickSz":"0.0001"},{"maxPx":"10000000","minPx":"0.005","tickSz":"0.0005"}]}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	bands, err := FetchTickBands(c, "ETH-USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bands) != 2 {
		t.Fatalf("expected 2 bands, got %d", len(bands))
	}
	if !bands[0].TickSz.Equal(decimal.RequireFromString("0.0001")) {
		t.Fatalf("unexpected tickSz for band 0: %v", bands[0].TickSz)
	}
}

func TestFindTickSize_PicksBandByMaxPx(t *testing.T) {
	bands := []TickBand{
		{MinPx: decimal.Zero, MaxPx: decimal.RequireFromString("0.005"), TickSz: decimal.RequireFromString("0.0001")},
		{MinPx: decimal.RequireFromString("0.005"), MaxPx: decimal.RequireFromString("10000000"), TickSz: decimal.RequireFromString("0.0005")},
	}

	got, err := FindTickSize(bands, decimal.RequireFromString("0.003"))
	if err != nil || !got.Equal(decimal.RequireFromString("0.0001")) {
		t.Fatalf("expected 0.0001 tick for low price, got %v err %v", got, err)
	}

	got, err = FindTickSize(bands, decimal.RequireFromString("1.5"))
	if err != nil || !got.Equal(decimal.RequireFromString("0.0005")) {
		t.Fatalf("expected 0.0005 tick for high price, got %v err %v", got, err)
	}
}

func TestFetchOrderBook_HandlesEmptyAsks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"0","msg":"","data":[{"asks":[],"bids":[["0.1055","5","0","1"]],"ts":"1","seqId":1}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	book, err := FetchOrderBook(c, "ETH-USD-260810-1700-C")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if book.Ask1 != nil {
		t.Fatalf("expected nil Ask1 for empty asks, got %+v", book.Ask1)
	}
}

func TestFetchOrderBook_ParsesTwoAsks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"0","msg":"","data":[{"asks":[["0.10","3","0","1"],["0.12","2","0","1"]],"bids":[],"ts":"1","seqId":1}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	book, err := FetchOrderBook(c, "X")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if book.Ask1 == nil || !book.Ask1.Px.Equal(decimal.RequireFromString("0.10")) {
		t.Fatalf("unexpected ask1: %+v", book.Ask1)
	}
	if book.Ask2 == nil || !book.Ask2.Px.Equal(decimal.RequireFromString("0.12")) {
		t.Fatalf("unexpected ask2: %+v", book.Ask2)
	}
}

func TestFetchOrderBook_ParsesNumOrders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"0","msg":"","data":[{"asks":[["0.10","3","0","1"],["0.12","7","0","4"]],"bids":[],"ts":"1","seqId":1}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	book, err := FetchOrderBook(c, "X")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if book.Ask1 == nil || book.Ask1.NumOrders != 1 {
		t.Fatalf("expected ask1 numOrders 1, got %+v", book.Ask1)
	}
	if book.Ask2 == nil || book.Ask2.NumOrders != 4 {
		t.Fatalf("expected ask2 numOrders 4, got %+v", book.Ask2)
	}
}

func TestFetchOrderBook_BadNumOrders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"0","msg":"","data":[{"asks":[["0.10","3","0","abc"]],"bids":[],"ts":"1","seqId":1}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	if _, err := FetchOrderBook(c, "X"); err == nil {
		t.Fatalf("expected error for unparseable numOrders")
	}
}

func TestFetchMarkPrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"0","msg":"","data":[{"instId":"X","instType":"OPTION","markPx":"0.1144782725373311","ts":"1"}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	px, err := FetchMarkPrice(c, "X")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !px.Equal(decimal.RequireFromString("0.1144782725373311")) {
		t.Fatalf("unexpected markPx: %v", px)
	}
}

func TestTickCache_FetchesOncePerInstFamily(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{"code":"0","msg":"","data":[{"instFamily":"ETH-USD","tickBand":[{"maxPx":"10000000","minPx":"0","tickSz":"0.0005"}]}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	cache := NewTickCache()
	if _, err := cache.Get(c, "ETH-USD"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := cache.Get(c, "ETH-USD"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 HTTP call, got %d", calls)
	}
}
