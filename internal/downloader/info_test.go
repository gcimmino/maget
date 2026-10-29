package downloader

import (
	"strings"
	"testing"
)

func TestInfoString(t *testing.T) {
	info := Info{
		Name:        "Big Buck Bunny",
		InfoHash:    "dd8255ecdc7ca55fb0bbf81323d87062db1f6d1c",
		Length:      3 * 1024 * 1024,
		PieceLength: 256 * 1024,
		NumPieces:   12,
		Trackers:    []string{"udp://tracker.example.com:6969"},
		Files: []FileInfo{
			{Path: "video.mp4", Length: 2 * 1024 * 1024},
			{Path: "readme.txt", Length: 1024 * 1024},
		},
	}

	got := info.String()
	for _, want := range []string{
		"Name:        Big Buck Bunny",
		"Info hash:   dd8255ecdc7ca55fb0bbf81323d87062db1f6d1c",
		"Size:        3.0 MiB (3145728 bytes)",
		"Pieces:      12 × 256.0 KiB",
		"udp://tracker.example.com:6969",
		"Files (2):",
		"video.mp4",
		"readme.txt",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestInfoStringEscapesControls(t *testing.T) {
	info := Info{
		Name:     "bad\x1b[2J\nname",
		InfoHash: "08ada5a7a6183aae1e09d831df6748d566095a10",
		Trackers: []string{"udp://tracker.example.com\a"},
		Files: []FileInfo{
			{Path: "video.mp4", Length: 1},
			{Path: "bad\rname.txt", Length: 1},
		},
	}

	got := info.String()
	for _, want := range []string{`bad\x1b[2J\nname`, `tracker.example.com\a`, `bad\rname.txt`} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing escaped text %q:\n%s", want, got)
		}
	}
	for _, control := range []string{"\x1b", "\a", "\r"} {
		if strings.Contains(got, control) {
			t.Errorf("output contains raw control character %q", control)
		}
	}
}

func TestInfoStringSingleFileHasNoFileList(t *testing.T) {
	info := Info{
		Name:      "file.iso",
		InfoHash:  "08ada5a7a6183aae1e09d831df6748d566095a10",
		Length:    1024,
		NumPieces: 1,
		Files:     []FileInfo{{Path: "file.iso", Length: 1024}},
	}

	got := info.String()
	if strings.Contains(got, "Files (") {
		t.Errorf("single-file torrent should not list files:\n%s", got)
	}
	if !strings.Contains(got, "Trackers:    (none — DHT only)") {
		t.Errorf("expected DHT-only tracker note:\n%s", got)
	}
}
