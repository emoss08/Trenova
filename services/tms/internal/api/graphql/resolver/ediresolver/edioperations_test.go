package ediresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioverview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEDIMessageAndFileOperations_CheckTheEDIPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("edimsg_").String()
	ids := []string{id}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"ediInboundFileInspection",
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).EDIInboundFileInspection(ctx, id)
				return err
			},
		},
		{"ediMessageInspection", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).EDIMessageInspection(ctx, id)
			return err
		}},
		{"ediFlowInstanceDetail", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).EDIFlowInstanceDetail(ctx, id)
			return err
		}},
		{
			"reprocessEdiInboundFile",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ReprocessEDIInboundFile(ctx, id)
				return err
			},
		},
		{
			"bulkReprocessEdiInboundFiles",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).BulkReprocessEDIInboundFiles(ctx, ids)
				return err
			},
		},
		{
			"retryEdiMessageDelivery",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).RetryEDIMessageDelivery(ctx, id)
				return err
			},
		},
		{"replayEdiMessage", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).ReplayEDIMessage(ctx, id)
			return err
		}},
		{
			"bulkRetryEdiMessageDelivery",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).BulkRetryEDIMessageDelivery(ctx, ids)
				return err
			},
		},
		{
			"releaseEdiFlowInstance",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ReleaseEDIFlowInstance(ctx, id)
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

func TestPartnerReadinessToModel_CarriesEachStep(t *testing.T) {
	t.Parallel()

	partnerID := pulid.MustNew("edip_")
	model := partnerReadinessToModel(&edioverview.EDIPartnerReadiness{
		PartnerID:      partnerID,
		CompletedCount: 1,
		TotalCount:     2,
		Items: []edioverview.EDIPartnerReadinessItem{
			{Key: "transport", Label: "Communication profile", Complete: true},
			{Key: "lading", Label: "Lading project", Complete: false},
		},
	})

	assert.Equal(t, partnerID.String(), model.PartnerID)
	assert.False(t, model.Ready)
	assert.Equal(t, 1, model.CompletedCount)
	require.Len(t, model.Items, 2)
	assert.Equal(t, "transport", model.Items[0].Key)
	assert.True(t, model.Items[0].Complete)
	assert.Equal(t, "Lading project", model.Items[1].Label)
}
