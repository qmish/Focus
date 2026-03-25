#!/bin/bash

set -euo pipefail

API_URL="${1:-https://api-stage.company.com}"

pass=0
fail=0

check_contains() {
  local name="$1"
  local haystack="$2"
  local needle="$3"
  if [[ "$haystack" == *"$needle"* ]]; then
    echo "[PASS] $name"
    pass=$((pass + 1))
  else
    echo "[FAIL] $name (missing: $needle)"
    fail=$((fail + 1))
  fi
}

echo "Jitsi branding regression:"
echo "  API: $API_URL"

status_code="$(curl -s -o /tmp/focus-branding.json -w "%{http_code}" "$API_URL/api/v1/branding/jitsi")"
if [ "$status_code" != "200" ]; then
  echo "[FAIL] branding endpoint status expected=200 got=$status_code"
  fail=$((fail + 1))
else
  echo "[PASS] branding endpoint status=200"
  pass=$((pass + 1))
fi

payload="$(cat /tmp/focus-branding.json)"
check_contains "has appName" "$payload" "\"appName\""
check_contains "has dynamicBrandingUrl" "$payload" "\"dynamicBrandingUrl\""
check_contains "has logoImageUrl" "$payload" "\"logoImageUrl\""
check_contains "has faviconUrl" "$payload" "\"faviconUrl\""
check_contains "has backgroundImageUrl" "$payload" "\"backgroundImageUrl\""
check_contains "has customTheme" "$payload" "\"customTheme\""
check_contains "has customIcons" "$payload" "\"customIcons\""
check_contains "uses pics assets" "$payload" "/pics/"

echo "Summary: pass=$pass fail=$fail"
if [ "$fail" -gt 0 ]; then
  exit 1
fi
