#!/bin/bash

set -euo pipefail

API_URL="${1:-https://api-stage.company.com}"
CHAT_URL="${2:-https://chat-stage.company.com}"
ADMIN_URL="${3:-https://admin-stage.company.com}"

pass=0
fail=0

check_status() {
  local name="$1"
  local url="$2"
  local expected="$3"

  code="$(curl -s -o /dev/null -w "%{http_code}" "$url")"
  if [ "$code" = "$expected" ]; then
    echo "[PASS] $name ($code)"
    pass=$((pass + 1))
  else
    echo "[FAIL] $name expected=$expected got=$code"
    fail=$((fail + 1))
  fi
}

echo "Stage smoke check:"
echo "  API:   $API_URL"
echo "  CHAT:  $CHAT_URL"
echo "  ADMIN: $ADMIN_URL"

check_status "API health" "$API_URL/health" "200"
check_status "API ready" "$API_URL/ready" "200"
check_status "Auth login redirect" "$API_URL/api/v1/auth/login" "302"
check_status "Rooms unauthorized" "$API_URL/api/v1/rooms" "401"
check_status "Admin stats unauthorized" "$API_URL/api/v1/admin/stats" "401"
check_status "Chat frontend reachable" "$CHAT_URL/" "200"
check_status "Admin frontend reachable" "$ADMIN_URL/" "200"

echo "Summary: pass=$pass fail=$fail"
if [ "$fail" -gt 0 ]; then
  exit 1
fi
