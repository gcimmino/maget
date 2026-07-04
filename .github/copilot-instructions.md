# Copilot Instructions — Maget

## Project overview

Maget is a Go command-line application that downloads a file from a torrent given a magnet link as a positional argument. The downloaded file is written to the current directory (or `-o` path) using the original filename from the torrent metadata.

**Status: greenfield.** No code exists yet — only `PLANNING.md` (purpose/architecture) and `TASK.md` (implementation task list with checkboxes). Consult both before implementing anything, and check off completed items in `TASK.md` as you go.

## Planned architecture (from PLANNING.md / TASK.md)

- Go module; entry point `main.go`, with logic in `internal/downloader/` and `internal/cli/`
- Dependency strategy: use `github.com/anacrolix/torrent` (handles magnet parsing, DHT, trackers, peer wire protocol, piece verification) rather than implementing the BitTorrent protocol from scratch. Core flow: `client.AddMagnet(...)` → wait on `torrent.GotInfo()` → `torrent.DownloadAll()`
- CLI flags: `-o/--output` (default: current dir), `-v` (verbose/progress), `-t/--timeout`

## Key conventions and decisions

- Validate magnet URIs (`magnet:?xt=urn:btih:...`); support both 40-char hex and 32-char base32 info hashes, and tracker-less (DHT-only) links
- Use the filename from torrent metadata, **not** the untrusted `dn=` magnet parameter
- Do not overwrite existing files (e.g., append `(1)`)
- Exit codes: 0 on success, non-zero on failure with clear stderr messages
- Handle Ctrl+C gracefully (clean up partial files or support resume)
- Open question flagged in TASK.md: multi-file torrent handling (download all vs. largest file, preserve folder structure) — ask the user before deciding

## Build and test

Standard Go tooling once the module is initialized:

```sh
go build ./...
go test ./...                                # all tests
go test ./internal/cli -run TestName -v      # single test
go vet ./...
```

Planned test coverage (TASK.md §8): unit tests for magnet parsing; integration test against a legal torrent (e.g., a Linux ISO); edge cases for single/multi-file, tracker-only, DHT-only, and malformed magnet links.
