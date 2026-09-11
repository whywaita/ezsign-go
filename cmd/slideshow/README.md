# Slideshow

Run `make build` to generate `bin/ezsign-go-slideshow`.
The program embeds the Go library and writes directly to the RC-S380 over USB.
No HTTP server or network connection is required.
libusb is required at runtime.
Do not run another CLI or API process against the same reader at the same time.

```sh
bin/ezsign-go-slideshow -dir /path/to/images -interval 60s
```

The program repeatedly writes `.png`, `.jpg`, and `.jpeg` files from the specified directory in filename order.
Extensions are case-insensitive.
Subdirectories and symbolic links are excluded.
The file list is read at startup; restart the program after adding files.

The first write starts immediately.
`-interval` is the delay between completing one write and starting the next (default: 60 seconds).
For example, if writing takes about 27 seconds, the default produces a start-to-start interval of about 87 seconds.
Failures are logged, and the program advances to the next image after the configured delay.
Ctrl+C cancels the wait or write in progress.

| Flag | Default | Description |
| --- | --- | --- |
| `-dir` | Required | Image directory |
| `-interval` | `60s` | Delay after each write |
| `-timeout` | `100s` | Timeout for each write |
| `-rotate` | `0` | Rotation relative to the default display orientation: 0 or 180 degrees |
| `-no-dither` | `false` | Disable dithering |

```sh
bin/ezsign-go-slideshow -dir /path/to/images -interval 2m -rotate 180 -no-dither
```
