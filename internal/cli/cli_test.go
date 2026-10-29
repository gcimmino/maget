package cli

import (
	"testing"
	"time"
)

const validMagnetHex = "magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=file.iso"
const validMagnetBase32 = "magnet:?xt=urn:btih:BFVNVMWTM33ULYVDXOSPJTDFPZZ2GCVU"

func TestParseValid(t *testing.T) {
	cfg, err := Parse([]string{"-o", "/tmp/out", "-v", "-i", "-t", "5m", validMagnetHex})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Magnet != validMagnetHex {
		t.Errorf("magnet = %q", cfg.Magnet)
	}
	if cfg.OutputDir != "/tmp/out" {
		t.Errorf("output = %q", cfg.OutputDir)
	}
	if !cfg.Verbose {
		t.Error("verbose = false")
	}
	if !cfg.Info {
		t.Error("info = false")
	}
	if cfg.Timeout != 5*time.Minute {
		t.Errorf("timeout = %v", cfg.Timeout)
	}
}

func TestParseDefaults(t *testing.T) {
	cfg, err := Parse([]string{validMagnetBase32})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OutputDir != "." || cfg.Verbose || cfg.Info || cfg.Timeout != 0 {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}

func TestParseInfoLongFlag(t *testing.T) {
	cfg, err := Parse([]string{"--info", validMagnetHex})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Info {
		t.Error("info = false")
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"no args", nil},
		{"not a magnet", []string{"http://example.com/file.torrent"}},
		{"bad hash length", []string{"magnet:?xt=urn:btih:abc123"}},
		{"non-hex 40 chars", []string{"magnet:?xt=urn:btih:zzzz5a7a6183aae1e09d831df6748d566095a10"}},
		{"no xt param", []string{"magnet:?dn=file.iso"}},
		{"extra args", []string{validMagnetHex, "extra"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Parse(tc.args); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestParseHelp(t *testing.T) {
	if _, err := Parse([]string{"-h"}); err != ErrHelp {
		t.Errorf("expected ErrHelp, got %v", err)
	}
}

func TestValidateMagnetHashes(t *testing.T) {
	if err := ValidateMagnet(validMagnetHex); err != nil {
		t.Errorf("hex hash: %v", err)
	}
	if err := ValidateMagnet(validMagnetBase32); err != nil {
		t.Errorf("base32 hash: %v", err)
	}
	// DHT-only (no trackers) must be accepted.
	if err := ValidateMagnet("magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10"); err != nil {
		t.Errorf("tracker-less magnet: %v", err)
	}
}

func TestParseMagnetComponents(t *testing.T) {
	m, err := ParseMagnet(validMagnetHex + "&tr=udp%3A%2F%2Ftracker.example.com%3A6969&tr=http%3A%2F%2Ftracker2.example.com%2Fannounce")
	if err != nil {
		t.Fatal(err)
	}
	if got := m.InfoHash.HexString(); got != "08ada5a7a6183aae1e09d831df6748d566095a10" {
		t.Errorf("info hash = %q", got)
	}
	if m.DisplayName != "file.iso" {
		t.Errorf("display name = %q", m.DisplayName)
	}
	if len(m.Trackers) != 2 {
		t.Errorf("trackers = %v", m.Trackers)
	}
}

func TestParseMagnetHexAndBase32Match(t *testing.T) {
	// Same info hash in both encodings must parse to identical bytes.
	hex, err := ParseMagnet("magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10")
	if err != nil {
		t.Fatal(err)
	}
	b32, err := ParseMagnet("magnet:?xt=urn:btih:BCW2LJ5GDA5K4HQJ3AY56Z2I2VTASWQQ")
	if err != nil {
		t.Fatal(err)
	}
	if hex.InfoHash != b32.InfoHash {
		t.Errorf("hex %v != base32 %v", hex.InfoHash, b32.InfoHash)
	}
}
