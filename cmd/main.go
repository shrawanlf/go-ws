package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
	"websocket/internal/ws"
)

func main() {
	openConnections := 0

	http.HandleFunc("/active-ws-connections", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("%d", openConnections)))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upgrader := ws.NewUpgrader(ws.UpgraderConfiguration{})
		conn, err := upgrader.Upgrade(w, r)

		if err != nil {
			log.Println(err)
			return
		}

		openConnections++

		for {
			go func() {
				defer func() {
					openConnections--
				}()

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
			}()

			time.Sleep(time.Minute * 5)
			return
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
