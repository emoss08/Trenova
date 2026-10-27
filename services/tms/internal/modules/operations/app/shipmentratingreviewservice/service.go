package shipmentratingreviewservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	DB           kernelports.Transactor
	Repo         shipmentports.RatingReviewRepository
	AuditService auditports.AuditService
	Realtime     agentports.RealtimeService `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	db           kernelports.Transactor
	repo         shipmentports.RatingReviewRepository
	auditService auditports.AuditService
	realtime     agentports.RealtimeService
	now          func() int64
}

var _ shipmentports.RatingReviewService = (*Service)(nil)

//nolint:gocritic // fx param structs are passed by value
func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.shipment-rating-review"),
		db:           p.DB,
		repo:         p.Repo,
		auditService: p.AuditService,
		realtime:     p.Realtime,
		now:          timeutils.NowUnix,
	}
}

func (s *Service) Approve(
	ctx context.Context,
	req *shipmentports.ApproveRatingReviewRequest,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	if req == nil || req.ShipmentID.IsNil() {
		return nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Shipment is required",
		)
	}
	if !actor.IsUser() || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Only a person can approve a rating the tariff held for review",
		)
	}

	var (
		original *shipment.Shipment
		saved    *shipment.Shipment
	)
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		entity, err := s.repo.GetForRatingReview(txCtx, req)
		if err != nil {
			return err
		}
		if entity.RatingDetail == nil {
			return errortypes.NewBusinessError(
				"This shipment's rating has nothing waiting for review",
			)
		}

		before := *entity
		detail := *entity.RatingDetail
		before.RatingDetail = entity.RatingDetail
		entity.RatingDetail = &detail

		changed, err := detail.ApproveReview(actor.UserID, s.now())
		if err != nil {
			return err
		}
		if !changed {
			saved = entity
			return nil
		}

		saved, err = s.repo.SaveRatingReview(txCtx, entity)
		if err != nil {
			return err
		}
		original = &before
		return nil
	})
	if err != nil {
		return nil, err
	}

	if original != nil {
		s.audit(original, saved, actor)
		s.publishBillingHoldChange(ctx, saved, actor)
	}
	return saved, nil
}

func (s *Service) audit(
	original, updated *shipment.Shipment,
	actor *permissionports.RequestActor,
) {
	auditActor := actor.AuditActor()
	reasons := updated.RatingDetail.ReviewReasons()
	if err := s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceShipment,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		APIKeyID:       auditActor.APIKeyID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	},
		auditservice.WithComment("Rating approved for billing: "+strings.Join(reasons, "; ")),
		auditservice.WithMetadata(map[string]any{
			"shipmentId": updated.ID.String(),
			"reviews":    reasons,
		}),
		auditservice.WithDiff(original, updated),
	); err != nil {
		s.l.Error("failed to log the rating review approval", zap.Error(err))
	}
}

func (s *Service) publishBillingHoldChange(
	ctx context.Context,
	saved *shipment.Shipment,
	actor *permissionports.RequestActor,
) {
	if s.realtime == nil {
		return
	}
	auditActor := actor.AuditActor()
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: saved.OrganizationID,
		BusinessUnitID: saved.BusinessUnitID,
		ActorUserID:    auditActor.UserID,
		ActorType:      auditActor.PrincipalType,
		ActorID:        auditActor.PrincipalID,
		ActorAPIKeyID:  auditActor.APIKeyID,
		Resource:       permission.ResourceBillingQueue.String(),
		Action:         "updated",
		Entity:         map[string]string{"shipmentId": saved.ID.String()},
	}); err != nil {
		s.l.Warn("failed to publish billing queue invalidation for a rating review",
			zap.String("shipmentId", saved.ID.String()), zap.Error(err))
	}
}
