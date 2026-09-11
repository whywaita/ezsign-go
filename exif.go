package ezsign

import (
	"bytes"
	"encoding/binary"
	"image"
)

func jpegOrientation(b []byte) int {
	if len(b) < 2 || b[0] != 255 || b[1] != 216 {
		return 1
	}
	for p := 2; p+4 <= len(b); {
		if b[p] != 255 {
			return 1
		}
		marker := b[p+1]
		if marker == 0xda || marker == 0xd9 {
			return 1
		}
		size := int(binary.BigEndian.Uint16(b[p+2:]))
		if size < 2 || size > len(b)-p-2 {
			return 1
		}
		data := b[p+4 : p+2+size]
		p += 2 + size
		if marker != 0xe1 || len(data) < 14 || !bytes.Equal(data[:6], []byte("Exif\x00\x00")) {
			continue
		}
		t := data[6:]
		var order binary.ByteOrder
		switch string(t[:2]) {
		case "II":
			order = binary.LittleEndian
		case "MM":
			order = binary.BigEndian
		default:
			return 1
		}
		if order.Uint16(t[2:]) != 42 {
			return 1
		}
		off := int(order.Uint32(t[4:]))
		if off < 0 || off > len(t)-2 {
			return 1
		}
		count := int(order.Uint16(t[off:]))
		for i := 0; i < count; i++ {
			pos := off + 2 + i*12
			if pos > len(t)-12 {
				return 1
			}
			entry := t[pos : pos+12]
			if order.Uint16(entry) == 0x112 && order.Uint16(entry[2:]) == 3 && order.Uint32(entry[4:]) == 1 {
				v := int(order.Uint16(entry[8:]))
				if v >= 1 && v <= 8 {
					return v
				}
				return 1
			}
		}
	}
	return 1
}
func orientImage(src image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return src
	}
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	out := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			sx, sy := x, y
			switch o {
			case 2:
				sx = w - 1 - x
			case 3:
				sx, sy = w-1-x, h-1-y
			case 4:
				sy = h - 1 - y
			case 5:
				sx, sy = y, x
			case 6:
				sx, sy = y, h-1-x
			case 7:
				sx, sy = w-1-y, h-1-x
			case 8:
				sx, sy = w-1-y, x
			}
			out.Set(x, y, src.At(bounds.Min.X+sx, bounds.Min.Y+sy))
		}
	}
	return out
}
