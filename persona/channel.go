package persona

import (
	"context"

	traq "github.com/traPtitech/go-traq"
)

type Channel struct {
	Name   string
	Path   string // 例： "team/sound/1DTM"
	ID     string
	Parent *Channel
}

func GetChannel(chID string) *Channel {
	resp, _, err := bot.Wsbot.API().ChannelApi.GetChannel(context.Background(), chID).Execute()
	if err != nil {
		CPrintf("[failed to get channel in GetChannel(%s)] %s", chID, err)
		return nil
	}

	var parent *Channel
	var path string

	parentID := resp.ParentId.Get()
	if parentID != nil { // resp.ParentId.IsSet() は常に true のようなので…
		parent = GetChannel(*parentID) // 親チャンネルを得る
		if parent != nil {
			return nil
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
	}
}

func (ch *Channel) GetChildren() []*Channel {
	if ch == nil {
		return []*Channel{}
	}
	resp, _, err := bot.Wsbot.API().ChannelApi.GetChannel(context.Background(), ch.ID).Execute()
	if err != nil {
		CPrintf("[failed to get children in GetChildren()] %s\nch = %v", err, ch)
		return []*Channel{}
	}

	children := []*Channel{}
	for _, child := range resp.Children {
		ch := GetChannel(child)
		if ch != nil {
			children = append(children, ch)
		}
	}
	return children
}

func (ch *Channel) GetRecentMessages(limit int32) []*Message {
	resp, _, err := bot.Wsbot.API().ChannelApi.GetMessages(context.Background(), ch.ID).Limit(limit).Execute()
	if err != nil {
		CPrintf("[failed to get recent messages in GetRecentMessages(%d)] %s\nch = %v", limit, err, ch)
		return nil
	}

	messages := []*Message{}
	for _, message := range resp {
		mes := GetMessage(message.Id)
		if mes != nil {
			messages = append(messages, mes)
		}
	}
	return messages
}

func (ch *Channel) Send(content string) {
	_, _, err := bot.Wsbot.API().
		MessageApi.PostMessage(context.Background(), ch.ID).
		PostMessageRequest(traq.PostMessageRequest{
			Content: content,
		}).Execute()

	// WebSocket を使わない場合、
	// apiClient := traq.NewAPIClient(traq.NewConfiguration())
	// _, _, err := apiClient.MessageApi.以下略

	if err != nil {
		CPrintf("[failed to send message in Send()] %s\nch = %v", err, ch)
	}
}

func (ch *Channel) Join() {
	_, err := bot.Wsbot.API().BotApi.LetBotJoinChannel(context.Background(), bot.ID).
		PostBotActionJoinRequest(*traq.NewPostBotActionJoinRequest(ch.ID)).Execute()
	if err != nil {
		CPrintf("[failed to join in Join()] %s\nch = %v", err, ch)
	}
}

func (ch *Channel) Leave() {
	_, err := bot.Wsbot.API().BotApi.LetBotLeaveChannel(context.Background(), bot.ID).
		PostBotActionLeaveRequest(*traq.NewPostBotActionLeaveRequest(ch.ID)).Execute()
	if err != nil {
		CPrintf("[failed to leave in Leave()] %s\nch = %v", err, ch)
	}
}
