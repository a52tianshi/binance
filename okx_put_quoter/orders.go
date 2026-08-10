package main

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

type PendingOrder struct {
	InstId    string
	OrdId     string
	Px        decimal.Decimal
	Sz        decimal.Decimal
	AccFillSz decimal.Decimal
}

func (o PendingOrder) RemainingSz() decimal.Decimal {
	return o.Sz.Sub(o.AccFillSz)
}

type pendingOrderResp struct {
	InstId    string `json:"instId"`
	OrdId     string `json:"ordId"`
	Side      string `json:"side"`
	OptType   string `json:"optType"`
	Px        string `json:"px"`
	Sz        string `json:"sz"`
	AccFillSz string `json:"accFillSz"`
}

// ordersPendingPageSize is OKX's maximum page size for orders-pending.
const ordersPendingPageSize = 100

// ordersPendingOrdTypes lists every ordType we want returned. Omitting
// ordType entirely risks OKX only returning a default subset of order
// types, silently excluding advanced limit variants (post-only, FOK, IOC,
// market-maker-protection, etc.) that this bot must still manage.
const ordersPendingOrdTypes = "market,limit,post_only,fok,ioc,optimal_limit_ioc,mmp_and_post_only,elp,rpi"

// optTypeFromInstId derives the option type (C or P) from the instId itself
// rather than trusting the response's optType field, which has been observed
// to come back empty for orders-pending rows in practice even though
// instType=OPTION was requested. Option instIds are always formatted
// {underlying}-{quote}-{expiry}-{strike}-{C|P}, so the last "-"-separated
// segment is authoritative.
func optTypeFromInstId(instId string) string {
	idx := strings.LastIndex(instId, "-")
	if idx == -1 || idx == len(instId)-1 {
		return ""
	}
	return instId[idx+1:]
}

// FetchOpenPutSellOrders walks every page of orders-pending (OKX caps a page
// at 100 rows; further pages are requested with after=<last ordId>) and returns
// the put sell orders across all of them, for any option underlying.
//
// It also logs a one-line diagnostic summary of every raw row seen before
// filtering (grouped by side/optType), so a mismatch between "what OKX
// returns" and "what we expected" is visible without adding a separate
// debug flag.
func FetchOpenPutSellOrders(c *Client, logger *log.Logger) ([]PendingOrder, error) {
	var result []PendingOrder
	after := ""
	rawTotal := 0
	tally := make(map[string]int)

	for {
		q := url.Values{
			"instType": {"OPTION"},
			"ordType":  {ordersPendingOrdTypes},
			"limit":    {strconv.Itoa(ordersPendingPageSize)},
		}
		if after != "" {
			q.Set("after", after)
		}

		var raw []pendingOrderResp
		if err := c.DoPrivate("GET", "/api/v5/trade/orders-pending", q, nil, &raw); err != nil {
			return nil, err
		}
		if len(raw) == 0 {
			break
		}
		rawTotal += len(raw)

		for _, r := range raw {
			optType := r.OptType
			if optType == "" {
				optType = optTypeFromInstId(r.InstId)
			}
			tally[r.Side+"/"+optType]++
			if r.Side != "sell" || optType != "P" {
				continue
			}
			px, err := decimal.NewFromString(r.Px)
			if err != nil {
				return nil, fmt.Errorf("parse px %q for %s: %w", r.Px, r.OrdId, err)
			}
			sz, err := decimal.NewFromString(r.Sz)
			if err != nil {
				return nil, fmt.Errorf("parse sz %q for %s: %w", r.Sz, r.OrdId, err)
			}
			accFillSz, err := decimal.NewFromString(r.AccFillSz)
			if err != nil {
				return nil, fmt.Errorf("parse accFillSz %q for %s: %w", r.AccFillSz, r.OrdId, err)
			}
			result = append(result, PendingOrder{
				InstId: r.InstId, OrdId: r.OrdId, Px: px, Sz: sz, AccFillSz: accFillSz,
			})
		}

		// A short page means we've reached the end.
		if len(raw) < ordersPendingPageSize {
			break
		}
		next := raw[len(raw)-1].OrdId
		if next == "" || next == after {
			// Defensive: never loop forever on a malformed cursor.
			break
		}
		after = next
	}

	logger.Printf("orders-pending: raw_rows=%d by_side_optType=%v matched_put_sell=%d", rawTotal, tally, len(result))
	return result, nil
}

func GroupByInstId(orders []PendingOrder) map[string][]PendingOrder {
	grouped := make(map[string][]PendingOrder)
	for _, o := range orders {
		grouped[o.InstId] = append(grouped[o.InstId], o)
	}
	return grouped
}
