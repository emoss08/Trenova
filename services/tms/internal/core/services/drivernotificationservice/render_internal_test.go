package drivernotificationservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// zapNop keeps the test logger out of the assertions.
func zapNop(t *testing.T) *zap.Logger {
	t.Helper()
	return zap.NewNop()
}

func notifiedWorker() *worker.Worker {
	return &worker.Worker{
		ID:        pulid.MustNew("wrk_"),
		UserID:    pulid.MustNew("usr_"),
		FirstName: "Marcus",
		LastName:  "Dell",
	}
}

// The hub owns two things no call site should restate: which template kind an
// event maps to, and who is being told. This asserts both, plus the fallback
// policy — a driver told in the shipped wording is better off than one not told.
func TestRenderKeysTheTemplateByEventTypeAndFillsTheRecipient(t *testing.T) {
	t.Parallel()

	resolver := mocks.NewMockDocumentTemplateResolver(t)
	resolver.EXPECT().
		RenderMessage(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *services.RenderMessageRequest,
		) (*services.RenderedMessage, error) {
			assert.Equal(t, documenttemplate.KindNotificationSettlementPaid, req.Kind)
			assert.True(t, req.FallbackToBuiltIn,
				"a lost notification is worse than an unstyled one")

			data, ok := req.Data.(documenttemplate.SettlementNotificationContext)
			require.True(t, ok)
			assert.Equal(t, "Marcus", data.FirstName)
			assert.Equal(t, "Dell", data.LastName)
			assert.Equal(t, "Marcus Dell", data.FullName)
			assert.Equal(t, "STL-1", data.SettlementNumber,
				"the caller's own data must survive the recipient being filled in")

			return &services.RenderedMessage{
				Subject: "You've been paid",
				Text:    "Settlement STL-1 was paid via ACH.",
			}, nil
		}).
		Once()

	svc := &Service{l: zapNop(t), templates: resolver}

	title, message, ok := svc.render(
		t.Context(),
		&DriverNotification{
			TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_")},
			WorkerID:   pulid.MustNew("wrk_"),
			EventType:  "dash.settlement_paid",
			Context: documenttemplate.SettlementNotificationContext{
				SettlementNumber: "STL-1",
			},
		},
		notifiedWorker(),
	)

	require.True(t, ok)
	assert.Equal(t, "You've been paid", title)
	assert.Equal(t, "Settlement STL-1 was paid via ACH.", message)
}

// A render that even the built-in cannot survive drops the notification. That
// matches how this service already treats a worker with no portal access: driver
// notification is best-effort and never fails the operation that triggered it.
func TestRenderDropsTheNotificationWhenNothingRenders(t *testing.T) {
	t.Parallel()

	resolver := mocks.NewMockDocumentTemplateResolver(t)
	resolver.EXPECT().
		RenderMessage(mock.Anything, mock.Anything).
		Return(nil, errors.New("the kind is not registered")).
		Once()

	svc := &Service{l: zapNop(t), templates: resolver}

	_, _, ok := svc.render(
		t.Context(),
		&DriverNotification{EventType: "dash.unknown_event"},
		notifiedWorker(),
	)
	assert.False(t, ok)

	// So does a deployment with no template rendering configured, rather than
	// sending a notification with an empty title.
	bare := &Service{l: zapNop(t)}
	_, _, ok = bare.render(
		t.Context(),
		&DriverNotification{EventType: "dash.load_assigned"},
		notifiedWorker(),
	)
	assert.False(t, ok)
}

// A context that does not carry a recipient block still renders. Not every future
// family is about one person.
func TestRenderToleratesAContextWithoutARecipient(t *testing.T) {
	t.Parallel()

	resolver := mocks.NewMockDocumentTemplateResolver(t)
	resolver.EXPECT().
		RenderMessage(mock.Anything, mock.Anything).
		Return(&services.RenderedMessage{Subject: "Title", Text: "Body"}, nil).
		Once()

	svc := &Service{l: zapNop(t), templates: resolver}

	title, message, ok := svc.render(
		t.Context(),
		&DriverNotification{EventType: "dash.load_assigned", Context: struct{}{}},
		notifiedWorker(),
	)

	require.True(t, ok)
	assert.Equal(t, "Title", title)
	assert.Equal(t, "Body", message)
}

func TestNotificationKindPrefixesTheEventType(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		documenttemplate.KindNotificationLoadAssigned,
		notificationKind("dash.load_assigned"),
	)
}
