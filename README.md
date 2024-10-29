# go-qourier

Go で traQ Bot を簡単に作るためのライブラリ兼テンプレートです。

- `persona`：Bot にメンションでコマンドを実行できる機能など（内部で [go-traq](https://github.com/traPtitech/go-traq) を利用）

- `storage`：Bot が終了しても存続するデータ保存場所（内部で MariaDB を利用）

### 使用例

[main.go](https://github.com/kitsne241/go-qourier/blob/main/main.go)

```go
package main

import (
	"fmt"

	cps "github.com/kitsne241/go-qourier/capsule"
	prs "github.com/kitsne241/go-qourier/persona"
)

type Date struct {
	Day  string `json:"day"`
	Hour int    `json:"hour"`
	Min  int    `json:"min"`
}

func main() {
	prs.SetUp(prs.Commands{
		"set": {Action: set, Syntax: "%s %d:%d"}, // @BOT_name set Sunday 21:00
		"get": {Action: get, Syntax: ""},         // @BOT_name get
	}, onMessage, nil) // onMessage, onFail

	cps.SetUp(Date{Day: "Sunday", Hour: 12, Min: 0}, false) // データベースに接続・必要に応じて初期化
	prs.Start()                                             // Bot を起動
}

func set(ms *prs.Message, day string, hour int, min int) error {
	ms.Channel.Send(fmt.Sprintf("On %s %02d:%02d, right?", day, hour, min))
	cps.Save(Date{Day: day, Hour: hour, Min: min})
	ms.Stamp("done-nya")
	return nil
}

func get(ms *prs.Message) error {
	date, _ := cps.Load[Date]()
	ms.Channel.Send(fmt.Sprintf("It was on %s %02d:%02d!", date.Day, date.Hour, date.Min))
	return nil
}

func onMessage(ms *prs.Message) {
	ms.Channel.Send(fmt.Sprintf("Oisu! Here is #%s", ms.Channel.Path))
}
```

### 環境構築と起動

1. このテンプレートを使用してリポジトリを作成・クローン

2. `.envTEMP` ファイルを `.env` と改名し Bot のアクセストークンと ID を入力（機密情報を Git に上げないよう注意）

3. ディレクトリ `persona` と `storage`、ファイル `go.mod` と `go.sum` を削除

4. Docker Desktop の起動を確認し、シェルを立ち上げ以下を実行

  ```shell
  go mod init リポジトリのパス  # 適宜変更
  go get github.com/kitsne241/go-qourier@latest
  go mod tidy
  task up  # storage を使用する場合
  go run main.go
  ```

### NeoShowcase への登録

5. リポジトリに対する特別な操作は必要とせず、そのまま登録できます

| Application Name | Branch | Deploy Type | Build Type | Use Database | Database | Start Immediately |
| :--------------: | :----: | :---------: | :--------: | :----------: | :------: | :---------------: |
|      Bot 名      |  main  |   Runtime   | Buildpack  |     Yes      | MariaDB  |  チェックしない   |

6. 他の項目はそのままにしておいて、適当な URL を設定してアプリケーションを作成

7. Settings から環境変数の設定を開き 2. で入力した Bot のアクセストークンと ID を追加

8. アプリケーションを起動