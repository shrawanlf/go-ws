package ws

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type serverConn struct {
	*baseConn
	lastPingId []byte
}

type UpgraderConfiguration struct{}

type upgrader struct{}

func NewUpgrader(conf UpgraderConfiguration) *upgrader {
	return &upgrader{}
}

func (ws *upgrader) Upgrade(w http.ResponseWriter, r *http.Request) (wsconnection, error) {
	conn, _, err := http.NewResponseController(w).Hijack()

	if err != nil {
		return nil, err
	}

	secureKey := r.Header.Get("Sec-Websocket-Key")
	if secureKey == "" {
		return nil, errors.New("")
	}

	hash := getAcceptHash(secureKey)

	lines := []string{
		"HTTP/1.1 101 Web Socket Protocol Handshake",
		"Server: go/echoserver",
		"Upgrade: WebSocket",
		"connection: Upgrade",
		"Sec-WebSocket-Accept: " + hash,
		"",
		"",
	}

	_, err = conn.Write([]byte(strings.Join(lines, "\r\n")))

	if err != nil {
		return nil, err
	}

	serverConnection := &serverConn{baseConn: &baseConn{conn: conn, state: ConnectionStateOpen, closeCh: make(chan bool)}}

	go serverConnection.startPinging()

	return serverConnection, nil
}

func (conn *serverConn) WriteMessage(data []byte) error {
	buf, err := prepareFrame(wsframe{
		fin:         1,
		opCode:      OpCodeText,
		payloadLen:  len(data),
		payloadData: &data,
	})

	if err != nil {
		return err
	}

	err = conn.write(buf)

	if err != nil {
		return err
	}

	return nil
}

func (conn *serverConn) setLastPingId(id []byte) {
	defer conn.mu.Unlock()
	conn.mu.Lock()
	conn.lastPingId = id
}

// TODO: Need to close this loop on client connection end
func (conn *serverConn) startPinging() {
	for {
		select {
		case <-conn.closeCh:
			log.Println("closing pinger for this connection")
			return
		default:
			fmt.Println("starting ping", string(conn.lastPingId))
			pingId := genRandomString(10)
			conn.setLastPingId(pingId)
			fmt.Println("set ping id")
			frame, err := prepareFrame(wsframe{
				fin:         1,
				rsv1:        0,
				opCode:      OpCodePing,
				payloadLen:  len(pingId),
				payloadData: &pingId,
			})

			if err != nil {
				log.Println(err)
				continue
			}

			err = conn.write(frame)

			if err != nil {
				log.Println(err)
				continue
			}

			time.Sleep(time.Second * 10)
		}
	}
}

func (conn *serverConn) closeFromClient() error {
	return conn.close(false)
}

func (conn *serverConn) Close() error {
	return conn.close(true)
}

/*
ReadMessage reads data from the connection and calls the callback function with the data.
A connection closed from client and failure in parsing data will return a CloseError
*/
func (conn *serverConn) ReadMessage() ([]byte, error) {
	var fin bool
	var data []byte

	// A closing or closed connection should prohibit data read
	if conn.state == ConnectionStateClosed || conn.state == ConnectionStateClosing {
		return data, nil
	}

	for {
		if fin {
			return data, nil
		}

		frame, err := conn.readIncomingFrame()

		if err != nil {
			fmt.Println("errorr", err)
			err := conn.Close()
			if err != nil {
				return nil, errors.New("Error in closing connection.")
			}
			return nil, errors.New("Error in parsing data.")
		}

		if ok, cf := frame.isControlFrame(); ok {
			if cf == OpCodeConnectionClose {
				err := conn.closeFromClient()
				if err != nil {
					return nil, &CloseError{Reason: "Error in closing connection."}
				}
				return nil, &CloseError{Reason: "connection closed from client."}
			}

			if cf == OpCodePong {
				fmt.Println("pong received")
				pingId := frame.payloadData
				if ok := bytes.Equal(*pingId, conn.lastPingId); !ok {
					fmt.Println("invalid ping id received")
				}

				fmt.Println("valid pong id received")
				return data, nil
			}
		}

		fin = frame.finalFrame()

		data = append(data, *frame.payloadData...)
	}
}
