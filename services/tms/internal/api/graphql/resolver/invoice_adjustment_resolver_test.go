package resolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
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

type invoiceAdjustmentResolverFixture struct {
	orgID       pulid.ID
	buID        pulid.ID
	userID      pulid.ID
	service     *mocks.MockInvoiceAdjustmentService
	permissions *recordingPermissionEngine
	resolver    *Resolver
}

func newInvoiceAdjustmentResolverFixture(t *testing.T) *invoiceAdjustmentResolverFixture {
	t.Helper()

	f := &invoiceAdjustmentResolverFixture{
		orgID:       pulid.MustNew("org_"),
		buID:        pulid.MustNew("bu_"),
		userID:      pulid.MustNew("usr_"),
		service:     mocks.NewMockInvoiceAdjustmentService(t),
		permissions: &recordingPermissionEngine{},
	}
	f.resolver = &Resolver{
		invoiceAdjustmentService: f.service,
		permissionEngine:         f.permissions,
	}

	return f
}

func (f *invoiceAdjustmentResolverFixture) auth(t *testing.T) *authctx.AuthContext {
	t.Helper()

	return &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    f.userID,
		UserID:         f.userID,
		OrganizationID: f.orgID,
		BusinessUnitID: f.buID,
	}
}

func TestQueryResolver_InvoiceAdjustmentApprovals_MapsTypedFiltersAndEncodesKeysetCursors(t *testing.T) {
	t.Parallel()

	f := newInvoiceAdjustmentResolverFixture(t)
	submitter := pulid.MustNew("usr_")
	first := pulid.MustNew("iadj_")
	second := pulid.MustNew("iadj_")
	kind := invoiceadjustment.KindWriteOff
	total := 12
	sort := []pagination.CursorSortField{
		{Field: "sortAt", Direction: "desc"},
		{Field: "id", Direction: "desc"},
	}

	var captured *repositories.ListApprovalQueueRequest
	f.service.EXPECT().
		ListApprovals(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req *repositories.ListApprovalQueueRequest) { captured = req }).
		Return(&pagination.CursorListResult[*repositories.InvoiceAdjustmentApprovalQueueItem]{
			Items: []*repositories.InvoiceAdjustmentApprovalQueueItem{
				{AdjustmentID: first, CreatedAt: 1_600_000_000},
				{AdjustmentID: second, CreatedAt: 1_600_000_000},
			},
			HasNextPage: true,
			TotalCount:  &total,
			CursorSort:  sort,
			CursorValues: [][]any{
				{int64(1_700_000_200), first.String()},
				{int64(1_700_000_100), second.String()},
			},
		}, nil).
		Once()

	ctx := gqlctx.WithAuthContext(t.Context(), f.auth(t))
	pageSize := 25
	query := "  ACME  "
	submitterID := submitter.String()

	conn, err := (&queryResolver{f.resolver}).InvoiceAdjustmentApprovals(
		ctx,
		gqlmodel.InvoiceAdjustmentApprovalsInput{
			First:         &pageSize,
			Query:         &query,
			Kind:          &kind,
			SubmittedByID: &submitterID,
		},
	)
	require.NoError(t, err)

	require.NotNil(t, f.permissions.request)
	assert.Equal(t, permission.ResourceInvoice.String(), f.permissions.request.Resource)
	assert.Equal(t, permission.OpRead, f.permissions.request.Operation)

	require.NotNil(t, captured)
	assert.Equal(t, f.orgID, captured.Filter.TenantInfo.OrgID)
	assert.Equal(t, f.buID, captured.Filter.TenantInfo.BuID)
	assert.Equal(t, "ACME", captured.Filter.Query)
	assert.Equal(t, 25, captured.Cursor.Limit)
	assert.Empty(t, captured.Cursor.After)
	assert.Equal(t, []domaintypes.FieldFilter{
		{Field: "kind", Operator: dbtype.OpEqual, Value: "WriteOff"},
		{Field: "submittedById", Operator: dbtype.OpEqual, Value: submitter.String()},
	}, captured.Filter.FieldFilters)

	require.Len(t, conn.Edges, 2)
	assert.Equal(t, second, conn.Edges[1].Node.AdjustmentID)
	assert.True(t, conn.PageInfo.HasNextPage)
	require.NotNil(t, conn.PageInfo.EndCursor)
	assert.Equal(t, conn.Edges[1].Cursor, *conn.PageInfo.EndCursor)
	require.NotNil(t, conn.TotalCount)
	assert.Equal(t, 12, *conn.TotalCount)

	decoded, err := pagination.DecodeCursor(*conn.PageInfo.EndCursor)
	require.NoError(t, err)
	assert.Equal(t, sort, decoded.Sort)
	assert.Equal(t, second, decoded.ID)
	sortAt, ok := pagination.CursorInt64Value(decoded.Values[0])
	require.True(t, ok)
	assert.Equal(t, int64(1_700_000_100), sortAt)
}

func TestQueryResolver_InvoiceAdjustmentApprovals_SendsNoFiltersWhenUnfiltered(t *testing.T) {
	t.Parallel()

	f := newInvoiceAdjustmentResolverFixture(t)
	var captured *repositories.ListApprovalQueueRequest
	f.service.EXPECT().
		ListApprovals(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req *repositories.ListApprovalQueueRequest) { captured = req }).
		Return(&pagination.CursorListResult[*repositories.InvoiceAdjustmentApprovalQueueItem]{}, nil).
		Once()

	blank := "   "
	empty := ""
	conn, err := (&queryResolver{f.resolver}).InvoiceAdjustmentApprovals(
		gqlctx.WithAuthContext(t.Context(), f.auth(t)),
		gqlmodel.InvoiceAdjustmentApprovalsInput{Query: &blank, SubmittedByID: &empty},
	)
	require.NoError(t, err)

	require.NotNil(t, captured)
	assert.Empty(t, captured.Filter.Query)
	assert.Empty(t, captured.Filter.FieldFilters)
	assert.Equal(t, pagination.DefaultLimit, captured.Cursor.Limit)
	assert.Empty(t, conn.Edges)
	assert.NotNil(t, conn.Edges)
	assert.False(t, conn.PageInfo.HasNextPage)
	assert.Nil(t, conn.PageInfo.EndCursor)
}

func TestQueryResolver_InvoiceAdjustmentApprovals_RejectsMalformedInputBeforeQuerying(t *testing.T) {
	t.Parallel()

	badID := "not-an-id"
	badCursor := "%%%"
	unknownKind := invoiceadjustment.Kind("Refund")

	tests := []struct {
		name  string
		input gqlmodel.InvoiceAdjustmentApprovalsInput
		field string
	}{
		{name: "submitter id", input: gqlmodel.InvoiceAdjustmentApprovalsInput{SubmittedByID: &badID}, field: "submittedById"},
		{name: "cursor", input: gqlmodel.InvoiceAdjustmentApprovalsInput{After: &badCursor}, field: "after"},
		{name: "kind", input: gqlmodel.InvoiceAdjustmentApprovalsInput{Kind: &unknownKind}, field: "kind"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newInvoiceAdjustmentResolverFixture(t)
			_, err := (&queryResolver{f.resolver}).InvoiceAdjustmentApprovals(
				gqlctx.WithAuthContext(t.Context(), f.auth(t)),
				tt.input,
			)

			var validationErr *errortypes.Error
			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, tt.field, validationErr.Field)
		})
	}
}

func TestMutationResolver_ApproveInvoiceAdjustment_RequiresApproveAndCallsTheService(t *testing.T) {
	t.Parallel()

	f := newInvoiceAdjustmentResolverFixture(t)
	adjustmentID := pulid.MustNew("iadj_")
	expected := &invoiceadjustment.InvoiceAdjustment{ID: adjustmentID}
	f.service.EXPECT().
		Approve(
			mock.Anything,
			&services.ApproveInvoiceAdjustmentRequest{
				AdjustmentID: adjustmentID,
				TenantInfo: pagination.TenantInfo{
					OrgID:  f.orgID,
					BuID:   f.buID,
					UserID: f.userID,
				},
			},
			mock.MatchedBy(func(actor *services.RequestActor) bool {
				return actor != nil && actor.UserID == f.userID
			}),
		).
		Return(expected, nil).
		Once()

	result, err := (&mutationResolver{f.resolver}).ApproveInvoiceAdjustment(
		gqlctx.WithAuthContext(t.Context(), f.auth(t)),
		adjustmentID.String(),
	)
	require.NoError(t, err)

	assert.Same(t, expected, result)
	require.NotNil(t, f.permissions.request)
	assert.Equal(t, permission.ResourceInvoice.String(), f.permissions.request.Resource)
	assert.Equal(t, permission.OpApprove, f.permissions.request.Operation)
}

func TestMutationResolver_RejectInvoiceAdjustment_PassesTheReason(t *testing.T) {
	t.Parallel()

	adjustmentID := pulid.MustNew("iadj_")
	reason := "Customer never agreed to the detention"

	tests := []struct {
		name   string
		reason *string
		want   string
	}{
		{name: "with a reason", reason: &reason, want: reason},
		{name: "without a reason", reason: nil, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newInvoiceAdjustmentResolverFixture(t)
			expected := &invoiceadjustment.InvoiceAdjustment{ID: adjustmentID}
			f.service.EXPECT().
				Reject(
					mock.Anything,
					&services.RejectInvoiceAdjustmentRequest{
						AdjustmentID: adjustmentID,
						Reason:       tt.want,
						TenantInfo: pagination.TenantInfo{
							OrgID:  f.orgID,
							BuID:   f.buID,
							UserID: f.userID,
						},
					},
					mock.Anything,
				).
				Return(expected, nil).
				Once()

			result, err := (&mutationResolver{f.resolver}).RejectInvoiceAdjustment(
				gqlctx.WithAuthContext(t.Context(), f.auth(t)),
				gqlmodel.RejectInvoiceAdjustmentInput{
					AdjustmentID: adjustmentID.String(),
					Reason:       tt.reason,
				},
			)
			require.NoError(t, err)

			assert.Same(t, expected, result)
			assert.Equal(t, permission.OpApprove, f.permissions.request.Operation)
		})
	}
}

func TestMutationResolver_InvoiceAdjustmentDecisions_RejectMalformedIDs(t *testing.T) {
	t.Parallel()

	f := newInvoiceAdjustmentResolverFixture(t)
	ctx := gqlctx.WithAuthContext(t.Context(), f.auth(t))
	mutations := &mutationResolver{f.resolver}

	_, approveErr := mutations.ApproveInvoiceAdjustment(ctx, "bogus")
	_, rejectErr := mutations.RejectInvoiceAdjustment(
		ctx,
		gqlmodel.RejectInvoiceAdjustmentInput{AdjustmentID: ""},
	)
	_, getErr := (&queryResolver{f.resolver}).InvoiceAdjustment(ctx, "bogus")

	for _, err := range []error{approveErr, rejectErr, getErr} {
		var validationErr *errortypes.Error
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "adjustmentId", validationErr.Field)
	}
}

func TestInvoiceAdjustmentFieldResolvers_SerializeDecimalsAndBlankRebillStrategy(t *testing.T) {
	t.Parallel()

	r := &Resolver{}
	item := &repositories.InvoiceAdjustmentApprovalQueueItem{
		NetDeltaAmount:        decimal.RequireFromString("-125.5000"),
		RerateVariancePercent: decimal.RequireFromString("0.012500"),
	}

	net, err := (&invoiceAdjustmentApprovalQueueItemResolver{r}).NetDeltaAmount(t.Context(), item)
	require.NoError(t, err)
	assert.Equal(t, "-125.5", net)

	strategy, err := (&invoiceAdjustmentApprovalQueueItemResolver{r}).RebillStrategy(t.Context(), item)
	require.NoError(t, err)
	assert.Nil(t, strategy)

	item.RebillStrategy = invoiceadjustment.RebillStrategyRerate
	strategy, err = (&invoiceAdjustmentApprovalQueueItemResolver{r}).RebillStrategy(t.Context(), item)
	require.NoError(t, err)
	require.NotNil(t, strategy)
	assert.Equal(t, invoiceadjustment.RebillStrategyRerate, *strategy)

	adjustment := &invoiceadjustment.InvoiceAdjustment{}
	adjustmentStrategy, err := (&invoiceAdjustmentResolver{r}).RebillStrategy(t.Context(), adjustment)
	require.NoError(t, err)
	assert.Nil(t, adjustmentStrategy)
}
