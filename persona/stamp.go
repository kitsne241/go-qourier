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
	// getAllStamps を使う意味がないので素直に API にアクセスしてスタンプの情報を得る
	// 型の意味合いが多少異なるので Count, User はそれぞれ初期値 0, nil として返す
	resp, _, err := bot.Wsbot.API().StampApi.GetStamp(context.Background(), stID).Execute()
	if err != nil {
		log.Println(color.HiYellowString("[failed to get stamp in GetStamp(%s)] %s", stID, err))
		return nil
	}
	return &Stamp{
		Name: resp.Name,
		ID:   stID,
	}
}

func (bot *Bot) NameGetStamp(name string) *Stamp {
	// NameGetUser と違い getAllStamps できれば欲しい情報は全て集まるので GetStamp は呼ばない
	stampNameID := bot.getAllStamps().ID
	stID, exists := stampNameID[name]
	if exists {
		return &Stamp{Name: name, ID: stID}
	} else {
		return nil
	}
}

func (ms *Message) Stamp(stamps ...string) {
	if ms == nil {
		return
	}
	stampNameID := ms.bot.getAllStamps().ID
	for _, stamp := range stamps {
		stID, exists := stampNameID[stamp]
		if !exists {
			log.Println(color.HiYellowString("[failed to put stamp to post in Stamp(\"%s\")] stamp \"%s\" not found", stamp, stamp))
		}
		_, err := ms.bot.Wsbot.API().MessageApi.AddMessageStamp(context.Background(), ms.ID, stID).
			PostMessageStampRequest(*traq.NewPostMessageStampRequestWithDefaults()).Execute()

		if err != nil {
			log.Println(color.HiYellowString(
				"[failed to put stamp to post in Stamp(\"%s\")] %s\nMessage: %s @%s \"%s\"", stamp, err, ms.CreatedAt, ms.Author, ms.Text,
			))
			// ユーザーやチャンネルと違いメッセージを一意に特定できる識別子は UUID しかないが、UUID そのものを表示させても…
		}
	}
}
