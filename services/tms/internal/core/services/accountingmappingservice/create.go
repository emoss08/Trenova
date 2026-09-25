package accountingmappingservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	requestIDPrefix    = "trn-"
	requestIDHashChars = 40
)

func creationRequestID(row *accountingsync.AccountingMapping) string {
	sum := sha256.Sum256([]byte(
		row.ConnectionID.String() + "|" + row.ID.String() + "|" + strconv.FormatInt(
			row.Version,
			10,
		),
	))
	return requestIDPrefix + hex.EncodeToString(sum[:])[:requestIDHashChars]
}

func (s *Service) CreateReferenceRecord(
	ctx context.Context,
	req *services.CreateAccountingReferenceRecordRequest,
) (*accountingsync.AccountingMapping, error) {
	row, err := s.GetMapping(ctx, req.TenantInfo, req.MappingID)
	if err != nil {
		return nil, err
	}
	if !row.ProviderKind.Creatable() {
		return nil, errortypes.NewBusinessError(
			"{0} records are kept by the bookkeeper and are not created from Trenova",
			string(row.ProviderKind),
		)
	}
	if row.State == accountingsync.MappingStateConfirmed {
		return nil, errortypes.NewBusinessError(
			"{0} is already mapped to {1}; clear it first to create a new record",
			row.TargetLabel,
			row.ExternalName,
		)
	}

	session, err := s.connectionService.Session(ctx, req.TenantInfo, row.ConnectionID)
	if err != nil {
		return nil, err
	}
	creator, ok := session.Connector.(services.AccountingReferenceCreator)
	if !ok {
		return nil, errortypes.NewBusinessError(
			"This accounting system cannot create records from Trenova",
		)
	}

	createReq := &services.AccountingCreateReferenceRequest{
		RealmID:     session.Connection.ExternalRealmID,
		AccessToken: session.AccessToken,
		RequestID:   creationRequestID(row),
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
		s.connectionService.ReportCallFailure(ctx, req.TenantInfo, row.ConnectionID, err)
		return nil, errortypes.NewBusinessError(
			"{0} did not create the record: {1}",
			accountingsync.ProviderName(session.Connection.IntegrationType),
			stringutils.OneLine(err.Error(), maxReasonRunes),
		).WithInternal(err)
	}

	if err = s.references.Upsert(ctx, &repositories.UpsertAccountingReferenceObjectsRequest{
		TenantInfo:   req.TenantInfo,
		ConnectionID: row.ConnectionID,
		Objects:      []*accountingsync.AccountingReferenceObject{created},
		SeenAt:       timeutils.NowUnix(),
	}); err != nil {
		return nil, err
	}

	source := accountingsync.MappingSourceCreatedInProvider
	if req.Source == accountingsync.MappingSourceAgent {
		source = accountingsync.MappingSourceAgent
	}
	before := jsonutils.MustToJSON(row)
	row.Confirm(&accountingsync.Choice{
		ExternalID:   created.ExternalID,
		ExternalName: created.Label(),
		Source:       source,
		Reason: "Created in " + accountingsync.ProviderName(
			session.Connection.IntegrationType,
		),
		ActorID: req.UserID,
		At:      timeutils.NowUnix(),
	})
	updated, err := s.mappings.Update(ctx, row)
	if err != nil {
		return nil, err
	}

	s.logAudit(
		updated,
		req.UserID,
		before,
		"Created "+created.Label()+" and mapped "+updated.TargetLabel+" to it",
	)
	s.publishInvalidation(
		ctx,
		req.TenantInfo,
		req.UserID,
		[]*accountingsync.AccountingMapping{updated},
	)
	return updated, nil
}

const maxReasonRunes = 300

func (s *Service) buildDraft(
	ctx context.Context,
	req *services.CreateAccountingReferenceRecordRequest,
	row *accountingsync.AccountingMapping,
	creator services.AccountingReferenceCreator,
	createReq *services.AccountingCreateReferenceRequest,
) (string, error) {
	switch row.ProviderKind {
	case accountingsync.ReferenceKindItem:
		return s.itemDraft(ctx, req, row, creator, createReq)
	case accountingsync.ReferenceKindCustomer:
		return s.customerDraft(ctx, req, row, creator, createReq)
	case accountingsync.ReferenceKindVendor:
		return s.vendorDraft(ctx, req, row, creator, createReq)
	case accountingsync.ReferenceKindAccount,
		accountingsync.ReferenceKindTerm,
		accountingsync.ReferenceKindPaymentMethod:
		return "", errortypes.NewBusinessError("This kind of record is not created from Trenova")
	default:
		return "", errortypes.NewBusinessError("This kind of record is not created from Trenova")
	}
}

func chosenName(
	creator services.AccountingReferenceCreator,
	kind accountingsync.ReferenceKind,
	override string,
	fallbacks ...string,
) (string, error) {
	for _, candidate := range append([]string{override}, fallbacks...) {
		if name := creator.SanitizeName(kind, candidate); name != "" {
			return name, nil
		}
	}
	return "", errortypes.NewValidationError("name", errortypes.ErrRequired, "A name is required")
}

func (s *Service) itemDraft(
	ctx context.Context,
	req *services.CreateAccountingReferenceRecordRequest,
	row *accountingsync.AccountingMapping,
	creator services.AccountingReferenceCreator,
	createReq *services.AccountingCreateReferenceRequest,
) (string, error) {
	revenue, err := s.mappings.GetByTarget(ctx, &repositories.GetAccountingMappingByTargetRequest{
		TenantInfo:   req.TenantInfo,
		ConnectionID: row.ConnectionID,
		TargetType:   accountingsync.TargetAccountRole,
		TrenovaKey:   accountingsync.AccountRoleRevenue,
	})
	if err != nil && !errortypes.IsNotFoundError(err) {
		return "", err
	}
	if revenue == nil || revenue.State != accountingsync.MappingStateConfirmed {
		return "", errortypes.NewBusinessError(
			"Confirm the revenue account first: a new item needs an income account",
		)
	}

	description, sku := row.TargetLabel, ""
	if row.TargetType == accountingsync.TargetAccessorialCharge {
		charge, getErr := s.accessorials.GetByID(ctx, repositories.GetAccessorialChargeByIDRequest{
			ID:         row.TrenovaObjectID,
			TenantInfo: &req.TenantInfo,
		})
		if getErr != nil {
			return "", getErr
		}
		description, sku = charge.Description, charge.Code
	}

	name, err := chosenName(
		creator,
		accountingsync.ReferenceKindItem,
		req.Name,
		description,
		row.TargetLabel,
		sku,
	)
	if err != nil {
		return "", err
	}
	createReq.Item = &services.AccountingItemDraft{
		Name:            name,
		Description:     description,
		Sku:             sku,
		IncomeAccountID: revenue.ExternalID,
	}
	return name, nil
}

func (s *Service) customerDraft(
	ctx context.Context,
	req *services.CreateAccountingReferenceRecordRequest,
	row *accountingsync.AccountingMapping,
	creator services.AccountingReferenceCreator,
	createReq *services.AccountingCreateReferenceRequest,
) (string, error) {
	found, err := s.customers.GetByIDs(ctx, repositories.GetCustomersByIDsRequest{
		TenantInfo:            req.TenantInfo,
		CustomerIDs:           []pulid.ID{row.TrenovaObjectID},
		CustomerFilterOptions: repositories.CustomerFilterOptions{IncludeState: true},
	})
	if err != nil {
		return "", err
	}
	if len(found) == 0 {
		return "", errortypes.NewNotFoundError("The customer for this mapping no longer exists")
	}
	cus := found[0]

	name, err := chosenName(creator, accountingsync.ReferenceKindCustomer, req.Name, cus.Name)
	if err != nil {
		return "", err
	}
	createReq.Party = &services.AccountingPartyDraft{
		DisplayName:  name,
		CompanyName:  cus.Name,
		AddressLine1: cus.AddressLine1,
		City:         cus.City,
		State:        stateCode(cus.State),
		PostalCode:   cus.PostalCode,
		Country:      stateCountry(cus.State),
	}
	return name, nil
}

func (s *Service) vendorDraft(
	ctx context.Context,
	req *services.CreateAccountingReferenceRecordRequest,
	row *accountingsync.AccountingMapping,
	creator services.AccountingReferenceCreator,
	createReq *services.AccountingCreateReferenceRequest,
) (string, error) {
	found, err := s.carriers.GetByIDs(ctx, repositories.GetCarriersByIDsRequest{
		TenantInfo:           req.TenantInfo,
		CarrierIDs:           []pulid.ID{row.TrenovaObjectID},
		CarrierFilterOptions: repositories.CarrierFilterOptions{IncludeState: true},
	})
	if err != nil {
		return "", err
	}
	if len(found) == 0 {
		return "", errortypes.NewNotFoundError("The carrier for this mapping no longer exists")
	}
	carr := found[0]

	name, err := chosenName(
		creator,
		accountingsync.ReferenceKindVendor,
		req.Name,
		carr.Name,
		carr.DBAName,
	)
	if err != nil {
		return "", err
	}
	draft := &services.AccountingPartyDraft{
		DisplayName: name,
		CompanyName: carr.Name,
		Email:       strings.TrimSpace(carr.Email),
		Is1099:      carr.Is1099Eligible,
	}
	applyCarrierAddress(draft, carr)
	createReq.Party = draft
	return name, nil
}

func applyCarrierAddress(draft *services.AccountingPartyDraft, carr *carrier.Carrier) {
	if strings.TrimSpace(carr.RemitAddressLine1) != "" {
		draft.AddressLine1 = carr.RemitAddressLine1
		draft.City = carr.RemitCity
		draft.PostalCode = carr.RemitPostalCode
		draft.State = stateCode(carr.RemitState)
		draft.Country = stateCountry(carr.RemitState)
		return
	}
	draft.AddressLine1 = carr.AddressLine1
	draft.City = carr.City
	draft.PostalCode = carr.PostalCode
	draft.State = stateCode(carr.State)
	draft.Country = stateCountry(carr.State)
}

func stateCode(state *usstate.UsState) string {
	if state == nil {
		return ""
	}
	return state.Abbreviation
}

func stateCountry(state *usstate.UsState) string {
	if state == nil {
		return ""
	}
	return state.CountryIso3
}

func (s *Service) duplicateNameError(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	row *accountingsync.AccountingMapping,
	name string,
	cause error,
) error {
	existing, err := s.references.Search(ctx, &repositories.SearchAccountingReferenceObjectsRequest{
		TenantInfo:   tenantInfo,
		ConnectionID: row.ConnectionID,
		Kind:         row.ProviderKind,
		Query:        name,
		Limit:        5,
	})
	if err == nil {
		for _, ref := range existing {
			if stringutils.NormalizeName(ref.Name) == stringutils.NormalizeName(name) {
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
