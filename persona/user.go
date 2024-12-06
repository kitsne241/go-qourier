package persona

import (
	"context"
	"log"

	"github.com/fatih/color"
)

type User struct {
	Nick  string `json:"nick"` // きつね
	Name  string `json:"name"` // kitsne
	ID    string `json:"id"`   // UUID
	IsBot bool   `json:"isbot"`
}

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

func NameGetUser(name string) *User {
	// ユーザー名（"kitsne" とか）から *User 型を得る
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
