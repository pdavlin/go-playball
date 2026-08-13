---
name: observe-tui
description: Visually verify TUI rendering and layout changes by capturing real frames of go-playball against live MLB games in tmux. Use when changing any UI code (internal/ui), when asked to check how something looks, when investigating layout shift or alignment bugs, or before merging visual changes.
---

# Observe the TUI against real games

Never reason about lipgloss output from code alone. This repo has a
capture rig that runs the real binary against real game data and
produces color-faithful HTML pages of actual rendered frames. Code
review missed a permanently wrapped help bar and an empty title line;
frame captures found both in minutes.

## Quick start

```bash
# List today's games with IDs and states (Live / Preview / Final)
./scripts/observe.sh

# Capture the initial frame of a game, then one frame per keystroke
./scripts/observe.sh <gameID> 2 3 4
```

The script rebuilds the binary every run (a stale binary silently
shows old behavior), launches it in a detached tmux pane, sends each
keystroke with a settle wait, and converts captures to one HTML page
via `scripts/ansi2html.py`. The page path is printed and opened.

Defaults: 120x40 pane, 3s settle. Override with `OBS_COLS`, `OBS_ROWS`,
`OBS_WAIT`. Narrow widths (e.g. `OBS_COLS=100`) are where wrap and
truncation bugs live — always test at least one narrow size.

Pick game state to match the code under test: Preview games exercise
the pregame tabs, Live games the live tabs, Final games the box score.
Evening Preview games are available during the day; Live games during
game windows (afternoon/evening CT in season).

## Manual patterns the script doesn't cover

Drive tmux directly when you need more than keystroke-then-capture:

```bash
tmux new-session -d -s obs -x 120 -y 40 "./go-playball game <id>"
sleep 5
tmux send-keys -t obs <key>
tmux capture-pane -t obs -e -p > frame.ansi   # -e keeps colors
tmux kill-session -t obs
```

**Layout-stability probe** (catching shift over time, e.g. across
data-refresh ticks): capture plain frames (`-p` without `-e`) in a
timed loop, then track where a landmark row sits in each frame:

```bash
for i in $(seq 0 11); do
  tmux capture-pane -t obs -p > "stab_$i.txt"; sleep 2
done
# then: grep -n for a help-bar keyword in each frame; the row number
# must not move between frames
```

**Before/after comparison** (proving a fix changed rendering): build
the pre-change binary from a temp worktree and run the same probe on
both:

```bash
git worktree add /tmp/oldtree <commit>
(cd /tmp/oldtree && go build -o /tmp/go-playball-old ./cmd/go-playball)
git worktree remove --force /tmp/oldtree
```

**Reading frames yourself**: strip ANSI before inspecting alignment:

```bash
python3 -c "import re,sys;print(re.sub(r'\x1b\[[0-9;]*m','',sys.stdin.read()))" < frame.ansi
```

## Sharing results

Build a labeled multi-frame page and send it so the user can see it:

```bash
python3 scripts/ansi2html.py out.html "Label A" a.ansi "Label B" b.ansi
```

Then send `out.html` with SendUserFile (`display: render`).

## Gotchas

- **Never run `tmux kill-server`.** It kills every session on the
  default socket — including the user's own tmux, if they have one
  running outside this rig — not just the observation session. This
  has happened: an agent used it as cleanup and took down an
  unrelated live session. Only ever kill the named session:
  `tmux kill-session -t "$SESSION"`.
- Always `tmux kill-session` when done (use a trap); orphaned sessions
  keep polling the MLB API.
- Frame captures are byte strings but terminal layout is columns:
  compare rune positions, not byte offsets — box-drawing glyphs like
  `─` are 3 bytes wide.
- The app takes ~5s after launch to fetch and render; capture too early
  and you get a blank or loading frame.
- Async tab data (pregame fetches) needs 2-4s after the keystroke
  before the skeleton spinner resolves into content.
- Game IDs come from `statsapi.mlb.com/api/v1/schedule?sportId=1&date=YYYY-MM-DD`
  (the no-arg script mode wraps this).
