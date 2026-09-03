#!/usr/bin/env bash
set -euo pipefail

ORDER_ID="${1:-}"
APP_URL="${APP_URL:-http://localhost:8080}"
N="${N:-50}"

if [[ -z "$ORDER_ID" ]]; then
  echo "usage: $0 <order_id>" >&2
  echo "env: APP_URL (default: http://localhost:8080), N (default: 50)" >&2
  exit 2
fi

base="evt_race_${ORDER_ID}_$(date +%s)_$$"

for i in $(seq 1 "$N"); do
  curl -sS -o /dev/null -X POST "$APP_URL/webhooks" \
    -H 'Content-Type: application/json' \
    --data-binary "{\"event_id\":\"${base}_${i}\",\"order_id\":$ORDER_ID,\"status\":\"paid\",\"amount\":1,\"currency\":\"RUB\"}" &
done

wait
echo "sent $N paid webhooks for order_id=$ORDER_ID"

