#!/usr/bin/env bash
# Применяет mobile/templates/Info.plist поверх сгенерированного tauri-init файла.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SRC="$REPO_ROOT/mobile/templates/Info.plist"

# Имя iOS-проекта определяется как <product_name>_iOS, где product_name из tauri.conf.json
DST_DIR="$REPO_ROOT/mobile/src-tauri/gen/apple"
if [[ ! -d "$DST_DIR" ]]; then
  echo "Generated Apple project not found at $DST_DIR. Run 'tauri ios init' first." >&2
  exit 2
fi

PROJECT_DIR="$(find "$DST_DIR" -maxdepth 1 -type d -name '*_iOS' | head -n1)"
if [[ -z "$PROJECT_DIR" ]]; then
  echo "iOS project directory (*_iOS) not found in $DST_DIR" >&2
  exit 3
fi

DST="$PROJECT_DIR/Info.plist"
cp -v "$SRC" "$DST"
echo "✓ Info.plist applied to $DST"
