// Package ezsign renders and writes images to a 4.2-inch four-color EZ Sign.
// Image conversion, LZO1X encoding, APDUs and RC-S380 ISO-DEP are implemented
// in Go. Hardware access uses libusb through CGO; Python is not required.
package ezsign

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/whywaita/ezsign-go/internal/rcs380"
	"image"
	_ "image/jpeg"
	"image/png"
	"time"
)

var (
	ErrBusy         = errors.New("display is already being written")
	ErrInvalidImage = errors.New("invalid image or rendering options")
)

// Transport is an activated ISO-DEP connection. Each Write opens and closes one.
type Transport interface {
	Transceive(context.Context, []byte) ([]byte, error)
	Close() error
}

// Config controls limits and optionally supplies a different APDU transport.
type Config struct {
	Timeout       time.Duration
	MaxImageBytes int
	MaxPixels     int64
	Open          func(context.Context) (Transport, error)
}
type WriteOptions struct {
	// Rotation is relative to the installed display orientation: 0 (default) or 180.
	Rotation int
	NoDither bool
}

// Result reports protocol completion rather than optical verification.
type Result struct {
	Status      string          `json:"status"`
	Width       int             `json:"width"`
	Height      int             `json:"height"`
	FrameSHA256 string          `json:"frame_sha256"`
	DurationMS  int64           `json:"duration_ms"`
	PreviewPNG  []byte          `json:"-"`
	Frame       []byte          `json:"-"`
	Session     json.RawMessage `json:"-"`
}

// Client serializes access. All default clients share a process-wide USB lock.
type Client struct {
	timeout   time.Duration
	maxBytes  int
	maxPixels int64
	busy      chan struct{}
	open      func(context.Context) (Transport, error)
}

var readerBusy = make(chan struct{}, 1)

func New(config Config) (*Client, error) {
	if config.Timeout == 0 {
		config.Timeout = 90 * time.Second
	}
	if config.MaxImageBytes == 0 {
		config.MaxImageBytes = 10 << 20
	}
	if config.MaxPixels == 0 {
		config.MaxPixels = 20_000_000
	}
	if config.Timeout < 0 || config.MaxImageBytes < 0 || config.MaxPixels < 0 {
		return nil, errors.New("limits must be positive")
	}
	busy := make(chan struct{}, 1)
	if config.Open == nil {
		config.Open = func(ctx context.Context) (Transport, error) { return rcs380.Open(ctx) }
		busy = readerBusy
	}
	return &Client{timeout: config.Timeout, maxBytes: config.MaxImageBytes, maxPixels: config.MaxPixels, busy: busy, open: config.Open}, nil
}
func (c *Client) Write(ctx context.Context, data []byte, o WriteOptions) (Result, error) {
	return c.execute(ctx, data, o, true)
}
func (c *Client) Render(ctx context.Context, data []byte, o WriteOptions) (Result, error) {
	return c.execute(ctx, data, o, false)
}
func (c *Client) WriteImage(ctx context.Context, img image.Image, o WriteOptions) (Result, error) {
	if img == nil {
		return Result{}, fmt.Errorf("%w: nil image", ErrInvalidImage)
	}
	if !c.validSize(img.Bounds().Dx(), img.Bounds().Dy()) {
		return Result{}, fmt.Errorf("%w: dimensions exceed limit", ErrInvalidImage)
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return Result{}, err
	}
	return c.Write(ctx, b.Bytes(), o)
}
func (c *Client) validSize(w, h int) bool { return w > 0 && h > 0 && int64(w) <= c.maxPixels/int64(h) }
func (c *Client) execute(ctx context.Context, data []byte, o WriteOptions, write bool) (result Result, err error) {
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if o.Rotation != 0 && o.Rotation != 180 {
		return result, fmt.Errorf("%w: rotation must be 0 or 180", ErrInvalidImage)
	}
	if len(data) == 0 || len(data) > c.maxBytes {
		return result, fmt.Errorf("%w: input size exceeds limit or is empty", ErrInvalidImage)
	}
	cfg, format, e := image.DecodeConfig(bytes.NewReader(data))
	if e != nil || (format != "png" && format != "jpeg") || !c.validSize(cfg.Width, cfg.Height) {
		return result, fmt.Errorf("%w: expected PNG/JPEG within pixel limit", ErrInvalidImage)
	}
	select {
	case c.busy <- struct{}{}:
		defer func() { <-c.busy }()
	default:
		return result, ErrBusy
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	started := time.Now()
	defer func() { result.DurationMS = time.Since(started).Milliseconds() }()
	img, _, e := image.Decode(bytes.NewReader(data))
	if e != nil {
		return result, fmt.Errorf("%w: %v", ErrInvalidImage, e)
	}
	img = orientImage(img, jpegOrientation(data))
	result.PreviewPNG, result.Frame, err = renderNative(ctx, img, o)
	if err != nil {
		return result, err
	}
	result.Width, result.Height = 400, 300
	sum := sha256.Sum256(result.Frame)
	result.FrameSHA256 = hex.EncodeToString(sum[:])
	result.Status = "rendered"
	if !write {
		return result, nil
	}
	type record struct {
		Request  string  `json:"request"`
		Response string  `json:"response,omitempty"`
		Error    string  `json:"error,omitempty"`
		Elapsed  float64 `json:"elapsed_seconds"`
	}
	log := struct {
		Started   time.Time  `json:"started"`
		Finished  time.Time  `json:"finished"`
		Status    string     `json:"status"`
		Error     string     `json:"error,omitempty"`
		Rotation  int        `json:"rotation"`
		FrameHash string     `json:"frame_sha256"`
		Device    DeviceInfo `json:"device"`
		Exchanges []record   `json:"exchanges"`
	}{Started: started, Status: "failed", Rotation: o.Rotation, FrameHash: result.FrameSHA256}
	defer func() {
		log.Finished = time.Now()
		if err != nil {
			log.Error = err.Error()
			result.Status = "failed"
		}
		result.Session, _ = json.MarshalIndent(log, "", "  ")
	}()
	transport, err := c.open(ctx)
	if err != nil {
		return result, err
	}
	defer transport.Close()
	commands, err := compressAPDUs(result.Frame)
	if err != nil {
		return result, err
	}
	exchange := func(ctx context.Context, cmd []byte) ([]byte, error) {
		start := time.Now()
		response, e := transport.Transceive(ctx, cmd)
		entry := record{Request: hex.EncodeToString(cmd), Response: hex.EncodeToString(response), Elapsed: time.Since(start).Seconds()}
		if e != nil {
			entry.Error = e.Error()
		}
		log.Exchanges = append(log.Exchanges, entry)
		return response, e
	}
	log.Device, err = upload(ctx, exchange, commands)
	if err != nil {
		return result, err
	}
	result.Status = "protocol_complete_visual_check_required"
	log.Status = result.Status
	return result, nil
}
