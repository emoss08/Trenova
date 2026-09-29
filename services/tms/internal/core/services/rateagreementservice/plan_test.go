package rateagreementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func reviewRequest(agreement *rateagreement.RateAgreement, comment string) *ApprovalActionRequest {
	return &ApprovalActionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  agreement.OrganizationID,
			BuID:   agreement.BusinessUnitID,
			UserID: pulid.MustNew("usr_"),
		},
		EntityID: agreement.ID,
		Comment:  comment,
	}
}

func TestPlanReview_ShowsEachMoveWithoutSavingIt(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		review Review
		from   rateagreement.Status
		to     rateagreement.Status
	}{
		{ReviewSubmit, rateagreement.StatusDraft, rateagreement.StatusInReview},
		{ReviewApprove, rateagreement.StatusInReview, rateagreement.StatusActive},
		{ReviewReject, rateagreement.StatusInReview, rateagreement.StatusDraft},
		{ReviewSuspend, rateagreement.StatusActive, rateagreement.StatusSuspended},
		{ReviewResume, rateagreement.StatusSuspended, rateagreement.StatusActive},
		{ReviewArchive, rateagreement.StatusActive, rateagreement.StatusArchived},
	} {
		t.Run(string(tc.review), func(t *testing.T) {
			t.Parallel()

			agreement := draftAgreement(tc.from)
			svc, repo := serviceFor(t, agreement)

			change, err := svc.PlanReview(t.Context(), tc.review, reviewRequest(agreement, "Checked"))
			require.NoError(t, err)
			assert.Equal(t, tc.from, change.Before.Status)
			assert.Equal(t, tc.to, change.After.Status)
			assert.Equal(t, tc.from, agreement.Status)
			repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		})
	}
}

func TestPlanReview_RefusesWhatTheWriteRefuses(t *testing.T) {
	t.Parallel()

	inReview := draftAgreement(rateagreement.StatusInReview)
	svc, _ := serviceFor(t, inReview)
	_, err := svc.PlanReview(t.Context(), ReviewReject, reviewRequest(inReview, ""))
	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "comment", fieldErr.Field)

	draft := draftAgreement(rateagreement.StatusDraft)
	svc, _ = serviceFor(t, draft)
	_, err = svc.PlanReview(t.Context(), ReviewApprove, reviewRequest(draft, "Looks right"))
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "status", fieldErr.Field)

	_, err = svc.PlanReview(t.Context(), Review("Expire"), reviewRequest(draft, ""))
	require.Error(t, err)
}

func TestReview_AppliesThePlannedMove(t *testing.T) {
	t.Parallel()

	agreement := draftAgreement(rateagreement.StatusActive)
	svc, _ := serviceFor(t, agreement)

	updated, err := svc.Review(t.Context(), ReviewSuspend, reviewRequest(agreement, "Credit hold"))
	require.NoError(t, err)
	assert.Equal(t, rateagreement.StatusSuspended, updated.Status)
	assert.Equal(t, "Credit hold", updated.ReviewComment)
}

func TestPlanCreate_StartsEveryAgreementAsADraftAndSavesNothing(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRateAgreementRepository(t)
	svc := New(Params{Logger: zap.NewNop(), Repo: repo, Validator: NewTestValidator()})

	entity := draftAgreement(rateagreement.StatusActive)
	entity.ID = pulid.Nil
	planned, err := svc.PlanCreate(t.Context(), entity)
	require.NoError(t, err)
	assert.Equal(t, rateagreement.StatusDraft, planned.Status)
	assert.Equal(t, int64(1), planned.CurrentVersionNumber)
	assert.Equal(t, rateagreement.StatusActive, entity.Status)

	entity.Code = ""
	_, err = svc.PlanCreate(t.Context(), entity)
	require.Error(t, err)
}

func agreementRepoReturning(
	t *testing.T,
	agreement *rateagreement.RateAgreement,
) *mocks.MockRateAgreementRepository {
	t.Helper()

	repo := mocks.NewMockRateAgreementRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetRateAgreementByIDRequest")).
		RunAndReturn(func(
			context.Context,
			*repositories.GetRateAgreementByIDRequest,
		) (*rateagreement.RateAgreement, error) {
			copied := *agreement

			return &copied, nil
		}).
		Maybe()

	return repo
}

func laneRule(rate string) *rateagreement.RateAgreementRule {
	templateID := pulid.MustNew("ft_")

	return &rateagreement.RateAgreementRule{
		FormulaTemplateID:     &templateID,
		ID:                    pulid.MustNew("rarl_"),
		Label:                 "GA to FL",
		OriginScopeType:       rategeo.ScopeTypeState,
		OriginScopeValue:      "GA",
		DestinationScopeType:  rategeo.ScopeTypeState,
		DestinationScopeValue: "FL",
		Direction:             rateagreement.DirectionDirectional,
		Rate:                  decimal.NewNullDecimal(decimal.RequireFromString(rate)),
		Status:                rateagreement.RuleStatusActive,
		EffectiveFrom:         100,
	}
}

func TestPlanUpdate_ShowsTheAmendmentAndSavesNothing(t *testing.T) {
	t.Parallel()

	stored := draftAgreement(rateagreement.StatusDraft)
	stored.Rules = []*rateagreement.RateAgreementRule{laneRule("2.00")}
	repo := agreementRepoReturning(t, stored)
	svc := New(Params{Logger: zap.NewNop(), Repo: repo, Validator: NewTestValidator()})

	edited := *stored
	edited.Name = "Acme Freight Agreement 2026"
	changed := *stored.Rules[0]
	changed.Rate = decimal.NewNullDecimal(decimal.RequireFromString("2.25"))
	edited.Rules = []*rateagreement.RateAgreementRule{&changed}

	plan, err := svc.PlanUpdate(t.Context(), &edited)
	require.NoError(t, err)
	assert.Equal(t, "Acme Freight Agreement", plan.Before.Name)
	assert.Equal(t, "Acme Freight Agreement 2026", plan.After.Name)
	assert.Equal(t, []pulid.ID{stored.Rules[0].ID}, plan.SupersededIDs)
	require.Len(t, plan.Inserts, 1)
	assert.Equal(t, "2.25", plan.Inserts[0].Rate.Decimal.String())
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "AmendRules", mock.Anything, mock.Anything)
}

func TestPlanDuplicate_CopiesAsADraftAndSavesNothing(t *testing.T) {
	t.Parallel()

	stored := draftAgreement(rateagreement.StatusActive)
	stored.Rules = []*rateagreement.RateAgreementRule{laneRule("2.00")}
	repo := agreementRepoReturning(t, stored)
	svc := New(Params{Logger: zap.NewNop(), Repo: repo, Validator: NewTestValidator()})

	plan, err := svc.PlanDuplicate(t.Context(), &DuplicateRateAgreementRequest{
		TenantInfo:      pagination.TenantInfo{OrgID: stored.OrganizationID, BuID: stored.BusinessUnitID},
		RateAgreementID: stored.ID,
		Code:            "ACME-2027",
	})
	require.NoError(t, err)
	assert.Equal(t, stored.ID, plan.Original.ID)
	assert.Equal(t, "ACME-2027", plan.Copy.Code)
	assert.Equal(t, rateagreement.StatusDraft, plan.Copy.Status)
	require.Len(t, plan.Copy.Rules, 1)
	assert.True(t, plan.Copy.Rules[0].ID.IsNil())
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestPlanAmendRules_ChecksTheAmendmentAndWritesNothing(t *testing.T) {
	t.Parallel()

	stored := draftAgreement(rateagreement.StatusActive)
	current := laneRule("2.00")
	stored.Rules = []*rateagreement.RateAgreementRule{current}
	repo := agreementRepoReturning(t, stored)
	svc := New(Params{Logger: zap.NewNop(), Repo: repo, Validator: NewTestValidator()})

	successor := laneRule("2.40")
	successor.ID = pulid.Nil
	plan, err := svc.PlanAmendRules(t.Context(), &repositories.AmendRateAgreementRulesRequest{
		TenantInfo:      pagination.TenantInfo{OrgID: stored.OrganizationID, BuID: stored.BusinessUnitID},
		RateAgreementID: stored.ID,
		EffectiveFrom:   1_000,
		SupersededIDs:   []pulid.ID{current.ID},
		Rules:           []*rateagreement.RateAgreementRule{successor},
	})
	require.NoError(t, err)
	assert.Equal(t, stored.ID, plan.Agreement.ID)
	require.Len(t, plan.Superseded, 1)
	assert.Equal(t, current.ID, plan.Superseded[0].ID)
	repo.AssertNotCalled(t, "AmendRules", mock.Anything, mock.Anything)

	_, err = svc.PlanAmendRules(t.Context(), &repositories.AmendRateAgreementRulesRequest{
		TenantInfo:      pagination.TenantInfo{OrgID: stored.OrganizationID, BuID: stored.BusinessUnitID},
		RateAgreementID: stored.ID,
		EffectiveFrom:   1_000,
	})
	require.Error(t, err)
}

func TestRateIncreaseApplicable_RefusesWhatApplyRefuses(t *testing.T) {
	t.Parallel()

	require.Error(t, rateIncreaseApplicable(&RateIncreasePlan{}))
	require.Error(t, rateIncreaseApplicable(&RateIncreasePlan{
		Lines:         []*RateIncreaseLine{{}},
		NegativeCount: 1,
	}))
	require.NoError(t, rateIncreaseApplicable(&RateIncreasePlan{
		Lines: []*RateIncreaseLine{{}},
	}))
}
