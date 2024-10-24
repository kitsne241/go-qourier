package persona

import (
	"context"
	"fmt"
	"log"
	"time"

	traq "github.com/traPtitech/go-traq"
	traqwsbot "github.com/traPtitech/traq-ws-bot"
)

type Bot struct {
	Wsbot *traqwsbot.Bot
	ID    string
	Name  string
}

type Message struct {
	Channel   *Channel
	Text      string
	ID        string
	CreatedAt time.Time // JST
	UpdatedAt time.Time // JST
}

type Channel struct {
	Name   string
	Path   string // 例： "team/sound/1DTM"
	ID     string
	Parent *Channel
}

func GetChannel(chID string) (*Channel, error) {
	resp, _, err := bot.Wsbot.API().ChannelApi.GetChannel(context.Background(), chID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	var parent *Channel
	var path string

	parentID := resp.ParentId.Get()
	if parentID != nil { // resp.ParentId.IsSet() は常に true のようなので…
		parent, err = GetChannel(*parentID) // 親チャンネルを得る
		if err != nil {
			return nil, fmt.Errorf("failed to get channel: %w", err)
		}
		path = parent.Path + "/" + resp.Name
	} else {
		path = resp.Name
	}

	return &Channel{
		Name:   resp.Name,
		Path:   path,
		ID:     chID,
		Parent: parent,
	}, nil
}

func GetMessage(chID string) (*Message, error) {
	resp, _, err := bot.Wsbot.API().MessageApi.GetMessage(context.Background(), chID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	ch, err := GetChannel(resp.ChannelId)
	if err != nil {
		return nil, err
	}

	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return nil, fmt.Errorf("failed to load location: %w", err)
	}

	return &Message{
		Channel:   ch,
		Text:      resp.Content,
		ID:        resp.Id,
		CreatedAt: resp.CreatedAt.In(jst),
		UpdatedAt: resp.UpdatedAt.In(jst),
	}, nil
}

func (ch *Channel) GetChildren() ([]*Channel, error) {
	resp, _, err := bot.Wsbot.API().ChannelApi.GetChannel(context.Background(), ch.ID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get children: %w", err)
	}

	children := []*Channel{}
	for _, child := range resp.Children {
		ch, err := GetChannel(child)
		if err != nil {
			return nil, fmt.Errorf("failed to get children: %w", err)
		}
		children = append(children, ch)
	}

	return children, nil
}

func (ch *Channel) GetRecentMessages(limit int) ([]*Message, error) {
	respAll := make([]traq.Message, 3000) // 上限はとりあえず 3000 とする

	for i := 0; i*150 < limit; i++ {
		resp, _, err := bot.Wsbot.API().ChannelApi.GetMessages(context.Background(), ch.ID).Limit(int32(150)).Offset(int32(150 * i)).Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to get recent messages: %w", err)
		}

		for j, res := range resp {
			respAll[i*150+j] = res
		}

		if len(resp) < 150 {
			respAll = respAll[:i*150+len(resp)] // 取りうる数の上限で respAll の長さを再規定
			break
		}
	}
	if len(respAll) > limit {
		respAll = respAll[:limit] // ユーザーが指定した limit で respAll の長さを再規定
	}

	// もともとの ChannelApi.GetMessages の仕様として、
	// 一度に 200 以上メッセージを読み込もうとすると失敗して 400 Bad Request が返るので 150 刻みに取得するように設計

	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return nil, fmt.Errorf("failed to load location: %w", err)
	}

	messages := make([]*Message, len(respAll))
	for i, message := range respAll {
		messages[i] = &Message{
			Channel:   ch,
			Text:      message.Content,
			ID:        message.Id,
			CreatedAt: message.CreatedAt.In(jst),
			UpdatedAt: message.UpdatedAt.In(jst),
		}
	}

	// ちょうど limit 個分、あるいは取れる分だけのメッセージの配列を返す
	return messages, nil
}

func (ch *Channel) Send(content string) {
	_, _, err := bot.Wsbot.API().
		MessageApi.
		PostMessage(context.Background(), ch.ID).
		PostMessageRequest(traq.PostMessageRequest{
			Content: content,
		}).
		Execute()

	// WebSocket を使わない場合、
	// apiClient := traq.NewAPIClient(traq.NewConfiguration())
	// _, _, err := apiClient.MessageApi.以下略

	if err != nil {
		log.Printf("failed to send message: %s", err)
		// 送信ができなくても大元のシステムに影響はないので、ログを出すのみで return はしないことにする
	}
}

func (ms *Message) Stamp(stamp string) {
	_, err := bot.Wsbot.API().
		MessageApi.AddMessageStamp(context.Background(), ms.ID, stampID[stamp]).
		PostMessageStampRequest(*traq.NewPostMessageStampRequestWithDefaults()).Execute()

	if err != nil {
		log.Printf("failed to put stamp: %s", err)
	}
}

func (ch *Channel) Join() {
	_, err := bot.Wsbot.API().BotApi.LetBotJoinChannel(context.Background(), bot.ID).
		PostBotActionJoinRequest(*traq.NewPostBotActionJoinRequest(ch.ID)).Execute()
	if err != nil {
		log.Printf("failed to join: %s", err)
	}

	_, _, err = bot.Wsbot.API().ChannelApi.GetChannel(context.Background(), ch.ID).Execute()
	if err != nil {
		log.Printf("failed to join: %s", err)
	}
}

func (ch *Channel) Leave() {
	_, err := bot.Wsbot.API().BotApi.LetBotLeaveChannel(context.Background(), bot.ID).
		PostBotActionLeaveRequest(*traq.NewPostBotActionLeaveRequest(ch.ID)).Execute()
	if err != nil {
		log.Printf("failed to leave: %s", err)
	}
	_, _, err = bot.Wsbot.API().ChannelApi.GetChannel(context.Background(), ch.ID).Execute()
	if err != nil {
		log.Printf("failed to leave: %s", err)
	}
}
