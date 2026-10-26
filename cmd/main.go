package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"websocket/internal/ws"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upgrader := ws.NewUpgrader(ws.UpgraderConfiguration{})
		conn, err := upgrader.Upgrade(w, r)

		if err != nil {
			log.Println(err)
			return
		}

		for {
			err := conn.ReadMessage(func(data []byte) {
				fmt.Println(string(data))
			})

			if err != nil {
				var ce *ws.CloseError
				if ok := errors.As(err, &ce); ok {
					fmt.Println(ce.Error())
				}
				log.Println(err)
				return
			}
			err = conn.WriteMessage([]byte("Hello"))

			if err != nil {
				log.Println("err", err)
				return
			}
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
