package ws

import (
	"errors"
	"log"
	"net"
	"sync"
)

type FRAME_LEN_CODE = byte
type ConnectionState string
type CloseFrameStatusCode = uint16

const (
	FRAME_LEN_CODE_7        FRAME_LEN_CODE = 125
	FRAME_LEN_CODE_7_16_BIT FRAME_LEN_CODE = 126
	FRAME_LEN_CODE_7_64_BIT FRAME_LEN_CODE = 127
)

const (
	CloseFrameStatusCodeNormal          CloseFrameStatusCode = 1000
	CloseFrameGoingAway                 CloseFrameStatusCode = 1001
	CloseFrameStatusCodeProtocolError   CloseFrameStatusCode = 1002
	CloseFrameStatusCodeFrameTooLarge   CloseFrameStatusCode = 1009
	CloseFrameStatusCodeUnexpectedError CloseFrameStatusCode = 1011
)

const (
	ConnectionStateOpen    ConnectionState = "Open"
	ConnectionStateClosing ConnectionState = "Closing"
	ConnectionStateClosed  ConnectionState = "Closed"
)

const (
	OpCodeContinuationFrame = 0
	OpCodeText              = 1
	OpCodeBinary            = 2
	OpCodeConnectionClose   = 8
	OpCodePing              = 9
	OpCodePong              = 10
)

type wsconnection interface {
	Close() error
	ReadMessage() ([]byte, error)
	WriteMessage([]byte) error
}

type baseConn struct {
	id      string
	writeMu sync.Mutex
	readMu  sync.RWMutex
	conn    net.Conn
	state   ConnectionState
	done    chan struct{}
}

func (conn *baseConn) setState(state ConnectionState) {
	conn.state = state
}

func (conn *baseConn) readIncomingFrame() (wsframe, error) {
	return parseIncomingFrame(conn.conn)
}

// writes data to the connection only if connection state is open.
// Returns error if connection state is Closing or Closed
func (conn *baseConn) write(data []byte) error {
	if conn.state == ConnectionStateClosed || conn.state == ConnectionStateClosing {
		return errors.New("connection in closing state")
	}

	conn.writeMu.Lock()
	defer conn.writeMu.Unlock()

	if _, err := conn.conn.Write(data); err != nil {
		return err
	}

	return nil
}

// Writes close frame to the client and closes TCP connection immediately. Does not wait for a close frame reply.
func (conn *serverConn) closeImmediate(statusCode CloseFrameStatusCode) error {
	if err := conn.initiateClose(statusCode); err != nil {
		return err
	}

	if err := conn.closeDone(); err != nil {
		return err
	}

	return nil
}

// Initiates a close frame to the client.
// Performing writes to the connection after initiating close will result in an error.
func (conn *serverConn) initiateClose(statusCode CloseFrameStatusCode) error {
	log.Println("initiating close")
	defer conn.setState(ConnectionStateClosing)
	closeFrame, err := connectionCloseFrame(statusCode, "closed from server")

	if err != nil {
		return err
	}

	if err = conn.write(closeFrame); err != nil {
		return err
	}

	return nil
}

// Closes the TCP connection and all go-routines spawned by this connection.
func (conn *serverConn) closeDone() error {
	log.Println("close done...")
	conn.setState(ConnectionStateClosed)
	close(conn.done)
	return conn.conn.Close()
}
