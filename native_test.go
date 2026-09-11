package ezsign

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"
)

func TestNativeInfoAndFrames(t *testing.T) {
	infoBytes := unhex("a007f0072002580190a10101c10401020304d101019000")
	info, err := parseInfo(infoBytes)
	if err != nil || info.Width != 400 || info.Height != 300 || !info.Compress {
		t.Fatal(info, err)
	}
	cmds, err := compressAPDUs(bytes.Repeat([]byte{0x55}, 30000))
	if err != nil || len(cmds) == 0 {
		t.Fatal(err)
	}
	for _, cmd := range cmds {
		if len(cmd) > 257 || int(cmd[4]) != len(cmd)-5 {
			t.Fatal("bad APDU")
		}
	}
}
func TestNativeUploadSequence(t *testing.T) {
	// Synthetic responses exercise the protocol without private capture files.
	log := struct {
		Exchanges []struct{ Request, Response string }
	}{
		Exchanges: []struct{ Request, Response string }{
			{"002000010420091210", "9000"},
			{"00a4040007d2760000850101", "9000"},
			{"f0d801fe050000000000", "6a86"},
			{"00d1000000", "a007f0072002580190a10101c10401020304d101019000"},
			{"f0d8000005000000000e", "345f636f6c6f722053637265656e9000"},
			{"f0d30001050000110000", "9000"},
			{"f0d4058000", "68c6"},
			{"f0d4850000", "9000"},
		},
	}
	n := 0
	var cmds [][]byte
	for _, e := range log.Exchanges {
		c, _ := hex.DecodeString(e.Request)
		if bytes.HasPrefix(c, []byte{0xf0, 0xd3}) {
			cmds = append(cmds, c)
		}
	}
	exchange := func(_ context.Context, cmd []byte) ([]byte, error) {
		want, _ := hex.DecodeString(log.Exchanges[n].Request)
		if !bytes.Equal(cmd, want) {
			t.Fatalf("%d got %x want %x", n, cmd, want)
		}
		r, _ := hex.DecodeString(log.Exchanges[n].Response)
		n++
		return r, nil
	}
	if _, err := upload(context.Background(), exchange, cmds); err != nil {
		t.Fatal(err)
	}
	if n != len(log.Exchanges) {
		t.Fatal(n)
	}
}
func TestLZOLiteralVectors(t *testing.T) {
	for _, n := range []int{1, 3, 4, 18, 238, 239, 2000} {
		b := bytes.Repeat([]byte{7}, n)
		compressed := lzoLiteral(b)
		if len(compressed) < n || !bytes.Equal(compressed[len(compressed)-3:], []byte{17, 0, 0}) {
			t.Fatal(n)
		}
	}
}
