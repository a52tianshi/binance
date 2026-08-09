package main

import (
	"fmt"
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

type amendOrderResult struct {
	SCode string `json:"sCode"`
	SMsg  string `json:"sMsg"`
}

func AmendOrder(c *Client, instId, ordId string, newPx decimal.Decimal) error {
	req := amendOrderReq{InstId: instId, OrdId: ordId, NewPx: newPx.String()}
	var out []amendOrderResult
	if err := c.DoPrivate("POST", "/api/v5/trade/amend-order", nil, req, &out); err != nil {
		return err
	}
	if len(out) == 0 {
		return fmt.Errorf("amend-order: empty response for ordId=%s", ordId)
	}
	if out[0].SCode != "0" {
		return fmt.Errorf("amend-order rejected: sCode=%s sMsg=%s", out[0].SCode, out[0].SMsg)
	}
	return nil
}

// AmendTracker remembers the last price we actually sent for each ordId.
// amend-order is processed asynchronously by OKX, so the next poll's
// orders-pending snapshot may still report the pre-amend price while the book
// already reflects the new one. Re-sending the same price in that window would
// cost queue priority for nothing.
type AmendTracker struct {
	lastSent map[string]decimal.Decimal
}

func NewAmendTracker() *AmendTracker {
	return &AmendTracker{lastSent: make(map[string]decimal.Decimal)}
}

func (t *AmendTracker) AlreadySent(ordId string, px decimal.Decimal) bool {
	last, ok := t.lastSent[ordId]
	return ok && last.Equal(px)
}

func (t *AmendTracker) Record(ordId string, px decimal.Decimal) {
	t.lastSent[ordId] = px
}

func runOnce(c *Client, cache *TickCache, cfg Config, logger *log.Logger, tracker *AmendTracker) error {
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
		decision := DecideNewPrice(QuoteInput{
			OurPx:  order.Px,
			OurSz:  order.RemainingSz(),
			MarkPx: markPx,
			Bands:  bands,
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
		if tracker.AlreadySent(order.OrdId, decision.NewPx) {
			logger.Printf("SKIP %s: amend to %s already sent (awaiting fresh snapshot)", instId, decision.NewPx)
			continue
		}
		if err := AmendOrder(c, instId, order.OrdId, decision.NewPx); err != nil {
			logger.Printf("ERROR %s: amend order: %v", instId, err)
			continue
		}
		tracker.Record(order.OrdId, decision.NewPx)
		logger.Printf("%s: reason=%s %s -> %s", instId, decision.Reason, order.Px, decision.NewPx)
	}
	return nil
}

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return // no .env file is fine; real env vars may already be set
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}

func main() {
	loadDotEnv(".env")
	cfg, err := LoadConfig(os.Args[1:], os.Getenv)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	client := NewClient(cfg, "https://www.okx.com")
	cache := NewTickCache()
	// One tracker for the life of the process, shared by every poll pass.
	tracker := NewAmendTracker()

	logger.Printf("starting okx_put_quoter: dry_run=%v interval=%v", cfg.DryRun, cfg.PollInterval)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	if err := runOnce(client, cache, cfg, logger, tracker); err != nil {
		logger.Printf("ERROR poll pass: %v", err)
	}
	for range ticker.C {
		if err := runOnce(client, cache, cfg, logger, tracker); err != nil {
			logger.Printf("ERROR poll pass: %v", err)
		}
	}
}
