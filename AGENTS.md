# Agentic Development Guide — Readwise Triage

## Commands

| Command | Description |
|---------|-------------|
| `make build` | Build binary |
| `make check` | fmt + vet + test |
| `make check-fast` | build + test |
| `go vet ./...` | Vet |
| `gofmt -w .` | Format |

## Architecture

```
cmd/readwise-triage/main.go    Entry point
internal/ui/
  model.go    State, Update(), key handlers, async cmds
  view.go     All View() rendering (configView→doneView, help)
  keys.go     KeyMap definitions (12 bindings)
  list_view.go  Custom table (Column struct, [][]string rows, no bubbles/table)
  styles.go   Single theme (DefaultTheme), Styles struct
internal/config/
  config.go       Config (yaml + env), LLMConfig
  triage_store.go JSON file store for triage decisions
internal/readwise/  Readwise Reader V3 API client
internal/triage/    LLM client (OpenAI-compatible), prompt, parser, result types
```

## Config

| Key | Env var | Default |
|-----|---------|---------|
| `readwise_token` | `READWISE_TOKEN` | — |
| `llm.api_key` | `LLM_API_KEY` | — |
| `llm.base_url` | `LLM_BASE_URL` | — |
| `llm.model` | `LLM_MODEL` | — |
| `llm.max_tokens` | `LLM_MAX_TOKENS` | `8192` |
| `inbox_days_ago` | `INBOX_DAYS_AGO` | `7` |
| — | `READWISE_TRIAGE_CONFIG` | `~/.config/readwise-triage/config.yaml` |

Env vars override config file. OpenAI-compatible only (no Anthropic).

## Triage Actions

| Action | Key | Readwise effect |
|--------|-----|-----------------|
| `read_now` | `r` | Stays in inbox |
| `later` | `l` | `location: later` |
| `archive` | `a` / `d` | `location: archive` |

No `needs_review`, no `delete` (archive covers both).

## Go Conventions

- Imports: stdlib / external / internal (3 groups)
- Pointer receivers for Model methods
- Wrap errors: `fmt.Errorf("context: %w", err)`
- `runewidth` for emoji padding/truncation
- `strings.Contains` for view tests (ANSI codes make exact matching fragile)

## TUI Rules

- `View()` dispatches by state, non-review views centered via `lipgloss.Place`
- Review view composes: header + table + detail(4 lines) + status + footer
- Pad review output to `m.height` lines for clean repaints
- Never render at exact terminal width; use `m.width - 1`
- Tag editor overlays on review view (split lines, stamp centered popup)
- `?` toggles compact footer ↔ full help
- macOS: Option+Arrow = `alt+b`/`alt+f` (not KeyLeft/KeyRight+Alt)

## Persistence

- Triage store: `~/.config/readwise-triage/triage_store.json` (JSON, not SQLite)
- Config: `~/.config/readwise-triage/config.yaml`
- Writes are immediate (no explicit Save needed for triage store)

## Testing

- Table-driven tests for parsing/string logic
- `mockHTTPClient` in readwise_test.go for API testing
- `Init()` returns `tea.Batch(spinner.Tick, startFetching())` — expect non-nil cmd
- Test state transitions by sending messages (ItemsLoadedMsg, ErrorMsg, etc.)

## Pre-commit Hook

`.githooks/pre-commit` checks `gofmt` and `go vet`. Run `make setup` to enable.
