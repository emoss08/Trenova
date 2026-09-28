package extractionrolloutservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) Update(
	ctx context.Context,
	req *services.UpdateExtractionRolloutRequest,
	actor *services.RequestActor,
) (*extractionrollout.ExtractionRollout, error) {
	current, err := s.Get(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if current.Version != req.Version {
		return nil, errortypes.NewBusinessError(
			"Someone else changed the rollout; reload it and try again",
		)
	}

	previous := auditable(current)
	current.Apply(&extractionrollout.Change{
		Enabled:                    req.Enabled,
		ProviderID:                 req.ProviderID,
		Percent:                    req.Percent,
		MaxAccuracyDropPoints:      req.MaxAccuracyDropPoints,
		MaxRejectionIncreasePoints: req.MaxRejectionIncreasePoints,
	}, s.now())
	auditActor := actor.AuditActorOrSystem()
	current.UpdatedByID = pulid.PtrOrNil(
		pulid.FirstNotNil(auditActor.UserID, auditActor.PrincipalID, req.TenantInfo.UserID),
	)

	multiErr := errortypes.NewMultiError()
	current.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if current.ProviderID != nil {
		if err = s.checkCandidate(
			ctx,
			req.TenantInfo,
			*current.ProviderID,
			current.Enabled,
		); err != nil {
			return nil, err
		}
	}

	saved, err := s.rollouts.Save(ctx, current)
	if err != nil {
		return nil, err
	}

	s.logAction(actor, &services.LogActionParams{
		Resource:       permission.ResourceAIProvider,
		ResourceID:     saved.ID.String(),
		Operation:      permission.OpUpdate,
		CurrentState:   jsonutils.MustToJSON(auditable(saved)),
		PreviousState:  jsonutils.MustToJSON(previous),
		OrganizationID: saved.OrganizationID,
		BusinessUnitID: saved.BusinessUnitID,
	}, "Extraction rollout updated")

	return saved, nil
}

func (s *Service) checkCandidate(
	ctx context.Context,
	tenant pagination.TenantInfo,
	providerID pulid.ID,
	enabled bool,
) error {
	provider, err := s.providers.GetByID(ctx, repositories.GetAIProviderByIDRequest{
		ID:         providerID,
		TenantInfo: tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return errortypes.NewValidationError(
				"providerId", errortypes.ErrInvalid, "The chosen AI provider no longer exists",
			)
		}
		return err
	}
	if !enabled {
		return nil
	}
	if ok, reason := provider.CanServeTask(aiprovider.TaskDocumentExtraction); !ok {
		return errortypes.NewValidationError(
			"providerId",
			errortypes.ErrInvalid,
			"{0} cannot run document extraction: {1}",
			provider.Name,
			reason,
		)
	}

	return nil
}
