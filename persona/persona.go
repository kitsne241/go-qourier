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

var stampID = map[string]string{} // スタンプの名前と ID の対応の辞書

var Wsbot *traqwsbot.Bot

var Me *User

// main.go で使うサブパッケージの関数は全て大文字から始める。小文字スタートのままではインポートが失敗する

func init() {
	godotenv.Load(".env")
}

func SetUp(commands map[string]*Command, onMessage func(*Message), onFail func(*Message, *Command, error)) error {
	// onMessage : 受け取ったメッセージがコマンドでない場合に呼ばれる関数
	// onFail    : 何らかの原因でコマンドの実行が失敗したときに呼ばれる関数

	for name, command := range commands {
		command.name = name
		command.action = varadic(command)
	}
	// Command 型の配列である引数 commands から {関数名: 実行関数} の辞書 commandsDic を得る
	// この際 varadic の内部で関数の構造が条件に適合しているかの審査を同時に行い、不適正なら panic する

	var err error
	Wsbot, err = traqwsbot.NewBot(&traqwsbot.Options{ // Bot を作成
		AccessToken: os.Getenv("ACCESS_TOKEN"),
	})
	if err != nil {
		panic(fmt.Sprintf("failed to initialize bot: %v", err))
	}

	Me, err = GetMe()
	if err != nil {
		panic(err)
	}

	Wsbot.OnMessageCreated(func(p *payload.MessageCreated) {
		mention := fmt.Sprintf("!{\"type\":\"user\",\"raw\":\"@%s\",\"id\":\"%s\"}", Me.Name, Me.ID)
		// メッセージ本文などではメンションは JSON 形式の文字列に置き換えられている

		ms, err := GetMessage(p.Message.ID)
		if err != nil {
			log.Printf("failed to react to message: %s", err)
			return
		}

		content := strings.Replace(ms.Text, mention, "@"+Me.Name, 1)
		elements := strings.SplitN(content, " ", 3)
		// 最初の 2 つの半角スペースを見つけて最大 3 つに切り分ける。@BOT_name / コマンド / 引数

		if len(elements) == 1 { // elements の長さが 1 なら少なくともコマンドではないのでメッセージとして処理
			if onMessage != nil {
				onMessage(ms)
			}
		} else {
			if len(elements) == 2 {
				elements = append(elements, "") // elements の長さが常に 3 になるように規格化
			}
			command, exists := commands[elements[1]]
			if (elements[0] == "@"+Me.Name) && exists {
				// Bot に対するメンションから始まり、かつコマンド名が次に来るならコマンドを実行

				if err = command.parseExecute(ms, elements[2]); err != nil {
					errMessage := fmt.Errorf("failed to run command: %s", err)
					if onFail != nil {
						onFail(ms, command, errMessage)
					} else {
						log.Print(errMessage)
					}
				}
			} else { // 登録コマンドの名前に一致するものがなければ、単にメッセージとして受け取った時の関数を実行
				if onMessage != nil {
					onMessage(ms)
				}
			}
		}
	})

	getAllStamps := func() (map[string]string, error) {
		resp, _, err := Wsbot.API().StampApi.GetStamps(context.Background()).Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to get stamps: %w", err)
		}

		result := make(map[string]string)
		for _, stamp := range resp { // resp にはtraQ の全てのスタンプの情報が入っている
			result[stamp.Name] = stamp.Id
		}

		return result, nil
	}

	if stampID, err = getAllStamps(); err != nil {
		return err
	}

	log.Printf("initialized bot")
	return nil
}

func Start() error {
	return Wsbot.Start()
}
