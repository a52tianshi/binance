package main

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/shopspring/decimal"
)

type TickBand struct {
	MinPx  decimal.Decimal
	MaxPx  decimal.Decimal
	TickSz decimal.Decimal
}

type tickBandsResp struct {
	InstFamily string `json:"instFamily"`
	TickBand   []struct {
		MinPx  string `json:"minPx"`
		MaxPx  string `json:"maxPx"`
		TickSz string `json:"tickSz"`
	} `json:"tickBand"`
}

func FetchTickBands(c *Client, instFamily string) ([]TickBand, error) {
	var raw []tickBandsResp
	q := url.Values{"instType": {"OPTION"}, "instFamily": {instFamily}}
	if err := c.DoPublic("GET", "/api/v5/public/instrument-tick-bands", q, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("no tick bands returned for instFamily=%s", instFamily)
	}
	bands := make([]TickBand, 0, len(raw[0].TickBand))
	for _, b := range raw[0].TickBand {
		minPx, err := decimal.NewFromString(b.MinPx)
		if err != nil {
			return nil, fmt.Errorf("parse minPx %q: %w", b.MinPx, err)
		}
		maxPx, err := decimal.NewFromString(b.MaxPx)
		if err != nil {
			return nil, fmt.Errorf("parse maxPx %q: %w", b.MaxPx, err)
		}
		tickSz, err := decimal.NewFromString(b.TickSz)
		if err != nil {
			return nil, fmt.Errorf("parse tickSz %q: %w", b.TickSz, err)
		}
		bands = append(bands, TickBand{MinPx: minPx, MaxPx: maxPx, TickSz: tickSz})
	}
	return bands, nil
}

func FindTickSize(bands []TickBand, px decimal.Decimal) (decimal.Decimal, error) {
	if len(bands) == 0 {
		return decimal.Decimal{}, fmt.Errorf("no tick bands available")
	}
	for _, b := range bands {
		if px.LessThanOrEqual(b.MaxPx) {
			return b.TickSz, nil
		}
	}
	return bands[len(bands)-1].TickSz, nil
}

type BookLevel struct {
	Px decimal.Decimal
	Sz decimal.Decimal
	// NumOrders is the 4th element of an OKX book row ([price, size, "0", numOrders]).
	// It is 0 when the row is shorter than 4 elements.
	NumOrders int
}

type OrderBook struct {
	Ask1 *BookLevel
	Ask2 *BookLevel
}

type orderBookResp struct {
	Asks [][]string `json:"asks"`
	Bids [][]string `json:"bids"`
}

func parseLevel(row []string) (*BookLevel, error) {
	if len(row) < 2 {
		return nil, fmt.Errorf("malformed book level: %v", row)
	}
	px, err := decimal.NewFromString(row[0])
	if err != nil {
		return nil, fmt.Errorf("parse level price %q: %w", row[0], err)
	}
	sz, err := decimal.NewFromString(row[1])
	if err != nil {
		return nil, fmt.Errorf("parse level size %q: %w", row[1], err)
	}
	lvl := &BookLevel{Px: px, Sz: sz}
	if len(row) >= 4 {
		numOrders, err := strconv.Atoi(row[3])
		if err != nil {
			return nil, fmt.Errorf("parse level numOrders %q: %w", row[3], err)
		}
		lvl.NumOrders = numOrders
	}
	return lvl, nil
}

func FetchOrderBook(c *Client, instId string) (OrderBook, error) {
	var raw []orderBookResp
	q := url.Values{"instId": {instId}, "sz": {"2"}}
	if err := c.DoPublic("GET", "/api/v5/market/books", q, &raw); err != nil {
		return OrderBook{}, err
	}
	if len(raw) == 0 {
		return OrderBook{}, fmt.Errorf("no order book returned for instId=%s", instId)
	}
	var book OrderBook
	if len(raw[0].Asks) > 0 {
		lvl, err := parseLevel(raw[0].Asks[0])
		if err != nil {
			return OrderBook{}, err
		}
		book.Ask1 = lvl
	}
	if len(raw[0].Asks) > 1 {
		lvl, err := parseLevel(raw[0].Asks[1])
		if err != nil {
			return OrderBook{}, err
		}
		book.Ask2 = lvl
	}
	return book, nil
}

type markPriceResp struct {
	MarkPx string `json:"markPx"`
}

func FetchMarkPrice(c *Client, instId string) (decimal.Decimal, error) {
	var raw []markPriceResp
	q := url.Values{"instType": {"OPTION"}, "instId": {instId}}
	if err := c.DoPublic("GET", "/api/v5/public/mark-price", q, &raw); err != nil {
		return decimal.Decimal{}, err
	}
	if len(raw) == 0 {
		return decimal.Decimal{}, fmt.Errorf("no mark price returned for instId=%s", instId)
	}
	px, err := decimal.NewFromString(raw[0].MarkPx)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("parse markPx %q: %w", raw[0].MarkPx, err)
	}
	return px, nil
}

type TickCache struct {
	bands map[string][]TickBand
}

func NewTickCache() *TickCache {
	return &TickCache{bands: make(map[string][]TickBand)}
}

func (t *TickCache) Get(c *Client, instFamily string) ([]TickBand, error) {
	if b, ok := t.bands[instFamily]; ok {
		return b, nil
	}
	b, err := FetchTickBands(c, instFamily)
	if err != nil {
		return nil, err
	}
	t.bands[instFamily] = b
	return b, nil
}
