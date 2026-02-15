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

### 4. Testing Best Practices
- **Logic over View**: Focus unit tests on `Update()` and helper methods that modify state. View rendering can be smoke-tested for non-empty output.
- **Table-Driven Tests**: Use table-driven tests for parsing logic (like JSON extraction) and string manipulations.
- **Key Binding Verification**: When testing keyboard input, use `tea.KeyMsg` and verify the resulting `Model` state or `tea.Cmd`.
- **Coverage**: Aim for high coverage in `internal/ui` to catch state transition regressions.

### 5. Readwise API Integration
- API logic resides in `internal/readwise/`.
- Use the `Client` struct for all communications.
- Follow the Readwise Reader V3 API specifications (see `READWISE_API.md`).
- **Rate Limiting**: Respect the 50 req/min limit on UPDATE/CREATE. Use `time.Ticker` for batch operations.

### 5. Persistence
- Triage results are stored in `~/.config/readwise-triage/triage_store.json`.
- Configuration is in `config.yaml` in the same directory.
- Use `internal/config` packages to manage these files.

---

## Tooling & Constraints
- **NO `as any`**: Do not bypass type safety.
- **NO Type Suppression**: Never use `@ts-ignore` or equivalent in Go.
- **Evidence Required**: No task is complete without `lsp_diagnostics` being clean on changed files and `go build ./...` succeeding.
- **Commit Pattern**: Create atomic commits with descriptive messages (e.g., `Fix: ...`, `Feat: ...`, `Refactor: ...`). Commit frequently along the way to maintain a clean and revertible history.

---

## Repository Specific Patterns

### Keyboard Shortcuts
When adding shortcuts, update:
1. `internal/ui/keys.go` (KeyMap definition)
2. `internal/ui/model.go` (Input handler)
3. `README.md` (Documentation table)

### Emoji Alignment
Use `github.com/mattn/go-runewidth` for all string manipulations involving emojis (e.g., padding and truncation) to ensure visual alignment in the TUI.
