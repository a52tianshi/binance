package main

import (
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// ethBands is the live-verified ETH-USD option tick-band table.
func ethBands() []TickBand {
	return []TickBand{
		{MinPx: decimal.Zero, MaxPx: d("0.005"), TickSz: d("0.0001")},
		{MinPx: d("0.005"), MaxPx: d("10000000"), TickSz: d("0.0005")},
	}
}

func TestDecideNewPrice_NoAsk1_NoAction(t *testing.T) {
	in := QuoteInput{OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), Bands: ethBands()}
	got := DecideNewPrice(in)
	if got.ShouldAmend {
		t.Fatalf("expected no amend when there is no ask1, got %+v", got)
	}
}

func TestDecideNewPrice_NotOnAsk1_Ask1AboveMark_JoinsAsk1(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), Bands: ethBands(),
		Ask1: &BookLevel{Px: d("0.11"), Sz: d("3"), NumOrders: 1},
	}
	got := DecideNewPrice(in)
	if !got.ShouldAmend || !got.NewPx.Equal(d("0.11")) || got.Reason != ReasonJoinAsk1 {
		t.Fatalf("expected join-ask1 to 0.11, got %+v", got)
	}
}

func TestDecideNewPrice_NotOnAsk1_Ask1BelowMark_FallsBackToMarkPlusTick(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), Bands: ethBands(),
		Ask1: &BookLevel{Px: d("0.08"), Sz: d("3"), NumOrders: 1},
	}
	got := DecideNewPrice(in)
	want := d("0.0905") // markPx + tickSz
	if !got.ShouldAmend || !got.NewPx.Equal(want) || got.Reason != ReasonMarkFallback {
		t.Fatalf("expected mark+tick fallback to %v, got %+v", want, got)
	}
}

func TestDecideNewPrice_OnAsk1_Solo_WideGap_Tightens(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), Bands: ethBands(),
		Ask1: &BookLevel{Px: d("0.10"), Sz: d("5"), NumOrders: 1}, // solo
		Ask2: &BookLevel{Px: d("0.11"), Sz: d("2"), NumOrders: 1},
	}
	got := DecideNewPrice(in)
	want := d("0.1095") // ask2 - tick, within the mark+50-tick cap of 0.115
	if !got.ShouldAmend || !got.NewPx.Equal(want) || got.Reason != ReasonTightenAsk1 {
		t.Fatalf("expected tighten to %v, got %+v", want, got)
	}
}

func TestDecideNewPrice_OnAsk1_Solo_NarrowGap_NoAction(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), Bands: ethBands(),
		Ask1: &BookLevel{Px: d("0.10"), Sz: d("5"), NumOrders: 1},
		Ask2: &BookLevel{Px: d("0.1005"), Sz: d("2"), NumOrders: 1}, // exactly 1 tick gap
	}
	got := DecideNewPrice(in)
	if got.ShouldAmend {
		t.Fatalf("expected no amend when gap == 1 tick, got %+v", got)
	}
}

func TestDecideNewPrice_OnAsk1_NotSolo_NoAction(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), Bands: ethBands(),
		Ask1: &BookLevel{Px: d("0.10"), Sz: d("8"), NumOrders: 2}, // more size than ours
		Ask2: &BookLevel{Px: d("0.11"), Sz: d("2"), NumOrders: 1},
	}
	got := DecideNewPrice(in)
	if got.ShouldAmend {
		t.Fatalf("expected no amend when not solo at ask1, got %+v", got)
	}
}

// Fix 4: numOrders is the authoritative solo signal; a size coincidence must
// not be enough.
func TestDecideNewPrice_OnAsk1_SizeMatchesButMultipleOrders_NoAction(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), Bands: ethBands(),
		Ask1: &BookLevel{Px: d("0.10"), Sz: d("5"), NumOrders: 2}, // size matches by coincidence
		Ask2: &BookLevel{Px: d("0.11"), Sz: d("2"), NumOrders: 1},
	}
	got := DecideNewPrice(in)
	if got.ShouldAmend {
		t.Fatalf("expected no amend when ask1 holds 2 orders, got %+v", got)
	}
}

func TestDecideNewPrice_OnAsk1_Solo_NoAsk2_NoAction(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), Bands: ethBands(),
		Ask1: &BookLevel{Px: d("0.10"), Sz: d("5"), NumOrders: 1},
	}
	got := DecideNewPrice(in)
	if got.ShouldAmend {
		t.Fatalf("expected no amend when ask2 is missing, got %+v", got)
	}
}

// Fix 1: every returned price must be an exact multiple of the tick size for
// the band it lands in, otherwise OKX rejects the amend forever.
func TestDecideNewPrice_AlwaysReturnsTickAlignedPrices(t *testing.T) {
	bands := ethBands()

	cases := []struct {
		name string
		in   QuoteInput
	}{
		{
			// The exact case from the plan discussion: ragged markPx must NOT
			// yield 0.1149782725373311.
			name: "ragged mark fallback",
			in: QuoteInput{
				OurPx: d("0.20"), OurSz: d("5"), MarkPx: d("0.1144782725373311"), Bands: bands,
				Ask1: &BookLevel{Px: d("0.05"), Sz: d("3"), NumOrders: 1},
			},
		},
		{
			name: "ragged mark fallback 2",
			in: QuoteInput{
				OurPx: d("0.20"), OurSz: d("5"), MarkPx: d("0.0733333333333333"), Bands: bands,
				Ask1: &BookLevel{Px: d("0.05"), Sz: d("3"), NumOrders: 1},
			},
		},
		{
			name: "ragged mark fallback in low band",
			in: QuoteInput{
				OurPx: d("0.20"), OurSz: d("5"), MarkPx: d("0.0031415926535897"), Bands: bands,
				Ask1: &BookLevel{Px: d("0.001"), Sz: d("3"), NumOrders: 1},
			},
		},
		{
			name: "mark exactly on band boundary",
			in: QuoteInput{
				OurPx: d("0.20"), OurSz: d("5"), MarkPx: d("0.005"), Bands: bands,
				Ask1: &BookLevel{Px: d("0.001"), Sz: d("3"), NumOrders: 1},
			},
		},
		{
			name: "mark exactly on a tick",
			in: QuoteInput{
				OurPx: d("0.20"), OurSz: d("5"), MarkPx: d("0.09"), Bands: bands,
				Ask1: &BookLevel{Px: d("0.05"), Sz: d("3"), NumOrders: 1},
			},
		},
		{
			name: "tighten with ragged ask2 far side",
			in: QuoteInput{
				OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.1044782725373311"), Bands: bands,
				Ask1: &BookLevel{Px: d("0.10"), Sz: d("5"), NumOrders: 1},
				Ask2: &BookLevel{Px: d("0.1075"), Sz: d("2"), NumOrders: 1},
			},
		},
		{
			// Cross-band tighten: ask1 sits in the 0.0001 band, ask2 in the
			// 0.0005 band.
			name: "tighten across band boundary",
			in: QuoteInput{
				OurPx: d("0.0045"), OurSz: d("5"), MarkPx: d("0.0045"), Bands: bands,
				Ask1: &BookLevel{Px: d("0.0045"), Sz: d("5"), NumOrders: 1},
				Ask2: &BookLevel{Px: d("0.0060"), Sz: d("2"), NumOrders: 1},
			},
		},
		{
			name: "tighten clamped by mark cap",
			in: QuoteInput{
				OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.0912345678901234"), Bands: bands,
				Ask1: &BookLevel{Px: d("0.10"), Sz: d("5"), NumOrders: 1},
				Ask2: &BookLevel{Px: d("5.0"), Sz: d("2"), NumOrders: 1},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DecideNewPrice(tc.in)
			if !got.ShouldAmend {
				t.Fatalf("expected an amend for this case, got %+v", got)
			}
			tick, err := FindTickSize(bands, got.NewPx)
			if err != nil {
				t.Fatalf("find tick size for %v: %v", got.NewPx, err)
			}
			if !got.NewPx.Mod(tick).IsZero() {
				t.Fatalf("price %v is not a multiple of tick %v (reason=%s)", got.NewPx, tick, got.Reason)
			}
		})
	}
}

func TestDecideNewPrice_RaggedMark_DoesNotEmitRawMarkPlusTick(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.20"), OurSz: d("5"), MarkPx: d("0.1144782725373311"), Bands: ethBands(),
		Ask1: &BookLevel{Px: d("0.05"), Sz: d("3"), NumOrders: 1},
	}
	got := DecideNewPrice(in)
	if !got.ShouldAmend {
		t.Fatalf("expected an amend, got %+v", got)
	}
	if got.NewPx.Equal(d("0.1149782725373311")) {
		t.Fatalf("returned the un-quantized markPx+tick price: %v", got.NewPx)
	}
	if !got.NewPx.Equal(d("0.1145")) {
		t.Fatalf("expected 0.1145 (smallest 0.0005 multiple above mark), got %v", got.NewPx)
	}
}

// Fix 1 cross-band tighten: the returned price must be aligned to the band it
// lands in (0.0005), not blindly to the ask1-side tick (0.0001).
func TestDecideNewPrice_TightenAcrossBandBoundary(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.0045"), OurSz: d("5"), MarkPx: d("0.0045"), Bands: ethBands(),
		Ask1: &BookLevel{Px: d("0.0045"), Sz: d("5"), NumOrders: 1},
		Ask2: &BookLevel{Px: d("0.0060"), Sz: d("2"), NumOrders: 1},
	}
	got := DecideNewPrice(in)
	if !got.ShouldAmend || got.Reason != ReasonTightenAsk1 {
		t.Fatalf("expected a tighten decision, got %+v", got)
	}
	if !got.NewPx.Equal(d("0.0055")) {
		t.Fatalf("expected 0.0055 (ask2 - 0.0005, aligned to the high band), got %v", got.NewPx)
	}
}

// Fix 2: a computed price identical to our current price must not re-amend.
func TestDecideNewPrice_Fallback_SamePriceAsOurs_NoAction(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.0905"), OurSz: d("5"), MarkPx: d("0.09"), Bands: ethBands(),
		Ask1: &BookLevel{Px: d("0.08"), Sz: d("3"), NumOrders: 1},
	}
	got := DecideNewPrice(in)
	if got.ShouldAmend {
		t.Fatalf("expected no amend when computed price equals our price, got %+v", got)
	}
}

// Fix 3: the tighten rule must not chase a junk ask2 arbitrarily far above mark.
func TestDecideNewPrice_Tighten_CappedRelativeToMark(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.10"), OurSz: d("5"), MarkPx: d("0.09"), Bands: ethBands(),
		Ask1: &BookLevel{Px: d("0.10"), Sz: d("5"), NumOrders: 1},
		Ask2: &BookLevel{Px: d("5.0"), Sz: d("2"), NumOrders: 1}, // stale junk quote
	}
	got := DecideNewPrice(in)
	if !got.ShouldAmend || got.Reason != ReasonTightenAsk1 {
		t.Fatalf("expected a tighten decision, got %+v", got)
	}
	capPx := d("0.115") // 0.09 + 50 * 0.0005
	if got.NewPx.GreaterThan(capPx) {
		t.Fatalf("price %v exceeds the mark+%d-tick cap of %v", got.NewPx, TightenCapTicksAboveMark, capPx)
	}
	if !got.NewPx.Equal(capPx) {
		t.Fatalf("expected the capped price %v, got %v", capPx, got.NewPx)
	}
}

// If the cap lands at or below where we already sit, do nothing.
func TestDecideNewPrice_Tighten_CapBelowOurPrice_NoAction(t *testing.T) {
	in := QuoteInput{
		OurPx: d("0.20"), OurSz: d("5"), MarkPx: d("0.09"), Bands: ethBands(),
		Ask1: &BookLevel{Px: d("0.20"), Sz: d("5"), NumOrders: 1},
		Ask2: &BookLevel{Px: d("5.0"), Sz: d("2"), NumOrders: 1},
	}
	got := DecideNewPrice(in)
	if got.ShouldAmend {
		t.Fatalf("expected no amend when the cap is below our price, got %+v", got)
	}
}
