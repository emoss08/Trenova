package extractionshadowservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionshadowports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	ReasonDisabled      = "disabled"
	ReasonSameProvider  = "same_provider"
	ReasonNotSampled    = "not_sampled"
	ReasonDailyLimit    = "daily_limit"
	ReasonSuperseded    = "superseded"
	ReasonUnavailable   = "unavailable"
	ReasonAlreadyExists = "already_sampled"

	startFailedReason = "The shadow extraction could not be started"
)

func (s *Service) ConsiderExtraction(
	ctx context.Context,
	req *extractionshadowports.ConsiderExtractionShadowRequest,
) (*extractionshadowports.ExtractionShadowDecision, error) {
	if s.starter == nil || s.predictor == nil {
		return &extractionshadowports.ExtractionShadowDecision{Reason: ReasonUnavailable}, nil
	}

	settings, err := s.GetSettings(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	candidate := settings.CandidateID()
	switch {
	case candidate.IsNil():
		return &extractionshadowports.ExtractionShadowDecision{Reason: ReasonDisabled}, nil
	case candidate == req.ProductionProviderID:
		return &extractionshadowports.ExtractionShadowDecision{Reason: ReasonSameProvider}, nil
	case !extractionshadow.Sampled(req.DocumentID, req.ExtractedAt, settings.SamplePercent):
		return &extractionshadowports.ExtractionShadowDecision{Reason: ReasonNotSampled}, nil
	}

	current, err := s.isCurrentExtraction(ctx, req)
	if err != nil {
		return nil, err
	}
	if !current {
		return &extractionshadowports.ExtractionShadowDecision{Reason: ReasonSuperseded}, nil
	}

	started, err := s.results.CountCreatedSince(
		ctx, req.TenantInfo, s.now()-timeutils.SecondsPerDay,
	)
	if err != nil {
		return nil, err
	}
	if started >= settings.DailyLimit {
		return &extractionshadowports.ExtractionShadowDecision{Reason: ReasonDailyLimit}, nil
	}

	provider, err := s.providers.GetByID(ctx, agentports.GetAIProviderByIDRequest{
		ID:         candidate,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			s.l.Warn("the shadow provider no longer exists",
				zap.String("providerId", candidate.String()),
			)
			return &extractionshadowports.ExtractionShadowDecision{Reason: ReasonDisabled}, nil
		}
		return nil, err
	}

	entity := &extractionshadow.ShadowResult{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		DocumentID:     req.DocumentID,
		ExtractedAt:    req.ExtractedAt,
		Status:         extractionshadow.ResultStatusPending,
		ProviderID:     provider.ID,
		ProviderName: stringutils.TruncateRunes(
			provider.Name,
			extractionshadow.MaxProviderNameRunes,
		),
		ProductionModel: stringutils.TruncateRunes(
			req.ProductionModel,
			extractionshadow.MaxModelRunes,
		),
		ProductionProviderID: pulid.PtrOrNil(req.ProductionProviderID),
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	result, created, err := s.results.Create(ctx, entity)
	if err != nil {
		return nil, err
	}
	if !created && (result.WorkflowID != "" || result.Status.IsSettled()) {
		return &extractionshadowports.ExtractionShadowDecision{
			Sampled:    true,
			ResultID:   result.ID,
			WorkflowID: result.WorkflowID,
			Reason:     ReasonAlreadyExists,
		}, nil
	}

	return s.start(ctx, result)
}

func (s *Service) start(
	ctx context.Context,
	result *extractionshadow.ShadowResult,
) (*extractionshadowports.ExtractionShadowDecision, error) {
	workflowID, err := s.starter.StartExtractionShadow(
		ctx,
		&extractionshadowports.ExtractionShadowStart{
			TenantInfo: tenantOf(result),
			ResultID:   result.ID,
		},
	)
	if err != nil {
		s.l.Error("failed to start a shadow extraction",
			zap.String("resultId", result.ID.String()),
			zap.Error(err),
		)
		result.Settle(extractionshadow.ResultStatusFailed, startFailedReason, s.now())
		if _, saveErr := s.results.Save(ctx, result); saveErr != nil {
			return nil, errors.Join(err, saveErr)
		}

		return &extractionshadowports.ExtractionShadowDecision{
			Sampled:  true,
			ResultID: result.ID,
			Reason:   startFailedReason,
		}, nil
	}

	result.WorkflowID = workflowID
	if _, err = s.results.Save(ctx, result); err != nil {
		return nil, err
	}

	return &extractionshadowports.ExtractionShadowDecision{
		Sampled:    true,
		ResultID:   result.ID,
		WorkflowID: workflowID,
	}, nil
}

func (s *Service) isCurrentExtraction(
	ctx context.Context,
	req *extractionshadowports.ConsiderExtractionShadowRequest,
) (bool, error) {
	content, err := s.contents.GetByDocumentID(ctx, req.DocumentID, req.TenantInfo)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}

	return content.LastExtractedAt != nil && *content.LastExtractedAt == req.ExtractedAt, nil
}
