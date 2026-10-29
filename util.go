package ws

import "io"

func readTill(reader io.Reader, len int) ([]byte, error) {
	buf := make([]byte, len)
	_, err := reader.Read(buf)
	return buf, err
}
