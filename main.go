package main

import (
	"fmt"

	prs "github.com/kitsne241/go-qourier/persona"
	srg "github.com/kitsne241/go-qourier/storage"
)

func main() {
	prs.SetUp(map[string]*prs.Command{
		"set": {Action: set, Syntax: "%s %d:%d"}, // @BOT_name set Sunday 21:00
		"get": {Action: get, Syntax: ""},         // @BOT_name get
	}, onMessage, nil, nil)

	srg.SetUp(nil) // データベースに接続

	prs.Start() // Bot を起動
}

type Date struct {
	Day  string `json:"day"`
	Hour int    `json:"hour"`
	Min  int    `json:"min"`
}

func set(ms *prs.Message, day string, hour int, min int) error {
	ms.Channel.Send(fmt.Sprintf("On %s %02d:%02d, right?", day, hour, min)) // ゼロ埋め
	srg.Save(Date{Day: day, Hour: hour, Min: min})
	ms.Stamp("done-nya") // 両側のコロンは入れずに
	return nil
}

func get(ms *prs.Message) error {
	var date Date
	srg.Load(&date)
	ms.Channel.Send(fmt.Sprintf("I remember it was on %s %02d:%02d!", date.Day, date.Hour, date.Min))
	return nil
}

func onMessage(ms *prs.Message) {
	ms.Channel.Send(fmt.Sprintf("Oisu! Here is #%s", ms.Channel.Path))
}
