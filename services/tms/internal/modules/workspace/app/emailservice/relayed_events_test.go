package emailservice

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const (
	relayRecipient = "dispatcher@acme.example"
	relayEventID   = "0192f3a4-5b6c-7d8e-9f01-23456789abcd"
)

type relayStore struct {
	events       map[string]*email.Event
	suppressions map[string]*email.Suppression
	updates      int
}

func (s relayStore) clone() relayStore {
	return relayStore{
		events:       maps.Clone(s.events),
		suppressions: maps.Clone(s.suppressions),
		updates:      s.updates,
	}
}

type relayRepo struct {
	emailports.EmailRepository

	mu              sync.Mutex
	message         *email.Message
	store           relayStore
	failSuppression bool
}

func newRelayRepo(msg *email.Message) *relayRepo {
	return &relayRepo{
		message: msg,
		store: relayStore{
			events:       map[string]*email.Event{},
			suppressions: map[string]*email.Suppression{},
		},
	}
}

func (r *relayRepo) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	r.mu.Lock()
	before := r.store.clone()
	r.mu.Unlock()
	err := fn(ctx)
	if err != nil {
		r.mu.Lock()
		r.store = before
		r.mu.Unlock()
	}
	return err
}

func (r *relayRepo) GetMessageByProviderID(
	_ context.Context,
	req emailports.GetEmailMessageByProviderIDRequest,
) (*email.Message, error) {
	if r.message.Provider != req.Provider || r.message.ProviderMessageID != req.ProviderMessageID {
		return nil, errortypes.NewNotFoundError("EmailMessage not found")
	}
	return r.message, nil
}

func (r *relayRepo) GetMessage(
	_ context.Context,
	req emailports.GetEmailEntityRequest,
) (*email.Message, error) {
	if r.message.ID != req.ID {
		return nil, errortypes.NewNotFoundError("EmailMessage not found")
	}
	copied := *r.message
	return &copied, nil
}

func (r *relayRepo) CreateEvent(_ context.Context, event *email.Event) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := string(event.Provider) + ":" + event.ProviderEventID
	if _, ok := r.store.events[key]; ok {
		return false, nil
	}
	r.store.events[key] = event
	return true, nil
}

func (r *relayRepo) ListEventRecipients(
	_ context.Context,
	req emailports.ListEmailEventRecipientsRequest,
) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []string{}
	for _, event := range r.store.events {
		if event.MessageID == req.MessageID && event.Type == req.Type {
			out = append(out, event.Recipient)
		}
	}
	return out, nil
}

func (r *relayRepo) UpdateMessage(_ context.Context, msg *email.Message) (*email.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store.updates++
	return msg, nil
}

func (r *relayRepo) CreateSuppression(
	_ context.Context,
	suppression *email.Suppression,
) (*email.Suppression, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failSuppression {
		return nil, errors.New("connection reset")
	}
	r.store.suppressions[suppression.OrganizationID.String()+":"+suppression.EmailAddress] = suppression
	return suppression, nil
}

func (r *relayRepo) snapshot() relayStore {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.store.clone()
}

type relayLocator struct {
	tenant  pagination.TenantInfo
	message *email.Message
	err     error
}

func (l relayLocator) TenantOfSystemMessage(
	_ context.Context,
	provider email.Provider,
	providerMessageID string,
) (pagination.TenantInfo, error) {
	if l.err != nil {
		return pagination.TenantInfo{}, l.err
	}
	if provider != l.message.Provider || providerMessageID != l.message.ProviderMessageID {
		return pagination.TenantInfo{}, errortypes.NewNotFoundError("EmailMessage not found")
	}
	return l.tenant, nil
}

type relayRig struct {
	svc     *Service
	repo    *relayRepo
	tenant  pagination.TenantInfo
	message *email.Message
	logs    *observer.ObservedLogs
}

func newRelayRig(t *testing.T, provider email.Provider, locatorErr error) *relayRig {
	t.Helper()
	tenant := testTenantInfo()
	msg := testEmailMessage(tenant)
	msg.Provider = provider
	msg.Sender = email.SenderTrenova
	msg.ToRecipients = []string{relayRecipient}
	repo := newRelayRepo(msg)
	core, logs := observer.New(zap.DebugLevel)
	return &relayRig{
		svc: &Service{
			l:              zap.New(core),
			db:             repo,
			repo:           repo,
			invoiceRepo:    noInvoiceAttempts{},
			systemMessages: relayLocator{tenant: tenant, message: msg, err: locatorErr},
		},
		repo:    repo,
		tenant:  tenant,
		message: msg,
		logs:    logs,
	}
}

func postmarkBounce(messageID string) *RelayedSystemEvent {
	return &RelayedSystemEvent{
		ID:       relayEventID,
		Provider: email.ProviderPostmark,
		Payload: fmt.Appendf(nil, `{
			"RecordType": "Bounce",
			"ID": 883953,
			"Type": "HardBounce",
			"MessageID": %q,
			"Email": %q,
			"BouncedAt": "2026-11-16T10:00:00Z",
			"Metadata": {"trenova_instance": %q}
		}`, messageID, relayRecipient, relayEventID),
	}
}

func (r *relayRig) assertNoRecipientLogged(t *testing.T) {
	t.Helper()
	for _, entry := range r.logs.All() {
		line := entry.Message + " " + fmt.Sprint(entry.ContextMap())
		assert.NotContains(t, line, relayRecipient)
	}
}

func TestARelayedBounceSuppressesTheRecipientForTheOrganizationThatSentIt(t *testing.T) {
	t.Parallel()

	r := newRelayRig(t, email.ProviderPostmark, nil)
	require.NoError(t, r.svc.HandleRelayedSystemEvent(
		t.Context(),
		postmarkBounce(r.message.ProviderMessageID),
	))

	store := r.repo.snapshot()
	require.Len(t, store.events, 1)
	require.Len(t, store.suppressions, 1)
	suppression := slices.Collect(maps.Values(store.suppressions))[0]
	assert.Equal(t, r.tenant.OrgID, suppression.OrganizationID)
	assert.Equal(t, r.tenant.BuID, suppression.BusinessUnitID)
	assert.Equal(t, relayRecipient, suppression.EmailAddress)
	assert.Equal(t, email.SuppressionReasonHardBounce, suppression.Reason)
	assert.Equal(t, 1, store.updates)
	r.assertNoRecipientLogged(t)
}

func TestTheSameRelayedEventTwiceChangesNothingTheSecondTime(t *testing.T) {
	t.Parallel()

	r := newRelayRig(t, email.ProviderPostmark, nil)
	event := postmarkBounce(r.message.ProviderMessageID)
	require.NoError(t, r.svc.HandleRelayedSystemEvent(t.Context(), event))
	first := r.repo.snapshot()

	require.NoError(t, r.svc.HandleRelayedSystemEvent(t.Context(), event))
	second := r.repo.snapshot()

	assert.Len(t, second.events, len(first.events))
	assert.Len(t, second.suppressions, len(first.suppressions))
	assert.Equal(t, first.updates, second.updates)
}

func TestARelayedEventThatFailsHalfwayIsAppliedWhole(t *testing.T) {
	t.Parallel()

	r := newRelayRig(t, email.ProviderPostmark, nil)
	event := postmarkBounce(r.message.ProviderMessageID)
	r.repo.failSuppression = true
	require.Error(t, r.svc.HandleRelayedSystemEvent(t.Context(), event))
	failed := r.repo.snapshot()
	assert.Empty(t, failed.events, "the event is rolled back with the suppression")
	assert.Empty(t, failed.suppressions)

	r.repo.failSuppression = false
	require.NoError(t, r.svc.HandleRelayedSystemEvent(t.Context(), event))
	store := r.repo.snapshot()
	assert.Len(t, store.events, 1)
	assert.Len(t, store.suppressions, 1)
}

func TestARelayedResendComplaintIsKeyedByTheRelayedEvent(t *testing.T) {
	t.Parallel()

	r := newRelayRig(t, email.ProviderResend, nil)
	require.NoError(t, r.svc.HandleRelayedSystemEvent(t.Context(), &RelayedSystemEvent{
		ID:       relayEventID,
		Provider: email.ProviderResend,
		Payload: fmt.Appendf(nil, `{
			"type": "email.complained",
			"created_at": "2026-11-16T10:00:00.000Z",
			"data": {"email_id": %q, "to": [%q], "tags": {"trenova_instance": %q}}
		}`, r.message.ProviderMessageID, relayRecipient, relayEventID),
	}))

	store := r.repo.snapshot()
	require.Contains(t, store.events, "Resend:"+relayEventID)
	require.Len(t, store.suppressions, 1)
	suppression := slices.Collect(maps.Values(store.suppressions))[0]
	assert.Equal(t, email.SuppressionReasonComplaint, suppression.Reason)
	assert.Equal(t, relayEventID, suppression.SourceEventID)
}

func TestARelayedEventThatIsNotOneToApplyIsHandledAndChangesNothing(t *testing.T) {
	t.Parallel()

	cases := map[string]*RelayedSystemEvent{
		"not JSON": {ID: relayEventID, Provider: email.ProviderPostmark, Payload: []byte("<html>")},
		"a soft bounce": {ID: relayEventID, Provider: email.ProviderPostmark, Payload: []byte(
			`{"RecordType":"Bounce","ID":1,"Type":"SoftBounce","MessageID":"provider-message-id","Email":"` +
				relayRecipient + `"}`,
		)},
		"a delivery": {ID: relayEventID, Provider: email.ProviderResend, Payload: []byte(
			`{"type":"email.delivered","data":{"email_id":"provider-message-id","to":["` +
				relayRecipient + `"]}}`,
		)},
		"an unknown provider": {ID: relayEventID, Provider: "SES", Payload: []byte(`{}`)},
	}
	for name, event := range cases {
		r := newRelayRig(t, email.ProviderPostmark, nil)
		require.NoError(t, r.svc.HandleRelayedSystemEvent(t.Context(), event), name)
		store := r.repo.snapshot()
		assert.Empty(t, store.events, name)
		assert.Empty(t, store.suppressions, name)
		r.assertNoRecipientLogged(t)
	}
}

func TestARelayedEventForAMessageThisInstanceDidNotSendIsHandledAndChangesNothing(t *testing.T) {
	t.Parallel()

	r := newRelayRig(t, email.ProviderPostmark, nil)
	require.NoError(t, r.svc.HandleRelayedSystemEvent(
		t.Context(),
		postmarkBounce("a-message-another-instance-sent"),
	))
	store := r.repo.snapshot()
	assert.Empty(t, store.events)
	assert.Empty(t, store.suppressions)
	r.assertNoRecipientLogged(t)
}

func TestARelayedEventIsRetriedWhenTheMessageCannotBeLookedUp(t *testing.T) {
	t.Parallel()

	r := newRelayRig(t, email.ProviderPostmark, errors.New("connection refused"))
	require.Error(t, r.svc.HandleRelayedSystemEvent(
		t.Context(),
		postmarkBounce(r.message.ProviderMessageID),
	))
	assert.Empty(t, r.repo.snapshot().events)
}
