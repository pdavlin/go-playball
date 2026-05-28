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

## Scouting reports & postgame recaps (optional)

Configure a provider and press `r` on a game in the schedule view. The
behavior is context-sensitive:

- **Preview games** → streams a short scouting report. Cached at
  `~/.config/go-playball/scouting/{gamePk}.json`.
- **Final games** → streams a five-section postgame recap (How It Was
  Won, Turning Point, On the Mound, Top Performer, Bullpen). Cached at
  `~/.config/go-playball/recap/{gamePk}.json`.
- **Live games** → no report; the help-bar entry is hidden.

Press `R` (shift-r) inside the modal to bust the cache and re-stream.

### Supported providers

| `scouting.provider` | Required keys                                  | Notes                                                       |
|---------------------|------------------------------------------------|-------------------------------------------------------------|
| `anthropic`         | `api_key`, `model`                             | Anthropic Messages API. Model e.g. `claude-haiku-4-5-20251001`. |
| `openrouter`        | `api_key`, `model`                             | OpenRouter chat-completion API. Model e.g. `anthropic/claude-3.5-haiku` or `openai/gpt-4o-mini`. |
| `openai-compatible` | `base_url`, `model` (api_key optional)         | Any OpenAI-compatible server: OpenAI, Groq, Together, Ollama, LM Studio. `base_url` is the server root (do not include `/v1/chat/completions`). |

Anthropic:

```bash
go-playball config scouting.provider anthropic
go-playball config scouting.api_key   "sk-ant-..."
go-playball config scouting.model     "claude-haiku-4-5-20251001"
```

OpenRouter:

```bash
go-playball config scouting.provider openrouter
go-playball config scouting.api_key   "sk-or-..."
go-playball config scouting.model     "anthropic/claude-3.5-haiku"
```

Local Ollama (OpenAI-compatible):

```bash
go-playball config scouting.provider openai-compatible
go-playball config scouting.base_url  "http://localhost:11434"
go-playball config scouting.model     "llama3.2"
# api_key intentionally unset
```

Validate credentials and connectivity without launching the TUI:

```bash
go-playball scouting test
```

Without a provider configured, the feature is invisible: no help-bar entry,
no key binding, no network calls.
