package main

import (
	"log"
	ws "websocket"
)

func main() {
	_, err := ws.NewWsClient(ws.WsClientConfigurationOpts{
		Url: "ws://localhost:13373",
	})

	if err != nil {
		log.Fatal(err)
	}
}
