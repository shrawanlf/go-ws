package ws

import (
	"net"
	"websocket/internal/parser"
)

type Conn struct {
	Conn net.Conn
}

type ConnectionState struct {
	Open bool
}

func (conn *Conn) Close() error {
	return conn.Conn.Close()
}

/*
ReadMessage reads data from the connection and calls the callback function with the data.
If the connection is closed, it will return a CloseError.
*/
func (conn *Conn) ReadMessage(cb func(data []byte)) error {
	payload, err := parser.ParseFrame(conn.Conn)

	if err != nil {
		return &CloseError{Reason: "Error in parsing data."}
	}

	if payload.ClientDisconnected {
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
