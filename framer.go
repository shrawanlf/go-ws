package ws

import (
	"encoding/binary"
	"errors"
	"io"
)

type FRAME_LEN_CODE = byte

const (
	FRAME_LEN_CODE_7        FRAME_LEN_CODE = 125
	FRAME_LEN_CODE_7_16_BIT FRAME_LEN_CODE = 126
	FRAME_LEN_CODE_7_64_BIT FRAME_LEN_CODE = 127
)

type wsframe struct {
	fin            byte
	rsv1           byte
	rsv2           byte
	rsv3           byte
	opCode         byte
	mask           byte
	payloadLenCode FRAME_LEN_CODE
	payloadLen     int
	maskingKey     []byte
	payloadData    *[]byte
}

func (frame wsframe) clientDiconnectionNews() bool {
	return frame.opCode == 0x08
}

func (frame wsframe) finalFrame() bool {
	return frame.fin == 1
}

func (frame wsframe) hasMaskingKey() bool {
	return frame.mask == 1
}

func (frame wsframe) hasValidMaskingKey() error {
	if frame.hasMaskingKey() && len(frame.maskingKey) != 4 {
		return errors.New("invalid masking key provided")
	}

	return nil
}

func prepareFrame(frame wsframe) ([]byte, error) {
	frameLength, payloadDataStartId := getFrameSize(&frame)
	frameBytes := make([]byte, frameLength)
	needsMask := false

	if frame.hasMaskingKey() {
		if err := frame.hasValidMaskingKey(); err != nil {
			return nil, err
		}
		frameBytes[1] = frameBytes[1] | (1 << 7)
		needsMask = true
	}

	if frame.fin == 1 {
		frameBytes[0] = frameBytes[0] | (1 << 7)
	}

	if frame.rsv1 == 1 {
		frameBytes[0] = frameBytes[0] | (1 << 4)
	}

	frameBytes[0] = frameBytes[0] | (frame.opCode << 0)

	if frame.payloadLenCode == 0 {
		return frameBytes, nil
	}

	if frame.payloadLenCode == FRAME_LEN_CODE_7 {
		frameBytes[1] = frameBytes[1] | byte(frame.payloadLen)
	} else {
		frameBytes[1] = frameBytes[1] | byte(frame.payloadLenCode)
	}

	switch frame.payloadLenCode {
	case FRAME_LEN_CODE_7:
		frameBytes[1] = frameBytes[1] | byte(frame.payloadLen)
	case FRAME_LEN_CODE_7_16_BIT:
		frameBytes[1] = frameBytes[1] | byte(126)
		binary.BigEndian.PutUint16(frameBytes[2:4], uint16(frame.payloadLen))
	case FRAME_LEN_CODE_7_64_BIT:
		frameBytes[1] = frameBytes[1] | byte(127)
		binary.BigEndian.PutUint64(frameBytes[2:10], uint64(frame.payloadLen))
	default:
		return nil, errors.New("invalid len code received")
	}

	if needsMask {
		maskData(*frame.payloadData, frame.maskingKey)
	}

	copy(frameBytes[payloadDataStartId:], *frame.payloadData)

	return frameBytes, nil
}

/**
* Parses the data into a wsInputPayload
 * data: the data to parse in []byte
*/
func parseIncomingFrame(reader io.Reader) (*wsframe, error) {
	//zeroth byte (fin, rsv1, rsv2, rsv3, opCode)
	frame := &wsframe{}

	firstByte, err := readTill(reader, 1)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	frame.mask = (secondByte[0] >> 7) & 0x01
	length := secondByte[0] & 0x7f

	// bigger payload lengths
	if length <= 125 {
		frame.payloadLen = int(length)
	} else if length == 126 {
		dat, err := readTill(reader, 2)
		if err != nil {
			return nil, err
		}
		frame.payloadLen = int(binary.BigEndian.Uint16(dat))
	} else if length == 127 {
		dat, err := readTill(reader, 8)
		if err != nil {
			return nil, err
		}
		frame.payloadLen = int(binary.BigEndian.Uint64(dat))
	}

	//11th byte (mask key)
	if frame.mask == 1 {
		frame.maskingKey, err = readTill(reader, 4)
		if err != nil {
			return nil, err
		}
	}

	remainingPayload, err := readTill(reader, frame.payloadLen)

	if err != nil {
		return nil, err
	}

	maskData(remainingPayload, frame.maskingKey)
	frame.payloadData = &remainingPayload
	return frame, nil
}
