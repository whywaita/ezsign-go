package ezsign

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestLZOMatchVector(t *testing.T) {
	want, _ := hex.DecodeString("14616263270800110000")
	if got := lzoCompress([]byte("abcabcabcabc")); !bytes.Equal(got, want) {
		t.Fatalf("%x", got)
	}
}
func TestUniformFrameCompresses(t *testing.T) {
	for _, value := range []byte{0, 0x55, 0xaa, 0xff} {
		commands, err := compressAPDUs(bytes.Repeat([]byte{value}, 30000))
		if err != nil || len(commands) != 15 {
			t.Fatalf("uniform %02x: %d commands, %v", value, len(commands), err)
		}
	}
}
