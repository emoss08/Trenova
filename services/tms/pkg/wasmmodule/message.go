package wasmmodule

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/i18n"
)

var errMessageArg = errors.New("a message argument must be text, a whole number or a message")

type Message struct {
	Template string
	Args     []any
}

type wireMessage struct {
	Message string                   `json:"message"`
	Args    []sonic.NoCopyRawMessage `json:"args"`
}

func (m *Message) UnmarshalJSON(data []byte) error {
	var wire wireMessage
	if err := sonic.Unmarshal(data, &wire); err != nil {
		return err
	}
	args := make([]any, 0, len(wire.Args))
	for _, raw := range wire.Args {
		arg, err := messageArg(raw)
		if err != nil {
			return err
		}
		args = append(args, arg)
	}
	m.Template = wire.Message
	m.Args = args
	return nil
}

func messageArg(raw []byte) (any, error) {
	if len(raw) > 0 && raw[0] == '{' {
		var nested Message
		if err := nested.UnmarshalJSON(raw); err != nil {
			return nil, err
		}
		return nested, nil
	}
	if len(raw) > 0 && raw[0] == '"' {
		var text string
		if err := sonic.Unmarshal(raw, &text); err != nil {
			return nil, err
		}
		return text, nil
	}
	count, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errMessageArg, raw)
	}
	return count, nil
}

func (m Message) IsZero() bool {
	return m.Template == ""
}

func (m Message) Localize(locale i18n.Locale) string {
	return i18n.Translate(locale, m.Template, m.Args...)
}

func (m Message) String() string {
	return m.Localize(i18n.Default)
}

func (m Message) LocalizedMessage() (message string, args []any) {
	return m.Template, m.Args
}

func (m Message) ValidationError(field string, code errortypes.ErrorCode) *errortypes.Error {
	return errortypes.NewValidationError(field, code, m.Template, m.Args...)
}

func (m Message) BusinessError() *errortypes.BusinessError {
	return errortypes.NewBusinessError(m.Template, m.Args...)
}
