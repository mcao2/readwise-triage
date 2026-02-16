# Agentic Development Guide - Readwise Triage

This repository contains a Go-based CLI tool for triaging Readwise Reader inbox items. It uses the Bubble Tea TUI framework and integrates with the Readwise Reader V3 API.

## Core Commands

### Building and Running
- **Build all**: `go build ./...`
- **Build binary**: `go build -o readwise-triage ./cmd/readwise-triage`
- **Run**: `./readwise-triage` (requires `READWISE_TOKEN`)

### Testing
- **Run all tests**: `go test ./...`
- **Run package tests**: `go test ./internal/ui/...`
- **Run single test**: `go test -v ./internal/ui -run TestExportItemsToJSON`
- **Update Golden files (if any)**: `go test ./... -update`

### Linting and Quality
- **LSP Diagnostics**: Use `lsp_diagnostics` tool on specific files.
- **Formatting**: `go fmt ./...`

---

## Code Style & Architecture

### 1. General Principles
- **Minimalism**: Fix only what is requested. Avoid large-scale refactors unless explicitly asked.
- **Consistency**: Follow existing patterns in the module you are modifying.
- **Standard Library**: Prefer the Go standard library over external dependencies where possible.

### 2. Go Specifics
- **Imports**: Group imports into three blocks separated by newlines:
  1. Standard library
  2. External dependencies (e.g., charmbracelet)
  3. Internal packages (`github.com/mcao2/readwise-triage/internal/...`)
- **Naming**:
  - `CamelCase` for exported identifiers (functions, types, fields).
  - `lowerCamelCase` for internal identifiers.
  - Short, descriptive variable names (e.g., `cfg` for Config, `m` for Model).
- **Error Handling**:
  - Check errors immediately.
  - Wrap errors using `fmt.Errorf("context: %w", err)`.
  - Use specific error types only when the caller needs to handle them programmatically.
- **Receivers**:
  - Use **pointer receivers** (`func (m *Model) ...`) for all methods that modify state or are part of the `tea.Model` interface implementation.
  - Value receivers should be avoided for consistency unless the type is a primitive or very small immutable struct.

### 3. TUI (Bubble Tea) Architecture
- **State Management**: The `Model` struct in `internal/ui/model.go` is the central state.
- **Component Communication**: Use `tea.Cmd` and messages (e.g., `ItemsLoadedMsg`) to handle asynchronous operations like API calls.
- **View Logic**: Keep `View()` methods pure. Delegate layout to sub-methods (e.g., `reviewingView()`) and use `lipgloss` for styling.
- **Pointer Consistency**: Always ensure `Update`, `View`, and `Init` use pointer receivers to avoid state loss during render cycles.
- **Stable Sorting**: When iterating over maps for UI elements (e.g., theme names), ALWAYS sort the keys first. Go map iteration is random and causes unstable UI behavior/test failures.
- **Selection Awareness**: Batch actions (`e`, `o`, `u`) MUST respect the selection state. Use `m.listView.GetSelected()` to determine if a targeted subset of items should be processed.
- **Spinner & Progress**: Use `bubbles/spinner` for loading states and `bubbles/progress` for progress bars. The spinner must be ticked via `Init()` returning `m.spinner.Tick`, and both spinner and progress `TickMsg`/`FrameMsg` must be handled in `Update()`.
- **Theme Propagation**: When cycling themes, update ALL theme-dependent components: `m.styles`, `m.listView.UpdateTableStyles()`, and `m.spinner.Style`. Missing any causes visual inconsistency.
- **Layout Structure**: The reviewing view uses a header bar (app name + count), table, detail pane, status line, and footer. Use `strings.Join(parts, "\n")` to compose these (NOT `lipgloss.JoinVertical` — see Terminal Width below). Other views use `m.styles.Border` or `m.styles.Card` for centered panel layouts.
- **Help System**: The `?` key toggles `m.showHelp` between a compact two-line footer (`renderReviewFooter`) and a full categorized overlay (`renderFullHelp`). Help entries use the `helpEntry{key, desc}` struct rendered via `renderHelpLine`.
- **Popup Overlays**: Modal popups (e.g., tag editor) should overlay on top of the existing view, not replace it. Render the background view first, split into lines, then stamp the centered popup lines over the middle rows. Use `lipgloss.PlaceHorizontal` per-line for horizontal centering within the overlay.
- **View Centering**: Non-review views (config, fetching, confirming, updating, done, message) are centered on screen via `lipgloss.Place(m.width, m.height, ...)` in the top-level `View()` method. Don't duplicate centering logic in individual view functions.
- **macOS Terminal Key Sequences**: macOS terminals send ESC+b / ESC+f for Option+Left/Right (parsed by Bubble Tea as `alt+b`/`alt+f` with `KeyRunes` type), NOT `KeyLeft`/`KeyRight` with `Alt=true`. Similarly, Option+Delete comes through as `alt+backspace`. When handling modifier+key combos, match on `msg.String()` (e.g., `"alt+left"`, `"alt+b"`) to cover both CSI and ESC-letter sequences. Follow the pattern in `bubbles/textinput`.
- **Detail Pane**: `ListView.DetailView()` renders item metadata below the table. It pads to a fixed `detailPaneHeight` (4 lines) so the total view height stays constant across items with varying metadata.
- **Column Widths**: Let the table column `Width` handle padding — don't hardcode trailing spaces in cell text. Use `runewidth.FillRight` only to pad to the column width.
- **Terminal Width — The Last Column Problem**: NEVER render any line at exactly the terminal width. When a line fills the last column, many terminal emulators (macOS Terminal, iTerm2, etc.) add an implicit line wrap, which breaks Bubble Tea's line tracking and causes visual corruption (stale content, multiple highlighted rows, header disappearing). Always use `m.width - 1` for `HeaderBar.Width()`, `FooterBar.Width()`, dividers, and ensure column widths + cell padding sum to less than terminal width. Avoid `lipgloss.JoinVertical` for full-screen layouts — it pads every line to the widest element's width, which can push lines to the terminal edge.
- **Custom Table Rendering**: The bubbles `table.Model` is used for row data storage, but its `View()` is bypassed. `ListView.View()` renders the header and visible rows directly with simple scrolling logic, avoiding the bubbles viewport's broken `YOffset` calculations.
- **Fixed-Height Output**: `reviewingView()` pads its output to exactly `m.height` lines so the alternate screen buffer repaints cleanly every frame.

### 4. Testing Best Practices
- **Logic over View**: Focus unit tests on `Update()` and helper methods that modify state. View rendering can be smoke-tested for non-empty output.
- **Table-Driven Tests**: Use table-driven tests for parsing logic (like JSON extraction) and string manipulations.
- **Key Binding Verification**: When testing keyboard input, use `tea.KeyMsg` and verify the resulting `Model` state or `tea.Cmd`.
- **Coverage**: Aim for high coverage in `internal/ui` to catch state transition regressions.
- **HTTP Mock Pattern**: `mockHTTPClient` in `readwise_test.go` captures requests (including body copies) and returns canned responses. Use it to verify request payloads, retry behavior, and call counts.
- **View Content Tests**: When testing view output, use `strings.Contains` on rendered content rather than exact string matching — lipgloss styling adds ANSI codes that make exact matching fragile.
- **Spinner in Tests**: `Init()` returns a spinner tick command. Tests that check `Init()` should expect a non-nil `tea.Cmd`. The spinner `TickMsg` can be triggered via `m.spinner.Tick()` for testing the update loop.

### 5. Readwise API Integration
- API logic resides in `internal/readwise/`.
- Use the `Client` struct for all communications.
- Follow the Readwise Reader V3 API specifications (see `READWISE_API.md`).
- **Rate Limiting**: Respect the 50 req/min limit on UPDATE/CREATE. Use `time.Ticker` at 1.5s intervals for batch operations.
- **429 Handling**: `doRequest` retries on 429 responses. It parses the `Retry-After` header when present, falling back to exponential backoff. Always handle 429 alongside 5xx in retry logic.
- **PATCH Body**: Don't include fields already present in the URL path (e.g., `document_id` is in `/update/<id>/`, so omit it from the JSON body).
- **Delete = Archive (intentional)**: The `delete` triage action archives items via PATCH rather than calling the DELETE endpoint. This is by design — it keeps the action reversible.

### 6. LLM Triage Pipeline
- The LLM classifies items into actions: `delete`, `archive`, `later`, `read_now`, and `needs_review`.
- **`needs_review`**: Escape hatch for items the LLM can't confidently classify (paywalled, ambiguous, insufficient context). Don't force every item into a definitive action.
- **Suggested Tags**: Tags flow from LLM generation → triage import → Readwise update. They are appended alongside action-based tags during `UpdateDocument`. During import, tags matching action names (`read_now`, `later`, `archive`, `delete`, `needs_review`) are automatically filtered out to avoid redundancy.
- **Token Efficiency**: Only generate LLM fields that are actually consumed downstream. Unused fields (e.g., `reading_guide`, `credibility_check`) waste tokens.

### 7. Persistence
- Triage results are stored in `~/.config/readwise-triage/triage.db` (SQLite via `modernc.org/sqlite`, pure Go, no CGO).
- The store persists the full `triage.Result` report for LLM-triaged items. `SetItem` takes a `*triage.Result` as the last parameter (nil for manual entries).
- On first run, if a legacy `triage_store.json` exists it is auto-migrated into SQLite and renamed to `.bak`.
- Writes are immediate (no explicit `Save()` needed). `Save()` is retained as a no-op for compatibility.
- Configuration is in `config.yaml` in the same directory. Preferences like location, lookback days, and theme are persisted here automatically when changed in the TUI.
- Use `internal/config` packages to manage these files.
- **Schema changes**: If you modify the `triage_entries` table schema, add an `ALTER TABLE` migration in `LoadTriageStore()` after the `CREATE TABLE IF NOT EXISTS` statement.

---

## Tooling & Constraints
- **Evidence Required**: No task is complete without `go vet ./...` being clean on changed files and `go build ./...` succeeding.
- **Commit Pattern**: Create atomic commits with descriptive messages (e.g., `Fix: ...`, `Feat: ...`, `Refactor: ...`). Commit frequently along the way to maintain a clean and revertible history.
- **Gitignore**: `cmd/readwise-triage` (the binary) is gitignored. When staging files, don't `git add` that path — only add the `.go` source file at `cmd/readwise-triage/main.go`.
- **Bubbles Subpackages**: Some `bubbles` subpackages (e.g., `progress`) pull in transitive dependencies (e.g., `harmonica`) that aren't in `go.sum` by default. Run `go get github.com/charmbracelet/bubbles/<subpackage>` to resolve missing `go.sum` entries before building.

---

## Repository Specific Patterns

### Keyboard Shortcuts
When adding shortcuts, update:
1. `internal/ui/keys.go` (KeyMap definition)
2. `internal/ui/model.go` (Input handler)
3. `README.md` (Documentation table)

### Emoji Alignment
Use `github.com/mattn/go-runewidth` for all string manipulations involving emojis (e.g., padding and truncation) to ensure visual alignment in the TUI.

### Tag Editor
- The `Enter` key in review view opens an inline tag editor popup (overlay on the review view).
- Supports cursor navigation (arrow keys), word jump (Option+Arrow / Alt+b/f), word delete (Option+Delete), and standard backspace.
- In batch mode, edited tags apply to all selected items.
- Tags are comma-separated; empty strings are filtered out on confirm.

### Theme & Styles Architecture
- Themes are defined in `internal/ui/styles.go` as `Theme` structs with color hex values. Each theme must include all fields including `Subtle` (used for borders/separators).
- `Styles` struct contains both semantic styles (`Title`, `Error`, `Success`) and layout styles (`HeaderBar`, `FooterBar`, `Border`, `Card`, `Detail`) plus help-specific styles (`HelpKey`, `HelpDesc`, `HelpSep`).
- When adding a new theme, add it to the `Themes` map. Themes are sorted alphabetically for stable cycling.
