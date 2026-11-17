package ws

import (
	"encoding/binary"
	"io"
)

type wsframe struct {
	//the fin bit, used to check if this frame is the only frame sent from sender
	fin byte
	// rsv1 bit for extension negotiation
	rsv1 byte
	rsv2 byte
	rsv3 byte
	// opcode defines the payload data type
	// a certain set of numbers (8, 9, 10) denote control frames
	opCode byte
	// mask bit ensures if the payload data is masked or not, masked from a client's end only
	mask byte
	// length code, denotes if the length of payload data needs more bytes to be read or is sufficient in the provided 7 bit
	// each length code is mapped based on the payload length
	payloadLenCode FRAME_LEN_CODE
	// calculated from payloadData, if no value provided
	payloadLen int
	// the 4 byte masking key is used both to mask/unmask the data
	maskingKey []byte
	// the actual application data
	payloadData *[]byte
}

// Checks if the incoming data is a control frame.
// The returned integer is the frame's OPCODE.
func (frame wsframe) isControlFrame() (bool, int) {
	opCode := frame.opCode
	if (opCode == OpCodeConnectionClose || opCode == OpCodePing || opCode == OpCodePong) && frame.finalFrame() {
		return true, int(frame.opCode)
	}

	return false, int(opCode)
}

func (frame wsframe) isCloseFrame() bool {
	if ok, opCode := frame.isControlFrame(); ok && opCode == OpCodeConnectionClose {
		return true
	}

	return false
}

func (frame wsframe) isUtf8Data() bool {
	if frame.opCode == OpCodeText && dataIsUtf8(*frame.payloadData) {
		return true
	}

	return false
}

func (frame wsframe) finalFrame() bool {
	return frame.fin == 1
}

func (frame wsframe) isContinuationFrame() bool {
	return frame.opCode == 0
}

func (frame wsframe) hasMaskingKey() bool {
	return frame.mask == 1
}

func (frame wsframe) hasValidMaskingKey() error {
	if frame.hasMaskingKey() && len(frame.maskingKey) != 4 {
		return errorInvalidMaskingKey
	}
	return nil
}

func connectionCloseFrame(code CloseFrameStatusCode, reason string) ([]byte, error) {
	// 2 bytes is the default size of a code
	if len(reason) > 123 {
		reason = reason[:123]
	}

	data := make([]byte, len(reason)+2)

	binary.BigEndian.PutUint16(data, code)
	copy(data[2:], reason)

	frame, err := prepareFrame(wsframe{
		fin:         1,
		opCode:      OpCodeConnectionClose,
		payloadLen:  len(data),
		payloadData: &data,
	})

	if err != nil {
		return nil, err
	}

	return frame, nil
}

/*
Prepares byte to write to the connection from the provided frame struct.
It is the callers responsibility to provide correct frame values.
See wsframe type for more info.
*/
func prepareFrame(frame wsframe) ([]byte, error) {
	frameLength, payloadDataStartId := getFrameSize(&frame)
	frameBytes := make([]byte, frameLength)

	if frame.fin == 1 {
		frameBytes[0] = frameBytes[0] | (1 << 7)
	}

	if frame.rsv1 == 1 {
		frameBytes[0] = frameBytes[0] | (1 << 6)
	}

	frameBytes[0] = frameBytes[0] | (frame.opCode << 0)

	// This denotes that there is not payload in this frame, so the remaining bytes are not processed and are of zeroth value
	if frame.payloadLenCode == 0 {
		return frameBytes, nil
	}

	if frame.hasMaskingKey() {
		if err := frame.hasValidMaskingKey(); err != nil {
			return nil, err
		}
		frameBytes[1] = frameBytes[1] | (1 << 7)
	}

	switch frame.payloadLenCode {
	case FRAME_LEN_CODE_7:
		frameBytes[1] = frameBytes[1] | byte(frame.payloadLen)
		copy(frameBytes[2:6], frame.maskingKey)
	case FRAME_LEN_CODE_7_16_BIT:
		frameBytes[1] = frameBytes[1] | 126
		binary.BigEndian.PutUint16(frameBytes[2:4], uint16(frame.payloadLen))
		copy(frameBytes[4:8], frame.maskingKey)
	case FRAME_LEN_CODE_7_64_BIT:
		frameBytes[1] = frameBytes[1] | 127
		binary.BigEndian.PutUint64(frameBytes[2:10], uint64(frame.payloadLen))
		copy(frameBytes[10:14], frame.maskingKey)
	default:
		return nil, errorInvalidLenCode
	}

	if frame.hasMaskingKey() {
		maskData(*frame.payloadData, frame.maskingKey)
	}

	copy(frameBytes[payloadDataStartId:], *frame.payloadData)

	return frameBytes, nil
}

/**
* Parses the data from connection to a frame struct.
* Returns error if incoming frame is not structured as per RFC-6455.
 */
func parseIncomingFrame(reader io.Reader) (wsframe, error) {
	frame := wsframe{}

	firstByte, err := readTill(reader, 1)
	if err != nil {
		return frame, err
	}

	frame.fin = (firstByte[0] >> 7) & 0x01
	frame.rsv1 = (firstByte[0] >> 6) & 0x01

	// rsv2 := (firstByte[0] >> 5) & 0x01
	// rsv3 := (firstByte[0] >> 4) & 0x01

	frame.opCode = firstByte[0] & 0x0f

	// second byte (masked check and length data)
	// up to the 10th byte for bigger length -> upto 64 bits
	secondByte, err := readTill(reader, 1)
	if err != nil {
		return frame, err
	}
	frame.mask = (secondByte[0] >> 7) & 0x01
	length := secondByte[0] & 0x7f

	switch {
	case length <= FRAME_LEN_CODE_7:
		frame.payloadLen = int(length)
		frame.payloadLenCode = FRAME_LEN_CODE_7
	case length == FRAME_LEN_CODE_7_16_BIT:
		dat, err := readTill(reader, 2)
		if err != nil {
			return frame, err
		}
		frame.payloadLen = int(binary.BigEndian.Uint16(dat))
		frame.payloadLenCode = FRAME_LEN_CODE_7_16_BIT
	case length == FRAME_LEN_CODE_7_64_BIT:
		dat, err := readTill(reader, 8)
		if err != nil {
			return frame, err
		}
		frame.payloadLen = int(binary.BigEndian.Uint64(dat))
		frame.payloadLenCode = FRAME_LEN_CODE_7_64_BIT
	}

	if frame.hasMaskingKey() {
		frame.maskingKey, err = readTill(reader, 4)
		if err != nil {
			return frame, err
		}
	}

	remainingPayload, err := readTill(reader, frame.payloadLen)

	if err != nil {
		return frame, err
	}

	maskData(remainingPayload, frame.maskingKey)
	frame.payloadData = &remainingPayload
	return frame, nil
}
