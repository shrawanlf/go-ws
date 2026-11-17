package ws

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"io"
	"net/http"
	"unicode/utf8"
)

func dataIsUtf8(data []byte) bool {
	return utf8.Valid(data)
}

func respHasHeaderWithValue(resp *http.Response, key string, value string) bool {
	val := resp.Header.Get(key)
	if len(val) == 0 || val != value {
		return false
	}

	return true
}

func genRandomBytes(length int) []byte {
	randBytes := make([]byte, length)
	rand.Read(randBytes)
	return randBytes
}

func genRandomString(length int) string {
	b := genRandomBytes(length)
	return base64.StdEncoding.EncodeToString(b)
}

func genSecureKey() string {
	return genRandomString(16)
}

func genMaskKey() []byte {
	return genRandomBytes(4)
}

func readTill(reader io.Reader, len int) ([]byte, error) {
	buf := make([]byte, len)
	_, err := reader.Read(buf)
	return buf, err
}

/*
Masks payload bytes
*/
func maskData(payload []byte, maskKey []byte) {
	for i := range payload {
		payload[i] ^= maskKey[i&3]
	}
}

func getFrameSize(frame *wsframe) (int, int) {
	totalFrameLength := 2
	var payloadLen int

	if frame.payloadLen == 0 && frame.payloadData != nil {
		payloadLen = len(*frame.payloadData)
		frame.payloadLen = payloadLen
	} else {
		payloadLen = frame.payloadLen
	}

	if payloadLen == 0 {
		return totalFrameLength, 0
	}

	if payloadLen <= 125 {
		frame.payloadLenCode = FRAME_LEN_CODE_7
	} else if payloadLen >= 126 && payloadLen <= 65535 {
		totalFrameLength += 2
		frame.payloadLenCode = FRAME_LEN_CODE_7_16_BIT
	} else if payloadLen >= 65536 {
		totalFrameLength += 8
		frame.payloadLenCode = FRAME_LEN_CODE_7_64_BIT
	}

	if frame.hasMaskingKey() {
		totalFrameLength += 4
	}

	payloadDataStartIdx := totalFrameLength
	totalFrameLength += payloadLen

	return totalFrameLength, payloadDataStartIdx
}

func getAcceptHash(key string) string {
	h := sha1.New()
	h.Write([]byte(key))
	h.Write([]byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
