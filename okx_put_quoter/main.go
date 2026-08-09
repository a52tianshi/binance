package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

func instFamilyFor(instId string) string {
	parts := strings.Split(instId, "-")
	if len(parts) < 2 {
		return instId
	}
	return parts[0] + "-" + parts[1]
}

type amendOrderReq struct {
	InstId string `json:"instId"`
	OrdId  string `json:"ordId"`
	NewPx  string `json:"newPx"`
}

func AmendOrder(c *Client, instId, ordId string, newPx decimal.Decimal) error {
	req := amendOrderReq{InstId: instId, OrdId: ordId, NewPx: newPx.String()}
	var out []map[string]interface{}
	return c.DoPrivate("POST", "/api/v5/trade/amend-order", nil, req, &out)
}

func runOnce(c *Client, cache *TickCache, cfg Config, logger *log.Logger) error {
	orders, err := FetchOpenEthPutSellOrders(c)
	if err != nil {
		return err
	}

	for instId, group := range GroupByInstId(orders) {
		if len(group) > 1 {
			logger.Printf("WARN %s: found %d open put-sell orders, expected at most 1, skipping", instId, len(group))
			continue
		}
		order := group[0]

		bands, err := cache.Get(c, instFamilyFor(instId))
		if err != nil {
			logger.Printf("ERROR %s: fetch tick bands: %v", instId, err)
			continue
		}
		book, err := FetchOrderBook(c, instId)
		if err != nil {
			logger.Printf("ERROR %s: fetch order book: %v", instId, err)
			continue
		}
		markPx, err := FetchMarkPrice(c, instId)
		if err != nil {
			logger.Printf("ERROR %s: fetch mark price: %v", instId, err)
			continue
		}
		refPx := order.Px
		if book.Ask1 != nil {
			refPx = book.Ask1.Px
		}
		tickSz, err := FindTickSize(bands, refPx)
		if err != nil {
			logger.Printf("ERROR %s: find tick size: %v", instId, err)
			continue
		}

		decision := DecideNewPrice(QuoteInput{
			OurPx:  order.Px,
			OurSz:  order.RemainingSz(),
			MarkPx: markPx,
			TickSz: tickSz,
			Ask1:   book.Ask1,
			Ask2:   book.Ask2,
		})
		if !decision.ShouldAmend {
			continue
		}

		if cfg.DryRun {
			logger.Printf("[DRY-RUN] %s: reason=%s %s -> %s", instId, decision.Reason, order.Px, decision.NewPx)
			continue
		}
		if err := AmendOrder(c, instId, order.OrdId, decision.NewPx); err != nil {
			logger.Printf("ERROR %s: amend order: %v", instId, err)
			continue
		}
		logger.Printf("%s: reason=%s %s -> %s", instId, decision.Reason, order.Px, decision.NewPx)
	}
	return nil
}

func main() {
	cfg, err := LoadConfig(os.Args[1:], os.Getenv)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	client := NewClient(cfg, "https://www.okx.com")
	cache := NewTickCache()

	logger.Printf("starting okx_put_quoter: dry_run=%v interval=%v", cfg.DryRun, cfg.PollInterval)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	if err := runOnce(client, cache, cfg, logger); err != nil {
		logger.Printf("ERROR poll pass: %v", err)
	}
	for range ticker.C {
		if err := runOnce(client, cache, cfg, logger); err != nil {
			logger.Printf("ERROR poll pass: %v", err)
		}
	}
}
