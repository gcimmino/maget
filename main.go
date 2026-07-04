package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"maget/internal/cli"
	"maget/internal/downloader"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("maget: ")

	cfg, err := cli.Parse(os.Args[1:])
	if err != nil {
		if errors.Is(err, cli.ErrHelp) {
			cli.Usage(os.Stdout)
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "maget: %v\n\n", err)
		cli.Usage(os.Stderr)
		os.Exit(2)
	}

	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

func run(cfg *cli.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	d, err := downloader.New(downloader.Options{
		OutputDir: cfg.OutputDir,
		InfoHash:  cfg.Parsed.InfoHash.HexString(),
		Verbose:   cfg.Verbose,
	})
	if err != nil {
		return err
	}
	defer d.Close()

	t, err := d.Resolve(ctx, cfg.Magnet)
	if err != nil {
		return err
	}
	log.Printf("downloading %q (%d bytes)", t.Name(), t.Length())

	if err := d.Download(ctx, t); err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			return fmt.Errorf("interrupted — partial data kept in %s; rerun the same magnet to resume", d.StagingDir())
		case errors.Is(err, context.DeadlineExceeded):
			return fmt.Errorf("timed out after %s — partial data kept in %s; rerun the same magnet to resume", cfg.Timeout, d.StagingDir())
		}
		return err
	}
	path, err := d.Finalize(t)
	if err != nil {
		return err
	}
	log.Printf("done: %s", path)
	return nil
}
