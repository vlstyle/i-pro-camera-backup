# i-pro-camera-backup

Panasonic i-PRO ネットワークカメラ（WV-S1531L / WV-S3530 など）のSDカード録画を、公式のHTTP CGIで取得し、ローカルディレクトリまたはAWS S3に1動画1ファイルで保存するツール。

1回の実行で1台のカメラの新しい録画を取得して終了するワンショットのプログラム（Go）。カメラごとにプロセス（Pod）を分けて、CronJobなどから定期実行する。

## 使い方

設定はすべて環境変数で渡す（一覧は[docs/requirements.md](docs/requirements.md#環境変数)、サンプルは[.env.example](.env.example)）。

```bash
docker run --rm --env-file .env -v "$PWD/data:/data" ghcr.io/vlstyle/i-pro-camera-backup:latest
```

`DEST_URL=file:///data`の場合は、コンテナの`/data`に書き込めるボリュームを渡す。`DEST_URL=s3://bucket/prefix/`の場合は、AWS SDK標準の環境変数（`AWS_ACCESS_KEY_ID`、`AWS_SECRET_ACCESS_KEY`、`AWS_REGION`など）も渡す。

イメージは`main`へのマージ時にCIがビルドして`ghcr.io/vlstyle/i-pro-camera-backup`に公開する（タグは`latest`とコミットのSHA）。手元でビルドする場合は`docker build -t i-pro-camera-backup .`。

終了コードは、成功が0、設定エラーが2、取得・保存の失敗が1。

## Kubernetes（Helmチャート）

`charts/i-pro-camera-backup/`が、カメラごとのCronJobを作るHelmチャート。`cameras`に名前とホストを並べると、カメラ1台につきCronJobが1つ作られる。環境ごとの値は利用する側のリポジトリで`values.yaml`として持つ（例は[`ci/test-values.yaml`](charts/i-pro-camera-backup/ci/test-values.yaml)、すべての値は[`values.yaml`](charts/i-pro-camera-backup/values.yaml)）。

- `image.tag`と`dest.url`は必須。`cameras[].name`は`[a-z0-9-]`（CronJob名に使う）。
- カメラの認証情報は、Secret（`credentialsSecret`、既定`camera-credentials`）の`<カメラ名>-user`・`<カメラ名>-password`から読む。
- `aws.webIdentity.roleArn`を指定すると、IRSA用のprojected tokenと`AWS_*`環境変数を設定する。

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
