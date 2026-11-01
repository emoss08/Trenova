package accountsreceivableresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerstatementports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStatements struct {
	customerstatementports.Service

	listed customerstatementports.ListSentRequest
	sent   *customerstatementports.SendRequest
}

func (f *fakeStatements) ListSent(
	_ context.Context,
	req customerstatementports.ListSentRequest,
) ([]*customerstatementports.SentStatement, error) {
	f.listed = req
	return []*customerstatementports.SentStatement{{SentTo: "ap@halstead.test"}}, nil
}

func (f *fakeStatements) Send(
	_ context.Context,
	req *customerstatementports.SendRequest,
	_ *permissionports.RequestActor,
) (*customerstatementports.SendResult, error) {
	f.sent = req
	return &customerstatementports.SendResult{FileName: "Statement.pdf"}, nil
}

func TestCustomerStatementOperations_CheckTheAccountsReceivablePermission(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_").String()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"sentCustomerStatements", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).SentCustomerStatements(ctx, customerID)
			return err
		}},
		{"sendCustomerStatement", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).SendCustomerStatement(
				ctx,
				customerID,
				gqlmodel.SendCustomerStatementInput{},
			)
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			resolvertest.RequireRefused(
				t,
				call.run(ctx, deps),
				engine,
				permission.ResourceAccountsReceivable,
				call.operation,
			)
		})
	}
}

func TestSentCustomerStatements_ReadsTheCustomersHistory(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	customerID := pulid.MustNew("cus_")
	statements := &fakeStatements{}
	deps := &Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &resolvertest.RecordingPermissionEngine{},
		},
		CustomerStatementService: statements,
	}

	sent, err := (&QueryResolver{deps}).SentCustomerStatements(
		resolvertest.AuthedContext(t, authCtx),
		customerID.String(),
	)

	require.NoError(t, err)
	require.Len(t, sent, 1)
	assert.Equal(t, customerstatementports.ListSentRequest{
		TenantInfo: resolvercore.TenantInfo(authCtx),
		CustomerID: customerID,
	}, statements.listed)
}

func TestSendCustomerStatement_ScopesTheRangeToTheCustomer(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	customerID := pulid.MustNew("cus_")
	startDate, asOfDate := 100, 200
	statements := &fakeStatements{}
	deps := &Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &resolvertest.RecordingPermissionEngine{},
		},
		CustomerStatementService: statements,
	}

	result, err := (&MutationResolver{deps}).SendCustomerStatement(
		resolvertest.AuthedContext(t, authCtx),
		customerID.String(),
		gqlmodel.SendCustomerStatementInput{
			StartDate: &startDate,
			AsOfDate:  &asOfDate,
			Note:      "As requested",
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "Statement.pdf", result.FileName)
	assert.Equal(t, &customerstatementports.SendRequest{
		StatementRequest: customerstatementports.StatementRequest{
			TenantInfo: resolvercore.TenantInfo(authCtx),
			CustomerID: customerID,
			StartDate:  100,
			AsOfDate:   200,
		},
		Note: "As requested",
	}, statements.sent)
}
