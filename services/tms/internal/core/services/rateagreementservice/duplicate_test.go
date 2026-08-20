package rateagreementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// A duplicate is the renewal workflow: the whole contract — lanes, breaks,
// accessorials, fuel terms — copied as a fresh draft with none of the
// original's identity, history, or approvals.
func TestDuplicate_CopiesTheContractAsAFreshDraft(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	submittedAt := int64(50)

	original := draftAgreement(rateagreement.StatusActive)
	original.SubmittedByID = &userID
	original.SubmittedAt = &submittedAt
	original.ApprovedByID = &userID
	original.ApprovedAt = &submittedAt
	original.CurrentVersionNumber = 7
	original.Version = 4

	priorRuleID := pulid.MustNew("rarl_")
	templateID := pulid.MustNew("ft_")
	original.Rules = []*rateagreement.RateAgreementRule{
		{
			ID:                    pulid.MustNew("rarl_"),
			Label:                 "GA to FL",
			Status:                rateagreement.RuleStatusActive,
			OriginScopeType:       "State",
			OriginScopeValue:      "GA",
			DestinationScopeType:  "State",
			DestinationScopeValue: "FL",
			Direction:             rateagreement.DirectionDirectional,
			FormulaTemplateID:     &templateID,
			Rate:                  decimal.NewNullDecimal(decimal.NewFromInt(2)),
			EffectiveFrom:         100,
			SupersedesRuleID:      &priorRuleID,
			Breaks: []*rateagreement.RateAgreementRuleBreak{
				{ID: pulid.MustNew("ragb_"), Rate: decimal.NewFromInt(1)},
			},
		},
	}
	original.Accessorials = []*rateagreement.RateAgreementAccessorial{
		{
			ID:                  pulid.MustNew("raac_"),
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              "Flat",
			Amount:              decimal.NewFromInt(25),
		},
	}

	repo := mocks.NewMockRateAgreementRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetRateAgreementByIDRequest")).
		RunAndReturn(func(
			_ context.Context,
			req *repositories.GetRateAgreementByIDRequest,
		) (*rateagreement.RateAgreement, error) {
			require.True(t, req.IncludeChildren, "a duplicate needs the whole contract")
			return original, nil
		}).
		Once()

	var created *rateagreement.RateAgreement
	repo.EXPECT().
		Create(mock.Anything, mock.AnythingOfType("*rateagreement.RateAgreement")).
		RunAndReturn(func(
			_ context.Context,
			entity *rateagreement.RateAgreement,
		) (*rateagreement.RateAgreement, error) {
			created = entity
			entity.ID = pulid.MustNew("ragr_")
			return entity, nil
		}).
		Once()
	repo.EXPECT().
		CreateVersion(mock.Anything, mock.AnythingOfType("*rateagreement.RateAgreementVersion")).
		RunAndReturn(func(
			_ context.Context,
			version *rateagreement.RateAgreementVersion,
		) (*rateagreement.RateAgreementVersion, error) {
			return version, nil
		}).
		Once()

	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Twice()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		Validator:    NewTestValidator(),
		AuditService: audit,
	})

	duplicate, err := svc.Duplicate(t.Context(), &DuplicateRateAgreementRequest{
		TenantInfo:      tenantOf(original),
		RateAgreementID: original.ID,
	}, userID)

	require.NoError(t, err)
	require.NotNil(t, created)

	assert.Equal(t, rateagreement.StatusDraft, duplicate.Status, "a copy starts unreviewed")
	assert.NotEqual(t, original.ID, duplicate.ID)
	assert.NotEqual(t, original.Code, created.Code, "the code must stay unique")
	assert.Nil(t, created.SubmittedByID)
	assert.Nil(t, created.ApprovedByID)
	assert.EqualValues(t, 1, created.CurrentVersionNumber)
	assert.EqualValues(t, 0, created.Version)

	require.Len(t, created.Rules, 1)
	copiedRule := created.Rules[0]
	assert.True(t, copiedRule.ID.IsNil())
	assert.Nil(t, copiedRule.SupersedesRuleID, "a copy has no lineage — its history starts here")
	assert.Equal(t, "ST:GA>ST:FL", copiedRule.LaneKey)
	require.Len(t, copiedRule.Breaks, 1)
	assert.True(t, copiedRule.Breaks[0].ID.IsNil())

	require.Len(t, created.Accessorials, 1)
	assert.True(t, created.Accessorials[0].ID.IsNil())

	// The original was returned by the repository and must not have been
	// mutated into the copy.
	assert.Equal(t, rateagreement.StatusActive, original.Status)
	assert.False(t, original.Rules[0].ID.IsNil())
}
