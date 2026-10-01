package modeladapter

import (
	"bytes"
	"net/http"
)

var contextOverflowPhrases = [...][]byte{
	[]byte("context_length_exceeded"),
	[]byte("prompt is too long"),
	[]byte("prompt too long"),
	[]byte("maximum context length"),
	[]byte("context length"),
	[]byte("context window"),
	[]byte("input is too long"),
	[]byte("input token count"),
	[]byte("too many input tokens"),
	[]byte("exceeds the maximum number of tokens"),
	[]byte("reduce the length of the messages"),
	[]byte("requested tokens exceed"),
	[]byte("model_context_window_exceeded"),
}

func contextOverflow(status int, payload []byte) bool {
	switch status {
	case http.StatusRequestEntityTooLarge:
		return true
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
	default:
		return false
	}

	lowered := bytes.ToLower(payload)
	for _, phrase := range contextOverflowPhrases {
		if bytes.Contains(lowered, phrase) {
			return true
		}
	}

	return false
}
