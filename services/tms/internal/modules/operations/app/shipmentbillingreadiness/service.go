package shipmentbillingreadiness

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Service struct {
	l                  *zap.Logger
	repo               shipmentports.ShipmentRepository
	customerRepo       customerports.CustomerRepository
	documentRepo       documentports.DocumentRepository
	billingRepo        tenantports.BillingControlRepository
	serviceFailureRepo servicefailureports.ServiceFailureRepository
}

func (s *Service) GetBillingReadiness(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*shipmentbillingports.ShipmentBillingReadiness, error) {
	entity, err := s.repo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return s.evaluateBillingReadiness(ctx, entity)
}

func (s *Service) ValidateBillingReadinessForStatusChange(
	ctx context.Context,
	entity *shipment.Shipment,
) *errortypes.MultiError {
	if entity.Status != shipmentvocab.StatusReadyToInvoice ||
		s.customerRepo == nil ||
		s.documentRepo == nil ||
		s.billingRepo == nil {
		return nil
	}

	readiness, err := s.evaluateBillingReadiness(ctx, entity)
	if err != nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			"Unable to evaluate shipment billing readiness",
		)
		return multiErr
	}

	if readiness.CanMarkReadyToInvoice {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"status",
		errortypes.ErrInvalidOperation,
		"Shipment cannot be marked ready to invoice until billing requirements are satisfied",
	)

	for _, failure := range readiness.ValidationFailures {
		field := failure.Field
		if field == "" {
			field = "status"
		}

		multiErr.Add(field, errortypes.ErrInvalidOperation, failure.Message)
	}

	for _, requirement := range readiness.MissingRequirements {
		multiErr.Add(
			"status",
			errortypes.ErrInvalidOperation,
			"Missing required document: {0}", requirement.DocumentTypeName,
		)
	}

	return multiErr
}

// billingControlFor reads the organization's billing policy, at most once per
// bulk transfer. The policy is one row per organization and every shipment in a
// run asks the same question of it.
func (s *Service) billingControlFor(
	ctx context.Context,
	orgID pulid.ID,
	cache *BillingReadinessCache,
) (*billingcontrol.BillingControl, error) {
	if cache != nil && cache.billingControlKnown {
		return cache.billingControl, nil
	}

	control, err := s.billingRepo.GetByOrgID(ctx, orgID)
	switch {
	case err == nil:
	case errortypes.IsNotFoundError(err):
		// A tenant that has not configured billing has no policy, which is an
		// answer worth caching rather than re-asking for every shipment.
		control = nil
	default:
		return nil, err
	}

	if cache != nil {
		cache.billingControl = control
		cache.billingControlKnown = true
	}

	return control, nil
}

func (s *Service) evaluateBillingReadiness(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipmentbillingports.ShipmentBillingReadiness, error) {
	return s.EvaluateBillingReadinessCached(ctx, entity, nil, false)
}

// EvaluateBillingReadinessCached is the readiness check with somewhere to put
// the answers that do not vary by shipment.
//
// A bulk transfer asks the same organization-level questions once per shipment,
// which is the difference between one query and five thousand. A nil cache means
// a single shipment is being evaluated on its own, and nothing is shared.
func (s *Service) EvaluateBillingReadinessCached(
	ctx context.Context,
	entity *shipment.Shipment,
	cache *BillingReadinessCache,
	detailsLoaded bool,
) (*shipmentbillingports.ShipmentBillingReadiness, error) {
	if s.customerRepo == nil || s.documentRepo == nil || s.billingRepo == nil {
		return nil, errortypes.NewConflictError("Shipment billing readiness service is unavailable")
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	resolution, err := s.resolvePayerShares(ctx, entity, tenantInfo, detailsLoaded)
	if err != nil {
		return nil, err
	}

	payers, err := s.customerRepo.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: resolution.PayerIDs(),
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return nil, err
	}

	sources := &Sources{
		Payers:    payersByID(payers),
		documents: make(map[string][]*document.Document, 1),
		failures:  make(map[pulid.ID][]*servicefailure.ServiceFailure, 1),
	}
	err = s.completePayers(ctx, tenantInfo, sources.Payers, resolution.DefaultPayerID)
	if err != nil {
		return nil, err
	}

	sources.control, err = s.billingControlFor(ctx, entity.OrganizationID, cache)
	if err != nil {
		return nil, err
	}

	docs, err := s.documentRepo.GetByResourceID(ctx, &documentports.GetDocumentsByResourceRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		ResourceID:   entity.ID.String(),
		ResourceType: "shipment",
	})
	if err != nil {
		return nil, err
	}
	sources.documents[entity.ID.String()] = docs

	if failures, failureErr := s.unresolvedServiceFailures(ctx, entity); failureErr != nil {
		s.l.Warn("failed to apply service failure billing context",
			zap.String("shipmentId", entity.ID.String()),
			zap.Error(failureErr),
		)
	} else {
		sources.failures[entity.ID] = failures
	}

	return s.ReadinessFrom(entity, resolution, sources), nil
}

// completePayers reads a default payer the batched read did not return, which
// readiness cannot be decided without.
func (s *Service) completePayers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	payers map[pulid.ID]*customer.Customer,
	defaultPayerIDs ...pulid.ID,
) error {
	for _, payerID := range defaultPayerIDs {
		if _, ok := payers[payerID]; ok {
			continue
		}
		payer, err := s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
			ID:         payerID,
			TenantInfo: tenantInfo,
			CustomerFilterOptions: customerports.CustomerFilterOptions{
				IncludeBillingProfile: true,
			},
		})
		if err != nil {
			return err
		}
		payers[payer.ID] = payer
	}

	return nil
}

// ReadinessFrom decides a shipment's billing readiness from what was read for
// it. It reads nothing itself.
func (s *Service) ReadinessFrom(
	entity *shipment.Shipment,
	resolution *chargeallocation.ShareResolution,
	sources *Sources,
) *shipmentbillingports.ShipmentBillingReadiness {
	var profile *customer.CustomerBillingProfile
	if payer := sources.Payers[resolution.DefaultPayerID]; payer != nil {
		profile = payer.BillingProfile
	}
	docs := sources.documents[entity.ID.String()]

	readiness := buildShipmentBillingReadiness(entity, profile, sources.control, docs)
	applyPayerReadiness(readiness, entity, resolution, sources.Payers, docs)
	applyUnresolvedServiceFailures(readiness, sources.failures[entity.ID])
	s.l.Debug(
		"evaluated shipment billing readiness",
		zap.String("shipmentId", entity.ID.String()),
		zap.String("shipmentStatus", string(entity.Status)),
		zap.String(
			"shipmentBillingRequirementEnforcement",
			string(readiness.Policy.ShipmentBillingRequirementEnforcement),
		),
		zap.String("readyToBillAssignmentMode", string(readiness.Policy.ReadyToBillAssignmentMode)),
		zap.String("billingQueueTransferMode", string(readiness.Policy.BillingQueueTransferMode)),
		zap.Bool("canMarkReadyToInvoice", readiness.CanMarkReadyToInvoice),
		zap.Bool("shouldAutoMarkReadyToInvoice", readiness.ShouldAutoMarkReadyToInvoice),
		zap.Bool("shouldAutoTransferToBilling", readiness.ShouldAutoTransferToBilling),
		zap.Int("requirementCount", len(readiness.Requirements)),
		zap.Int("missingRequirementCount", len(readiness.MissingRequirements)),
		zap.Int("validationFailureCount", len(readiness.ValidationFailures)),
	)

	return readiness
}

func (s *Service) unresolvedServiceFailures(
	ctx context.Context,
	entity *shipment.Shipment,
) ([]*servicefailure.ServiceFailure, error) {
	if s.serviceFailureRepo == nil || entity == nil {
		return nil, nil
	}

	return s.serviceFailureRepo.ListUnresolvedByShipment(
		ctx,
		&servicefailureports.ServiceFailuresByShipmentRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			ShipmentID: entity.ID,
		},
	)
}

// resolvePayerShares divides the shipment among its payers, reloading it with
// charges and allocations when the caller handed over a bare header.
func (s *Service) resolvePayerShares(
	ctx context.Context,
	entity *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
	detailsLoaded bool,
) (*chargeallocation.ShareResolution, error) {
	source := entity
	if !detailsLoaded &&
		(entity.AdditionalCharges == nil || entity.ChargeAllocations == nil) &&
		s.repo != nil && entity.ID.IsNotNil() {
		full, err := s.repo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
			ID:         entity.ID,
			TenantInfo: tenantInfo,
			ShipmentOptions: shipmentports.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		})
		if err != nil {
			return nil, err
		}
		source = full
	}

	resolution, err := chargeallocation.ResolveShares(ctx, source, source.ChargeAllocations)
	if err != nil {
		return nil, err
	}

	return resolution, nil
}

func (s *Service) ReadinessSourcesFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	pending []PendingReadiness,
) (*Sources, error) {
	sources := &Sources{
		Payers:    map[pulid.ID]*customer.Customer{},
		documents: make(map[string][]*document.Document, len(pending)),
		failures:  make(map[pulid.ID][]*servicefailure.ServiceFailure, len(pending)),
	}
	if len(pending) == 0 {
		return sources, nil
	}

	payerIDs := make([]pulid.ID, 0, len(pending))
	defaultPayers := make([]pulid.ID, 0, len(pending))
	shipmentIDs := make([]pulid.ID, 0, len(pending))
	resourceIDs := make([]string, 0, len(pending))
	for _, entry := range pending {
		payerIDs = append(payerIDs, entry.Resolution.PayerIDs()...)
		defaultPayers = append(defaultPayers, entry.Resolution.DefaultPayerID)
		shipmentIDs = append(shipmentIDs, entry.Entity.ID)
		resourceIDs = append(resourceIDs, entry.Entity.ID.String())
	}

	payers, err := s.customerRepo.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: sliceutils.Dedupe(payerIDs),
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return nil, err
	}
	sources.Payers = payersByID(payers)
	if err = s.completePayers(
		ctx,
		tenantInfo,
		sources.Payers,
		sliceutils.Dedupe(defaultPayers)...,
	); err != nil {
		return nil, err
	}

	if sources.control, err = s.billingControlFor(
		ctx,
		tenantInfo.OrgID,
		new(BillingReadinessCache),
	); err != nil {
		return nil, err
	}

	docs, err := s.documentRepo.GetByResourceIDs(
		ctx,
		&documentports.GetDocumentsByResourceIDsRequest{
			TenantInfo:   tenantInfo,
			ResourceType: "shipment",
			ResourceIDs:  resourceIDs,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, doc := range docs {
		if doc != nil {
			sources.documents[doc.ResourceID] = append(sources.documents[doc.ResourceID], doc)
		}
	}

	s.readServiceFailures(ctx, tenantInfo, shipmentIDs, sources)

	return sources, nil
}

// readServiceFailures adds the unresolved service failures, which only warn;
// a failure to read them is logged and costs the warning, as it does for one
// shipment.
func (s *Service) readServiceFailures(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
	sources *Sources,
) {
	if s.serviceFailureRepo == nil {
		return
	}

	failures, err := s.serviceFailureRepo.ListUnresolvedByShipmentIDs(
		ctx,
		&servicefailureports.ServiceFailuresByShipmentIDsRequest{
			TenantInfo:  tenantInfo,
			ShipmentIDs: shipmentIDs,
		},
	)
	if err != nil {
		s.l.Warn("failed to read service failures for billing transfer plan", zap.Error(err))
		return
	}
	for _, failure := range failures {
		if failure != nil {
			sources.failures[failure.ShipmentID] = append(
				sources.failures[failure.ShipmentID],
				failure,
			)
		}
	}
}

type Params struct {
	fx.In

	Logger             *zap.Logger
	Repo               shipmentports.ShipmentRepository
	CustomerRepo       customerports.CustomerRepository
	DocumentRepo       documentports.DocumentRepository
	BillingRepo        tenantports.BillingControlRepository
	ServiceFailureRepo servicefailureports.ServiceFailureRepository `optional:"true"`
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		l:                  p.Logger.Named("service.shipment"),
		repo:               p.Repo,
		customerRepo:       p.CustomerRepo,
		documentRepo:       p.DocumentRepo,
		billingRepo:        p.BillingRepo,
		serviceFailureRepo: p.ServiceFailureRepo,
	}
}

var _ shipmentserviceports.ShipmentBillingReadinessReader = (*Service)(nil)
