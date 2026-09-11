# HTTP API

Start the server:

```sh
bin/ezsign-go-api -listen 127.0.0.1:8080
```

Send the image file directly in the request body.
For JPEG files, use `Content-Type: image/jpeg`.

```sh
curl --fail-with-body --max-time 100 \
  -H 'Content-Type: image/png' \
  --data-binary @image.png \
  'http://127.0.0.1:8080/v1/image'
```

| Endpoint | Behavior |
| --- | --- |
| `GET /healthz` | Report whether the HTTP process is alive |
| `POST /v1/image` | Wait for the write to complete and return a JSON result |

`POST /v1/image` accepts the query parameters `rotation=0` or `180` (default `0`) and `dither=true` or `false` (default `true`).
Input is limited to 10 MiB and 20 million pixels.
Requests made while an update is in progress receive HTTP 409.

Successful responses include the image dimensions, the SHA-256 of the pixel data, and the processing time.
The status `protocol_complete_visual_check_required` indicates protocol completion; it does not verify that the physical display matches the image.

The API has no authentication and listens on localhost by default.
For access from other hosts, configure authentication and TLS through a reverse proxy or equivalent.
