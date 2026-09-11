package rcs380

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestFrame(t *testing.T) {
	payload := []byte{0xd6, 0x2a, 1}
	f := encodeFrame(payload)
	want := []byte{0, 0, 255, 255, 255, 3, 0, 253, 0xd6, 0x2a, 1, 255, 0}
	if !bytes.Equal(f, want) {
		t.Fatalf("%x", f)
	}
	got, err := decodeFrame(f)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatal(got, err)
	}
	for _, i := range []int{0, 5, 7, 8, 11, 12} {
		bad := append([]byte(nil), f...)
		bad[i] ^= 1
		if _, err := decodeFrame(bad); err == nil {
			t.Fatalf("accepted corrupt offset %d", i)
		}
	}
}
func TestISOChainingAndWTX(t *testing.T) {
	n := 0
	dep := isoDep{miu: 3, fwt: time.Millisecond}
	dep.exchange = func(_ context.Context, b []byte, _ time.Duration) ([]byte, error) {
		n++
		switch n {
		case 1:
			if !bytes.Equal(b, []byte{0x12, 1, 2, 3}) {
				t.Fatal(b)
			}
			return []byte{0xa2}, nil
		case 2:
			if !bytes.Equal(b, []byte{3, 4}) {
				t.Fatal(b)
			}
			return []byte{0xf2, 2}, nil
		case 3:
			if !bytes.Equal(b, []byte{0xf2, 2}) {
				t.Fatal(b)
			}
			return []byte{0x13, 0x90}, nil
		case 4:
			if !bytes.Equal(b, []byte{0xa2}) {
				t.Fatal(b)
			}
			return []byte{2, 0}, nil
		}
		t.Fatal("unexpected exchange")
		return nil, nil
	}
	r, err := dep.transceive(context.Background(), []byte{1, 2, 3, 4})
	if err != nil || !bytes.Equal(r, []byte{0x90, 0}) {
		t.Fatal(r, err)
	}
}
func TestISORejectsMalformed(t *testing.T) {
	for _, r := range [][]byte{nil, {0xf2}, {0xf2, 0}, {0xf2, 60}, {3, 0x90, 0}, {0xa2}} {
		dep := isoDep{miu: 253, fwt: time.Millisecond, exchange: func(context.Context, []byte, time.Duration) ([]byte, error) { return r, nil }}
		if _, err := dep.transceive(context.Background(), []byte{1}); err == nil {
			t.Fatalf("accepted %x", r)
		}
	}
}
