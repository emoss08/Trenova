/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
	PermissionService services.PermissionService
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

// NewSuggestionService creates a new instance of the SuggestionService, which is responsible for
// managing dedicated lane suggestions and their lifecycle.
//
// Parameters:
//   - p: SuggestionServiceParams containing all the dependencies for the service.
//
// Returns:
//   - *SuggestionService: A new SuggestionService instance.
//
//nolint:gocritic // This is a constructor
func NewSuggestionService(p SuggestionServiceParams) *SuggestionService {
	log := p.Logger.With().
		Str("service", "dedicated_lane_suggestion").
		Logger()

	return &SuggestionService{
		l:              &log,
		suggRepo:       p.SuggestionRepo,
		dlRepo:         p.DedicatedLaneRepo,
		ps:             p.PermissionService,
		as:             p.AuditService,
		patternService: p.PatternService,
	}
}

// List returns a list of dedicated lane suggestions based on the provided request.
//
// Parameters:
//   - ctx: The context for the request.
//   - req: The request containing the filter options for the list.
//
// Returns:
//   - *ports.ListResult[*dedicatedlane.DedicatedLaneSuggestion]: A list of dedicated lane suggestions.
//   - error: An error if the request fails.
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

// AcceptSuggestion accepts a suggestion and creates a dedicated lane.
//
// Parameters:
//   - ctx: The context for the request.
//   - req: The request containing the suggestion ID and the user ID who is accepting the suggestion.
//
// Returns:
//   - *dedicatedlane.DedicatedLane: The created dedicated lane.
//   - error: An error if the request fails.
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
		permission.ActionCreate,
		req.ProcessedByID,
		req.BusinessUnitID,
		req.OrganizationID,
	); err != nil {
		return nil, err
	}

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

	createdLane, err := ss.dlRepo.Create(ctx, dedicatedLane)
	if err != nil {
		log.Error().Err(err).Msg("failed to create dedicated lane")
		return nil, eris.Wrap(err, "create dedicated lane")
	}

	now := timeutils.NowUnix()
	suggestion.Status = dedicatedlane.SuggestionStatusAccepted
	suggestion.ProcessedByID = &req.ProcessedByID
	suggestion.ProcessedAt = &now
	suggestion.CreatedDedicatedLaneID = &createdLane.ID

	_, err = ss.suggRepo.Update(ctx, suggestion)
	if err != nil {
		log.Error().Err(err).Msg("failed to update suggestion status")
		// ! Don't fail the operation, but log the error
	}

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

// RejectSuggestion rejects a suggestion and updates the suggestion status.
//
// Parameters:
//   - ctx: The context for the request.
//   - req: The request containing the suggestion ID and the user ID who is rejecting the suggestion.
//
// Returns:
//   - *dedicatedlane.DedicatedLaneSuggestion: The updated suggestion.
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

	if suggestion.IsProcessed() {
		return nil, errors.NewValidationError(
			"suggestion",
			"already_processed",
			"Suggestion has already been processed",
		)
	}

	now := timeutils.NowUnix()
	suggestion.Status = dedicatedlane.SuggestionStatusRejected
	suggestion.ProcessedByID = &req.ProcessedByID
	suggestion.ProcessedAt = &now

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

// AnalyzePatterns analyzes patterns for a given request.
//
// Parameters:
//   - ctx: The context for the request.
//   - req: The request containing the pattern analysis request.
//
// Returns:
//   - *dedicatedlane.PatternAnalysisResult: The result of the pattern analysis.
//   - error: An error if the request fails.
func (ss *SuggestionService) AnalyzePatterns(
	ctx context.Context,
	req *dedicatedlane.PatternAnalysisRequest,
) (*dedicatedlane.PatternAnalysisResult, error) {
	return ss.patternService.AnalyzePatterns(ctx, req)
}

// ExpireOldSuggestions expires old suggestions for a given organization and business unit.
//
// Parameters:
//   - ctx: The context for the request.
//   - orgID: The organization ID.
//   - buID: The business unit ID
//
// Returns:
//   - int64: The number of suggestions expired.
//   - error: An error if the request fails.
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

// checkPermission checks if the user has the required permission to perform the action.
//
// Parameters:
//   - ctx: The context for the request.
//   - action: The action to check permission for.
//   - userID: The user ID.
//   - buID: The business unit ID.
//   - orgID: The organization ID.
//
// Returns:
//   - error: An error if the user does not have the required permission.
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

	result, err := ss.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceDedicatedLane,
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
