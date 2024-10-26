package persona

import (
	"context"

	cp "github.com/kitsne241/go-qourier/cprint"
)

type User struct {
	Nick  string // きつね
	Name  string // @kitsne
	ID    string // UUID
	IsBot bool
}

func GetUser(usID string) *User {
	resp, _, err := Wsbot.API().UserApi.GetUser(context.Background(), usID).Execute()
	if err != nil {
		cp.CPrintf("[failed to get user in GetUser(%d)] %s", usID, err)
		return nil
	}

	return &User{
		Nick:  resp.DisplayName,
		Name:  resp.Name,
		ID:    usID,
		IsBot: resp.Bot,
	}
}

// func GetUserFromName(name string) *User {
// 	// 工事中
// }

func GetMe() *User {
	resp, _, err := Wsbot.API().MeApi.GetMe(context.Background()).Execute()
	if err != nil {
		cp.CPrintf("[failed to get myself in GetMe()] %s", err) // すごい文面だ…
		return nil
	}

	return &User{
		Nick:  resp.DisplayName,
		Name:  resp.Name,
		ID:    resp.Id,
		IsBot: true,
	}
}
