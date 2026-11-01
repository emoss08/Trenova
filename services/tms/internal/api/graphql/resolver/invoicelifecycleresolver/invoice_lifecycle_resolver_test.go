package invoicelifecycleresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSendInvoice_TakesInvoiceSubmitAndSendsTheInvoice(t *testing.T) {
	t.Parallel()

	engine := &resolvertest.RecordingPermissionEngine{}
	sending := invoiceportsmock.NewMockInvoiceSendService(t)
	r := &MutationResolver{&Deps{
		Core:               &resolvercore.Core{PermissionEngine: engine},
		InvoiceSendService: sending,
	}}
	authCtx := resolvertest.UserAuthContext()
	invoiceID := pulid.MustNew("inv_")
	sent := &invoiceports.InvoiceSendResult{Invoice: &invoice.Invoice{ID: invoiceID}}

	var captured *invoiceports.InvoiceSendRequest
	sending.EXPECT().
		Send(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *invoiceports.InvoiceSendRequest, _ *permissionports.RequestActor) (*invoiceports.InvoiceSendResult, error) {
			captured = req
			return sent, nil
		}).
		Once()

	result, err := r.SendInvoice(resolvertest.AuthedContext(t, authCtx), invoiceID.String())

	require.NoError(t, err)
	assert.Same(t, sent, result)
	assert.Equal(t, permission.ResourceInvoice.String(), engine.Request.Resource)
	assert.Equal(t, permission.OpSubmit, engine.Request.Operation)
	assert.Equal(t, invoiceID, captured.InvoiceID)
	assert.Equal(t, authCtx.OrganizationID, captured.TenantInfo.OrgID)
}

func TestPostInvoice_RefusedWithoutInvoicePost(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	r := &MutationResolver{&Deps{
		Core:                  &resolvercore.Core{PermissionEngine: engine},
		InvoicePostingService: invoiceportsmock.NewMockInvoicePostingService(t),
	}}

	_, err := r.PostInvoice(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("inv_").String(),
	)

	resolvertest.RequireRefused(t, err, engine, permission.ResourceInvoice, permission.OpPost)
}

func TestUpdateInvoiceDraft_LeavesAbsentFieldsAloneAndMapsTheLists(t *testing.T) {
	t.Parallel()

	drafts := invoiceportsmock.NewMockInvoiceDraftService(t)
	r := &MutationResolver{&Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &resolvertest.RecordingPermissionEngine{},
		},
		InvoiceDraftService: drafts,
	}}
	invoiceID := pulid.MustNew("inv_")
	attachmentID := pulid.MustNew("doc_")
	memo := "Thanks for your business"

	var captured *invoiceports.UpdateInvoiceDraftRequest
	drafts.EXPECT().
		UpdateDraft(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *invoiceports.UpdateInvoiceDraftRequest, _ *permissionports.RequestActor) (*invoice.Invoice, error) {
			captured = req
			return &invoice.Invoice{ID: invoiceID}, nil
		}).
		Once()

	_, err := r.UpdateInvoiceDraft(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		invoiceID.String(),
		gqlmodel.UpdateInvoiceDraftInput{
			Memo:          &memo,
			EmailTo:       []string{"ap@acme.com"},
			AttachmentIds: []string{attachmentID.String()},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, &memo, captured.Memo)
	assert.Nil(t, captured.EmailSubject)
	assert.Nil(t, captured.EmailCC)
	require.NotNil(t, captured.EmailTo)
	assert.Equal(t, []string{"ap@acme.com"}, *captured.EmailTo)
	require.NotNil(t, captured.AttachmentIDs)
	assert.Equal(t, []pulid.ID{attachmentID}, *captured.AttachmentIDs)
}

func TestInvoiceEmailAttempts_CapsThePageAndSaysWhetherMoreRemain(t *testing.T) {
	t.Parallel()

	queries := invoiceportsmock.NewMockInvoiceQueryService(t)
	r := &QueryResolver{&Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &resolvertest.RecordingPermissionEngine{},
		},
		InvoiceQueryService: queries,
	}}
	limit := 1000
	offset := 0

	var captured invoiceports.ListInvoiceEmailAttemptsRequest
	queries.EXPECT().
		ListEmailAttempts(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req invoiceports.ListInvoiceEmailAttemptsRequest) (*pagination.ListResult[*invoice.EmailAttempt], error) {
			captured = req
			return &pagination.ListResult[*invoice.EmailAttempt]{
				Items:      []*invoice.EmailAttempt{{ID: pulid.MustNew("invea_")}},
				Total:      150,
				TotalExact: true,
			}, nil
		}).
		Once()

	page, err := r.InvoiceEmailAttempts(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("inv_").String(),
		&limit,
		&offset,
	)

	require.NoError(t, err)
	assert.Equal(t, maxAttemptPageLimit, captured.Filter.Pagination.Limit)
	assert.Len(t, page.Items, 1)
	assert.Equal(t, 150, page.Total)
	assert.True(t, page.HasMore)
}
