package parser

import (
	"bytes"
	"encoding/binary"
)

type WriteParams struct {
	Fin    bool
	Rsv1   bool
	Rsv2   bool
	Rsv3   bool
	OpCode byte
	Length int
	Data   []byte
}

func WriteData(data []byte) ([]byte, error) {
	out := bytes.NewBuffer(nil)
	firstByte := byte(0)
	// 0000 0000
	// FIN, RSV1, RSV2, RSV3 are not set for now, defaults to 0
	firstByte = firstByte | (1 << 7)

	// OPCODE
	firstByte = firstByte | (1 << 0)

	out.WriteByte(firstByte)

	// LENGTH
	length := len(data)

	if length <= 125 {
		out.WriteByte(0 | byte(length))
	} else if length >= 126 && length <= 65535 {
		out.WriteByte(0 | byte(126))
		err := binary.Write(out, binary.BigEndian, uint16(length))
		if err != nil {
			return nil, err
		}
	} else if length >= 65536 {
		out.WriteByte(0 | byte(127))
		err := binary.Write(out, binary.BigEndian, uint64(length))
		if err != nil {
			return nil, err
		}
	}

	// skip masking key, writing data should not have a masking key

	binary.Write(out, binary.BigEndian, data)

	return out.Bytes(), nil
}
