package ws

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type prepareWriteFrameArgs struct {
	data    []byte
	maskKey []byte
}

func prepareWriteFrame(args prepareWriteFrameArgs) ([]byte, error) {
	out := bytes.NewBuffer(nil)
	firstByte := byte(0)
	// FIN, RSV1, RSV2, RSV3 are not set for now, defaults to 0
	firstByte = firstByte | (1 << 7)

	// OPCODE
	firstByte = firstByte | (1 << 0)

	out.WriteByte(firstByte)

	firstByte = byte(0)

	if args.maskKey != nil {
		firstByte = firstByte | (1 << 7)
	}

	// LENGTH
	length := len(args.data)

	if length <= 125 {
		out.WriteByte(firstByte | byte(length))
	} else if length >= 126 && length <= 65535 {
		out.WriteByte(firstByte | byte(126))
		err := binary.Write(out, binary.BigEndian, uint16(length))
		if err != nil {
			return nil, err
		}
	} else if length >= 65536 {
		out.WriteByte(firstByte | byte(127))
		err := binary.Write(out, binary.BigEndian, uint64(length))
		if err != nil {
			return nil, err
		}
	}

	binary.Write(out, binary.BigEndian, args.maskKey)

	// skip masking key, writing data should not have a masking key

	binary.Write(out, binary.BigEndian, args.data)

	return out.Bytes(), nil
}

type wsInputPayload struct {
	Length int
	OpCode byte
	// if fin is true, this is the final fragment of the websocket message
	Fin                bool
	ClientDisconnected bool
	Masked             bool
	MaskKey            []byte
	Data               []byte
}

/**
* Parses the data into a wsInputPayload
 * data: the data to parse in []byte
*/
func parseIncomingFrame(reader io.Reader) (*wsInputPayload, error) {
	//zeroth byte (fin, rsv1, rsv2, rsv3, opCode)
	payload := &wsInputPayload{}

	firstByte, err := readTill(reader, 1)
	if err != nil {
		return nil, err
	}

	fin := (firstByte[0] >> 7) & 0x01
	payload.Fin = fin == 1
	rsv1 := (firstByte[0] >> 6) & 0x01
	rsv2 := (firstByte[0] >> 5) & 0x01
	rsv3 := (firstByte[0] >> 4) & 0x01

	opCode := firstByte[0] & 0x0f
	payload.OpCode = opCode

	if opCode == 0x08 {
		payload.ClientDisconnected = true
		return payload, nil
	}

	fmt.Println(fin, rsv1, rsv2, rsv3, opCode)

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

	remainingPayload, err := readTill(reader, payload.Length)

	if err != nil {
		return nil, err
	}

	payload.Data = maskData(remainingPayload, payload.MaskKey)
	return payload, nil
}
