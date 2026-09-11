// Package httpapi exposes an EZ Sign writer through net/http.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"

	ezsign "github.com/whywaita/ezsign-go"
)

// Writer can be implemented by *ezsign.Client or a test double.
type Writer interface {
	Write(context.Context, []byte, ezsign.WriteOptions) (ezsign.Result, error)
}

// Config controls request size.
type Config struct {
	MaxBytes int64
}

// NewHandler serves GET /healthz and POST /v1/image. Successful POST responses
// wait for protocol completion. They do not claim optical verification.
func NewHandler(writer Writer, config Config) http.Handler {
	if config.MaxBytes <= 0 {
		config.MaxBytes = 10 << 20
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { reply(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /v1/image", func(w http.ResponseWriter, r *http.Request) {
		options := ezsign.WriteOptions{}
		if s := r.URL.Query().Get("rotation"); s != "" {
			n, err := strconv.Atoi(s)
			if err != nil || (n != 0 && n != 180) {
				fail(w, 400, "rotation must be 0 or 180")
				return
			}
			options.Rotation = n
		}
		if s := r.URL.Query().Get("dither"); s != "" {
			v, err := strconv.ParseBool(s)
			if err != nil {
				fail(w, 400, "dither must be true or false")
				return
			}
			options.NoDither = !v
		}
		ct, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || (ct != "image/png" && ct != "image/jpeg" && ct != "application/octet-stream") {
			fail(w, 415, "Content-Type must be image/png, image/jpeg or application/octet-stream")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, config.MaxBytes)
		defer r.Body.Close()
		data, err := io.ReadAll(r.Body)
		if err != nil {
			var large *http.MaxBytesError
			if errors.As(err, &large) {
				fail(w, 413, "image exceeds upload limit")
			} else {
				fail(w, 400, "cannot read image")
			}
			return
		}
		if len(data) == 0 {
			fail(w, 400, "image is empty")
			return
		}
		result, err := writer.Write(r.Context(), data, options)
		if err != nil {
			switch {
			case errors.Is(err, ezsign.ErrBusy):
				w.Header().Set("Retry-After", "30")
				fail(w, 409, "display is busy")
			case errors.Is(err, ezsign.ErrInvalidImage):
				fail(w, 400, "invalid PNG/JPEG or image dimensions exceed limit")
			case errors.Is(err, context.DeadlineExceeded):
				fail(w, 504, "update timed out; display state is unverified")
			case errors.Is(err, context.Canceled):
				fail(w, 408, "update canceled; display state is unverified")
			default:
				slog.Error("display update failed", "error", err)
				fail(w, 502, "display update failed; display state is unverified")
			}
			return
		}
		reply(w, 200, result)
	})
	return mux
}
func fail(w http.ResponseWriter, status int, message string) {
	reply(w, status, map[string]string{"error": message})
}
func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
