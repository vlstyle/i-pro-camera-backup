# i-pro-camera-downloader

Panasonic i-PRO ネットワークカメラ（WV-S1531L / WV-S3530 など）のSDカード録画を、公式のHTTP CGIで取得し、ローカルディレクトリまたはAWS S3に1動画1ファイルで保存するツール。

現在は設計整理の段階で、実装はこれから（Go）。

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
