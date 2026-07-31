package modeprofileservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const ResourceTypeShipment = "shipment"

type ServiceParams struct {
	fx.In

	ProfileRepo   repositories.ModeProfileRepository
	DeviationRepo repositories.DeviationRepository
	Logger        *zap.Logger
}

type service struct {
	profileRepo   repositories.ModeProfileRepository
	deviationRepo repositories.DeviationRepository
	l             *zap.Logger
}

func NewService(p ServiceParams) services.ModeProfileService {
	return &service{
		profileRepo:   p.ProfileRepo,
		deviationRepo: p.DeviationRepo,
		l:             p.Logger.Named("mode-profile-service"),
	}
}

func severityToEnforcement(severity errortypes.Severity) tenant.EnforcementLevel {
	if severity == errortypes.SeverityRequireReview {
		return tenant.EnforcementLevelRequireReview
	}
	return tenant.EnforcementLevelWarn
}

func (s *service) RecordDeviations(
	ctx context.Context,
	req *services.RecordDeviationsRequest,
) error {
	if req.Policy == nil {
		return nil
	}

	log := s.l.With(
		zap.String("operation", "RecordDeviations"),
		zap.String("resourceId", req.ResourceID.String()),
	)

	now := timeutils.NowUnix()
	deviations := make([]*modeprofile.Deviation, 0, len(req.Advisories))
	firedKeys := make([]string, 0, len(req.Advisories))

	for _, advisory := range req.Advisories {
		if advisory == nil || advisory.RuleKey == "" {
			continue
		}

		ruleKey := modeprofile.RuleKey(advisory.RuleKey)
		resolved, ok := req.Policy.Rule(ruleKey)
		if !ok {
			continue
		}

		provenance := resolved.Provenance
		deviation := &modeprofile.Deviation{
			BusinessUnitID: req.TenantInfo.BuID,
			OrganizationID: req.TenantInfo.OrgID,
			ModeProfileID:  req.Policy.ProfileID,
			RuleKey:        ruleKey,
			Capability:     resolved.Capability,
			Enforcement:    severityToEnforcement(advisory.Severity),
			ResourceType:   req.ResourceType,
			ResourceID:     req.ResourceID,
			Field:          advisory.Field,
			Message:        advisory.Message,
			State:          modeprofile.DeviationStateOpen,
			OccurredAt:     now,
			Provenance:     &provenance,
		}

		deviations = append(deviations, deviation)
		firedKeys = append(firedKeys, advisory.RuleKey)
	}

	if err := s.deviationRepo.SupersedeOpenForResource(
		ctx,
		&repositories.SupersedeDeviationsRequest{
			TenantInfo:   req.TenantInfo,
			ResourceType: req.ResourceType,
			ResourceID:   req.ResourceID,
		},
	); err != nil {
		log.Error("failed to supersede prior deviations", zap.Error(err))
		return err
	}

	if len(deviations) == 0 {
		return nil
	}

	if err := s.deviationRepo.RecordMany(ctx, deviations); err != nil {
		log.Error("failed to record deviations", zap.Error(err))
		return err
	}

	return nil
}

func (s *service) Acknowledge(
	ctx context.Context,
	req *services.AcknowledgeDeviationServiceRequest,
) (*modeprofile.Deviation, error) {
	multiErr := errortypes.NewMultiError()

	if len(req.Reason) < modeprofile.MinAcknowledgementReasonLength {
		multiErr.Add("reason", errortypes.ErrInvalid,
			"Provide a reason of at least 10 characters explaining why this deviation is accepted")
	}

	if req.AcknowledgedByID.IsNil() {
		multiErr.Add("acknowledgedById", errortypes.ErrRequired,
			"An acknowledging user is required")
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.deviationRepo.Acknowledge(ctx, &repositories.AcknowledgeDeviationRequest{
		DeviationID:      req.DeviationID,
		TenantInfo:       req.TenantInfo,
		AcknowledgedByID: req.AcknowledgedByID,
		Reason:           req.Reason,
	})
}

func (s *service) ListDeviations(
	ctx context.Context,
	req *repositories.ListDeviationsRequest,
) (*pagination.ListResult[*modeprofile.Deviation], error) {
	return s.deviationRepo.List(ctx, req)
}

func (s *service) Ledger(
	ctx context.Context,
	req *repositories.DeviationLedgerRequest,
) ([]*repositories.DeviationLedgerEntry, error) {
	return s.deviationRepo.Ledger(ctx, req)
}
