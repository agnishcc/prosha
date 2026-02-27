# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`worktree-tui` — a keyboard-driven terminal UI for managing git worktrees, built with Go + Bubbletea. Binary name: `worktree-tui` (users invoke it via a `wt()` shell wrapper).

## Common Commands

```bash
# Install dependencies (run once after cloning)
go mod tidy

# Run from source
go run .

# Build binary
go build -o wt .

# Install globally
go install .

# Vet + build check (no test suite yet)
go vet ./...
```

## Architecture

The app follows the **Elm architecture** via [Bubbletea](https://github.com/charmbracelet/bubbletea): one `Model` struct, a pure `Update(msg)` function, and a pure `View()` render function.

```
main.go                      — tea.NewProgram entry point
internal/
  types/types.go             — Worktree, Commit structs; AppState enum
  git/git.go                 — all git shell operations (os/exec, no git library)
  ui/
    model.go                 — Model struct, Init(), async message/command types
    update.go                — Update() + per-state key handlers
    view.go                  — View() + all render helpers
    styles.go                — Lipgloss style vars (Catppuccin Mocha palette)
```

### State machine

```
AppState
  StateNoGit          → git init prompt
  StateShellSetup     → first-run wt() shell wrapper prompt
  StateList           → left pane (worktree list) + right pane (detail)
  StateNewWorktree    → modal overlay: type selector + name input
  StateEditWorktree   → modal overlay: branch rename input
  StateDeleteConfirm  → modal overlay: y/N confirmation
```

### Key data flow

1. `Init()` dispatches `checkGitRepo` command.
2. `gitCheckMsg` → transition to `StateNoGit`, `StateShellSetup`, or `StateList` + `loadWorktrees()`.
3. `worktreesLoadedMsg` → populates `m.worktrees`, re-entered after every mutation (create/delete/rename).
4. `cursor` is the single source of selection truth: `0` = "+ new worktree", `1..n` = `m.worktrees[cursor-1]`.

### Git operations

All git calls shell out to the `git` binary via `os/exec`. No go-git dependency.
Worktrees are created under `.wt/<type>-<name>` in the repo root.

### CD-on-exit

Pressing `c` writes the selected worktree path to `/tmp/.wt_cd_path` then quits. The shell wrapper (`wt()` function appended to `.zshrc`/`.bashrc`) reads this file and calls `cd`. A one-time marker at `~/.config/worktree-tui/integrated` prevents re-showing the setup prompt.

## Tech Stack

| Layer | Choice |
|-------|--------|
| Language | Go 1.21 |
| TUI framework | [Bubbletea](https://github.com/charmbracelet/bubbletea) |
| Styling | [Lipgloss](https://github.com/charmbracelet/lipgloss) (Catppuccin Mocha) |
| Git ops | `os/exec` shell-outs |

## Layout

```
╭───────────── header (full width) ──────────────────╮
│  ⎇  worktree                      reponame  branch │
╰────────────────────────────────────────────────────╯

╭── left (~25%) ──╮  ╭── right (~75%) ──────────────╮
│ + new worktree  │  │  branch name                  │
│▌ selected       │  │  ◎ Branch  ...                │
│  other          │  │  ◎ Path    ...                │
╰─────────────────╯  │  ● hash  message  time        │
                     ╰──────────────────────────────╯

n  new    d  delete    e  edit    c  cd    ↑↓ / j k  navigate    q  quit
```

Modals (new / edit / delete) are rendered centered via `lipgloss.Place` over a dark background instead of overlaid on the list — this avoids ANSI-aware line-merging complexity.
