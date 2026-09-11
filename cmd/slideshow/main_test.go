package main

import (
	"context"
	"errors"
	ezsign "github.com/whywaita/ezsign-go"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestImages(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.JPG", "a.png", "ignore.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := images(dir)
	if err != nil || !reflect.DeepEqual(got, []string{filepath.Join(dir, "a.png"), filepath.Join(dir, "b.JPG")}) {
		t.Fatal(got, err)
	}
	if _, err := images(t.TempDir()); err == nil {
		t.Fatal("accepted empty directory")
	}
}
func TestLoopWrapsAndCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var got []string
	err := loop(ctx, []string{"a", "b"}, time.Millisecond, func(_ context.Context, p string) error {
		got = append(got, p)
		if len(got) == 3 {
			cancel()
		}
		return nil
	})
	if err != context.Canceled || !reflect.DeepEqual(got, []string{"a", "b", "a"}) {
		t.Fatal(got, err)
	}
}

type fakeWriter struct {
	data    []byte
	options ezsign.WriteOptions
	err     error
	calls   int
}

func (w *fakeWriter) Write(ctx context.Context, b []byte, o ezsign.WriteOptions) (ezsign.Result, error) {
	w.calls++
	w.data = b
	w.options = o
	return ezsign.Result{}, w.err
}
func TestWriteFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(p, []byte("image data"), 0600); err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("USB failed")
	for _, wantErr := range []error{nil, sentinel} {
		w := &fakeWriter{err: wantErr}
		options := ezsign.WriteOptions{Rotation: 180, NoDither: true}
		err := writeFile(context.Background(), w, p, options)
		if !errors.Is(err, wantErr) || string(w.data) != "image data" || w.options != options || w.calls != 1 {
			t.Fatal(w, err)
		}
	}
	w := &fakeWriter{}
	if err := writeFile(context.Background(), w, p+"missing", ezsign.WriteOptions{}); err == nil || w.calls != 0 {
		t.Fatal(err, w.calls)
	}
}
