package invoiceshareresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceshare"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeShareService struct {
	invoiceports.InvoiceShareService
	shared     *invoiceports.ShareInvoiceRequest
	candidates *invoiceports.ListShareCandidatesRequest
}

func (f *fakeShareService) Share(
	_ context.Context,
	req *invoiceports.ShareInvoiceRequest,
	_ *permissionports.RequestActor,
) (*invoiceports.ShareInvoiceResult, error) {
	f.shared = req
	return &invoiceports.ShareInvoiceResult{RecipientCount: len(req.UserIDs)}, nil
}

func (f *fakeShareService) ListCandidates(
	_ context.Context,
	req *invoiceports.ListShareCandidatesRequest,
) (*pagination.ListResult[*invoiceports.ShareCandidate], error) {
	f.candidates = req
	return &pagination.ListResult[*invoiceports.ShareCandidate]{}, nil
}

func TestShareInvoice_ChecksInvoiceReadAndMapsTheRecipients(t *testing.T) {
	t.Parallel()

	engine := &resolvertest.RecordingPermissionEngine{}
	service := &fakeShareService{}
	r := &MutationResolver{&Deps{
		Core:                &resolvercore.Core{PermissionEngine: engine},
		InvoiceShareService: service,
	}}
	authCtx := resolvertest.UserAuthContext()
	invoiceID := pulid.MustNew("inv_")
	userID := pulid.MustNew("usr_")
	note := "  Check the detention line  "

	result, err := r.ShareInvoice(
		resolvertest.AuthedContext(t, authCtx),
		invoiceID.String(),
		gqlmodel.ShareInvoiceInput{
			UserIds: []string{userID.String()},
			Note:    &note,
			Tab:     invoiceshare.ShareTabCharges,
		},
	)

	require.NoError(t, err)
	assert.Equal(t, 1, result.RecipientCount)
	assert.Equal(t, permission.ResourceInvoice.String(), engine.Request.Resource)
	assert.Equal(t, permission.OpRead, engine.Request.Operation)
	assert.Equal(t, invoiceID, service.shared.InvoiceID)
	assert.Equal(t, []pulid.ID{userID}, service.shared.UserIDs)
	assert.Equal(t, "Check the detention line", service.shared.Note)
	assert.Equal(t, invoiceshare.ShareTabCharges, service.shared.Tab)
	assert.Equal(t, authCtx.OrganizationID, service.shared.TenantInfo.OrgID)
}

func TestShareInvoice_RefusedWithoutInvoiceRead(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	r := &MutationResolver{&Deps{
		Core:                &resolvercore.Core{PermissionEngine: engine},
		InvoiceShareService: &fakeShareService{},
	}}

	_, err := r.ShareInvoice(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("inv_").String(),
		gqlmodel.ShareInvoiceInput{Tab: invoiceshare.ShareTabOverview},
	)

	resolvertest.RequireRefused(t, err, engine, permission.ResourceInvoice, permission.OpRead)
}

func TestInvoiceShareCandidates_CapsTheLimitAndTrimsTheQuery(t *testing.T) {
	t.Parallel()

	service := &fakeShareService{}
	r := &QueryResolver{&Deps{
		Core:                &resolvercore.Core{PermissionEngine: &resolvertest.RecordingPermissionEngine{}},
		InvoiceShareService: service,
	}}
	query := "  dana "
	limit := 500

	candidates, err := r.InvoiceShareCandidates(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("inv_").String(),
		&query,
		&limit,
	)

	require.NoError(t, err)
	assert.Empty(t, candidates)
	assert.Equal(t, "dana", service.candidates.Query)
	assert.Equal(t, maxCandidateLimit, service.candidates.Pagination.Limit)
}
