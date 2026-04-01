# go-playball

A terminal-based MLB game viewer built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

[![Nightly Build](https://github.com/pdavlin/go-playball/actions/workflows/build-main.yml/badge.svg)](https://github.com/pdavlin/go-playball/releases/tag/nightly)

## Install

### Homebrew (macOS / Linux)

```bash
brew install pdavlin/go-playball/go-playball
```

### Scoop (Windows)

```powershell
scoop bucket add go-playball https://github.com/pdavlin/scoop-go-playball
scoop install go-playball
```

### Download

Grab the latest stable release or nightly build from [GitHub Releases](https://github.com/pdavlin/go-playball/releases).

Available platforms: macOS (Intel & Apple Silicon), Linux (amd64 & arm64), Windows (amd64).

### Build from Source

```bash
go install github.com/pdavlin/go-playball/cmd/go-playball@latest
```

Or clone and build:

```bash
git clone https://github.com/pdavlin/go-playball.git
cd go-playball
go build -o go-playball ./cmd/go-playball/
```

## Usage

```
go-playball                            Start the application
go-playball live                       List live games (tab-separated: ID, matchup)
go-playball game <game_id>             Launch directly into a specific game
go-playball config                     Show current configuration
go-playball config <key> <value>       Set a configuration value
go-playball config --unset <key>       Reset a key to its default
go-playball help                       Show help message
go-playball version                    Show version information
```

## Keyboard Shortcuts

| Key             | Action                          |
|-----------------|---------------------------------|
| `c`             | Switch to schedule view         |
| `s`             | Switch to standings view        |
| `↑`/`↓` or `j`/`k` | Navigate items             |
| `Enter`         | View selected game              |
| `p`             | Previous day (schedule view)    |
| `n`             | Next day (schedule view)        |
| `t`             | Today (schedule view)           |
| `g`             | Scroll to top (game view)       |
| `G`             | Scroll to bottom (game view)    |
| `q`             | Quit                            |

## Configuration

Configuration is stored at `~/.config/go-playball/config.json`.

```bash
# Add a favorite team
go-playball config favorite_teams "New York Yankees"

# Set a theme color
go-playball config colors.primary "#00D9FF"

# Set an event color
go-playball config event_colors.walk "#00FF00"

# Reset to default
go-playball config --unset event_colors.walk
```

Run `go-playball help` for a full list of configuration keys.
