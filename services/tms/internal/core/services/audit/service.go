package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/auditjobs"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	LC              fx.Lifecycle
	AuditRepository repositories.AuditRepository
	TemporalClient  client.Client
	Logger          *zap.Logger
	Config          *config.Config
}

type service struct {
	repo           repositories.AuditRepository
	temporalClient client.Client
	logger         *zap.Logger
	config         *config.Config
	sdm            *SensitiveDataManager
}

//nolint:gocritic // dependency injection
func NewService(p ServiceParams) services.AuditService {
	srv := &service{
		repo:           p.AuditRepository,
		temporalClient: p.TemporalClient,
		logger:         p.Logger.Named("service.audit"),
		config:         p.Config,
		sdm:            NewSensitiveDataManager(p.Config.Security.Encryption),
	}

	srv.configureSensitiveDataManager(p.Config.App.Env)
	if err := srv.registerDefaultSensitiveFields(); err != nil {
		p.Logger.Error("failed to register default sensitive fields", zap.Error(err))
	}

	return srv
}

func (s *service) LogAction(params *services.LogActionParams, opts ...services.LogOption) error {
	entry := &audit.Entry{
		ID:             pulid.MustNew("ae_"),
		Resource:       params.Resource,
		ResourceID:     params.ResourceID,
		Operation:      params.Operation,
		CurrentState:   params.CurrentState,
		PreviousState:  params.PreviousState,
		UserID:         params.UserID,
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		Timestamp:      time.Now().Unix(),
		Category:       audit.CategorySystem,
		Metadata:       make(map[string]any),
		Critical:       params.Critical,
	}

	for _, opt := range opts {
		if err := opt(entry); err != nil {
			return err
		}
	}

	if err := entry.Validate(); err != nil {
		s.logger.Error("invalid audit entry", zap.Error(err))
		return fmt.Errorf("invalid audit entry: %w", err)
	}

	if err := s.sdm.SanitizeEntry(entry); err != nil {
		s.logger.Error("failed to sanitize sensitive data", zap.Error(err))
		return fmt.Errorf("failed to sanitize sensitive data: %w", err)
	}

	if params.Critical {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.repo.InsertAuditEntries(ctx, []*audit.Entry{entry}); err != nil {
			s.logger.Error("failed to insert critical audit entry",
				zap.Error(err),
				zap.String("resource", string(params.Resource)),
			)
			return fmt.Errorf("failed to insert critical audit entry: %w", err)
		}
		return nil
	}

	workflowID := "audit-entry-" + entry.ID.String()
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: temporaltype.AuditTaskQueue,
	}

	payload := &auditjobs.ProcessAuditBatchPayload{
		BasePayload: temporaltype.BasePayload{
			Timestamp:      time.Now().Unix(),
			OrganizationID: params.OrganizationID,
			BusinessUnitID: params.BusinessUnitID,
		},
		Entries: []*audit.Entry{entry},
		BatchID: pulid.MustNew("aeb_"),
	}

	_, err := s.temporalClient.ExecuteWorkflow(
		context.Background(),
		workflowOptions,
		auditjobs.ProcessAuditBatchWorkflow,
		payload,
	)
	if err != nil {
		s.logger.Error("failed to start audit workflow",
			zap.Error(err),
			zap.String("workflowID", workflowID),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if fallbackErr := s.repo.InsertAuditEntries(ctx, []*audit.Entry{entry}); fallbackErr != nil {
			s.logger.Error("fallback direct insert also failed", zap.Error(fallbackErr))
			return fmt.Errorf("failed to insert audit entry: %w", fallbackErr)
		}
	}

	return nil
}

func (s *service) List(
	ctx context.Context,
	opts *pagination.QueryOptions,
) (*pagination.ListResult[*audit.Entry], error) {
	log := s.logger.With(
		zap.String("operation", "List"),
		zap.String("buID", opts.TenantOpts.BuID.String()),
		zap.String("userID", opts.TenantOpts.UserID.String()),
	)

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error("failed to list audit entries", zap.Error(err))
		return nil, fmt.Errorf("failed to list audit entries: %w", err)
	}

	return entities, nil
}

func (s *service) ListByResourceID(
	ctx context.Context,
	opts repositories.ListByResourceIDRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	log := s.logger.With(
		zap.String("operation", "ListByResourceID"),
		zap.String("resourceID", opts.ResourceID.String()),
	)

	entities, err := s.repo.ListByResourceID(ctx, opts)
	if err != nil {
		log.Error("failed to list audit entries by resource id", zap.Error(err))
		return nil, fmt.Errorf("failed to list audit entries by resource id: %w", err)
	}

	return entities, nil
}

func (s *service) GetByID(
	ctx context.Context,
	opts repositories.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	log := s.logger.With(
		zap.String("operation", "GetByID"),
		zap.String("auditEntryID", opts.ID.String()),
	)

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error("failed to get audit entry", zap.Error(err))
		return nil, fmt.Errorf("failed to get audit entry by id: %w", err)
	}

	return entity, nil
}

func (s *service) RegisterSensitiveFields(
	resource permission.Resource,
	fields []services.SensitiveField,
) error {
	return s.sdm.RegisterSensitiveFields(resource, fields)
}

func (s *service) registerDefaultSensitiveFields() error {
	if err := s.sdm.RegisterSensitiveFields(permission.ResourceUser, []services.SensitiveField{
		{Name: "password", Action: services.SensitiveFieldOmit},
		{Name: "hashedPassword", Action: services.SensitiveFieldOmit},
		{Name: "emailAddress", Action: services.SensitiveFieldMask},
		{Name: "address", Action: services.SensitiveFieldMask},
	}); err != nil {
		return err
	}

	if err := s.sdm.RegisterSensitiveFields(permission.ResourceOrganization, []services.SensitiveField{
		{Name: "logoUrl", Action: services.SensitiveFieldMask},
		{Name: "taxId", Action: services.SensitiveFieldMask},
	}); err != nil {
		return err
	}

	if err := s.sdm.RegisterSensitiveFields(permission.ResourceWorker, []services.SensitiveField{
		{Name: "licenseNumber", Action: services.SensitiveFieldMask},
		{Name: "dateOfBirth", Action: services.SensitiveFieldMask},
		{Path: "profile", Name: "licenseNumber", Action: services.SensitiveFieldMask},
	}); err != nil {
		return err
	}

	if err := s.sdm.RegisterSensitiveFields(permission.ResourceIntegration, []services.SensitiveField{
		{Path: "configuration", Action: services.SensitiveFieldOmit},
	}); err != nil {
		return err
	}

	return nil
}

func (s *service) configureSensitiveDataManager(environment string) {
	switch environment {
	case "production", "prod":
		s.sdm.SetAutoDetect(true)
		s.sdm.SetMaskStrategy(MaskStrategyStrict)
		s.logger.Info(
			"sensitive data manager configured for production (strict masking, auto-detect ON)",
		)

	case "staging", "stage":
		s.sdm.SetAutoDetect(true)
		s.sdm.SetMaskStrategy(MaskStrategyDefault)
		s.logger.Info(
			"sensitive data manager configured for staging (default masking, auto-detect ON)",
		)

	case "development", "dev":
		s.sdm.SetAutoDetect(true)
		s.sdm.SetMaskStrategy(MaskStrategyPartial)
		s.logger.Info(
			"sensitive data manager configured for development (partial masking, auto-detect ON)",
		)

	case "test", "testing":
		s.sdm.SetAutoDetect(false)
		s.sdm.SetMaskStrategy(MaskStrategyPartial)
		s.logger.Info(
			"sensitive data manager configured for testing (partial masking, auto-detect OFF)",
		)

	default:
		s.sdm.SetAutoDetect(true)
		s.sdm.SetMaskStrategy(MaskStrategyDefault)
		s.logger.Warn(
			"unknown environment, using default configuration",
			zap.String("environment", environment),
		)
	}
}
