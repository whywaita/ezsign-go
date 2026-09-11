package ezsign

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

type DeviceInfo struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Colors   int    `json:"colors"`
	ScanType byte   `json:"scan_type"`
	Compress bool   `json:"compress"`
	UID      string `json:"uid"`
}

func parseInfo(response []byte) (DeviceInfo, error) {
	var info DeviceInfo
	if !bytes.HasSuffix(response, []byte{0x90, 0}) {
		return info, errors.New("information request failed")
	}
	b := response[:len(response)-2]
	fields := map[byte][]byte{}
	for len(b) > 0 {
		if len(b) < 2 || int(b[1]) > len(b)-2 {
			return info, errors.New("truncated information TLV")
		}
		tag, n := b[0], int(b[1])
		if _, ok := fields[tag]; ok {
			return info, errors.New("duplicate information TLV")
		}
		fields[tag] = b[2 : 2+n]
		b = b[2+n:]
	}
	for tag, n := range map[byte]int{0xa0: 7, 0xa1: 1, 0xc1: 4, 0xd1: 1} {
		if len(fields[tag]) < n {
			return info, errors.New("missing information field")
		}
	}
	a := fields[0xa0]
	info.Width = int(binary.BigEndian.Uint16(a[5:7]))
	info.Height = int(binary.BigEndian.Uint16(a[3:5]))
	info.Colors = 4
	if a[2] == 0x20 {
		info.Colors = 2
	}
	double := func(n int) bool { return n == 600 || n == 592 || n == 500 || n == 256 }
	if double(info.Height) {
		info.Height /= 2
		info.Colors = 4
	} else if double(info.Width) {
		info.Width /= 2
		info.Colors = 4
	}
	info.ScanType = fields[0xa1][0]
	info.Compress = fields[0xd1][0] != 0
	info.UID = hex.EncodeToString(fields[0xc1])
	return info, nil
}

// lzoLiteral emits a valid LZO1X stream using a literal run and end marker.
// It avoids a native LZO dependency; the maximum 2000-byte block becomes 2012 bytes.
func lzoLiteral(b []byte) []byte {
	n := len(b)
	var out []byte
	if n <= 238 {
		out = append(out, byte(17+n))
	} else {
		out = append(out, 0)
		remaining := n - 18
		for remaining > 255 {
			out = append(out, 0)
			remaining -= 255
		}
		out = append(out, byte(remaining))
	}
	out = append(out, b...)
	return append(out, 17, 0, 0)
}
func compressAPDUs(frame []byte) ([][]byte, error) {
	if len(frame) != 30000 {
		return nil, errors.New("frame must be 30000 bytes")
	}
	var commands [][]byte
	for start := 0; start < len(frame); start += 2000 {
		compressed := lzoCompress(frame[start : start+2000])
		for offset := 0; offset < len(compressed); offset += 250 {
			end := min(offset+250, len(compressed))
			last := byte(0)
			if end == len(compressed) {
				last = 1
			}
			cmd := []byte{0xf0, 0xd3, 0, last, byte(end - offset + 2), byte(start / 2000), byte(offset / 250)}
			commands = append(commands, append(cmd, compressed[offset:end]...))
		}
	}
	return commands, nil
}
func unhex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
func upload(ctx context.Context, exchange func(context.Context, []byte) ([]byte, error), commands [][]byte) (DeviceInfo, error) {
	var info DeviceInfo
	send := func(cmd string) ([]byte, error) { return exchange(ctx, unhex(cmd)) }
	ok := func(b []byte, err error) error {
		if err != nil {
			return err
		}
		if !bytes.Equal(b, []byte{0x90, 0}) {
			return fmt.Errorf("unexpected APDU response %x", b)
		}
		return nil
	}
	for _, cmd := range []string{"002000010420091210", "00a4040007d2760000850101"} {
		if err := ok(send(cmd)); err != nil {
			return info, err
		}
	}
	init, err := send("f0d801fe050000000000")
	if err != nil {
		return info, err
	}
	if !bytes.Equal(init, unhex("9000")) && !bytes.Equal(init, unhex("6a86")) && !bytes.Equal(init, unhex("6985")) {
		return info, fmt.Errorf("initialization response %x", init)
	}
	data, err := send("00d1000000")
	if err != nil {
		return info, err
	}
	info, err = parseInfo(data)
	if err != nil {
		return info, err
	}
	if info.Width != 400 || info.Height != 300 || info.Colors != 4 || info.ScanType != 1 || !info.Compress {
		return info, fmt.Errorf("unsupported device: %+v", info)
	}
	config, err := send("f0d8000005000000000e")
	if err != nil {
		return info, err
	}
	if !bytes.Equal(config, unhex("6985")) && (len(config) != 16 || !bytes.HasSuffix(config, unhex("9000"))) {
		return info, fmt.Errorf("configuration response %x", config)
	}
	for i, cmd := range commands {
		if err := ok(exchange(ctx, cmd)); err != nil {
			return info, fmt.Errorf("fragment %d: %w", i, err)
		}
	}
	response, err := send("f0d4058000")
	if err != nil {
		return info, err
	}
	if bytes.Equal(response, unhex("68c6")) || bytes.Equal(response, unhex("6986")) {
		for i := 0; i < 5; i++ {
			response, err = send("f0d4850000")
			if err != nil {
				return info, err
			}
			if bytes.Equal(response, unhex("9000")) || bytes.Equal(response, unhex("009000")) {
				return info, nil
			}
			if bytes.Equal(response, unhex("6600")) {
				break
			}
		}
		return info, fmt.Errorf("refresh fallback failed: %x", response)
	}
	if bytes.Equal(response, unhex("019000")) {
		return info, nil
	}
	if !bytes.Equal(response, unhex("9000")) && !bytes.Equal(response, unhex("009000")) {
		return info, fmt.Errorf("refresh failed: %x", response)
	}
	for i := 0; i < 300; i++ {
		response, err = send("f0de000001")
		if err != nil {
			return info, err
		}
		if bytes.Equal(response, unhex("9000")) || bytes.Equal(response, unhex("009000")) {
			return info, nil
		}
		if !bytes.Equal(response, unhex("019000")) {
			return info, fmt.Errorf("refresh status: %x", response)
		}
		select {
		case <-ctx.Done():
			return info, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return info, errors.New("refresh polling limit exceeded")
}
