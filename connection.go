package ws

import (
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
	CloseFrameStatusCodeNormal                CloseFrameStatusCode = 1000
	CloseFrameGoingAway                       CloseFrameStatusCode = 1001
	CloseFrameStatusCodeProtocolError         CloseFrameStatusCode = 1002
	CloseFrameStatusCodeMessageLengthOverflow CloseFrameStatusCode = 1009
	CloseFrameStatusCodeUnexpectedError       CloseFrameStatusCode = 1011
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

func (conn *baseConn) write(data []byte) error {
	conn.writeMu.Lock()
	defer conn.writeMu.Unlock()

	if _, err := conn.conn.Write(data); err != nil {
		return err
	}

	return nil
}

func (conn *serverConn) close(reply bool) error {
	defer func() {
		conn.setState(ConnectionStateClosed)
		close(conn.done)
	}()

	closeFrame, err := connectionCloseFrame(CloseFrameStatusCodeNormal, "closed from server")

	if err != nil {
		return err
	}

	conn.setState(ConnectionStateClosing)
	err = conn.write(closeFrame)

	if err != nil {
		return err
	}

	if reply {
		frame, err := conn.readIncomingFrame()

		if err != nil {
			return err
		}

		if ok := frame.isCloseFrame(); !ok {
			return conn.conn.Close()
		}
	}

	return conn.conn.Close()
}
