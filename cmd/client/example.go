package main

import (
	"log"
	ws "websocket"
)

func main() {
	conn, err := ws.NewWsClient(ws.WsClientConfigurationOpts{
		Url: "ws://localhost:13373",
	})

	if err != nil {
		log.Fatal(err)
	}

	err = conn.WriteMessage([]byte("hello from the running client."))

	if err != nil {
		log.Fatal(err)
	}
}
