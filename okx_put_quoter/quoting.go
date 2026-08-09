package main

import "github.com/shopspring/decimal"

type Reason string

const (
	ReasonNone         Reason = ""
	ReasonJoinAsk1     Reason = "join_ask1"
	ReasonTightenAsk1  Reason = "tighten_ask1"
	ReasonMarkFallback Reason = "mark_fallback"
)

type QuoteInput struct {
	OurPx  decimal.Decimal
	OurSz  decimal.Decimal
	MarkPx decimal.Decimal
	TickSz decimal.Decimal
	Ask1   *BookLevel
	Ask2   *BookLevel
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
		solo := in.Ask1.Sz.Equal(in.OurSz)
		if solo && in.Ask2 != nil {
			gap := in.Ask2.Px.Sub(in.Ask1.Px)
			if gap.GreaterThan(in.TickSz) {
				return Decision{ShouldAmend: true, NewPx: in.Ask2.Px.Sub(in.TickSz), Reason: ReasonTightenAsk1}
			}
		}
		return Decision{}
	}

	if in.Ask1.Px.GreaterThan(in.MarkPx) {
		return Decision{ShouldAmend: true, NewPx: in.Ask1.Px, Reason: ReasonJoinAsk1}
	}
	return Decision{ShouldAmend: true, NewPx: in.MarkPx.Add(in.TickSz), Reason: ReasonMarkFallback}
}
