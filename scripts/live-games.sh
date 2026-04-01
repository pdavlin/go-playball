#!/bin/bash
#
# Open all live MLB games as panes in a single tmux window.
#
# Usage:
#   ./scripts/live-games.sh              # uses "playball" as session name
#   ./scripts/live-games.sh my-session   # custom session name

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PLAYBALL="${SCRIPT_DIR}/../go-playball"

if [ ! -x "$PLAYBALL" ]; then
  echo "Binary not found at $PLAYBALL"
  echo "Run 'go build -o go-playball ./cmd/go-playball/' first."
  exit 1
fi

SESSION="${1:-playball}"

live_output=$("$PLAYBALL" live 2>&1) || {
  echo "$live_output"
  exit 1
}

game_count=$(echo "$live_output" | wc -l | tr -d ' ')
echo "Found $game_count live game(s). Opening in tmux session '$SESSION'..."

first=true
while IFS=$'\t' read -r id matchup; do
  if $first; then
    tmux new-session -d -s "$SESSION" "$PLAYBALL game $id"
    first=false
  else
    tmux split-window -t "$SESSION" "$PLAYBALL game $id"
    tmux select-layout -t "$SESSION" tiled
  fi
done <<< "$live_output"

# Final tiled layout pass to even things out
tmux select-layout -t "$SESSION" tiled

# Attach if we're not already inside tmux
if [ -z "${TMUX:-}" ]; then
  tmux attach -t "$SESSION"
else
  tmux switch-client -t "$SESSION"
fi
