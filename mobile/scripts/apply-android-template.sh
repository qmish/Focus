#!/usr/bin/env bash
# Применяет mobile/templates/AndroidManifest.xml поверх сгенерированного
# tauri-init файла. Запускается в CI после `tauri android init`.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SRC="$REPO_ROOT/mobile/templates/AndroidManifest.xml"
DST="$REPO_ROOT/mobile/src-tauri/gen/android/app/src/main/AndroidManifest.xml"

if [[ ! -f "$SRC" ]]; then
  echo "Template not found: $SRC" >&2
  exit 1
fi
if [[ ! -d "$(dirname "$DST")" ]]; then
  echo "Generated Android project not found at $(dirname "$DST"). Run 'tauri android init' first." >&2
  exit 2
fi

cp -v "$SRC" "$DST"
echo "✓ AndroidManifest applied"
