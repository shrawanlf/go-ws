package ws

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"
)

type WsClientConfigurationOpts struct {
	Url string
}

func NewWsClient(opts WsClientConfigurationOpts) (*connection, error) {
	url, err := url.Parse(opts.Url)

	if err != nil {
		return nil, err
	}

	if url.Scheme != "ws" && url.Scheme != "wss" {
		return nil, errors.New(url.Scheme + "unsupported scheme provided")
	}

	address := url.Host
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)

	if err != nil {
		return nil, err
	}

	path := url.RequestURI()
	key := genSecureKey()

	req := "GET " + path + " HTTP/1.1\r\n" +
		"Host: " + url.Host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"

	conn.SetDeadline(time.Now().Add(10 * time.Second)) // handshake timeout
	if _, err := conn.Write([]byte(req)); err != nil {
		conn.Close()
		return nil, err
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		conn.Close()
		return nil, err
	}

	if resp.StatusCode != http.StatusSwitchingProtocols ||
		respHasHeaderWithValue(resp, "Connection", "upgrade") ||
		respHasHeaderWithValue(resp, "Upgrade", "websocket") ||
		resp.Header.Get("Sec-Websocket-Accept") != getAcceptHash(key) {
		return nil, conn.Close()
	}

	return &connection{conn: conn, maskDataBytes: true}, nil
}
