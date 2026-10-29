// Package cli handles command line argument parsing and validation for maget.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/anacrolix/torrent/metainfo"
)

// Magnet holds the parsed components of a magnet URI.
type Magnet struct {
	InfoHash    metainfo.Hash // 20-byte BitTorrent info hash
	DisplayName string        // dn param — untrusted, informational only
	Trackers    []string      // tr params — may be empty (DHT-only)
}

// Config holds the parsed command line options.
type Config struct {
	Magnet    string
	Parsed    *Magnet
	OutputDir string
	Verbose   bool
	Timeout   time.Duration
	// Info requests only showing torrent information; nothing is downloaded.
	Info bool
}

// ErrHelp is returned when the user requested the help text.
var ErrHelp = flag.ErrHelp

const usageText = `Usage: maget [options] <magnet-link>

Download a file from a torrent using a magnet link.

Options:
  -o, --output <dir>   output directory (default: current directory)
  -i, --info           show torrent information without downloading
  -v                   verbose output with progress logging
  -t, --timeout <dur>  give up after this duration, e.g. 10m (default: none)
  -h, --help           show this help text
`

// Usage writes the help text to w.
func Usage(w io.Writer) {
	fmt.Fprint(w, usageText)
}

// Parse parses the given arguments (excluding the program name) into a Config.
// It returns ErrHelp if help was requested.
func Parse(args []string) (*Config, error) {
	fs := flag.NewFlagSet("maget", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	cfg := &Config{OutputDir: "."}
	fs.StringVar(&cfg.OutputDir, "o", ".", "output directory")
	fs.StringVar(&cfg.OutputDir, "output", ".", "output directory")
	fs.BoolVar(&cfg.Verbose, "v", false, "verbose output")
	fs.BoolVar(&cfg.Info, "i", false, "show torrent information without downloading")
	fs.BoolVar(&cfg.Info, "info", false, "show torrent information without downloading")
	fs.DurationVar(&cfg.Timeout, "t", 0, "timeout")
	fs.DurationVar(&cfg.Timeout, "timeout", 0, "timeout")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, ErrHelp
		}
		return nil, err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return nil, errors.New("missing magnet link argument")
	}
	if len(rest) > 1 {
		return nil, fmt.Errorf("unexpected extra arguments: %s", strings.Join(rest[1:], " "))
	}

	cfg.Magnet = rest[0]
	m, err := ParseMagnet(cfg.Magnet)
	if err != nil {
		return nil, err
	}
	cfg.Parsed = m
	return cfg, nil
}

// ParseMagnet parses and validates a BitTorrent magnet URI
// (magnet:?xt=urn:btih:<40-char hex or 32-char base32 info hash>).
// Tracker-less (DHT-only) links are valid.
func ParseMagnet(s string) (*Magnet, error) {
	if !strings.HasPrefix(s, "magnet:?") {
		return nil, fmt.Errorf("not a magnet link: %q", s)
	}
	spec, err := metainfo.ParseMagnetUri(s)
	if err != nil {
		return nil, fmt.Errorf("invalid magnet link: %w", err)
	}
	m := &Magnet{
		InfoHash:    spec.InfoHash,
		DisplayName: spec.DisplayName,
		Trackers:    append([]string(nil), spec.Trackers...),
	}
	return m, nil
}

// ValidateMagnet checks that s looks like a valid BitTorrent magnet URI.
func ValidateMagnet(s string) error {
	_, err := ParseMagnet(s)
	return err
}
