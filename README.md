# i-pro-camera-downloader

Panasonic i-PRO ネットワークカメラ（WV-S1531L / WV-S3530 など）のSDカード録画を、カメラのWeb UIが使うHTTPのやり取りで取得するツール。

現在は設計整理の段階で、実装はこれから（Go）。

## ドキュメント

- [docs/requirements.md](docs/requirements.md): このプロダクトで成し遂げること、スコープ、未決事項
- [docs/protocol.md](docs/protocol.md): カメラとのHTTPやり取りの整理（既存Python実装の解析結果）
- [docs/legacy-python.md](docs/legacy-python.md): 既存Python実装の構成・挙動・課題

## 対象機種

| 機種 | 備考 |
|---|---|
| WV-S1531L | 動作実績あり（既存Python実装の運用先） |
| WV-S3530 | 動作実績あり（既存Python実装の運用先） |

## ライセンス

[MIT](LICENSE)
