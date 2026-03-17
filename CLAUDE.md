# md-preview-cli

A CLI tool that renders markdown files in a native frameless window. Designed for terminal-only agents like Claude Code.

## Architecture

- **Language**: Go with CGO (webview + Cocoa frameless)
- **Two-process model**: CLI spawns GUI subprocess via `--internal-gui=<config.json>`, exits immediately
- **Multi-window host**: First invocation spawns a persistent host process (`--internal-host`). Subsequent invocations join via IPC socket, opening new windows in the same process. The host manages all windows, the dock icon, and the NSApp event loop. IPC server uses try-listen-first to prevent race conditions when two CLIs spawn simultaneously.
- **CLI output**: Every invocation prints `Previewing <filename>` to stdout on success (or `(reused)` if window already open). Agents depend on this output.
- **File dedup**: Same file cannot open twice. FilePaths are normalized via `filepath.Abs` + `filepath.EvalSymlinks`. Stdin is exempt (always creates new window).
- **Socket isolation**: IPC socket includes UID in filename (`md-preview-cli-<uid>.sock`) to prevent multi-user collision.
- **Rendering**: Server-side Goldmark (CommonMark + GFM) + Chroma syntax highlighting
- **Client-side**: KaTeX (math) and Mermaid.js (diagrams) lazy-loaded only when present
- **Single binary**: All HTML/CSS/JS/fonts embedded via `go:embed`
- **Window close**: Native `[NSWindow close]` instead of `webview.Destroy()`. The webview destructor's `deplete_run_loop_event_queue()` deadlocks when called from a GCD main queue block (which is always the case since `CloseWindow` runs via `Dispatch`). Closed webview C objects are intentionally leaked — acceptable because the process exits when the last window closes.

## Key Directories

- `cmd/` — CLI flag parsing, stdin detection, GUI spawning
- `internal/gui/` — GUI process lifecycle, webview, frameless macOS window (CGO). Two entry points: `Run()` for single-window accessory mode, `RunHost()` for multi-window daemon with dock icon
- `internal/ipc/` — Unix socket IPC server for CLI-to-host communication
- `internal/server/` — HTTP server, WebSocket hub, auto-shutdown
- `internal/renderer/` — Goldmark markdown→HTML pipeline, TOC extraction, math/mermaid placeholders
- `internal/watcher/` — fsnotify + stat-based polling file watchers
- `web/` — Embedded assets (HTML template, JS, CSS, vendored libraries)
- `testdata/` — Test markdown files

## Build

```bash
# Download vendored JS dependencies (first time only)
make deps

# Debug build — includes debug symbols, version="dev" (~21 MB)
make build

# Release build — stripped symbols, version from git (~16 MB)
make release

# Run
./bin/md-preview-cli testdata/full-spec.md
echo "# Hello" | ./bin/md-preview-cli
```

## Testing

```bash
make test
go test -v ./internal/renderer/...
```

## Conventions

- Follow the mermaid-preview-cli patterns for GUI, server, watcher, WebSocket
- Post-process HTML rather than custom Goldmark AST transformers where simpler
- Theme system uses CSS custom properties with system/light/dark modes
- All keyboard shortcuts are vim-inspired (j/k, n/p, g g/G)
