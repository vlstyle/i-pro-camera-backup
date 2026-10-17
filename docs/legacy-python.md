# 既存Python実装の整理

出典: `vlstyle/temp-repos`（private）の`camera/`。Goへの再構成元。

## 構成

| ファイル | 役割 |
|---|---|
| `get-movie.py` | エントリポイント。設定読込→カメラごとに`Crawler`を実行 |
| `crawler.py` | カメラとのHTTPやり取り、増分管理、保存（`Crawler`/`ReplayLog`/`ConfigManager`） |
| `zip-movie.sh` | 取得済みmp4を1時間単位でzip化し、S3へ移動 |
| `sample/` | 設定ファイルの例（app_config / credentials / log_config） |

## 設定

- `app_config.json`: `log_config`、`credential_config`、`lasttime_file`（増分管理ファイル名）、`save_dir`
- `credentials.json`: カメラ名をキーに`host`/`id`/`password`
- `log_config.json`: Python logging形式（コンソール出力。ファイル出力は未設定のプレースホルダ）

## 動作

1. カメラごとに`{save_dir}/{カメラ名}/`を作成
2. `lasttime.dat`から前回処理済みの`time_info`を読む（無ければ0）
3. [protocol.md](protocol.md)の手順1〜4で録画ログを取得し、古い順に処理
4. `time_info`が前回値より小さければスキップ。同一なら`fname_no`だけ取得（重複判定用）。大きければ`fname_no`を取得し、直前と異なればダウンロード
5. 1録画処理するたびに`lasttime.dat`を更新、1秒スリープ
6. `zip-movie.sh`が別途、`yymmddHH`単位でzip化（現在時刻の時間帯は除外）して`aws s3 mv`で`s3://camera-vl-style/{カメラ名}/{yyyy}/{yyyymm}/{yyyymmdd}/`へ移動

## 課題・改善候補（Go版で解消したい点）

- 動画全体をメモリに載せてから書き込む（ストリーミング保存にしたい）
- ダウンロード失敗（200以外）でも`lasttime.dat`を更新するため、取りこぼす可能性がある
- 500エラーで全カメラの処理が中断される（カメラ単位で継続したい）
- 同一`time_info`の再処理で毎回`fname_no`を取得する無駄がある
- 書き込みが途中終了すると不完全なmp4が残る（一時ファイル→renameにしたい）
- 正規表現でHTML内JS変数を抜く方式は脆い。失敗時の扱い（Noneの参照で例外）が未整備
- ログのファイル出力が未設定、リトライ・タイムアウトが無い
- 取得とzip/S3退避が別言語・別手順（Python+bash+jq+awscli）で、実行環境の依存が多い
- `credentials.json`が平文
