# Image writer CLI

Connect the RC-S380 over USB and place the EZ Sign display on it.
Keep the display in place until the write finishes.
Use only one process per reader.

```sh
bin/ezsign -write image.png
```

Omit `-write` to generate a preview without updating the display:

```sh
bin/ezsign -output output/preview image.png
```

The output directory contains `preview.png` and the transfer data in `frame.bin`.
Device writes also produce a `session.json` communication log.
Place flags before the image path.

| Flag | Default | Description |
| --- | --- | --- |
| `-write` | `false` | Write to the display |
| `-output` | `output` | Output directory |
| `-rotate` | `0` | Rotation relative to the default orientation: `0` or `180` |
| `-no-dither` | `false` | Disable the dithering used to approximate intermediate colors |
| `-timeout` | `90s` | Operation timeout |

Use `-rotate 180` if the image appears upside down.
