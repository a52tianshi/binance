package main

import (
	"fmt"
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

// FetchOpenEthPutSellOrders walks every page of orders-pending (OKX caps a page
// at 100 rows; further pages are requested with after=<last ordId>) and returns
// the ETH put sell orders across all of them.
func FetchOpenEthPutSellOrders(c *Client) ([]PendingOrder, error) {
	var result []PendingOrder
	after := ""

	for {
		q := url.Values{
			"instType": {"OPTION"},
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

		for _, r := range raw {
			if r.Side != "sell" || r.OptType != "P" || !strings.HasPrefix(r.InstId, "ETH-") {
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

	return result, nil
}

func GroupByInstId(orders []PendingOrder) map[string][]PendingOrder {
	grouped := make(map[string][]PendingOrder)
	for _, o := range orders {
		grouped[o.InstId] = append(grouped[o.InstId], o)
	}
	return grouped
}
