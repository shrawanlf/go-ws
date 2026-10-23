package ws

import (
	"bytes"
	"net"
	"websocket/internal/parser"
)

type Conn struct {
	Conn net.Conn
}

type ConnectionState struct {
	Open bool
}

func (conn *Conn) readFromConn() ([]byte, error) {
	chunkLength := 1024
	chunk := make([]byte, chunkLength)
	data := bytes.NewBuffer(make([]byte, 0))

	for {
		n, err := conn.Conn.Read(chunk)

		if err != nil {
			return nil, err
		}

		data.Write(chunk[:n])

		if n != chunkLength {
			break
		}
	}

	return data.Bytes(), nil
}

func (conn *Conn) Close() error {
	return conn.Conn.Close()
}

func (conn *Conn) ReadMessage(cb func(data []byte)) error {
	data, err := conn.readFromConn()
	if err != nil {
		return err
	}
	payload := parser.ParseInputData(data)

	if payload.Disconnected {
		err := conn.Close()
		if err != nil {
			return &CloseError{Reason: "Error in closing connection."}
		}
		return &CloseError{Reason: "Connection closed from client."}
	}

	cb(payload.Data)

	return nil
}

func (conn *Conn) WriteMessage(data []byte) error {
	buf, err := parser.WriteData(data)

	if err != nil {
		return err
	}

	conn.Conn.Write(buf)
	return nil
}
