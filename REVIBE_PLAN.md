# Readwise Triage Re-Vibe Plan (v2 — Council-Reviewed)

## Goal
Re-vibe the readwise-triage app (Go + Bubble Tea TUI) following lean/small/KISS/DRY principles. The user no longer uses it because the TUI is "clanky." Reference TUIs: grok-build (Rust/ratatui, full-screen, auto-theme, status bar, line editor) and deepseek-harness (TS, plugin architecture, web UI).

## Current State
- ~4,600 LOC production, ~5,600 LOC tests, 139 commits (all from Feb 2026, then abandoned)
- Core loop: fetch Readwise inbox → AI triages (action + tags) → review → push back
- 8 TUI states, 30+ keybindings, 2 screens (config + review), 5 themes, 2 LLM triage paths, 5 actions, priority system

---

## 1. CUT (remove entirely)

### 1a. Manual export/import workflow (e/i keys)
- Files: manual_triage.go (518 LOC), manual_triage_test.go (492 LOC), prompt.go (108 LOC)
- **parser.go is PARTIALLY trimmed, NOT fully removed**: `ParseTriageResponse`, `extractJSON`, `fixTrailingCommas`, `IsJSONArray` (~100 LOC) are used by the auto-triage path (client.go:287). Only `ParseSummary`, `extractSection`, `extractListItems`, `Summary` struct (~60 LOC) are dead code and can be removed.
- `extractJSONArray`, `sanitizeLLMJSON`, `extractJSONArraysFromCodeBlocks`, `extractAllJSONArrays` live in manual_triage.go (not parser.go) and ARE fully removed with it.
- Rationale: Clipboard roundtrip is a pre-API-era pattern. Auto-triage (T key) covers this.
- **Note**: ExportItemsToFile/ImportTriageResultsFromFile/ValidateTriageJSON are called in model_test.go — those test cases must also be removed when manual_triage.go is deleted.

### 1b. Dead huh forms — edit_form.go + batch_form.go
- 184 LOC total. Never instantiated in production code (verified by grep).
- Drops the charmbracelet/huh dependency (and transitive: clipperhouse/displaywidth, stringish, uax29).

### 1c. Provider presets + Anthropic API format — full removal
- Drop the `providerDefaults` map entirely.
- **Also remove api_format, AnthropicRequest, AnthropicResponse, and all anthropic header logic** from client.go. The user explicitly asked for OpenAI-compatible providers only.
- LLM config becomes: `api_key`, `base_url`, `model` — three fields, one request/response shape.
- `NewLLMClient(apiKey, baseURL, model)` — no provider, no apiFormat.
- ~110 LOC simpler in client.go (removes ~60 LOC of Anthropic types + branching + ~50 LOC of provider presets).

### 1d. Priority system (high/medium/low)
- Removes: 3 keybindings (1/2/3), Priority field on Item/TriageEntry/TriageDecision, `priority:` tag prefixing (model.go:580-581), batch priority, 🔴🟡🟢 rendering.
- **Acknowledged behavior change**: items will no longer get `priority:high/medium/low` tags pushed to Readwise. This is intended — priority is redundant with tags.
- Also remove `priority` from the auto-triage prompt (auto_prompt.go) and the Result type (types.go).

### 1e. needs_review action
- Removes: 5th action, its rendering (👁), update logic branching.
- Also remove `needs_review` from the auto-triage prompt (auto_prompt.go:25) and validActions map.
- Rationale: If LLM is uncertain, it sets action: "later" + tag "needs-review".

### 1f. `delete` action — collapse into `archive`
- Both `delete` and `archive` map to `location: archive` in update.go. Readwise has no real delete endpoint. Two UI actions for the same backend outcome is taxonomy bloat.
- **Three actions only**: read_now, later, archive. If "discard" semantics matter, use a tag like `trash` under archive.
- Removes the `d` keybinding.

### 1g. Legacy migration code
- JSON→SQLite migration in triage_store.go, default_days_ago→inbox_days_ago compat in config.go.
- 6 months old, no external users.

### 1h. Trim prompt/result model to match cuts
- **auto_prompt.go**: Remove `needs_review` and `priority` from the action list and output schema. Trim output to only `action` + `suggested_tags` (drop `reason` — not consumed downstream).
- **types.go**: Trim `Result` to `ID`, `Title`, `Action`, `SuggestedTags` only. Drop `ContentAnalysis`, `CredibilityCheck`, `ReadingGuide`, `Priority`, `Reason` fields — all dead after the manual export path is removed.
- **prompt.go**: Deleted entirely (part of 1a).

---

## 2. SIMPLIFY (streamline)

### 2a. LLM config: OpenAI-compatible only, no presets, no api_format
- Current: providerDefaults map (4 presets) + api_format branching (openai vs anthropic) + 5 env var overrides.
- Simplified: `api_key`, `base_url`, `model`. Drop provider concept, api_format, and all Anthropic types.
- `NewLLMClient(apiKey, baseURL, model)` — one request/response shape (ChatRequest/ChatResponse).
- Env vars: `LLM_API_KEY`, `LLM_BASE_URL`, `LLM_MODEL` only.

### 2b. Feed vs Inbox → single location (inbox only)
- Drop feed entirely for the MVP. No `--location feed` CLI flag (it would reintroduce location-aware update logic).
- Document feed as out of scope. Re-adding later is cleaner than carrying half-supported logic.
- One lookback, one fetch path, one update path.

### 2c. Themes: 5 → 1
- Current: 5 themes + cycling + persistence + propagation.
- Simplified: One clean dark theme. No GetThemeNames(), no cycleTheme(), no theme persistence, no auto-detect (unnecessary for a single theme).

### 2d. SQLite → JSON file
- Current: modernc.org/sqlite for a flat key→value map of {id: {action, tags}}.
- Simplified: JSON file. Write to temp + rename for atomicity. Drops modernc.org/sqlite + 4 transitive deps. ~100 LOC simpler, no schema migrations.

### 2e. Config file → env vars + JSON store
- After removing theme, location, feed, and use_llm_triage, the only persisted field is inbox_days_ago.
- Read READWISE_TOKEN and LLM settings from env only.
- Persist the last lookback in the JSON triage store, or default to 7 days.
- Remove all backward-compat and token-preservation logic from config.go. If a config file is kept at all, it's minimal — but prefer env-only.

### 2f. Remove bubbles/table from ListView
- Current: ListView uses `table.Model` for row storage but bypasses its viewport with custom rendering.
- Simplified: Store rows as `[][]string` directly. Drop `table.Model`, `UpdateTable`, `SyncCursor`, and table style synchronization.
- Removes a chunk of list_view.go complexity.

### 2g. Move validActions to internal/triage
- `validActions` currently lives in manual_triage.go, which is being deleted.
- Move a small allowed-action set to `internal/triage` and reuse it for both prompt generation and result application.
- Prevents build break when manual_triage.go is removed.

---

## 3. REVISE — TUI re-vibe (inspired by grok-build)

### Problems with current TUI:
1. Config screen → review screen two-step dance
2. Modal popup for tag editing with manual rune/cursor/word-boundary management (~80 LOC)
3. Keybinding explosion (30+ keys across 2 modes)

### grok-build patterns to borrow:
- Full-screen, single view (no mode switching)
- Status bar at bottom
- Line editor for input (bubbles/textinput, not modal popup)
- Mouse interactive (already has WithMouseCellMotion)

### Proposed lean TUI layout:
```
┌─────────────────────────────────────────────────────┐
│ Readwise Triage · 24 items · last 7 days        ●3  │  ← status bar
├─────────────────────────────────────────────────────┤
│  🔥 Read    · ai, productivity        Title here... │
│  ⏰ Later   · rust, wasm              Another title │
│  📁 Archive · —                      Third title   │
│  · New     · —                       Untriaged...  │  ← list (j/k, mouse)
├─────────────────────────────────────────────────────┤
│ Title: Full article title here                      │  ← detail pane
│ url · src:blog.com · 5min · 1.2k words              │
│ Summary excerpt...                                  │
├─────────────────────────────────────────────────────┤
│ j/k navigate · r/l/a triage · T auto-triage · u push│  ← single help line
└─────────────────────────────────────────────────────┘
```

### Keybinding set: ~11 keys
| Key | Action |
|-----|--------|
| j/k | navigate (mouse click/scroll too) |
| r/l/a | triage: read_now / later / archive |
| x | toggle selection (lightweight multi-select; action keys apply to selected set) |
| t | edit tags (inline textinput, not popup) |
| T | auto-triage with LLM |
| u | push to Readwise |
| o | open in browser |
| f | fetch more (+7 days) |
| q | quit |

**Batch selection**: Keep lightweight multi-select (`x` toggles, action keys apply to selected set) but drop the dedicated batch-priority/tag forms. This preserves real bulk value (e.g., "archive all newsletter issues") with one extra key.

Cut: priority (1/2/3), needs_review (n), delete (d), export/import (e/i), theme cycling (t on config), config screen keys, help toggle (?), refresh (R).

### model.go: 1,503 → ~500 LOC, split into 2-3 files (NOT 4)
- `model.go` — state struct, Update dispatch, key handling, fetch/triage/update commands (~300 LOC)
- `view.go` — View rendering + status bar + detail pane (~200 LOC)
- `keys.go` — already exists, keep key definitions here (~50 LOC)

Rationale: For a ~500 LOC UI, 4 files is rearranging complexity, not reducing it. 2-3 files is KISS.

### Tag editing: inline textinput as a status line
- Use `bubbles/textinput` rendered as a single status line at the bottom, not a centered popup overlay.
- Removes the manual line-stamping, rune/cursor/word-boundary management code (~80 LOC in model.go).

---

## Quantified Impact (revised)
| Metric | Current | Re-vibed | Savings |
|---|---|---|---|
| Production LOC | 4,609 | ~1,800 | −61% |
| Direct dependencies | 7 | 4 (bubbles, bubbletea, lipgloss, go-runewidth) | drop huh, sqlite, clipboard, yaml.v3 |
| Keybindings | 30+ | ~11 | −63% |
| TUI states | 8 | 3 (list, triaging, updating) | −63% |
| TUI screens | 2 | 1 | −50% |
| LLM triage paths | 2 | 1 | −50% |
| LLM API formats | 2 (openai + anthropic) | 1 (openai) | −50% |
| Themes | 5 | 1 | −80% |
| Actions | 5 | 3 (read_now, later, archive) | −40% |
| UI files | 1 monolith | 3 (model, view, keys) | manageable |

## What stays
Core loop (fetch → AI triage → review → push), Readwise API integration, tag editing (inline), lightweight batch selection, persistence (JSON), browser open, progressive fetch.

## Execution order
1. CUT phase: dead code + manual export/import removal + Anthropic format removal + priority/needs_review/delete removal (pure subtraction, zero risk)
2. SIMPLIFY phase: LLM config, themes, storage (SQLite→JSON), config trimming, ListView table removal, validActions move
3. REVISE phase: TUI re-vibe (layout, keybindings, model.go split into 2-3 files, inline tag editing)
