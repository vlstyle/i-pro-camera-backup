# i-pro-camera-downloader

Panasonic i-PRO ネットワークカメラ（WV-S1531L / WV-S3530 など）のSDカード録画を、公式のHTTP CGIで取得し、ローカルディレクトリまたはAWS S3に1動画1ファイルで保存するツール。

1回の実行で1台のカメラの新しい録画を取得して終了するワンショットのプログラム（Go）。カメラごとにプロセス（Pod）を分けて、CronJobなどから定期実行する。

## 使い方

設定はすべて環境変数で渡す（一覧は[docs/requirements.md](docs/requirements.md#環境変数)、サンプルは[.env.example](.env.example)）。

```bash
docker build -t i-pro-camera-downloader .
docker run --rm --env-file .env -v "$PWD/data:/data" i-pro-camera-downloader
```

`DEST_URL=file:///data`の場合は、コンテナの`/data`に書き込めるボリュームを渡す。`DEST_URL=s3://bucket/prefix/`の場合は、AWS SDK標準の環境変数（`AWS_ACCESS_KEY_ID`、`AWS_SECRET_ACCESS_KEY`、`AWS_REGION`など）も渡す。

終了コードは、成功が0、設定エラーが2、取得・保存の失敗が1。

## 開発

```bash
go vet ./...
go test ./...
```

## ドキュメント

- [docs/requirements.md](docs/requirements.md): このプロダクトで成し遂げること、動作、設定、未決事項
- [docs/protocol.md](docs/protocol.md): カメラとのHTTPやり取り（CGI仕様と実機での確認結果）

## 対象機種

| 機種 | 備考 |
|---|---|
| WV-S1531L | 実機で動作確認済み |
| WV-S3530 | 実機で動作確認済み |

## ライセンス

[MIT](LICENSE)
