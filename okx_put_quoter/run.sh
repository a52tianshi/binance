#!/usr/bin/env bash
# Builds and runs okx_put_quoter in the background, logging to logs/.
# Usage: ./run.sh [extra flags passed to the binary, e.g. --dry-run --interval 5]
set -euo pipefail

cd "$(dirname "$0")"

mkdir -p logs
LOG_FILE="logs/okx_put_quoter_$(date +%Y%m%d_%H%M%S).log"
PID_FILE="okx_put_quoter.pid"

if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
  echo "Already running with PID $(cat "$PID_FILE"). Stop it first (kill \$(cat $PID_FILE))." >&2
  exit 1
fi

go build -o okx_put_quoter .

nohup ./okx_put_quoter "$@" >>"$LOG_FILE" 2>&1 &
echo $! > "$PID_FILE"

echo "started okx_put_quoter (PID $(cat "$PID_FILE")), logging to $LOG_FILE"
echo "tail -f $LOG_FILE   # to follow logs"
echo "kill \$(cat $PID_FILE)   # to stop"
