package triagebriefingservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagesupport"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks/triageportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var tenantInfo = pagination.TenantInfo{OrgID: "org_test", BuID: "bu_test"}

type fakeSettings struct {
	inboundmessageports.TriageSettingsRepository
}

func (fakeSettings) Get(_ context.Context, info pagination.TenantInfo) (*triage.Settings, error) {
	return triage.DefaultSettings(info.OrgID, info.BuID), nil
}

type fakeMailboxes struct {
	inboundmessageports.InboundMailboxRepository

	items []*inboundmessage.Mailbox
}

func (f fakeMailboxes) List(
	context.Context,
	*inboundmessageports.ListMailboxesRequest,
) (*pagination.ListResult[*inboundmessage.Mailbox], error) {
	return &pagination.ListResult[*inboundmessage.Mailbox]{
		Items:      f.items,
		Total:      len(f.items),
		TotalExact: true,
	}, nil
}

type fakeMessages struct {
	inboundmessageports.InboundMessageRepository
}

func (fakeMessages) ListAttachmentsByMessageIDs(
	context.Context,
	inboundmessageports.ListAttachmentsByMessageIDsRequest,
) (map[pulid.ID][]*inboundmessage.InboundAttachment, error) {
	return map[pulid.ID][]*inboundmessage.InboundAttachment{}, nil
}

type fakeRules struct {
	suggestion *inboundmessageports.RuleSuggestion
}

func (f fakeRules) Suggest(
	context.Context,
	pagination.TenantInfo,
) (*inboundmessageports.RuleSuggestion, error) {
	return f.suggestion, nil
}

func (fakeRules) SenderRules(
	context.Context,
	pagination.TenantInfo,
	*inboundmessage.InboundMessage,
) ([]*triage.Rule, error) {
	return []*triage.Rule{}, nil
}

type chicagoOrganizations struct{}

func (chicagoOrganizations) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: "America/Chicago"}, nil
}

func newService(
	t *testing.T,
	insights *triageportsmock.MockTriageInsightRepository,
	cache triageports.TriageCache,
) *Service {
	t.Helper()

	models := triageportsmock.NewMockInboundModelAvailability(t)
	models.EXPECT().ReadsInboundMail(mock.Anything, tenantInfo).Return(true, nil).Maybe()
	now := time.Date(2026, time.October, 12, 15, 0, 0, 0, time.UTC).Unix()

	return &Service{
		l:         zap.NewNop(),
		insights:  insights,
		messages:  fakeMessages{},
		mailboxes: fakeMailboxes{},
		settings:  fakeSettings{},
		rules: fakeRules{suggestion: &inboundmessageports.RuleSuggestion{
			Sender: "acme.example",
		}},
		cache:  cache,
		models: models,
		clock:  triagesupport.NewClock(chicagoOrganizations{}, func() int64 { return now }),
	}
}

func expectGather(insights *triageportsmock.MockTriageInsightRepository) {
	insights.EXPECT().BriefingCounts(mock.Anything, mock.Anything).
		Return(&triageports.BriefingCounts{Waiting: 2, AnyMessage: true}, nil).Once()
	insights.EXPECT().HandledByNova(mock.Anything, mock.Anything).
		Return([]*triageports.HandledRow{}, nil).Once()
	insights.EXPECT().Arrivals(mock.Anything, mock.Anything).
		Return([]*triageports.ArrivalRow{}, nil).Once()
	insights.EXPECT().River(mock.Anything, mock.Anything).
		Return([]*triageports.RiverRow{}, nil).Once()
	insights.EXPECT().Waiting(mock.Anything, triageports.WaitingRequest{
		TenantInfo: tenantInfo,
		Limit:      MaxWaitingScanned,
	}).Return([]*inboundmessage.InboundMessage{}, nil).Once()
	insights.EXPECT().CorrectionTotals(mock.Anything, mock.Anything).
		Return(&triageports.CorrectionTotals{}, nil).Once()
}

func TestBriefing_ComputesOnAMissAndServesTheCacheAfter(t *testing.T) {
	t.Parallel()

	insights := triageportsmock.NewMockTriageInsightRepository(t)
	cache := triageportsmock.NewMockTriageCache(t)
	service := newService(t, insights, cache)
	key := &triageports.CacheKey{Section: triageports.CacheBriefing, TenantInfo: tenantInfo, Epoch: 4}

	cache.EXPECT().QueueEpoch(mock.Anything, tenantInfo).Return(4, nil).Twice()
	cache.EXPECT().Get(mock.Anything, key, mock.Anything).Return(false, nil).Once()
	expectGather(insights)
	var stored *inboundmessageports.Briefing
	cache.EXPECT().Set(mock.Anything, key, mock.Anything, BriefingTTL).
		Run(func(_ context.Context, _ *triageports.CacheKey, value any, _ time.Duration) {
			stored = value.(*inboundmessageports.Briefing)
		}).
		Return(nil).Once()

	briefing, err := service.Briefing(t.Context(), tenantInfo)
	require.NoError(t, err)
	assert.Equal(t, 2, briefing.Waiting)
	assert.True(t, briefing.NovaEnabled)
	require.NotNil(t, briefing.Suggestion)
	assert.Same(t, briefing, stored)

	cache.EXPECT().Get(mock.Anything, key, mock.Anything).
		Run(func(_ context.Context, _ *triageports.CacheKey, dest any) {
			*dest.(*inboundmessageports.Briefing) = inboundmessageports.Briefing{Waiting: 9}
		}).
		Return(true, nil).Once()

	cached, err := service.Briefing(t.Context(), tenantInfo)
	require.NoError(t, err)
	assert.Equal(t, 9, cached.Waiting, "a hit is served without reading the queue")
}

func TestBriefing_WorksWithoutTheCache(t *testing.T) {
	t.Parallel()

	insights := triageportsmock.NewMockTriageInsightRepository(t)
	cache := triageportsmock.NewMockTriageCache(t)
	service := newService(t, insights, cache)
	cache.EXPECT().QueueEpoch(mock.Anything, tenantInfo).Return(0, assert.AnError).Once()
	expectGather(insights)

	briefing, err := service.Briefing(t.Context(), tenantInfo)

	require.NoError(t, err)
	assert.Equal(t, 2, briefing.Waiting)
}

func TestInvalidator_BumpsTheQueueEpoch(t *testing.T) {
	t.Parallel()

	cache := triageportsmock.NewMockTriageCache(t)
	cache.EXPECT().BumpQueueEpoch(mock.Anything, tenantInfo).Return(nil).Once()

	NewInvalidator(InvalidatorParams{Logger: zap.NewNop(), Cache: cache}).
		InvalidateBriefing(t.Context(), tenantInfo)
}

func TestProfile_ReadsTheSender(t *testing.T) {
	t.Parallel()

	seconds := 2700.0
	profile := Profile(&ProfileInputs{
		Address: "ap@kestrel.example",
		Domain:  "kestrel.example",
		Stats: &triageports.SenderStats{
			Messages:           14,
			UsualSettleSeconds: &seconds,
			Kinds: []triageports.SenderKindRow{
				{Classification: inboundmessage.ClassificationInvoice, Count: 12},
			},
			Recent: []*triageports.MessageRow{{ID: "imsg_2", Subject: "Invoice KF-5521"}},
		},
		Party: &triageports.SenderParty{Role: inboundmessageports.SenderRoleCarrier, Since: 1_600_000_000},
		Rules: []*triage.Rule{{ID: "trule_1"}},
	})

	assert.Equal(t, "ap", profile.Name)
	assert.Equal(t, inboundmessageports.SenderRoleCarrier, profile.Role)
	require.NotNil(t, profile.Since)
	require.NotNil(t, profile.UsualReplyMinutes)
	assert.Equal(t, 45, *profile.UsualReplyMinutes)
	require.Len(t, profile.Kinds, 1)
	require.Len(t, profile.Recent, 1)
	assert.Equal(t, "KF-5521", profile.Recent[0].Reference)
	assert.Len(t, profile.Rules, 1)
}
