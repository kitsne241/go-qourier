package persona

// traQ における Bot 自身の動作の関数
// go-traq をさらに機能を絞って discord.py 風にラップしたもの

import (
	"context"
	"fmt"
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

type Commands map[string]*Command

var Wsbot *traqwsbot.Bot

var Me *User

var stampNameID = map[string]string{}   // "tada" -> "8bfd4032-18d1-477f-894c-08855b46fd2f"
var userNameID = map[string]string{}    // "kitsne" -> "a77f54f2-a7dc-4dab-ad6d-5c5df7e9ecfa"
var channelPathID = map[string]string{} // "gps/times/kitsnegra" -> "019275db-f2fd-7922-81c9-956aab18612d"

// main.go で使うサブパッケージの関数は全て大文字から始める。小文字スタートのままではインポートが失敗する

func init() {
	godotenv.Load(".env")
}

func SetUp(
	commands Commands,
	onMessage func(*Message),
	onFail func(*Message, *Command, error),
) {
	// onMessage : 受け取ったメッセージがコマンドでない場合に呼ばれる関数
	// onFail    : 何らかの原因でコマンドの実行が失敗したときに呼ばれる関数

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

	Wsbot, err = traqwsbot.NewBot(&traqwsbot.Options{ // Bot を作成
		AccessToken: os.Getenv("ACCESS_TOKEN"),
	})
	if err != nil {
		panic(color.HiRedString("[failed to create a new bot] %s", err))
	}

	if Me = getMe(); Me == nil {
		panic(color.HiRedString("[failed to build a bot] make sure ACCESS_TOKEN is set!"))
	}

	mention := fmt.Sprintf(`!{"type":"user","raw":"@%s","id":"%s"}`, Me.Name, Me.ID)
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
					log.Println(color.HiYellowString("[failed to run command '%s'] %s", elements[1], err))
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

	Wsbot.OnPing(func(p *payload.Ping) {
		getAllStamps()
		getAllUsers()
		getAllChannels()

		log.Println(color.GreenString("[initialized bot]"))
	})
	// 定期的に呼ばれる Ping で Bot のリフレッシュをしたり

	getAllStamps()
	getAllUsers()
	getAllChannels()

	log.Println(color.GreenString("[initialized bot]"))
}

func Start() error {
	if Wsbot == nil {
		panic(color.HiRedString("[bot is not set up]"))
	}
	return Wsbot.Start()
}

func getAllStamps() {
	stamps, _, err := Wsbot.API().StampApi.GetStamps(context.Background()).Execute()
	if err != nil {
		log.Println(color.HiYellowString("[failed to get stamps in getAllStamps()] %s", err))
	}

	stampNameID = map[string]string{}
	for _, stamp := range stamps { // resp にはtraQ の全てのスタンプの情報が入っている
		stampNameID[stamp.Name] = stamp.Id
	}
}

func getAllUsers() {
	users, _, err := Wsbot.API().UserApi.GetUsers(context.Background()).IncludeSuspended(true).Execute()
	if err != nil {
		log.Println(color.HiYellowString("[failed to get users in getAllUsers()] %s", err))
	}

	userNameID = map[string]string{}
	for _, user := range users { // resp にはtraQ の全てのスタンプの情報が入っている
		userNameID[user.Name] = user.Id
	}
}

func getAllChannels() {
	// 一度に何百回も API にアクセスするとエラーを生じがちなので
	// たった一度の API アクセスからチャンネルの path と ID の対応表を作りたい
	// GetChannels によって全てのパブリックチャンネルについて チャンネルのID・親チャンネルのID・チャンネルの名前 の 3 つが分かるので、
	// 親子の関連付けからチャンネルの親子関係のグラフを作成し、それぞれのチャンネルの名前を末尾まで継承してパスを作る

	channels, _, err := Wsbot.API().ChannelApi.GetChannels(context.Background()).IncludeDm(false).Execute()
	if err != nil {
		log.Println(color.HiYellowString("[failed to get channels in getAllChannels()] %s", err))
	}

	tree := map[string]string{}
	channelIDName := map[string]string{}
	channelPathID = map[string]string{}

	for _, channel := range channels.Public { // resp にはtraQ の全てのスタンプの情報が入っている
		channelIDName[channel.Id] = channel.Name
		parentID := channel.ParentId.Get()
		if parentID != nil {
			tree[channel.Id] = *parentID
		}
	}

	for _, channel := range channels.Public {
		currentID := channel.Id
		path := channelIDName[currentID]
		for {
			var exists bool
			currentID, exists = tree[currentID]
			if !exists {
				break
			}
			path = channelIDName[currentID] + "/" + path
		}
		channelPathID[path] = channel.Id
	}
}
