package wasmmodule

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
)

var (
	okPrefix = []byte(`{"ok":`)

	errUnreadableResponse = errors.New("the response is not a module envelope")
)

type responseEnvelope struct {
	OK    sonic.NoCopyRawMessage `json:"ok"`
	Error *wireRequestError      `json:"error"`
}

type wireRequestError struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func (r *Runtime) envelope(op string, input []byte) ([]byte, error) {
	name, err := sonic.Marshal(op)
	if err != nil {
		return nil, &RequestError{
			Module: r.spec.Name,
			Op:     op,
			Code:   CodeMalformed,
			Detail: err.Error(),
		}
	}
	size := len(`{"op":,"input":}`) + len(name) + len(input)
	if size > MaxRequestBytes {
		return nil, &RequestError{
			Module: r.spec.Name,
			Op:     op,
			Code:   CodeTooLarge,
			Detail: fmt.Sprintf(
				"the request is %d bytes; the module takes at most %d",
				size,
				MaxRequestBytes,
			),
		}
	}
	request := make([]byte, 0, size)
	request = append(request, `{"op":`...)
	request = append(request, name...)
	if len(input) > 0 {
		request = append(request, `,"input":`...)
		request = append(request, input...)
	}
	return append(request, '}'), nil
}

func (r *Runtime) decodeResponse(op string, response []byte) ([]byte, error) {
	if bytes.HasPrefix(response, okPrefix) && response[len(response)-1] == '}' {
		return response[len(okPrefix) : len(response)-1], nil
	}
	var decoded responseEnvelope
	if err := sonic.Unmarshal(response, &decoded); err != nil {
		return nil, fmt.Errorf("%w: %w: %w", ErrTrap, errUnreadableResponse, err)
	}
	switch {
	case decoded.Error != nil:
		return nil, &RequestError{
			Module: r.spec.Name,
			Op:     op,
			Code:   decoded.Error.Code,
			Detail: decoded.Error.Detail,
		}
	case len(decoded.OK) > 0:
		return decoded.OK, nil
	default:
		return nil, fmt.Errorf("%w: %w: neither ok nor error", ErrTrap, errUnreadableResponse)
	}
}
