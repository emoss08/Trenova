package emailservice

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type recordingObserver struct {
	mu      sync.Mutex
	changes []emailports.EmailMessageChange
}

func (o *recordingObserver) EmailMessageChanged(_ context.Context, change emailports.EmailMessageChange) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.changes = append(o.changes, change)
}

func (o *recordingObserver) seen() []emailports.EmailMessageChange {
	o.mu.Lock()
	defer o.mu.Unlock()

	return append([]emailports.EmailMessageChange(nil), o.changes...)
}

/*
An observer may depend, through its own dependencies, on sending email: the
inbound reply tracker reaches the notification service, which sends it. Taken by
the constructor, observers made that a dependency cycle and no process started.
They are registered once the graph is built, so an observer that needs the email
service still hears every message change.
*/
func TestRegisterObservers_HandsTheGraphsObserversToTheEmailService(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{}
	var svc *Service
	app := fxtest.New(t,
		fx.Provide(
			func() *Service { return &Service{} },
			func(email *Service) emailports.EmailService { return email },
			fx.Annotate(
				func(emailports.EmailService) emailports.EmailMessageObserver { return observer },
				fx.ResultTags(emailports.EmailMessageObserverGroup),
			),
		),
		fx.Invoke(RegisterObservers),
		fx.Populate(&svc),
	)
	app.RequireStart()
	t.Cleanup(app.RequireStop)

	msg := &email.Message{ID: pulid.MustNew("emsg_")}
	svc.messageChanged(t.Context(), msg, nil)

	seen := observer.seen()
	require.Len(t, seen, 1)
	assert.Same(t, msg, seen[0].Message)
}

func TestObserve_AddsToTheObserversAlreadyRegistered(t *testing.T) {
	t.Parallel()

	first, second := &recordingObserver{}, &recordingObserver{}
	svc := &Service{}
	svc.messageChanged(t.Context(), &email.Message{}, nil)

	svc.Observe(first)
	svc.Observe()
	svc.Observe(second)
	svc.messageChanged(t.Context(), &email.Message{}, nil)

	assert.Len(t, first.seen(), 1)
	assert.Len(t, second.seen(), 1)
}
