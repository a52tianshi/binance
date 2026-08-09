package main

import "github.com/shopspring/decimal"

type Reason string

const (
	ReasonNone         Reason = ""
	ReasonJoinAsk1     Reason = "join_ask1"
	ReasonTightenAsk1  Reason = "tighten_ask1"
	ReasonMarkFallback Reason = "mark_fallback"
)

// TightenCapTicksAboveMark bounds how far above mark price the tighten rule is
// allowed to push our quote. Without it, a stale/junk ask2 would drag our order
// to a technically-best-but-unfillable price. Tune here.
const TightenCapTicksAboveMark = 50

// tickDivPrecision is the scale used when dividing a price by a tick size to
// find how many whole ticks it contains. Prices carry at most ~16 decimals and
// tick sizes are >= 1e-4, so 16 digits leaves ample headroom for Ceil/Floor to
// land on the correct integer.
const tickDivPrecision = 16

// ceilToTick returns the smallest multiple of tick that is >= px.
func ceilToTick(px, tick decimal.Decimal) decimal.Decimal {
	if tick.IsZero() {
		return px
	}
	return px.DivRound(tick, tickDivPrecision).Ceil().Mul(tick)
}

// floorToTick returns the largest multiple of tick that is <= px.
func floorToTick(px, tick decimal.Decimal) decimal.Decimal {
	if tick.IsZero() {
		return px
	}
	return px.DivRound(tick, tickDivPrecision).Floor().Mul(tick)
}

type QuoteInput struct {
	OurPx  decimal.Decimal
	OurSz  decimal.Decimal
	MarkPx decimal.Decimal
	// Bands is the full tick-band table for the instrument. Tick size is
	// resolved per-candidate-price, because a computed price can land in a
	// different band than the price it was derived from.
	Bands []TickBand
	Ask1  *BookLevel
	Ask2  *BookLevel
}

type Decision struct {
	ShouldAmend bool
	NewPx       decimal.Decimal
	Reason      Reason
}

func DecideNewPrice(in QuoteInput) Decision {
	if in.Ask1 == nil {
		return Decision{}
	}

	if in.OurPx.Equal(in.Ask1.Px) {
		return decideTighten(in)
	}

	if in.Ask1.Px.GreaterThan(in.MarkPx) {
		// Joining another live order's actual price: it is already a valid
		// tick multiple by construction, so no quantization is needed.
		if in.Ask1.Px.Equal(in.OurPx) {
			return Decision{}
		}
		return Decision{ShouldAmend: true, NewPx: in.Ask1.Px, Reason: ReasonJoinAsk1}
	}

	return decideMarkFallback(in)
}

// decideMarkFallback computes the smallest tick-aligned price strictly above
// mark price.
func decideMarkFallback(in QuoteInput) Decision {
	tick, err := FindTickSize(in.Bands, in.MarkPx)
	if err != nil {
		return Decision{}
	}
	cand := nextTickAbove(in.MarkPx, tick)

	// The candidate may have crossed into a different tick band; re-resolve
	// once against the band it actually landed in. Bands are few and
	// monotonic, so one extra pass is sufficient.
	if tick2, err := FindTickSize(in.Bands, cand); err == nil && !tick2.Equal(tick) {
		cand = nextTickAbove(in.MarkPx, tick2)
	}

	if cand.Equal(in.OurPx) {
		return Decision{}
	}
	return Decision{ShouldAmend: true, NewPx: cand, Reason: ReasonMarkFallback}
}

// nextTickAbove returns the smallest multiple of tick strictly greater than px.
func nextTickAbove(px, tick decimal.Decimal) decimal.Decimal {
	cand := ceilToTick(px, tick)
	if cand.LessThanOrEqual(px) {
		cand = cand.Add(tick)
	}
	return cand
}

// decideTighten handles the case where our order sits at ask1.
func decideTighten(in QuoteInput) Decision {
	// Solo only if the book says exactly one order sits at ask1 AND its size
	// matches our remaining size (belt and suspenders).
	solo := in.Ask1.NumOrders == 1 && in.Ask1.Sz.Equal(in.OurSz)
	if !solo || in.Ask2 == nil {
		return Decision{}
	}

	tickAsk1, err := FindTickSize(in.Bands, in.Ask1.Px)
	if err != nil {
		return Decision{}
	}
	if !in.Ask2.Px.Sub(in.Ask1.Px).GreaterThan(tickAsk1) {
		return Decision{}
	}

	tickAsk2, err := FindTickSize(in.Bands, in.Ask2.Px)
	if err != nil {
		return Decision{}
	}
	raw := in.Ask2.Px.Sub(tickAsk2)

	// raw may sit in a different band than ask2; quantize down using the band
	// raw itself falls into.
	tickRaw, err := FindTickSize(in.Bands, raw)
	if err != nil {
		return Decision{}
	}
	cand := floorToTick(raw, tickRaw)

	// Never chase a junk ask2 further than a fixed number of ticks above mark.
	tickAtMark, err := FindTickSize(in.Bands, in.MarkPx)
	if err != nil {
		return Decision{}
	}
	capPx := in.MarkPx.Add(tickAtMark.Mul(decimal.NewFromInt(TightenCapTicksAboveMark)))
	if cand.GreaterThan(capPx) {
		tickCap, err := FindTickSize(in.Bands, capPx)
		if err != nil {
			return Decision{}
		}
		cand = floorToTick(capPx, tickCap)
	}

	// Tightening should only ever raise our price; if the cap (or rounding)
	// leaves us at or below where we already are, do nothing.
	if cand.LessThanOrEqual(in.OurPx) {
		return Decision{}
	}
	return Decision{ShouldAmend: true, NewPx: cand, Reason: ReasonTightenAsk1}
}
