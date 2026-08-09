package main

import (
	"fmt"
	"net/url"
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

func FetchOpenEthPutSellOrders(c *Client) ([]PendingOrder, error) {
	var raw []pendingOrderResp
	q := url.Values{"instType": {"OPTION"}}
	if err := c.DoPrivate("GET", "/api/v5/trade/orders-pending", q, nil, &raw); err != nil {
		return nil, err
	}

	var result []PendingOrder
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
	return result, nil
}

func GroupByInstId(orders []PendingOrder) map[string][]PendingOrder {
	grouped := make(map[string][]PendingOrder)
	for _, o := range orders {
		grouped[o.InstId] = append(grouped[o.InstId], o)
	}
	return grouped
}
