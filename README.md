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

Set the following environment variables:

```bash
export READWISE_TOKEN="your_readwise_token_here"
export PERPLEXITY_API_KEY="your_perplexity_api_key_here"  # Optional for manual mode
```

Get your Readwise token at: https://readwise.io/access_token

## Usage

```bash
# Run the application
./readwise-triage
```

### Keyboard Shortcuts

**Global**:
- `q` / `Ctrl+C` - Quit
- `?` - Toggle help

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
