package customer

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/core/temporaljobs/searchjobs"
	"github.com/emoss08/trenova/internal/infrastructure/meilisearch/providers"
	"github.com/emoss08/trenova/pkg/meilisearchtype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/customervalidator"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger         *zap.Logger
	Repo           repositories.CustomerRepository
	TemporalClient client.Client
	AuditService   services.AuditService
	Validator      *customervalidator.Validator
	SearchHelper   *providers.SearchHelper
}

type Service struct {
	l              *zap.Logger
	repo           repositories.CustomerRepository
	temporalClient client.Client
	as             services.AuditService
	v              *customervalidator.Validator
	searchHelper   *providers.SearchHelper
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:              p.Logger.Named("service.customer"),
		repo:           p.Repo,
		temporalClient: p.TemporalClient,
		as:             p.AuditService,
		v:              p.Validator,
		searchHelper:   p.SearchHelper,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListCustomerRequest,
) (*pagination.ListResult[*customer.Customer], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetCustomerByIDRequest,
) (*customer.Customer, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetDocumentRequirements(
	ctx context.Context,
	cusID pulid.ID,
) ([]*repositories.CustomerDocRequirementResponse, error) {
	return s.repo.GetDocumentRequirements(ctx, cusID)
}

func (s *Service) Create(
	ctx context.Context,
	entity *customer.Customer,
	userID pulid.ID,
) (*customer.Customer, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("userID", userID.String()),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, entity); err != nil {
		log.With(
			zap.Any("entity", entity),
			zap.Error(err),
		).Error("failed to validate customer")
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceCustomer,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Customer created"),
	)
	if err != nil {
		log.Error("failed to log customer creation", zap.Error(err))
	}

	if err = s.IndexInSearch(ctx, createdEntity.ID, createdEntity.OrganizationID, createdEntity.BusinessUnitID); err != nil {
		log.Warn("failed to index created customer in search", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *customer.Customer,
	userID pulid.ID,
) (*customer.Customer, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err := s.v.Validate(ctx, valCtx, entity); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:    entity.ID,
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update customer", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceCustomer,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Customer updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log customer update", zap.Error(err))
	}

	if err = s.IndexInSearch(ctx, updatedEntity.ID, updatedEntity.OrganizationID, updatedEntity.OrganizationID); err != nil {
		log.Warn("failed to index customer in search", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) IndexInSearch(ctx context.Context, cusID, orgID, buID pulid.ID) error {
	log := s.l.With(
		zap.String("operation", "IndexInSearch"),
		zap.String("customerID", cusID.String()),
		zap.String("orgID", orgID.String()),
		zap.String("buID", buID.String()),
	)

	payload := &searchjobs.IndexEntityPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		EntityType: meilisearchtype.EntityTypeCustomer,
		EntityID:   cusID,
	}

	workflowID := fmt.Sprintf("index-customer-in-search-%s-%d", cusID.String(), time.Now().Unix())

	_, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: searchjobs.SearchTaskQueue,
		},
		searchjobs.IndexEntityWorkflow,
		payload,
	)
	if err != nil {
		log.Error("failed to execute workflow", zap.Error(err))
	}

	return nil
}
