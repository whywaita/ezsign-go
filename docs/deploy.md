# Linuxへの配置と実行

## 実行条件

Raspberry Pi上のUbuntu（aarch64、カーネル `6.8.0-1064-raspi`）で、RC-S380のUSBインターフェースを `port100` ドライバーが確保する競合を確認しました。
`port100` を外すことで問題が解消したことをユーザーが確認しています。

Linux arm64用にCGOを有効にしてビルドしたバイナリを配置します。
実行環境にはlibusbが必要です。

```sh
sudo apt update
sudo apt install libusb-1.0-0
```

## RC-S380が使用中になる場合

次のエラーはUSBインターフェースが使用中であることを示します。
`sudo` を付けても、ドライバーとの競合は解消しません。

```text
claim RC-S380 interface: -6
```

slideshowをCtrl+Cで停止し、ドライバーと競合プロセスを確認します。

```sh
lsusb -t
pgrep -af 'pcscd|ezsign'
```

RC-S380の行に `Driver=port100` が表示される場合、次のコマンドでカーネルモジュールを一時的に外します。
この操作は同じモジュールを使用するほかの機器にも影響します。

```sh
sudo modprobe -r port100
```

別のezsignプロセスがある場合は終了させます。
`pcscd` が機器を使用している場合は、必要に応じて `pcscd.socket` と `pcscd.service` を停止します。
今回の環境では競合プロセスはなく、原因は `port100` でした。

## slideshowの実行

バイナリをカレントディレクトリ、画像を `images/` に配置した例です。
HTTP APIの起動は不要です。

```sh
sudo ./ezsign-go-slideshow -dir ./images -interval 10s
```

`-interval` は書き込み終了後から次の開始までの待ち時間です。
終了する場合はCtrl+Cを押します。

## ドライバーを戻す

slideshowを終了した後、次のコマンドで戻します。

```sh
sudo modprobe port100
```

`modprobe -r` は永続的な無効化ではありません。
再起動や機器の再接続後に競合が再発した場合は、`lsusb -t` で確認してください。
