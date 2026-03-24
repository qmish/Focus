#!/bin/bash
# Security scan script for Focus API
# Requires: OWASP ZAP installed

TARGET_URL=${1:-http://localhost:8080}
REPORT_DIR="tests/security/reports"

mkdir -p $REPORT_DIR

echo "Starting OWASP ZAP security scan..."
echo "Target: $TARGET_URL"

# Baseline scan
zap-baseline.py \
  -t $TARGET_URL \
  -r $REPORT_DIR/baseline_report.html \
  -J $REPORT_DIR/baseline_report.json \
  -a \
  -j

echo "Baseline scan completed. Report: $REPORT_DIR/baseline_report.html"

# Full scan (optional, takes longer)
if [ "$2" == "--full" ]; then
  echo "Starting full security scan..."
  zap-full-scan.py \
    -t $TARGET_URL \
    -r $REPORT_DIR/full_report.html \
    -J $REPORT_DIR/full_report.json
  
  echo "Full scan completed. Report: $REPORT_DIR/full_report.html"
fi

# API scan with OpenAPI spec
if [ -f "API_Go/docs/swagger.json" ]; then
  echo "Starting API security scan..."
  zap-api-scan.py \
    -t API_Go/docs/swagger.json \
    -f openapi \
    -r $REPORT_DIR/api_report.html \
    -J $REPORT_DIR/api_report.json
  
  echo "API scan completed. Report: $REPORT_DIR/api_report.html"
fi

echo "Security scan completed!"
