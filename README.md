# ezsign-go

EZ Signに画像を書き込むGoライブラリです。
画像を1枚ずつ書き込むCLI、スライドショー、HTTP APIを提供します。

対象はRC-S380/Sと4.2インチ4色のEZ Sign（400×300ピクセル）です。
PNGとJPEGに対応し、画像の拡縮と4色への減色を行います。

## インストール

Go 1.27.1以降、Cコンパイラー、libusbの開発パッケージが必要です。
libusbをインストールしてからビルドしてください。

macOS:

```sh
brew install libusb
```

Ubuntu:

```sh
sudo apt update
sudo apt install build-essential libusb-1.0-0-dev
```

リポジトリ直下で実行します。

```sh
CGO_ENABLED=1 make build
```

バイナリは `bin/` に生成されます。
実行環境にもlibusbが必要です。
Linuxへの配置やUSBドライバーとの競合への対処は [デプロイ手順](docs/deploy.md) を参照してください。

## 使い方

RC-S380をUSB接続し、EZ Signを載せます。
同じリーダーを使用するプログラムは1つずつ実行してください。

### 画像を書き込む

```sh
bin/ezsign -write image.png
```

`-write` を省略すると、画像を書き込まずにプレビューを生成します。
上下を反転する場合は、画像パスの前に `-rotate 180` を指定します。

### スライドショーを表示する

```sh
bin/ezsign-go-slideshow -dir ./images -interval 60s
```

ディレクトリ内のPNGとJPEGをファイル名順に繰り返し書き込みます。
`-interval` は書き込み終了後から次の開始までの待ち時間です。
Ctrl+Cで停止します。

### HTTP APIを使う

```sh
bin/ezsign-go-api -listen 127.0.0.1:8080
```

別のターミナルから画像を送信します。

```sh
curl --fail-with-body --max-time 100 \
  -H 'Content-Type: image/png' \
  --data-binary @image.png \
  http://127.0.0.1:8080/v1/image
```

APIに認証機能はありません。
外部へ公開する場合は、リバースプロキシなどで認証とTLSを設定してください。

### Goから使う

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

`WriteImage` は `image.Image` を受け付けます。
`Render` は機器に接続せず、プレビューと転送用データを生成します。

## ドキュメント

- [CLIのオプションと出力ファイル](docs/cli.md)
- [スライドショーの設定](cmd/slideshow/README.md)
- [HTTP APIの仕様](docs/http-api.md)
- [Linuxへのデプロイとトラブルシューティング](docs/deploy.md)
- [EZ Signの通信プロトコル](docs/protocol.md)

## 開発

```sh
make fmt    # コードの整形
make check  # race検出付きテストとgo vet
make build  # ビルド
```

不具合の報告は [Issues](https://github.com/whywaita/ezsign-go/issues) へお願いします。
OS、機器の型番、実行コマンド、エラーメッセージを添えてください。

## ライセンス

[MIT License](LICENSE)
