// Package downloader wraps the anacrolix/torrent client to fetch files
// from magnet links.
package downloader

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	alog "github.com/anacrolix/log"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
	"github.com/anacrolix/torrent/storage/disabled"
)

// Options configures a Downloader.
type Options struct {
	// OutputDir is where downloaded files end up.
	OutputDir string
	// InfoHash (hex) keys the staging directory, so re-running the same
	// magnet resumes a previous partial download.
	InfoHash string
	// Verbose enables the torrent client's internal logging.
	Verbose bool
	// InfoOnly disables disk storage because no content will be downloaded.
	InfoOnly bool
}

// Downloader owns a torrent client for the lifetime of a download.
// Data is downloaded into a hidden staging directory inside OutputDir and
// moved to its final (non-conflicting) name by Finalize.
type Downloader struct {
	client     *torrent.Client
	outputDir  string
	stagingDir string
}

// New creates a Downloader writing into opts.OutputDir. Information-only
// runs use disabled storage and leave no files on disk.
func New(opts Options) (*Downloader, error) {
	stagingDir := filepath.Join(opts.OutputDir, ".maget-"+opts.InfoHash)

	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = stagingDir
	// Let the OS choose a free port so concurrent maget processes can run.
	cfg.ListenPort = 0
	if opts.InfoOnly {
		cfg.DefaultStorage = metadataOnlyStorage{}
	} else if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating staging directory: %w", err)
	}
	// One-shot downloader: UPnP port mapping is unnecessary and noisy on
	// routers that reject it.
	cfg.NoDefaultPortForwarding = true
	// Peer discovery: trackers from the magnet link plus DHT (enabled by
	// default), so tracker-less magnets still work. Metadata is fetched
	// from peers via the ut_metadata extension.
	if !opts.Verbose {
		cfg.Logger = cfg.Logger.WithFilterLevel(alog.Error)
	}

	client, err := torrent.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating torrent client: %w", err)
	}
	return &Downloader{
		client:     client,
		outputDir:  opts.OutputDir,
		stagingDir: stagingDir,
	}, nil
}

// Close shuts down the torrent client and all its connections.
func (d *Downloader) Close() {
	d.client.Close()
	<-d.client.Closed()
}

// metadataOnlyStorage adapts the disabled backend to the current storage
// interface, allowing metadata resolution without any filesystem storage.
type metadataOnlyStorage struct{}

func (metadataOnlyStorage) OpenTorrent(
	_ context.Context,
	info *metainfo.Info,
	infoHash metainfo.Hash,
) (storage.TorrentImpl, error) {
	return disabled.Client{}.OpenTorrent(info, infoHash)
}

// Resolve adds the magnet link to the client, discovers peers, and waits
// until the torrent metadata (file list, piece hashes) has been fetched
// or ctx is done. It fails if the metadata does not contain a usable
// file name.
func (d *Downloader) Resolve(ctx context.Context, magnet string) (*torrent.Torrent, error) {
	t, err := d.client.AddMagnet(magnet)
	if err != nil {
		return nil, fmt.Errorf("adding magnet: %w", err)
	}

	log.Printf("fetching metadata for %s ...", t.InfoHash().HexString())
	select {
	case <-t.GotInfo():
	case <-ctx.Done():
		t.Drop()
		return nil, fmt.Errorf("fetching torrent metadata: %w", ctx.Err())
	}

	if err := validateName(t.Info().Name); err != nil {
		t.Drop()
		return nil, fmt.Errorf("cannot determine file name from torrent metadata: %w", err)
	}
	return t, nil
}

// validateName rejects metadata names that are empty or unsafe to use as a
// file name in the output directory.
func validateName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return fmt.Errorf("name is empty")
	case name == "." || name == "..":
		return fmt.Errorf("invalid name %q", name)
	case strings.ContainsRune(name, os.PathSeparator):
		return fmt.Errorf("name %q contains a path separator", name)
	}
	return nil
}

// Download starts downloading all files of the torrent and blocks until
// every piece has been received and verified, or ctx is done. Piece
// verification (SHA-1), peer management, and retries are handled by the
// torrent client. A progress line (%, speed, peers, ETA) is written to
// stderr once per second.
func (d *Downloader) Download(ctx context.Context, t *torrent.Torrent) error {
	t.DownloadAll()

	progressCtx, stopProgress := context.WithCancel(ctx)
	progressDone := make(chan struct{})
	go func() {
		defer close(progressDone)
		reportProgress(progressCtx, t)
	}()

	waitComplete(ctx, t)
	stopProgress()
	<-progressDone

	if !t.Complete().Bool() {
		t.Drop()
		return fmt.Errorf("downloading %q: %w", t.Name(), ctx.Err())
	}
	return nil
}

// StagingDir returns the directory holding partial download data, kept
// across runs so the same magnet can resume.
func (d *Downloader) StagingDir() string {
	return d.stagingDir
}

// reportProgress writes a single updating progress line to stderr until
// ctx is done.
func reportProgress(ctx context.Context, t *torrent.Torrent) {
	const interval = time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	prev := t.BytesCompleted()
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr)
			return
		case <-ticker.C:
			completed, total := t.BytesCompleted(), t.Length()
			speed := float64(completed-prev) / interval.Seconds()
			prev = completed

			pct := 0.0
			if total > 0 {
				pct = float64(completed) / float64(total) * 100
			}
			eta := "?"
			if speed > 0 {
				d := time.Duration(float64(total-completed)/speed) * time.Second
				eta = d.Round(time.Second).String()
			}
			fmt.Fprintf(os.Stderr, "\r\033[K%5.1f%%  %s / %s  %s/s  peers %d  ETA %s",
				pct, formatBytes(completed), formatBytes(total),
				formatBytes(int64(speed)), t.Stats().ActivePeers, eta)
		}
	}
}

// formatBytes renders n in human-readable binary units.
func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// Finalize moves the completed download (a single file, or a directory for
// multi-file torrents — folder structure preserved) from the staging
// directory into the output directory, avoiding overwriting existing files
// by appending " (1)", " (2)", ... to the name. It returns the final path
// and removes the staging directory.
func (d *Downloader) Finalize(t *torrent.Torrent) (string, error) {
	src := filepath.Join(d.stagingDir, t.Name())
	dst, err := nonConflictingPath(filepath.Join(d.outputDir, t.Name()))
	if err != nil {
		return "", err
	}
	if err := os.Rename(src, dst); err != nil {
		return "", fmt.Errorf("moving download to output directory: %w", err)
	}
	if err := os.RemoveAll(d.stagingDir); err != nil {
		log.Printf("warning: could not remove staging directory %s: %v", d.stagingDir, err)
	}
	return dst, nil
}

// nonConflictingPath returns path if nothing exists there, otherwise the
// first "name (N).ext" variant that does not exist.
func nonConflictingPath(path string) (string, error) {
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return path, nil
	} else if err != nil {
		return "", err
	}
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", stem, i, ext)
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
}

// waitComplete blocks until the torrent is fully downloaded or ctx is done.
func waitComplete(ctx context.Context, t *torrent.Torrent) {
	sub := t.SubscribePieceStateChanges()
	defer sub.Close()
	for {
		if t.Complete().Bool() {
			return
		}
		select {
		case <-sub.Values:
		case <-ctx.Done():
			return
		}
	}
}
