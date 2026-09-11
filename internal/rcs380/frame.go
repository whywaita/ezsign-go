// Package rcs380 implements the Sony NFC Port-100 framing and ISO-DEP subset
// needed by EZ Sign. Protocol reference: nfcpy 1.0.4 rcs380/tt4 drivers.
package rcs380

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

var ack = []byte{0, 0, 255, 0, 255, 0}

func encodeFrame(data []byte) []byte {
	f := make([]byte, len(data)+10)
	copy(f, []byte{0, 0, 255, 255, 255})
	binary.LittleEndian.PutUint16(f[5:], uint16(len(data)))
	f[7] = 0 - f[5] - f[6]
	copy(f[8:], data)
	for _, b := range data {
		f[len(f)-2] -= b
	}
	return f
}
func decodeFrame(f []byte) ([]byte, error) {
	if len(f) < 10 || !bytes.Equal(f[:5], []byte{0, 0, 255, 255, 255}) {
		return nil, errors.New("invalid Sony frame header")
	}
	n := int(binary.LittleEndian.Uint16(f[5:]))
	if len(f) != n+10 || f[5]+f[6]+f[7] != 0 || f[len(f)-1] != 0 {
		return nil, errors.New("invalid Sony frame length/checksum")
	}
	sum := byte(0)
	for _, b := range f[8 : len(f)-1] {
		sum += b
	}
	if sum != 0 {
		return nil, errors.New("invalid Sony data checksum")
	}
	return f[8 : 8+n], nil
}

type isoDep struct {
	miu      int
	fwt      time.Duration
	pn       byte
	exchange func(context.Context, []byte, time.Duration) ([]byte, error)
}

func (d *isoDep) round(ctx context.Context, frame []byte, timeout time.Duration) ([]byte, error) {
	for count := 0; count < 256; count++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		r, err := d.exchange(ctx, frame, timeout)
		if err != nil {
			return nil, err
		}
		if len(r) == 0 {
			return nil, errors.New("empty ISO-DEP response")
		}
		if r[0]&0xfe != 0xf2 {
			return r, nil
		}
		if len(r) != 2 || r[1] < 1 || r[1] > 59 {
			return nil, errors.New("invalid ISO-DEP WTX")
		}
		frame = r
		timeout = time.Duration(r[1]) * d.fwt
	}
	return nil, errors.New("ISO-DEP WTX limit exceeded")
}
func (d *isoDep) transceive(ctx context.Context, cmd []byte) ([]byte, error) {
	if len(cmd) == 0 || d.miu < 1 {
		return nil, errors.New("empty APDU or invalid MIU")
	}
	var r []byte
	var err error
	for pos := 0; pos < len(cmd); {
		end := min(pos+d.miu, len(cmd))
		more := end < len(cmd)
		pcb := byte(2) | d.pn
		if more {
			pcb |= 0x10
		}
		frame := append([]byte{pcb}, cmd[pos:end]...)
		for retry := 0; retry < 3; retry++ {
			r, err = d.round(ctx, frame, 3*time.Second)
			if err != nil {
				return nil, err
			}
			if r[0] == 0xa2|(d.pn^1) {
				if retry == 2 {
					return nil, errors.New("ISO-DEP retransmission limit")
				}
				continue
			}
			break
		}
		if r[0]&1 != d.pn {
			return nil, errors.New("ISO-DEP block number mismatch")
		}
		if more {
			if len(r) != 1 || r[0]&0xfe != 0xa2 {
				return nil, errors.New("expected ISO-DEP ACK")
			}
		} else if r[0]&0xee != 2 {
			return nil, fmt.Errorf("expected ISO-DEP data, got %x", r)
		}
		d.pn ^= 1
		pos = end
	}
	result := append([]byte(nil), r[1:]...)
	for count := 0; r[0]&0x10 != 0; count++ {
		if count >= 255 {
			return nil, errors.New("ISO-DEP response too long")
		}
		r, err = d.round(ctx, []byte{0xa2 | d.pn}, 3*time.Second)
		if err != nil {
			return nil, err
		}
		if r[0]&0xee != 2 || r[0]&1 != d.pn {
			return nil, errors.New("invalid chained ISO-DEP response")
		}
		result = append(result, r[1:]...)
		d.pn ^= 1
	}
	return result, nil
}
