package persona

// traQ における Bot 自身の動作の関数
// go-traq をさらに機能を絞って discord.py 風にラップしたもの

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

type Bot struct {
	Wsbot         *traqwsbot.Bot
	Me            *User
	Commands      map[string]*Command
	OnMessage     func(*Message)
	OnFail        func(*Message, *Command, error)
	OnStampUpdate func(*Message)
}

type Commands map[string]*Command

func init() {
	godotenv.Load(".env")
}

// main.go で使うサブパッケージの関数は全て大文字から始める。小文字スタートのままではインポートが失敗する

func (bot *Bot) SetUp(commands Commands) {
	var err error
	for name, command := range commands {
		command.Name = name
		command.action, err = varadic(command)
		if err != nil {
			panic(color.HiRedString("[failed to register command '%s'] %s", name, err))
		}
	}
	// Command 型の配列である引数 commands から {関数名: 実行関数} の辞書 commandsDic を得る
	// この際 varadic の内部で関数の構造が条件に適合しているかの審査を同時に行い、不適正なら panic する

	bot.Wsbot, err = traqwsbot.NewBot(&traqwsbot.Options{ // Bot を作成
		AccessToken: os.Getenv("ACCESS_TOKEN"),
	})
	if err != nil {
		panic(color.HiRedString("[failed to create a new bot] %s", err))
	}

	if bot.Me = bot.getMe(); bot.Me == nil {
		panic(color.HiRedString("[failed to build a bot] make sure ACCESS_TOKEN is set!"))
	}

	bot.Wsbot.OnMessageCreated(func(p *payload.MessageCreated) {
		ms := bot.GetMessage(p.Message.ID)
		if ms == nil {
			return
		}

		// 送られてきたメッセージがコマンドであるならば適切に解釈してコマンドを実行する

		_, embeds := Unembed(ms.Text)

		if (len(embeds) > 0) && (embeds[0].Start == 0) {
			if (embeds[0].Type == "user") && (embeds[0].ID == bot.Me.ID) {
				// メッセージの最初で Bot 自身に対するメンションがなされている場合

				elements := strings.SplitN(strings.TrimSpace(ms.Text[embeds[0].End:]), " ", 2)
				// "@BOT_name" 以降のメッセージテキストで最初の半角スペースを見つけて最大 2 つに切り分ける
				elements = append(elements, make([]string, 2-len(elements))...) // 常に elements の長さを 2 にする
				command, exists := commands[elements[0]]
				if exists {
					// "@BOT_name コマンド" または "@BOT_name コマンド 引数" の形式のみコマンドとして認識
					if err = command.parseExecute(ms, elements[1]); err != nil {
						if bot.OnFail != nil {
							bot.OnFail(ms, command, err)
						} else {
							log.Println(color.HiYellowString("[failed to run command '%s'] %s", elements[0], err))
						}
					}
					return
				}
			}
		}

		// コマンドの実行条件に当てはまらなかった場合、通常メッセージとして扱い onMessage を実行する
		if bot.OnMessage != nil {
			bot.OnMessage(ms)
		}
	})

	bot.Wsbot.OnBotMessageStampsUpdated(func(p *payload.BotMessageStampsUpdated) {
		ms := bot.GetMessage(p.MessageID)
		if ms == nil {
			return
		}

		// どのスタンプが変更されたかの情報までは提供されていない
		// 必要があれば逐一データベースに保存して変更前と照合することで情報を得ることはできる
		bot.OnStampUpdate(ms)
	})

	log.Println(color.GreenString("[initialized bot]"))
}

func (bot *Bot) Start() error {
	if bot.Wsbot == nil {
		panic(color.HiRedString("[bot is not set up]"))
	}
	return bot.Wsbot.Start()
}
