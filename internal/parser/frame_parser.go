package parser

import (
	"encoding/binary"
	"fmt"
	"io"
)

type WsInputPayload struct {
	Length int
	OpCode byte
	// if fin is true, this is the final fragment of the websocket message
	Fin                bool
	ClientDisconnected bool
	Masked             bool
	MaskKey            []byte
	Data               []byte
}

func readTill(reader io.Reader, len int) ([]byte, error) {
	buf := make([]byte, len)
	_, err := reader.Read(buf)
	return buf, err
}

/**
* Parses the data into a WsInputPayload
 * data: the data to parse in []byte
*/
func ParseFrame(reader io.Reader) (*WsInputPayload, error) {
	//zeroth byte (fin, rsv1, rsv2, rsv3, opCode)
	payload := &WsInputPayload{}

	firstByte, err := readTill(reader, 1)
	if err != nil {
		return nil, err
	}

	fin := (firstByte[0] >> 7) & 0x01
	payload.Fin = fin == 0
	rsv1 := (firstByte[0] >> 6) & 0x01
	rsv2 := (firstByte[0] >> 5) & 0x01
	rsv3 := (firstByte[0] >> 4) & 0x01

	opCode := firstByte[0] & 0x0f
	payload.OpCode = opCode

	if opCode == 0x08 {
		payload.ClientDisconnected = true
		return payload, nil
	}

	fmt.Println(rsv1, rsv2, rsv3, opCode)

	// second byte (masked check and length data)
	// up to the 10th byte for bigger length -> upto 64 bits
	secondByte, err := readTill(reader, 1)
	if err != nil {
		return nil, err
	}
	mask := (secondByte[0] >> 7) & 0x01
	payload.Masked = mask != 0
	length := secondByte[0] & 0x7f

	// bigger payload lengths
	if length <= 125 {
		payload.Length = int(length)
	} else if length == 126 {
		dat, err := readTill(reader, 2)
		if err != nil {
			return nil, err
		}
		payload.Length = int(binary.BigEndian.Uint16(dat))
	} else if length == 127 {
		dat, err := readTill(reader, 8)
		if err != nil {
			return nil, err
		}
		payload.Length = int(binary.BigEndian.Uint64(dat))
	}

	//11th byte (mask key)
	if payload.Masked {
		maskKey, err := readTill(reader, 4)
		if err != nil {
			return nil, err
		}
		payload.MaskKey = maskKey
	}

	payloadData := make([]byte, payload.Length)
	remainingPayload, err := readTill(reader, payload.Length)

	if err != nil {
		return nil, err
	}

	for i := 0; i < payload.Length; i++ {
		payloadData[i] = remainingPayload[i] ^ payload.MaskKey[i&0x03]
	}

	payload.Data = payloadData
	return payload, nil
}
