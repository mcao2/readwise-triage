# Readwise Triage

A CLI tool for triaging Readwise Reader inbox items with LLM-assisted or manual categorization.

## Features

- **LLM-Assisted Workflow**:
  - Export untriaged items as JSON with a specialized prompt (`e`).
  - Paste to any LLM of your choice for categorization.
  - Import results back into the TUI (`i`).
- **Persistence**: Triage decisions and preferences (location, lookback days, theme) are saved locally across sessions.
- **Interactive List View**:
  - Navigate with vim-style keys (`j`/`k`).
  - Visual indicators for actions (🔥⏰📁) and priority (🔴🟡🟢).
  - Open articles directly in your browser (`o`).
- **Quick Triage**: One-key shortcuts for actions (`r`, `l`, `a`) and priorities (`1`, `2`, `3`).
- **Batch Operations**: Select multiple items with `x`/`space` to apply actions to all at once.
- **Feed Support**: Triage RSS/feed items in addition to inbox; toggle with `h`/`l` on the config screen.
- **Progressive Fetch**: Increase the lookback window to find older items (`f`), with independent windows per location.

## Keyboard Interactions

| Key | Context | Action |
|-----|---------|--------|
| `Enter` | Config | Start fetching items |
| `h` / `l` | Config | Toggle location: **Inbox** / **Feed** |
| `j` / `k` | Config | Adjust lookback days (-7 / +7) |
| `t` | Config | Cycle through color themes |
| `j` / `k` | Review | Navigate down / up |
| `x` / `Space` | Review | Toggle selection (Batch mode) |
| `r` | Review | Set action: **Read Now** (keeps in inbox, adds tag) |
| `l` | Review | Set action: **Later** (moves to Later) |
| `a` | Review | Set action: **Archive** (moves to Archive) |
| `d` | Review | Set action: **Delete** (moves to Archive) |
| `n` | Review | Set action: **Needs Review** (flags for human review) |
| `1` / `2` / `3` | Review | Set priority: **High** / **Medium** / **Low** |
| `Enter` | Review | **Edit Tags** (comma-separated, applies to selection in batch mode) |
| `e` | Review | **Export** items to clipboard (Selected items if active, else untriaged) |
| `i` | Review | **Import** triage results from clipboard |
| `o` | Review | **Open** URL(s) in default browser (Selected items if active, else current) |
| `f` | Review | **Fetch More** (adds 7 days to lookback window) |
| `R` | Review | **Refresh** from Readwise (re-fetch with current lookback) |
| `u` | Review | **Update** Readwise (Apply changes to Selected items if active, else all triaged) |
| `Esc` | Review | **Back** to config screen |
| `q` / `Ctrl+C` | Global | Quit |
| `?` | Global | Toggle help |

## Requirements

- Go 1.24+
- A [Readwise Reader](https://readwise.io/) account and API token ([get one here](https://readwise.io/access_token))

## Installation

```bash
# Clone the repository
git clone https://github.com/mcao2/readwise-triage.git
cd readwise-triage

# Enable pre-commit hooks (gofmt + go vet)
make setup

# Build the binary
go build -o readwise-triage ./cmd/readwise-triage

# Install to $GOPATH/bin
go install ./cmd/readwise-triage
```

## Configuration

You can configure `readwise-triage` using either **environment variables** or a **config file**. Environment variables take precedence over config file values.

### Config File

The application automatically creates a config directory at `~/.config/readwise-triage/`. You can create `config.yaml` there:

```yaml
# Readwise Triage Configuration
# Get your token at: https://readwise.io/access_token

# Required: Your Readwise API token
readwise_token: "your_token_here"

# Optional: Default number of days to fetch for inbox (default: 7)
inbox_days_ago: 7

# Optional: Default number of days to fetch for feed (default: 7)
feed_days_ago: 7

# Optional: Color theme (default, catppuccin, dracula, nord, gruvbox)
theme: "default"

# Optional: Last-used location, remembered across sessions (new or feed)
location: "new"
```

### Persistence

Triage decisions are saved to `~/.config/readwise-triage/triage.db` (SQLite). Preferences (location, lookback days, theme) are saved to `config.yaml`. This allows you to:
1. Re-open the tool and see your previous decisions.
2. Only export "raw" items that haven't been triaged yet.

## Workflow

1. **Configure**: Choose location (Inbox or Feed), adjust lookback days, pick a theme.
2. **Fetch**: Load items from Readwise.
2. **Export (`e`)**: Copy untriaged items and the triage prompt to your clipboard.
3. **LLM**: Paste into any LLM (ChatGPT, Claude, Gemini, etc.), then copy the resulting JSON array.
4. **Import (`i`)**: Paste the results back into the tool.
5. **Review**: Manually adjust any items or use batch selection (`x`).
6. **Update (`u`)**: Apply all triaged changes to your Readwise Reader account.


## Project Structure

```
.
├── cmd/readwise-triage/    # Entry point
├── internal/
│   ├── readwise/          # Readwise API client
│   ├── triage/            # LLM integration
│   ├── ui/                # TUI components
│   └── config/            # Configuration
└── go.mod
```

## Tech Stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Huh](https://github.com/charmbracelet/huh) - Forms
- [Bubbles](https://github.com/charmbracelet/bubbles) - Components

## License

MIT
