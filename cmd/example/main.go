package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	ws "websocket"
)

func main() {
	openConnections := 0

	http.HandleFunc("/oc", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("%d", openConnections)))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		openConnections++
		upgrader := ws.NewUpgrader(ws.UpgraderConfiguration{})
		conn, err := upgrader.Upgrade(w, r)

		defer func() {
			openConnections--
		}()

		if err != nil {
			log.Println(err)
			return
		}

		for {
			err := conn.ReadMessage(func(data []byte) {
				fmt.Println("data", string(data))
			})

			if err != nil {
				var ce *ws.CloseError
				if ok := errors.As(err, &ce); ok {
					fmt.Println(ce.Error())
				}
				log.Println(err)
				return
			}
		}
	})

	log.Fatal(http.ListenAndServe(":13373", nil))
}
