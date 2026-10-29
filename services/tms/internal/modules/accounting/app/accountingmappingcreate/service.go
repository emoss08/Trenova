package accountingmappingcreate

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingkit"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingreads"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingtargets"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/textsim"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	References   accountingmappingports.AccountingReferenceObjectRepository
	Mappings     accountingmappingports.AccountingMappingRepository
	Sessions     accountingconnectionports.AccountingSessionProvider
	Accessorials accessorialchargeports.AccessorialChargeRepository
	Customers    customerports.CustomerRepository
	Carriers     carrierports.CarrierRepository
	Kit          *accountingmappingkit.Kit
	Catalog      *accountingmappingtargets.Catalog
	Reads        *accountingmappingreads.Service
}

type Service struct {
	l            *zap.Logger
	references   accountingmappingports.AccountingReferenceObjectRepository
	mappings     accountingmappingports.AccountingMappingRepository
	sessions     accountingconnectionports.AccountingSessionProvider
	accessorials accessorialchargeports.AccessorialChargeRepository
	customers    customerports.CustomerRepository
	carriers     carrierports.CarrierRepository
	kit          *accountingmappingkit.Kit
	catalog      *accountingmappingtargets.Catalog
	reads        *accountingmappingreads.Service
}

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.accounting-mapping"),
		references:   p.References,
		mappings:     p.Mappings,
		sessions:     p.Sessions,
		accessorials: p.Accessorials,
		customers:    p.Customers,
		carriers:     p.Carriers,
		kit:          p.Kit,
		catalog:      p.Catalog,
		reads:        p.Reads,
	}
}

var _ accountingmappingports.AccountingReferenceCreator = (*Service)(nil)

func (s *Service) CreateReferenceRecord(
	ctx context.Context,
	req *accountingmappingports.CreateAccountingReferenceRecordRequest,
) (*accountingmapping.AccountingMapping, error) {
	row, err := s.reads.GetMapping(ctx, req.TenantInfo, req.MappingID)
	if err != nil {
		return nil, err
	}
	if !row.ProviderKind.Creatable() {
		return nil, errortypes.NewBusinessError(
			"{0} records are kept by the bookkeeper and are not created from Trenova",
			string(row.ProviderKind),
		)
	}
	if row.State == accountingmapping.MappingStateConfirmed {
		return nil, errortypes.NewBusinessError(
			"{0} is already mapped to {1}; clear it first to create a new record",
			row.TargetLabel,
			row.ExternalName,
		)
	}

	session, err := s.sessions.Session(ctx, req.TenantInfo, row.ConnectionID)
	if err != nil {
		return nil, err
	}
	creator, ok := session.Connector.(accountingconnectorports.AccountingReferenceCreator)
	if !ok {
		return nil, errortypes.NewBusinessError(
			"This accounting system cannot create records from Trenova",
		)
	}

	createReq := &accountingconnectorports.AccountingCreateReferenceRequest{
		RealmID:     session.Connection.ExternalRealmID,
		AccessToken: session.AccessToken,
		RequestID:   CreationRequestID(row),
		Kind:        row.ProviderKind,
	}
	name, err := s.buildDraft(ctx, req, row, creator, createReq)
	if err != nil {
		return nil, err
	}

	created, err := creator.CreateReference(ctx, createReq)
	if err != nil {
		if creator.IsDuplicateName(err) {
			return nil, s.duplicateNameError(ctx, req.TenantInfo, row, name, err)
		}
		s.sessions.ReportCallFailure(ctx, req.TenantInfo, row.ConnectionID, err)
		return nil, errortypes.NewBusinessError(
			"{0} did not create the record: {1}",
			accountingprovider.ProviderName(session.Connection.IntegrationType),
			stringutils.OneLine(err.Error(), maxReasonRunes),
		).WithInternal(err)
	}

	if err = s.references.Upsert(
		ctx,
		&accountingmappingports.UpsertAccountingReferenceObjectsRequest{
			TenantInfo:   req.TenantInfo,
			ConnectionID: row.ConnectionID,
			Objects:      []*accountingmapping.AccountingReferenceObject{created},
			SeenAt:       timeutils.NowUnix(),
		},
	); err != nil {
		return nil, err
	}

	before := jsonutils.MustToJSON(row)
	updated, err := s.confirmCreated(
		ctx,
		req.TenantInfo,
		row,
		accountingmappingkit.CreatedReferenceChoice(
			&accountingmappingkit.CreatedReference{
				ExternalID:      created.ExternalID,
				ExternalName:    created.Label(),
				RequestedSource: req.Source,
				IntegrationType: session.Connection.IntegrationType,
				ActorID:         req.UserID,
				At:              timeutils.NowUnix(),
			},
		),
	)
	if err != nil {
		return nil, err
	}

	s.kit.LogAudit(
		updated,
		req.UserID,
		before,
		"Created "+created.Label()+" and mapped "+updated.TargetLabel+" to it",
	)
	s.kit.PublishInvalidation(
		ctx,
		req.TenantInfo,
		req.UserID,
		[]*accountingmapping.AccountingMapping{updated},
	)
	s.kit.RequeueMappingBlocked(ctx, req.TenantInfo, updated.ConnectionID)
	return updated, nil
}

func (s *Service) confirmCreated(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	row *accountingmapping.AccountingMapping,
	choice *accountingmapping.Choice,
) (*accountingmapping.AccountingMapping, error) {
	row.Confirm(choice)
	updated, err := s.mappings.Update(ctx, row)
	if err == nil || !errortypes.IsVersionMismatchError(err) {
		return updated, err
	}

	current, getErr := s.reads.GetMapping(ctx, tenantInfo, row.ID)
	if getErr != nil {
		return nil, getErr
	}
	if current.State == accountingmapping.MappingStateConfirmed {
		return nil, errortypes.NewBusinessError(
			"{0} was created, but {1} was mapped to {2} meanwhile; map it again if that is wrong",
			choice.ExternalName,
			current.TargetLabel,
			current.ExternalName,
		).WithInternal(err)
	}
	current.Confirm(choice)
	return s.mappings.Update(ctx, current)
}

func (s *Service) buildDraft(
	ctx context.Context,
	req *accountingmappingports.CreateAccountingReferenceRecordRequest,
	row *accountingmapping.AccountingMapping,
	creator accountingconnectorports.AccountingReferenceCreator,
	createReq *accountingconnectorports.AccountingCreateReferenceRequest,
) (string, error) {
	switch row.ProviderKind {
	case accountingprovider.ReferenceKindItem:
		return s.itemDraft(ctx, req, row, creator, createReq)
	case accountingprovider.ReferenceKindCustomer:
		return s.customerDraft(ctx, req, row, creator, createReq)
	case accountingprovider.ReferenceKindVendor:
		return s.vendorDraft(ctx, req, row, creator, createReq)
	case accountingprovider.ReferenceKindAccount,
		accountingprovider.ReferenceKindTerm,
		accountingprovider.ReferenceKindPaymentMethod:
		return "", errortypes.NewBusinessError("This kind of record is not created from Trenova")
	default:
		return "", errortypes.NewBusinessError("This kind of record is not created from Trenova")
	}
}

func (s *Service) itemDraft(
	ctx context.Context,
	req *accountingmappingports.CreateAccountingReferenceRecordRequest,
	row *accountingmapping.AccountingMapping,
	creator accountingconnectorports.AccountingReferenceCreator,
	createReq *accountingconnectorports.AccountingCreateReferenceRequest,
) (string, error) {
	revenue, err := s.mappings.GetByTarget(
		ctx,
		&accountingmappingports.GetAccountingMappingByTargetRequest{
			TenantInfo:   req.TenantInfo,
			ConnectionID: row.ConnectionID,
			TargetType:   accountingsync.TargetAccountRole,
			TrenovaKey:   accountingsync.AccountRoleRevenue,
		},
	)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return "", err
	}
	if revenue == nil || revenue.State != accountingmapping.MappingStateConfirmed {
		return "", errortypes.NewBusinessError(
			"Confirm the revenue account first: a new item needs an income account",
		)
	}

	description, sku := row.TargetLabel, ""
	if row.TargetType == accountingsync.TargetAccessorialCharge {
		charge, getErr := s.accessorials.GetByID(
			ctx,
			accessorialchargeports.GetAccessorialChargeByIDRequest{
				ID:         row.TrenovaObjectID,
				TenantInfo: &req.TenantInfo,
			},
		)
		if getErr != nil {
			return "", getErr
		}
		description, sku = charge.Description, charge.Code
	}

	name, err := chosenName(
		creator,
		accountingprovider.ReferenceKindItem,
		req.Name,
		description,
		row.TargetLabel,
		sku,
	)
	if err != nil {
		return "", err
	}
	createReq.Item = &accountingconnectorports.AccountingItemDraft{
		Name:            name,
		Description:     description,
		Sku:             sku,
		IncomeAccountID: revenue.ExternalID,
	}
	return name, nil
}

func (s *Service) customerDraft(
	ctx context.Context,
	req *accountingmappingports.CreateAccountingReferenceRecordRequest,
	row *accountingmapping.AccountingMapping,
	creator accountingconnectorports.AccountingReferenceCreator,
	createReq *accountingconnectorports.AccountingCreateReferenceRequest,
) (string, error) {
	cus, err := s.customerWithState(ctx, req.TenantInfo, row.TrenovaObjectID)
	if err != nil {
		return "", err
	}

	name, err := chosenName(creator, accountingprovider.ReferenceKindCustomer, req.Name, cus.Name)
	if err != nil {
		return "", err
	}
	createReq.Party = customerParty(cus, name)
	return name, nil
}

func (s *Service) customerWithState(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) (*customer.Customer, error) {
	found, err := s.customers.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
		TenantInfo:            tenantInfo,
		CustomerIDs:           []pulid.ID{customerID},
		CustomerFilterOptions: customerports.CustomerFilterOptions{IncludeState: true},
	})
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, errortypes.NewNotFoundError("The customer for this mapping no longer exists")
	}
	return found[0], nil
}

func (s *Service) CustomerParty(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) (*accountingconnectorports.AccountingPartyDraft, error) {
	cus, err := s.customerWithState(ctx, tenantInfo, customerID)
	if err != nil {
		return nil, err
	}
	return customerParty(cus, cus.Name), nil
}

func (s *Service) vendorDraft(
	ctx context.Context,
	req *accountingmappingports.CreateAccountingReferenceRecordRequest,
	row *accountingmapping.AccountingMapping,
	creator accountingconnectorports.AccountingReferenceCreator,
	createReq *accountingconnectorports.AccountingCreateReferenceRequest,
) (string, error) {
	if row.TargetType == accountingsync.TargetDriver {
		return s.driverDraft(ctx, req, row, creator, createReq)
	}
	carr, err := s.carrierWithState(ctx, req.TenantInfo, row.TrenovaObjectID)
	if err != nil {
		return "", err
	}

	name, err := chosenName(
		creator,
		accountingprovider.ReferenceKindVendor,
		req.Name,
		carr.Name,
		carr.DBAName,
	)
	if err != nil {
		return "", err
	}
	createReq.Party = carrierParty(carr, name)
	return name, nil
}

func (s *Service) carrierWithState(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
) (*carrier.Carrier, error) {
	found, err := s.carriers.GetByIDs(ctx, carrierports.GetCarriersByIDsRequest{
		TenantInfo:           tenantInfo,
		CarrierIDs:           []pulid.ID{carrierID},
		CarrierFilterOptions: carrierports.CarrierFilterOptions{IncludeState: true},
	})
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, errortypes.NewNotFoundError("The carrier for this mapping no longer exists")
	}
	return found[0], nil
}

func (s *Service) driverDraft(
	ctx context.Context,
	req *accountingmappingports.CreateAccountingReferenceRecordRequest,
	row *accountingmapping.AccountingMapping,
	creator accountingconnectorports.AccountingReferenceCreator,
	createReq *accountingconnectorports.AccountingCreateReferenceRequest,
) (string, error) {
	wrk, err := s.catalog.WorkerWithState(ctx, req.TenantInfo, row.TrenovaObjectID)
	if err != nil {
		return "", err
	}
	name, err := chosenName(
		creator,
		accountingprovider.ReferenceKindVendor,
		req.Name,
		accountingmappingtargets.WorkerName(wrk),
	)
	if err != nil {
		return "", err
	}
	createReq.Party = driverParty(wrk, name)
	return name, nil
}

func (s *Service) VendorParty(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	targetType accountingsync.MappingTargetType,
	objectID pulid.ID,
) (*accountingconnectorports.AccountingPartyDraft, error) {
	switch targetType {
	case accountingsync.TargetCarrier:
		carr, err := s.carrierWithState(ctx, tenantInfo, objectID)
		if err != nil {
			return nil, err
		}
		return carrierParty(carr, carr.Name), nil
	case accountingsync.TargetDriver:
		wrk, err := s.catalog.WorkerWithState(ctx, tenantInfo, objectID)
		if err != nil {
			return nil, err
		}
		return driverParty(wrk, accountingmappingtargets.WorkerName(wrk)), nil
	case accountingsync.TargetAccountRole,
		accountingsync.TargetLineType,
		accountingsync.TargetAccessorialCharge,
		accountingsync.TargetItemRole,
		accountingsync.TargetCustomer,
		accountingsync.TargetGLAccount,
		accountingsync.TargetPaymentTerm,
		accountingsync.TargetPaymentMethod:
		return nil, notVendorError(targetType)
	default:
		return nil, notVendorError(targetType)
	}
}

func (s *Service) duplicateNameError(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	row *accountingmapping.AccountingMapping,
	name string,
	cause error,
) error {
	existing, err := s.references.Search(
		ctx,
		&accountingmappingports.SearchAccountingReferenceObjectsRequest{
			TenantInfo:   tenantInfo,
			ConnectionID: row.ConnectionID,
			Kind:         row.ProviderKind,
			Query:        name,
			Limit:        5,
		},
	)
	if err == nil {
		for _, ref := range existing {
			if textsim.NormalizeName(ref.Name) == textsim.NormalizeName(name) {
				return errortypes.NewBusinessError(
					"{0} already exists in the accounting system; map {1} to it instead",
					ref.Label(),
					row.TargetLabel,
				).WithInternal(cause)
			}
		}
	}
	return errortypes.NewBusinessError(
		"A record named {0} already exists in the accounting system. Refresh the reference data and map to it, or choose another name.",
		name,
	).WithInternal(cause)
}

func notVendorError(targetType accountingsync.MappingTargetType) error {
	return errortypes.NewBusinessError("{0} is not a vendor", string(targetType))
}
