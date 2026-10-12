package emailports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
)

const EmailMessageObserverGroup = `group:"email_message_observers"`

type EmailMessageChange struct {
	Message *email.Message
	Event   *email.Event
}

type EmailMessageObserver interface {
	EmailMessageChanged(ctx context.Context, change EmailMessageChange)
}
