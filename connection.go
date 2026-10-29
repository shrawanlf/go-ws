package ws

import (
	"net"
)

type connection struct {
	conn net.Conn
}

func (conn *connection) Close() error {
	return conn.conn.Close()
}

/*
ReadMessage reads data from the connection and calls the callback function with the data.
A connection closed from client and failure in parsing data will return a CloseError
*/
func (conn *connection) ReadMessage(cb func(payloadData []byte)) error {
	var fin bool
	var data []byte

	for {
		if fin {
			break
		}

		payload, err := parseFrame(conn.conn)
		fin = payload.Fin

		if err != nil {
			err := conn.Close()
			if err != nil {
				return &CloseError{Reason: "Error in closing connection."}
			}
			return &CloseError{Reason: "Error in parsing data."}
		}

		if payload.ClientDisconnected {
			err := conn.Close()
			if err != nil {
				return &CloseError{Reason: "Error in closing connection."}
			}
			return &CloseError{Reason: "connection closed from client."}
		}

		data = append(data, payload.Data...)
	}

	cb(data)

	return nil
}

func (conn *connection) WriteMessage(data []byte) error {
	buf, err := writeData(data)

	if err != nil {
		return err
	}

	conn.conn.Write(buf)
	return nil
}
