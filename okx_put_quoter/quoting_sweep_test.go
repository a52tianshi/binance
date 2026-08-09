package main

import (
	"math/rand"
	"testing"

	"github.com/shopspring/decimal"
)

func TestSweepTickAlignment(t *testing.T) {
	bands := ethBands()
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 200000; i++ {
		mark := decimal.NewFromFloat(r.Float64() * 2).Round(16)
		in := QuoteInput{
			OurPx: decimal.NewFromInt(99), OurSz: d("5"), MarkPx: mark, Bands: bands,
			Ask1: &BookLevel{Px: decimal.Zero, Sz: d("3"), NumOrders: 1},
		}
		got := DecideNewPrice(in)
		if !got.ShouldAmend {
			t.Fatalf("mark %v: expected amend", mark)
		}
		tick, _ := FindTickSize(bands, got.NewPx)
		if !got.NewPx.Mod(tick).IsZero() {
			t.Fatalf("mark %v -> %v not multiple of %v", mark, got.NewPx, tick)
		}
		if !got.NewPx.GreaterThan(mark) {
			t.Fatalf("mark %v -> %v not strictly above mark", mark, got.NewPx)
		}
		if got.NewPx.Sub(mark).GreaterThan(tick.Mul(decimal.NewFromInt(2))) {
			t.Fatalf("mark %v -> %v overshoots", mark, got.NewPx)
		}
	}
}

func TestSweepTightenAlignment(t *testing.T) {
	bands := ethBands()
	r := rand.New(rand.NewSource(2))
	for i := 0; i < 200000; i++ {
		ask1 := decimal.NewFromFloat(r.Float64() * 0.5).Round(16)
		ask2 := ask1.Add(decimal.NewFromFloat(r.Float64() * 0.5).Round(16))
		mark := decimal.NewFromFloat(r.Float64() * 2).Round(16)
		in := QuoteInput{
			OurPx: ask1, OurSz: d("5"), MarkPx: mark, Bands: bands,
			Ask1: &BookLevel{Px: ask1, Sz: d("5"), NumOrders: 1},
			Ask2: &BookLevel{Px: ask2, Sz: d("2"), NumOrders: 1},
		}
		got := DecideNewPrice(in)
		if !got.ShouldAmend {
			continue
		}
		tick, _ := FindTickSize(bands, got.NewPx)
		if !got.NewPx.Mod(tick).IsZero() {
			t.Fatalf("ask1=%v ask2=%v -> %v not multiple of %v", ask1, ask2, got.NewPx, tick)
		}
		tickAtMark, _ := FindTickSize(bands, mark)
		capPx := mark.Add(tickAtMark.Mul(decimal.NewFromInt(TightenCapTicksAboveMark)))
		if got.NewPx.GreaterThan(capPx) {
			t.Fatalf("ask1=%v ask2=%v mark=%v -> %v exceeds cap %v", ask1, ask2, mark, got.NewPx, capPx)
		}
		if got.NewPx.GreaterThanOrEqual(ask2) {
			t.Fatalf("ask1=%v ask2=%v -> %v is not below ask2", ask1, ask2, got.NewPx)
		}
	}
}
