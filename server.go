package ws

import (
	"errors"
	"net/http"
	"strings"
)

type serverConn struct {
	*baseConn
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

	return &serverConn{baseConn: &baseConn{conn: conn}}, nil
}

func (conn *serverConn) WriteMessage(data []byte) error {
	buf, err := prepareFrame(wsframe{
		fin:         1,
		rsv1:        0,
		opCode:      1,
		mask:        0,
		payloadLen:  len(data),
		payloadData: &data,
	})

	if err != nil {
		return err
	}

	conn.conn.Write(buf)
	return nil
}
