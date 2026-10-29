# tasks

## 1. Project setup

- [x] Init Go module (`go mod init maget`)
- [x] Decide dependency strategy: use `github.com/anacrolix/torrent` (handles magnet parsing, DHT, trackers, peer wire protocol, piece verification)
- [x] Set up project structure: `main.go`, `internal/downloader/`, `internal/cli/`

## 2. CLI argument handling

- [x] Accept a magnet link as a positional argument
- [x] Validate the input looks like a valid magnet URI (`magnet:?xt=urn:btih:...`)
- [x] Support optional flags: `-o/--output` (output directory, default = current dir), `-v` (verbose/progress logging), `-t/--timeout`
- [x] Support `-i/--info` to show torrent details (name, hash, size, pieces, trackers, files) without downloading
- [x] Print usage/help text on missing or malformed input

## 3. Magnet link parsing

- [x] Parse magnet URI query params: `xt` (info hash), `dn` (display name, optional/untrusted), `tr` (tracker URLs)
- [x] Handle both 32-char base32 and 40-char hex info hashes
- [x] Handle magnet links with no trackers (DHT-only)

## 4. Peer/metadata discovery

- [x] Connect to trackers (HTTP/UDP) and/or bootstrap DHT nodes to find peers
- [x] Fetch torrent metadata (the actual file list/piece info) via the ut_metadata extension, since magnet links don't include a .torrent file directly
- [x] Handle multi-file torrents (decision: download all files)

## 5. Downloading

- [x] Establish peer connections (peer wire protocol: handshake, bitfield, interested/choke messages) — handled by anacrolix/torrent
- [x] Request and download pieces — handled by anacrolix/torrent
- [x] Verify each piece against its SHA-1 hash — handled by anacrolix/torrent
- [x] Reassemble pieces into the final file(s) in order — handled by anacrolix/torrent
- [x] Handle stalls/dead peers and retry logic — handled by anacrolix/torrent

## 6. Output handling

- [x] Determine the "original" filename from torrent metadata (not from the untrusted `dn` param)
- [x] Write output to the current working directory (or `-o` path)
- [x] Avoid overwriting existing files (append ` (1)`, ` (2)`, ... before the extension)
- [x] For multi-file torrents, preserve the torrent's folder structure (decision: yes)

## 7. Progress & UX

- [x] Print download progress (%, speed, peers connected, ETA)
- [x] Handle Ctrl+C gracefully (partial data kept in staging dir; rerunning the same magnet resumes)
- [x] Exit codes: 0 success, non-zero on failure (1 runtime, 2 usage), with clear stderr messages

## 8. Testing

- [x] Unit tests for magnet link parsing (`internal/cli/cli_test.go`)
- [x] Integration test against a known public-domain/legal torrent — verified manually end-to-end with the Big Buck Bunny torrent (multi-file, ~264 MiB, pieces hash-verified by the client); not codified as an automated test (needs network/time)
- [x] Test edge cases: multi-file torrent (manual), tracker-only (manual), DHT-only + malformed magnet links (unit tests); conflict/no-overwrite behavior covered in `internal/downloader/downloader_test.go`

---

**Library vs. from-scratch:** Using `anacrolix/torrent` reduces most of tasks 4–5 to calling its API (`client.AddMagnet(...)`, waiting on `torrent.GotInfo()`, then `torrent.DownloadAll()`). If you want to implement the wire protocol and DHT yourself instead, I can expand tasks 4–5 into much finer-grained subtasks (bencode decoding, Kademlia DHT lookups, peer handshake byte layout, choke/unchoke algorithm, etc.) — let me know which direction you want and I can go deeper or start writing the actual Go code.
