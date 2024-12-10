package persona

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/fatih/color"
	traq "github.com/traPtitech/go-traq"
)

type Channel struct {
	Name   string   `json:"name"`
	Path   string   `json:"path"` // 例： "team/sound/1DTM"
	ID     string   `json:"id"`
	Parent *Channel `json:"parent"`
	bot    *Bot
}

func (bot *Bot) GetChannel(chID string) *Channel {
	resp, _, err := bot.Wsbot.API().ChannelApi.GetChannel(context.Background(), chID).Execute()
	if err != nil {
		log.Println(color.HiYellowString("[failed to get channel in GetChannel(%s)] %s", chID, err))
		return nil
	}

	var parent *Channel
	var path string

	parentID := resp.ParentId.Get()
	if parentID != nil { // resp.ParentId.IsSet() は常に true のようなので…
		parent = bot.GetChannel(*parentID) // 親チャンネルを得る
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
		bot:    bot,
	}
}

func (bot *Bot) PathGetChannel(path string) *Channel {
	chID, exists := channelPathID[path]
	if !exists {
		log.Println(color.HiYellowString("[failed to get channel in PathGetChannel(\"%s\")] not found such channel", path))
		return nil
	}
	// チャンネルの path（"gps/times/kitsnegra" とか）から *Channel 型を得る
	return bot.GetChannel(chID)
}

func (ch *Channel) GetChildren() []*Channel {
	if ch == nil {
		return []*Channel{}
	}
	resp, _, err := ch.bot.Wsbot.API().ChannelApi.GetChannel(context.Background(), ch.ID).Execute()
	if err != nil {
		log.Println(color.HiYellowString("[failed to get children of #%s in GetChildren()] %s", ch.Path, err))
		return []*Channel{}
	}

	children := []*Channel{}
	for _, child := range resp.Children {
		if c := ch.bot.GetChannel(child); c != nil {
			children = append(children, c)
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
		resp, _, err := ch.bot.Wsbot.API().ChannelApi.GetMessages(context.Background(), ch.ID).
			Limit(int32(150)).Offset(int32(150 * i)).Execute()
		if err != nil {
			log.Println(color.HiYellowString(
				"[failed to get recent messages on #%s in GetRecentMessages(%d)] %s", ch.Path, limit, err,
			))
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
		log.Println(color.HiYellowString("[failed to load location in GetRecentMessages(%d)] %s", limit, err))
		return nil
	}

	// 連続して API にアクセスすると失敗するので、こちらでは GetMessage は使っていない。書き換え時注意！

	userDic := map[string]*User{}
	// ユーザーの UUID と User 型との対応の辞書
	// 同じユーザーに対して何度も GetUser をするのは処理の無駄が激しく API の制限も受けやすいので、
	// 一時的に情報を保存の上再利用して制限を回避する

	addUser := func(userId string) { // 与えられた UUID をもつユーザーがまだ userDic になければ追加する
		_, exists := userDic[userId]
		if !exists {
			user := ch.bot.GetUser(userId)
			if user != nil {
				userDic[userId] = user
			}
		}
	}

	messages := make([]*Message, len(respAll))
	for i, message := range respAll {
		addUser(message.UserId)

		stamps := []*Stamp{}
		for _, mstamp := range message.Stamps {
			addUser(mstamp.UserId)
			stamps = append(stamps, &Stamp{
				Name:  stampIDName[mstamp.StampId],
				ID:    mstamp.StampId,
				User:  userDic[mstamp.UserId],
				Count: int(mstamp.Count),
				bot:   ch.bot,
			})
		}

		messages[i] = &Message{
			Channel:   ch,
			Text:      message.Content,
			ID:        message.Id,
			CreatedAt: message.CreatedAt.In(jst),
			UpdatedAt: message.UpdatedAt.In(jst),
			Author:    userDic[message.UserId],
			Stamps:    stamps,
			bot:       ch.bot,
		}
	}

	return messages
}

func (ch *Channel) Send(content string) {
	if ch == nil {
		return
	}
	if content == "" {
		log.Println(color.HiYellowString("[failed to send message on #%s in Send()] message is empty", ch.Path))
		return // 空白のメッセージは 400 Bad Request で弾かれるが、原因究明の手間を省くためにエラーメッセージ付でここで弾いてしまう
	}
	_, _, err := ch.bot.Wsbot.API().MessageApi.PostMessage(context.Background(), ch.ID).
		PostMessageRequest(traq.PostMessageRequest{Content: content}).Execute()

	// traq-ws-bot を使わない場合、
	// apiClient := traq.NewAPIClient(traq.NewConfiguration())
	// _, _, err := apiClient.MessageApi.以下略

	if err != nil {
		log.Println(color.HiYellowString("[failed to send message on #%s in Send()] %s", ch.Path, err))
	}
}

func (ch *Channel) Join() {
	// ここでの Join は「このチャンネルにおける自身へのメンション以外の投稿イベントを購読する」こと
	// チャンネルへの投稿、チャンネルの直近の投稿の取得、メンションへの反応などはチャンネルに Join していなくても可能

	if ch == nil {
		return
	}
	// Bot のユーザーとしての ID と BOT_ID とは別もの

	_, err := ch.bot.Wsbot.API().BotApi.LetBotJoinChannel(context.Background(), os.Getenv("BOT_ID")).
		PostBotActionJoinRequest(*traq.NewPostBotActionJoinRequest(ch.ID)).Execute()
	if err != nil {
		log.Println(color.HiYellowString(
			"[failed to join into #%s in Join()] make sure BOT_ID is set!: %s", ch.Path, err,
		))
	}
}

func (ch *Channel) Leave() {
	if ch == nil {
		return
	}
	_, err := ch.bot.Wsbot.API().BotApi.LetBotLeaveChannel(context.Background(), os.Getenv("BOT_ID")).
		PostBotActionLeaveRequest(*traq.NewPostBotActionLeaveRequest(ch.ID)).Execute()
	if err != nil {
		log.Println(color.HiYellowString(
			"[failed to leave from #%s in Leave()] make sure BOT_ID is set!: %s", ch.Path, err,
		))
	}
}
