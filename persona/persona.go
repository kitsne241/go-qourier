package persona

// traQ における Bot 自身の動作の関数
// go-traq をさらに機能を絞って discord.py 風にラップしたもの

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	cp "github.com/kitsne241/go-qourier/cprint"
	traqwsbot "github.com/traPtitech/traq-ws-bot"
	payload "github.com/traPtitech/traq-ws-bot/payload"
)

type Command struct {
	Action any    // *Message 型 とその他 0 個以上の引数を持ち、error 型を返す関数
	Syntax string // %s（文字列）, %d（数）, %x（無視）を用いた文字列として指定するコマンドの型

	// 以下は SetUp の実行によって自動で追加される
	name   string                       // Bot を呼び出すときのコマンド名
	action func(*Message, ...any) error // Action を可変引数化した関数。実際に実行されるのはこっち
}

var Wsbot *traqwsbot.Bot

var Me *User

var stampID = map[string]string{} // スタンプの名前と ID の対応の辞書

// main.go で使うサブパッケージの関数は全て大文字から始める。小文字スタートのままではインポートが失敗する

func init() {
	godotenv.Load(".env")
}

func SetUp(commands map[string]*Command, onMessage func(*Message), onFail func(*Message, *Command, error)) {
	// onMessage : 受け取ったメッセージがコマンドでない場合に呼ばれる関数
	// onFail    : 何らかの原因でコマンドの実行が失敗したときに呼ばれる関数

	var err error
	for name, command := range commands {
		command.name = name
		command.action, err = varadic(command)
		if err != nil {
			cp.CPanic("[failed to register command '%s'] %s", name, err)
		}
	}
	// Command 型の配列である引数 commands から {関数名: 実行関数} の辞書 commandsDic を得る
	// この際 varadic の内部で関数の構造が条件に適合しているかの審査を同時に行い、不適正なら panic する

	Wsbot, err = traqwsbot.NewBot(&traqwsbot.Options{ // Bot を作成
		AccessToken: os.Getenv("ACCESS_TOKEN"),
	})
	if err != nil {
		cp.CPanic("[failed to create a new bot] %s", err)
	}

	if Me = GetMe(); Me == nil {
		cp.CPanic("[failed to build a bot]")
	}

	mention := fmt.Sprintf("!{\"type\":\"user\",\"raw\":\"@%s\",\"id\":\"%s\"}", Me.Name, Me.ID)
	// メッセージ本文などではメンションは JSON 形式の文字列に置き換えられている

	Wsbot.OnMessageCreated(func(p *payload.MessageCreated) {
		ms := GetMessage(p.Message.ID)
		if ms == nil {
			return
		}

		content := strings.Replace(ms.Text, mention, "@"+Me.Name, 1)
		elements := strings.SplitN(content, " ", 3)
		// 最初の 2 つの半角スペースを見つけて最大 3 つに切り分ける。@BOT_name / コマンド / 引数
		elements = append(elements, make([]string, 3-len(elements))...)
		// elements の長さが常に 3 になるように規格化

		command, exists := commands[elements[1]]
		if (elements[0] == "@"+Me.Name) && exists {
			// Bot に対するメンションから始まり、かつコマンド名が次に来るならコマンドを実行

			if err = command.parseExecute(ms, elements[2]); err != nil {
				if onFail != nil {
					onFail(ms, command, err)
				} else {
					cp.CPrintf("[failed to run command '%s'] %s", elements[1], err)
				}
			}
		} else {
			// 登録コマンドの名前に一致するものがない、あるいはそもそも elements の初期の長さが 1 のとき
			// 単にメッセージとして受け取った時の関数を実行
			if onMessage != nil {
				onMessage(ms)
			}
		}
	})

	stampID = getAllStamps()
	log.Printf("[initialized bot]")
}

func Start() error {
	return Wsbot.Start()
}

func getAllStamps() map[string]string {
	resp, _, err := Wsbot.API().StampApi.GetStamps(context.Background()).Execute()
	if err != nil {
		cp.CPrintf("[failed to get stamps in GetAllStamps()] %s", err)
		return map[string]string{}
	}

	result := make(map[string]string)
	for _, stamp := range resp { // resp にはtraQ の全てのスタンプの情報が入っている
		result[stamp.Name] = stamp.Id
	}
	return result
}
