package customerresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/trackingprofile"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/intutils"
)

func customerFromInput(
	input *gqlmodel.CustomerInput,
	authCtx *authctx.AuthContext,
) (*customer.Customer, error) {
	entity := &customer.Customer{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	if err := applyCustomerInput(entity, input); err != nil {
		return nil, err
	}

	return entity, nil
}

func applyCustomerInput(entity *customer.Customer, input *gqlmodel.CustomerInput) error {
	stateID, err := gqlid.RequiredID("stateId", input.StateID)
	if err != nil {
		return err
	}

	entity.Status = input.Status
	entity.Code = input.Code
	entity.Name = input.Name
	entity.AddressLine1 = input.AddressLine1
	entity.AddressLine2 = gqlscalar.StringValue(input.AddressLine2)
	entity.City = input.City
	entity.StateID = stateID
	entity.PostalCode = input.PostalCode
	entity.Longitude = input.Longitude
	entity.Latitude = input.Latitude
	entity.PlaceID = gqlscalar.StringValue(input.PlaceID)
	entity.ExternalID = gqlscalar.StringValue(input.ExternalID)
	entity.DOTNumber = gqlscalar.StringValue(input.DOTNumber)
	entity.MCNumber = gqlscalar.StringValue(input.McNumber)
	entity.BrokerVettingEnabled = input.BrokerVettingEnabled
	entity.CheckCallIntervalHours = int16Ptr(input.CheckCallIntervalHours)
	entity.EtaRiskThresholdMinutes = int16Ptr(input.EtaRiskThresholdMinutes)
	entity.AllowConsolidation = input.AllowConsolidation
	entity.ExclusiveConsolidation = input.ExclusiveConsolidation
	entity.ConsolidationPriority = input.ConsolidationPriority
	entity.StatusUpdatePreference = input.StatusUpdatePreference
	entity.StatusUpdateRecipients = gqlscalar.StringValue(input.StatusUpdateRecipients)
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	if input.BillingProfile != nil {
		if entity.BillingProfile == nil {
			entity.BillingProfile = &customer.CustomerBillingProfile{}
		}
		if err = applyBillingProfileInput(entity.BillingProfile, input.BillingProfile); err != nil {
			return err
		}
	}
	if input.EmailProfile != nil {
		if entity.EmailProfile == nil {
			entity.EmailProfile = &customer.CustomerEmailProfile{}
		}
		applyEmailProfileInput(entity.EmailProfile, input.EmailProfile)
	}
	if input.TrackingProfile != nil {
		if entity.TrackingProfile == nil {
			entity.TrackingProfile = &trackingprofile.Profile{}
		}
		applyTrackingProfileInput(entity.TrackingProfile, input.TrackingProfile)
	}

	return nil
}

func applyBillingProfileInput(
	profile *customer.CustomerBillingProfile,
	input *gqlmodel.CustomerBillingProfileInput,
) error {
	if err := applyBillingProfileAmounts(profile, input); err != nil {
		return err
	}
	if err := applyBillingProfileReferences(profile, input); err != nil {
		return err
	}

	profile.InvoiceDelivery = input.InvoiceDelivery
	profile.BillingCycle = input.BillingCycle
	profile.BillingCycleAnchorDay = intutils.SafeToInt16(input.BillingCycleAnchorDay)
	profile.BillingCycleTimezone = input.BillingCycleTimezone
	profile.SplitBy = input.SplitBy
	profile.SectionBy = input.SectionBy
	profile.InvoiceDetail = input.InvoiceDetail
	profile.MaxShipmentsPerInvoice = intutils.SafeToInt16(input.MaxShipmentsPerInvoice)
	profile.PaymentTerm = input.PaymentTerm
	profile.HasBillingControlOverrides = input.HasBillingControlOverrides
	profile.CreditStatus = input.CreditStatus
	profile.EnforceCreditLimit = input.EnforceCreditLimit
	profile.AutoCreditHold = input.AutoCreditHold
	profile.CreditHoldReason = gqlscalar.StringValue(input.CreditHoldReason)
	profile.AutoSendInvoiceOnGeneration = input.AutoSendInvoiceOnGeneration
	profile.SendPaymentReminders = input.SendPaymentReminders
	profile.SendStatements = input.SendStatements
	profile.EmailInvoiceEnabled = input.EmailInvoiceEnabled
	profile.EDIInvoiceEnabled = input.EDIInvoiceEnabled
	profile.InvoiceNumberFormat = input.InvoiceNumberFormat
	profile.CustomerInvoicePrefix = gqlscalar.StringValue(input.CustomerInvoicePrefix)
	profile.InvoiceCopies = intutils.SafeToInt16(input.InvoiceCopies)
	profile.ApplyLateCharges = input.ApplyLateCharges
	profile.GracePeriodDays = intutils.SafeToInt16(input.GracePeriodDays)
	profile.TaxExempt = input.TaxExempt
	profile.TaxExemptNumber = gqlscalar.StringValue(input.TaxExemptNumber)
	profile.EnforceCustomerBillingReq = input.EnforceCustomerBillingReq
	profile.ValidateCustomerRates = input.ValidateCustomerRates
	profile.AutoTransfer = input.AutoTransfer
	profile.AutoMarkReadyToBill = input.AutoMarkReadyToBill
	profile.AutoApprove = input.AutoApprove
	profile.AutoBill = input.AutoBill
	profile.CountLateOnlyOnAppointmentStops = input.CountLateOnlyOnAppointmentStops
	profile.AutoApplyAccessorials = input.AutoApplyAccessorials
	profile.BillingCurrency = input.BillingCurrency
	profile.RequirePONumber = input.RequirePONumber
	profile.RequireBOLNumber = input.RequireBOLNumber
	profile.RequireDeliveryNumber = input.RequireDeliveryNumber
	profile.BillingNotes = gqlscalar.StringValue(input.BillingNotes)
	profile.FuelSurchargeMode = input.FuelSurchargeMode

	return nil
}

func applyBillingProfileAmounts(
	profile *customer.CustomerBillingProfile,
	input *gqlmodel.CustomerBillingProfileInput,
) error {
	creditLimit, err := gqldecimal.NullDecimalFromStringPtr(
		input.CreditLimit,
		"billingProfile.creditLimit",
	)
	if err != nil {
		return err
	}
	minConsolidated, err := gqldecimal.NullDecimalFromStringPtr(
		input.MinConsolidatedAmount,
		"billingProfile.minConsolidatedAmount",
	)
	if err != nil {
		return err
	}
	lateChargeRate, err := gqldecimal.NullDecimalFromStringPtr(
		input.LateChargeRate,
		"billingProfile.lateChargeRate",
	)
	if err != nil {
		return err
	}

	profile.CreditLimit = creditLimit
	profile.MinConsolidatedAmount = minConsolidated
	profile.LateChargeRate = lateChargeRate

	return nil
}

func applyBillingProfileReferences(
	profile *customer.CustomerBillingProfile,
	input *gqlmodel.CustomerBillingProfileInput,
) error {
	revenueAccountID, err := gqlid.OptionalScopedID(
		"billingProfile.revenueAccountId",
		input.RevenueAccountID,
	)
	if err != nil {
		return err
	}
	arAccountID, err := gqlid.OptionalScopedID("billingProfile.arAccountId", input.ArAccountID)
	if err != nil {
		return err
	}
	defaultBillerID, err := gqlid.OptionalScopedID(
		"billingProfile.defaultBillerId",
		input.DefaultBillerID,
	)
	if err != nil {
		return err
	}
	fuelSurchargeProgramID, err := gqlid.OptionalScopedID(
		"billingProfile.fuelSurchargeProgramId",
		input.FuelSurchargeProgramID,
	)
	if err != nil {
		return err
	}
	factoringCompanyID, err := gqlid.OptionalScopedID(
		"billingProfile.factoringCompanyId",
		input.FactoringCompanyID,
	)
	if err != nil {
		return err
	}
	documentTypeIDs, err := gqlid.ParseIDListField(
		"billingProfile.documentTypes",
		input.DocumentTypeIds,
	)
	if err != nil {
		return err
	}

	profile.RevenueAccountID = gqlid.IDValuePtr(revenueAccountID)
	profile.ARAccountID = gqlid.IDValuePtr(arAccountID)
	profile.DefaultBillerID = gqlid.IDValuePtr(defaultBillerID)
	profile.FuelSurchargeProgramID = gqlid.IDValuePtr(fuelSurchargeProgramID)
	profile.FactoringCompanyID = gqlid.IDValuePtr(factoringCompanyID)
	profile.DocumentTypes = make([]*documenttype.DocumentType, 0, len(documentTypeIDs))
	for _, id := range documentTypeIDs {
		profile.DocumentTypes = append(profile.DocumentTypes, &documenttype.DocumentType{ID: id})
	}

	return nil
}

func applyEmailProfileInput(
	profile *customer.CustomerEmailProfile,
	input *gqlmodel.CustomerEmailProfileInput,
) {
	profile.Subject = gqlscalar.StringValue(input.Subject)
	profile.Comment = gqlscalar.StringValue(input.Comment)
	profile.ToRecipients = gqlscalar.StringValue(input.ToRecipients)
	profile.CCRecipients = gqlscalar.StringValue(input.CcRecipients)
	profile.BCCRecipients = gqlscalar.StringValue(input.BCCRecipients)
	profile.AttachmentName = gqlscalar.StringValue(input.AttachmentName)
	profile.ReadReceipt = input.ReadReceipt
	profile.IncludeShipmentDetail = input.IncludeShipmentDetail
}

func applyTrackingProfileInput(
	profile *trackingprofile.Profile,
	input *gqlmodel.CustomerTrackingProfileInput,
) {
	profile.LinkEnabled = input.LinkEnabled
	profile.VisibleFields = input.VisibleFields
	profile.LinkTTLDays = clampLinkTTLDays(input.LinkTTLDays)
	profile.NotifyPickedUp = input.NotifyPickedUp
	profile.NotifyDelayed = input.NotifyDelayed
	profile.NotifyDelivered = input.NotifyDelivered
	profile.NotifyPodReady = input.NotifyPodReady
	profile.IncludeLink = input.IncludeLink
	profile.Recipients = gqlscalar.StringValue(input.Recipients)
}

func clampLinkTTLDays(days int) int16 {
	switch {
	case days < 0:
		return 0
	case days > int(trackingprofile.MaxLinkTTLDays)+1:
		return trackingprofile.MaxLinkTTLDays + 1
	default:
		return int16(days)
	}
}

func applyCustomerPatch(entity *customer.Customer, input *gqlmodel.CustomerPatchInput) error {
	if err := applyCustomerPatchRequired(entity, input); err != nil {
		return err
	}
	stateRaw, err := gqlscalar.RequiredOmittable("stateId", "State", input.StateID)
	if err != nil {
		return err
	}
	if stateRaw.Set {
		stateID, idErr := gqlid.RequiredID("stateId", stateRaw.Value)
		if idErr != nil {
			return idErr
		}
		entity.StateID = stateID
	}
	if raw, ok := input.CheckCallIntervalHours.ValueOK(); ok {
		entity.CheckCallIntervalHours = int16Ptr(raw)
	}
	if raw, ok := input.EtaRiskThresholdMinutes.ValueOK(); ok {
		entity.EtaRiskThresholdMinutes = int16Ptr(raw)
	}
	gqlscalar.ApplyClearable(input.AddressLine2, &entity.AddressLine2)
	gqlscalar.ApplyNullable(input.Longitude, &entity.Longitude)
	gqlscalar.ApplyNullable(input.Latitude, &entity.Latitude)
	gqlscalar.ApplyClearable(input.PlaceID, &entity.PlaceID)
	gqlscalar.ApplyClearable(input.ExternalID, &entity.ExternalID)
	gqlscalar.ApplyClearable(input.DOTNumber, &entity.DOTNumber)
	gqlscalar.ApplyClearable(input.McNumber, &entity.MCNumber)
	gqlscalar.ApplyClearable(input.StatusUpdateRecipients, &entity.StatusUpdateRecipients)

	return nil
}

func applyCustomerPatchRequired(
	entity *customer.Customer,
	input *gqlmodel.CustomerPatchInput,
) error {
	if err := gqlscalar.ApplyRequired(
		"status",
		"Status",
		input.Status,
		&entity.Status,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired("code", "Code", input.Code, &entity.Code); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired("name", "Name", input.Name, &entity.Name); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"addressLine1",
		"Address line 1",
		input.AddressLine1,
		&entity.AddressLine1,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired("city", "City", input.City, &entity.City); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"postalCode",
		"Postal code",
		input.PostalCode,
		&entity.PostalCode,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"brokerVettingEnabled",
		"Broker vetting",
		input.BrokerVettingEnabled,
		&entity.BrokerVettingEnabled,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"allowConsolidation",
		"Allow consolidation",
		input.AllowConsolidation,
		&entity.AllowConsolidation,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"exclusiveConsolidation",
		"Exclusive consolidation",
		input.ExclusiveConsolidation,
		&entity.ExclusiveConsolidation,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"consolidationPriority",
		"Consolidation priority",
		input.ConsolidationPriority,
		&entity.ConsolidationPriority,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"statusUpdatePreference",
		"Status update preference",
		input.StatusUpdatePreference,
		&entity.StatusUpdatePreference,
	); err != nil {
		return err
	}

	return gqlscalar.ApplyRequiredInt("version", "Version", input.Version, &entity.Version)
}

func int16Ptr(value *int) *int16 {
	if value == nil {
		return nil
	}
	converted := intutils.SafeToInt16(*value)

	return &converted
}
