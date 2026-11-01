package ws

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"
)

type clientConn struct {
	conn net.Conn
	*baseConn
}

type WsClientConfigurationOpts struct {
	Url string
}

func (conn *clientConn) WriteMessage(data []byte) error {
	buf, err := prepareFrame(wsframe{
		fin:         1,
		rsv1:        0,
		opCode:      1,
		mask:        1,
		maskingKey:  genMaskKey(),
		payloadLen:  len(data),
		payloadData: &data,
	})

	if err != nil {
		return err
	}

	conn.conn.Write(buf)
	return nil
}

func NewWsClient(opts WsClientConfigurationOpts) (*clientConn, error) {
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

	return &clientConn{baseConn: &baseConn{conn: conn}, conn: conn}, nil
}
