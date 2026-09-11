package ezsign

import (
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

func TestEXIFOrientation(t *testing.T) {
	tiff := make([]byte, 26)
	copy(tiff, "II")
	binary.LittleEndian.PutUint16(tiff[2:], 42)
	binary.LittleEndian.PutUint32(tiff[4:], 8)
	binary.LittleEndian.PutUint16(tiff[8:], 1)
	binary.LittleEndian.PutUint16(tiff[10:], 0x112)
	binary.LittleEndian.PutUint16(tiff[12:], 3)
	binary.LittleEndian.PutUint32(tiff[14:], 1)
	binary.LittleEndian.PutUint16(tiff[18:], 6)
	payload := append([]byte("Exif\x00\x00"), tiff...)
	jpeg := append([]byte{255, 216, 255, 225, 0, byte(len(payload) + 2)}, payload...)
	jpeg = append(jpeg, 255, 217)
	if n := jpegOrientation(jpeg); n != 6 {
		t.Fatal(n)
	}
	for i := 0; i < len(jpeg)-2; i++ {
		_ = jpegOrientation(jpeg[:i])
	}
}
func TestOrientImage(t *testing.T) {
	im := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	im.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 255})
	for _, tc := range []struct{ o, x, y, w, h int }{{1, 0, 0, 2, 3}, {2, 1, 0, 2, 3}, {3, 1, 2, 2, 3}, {4, 0, 2, 2, 3}, {5, 0, 0, 3, 2}, {6, 2, 0, 3, 2}, {7, 2, 1, 3, 2}, {8, 0, 1, 3, 2}} {
		out := orientImage(im, tc.o)
		r, _, _, _ := out.At(tc.x, tc.y).RGBA()
		if r != 65535 || out.Bounds().Dx() != tc.w || out.Bounds().Dy() != tc.h {
			t.Fatal(tc)
		}
	}
}
