package ezsign

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"
)

func testPNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestRejectInvalidInput(t *testing.T) {
	c, _ := New(Config{Open: func(context.Context) (Transport, error) { t.Fatal("opened hardware"); return nil, nil }})
	for _, tc := range []struct {
		b []byte
		o WriteOptions
	}{{[]byte("bad"), WriteOptions{}}, {testPNG(t), WriteOptions{Rotation: 90}}} {
		if _, err := c.Write(context.Background(), tc.b, tc.o); !errors.Is(err, ErrInvalidImage) {
			t.Fatal(err)
		}
	}
}
func TestNativeRendering(t *testing.T) {
	c, _ := New(Config{Open: func(context.Context) (Transport, error) { t.Fatal("Render opened hardware"); return nil, nil }})
	r, err := c.Render(context.Background(), testPNG(t), WriteOptions{})
	if err != nil || r.Status != "rendered" || !bytes.Equal(r.Frame, bytes.Repeat([]byte{0x55}, 30000)) {
		t.Fatal(r.Status, err)
	}
	im := image.NewNRGBA(image.Rect(0, 0, 400, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 400; x++ {
			im.Set(x, y, color.White)
		}
	}
	im.Set(0, 0, color.NRGBA{255, 0, 0, 255})
	_, frame, err := renderNative(context.Background(), im, WriteOptions{NoDither: true})
	if err != nil || frame[99] != 0x57 {
		t.Fatal(err, frame[99])
	}
	_, frame, err = renderNative(context.Background(), im, WriteOptions{Rotation: 180, NoDither: true})
	if err != nil || frame[29900] != 0xd5 {
		t.Fatal(err, frame[29900])
	}
}
func TestBusyAndCancellation(t *testing.T) {
	entered := make(chan struct{})
	c, _ := New(Config{Open: func(ctx context.Context) (Transport, error) { close(entered); <-ctx.Done(); return nil, ctx.Err() }})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	data := testPNG(t)
	go func() { _, err := c.Write(ctx, data, WriteOptions{}); done <- err }()
	<-entered
	if _, err := c.Write(context.Background(), data, WriteOptions{}); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("did not cancel")
	}
}
func TestHardwareFailureNotSuccess(t *testing.T) {
	c, _ := New(Config{Open: func(context.Context) (Transport, error) { return nil, errors.New("reader missing") }})
	r, err := c.Write(context.Background(), testPNG(t), WriteOptions{})
	if err == nil || r.Status != "failed" || len(r.Session) == 0 {
		t.Fatal(r, err)
	}
}
