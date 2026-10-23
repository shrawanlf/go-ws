package ws

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type Ws struct {
	conn    net.Conn
	bufioRw bufio.ReadWriter
	header  http.Header
}

type UpgraderConfiguration struct {
}

type upgrader struct{}

func NewUpgrader(conf UpgraderConfiguration) *upgrader {
	return &upgrader{}
}

func (ws *upgrader) Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	newWs, err := newWebsocket(w, r)

	if err != nil {
		return nil, err
	}

	err = newWs.handshake()

	if err != nil {
		return nil, err
	}

	conn := &Conn{Conn: newWs.conn}

	return conn, nil
}

func newWebsocket(w http.ResponseWriter, r *http.Request) (*Ws, error) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("no support for hijacking")
	}

	conn, bufioRw, error := hj.Hijack()

	if error != nil {
		return nil, error
	}

	return &Ws{conn: conn, bufioRw: *bufioRw, header: r.Header}, nil
}

func getAcceptHash(key string) string {
	h := sha1.New()
	h.Write([]byte(key))
	h.Write([]byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (ws *Ws) handshake() error {
	hash := getAcceptHash(ws.header.Get("Sec-WebSocket-Key"))
	lines := []string{
		"HTTP/1.1 101 Web Socket Protocol Handshake",
		"Server: go/echoserver",
		"Upgrade: WebSocket",
		"connection: Upgrade",
		"Sec-WebSocket-Accept: " + hash,
		"",
		"",
	}
	_, err := ws.conn.Write([]byte(strings.Join(lines, "\r\n")))
	return err
}
