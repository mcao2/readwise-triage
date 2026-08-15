# Readwise Triage

A CLI tool for triaging Readwise Reader inbox items with LLM-assisted categorization.

## Features

- **Automated LLM Triage**: Press `T` to auto-triage items via any OpenAI-compatible API (OpenAI, Ollama, OpenRouter, etc.).
- **Interactive List View**: Navigate with vim-style keys (`j`/`k`), visual indicators for actions (🔥⏰📁), open articles in browser (`o`).
- **Quick Triage**: One-key shortcuts for actions (`r` = read, `l` = later, `a` = archive).
- **Tag Editing**: Edit AI-suggested tags inline with `e`.
- **Batch Operations**: Select multiple items with `x`/`space` to apply actions to all at once.
- **Persistence**: Triage decisions and lookback days are saved locally across sessions.

## Configuration

Set credentials via environment variables:

```sh
export READWISE_TOKEN="your-readwise-access-token"
export LLM_API_KEY="your-llm-api-key"
```

Or in `~/.config/readwise-triage/config.yaml`:

```yaml
readwise_token: ""
llm:
  base_url: ""    # e.g. https://api.openai.com
  api_key: ""
  model: ""       # e.g. gpt-4o-mini
inbox_days_ago: 7
```

## Keyboard Interactions

| Key | Context | Action |
|-----|---------|--------|
| `Enter` | Config | Start fetching items |
| `j` / `k` | Config | Adjust lookback days (-7 / +7) |
| `0`-`9` | Config | Type exact lookback days |
| `j` / `k` | Review | Navigate down / up |
| `r` | Review | Mark as **read** |
| `l` | Review | Mark as **later** |
| `a` | Review | Mark as **archive** |
| `d` | Review | Archive (same as `a`) |
| `x` / `space` | Review | Toggle selection |
| `o` | Review | Open URL in browser |
| `u` | Review | Update Readwise with triage |
| `e` | Review | Edit tags |
| `f` | Review | Fetch more items |
| `T` | Review | Auto-triage with LLM |
| `R` | Review | Refresh from Readwise |
| `?` | Any | Toggle help |
| `q` | Any | Quit |

## Build

```sh
go build -o readwise-triage ./cmd/readwise-triage
```
