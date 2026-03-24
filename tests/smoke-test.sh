#!/bin/bash
# Smoke tests for production deployment

set -e

API_URL=${1:-http://localhost:8080}
FRONTEND_URL=${2:-http://localhost:3000}
ADMIN_URL=${3:-http://localhost:3001}

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║          Focus Messenger Smoke Tests                      ║"
echo "╚═══════════════════════════════════════════════════════════╝"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS=0
FAIL=0

# Test function
test_endpoint() {
  local name=$1
  local url=$2
  local expected_status=$3
  
  echo -n "Testing $name... "
  
  status=$(curl -s -o /dev/null -w "%{http_code}" "$url")
  
  if [ "$status" -eq "$expected_status" ]; then
    echo -e "${GREEN}✓ PASS${NC} ($status)"
    ((PASS++))
  else
    echo -e "${RED}✗ FAIL${NC} (expected $expected_status, got $status)"
    ((FAIL++))
  fi
}

# API Tests
echo ""
echo "=== API Tests ==="
test_endpoint "Health Check" "$API_URL/health" 200
test_endpoint "Readiness Check" "$API_URL/ready" 200
test_endpoint "Auth Login (redirect)" "$API_URL/api/v1/auth/login" 302
test_endpoint "Rooms (unauthorized)" "$API_URL/api/v1/rooms" 401
test_endpoint "Admin Stats (unauthorized)" "$API_URL/api/v1/admin/stats" 401

# Frontend Tests
echo ""
echo "=== Frontend Tests ==="
test_endpoint "Frontend Load" "$FRONTEND_URL/" 200

# Admin Tests
echo ""
echo "=== Admin Panel Tests ==="
test_endpoint "Admin Load" "$ADMIN_URL/" 200

# Summary
echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║                    Test Summary                           ║"
echo "╠═══════════════════════════════════════════════════════════╣"
printf "║  Passed: %-10d Failed: %-10d                 ║\n" $PASS $FAIL
echo "╚═══════════════════════════════════════════════════════════╝"

if [ $FAIL -gt 0 ]; then
  echo -e "${RED}Smoke tests FAILED${NC}"
  exit 1
else
  echo -e "${GREEN}Smoke tests PASSED${NC}"
  exit 0
fi
