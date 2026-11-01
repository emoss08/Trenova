package editransferresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func TestEDITransferOperations_CheckTheEDIPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("edilt_").String()
	ids := []string{id}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"ediTransferMappingPreview",
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).EDITransferMappingPreview(ctx, id)
				return err
			},
		},
		{"submitEdiLoadTender", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).SubmitEDILoadTender(
				ctx,
				gqlmodel.SubmitEDILoadTenderInput{SourceShipmentID: pulid.MustNew("shp_").String()},
			)
			return err
		}},
		{"approveEdiTransfer", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).ApproveEDITransfer(ctx, id, nil)
			return err
		}},
		{"rejectEdiTransfer", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).RejectEDITransfer(ctx, id, "Wrong lane")
			return err
		}},
		{"cancelEdiTransfer", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CancelEDITransfer(ctx, id)
			return err
		}},
		{
			"bulkApproveEdiTransfers",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).BulkApproveEDITransfers(ctx, ids)
				return err
			},
		},
		{
			"bulkRejectEdiTransfers",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).BulkRejectEDITransfers(ctx, ids, "Wrong lane")
				return err
			},
		},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(t, err, engine, permission.ResourceEDI, call.operation)
		})
	}
}
