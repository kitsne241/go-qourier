# persona

traQ Bot がメンションから複数のコマンドを認識できるように実装しようとすると、コマンドを理解するためのパーサーの実装は冗長になりがちです。persona パッケージは、Bot が受け取ったメンションからコマンドを解釈して実行するための枠組みを提供し、その他メッセージやユーザー、チャンネルの取り扱いに関する基本的な関数を備えています。このパッケージが十全に機能するためには以下の環境変数が登録されている必要があります。

| 変数名           | 説明                                |
| ---------------- | ----------------------------------- |
| **ACCESS_TOKEN** | Bot のアクセストークン              |
| **BOT_ID**       | Bot の ID。Bot User ID ではないほう |

このパッケージは投稿されたメッセージをトリガーとして操作を実行する（あるいは cron などの外部パッケージを導入することで定期的に動作する）Bot の開発を主な用途として想定しています。このパッケージで用意されていないリクエストの送受信は `prs.Wsbot` から [traq-ws-bot](https://github.com/traPtitech/traq-ws-bot) 及び [go-traq](https://github.com/traPtitech/go-traq/tree/master) が提供する関数にアクセスして実現することができます。詳細は [バックエンド備忘録 - traQ Bot](https://wiki.trap.jp/user/kitsne/memo/バックエンド備忘録%20-%20traQ%20Bot) を確認してください。

## 型

### Command

Bot が実行可能なコマンドを表現する型です。関数 `SetUp` の引数に含まれているコマンドについては、traQ から `Syntax` の形式で Bot にメンションを送ることで Bot が `Action` を実行できるようになります。

```go
type Command struct {
	Action any
	Syntax string
	Name   string
}
```

| フィールド | 説明                                                                                                |
| ---------- | --------------------------------------------------------------------------------------------------- |
| **Action** | Bot が実行する関数。 `*Message` 型とその他 0 個以上の引数を持ち、`error` 型を返す                   |
| **Syntax** | コマンドを実行する時の文法。`%s`（文字列）、`%d`（数）、`%x`（無視）を用いることができる            |
| **Name**   | コマンドの名称。Bot へのメンションにこの文字列が続くメッセージを送信すると Bot がコマンドを実行する |

`Action` の定義が `any` 型なのは、Go には条件を満たす関数を適切に示す型が存在しないためです。安全のため、関数 `SetUp` 実行時に関数を可変数引数関数に変換し、条件を満たさない（変換できない）場合は `panic` を起こします。

関数 `SetUp` に渡すときに `Name` フィールドは不要です。README.md の書き方に従い `map` のキーとして名称を渡してください。

### Commands

Command 型のコマンドとその名称との対応を表す `map[string]*Command` 型を特別に `Commands` と名付けたものです。関数 `SetUp` の第一引数はこの型で渡されます。

```go
type Commands map[string]*Command
```

### Channel

traQ チャンネルを表現する型です。

```go
type Channel struct {
	Name   string
	Path   string
	ID     string
	Parent *Channel
}
```

| フィールド | 説明                                            | 例                      |
| ---------- | ----------------------------------------------- | ----------------------- |
| **Name**   | チャンネル名                                    | `"kitsnegra"`           |
| **Path**   | チャンネルの所在を表す文字列。頭に # がない形式 | `"gps/times/kitsnegra"` |
| **ID**     | チャンネルのもつ UUID                           |                         |
| **Parent** | 自身の親チャンネルを表す `Channel` 型のポインタ |                         |

### User

traQ ユーザーを表現する型です。

```go
type User struct {
	Nick  string
	Name  string
	ID    string
	IsBot bool
}
```

| フィールド | 説明                                       | 例         |
| ---------- | ------------------------------------------ | ---------- |
| **Nick**   | ユーザーの表示名                           | `"きつね"` |
| **Name**   | ユーザー名。頭に @ がない形式              | `"kitsne"` |
| **ID**     | ユーザーのもつ UUID。`Name` との混同に注意 |            |
| **IsBot**  | ユーザー自身が Bot であるか否か            | `false`    |

### Message

traQ のメッセージを表現する型です。

```go
type Message struct {
	Channel   *Channel
	Text      string
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Author    *User
}
```

| フィールド    | 説明                                             |
| ------------- | ------------------------------------------------ |
| **Channel**   | メッセージが投稿されたチャンネル                 |
| **Text**      | メッセージの本文。`!{}` 形式の埋め込みなどを含む |
| **ID**        | ユーザーのもつ UUID。`Name` との混同に注意       |
| **CreatedAt** | メッセージの投稿日時を JST で                    |  |
| **UpdatedAt** | メッセージの編集日時を JST で                    |
| **Author**    | メッセージを投稿したユーザー                     |

## 変数

### Wsbot

traq-ws-bot によって型定義がなされている WebSocket 通信式 traQ Bot のインスタンスです。関数 `SetUp` の実行によって内容が定義されます。

```go
var Wsbot *traqwsbot.Bot
```

### Me

Bot 自身の情報をまとめた `*User` 型です。関数 `SetUp` の実行によって内容が定義されます。

```go
var Me *User
```

## 非メソッド関数

### SetUp

内部で traq-ws-bot を動かすことで traQ に接続し、与えられたコマンドの変換と追加、他の関数の実行に必要な情報の取得などをまとめて実行します。初期化に失敗するとその場で `panic` します。このパッケージとは異なる方法で traQ Bot API を活用したい場合には、この関数の実行後に `prs.Wsbot` を利用することができます。

```go
func SetUp(
	commands Commands,
	onMessage func(*Message),
	onFail func(*Message, *Command, error),
)
```

#### 引数

| 名称          | 説明                                                                         |
| ------------- | ---------------------------------------------------------------------------- |
| **commands**  | Bot が実行を可能にするコマンドとその名称の対応。nil でもよい                 |
| **onMessage** | コマンドに当てはまらないメッセージを受け取った際に実行する関数。nil でもよい |
| **onFail**    | いずれかのコマンドの実行に失敗した時に実行する関数。nil でもよい             |

### GetChannel

与えられた引数の UUID を持つチャンネルを取得して返します。見つからない場合は `nil` を返します。

```go
func GetChannel(chID string) *Channel
```

### PathGetChannel

与えられた引数のパス（# を除く文字列）を持つチャンネルを取得して返します。見つからない場合は `nil` を返します。

```go
func PathGetChannel(path string) *Channel
```

### GetUser

与えられた引数の UUID を持つユーザーを取得して返します。見つからない場合は `nil` を返します。

```go
func GetUser(usID string) *User
```

### NameGetUser

与えられた引数の名前（@ を除く文字列）を持つユーザーを取得して返します。見つからない場合は `nil` を返します。

```go
func NameGetUser(name string) *User
```

### GetMessage

与えられた引数の UUID を持つメッセージを取得して返します。見つからない場合は `nil` を返します。

```go
func GetMessage(msID string) *Message
```

### Unembed

与えられた引数の文字列に含まれる埋め込みを外します。`Message.Text` 型は埋め込みを含むことがあるので、埋め込みのない状態のメッセージ本文が必要な場合はこの関数を使用してください。

```go
func Unembed(text string) string
```

## *Channel 型のメソッド

### GetChildren

子チャンネルの配列を `[]*Channel` 型で返します。見つからない場合及びレシーバが `nil` である場合は空の配列を返します。

```go
func (ch *Channel) GetChildren() []*Channel
```

### GetRecentMessages

チャンネルの直近の投稿を最大で `limit` 個だけ読み込んで返します。チャンネルに投稿されたメッセージが `limit` 個未満の場合、投稿されたメッセージ全てを読み込んで返却します。レシーバが `nil` である場合は空の配列を返します。

```go
func (ch *Channel) GetRecentMessages(limit int) []*Message
```

### Send

Bot としてチャンネルにテキスト `content` を投稿します。レシーバが `nil` である場合は何もしません。

```go
func (ch *Channel) Send(content string)
```

### Join

チャンネルに参加します。チャンネルに参加するとそのチャンネルでの Bot 自身へのメンション以外の投稿を購読できるようになります。レシーバが `nil` である場合は何もしません。

```go
func (ch *Channel) Join()
```

### Leave

チャンネルから脱退します。レシーバが `nil` である場合は何もしません。

```go
func (ch *Channel) Leave()
```

## *User 型のメソッド

現状ありません。欲しい機能がある場合には Issue を立ててもらえれば検討します。あるいは、プルリクエストを投げたり開発に協力してもらえたら大歓迎です。

## *Message 型のメソッド

### Stamp

メッセージにスタンプをつける多変数引数関数です。スタンプ名は両側のコロンを除外した文字列として指定します。レシーバが `nil` である場合は何もしません。

```go
func (ms *Message) Stamp(stamps ...string)
```
