//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package customerservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         customerports.CustomerRepository
	CacheRepo    customerports.CustomerCacheRepository
	Validator    *Validator
	AuditService auditports.AuditService
	Realtime     agentports.RealtimeService
	Transformer  accessorialchargeports.DataTransformer
	DB           kernelports.Transactor                     `optional:"true"`
	Sync         accountingsyncports.AccountingSyncEnqueuer `optional:"true"`
	Credit       customerports.CreditReviewer               `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         customerports.CustomerRepository
	cacheRepo    customerports.CustomerCacheRepository
	validator    *Validator
	auditService auditports.AuditService
	realtime     agentports.RealtimeService
	transformer  accessorialchargeports.DataTransformer
	db           kernelports.Transactor
	sync         accountingsyncports.AccountingSyncEnqueuer
	credit       customerports.CreditReviewer
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.customer"),
		repo:         p.Repo,
		cacheRepo:    p.CacheRepo,
		validator:    p.Validator,
		auditService: p.AuditService,
		realtime:     p.Realtime,
		transformer:  p.Transformer,
		db:           p.DB,
		sync:         p.Sync,
		credit:       p.Credit,
	}
}

func (s *Service) updateAndQueueSync(
	ctx context.Context,
	entity *customer.Customer,
	original *customer.Customer,
) (*customer.Customer, error) {
	if s.db == nil || s.sync == nil {
		return s.repo.Update(ctx, entity)
	}

	var updated *customer.Customer
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		var txErr error
		if updated, txErr = s.repo.Update(txCtx, entity); txErr != nil {
			return txErr
		}
		if updated.SamePartyDetails(original) {
			return nil
		}
		return accountingsyncports.EnqueueAccountingSync(
			txCtx,
			s.sync,
			accountingsyncports.CustomerSyncRequest(
				pagination.TenantInfo{OrgID: updated.OrganizationID, BuID: updated.BusinessUnitID},
				updated.ID,
				updated.Name,
				updated.Version,
			),
		)
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) List(
	ctx context.Context,
	req *customerports.ListCustomerRequest,
) (*pagination.ListResult[*customer.Customer], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *customerports.ListCustomerConnectionRequest,
) (*pagination.CursorListResult[*customer.Customer], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req customerports.GetCustomerByIDRequest,
) (*customer.Customer, error) {
	entity, err := s.cacheRepo.GetByID(ctx, req)
	if err == nil {
		return entity, nil
	}
	if !errors.Is(err, customerports.ErrCacheMiss) {
		s.l.Warn(
			"failed to load customer from cache",
			zap.Error(err),
			zap.String("customerID", req.ID.String()),
		)
	}

	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetByIDs(
	ctx context.Context,
	req customerports.GetCustomersByIDsRequest,
) ([]*customer.Customer, error) {
	return s.repo.GetByIDs(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *customerports.CustomerSelectOptionsRequest,
) (*pagination.ListResult[*customer.Customer], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) GetBillingProfile(
	ctx context.Context,
	req customerports.GetCustomerBillingProfileRequest,
) (*customer.CustomerBillingProfile, error) {
	return s.repo.GetBillingProfile(ctx, req)
}

func (s *Service) BulkUpdateStatus(
	ctx context.Context,
	req *customerports.BulkUpdateCustomerStatusRequest,
) ([]*customer.Customer, error) {
	log := s.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	changes, err := s.PlanBulkUpdateStatus(ctx, req)
	if err != nil {
		return nil, err
	}
	originalEntities := kernelports.Befores(changes)

	entities, err := s.repo.BulkUpdateStatus(ctx, req)
	if err != nil {
		log.Error("failed to bulk update customer status", zap.Error(err))
		return nil, err
	}

	entries := auditservice.BuildBulkLogEntries(
		&auditservice.BulkLogEntriesParams[*customer.Customer]{
			Resource:  permission.ResourceCustomer,
			Operation: permission.OpUpdate,
			UserID:    req.TenantInfo.UserID,
			Updated:   entities,
			Originals: originalEntities,
		},
		auditservice.WithComment("Customer status updated"),
	)

	if err = s.auditService.LogActions(entries); err != nil {
		log.Error("failed to log audit actions", zap.Error(err))
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ActorUserID:    req.TenantInfo.UserID,
		Resource:       "customers",
		Action:         "bulk_updated",
	}); err != nil {
		log.Warn("failed to publish customer invalidation", zap.Error(err))
	}

	return entities, nil
}

func (s *Service) Create(
	ctx context.Context,
	entity *customer.Customer,
	actor *permissionports.RequestActor,
) (*customer.Customer, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
	)

	planned, err := s.PlanCreate(ctx, entity)
	if err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, planned)
	if err != nil {
		log.Error("failed to create customer", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceCustomer,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	},
		auditservice.WithComment("Customer created"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
		ActorUserID:    auditActor.UserID,
		ActorType:      auditActor.PrincipalType,
		ActorID:        auditActor.PrincipalID,
		ActorAPIKeyID:  auditActor.APIKeyID,
		Resource:       "customers",
		Action:         "created",
		RecordID:       createdEntity.GetID(),
		Entity:         createdEntity,
	}); err != nil {
		log.Warn("failed to publish customer invalidation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *customer.Customer,
	actor *permissionports.RequestActor,
) (*customer.Customer, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
	)

	change, err := s.PlanUpdate(ctx, entity)
	if err != nil {
		return nil, err
	}
	original := change.Before

	updatedEntity, err := s.updateAndQueueSync(ctx, entity, original)
	if err != nil {
		log.Error("failed to update customer", zap.Error(err))
		return nil, err
	}

	s.reviewCredit(ctx, updatedEntity, original)

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceCustomer,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Customer updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
		ActorUserID:    auditActor.UserID,
		ActorType:      auditActor.PrincipalType,
		ActorID:        auditActor.PrincipalID,
		ActorAPIKeyID:  auditActor.APIKeyID,
		Resource:       "customers",
		Action:         "updated",
		RecordID:       updatedEntity.GetID(),
		Entity:         updatedEntity,
	}); err != nil {
		log.Warn("failed to publish customer invalidation", zap.Error(err))
	}

	return updatedEntity, nil
}

// reviewCredit holds or releases the customer at once when its credit terms
// change, rather than leaving the new limit to the next posting or the
// nightly sweep, and returns the standing the review left.
func (s *Service) reviewCredit(ctx context.Context, updated, original *customer.Customer) {
	if s.credit == nil || updated == nil || !updated.HasBillingProfile() ||
		!creditTermsChanged(original, updated) {
		return
	}
	tenant := pagination.TenantInfo{OrgID: updated.OrganizationID, BuID: updated.BusinessUnitID}
	if err := s.credit.Review(ctx, tenant, []pulid.ID{updated.ID}); err != nil {
		s.l.Warn("failed to review credit after a customer update; the nightly sweep will retry",
			zap.String("customerId", updated.ID.String()), zap.Error(err))
		return
	}
	profile, err := s.repo.GetBillingProfile(ctx, customerports.GetCustomerBillingProfileRequest{
		CustomerID: updated.ID,
		TenantInfo: tenant,
	})
	if err != nil {
		s.l.Warn("failed to reread the billing profile after a credit review", zap.Error(err))
		return
	}
	updated.BillingProfile = profile
}

func creditTermsChanged(original, updated *customer.Customer) bool {
	if original == nil || !original.HasBillingProfile() {
		return true
	}
	before, after := original.BillingProfile, updated.BillingProfile
	return before.EnforceCreditLimit != after.EnforceCreditLimit ||
		before.AutoCreditHold != after.AutoCreditHold ||
		before.CreditStatus != after.CreditStatus ||
		before.CreditLimit.Valid != after.CreditLimit.Valid ||
		!before.CreditLimit.Decimal.Equal(after.CreditLimit.Decimal)
}
