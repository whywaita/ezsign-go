package httpapi

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	ezsign "github.com/whywaita/ezsign-go"
)

type fakeWriter struct {
	calls   int
	options ezsign.WriteOptions
	data    string
	err     error
}

func (f *fakeWriter) Write(_ context.Context, b []byte, o ezsign.WriteOptions) (ezsign.Result, error) {
	f.calls++
	f.options = o
	f.data = string(b)
	return ezsign.Result{Status: "protocol_complete_visual_check_required", Width: 400, Height: 300}, f.err
}
func TestWriteImage(t *testing.T) {
	w := &fakeWriter{}
	h := NewHandler(w, Config{})
	r := httptest.NewRequest("POST", "/v1/image?rotation=180&dither=false", strings.NewReader("image bytes"))

	r.Header.Set("Content-Type", "image/png")
	out := httptest.NewRecorder()
	h.ServeHTTP(out, r)
	if out.Code != 200 || w.calls != 1 || w.options.Rotation != 180 || !w.options.NoDither || w.data != "image bytes" {
		t.Fatalf("%d %s %+v", out.Code, out.Body, w)
	}
}
func TestRejectBeforeWriting(t *testing.T) {
	for _, tc := range []struct {
		name, url, ct, body string
		limit               int64
		status              int
	}{
		{"rotation", "/v1/image?rotation=90", "image/png", "data", 100, 400},
		{"dither", "/v1/image?dither=maybe", "image/png", "data", 100, 400},
		{"type", "/v1/image", "text/html", "data", 100, 415},
		{"large", "/v1/image", "image/png", "longdata", 3, 413},
		{"empty", "/v1/image", "image/png", "", 100, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := &fakeWriter{}
			h := NewHandler(w, Config{MaxBytes: tc.limit})
			r := httptest.NewRequest("POST", tc.url, strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.ct)
			out := httptest.NewRecorder()
			h.ServeHTTP(out, r)
			if out.Code != tc.status || w.calls != 0 {
				t.Fatalf("%d %s calls=%d", out.Code, out.Body, w.calls)
			}
		})
	}
}
func TestFailureStatus(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{{ezsign.ErrBusy, 409}, {ezsign.ErrInvalidImage, 400}, {context.DeadlineExceeded, 504}, {context.Canceled, 408}, {errors.New("usb unavailable /private/path"), 502}} {
		w := &fakeWriter{err: tc.err}
		h := NewHandler(w, Config{})
		r := httptest.NewRequest("POST", "/v1/image", strings.NewReader("data"))
		r.Header.Set("Content-Type", "image/png")
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		if out.Code != tc.status || strings.Contains(out.Body.String(), "/private/path") {
			t.Fatalf("%d %s", out.Code, out.Body)
		}
	}
}
func TestHealthAndMethod(t *testing.T) {
	h := NewHandler(&fakeWriter{}, Config{})
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"GET", "/healthz", 200}, {"GET", "/v1/image", 405}, {"POST", "/missing", 404}} {
		out := httptest.NewRecorder()
		h.ServeHTTP(out, httptest.NewRequest(tc.method, tc.path, nil))
		if out.Code != tc.status {
			t.Fatal(out.Code)
		}
	}
}
