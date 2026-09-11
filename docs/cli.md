# 画像書き込みCLI

RC-S380をUSB接続し、EZ Signを載せて実行します。
書き込みが終わるまで本体を動かさないでください。
同じリーダーは1つのプロセスで使用します。

```sh
bin/ezsign -write image.png
```

書き込まずにプレビューを生成する場合は、`-write` を省略します。

```sh
bin/ezsign -output output/preview image.png
```

出力先には `preview.png` と転送用の `frame.bin` を保存します。
実機へ書き込む場合は、通信ログの `session.json` も保存します。
フラグは画像パスの前に指定してください。

| フラグ | 既定値 | 内容 |
| --- | --- | --- |
| `-write` | `false` | 実機へ書き込む |
| `-output` | `output` | ファイルの出力先 |
| `-rotate` | `0` | 基準方向からの回転角度。`0` または `180` |
| `-no-dither` | `false` | 中間色を点描で表現する処理を無効にする |
| `-timeout` | `90s` | 処理のタイムアウト |

表示が上下逆になる場合は `-rotate 180` を指定します。
