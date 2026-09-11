# EZ Sign NFC protocol research

Research date: 2026-09-10.
Communication logs, camera images, and previews are not included in this repository.
Hardware results in this document are based on locally retained evidence.
Test patterns and whywaita's X profile icon were written to a 4.2-inch four-color display from a Mac through an RC-S380/S, with visual confirmation through Desk View.

## Scope and evidence labels

This document follows the [research guide](research/ai_agent_ez_sign_protocol_guide.md) and distinguishes local hardware tests from public USB captures.
**OBSERVED** means directly observed in a source or log; **REPRODUCED** means reproduced in an independent execution; **INFERRED** means an interpretation; **UNKNOWN** means unverified; **CONFLICTING** means sources or conditions disagree.
An OBSERVED result from a public capture does not establish successful display output on the local device.

| Subject | Scope | Evidence |
| --- | --- | --- |
| Local 4.2-inch four-color display | Model identified by the user; detection, ISO-DEP connection, selection, and information retrieval performed twice with RC-S380/S | REPRODUCED: LOCAL-42-001, LOCAL-42-002 |
| Official app | Runs on the user's iPhone; app and iOS versions were not recorded | UNKNOWN: record versions on the next update |
| Updates from iPhone | The user confirmed placing the display on the RC-S380; a successful white update, duration, and screen result were not recorded | UNKNOWN: check app completion and physical output |
| Public 2.9-inch two-color captures | Requests, responses, compressed data, and accompanying text compared across eight captures | OBSERVED: PUB-*, SRC-README |
| Local four-color updates | Four-color corner pattern and profile icon transferred and visually checked; repeated updates without removing the display | REPRODUCED: LOCAL-42-PATTERN, LOCAL-42-ICON |

Checking communication responses is not cryptographic product authentication.
No certificate- or signature-based product authentication was performed.

## Current Go implementation

In the current API, rotation 0 (the default) produces the same orientation as rotation 180 in the earlier implementation, matching the user's display placement.
Saved sessions retain the `rotation` value used at the time of testing.
The rendering reference orientation changed by 180 degrees; pixel transfer order did not change.

Image conversion, JPEG EXIF orientation, four-color quantization, LZO1X compression, APDUs, RC-S380 control, NFC-A discovery, ISO-DEP chaining, and waiting-time extension are implemented in Go.
The runtime does not start Python or load Python packages.
USB access uses libusb through CGO.
See the [README](../README.md) for usage, [protocol.go](../protocol.go) for transfers, and [internal/rcs380](../internal/rcs380/) for reader control.

REPRODUCED: the Go CLI sent the rotated profile icon and received update completion in 29,719 ms.
The complete APDU log and preview were retained locally and are not distributed.
This initial Go test used literal-only LZO1X streams; back-reference compression was added afterward.
An independent lzokay decoder successfully reconstructed 1,000 random, four-value, periodic, and solid-color inputs produced by the Go compressor.
lzokay was used only for development-time comparison.

REPRODUCED: curl sent the same image through the Go HTTP API with compression enabled, receiving HTTP 200 and update completion in 27,252 ms.
The pixel-data SHA-256 in the locally retained HTTP result matched the CLI output.
UNKNOWN: the camera frame taken after the Go update did not contain the display, so its latest physical output was not visually checked.
The physical-display photographs discussed below belong to the earlier Python implementation.
The Go renderer uses bilinear interpolation and does not guarantee pixel-for-pixel equality with the earlier Pillow renderer.

## Successful four-color session with the initial implementation

REPRODUCED: an initial Python CLI, developed from analysis of official distributions, wrote a four-color test pattern and the profile icon to the local device.
That implementation was removed after the Go port was completed.
Current usage is in the [README](../README.md); the implementation is in [protocol.go](../protocol.go) and [render.go](../render.go).
LOCAL-42-PATTERN and LOCAL-42-ICON contain complete request/response logs retained locally, not distributed with the repository.

| Step | Request | Local response and handling |
| --- | --- | --- |
| 1 | `00 20 00 01 04 20 09 12 10` | `90 00`; VERIFY before writing |
| 2 | `00 A4 04 00 07 D2 76 00 00 85 01 01` | `90 00` |
| 3 | `F0 D8 01 FE 05 00 00 00 00 00` | `6A 86`; continue as the official app does |
| 4 | `00 D1 00 00 00` | Device information followed by `90 00` |
| 5 | `F0 D8 00 00 05 00 00 00 00 0E` | ASCII `4_color Screen` followed by `90 00` |
| 6 | `F0 D3 00 end Lc block fragment payload` | `90 00` for every fragment |
| 7 | `F0 D4 05 80 00` | `68 C6` after about 0.67 seconds |
| 8 | `F0 D4 85 00 00` | `90 00` after about 24.93 seconds; physical output checked afterward |

OBSERVED: for `68 C6` and `69 86`, the official Android app's `_handle6986Response` (`0x7f1648`) sends `F0 D4 85 00 00`.
Implementing this behavior completed updates on the local unit.
The successful path did not use DE polling.
`F0 D4 85 80 00` and `F0 D4 85 00 00` are distinct commands.
With nfcpy, even `transceive(timeout=3.0)` received a response after about 25 seconds because of ISO-DEP waiting-time extensions.
The three-second argument is not a deadline for the entire call.

REPRODUCED: the image is 400 × 300 pixels, two bits per pixel, totaling 30,000 bytes without extra padding.
Color codes are black=0, white=1, yellow=2, and red=3; the leftmost pixel occupies the most significant two bits.
Rows are transferred from bottom to top.
Using the original Desk View orientation with a top-left coordinate origin, pixel `(x,y)` maps to `offset=(299-y)*100+x//4` and bit shift `6-2*(x%4)`.
Each 2,000-byte uncompressed block is independently LZO1X-compressed, giving 15 blocks.
The test pattern required 16 APDUs and the dithered icon required 40 APDUs.
The sum of APDU response times for the icon was about 27.05 seconds.

The locally retained test photograph showed black “1 TL” at top left, white “2 TR” at top right, yellow “3 BL” at bottom left, and red “4 BR” at bottom right.
The icon photograph was compared with its preview for face orientation, the “why” lettering on the right, and side margins.
The first icon matched the camera orientation, but the user reported that it was upside down from their position.
It was resent with the earlier CLI's `--rotate 180` option, with the final log, preview, and photograph retained locally.
Camera-relative coordinates and the user's physical viewing direction must be distinguished.
Visual checking used Desk View photographs, not pixel-level color measurement or a quantified image-match score.
The source image's blue cannot be reproduced on the four-color display and was quantized and dithered.
The icon source and SHA-256 were recorded in a local image inventory that is not distributed.

## Local connection tests

OBSERVED: macOS 26.4.1 (25E253), arm64, nfcpy 1.0.4, and libusb 1.0.30.
`ioreg` reported SONY RC-S380/S with USB VID:PID `054C:06C1`.
Although `system_profiler SPSmartCardsDataType` listed no readers, nfcpy could open the reader in an environment with USB access.
libusb enumeration was empty in a restricted environment; that failure was not treated as evidence of an unsupported reader.

REPRODUCED: after the user placed the display on the reader, two runs that closed and reopened the reader produced matching results.
These initial runs did not send image-write commands.

| Item | Measured value | Evidence |
| --- | --- | --- |
| Detection | `106A`, `sens_res=0400`, `sel_res=28` | LOCAL-42-001, 002 |
| NFCID1 | `4E 80 D6 1D` | LOCAL-42-001, 002 |
| Connection | `Type4ATag`, MIU=253, FWT=0.309314 seconds | LOCAL-42-001, 002 |
| SELECT response | `90 00` | LOCAL-42-001, 002 |
| Information response | 54 data bytes followed by `90 00` | LOCAL-42-001, 002 |

The following requests were passed to `Type4Tag.transceive` in order, with a timeout argument of 1.0 second.
In the first run, SELECT took about 37 ms and information retrieval about 12 ms.
Neither these measurements nor the argument specifies a display-update timeout.

```text
00 A4 04 00 07 D2 76 00 00 85 01 01
00 D1 00 00 00
```

Complete information response:

```text
A0 07 F0 07 20 02 58 01 90
A1 07 01 12 00 30 FF FF FF
B1 01 08
B2 01 14
B3 01 00
C0 0A 53 45 41 42 30 34 38 36 39 30
C1 04 4E 80 D6 1D
D1 07 01 20 00 00 00 00 00
90 00
```

## Communication layers

The local column below describes the initial information-retrieval tests; later image-update results are documented above.

| Layer | Public capture path | Initial local path | Confidence |
| --- | --- | --- | --- |
| Host API | Public writer uses pyscard / PC/SC | nfcpy `Type4Tag.transceive` | OBSERVED: SRC-WRITER, LOCAL-42-* |
| USB | CCID requests and responses inside USBPcap | Direct USB through nfcpy's RC-S380 driver | OBSERVED: PUB-*, SRC-NFCPY, LOCAL-42-* |
| Radio | USB logs do not directly capture RF frames | 106 kbps Type A discovery and Type4ATag connection | REPRODUCED: LOCAL-42-* |
| Device commands | `00 A4`, `00 D1`, `F0 D3/D4/D8/DE` | Initially measured only `00 A4` and `00 D1` | OBSERVED: PUB-*, LOCAL-42-* |

INFERRED: `FF 5C` and `FF 5D` are candidate reader-control commands.
The public writer classifies them that way, but USB logs alone do not establish whether they reach the display over RF.
There is no evidence supporting their direct inclusion in RC-S380 or iPhone device-command sequences.
REPRODUCED: D3 was accepted locally when preceded by `00 20 00 01 04 20 09 12 10`.
Without that command, D8 and D3 returned `69 85`.
The same command appears in the official Android app's `_PreviewPageState._checkDevice` (`0x7fbfec`).

OBSERVED: Sony's download page lists no macOS driver support for the RC-S380.
That is distinct from the working direct-USB path through nfcpy (SRC-SONY).
Passive interception of iPhone-to-display traffic with an RC-S380 was not established.
The local logs contain Mac requests made after the iPhone was moved away, not iPhone transmissions.

## Reconstructing public captures

OBSERVED: analysis was pinned to public repository commit `4a6200ef7420d42ef3274909d9bf973edcdc6458`.
Eight `sample_log/*.pcapng` files were read with Scapy's PcapNgReader, numbering packets from 1.
USBPcap BULK transfers, direction, outgoing/completing IRPs, and actual payload lengths were checked to avoid counting empty completion notifications as duplicate APDUs.

OBSERVED: each analyzed BULK payload exactly matched a 10-byte CCID header plus `dwLength` data bytes.
OUT `0x6F` messages were extracted as requests and matched to IN responses using slot, sequence, and time order.
No unmatched requests remained in any of the eight captures.
Extracted `F0 D3` commands matched the accompanying UTF-16 text byte for byte.
This validates those captures; it does not establish a general-purpose parser for arbitrary fragmented USB transfers.

CCID response type `0x80` alone does not identify a card APDU response.
For example, a `0x80` response to a power-on request contains an ATR, whose final two bytes must not be interpreted as a status word (PUB-W, SRC-CCID).

## Public 2.9-inch two-color session

This is the observed PUB-W sequence, not a recommended sequence for the four-color display.
Packet pairs match the same CCID sequence number.
Full bytes can be recovered from the raw capture.

| Packets | Request or operation | Response | Classification |
| --- | --- | --- | --- |
| 19→22 | CCID `0x63` IccPowerOff | SlotStatus | OBSERVED |
| 23→26 | CCID `0x62` IccPowerOn | ATR `3B 86 80 01 90 72 3C 50 52 03 D8` | OBSERVED |
| 27→30 | CCID `0x6C` GetParameters | `11 10 01 4D 00 FE 00` | OBSERVED |
| 31→34 | `00 20 00 01 04 20 09 12 10` | `90 00` | OBSERVED; role UNKNOWN |
| 35→38 | `FF 5C 00 00 03 00 01 01` | `90 00` | OBSERVED; reader control INFERRED |
| 39→42 | `FF 5D 00 00 01 00` | `01 01 90 00` | OBSERVED; reader control INFERRED |
| 43→46 | `00 A4 04 00 07 D2 76 00 00 85 01 01` | `90 00` | OBSERVED; NDEF selection |
| 47→50 | `F0 D8 01 FE 05 00 00 00 00 00` | `6A 86` | OBSERVED |
| 51→54 | `00 D1 00 00 00` | Information followed by `90 00` | OBSERVED |
| 55→58 | `F0 D8 00 00 05 00 00 00 00 0E` | Fourteen `FF` bytes followed by `90 00` | OBSERVED; role UNKNOWN |
| 59→62, 63→66, 67→70 | Three white-image `F0 D3` requests | `90 00` each | OBSERVED |
| 71→74 | `F0 D4 85 80 00` | `90 00` | OBSERVED; refresh initiation INFERRED |
| 75+4k→78+4k, k=0…34 | `F0 DE 00 00 01` | `01 90 00` | OBSERVED; busy INFERRED |
| 215→218 | `F0 DE 00 00 01` | `00 90 00` | OBSERVED; completion INFERRED |
| 219→222 | CCID IccPowerOn | Same ATR | OBSERVED |
| 223→226 | CCID IccPowerOff | SlotStatus | OBSERVED |

OBSERVED: in PUB-W, the `F0 D4` response arrived after about 0.658 seconds, and the final `00 90 00` arrived 4.362134 seconds after the D4 request.
There were 36 `F0 DE` polls, typically about 0.102 seconds from one response to the next request.
About 10 seconds elapsed between the final IccPowerOn response and IccPowerOff request.
USB logs alone do not establish RF power throughout that period or when the image became stable.

CONFLICTING: SRC-WRITER implements at most 30 polls at 0.5-second intervals, unlike the capture.
It warns and continues on non-`9000` responses, and prints `Done.` even after the polling limit.
That message must not be used as evidence of a successful update.

## Commands and fields

Offsets are zero-based from the start of the request APDU; byte values are hexadecimal.
Confidence in the fixed values below is high within the cited evidence.
Proprietary command meanings and cross-model compatibility require separate evaluation.

### SELECT and information retrieval

| Command | Offset:length | Encoding / value | Meaning and confidence | Evidence |
| --- | --- | --- | --- | --- |
| SELECT | 0:4 | bytes `00 A4 04 00` | Header; OBSERVED, high | LOCAL-42-*, PUB-W |
| SELECT | 4:1 | uint8 `07` | Seven data bytes; OBSERVED, high | Same |
| SELECT | 5:7 | bytes `D2 76 00 00 85 01 01` | NDEF AID; OBSERVED, high | Same, SRC-TYPE4 |
| Information | 0:4 | bytes `00 D1 00 00` | Request returning information; REPRODUCED, high | LOCAL-42-* |
| Information | 4:1 | byte `00` | Could be short APDU Le=256; INFERRED, medium | LOCAL-42-*, PUB-W |

OBSERVED: the entire information payload parses as one-byte tags, one-byte lengths, and values of that length.
The offsets below locate values relative to the start of the response data, excluding the status word.
A general specification for automatic model and color-count detection was not established.

| Tag | Value offset:length | Two-color PUB-W | Local four-color unit | Meaning and confidence |
| --- | --- | --- | --- | --- |
| A0 | 2:7 | `F0 01 20 00 80 01 28` | `F0 07 20 02 58 01 90` | Raw values OBSERVED, high; model information INFERRED |
| A1 | 11:7 | `00 12 00 30 FF FF FF` | `01 12 00 30 FF FF FF` | Raw values OBSERVED, high; first byte is scanType in the Android implementation; remainder unresolved |
| B1 | 20:1 | `2E` | `08` | Raw values OBSERVED, high; meaning UNKNOWN |
| B2 | 23:1 | `14` | `14` | Raw values OBSERVED, high; meaning UNKNOWN |
| B3 | 26:1 | `00` | `00` | Raw values OBSERVED, high; meaning UNKNOWN |
| C0 | 29:10 | ASCII `SEAA000265` | ASCII `SEAB048690` | Strings OBSERVED, high; serial number INFERRED |
| C1 | 41:4 | `72 3C 50 52` | `4E 80 D6 1D` | Matches local NFCID1; REPRODUCED, high |
| D1 | 47:7 | `01 20 00 00 00 00 00` | Same | Raw values OBSERVED, high; nonzero first byte enables compression in Android; remainder unresolved |

INFERRED: interpreting A0's final four bytes as two big-endian 16-bit integers gives 128 and 296 for the two-color example, and 600 and 400 for the four-color unit.
The former matches the published two-color pixel dimensions.
OBSERVED: the official Android implementation divides 600 by two and stores width=400, height=300 (STATIC-APK-127), interpreting this response as a four-color screen.
The manufacturer's definition of the doubled dimension remains unresolved.
At this stage of the investigation, coordinate mapping also remained open; the later four-color tests above establish the local mapping and data length.

### F0 D8, F0 D4, and F0 DE

This table applies only to PUB-W.
For local four-color results, see the successful session above.

| Command | Offset:length | Encoding / value | Interpretation and confidence |
| --- | --- | --- | --- |
| D8, first | 0:2, 2:2, 4:1, 5:5 | bytes `F0 D8`, `01 FE`, uint8 `05`, `00 00 00 00 00` | Header and length OBSERVED, high; function UNKNOWN |
| D8, second | 0:2, 2:2, 4:1, 5:5 | bytes `F0 D8`, `00 00`, uint8 `05`, `00 00 00 00 0E` | Raw values OBSERVED, high; relationship between final 0E and 14 response bytes INFERRED |
| D4 | 0:2, 2:2, 4:1 | bytes `F0 D4`, `85 80`, `00` | Refresh initiation INFERRED, medium; P1/P2 meanings UNKNOWN |
| DE | 0:2, 2:2, 4:1 | bytes `F0 DE`, `00 00`, `01` | Status query INFERRED, medium; final 01 may be Le=1 |

`F0 DE 00 00 01` is five bytes long; do not treat its final `01` as Lc=1 with missing data.
A parser cannot interpret every proprietary request's fifth byte as Lc.
Comparing `F0 D8` response ranges on the same model is a proposed next test for its configuration semantics.

### F0 D3 transfer format

OBSERVED: this format reconstructs all image data in the eight public captures.
Block and fragment numbering are INFERRED from successful LZO1X decompression of concatenated payloads.

| Offset | Length | Encoding / observed value | Meaning and confidence | Evidence |
| --- | --- | --- | --- | --- |
| 0 | 1 | byte `F0` | CLA; OBSERVED, high | PUB-* |
| 1 | 1 | byte `D3` | INS; OBSERVED, high | PUB-* |
| 2 | 1 | byte `00` | P1; meaning UNKNOWN | PUB-* |
| 3 | 1 | uint8 `00` / `01` | P2; nonfinal/final fragment of compressed block INFERRED, high | PUB-* |
| 4 | 1 | uint8, observed maximum `FC` | Lc, always APDU length minus 5; OBSERVED, high | PUB-* |
| 5 | 1 | uint8 `00` / `01` / `02` | Uncompressed block index INFERRED, high | PUB-* |
| 6 | 1 | uint8 `00`…`03` | Fragment index within block INFERRED, high | PUB-* |
| 7 | Lc−2 | bytes, at most 250 | LZO1X stream fragment; OBSERVED, high | PUB-*, DEC-001 |

OBSERVED: a final fragment can contain 250 compressed bytes (PUB-C, packet 83), so length alone does not identify the final fragment.
P2=`01` marks the end of each compressed block, not just the final block of the image.
Index limits, wraparound, retransmission, and missing-fragment behavior remain UNKNOWN and require further testing on larger inputs.

## Image compression and layout

OBSERVED: bytes from offset 7 of each `F0 D3` request were concatenated in transmission order by the index at offset 5, then decompressed with LZO 2.10 `lzo1x_decompress_safe`.
All 24 blocks across eight images returned zero, with actual output lengths of 2,000, 2,000, and 736 bytes per image (DEC-001).
The returned output length, rather than allocated buffer capacity, was used.
Each complete image contained 4,736 bytes, matching `296 × 128 ÷ 8`.

| Capture | D3 requests | Compressed block lengths | Decompressed result or property |
| --- | --- | --- | --- |
| PUB-W | 3 | 20 / 20 / 15 | All 4,736 bytes are `FF` |
| PUB-B | 3 | 20 / 20 / 15 | All 4,736 bytes are `00` |
| PUB-HS | 3 | 20 / 20 / 15 | All 4,736 bytes are `55` |
| PUB-VS | 3 | 41 / 41 / 36 | Alternating runs of sixteen `00` and sixteen `FF` bytes |
| PUB-C | 7 | 793 / 253 / 250 | Nonuniform; all three blocks decompressed |
| PUB-D | 9 | 749 / 785 / 276 | Nonuniform; all three blocks decompressed |
| PUB-H | 9 | 762 / 796 / 229 | Nonuniform; all three blocks decompressed |
| PUB-S | 7 | 796 / 248 / 255 | Nonuniform; all three blocks decompressed |

INFERRED: based on the public filenames, bit 1 represents white and bit 0 black.
Horizontal stripes becoming `55` and vertical stripes switching every 16 bytes are consistent with 128 vertical pixels stored in 16 bytes, followed by 296 columns in sequence.
The source images and physical display were not compared, so origin, mirroring, rotation, and MSB/LSB order remain UNKNOWN for this two-color format.
Asymmetric corner markers and adjacent single-pixel comparisons would resolve these questions.

INFERRED: a candidate two-color encoder splits the 4,736-byte image into 2,000-byte blocks, compresses each independently with LZO1X, fragments each stream into at most 250 bytes, and wraps them in D3 commands.
Acceptance of another encoder's streams on the two-color hardware was not tested, so arbitrary-image support for that model is not established.
Do not assume that its color codes, plane structure, block size, or compression scheme apply to the four-color model.

## Responses, state transitions, and failure diagnosis

| Observation or failure | Assessment | Next action |
| --- | --- | --- |
| SELECT / D1 returns `90 00` | REPRODUCED: normal response to that request | Save information; distinguish it from display success |
| First D8 returns `6A 86` | OBSERVED in all eight public captures, followed by more commands | Treat as known behavior for this request only; do not ignore all errors |
| D3 / D4 returns `90 00` | OBSERVED in public captures | Candidate acceptance; check all blocks, status, and physical output |
| DE changes `01 90 00` → `00 90 00` | OBSERVED in all eight; busy → complete INFERRED | Compare with physical output for the four-color device |
| USB enumeration unavailable | OBSERVED in a restricted environment | Check USB access |
| Tag not detected | OBSERVED before placing the display | Check placement and move the iPhone away |
| Timeout, removal, or missing fragment | UNKNOWN: fault injection not tested | Test one condition at a time after obtaining a successful sequence; do not assume partial retransmission works |
| Normal responses without a changed display | UNKNOWN: not evaluated in these initial tests | Compare source image, refresh start, completion, power, and placement |

A candidate state machine is `detect → connect → select → read information → verify model → transfer data → request refresh → wait for completion → inspect display → verify reconnection`.
The initial tests reproduced information retrieval and reconnection only.
The opening sections document the subsequent successful update implementation and sequence.
Full success criteria include responses for all blocks, completion notification, a change to the expected image, and another update after reconnection.

## Test vectors and validation

### Complete D3 sequence for the public white image

OBSERVED: matches PUB-W packets 59, 63, and 67.
Lc values are 22, 22, and 17; APDU lengths are 27, 27, and 22; total Lc is 61.
Excluding block and fragment indices, the compressed payload totals 55 bytes.
This sequence alone does not include initialization or refresh completion.

```text
F0 D3 00 01 16 00 00 02 FF FF FF FF FF 20 00 00 00 00 00 00 00 B1 00 00 11 00 00
F0 D3 00 01 16 01 00 02 FF FF FF FF FF 20 00 00 00 00 00 00 00 B1 00 00 11 00 00
F0 D3 00 01 11 02 00 02 FF FF FF FF FF 20 00 00 BC 00 00 11 00 00
```

SHA-256 of the three decompressed blocks concatenated in index order:

```text
white: a3671594682c80e5f08721602dd0136dd1b6c099160ac98ffa6d5b2fca4ba9af
black: 373e58db31dbad517dfede6bb84a58f4f7d5bf03630597ca677658b8bd136106
horizontal_stripes: 15ff8a0281455a6cd649e881d5a5c7511081ffa8b74314ba366b5ac8d6af920d
vertical_stripes: dbb1c2a9c9fe6d6d5ef040dd83e21aac50650737ae48eb801724d2db35787985
```

### Revalidation procedure

1. Download the raw captures from the pinned commit below and compare SHA-256 values.
2. Validate USBPcap and CCID lengths and pair every request with its response.
3. Extract D3 requests and check `len(APDU) == 5 + APDU[4]`.
4. Check that fragment indices start at zero and are contiguous within each block, with P2=`01` only on the final fragment.
5. Concatenate bytes from offset 7, decompress as LZO1X, and compare return codes, actual output lengths, and the hashes above.
6. For the local four-color information test, repeat only the two LOCAL-42-* requests and compare status words and information payloads.

OBSERVED: four CCID unit tests covered a request, a response, truncation, and excess bytes; they failed before implementation and passed afterward.
Length checks, request/response pairing, equality with accompanying D3 text, and decompression of 24 blocks passed for all eight captures.
These are offline analysis checks, not physical four-color display-update tests.

## Four-color behavior found in official distributions

OBSERVED (STATIC-APK-127): two Windows distributions and Android version 1.2.7 were downloaded from official distribution pages and unpacked for static analysis on a Mac.
Sources and SHA-256 values were recorded in a local distribution inventory; function addresses and reproduction conditions are in a private static-analysis record.
Neither the Windows nor Android app was run on its target OS for this analysis.

The Android implementation interprets the local response as 400 × 300, four colors, scanType=1, with compression enabled.
Its four-color packing path uses black=0, white=1, yellow=2, and red=3, packing four pixels per byte from the most significant bits.
Yellow BMP input is matched against exactly RGB `FF C0 00`.
Independent 2,000-byte LZO1X blocks and D3 fragments of at most 250 bytes were also identified.
The same format was subsequently transferred to the local display and visually checked (LOCAL-42-PATTERN, LOCAL-42-ICON).

The refresh function generates `F0 D4 05 80 00`, unlike `F0 D4 85 80 00` in the public Windows capture.
The DE query is `F0 DE 00 00 01`: `01 90 00` means continue waiting, while `00 90 00` or `90 00` proceeds toward completion.
The same response immediately after D4 is handled differently, so responses must be interpreted together with their request commands.

An encoder and sender were implemented on the Mac, and coordinate mapping was compared against the local four-color display.
Generalization to other models remains unverified.
A Windows environment is not required for this analysis path.

## Open questions and next experiments

| Priority | Question | Smallest next experiment |
| --- | --- | --- |
| P0 | Successful physical output from the official iPhone app | Send white then black; record app completion, display changes, app/iOS versions, and duration |
| Complete | Full local four-color sequence | All requests/responses recorded in LOCAL-42-PATTERN and LOCAL-42-ICON |
| Complete | Local four-color mapping and format selection | Bottom-to-top rows, left-to-right pixels, two bits from the MSB; checked with corner labels |
| Complete | RC-S380 updates and power delivery | Pattern with 16 fragments and icon with 40 fragments updated |
| P1 | Model-information field definitions | Compare another unit of the same model to separate per-device values |
| P1 | Partial retransmission and recovery | After stable normal updates, test recovery from one removal |
| P2 | Long-running repeated updates | Pattern-to-icon updates without removal succeeded; long-duration endurance testing remains outstanding |

Arbitrary-image updates from the Mac and physical output were confirmed during this investigation.
iPhone traffic was not captured.

## Evidence inventory and references

| ID | Source |
| --- | --- |
| LOCAL-42-001 | First local hardware log; retained locally, not distributed |
| LOCAL-42-002 | Reconnection hardware log; retained locally, not distributed |
| SRC-README | [Public writer setup](https://github.com/hijimasa/EZ-Sign-nfc-unofficial-writer/blob/4a6200ef7420d42ef3274909d9bf973edcdc6458/README.md) |
| SRC-WRITER | [Public writer implementation](https://github.com/hijimasa/EZ-Sign-nfc-unofficial-writer/blob/4a6200ef7420d42ef3274909d9bf973edcdc6458/write_known_binary.py) |
| PUB-* | [Raw captures at the pinned commit](https://github.com/hijimasa/EZ-Sign-nfc-unofficial-writer/tree/4a6200ef7420d42ef3274909d9bf973edcdc6458/sample_log); filename mapping below |
| DEC-001 | Results of LZO 2.10 `lzo1x_decompress_safe` on PUB-*; inputs and reconstruction described above. [LZO source information](https://www.oberhumer.com/opensource/lzo/) |
| SRC-USBPCAP | [USBPcap capture format](https://desowin.org/usbpcap/captureformat.html) |
| SRC-CCID | [USB CCID Revision 1.1](https://www.usb.org/sites/default/files/DWG_Smart-Card_CCID_Rev110.pdf) |
| SRC-NFCPY | [nfcpy 1.0.4 RC-S380 driver](https://github.com/nfcpy/nfcpy/blob/v1.0.4/src/nfc/clf/rcs380.py) |
| SRC-TYPE4 | [nfcpy 1.0.4 Type 4 / ISO-DEP implementation](https://github.com/nfcpy/nfcpy/blob/v1.0.4/src/nfc/tag/tt4.py) |
| SRC-SONY | [Sony driver support](https://www.sony.co.jp/Products/felica/consumer/support/download/) |
| SRC-PRODUCT | [Manufacturer's two-color product specifications](https://www.santekshop.com/products/santek-ez-sign-nfc-e-paper-2-color-4-2-inch) |
| SRC-APP | [Official app information](https://www.santekshop.com/pages/santek-ez-sign), [FAQ](https://www.santekshop.com/pages/faqs) |

SHA-256 values calculated from the original downloaded captures:

| ID | File | SHA-256 |
| --- | --- | --- |
| PUB-W | white.pcapng | `af864ce32a87108cc63fc3fddde049463d4fe3e3edbb457088aafc78b1aae254` |
| PUB-B | black.pcapng | `599c832c4458b27ec5f24dcc0fcfe18c832701d5c269a72abe89598d61ea4c9f` |
| PUB-HS | horizontal_stripes.pcapng | `2ab8ecee4a236181ff9bea031c23c4a59a7caa91f243d20d250169337ca05dcf` |
| PUB-VS | vertical_stripes.pcapng | `8ae59d99881331ca33929e1e56f4b0dc2039e9154d3c6a7e9238312d7d202a3e` |
| PUB-C | C_A.pcapng | `a99f2c5ff15795605e76b0ed324a1f35574fef1810fea87f3f06ef9b6700e501` |
| PUB-D | D_A.pcapng | `7c213c32505458cbe117f0fed7cb8b4ac5faa06f17b3d4005f0f59bf32f254fb` |
| PUB-H | H_A.pcapng | `3a670c9bdb66dda75cf6b0e5bfe39a879733b71dd5f50c817347f3b54e6904be` |
| PUB-S | S_A.pcapng | `7a2666a1c10c79a755520ff14e621093bddb828e48c29c5890f04994886b0076` |
