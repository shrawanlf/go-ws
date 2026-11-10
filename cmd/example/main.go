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
		fmt.Println("client connected")
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
			data, err := conn.ReadMessage()

			if err != nil {
				var ce *ws.CloseError
				if ok := errors.As(err, &ce); ok {
					log.Println(ce.Error())
					return
				}
				log.Println(err)
				return
			}

			fmt.Println("data", string(data))

			err = conn.WriteMessage([]byte("Hello there from server"))

			if err != nil {
				log.Println(err)
				return
			}

			// err = conn.Close()
			//
			// if err != nil {
			// 	log.Println("err", err)
			// 	return
			// }
		}
	})

	log.Fatal(http.ListenAndServe(":13373", nil))
}
