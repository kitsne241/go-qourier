package persona

import (
	"context"
	"time"

	cp "github.com/kitsne241/go-qourier/cprint"
	traq "github.com/traPtitech/go-traq"
)

type Channel struct {
	Name   string
	Path   string // 例： "team/sound/1DTM"
	ID     string
	Parent *Channel
}

func GetChannel(chID string) *Channel {
	resp, _, err := Wsbot.API().ChannelApi.GetChannel(context.Background(), chID).Execute()
	if err != nil {
		cp.CPrintf("[failed to get channel in GetChannel(%s)] %s", chID, err)
		return nil
	}

	var parent *Channel
	var path string

	parentID := resp.ParentId.Get()
	if parentID != nil { // resp.ParentId.IsSet() は常に true のようなので…
		parent = GetChannel(*parentID) // 親チャンネルを得る
		if parent == nil {
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

// func GetChannelFromPath(path string) *Channel {
// 	// 工事中
// }

func (ch *Channel) GetChildren() []*Channel {
	if ch == nil {
		return []*Channel{}
	}
	resp, _, err := Wsbot.API().ChannelApi.GetChannel(context.Background(), ch.ID).Execute()
	if err != nil {
		cp.CPrintf("[failed to get children in GetChildren()] %s\nch = %v", err, ch)
		return []*Channel{}
	}

	children := []*Channel{}
	for _, child := range resp.Children {
		if ch := GetChannel(child); ch != nil {
			children = append(children, ch)
		}
	}
	return children
}

func (ch *Channel) GetRecentMessages(limit int) []*Message {
	if ch == nil {
		return []*Message{}
	}
	// resp, _, err := Wsbot.API().ChannelApi.GetMessages(context.Background(), ch.ID).Limit(limit).Execute()
	// if err != nil {
	// 	cp.CPrintf("[failed to get recent messages in GetRecentMessages(%d)] %s\nch = %v", limit, err, ch)
	// 	return []*Message{}
	// }

	respAll := make([]traq.Message, 3000) // 上限はとりあえず 3000 とする
	for i := 0; i*150 < limit; i++ {
		resp, _, err := Wsbot.API().ChannelApi.GetMessages(context.Background(), ch.ID).
			Limit(int32(150)).Offset(int32(150 * i)).Execute()
		if err != nil {
			cp.CPrintf("[failed to get recent messages in GetRecentMessages(%d)] %s\nch = %v", limit, err, ch)
			return []*Message{}
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
		cp.CPrintf("[failed to load location in GetRecentMessages(%d)] %s\nch = %v", limit, err)
		return nil
	}

	// 連続して API にアクセスすると失敗するので、こちらでは GetMessage は使っていない。書き換え時注意！

	userDic := map[string]*User{}
	// ユーザーの UUID と User 型との対応の辞書
	// 同じユーザーに対して何度も GetUser をするのは処理の無駄が激しく API の制限も受けやすいので、
	// 一時的に情報を保存の上再利用して制限を回避する

	messages := make([]*Message, len(respAll))
	for i, message := range respAll {
		_, exists := userDic[message.UserId]
		if !exists {
			user := GetUser(message.UserId)
			if user != nil {
				userDic[message.UserId] = user
			}
		}

		messages[i] = &Message{
			Channel:   ch,
			Text:      message.Content,
			ID:        message.Id,
			CreatedAt: message.CreatedAt.In(jst),
			UpdatedAt: message.UpdatedAt.In(jst),
			Author:    userDic[message.UserId],
		}
	}

	return messages
}

func (ch *Channel) Send(content string) {
	if ch == nil {
		return
	}
	_, _, err := Wsbot.API().MessageApi.PostMessage(context.Background(), ch.ID).
		PostMessageRequest(traq.PostMessageRequest{Content: content}).Execute()

	// traq-ws-bot を使わない場合、
	// apiClient := traq.NewAPIClient(traq.NewConfiguration())
	// _, _, err := apiClient.MessageApi.以下略

	if err != nil {
		cp.CPrintf("[failed to send message in Send()] %s\nch = %v", err, ch)
	}
}

func (ch *Channel) Join() {
	if ch == nil {
		return
	}
	_, err := Wsbot.API().BotApi.LetBotJoinChannel(context.Background(), Me.ID).
		PostBotActionJoinRequest(*traq.NewPostBotActionJoinRequest(ch.ID)).Execute()
	if err != nil {
		cp.CPrintf("[failed to join in Join()] %s\nch = %v", err, ch)
	}
}

func (ch *Channel) Leave() {
	if ch == nil {
		return
	}
	_, err := Wsbot.API().BotApi.LetBotLeaveChannel(context.Background(), Me.ID).
		PostBotActionLeaveRequest(*traq.NewPostBotActionLeaveRequest(ch.ID)).Execute()
	if err != nil {
		cp.CPrintf("[failed to leave in Leave()] %s\nch = %v", err, ch)
	}
}
