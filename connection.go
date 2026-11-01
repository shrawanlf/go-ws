package ws

import (
	"errors"
	"net"
)

type wsconnection interface {
	Close() error
	ReadMessage() ([]byte, error)
	WriteMessage([]byte) error
}

type baseConn struct {
	conn net.Conn
}

func (conn *baseConn) Close() error {
	return conn.conn.Close()
}

/*
ReadMessage reads data from the connection and calls the callback function with the data.
A connection closed from client and failure in parsing data will return a CloseError
*/
func (conn *baseConn) ReadMessage() ([]byte, error) {
	var fin bool
	var data []byte

	for {
		if fin {
			return data, nil
		}

		frame, err := parseIncomingFrame(conn.conn)

		if err != nil {
			err := conn.Close()
			if err != nil {
				return nil, errors.New("Error in closing connection.")
			}
			return nil, errors.New("Error in parsing data.")
		}

		if frame.clientDiconnectionNews() {
			err := conn.Close()
			if err != nil {
				return nil, &CloseError{Reason: "Error in closing connection."}
			}
			return nil, &CloseError{Reason: "connection closed from client."}
		}

		fin = frame.finalFrame()

		data = append(data, *frame.payloadData...)
	}
}
