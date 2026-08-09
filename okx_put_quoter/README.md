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
go test ./okx_put_quoter/...
```
