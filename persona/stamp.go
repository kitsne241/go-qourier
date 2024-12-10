package persona

import (
	"context"
	"log"

	"github.com/fatih/color"
	traq "github.com/traPtitech/go-traq"
)

// スタンプそのものというよりは、特定ユーザーによって投稿にスタンプが 1 つ以上つけられた『状態』を表す型
type Stamp struct {
	Name  string `json:"name"`
	ID    string `json:"id"`
	Count int    `json:"count"`
	User  *User  `json:"user"`
	bot   *Bot
}

func (bot *Bot) GetStamp(stID string) *Stamp {
	// 型の意味合いが多少異なるので Count, User はそれぞれ初期値 0, nil として返す
	// API にはアクセスせず、手持ちの連想配列（辞書）から UUID と名前の組を得て返却する
	// Bot の購読設定で STAMP_CREATED にチェックを入れていないと更新されないので、新しいスタンプは nil で返ることがある
	name, exists := stampIDName[stID]
	if exists {
		return &Stamp{Name: name, ID: stID}
	} else {
		return nil
	}
}

func (bot *Bot) NameGetStamp(name string) *Stamp {
	id, exists := stampNameID[name]
	if exists {
		return &Stamp{Name: name, ID: id}
	} else {
		return nil
	}
}

func (ms *Message) Stamp(stamps ...string) {
	if ms == nil {
		return
	}
	for _, stamp := range stamps {
		stampID, exists := stampNameID[stamp]
		if !exists {
			log.Println(color.HiYellowString("[failed to put stamp to post in Stamp(\"%s\")] stamp \"%s\" not found", stamp, stamp))
		}
		_, err := ms.bot.Wsbot.API().MessageApi.AddMessageStamp(context.Background(), ms.ID, stampID).
			PostMessageStampRequest(*traq.NewPostMessageStampRequestWithDefaults()).Execute()

		if err != nil {
			log.Println(color.HiYellowString(
				"[failed to put stamp to post in Stamp(\"%s\")] %s\nMessage: %s @%s \"%s\"", stamp, err, ms.CreatedAt, ms.Author, ms.Text,
			))
			// ユーザーやチャンネルと違いメッセージを一意に特定できる識別子は UUID しかないが、UUID そのものを表示させても…
		}
	}
}
