package parser

import (
	"encoding/binary"
	"fmt"
)

type WsInputPayload struct {
	Length       int
	OpCode       []byte
	Disconnected bool
	Masked       bool
	MaskKey      []byte
	Data         []byte
}

/**
* Parses the data into a WsInputPayload
 * data: the data to parse in []byte
*/
func ParseInputData(data []byte) *WsInputPayload {
	readIdx := int8(0)

	//zeroth byte (first one, metadata)
	//no use of these values for now
	payload := &WsInputPayload{}
	firstByte := data[readIdx]
	fin := (firstByte >> 7) & 0x01
	rsv1 := (firstByte >> 6) & 0x01
	rsv2 := (firstByte >> 5) & 0x01
	rsv3 := (firstByte >> 4) & 0x01

	opCode := firstByte & 0x0f

	if opCode == 0x08 {
		payload.Disconnected = true
		return payload
	}

	fmt.Println(fin, rsv1, rsv2, rsv3, opCode)

	readIdx += 1

	// second byte (masked check and length data)
	// up to the 10th byte for bigger length -> upto 64 bits
	secondByte := data[readIdx]
	mask := (secondByte >> 7) & 0x01
	payload.Masked = mask != 0
	length := secondByte & 0x7f

	readIdx += 1

	// bigger payload lengths
	if length <= 125 {
		payload.Length = int(length)
	} else if length == 126 {
		payload.Length = int(binary.BigEndian.Uint16(data[2:4]))
		readIdx += 2
	} else if length == 127 {
		payload.Length = int(binary.BigEndian.Uint64(data[2:10]))
		readIdx += 8
	}

	//11th byte (mask key)
	if payload.Masked {
		maskKey := data[readIdx : readIdx+4]
		payload.MaskKey = maskKey
		readIdx += 4
	}

	payloadData := make([]byte, payload.Length)
	remainingPayload := data[readIdx:]

	for i := 0; i < payload.Length; i++ {
		payloadData[i] = remainingPayload[i] ^ payload.MaskKey[i&0x03]
	}

	payload.Data = payloadData

	return payload
}
