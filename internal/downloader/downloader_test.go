package downloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateName(t *testing.T) {
	for _, name := range []string{"file.iso", "My Album", "a"} {
		if err := validateName(name); err != nil {
			t.Errorf("validateName(%q) = %v, want nil", name, err)
		}
	}
	for _, name := range []string{"", "   ", ".", "..", "dir/file.iso"} {
		if err := validateName(name); err == nil {
			t.Errorf("validateName(%q) = nil, want error", name)
		}
	}
}

func TestNewCreatesStagingDir(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "new", "output")
	d, err := New(Options{OutputDir: outputDir, InfoHash: "0123456789abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	for _, path := range []string{outputDir, d.StagingDir()} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o755 {
			t.Errorf("%s permissions = %o, want 755", path, got)
		}
	}
}

func TestNewInfoOnlyCreatesNoDirectories(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "new", "output")
	d, err := New(Options{
		OutputDir: outputDir,
		InfoHash:  "0123456789abcdef",
		InfoOnly:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		t.Errorf("info-only run created %s or returned unexpected error: %v", outputDir, err)
	}
}

func TestNewConcurrentClients(t *testing.T) {
	outputDir := t.TempDir()
	first, err := New(Options{OutputDir: outputDir, InfoHash: "first"})
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()

	second, err := New(Options{
		OutputDir: outputDir,
		InfoHash:  "second",
		InfoOnly:  true,
	})
	if err != nil {
		t.Fatalf("starting concurrent info-only client: %v", err)
	}
	second.Close()
}

func TestNonConflictingPath(t *testing.T) {
	dir := t.TempDir()

	// No conflict: path returned as-is.
	p := filepath.Join(dir, "file.iso")
	got, err := nonConflictingPath(p)
	if err != nil {
		t.Fatal(err)
	}
	if got != p {
		t.Errorf("got %q, want %q", got, p)
	}

	// Conflict: " (1)" inserted before the extension.
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = nonConflictingPath(p)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "file (1).iso"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Second conflict: counter increments.
	if err := os.WriteFile(filepath.Join(dir, "file (1).iso"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = nonConflictingPath(p)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "file (2).iso"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Directories (multi-file torrents) conflict too.
	d := filepath.Join(dir, "album")
	if err := os.Mkdir(d, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err = nonConflictingPath(d)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "album (1)"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
