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
	Channel   *Channel  `json:"channel"`
	Text      string    `json:"text"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdat"` // JST
	UpdatedAt time.Time `json:"updatedat"` // JST
	Author    *User     `json:"author"`
	Stamps    []*Stamp  `json:"stamps"`
	bot       *Bot
}

// 特定ユーザーからの特定スタンプ
type Stamp struct {
	Name  string `json:"name"`
	ID    string `json:"id"`
	Count int    `json:"count"`
	User  *User  `json:"user"`
	bot   *Bot
}

// 基本的に error は出さずに異常ログのみ、呼び出し元には nil あるいは空の配列として伝える方針
// 適切な引数による実行の上で API との接続で問題が生じた場合はエラーメッセージがエラーの原因に直接結びつかない気がするため

func (bot *Bot) GetMessage(msID string) *Message {
	resp, _, err := bot.Wsbot.API().MessageApi.GetMessage(context.Background(), msID).Execute()
	if err != nil {
		log.Println(color.HiYellowString("[failed to get message in GetMessage(%s)] %s", msID, err))
		return nil
	}

	ch := bot.GetChannel(resp.ChannelId)
	if ch == nil {
		return nil
	}

	user := bot.GetUser(resp.UserId)
	if user == nil {
		return nil
	}

	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		log.Println(color.HiYellowString("[failed to load location in GetMessage(%s)] %s", msID, err))
		return nil
	}

	userDic := map[string]*User{}
	addUser := func(userId string) { // 与えられた UUID をもつユーザーがまだ userDic になければ追加する
		_, exists := userDic[userId]
		if !exists {
			user := bot.GetUser(userId)
			if user != nil {
				userDic[userId] = user
			}
		}
	}

	stamps := []*Stamp{}
	for _, mstamp := range resp.Stamps {
		addUser(mstamp.UserId)
		stamps = append(stamps, &Stamp{
			Name:  stampIDName[mstamp.StampId],
			ID:    mstamp.StampId,
			User:  userDic[mstamp.UserId],
			Count: int(mstamp.Count),
			bot:   bot,
		})
	}

	return &Message{
		Channel:   ch,
		Text:      resp.Content,
		ID:        resp.Id,
		CreatedAt: resp.CreatedAt.In(jst),
		UpdatedAt: resp.UpdatedAt.In(jst),
		Author:    user,
		Stamps:    stamps,
		bot:       bot,
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

// traQ 内部で使われている Unembedder と概ね同じロジック（TypeScript で書かれている）を Go で再実装
// traq-ws-bot でメッセージイベントを受け取った時には PlainText を取得できるが、go-traq の API としては提供されていない
// https://github.com/traPtitech/traQ_S-UI/blob/master/src/lib/markdown/internalLinkUnembedder.ts
// 基本的に埋め込みは type, raw, id の 3 つのキーのみから構成される JSON 文字列 !{ ... } である

type Embed struct {
	Type  string `json:"type"`
	Raw   string `json:"raw"`
	ID    string `json:"id"`
	Start int    `json:"start"` // 埋め込みの開始位置
	End   int    `json:"end"`   // 埋め込みの終了位置
}

func Unembed(text string) (string, []Embed) {
	textRune := []rune(text)
	inEmbed := false

	embeds := []Embed{}
	data := Embed{}

	for i := 0; i < len(textRune); i++ {
		if inEmbed {
			if textRune[i] == '}' {
				inEmbed = false
				data.End = i + 1
				err := json.Unmarshal([]byte(string(textRune[data.Start+1:i+1])), &data)
				if err == nil {
					embeds = append(embeds, data)
				}
			}
		} else {
			if (i < len(textRune)-1) && textRune[i] == '!' && textRune[i+1] == '{' {
				inEmbed = true
				data = Embed{Start: i}
			}
		}
	}

	slices.Reverse(embeds)
	// 得られた embed を後ろから順に置き換えて埋め込みを解消する

	for _, data := range embeds {
		tempRune := append([]rune(data.Raw), textRune[data.End:]...)
		textRune = append(textRune[:data.Start], tempRune...)
	}

	slices.Reverse(embeds)
	return string(textRune), embeds
}
