package bankreceiptresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptworkitemservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptbatch"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptbatchports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks/bankreceiptbatchportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/bankreceiptportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/bankreceiptworkitemportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBankReceiptOperations_CheckTheirPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("brcpt_").String()
	itemID := pulid.MustNew("brwi_").String()
	userID := pulid.MustNew("usr_").String()

	calls := []struct {
		name      string
		resource  permission.Resource
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"bankReceiptBatches",
			permission.ResourceBankReceipt,
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).BankReceiptBatches(ctx)
				return err
			},
		},
		{
			"bankReceiptBatch",
			permission.ResourceBankReceipt,
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).BankReceiptBatch(ctx, id)
				return err
			},
		},
		{
			"bankReceipt",
			permission.ResourceBankReceipt,
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).BankReceipt(ctx, id)
				return err
			},
		},
		{
			"bankReceiptExceptions",
			permission.ResourceBankReceipt,
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).BankReceiptExceptions(ctx)
				return err
			},
		},
		{
			"bankReceiptMatchSuggestions",
			permission.ResourceBankReceipt,
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).BankReceiptMatchSuggestions(ctx, id)
				return err
			},
		},
		{
			"bankReceiptSummary",
			permission.ResourceBankReceipt,
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).BankReceiptSummary(ctx, nil)
				return err
			},
		},
		{
			"importBankReceiptBatch",
			permission.ResourceBankReceipt,
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ImportBankReceiptBatch(
					ctx,
					gqlmodel.ImportBankReceiptBatchInput{},
				)
				return err
			},
		},
		{
			"matchBankReceipt",
			permission.ResourceBankReceipt,
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).MatchBankReceipt(
					ctx,
					id,
					pulid.MustNew("cpay_").String(),
				)
				return err
			},
		},
		{
			"bankReceiptWorkItems",
			permission.ResourceBankReceiptWorkItem,
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).BankReceiptWorkItems(ctx)
				return err
			},
		},
		{
			"bankReceiptWorkItem",
			permission.ResourceBankReceiptWorkItem,
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).BankReceiptWorkItem(ctx, itemID)
				return err
			},
		},
		{
			"assignBankReceiptWorkItem",
			permission.ResourceBankReceiptWorkItem,
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).AssignBankReceiptWorkItem(ctx, itemID, userID)
				return err
			},
		},
		{
			"startBankReceiptWorkItemReview",
			permission.ResourceBankReceiptWorkItem,
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).StartBankReceiptWorkItemReview(ctx, itemID)
				return err
			},
		},
		{
			"resolveBankReceiptWorkItem",
			permission.ResourceBankReceiptWorkItem,
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ResolveBankReceiptWorkItem(
					ctx,
					itemID,
					gqlmodel.ResolveBankReceiptWorkItemInput{
						ResolutionType: bankreceiptworkitem.ResolutionMarkedFalsePositive,
					},
				)
				return err
			},
		},
		{
			"dismissBankReceiptWorkItem",
			permission.ResourceBankReceiptWorkItem,
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).DismissBankReceiptWorkItem(
					ctx,
					itemID,
					"Duplicate line",
				)
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

			resolvertest.RequireRefused(
				t,
				call.run(ctx, deps),
				engine,
				call.resource,
				call.operation,
			)
		})
	}
}

func TestImportBankReceiptBatch_MapsTheFileIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	memo := "ACH HALSTEAD"
	batch := &bankreceiptbatch.BankReceiptBatch{ID: pulid.MustNew("brib_")}
	batches := bankreceiptbatchportsmock.NewMockBankReceiptBatchService(t)
	batches.EXPECT().
		Import(mock.Anything, &bankreceiptbatchports.ImportBankReceiptBatchRequest{
			Source:    "Lockbox",
			Reference: "LB-1031",
			Receipts: []*bankreceiptbatchports.ImportBankReceiptBatchLine{
				{ReceiptDate: 1792000000, AmountMinor: 125050, ReferenceNumber: "981", Memo: memo},
				{ReceiptDate: 1792000000, AmountMinor: 4000, ReferenceNumber: "982"},
			},
			TenantInfo: resolvercore.TenantInfo(authCtx),
		}, mock.Anything).
		Return(&bankreceiptbatchports.BankReceiptBatchResult{Batch: batch}, nil)

	deps := &Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &resolvertest.RecordingPermissionEngine{},
		},
		BankReceiptBatchService: batches,
	}

	result, err := (&MutationResolver{deps}).ImportBankReceiptBatch(
		resolvertest.AuthedContext(t, authCtx),
		gqlmodel.ImportBankReceiptBatchInput{
			Source:    "Lockbox",
			Reference: "LB-1031",
			Receipts: []*gqlmodel.ImportBankReceiptBatchLineInput{
				{ReceiptDate: 1792000000, AmountMinor: 125050, ReferenceNumber: "981", Memo: &memo},
				{ReceiptDate: 1792000000, AmountMinor: 4000, ReferenceNumber: "982"},
			},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, batch, result.Batch)
}

func TestBankReceipt_ReadsTheReceiptInTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	receipt := &bankreceipt.BankReceipt{ID: pulid.MustNew("brcpt_"), AmountMinor: 125050}
	repo := bankreceiptportsmock.NewMockBankReceiptRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, bankreceiptports.GetBankReceiptByIDRequest{
			ID:         receipt.ID,
			TenantInfo: resolvercore.TenantInfo(authCtx),
		}).
		Return(receipt, nil)

	deps := &Deps{
		Core: &resolvercore.Core{PermissionEngine: &resolvertest.RecordingPermissionEngine{}},
		BankReceiptService: bankreceiptservice.New(bankreceiptservice.Params{
			Logger: zap.NewNop(),
			Repo:   repo,
		}),
	}

	got, err := (&QueryResolver{deps}).BankReceipt(
		resolvertest.AuthedContext(t, authCtx),
		receipt.ID.String(),
	)

	require.NoError(t, err)
	assert.Equal(t, receipt, got)
}

func TestBankReceiptWorkItems_ListsTheActiveItems(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	items := []*bankreceiptworkitem.WorkItem{{ID: pulid.MustNew("brwi_")}}
	repo := bankreceiptworkitemportsmock.NewMockBankReceiptWorkItemRepository(t)
	repo.EXPECT().ListActive(mock.Anything, resolvercore.TenantInfo(authCtx)).Return(items, nil)

	deps := &Deps{
		Core: &resolvercore.Core{PermissionEngine: &resolvertest.RecordingPermissionEngine{}},
		BankReceiptWorkItemService: bankreceiptworkitemservice.New(
			bankreceiptworkitemservice.Params{
				Logger: zap.NewNop(),
				Repo:   repo,
			},
		),
	}

	got, err := (&QueryResolver{deps}).BankReceiptWorkItems(resolvertest.AuthedContext(t, authCtx))

	require.NoError(t, err)
	assert.Equal(t, items, got)
}

func TestBankReceiptWorkItemResolutionType_IsNullWhileOpen(t *testing.T) {
	t.Parallel()

	resolver := &BankReceiptWorkItemResolver{}

	open, err := resolver.ResolutionType(t.Context(), &bankreceiptworkitem.WorkItem{})
	require.NoError(t, err)
	assert.Nil(t, open)

	resolved, err := resolver.ResolutionType(t.Context(), &bankreceiptworkitem.WorkItem{
		ResolutionType: bankreceiptworkitem.ResolutionSuperseded,
	})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, bankreceiptworkitem.ResolutionSuperseded, *resolved)
}
