package ws

import (
	"errors"
	"fmt"
	"net"
)

type connection struct {
	conn          net.Conn
	maskDataBytes bool
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
			cb(data)
			return nil
		}

		payload, err := parseIncomingFrame(conn.conn)

		if err != nil {
			fmt.Println("raw error in reading message", err)
			err := conn.Close()
			if err != nil {
				return errors.New("Error in closing connection.")
			}
			return errors.New("Error in parsing data.")
		}

		fin = payload.Fin

		if payload.ClientDisconnected {
			err := conn.Close()
			if err != nil {
				return &CloseError{Reason: "Error in closing connection."}
			}
			return &CloseError{Reason: "connection closed from client."}
		}

		data = append(data, payload.Data...)
	}
}

func (conn *connection) WriteMessage(data []byte) error {
	args := prepareWriteFrameArgs{
		data:    data,
		maskKey: nil,
	}
	if conn.maskDataBytes {
		args.maskKey = genMaskKey()
	}

	buf, err := prepareWriteFrame(args)

	fmt.Printf("write bytes %b", buf)

	if err != nil {
		return err
	}

	conn.conn.Write(buf)
	return nil
}
