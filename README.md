# go-qourier

Go で traQ Bot を簡単に作るためのライブラリ兼テンプレートです。

- `persona`：Bot にメンションでコマンドを実行できる機能など（内部で [go-traq](https://github.com/traPtitech/go-traq) を利用）



- `capsule`：Bot が終了しても存続するデータ保存場所（内部で MariaDB を利用）

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
	cps.SetUp(Date{Day: "Sunday", Hour: 12, Min: 0}, false) // データベースに接続・必要に応じて初期化
	prs.SetUp(prs.Commands{
		"set": {Action: set, Syntax: "%s %d:%d"}, // @BOT_name set Sunday 21:00
		"get": {Action: get, Syntax: ""},         // @BOT_name get
	})

	prs.OnMessage = func(ms *prs.Message) {
		ms.Channel.Send(fmt.Sprintf("Oisu! Here is #%s", ms.Channel.Path))
	}

	prs.Start() // Bot を起動
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
```

### 環境構築と起動

1. このテンプレートを使用してリポジトリを作成・クローン

2. `.envTEMP` ファイルを `.env` と改名し Bot のアクセストークンと ID を入力（機密情報を Git に上げないよう注意）

3. ディレクトリ `persona`・`capsule`、ファイル `go.mod`・`go.sum`・`README.md` を削除

4. Docker Desktop の起動を確認し、シェルを立ち上げ以下を実行

  ```shell
  go mod init リポジトリのパス  # 適宜変更
  go get github.com/kitsne241/go-qourier@latest
  go mod tidy
  task up
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

## 各パッケージの説明

### persona

traQ Bot がメンションから複数のコマンドを認識できるように実装しようとすると、コマンドを理解するためのパーサーの実装は冗長になりがちです。persona パッケージは、Bot が受け取ったメンションからコマンドを解釈して実行するための枠組みを提供し、その他メッセージやユーザー、チャンネルの取り扱いに関する基本的な関数を備えています。

このパッケージが十全に機能するためには以下の環境変数が登録されている必要があります。

| 変数名           | 説明                                |
| ---------------- | ----------------------------------- |
| **ACCESS_TOKEN** | Bot のアクセストークン              |
| **BOT_ID**       | Bot の ID。Bot User ID ではないほう |

このパッケージは投稿されたメッセージをトリガーとして操作を実行する（あるいは cron などの外部パッケージを導入することで定期的に動作する）Bot の開発を主な用途として想定しています。このパッケージで用意されていないリクエストの送受信は `prs.Wsbot` から [traq-ws-bot](https://github.com/traPtitech/traq-ws-bot) 及び [go-traq](https://github.com/traPtitech/go-traq/tree/master) が提供する関数にアクセスして実現することができます。詳細は [Go による traQ Bot 開発](https://wiki.trap.jp/user/kitsne/memo/Go%20による%20traQ%20Bot%20開発) などいくつか traP Wiki に記事があるので参考にしてください。

### capsule

プログラムを再起動すると変数などに保存されたデータは失われてしまいます。プログラムの停止や再起動に影響を受けずにデータを永続的に保存する方法として、データベースを使用することが一般的です。capsule パッケージは、さほど大きくないデータを JSON 形式でデータベースに登録し永続化するための各種関数を提供します。

<details><p></p>
データベースは次のような階層構造を持ちます。

> データベース　＞　テーブル　＞　レコード　＞　フィールド

フィールドとは属性のことで、レコードはいくつかのフィールドを持ちます。完全に同じフィールドを持つ沢山のレコードがひとつのテーブルに収められ、それぞれ異なるフィールドの組を持ちうるいくつかのテーブルがひとつのデータベースを成します。

たとえば SNS の投稿の情報をデータベースで管理する場合、全ての投稿を網羅したテーブルを作成し、ひとつの投稿がひとつのレコードに対応するようにデータベースを設計できます。投稿はその内容、それ自体の ID、作者の ID、投稿日時などといった属性を持つので、それぞれをフィールドとして扱えば収まりが良さそうです。

データベースは SNS の投稿のような同じフィールドの組を持つ膨大な数のレコードを管理するのに適していますが、朗読できる程度の規模の設定を保存しておくには少々オーバースペックです。そこで、この capsule モジュールは以下のようにデータベースを扱います。

- テーブル `config` を作成し、ひとつだけレコードを用意する

- レコードが持つフィールドは JSON 形式の文字列を収納できる `json` のみ

すなわち、データベースにはたったひとつの JSON 文字列を保存します。各要素に json タグを持つ Go の構造体は JSON 文字列との間で容易に相互に変換でき、この方法は構造体自身の構造をデータベースの階層構造に対応させる方法に比べて高い汎用性を備えています。

MySQL や MariaDB などの多くのデータベース管理システムでは、レコードの最大容量（行サイズ）に 64 kB 程度の制限がかかっています。1 文字あたり 1B とすると 65,000 字程度で上限に達する計算になります（妥当か否かは…）。しかし、MySQL や MariaDB でサポートされている JSON 型のサイズの扱いは LONGTEXT 型に準ずるため（他の制約がない限り）最大で約 4 GB までのデータを収納することができます。これらの型をデータベースに保存しようとすると、データそのものは別の領域に保存され、レコードのフィールドにはそのデータへのポインタが格納される仕様になっているようです。

[MySQL 8.0 リファレンスマニュアル](https://dev.mysql.com/doc/refman/8.0/ja/storage-requirements.html)

また、このモジュールで用意されている関数はテーブル `config` の外側に対する処理をほとんど行わないので、このデータベースに他のテーブルを作成して処理を加えても `config` に保存された JSON データの読み書きには影響を与えません。実例として [BOT_neku](https://git.trap.jp/kitsne/bot_neku) では単語のつながりを保存する別のテーブルを手動で作成して用いています。

---

</details>

このパッケージが十全に機能するためには以下の環境変数が登録されている必要があります。

| 変数名                  | 説明                                       |
| ----------------------- | ------------------------------------------ |
| **NS_MARIADB_USER**     | データベースにアクセスするユーザー名       |
| **NS_MARIADB_PASSWORD** | データベースにアクセスするためのパスワード |
| **NS_MARIADB_HOSTNAME** | データベースのホスト名                     |
| **NS_MARIADB_PORT**     | データベースが用意するアクセスポート       |
| **NS_MARIADB_DATABASE** | データベース名                             |

環境変数の頭に NS_MARIADB とつくのは、現在の NeoShowcase の環境変数の設定に合わせることでリポジトリのデプロイ時の操作を単純にするためです。

主な用途として NeoShowcase 上で運用する traQ Bot のためのデータ永続化を想定していますが、他にも何らかの理由で少量のデータを保っておきたい場合にこのパッケージを用いることができます。データベースに対してより高度な操作をする場合は `cps.Db` から [sqlx](https://github.com/jmoiron/sqlx) が用意する関数にアクセスすることができます。詳細は Web エンジニアになろう講習会の『Go でデータベースを扱う』の項を確認してください。