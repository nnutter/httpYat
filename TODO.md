# TODO

httpYat is a TUI over [httpYac](https://httpyac.github.io/) `.http` files. The TUI lists requests; `httpyac send` executes them.

## Done

- Parse one `.http` file for the catalog (regions, names, vars, line numbers)
- Bubble Tea UI: request list, variable overlay, response pane
- Send via `httpyac send --json` (`-n` / `-l`, `--var`, `--timeout`)
- Panes clip and scroll within the window so the footer stays visible

## Next

### Workspace loading

- Accept multiple files: `httpyat a.http b.http`
- Open a directory of `.http` / `.rest` files
- `-r` / `--recursive` for nested directories
- Group the request list by file (and folder when recursive)
- Scroll long request lists

### Environments

- `-e` / `--env` passed through to `httpyac send -e`
- TUI switcher for env names from `http-client.env.json` / `.env`

### TUI polish

- `stripForWidth` is a no-op stub in `internal/tui/view.go`
- Filter / jump to a request
- Reload the file(s) after edits
- Copy response body
- Show httpyac stderr / test results, not just JSON body

### Product

- README: install (`httpyac` on `PATH`), usage, keys
- CI (mise already has `test` / `lint` / `cover`)
- `golangci-lint` config to match the lint task
- LICENSE
- Replace the `WIP` commit with a real history before publishing
