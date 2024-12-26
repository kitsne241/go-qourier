package persona

import (
	"context"
	"log"

	"github.com/fatih/color"
	traq "github.com/traPtitech/go-traq"
)

// traQ の投稿に対し、あるユーザーによって 1 つ以上つけられたスタンプを表す型
type Stamp struct {
	Name  string `json:"name"`  // "tada"
	ID    string `json:"id"`    // "8bfd4032-18d1-477f-894c-08855b46fd2f"
	Count int    `json:"count"` // 1
	User  *User  `json:"user"`
}

// 引数の UUID をもつスタンプを取得。Count と User は無意味な値
func GetStamp(stID string) *Stamp {
	resp, _, err := Wsbot.API().StampApi.GetStamp(context.Background(), stID).Execute()
	if err != nil {
		log.Println(color.HiYellowString("[failed to get stamp in GetStamp(%s)] %s", stID, err))
		return nil
	}
	return &Stamp{
		Name: resp.Name,
		ID:   stID,
	}
}

// 引数の名前をもつスタンプを取得。Count と User は無意味な値
func NameGetStamp(name string) *Stamp {
	stampNameID := allStamps.ID
	stID, exists := stampNameID[name]
	if exists {
		return &Stamp{Name: name, ID: stID}
	} else {
		return nil
	}
}

// メッセージに引数のスタンプを順番につける
func (ms *Message) Stamp(stamps ...string) {
	if ms == nil {
		return
	}
	stampNameID := allStamps.ID
	for _, stamp := range stamps {
		stID, exists := stampNameID[stamp]
		if !exists {
			log.Println(color.HiYellowString("[failed to put stamp to post in Stamp(\"%s\")] stamp \"%s\" not found", stamp, stamp))
		}
		_, err := Wsbot.API().MessageApi.AddMessageStamp(context.Background(), ms.ID, stID).
			PostMessageStampRequest(*traq.NewPostMessageStampRequestWithDefaults()).Execute()

		if err != nil {
			log.Println(color.HiYellowString(
				"[failed to put stamp to post in Stamp(\"%s\")] %s\nMessage: %s @%s \"%s\"", stamp, err, ms.CreatedAt, ms.Author, ms.Text,
			))
			// ユーザーやチャンネルと違いメッセージを一意に特定できる識別子は UUID しかないが、UUID そのものを表示させても…
		}
	}
}
