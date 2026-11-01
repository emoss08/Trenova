package billingqueueworkspaceresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingapprovalrun"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueuefilterpreset"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueuefilterpresetportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeApprovalService struct {
	billingqueueports.BillingQueueApprovalService
	started *billingqueueports.StartBillingQueueApprovalRequest
}

func (f *fakeApprovalService) Start(
	_ context.Context,
	req *billingqueueports.StartBillingQueueApprovalRequest,
) (*billingapprovalrun.ApprovalRun, error) {
	f.started = req
	return &billingapprovalrun.ApprovalRun{ID: pulid.MustNew("bqar_"), Action: req.Action}, nil
}

func grantingEngine(grants ...string) *resolvertest.GrantingPermissionEngine {
	granted := make(map[string]bool, len(grants))
	for _, grant := range grants {
		granted[grant] = true
	}

	return &resolvertest.GrantingPermissionEngine{Granted: granted}
}

func grant(resource permission.Resource, operation permission.Operation) string {
	return resource.String() + "|" + string(operation)
}

func TestStartBillingQueueBulkRun_MapsTheActionItemsAndParams(t *testing.T) {
	t.Parallel()

	approvals := &fakeApprovalService{}
	r := &MutationResolver{&Deps{
		Core: &resolvercore.Core{PermissionEngine: grantingEngine(
			grant(permission.ResourceBillingQueue, permission.OpUpdate),
		)},
		BillingQueueApprovalService: approvals,
	}}
	authCtx := resolvertest.UserAuthContext()
	itemID := pulid.MustNew("bqi_")
	hold := billingqueue.HoldReasonCustomerDispute
	reason := "Disputed rate"

	run, err := r.StartBillingQueueBulkRun(
		resolvertest.AuthedContext(t, authCtx),
		gqlmodel.StartBillingQueueBulkRunInput{
			Action:         billingapprovalrun.RunActionHold,
			ItemIds:        []string{itemID.String()},
			IdempotencyKey: "key-1",
			Params: &gqlmodel.BillingQueueBulkRunParamsInput{
				HoldReasonCode: &hold,
				Reason:         &reason,
			},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, billingapprovalrun.RunActionHold, run.Action)
	assert.Equal(t, []pulid.ID{itemID}, approvals.started.ItemIDs)
	assert.Equal(t, "key-1", approvals.started.IdempotencyKey)
	assert.Equal(t, string(hold), approvals.started.Params.HoldReasonCode)
	assert.Equal(t, reason, approvals.started.Params.Reason)
	assert.Equal(t, authCtx.UserID, approvals.started.TenantInfo.UserID)
}

func TestStartBillingQueueBulkRun_PostingNeedsInvoicePost(t *testing.T) {
	t.Parallel()

	approvals := &fakeApprovalService{}
	r := &MutationResolver{&Deps{
		Core: &resolvercore.Core{PermissionEngine: grantingEngine(
			grant(permission.ResourceBillingQueue, permission.OpUpdate),
		)},
		BillingQueueApprovalService: approvals,
	}}

	_, err := r.StartBillingQueueBulkRun(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		gqlmodel.StartBillingQueueBulkRunInput{
			Action:         billingapprovalrun.RunActionPost,
			ItemIds:        []string{pulid.MustNew("bqi_").String()},
			IdempotencyKey: "key-2",
		},
	)

	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)
	assert.Nil(t, approvals.started)
}

func TestStartBillingQueueBulkRun_RefusesMoreThanFiveHundredItems(t *testing.T) {
	t.Parallel()

	approvals := &fakeApprovalService{}
	r := &MutationResolver{&Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &resolvertest.RecordingPermissionEngine{},
		},
		BillingQueueApprovalService: approvals,
	}}
	ids := make([]string, billingapprovalrun.MaxApprovalRunItems+1)
	for i := range ids {
		ids[i] = pulid.MustNew("bqi_").String()
	}

	_, err := r.StartBillingQueueBulkRun(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		gqlmodel.StartBillingQueueBulkRunInput{
			Action:         billingapprovalrun.RunActionApprove,
			ItemIds:        ids,
			IdempotencyKey: "key-3",
		},
	)

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "itemIds", validation.Field)
	assert.Nil(t, approvals.started)
}

func TestRedetectBillingQueuePodSignatures_NeedsDocumentUpdateToo(t *testing.T) {
	t.Parallel()

	engine := grantingEngine(grant(permission.ResourceBillingQueue, permission.OpUpdate))
	r := &MutationResolver{&Deps{Core: &resolvercore.Core{PermissionEngine: engine}}}

	_, err := r.RedetectBillingQueuePodSignatures(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
	)

	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)
	require.Len(t, engine.Requests, 2)
	assert.Equal(t, permission.ResourceDocument.String(), engine.Requests[1].Resource)
	assert.Equal(t, permission.OpUpdate, engine.Requests[1].Operation)
}

func TestBillingQueueRecommendations_PassesWhatTheCallerMayPostAndAssign(t *testing.T) {
	t.Parallel()

	service := billingqueueportsmock.NewMockBillingQueueService(t)
	r := &QueryResolver{&Deps{
		Core: &resolvercore.Core{PermissionEngine: grantingEngine(
			grant(permission.ResourceBillingQueue, permission.OpRead),
			grant(permission.ResourceInvoice, permission.OpPost),
		)},
		BillingQueueService: service,
	}}
	authCtx := resolvertest.UserAuthContext()

	var captured *billingqueueports.ListRecommendationsRequest
	service.EXPECT().
		Recommendations(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *billingqueueports.ListRecommendationsRequest,
		) ([]*billingqueueports.Recommendation, error) {
			captured = req
			return nil, nil
		}).
		Once()

	recommendations, err := r.BillingQueueRecommendations(resolvertest.AuthedContext(t, authCtx))

	require.NoError(t, err)
	assert.Empty(t, recommendations)
	assert.NotNil(t, recommendations)
	assert.True(t, captured.CanPost)
	assert.False(t, captured.CanAssign)
	assert.Equal(t, authCtx.UserID, captured.UserID)
}

func TestCreateBillingQueueFilterPreset_ScopesToTheCallerAndValidates(t *testing.T) {
	t.Parallel()

	repo := billingqueuefilterpresetportsmock.NewMockBillingQueueFilterPresetRepository(t)
	r := &MutationResolver{&Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &resolvertest.RecordingPermissionEngine{},
		},
		BillingQueueFilterPresetRepo: repo,
	}}
	authCtx := resolvertest.UserAuthContext()

	repo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(entity *billingqueuefilterpreset.BillingQueueFilterPreset) bool {
			return entity.UserID == authCtx.UserID &&
				entity.OrganizationID == authCtx.OrganizationID &&
				entity.Name == "Disputes" &&
				entity.IsDefault
		})).
		RunAndReturn(func(
			_ context.Context,
			entity *billingqueuefilterpreset.BillingQueueFilterPreset,
		) (*billingqueuefilterpreset.BillingQueueFilterPreset, error) {
			return entity, nil
		}).
		Once()
	isDefault := true

	preset, err := r.CreateBillingQueueFilterPreset(
		resolvertest.AuthedContext(t, authCtx),
		gqlmodel.BillingQueueFilterPresetInput{
			Name:      "Disputes",
			Filters:   map[string]any{"status": "OnHold"},
			IsDefault: &isDefault,
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "Disputes", preset.Name)

	_, err = r.CreateBillingQueueFilterPreset(
		resolvertest.AuthedContext(t, authCtx),
		gqlmodel.BillingQueueFilterPresetInput{Filters: map[string]any{}},
	)
	require.Error(t, err)
}

func TestBillingQueueSearchDatesResolver_UnsetWindowsReadAsNull(t *testing.T) {
	t.Parallel()

	r := &BillingQueueSearchDatesResolver{}
	dates := &billingqueueports.SearchDates{From: "2026-10-01"}

	lastDays, err := r.LastDays(t.Context(), dates)
	require.NoError(t, err)
	assert.Nil(t, lastDays)

	from, err := r.From(t.Context(), dates)
	require.NoError(t, err)
	require.NotNil(t, from)
	assert.Equal(t, "2026-10-01", *from)

	to, err := r.To(t.Context(), dates)
	require.NoError(t, err)
	assert.Nil(t, to)
}
