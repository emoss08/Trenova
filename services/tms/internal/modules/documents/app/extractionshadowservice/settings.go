package extractionshadowservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionshadowports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) GetSettings(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*extractionshadow.ShadowSettings, error) {
	settings, err := s.settings.Get(ctx, tenant)
	switch {
	case err == nil:
		return settings, nil
	case errortypes.IsNotFoundError(err):
		return extractionshadow.DefaultSettings(tenant.OrgID, tenant.BuID), nil
	default:
		return nil, err
	}
}

func (s *Service) UpdateSettings(
	ctx context.Context,
	req *extractionshadowports.UpdateExtractionShadowSettingsRequest,
	actor *permissionports.RequestActor,
) (*extractionshadow.ShadowSettings, error) {
	current, err := s.GetSettings(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if current.Version != req.Version {
		return nil, errortypes.NewBusinessError(
			"Someone else changed the shadow settings; reload them and try again",
		)
	}

	previous := auditableSettings(current)
	current.Enabled = req.Enabled
	current.SamplePercent = req.SamplePercent
	current.DailyLimit = req.DailyLimit
	current.ProviderID = pulid.PtrOrNil(req.ProviderID)
	current.UpdatedByID = pulid.PtrOrNil(actorID(actor, req.TenantInfo))

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

	saved, err := s.settings.Save(ctx, current)
	if err != nil {
		return nil, err
	}

	s.logAction(actor, &auditports.LogActionParams{
		Resource:       permission.ResourceAgentEvalSuite,
		ResourceID:     saved.ID.String(),
		Operation:      permission.OpUpdate,
		CurrentState:   jsonutils.MustToJSON(auditableSettings(saved)),
		PreviousState:  jsonutils.MustToJSON(previous),
		OrganizationID: saved.OrganizationID,
		BusinessUnitID: saved.BusinessUnitID,
	}, "Extraction shadow settings updated")

	return saved, nil
}

func (s *Service) checkCandidate(
	ctx context.Context,
	tenant pagination.TenantInfo,
	providerID pulid.ID,
	enabled bool,
) error {
	provider, err := s.providers.GetByID(ctx, aiproviderports.GetAIProviderByIDRequest{
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

func auditableSettings(settings *extractionshadow.ShadowSettings) map[string]any {
	return map[string]any{
		"id":            settings.ID,
		"enabled":       settings.Enabled,
		"providerId":    settings.ProviderID,
		"samplePercent": settings.SamplePercent,
		"dailyLimit":    settings.DailyLimit,
	}
}

func actorID(actor *permissionports.RequestActor, tenant pagination.TenantInfo) pulid.ID {
	auditActor := actor.AuditActorOrSystem()
	return pulid.FirstNotNil(auditActor.UserID, auditActor.PrincipalID, tenant.UserID)
}
