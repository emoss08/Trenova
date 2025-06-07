package dedicatedlane

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type SuggestionServiceParams struct {
	fx.In

	Logger            *logger.Logger
	SuggestionRepo    repositories.DedicatedLaneSuggestionRepository
	DedicatedLaneRepo repositories.DedicatedLaneRepository
	PermService       services.PermissionService
	AuditService      services.AuditService
	PatternService    *PatternService
}

type SuggestionService struct {
	l              *zerolog.Logger
	suggRepo       repositories.DedicatedLaneSuggestionRepository
	dlRepo         repositories.DedicatedLaneRepository
	ps             services.PermissionService
	as             services.AuditService
	patternService *PatternService
}

// NewSuggestionService creates a new SuggestionService
//
//nolint:gocritic // we want to use the field names in the struct
func NewSuggestionService(p SuggestionServiceParams) *SuggestionService {
	log := p.Logger.With().
		Str("service", "dedicated_lane_suggestion").
		Logger()

	return &SuggestionService{
		l:              &log,
		suggRepo:       p.SuggestionRepo,
		dlRepo:         p.DedicatedLaneRepo,
		ps:             p.PermService,
		as:             p.AuditService,
		patternService: p.PatternService,
	}
}

func (ss *SuggestionService) List(
	ctx context.Context,
	req *repositories.ListDedicatedLaneSuggestionRequest,
) (*ports.ListResult[*dedicatedlane.DedicatedLaneSuggestion], error) {
	if err := ss.checkPermission(
		ctx,
		permission.ActionRead,
		req.Filter.TenantOpts.UserID,
		req.Filter.TenantOpts.BuID,
		req.Filter.TenantOpts.OrgID,
	); err != nil {
		return nil, err
	}

	return ss.suggRepo.List(ctx, req)
}

func (ss *SuggestionService) Get(
	ctx context.Context,
	req *repositories.GetDedicatedLaneSuggestionByIDRequest,
) (*dedicatedlane.DedicatedLaneSuggestion, error) {
	if err := ss.checkPermission(
		ctx,
		permission.ActionRead,
		req.UserID,
		req.BuID,
		req.OrgID,
	); err != nil {
		return nil, err
	}

	return ss.suggRepo.GetByID(ctx, req)
}

// AcceptSuggestion accepts a suggestion and creates a dedicated lane
//
//nolint:funlen // this is a long function, but it's not complex
func (ss *SuggestionService) AcceptSuggestion(
	ctx context.Context,
	req *dedicatedlane.SuggestionAcceptRequest,
) (*dedicatedlane.DedicatedLane, error) {
	log := ss.l.With().
		Str("operation", "AcceptSuggestion").
		Str("suggestionId", req.SuggestionID.String()).
		Logger()

	if err := ss.checkPermission(
		ctx,
		permission.ActionCreate, // Creating a dedicated lane
		req.ProcessedByID,
		req.BusinessUnitID,
		req.OrganizationID,
	); err != nil {
		return nil, err
	}

	// Get the suggestion
	suggestion, err := ss.suggRepo.GetByID(ctx, &repositories.GetDedicatedLaneSuggestionByIDRequest{
		ID:     req.SuggestionID,
		OrgID:  req.OrganizationID,
		BuID:   req.BusinessUnitID,
		UserID: req.ProcessedByID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get suggestion")
		return nil, err
	}

	// Validate suggestion can be accepted
	if suggestion.IsExpired() {
		return nil, errors.NewValidationError("suggestion", "expired", "Suggestion has expired")
	}

	if suggestion.IsProcessed() {
		return nil, errors.NewValidationError(
			"suggestion",
			"already_processed",
			"Suggestion has already been processed",
		)
	}

	// Create dedicated lane from suggestion
	dedicatedLaneName := suggestion.SuggestedName
	if req.DedicatedLaneName != nil && *req.DedicatedLaneName != "" {
		dedicatedLaneName = *req.DedicatedLaneName
	}

	dedicatedLane := &dedicatedlane.DedicatedLane{
		BusinessUnitID:        req.BusinessUnitID,
		OrganizationID:        req.OrganizationID,
		Name:                  dedicatedLaneName,
		CustomerID:            suggestion.CustomerID,
		OriginLocationID:      suggestion.OriginLocationID,
		DestinationLocationID: suggestion.DestinationLocationID,
		ServiceTypeID:         *suggestion.ServiceTypeID,
		ShipmentTypeID:        *suggestion.ShipmentTypeID,
		PrimaryWorkerID:       req.PrimaryWorkerID,
		SecondaryWorkerID:     req.SecondaryWorkerID,
		TrailerTypeID:         suggestion.TrailerTypeID,
		TractorTypeID:         suggestion.TractorTypeID,
		AutoAssign:            req.AutoAssign,
	}

	// Create the dedicated lane
	createdLane, err := ss.dlRepo.Create(ctx, dedicatedLane)
	if err != nil {
		log.Error().Err(err).Msg("failed to create dedicated lane")
		return nil, eris.Wrap(err, "create dedicated lane")
	}

	// Update suggestion status
	now := timeutils.NowUnix()
	suggestion.Status = dedicatedlane.SuggestionStatusAccepted
	suggestion.ProcessedByID = &req.ProcessedByID
	suggestion.ProcessedAt = &now
	suggestion.CreatedDedicatedLaneID = &createdLane.ID

	_, err = ss.suggRepo.Update(ctx, suggestion)
	if err != nil {
		log.Error().Err(err).Msg("failed to update suggestion status")
		// Don't fail the operation, but log the error
	}

	// Log audit trail
	err = ss.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDedicatedLane,
			ResourceID:     createdLane.GetID(),
			Action:         permission.ActionCreate,
			UserID:         req.ProcessedByID,
			CurrentState:   jsonutils.MustToJSON(createdLane),
			OrganizationID: createdLane.OrganizationID,
			BusinessUnitID: createdLane.BusinessUnitID,
		},
		audit.WithComment(
			fmt.Sprintf("Dedicated lane created from suggestion %s", req.SuggestionID.String()),
		),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log dedicated lane creation")
	}

	log.Info().
		Str("dedicatedLaneId", createdLane.ID.String()).
		Msg("suggestion accepted and dedicated lane created")

	return createdLane, nil
}

func (ss *SuggestionService) RejectSuggestion(
	ctx context.Context,
	req *dedicatedlane.SuggestionRejectRequest,
) (*dedicatedlane.DedicatedLaneSuggestion, error) {
	log := ss.l.With().
		Str("operation", "RejectSuggestion").
		Str("suggestionId", req.SuggestionID.String()).
		Logger()

	if err := ss.checkPermission(
		ctx,
		permission.ActionUpdate,
		req.ProcessedByID,
		req.BusinessUnitID,
		req.OrganizationID,
	); err != nil {
		return nil, err
	}

	// Get the suggestion
	suggestion, err := ss.suggRepo.GetByID(ctx, &repositories.GetDedicatedLaneSuggestionByIDRequest{
		ID:     req.SuggestionID,
		OrgID:  req.OrganizationID,
		BuID:   req.BusinessUnitID,
		UserID: req.ProcessedByID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get suggestion")
		return nil, err
	}

	// Validate suggestion can be rejected
	if suggestion.IsProcessed() {
		return nil, errors.NewValidationError(
			"suggestion",
			"already_processed",
			"Suggestion has already been processed",
		)
	}

	// Update suggestion status
	now := timeutils.NowUnix()
	suggestion.Status = dedicatedlane.SuggestionStatusRejected
	suggestion.ProcessedByID = &req.ProcessedByID
	suggestion.ProcessedAt = &now

	// Add reject reason to pattern details
	if req.RejectReason != "" {
		suggestion.PatternDetails["rejectReason"] = req.RejectReason
		suggestion.PatternDetails["rejectedAt"] = now
	}

	updatedSuggestion, err := ss.suggRepo.Update(ctx, suggestion)
	if err != nil {
		log.Error().Err(err).Msg("failed to update suggestion status")
		return nil, eris.Wrap(err, "update suggestion status")
	}

	log.Info().Msg("suggestion rejected")

	return updatedSuggestion, nil
}

func (ss *SuggestionService) AnalyzePatterns(
	ctx context.Context,
	req *dedicatedlane.PatternAnalysisRequest,
	userID pulid.ID,
) (*dedicatedlane.PatternAnalysisResult, error) {
	if err := ss.checkPermission(
		ctx,
		permission.ActionCreate, // Creating suggestions
		userID,
		req.BusinessUnitID,
		req.OrganizationID,
	); err != nil {
		return nil, err
	}

	return ss.patternService.AnalyzePatterns(ctx, req)
}

func (ss *SuggestionService) ExpireOldSuggestions(
	ctx context.Context,
	orgID, buID pulid.ID,
) (int64, error) {
	log := ss.l.With().
		Str("operation", "ExpireOldSuggestions").
		Str("orgId", orgID.String()).
		Logger()

	expired, err := ss.suggRepo.ExpireOldSuggestions(ctx, orgID, buID)
	if err != nil {
		log.Error().Err(err).Msg("failed to expire old suggestions")
		return 0, err
	}

	log.Info().Int64("expiredCount", expired).Msg("old suggestions expired")

	return expired, nil
}

func (ss *SuggestionService) checkPermission(
	ctx context.Context,
	action permission.Action,
	userID, buID, orgID pulid.ID,
) error {
	log := ss.l.With().
		Str("operation", "checkPermission").
		Str("action", string(action)).
		Str("userID", userID.String()).
		Str("buID", buID.String()).
		Str("orgID", orgID.String()).
		Logger()

	// Check if user has permission
	result, err := ss.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceDedicatedLane, // Using same resource as dedicated lanes
				Action:         action,
				BusinessUnitID: buID,
				OrganizationID: orgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return errors.NewAuthorizationError(
			fmt.Sprintf(
				"You do not have permission to %s dedicated lane suggestions",
				string(action),
			),
		)
	}

	return nil
}
