package assistantturnservice

import (
	"github.com/bytedance/sonic"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

// frameOf builds a frame the relay invented rather than read.
//
// It carries no id: an id is a position in the stream, and there is no
// position for a frame that was never in it. A reader that stored this one as
// its cursor would resume from nowhere.
func frameOf(event string, data any) serviceports.TurnStreamFrame {
	encoded, err := sonic.Marshal(data)
	if err != nil {
		// Every caller passes a literal map of strings, so this cannot fail in
		// practice; an empty object still closes the reader's connection,
		// which is the part that matters.
		encoded = []byte(`{}`)
	}

	return serviceports.TurnStreamFrame{Event: event, Data: encoded}
}
