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

func (ch *Channel) GetRecentMessages(limit int32) ([]*Message, error) {
	resp, _, err := bot.Wsbot.API().ChannelApi.GetMessages(context.Background(), ch.ID).Limit(limit).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get recent messages: %w", err)
	}

	messages := []*Message{}
	for _, message := range resp {
		mes, err := GetMessage(message.Id)
		if err != nil {
			return nil, fmt.Errorf("failed to get recent messages: %w", err)
		}
		messages = append(messages, mes)
	}

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
