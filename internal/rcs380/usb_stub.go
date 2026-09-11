//go:build !cgo || (!darwin && !linux)

package rcs380

import (
	"context"
	"errors"
	"time"
)

type usb struct{}

func openUSB() (*usb, error) {
	return nil, errors.New("RC-S380 requires CGO and libusb on macOS or Linux")
}
func (*usb) close()                              {}
func (*usb) write(context.Context, []byte) error { return errors.New("USB unavailable") }
func (*usb) read(context.Context, time.Duration) ([]byte, error) {
	return nil, errors.New("USB unavailable")
}
