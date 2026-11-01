package formulatemplateresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func wireValue(t *testing.T, value any) any {
	t.Helper()
	raw, ok := value.(rules.RawJSON)
	require.True(t, ok, "%T", value)
	var decoded any
	require.NoError(t, sonic.Unmarshal(raw, &decoded))
	return decoded
}

func requireValidationField(t *testing.T, err error, field string) {
	t.Helper()
	var multi *errortypes.MultiError
	if errors.As(err, &multi) {
		require.NotEmpty(t, multi.Errors)
		assert.Equal(t, field, multi.Errors[0].Field)
		return
	}
	var single *errortypes.Error
	require.True(t, errors.As(err, &single), "a validation error on %s: %v", field, err)
	assert.Equal(t, field, single.Field)
}

func requireAuthorizationError(t *testing.T, err error) {
	t.Helper()
	var authzErr *errortypes.AuthorizationError
	require.True(t, errors.As(err, &authzErr), "the call must be refused: %v", err)
}

func TestFormulaTemplateOperations_CheckTheirPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ft_").String()
	source := perMileSource
	calls := []struct {
		name      string
		resource  permission.Resource
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"formulaTemplateUsage", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaTemplateUsage(ctx, id)
				return err
			}},
		{"formulaTemplateVersions", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaTemplateVersions(ctx, id, nil, nil)
				return err
			}},
		{"formulaTemplateVersion", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaTemplateVersion(ctx, id, 1)
				return err
			}},
		{"formulaTemplateScheduledVersions", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaTemplateScheduledVersions(ctx, id)
				return err
			}},
		{"formulaTemplateVersionDiff", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaTemplateVersionDiff(ctx, id, 1, 2)
				return err
			}},
		{"formulaTemplateLineage", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaTemplateLineage(ctx, id)
				return err
			}},
		{"formulaTemplateReviews", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaTemplateReviews(ctx, id)
				return err
			}},
		{"formulaTemplateReviewers", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaTemplateReviewers(ctx, id)
				return err
			}},
		{"formulaTemplateReadiness", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaTemplateReadiness(ctx, id)
				return err
			}},
		{"formulaTemplateReviewDiff", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaTemplateReviewDiff(ctx, id)
				return err
			}},
		{"formulaTemplateStudioContext", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaTemplateStudioContext(ctx)
				return err
			}},
		{"formulaTemplateTableCells", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaTemplateTableCells(
					ctx,
					gqlmodel.FormulaTableCellsInput{},
				)
				return err
			}},
		{"formulaTemplateStandards", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaTemplateStandards(ctx)
				return err
			}},
		{"explainFormulaTemplate", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).ExplainFormulaTemplate(
					ctx,
					gqlmodel.FormulaExplainInput{
						Draft: &gqlmodel.FormulaDraftInput{Source: source},
					},
				)
				return err
			}},
		{"describeFormulaTemplateInputs", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).DescribeFormulaTemplateInputs(
					ctx,
					gqlmodel.FormulaDraftInput{Source: source},
				)
				return err
			}},
		{"suggestFormulaTemplateTests", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).SuggestFormulaTemplateTests(
					ctx,
					gqlmodel.FormulaDraftInput{Source: source},
				)
				return err
			}},
		{"explainFormulaTemplateTest", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).ExplainFormulaTemplateTest(
					ctx,
					gqlmodel.FormulaExplainTestInput{
						Draft:    &gqlmodel.FormulaDraftInput{Source: source},
						TestName: "a test",
					},
				)
				return err
			}},
		{"formulaTemplateReviewNote", permission.ResourceFormulaTemplate, permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaTemplateReviewNote(ctx, id)
				return err
			}},
		{"createFormulaTemplate", permission.ResourceFormulaTemplate, permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateFormulaTemplate(
					ctx,
					gqlmodel.CreateFormulaTemplateInput{Source: source},
				)
				return err
			}},
		{"updateFormulaTemplate", permission.ResourceFormulaTemplate, permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateFormulaTemplate(
					ctx,
					id,
					gqlmodel.UpdateFormulaTemplateInput{Source: source, Version: 1},
				)
				return err
			}},
		{"duplicateFormulaTemplates", permission.ResourceFormulaTemplate, permission.OpDuplicate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).DuplicateFormulaTemplates(ctx, []string{id})
				return err
			}},
		{"bulkUpdateFormulaTemplateStatus", permission.ResourceFormulaTemplate, permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).BulkUpdateFormulaTemplateStatus(
					ctx,
					gqlmodel.BulkUpdateFormulaTemplateStatusInput{
						TemplateIds: []string{id},
						Status:      formulatemplate.StatusInactive,
					},
				)
				return err
			}},
		{"importFormulaTemplates", permission.ResourceFormulaTemplate, permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ImportFormulaTemplates(
					ctx,
					gqlmodel.ImportFormulaTemplatesInput{ExportVersion: "2.0"},
				)
				return err
			}},
		{"installStandardFormulaTemplates", permission.ResourceFormulaTemplate, permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).InstallStandardFormulaTemplates(ctx)
				return err
			}},
		{"rollbackFormulaTemplate", permission.ResourceFormulaTemplate, permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).RollbackFormulaTemplate(
					ctx,
					id,
					gqlmodel.RollbackFormulaVersionInput{TargetVersion: 1},
				)
				return err
			}},
		{"forkFormulaTemplate", permission.ResourceFormulaTemplate, permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ForkFormulaTemplate(
					ctx,
					id,
					gqlmodel.ForkFormulaTemplateInput{NewName: "Copy"},
				)
				return err
			}},
		{
			"updateFormulaTemplateVersionTags",
			permission.ResourceFormulaTemplate,
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateFormulaTemplateVersionTags(
					ctx,
					id,
					1,
					[]formulatemplate.VersionTag{formulatemplate.VersionTagStable},
				)
				return err
			},
		},
		{"updateFormulaTemplateVersionEffectiveDate", permission.ResourceFormulaTemplate,
			permission.OpApprove,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateFormulaTemplateVersionEffectiveDate(
					ctx,
					id,
					1,
					nil,
				)
				return err
			}},
		{"submitFormulaTemplate", permission.ResourceFormulaTemplate, permission.OpSubmit,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).SubmitFormulaTemplate(
					ctx,
					id,
					gqlmodel.SubmitFormulaReviewInput{},
				)
				return err
			}},
		{"approveFormulaTemplate", permission.ResourceFormulaTemplate, permission.OpApprove,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ApproveFormulaTemplate(
					ctx,
					id,
					gqlmodel.ApproveFormulaTemplateInput{},
				)
				return err
			}},
		{"rejectFormulaTemplate", permission.ResourceFormulaTemplate, permission.OpReject,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).RejectFormulaTemplate(ctx, id, nil)
				return err
			}},
		{"requestFormulaTemplateChanges", permission.ResourceFormulaTemplate, permission.OpReject,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).RequestFormulaTemplateChanges(ctx, id, nil)
				return err
			}},
		{"openFormulaTemplateAssistantThread", permission.ResourceFormulaTemplate,
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).OpenFormulaTemplateAssistantThread(ctx, &id)
				return err
			}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(t, err, engine, call.resource, call.operation)
		})
	}
}

func TestAuthoringOperations_AcceptCreateOrUpdate(t *testing.T) {
	t.Parallel()

	calls := map[string]func(ctx context.Context, f *fixture) error{
		"formulaTemplatePreview": func(ctx context.Context, f *fixture) error {
			_, err := f.query().FormulaTemplatePreview(
				ctx,
				gqlmodel.FormulaTemplatePreviewInput{Source: brokenSource},
			)
			return err
		},
		"formulaTemplateScaffoldTest": func(ctx context.Context, f *fixture) error {
			_, err := f.query().FormulaTemplateScaffoldTest(
				ctx,
				gqlmodel.FormulaTemplateScaffoldTestInput{Source: perMileSource},
			)
			return err
		},
		"formulaTemplateBacktest": func(ctx context.Context, f *fixture) error {
			_, err := f.query().FormulaTemplateBacktest(
				ctx,
				"not-an-id",
				gqlmodel.FormulaTemplateBacktestInput{},
			)
			return err
		},
	}

	for name, run := range calls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			refused := newFixture(t, fixtureOptions{granted: map[string]bool{
				"formula_template|read": true,
			}})
			err := run(refused.ctx(t), refused)
			requireAuthorizationError(t, err)
			require.Len(t, refused.engine.Requests, 2)
			assert.Equal(t, permission.OpCreate, refused.engine.Requests[0].Operation)
			assert.Equal(t, permission.OpUpdate, refused.engine.Requests[1].Operation)

			updater := newFixture(t, fixtureOptions{granted: map[string]bool{
				"formula_template|update": true,
			}})
			err = run(updater.ctx(t), updater)
			var authzErr *errortypes.AuthorizationError
			assert.False(t, errors.As(err, &authzErr), "an updater may author: %v", err)
		})
	}
}

func TestApprovalImpact_AcceptsAnApprover(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: map[string]bool{
		"formula_template|approve": true,
	}})

	_, err := f.query().FormulaTemplateApprovalImpact(f.ctx(t), "not-an-id", nil)

	require.Error(t, err)
	var authzErr *errortypes.AuthorizationError
	assert.False(t, errors.As(err, &authzErr), "an approver may see the impact: %v", err)
	require.Len(t, f.engine.Requests, 3)
	assert.Equal(t, permission.OpApprove, f.engine.Requests[2].Operation)
}

func TestFormulaTemplateBacktest_NeedsShipmentRead(t *testing.T) {
	t.Parallel()

	grants := authorGrants()
	delete(grants, "shipment|read")
	f := newFixture(t, fixtureOptions{granted: grants})

	_, err := f.query().FormulaTemplateBacktest(
		f.ctx(t),
		pulid.MustNew("ft_").String(),
		gqlmodel.FormulaTemplateBacktestInput{},
	)

	requireAuthorizationError(t, err)
	last := f.engine.Requests[len(f.engine.Requests)-1]
	assert.Equal(t, permission.ResourceShipment.String(), last.Resource)
}

func TestCreateFormulaTemplate_StoresTheCheckedSourceInTheCallersTenant(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	f.repo.On("Create", mock.Anything, mock.Anything).Return(returnsTemplate, nil)
	f.versions.On("Create", mock.Anything, mock.Anything).Return(returnsVersion, nil)
	description := "Linehaul per loaded mile"

	created, err := f.mutation().
		CreateFormulaTemplate(f.ctx(t), gqlmodel.CreateFormulaTemplateInput{
			Source:      perMileSource,
			Description: &description,
		})

	require.NoError(t, err)
	assert.Equal(t, "Per Mile", created.Name)
	assert.Equal(t, description, created.Description)
	assert.Equal(t, formulatemplate.StatusDraft, created.Status)
	assert.Equal(t, f.authCtx.OrganizationID, created.OrganizationID)
	assert.Equal(t, f.authCtx.BusinessUnitID, created.BusinessUnitID)
	assert.EqualValues(t, 1, created.TestCount)

	inputs, err := (&FormulaTemplateResolver{f.deps}).Inputs(f.ctx(t), created)
	require.NoError(t, err)
	require.Len(t, inputs, 2)
	assert.Equal(t, "rate", inputs[0].Name)
	assert.True(t, inputs[0].Required)
	assert.False(t, inputs[0].HasDefault)
	assert.Equal(t, "minimumCharge", inputs[1].Name)
	assert.True(t, inputs[1].HasDefault)
}

func TestCreateFormulaTemplate_ASourceThatDoesNotCheckIsRefusedAtItsRange(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})

	_, err := f.mutation().CreateFormulaTemplate(
		f.ctx(t),
		gqlmodel.CreateFormulaTemplateInput{Source: brokenSource},
	)

	requireValidationField(t, err, "source")
	f.repo.AssertNotCalled(t, "Create")
}

func TestFormulaTemplatePreview_AnswersDiagnosticsBillAndTests(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	f.rater.result = &tariffports.RateResult{
		Bill: &tarifftypes.Bill{Total: decimal.NewFromInt(285), Lines: []tarifftypes.Line{}},
		WireBill: &rules.TariffBill{
			Lines:            []rules.TariffLine{},
			FreightTotal:     rules.TariffQuantity{Value: decimal.NewFromInt(285), Unit: "$"},
			AccessorialTotal: rules.TariffQuantity{Value: decimal.Zero, Unit: "$"},
			Total:            rules.TariffQuantity{Value: decimal.NewFromInt(285), Unit: "$"},
		},
	}

	result, err := f.query().FormulaTemplatePreview(f.ctx(t), gqlmodel.FormulaTemplatePreviewInput{
		Source: perMileSource,
		Inputs: map[string]any{"rate": map[string]any{"$q": "2.85", "u": "$/mi"}},
	})

	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, []any{}, wireValue(t, result.Diagnostics))
	bill, ok := wireValue(t, result.Bill).(map[string]any)
	require.True(t, ok)
	assert.Equal(t, map[string]any{"$q": "285", "u": "$"}, bill["total"],
		"the client reads the engine's bill: totals are quantities with their unit")
	tests, ok := wireValue(t, result.Tests).([]any)
	require.True(t, ok)
	require.Len(t, tests, 1)
	assert.Equal(t, "passed", tests[0].(map[string]any)["status"])
	assert.Equal(t, []any{}, wireValue(t, result.Trace))
	require.Len(t, f.rater.requests, 1)
	assert.JSONEq(t, `{"$q":"2.85","u":"$/mi"}`, string(f.rater.requests[0].Inputs["rate"]))
	assert.Empty(t, f.records.requests)
}

func TestFormulaTemplatePreview_ASourceThatDoesNotCheckAnswersItsDiagnostics(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})

	result, err := f.query().FormulaTemplatePreview(
		f.ctx(t),
		gqlmodel.FormulaTemplatePreviewInput{Source: brokenSource},
	)

	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, wireValue(t, result.Diagnostics))
	assert.Nil(t, result.Bill)
	assert.Empty(t, f.rater.requests)
}

func TestFormulaTemplatePreview_ARealShipmentNeedsShipmentRead(t *testing.T) {
	t.Parallel()

	grants := authorGrants()
	delete(grants, "shipment|read")
	f := newFixture(t, fixtureOptions{granted: grants})
	shipmentID := pulid.MustNew("shp_").String()

	_, err := f.query().FormulaTemplatePreview(f.ctx(t), gqlmodel.FormulaTemplatePreviewInput{
		Source:     perMileSource,
		ShipmentID: &shipmentID,
	})
	requireAuthorizationError(t, err)
	assert.Empty(t, f.records.requests)

	allowed := newFixture(t, fixtureOptions{granted: authorGrants()})
	allowed.rater.result = &tariffports.RateResult{
		Bill: &tarifftypes.Bill{Lines: []tarifftypes.Line{}},
	}
	_, err = allowed.query().FormulaTemplatePreview(allowed.ctx(t),
		gqlmodel.FormulaTemplatePreviewInput{Source: perMileSource, ShipmentID: &shipmentID})
	require.NoError(t, err)
	require.Len(t, allowed.records.requests, 1)
	assert.Equal(t, shipmentID, allowed.records.requests[0].ShipmentID.String())
}

func TestFormulaTemplateScaffoldTest_RefusesARequestWithoutAName(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})

	_, err := f.query().FormulaTemplateScaffoldTest(
		f.ctx(t),
		gqlmodel.FormulaTemplateScaffoldTestInput{
			Source:     perMileSource,
			ShipmentID: pulid.MustNew("shp_").String(),
		},
	)

	requireValidationField(t, err, "name")
	assert.Empty(t, f.records.requests)
}

func TestFormulaTemplateStudioContext_ServesTheOrganisationsTariffContext(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})

	studio, err := f.query().FormulaTemplateStudioContext(f.ctx(t))

	require.NoError(t, err)
	tariffCtx, ok := wireValue(t, studio.Context).(map[string]any)
	require.True(t, ok)
	settings, ok := tariffCtx["settings"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "mi", settings["distanceUnit"])
	assert.Equal(t, "12:40:1767225600", studio.TablesStamp)
	assert.NotNil(t, studio.Libraries)
}

func TestFormulaTemplateTableCells_AnswersTheStampAndRefusesNoNeeds(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: map[string]bool{"formula_template|read": true}})

	cells, err := f.query().FormulaTemplateTableCells(f.ctx(t), gqlmodel.FormulaTableCellsInput{
		Needs: []*gqlmodel.FormulaTableNeedInput{{Table: "Missing", Keys: []string{"A"}}},
	})
	require.NoError(t, err)
	assert.Equal(t, "12:40:1767225600", cells.Stamp)
	assert.Empty(t, cells.Tables)

	_, err = f.query().FormulaTemplateTableCells(f.ctx(t), gqlmodel.FormulaTableCellsInput{
		Needs: []*gqlmodel.FormulaTableNeedInput{},
	})
	requireValidationField(t, err, "needs")
}

func TestFormulaTemplateStandards_ListsTariffSources(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})

	standards, err := f.query().FormulaTemplateStandards(f.ctx(t))

	require.NoError(t, err)
	require.NotEmpty(t, standards)
	for _, standard := range standards {
		assert.NotEmpty(t, standard.Source, standard.Name)
	}
}

func TestFormulaTemplateUsage_PassesTheTenant(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	templateID := pulid.MustNew("ft_")
	f.repo.On("CountUsages", mock.Anything, mock.MatchedBy(
		func(req *formulatemplateports.GetTemplateUsageRequest) bool {
			return req.TemplateID == templateID &&
				req.TenantInfo.OrgID == f.authCtx.OrganizationID &&
				req.TenantInfo.BuID == f.authCtx.BusinessUnitID
		},
	)).Return(&formulatemplateports.GetTemplateUsageResponse{
		InUse:  true,
		Usages: []formulatemplateports.TemplateUsageCount{{Type: "Shipment", Count: 3}},
	}, nil)

	usage, err := f.query().FormulaTemplateUsage(f.ctx(t), templateID.String())

	require.NoError(t, err)
	assert.True(t, usage.InUse)
	assert.Equal(t, 3, usage.Usages[0].Count)
}

func TestFormulaTemplateVersionDiff_NeedsTwoDifferentPositiveVersions(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	id := pulid.MustNew("ft_").String()

	_, err := f.query().FormulaTemplateVersionDiff(f.ctx(t), id, 0, 2)
	requireValidationField(t, err, "fromVersion")

	_, err = f.query().FormulaTemplateVersionDiff(f.ctx(t), id, 2, 2)
	requireValidationField(t, err, "toVersion")
}

func TestExplainFormulaTemplate_PassesTheDraftInputsAndShipment(t *testing.T) {
	t.Parallel()

	studio := &recordingStudio{}
	f := newFixture(t, fixtureOptions{granted: authorGrants(), studio: studio})
	shipmentID := pulid.MustNew("shp_").String()
	name := "Per Mile"

	result, err := f.query().ExplainFormulaTemplate(f.ctx(t), gqlmodel.FormulaExplainInput{
		Draft:      &gqlmodel.FormulaDraftInput{Name: &name, Source: perMileSource},
		Inputs:     map[string]any{"rate": map[string]any{"$q": "2.85", "u": "$/mi"}},
		ShipmentID: &shipmentID,
	})

	require.NoError(t, err)
	require.Len(t, studio.explained, 1)
	request := studio.explained[0]
	assert.Equal(t, perMileSource, request.Draft.Source)
	assert.Equal(t, "Per Mile", request.Draft.Name)
	assert.Equal(t, shipmentID, request.ShipmentID.String())
	assert.Contains(t, request.Inputs, "rate")
	assert.Equal(t, f.authCtx.OrganizationID, request.TenantInfo.OrgID)
	require.Len(t, studio.actors, 1)
	assert.Equal(t, f.authCtx.UserID, studio.actors[0].UserID)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "It charges by the mile.", result.Items[0].Says)
	assert.Equal(t, &gqlmodel.FormulaSourceRange{From: 3, To: 9}, result.Items[0].Range)
	assert.Nil(t, result.Preview)
}

func TestExplainFormulaTemplate_RefusesAnEmptySourceAndAMissingAssistant(t *testing.T) {
	t.Parallel()

	studio := &recordingStudio{}
	f := newFixture(t, fixtureOptions{granted: authorGrants(), studio: studio})
	_, err := f.query().ExplainFormulaTemplate(f.ctx(t), gqlmodel.FormulaExplainInput{
		Draft: &gqlmodel.FormulaDraftInput{Source: ""},
	})
	requireValidationField(t, err, "source")
	assert.Empty(t, studio.explained)

	missing := newFixture(t, fixtureOptions{granted: authorGrants()})
	_, err = missing.query().ExplainFormulaTemplate(missing.ctx(t), gqlmodel.FormulaExplainInput{
		Draft: &gqlmodel.FormulaDraftInput{Source: perMileSource},
	})
	var businessErr *errortypes.BusinessError
	require.True(t, errors.As(err, &businessErr), "%v", err)
}

func TestStudioQueries_NeedTheAssistant(t *testing.T) {
	t.Parallel()

	studio := &recordingStudio{}
	f := newFixture(t, fixtureOptions{
		granted: map[string]bool{"formula_template|read": true},
		studio:  studio,
	})
	draft := gqlmodel.FormulaDraftInput{Source: perMileSource}

	_, err := f.query().DescribeFormulaTemplateInputs(f.ctx(t), draft)
	requireAuthorizationError(t, err)
	_, err = f.query().SuggestFormulaTemplateTests(f.ctx(t), draft)
	requireAuthorizationError(t, err)
	_, err = f.query().ExplainFormulaTemplateTest(f.ctx(t), gqlmodel.FormulaExplainTestInput{
		Draft:    &draft,
		TestName: "a short haul bills its miles",
	})
	requireAuthorizationError(t, err)
	assert.Empty(t, studio.described)
	assert.Empty(t, studio.suggested)
	assert.Empty(t, studio.tested)
}

func TestSuggestFormulaTemplateTests_AnswersRunnableBlocks(t *testing.T) {
	t.Parallel()

	studio := &recordingStudio{}
	f := newFixture(t, fixtureOptions{granted: authorGrants(), studio: studio})

	result, err := f.query().SuggestFormulaTemplateTests(
		f.ctx(t),
		gqlmodel.FormulaDraftInput{Source: perMileSource},
	)

	require.NoError(t, err)
	require.Len(t, studio.suggested, 1)
	assert.Equal(t, perMileSource, studio.suggested[0].Draft.Source)
	require.Len(t, result.Tests, 1)
	assert.Equal(t, "passed", result.Tests[0].Status)
	assert.Equal(t, []any{}, wireValue(t, result.Tests[0].Failures))
}

func TestDescribeFormulaTemplateInputs_AnswersADocPerInput(t *testing.T) {
	t.Parallel()

	studio := &recordingStudio{}
	f := newFixture(t, fixtureOptions{granted: authorGrants(), studio: studio})

	result, err := f.query().DescribeFormulaTemplateInputs(
		f.ctx(t),
		gqlmodel.FormulaDraftInput{Source: perMileSource},
	)

	require.NoError(t, err)
	assert.Equal(t, map[string]any{"rate": "What the contract pays per mile"}, result.Descriptions)
}

func TestExplainFormulaTemplateTest_NeedsTheTestsName(t *testing.T) {
	t.Parallel()

	studio := &recordingStudio{}
	f := newFixture(t, fixtureOptions{granted: authorGrants(), studio: studio})
	draft := &gqlmodel.FormulaDraftInput{Source: perMileSource}

	_, err := f.query().ExplainFormulaTemplateTest(
		f.ctx(t),
		gqlmodel.FormulaExplainTestInput{Draft: draft, TestName: "  "},
	)
	requireValidationField(t, err, "testName")
	assert.Empty(t, studio.tested)

	result, err := f.query().ExplainFormulaTemplateTest(
		f.ctx(t),
		gqlmodel.FormulaExplainTestInput{Draft: draft, TestName: "a short haul bills its miles"},
	)
	require.NoError(t, err)
	require.Len(t, studio.tested, 1)
	assert.Equal(t, "a short haul bills its miles", studio.tested[0].TestName)
	assert.Equal(t, gqlmodel.FormulaTestVerdict("unclear"), result.Verdict)
	assert.Nil(t, result.Test)
}

func TestFormulaTemplateReviewNote_IncludesTheBacktestOnlyForShipmentReaders(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ft_").String()
	grants := authorGrants()
	delete(grants, "shipment|read")
	studio := &recordingStudio{}
	f := newFixture(t, fixtureOptions{granted: grants, studio: studio})
	note, err := f.query().FormulaTemplateReviewNote(f.ctx(t), id)
	require.NoError(t, err)
	require.Len(t, studio.notes, 1)
	assert.False(t, studio.notes[0].IncludeBacktest)
	assert.Equal(t, "Raised the rate.", note.Note)
	assert.Nil(t, note.Backtest)

	studio = &recordingStudio{}
	f = newFixture(t, fixtureOptions{granted: authorGrants(), studio: studio})
	_, err = f.query().FormulaTemplateReviewNote(f.ctx(t), id)
	require.NoError(t, err)
	require.Len(t, studio.notes, 1)
	assert.True(t, studio.notes[0].IncludeBacktest)
}

func TestSubmitFormulaTemplate_RefusesAMalformedReviewer(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	reviewer := "not-an-id"

	_, err := f.mutation().SubmitFormulaTemplate(
		f.ctx(t),
		pulid.MustNew("ft_").String(),
		gqlmodel.SubmitFormulaReviewInput{ReviewerID: &reviewer},
	)

	requireValidationField(t, err, "reviewerId")
}

func TestOpenFormulaTemplateAssistantThread_OpensTheTemplatesConversation(t *testing.T) {
	t.Parallel()

	threadID := pulid.MustNew("athr_")
	assistant := &recordingAssistant{opened: &assistantports.PageThread{
		Thread: &conversation.Thread{
			ID:     threadID,
			Status: conversation.ThreadStatusActive,
			Origin: conversation.ThreadOriginFormula,
		},
		Agent: assistantports.PageAgent{ID: pulid.MustNew("agd_"), Name: "Formula assistant"},
	}}
	f := newFixture(t, fixtureOptions{granted: authorGrants(), assistant: assistant})
	templateID := pulid.MustNew("ft_")
	f.repo.On("GetByID", mock.Anything, mock.MatchedBy(
		func(req formulatemplateports.GetFormulaTemplateByIDRequest) bool {
			return req.TemplateID == templateID
		},
	)).Return(&formulatemplate.FormulaTemplate{ID: templateID, Name: "Per Mile"}, nil)
	raw := templateID.String()

	opened, err := f.mutation().OpenFormulaTemplateAssistantThread(f.ctx(t), &raw)

	require.NoError(t, err)
	require.Len(t, assistant.requests, 1)
	request := assistant.requests[0]
	assert.Equal(t, conversation.ThreadOriginFormula, request.Origin)
	assert.Equal(t, agent.SubjectFormulaTemplate, request.SubjectType)
	assert.Equal(t, templateID, request.SubjectID)
	assert.Equal(t, f.authCtx.UserID, assistant.actors[0].UserID)
	assert.Equal(t, threadID.String(), opened.Thread.ID)
	assert.Equal(t, "Active", opened.Thread.Status)
	assert.Equal(t, "Formula assistant", opened.Agent.Name)
	assert.Empty(t, opened.Agent.Starters)
}

func TestOpenFormulaTemplateAssistantThread_ANewTemplateHasNoSubject(t *testing.T) {
	t.Parallel()

	assistant := &recordingAssistant{opened: &assistantports.PageThread{
		Thread: &conversation.Thread{ID: pulid.MustNew("athr_")},
	}}
	f := newFixture(t, fixtureOptions{granted: authorGrants(), assistant: assistant})

	_, err := f.mutation().OpenFormulaTemplateAssistantThread(f.ctx(t), nil)

	require.NoError(t, err)
	require.Len(t, assistant.requests, 1)
	assert.True(t, assistant.requests[0].SubjectID.IsNil())
}

func TestOpenFormulaTemplateAssistantThread_RefusesWhatItCannotFindOrRead(t *testing.T) {
	t.Parallel()

	assistant := &recordingAssistant{}
	f := newFixture(t, fixtureOptions{granted: authorGrants(), assistant: assistant})
	f.repo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Formula template not found"))
	raw := pulid.MustNew("ft_").String()

	_, err := f.mutation().OpenFormulaTemplateAssistantThread(f.ctx(t), &raw)
	require.Error(t, err)
	assert.Empty(t, assistant.requests)

	malformed := "nope"
	_, err = f.mutation().OpenFormulaTemplateAssistantThread(f.ctx(t), &malformed)
	requireValidationField(t, err, "templateId")

	noAssistant := newFixture(t, fixtureOptions{
		granted:   map[string]bool{"formula_template|read": true},
		assistant: assistant,
	})
	_, err = noAssistant.mutation().OpenFormulaTemplateAssistantThread(noAssistant.ctx(t), nil)
	requireAuthorizationError(t, err)
	assert.Empty(t, assistant.requests)
}
