# OKX PUT 卖单自动改价工具 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone Go program (`okx_put_quoter/`) that polls the account's open ETH PUT sell orders every N seconds and automatically amends their price per two rules (join best ask when profitable, tighten price when solely holding best ask with a wide gap to second-best ask), with a dry-run mode.

**Architecture:** Single `package main` binary. One goroutine, sequential polling loop (no concurrency needed — avoids races and lets the tick-size cache be a plain map). A small hand-rolled OKX v5 REST client (HMAC-SHA256 signing per OKX spec) backs four read calls (pending orders, order book, mark price, tick bands) and one write call (amend order). Decision logic is a pure function, unit-tested in isolation from all I/O.

**Tech Stack:** Go (module `main`, existing `go.mod`, go 1.20). New dependency: `github.com/shopspring/decimal` for exact price arithmetic (OKX prices/tick sizes are decimal strings; float64 would risk rounding errors in price comparisons). Existing dependency `github.com/natefinch/lumberjack` reused for log rotation, consistent with the rest of the repo.

## Global Constraints

- Only ETH options: filter `instId` with prefix `ETH-`.
- Only `optType=P` (put) and `side=sell` orders.
- Assume at most one of our open orders per `instId`; if more than one is found, log a warning and skip that `instId` entirely (do not amend).
- Tick size per `instId` is fetched from `GET /api/v5/public/instrument-tick-bands` **once per instId**, then cached in memory for the life of the process (verified live: response shape is `{"data":[{"instFamily":"...","tickBand":[{"minPx":"0","maxPx":"0.005","tickSz":"0.0001"},{"minPx":"0.005","maxPx":"10000000","tickSz":"0.0005"}]}]}`).
- Amend via `POST /api/v5/trade/amend-order` (price only, never touch size).
- Credentials come from `okx_put_quoter/.env` (gitignored): `OKX_API_KEY`, `OKX_API_SECRET`, `OKX_API_PASSPHRASE`.
- `--dry-run` flag (default false) must prevent any real amend call and instead log what would have happened.
- `--interval` flag (default `5`, seconds) controls poll cadence.
- All prices/sizes handled as `decimal.Decimal`, never `float64`, to avoid comparison bugs at tick-size boundaries.

## Verified API Reference (confirmed live during design, not just docs)

- `GET /api/v5/trade/orders-pending?instType=OPTION` → data rows include `instId`, `ordId`, `side`, `optType`, `px`, `sz`, `accFillSz`. Remaining size = `sz - accFillSz` (compute manually; do not rely on an unverified `leavesSz` field).
- `GET /api/v5/market/books?instId=X&sz=2` → `data[0].asks` / `data[0].bids` are arrays of `[price, size, "0", numOrders]` (4-element string arrays). **Asks/bids can be empty arrays** — code must handle that.
- `GET /api/v5/public/mark-price?instType=OPTION&instId=X` → `data[0].markPx` (string decimal).
- `GET /api/v5/public/instrument-tick-bands?instType=OPTION&instFamily=ETH-USD` → `data[0].tickBand` is a list of `{minPx, maxPx, tickSz}`, ascending by `maxPx`. Given a price, pick the first band where `price <= maxPx`.
- `POST /api/v5/trade/amend-order` body `{instId, ordId, newPx}` → top-level `code`, and per-item `data[0].sCode`/`data[0].sMsg` for the actual result (top-level `code=="0"` only means the request was accepted for processing).
- Auth headers (private endpoints only — the four GETs above are all public and need no auth, only `orders-pending` and `amend-order` are private): `OK-ACCESS-KEY`, `OK-ACCESS-SIGN` (base64 HMAC-SHA256 of `timestamp+method+requestPath(+queryString)+body`, method uppercase, timestamp ISO-8601 ms e.g. `2020-12-08T09:08:57.715Z`, body="" for GET), `OK-ACCESS-TIMESTAMP`, `OK-ACCESS-PASSPHRASE`.

---

## File Structure

```
okx_put_quoter/
  config.go          // env + flag parsing
  config_test.go
  okx_client.go       // HTTP client, HMAC signing, generic doPublic/doPrivate
  okx_client_test.go
  market.go           // order book, mark price, tick-band cache
  market_test.go
  orders.go           // fetch + filter + group pending orders
  orders_test.go
  quoting.go          // pure decision function
  quoting_test.go
  main.go             // wiring + poll loop
  .env.example
  README.md
```

---

### Task 1: Config loading

**Files:**
- Create: `okx_put_quoter/config.go`
- Test: `okx_put_quoter/config_test.go`

**Interfaces:**
- Produces: `type Config struct { APIKey, APISecret, APIPassphrase string; DryRun bool; PollInterval time.Duration }` and `func LoadConfig(args []string, getenv func(string) string) (Config, error)`.

- [ ] **Step 1: Write the failing test**

```go
// okx_put_quoter/config_test.go
package main

import (
	"testing"
	"time"
)

func TestLoadConfig_DefaultsAndEnv(t *testing.T) {
	env := map[string]string{
		"OKX_API_KEY":        "key123",
		"OKX_API_SECRET":     "secret123",
		"OKX_API_PASSPHRASE": "pass123",
	}
	getenv := func(k string) string { return env[k] }

	cfg, err := LoadConfig([]string{}, getenv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "key123" || cfg.APISecret != "secret123" || cfg.APIPassphrase != "pass123" {
		t.Fatalf("credentials not loaded from env: %+v", cfg)
	}
	if cfg.DryRun != false {
		t.Fatalf("expected dry-run default false, got true")
	}
	if cfg.PollInterval != 5*time.Second {
		t.Fatalf("expected default interval 5s, got %v", cfg.PollInterval)
	}
}

func TestLoadConfig_FlagsOverride(t *testing.T) {
	env := map[string]string{
		"OKX_API_KEY":        "key123",
		"OKX_API_SECRET":     "secret123",
		"OKX_API_PASSPHRASE": "pass123",
	}
	getenv := func(k string) string { return env[k] }

	cfg, err := LoadConfig([]string{"--dry-run", "--interval", "10"}, getenv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.DryRun {
		t.Fatalf("expected dry-run true")
	}
	if cfg.PollInterval != 10*time.Second {
		t.Fatalf("expected interval 10s, got %v", cfg.PollInterval)
	}
}

func TestLoadConfig_MissingCredentials(t *testing.T) {
	getenv := func(k string) string { return "" }
	_, err := LoadConfig([]string{}, getenv)
	if err == nil {
		t.Fatalf("expected error for missing credentials")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd okx_put_quoter && go test ./... -run TestLoadConfig -v`
Expected: FAIL (build error — `LoadConfig`/`Config` not defined)

- [ ] **Step 3: Write minimal implementation**

```go
// okx_put_quoter/config.go
package main

import (
	"flag"
	"fmt"
	"time"
)

type Config struct {
	APIKey        string
	APISecret     string
	APIPassphrase string
	DryRun        bool
	PollInterval  time.Duration
}

func LoadConfig(args []string, getenv func(string) string) (Config, error) {
	fs := flag.NewFlagSet("okx_put_quoter", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "log intended amends without sending them")
	intervalSec := fs.Int("interval", 5, "poll interval in seconds")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	cfg := Config{
		APIKey:        getenv("OKX_API_KEY"),
		APISecret:     getenv("OKX_API_SECRET"),
		APIPassphrase: getenv("OKX_API_PASSPHRASE"),
		DryRun:        *dryRun,
		PollInterval:  time.Duration(*intervalSec) * time.Second,
	}

	if cfg.APIKey == "" || cfg.APISecret == "" || cfg.APIPassphrase == "" {
		return Config{}, fmt.Errorf("OKX_API_KEY, OKX_API_SECRET and OKX_API_PASSPHRASE must be set")
	}
	return cfg, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd okx_put_quoter && go test ./... -run TestLoadConfig -v`
Expected: PASS (all three tests)

- [ ] **Step 5: Commit**

```bash
git add okx_put_quoter/config.go okx_put_quoter/config_test.go
git commit -m "okx_put_quoter: add config loading from env + flags"
```

---

### Task 2: OKX REST client with HMAC signing

**Files:**
- Create: `okx_put_quoter/okx_client.go`
- Test: `okx_put_quoter/okx_client_test.go`

**Interfaces:**
- Consumes: `Config` (Task 1) for credentials.
- Produces:
  - `type Client struct { ... }` with `func NewClient(cfg Config, baseURL string) *Client`
  - `func (c *Client) DoPublic(method, path string, query url.Values, out interface{}) error`
  - `func (c *Client) DoPrivate(method, path string, query url.Values, body interface{}, out interface{}) error`
  - Both unmarshal the OKX envelope `{code, msg, data}` into `out` (out receives the raw `data` field, already JSON-decoded into whatever type is passed), and return an error if `code != "0"`.

- [ ] **Step 1: Write the failing test**

```go
// okx_put_quoter/okx_client_test.go
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDoPrivate_SignsRequestCorrectly(t *testing.T) {
	cfg := Config{APIKey: "key123", APISecret: "secret123", APIPassphrase: "pass123"}

	var gotKey, gotSign, gotTs, gotPass, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("OK-ACCESS-KEY")
		gotSign = r.Header.Get("OK-ACCESS-SIGN")
		gotTs = r.Header.Get("OK-ACCESS-TIMESTAMP")
		gotPass = r.Header.Get("OK-ACCESS-PASSPHRASE")
		gotMethod = r.Method
		w.Write([]byte(`{"code":"0","msg":"","data":[{"ok":true}]}`))
	}))
	defer srv.Close()

	c := NewClient(cfg, srv.URL)
	var out []map[string]bool
	err := c.DoPrivate("GET", "/api/v5/trade/orders-pending", url.Values{"instType": {"OPTION"}}, nil, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 || !out[0]["ok"] {
		t.Fatalf("unexpected decoded data: %+v", out)
	}
	if gotKey != "key123" || gotPass != "pass123" || gotMethod != "GET" {
		t.Fatalf("unexpected headers: key=%s pass=%s method=%s", gotKey, gotPass, gotMethod)
	}
	if gotTs == "" || gotSign == "" {
		t.Fatalf("expected timestamp and signature headers to be set")
	}

	// Recompute expected signature the same way OKX requires and check it matches.
	requestPath := "/api/v5/trade/orders-pending?instType=OPTION"
	preHash := gotTs + "GET" + requestPath
	mac := hmac.New(sha256.New, []byte(cfg.APISecret))
	mac.Write([]byte(preHash))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if gotSign != want {
		t.Fatalf("signature mismatch: got %s want %s", gotSign, want)
	}
}

func TestDoPublic_NoAuthHeaders(t *testing.T) {
	cfg := Config{APIKey: "key123", APISecret: "secret123", APIPassphrase: "pass123"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("OK-ACCESS-KEY") != "" {
			t.Errorf("public endpoint should not send auth headers")
		}
		w.Write([]byte(`{"code":"0","msg":"","data":[{"markPx":"1.23"}]}`))
	}))
	defer srv.Close()

	c := NewClient(cfg, srv.URL)
	var out []map[string]string
	if err := c.DoPublic("GET", "/api/v5/public/mark-price", url.Values{"instType": {"OPTION"}}, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out[0]["markPx"] != "1.23" {
		t.Fatalf("unexpected data: %+v", out)
	}
}

func TestDoPrivate_APIErrorReturned(t *testing.T) {
	cfg := Config{APIKey: "k", APISecret: "s", APIPassphrase: "p"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"50001","msg":"boom","data":[]}`))
	}))
	defer srv.Close()

	c := NewClient(cfg, srv.URL)
	var out json.RawMessage
	err := c.DoPrivate("GET", "/api/v5/trade/orders-pending", nil, nil, &out)
	if err == nil {
		t.Fatalf("expected error for non-zero code")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd okx_put_quoter && go test ./... -run 'TestDoPrivate|TestDoPublic' -v`
Expected: FAIL (build error — `Client`/`NewClient` not defined)

- [ ] **Step 3: Write minimal implementation**

```go
// okx_put_quoter/okx_client.go
package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	cfg        Config
	baseURL    string
	httpClient *http.Client
}

func NewClient(cfg Config, baseURL string) *Client {
	return &Client{cfg: cfg, baseURL: baseURL, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

type okxEnvelope struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func (c *Client) DoPublic(method, path string, query url.Values, out interface{}) error {
	return c.do(method, path, query, nil, out, false)
}

func (c *Client) DoPrivate(method, path string, query url.Values, body interface{}, out interface{}) error {
	return c.do(method, path, query, body, out, true)
}

func (c *Client) do(method, path string, query url.Values, body interface{}, out interface{}, signed bool) error {
	requestPath := path
	if len(query) > 0 {
		requestPath = path + "?" + query.Encode()
	}

	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyBytes = b
	}

	req, err := http.NewRequest(method, c.baseURL+requestPath, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if signed {
		ts := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
		preHash := ts + method + requestPath + string(bodyBytes)
		mac := hmac.New(sha256.New, []byte(c.cfg.APISecret))
		mac.Write([]byte(preHash))
		sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))

		req.Header.Set("OK-ACCESS-KEY", c.cfg.APIKey)
		req.Header.Set("OK-ACCESS-SIGN", sign)
		req.Header.Set("OK-ACCESS-TIMESTAMP", ts)
		req.Header.Set("OK-ACCESS-PASSPHRASE", c.cfg.APIPassphrase)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var env okxEnvelope
	if err := json.Unmarshal(respBytes, &env); err != nil {
		return fmt.Errorf("decode envelope: %w (body=%s)", err, string(respBytes))
	}
	if env.Code != "0" {
		return fmt.Errorf("okx api error code=%s msg=%s", env.Code, env.Msg)
	}
	if out != nil {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("decode data: %w (data=%s)", err, string(env.Data))
		}
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd okx_put_quoter && go test ./... -run 'TestDoPrivate|TestDoPublic' -v`
Expected: PASS (all three tests)

- [ ] **Step 5: Commit**

```bash
git add okx_put_quoter/okx_client.go okx_put_quoter/okx_client_test.go
git commit -m "okx_put_quoter: add signed OKX REST client"
```

---

### Task 3: Market data (order book, mark price, tick bands)

**Files:**
- Create: `okx_put_quoter/market.go`
- Test: `okx_put_quoter/market_test.go`
- Modify: `go.mod` / `go.sum` (add `github.com/shopspring/decimal`)

**Interfaces:**
- Consumes: `*Client` (Task 2).
- Produces:
  - `type TickBand struct { MinPx, MaxPx, TickSz decimal.Decimal }`
  - `func FetchTickBands(c *Client, instFamily string) ([]TickBand, error)`
  - `func FindTickSize(bands []TickBand, px decimal.Decimal) (decimal.Decimal, error)`
  - `type BookLevel struct { Px, Sz decimal.Decimal }`
  - `type OrderBook struct { Ask1, Ask2 *BookLevel }` (nil means that level doesn't exist)
  - `func FetchOrderBook(c *Client, instId string) (OrderBook, error)`
  - `func FetchMarkPrice(c *Client, instId string) (decimal.Decimal, error)`
  - `type TickCache struct{ bands map[string][]TickBand }`, `func NewTickCache() *TickCache`, `func (t *TickCache) Get(c *Client, instFamily string) ([]TickBand, error)` (fetches once, caches by instFamily).

- [ ] **Step 1: Add the decimal dependency**

Run: `cd /Users/chenquan/Documents/GitHub/binance && go get github.com/shopspring/decimal`
Expected: `go.mod`/`go.sum` updated with the new require line.

- [ ] **Step 2: Write the failing test**

```go
// okx_put_quoter/market_test.go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
)

func TestFetchTickBands_ParsesLiveShapedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"0","msg":"","data":[{"instFamily":"ETH-USD","instType":"OPTION","seriesId":"","tickBand":[{"maxPx":"0.005","minPx":"0","tickSz":"0.0001"},{"maxPx":"10000000","minPx":"0.005","tickSz":"0.0005"}]}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	bands, err := FetchTickBands(c, "ETH-USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bands) != 2 {
		t.Fatalf("expected 2 bands, got %d", len(bands))
	}
	if !bands[0].TickSz.Equal(decimal.RequireFromString("0.0001")) {
		t.Fatalf("unexpected tickSz for band 0: %v", bands[0].TickSz)
	}
}

func TestFindTickSize_PicksBandByMaxPx(t *testing.T) {
	bands := []TickBand{
		{MinPx: decimal.Zero, MaxPx: decimal.RequireFromString("0.005"), TickSz: decimal.RequireFromString("0.0001")},
		{MinPx: decimal.RequireFromString("0.005"), MaxPx: decimal.RequireFromString("10000000"), TickSz: decimal.RequireFromString("0.0005")},
	}

	got, err := FindTickSize(bands, decimal.RequireFromString("0.003"))
	if err != nil || !got.Equal(decimal.RequireFromString("0.0001")) {
		t.Fatalf("expected 0.0001 tick for low price, got %v err %v", got, err)
	}

	got, err = FindTickSize(bands, decimal.RequireFromString("1.5"))
	if err != nil || !got.Equal(decimal.RequireFromString("0.0005")) {
		t.Fatalf("expected 0.0005 tick for high price, got %v err %v", got, err)
	}
}

func TestFetchOrderBook_HandlesEmptyAsks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"0","msg":"","data":[{"asks":[],"bids":[["0.1055","5","0","1"]],"ts":"1","seqId":1}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	book, err := FetchOrderBook(c, "ETH-USD-260810-1700-C")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if book.Ask1 != nil {
		t.Fatalf("expected nil Ask1 for empty asks, got %+v", book.Ask1)
	}
}

func TestFetchOrderBook_ParsesTwoAsks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"0","msg":"","data":[{"asks":[["0.10","3","0","1"],["0.12","2","0","1"]],"bids":[],"ts":"1","seqId":1}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	book, err := FetchOrderBook(c, "X")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if book.Ask1 == nil || !book.Ask1.Px.Equal(decimal.RequireFromString("0.10")) {
		t.Fatalf("unexpected ask1: %+v", book.Ask1)
	}
	if book.Ask2 == nil || !book.Ask2.Px.Equal(decimal.RequireFromString("0.12")) {
		t.Fatalf("unexpected ask2: %+v", book.Ask2)
	}
}

func TestFetchMarkPrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"0","msg":"","data":[{"instId":"X","instType":"OPTION","markPx":"0.1144782725373311","ts":"1"}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	px, err := FetchMarkPrice(c, "X")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !px.Equal(decimal.RequireFromString("0.1144782725373311")) {
		t.Fatalf("unexpected markPx: %v", px)
	}
}

func TestTickCache_FetchesOncePerInstFamily(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{"code":"0","msg":"","data":[{"instFamily":"ETH-USD","tickBand":[{"maxPx":"10000000","minPx":"0","tickSz":"0.0005"}]}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	cache := NewTickCache()
	if _, err := cache.Get(c, "ETH-USD"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := cache.Get(c, "ETH-USD"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 HTTP call, got %d", calls)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd okx_put_quoter && go test ./... -run 'TestFetch|TestFind|TestTickCache' -v`
Expected: FAIL (build error — types/functions not defined)

- [ ] **Step 4: Write minimal implementation**

```go
// okx_put_quoter/market.go
package main

import (
	"encoding/json"
	"fmt"
	"net/url"

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
	return &BookLevel{Px: px, Sz: sz}, nil
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

var _ = json.RawMessage{}
```

(Remove the trailing `var _ = json.RawMessage{}` line if `encoding/json` ends up unused — check with `goimports`/`go vet` in Step 5.)

- [ ] **Step 5: Run test to verify it passes, then clean up unused imports**

Run: `cd okx_put_quoter && go vet ./... && go test ./... -run 'TestFetch|TestFind|TestTickCache' -v`
Expected: PASS. If `go vet` flags the unused `encoding/json` import, delete that import and the placeholder line from `market.go`.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum okx_put_quoter/market.go okx_put_quoter/market_test.go
git commit -m "okx_put_quoter: add order book, mark price, tick band fetch + cache"
```

---

### Task 4: Fetch and filter pending PUT sell orders

**Files:**
- Create: `okx_put_quoter/orders.go`
- Test: `okx_put_quoter/orders_test.go`

**Interfaces:**
- Consumes: `*Client` (Task 2), `decimal.Decimal` (shopspring).
- Produces:
  - `type PendingOrder struct { InstId, OrdId string; Px, Sz, AccFillSz decimal.Decimal }`
  - `func (o PendingOrder) RemainingSz() decimal.Decimal` (= `Sz.Sub(AccFillSz)`)
  - `func FetchOpenEthPutSellOrders(c *Client) ([]PendingOrder, error)` — calls `orders-pending` with `instType=OPTION`, filters client-side to `side=="sell"`, `optType=="P"`, `instId` prefix `"ETH-"`.
  - `func GroupByInstId(orders []PendingOrder) map[string][]PendingOrder`

- [ ] **Step 1: Write the failing test**

```go
// okx_put_quoter/orders_test.go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
)

func TestFetchOpenEthPutSellOrders_FiltersCorrectly(t *testing.T) {
	body := `{"code":"0","msg":"","data":[
		{"instId":"ETH-USD-260810-1700-P","ordId":"1","side":"sell","optType":"P","px":"0.10","sz":"5","accFillSz":"0"},
		{"instId":"ETH-USD-260810-1700-C","ordId":"2","side":"sell","optType":"C","px":"0.20","sz":"5","accFillSz":"0"},
		{"instId":"ETH-USD-260810-1700-P","ordId":"3","side":"buy","optType":"P","px":"0.05","sz":"5","accFillSz":"0"},
		{"instId":"BTC-USD-260810-30000-P","ordId":"4","side":"sell","optType":"P","px":"1.0","sz":"1","accFillSz":"0"}
	]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	defer srv.Close()

	c := NewClient(Config{}, srv.URL)
	orders, err := FetchOpenEthPutSellOrders(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 matching order, got %d: %+v", len(orders), orders)
	}
	if orders[0].OrdId != "1" {
		t.Fatalf("unexpected order matched: %+v", orders[0])
	}
}

func TestPendingOrder_RemainingSz(t *testing.T) {
	o := PendingOrder{Sz: decimal.RequireFromString("5"), AccFillSz: decimal.RequireFromString("2")}
	if !o.RemainingSz().Equal(decimal.RequireFromString("3")) {
		t.Fatalf("expected remaining 3, got %v", o.RemainingSz())
	}
}

func TestGroupByInstId(t *testing.T) {
	orders := []PendingOrder{
		{InstId: "A", OrdId: "1"},
		{InstId: "A", OrdId: "2"},
		{InstId: "B", OrdId: "3"},
	}
	grouped := GroupByInstId(orders)
	if len(grouped["A"]) != 2 || len(grouped["B"]) != 1 {
		t.Fatalf("unexpected grouping: %+v", grouped)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd okx_put_quoter && go test ./... -run 'TestFetchOpenEthPutSellOrders|TestPendingOrder_RemainingSz|TestGroupByInstId' -v`
Expected: FAIL (build error)

- [ ] **Step 3: Write minimal implementation**

```go
// okx_put_quoter/orders.go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd okx_put_quoter && go test ./... -run 'TestFetchOpenEthPutSellOrders|TestPendingOrder_RemainingSz|TestGroupByInstId' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add okx_put_quoter/orders.go okx_put_quoter/orders_test.go
git commit -m "okx_put_quoter: fetch and filter open ETH PUT sell orders"
```

---

### Task 5: Quoting decision logic (pure function)

**Files:**
- Create: `okx_put_quoter/quoting.go`
- Test: `okx_put_quoter/quoting_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks except `decimal.Decimal`.
- Produces:
  - `type QuoteInput struct { OurPx, OurSz, MarkPx, TickSz decimal.Decimal; Ask1, Ask2 *BookLevel }` (reuses `BookLevel` from Task 3)
  - `type Reason string` with constants `ReasonNone`, `ReasonJoinAsk1`, `ReasonTightenAsk1`, `ReasonMarkFallback`
  - `type Decision struct { ShouldAmend bool; NewPx decimal.Decimal; Reason Reason }`
  - `func DecideNewPrice(in QuoteInput) Decision`

- [ ] **Step 1: Write the failing test**

```go
// okx_put_quoter/quoting_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd okx_put_quoter && go test ./... -run TestDecideNewPrice -v`
Expected: FAIL (build error)

- [ ] **Step 3: Write minimal implementation**

```go
// okx_put_quoter/quoting.go
package main

import "github.com/shopspring/decimal"

type Reason string

const (
	ReasonNone          Reason = ""
	ReasonJoinAsk1      Reason = "join_ask1"
	ReasonTightenAsk1   Reason = "tighten_ask1"
	ReasonMarkFallback  Reason = "mark_fallback"
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd okx_put_quoter && go test ./... -run TestDecideNewPrice -v`
Expected: PASS (all 7 cases)

- [ ] **Step 5: Commit**

```bash
git add okx_put_quoter/quoting.go okx_put_quoter/quoting_test.go
git commit -m "okx_put_quoter: add pure quoting decision logic"
```

---

### Task 6: Amend order call + main poll loop

**Files:**
- Create: `okx_put_quoter/main.go`
- Test: `okx_put_quoter/main_test.go` (covers `AmendOrder` and `instFamilyFor`; the loop itself is exercised manually via dry-run, per the design doc's YAGNI call on integration tests)

**Interfaces:**
- Consumes: everything from Tasks 1-5.
- Produces:
  - `func AmendOrder(c *Client, instId, ordId string, newPx decimal.Decimal) error`
  - `func instFamilyFor(instId string) string` (ETH option instId format is `ETH-USD-YYMMDD-STRIKE-C/P`; instFamily is `ETH-USD`, i.e. first two `-`-separated segments)
  - `func runOnce(c *Client, cache *TickCache, cfg Config, logger *log.Logger) error` (one full poll pass, used by `main`'s loop)
  - `func main()`

- [ ] **Step 1: Write the failing test**

```go
// okx_put_quoter/main_test.go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
)

func TestInstFamilyFor(t *testing.T) {
	got := instFamilyFor("ETH-USD-260810-1700-P")
	if got != "ETH-USD" {
		t.Fatalf("expected ETH-USD, got %s", got)
	}
}

func TestAmendOrder_SendsExpectedBody(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		gotBody = string(buf)
		w.Write([]byte(`{"code":"0","msg":"","data":[{"sCode":"0","sMsg":""}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{APIKey: "k", APISecret: "s", APIPassphrase: "p"}, srv.URL)
	err := AmendOrder(c, "ETH-USD-260810-1700-P", "42", decimal.RequireFromString("0.1095"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{`"instId":"ETH-USD-260810-1700-P"`, `"ordId":"42"`, `"newPx":"0.1095"`} {
		if !contains(gotBody, want) {
			t.Fatalf("expected body to contain %s, got %s", want, gotBody)
		}
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd okx_put_quoter && go test ./... -run 'TestInstFamilyFor|TestAmendOrder' -v`
Expected: FAIL (build error)

- [ ] **Step 3: Write minimal implementation**

```go
// okx_put_quoter/main.go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd okx_put_quoter && go test ./... -v`
Expected: PASS (every test across all six files)

- [ ] **Step 5: Build the binary**

Run: `cd okx_put_quoter && go build -o /tmp/okx_put_quoter .`
Expected: builds with no errors.

- [ ] **Step 6: Commit**

```bash
git add okx_put_quoter/main.go okx_put_quoter/main_test.go
git commit -m "okx_put_quoter: wire up amend-order call and poll loop"
```

---

### Task 7: Credentials scaffolding, docs, gitignore

**Files:**
- Create: `okx_put_quoter/.env.example`
- Create: `okx_put_quoter/README.md`
- Modify: `.gitignore`

**Interfaces:** None (docs/config only).

- [ ] **Step 1: Add `.env` to `.gitignore`**

Edit root `.gitignore`, add a line:
```
.env
```

- [ ] **Step 2: Create `.env.example`**

```
# okx_put_quoter/okx_put_quoter/.env.example
# Copy this file to okx_put_quoter/.env and fill in real values. Never commit .env.
OKX_API_KEY=
OKX_API_SECRET=
OKX_API_PASSPHRASE=
```

- [ ] **Step 3: Create README**

```markdown
# okx_put_quoter

Polls the OKX account's open ETH PUT sell options every N seconds and
automatically amends their price:

- If our order is not the best ask (ask1) and ask1 is above mark price,
  move our price up to ask1.
- If ask1 is at or below mark price (abnormal), move our price to
  mark price + 1 tick instead.
- If our order *is* ask1 and we're the only order at that price, and the
  gap to ask2 is more than one tick, tighten our price to ask2 - 1 tick
  (keeps us best without giving away more edge than necessary).

Assumes at most one open PUT sell order per instId; if more than one is
found for the same instId, that instId is skipped with a warning.

## Setup

```bash
cp .env.example .env
# edit .env with real OKX_API_KEY / OKX_API_SECRET / OKX_API_PASSPHRASE
```

## Run

```bash
# from repo root
go run ./okx_put_quoter --dry-run --interval 5
```

Drop `--dry-run` once you've confirmed the logged decisions look correct.

## Test

```bash
go test ./okx_put_quoter/...
```
```

- [ ] **Step 4: Load `.env` at runtime**

`main()` currently reads credentials via `os.Getenv`, which will not see
values from a `.env` file automatically. Add a minimal loader with no new
dependency: at the top of `main()` in `okx_put_quoter/main.go`, before
`LoadConfig`, add:

```go
loadDotEnv(".env")
```

and add this function to `okx_put_quoter/main.go`:

```go
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
```

- [ ] **Step 5: Run full test suite once more**

Run: `cd okx_put_quoter && go test ./... -v`
Expected: PASS (loadDotEnv has no dedicated test — it's a thin, manually-verifiable I/O shim — but confirm the package still builds and all prior tests still pass)

- [ ] **Step 6: Manual smoke test (dry-run against real account)**

Run: `cd /Users/chenquan/Documents/GitHub/binance && go run ./okx_put_quoter --dry-run --interval 5`
Expected: starts, logs "starting okx_put_quoter: dry_run=true interval=5s", and every 5s either logs nothing (no open ETH put-sell orders needing action) or logs `[DRY-RUN] ...` lines with sane instId/price values. Stop with Ctrl+C. Only remove `--dry-run` after manually confirming the logged decisions look correct for a few cycles.

- [ ] **Step 7: Commit**

```bash
git add .gitignore okx_put_quoter/.env.example okx_put_quoter/README.md okx_put_quoter/main.go
git commit -m "okx_put_quoter: add .env loading, README, and gitignore entry"
```

---

## Self-Review Notes

- **Spec coverage:** Rule A (join ask1 when profitable) → Task 5 `ReasonJoinAsk1`; boundary case (ask1 <= mark) → Task 5 `ReasonMarkFallback`; Rule B (tighten when solely at ask1 with wide gap) → Task 5 `ReasonTightenAsk1`; tick-band lazy fetch+cache → Task 3 `TickCache`; multiple-orders-per-instId guard → Task 6 `runOnce`; dry-run → Task 6 `runOnce`/`main`; credentials via `.env` → Task 7. All design sections are covered.
- **Placeholder scan:** no TBD/TODO; every step has runnable code.
- **Type consistency:** `BookLevel` defined once in Task 3, reused as-is (not redefined) in Task 5's `QuoteInput` and Task 6's `runOnce`. `Config`, `Client`, `PendingOrder`, `TickBand`, `TickCache`, `Decision`/`Reason` are each defined exactly once and used with matching field names across later tasks.
