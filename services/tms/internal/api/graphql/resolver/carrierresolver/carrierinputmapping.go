package carrierresolver

import (
	"fmt"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

func carrierFromInput(
	input *gqlmodel.CarrierInput,
	authCtx *authctx.AuthContext,
) (*carrier.Carrier, error) {
	entity := &carrier.Carrier{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	if err := applyCarrierInput(entity, input); err != nil {
		return nil, err
	}

	return entity, nil
}

func applyCarrierInput(entity *carrier.Carrier, input *gqlmodel.CarrierInput) error {
	stateID, err := optionalCarrierID("stateId", input.StateID)
	if err != nil {
		return err
	}
	remitStateID, err := optionalCarrierID("remitStateId", input.RemitStateID)
	if err != nil {
		return err
	}
	contacts, err := carrierContactsFromInput(input.Contacts)
	if err != nil {
		return err
	}
	policies, err := carrierInsurancePoliciesFromInput(input.InsurancePolicies)
	if err != nil {
		return err
	}

	entity.StateID = stateID
	entity.RemitStateID = remitStateID
	entity.Status = input.Status
	entity.Code = input.Code
	entity.Name = input.Name
	entity.DBAName = gqlscalar.StringValue(input.DbaName)
	entity.CarrierType = input.CarrierType
	entity.DOTNumber = gqlscalar.StringValue(input.DOTNumber)
	entity.MCNumber = gqlscalar.StringValue(input.McNumber)
	entity.SCAC = gqlscalar.StringValue(input.SCAC)
	entity.ComplianceStatus = input.ComplianceStatus
	entity.SafetyRating = input.SafetyRating
	entity.QualifiedAt = gqlscalar.Int64Ptr(input.QualifiedAt)
	entity.DisqualifiedReason = gqlscalar.StringValue(input.DisqualifiedReason)
	entity.TaxID = gqlscalar.StringValue(input.TaxID)
	entity.TaxIDType = input.TaxIDType
	entity.W9OnFile = input.W9OnFile
	entity.Is1099Eligible = input.Is1099Eligible
	entity.PaymentMethod = input.PaymentMethod
	entity.PaymentTermDays = input.PaymentTermDays
	entity.RemitToName = gqlscalar.StringValue(input.RemitToName)
	entity.RemitAddressLine1 = gqlscalar.StringValue(input.RemitAddressLine1)
	entity.RemitAddressLine2 = gqlscalar.StringValue(input.RemitAddressLine2)
	entity.RemitCity = gqlscalar.StringValue(input.RemitCity)
	entity.RemitPostalCode = gqlscalar.StringValue(input.RemitPostalCode)
	entity.AddressLine1 = gqlscalar.StringValue(input.AddressLine1)
	entity.AddressLine2 = gqlscalar.StringValue(input.AddressLine2)
	entity.City = gqlscalar.StringValue(input.City)
	entity.PostalCode = gqlscalar.StringValue(input.PostalCode)
	entity.Phone = gqlscalar.StringValue(input.Phone)
	entity.Email = gqlscalar.StringValue(input.Email)
	entity.ExternalID = gqlscalar.StringValue(input.ExternalID)
	entity.Notes = gqlscalar.StringValue(input.Notes)
	entity.Contacts = contacts
	entity.InsurancePolicies = policies
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return nil
}

func optionalCarrierID(field string, raw *string) (*pulid.ID, error) {
	id, err := gqlid.OptionalScopedID(field, raw)
	if err != nil {
		return nil, err
	}

	return gqlid.IDValuePtr(id), nil
}

func carrierContactsFromInput(
	input []*gqlmodel.CarrierContactInput,
) ([]*carrier.CarrierContact, error) {
	contacts := make([]*carrier.CarrierContact, 0, len(input))
	for i, item := range input {
		if item == nil {
			continue
		}
		id, err := gqlid.OptionalScopedID(fmt.Sprintf("contacts[%d].id", i), item.ID)
		if err != nil {
			return nil, err
		}

		contacts = append(contacts, &carrier.CarrierContact{
			ID:                        id,
			Name:                      item.Name,
			Title:                     gqlscalar.StringValue(item.Title),
			Email:                     gqlscalar.StringValue(item.Email),
			Phone:                     gqlscalar.StringValue(item.Phone),
			IsPrimary:                 item.IsPrimary,
			ReceivesRateConfirmations: item.ReceivesRateConfirmations,
			Version:                   gqlscalar.Int64Value(item.Version),
		})
	}

	return contacts, nil
}

func carrierInsurancePoliciesFromInput(
	input []*gqlmodel.CarrierInsurancePolicyInput,
) ([]*carrier.CarrierInsurancePolicy, error) {
	policies := make([]*carrier.CarrierInsurancePolicy, 0, len(input))
	for i, item := range input {
		if item == nil {
			continue
		}
		id, err := gqlid.OptionalScopedID(fmt.Sprintf("insurancePolicies[%d].id", i), item.ID)
		if err != nil {
			return nil, err
		}
		coverage, err := gqldecimal.ParseDecimalField(
			fmt.Sprintf("insurancePolicies[%d].coverageAmount", i),
			item.CoverageAmount,
			true,
		)
		if err != nil {
			return nil, err
		}

		policies = append(policies, &carrier.CarrierInsurancePolicy{
			ID:             id,
			PolicyType:     item.PolicyType,
			PolicyNumber:   item.PolicyNumber,
			ProviderName:   item.ProviderName,
			CoverageAmount: coverage,
			EffectiveDate:  int64(item.EffectiveDate),
			ExpirationDate: int64(item.ExpirationDate),
			IsVerified:     item.IsVerified,
			Version:        gqlscalar.Int64Value(item.Version),
		})
	}

	return policies, nil
}

func applyCarrierPatch(entity *carrier.Carrier, input *gqlmodel.CarrierPatchInput) error {
	if err := gqlscalar.ApplyRequired("status", "Status", input.Status, &entity.Status); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired("name", "Name", input.Name, &entity.Name); err != nil {
		return err
	}
	gqlscalar.ApplyClearable(input.DbaName, &entity.DBAName)
	if err := gqlscalar.ApplyRequired(
		"carrierType",
		"Carrier type",
		input.CarrierType,
		&entity.CarrierType,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"complianceStatus",
		"Compliance status",
		input.ComplianceStatus,
		&entity.ComplianceStatus,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"safetyRating",
		"Safety rating",
		input.SafetyRating,
		&entity.SafetyRating,
	); err != nil {
		return err
	}
	gqlscalar.ApplyClearable(input.DisqualifiedReason, &entity.DisqualifiedReason)
	if err := gqlscalar.ApplyRequired(
		"paymentMethod",
		"Payment method",
		input.PaymentMethod,
		&entity.PaymentMethod,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"paymentTermDays",
		"Payment term days",
		input.PaymentTermDays,
		&entity.PaymentTermDays,
	); err != nil {
		return err
	}
	gqlscalar.ApplyClearable(input.Phone, &entity.Phone)
	gqlscalar.ApplyClearable(input.Email, &entity.Email)
	gqlscalar.ApplyClearable(input.ExternalID, &entity.ExternalID)
	gqlscalar.ApplyClearable(input.Notes, &entity.Notes)

	return gqlscalar.ApplyRequiredInt("version", "Version", input.Version, &entity.Version)
}
