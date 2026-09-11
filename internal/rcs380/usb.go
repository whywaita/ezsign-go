//go:build cgo && (darwin || linux)

package rcs380

/*
#cgo darwin CFLAGS: -I/opt/homebrew/include/libusb-1.0 -I/usr/local/include/libusb-1.0
#cgo darwin LDFLAGS: -L/opt/homebrew/lib -L/usr/local/lib -lusb-1.0
#cgo linux CFLAGS: -I/usr/include/libusb-1.0
#cgo linux LDFLAGS: -lusb-1.0
#include <libusb.h>
*/
import "C"
import (
	"context"
	"errors"
	"fmt"
	"time"
	"unsafe"
)

type usb struct {
	ctx    *C.libusb_context
	handle *C.libusb_device_handle
}

func openUSB() (*usb, error) {
	u := &usb{}
	if code := C.libusb_init(&u.ctx); code != 0 {
		return nil, fmt.Errorf("libusb init: %d", code)
	}
	u.handle = C.libusb_open_device_with_vid_pid(u.ctx, 0x054c, 0x06c1)
	if u.handle == nil {
		C.libusb_exit(u.ctx)
		return nil, errors.New("RC-S380/S (054c:06c1) not found or USB access denied")
	}
	if code := C.libusb_claim_interface(u.handle, 0); code != 0 {
		C.libusb_close(u.handle)
		C.libusb_exit(u.ctx)
		return nil, fmt.Errorf("claim RC-S380 interface: %d", code)
	}
	return u, nil
}
func (u *usb) close() {
	if u.handle != nil {
		C.libusb_release_interface(u.handle, 0)
		C.libusb_close(u.handle)
		C.libusb_exit(u.ctx)
		u.handle = nil
	}
}
func (u *usb) bulk(endpoint byte, data []byte, ms uint) (int, error) {
	var n C.int
	var ptr *C.uchar
	if len(data) > 0 {
		ptr = (*C.uchar)(unsafe.Pointer(&data[0]))
	}
	code := C.libusb_bulk_transfer(u.handle, C.uchar(endpoint), ptr, C.int(len(data)), &n, C.uint(ms))
	if code != 0 {
		return int(n), fmt.Errorf("USB transfer: %s (%d)", C.GoString(C.libusb_error_name(code)), code)
	}
	return int(n), nil
}
func (u *usb) write(ctx context.Context, b []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	n, err := u.bulk(0x02, b, 1000)
	if err != nil {
		return err
	}
	if n != len(b) {
		return errors.New("short USB write")
	}
	if len(b)%64 == 0 {
		_, err = u.bulk(0x02, nil, 1000)
	}
	return err
}
func (u *usb) read(ctx context.Context, timeout time.Duration) ([]byte, error) {
	until := time.Now().Add(timeout)
	b := make([]byte, 300)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		remaining := time.Until(until)
		if remaining <= 0 {
			return nil, context.DeadlineExceeded
		}
		ms := uint(max(1, min(100, remaining.Milliseconds())))
		var n C.int
		code := C.libusb_bulk_transfer(u.handle, 0x81, (*C.uchar)(unsafe.Pointer(&b[0])), C.int(len(b)), &n, C.uint(ms))
		if code == C.LIBUSB_ERROR_TIMEOUT && n == 0 {
			continue
		}
		if code != 0 {
			return nil, fmt.Errorf("USB read: %s", C.GoString(C.libusb_error_name(code)))
		}
		if n == 0 {
			return nil, errors.New("empty USB read")
		}
		return b[:int(n)], nil
	}
}
