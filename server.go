package ws

import (
	"bytes"
	"log"
	"net/http"
	"strings"
	"time"
)

type serverConn struct {
	*baseConn
	id         string
	lastPingId []byte
	cfg        serverCfg
}

type serverCfg = UpgraderConfiguration

type UpgraderConfiguration struct {
	// max tries to send ping frame to client
	// defaults to 3
	PingMaxTries int

	//ping interval in ms
	PingInterval int

	//
	MaxPayloadSize int
}

type upgrader struct {
	cfg UpgraderConfiguration
}

func NewUpgrader(conf UpgraderConfiguration) *upgrader {
	if conf.PingMaxTries == 0 {
		conf.PingMaxTries = 3
	}

	return &upgrader{
		cfg: conf,
	}
}

func (ws *upgrader) Upgrade(w http.ResponseWriter, r *http.Request) (wsconnection, error) {
	conn, _, err := http.NewResponseController(w).Hijack()

	if err != nil {
		return nil, err
	}

	secureKey := r.Header.Get("Sec-Websocket-Key")
	if secureKey == "" {
		return nil, errorGeneratingSecureKey
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

	serverConnection := &serverConn{
		cfg: ws.cfg,
		id:  genRandomString(10),
		baseConn: &baseConn{
			conn:  conn,
			state: ConnectionStateOpen,
			done:  make(chan struct{}),
		},
	}

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
	tries := 0

	for {
		select {
		case <-conn.done:
			log.Println("closing pinger for this connection")
			return
		default:
			if tries >= conn.cfg.PingMaxTries {
				log.Println("closing pinger after max tries...")
				return
			}

			pingId := genRandomBytes(10)
			conn.setLastPingId(pingId)
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
				tries++
				log.Println(err)
			} else {
				tries = 0
			}

			time.Sleep(time.Millisecond * time.Duration(conn.cfg.PingInterval))
		}
	}
}

func (conn *serverConn) Close() error {
	return conn.initiateClose(CloseFrameStatusCodeNormal)
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
			if err := conn.closeImmediate(CloseFrameStatusCodeProtocolError); err != nil {
				return nil, errorClosingConnection
			}

			return nil, errorPasingFrame
		}

		if ok, cf := frame.isControlFrame(); ok {
			if cf == OpCodeConnectionClose {
				log.Println("close frame received")

				// connection state Closing means this is a reply from client for our initiated close frame
				// close done is enough
				if conn.state == ConnectionStateClosing {
					if err := conn.closeDone(); err != nil {
						return nil, &CloseError{Reason: "Error in closing connection."}
					}
				}

				// simply initiate close and immediately close the connection on a close frame from client
				if err := conn.closeImmediate(CloseFrameStatusCodeNormal); err != nil {
					return nil, &CloseError{Reason: "Error in closing connection."}
				}

				return nil, &CloseError{Reason: "connection closed from client."}
			}

			if cf == OpCodePong {
				log.Println("pong received")
				pingId := frame.payloadData
				if ok := bytes.Equal(*pingId, conn.lastPingId); !ok {
					log.Println("invalid ping id received")
					if err := conn.closeImmediate(CloseFrameStatusCodeProtocolError); err != nil {
						return nil, errorClosingConnection
					}
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
