package triageruleservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/triageportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testOrg = pulid.ID("org_test")
	testBU  = pulid.ID("bu_test")
)

type evaluatorRig struct {
	evaluator   *Evaluator
	rules       *triageportsmock.MockTriageRuleRepository
	store       *triageportsmock.MockTriageMessageRepository
	permissions *permissionportsmock.MockPermissionEngine
	messages    *fakeMessages
	filer       *fakeFiler
	drafter     *fakeDrafter
	parties     *fakeParties
	replies     *fakeReplies
	conflicts   *fakeConflicts
}

func newEvaluatorRig(t *testing.T, message *inboundmessage.InboundMessage) *evaluatorRig {
	t.Helper()

	rig := &evaluatorRig{
		rules:       triageportsmock.NewMockTriageRuleRepository(t),
		store:       triageportsmock.NewMockTriageMessageRepository(t),
		permissions: permissionportsmock.NewMockPermissionEngine(t),
		messages:    &fakeMessages{stored: message},
		filer:       &fakeFiler{},
		drafter:     &fakeDrafter{},
		parties:     &fakeParties{},
		replies:     &fakeReplies{latest: map[pulid.ID]*triage.Reply{}},
		conflicts:   &fakeConflicts{},
	}
	rig.evaluator = &Evaluator{
		l:           zap.NewNop(),
		db:          fakeTx{},
		rules:       rig.rules,
		store:       rig.store,
		messages:    rig.messages,
		parties:     rig.parties,
		permissions: rig.permissions,
		filer:       rig.filer,
		drafter:     rig.drafter,
		replies:     rig.replies,
		conflicts:   rig.conflicts,
		now:         func() int64 { return 1_760_000_000 },
	}

	return rig
}

func (r *evaluatorRig) withRules(rules ...*triage.Rule) {
	r.rules.EXPECT().
		List(mock.Anything, triageports.ListRulesRequest{
			TenantInfo:  tenantOf(r.messages.stored),
			EnabledOnly: true,
		}).
		Return(rules, nil).
		Once()
}

func (r *evaluatorRig) expectClaim(rule *triage.Rule) {
	r.store.EXPECT().
		ClaimRule(mock.Anything, mock.MatchedBy(func(req *triageports.ClaimRuleRequest) bool {
			return req.RuleID == rule.ID && req.Action == rule.Action
		})).
		Return(&triageports.ClaimedRule{Claimed: true, Version: 8}, nil).
		Once()
	r.rules.EXPECT().
		RecordMatch(mock.Anything, mock.MatchedBy(func(req *triageports.RecordRuleMatchRequest) bool {
			return req.RuleID == rule.ID
		})).
		Return(nil).
		Once()
}

func waitingPOD() *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		ID:                pulid.ID("imsg_pod"),
		OrganizationID:    testOrg,
		BusinessUnitID:    testBU,
		MailboxID:         pulid.ID("imbx_intake"),
		FromAddress:       "luis@drivers.trenova.example",
		SenderDomain:      "drivers.trenova.example",
		Classification:    inboundmessage.ClassificationProofOfDelivery,
		Status:            inboundmessage.StatusInReview,
		NovaRead:          true,
		MatchedShipmentID: pulid.ID("shp_1"),
		Version:           7,
		Attachments: []*inboundmessage.InboundAttachment{
			{ID: pulid.ID("imsga_1"), DocumentID: pulid.ID("doc_1"), Kind: inboundmessage.AttachmentProofOfDelivery},
		},
	}
}

func ruleFor(id string, action inboundmessage.NovaAction, scope triage.SenderScope, sender string) *triage.Rule {
	return &triage.Rule{
		ID:             pulid.ID(id),
		OrganizationID: testOrg,
		BusinessUnitID: testBU,
		Classification: inboundmessage.ClassificationProofOfDelivery,
		SenderScope:    scope,
		Sender:         sender,
		Action:         action,
		Enabled:        true,
		CreatedAt:      1,
	}
}

func TestApplyRules_TheMostSpecificRuleWins(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	rig := newEvaluatorRig(t, message)
	broad := ruleFor("trule_any", inboundmessage.NovaActionIgnore, triage.SenderAnyone, "")
	specific := ruleFor("trule_domain", inboundmessage.NovaActionFileOnShipment, triage.SenderDomain,
		"drivers.trenova.example")
	rig.withRules(broad, specific)
	rig.filer.filed = 1
	rig.messages.attachments = []*inboundmessage.InboundAttachment{
		{ID: "imsga_1", DocumentID: "doc_1", FiledShipmentID: "shp_1"},
	}
	rig.expectClaim(specific)

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	require.NotNil(t, applied)
	assert.Equal(t, specific.ID, applied.Rule.ID)
	assert.True(t, applied.Settled)
	assert.Equal(t, inboundmessage.StatusActioned, message.Status)
	assert.Equal(t, specific.ID, message.HandledByRuleID)
	assert.Equal(t, inboundmessage.NovaActionFileOnShipment, message.HandledAction)
	assert.Equal(t, int64(8), message.Version)
	assert.True(t, rig.filer.got.ByNova)
	assert.False(t, rig.filer.got.QueueForPay)
}

func TestApplyRules_FileAndQueuePayAsksForPay(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	rig := newEvaluatorRig(t, message)
	rule := ruleFor("trule_pay", inboundmessage.NovaActionFileAndQueuePay, triage.SenderAnyone, "")
	rig.withRules(rule)
	rig.filer.filed = 1
	rig.expectClaim(rule)

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	require.NotNil(t, applied)
	assert.True(t, rig.filer.got.QueueForPay)
}

func TestApplyRules_FilingWaitsWhenNothingWasFiled(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	rig := newEvaluatorRig(t, message)
	rule := ruleFor("trule_file", inboundmessage.NovaActionFileOnShipment, triage.SenderAnyone, "")
	rig.withRules(rule)
	rig.messages.attachments = []*inboundmessage.InboundAttachment{
		{ID: "imsga_1", DocumentID: "doc_1"},
	}

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	assert.Nil(t, applied)
	assert.Equal(t, inboundmessage.StatusInReview, message.Status)
	assert.True(t, message.HandledByRuleID.IsNil())
}

func TestApplyRules_ARetryAfterFilingStillSettles(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	rig := newEvaluatorRig(t, message)
	rule := ruleFor("trule_file", inboundmessage.NovaActionFileOnShipment, triage.SenderAnyone, "")
	rig.withRules(rule)
	rig.messages.attachments = []*inboundmessage.InboundAttachment{
		{ID: "imsga_1", DocumentID: "doc_1", FiledShipmentID: "shp_1"},
	}
	rig.expectClaim(rule)

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	require.NotNil(t, applied)
	assert.True(t, applied.Settled)
}

func TestApplyRules_AnOpenDuplicateHoldsFilingBack(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	rig := newEvaluatorRig(t, message)
	rule := ruleFor("trule_file", inboundmessage.NovaActionFileOnShipment, triage.SenderAnyone, "")
	rig.withRules(rule)
	rig.conflicts.conflicts = []inboundmessage.Conflict{
		{ID: "dup:imsga_1", Kind: inboundmessage.ConflictDuplicateDocument, AttachmentID: "imsga_1"},
	}

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	assert.Nil(t, applied)
	assert.Zero(t, rig.filer.calls)
}

func TestApplyRules_NothingToFileWithoutAShipment(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	message.MatchedShipmentID = pulid.Nil
	rig := newEvaluatorRig(t, message)
	rig.withRules(ruleFor("trule_file", inboundmessage.NovaActionFileOnShipment, triage.SenderAnyone, ""))

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	assert.Nil(t, applied)
	assert.Zero(t, rig.filer.calls)
}

func TestApplyRules_IgnoreSettlesAsIgnored(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	rig := newEvaluatorRig(t, message)
	rule := ruleFor("trule_ignore", inboundmessage.NovaActionIgnore, triage.SenderAnyone, "")
	rig.withRules(rule)
	rig.expectClaim(rule)

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	require.NotNil(t, applied)
	assert.True(t, applied.Settled)
	assert.Equal(t, inboundmessage.StatusIgnored, message.Status)
	assert.Equal(t, ignoredNote, message.ReviewNote)
	assert.True(t, message.ReviewedBy.IsNil())
}

func TestApplyRules_DraftReplyLeavesTheMessageWaiting(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	rig := newEvaluatorRig(t, message)
	rule := ruleFor("trule_draft", inboundmessage.NovaActionDraftReply, triage.SenderAnyone, "")
	rig.withRules(rule)
	rig.expectClaim(rule)

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	require.NotNil(t, applied)
	assert.False(t, applied.Settled)
	assert.Equal(t, 1, rig.drafter.replies)
	assert.Equal(t, inboundmessage.StatusInReview, message.Status)
	assert.Equal(t, rule.ID, message.HandledByRuleID)
}

func TestApplyRules_AnExistingReplyIsNotDraftedTwice(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	rig := newEvaluatorRig(t, message)
	rule := ruleFor("trule_draft", inboundmessage.NovaActionDraftReply, triage.SenderAnyone, "")
	rig.withRules(rule)
	rig.replies.latest = map[pulid.ID]*triage.Reply{message.ID: {ID: "irep_1"}}
	rig.expectClaim(rule)

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	require.NotNil(t, applied)
	assert.Zero(t, rig.drafter.replies)
}

func TestApplyRules_DraftLoadKeepsTheDraftItWrote(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	message.Classification = inboundmessage.ClassificationTender
	rig := newEvaluatorRig(t, message)
	rule := ruleFor("trule_load", inboundmessage.NovaActionDraftLoad, triage.SenderAnyone, "")
	rule.Classification = inboundmessage.ClassificationTender
	rig.withRules(rule)
	rig.drafter.onLoad = func() {
		rig.messages.stored.LoadDraft = &inboundmessage.LoadDraft{Source: "tender.pdf"}
		rig.messages.stored.Version = 9
	}
	rig.expectClaim(rule)

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	require.NotNil(t, applied)
	assert.Equal(t, 1, rig.drafter.loads)
	require.NotNil(t, message.LoadDraft)
	assert.Equal(t, int64(8), message.Version, "the claim's version is the one the row ends on")
}

func TestApplyRules_LinkBySenderLinksTheCustomer(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	rig := newEvaluatorRig(t, message)
	rule := ruleFor("trule_link", inboundmessage.NovaActionLinkBySender, triage.SenderAnyone, "")
	rig.withRules(rule)
	rig.parties.customerID = "cus_northwind"
	rig.expectClaim(rule)

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	require.NotNil(t, applied)
	assert.Equal(t, pulid.ID("cus_northwind"), message.MatchedCustomerID)
	assert.Equal(t, linkedNote, message.MatchReason)
}

func TestApplyRules_LinkBySenderWaitsWhenNobodyMatches(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	rig := newEvaluatorRig(t, message)
	rig.withRules(ruleFor("trule_link", inboundmessage.NovaActionLinkBySender, triage.SenderAnyone, ""))

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	assert.Nil(t, applied)
}

func TestApplyRules_AssignGivesTheMessageAnOwner(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	rig := newEvaluatorRig(t, message)
	rule := ruleFor("trule_assign", inboundmessage.NovaActionAssign, triage.SenderAnyone, "")
	rule.AssigneeID = "usr_dana"
	rig.withRules(rule)
	rig.permissions.EXPECT().Check(mock.Anything, mock.Anything).
		Return(&permissionports.PermissionCheckResult{Allowed: true}, nil).Once()
	rig.expectClaim(rule)

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	require.NotNil(t, applied)
	assert.Equal(t, pulid.ID("usr_dana"), message.OwnerID)
	assert.Equal(t, inboundmessage.StatusInReview, message.Status)
}

func TestApplyRules_AssignSkipsSomeoneWhoCannotReadTheQueue(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	rig := newEvaluatorRig(t, message)
	rule := ruleFor("trule_assign", inboundmessage.NovaActionAssign, triage.SenderAnyone, "")
	rule.AssigneeID = "usr_gone"
	rig.withRules(rule)
	rig.permissions.EXPECT().Check(mock.Anything, mock.Anything).
		Return(&permissionports.PermissionCheckResult{Allowed: false}, nil).Once()

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	assert.Nil(t, applied)
	assert.True(t, message.OwnerID.IsNil())
}

func TestApplyRules_ARetriedActivityDoesNotActOrCountTwice(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	message.HandledByRuleID = "trule_file"
	message.HandledAction = inboundmessage.NovaActionFileOnShipment
	rig := newEvaluatorRig(t, message)
	rule := ruleFor("trule_file", inboundmessage.NovaActionFileOnShipment, triage.SenderAnyone, "")
	rig.rules.EXPECT().
		GetByID(mock.Anything, triageports.GetRuleRequest{
			TenantInfo: tenantOf(message),
			ID:         rule.ID,
		}).
		Return(rule, nil).
		Once()

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	require.NotNil(t, applied)
	assert.True(t, applied.Settled)
	assert.Equal(t, inboundmessage.StatusActioned, message.Status)
	assert.Zero(t, rig.filer.calls)
}

func TestApplyRules_LeavesMessagesNoRuleMayTouch(t *testing.T) {
	t.Parallel()

	tests := map[string]func(*inboundmessage.InboundMessage){
		"unread":      func(m *inboundmessage.InboundMessage) { m.NovaRead = false },
		"quarantined": func(m *inboundmessage.InboundMessage) { m.Status = inboundmessage.StatusQuarantined },
		"snoozed":     func(m *inboundmessage.InboundMessage) { m.Parked = inboundmessage.ParkingSnoozed },
		"unreadable":  func(m *inboundmessage.InboundMessage) { m.FailureText = "could not read" },
		"settled":     func(m *inboundmessage.InboundMessage) { m.Status = inboundmessage.StatusActioned },
	}

	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			message := waitingPOD()
			change(message)
			rig := newEvaluatorRig(t, message)

			applied, err := rig.evaluator.ApplyRules(t.Context(), message)

			require.NoError(t, err)
			assert.Nil(t, applied)
		})
	}
}

func TestApplyRules_ADisabledRuleNeverMatches(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	rig := newEvaluatorRig(t, message)
	rule := ruleFor("trule_off", inboundmessage.NovaActionIgnore, triage.SenderAnyone, "")
	rule.Enabled = false
	rig.withRules(rule)

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	assert.Nil(t, applied)
}

func TestApplyRules_ALostClaimReplaysTheWinner(t *testing.T) {
	t.Parallel()

	message := waitingPOD()
	rig := newEvaluatorRig(t, message)
	rule := ruleFor("trule_ignore", inboundmessage.NovaActionIgnore, triage.SenderAnyone, "")
	rig.withRules(rule)
	rig.store.EXPECT().ClaimRule(mock.Anything, mock.Anything).
		Return(&triageports.ClaimedRule{}, nil).Once()
	rig.messages.stored = func() *inboundmessage.InboundMessage {
		claimed := *message
		claimed.HandledByRuleID = rule.ID
		claimed.HandledAction = rule.Action
		claimed.Version = 9

		return &claimed
	}()
	rig.rules.EXPECT().GetByID(mock.Anything, mock.Anything).Return(rule, nil).Once()

	applied, err := rig.evaluator.ApplyRules(t.Context(), message)

	require.NoError(t, err)
	require.NotNil(t, applied)
	assert.Equal(t, int64(9), message.Version)
	assert.Equal(t, inboundmessage.StatusIgnored, message.Status)
}
