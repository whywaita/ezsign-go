# HTTP API

サーバーを起動します。

```sh
bin/ezsign-go-api -listen 127.0.0.1:8080
```

画像ファイルをリクエストボディへ直接送信します。
JPEGの場合は `Content-Type: image/jpeg` を指定します。

```sh
curl --fail-with-body --max-time 100 \
  -H 'Content-Type: image/png' \
  --data-binary @image.png \
  'http://127.0.0.1:8080/v1/image'
```

| エンドポイント | 動作 |
| --- | --- |
| `GET /healthz` | HTTPプロセスの生存状態を返す |
| `POST /v1/image` | 書き込み完了を待ち、結果をJSONで返す |

`POST /v1/image` はクエリ `rotation=0` または `180`（既定 `0`）と、`dither=true` または `false`（既定 `true`）を受け付けます。
入力上限は10 MiB、2,000万画素です。
更新中のリクエストにはHTTP 409を返します。

成功時は画像サイズ、画素データのSHA-256、処理時間を返します。
`status` の `protocol_complete_visual_check_required` は通信上の更新完了を表し、実画面の画像一致を判定するものではありません。

APIに認証機能はありません。
既定ではlocalhostで待ち受けます。
外部から接続する場合は、リバースプロキシなどで認証とTLSを設定してください。
