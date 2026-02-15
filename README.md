# Readwise Triage

A CLI tool for triaging Readwise Reader inbox items with LLM-assisted or manual categorization.

## Features

- **Manual LLM Workflow**:
  - Export untriaged items as JSON with a specialized prompt (`e`).
  - Paste to LLM (e.g., Perplexity/GPT) for categorization.
  - Import results back into the TUI (`i`).
- **Persistence**: Triage decisions are saved locally across sessions.
- **Interactive List View**:
  - Navigate with vim-style keys (`j`/`k`).
  - Visual indicators for actions (🔥⏰📁) and priority (🔴🟡🟢).
  - Open articles directly in your browser (`o`).
- **Quick Triage**: One-key shortcuts for actions (`r`, `l`, `a`) and priorities (`1`, `2`, `3`).
- **Batch Operations**: Select multiple items with `x`/`space` to apply actions to all at once.
- **Progressive Fetch**: Increase the lookback window to find older items (`f`).

## Keyboard Interactions

| Key | Context | Action |
|-----|---------|--------|
| `Enter` | Config | Start fetching items |
| `m` | Config | Toggle between LLM/Manual mode |
| `t` | Config | Cycle through color themes |
| `j` / `k` | Review | Navigate down / up |
| `x` / `Space` | Review | Toggle selection (Batch mode) |
| `r` | Review | Set action: **Read Now** (keeps in inbox, adds tag) |
| `l` | Review | Set action: **Later** (moves to Later) |
| `a` | Review | Set action: **Archive** (moves to Archive) |
| `1` / `2` / `3` | Review | Set priority: **High** / **Medium** / **Low** |
| `e` | Review | **Export** untriaged items + prompt to clipboard |
| `i` | Review | **Import** triage results from clipboard |
| `o` | Review | **Open** URL in default browser |
| `f` | Review | **Fetch More** (adds 7 days to lookback window) |
| `u` | Review | **Update** Readwise (apply all triaged changes) |
| `q` / `Ctrl+C` | Global | Quit |
| `?` | Global | Toggle help |

## Installation

```bash
# Clone the repository
git clone https://github.com/mcao2/readwise-triage.git
cd readwise-triage

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

# Optional: Default number of days to fetch (default: 7)
default_days_ago: 7

# Optional: Color theme (default, catppuccin, dracula, nord, gruvbox)
theme: "default"
```

### Persistence

Triage decisions are saved to `~/.config/readwise-triage/triage_store.json`. This allows you to:
1. Re-open the tool and see your previous decisions.
2. Only export "raw" items that haven't been triaged yet.

## Workflow

1. **Fetch**: Load inbox items from Readwise.
2. **Export (`e`)**: Copy untriaged items and the triage prompt to your clipboard.
3. **LLM**: Paste into Perplexity or ChatGPT, then copy the resulting JSON array.
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
