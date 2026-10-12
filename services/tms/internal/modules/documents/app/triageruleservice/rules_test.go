package triageruleservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagesupport"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/triageportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var testTenant = pagination.TenantInfo{OrgID: testOrg, BuID: testBU, UserID: "usr_admin"}

type serviceRig struct {
	service     *Service
	rules       *triageportsmock.MockTriageRuleRepository
	insights    *triageportsmock.MockTriageInsightRepository
	cache       *triageportsmock.MockTriageCache
	permissions *permissionportsmock.MockPermissionEngine
	audit       *auditportsmock.MockAuditService
	activities  *fakeActivities
	now         int64
}

func newServiceRig(t *testing.T) *serviceRig {
	t.Helper()

	rig := &serviceRig{
		rules:       triageportsmock.NewMockTriageRuleRepository(t),
		insights:    triageportsmock.NewMockTriageInsightRepository(t),
		cache:       triageportsmock.NewMockTriageCache(t),
		permissions: permissionportsmock.NewMockPermissionEngine(t),
		audit:       auditportsmock.NewMockAuditService(t),
		activities:  &fakeActivities{},
		now:         time.Date(2026, time.October, 12, 15, 0, 0, 0, time.UTC).Unix(),
	}
	rig.service = &Service{
		l:        zap.NewNop(),
		rules:    rig.rules,
		insights: rig.insights,
		mailboxes: &fakeMailboxes{known: map[pulid.ID]*inboundmessage.Mailbox{
			"imbx_intake": {ID: "imbx_intake"},
		}},
		permissions: rig.permissions,
		activities:  rig.activities,
		cache:       rig.cache,
		audit:       rig.audit,
		clock: triagesupport.NewClock(fixedOrganizations{timezone: "America/Chicago"},
			func() int64 { return rig.now }),
	}

	return rig
}

func (r *serviceRig) expectNoDuplicate() {
	r.rules.EXPECT().FindBySentence(mock.Anything, mock.Anything).Return(nil, nil).Once()
}

func (r *serviceRig) expectRulesChanged() {
	r.cache.EXPECT().DropEnabledRules(mock.Anything, testTenant).Return(nil).Once()
	r.cache.EXPECT().BumpQueueEpoch(mock.Anything, testTenant).Return(nil).Once()
	r.audit.EXPECT().LogAction(mock.Anything).Return(nil).Once()
}

func podRule() *triage.Rule {
	return &triage.Rule{
		Classification: inboundmessage.ClassificationProofOfDelivery,
		SenderScope:    triage.SenderDomain,
		Sender:         "  Drivers.Trenova.Example ",
		MailboxID:      "imbx_intake",
		Action:         inboundmessage.NovaActionFileOnShipment,
		Enabled:        true,
	}
}

func fieldsOf(t *testing.T, err error) map[string]bool {
	t.Helper()

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make(map[string]bool, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields[fieldErr.Field] = true
	}

	return fields
}

func TestCreate_SavesANormalizedSentence(t *testing.T) {
	t.Parallel()

	rig := newServiceRig(t)
	rig.expectNoDuplicate()
	rig.rules.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(rule *triage.Rule) bool {
			return rule.Sender == "drivers.trenova.example" &&
				rule.OrganizationID == testOrg &&
				rule.CreatedByID == "usr_admin" &&
				rule.Enabled
		}), triage.MaxRulesPerTenant).
		RunAndReturn(func(_ context.Context, rule *triage.Rule, _ int) (*triage.Rule, error) {
			rule.ID = "trule_new"

			return rule, nil
		}).
		Once()
	rig.expectRulesChanged()

	created, err := rig.service.Create(t.Context(), SaveRequest{
		TenantInfo: testTenant,
		ActorID:    "usr_admin",
		Rule:       podRule(),
	})

	require.NoError(t, err)
	assert.Equal(t, pulid.ID("trule_new"), created.ID)
}

func TestCreate_RefusesTheSameSentenceTwice(t *testing.T) {
	t.Parallel()

	rig := newServiceRig(t)
	rig.rules.EXPECT().FindBySentence(mock.Anything, mock.Anything).
		Return(&triage.Rule{ID: "trule_old"}, nil).Once()

	_, err := rig.service.Create(t.Context(), SaveRequest{
		TenantInfo: testTenant,
		ActorID:    "usr_admin",
		Rule:       podRule(),
	})

	assert.True(t, fieldsOf(t, err)["classification"])
}

func TestCreate_RefusesASourceThatIsNotTheOrganizations(t *testing.T) {
	t.Parallel()

	rig := newServiceRig(t)
	rig.expectNoDuplicate()
	rule := podRule()
	rule.MailboxID = "imbx_elsewhere"

	_, err := rig.service.Create(t.Context(), SaveRequest{TenantInfo: testTenant, Rule: rule})

	assert.True(t, fieldsOf(t, err)["mailboxId"])
}

func TestCreate_RefusesAnAssigneeWhoCannotReadTheQueue(t *testing.T) {
	t.Parallel()

	rig := newServiceRig(t)
	rig.expectNoDuplicate()
	rig.permissions.EXPECT().
		Check(mock.Anything, mock.MatchedBy(func(req *permissionports.PermissionCheckRequest) bool {
			return req.UserID == "usr_driver" && req.Resource == "inbound_message"
		})).
		Return(&permissionports.PermissionCheckResult{Allowed: false}, nil).
		Once()
	rule := podRule()
	rule.Action = inboundmessage.NovaActionAssign
	rule.AssigneeID = "usr_driver"

	_, err := rig.service.Create(t.Context(), SaveRequest{TenantInfo: testTenant, Rule: rule})

	assert.True(t, fieldsOf(t, err)["assigneeId"])
}

func TestCreate_RefusesAnInvalidSentence(t *testing.T) {
	t.Parallel()

	rig := newServiceRig(t)
	rule := podRule()
	rule.Sender = "not a domain"

	_, err := rig.service.Create(t.Context(), SaveRequest{TenantInfo: testTenant, Rule: rule})

	assert.True(t, fieldsOf(t, err)["sender"])
}

func TestCreate_SaysWhenTheOrganizationHasTooManyRules(t *testing.T) {
	t.Parallel()

	rig := newServiceRig(t)
	rig.expectNoDuplicate()
	rig.rules.EXPECT().Create(mock.Anything, mock.Anything, triage.MaxRulesPerTenant).
		Return(nil, triageports.ErrTooManyRules).Once()

	_, err := rig.service.Create(t.Context(), SaveRequest{TenantInfo: testTenant, Rule: podRule()})

	assert.True(t, errortypes.IsBusinessError(err))
}

func TestUpdate_KeepsTheCounters(t *testing.T) {
	t.Parallel()

	rig := newServiceRig(t)
	original := podRule()
	original.ID = "trule_1"
	original.OrganizationID = testOrg
	original.BusinessUnitID = testBU
	original.MatchCount = 142
	original.LastMessageID = "imsg_last"
	original.Version = 3
	rig.rules.EXPECT().GetByID(mock.Anything, triageports.GetRuleRequest{
		TenantInfo: testTenant,
		ID:         "trule_1",
	}).Return(original, nil).Once()
	rig.expectNoDuplicate()
	rig.rules.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(rule *triage.Rule) bool {
			return rule.MatchCount == 142 && rule.LastMessageID == "imsg_last" &&
				rule.Version == 3 && rule.Action == inboundmessage.NovaActionIgnore
		})).
		RunAndReturn(func(_ context.Context, rule *triage.Rule) (*triage.Rule, error) {
			return rule, nil
		}).
		Once()
	rig.expectRulesChanged()

	input := podRule()
	input.Action = inboundmessage.NovaActionIgnore
	updated, err := rig.service.Update(t.Context(), &UpdateRequest{
		TenantInfo: testTenant,
		ActorID:    "usr_admin",
		ID:         "trule_1",
		Version:    3,
		Rule:       input,
	})

	require.NoError(t, err)
	assert.Equal(t, inboundmessage.NovaActionIgnore, updated.Action)
}

func TestTeach_ReturnsAndEnablesTheRuleThatAlreadyExists(t *testing.T) {
	t.Parallel()

	rig := newServiceRig(t)
	existing := &triage.Rule{ID: "trule_old", Action: inboundmessage.NovaActionIgnore, Enabled: false}
	rig.rules.EXPECT().FindBySentence(mock.Anything, mock.Anything).Return(existing, nil).Once()
	rig.rules.EXPECT().GetByID(mock.Anything, mock.Anything).Return(existing, nil).Once()
	enabled := *existing
	enabled.Enabled = true
	rig.rules.EXPECT().SetEnabled(mock.Anything, triageports.SetRuleEnabledRequest{
		TenantInfo: testTenant,
		ID:         "trule_old",
		Enabled:    true,
	}).Return(&enabled, nil).Once()
	rig.expectRulesChanged()

	taught, err := rig.service.Teach(t.Context(), inboundmessageports.TeachRequest{
		TenantInfo:     testTenant,
		ActorID:        "usr_admin",
		MessageID:      "imsg_1",
		Classification: inboundmessage.ClassificationOther,
		SenderDomain:   "newsletters.example",
		Action:         inboundmessage.NovaActionIgnore,
	})

	require.NoError(t, err)
	assert.True(t, taught.Enabled)
	require.Len(t, rig.activities.recorded, 1)
	assert.Equal(t, triage.ActivityRuleCreated, rig.activities.recorded[0].Kind)
	assert.Equal(t, pulid.ID("trule_old"), rig.activities.recorded[0].RecordID)
}

func TestTeach_NeedsASender(t *testing.T) {
	t.Parallel()

	rig := newServiceRig(t)

	_, err := rig.service.Teach(t.Context(), inboundmessageports.TeachRequest{
		TenantInfo: testTenant,
		Action:     inboundmessage.NovaActionIgnore,
	})

	require.Error(t, err)
}

func TestMatchesThisMonth_CountsFromTheLocalMonthStart(t *testing.T) {
	t.Parallel()

	rig := newServiceRig(t)
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	rig.rules.EXPECT().
		CountSettledSince(mock.Anything, &triageports.CountRuleSettlementsRequest{
			TenantInfo: testTenant,
			RuleIDs:    []pulid.ID{"trule_1"},
			Since:      time.Date(2026, time.October, 1, 0, 0, 0, 0, chicago).Unix(),
		}).
		Return(map[pulid.ID]int{"trule_1": 142}, nil).
		Once()

	counts, err := rig.service.MatchesThisMonth(t.Context(), testTenant, []pulid.ID{"trule_1"})

	require.NoError(t, err)
	assert.Equal(t, 142, counts["trule_1"])
}
