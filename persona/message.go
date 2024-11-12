package persona

import (
	"context"
	"encoding/json"
	"log"
	"slices"
	"time"

	"github.com/fatih/color"
	traq "github.com/traPtitech/go-traq"
)

type Message struct {
	Channel   *Channel
	Text      string
	ID        string
	CreatedAt time.Time // JST
	UpdatedAt time.Time // JST
	Author    *User
}

// 基本的に error は出さずに異常ログのみ、呼び出し元には nil あるいは空の配列として伝える方針
// 適切な引数による実行の上で API との接続で問題が生じた場合はエラーメッセージがエラーの原因に直接結びつかない気がするため

func GetMessage(msID string) *Message {
	resp, _, err := Wsbot.API().MessageApi.GetMessage(context.Background(), msID).Execute()
	if err != nil {
		log.Println(color.HiYellowString("[failed to get message in GetMessage(%s)] %s", msID, err))
		return nil
	}

	ch := GetChannel(resp.ChannelId)
	if ch == nil {
		return nil
	}

	user := GetUser(resp.UserId)
	if user == nil {
		return nil
	}

	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		log.Println(color.HiYellowString("[failed to load location in GetMessage(%s)] %s", msID, err))
		return nil
	}

	return &Message{
		Channel:   ch,
		Text:      resp.Content,
		ID:        resp.Id,
		CreatedAt: resp.CreatedAt.In(jst),
		UpdatedAt: resp.UpdatedAt.In(jst),
		Author:    user,
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
		_, err := Wsbot.API().MessageApi.AddMessageStamp(context.Background(), ms.ID, stampID).
			PostMessageStampRequest(*traq.NewPostMessageStampRequestWithDefaults()).Execute()

		if err != nil {
			log.Println(color.HiYellowString(
				"[failed to put stamp to post in Stamp(\"%s\")] %s\nMessage: %s @%s \"%s\"", stamp, err, ms.CreatedAt, ms.Author, ms.Text,
			))
			// ユーザーやチャンネルと違いメッセージを一意に特定できる識別子は UUID しかないが、UUID そのものを表示させても…
		}
	}
}

// traQ 内部で使われている Unembedder と概ね同じロジック（TypeScript で書かれている）を Go で再実装
// traq-ws-bot でメッセージイベントを受け取った時には PlainText を取得できるが、go-traq の API としては提供されていない
// https://github.com/traPtitech/traQ_S-UI/blob/master/src/lib/markdown/internalLinkUnembedder.ts
// 基本的に埋め込みは type, raw, id の 3 つのキーのみから構成される JSON 文字列 !{ ... } である

type EmbedData struct {
	Type  string `json:"type"`
	Raw   string `json:"raw"`
	Id    string `json:"id"`
	Start int    // 埋め込みの開始位置
	End   int    // 埋め込みの終了位置
}

func Unembed(text string) string {
	textRune := []rune(text)
	inEmbed := false

	embedData := []EmbedData{}
	data := EmbedData{}

	for i := 0; i < len(textRune); i++ {
		if inEmbed {
			if textRune[i] == '}' {
				inEmbed = false
				data.End = i + 1
				err := json.Unmarshal([]byte(string(textRune[data.Start+1:i+1])), &data)
				if err == nil {
					embedData = append(embedData, data)
				}
			}
		} else {
			if (i < len(textRune)-1) && textRune[i] == '!' && textRune[i+1] == '{' {
				log.Println(textRune[i], textRune[i+1])
				inEmbed = true
				data = EmbedData{Start: i}
			}
		}
	}

	slices.Reverse(embedData)
	// 得られた embedData を後ろから順に置き換えて埋め込みを解消する

	for _, data := range embedData {
		tempRune := append([]rune(data.Raw), textRune[data.End:]...)
		textRune = append(textRune[:data.Start], tempRune...)
	}

	return string(textRune)
}
