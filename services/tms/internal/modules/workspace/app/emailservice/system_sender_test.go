package emailservice

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/systememail"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/testutil/mocks/emailportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/remedy"
	"github.com/emoss08/trenova/shared/mailtext"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const (
	trenovaTestKey      = "pm-trenova-0123456789abcdef0123"
	trenovaTestFrom     = "no-reply@mail.trenova.app"
	trenovaTestInstance = "0192f3a4-5b6c-7d8e-9f01-23456789abcd"
	memberAddress       = "dispatcher@acme.example"
	outsideAddress      = "buyer@customer.example"
)

type fakeTrenovaSender struct {
	mu         sync.Mutex
	credential *systememail.Credential
	calls      int
}

func (f *fakeTrenovaSender) Current(context.Context) (*systememail.Credential, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.credential, nil
}

func (f *fakeTrenovaSender) asked() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

type fakeDirectory struct {
	members []string
}

func (f fakeDirectory) MemberAddresses(
	_ context.Context,
	_ pagination.TenantInfo,
	addresses []string,
) ([]string, error) {
	out := []string{}
	for _, address := range addresses {
		if slices.Contains(f.members, mailtext.NormalizeEmailAddress(address)) {
			out = append(out, mailtext.NormalizeEmailAddress(address))
		}
	}
	return out, nil
}

type fakeRateLimits struct {
	mu   sync.Mutex
	deny string
	seen []kernelports.RateLimitRequest
}

func (f *fakeRateLimits) Check(
	_ context.Context,
	requests []kernelports.RateLimitRequest,
) ([]kernelports.RateLimitDecision, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seen = append(f.seen, requests...)
	out := make([]kernelports.RateLimitDecision, len(requests))
	for i, req := range requests {
		allowed := f.deny == "" || !strings.HasPrefix(req.Key, f.deny)
		out[i] = kernelports.RateLimitDecision{Allowed: allowed}
		if !allowed {
			out[i].RetryAfter = 12 * time.Minute
		}
	}
	return out, nil
}

type fakeOrganizations struct {
	tenantports.OrganizationRepository
}

func (fakeOrganizations) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Name: "Acme Freight"}, nil
}

type fakeProvider struct {
	mu       sync.Mutex
	config   map[string]string
	metadata map[string]string
	from     string
	err      error
	calls    int
}

func (f *fakeProvider) Provider() email.Provider          { return email.ProviderPostmark }
func (f *fakeProvider) IntegrationType() integration.Type { return integration.TypePostmark }

func (f *fakeProvider) Send(
	_ context.Context,
	req SendProviderRequest,
) (*SendProviderResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.config = req.Config
	f.metadata = req.Message.Metadata
	f.from = req.Message.From
	if f.err != nil {
		return nil, f.err
	}
	return &SendProviderResponse{ProviderMessageID: "pm-message-1"}, nil
}

type trenovaRig struct {
	svc        *Service
	repo       *emailportsmock.MockEmailRepository
	workflows  *kernelportsmock.MockWorkflowStarter
	trenova    *fakeTrenovaSender
	rateLimits *fakeRateLimits
	provider   *fakeProvider
	logs       *observer.ObservedLogs
	tenant     pagination.TenantInfo
	profile    *email.Profile
	created    []*email.Message
	mu         sync.Mutex
}

type rigOptions struct {
	withProfile bool
	withTrenova bool
	suppressed  []string
}

func newTrenovaRig(t *testing.T, opts rigOptions) *trenovaRig {
	t.Helper()

	r := &trenovaRig{
		repo:       emailportsmock.NewMockEmailRepository(t),
		workflows:  kernelportsmock.NewMockWorkflowStarter(t),
		trenova:    &fakeTrenovaSender{},
		rateLimits: &fakeRateLimits{},
		provider:   &fakeProvider{},
		tenant:     testTenantInfo(),
	}
	if opts.withTrenova {
		r.trenova.credential = &systememail.Credential{
			Provider:    email.ProviderPostmark,
			APIKey:      trenovaTestKey,
			FromAddress: trenovaTestFrom,
			FromName:    "Trenova",
		}
	}
	r.profile = &email.Profile{
		ID:             pulid.MustNew("emlprof_"),
		BusinessUnitID: r.tenant.BuID,
		OrganizationID: r.tenant.OrgID,
		Provider:       email.ProviderResend,
		SenderEmail:    "ops@acme.example",
		SenderName:     "Acme Operations",
		Status:         email.ProfileStatusActive,
	}
	if opts.withProfile {
		r.repo.EXPECT().GetAssignedProfile(mock.Anything, r.tenant, mock.Anything).
			Return(r.profile, nil).Maybe()
	} else {
		r.repo.EXPECT().GetAssignedProfile(mock.Anything, r.tenant, mock.Anything).
			Return(nil, errortypes.NewNotFoundError("EmailProfile not found")).Maybe()
	}
	r.repo.EXPECT().HasSuppression(mock.Anything, r.tenant, mock.Anything).
		RunAndReturn(func(_ context.Context, _ pagination.TenantInfo, address string) (bool, error) {
			return slices.Contains(opts.suppressed, mailtext.NormalizeEmailAddress(address)), nil
		}).Maybe()
	r.repo.EXPECT().CreateMessageOnce(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, msg *email.Message) (*email.Message, bool, error) {
			r.mu.Lock()
			defer r.mu.Unlock()
			msg.ID = pulid.MustNew("emlmsg_")
			r.created = append(r.created, msg)
			return msg, true, nil
		}).Maybe()
	r.workflows.EXPECT().StartWorkflow(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil).Maybe()

	core, logs := observer.New(zap.DebugLevel)
	r.logs = logs
	r.svc = &Service{
		l:               zap.New(core),
		instanceID:      trenovaTestInstance,
		repo:            r.repo,
		invoiceRepo:     noInvoiceAttempts{},
		validator:       NewValidator(),
		workflowStarter: r.workflows,
		features:        plantest.AllFeatures(),
		providerSenders: map[email.Provider]ProviderSender{email.ProviderPostmark: r.provider},
		systemSenders:   r.trenova,
		directory:       fakeDirectory{members: []string{memberAddress}},
		rateLimits:      r.rateLimits,
		organizations:   fakeOrganizations{},
	}
	return r
}

func (r *trenovaRig) request(typ email.Type, to ...string) *emailports.SendEmailRequest {
	return &emailports.SendEmailRequest{
		TenantInfo: r.tenant,
		Type:       typ,
		Purpose:    email.PurposeGeneral,
		To:         to,
		Subject:    "Hello",
		HTML:       "<p>Hello</p>",
		Text:       "Hello",
	}
}

func (r *trenovaRig) sentMessages() []*email.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.created)
}

func (r *trenovaRig) assertNoSecretLogged(t *testing.T) {
	t.Helper()
	for _, entry := range r.logs.All() {
		line := entry.Message + " " + fmt.Sprint(entry.ContextMap())
		assert.NotContains(t, line, trenovaTestKey)
	}
}

func TestASystemEmailWithNoProfileGoesThroughTheTrenovaSender(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{withTrenova: true})
	msg, err := r.svc.Send(t.Context(), r.request(email.TypePasswordReset, memberAddress))

	require.NoError(t, err)
	assert.Equal(t, email.SenderTrenova, msg.Sender)
	assert.True(t, msg.ProfileID.IsNil())
	assert.Equal(t, email.TypePasswordReset, msg.Type)
	assert.Equal(t, email.KindSystem, msg.Kind)
	assert.Equal(t, email.ProviderPostmark, msg.Provider)
	assert.Equal(t, trenovaTestFrom, msg.FromEmail)
	assert.Equal(t, "Acme Freight via Trenova", msg.FromName)
	assert.Empty(t, msg.ReplyToEmail)
	require.Len(t, r.rateLimits.seen, 1)
	assert.Equal(t, trenovaInstanceKey, r.rateLimits.seen[0].Key)
	r.assertNoSecretLogged(t)
}

func TestASystemEmailWithNoProfileAndNoTrenovaSenderBehavesAsToday(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{})
	msg, err := r.svc.Send(t.Context(), r.request(email.TypePasswordReset, memberAddress))

	require.Nil(t, msg)
	require.ErrorIs(t, err, emailports.ErrNoEmailProfile)
	kind, ok := remedy.Of(err)
	require.True(t, ok)
	assert.Equal(t, remedy.SetUpEmailProfile, kind)
	assert.Empty(t, r.sentMessages())
	assert.Empty(t, r.rateLimits.seen)
}

func TestEveryCustomerFacingEmailWithNoProfileIsRefusedAndNeverReachesTheTrenovaSender(
	t *testing.T,
) {
	t.Parallel()

	for _, typ := range email.Types() {
		if typ.Kind() != email.KindCustomerFacing {
			continue
		}
		r := newTrenovaRig(t, rigOptions{withTrenova: true})
		msg, err := r.svc.Send(t.Context(), r.request(typ, memberAddress))

		require.Nil(t, msg, typ)
		require.ErrorIs(t, err, emailports.ErrNoEmailProfile, typ)
		assert.Zero(t, r.trenova.asked(), "%s asked for the Trenova sender", typ)
		assert.Empty(t, r.sentMessages(), typ)

		sender, err := r.svc.ResolveSender(t.Context(), r.request(typ, memberAddress))
		require.Nil(t, sender, typ)
		require.ErrorIs(t, err, emailports.ErrNoEmailProfile, typ)
		assert.Zero(t, r.trenova.asked(), "%s previewed the Trenova sender", typ)
	}
}

func TestACustomerFacingEmailWithAProfileGoesThroughIt(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{withProfile: true, withTrenova: true})
	msg, err := r.svc.Send(t.Context(), r.request(email.TypeInvoice, outsideAddress))

	require.NoError(t, err)
	assert.Equal(t, email.SenderOrganization, msg.Sender)
	assert.Equal(t, r.profile.ID, msg.ProfileID)
	assert.Equal(t, email.KindCustomerFacing, msg.Kind)
	assert.Equal(t, "ops@acme.example", msg.FromEmail)
	assert.Zero(t, r.trenova.asked())
	assert.Empty(t, r.rateLimits.seen)
}

func TestASystemEmailWithAProfileUsesTheOrganizationsProfile(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{withProfile: true, withTrenova: true})
	msg, err := r.svc.Send(t.Context(), r.request(email.TypeNotification, memberAddress))

	require.NoError(t, err)
	assert.Equal(t, email.SenderOrganization, msg.Sender)
	assert.Equal(t, r.profile.ID, msg.ProfileID)
	assert.Zero(t, r.trenova.asked())
}

func TestTheTrenovaSenderOnlyReachesTheOrganizationsOwnPeople(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{withTrenova: true})
	req := r.request(email.TypeReportDelivery, memberAddress)
	req.CC = []string{outsideAddress}
	msg, err := r.svc.Send(t.Context(), req)

	require.Nil(t, msg)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Contains(t, business.Error(), outsideAddress)
	kind, ok := remedy.Of(err)
	require.True(t, ok)
	assert.Equal(t, remedy.SetUpEmailProfile, kind)
	assert.Empty(t, r.sentMessages())
}

func TestAnInvitationMayReachSomeoneWhoIsNotYetAUser(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{withTrenova: true})
	msg, err := r.svc.Send(t.Context(), r.request(email.TypeAccountInvitation, outsideAddress))

	require.NoError(t, err)
	assert.Equal(t, email.SenderTrenova, msg.Sender)
	keys := make([]string, 0, len(r.rateLimits.seen))
	for _, req := range r.rateLimits.seen {
		keys = append(keys, req.Key)
	}
	assert.Equal(t, []string{
		trenovaInstanceKey,
		trenovaInvitationPrefix + r.tenant.OrgID.String(),
	}, keys)
}

func TestInvitationsThroughTheTrenovaSenderAreRateLimitedPerOrganization(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{withTrenova: true})
	r.rateLimits.deny = trenovaInvitationPrefix
	msg, err := r.svc.Send(t.Context(), r.request(email.TypeDriverPortalInvitation, outsideAddress))

	require.Nil(t, msg)
	require.True(t, errortypes.IsRateLimitError(err))
	retryAfter, ok := errortypes.RetryAfterOf(err)
	require.True(t, ok)
	assert.Equal(t, 12*time.Minute, retryAfter)
	assert.Contains(t, err.Error(), "invitations")
	assert.Empty(t, r.sentMessages())
}

func TestTheTrenovaSenderIsRateLimitedPerInstance(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{withTrenova: true})
	r.rateLimits.deny = trenovaInstanceKey
	msg, err := r.svc.Send(t.Context(), r.request(email.TypeNotification, memberAddress))

	require.Nil(t, msg)
	require.True(t, errortypes.IsRateLimitError(err))
	assert.Empty(t, r.sentMessages())
}

func TestTheSuppressionListIsHonouredByTheTrenovaSender(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{withTrenova: true, suppressed: []string{memberAddress}})
	msg, err := r.svc.Send(t.Context(), r.request(email.TypeNotification, memberAddress))

	require.Nil(t, msg)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Contains(t, business.Error(), "suppression list")
	assert.Empty(t, r.sentMessages())

	other := "planner@acme.example"
	r = newTrenovaRig(t, rigOptions{withTrenova: true, suppressed: []string{other}})
	req := r.request(email.TypeAccountInvitation, outsideAddress)
	req.BCC = []string{other}
	msg, err = r.svc.Send(t.Context(), req)
	require.NoError(t, err)
	assert.Empty(t, msg.BCCRecipients)
}

func TestResolveSenderNamesTheTrenovaSender(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{withTrenova: true})
	sender, err := r.svc.ResolveSender(t.Context(), r.request(email.TypeNotification, memberAddress))

	require.NoError(t, err)
	assert.True(t, sender.ViaTrenova)
	assert.True(t, sender.ProfileID.IsNil())
	assert.Equal(t, trenovaTestFrom, sender.Email)
	assert.Equal(t, "Acme Freight via Trenova", sender.Name)
}

func (r *trenovaRig) persisted(t *testing.T, kind email.Kind, typ email.Type) *email.Message {
	t.Helper()
	msg := &email.Message{
		ID:             pulid.MustNew("emlmsg_"),
		BusinessUnitID: r.tenant.BuID,
		OrganizationID: r.tenant.OrgID,
		Purpose:        email.PurposeGeneral,
		Type:           typ,
		Kind:           kind,
		Sender:         email.SenderTrenova,
		Provider:       email.ProviderPostmark,
		IdempotencyKey: "reset-1",
		Status:         email.MessageStatusQueued,
		FromEmail:      trenovaTestFrom,
		FromName:       "Acme Freight via Trenova",
		ToRecipients:   []string{memberAddress},
		Subject:        "Reset your password",
	}
	r.repo.EXPECT().GetMessage(mock.Anything, mock.Anything).Return(msg, nil).Once()
	r.repo.EXPECT().UpdateMessage(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, m *email.Message) (*email.Message, error) {
			return m, nil
		}).Maybe()
	r.repo.EXPECT().ListAttachments(mock.Anything, mock.Anything).
		Return([]*email.Attachment{}, nil).Maybe()
	return msg
}

func (r *trenovaRig) sendPersisted(t *testing.T, msg *email.Message) (*email.Message, error) {
	t.Helper()
	return r.svc.SendPersisted(t.Context(), &emailports.SendPersistedEmailRequest{
		TenantInfo: r.tenant,
		MessageID:  msg.ID,
		HTML:       "<p>Reset</p>",
		Text:       "Reset",
	})
}

func TestSendPersistedSendsATrenovaMessageWithTheTrenovaCredential(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{withTrenova: true})
	msg := r.persisted(t, email.KindSystem, email.TypePasswordReset)

	sent, err := r.sendPersisted(t, msg)

	require.NoError(t, err)
	assert.Equal(t, email.MessageStatusSent, sent.Status)
	assert.Equal(t, map[string]string{"serverToken": trenovaTestKey}, r.provider.config)
	assert.Equal(t, mailtext.FormatEmailAddress("Acme Freight via Trenova", trenovaTestFrom), r.provider.from)
	assert.Equal(
		t,
		map[string]string{systememail.InstanceTag: trenovaTestInstance},
		r.provider.metadata,
	)
	r.assertNoSecretLogged(t)
}

func TestSendPersistedRefusesATrenovaMessageWhenTheInstanceHasNoID(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{withTrenova: true})
	r.svc.instanceID = ""
	msg := r.persisted(t, email.KindSystem, email.TypePasswordReset)

	sent, err := r.sendPersisted(t, msg)

	require.ErrorIs(t, err, ErrNonRetryableSend)
	assert.Equal(t, email.MessageStatusFailed, sent.Status)
	assert.Zero(t, r.provider.calls, "Trenova's sender never carries an untagged message")
}

func TestSendPersistedNeverSendsACustomerFacingMessageThroughTrenova(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{withTrenova: true})
	msg := r.persisted(t, email.KindCustomerFacing, email.TypeInvoice)

	sent, err := r.sendPersisted(t, msg)

	require.ErrorIs(t, err, ErrNonRetryableSend)
	assert.Equal(t, email.MessageStatusFailed, sent.Status)
	assert.Zero(t, r.provider.calls)
	assert.Zero(t, r.trenova.asked())
}

func TestSendPersistedFailsAMessageWhoseTrenovaSenderWasRemoved(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{})
	msg := r.persisted(t, email.KindSystem, email.TypePasswordReset)

	sent, err := r.sendPersisted(t, msg)

	require.ErrorIs(t, err, ErrNonRetryableSend)
	assert.Equal(t, email.MessageStatusFailed, sent.Status)
	assert.Zero(t, r.provider.calls)
}

func TestATrenovaProviderFailureNamesNoSecretAndKeepsItsRetryClass(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{withTrenova: true})
	r.provider.err = fmt.Errorf(
		"%w: postmark answered 503 for key %s",
		ErrRetryableSend,
		trenovaTestKey,
	)
	msg := r.persisted(t, email.KindSystem, email.TypePasswordReset)

	sent, err := r.sendPersisted(t, msg)

	require.ErrorIs(t, err, ErrRetryableSend)
	assert.NotContains(t, err.Error(), trenovaTestKey)
	assert.NotContains(t, sent.LastError, trenovaTestKey)
	assert.NotContains(t, sent.LastError, "503")
	require.NotEmpty(t, r.logs.All())
	r.assertNoSecretLogged(t)
}

func TestTheSettingsLineShowsWhenTrenovaCarriesTheOrganizationsSystemEmail(t *testing.T) {
	t.Parallel()

	r := newTrenovaRig(t, rigOptions{withTrenova: true})
	inUse, from, err := r.svc.SystemEmailInUse(t.Context(), r.tenant)
	require.NoError(t, err)
	assert.True(t, inUse)
	assert.Equal(t, trenovaTestFrom, from)

	r = newTrenovaRig(t, rigOptions{withProfile: true, withTrenova: true})
	inUse, from, err = r.svc.SystemEmailInUse(t.Context(), r.tenant)
	require.NoError(t, err)
	assert.False(t, inUse)
	assert.Empty(t, from)

	r = newTrenovaRig(t, rigOptions{})
	inUse, _, err = r.svc.SystemEmailInUse(t.Context(), r.tenant)
	require.NoError(t, err)
	assert.False(t, inUse)
}
