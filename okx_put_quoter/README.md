# okx_put_quoter

Polls the OKX account's open PUT sell option orders across all underlyings
(ETH, BTC, SOL, etc.) every N seconds and automatically amends their price.
Examples below use `ETH-USD-260810-1700-P`, but any underlying works the
same way:

- If our order is not the best ask (ask1) and ask1 is above mark price,
  move our price **down** to ask1 so we join the best ask. (Our order is a
  sell, so whenever we're not at ask1 our price is above it; joining always
  lowers our price.)
- If ask1 is at or below mark price (abnormal), move our price to the
  smallest tick-aligned price above mark price instead.
- If our order *is* ask1 and we're the only order at that price (the book
  reports `numOrders == 1` and its size matches our remaining size), and the
  gap to ask2 is more than one tick, raise our price to ask2 - 1 tick
  (keeps us best without giving away more edge than necessary). This is
  capped at `mark price + TightenCapTicksAboveMark` ticks (50 by default,
  see `quoting.go`) so a stale/junk ask2 can't drag us to an unfillable
  price.

Every computed price is quantized to an exact multiple of the tick size for
the band it falls into (tick size varies by price band), because OKX rejects
amends at prices that aren't on a tick. A decision that would leave the price
unchanged is dropped rather than re-sent.

Assumes at most one open PUT sell order per instId; if more than one is
found for the same instId, that instId is skipped with a warning.

## Setup

```bash
cd okx_put_quoter
cp .env.example .env
# edit .env with real OKX_API_KEY / OKX_API_SECRET / OKX_API_PASSPHRASE
```

## Run

```bash
cd okx_put_quoter
go run . --dry-run --interval 5
```

Drop `--dry-run` once you've confirmed the logged decisions look correct.

## Test

```bash
cd okx_put_quoter
go vet ./... && go test ./...
```
