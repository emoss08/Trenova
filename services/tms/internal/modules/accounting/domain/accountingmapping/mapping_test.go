package accountingmapping_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func roleMapping(key string) *accountingmapping.AccountingMapping {
	return &accountingmapping.AccountingMapping{
		ConnectionID: pulid.MustNew("acctc_"),
		TargetType:   accountingsync.TargetAccountRole,
		TrenovaKey:   key,
		TargetLabel:  "Accounts receivable",
		State:        accountingmapping.MappingStateUnmatched,
	}
}

func proposal(externalID string, confidence float64) *accountingmapping.Proposal {
	return &accountingmapping.Proposal{
		ExternalID:   externalID,
		ExternalName: "Account " + externalID,
		Source:       accountingmapping.MappingSourceSuggested,
		Confidence:   confidence,
		Reason:       "Same name",
		Candidates: []accountingmapping.MappingCandidate{
			{ExternalID: externalID, Name: "Account " + externalID, Score: confidence},
			{ExternalID: "90", Name: "Other", Score: 0.4},
		},
	}
}

func TestApplyProposalMovesAnUnmatchedRowToProposed(t *testing.T) {
	t.Parallel()

	m := roleMapping(accountingsync.AccountRoleAR)
	changed := m.ApplyProposal(proposal("84", 0.97))

	assert.True(t, changed)
	assert.Equal(t, accountingmapping.MappingStateProposed, m.State)
	assert.Equal(t, "84", m.ExternalID)
	require.NotNil(t, m.Confidence)
	assert.InDelta(t, 0.97, *m.Confidence, 1e-9)
	assert.True(t, m.Prechecked())
	assert.Len(t, m.Signals.Candidates, 2)
	assert.False(
		t,
		m.ApplyProposal(proposal("84", 0.97)),
		"the same proposal again changes nothing",
	)
}

func TestApplyProposalWithoutAChoiceKeepsTheRowUnmatchedWithCandidates(t *testing.T) {
	t.Parallel()

	m := roleMapping(accountingsync.AccountRoleAR)
	p := proposal("", 0)
	p.Candidates = []accountingmapping.MappingCandidate{
		{ExternalID: "1", Score: 0.5},
		{ExternalID: "2", Score: 0.4},
	}
	m.ApplyProposal(p)

	assert.Equal(t, accountingmapping.MappingStateUnmatched, m.State)
	assert.Empty(t, m.ExternalID)
	assert.Nil(t, m.Confidence)
	assert.Len(t, m.Signals.Candidates, 2)
}

func TestApplyProposalCapsAModelPickBelowThePrecheckBand(t *testing.T) {
	t.Parallel()

	m := roleMapping(accountingsync.AccountRoleRevenue)
	p := proposal("79", 0.99)
	p.Source = accountingmapping.MappingSourceModel
	m.ApplyProposal(p)

	require.NotNil(t, m.Confidence)
	assert.InDelta(t, accountingmapping.MaxModelConfidence, *m.Confidence, 1e-9)
	assert.False(t, m.Prechecked())
}

func TestApplyProposalKeepsAtMostThreeCandidates(t *testing.T) {
	t.Parallel()

	m := roleMapping(accountingsync.AccountRoleAR)
	p := proposal("1", 0.8)
	p.Candidates = []accountingmapping.MappingCandidate{
		{ExternalID: "1"}, {ExternalID: "2"}, {ExternalID: "3"}, {ExternalID: "4"},
	}
	m.ApplyProposal(p)

	assert.Len(t, m.Signals.Candidates, accountingmapping.MaxCandidates)
}

func TestApplyProposalNeverTouchesAConfirmedOrDeliberateRow(t *testing.T) {
	t.Parallel()

	confirmed := roleMapping(accountingsync.AccountRoleAR)
	confirmed.Confirm(&accountingmapping.Choice{
		ExternalID: "84",
		Source:     accountingmapping.MappingSourceSuggested,
		ActorID:    pulid.MustNew("usr_"),
		At:         100,
	})
	assert.False(t, confirmed.ApplyProposal(proposal("12", 0.99)))
	assert.Equal(t, "84", confirmed.ExternalID)

	manual := roleMapping(accountingsync.AccountRoleAR)
	manual.State = accountingmapping.MappingStateProposed
	manual.ExternalID = "84"
	manual.Source = accountingmapping.MappingSourceManual
	assert.False(t, manual.ApplyProposal(proposal("12", 0.99)))
}

func TestRejectRemembersTheCandidateAndItIsNeverProposedAgain(t *testing.T) {
	t.Parallel()

	m := roleMapping(accountingsync.AccountRoleAR)
	m.ApplyProposal(proposal("84", 0.97))

	require.NoError(t, m.Reject())
	assert.Equal(t, accountingmapping.MappingStateUnmatched, m.State)
	assert.Empty(t, m.ExternalID)
	assert.True(t, m.WasRejected("84"))
	for _, candidate := range m.Signals.Candidates {
		assert.NotEqual(t, "84", candidate.ExternalID)
	}

	m.ApplyProposal(proposal("84", 0.99))
	assert.Equal(
		t,
		accountingmapping.MappingStateUnmatched,
		m.State,
		"a rejected candidate is not proposed again",
	)
	for _, candidate := range m.Signals.Candidates {
		assert.NotEqual(t, "84", candidate.ExternalID)
	}

	require.ErrorIs(t, m.Reject(), accountingmapping.ErrMappingNotProposed)
}

func TestConfirmingARejectedCandidateForgetsTheRejection(t *testing.T) {
	t.Parallel()

	m := roleMapping(accountingsync.AccountRoleAR)
	m.ApplyProposal(proposal("84", 0.97))
	require.NoError(t, m.Reject())

	userID := pulid.MustNew("usr_")
	m.Confirm(&accountingmapping.Choice{
		ExternalID:   "84",
		ExternalName: "Accounts Receivable (A/R)",
		Source:       accountingmapping.MappingSourceManual,
		ActorID:      userID,
		At:           200,
	})

	assert.Equal(t, accountingmapping.MappingStateConfirmed, m.State)
	assert.Equal(t, userID, m.ConfirmedByID)
	require.NotNil(t, m.ConfirmedAt)
	assert.Equal(t, int64(200), *m.ConfirmedAt)
	assert.False(t, m.WasRejected("84"))
}

func TestClearReturnsARowToUnmatched(t *testing.T) {
	t.Parallel()

	m := roleMapping(accountingsync.AccountRoleAR)
	require.ErrorIs(t, m.Clear(), accountingmapping.ErrMappingNotSet)

	m.Confirm(
		&accountingmapping.Choice{
			ExternalID: "84",
			Source:     accountingmapping.MappingSourceManual,
			At:         1,
		},
	)
	require.NoError(t, m.Clear())
	assert.Equal(t, accountingmapping.MappingStateUnmatched, m.State)
	assert.Empty(t, m.ExternalID)
	assert.Nil(t, m.ConfirmedAt)
	assert.True(t, m.Rescorable())
	assert.True(t, m.WasRejected("84"), "a record a person cleared is not proposed again")

	assert.False(t, m.ApplyProposal(proposal("84", 0.99)) && m.ExternalID == "84")
	assert.Empty(t, m.ExternalID)
}

func TestModelReviewIsAskedOncePerCandidateSet(t *testing.T) {
	t.Parallel()

	m := roleMapping(accountingsync.AccountRoleRevenue)
	m.ApplyProposal(proposal("", 0))
	require.NotEmpty(t, m.Signals.Candidates)
	assert.True(t, m.NeedsModelReview())

	m.MarkModelReviewed()
	assert.False(t, m.NeedsModelReview(), "the same candidates are not sent to the model twice")

	m.Signals.Candidates = append(
		m.Signals.Candidates,
		accountingmapping.MappingCandidate{ExternalID: "99", Name: "Sales"},
	)
	assert.True(t, m.NeedsModelReview(), "a new candidate is worth another look")
}

func TestAModelPickSurvivesARescoreWhileItsRecordIsStillACandidate(t *testing.T) {
	t.Parallel()

	m := roleMapping(accountingsync.AccountRoleRevenue)
	model := proposal("79", 0.88)
	model.Source = accountingmapping.MappingSourceModel
	m.ApplyProposal(model)
	require.Equal(t, accountingmapping.MappingStateProposed, m.State)

	unsure := &accountingmapping.Proposal{Candidates: []accountingmapping.MappingCandidate{
		{ExternalID: "79", Name: "Freight Income", Score: 0.6},
		{ExternalID: "90", Name: "Other", Score: 0.4},
	}}
	assert.True(t, m.KeepsModelPick(unsure))

	gone := &accountingmapping.Proposal{Candidates: unsure.Candidates[1:]}
	assert.False(
		t,
		m.KeepsModelPick(gone),
		"the pick goes once its record is no longer a candidate",
	)

	sure := proposal("80", 0.97)
	assert.False(t, m.KeepsModelPick(sure), "a sure deterministic match replaces the model's pick")
}

func TestRequiredTargetsAreTheReceivablesMinimum(t *testing.T) {
	t.Parallel()

	assert.True(
		t,
		accountingprovider.IsRequiredTarget(
			accountingsync.TargetAccountRole,
			accountingsync.AccountRoleAR,
		),
	)
	assert.True(
		t,
		accountingprovider.IsRequiredTarget(
			accountingsync.TargetAccountRole,
			accountingsync.AccountRoleRevenue,
		),
	)
	assert.True(
		t,
		accountingprovider.IsRequiredTarget(
			accountingsync.TargetAccountRole,
			accountingsync.AccountRoleDeposit,
		),
	)
	assert.True(t, accountingprovider.IsRequiredTarget(accountingsync.TargetLineType, "Freight"))
	assert.False(t, accountingprovider.IsRequiredTarget(accountingsync.TargetLineType, "Memo"))
	assert.False(
		t,
		accountingprovider.IsRequiredTarget(
			accountingsync.TargetAccountRole,
			accountingsync.AccountRoleAP,
		),
	)
	assert.False(t, accountingprovider.IsRequiredTarget(accountingsync.TargetCustomer, ""))
}

func TestMappingValidationKeysEachTargetOneWay(t *testing.T) {
	t.Parallel()

	valid := roleMapping(accountingsync.AccountRoleAR)
	multiErr := errortypes.NewMultiError()
	valid.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())

	badKey := roleMapping("Bogus")
	multiErr = errortypes.NewMultiError()
	badKey.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	customerWithoutRecord := &accountingmapping.AccountingMapping{
		ConnectionID: pulid.MustNew("acctc_"),
		TargetType:   accountingsync.TargetCustomer,
		TargetLabel:  "Acme",
		State:        accountingmapping.MappingStateUnmatched,
	}
	multiErr = errortypes.NewMultiError()
	customerWithoutRecord.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	proposedWithoutRecord := roleMapping(accountingsync.AccountRoleAR)
	proposedWithoutRecord.State = accountingmapping.MappingStateProposed
	multiErr = errortypes.NewMultiError()
	proposedWithoutRecord.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestTargetTypesMapToTheirProviderKind(t *testing.T) {
	t.Parallel()

	qbo := accountingprovider.MustProfile(integration.TypeQuickBooksOnline)
	for _, target := range accountingsync.AllMappingTargetTypes() {
		assert.True(t, qbo.KindFor(target).IsValid(), target)
		if target.KeyedByObject() {
			assert.Empty(t, target.Keys(), target)
		} else {
			assert.NotEmpty(t, target.Keys(), target)
		}
	}
	assert.Equal(
		t,
		accountingprovider.ReferenceKindVendor,
		qbo.KindFor(accountingsync.TargetCarrier),
	)
	assert.Equal(
		t,
		accountingprovider.ReferenceKindItem,
		qbo.KindFor(accountingsync.TargetAccessorialCharge),
	)

	xero := accountingprovider.MustProfile(integration.TypeXero)
	assert.Equal(
		t,
		accountingprovider.ReferenceKindAccount,
		xero.KindFor(accountingsync.TargetAccessorialCharge),
	)
	assert.Equal(
		t,
		accountingprovider.ReferenceKindAccount,
		xero.KindFor(accountingsync.TargetLineType),
	)
	assert.False(t, xero.Offers(accountingsync.TargetPaymentTerm))
	assert.False(t, xero.Offers(accountingsync.TargetPaymentMethod))
	assert.False(t, xero.OffersKey(accountingsync.TargetAccountRole, accountingsync.AccountRoleAR))
	assert.True(
		t,
		xero.IsRequiredTarget(accountingsync.TargetAccountRole, accountingsync.AccountRoleDeposit),
	)
	assert.False(t, xero.SupportsMode(accountingprovider.SyncModeLedger))
	assert.True(t, xero.SupportsMode(accountingprovider.SyncModeDocument))
}

func TestReferenceObjectUsability(t *testing.T) {
	t.Parallel()

	removedAt := int64(5)
	cases := []struct {
		name string
		obj  accountingmapping.AccountingReferenceObject
		want bool
	}{
		{
			"active account",
			accountingmapping.AccountingReferenceObject{
				Kind:   accountingprovider.ReferenceKindAccount,
				Active: true,
			},
			true,
		},
		{
			"inactive",
			accountingmapping.AccountingReferenceObject{
				Kind: accountingprovider.ReferenceKindAccount,
			},
			false,
		},
		{
			"removed",
			accountingmapping.AccountingReferenceObject{
				Kind:      accountingprovider.ReferenceKindAccount,
				Active:    true,
				RemovedAt: &removedAt,
			},
			false,
		},
		{
			"service item",
			accountingmapping.AccountingReferenceObject{
				Kind:     accountingprovider.ReferenceKindItem,
				Active:   true,
				ItemType: "Service",
			},
			true,
		},
		{
			"category item",
			accountingmapping.AccountingReferenceObject{
				Kind:     accountingprovider.ReferenceKindItem,
				Active:   true,
				ItemType: "Category",
			},
			false,
		},
		{
			"group item",
			accountingmapping.AccountingReferenceObject{
				Kind:     accountingprovider.ReferenceKindItem,
				Active:   true,
				ItemType: "Group",
			},
			false,
		},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, tc.obj.Usable(), tc.name)
	}
}

func TestConnectStartsANewConnectionAtTheModeStepAndKeepsAFinishedOne(t *testing.T) {
	t.Parallel()

	fresh := &accountingconnection.AccountingConnection{}
	fresh.Connect(pulid.MustNew("usr_"), accountingconnection.TokenGrant{}, 10)
	assert.Equal(t, accountingconnection.SetupStepMode, fresh.SetupStep)

	finished := &accountingconnection.AccountingConnection{
		SetupStep: accountingconnection.SetupStepComplete,
	}
	finished.Connect(pulid.MustNew("usr_"), accountingconnection.TokenGrant{}, 10)
	assert.Equal(t, accountingconnection.SetupStepComplete, finished.SetupStep)
}
