# maget

`wget` for magnet link.

A command-line tool to download files from torrents using a magnet link.

Give it a magnet link, and it downloads the torrent's content into the current
directory (or a directory of your choice), named with the original filename
from the torrent metadata.

## Installation

Requires [Go](https://go.dev/dl/) 1.22 or later.

```sh
git clone <repo-url> maget
cd maget
go build -o maget .
```

Optionally move the binary somewhere on your `PATH`:

```sh
mv maget /usr/local/bin/
```

## Usage

```
maget [options] <magnet-link>
```

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `-o`, `--output <dir>` | Output directory | current directory |
| `-i`, `--info` | Show torrent information (name, size, files, trackers) without downloading | off |
| `-v` | Verbose output (torrent client logs) | off |
| `-t`, `--timeout <dur>` | Give up after this duration, e.g. `10m`, `1h30m` | none |
| `-h`, `--help` | Show help text | |

### Examples

Download into the current directory:

```sh
maget "magnet:?xt=urn:btih:dd8255ecdc7ca55fb0bbf81323d87062db1f6d1c&dn=Big+Buck+Bunny"
```

Download into a specific directory with a 30-minute timeout:

```sh
maget -o ~/Downloads -t 30m "magnet:?xt=urn:btih:..."
```

Inspect a torrent before downloading anything:

```sh
maget -i "magnet:?xt=urn:btih:..."
```

```
Name:        Big Buck Bunny
Info hash:   dd8255ecdc7ca55fb0bbf81323d87062db1f6d1c
Size:        263.6 MiB (276445467 bytes)
Pieces:      1055 × 256.0 KiB
Trackers:    (none — DHT only)
Files (3):
       140 B  Big Buck Bunny.en.srt
   263.3 MiB  Big Buck Bunny.mp4
   303.1 KiB  poster.jpg
```

Fetching the file list and total size requires the torrent metadata, so `-i`
still needs a peer connection (it does not download any content).

> **Tip:** always quote the magnet link — it contains `&` characters that the
> shell would otherwise interpret.

While downloading, a progress line shows percent complete, downloaded/total
size, speed, connected peers, and ETA:

```
 42.3%  116.9 MiB / 263.6 MiB  38.3 MiB/s  peers 11  ETA 4s
```

## Behavior

- **Magnet links**: hex (40-char) and base32 (32-char) info hashes are
  supported. Tracker-less links work too — peers are found via DHT.
- **Filenames** come from the torrent metadata, not the untrusted `dn=`
  parameter of the magnet link.
- **No overwriting**: if a file or directory with the same name already
  exists, the download is saved as `name (1)`, `name (2)`, and so on.
- **Multi-file torrents** are downloaded in full, preserving the torrent's
  folder structure.
- **Interrupt & resume**: press `Ctrl+C` to stop. Partial data is kept in a
  hidden `.maget-<infohash>/` staging directory inside the output directory;
  rerunning the same magnet link resumes where it left off.
- **Exit codes**: `0` success, `1` runtime failure (timeout, interrupt,
  network), `2` usage error (bad flags or malformed magnet link).

## Development

```sh
go build ./...   # build
go test ./...    # run tests
go vet ./...     # static checks
```

Only download content you have the right to download.
