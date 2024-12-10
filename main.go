package main

import (
	"fmt"

	cps "github.com/kitsne241/go-qourier/capsule"
	prs "github.com/kitsne241/go-qourier/persona"
)

type Date struct {
	Day  string `json:"day"`
	Hour int    `json:"hour"`
	Min  int    `json:"min"`
}

func main() {
	prs.SetUp(prs.Commands{
		"set": {Action: set, Syntax: "%s %d:%d"}, // @BOT_name set Sunday 21:00
		"get": {Action: get, Syntax: ""},         // @BOT_name get
	})

	prs.OnMessage = func(ms *prs.Message) {
		ms.Channel.Send(fmt.Sprintf("Oisu! Here is #%s", ms.Channel.Path))
	}

	cps.SetUp(Date{Day: "Sunday", Hour: 12, Min: 0}, false) // データベースに接続・必要に応じて初期化
	prs.Start()                                             // Bot を起動
}

func set(ms *prs.Message, day string, hour int, min int) error {
	ms.Channel.Send(fmt.Sprintf("On %s %02d:%02d, right?", day, hour, min))
	cps.Save(Date{Day: day, Hour: hour, Min: min})
	ms.Stamp("done-nya")
	return nil
}

func get(ms *prs.Message) error {
	date, _ := cps.Load[Date]()
	ms.Channel.Send(fmt.Sprintf("It was on %s %02d:%02d!", date.Day, date.Hour, date.Min))
	return nil
}
