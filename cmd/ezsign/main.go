// Command ezsign renders PNG/JPEG images and optionally writes an EZ Sign.
package main

import (
	"context"
	"flag"
	"fmt"
	ezsign "github.com/whywaita/ezsign-go"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	write := flag.Bool("write", false, "write the physical display")
	rotation := flag.Int("rotate", 0, "rotation: 0 or 180")
	noDither := flag.Bool("no-dither", false, "disable dithering")
	out := flag.String("output", "output", "output directory")
	timeout := flag.Duration("timeout", 90*time.Second, "operation timeout")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: ezsign [flags] image.png")
		os.Exit(2)
	}
	data, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	client, err := ezsign.New(ezsign.Config{Timeout: *timeout})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	options := ezsign.WriteOptions{Rotation: *rotation, NoDither: *noDither}
	var result ezsign.Result
	if *write {
		result, err = client.Write(ctx, data, options)
	} else {
		result, err = client.Render(ctx, data, options)
	}
	if mkdirErr := os.MkdirAll(*out, 0700); mkdirErr != nil {
		fmt.Fprintln(os.Stderr, mkdirErr)
		os.Exit(1)
	}
	for name, b := range map[string][]byte{"preview.png": result.PreviewPNG, "frame.bin": result.Frame, "session.json": result.Session} {
		if len(b) > 0 {
			if e := os.WriteFile(filepath.Join(*out, name), b, 0600); e != nil {
				fmt.Fprintln(os.Stderr, e)
				os.Exit(1)
			}
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s: %dx%d frame=%s duration=%dms\n", result.Status, result.Width, result.Height, result.FrameSHA256, result.DurationMS)
}
