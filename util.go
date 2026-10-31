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
	maskKey := make([]byte, 4)
	_, _ = rand.Read(maskKey)
	return maskKey
}

func readTill(reader io.Reader, len int) ([]byte, error) {
	buf := make([]byte, len)
	_, err := reader.Read(buf)
	return buf, err
}

func maskData(payload []byte, maskKey []byte) []byte {
	payloadLen := len(payload)
	maskedPayload := make([]byte, payloadLen)
	for i, byte := range payload {
		maskedPayload[i] = byte ^ maskKey[i&0x03]
	}

	return maskedPayload
}
