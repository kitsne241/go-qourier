package persona

import (
	"context"
	"log"

	"github.com/fatih/color"
)

// traQ のユーザーを表現する型
type User struct {
	Nick  string `json:"nick"`  // "きつね"
	Name  string `json:"name"`  // "kitsne"
	ID    string `json:"id"`    // "a77f54f2-a7dc-4dab-ad6d-5c5df7e9ecfa"
	IsBot bool   `json:"isbot"` // false
}

// 引数の UUID をもつユーザーを取得
func GetUser(usID string) *User {
	resp, _, err := Wsbot.API().UserApi.GetUser(context.Background(), usID).Execute()
	if err != nil {
		log.Println(color.HiYellowString("[failed to get user in GetUser(%d)] %s", usID, err))
		return nil
	}

	return &User{
		Nick:  resp.DisplayName,
		Name:  resp.Name,
		ID:    usID,
		IsBot: resp.Bot,
	}
}

// 引数のユーザー名（traQ ID）をもつユーザーを取得
func NameGetUser(name string) *User {
	userNameID := getAllUsers().ID
	usID, exists := userNameID[name]
	if !exists {
		log.Println(color.HiYellowString("[failed to get user in NameGetUser(\"%s\")] not found such user", name))
		return nil
	}
	return GetUser(usID)
}

func getMe() *User {
	resp, _, err := Wsbot.API().MeApi.GetMe(context.Background()).Execute()
	if err != nil {
		log.Println(color.HiYellowString("[failed to get myself in GetMe()] %s", err)) // すごい文面だ…
		return nil
	}

	return &User{
		Nick:  resp.DisplayName,
		Name:  resp.Name,
		ID:    resp.Id,
		IsBot: true,
	}
}
