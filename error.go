package ws

import (
	"errors"
	"fmt"
)

type CloseError struct {
	Reason string
}

func (e *CloseError) Error() string {
	return fmt.Sprintf("Error in close: %s", e.Reason)
}

var errorClosingConnection = errors.New("error closing connection")
var errorPasingFrame = errors.New("error in parsing frame")
var errorGeneratingSecureKey = errors.New("error in generating secure key")
var errorInvalidLenCode = errors.New("invalid len code received")
var errorInvalidMaskingKey = errors.New("invalid masking key provided")
var errorInvalidUtf8Data = errors.New("utf-8 data not received for text frame")
