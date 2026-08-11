package tenderjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Lifecycle serviceports.TenderLifecycle
	Repo      repositories.TenderRepository
	Workflows serviceports.WorkflowStarter
	Logger    *zap.Logger
}

type Activities struct {
	lifecycle serviceports.TenderLifecycle
	repo      repositories.TenderRepository
	workflows serviceports.WorkflowStarter
	l         *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		lifecycle: p.Lifecycle,
		repo:      p.Repo,
		workflows: p.Workflows,
		l:         p.Logger.Named("temporal.tender-activities"),
	}
}

type TenderActivityInput struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	TenderID   pulid.ID              `json:"tenderId"`
	OfferID    pulid.ID              `json:"offerId"`

	Action tender.ResponseAction `json:"action"`
	Source tender.ResponseSource `json:"source"`
	Reason string                `json:"reason"`

	ActorUserID pulid.ID `json:"actorUserId"`
}

func (a *Activities) LoadPlanActivity(
	ctx context.Context,
	in *TenderActivityInput,
) (*serviceports.TenderPlan, error) {
	return a.lifecycle.LoadPlan(ctx, in.TenantInfo, in.TenderID)
}

func (a *Activities) DispatchOfferActivity(
	ctx context.Context,
	in *TenderActivityInput,
) (*serviceports.TenderDispatchResult, error) {
	return a.lifecycle.DispatchOffer(ctx, in.TenantInfo, in.OfferID)
}

func (a *Activities) ExpireOfferActivity(
	ctx context.Context,
	in *TenderActivityInput,
) error {
	return a.lifecycle.ExpireOffer(ctx, in.TenantInfo, in.OfferID)
}

func (a *Activities) DeclineOfferActivity(
	ctx context.Context,
	in *TenderActivityInput,
) error {
	return a.lifecycle.DeclineOffer(ctx, in.TenantInfo, in.OfferID, in.Source, in.Reason)
}

func (a *Activities) AcceptOfferActivity(
	ctx context.Context,
	in *TenderActivityInput,
) (*serviceports.TenderAcceptResult, error) {
	return a.lifecycle.AcceptOffer(ctx, in.TenantInfo, in.OfferID, in.Source)
}

func (a *Activities) FinalizeAcceptedActivity(
	ctx context.Context,
	in *TenderActivityInput,
) error {
	return a.lifecycle.FinalizeAccepted(ctx, in.TenantInfo, in.TenderID, in.OfferID)
}

func (a *Activities) IssueRateConfirmationActivity(
	ctx context.Context,
	in *TenderActivityInput,
) error {
	return a.lifecycle.IssueRateConfirmation(ctx, in.TenantInfo, in.OfferID)
}

func (a *Activities) MarkNeedsReviewActivity(
	ctx context.Context,
	in *TenderActivityInput,
) error {
	return a.lifecycle.MarkNeedsReview(ctx, in.TenantInfo, in.TenderID, in.OfferID, in.Reason)
}

func (a *Activities) MarkExhaustedActivity(
	ctx context.Context,
	in *TenderActivityInput,
) error {
	return a.lifecycle.MarkExhausted(ctx, in.TenantInfo, in.TenderID)
}

func (a *Activities) CancelTenderActivity(
	ctx context.Context,
	in *TenderActivityInput,
) error {
	return a.lifecycle.CancelFromWorkflow(
		ctx, in.TenantInfo, in.TenderID, in.Reason, in.ActorUserID,
	)
}

func (a *Activities) RecordLateResponseActivity(
	ctx context.Context,
	in *TenderActivityInput,
) error {
	return a.lifecycle.RecordLateResponse(ctx, in.TenantInfo, in.OfferID, in.Action, in.Source)
}
