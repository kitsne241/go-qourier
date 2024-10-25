package persona

import (
	"context"
	"time"

	traq "github.com/traPtitech/go-traq"
)

type Message struct {
	Channel   *Channel
	Text      string
	ID        string
	CreatedAt time.Time // JST
	UpdatedAt time.Time // JST
}

// 基本的に error は出さずに nil あるいは空の配列として伝える方針
// 適切な引数による実行の上で API との接続で問題が生じた場合はエラーメッセージがエラーの原因に直接結びつかない気がするため

func GetMessage(chID string) *Message {
	resp, _, err := bot.Wsbot.API().MessageApi.GetMessage(context.Background(), chID).Execute()
	if err != nil {
		CPrintf("[failed to get message in GetMessage(%s)] %s", chID, err)
		return nil
	}

	ch := GetChannel(resp.ChannelId)
	if ch == nil {
		return nil
	}

	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		CPrintf("[failed to load location in GetMessage(%s)] %s", chID, err)
		return nil
	}

	return &Message{
		Channel:   ch,
		Text:      resp.Content,
		ID:        resp.Id,
		CreatedAt: resp.CreatedAt.In(jst),
		UpdatedAt: resp.UpdatedAt.In(jst),
	}
}

func (ms *Message) Stamp(stamp string) {
	_, err := bot.Wsbot.API().
		MessageApi.AddMessageStamp(context.Background(), ms.ID, stampID[stamp]).
		PostMessageStampRequest(*traq.NewPostMessageStampRequestWithDefaults()).Execute()

	if err != nil {
		CPrintf("[failed to put stamp in Stamp()] %s\nms = %v", err, ms)
	}
}
