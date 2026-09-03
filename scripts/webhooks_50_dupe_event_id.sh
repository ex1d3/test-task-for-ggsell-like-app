#!/usr/bin/env bash
set -euo pipefail

ORDER_ID="${1:-}"
EVENT_ID="${2:-}"
APP_URL="${APP_URL:-http://localhost:8080}"
N="${N:-50}"

if [[ -z "$ORDER_ID" || -z "$EVENT_ID" ]]; then
  echo "usage: $0 <order_id> <event_id>" >&2
  echo "env: APP_URL (default: http://localhost:8080), N (default: 50)" >&2
  exit 2
fi

for i in $(seq 1 "$N"); do
  curl -sS -o /dev/null -X POST "$APP_URL/webhooks" \
    -H 'Content-Type: application/json' \
    --data-binary "{\"event_id\":\"$EVENT_ID\",\"order_id\":$ORDER_ID,\"status\":\"paid\",\"amount\":1,\"currency\":\"RUB\"}" &
done

wait
echo "sent $N duplicate event_id webhooks for order_id=$ORDER_ID event_id=$EVENT_ID"

