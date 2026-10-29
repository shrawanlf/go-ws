package ws

type CloseError struct {
	Reason string
}

func (e *CloseError) Error() string {
	return e.Reason
}
