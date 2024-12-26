package persona

// traQ における Bot 自身の動作の関数
// go-traq をさらに機能を絞って discord.py 風にラップしたもの
// 方針は「内部に長命な情報を持たず」して「できる限り少ない API 呼び出しで必要な情報を得る」こと

import (
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
	traqwsbot "github.com/traPtitech/traq-ws-bot"
	payload "github.com/traPtitech/traq-ws-bot/payload"
)

type Command struct {
	Action any    // *Message 型 とその他 0 個以上の引数を持ち、error 型を返す関数
	Syntax string // %s（文字列）, %d（数）, %x（無視）を用いた文字列として指定するコマンドの型

	// 以下は SetUp の実行によって自動で追加される
	Name   string                       // Bot を呼び出すときのコマンド名
	action func(*Message, ...any) error // Action を可変引数化した関数。実際に実行されるのはこっち
}

// コマンド以外で新規メッセージを受け取ったときに呼ばれる関数
var OnMessage func(*Message)

// コマンドの実行に失敗したときに呼ばれる関数
var OnFail func(*Message, *Command, error)

// 投稿のメッセージにスタンプが追加・削除されたときに呼ばれる関数
var OnStampUpdate func(*Message)

// WebSocket Bot 本体
var Wsbot *traqwsbot.Bot

var Me *User

type Commands map[string]*Command

func init() {
	godotenv.Load(".env")
}

// コマンドセットを Bot に入力して初期化する
func SetUp(commands Commands) {
	err := error(nil)
	for name, command := range commands {
		command.Name = name
		command.action, err = varadic(command)
		if err != nil {
			panic(color.HiRedString("[failed to register command '%s'] %s", name, err))
		}
	}
	// Command 型の配列である引数 commands から {関数名: 実行関数} の辞書 commandsDic を得る
	// この際 varadic の内部で関数の構造が条件に適合しているかの審査を同時に行い、不適正なら panic する

	Wsbot, err = traqwsbot.NewBot(&traqwsbot.Options{ // Bot を作成
		AccessToken: os.Getenv("ACCESS_TOKEN"),
	})
	if err != nil {
		panic(color.HiRedString("[failed to create a new bot] %s", err))
	}

	if Me = getMe(); Me == nil {
		panic(color.HiRedString("[failed to build a bot] make sure ACCESS_TOKEN is set!"))
	}

	Wsbot.OnMessageCreated(func(p *payload.MessageCreated) {
		ms := GetMessage(p.Message.ID)
		if ms == nil {
			return
		}

		// 送られてきたメッセージがコマンドであるならば適切に解釈してコマンドを実行する

		_, embeds := Unembed(ms.Text)

		if (len(embeds) > 0) && (embeds[0].Start == 0) {
			if (embeds[0].Type == "user") && (embeds[0].ID == Me.ID) {
				// メッセージの最初で Bot 自身に対するメンションがなされている場合

				elements := strings.SplitN(strings.TrimSpace(ms.Text[embeds[0].End:]), " ", 2)
				// "@BOT_name" 以降のメッセージテキストで最初の半角スペースを見つけて最大 2 つに切り分ける
				elements = append(elements, make([]string, 2-len(elements))...) // 常に elements の長さを 2 にする
				command, exists := commands[elements[0]]
				if exists {
					// "@BOT_name コマンド" または "@BOT_name コマンド 引数" の形式のみコマンドとして認識
					if err = command.parseExecute(ms, elements[1]); err != nil {
						if OnFail != nil {
							OnFail(ms, command, err)
						} else {
							log.Println(color.HiYellowString("[failed to run command '%s'] %s", elements[0], err))
						}
					}
					return
				}
			}
		}

		// コマンドの実行条件に当てはまらなかった場合、通常メッセージとして扱い onMessage を実行する
		if OnMessage != nil {
			OnMessage(ms)
		}
	})

	Wsbot.OnBotMessageStampsUpdated(func(p *payload.BotMessageStampsUpdated) {
		ms := GetMessage(p.MessageID)
		if ms == nil {
			return
		}

		// どのスタンプが変更されたかの情報までは提供されていない
		// 必要があれば逐一データベースに保存して変更前と照合することで情報を得ることはできる
		OnStampUpdate(ms)
	})

	log.Println(color.GreenString("[initialized bot]"))
}

// Bot を起動。Bot が停止するとエラーを表示し panic する
func Start() error {
	if Wsbot == nil {
		panic(color.HiRedString("[bot is not set up]"))
	}
	err := Wsbot.Start()
	panic(color.HiRedString("[bot shut down] %s", err))
}
