# ezsign-go

A Go library for writing images to EZ Sign displays, with a command-line tool, a slideshow program, and an HTTP API.

Supports the RC-S380/S reader and the 4.2-inch, four-color EZ Sign display (400 × 300 pixels).
PNG and JPEG images are resized and converted to the display's four-color palette.

## Installation

Requires Go 1.27.1 or later, a C compiler, and libusb development files.
Install libusb before building.

macOS:

```sh
brew install libusb
```

Ubuntu:

```sh
sudo apt update
sudo apt install build-essential libusb-1.0-0-dev
```

Run from the repository root:

```sh
CGO_ENABLED=1 make build
```

Binaries are written to `bin/`.
libusb is also required at runtime.
See [deployment](docs/deploy.md) for Linux setup and USB driver conflicts.

## Usage

Connect the RC-S380 over USB and place the EZ Sign display on the reader.
Run only one program against the same reader at a time.

### Write an image

```sh
bin/ezsign -write image.png
```

Omit `-write` to generate a preview without updating the display.
To turn the image upside down, add `-rotate 180` before the image path.

### Run a slideshow

```sh
bin/ezsign-go-slideshow -dir ./images -interval 60s
```

Writes PNG and JPEG files in filename order, repeating from the beginning after the last image.
`-interval` sets the delay between completing one write and starting the next.
Press Ctrl+C to stop.

### Use the HTTP API

```sh
bin/ezsign-go-api -listen 127.0.0.1:8080
```

Send an image from another terminal:

```sh
curl --fail-with-body --max-time 100 \
  -H 'Content-Type: image/png' \
  --data-binary @image.png \
  http://127.0.0.1:8080/v1/image
```

The API has no authentication.
Configure authentication and TLS through a reverse proxy or equivalent before exposing it externally.

### Use the Go library

```go
package main

import (
	"context"
	"log"
	"os"

	ezsign "github.com/whywaita/ezsign-go"
)

func main() {
	client, err := ezsign.New(ezsign.Config{})
	if err != nil {
		log.Fatal(err)
	}
	data, err := os.ReadFile("image.png")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := client.Write(context.Background(), data, ezsign.WriteOptions{}); err != nil {
		log.Fatal(err)
	}
}
```

`WriteImage` accepts an `image.Image`.
`Render` generates a preview and transfer data without connecting to a reader.

## Documentation

- [CLI options and output files](docs/cli.md)
- [Slideshow configuration](cmd/slideshow/README.md)
- [HTTP API](docs/http-api.md)
- [Linux deployment and troubleshooting](docs/deploy.md)
- [EZ Sign protocol](docs/protocol.md)

## Development

```sh
make fmt    # Format Go code
make check  # Run tests with the race detector and go vet
make build  # Build binaries
```

Report bugs through [Issues](https://github.com/whywaita/ezsign-go/issues).
Include your OS, device model, command, and error message.

## License

[MIT License](LICENSE)
