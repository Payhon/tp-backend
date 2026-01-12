package protocol

import "fmt"

type ProtocolError struct {
	Message string
	Extra   map[string]any
}

func (e *ProtocolError) Error() string {
	if len(e.Extra) == 0 {
		return e.Message
	}
	return fmt.Sprintf("%s (%v)", e.Message, e.Extra)
}
