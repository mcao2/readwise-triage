# Readwise Triage

A TUI for triaging Readwise Reader inbox items with LLM-assisted categorization.

## Build

| Command | Description |
|---------|-------------|
| `make build` | Build binary |
| `make run` | Run via `go run` |
| `make check` | fmt + vet + test |
| `make check-fast` | build + test |
| `make install` | Build + copy to `$GOPATH/bin` |
| `make clean` | Remove binary |

## Configuration

### Config file: `~/.config/readwise-triage/config.yaml`

Override path with `READWISE_TRIAGE_CONFIG`.

```yaml
readwise_token: ""
llm:
  api_key: ""
  base_url: ""
  model: ""
inbox_days_ago: 7
```

### Parameters

| Param | Key | Env var | Default | Required |
|-------|-----|---------|---------|----------|
| Readwise token | `readwise_token` | `READWISE_TOKEN` | — | Yes |
| LLM API key | `llm.api_key` | `LLM_API_KEY` | — | Cloud only |
| LLM base URL | `llm.base_url` | `LLM_BASE_URL` | — | Cloud only |
| LLM model | `llm.model` | `LLM_MODEL` | — | Yes (for triage) |
| Inbox days | `inbox_days_ago` | `INBOX_DAYS_AGO` | `7` | No |
| Config path | — | `READWISE_TRIAGE_CONFIG` | `~/.config/readwise-triage/config.yaml` | No |

Env vars override config file. Any OpenAI-compatible API works (OpenAI, Ollama, OpenRouter, etc.).

## Keys

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate down / up |
| `r` | Mark read |
| `l` | Mark later |
| `a` / `d` | Archive |
| `x` / `space` | Toggle selection |
| `enter` | Edit tags |
| `o` | Open URL in browser |
| `u` | Push to Readwise |
| `f` | Fetch more (+7 days) |
| `T` | Auto-triage with LLM |
| `R` | Refresh from Readwise |
| `?` | Toggle help |
| `q` | Quit |

## Persistence

| File | Content |
|------|---------|
| `~/.config/readwise-triage/config.yaml` | Config (token, LLM, days) |
| `~/.config/readwise-triage/triage_store.json` | Triage decisions + LLM reports |
