# カメラとのHTTPやり取り

既存Python実装（[legacy-python.md](legacy-python.md)）のコードから読み取った内容。機種・ファームウェアによる差異は未確認。

## 前提

- 通信はHTTP（平文）、認証はBasic認証（ユーザーID/パスワード）
- 一連のやり取りは同一セッション内で行い、`Cookie: Session=0`、ブラウザ相当の`User-Agent`/`Accept`を付与する
- `Referer`は直前に取得したURLを設定する

## 手順

| # | リクエスト | 目的 |
|---|---|---|
| 1 | `GET /` から `/live/menubar.html?Language=1&ViewMode=pull` まで、下記の順に8ページ | Web UIの画面遷移を再現してセッションを初期化 |
| 2 | `GET /cgi-bin/getuid?FILE=1&reply=ws&vcodec=jpeg_2&quality=1&Language=1&ViewMode=pull` | 応答HTMLの`var giUid = (\d+);`からuidを取得 |
| 3 | `GET /cgi-bin/list_disp?uid={uid}&reply=ws&Language=1&ViewMode=pull` | 応答は使わないが、呼び出しが必要（ブラウザの挙動を再現） |
| 4 | `GET /cgi-bin/list_replaylog?page_control=start_search&uid={uid}&device=sd1&channel=1&type=alm&type=local&type=ftp&from=first&to=now&Language=1&ViewMode=pull` | 録画ログ一覧の取得。応答の`var sData = "...";`がカンマ区切り |
| 5 | `GET /cgi-bin/get_mov?res_type=dl_html&get_type=1&alm_kind=..&time_info=..&uid=..&channel=..&device=..&Language=1` | 応答の`var giSavePage1 =parseInt("(\d+)");`からfname_noを取得 |
| 6 | `GET /cgi-bin/get_mov?res_type=force_dl&fname_no={fname_no}&get_type=1&alm_kind=..&time_info=..&uid=..&channel=..&device=..&Language=1` | MP4本体のダウンロード |

手順1のページ順:

1. `/`
2. `/live/index2.html?Language=1`
3. `/live/index.html?Language=1&ViewMode=pull`
4. `/live/titlebar.html?Language=1&ViewMode=pull`
5. `/live/titlebar_title.html?Language=1&ViewMode=pull`
6. `/live/titlebar_main.html?Language=1&ViewMode=pull`
7. `/live/titlebar_hide.html?Language=1&ViewMode=pull`
8. `/live/menubar.html?Language=1&ViewMode=pull`

## 録画ログ1行の構造

`sData`をカンマで分割した1要素が1録画。既存実装が使っている位置は次のとおり（それ以外のフィールドの意味は未解析）。

| 位置 | 意味 | 使い道 |
|---|---|---|
| `[0]` | SDカード番号 | `device=sd{n}` |
| `[2]` | チャンネル | `channel` |
| `[8:20]` | 録画時刻（12桁、`yymmddHHMMSS`と推定） | `time_info`、ファイル名、増分取得のキー |
| `[23]` | 録画種別（alm_kind） | `alm_kind` |

- 一覧は新しい順で返る想定で、既存実装は反転して古い順に処理している
- 同一の録画が複数行（種別違い等）に現れることがあり、`fname_no`が同じなら重複ダウンロードしない運用になっている

## レスポンス処理の注意

- ステータス500はエラー扱い（既存実装は処理全体を中断）。200以外は警告のみ
- 応答はHTML内のJavaScript変数を正規表現で抜き出している（uid、sData、giSavePage1）
- 動画は応答ボディをそのまま保存（ファイル名は`{fname_no}_{time_info}.mp4`）
