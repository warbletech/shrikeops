#!/usr/bin/env bash
# report-viewer.sh — Open a scan JSON report in the HTML viewer
# Usage: ./scripts/report-viewer.sh [report.json]
#
# Without a file arg, opens the viewer with the drop zone.
# With a file arg, injects the JSON into the viewer via query param.

set -euo pipefail

VIEWER="$(dirname "$0")/../reports/viewer.html"
FILE="${1:-}"

if [[ -n "$FILE" && -f "$FILE" ]]; then
  # Inline the JSON as a script tag in a temp HTML page that auto-loads it
  TMP=$(mktemp /tmp/shrike-report-XXXX.html)
  JSON=$(cat "$FILE" | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin)))" 2>/dev/null || cat "$FILE")
  sed "s|</script>|; window.addEventListener('DOMContentLoaded', () => render($JSON));</script>|" "$VIEWER" > "$TMP"
  echo "📊 Opening report: $FILE"
  open "$TMP" 2>/dev/null || xdg-open "$TMP" 2>/dev/null || start "$TMP" 2>/dev/null || echo "Open manually: $TMP"
else
  echo "📊 Opening ShrikeOps Report Viewer"
  open "$VIEWER" 2>/dev/null || xdg-open "$VIEWER" 2>/dev/null || start "$VIEWER" 2>/dev/null || echo "Open manually: $VIEWER"
fi
