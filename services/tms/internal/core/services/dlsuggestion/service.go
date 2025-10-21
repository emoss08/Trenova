package dlsuggestion

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	Repo              repositories.DedicatedLaneSuggestionRepository
	DedicatedLaneRepo repositories.DedicatedLaneRepository
	AuditService      services.AuditService
}

type Service struct {
	l      *zap.Logger
	repo   repositories.DedicatedLaneSuggestionRepository
	dlRepo repositories.DedicatedLaneRepository
	as     services.AuditService
}

func NewService(p Params) *Service {
	return &Service{
		l:      p.Logger.Named("service.dedicatedlane-suggestion"),
		repo:   p.Repo,
		dlRepo: p.DedicatedLaneRepo,
		as:     p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListDedicatedLaneSuggestionRequest,
) (*pagination.ListResult[*dedicatedlane.Suggestion], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetDedicatedLaneSuggestionByIDRequest,
) (*dedicatedlane.Suggestion, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Accept(
	ctx context.Context,
	req *repositories.SuggestionAcceptRequest,
) (*dedicatedlane.DedicatedLane, error) {
	log := s.l.With(
		zap.String("operation", "Accept"),
		zap.Any("req", req),
	)

	sug, err := s.repo.GetByID(ctx, &repositories.GetDedicatedLaneSuggestionByIDRequest{
		ID:    req.SuggestionID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
	})
	if err != nil {
		log.Error("failed to get dedicated lane suggestion", zap.Error(err))
		return nil, err
	}

	if sug.IsExpired() {
		return nil, errortypes.NewValidationError(
			"suggestionId",
			errortypes.ErrInvalid,
			"suggestion has expired",
		)
	}

	if sug.IsProcessed() {
		return nil, errortypes.NewValidationError(
			"suggestionId",
			errortypes.ErrInvalid,
			"suggestion has already been processed",
		)
	}

	dlName := sug.SuggestedName
	if req.DedicatedLaneName != nil && *req.DedicatedLaneName != "" {
		dlName = *req.DedicatedLaneName
	}

	entity := &dedicatedlane.DedicatedLane{
		BusinessUnitID:        req.BuID,
		OrganizationID:        req.OrgID,
		Name:                  dlName,
		CustomerID:            sug.CustomerID,
		OriginLocationID:      sug.OrganizationID,
		DestinationLocationID: sug.DestinationLocationID,
		ServiceTypeID:         pulid.ConvertFromPtr(sug.ServiceTypeID),
		ShipmentTypeID:        pulid.ConvertFromPtr(sug.ShipmentTypeID),
		PrimaryWorkerID:       req.PrimaryWorkerID,
		SecondaryWorkerID:     req.SecondaryWorkerID,
		TrailerTypeID:         sug.TrailerTypeID,
		TractorTypeID:         sug.TractorTypeID,
		AutoAssign:            req.AutoAssign,
	}

	createdDL, err := s.dlRepo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create dedicated lane", zap.Error(err))
		return nil, err
	}

	now := utils.NowUnix()
	sug.Status = dedicatedlane.SuggestionStatusAccepted
	sug.ProcessedAt = &now
	sug.ProcessedByID = &req.ProcessedByID
	sug.CreatedDedicatedLaneID = &createdDL.ID

	_, err = s.repo.Update(ctx, sug)
	if err != nil {
		log.Error("failed to update dedicated lane suggestion", zap.Error(err))
		// do not return error, continue with the flow
	}

	err = s.as.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDedicatedLane,
		ResourceID:     createdDL.GetID(),
		Operation:      permission.OpCreate,
		UserID:         req.ProcessedByID,
		CurrentState:   jsonutils.MustToJSON(createdDL),
		OrganizationID: createdDL.OrganizationID,
		BusinessUnitID: createdDL.BusinessUnitID,
	}, audit.WithComment(fmt.Sprintf("Dedicated lane created from suggestion %s", sug.ID.String())))
	if err != nil {
		log.Error("failed to log dedicated lane creation", zap.Error(err))
		// do not return error, continue with the flow
	}

	return createdDL, nil
}

func (s *Service) Reject(
	ctx context.Context,
	req *repositories.SuggestionRejectRequest,
) (*dedicatedlane.Suggestion, error) {
	log := s.l.With(
		zap.String("operation", "Reject"),
		zap.Any("req", req),
	)

	sug, err := s.repo.GetByID(ctx, &repositories.GetDedicatedLaneSuggestionByIDRequest{
		ID:    req.SuggestionID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
	})
	if err != nil {
		log.Error("failed to get dedicated lane suggestion", zap.Error(err))
		return nil, err
	}

	if sug.IsProcessed() {
		return nil, errortypes.NewValidationError(
			"suggestionId",
			errortypes.ErrInvalid,
			"suggestion has already been processed",
		)
	}

	now := utils.NowUnix()
	sug.Status = dedicatedlane.SuggestionStatusRejected
	sug.ProcessedByID = &req.ProcessedByID
	sug.ProcessedAt = &now

	if req.RejectReason != "" {
		sug.PatternDetails["rejectReason"] = req.RejectReason
		sug.PatternDetails["rejectedAt"] = now
	}

	entity, err := s.repo.Update(ctx, sug)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *Service) ExpireOldSuggestions(
	ctx context.Context,
	orgID, buID pulid.ID,
) (int64, error) {
	return s.repo.ExpireOldSuggestions(ctx, orgID, buID)
}
