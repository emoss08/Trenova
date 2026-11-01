package invoicestatementresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoicerunports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStatementService struct {
	invoicerunports.InvoiceStatementService
	billed *invoicerunports.BillStatementNowRequest
	run    *invoicerun.InvoiceRun
}

func (f *fakeStatementService) BillStatementNow(
	_ context.Context,
	req *invoicerunports.BillStatementNowRequest,
	_ *permissionports.RequestActor,
) (*invoicerunports.CommitInvoiceRunResult, error) {
	f.billed = req
	return &invoicerunports.CommitInvoiceRunResult{Run: f.run, SuccessCount: 1}, nil
}

func TestBillStatementNow_TakesInvoiceRunApproveAndMapsTheHeldItems(t *testing.T) {
	t.Parallel()

	engine := &resolvertest.RecordingPermissionEngine{}
	run := &invoicerun.InvoiceRun{ID: pulid.MustNew("ivr_"), Number: "RUN-7"}
	service := &fakeStatementService{run: run}
	r := &MutationResolver{&Deps{
		Core:                    &resolvercore.Core{PermissionEngine: engine},
		InvoiceStatementService: service,
	}}
	customerID := pulid.MustNew("cus_")
	heldID := pulid.MustNew("bqi_")
	reason := "Customer asked for an early bill"

	result, err := r.BillStatementNow(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		customerID.String(),
		gqlmodel.BillStatementNowInput{
			Reason: &reason,
			Exclude: []*gqlmodel.StatementExclusionInput{
				{BillingQueueItemID: heldID.String(), Reason: "Held back"},
			},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, 1, result.SuccessCount)
	assert.Equal(t, permission.ResourceInvoiceRun.String(), engine.Request.Resource)
	assert.Equal(t, permission.OpApprove, engine.Request.Operation)
	assert.Equal(t, customerID, service.billed.CustomerID)
	assert.Equal(t, reason, service.billed.Reason)
	assert.Equal(t, []invoicerunports.StatementExclusion{
		{BillingQueueItemID: heldID, Reason: "Held back"},
	}, service.billed.Exclude)

	runID, err := (&CommitInvoiceRunResultResolver{}).RunID(t.Context(), result)
	require.NoError(t, err)
	assert.Equal(t, run.ID.String(), runID)
}

func TestBillStatementNow_RefusedWithoutInvoiceRunApprove(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	r := &MutationResolver{&Deps{
		Core:                    &resolvercore.Core{PermissionEngine: engine},
		InvoiceStatementService: &fakeStatementService{},
	}}

	_, err := r.BillStatementNow(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("cus_").String(),
		gqlmodel.BillStatementNowInput{},
	)

	resolvertest.RequireRefused(t, err, engine, permission.ResourceInvoiceRun, permission.OpApprove)
}

func TestOpenStatementResolver_MinimumAmountIsNullWhenUnset(t *testing.T) {
	t.Parallel()

	r := &OpenStatementResolver{}
	statement := &invoicerunports.OpenStatement{
		TotalAmount: decimal.RequireFromString("1250.5"),
	}

	minimum, err := r.MinimumAmount(t.Context(), statement)
	require.NoError(t, err)
	assert.Nil(t, minimum)

	total, err := r.TotalAmount(t.Context(), statement)
	require.NoError(t, err)
	assert.Equal(t, "1250.5", total)
}
