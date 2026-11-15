package ws

import (
	"bytes"
	"errors"
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

	serverConnection := &serverConn{baseConn: &baseConn{conn: conn, state: ConnectionStateOpen, done: make(chan struct{})}}

	go serverConnection.startPing()

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
	conn.lastPingId = id
}

func (conn *serverConn) startPing() {
	for {
		select {
		case <-conn.done:
			log.Println("closing pinger for this connection")
			return
		default:
			log.Println("starting ping", string(conn.lastPingId))
			pingId := genRandomString(10)
			conn.setLastPingId(pingId)
			log.Println("set ping id")
			frame, err := prepareFrame(wsframe{
				fin:         1,
				rsv1:        0,
				mask:        0,
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
	dataBuf := bytes.NewBuffer(nil)

	// fin => 1, end return the data as a whole at once.
	// fin => 0, means continuation frame, save the last opCode
	// fin => 0 & opCode = 0, means previous opCode with data
	// fin => 1, finally means finished.

	for {
		if fin {
			log.Println("yup fin")
			return dataBuf.Bytes(), nil
		}

		// A closing or closed connection should prohibit data read
		if conn.state == ConnectionStateClosed || conn.state == ConnectionStateClosing {
			log.Println("connection closing")
			continue
		}

		frame, err := conn.readIncomingFrame()

		if err != nil {
			log.Println("error raw", err)
			err := conn.Close()
			if err != nil {
				return nil, errors.New("Error in closing connection.")
			}
			return nil, errors.New("Error in parsing data.")
		}

		if ok, cf := frame.isControlFrame(); ok {
			if cf == OpCodeConnectionClose {
				log.Println("close frame received")
				err := conn.closeFromClient()
				if err != nil {
					return nil, &CloseError{Reason: "Error in closing connection."}
				}
				return nil, &CloseError{Reason: "connection closed from client."}
			}

			if cf == OpCodePong {
				log.Println("pong received")
				pingId := frame.payloadData
				if ok := bytes.Equal(*pingId, conn.lastPingId); !ok {
					log.Println("invalid ping id received")
				}

				log.Println("valid pong id received")

				// skip on pong data receive
				// first byte should always have a fin bit enabled
				dataBuf.Reset()
				if !frame.finalFrame() {
					return nil, nil
				}
				continue
			}
		}

		fin = frame.finalFrame()

		dataBuf.Write(*frame.payloadData)
	}
}
