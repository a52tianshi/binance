package main

import (
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestDecideNewPrice_NoAsk1_NoAction(t *testing.T) {
	in := QuoteInput{OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), TickSz: d("0.0005")}
	got := DecideNewPrice(in)
	if got.ShouldAmend {
		t.Fatalf("expected no amend when there is no ask1, got %+v", got)
	}
}

func TestDecideNewPrice_NotOnAsk1_Ask1AboveMark_JoinsAsk1(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), TickSz: d("0.0005"),
		Ask1: &BookLevel{Px: d("0.11"), Sz: d("3")},
	}
	got := DecideNewPrice(in)
	if !got.ShouldAmend || !got.NewPx.Equal(d("0.11")) || got.Reason != ReasonJoinAsk1 {
		t.Fatalf("expected join-ask1 to 0.11, got %+v", got)
	}
}

func TestDecideNewPrice_NotOnAsk1_Ask1BelowMark_FallsBackToMarkPlusTick(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), TickSz: d("0.0005"),
		Ask1: &BookLevel{Px: d("0.08"), Sz: d("3")},
	}
	got := DecideNewPrice(in)
	want := d("0.0905") // markPx + tickSz
	if !got.ShouldAmend || !got.NewPx.Equal(want) || got.Reason != ReasonMarkFallback {
		t.Fatalf("expected mark+tick fallback to %v, got %+v", want, got)
	}
}

func TestDecideNewPrice_OnAsk1_Solo_WideGap_Tightens(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), TickSz: d("0.0005"),
		Ask1: &BookLevel{Px: d("0.10"), Sz: d("5")}, // sz matches ourSz => solo
		Ask2: &BookLevel{Px: d("0.12"), Sz: d("2")},
	}
	got := DecideNewPrice(in)
	want := d("0.1195") // ask2 - tick
	if !got.ShouldAmend || !got.NewPx.Equal(want) || got.Reason != ReasonTightenAsk1 {
		t.Fatalf("expected tighten to %v, got %+v", want, got)
	}
}

func TestDecideNewPrice_OnAsk1_Solo_NarrowGap_NoAction(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), TickSz: d("0.0005"),
		Ask1: &BookLevel{Px: d("0.10"), Sz: d("5")},
		Ask2: &BookLevel{Px: d("0.1005"), Sz: d("2")}, // exactly 1 tick gap
	}
	got := DecideNewPrice(in)
	if got.ShouldAmend {
		t.Fatalf("expected no amend when gap == 1 tick, got %+v", got)
	}
}

func TestDecideNewPrice_OnAsk1_NotSolo_NoAction(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), TickSz: d("0.0005"),
		Ask1: &BookLevel{Px: d("0.10"), Sz: d("8")}, // more size than ours => others present
		Ask2: &BookLevel{Px: d("0.12"), Sz: d("2")},
	}
	got := DecideNewPrice(in)
	if got.ShouldAmend {
		t.Fatalf("expected no amend when not solo at ask1, got %+v", got)
	}
}

func TestDecideNewPrice_OnAsk1_Solo_NoAsk2_NoAction(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), TickSz: d("0.0005"),
		Ask1: &BookLevel{Px: d("0.10"), Sz: d("5")},
	}
	got := DecideNewPrice(in)
	if got.ShouldAmend {
		t.Fatalf("expected no amend when ask2 is missing, got %+v", got)
	}
}
