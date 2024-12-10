package persona

import (
	"context"
	"log"

	"github.com/fatih/color"
)

type bimap struct {
	ID     map[string]string
	Symbol map[string]string
	// 一意に定まるので両方とも Identifier といえばそうだけど、ここでは ID とは UUID のことにする
}

func (bot *Bot) getAllStamps() bimap {
	stamps, _, err := bot.Wsbot.API().StampApi.GetStamps(context.Background()).Execute()
	if err != nil {
		log.Println(color.HiYellowString("[failed to get stamps in getAllStamps()] %s", err))
	}

	stampNameID := map[string]string{}
	stampIDName := map[string]string{}
	for _, stamp := range stamps { // resp にはtraQ の全てのスタンプの情報が入っている
		stampIDName[stamp.Id] = stamp.Name
		stampNameID[stamp.Name] = stamp.Id
	}
	return bimap{stampNameID, stampIDName}
}

func (bot *Bot) getAllUsers() bimap {
	users, _, err := bot.Wsbot.API().UserApi.GetUsers(context.Background()).IncludeSuspended(true).Execute()
	if err != nil {
		log.Println(color.HiYellowString("[failed to get users in getAllUsers()] %s", err))
	}

	userNameID := map[string]string{}
	userIDName := map[string]string{}
	for _, user := range users {
		userNameID[user.Name] = user.Id
		userIDName[user.Id] = user.Name
	}
	return bimap{userNameID, userIDName}
}

func (bot *Bot) getAllChannels() bimap {
	// 一度に何百回も API にアクセスするとエラーを生じがちなので
	// たった一度の API アクセスからチャンネルの path と ID の対応表を作りたい
	// GetChannels によって全てのパブリックチャンネルについて チャンネルのID・親チャンネルのID・チャンネルの名前 の 3 つが分かるので、
	// 親子の関連付けからチャンネルの親子関係のグラフを作成し、それぞれのチャンネルの名前を末尾まで継承してパスを作る

	channels, _, err := bot.Wsbot.API().ChannelApi.GetChannels(context.Background()).IncludeDm(false).Execute()
	if err != nil {
		log.Println(color.HiYellowString("[failed to get channels in getAllChannels()] %s", err))
	}

	idTree := map[string]string{} // 子チャンネルの UUID をキー、親チャンネルの UUID を値にもつ
	channelIDName := map[string]string{}
	channelPathID := map[string]string{}
	channelIDPath := map[string]string{}

	for _, channel := range channels.Public { // resp にはtraQ の全てのスタンプの情報が入っている
		channelIDName[channel.Id] = channel.Name
		parentID := channel.ParentId.Get()
		if parentID != nil {
			idTree[channel.Id] = *parentID
		}
	}
	// UUID の木構造と、それぞれの UUID をもつチャンネルの名称を取得。この木からそれぞれのパスを作る

	for _, channel := range channels.Public {
		currentID := channel.Id
		path := channelIDName[currentID]
		for {
			currentID, exists := idTree[currentID]
			if !exists {
				break
			}
			path = channelIDName[currentID] + "/" + path
		}
		channelPathID[path] = channel.Id
		channelIDPath[channel.Id] = path
	}
	return bimap{channelPathID, channelIDPath}
}
