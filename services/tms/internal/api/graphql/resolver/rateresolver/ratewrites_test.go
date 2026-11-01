package rateresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateagreementservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type permissionCase struct {
	name      string
	resource  permission.Resource
	operation permission.Operation
	run       func(ctx context.Context, deps *Deps) error
}

func strPtr(value string) *string { return &value }

func ratePermissionCases() []permissionCase {
	agreementID := pulid.MustNew("rag_").String()
	matrixID := pulid.MustNew("rmx_").String()
	shipmentID := pulid.MustNew("shp_").String()
	agreementInput := gqlmodel.RateAgreementInput{Code: "ACME", Name: "Acme"}
	matrixInput := gqlmodel.RateMatrixInput{Code: "ZONES", Name: "Zones", ValueType: "Money"}
	increase := gqlmodel.RateIncreaseInput{Adjustment: &gqlmodel.RateIncreaseAdjustmentInput{}}
	agreement, matrix, quote := permission.ResourceRateAgreement, permission.ResourceRateMatrix,
		permission.ResourceRateQuote
	query := func(deps *Deps) *QueryResolver { return &QueryResolver{deps} }
	mutation := func(deps *Deps) *MutationResolver { return &MutationResolver{deps} }

	return []permissionCase{
		{
			"rateAgreementLanePage",
			agreement,
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := query(d).RateAgreementLanePage(ctx, agreementID, nil, nil)
				return err
			},
		},
		{
			"rateAgreementLaneSummary",
			agreement,
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := query(d).RateAgreementLaneSummary(ctx, agreementID)
				return err
			},
		},
		{
			"rateAgreementVersions",
			agreement,
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := query(d).RateAgreementVersions(ctx, agreementID, nil)
				return err
			},
		},
		{
			"rateAgreementLaneHistory",
			agreement,
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := query(d).RateAgreementLaneHistory(ctx, agreementID, "GA>IL")
				return err
			},
		},
		{
			"previewRateIncrease",
			agreement,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := query(d).PreviewRateIncrease(ctx, increase)
				return err
			},
		},
		{
			"rateMatrixReviewers",
			matrix,
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := query(d).RateMatrixReviewers(ctx, matrixID)
				return err
			},
		},
		{
			"appliedShipmentRateQuote",
			quote,
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := query(d).AppliedShipmentRateQuote(ctx, shipmentID, nil)
				return err
			},
		},
		{
			"shopShipmentCarriers",
			quote,
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := query(d).ShopShipmentCarriers(ctx, shipmentID, nil)
				return err
			},
		},
		{
			"createRateAgreement",
			agreement,
			permission.OpCreate,
			func(ctx context.Context, d *Deps) error {
				_, err := mutation(d).CreateRateAgreement(ctx, agreementInput)
				return err
			},
		},
		{
			"updateRateAgreement",
			agreement,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := mutation(d).UpdateRateAgreement(ctx, agreementID, agreementInput)
				return err
			},
		},
		{
			"duplicateRateAgreement",
			agreement,
			permission.OpDuplicate,
			func(ctx context.Context, d *Deps) error {
				_, err := mutation(d).DuplicateRateAgreement(ctx, agreementID, nil, nil)
				return err
			},
		},
		{
			"submitRateAgreement",
			agreement,
			permission.OpSubmit,
			func(ctx context.Context, d *Deps) error {
				_, err := mutation(d).SubmitRateAgreement(ctx, agreementID, nil)
				return err
			},
		},
		{
			"approveRateAgreement",
			agreement,
			permission.OpApprove,
			func(ctx context.Context, d *Deps) error {
				_, err := mutation(d).ApproveRateAgreement(ctx, agreementID, nil)
				return err
			},
		},
		{
			"rejectRateAgreement",
			agreement,
			permission.OpReject,
			func(ctx context.Context, d *Deps) error {
				_, err := mutation(d).RejectRateAgreement(ctx, agreementID, nil)
				return err
			},
		},
		{
			"suspendRateAgreement",
			agreement,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := mutation(d).SuspendRateAgreement(ctx, agreementID, nil)
				return err
			},
		},
		{
			"resumeRateAgreement",
			agreement,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := mutation(d).ResumeRateAgreement(ctx, agreementID, nil)
				return err
			},
		},
		{
			"archiveRateAgreement",
			agreement,
			permission.OpArchive,
			func(ctx context.Context, d *Deps) error {
				_, err := mutation(d).ArchiveRateAgreement(ctx, agreementID, nil)
				return err
			},
		},
		{
			"applyRateIncrease",
			agreement,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := mutation(d).ApplyRateIncrease(ctx, increase)
				return err
			},
		},
		{"createRateMatrix", matrix, permission.OpCreate, func(ctx context.Context, d *Deps) error {
			_, err := mutation(d).CreateRateMatrix(ctx, matrixInput)
			return err
		}},
		{"updateRateMatrix", matrix, permission.OpUpdate, func(ctx context.Context, d *Deps) error {
			_, err := mutation(d).UpdateRateMatrix(ctx, matrixID, matrixInput)
			return err
		}},
		{
			"replaceRateMatrixCells",
			matrix,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := mutation(d).ReplaceRateMatrixCells(ctx, matrixID, nil)
				return err
			},
		},
		{"submitRateMatrix", matrix, permission.OpSubmit, func(ctx context.Context, d *Deps) error {
			_, err := mutation(d).SubmitRateMatrix(ctx, matrixID, nil)
			return err
		}},
		{
			"approveRateMatrix",
			matrix,
			permission.OpApprove,
			func(ctx context.Context, d *Deps) error {
				_, err := mutation(d).ApproveRateMatrix(ctx, matrixID, nil)
				return err
			},
		},
		{"rejectRateMatrix", matrix, permission.OpReject, func(ctx context.Context, d *Deps) error {
			_, err := mutation(d).RejectRateMatrix(ctx, matrixID, nil)
			return err
		}},
		{
			"requestRateMatrixChanges",
			matrix,
			permission.OpReject,
			func(ctx context.Context, d *Deps) error {
				_, err := mutation(d).RequestRateMatrixChanges(ctx, matrixID, nil)
				return err
			},
		},
		{
			"archiveRateMatrix",
			matrix,
			permission.OpArchive,
			func(ctx context.Context, d *Deps) error {
				_, err := mutation(d).ArchiveRateMatrix(ctx, matrixID, nil)
				return err
			},
		},
		{
			"restoreRateMatrixVersion",
			matrix,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := mutation(d).RestoreRateMatrixVersion(ctx, matrixID, 1)
				return err
			},
		},
	}
}

func TestRateOperations_CheckTheRESTRoutesPermission(t *testing.T) {
	t.Parallel()

	for _, tc := range ratePermissionCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			resolvertest.RequireRefused(t, tc.run(ctx, deps), engine, tc.resource, tc.operation)
		})
	}
}

func TestShopShipmentCarriers_RefusesBestMarginWithoutTheMarginsGroup(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	engine.Granted[permission.ResourceRateQuote.String()+"|"+string(permission.OpRead)] = true
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())
	strategy := ratequoteports.ShopStrategyBestMargin

	_, err := (&QueryResolver{deps}).ShopShipmentCarriers(
		ctx,
		pulid.MustNew("shp_").String(),
		&gqlmodel.ShopShipmentCarriersInput{Strategy: &strategy},
	)

	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)
}

func TestRateAgreementFromInput_MapsTheEditorsSaveIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	customerID := pulid.MustNew("cus_")
	ruleID := pulid.MustNew("ragr_")
	templateID := pulid.MustNew("ft_")
	chargeID := pulid.MustNew("acc_")
	commodityID := pulid.MustNew("com_")
	version, minStops, freeUnits := 4, 2, 30

	entity, err := rateAgreementFromInput(&gqlmodel.RateAgreementInput{
		PartyType:          gqlmodel.RateAgreementPartyTypeCustomer,
		CustomerID:         strPtr(customerID.String()),
		Code:               "ACME-2026",
		Name:               "Acme 2026",
		AgreementType:      gqlmodel.RateAgreementTypeContract,
		EffectiveFrom:      1_767_225_600,
		Currency:           "USD",
		MarginFloorPercent: strPtr("12.5"),
		Version:            &version,
		Rules: []*gqlmodel.RateAgreementRuleInput{{
			ID:                   strPtr(ruleID.String()),
			OriginScopeType:      "State",
			OriginScopeValue:     strPtr("GA"),
			DestinationScopeType: "State",
			CommodityIds:         []string{commodityID.String()},
			FreightClasses:       []string{"70"},
			MinWeight:            strPtr("1000"),
			MinStops:             &minStops,
			FormulaTemplateID:    templateID.String(),
			EffectiveFrom:        1_767_225_600,
		}},
		Accessorials: []*gqlmodel.RateAgreementAccessorialInput{{
			AccessorialChargeID: chargeID.String(),
			Method:              "PerUnit",
			RateUnit:            strPtr("Hour"),
			Amount:              "75",
			FreeUnits:           &freeUnits,
		}},
	}, authCtx)
	require.NoError(t, err)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, &customerID, entity.CustomerID)
	assert.Equal(t, rateagreement.PartyTypeCustomer, entity.PartyType)
	assert.Equal(t, "12.5", entity.MarginFloorPercent.Decimal.String())
	assert.False(t, entity.MaxPayPercentOfSell.Valid)
	assert.Equal(t, int64(4), entity.Version)
	assert.Nil(t, entity.FuelBinding)

	require.Len(t, entity.Rules, 1)
	rule := entity.Rules[0]
	assert.Equal(t, ruleID, rule.ID)
	assert.Equal(t, rateagreement.RuleStatusActive, rule.Status)
	assert.Equal(t, rateagreement.DirectionDirectional, rule.Direction)
	assert.Equal(t, []pulid.ID{commodityID}, rule.CommodityIDs)
	assert.Nil(t, rule.ServiceTypeIDs)
	assert.Equal(t, "1000", rule.MinWeight.Decimal.String())
	assert.Equal(t, int16(2), *rule.MinStops)
	assert.Equal(t, templateID, rule.FormulaTemplateID)

	require.Len(t, entity.Accessorials, 1)
	assert.Equal(t, chargeID, entity.Accessorials[0].AccessorialChargeID)
	assert.Equal(t, "75", entity.Accessorials[0].Amount.String())
	assert.Equal(t, int16(30), *entity.Accessorials[0].FreeUnits)
}

func TestApplyRateAgreementInput_KeepsTheReviewTermsTheInputDoesNotCarry(t *testing.T) {
	t.Parallel()

	approver := pulid.MustNew("usr_")
	approvedAt := int64(1_767_000_000)
	existing := &rateagreement.RateAgreement{
		ApprovedByID:  &approver,
		ApprovedAt:    &approvedAt,
		ReviewComment: "Approved for 2026",
	}

	require.NoError(t, applyRateAgreementInput(existing, &gqlmodel.RateAgreementInput{
		Code: "ACME", Name: "Acme", Currency: "USD", EffectiveFrom: 1,
	}))

	assert.Equal(t, &approver, existing.ApprovedByID)
	assert.Equal(t, &approvedAt, existing.ApprovedAt)
	assert.Equal(t, "Approved for 2026", existing.ReviewComment)
}

func TestRateAgreementFromInput_NamesAMalformedLaneTemplate(t *testing.T) {
	t.Parallel()

	_, err := rateAgreementFromInput(&gqlmodel.RateAgreementInput{
		Rules: []*gqlmodel.RateAgreementRuleInput{{FormulaTemplateID: "nope"}},
	}, resolvertest.UserAuthContext())

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "formulaTemplateId", valErr.Field)
}

func TestRateMatrixCellsFromInput_IndexesABadCellsError(t *testing.T) {
	t.Parallel()

	_, err := rateMatrixCellsFromInput([]*gqlmodel.RateMatrixCellInput{
		{Value: "1.25"},
		{Value: "1.30", D0Min: strPtr("ten")},
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "cells[1].d0Min", multiErr.Errors[0].Field)
}

func TestRateMatrixCellsFromInput_LeavesUnsentKeysEmptyAndCellsDeficitEligible(t *testing.T) {
	t.Parallel()

	cells, err := rateMatrixCellsFromInput([]*gqlmodel.RateMatrixCellInput{
		{D0Key: strPtr("30301"), D1Key: strPtr("60601"), Value: "2.35"},
	})
	require.NoError(t, err)

	require.Len(t, cells, 1)
	assert.Equal(t, "30301", cells[0].D0Key)
	assert.Empty(t, cells[0].D2Key)
	assert.Equal(t, "2.35", cells[0].Value.String())
	assert.False(t, cells[0].D0Min.Valid)
	assert.True(t, cells[0].DeficitEligible)
}

func TestSubmitRateMatrixRequest_NamesAMalformedReviewer(t *testing.T) {
	t.Parallel()

	_, err := submitRateMatrixRequest(
		pulid.MustNew("rmx_"),
		&gqlmodel.SubmitRateMatrixInput{ReviewerID: strPtr("bogus")},
		pagination.TenantInfo{},
	)

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "reviewerId", valErr.Field)
}

func TestRateIncreaseRequestFromInput_ScopesTheIncreaseToTheCallersTenant(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	agreementID := pulid.MustNew("rag_")
	carrier := gqlmodel.RateAgreementPartyTypeCarrier

	req, err := rateIncreaseRequestFromInput(&gqlmodel.RateIncreaseInput{
		AgreementIds:  []string{agreementID.String()},
		PartyType:     &carrier,
		Adjustment:    &gqlmodel.RateIncreaseAdjustmentInput{PercentChange: strPtr("4.5")},
		EffectiveFrom: 1_767_225_600,
	}, tenant)
	require.NoError(t, err)

	assert.Equal(t, tenant, req.TenantInfo)
	assert.Equal(t, []pulid.ID{agreementID}, req.AgreementIDs)
	assert.Equal(t, rateagreement.PartyTypeCarrier, req.PartyType)
	assert.Equal(t, "4.5", req.Adjustment.PercentChange.Decimal.String())
	assert.Equal(t, int64(1_767_225_600), req.EffectiveFrom)
}

func TestShopRequestFromInput_NeverPersists(t *testing.T) {
	t.Parallel()

	carrierID := pulid.MustNew("car_")
	limit := 5

	req, err := shopRequestFromInput(
		pulid.MustNew("shp_").String(),
		&gqlmodel.ShopShipmentCarriersInput{
			CarrierIds: []string{carrierID.String()},
			Limit:      &limit,
		},
		pagination.TenantInfo{},
	)
	require.NoError(t, err)

	assert.False(t, req.Persist)
	assert.Equal(t, []pulid.ID{carrierID}, req.CarrierIDs)
	assert.Equal(t, 5, req.Limit)
	assert.Empty(t, req.Strategy)
}

func TestWithholdShopResult_ClearsEveryMarginWithoutTheMarginsGroup(t *testing.T) {
	t.Parallel()

	margin := decimal.NewNullDecimal(decimal.RequireFromString("150"))
	result := &ratequoteports.ShopResult{
		Strategy:  ratequoteports.ShopStrategyLeastCost,
		SellTotal: decimal.NewNullDecimal(decimal.RequireFromString("2000")),
		Options: []*ratequoteports.ShopOption{{
			CarrierID: pulid.MustNew("car_"),
			Cost:      decimal.RequireFromString("1850"),
			Margin: &ratetypes.MarginVerdict{
				Explanation: "Leaves 150",
				Amount:      decimal.NewFromInt(150),
			},
			Note: "Leaves 150",
			Quote: &ratequote.RateQuote{
				MarginAmount: margin,
				CostAmount:   margin,
				Trace:        &ratetypes.Trace{Totals: ratetypes.Totals{Margin: margin}},
			},
		}},
	}

	withholdShopResult(result, protectedfield.AllGroups()&^
		protectedfield.GroupSet(0).With(protectedfield.Margins))
	model, err := rateShopResultToModel(result)
	require.NoError(t, err)

	assert.Nil(t, model.SellTotal)
	require.Len(t, model.Options, 1)
	option := model.Options[0]
	assert.Nil(t, option.Margin)
	assert.Empty(t, option.Note)
	assert.Equal(t, "1850", option.Cost)
	require.NotNil(t, option.Quote)
	assert.Nil(t, option.Quote.MarginAmount)
	assert.Nil(t, option.Quote.CostAmount)
	totals, _ := option.Quote.Trace["totals"].(map[string]any)
	assert.NotContains(t, totals, "margin")
}

func TestRateIncreasePlanToModel_StatesEveryLanesBeforeAndAfter(t *testing.T) {
	t.Parallel()

	ruleID := pulid.MustNew("ragr_")
	plan, err := rateIncreasePlanToModel(&rateagreementservice.RateIncreasePlan{
		EffectiveFrom:  1_767_225_600,
		AgreementCount: 1,
		SkippedNoMoney: 2,
		Lines: []*rateagreementservice.RateIncreaseLine{{
			RuleID: ruleID,
			Before: tarifftypes.Inputs{},
			After:  tarifftypes.Inputs{},
		}},
	})
	require.NoError(t, err)

	assert.Equal(t, 1_767_225_600, plan.EffectiveFrom)
	assert.Equal(t, 2, plan.SkippedNoMoney)
	require.Len(t, plan.Lines, 1)
	assert.Equal(t, ruleID.String(), plan.Lines[0].RuleID)
	assert.Equal(t, []string{}, plan.Lines[0].Changed)
}

func TestRateAgreementVersionToModel_WithholdsTheMarginChangeWithoutTheGroup(t *testing.T) {
	t.Parallel()

	version := &rateagreement.RateAgreementVersion{
		VersionNumber: 3,
		ChangeSummary: map[string]jsonutils.FieldChange{
			"marginFloorPercent": {From: "10", To: "12"},
			"name":               {From: "Acme", To: "Acme 2026"},
		},
		AccessorialNames: map[string]string{"acc_1": "DET"},
	}
	version.WithholdFields(protectedfield.AllGroups() &^
		protectedfield.GroupSet(0).With(protectedfield.Margins))

	model, err := rateAgreementVersionToModel(version)
	require.NoError(t, err)

	assert.Equal(t, 3, model.VersionNumber)
	assert.Contains(t, model.ChangeSummary, "name")
	assert.NotContains(t, model.ChangeSummary, "marginFloorPercent")
	assert.Equal(t, map[string]any{"acc_1": "DET"}, model.AccessorialNames)
	assert.Nil(t, model.AccessorialTerms)
}
