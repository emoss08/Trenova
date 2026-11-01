package invoiceadjustmentresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceadjustmentportsmock"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpdateInvoiceAdjustmentDraft_TakesInvoiceUpdateAndMapsTheLines(t *testing.T) {
	t.Parallel()

	engine := &resolvertest.RecordingPermissionEngine{}
	drafts := invoiceadjustmentportsmock.NewMockInvoiceAdjustmentDraftService(t)
	r := &MutationResolver{&Deps{
		Core:                          &resolvercore.Core{PermissionEngine: engine},
		InvoiceAdjustmentDraftService: drafts,
	}}
	adjustmentID := pulid.MustNew("iadj_")
	lineID := pulid.MustNew("invl_")
	documentID := pulid.MustNew("doc_")
	reason := "Detention billed twice"

	var captured *invoiceadjustmentports.UpdateDraftInvoiceAdjustmentRequest
	drafts.EXPECT().
		UpdateDraft(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *invoiceadjustmentports.UpdateDraftInvoiceAdjustmentRequest,
			_ *permissionports.RequestActor,
		) (*invoiceadjustment.InvoiceAdjustment, error) {
			captured = req
			return &invoiceadjustment.InvoiceAdjustment{ID: adjustmentID}, nil
		}).
		Once()

	_, err := r.UpdateInvoiceAdjustmentDraft(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		adjustmentID.String(),
		gqlmodel.InvoiceAdjustmentDraftInput{
			Kind:                  invoiceadjustment.KindCreditOnly,
			Reason:                &reason,
			ReferencedDocumentIds: []string{documentID.String()},
			Lines: []*gqlmodel.InvoiceAdjustmentLineInput{{
				OriginalLineID: lineID.String(),
				CreditQuantity: "1",
				CreditAmount:   "125.50",
				RebillQuantity: "0",
				RebillAmount:   "0",
			}},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, permission.ResourceInvoice.String(), engine.Request.Resource)
	assert.Equal(t, permission.OpUpdate, engine.Request.Operation)
	assert.Equal(t, adjustmentID, captured.AdjustmentID)
	assert.Equal(t, invoiceadjustment.KindCreditOnly, captured.Kind)
	assert.Equal(t, invoiceadjustment.RebillStrategyCloneExact, captured.RebillStrategy)
	assert.Equal(t, reason, captured.Reason)
	assert.Equal(t, []pulid.ID{documentID}, captured.ReferencedDocumentIDs)
	require.Len(t, captured.Lines, 1)
	assert.Equal(t, lineID, captured.Lines[0].OriginalLineID)
	assert.True(t, decimal.RequireFromString("125.50").Equal(captured.Lines[0].CreditAmount))
	assert.Equal(t, map[string]any{}, captured.Lines[0].ReplacementPayload)
}

func TestUpdateInvoiceAdjustmentDraft_RejectsAnAmountThatIsNotANumber(t *testing.T) {
	t.Parallel()

	r := &MutationResolver{&Deps{
		Core: &resolvercore.Core{PermissionEngine: &resolvertest.RecordingPermissionEngine{}},
		InvoiceAdjustmentDraftService: invoiceadjustmentportsmock.NewMockInvoiceAdjustmentDraftService(
			t,
		),
	}}

	_, err := r.UpdateInvoiceAdjustmentDraft(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("iadj_").String(),
		gqlmodel.InvoiceAdjustmentDraftInput{
			Kind: invoiceadjustment.KindCreditOnly,
			Lines: []*gqlmodel.InvoiceAdjustmentLineInput{{
				OriginalLineID: pulid.MustNew("invl_").String(),
				CreditQuantity: "1",
				CreditAmount:   "lots",
				RebillQuantity: "0",
				RebillAmount:   "0",
			}},
		},
	)

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "lines.creditAmount", validation.Field)
}

func TestSubmitInvoiceAdjustmentDraft_RefusedWithoutInvoiceUpdate(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	r := &MutationResolver{&Deps{
		Core: &resolvercore.Core{PermissionEngine: engine},
		InvoiceAdjustmentDraftService: invoiceadjustmentportsmock.NewMockInvoiceAdjustmentDraftService(
			t,
		),
	}}

	_, err := r.SubmitInvoiceAdjustmentDraft(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("iadj_").String(),
	)

	resolvertest.RequireRefused(t, err, engine, permission.ResourceInvoice, permission.OpUpdate)
}

func TestInvoiceAdjustmentBatches_NarrowsToTheStatusAndCapsThePage(t *testing.T) {
	t.Parallel()

	f := newInvoiceAdjustmentResolverFixture(t)
	r := &QueryResolver{f.resolver}
	limit := 500
	query := "  acme "
	status := "Failed"

	var captured pagination.QueryOptions
	f.service.EXPECT().
		ListBatches(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			opts pagination.QueryOptions,
		) (*pagination.ListResult[*invoiceadjustment.InvoiceAdjustmentBatch], error) {
			captured = opts
			return &pagination.ListResult[*invoiceadjustment.InvoiceAdjustmentBatch]{
				Items:      []*invoiceadjustment.InvoiceAdjustmentBatch{{ID: pulid.MustNew("iadjb_")}},
				Total:      1,
				TotalExact: true,
			}, nil
		}).
		Once()

	page, err := r.InvoiceAdjustmentBatches(
		resolvertest.AuthedContext(t, f.auth(t)),
		gqlmodel.InvoiceAdjustmentQueueInput{Limit: &limit, Query: &query, Status: &status},
	)

	require.NoError(t, err)
	assert.Equal(t, maxQueueLimit, captured.Pagination.Limit)
	assert.Equal(t, "acme", captured.Query)
	assert.Equal(t, f.orgID, captured.TenantInfo.OrgID)
	assert.Equal(t, []domaintypes.FieldFilter{
		{Field: "status", Operator: dbtype.OpEqual, Value: "Failed"},
	}, captured.FieldFilters)
	assert.Len(t, page.Items, 1)
	assert.True(t, page.TotalExact)
}

func TestInvoiceAdjustmentPreviewResolver_ErrorsAreKeyedListsOfMessages(t *testing.T) {
	t.Parallel()

	errs, err := (&InvoiceAdjustmentPreviewResolver{}).Errors(
		t.Context(),
		&invoiceadjustmentports.InvoiceAdjustmentPreview{
			Errors: map[string][]string{"lines[0].creditAmount": {"Exceeds the line"}},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, map[string]any{"lines[0].creditAmount": []any{"Exceeds the line"}}, errs)
}
