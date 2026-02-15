# Readwise Triage

A CLI tool for triaging Readwise Reader inbox items with AI-powered or manual categorization.

## Features

- **Two Triage Modes**:
  - **LLM Auto-Triage**: Uses Perplexity AI to automatically categorize items
  - **Manual Triage**: Manually review and categorize items using keyboard shortcuts

- **Interactive List View**:
  - Navigate with vim-style keys (j/k)
  - Visual indicators for actions (🔥⏰📁🗑️) and priority (🔴🟡🟢)
  - Multi-select support for batch operations

- **Quick Actions**:
  - `r` - Set action: read_now
  - `l` - Set action: later  
  - `a` - Set action: archive
  - `d` - Set action: delete
  - `1/2/3` - Set priority: high/medium/low

- **Batch Operations**:
  - Select multiple items with `x`
  - Apply changes to all selected items
  - Filter by current action

- **Full Edit Form**:
  - Edit action, priority, reason, and tags
  - Built with Huh forms for beautiful TUI experience

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

Create a config file at `~/.config/readwise-triage/config.yaml`:

```yaml
# Readwise Triage Configuration
# Get your token at: https://readwise.io/access_token

# Required: Your Readwise API token
readwise_token: "your_token_here"

# Optional: Perplexity API key for LLM auto-triage
# Get your key at: https://www.perplexity.ai/settings/api
perplexity_api_key: "your_api_key_here"

# Optional: Default number of days to fetch (default: 7)
default_days_ago: 7

# Optional: Color theme (default, catppuccin, dracula, nord, gruvbox)
theme: "default"
```

### Themes

Readwise Triage comes with several built-in color themes:

- **default** - Clean purple/blue theme
- **catppuccin** - Soft pastel colors  
- **dracula** - Popular dark theme
- **nord** - Arctic-inspired colors
- **gruvbox** - Retro groove colors

Press `t` in the config screen to cycle through themes, or set it in your config file.

To use a custom config file location, set the `READWISE_TRIAGE_CONFIG` environment variable:

```bash
export READWISE_TRIAGE_CONFIG="/path/to/your/config.yaml"
```

### Environment Variables

Alternatively, you can use environment variables (which override config file values):

```bash
export READWISE_TOKEN="your_readwise_token_here"
export PERPLEXITY_API_KEY="your_perplexity_api_key_here"  # Optional for manual mode
export DEFAULT_DAYS_AGO=7  # Optional
```

## Usage

```bash
# Run the application
./readwise-triage
```

### Keyboard Shortcuts

**Global**:
- `q` / `Ctrl+C` - Quit
- `?` - Toggle help

**Config Screen**:
- `Enter` - Start fetching items
- `m` - Toggle between LLM/Manual mode
- `t` - Cycle through color themes

**Navigation**:
- `j` / `↓` - Move down
- `k` / `↑` - Move up
- `h` / `←` - Previous screen
- `l` / `→` / `Enter` - Select/Open

**In Review Mode**:
- `x` - Toggle selection
- `r` - Set action: read_now
- `l` - Set action: later
- `a` - Set action: archive
- `d` - Set action: delete
- `1` - Set priority: high
- `2` - Set priority: medium
- `3` - Set priority: low
- `Enter` - Edit item details
- `b` - Batch edit selected items
- `u` - Update Readwise

## Workflow

1. **Start**: Choose between LLM Auto-Triage or Manual Triage mode
2. **Fetch**: Load inbox items from Readwise
3. **Triage** (if auto mode): AI categorizes items
4. **Review**: Navigate and categorize items
5. **Edit**: Fine-tune individual items or batch edit
6. **Update**: Apply changes back to Readwise

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
