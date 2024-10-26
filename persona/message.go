package persona

import (
	"context"
	"time"

	cp "github.com/kitsne241/go-qourier/cprint"
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
		cp.CPrintf("[failed to get message in GetMessage(%s)] %s", msID, err)
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
		cp.CPrintf("[failed to load location in GetMessage(%s)] %s", msID, err)
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

func (ms *Message) Stamp(stamp string) {
	if ms == nil {
		return
	}
	_, err := Wsbot.API().MessageApi.AddMessageStamp(context.Background(), ms.ID, stampID[stamp]).
		PostMessageStampRequest(*traq.NewPostMessageStampRequestWithDefaults()).Execute()

	if err != nil {
		cp.CPrintf("[failed to put stamp in Stamp()] %s\nms = %v", err, ms)
	}
}
