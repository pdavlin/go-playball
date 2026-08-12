#!/bin/bash
#
# Capture rendered frames of the TUI against a real game, without a
# terminal of your own: runs the binary in a fixed-size tmux pane,
# drives it with keystrokes, and converts the captures to a single
# color-faithful HTML page.
#
# Usage:
#   ./scripts/observe.sh                    # list today's games + IDs
#   ./scripts/observe.sh <gameID>           # capture the initial frame
#   ./scripts/observe.sh <gameID> 2 3 4     # also capture a frame after
#                                           # each listed keystroke
#
# Each keystroke argument is sent via tmux send-keys, followed by a
# short settle wait and a capture. The HTML page path is printed and
# opened when possible.
#
# Env overrides: OBS_COLS (120), OBS_ROWS (40), OBS_WAIT seconds (3).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PLAYBALL="${SCRIPT_DIR}/../go-playball"
COLS="${OBS_COLS:-120}"
ROWS="${OBS_ROWS:-40}"
WAIT="${OBS_WAIT:-3}"
SESSION="observe-$$"

if [ $# -eq 0 ]; then
  date_str=$(date +%Y-%m-%d)
  curl -s "https://statsapi.mlb.com/api/v1/schedule?sportId=1&date=${date_str}" |
    python3 -c '
import json, sys
d = json.load(sys.stdin)
for date in d.get("dates", []):
    for g in date["games"]:
        print("%-8s %-8s %-12s %s @ %s" % (
            g["gamePk"],
            g["status"]["abstractGameState"],
            g["status"]["detailedState"],
            g["teams"]["away"]["team"]["name"],
            g["teams"]["home"]["team"]["name"],
        ))
'
  exit 0
fi

GAME_ID="$1"
shift

# Always rebuild: the rig exists to verify current code, and a stale
# binary at the repo root would silently show old behavior.
echo "Building go-playball..."
(cd "$SCRIPT_DIR/.." && go build -o go-playball ./cmd/go-playball/)

OUT_DIR=$(mktemp -d "${TMPDIR:-/tmp}/observe-${GAME_ID}-XXXX")
trap 'tmux kill-session -t "$SESSION" 2>/dev/null || true' EXIT

tmux new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
  "$PLAYBALL game $GAME_ID"
sleep 5

frames=()
tmux capture-pane -t "$SESSION" -e -p > "$OUT_DIR/initial.ansi"
frames+=("initial" "$OUT_DIR/initial.ansi")

for key in "$@"; do
  tmux send-keys -t "$SESSION" "$key"
  sleep "$WAIT"
  safe=$(echo "$key" | tr -c 'A-Za-z0-9' '_')
  tmux capture-pane -t "$SESSION" -e -p > "$OUT_DIR/key_${safe}.ansi"
  frames+=("after key: $key" "$OUT_DIR/key_${safe}.ansi")
done

tmux kill-session -t "$SESSION"

HTML="$OUT_DIR/observe-${GAME_ID}.html"
python3 "$SCRIPT_DIR/ansi2html.py" "$HTML" "${frames[@]}"
echo "$HTML"
command -v open >/dev/null && open "$HTML"
