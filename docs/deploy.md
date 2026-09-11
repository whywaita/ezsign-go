# Linux deployment

## Runtime requirements

On Ubuntu running on a Raspberry Pi (aarch64, kernel `6.8.0-1064-raspi`), the `port100` driver claimed the RC-S380 USB interface and prevented this program from using it.
The user confirmed that unloading `port100` resolved the conflict.

Deploy binaries built for Linux arm64 with CGO enabled.
libusb is required at runtime:

```sh
sudo apt update
sudo apt install libusb-1.0-0
```

## RC-S380 interface is busy

This error means that the USB interface is already in use.
Running with `sudo` does not resolve a driver conflict.

```text
claim RC-S380 interface: -6
```

Stop the slideshow with Ctrl+C, then check the driver and competing processes:

```sh
lsusb -t
pgrep -af 'pcscd|ezsign'
```

If the RC-S380 entry shows `Driver=port100`, temporarily unload the kernel module:

```sh
sudo modprobe -r port100
```

This also affects other devices using the same module.
Stop any other ezsign processes using the reader.
If `pcscd` is using the device, stop `pcscd.socket` and `pcscd.service` as needed.
In the reported environment, there were no competing processes; `port100` was the cause.

## Run the slideshow

This example assumes the binary is in the current directory and images are in `images/`.
The HTTP API does not need to be running.

```sh
sudo ./ezsign-go-slideshow -dir ./images -interval 10s
```

`-interval` is the delay after one write finishes and before the next starts.
Press Ctrl+C to stop.

## Restore the driver

After stopping the slideshow, reload the module:

```sh
sudo modprobe port100
```

`modprobe -r` does not permanently disable the module.
If the conflict returns after a reboot or reconnecting the reader, check `lsusb -t` again.
