package downloader

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/anacrolix/torrent"
)

// Info summarizes a resolved torrent's metadata for display.
type Info struct {
	Name        string
	InfoHash    string
	Length      int64
	PieceLength int64
	NumPieces   int
	Trackers    []string
	Files       []FileInfo
}

// FileInfo describes one file inside a torrent.
type FileInfo struct {
	Path   string
	Length int64
}

// Describe extracts display information from a torrent whose metadata has
// already been resolved (see Downloader.Resolve).
func Describe(t *torrent.Torrent) Info {
	i := Info{
		Name:     t.Name(),
		InfoHash: t.InfoHash().HexString(),
		Length:   t.Length(),
		Trackers: trackers(t),
	}
	if info := t.Info(); info != nil {
		i.PieceLength = info.PieceLength
	}
	i.NumPieces = int(t.NumPieces())
	for _, f := range t.Files() {
		i.Files = append(i.Files, FileInfo{Path: f.DisplayPath(), Length: f.Length()})
	}
	return i
}

// trackers returns the announce URLs known for t, de-duplicated and in
// announce order.
func trackers(t *torrent.Torrent) []string {
	mi := t.Metainfo()
	seen := make(map[string]bool)
	var urls []string
	add := func(u string) {
		if u != "" && !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
	}
	add(mi.Announce)
	for _, tier := range mi.AnnounceList {
		for _, u := range tier {
			add(u)
		}
	}
	return urls
}

// String renders the info as human-readable text.
func (i Info) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "Name:        %s\n", escapeControls(i.Name))
	fmt.Fprintf(&b, "Info hash:   %s\n", i.InfoHash)
	fmt.Fprintf(&b, "Size:        %s (%d bytes)\n", formatBytes(i.Length), i.Length)
	if i.PieceLength > 0 {
		fmt.Fprintf(&b, "Pieces:      %d × %s\n", i.NumPieces, formatBytes(i.PieceLength))
	} else {
		fmt.Fprintf(&b, "Pieces:      %d\n", i.NumPieces)
	}

	if len(i.Trackers) == 0 {
		b.WriteString("Trackers:    (none — DHT only)\n")
	} else {
		b.WriteString("Trackers:\n")
		for _, u := range i.Trackers {
			fmt.Fprintf(&b, "  %s\n", escapeControls(u))
		}
	}

	// A single-file torrent is already fully described by its name and size.
	if len(i.Files) > 1 {
		fmt.Fprintf(&b, "Files (%d):\n", len(i.Files))
		for _, f := range i.Files {
			fmt.Fprintf(&b, "  %10s  %s\n", formatBytes(f.Length), escapeControls(f.Path))
		}
	}

	return b.String()
}

// escapeControls renders terminal control characters from untrusted torrent
// metadata as visible escape sequences while preserving ordinary Unicode.
func escapeControls(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if !unicode.IsControl(r) {
			b.WriteRune(r)
			continue
		}
		quoted := strconv.QuoteRune(r)
		b.WriteString(quoted[1 : len(quoted)-1])
	}
	return b.String()
}
