package telematicsservice

import (
	"context"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type ProcessWebhookRequest struct {
	ProviderType integration.Type
	Token        string
	Body         []byte
	Signature    string
	Timestamp    string
}

func (s *Service) ProcessWebhook(
	ctx context.Context,
	req *ProcessWebhookRequest,
) error {
	cfg, err := s.repo.GetWebhookConfigByToken(ctx, req.ProviderType, req.Token)
	if err != nil {
		return err
	}

	if strings.TrimSpace(cfg.WebhookSecret) == "" {
		return errortypes.NewValidationError(
			"webhookSecret",
			errortypes.ErrInvalid,
			"Telematics webhook secret is not configured",
		)
	}

	secret, err := s.encryptionService.DecryptString(cfg.WebhookSecret)
	if err != nil {
		return errortypes.NewBusinessError(
			"failed to resolve telematics webhook secret",
		).WithInternal(err)
	}

	provider, err := s.providerFactory.ProviderOfType(ctx, cfg.TenantInfo, req.ProviderType)
	if err != nil {
		return err
	}

	if err = provider.VerifyWebhookSignature(
		secret,
		req.Timestamp,
		req.Body,
		req.Signature,
		time.Now(),
		webhookMaxSkew,
	); err != nil {
		return errortypes.NewValidationError(
			"signature",
			errortypes.ErrInvalid,
			"Invalid telematics webhook signature",
		)
	}

	event, err := provider.ParseWebhookEvent(req.Body)
	if err != nil {
		return errortypes.NewValidationError(
			"body",
			errortypes.ErrInvalid,
			"Invalid telematics webhook payload",
		)
	}

	return s.handleEvent(ctx, cfg.TenantInfo, string(provider.Type()), event)
}

func (s *Service) handleEvent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	providerType string,
	event *services.ProviderWebhookEvent,
) error {
	record := &telematics.TelematicsEvent{
		ID:             telematics.NewEventID(),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Provider:       providerType,
		EventID:        event.EventID,
		EventType:      telematics.EventType(event.EventType),
		OccurredAt:     event.OccurredAt,
		CreatedAt:      timeutils.NowUnix(),
	}

	if len(event.Payload) > 0 {
		payload := make(map[string]any)
		if err := sonic.Unmarshal(event.Payload, &payload); err == nil {
			record.Payload = payload
		}
	}

	vehicleID := event.VehicleID
	vehicleVIN := ""
	driverID := event.DriverID
	applyGeofenceEvent(record, event.Geofence, &vehicleID, &vehicleVIN, &driverID)

	if vehicleID != "" || vehicleVIN != "" {
		s.resolveEventTractor(ctx, tenantInfo, vehicleID, vehicleVIN, record)
	}
	if driverID != "" {
		s.resolveEventWorker(ctx, tenantInfo, driverID, record)
	}

	if record.LocationID.IsNil() && event.Stop != nil {
		if stopID, ok := stopExternalLocationID(event.Stop.AddressExternalIDs); ok {
			record.LocationID = stopID
		} else if stopID, ok = stopExternalLocationID(event.Stop.StopExternalIDs); ok {
			record.LocationID = stopID
		}
	}

	inserted, err := s.repo.InsertEvent(ctx, record)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}

	s.dispatchEventSideEffects(ctx, tenantInfo, providerType, record, event)

	s.publishInvalidation(ctx, tenantInfo, "telematicsEvent")
	return nil
}

func (s *Service) dispatchEventSideEffects(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	providerType string,
	record *telematics.TelematicsEvent,
	event *services.ProviderWebhookEvent,
) {
	switch event.Kind {
	case services.ProviderEventKindGeofenceEntry, services.ProviderEventKindStopArrival:
		s.applyStopEvent(ctx, tenantInfo, record, repositories.StopActualActionArrive)
	case services.ProviderEventKindGeofenceExit, services.ProviderEventKindStopDeparture:
		s.applyStopEvent(ctx, tenantInfo, record, repositories.StopActualActionDepart)
	case services.ProviderEventKindFormSubmission:
		if err := s.handleFormEvent(ctx, tenantInfo, providerType, event.Form); err != nil {
			s.l.Warn("failed to ingest telematics form event",
				zap.String("organizationId", tenantInfo.OrgID.String()),
				zap.Error(err))
		}
	case services.ProviderEventKindOther:
	}
}

func (s *Service) handleFormEvent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	providerType string,
	form *services.ProviderFormEvent,
) error {
	if form == nil {
		return nil
	}
	workersByExternalID, err := s.workersByExternalID(ctx, tenantInfo)
	if err != nil {
		return err
	}
	mappingsByTemplate, err := s.formMappingsByTemplate(ctx, tenantInfo)
	if err != nil {
		return err
	}
	return s.ingestFormSubmission(
		ctx,
		tenantInfo,
		workersByExternalID,
		mappingsByTemplate,
		&ingestFormInput{
			Provider:     providerType,
			SubmissionID: form.SubmissionID,
			TemplateID:   form.TemplateID,
			TemplateName: form.TemplateName,
			DriverID:     form.DriverID,
			RouteStopID:  form.RouteStopID,
			SubmittedAt:  form.SubmittedAt,
			Fields:       form.Fields,
		},
	)
}

func applyGeofenceEvent(
	record *telematics.TelematicsEvent,
	geofence *services.ProviderGeofenceEvent,
	vehicleID *string,
	vehicleVIN *string,
	driverID *string,
) {
	if geofence == nil {
		return
	}
	record.AddressName = geofence.AddressName
	if locationID, ok := geofence.AddressExternalIDs["trenovaLocationId"]; ok {
		if parsed, parseErr := pulid.Parse(locationID); parseErr == nil {
			record.LocationID = parsed
		}
	}
	if *vehicleID == "" {
		*vehicleID = geofence.VehicleID
	}
	*vehicleVIN = geofence.VehicleVIN
	if *driverID == "" {
		*driverID = geofence.DriverID
	}
}

func (s *Service) resolveEventTractor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	providerVehicleID string,
	vin string,
	record *telematics.TelematicsEvent,
) {
	mappings, err := s.repo.ListTractorMappings(ctx, tenantInfo)
	if err != nil {
		s.l.Warn("failed to resolve tractor for telematics event",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Error(err))
		return
	}

	normalizedVin := strings.ToUpper(strings.TrimSpace(vin))
	for _, mapping := range mappings {
		if providerVehicleID != "" && mapping.ExternalID == providerVehicleID {
			record.TractorID = mapping.TractorID
			return
		}
		if normalizedVin != "" &&
			strings.ToUpper(strings.TrimSpace(mapping.Vin)) == normalizedVin {
			record.TractorID = mapping.TractorID
			return
		}
	}
}

func (s *Service) resolveEventWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	providerDriverID string,
	record *telematics.TelematicsEvent,
) {
	workersByExternalID, err := s.workersByExternalID(ctx, tenantInfo)
	if err != nil {
		s.l.Warn("failed to resolve worker for telematics event",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Error(err))
		return
	}
	if workerID, ok := workersByExternalID[providerDriverID]; ok {
		record.WorkerID = workerID
	}
}
