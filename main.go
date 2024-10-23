package main

import (
	"fmt"

	crr "github.com/kitsne241/go-qourier/persona"
	srg "github.com/kitsne241/go-qourier/storage"
)

func main() {
	crr.SetUp(map[string]*crr.Command{
		"set": {Action: set, Syntax: "%s %d:%d"}, // @BOT_name set Sunday 21:00
		"get": {Action: get, Syntax: ""},         // @BOT_name get
	}, onMessage, nil)

	srg.SetUp() // データベースに接続

	crr.Start() // Bot を起動
}

type Date struct {
	Day  string `json:"day"`
	Hour int    `json:"hour"`
	Min  int    `json:"min"`
}

func set(ms *crr.Message, day string, hour int, min int) {
	ms.Channel.Send(fmt.Sprintf("On %s %02d:%02d, right?", day, hour, min)) // ゼロ埋め
	srg.Save(Date{Day: day, Hour: hour, Min: min})
	ms.Stamp("done-nya") // 両側のコロンは入れずに
}

func get(ms *crr.Message) {
	var date Date
	srg.Load(&date)
	ms.Channel.Send(fmt.Sprintf("I remember it was on %s %02d:%02d!", date.Day, date.Hour, date.Min))
}

func onMessage(ms *crr.Message) {
	ms.Channel.Send(fmt.Sprintf("Oisu! Here is #%s", ms.Channel.Path))
}
