package rcs380

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// Reader owns one claimed USB interface and one ISO-DEP session.
type Reader struct {
	usb *usb
	dep isoDep
	UID string
}

func hx(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
func (r *Reader) command(ctx context.Context, code byte, data []byte) ([]byte, error) {
	if err := r.usb.write(ctx, encodeFrame(append([]byte{0xd6, code}, data...))); err != nil {
		return nil, err
	}
	a, err := r.usb.read(ctx, time.Second)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(a, ack) {
		return nil, fmt.Errorf("Sony command %02x: expected ACK, got %x", code, a)
	}
	frame, err := r.usb.read(ctx, 7*time.Second)
	if err != nil {
		return nil, err
	}
	response, err := decodeFrame(frame)
	if err != nil {
		return nil, err
	}
	if len(response) < 2 || response[0] != 0xd7 || response[1] != code+1 {
		return nil, fmt.Errorf("Sony response code: %x", response)
	}
	return response[2:], nil
}
func (r *Reader) setting(ctx context.Context, code byte, data []byte) error {
	b, err := r.command(ctx, code, data)
	if err != nil {
		return err
	}
	if len(b) != 1 || b[0] != 0 {
		return fmt.Errorf("Sony command %02x status: %x", code, b)
	}
	return nil
}
func (r *Reader) rf(ctx context.Context, data []byte, timeout time.Duration) ([]byte, error) {
	ms := max(1, timeout.Milliseconds())
	units := min((ms+1)*10, 65535)
	payload := []byte{byte(units), byte(units >> 8)}
	payload = append(payload, data...)
	b, err := r.command(ctx, 4, payload)
	if err != nil {
		return nil, err
	}
	if len(b) < 5 {
		return nil, errors.New("short InCommRF response")
	}
	if status := binary.LittleEndian.Uint32(b); status != 0 {
		return nil, fmt.Errorf("InCommRF status %08x", status)
	}
	return b[5:], nil
}

var defaults = hx("00180101020103000400050006000708080009000a000b000c000e040f001000110012001306")

func (r *Reader) exchange(ctx context.Context, b []byte, timeout time.Duration) ([]byte, error) {
	for _, s := range []struct {
		c byte
		d []byte
	}{{0, hx("02030f03")}, {2, defaults}, {2, hx("04010501")}} {
		if err := r.setting(ctx, s.c, s.d); err != nil {
			return nil, err
		}
	}
	return r.rf(ctx, b, timeout)
}

// Open activates a single NFC-A ISO-DEP tag. Other tag families are unsupported.
func Open(ctx context.Context) (reader *Reader, err error) {
	u, err := openUSB()
	if err != nil {
		return nil, err
	}
	r := &Reader{usb: u}
	defer func() {
		if err != nil {
			r.Close()
		}
	}()
	if err = u.write(ctx, ack); err != nil {
		return nil, err
	}
	for i := 0; i < 8; i++ {
		if _, e := u.read(ctx, 10*time.Millisecond); e != nil {
			break
		}
	}
	if err = r.setting(ctx, 0x2a, []byte{1}); err != nil {
		return nil, err
	}
	if _, err = r.command(ctx, 0x20, nil); err != nil {
		return nil, err
	}
	if _, err = r.command(ctx, 0x22, nil); err != nil {
		return nil, err
	}
	if err = r.setting(ctx, 6, []byte{0}); err != nil {
		return nil, err
	}
	for _, s := range []struct {
		c byte
		d []byte
	}{{0, hx("02030f03")}, {2, defaults}, {2, hx("00060100020005010707")}} {
		if err = r.setting(ctx, s.c, s.d); err != nil {
			return nil, err
		}
	}
	atqa, err := r.rf(ctx, []byte{0x26}, 30*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("detect NFC tag: %w", err)
	}
	if len(atqa) != 2 {
		return nil, errors.New("invalid ATQA")
	}
	if err = r.setting(ctx, 2, hx("04010708")); err != nil {
		return nil, err
	}
	var uid []byte
	var sak []byte
	for _, sel := range []byte{0x93, 0x95, 0x97} {
		if err = r.setting(ctx, 2, hx("01000200")); err != nil {
			return nil, err
		}
		var serial []byte
		serial, err = r.rf(ctx, []byte{sel, 0x20}, 30*time.Millisecond)
		if err != nil {
			return nil, err
		}
		if len(serial) != 5 || serial[0]^serial[1]^serial[2]^serial[3] != serial[4] {
			return nil, errors.New("invalid anticollision response")
		}
		if err = r.setting(ctx, 2, hx("01010201")); err != nil {
			return nil, err
		}
		sak, err = r.rf(ctx, append([]byte{sel, 0x70}, serial...), 30*time.Millisecond)
		if err != nil {
			return nil, err
		}
		if len(sak) != 1 {
			return nil, errors.New("invalid SAK")
		}
		if sak[0]&4 != 0 {
			uid = append(uid, serial[1:4]...)
		} else {
			uid = append(uid, serial[:4]...)
			break
		}
	}
	if len(sak) != 1 || sak[0]&0x24 != 0x20 {
		return nil, errors.New("tag is not ISO-DEP")
	}
	r.UID = hex.EncodeToString(uid)
	ats, err := r.exchange(ctx, []byte{0xe0, 0x80}, 30*time.Millisecond)
	if err != nil {
		return nil, err
	}
	if len(ats) < 2 || int(ats[0]) != len(ats) || ats[1]&15 > 8 {
		return nil, errors.New("invalid ATS")
	}
	fsc := []int{16, 24, 32, 40, 48, 64, 96, 128, 256}[ats[1]&15]
	pos := 2
	fwi := byte(4)
	if ats[1]&0x10 != 0 {
		pos++
	}
	if ats[1]&0x20 != 0 {
		if pos >= len(ats) {
			return nil, errors.New("truncated ATS")
		}
		fwi = ats[pos] >> 4
	}
	if fwi > 14 {
		return nil, errors.New("invalid ATS FWI")
	}
	r.dep = isoDep{miu: fsc - 3, fwt: time.Duration(float64(time.Second) * 4096 / 13.56e6 * float64(uint64(1)<<fwi)), exchange: r.exchange}
	return r, nil
}
func (r *Reader) Transceive(ctx context.Context, b []byte) ([]byte, error) {
	return r.dep.transceive(ctx, b)
}
func (r *Reader) Close() error {
	if r.usb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		_ = r.usb.write(ctx, ack)
		_ = r.setting(ctx, 6, []byte{0})
		r.usb.close()
		r.usb = nil
	}
	return nil
}
