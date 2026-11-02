package billingqueuejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/billingqueuejobs/billingqueuecontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/zap"
)

type SubmitNovaReviewsInput struct {
	Payload *billingqueuecontract.NovaGenerationPayload `json:"payload"`
	After   *billingqueueports.ReviewCursor             `json:"after,omitempty"`
	Inline  bool                                        `json:"inline"`
}

type NovaBatchInput struct {
	Payload    *billingqueuecontract.NovaGenerationPayload `json:"payload"`
	Submission *billingqueueports.ReviewSubmission         `json:"submission"`
}

type NovaProgressInput struct {
	Payload *billingqueuecontract.NovaGenerationPayload `json:"payload"`
	// Records also tells the queue its items or brief changed.
	Records bool `json:"records"`
}

func (a *Activities) SubmitNovaReviewsActivity(
	ctx context.Context,
	input *SubmitNovaReviewsInput,
) (*billingqueueports.ReviewSubmission, error) {
	tenant := input.Payload.Tenant()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	return a.assist.SubmitReviews(ctx, &billingqueueports.SubmitReviewsRequest{
		TenantInfo: tenant,
		After:      input.After,
		Inline:     input.Inline,
	})
}

func (a *Activities) NovaBatchEndedActivity(
	ctx context.Context,
	input *NovaBatchInput,
) (bool, error) {
	tenant := input.Payload.Tenant()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	return a.assist.ReviewBatchEnded(ctx, tenant, input.Submission)
}

func (a *Activities) CancelNovaReviewsActivity(ctx context.Context, input *NovaBatchInput) error {
	tenant := input.Payload.Tenant()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	return a.assist.CancelReviews(ctx, tenant, input.Submission)
}

func (a *Activities) ApplyNovaReviewsActivity(
	ctx context.Context,
	input *NovaBatchInput,
) (int, error) {
	tenant := input.Payload.Tenant()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	return a.assist.ApplyReviews(ctx, &billingqueueports.ApplyReviewsRequest{
		TenantInfo: tenant,
		Submission: input.Submission,
	})
}

// NovaLeftUnreviewedActivity is whether a generation finishing leaves items
// Nova can review and has not.
func (a *Activities) NovaLeftUnreviewedActivity(
	ctx context.Context,
	payload *billingqueuecontract.NovaGenerationPayload,
) (bool, error) {
	tenant := payload.Tenant()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	return a.assist.HasUnreviewedItems(ctx, tenant)
}

// PublishNovaProgressActivity tells the queue's readers the generation moved,
// and, when reviews or the brief were written, that the queue changed.
func (a *Activities) PublishNovaProgressActivity(
	ctx context.Context,
	input *NovaProgressInput,
) error {
	tenant := input.Payload.Tenant()
	resources := []string{billingqueuecontract.NovaGenerationRealtimeResource}
	if input.Records {
		resources = append(resources, permission.ResourceBillingQueue.String())
	}
	for _, resource := range resources {
		if err := a.realtime.PublishResourceInvalidation(ctx, &realtimeports.PublishResourceInvalidationRequest{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			Resource:       resource,
			Action:         "updated",
		}); err != nil {
			a.l.Warn("failed to publish billing queue Nova progress",
				zap.String("organizationId", tenant.OrgID.String()), zap.Error(err))
		}
	}

	return nil
}

// RequestNovaGenerationsActivity starts a generation for each organization
// with items Nova has not reviewed, when Nova can review for it.
func (a *Activities) RequestNovaGenerationsActivity(ctx context.Context) (int, error) {
	tenants, err := a.queueRepo.ListNovaBacklogTenants(ctx, novaSweepTenants)
	if err != nil {
		return 0, err
	}

	requested := 0
	for _, tenant := range tenants {
		scoped := dbscope.WithTenant(ctx, tenant.DBTenant())
		available, availableErr := a.assist.Availability(scoped, tenant)
		if availableErr != nil {
			a.l.Warn("could not read whether Nova reviews for an organization",
				zap.String("organizationId", tenant.OrgID.String()), zap.Error(availableErr))
			continue
		}
		if !available.Review {
			continue
		}
		a.generator.RequestGeneration(scoped, tenant, billingqueueports.NovaTriggerSchedule)
		requested++
	}

	return requested, nil
}
