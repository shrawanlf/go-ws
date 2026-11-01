package ws

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
)

func respHasHeaderWithValue(resp *http.Response, key string, value string) bool {
	val := resp.Header.Get(key)
	if len(val) == 0 || val != value {
		return false
	}

	return true
}

func genSecureKey() string {
	keyBytes := make([]byte, 16)
	_, _ = rand.Read(keyBytes)
	return base64.StdEncoding.EncodeToString(keyBytes)
}

func genMaskKey() []byte {
	var maskKey []byte
	_, _ = rand.Read(maskKey[:])
	return maskKey
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
	payloadDataStartIdx := 2
	var payloadLen int

	if frame.payloadLen == 0 {
		payloadLen = len(*frame.payloadData)
		frame.payloadLen = payloadLen
	} else {
		payloadLen = frame.payloadLen
	}

	if payloadLen == 0 {
		return 0, 0
	}

	if payloadLen <= 125 {
		frame.payloadLenCode = 125
	} else if payloadLen >= 126 && payloadLen <= 65535 {
		totalFrameLength += 2
		frame.payloadLenCode = 126
	} else if payloadLen >= 65536 {
		totalFrameLength += 8
		frame.payloadLenCode = 127
	}

	if frame.hasMaskingKey() {
		payloadDataStartIdx = totalFrameLength + 4
	} else {
		payloadDataStartIdx = totalFrameLength
	}
	totalFrameLength += payloadLen

	return totalFrameLength, payloadDataStartIdx
}
