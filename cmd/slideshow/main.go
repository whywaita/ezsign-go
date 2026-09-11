// Command slideshow repeatedly writes directory images to EZ Sign through the Go library.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	ezsign "github.com/whywaita/ezsign-go"
)

func images(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".png", ".jpg", ".jpeg":
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no PNG/JPEG images in %s", dir)
	}
	return files, nil
}

type imageWriter interface {
	Write(context.Context, []byte, ezsign.WriteOptions) (ezsign.Result, error)
}

func writeFile(ctx context.Context, writer imageWriter, path string, options ezsign.WriteOptions) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	// Bound allocation before passing the image to the library's validator.
	const maxBytes = 10 << 20
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxBytes {
		return fmt.Errorf("%w: image exceeds 10 MiB", ezsign.ErrInvalidImage)
	}
	_, err = writer.Write(ctx, data, options)
	return err
}

// loop waits after each completed write, so writes never overlap.
func loop(ctx context.Context, files []string, interval time.Duration, write func(context.Context, string) error) error {
	if len(files) == 0 || interval <= 0 {
		return errors.New("images and a positive interval are required")
	}
	for i := 0; ; i = (i + 1) % len(files) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := write(ctx, files[i]); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("failed %s: %v", files[i], err)
		} else {
			log.Printf("updated %s", files[i])
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func main() {
	dir := flag.String("dir", "", "directory containing PNG/JPEG images (required)")
	interval := flag.Duration("interval", 60*time.Second, "wait after each write completes")
	rotation := flag.Int("rotate", 0, "rotation: 0 or 180")
	noDither := flag.Bool("no-dither", false, "disable dithering")
	timeout := flag.Duration("timeout", 100*time.Second, "timeout per write")
	flag.Parse()
	if *dir == "" || *interval <= 0 || *timeout <= 0 || flag.NArg() != 0 {
		log.Fatal("use -dir PATH with positive -interval and -timeout")
	}
	if *rotation != 0 && *rotation != 180 {
		log.Fatal("rotate must be 0 or 180")
	}
	files, err := images(*dir)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client, err := ezsign.New(ezsign.Config{Timeout: *timeout})
	if err != nil {
		log.Fatal(err)
	}
	options := ezsign.WriteOptions{Rotation: *rotation, NoDither: *noDither}
	err = loop(ctx, files, *interval, func(ctx context.Context, path string) error { return writeFile(ctx, client, path, options) })
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}
