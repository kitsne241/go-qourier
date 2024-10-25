module github.com/kitsne241/go-qourier/persona

go 1.22.6

replace github.com/kitsne241/go-qourier/cprint => ./cprint

require (
	github.com/joho/godotenv v1.5.1
	github.com/traPtitech/go-traq v0.0.0-20240725071454-97c7b85dc879
	github.com/traPtitech/traq-ws-bot v1.2.1
)

require (
	github.com/gofrs/uuid/v5 v5.3.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/oauth2 v0.22.0 // indirect
)
