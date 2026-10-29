//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package shipmentbillingreadiness

import (
	"slices"
	"sort"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

type rateValidationResult struct {
	Code    string
	Field   string
	Message string
}

// BillingReadinessCache carries what every shipment in one bulk transfer shares.
// It is scoped to a single call and never outlives it, so it cannot go stale
// against a policy change the way a process-wide cache could.
type BillingReadinessCache struct {
	billingControl      *billingcontrol.BillingControl
	billingControlKnown bool
}

// Sources is what a readiness evaluation reads beyond the shipment
// itself. One shipment reads them for itself; a plan over many reads each
// once for all of them, and both decide from the same function.
type Sources struct {
	control   *billingcontrol.BillingControl
	Payers    map[pulid.ID]*customer.Customer
	documents map[string][]*document.Document
	failures  map[pulid.ID][]*servicefailure.ServiceFailure
}

func payersByID(payers []*customer.Customer) map[pulid.ID]*customer.Customer {
	byID := make(map[pulid.ID]*customer.Customer, len(payers))
	for _, payer := range payers {
		if payer != nil {
			byID[payer.ID] = payer
		}
	}

	return byID
}

func applyUnresolvedServiceFailures(
	readiness *shipmentbillingports.ShipmentBillingReadiness,
	unresolved []*servicefailure.ServiceFailure,
) {
	if readiness == nil || len(unresolved) == 0 {
		return
	}

	ids := make([]string, 0, len(unresolved))
	for _, failure := range unresolved {
		if failure == nil {
			continue
		}
		ids = append(ids, failure.ID.String())
	}
	if len(ids) == 0 {
		return
	}
	readiness.ServiceFailureContext.HasUnresolved = true
	readiness.ServiceFailureContext.UnresolvedCount = len(ids)
	readiness.ServiceFailureContext.ServiceFailureIDs = ids
	readiness.Warnings = append(readiness.Warnings, shipmentbillingports.ShipmentBillingWarning{
		Code:    "unresolved_service_failures",
		Message: "Shipment has unresolved service failures",
		Context: map[string]any{
			"serviceFailureIds": ids,
			"unresolvedCount":   len(ids),
		},
	})
}

// applyPayerReadiness folds every other payer into the readiness the primary
// payer's profile produced: their document and BOL requirements join the list,
// a payer on credit hold blocks under the same enforcement as a missing
// document, and auto-approval needs every payer's consent.
func applyPayerReadiness(
	readiness *shipmentbillingports.ShipmentBillingReadiness,
	entity *shipment.Shipment,
	resolution *chargeallocation.ShareResolution,
	payersByID map[pulid.ID]*customer.Customer,
	docs []*document.Document,
) {
	if readiness == nil || resolution == nil {
		return
	}
	readiness.Payers = make(
		[]shipmentbillingports.ShipmentBillingPayerReadiness,
		0,
		len(resolution.Shares),
	)

	requirementIssues := HasShipmentRequirementIssues(readiness)
	rateIssues := HasRateIssues(readiness)
	autoApprove := true
	known := make(map[string]struct{}, len(readiness.Requirements))
	for _, requirement := range readiness.Requirements {
		known[requirement.DocumentTypeID] = struct{}{}
	}

	for _, share := range resolution.Shares {
		payer := payersByID[share.PayerID]
		var profile *customer.CustomerBillingProfile
		entry := shipmentbillingports.ShipmentBillingPayerReadiness{
			PayerID:     share.PayerID,
			IsPrimary:   share.PayerID == resolution.DefaultPayerID,
			ShareAmount: share.TotalAmount,
		}
		if payer != nil {
			entry.PayerName = payer.Name
			entry.PayerCode = payer.Code
			profile = payer.BillingProfile
		}
		if profile != nil {
			entry.CreditStatus = profile.CreditStatus
			entry.CreditHold = profile.BlocksBilling()
		}

		if !entry.IsPrimary && profile != nil {
			for _, requirement := range buildDocumentRequirements(profile.DocumentTypes, docs) {
				if _, ok := known[requirement.DocumentTypeID]; ok {
					continue
				}
				known[requirement.DocumentTypeID] = struct{}{}
				readiness.Requirements = append(readiness.Requirements, requirement)
				if !requirement.Satisfied {
					readiness.MissingRequirements = append(
						readiness.MissingRequirements,
						requirement,
					)
				}
			}
			if readiness.Policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock &&
				profile.RequireBOLNumber &&
				entity.BOL == "" &&
				!hasValidationCode(readiness.ValidationFailures, "missing_bol") {
				readiness.ValidationFailures = append(readiness.ValidationFailures,
					shipmentbillingports.ShipmentBillingValidation{
						Field:   "bol",
						Code:    "missing_bol",
						Message: "BOL is required before the shipment can be invoiced",
					})
			}
		}

		if entry.CreditHold {
			readiness.ValidationFailures = append(readiness.ValidationFailures,
				shipmentbillingports.ShipmentBillingValidation{
					Field:   "billToCustomerId",
					Code:    "credit_hold",
					Message: entry.PayerName + " is on credit hold and cannot be billed",
				})
		}

		readiness.Payers = append(readiness.Payers, entry)
	}

	requirementIssues = HasShipmentRequirementIssues(readiness)
	rateIssues = HasRateIssues(readiness)
	for i := range readiness.Payers {
		payer := payersByID[readiness.Payers[i].PayerID]
		var profile *customer.CustomerBillingProfile
		if payer != nil {
			profile = payer.BillingProfile
		}
		readiness.Payers[i].ShouldAutoApproveBilling = shouldAutoApproveBilling(
			readiness.Policy,
			profile,
			requirementIssues,
			rateIssues,
		)
		autoApprove = autoApprove && readiness.Payers[i].ShouldAutoApproveBilling
	}

	readiness.CanMarkReadyToInvoice = isBillingReadyStatus(entity.Status) &&
		canProceedManually(readiness.Policy, requirementIssues, rateIssues)
	readiness.ShouldAutoMarkReadyToInvoice = readiness.Policy.ReadyToBillAssignmentMode == billingcontrol.ReadyToBillAssignmentModeAutomaticWhenEligible &&
		entity.Status == shipmentvocab.StatusCompleted &&
		canAutoProgress(readiness.Policy, requirementIssues, rateIssues)
	readiness.ShouldAutoTransferToBilling = readiness.ShouldAutoMarkReadyToInvoice &&
		readiness.Policy.BillingQueueTransferMode == billingcontrol.BillingQueueTransferModeAutomaticWhenReady
	readiness.ShouldAutoApproveBilling = autoApprove && len(readiness.Payers) > 0
}

func hasValidationCode(
	failures []shipmentbillingports.ShipmentBillingValidation,
	code string,
) bool {
	for _, failure := range failures {
		if failure.Code == code {
			return true
		}
	}

	return false
}

func buildShipmentBillingReadiness(
	entity *shipment.Shipment,
	billingProfile *customer.CustomerBillingProfile,
	billingControl *billingcontrol.BillingControl,
	docs []*document.Document,
) *shipmentbillingports.ShipmentBillingReadiness {
	readiness := &shipmentbillingports.ShipmentBillingReadiness{
		ShipmentID:          entity.ID.String(),
		ShipmentStatus:      entity.Status,
		Requirements:        []shipmentbillingports.ShipmentBillingRequirement{},
		MissingRequirements: []shipmentbillingports.ShipmentBillingRequirement{},
		ValidationFailures:  []shipmentbillingports.ShipmentBillingValidation{},
		Warnings:            []shipmentbillingports.ShipmentBillingWarning{},
		ServiceFailureContext: shipmentbillingports.ShipmentServiceFailureBillingContext{
			ServiceFailureIDs: []string{},
		},
		Policy: shipmentbillingports.ShipmentBillingReadinessPolicy{
			ShipmentBillingRequirementEnforcement: resolveShipmentBillingRequirementEnforcement(
				billingProfile,
				billingControl,
			),
			RateValidationEnforcement: resolveRateValidationEnforcement(
				billingProfile,
				billingControl,
			),
			BillingExceptionDisposition: resolveBillingExceptionDisposition(
				billingControl,
			),
			NotifyOnBillingExceptions: resolveNotifyOnBillingExceptions(billingControl),
			ReadyToBillAssignmentMode: resolveReadyToBillAssignmentMode(
				billingProfile,
				billingControl,
			),
			BillingQueueTransferMode: resolveBillingQueueTransferMode(
				billingProfile,
				billingControl,
			),
		},
	}

	if billingProfile == nil {
		appendRateValidationFailure(entity, billingControl, readiness)
		appendRateDepartureReasonFailure(entity, billingControl, readiness)
		requirementIssues := HasShipmentRequirementIssues(readiness)
		rateIssues := HasRateIssues(readiness)
		readiness.CanMarkReadyToInvoice = entity.Status == shipmentvocab.StatusCompleted &&
			canProceedManually(readiness.Policy, requirementIssues, rateIssues)
		readiness.ShouldAutoMarkReadyToInvoice = readiness.Policy.ReadyToBillAssignmentMode == billingcontrol.ReadyToBillAssignmentModeAutomaticWhenEligible &&
			entity.Status == shipmentvocab.StatusCompleted &&
			canAutoProgress(readiness.Policy, requirementIssues, rateIssues)
		readiness.ShouldAutoTransferToBilling = readiness.ShouldAutoMarkReadyToInvoice &&
			readiness.Policy.BillingQueueTransferMode == billingcontrol.BillingQueueTransferModeAutomaticWhenReady
		// A customer with no billing profile has not opted into auto-approval.
		return readiness
	}

	requirements := buildDocumentRequirements(billingProfile.DocumentTypes, docs)
	readiness.Requirements = requirements
	readiness.MissingRequirements = slices.DeleteFunc(
		slices.Clone(requirements),
		func(item shipmentbillingports.ShipmentBillingRequirement) bool {
			return item.Satisfied
		},
	)

	if readiness.Policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock {
		if billingProfile.RequireBOLNumber && entity.BOL == "" {
			readiness.ValidationFailures = append(
				readiness.ValidationFailures,
				shipmentbillingports.ShipmentBillingValidation{
					Field:   "bol",
					Code:    "missing_bol",
					Message: "BOL is required before the shipment can be invoiced",
				},
			)
		}
	}

	appendRateValidationFailure(entity, billingControl, readiness)
	appendRateDepartureReasonFailure(entity, billingControl, readiness)

	requirementIssues := HasShipmentRequirementIssues(readiness)
	rateIssues := HasRateIssues(readiness)

	readiness.CanMarkReadyToInvoice = isBillingReadyStatus(entity.Status) &&
		canProceedManually(readiness.Policy, requirementIssues, rateIssues)
	readiness.ShouldAutoMarkReadyToInvoice = readiness.Policy.ReadyToBillAssignmentMode == billingcontrol.ReadyToBillAssignmentModeAutomaticWhenEligible &&
		entity.Status == shipmentvocab.StatusCompleted &&
		canAutoProgress(readiness.Policy, requirementIssues, rateIssues)
	readiness.ShouldAutoTransferToBilling = readiness.ShouldAutoMarkReadyToInvoice &&
		readiness.Policy.BillingQueueTransferMode == billingcontrol.BillingQueueTransferModeAutomaticWhenReady
	readiness.ShouldAutoApproveBilling = shouldAutoApproveBilling(
		readiness.Policy,
		billingProfile,
		requirementIssues,
		rateIssues,
	)

	return readiness
}

// shouldAutoApproveBilling reports whether this shipment may clear the billing
// queue without a biller looking at it.
//
// Deliberately stricter than auto-transfer. canAutoProgress lets a shipment move
// while an enforcement level is Ignore, issues and all — which is defensible for
// getting freight into a work queue, and is not defensible for approving it,
// because approval is the step that puts the freight in front of the customer.
// So any requirement or rate issue at all stops auto-approval, whatever the
// organization's enforcement level says.
//
// The organization's queue-transfer mode is the organization-level gate, so a
// shop that has not enabled automatic transfer cannot be auto-approving
// anything; AutoApprove on the profile is the customer's opt-in within it. It
// reads the mode rather than ShouldAutoTransferToBilling because that flag also
// requires the shipment to still be Completed, and by the time anything asks
// this question the shipment is already ReadyToInvoice.
func shouldAutoApproveBilling(
	policy shipmentbillingports.ShipmentBillingReadinessPolicy,
	billingProfile *customer.CustomerBillingProfile,
	requirementIssues bool,
	rateIssues bool,
) bool {
	return policy.BillingQueueTransferMode == billingcontrol.BillingQueueTransferModeAutomaticWhenReady &&
		billingProfile != nil &&
		billingProfile.AutoApprove &&
		!requirementIssues &&
		!rateIssues
}

func isBillingReadyStatus(status shipmentvocab.Status) bool {
	return status == shipmentvocab.StatusCompleted ||
		status == shipmentvocab.StatusReadyToInvoice ||
		status == shipmentvocab.StatusInvoiced
}

func resolveShipmentBillingRequirementEnforcement(
	billingProfile *customer.CustomerBillingProfile,
	billingControl *billingcontrol.BillingControl,
) tenant.EnforcementLevel {
	orgLevel := tenant.EnforcementLevelIgnore
	if billingControl != nil {
		orgLevel = billingControl.ShipmentBillingRequirementEnforcement
	}

	customerLevel := tenant.EnforcementLevelIgnore
	if billingProfile != nil && billingProfile.EnforceCustomerBillingReq {
		customerLevel = tenant.EnforcementLevelBlock
	}

	return stricterEnforcementLevel(orgLevel, customerLevel)
}

func resolveReadyToBillAssignmentMode(
	billingProfile *customer.CustomerBillingProfile,
	billingControl *billingcontrol.BillingControl,
) billingcontrol.ReadyToBillAssignmentMode {
	if billingControl == nil {
		if billingProfile != nil && billingProfile.AutoMarkReadyToBill {
			return billingcontrol.ReadyToBillAssignmentModeAutomaticWhenEligible
		}

		return billingcontrol.ReadyToBillAssignmentModeManualOnly
	}

	orgMode := billingControl.ReadyToBillAssignmentMode

	if orgMode != billingcontrol.ReadyToBillAssignmentModeAutomaticWhenEligible {
		return billingcontrol.ReadyToBillAssignmentModeManualOnly
	}

	if billingProfile == nil || billingProfile.AutoMarkReadyToBill {
		return billingcontrol.ReadyToBillAssignmentModeAutomaticWhenEligible
	}

	return billingcontrol.ReadyToBillAssignmentModeManualOnly
}

func resolveRateValidationEnforcement(
	billingProfile *customer.CustomerBillingProfile,
	billingControl *billingcontrol.BillingControl,
) tenant.EnforcementLevel {
	orgLevel := tenant.EnforcementLevelIgnore
	if billingControl != nil {
		orgLevel = billingControl.RateValidationEnforcement
	}

	customerLevel := tenant.EnforcementLevelIgnore
	if billingProfile != nil && billingProfile.ValidateCustomerRates {
		customerLevel = tenant.EnforcementLevelRequireReview
	}

	return stricterEnforcementLevel(orgLevel, customerLevel)
}

func resolveBillingExceptionDisposition(
	billingControl *billingcontrol.BillingControl,
) billingcontrol.BillingExceptionDisposition {
	if billingControl == nil {
		return billingcontrol.BillingExceptionDispositionRouteToBillingReview
	}

	return billingControl.BillingExceptionDisposition
}

func resolveNotifyOnBillingExceptions(
	billingControl *billingcontrol.BillingControl,
) bool {
	if billingControl == nil {
		return false
	}

	return billingControl.NotifyOnBillingExceptions
}

func resolveBillingQueueTransferMode(
	billingProfile *customer.CustomerBillingProfile,
	billingControl *billingcontrol.BillingControl,
) billingcontrol.BillingQueueTransferMode {
	if billingControl == nil {
		if billingProfile != nil && billingProfile.AutoTransfer {
			return billingcontrol.BillingQueueTransferModeAutomaticWhenReady
		}

		return billingcontrol.BillingQueueTransferModeManualOnly
	}

	orgMode := billingControl.BillingQueueTransferMode

	if orgMode != billingcontrol.BillingQueueTransferModeAutomaticWhenReady {
		return billingcontrol.BillingQueueTransferModeManualOnly
	}

	if billingProfile == nil || billingProfile.AutoTransfer {
		return billingcontrol.BillingQueueTransferModeAutomaticWhenReady
	}

	return billingcontrol.BillingQueueTransferModeManualOnly
}

func stricterEnforcementLevel(
	left tenant.EnforcementLevel,
	right tenant.EnforcementLevel,
) tenant.EnforcementLevel {
	if enforcementRank(left) >= enforcementRank(right) {
		return left
	}

	return right
}

func enforcementRank(level tenant.EnforcementLevel) int {
	switch level { //nolint:exhaustive // only actionable enum states require explicit handling here
	case tenant.EnforcementLevelBlock:
		return 3
	case tenant.EnforcementLevelRequireReview:
		return 2
	case tenant.EnforcementLevelWarn:
		return 1
	default:
		return 0
	}
}

func buildDocumentRequirements(
	documentTypes []*documenttype.DocumentType,
	docs []*document.Document,
) []shipmentbillingports.ShipmentBillingRequirement {
	if len(documentTypes) == 0 {
		return []shipmentbillingports.ShipmentBillingRequirement{}
	}

	now := timeutils.NowUnix()
	accepted := make(map[pulid.ID][]string, len(documentTypes))
	ineligible := make(
		map[pulid.ID][]shipmentbillingports.ShipmentBillingIneligibleDocument,
		len(documentTypes),
	)
	for _, doc := range docs {
		if doc == nil || doc.DocumentTypeID == nil {
			continue
		}

		typeID := *doc.DocumentTypeID
		standing := doc.StandingAt(now)
		if standing == document.StandingAccepted {
			accepted[typeID] = append(accepted[typeID], doc.ID.String())
			continue
		}

		ineligible[typeID] = append(
			ineligible[typeID],
			shipmentbillingports.ShipmentBillingIneligibleDocument{
				DocumentID: doc.ID.String(),
				Standing:   standing,
			},
		)
	}

	requirements := make([]shipmentbillingports.ShipmentBillingRequirement, 0, len(documentTypes))
	for _, docType := range documentTypes {
		if docType == nil {
			continue
		}

		documentIDs := accepted[docType.ID]
		if documentIDs == nil {
			documentIDs = []string{}
		}
		rejected := ineligible[docType.ID]
		if rejected == nil {
			rejected = []shipmentbillingports.ShipmentBillingIneligibleDocument{}
		}

		requirements = append(requirements, shipmentbillingports.ShipmentBillingRequirement{
			DocumentTypeID:      docType.ID.String(),
			DocumentTypeCode:    docType.Code,
			DocumentTypeName:    docType.Name,
			Satisfied:           len(documentIDs) > 0,
			DocumentCount:       len(documentIDs),
			DocumentIDs:         documentIDs,
			IneligibleDocuments: rejected,
		})
	}

	sort.SliceStable(requirements, func(i, j int) bool {
		if requirements[i].DocumentTypeCode == requirements[j].DocumentTypeCode {
			return requirements[i].DocumentTypeName < requirements[j].DocumentTypeName
		}

		return requirements[i].DocumentTypeCode < requirements[j].DocumentTypeCode
	})

	return requirements
}

func appendRateValidationFailure(
	entity *shipment.Shipment,
	billingControl *billingcontrol.BillingControl,
	readiness *shipmentbillingports.ShipmentBillingReadiness,
) {
	result := evaluateRateValidation(
		entity,
		billingControl,
		readiness.Policy.RateValidationEnforcement,
	)
	if result == nil {
		return
	}

	readiness.ValidationFailures = append(
		readiness.ValidationFailures,
		shipmentbillingports.ShipmentBillingValidation{
			Field:   result.Field,
			Code:    result.Code,
			Message: result.Message,
		},
	)
}

// appendRateDepartureReasonFailure holds a shipment priced away from its
// contract to the organization's own policy.
//
// A rate that departs from the contract is set by editing the shipment's rating
// fields, which is an ordinary save and cannot be made to demand an explanation
// at the moment it happens. So the demand lands here instead, where it has to
// be satisfied before the shipment can bill — which is the point at which the
// reason is actually needed, since it is what the invoice is explained with.
func appendRateDepartureReasonFailure(
	entity *shipment.Shipment,
	billingControl *billingcontrol.BillingControl,
	readiness *shipmentbillingports.ShipmentBillingReadiness,
) {
	if billingControl == nil || !billingControl.RequireRateOverrideReason {
		return
	}

	if entity == nil || entity.AutoRated || !entity.HasRateOverride() ||
		entity.RateOverrideReason != "" {
		return
	}

	readiness.ValidationFailures = append(
		readiness.ValidationFailures,
		shipmentbillingports.ShipmentBillingValidation{
			Field:   "rateOverrideReason",
			Code:    "rate_departure_reason_missing",
			Message: "Your organization requires a reason when a shipment is not billed at its contract rate",
		},
	)
}

func evaluateRateValidation(
	entity *shipment.Shipment,
	billingControl *billingcontrol.BillingControl,
	enforcement tenant.EnforcementLevel,
) *rateValidationResult {
	if enforcement == tenant.EnforcementLevelIgnore {
		return nil
	}

	if entity == nil || entity.RatingDetail == nil ||
		(entity.RatingDetail.Bill == nil && !entity.RatingDetail.RatedByHand()) {
		return &rateValidationResult{
			Field:   "ratingDetail",
			Code:    "rate_missing_basis",
			Message: "A calculated rate basis is required before billing can proceed",
		}
	}
	if entity.RatingDetail.Bill == nil {
		return nil
	}

	expected := entity.RatingDetail.Bill.FreightTotal
	actual := entity.FreightChargeAmount.Decimal
	variancePercent := calculateVariancePercent(expected, actual)
	if variancePercent.IsZero() {
		return nil
	}
	if billingControl != nil &&
		enforcement == tenant.EnforcementLevelRequireReview &&
		billingControl.RateVarianceAutoResolutionMode == billingcontrol.RateVarianceAutoResolutionModeBypassReviewWithinTolerance &&
		!variancePercent.GreaterThan(billingControl.RateVarianceTolerancePercent) {
		return nil
	}

	return &rateValidationResult{
		Field:   "freightChargeAmount",
		Code:    "rate_variance_requires_action",
		Message: "Rate validation requires billing review before billing can proceed",
	}
}

func calculateVariancePercent(
	expected decimal.Decimal,
	actual decimal.Decimal,
) decimal.Decimal {
	if expected.IsZero() {
		if actual.IsZero() {
			return decimal.Zero
		}

		return decimal.NewFromInt(100)
	}

	return actual.Sub(expected).Abs().Div(expected.Abs()).Mul(decimal.NewFromInt(100))
}

func hasValidationCodePrefix(
	failures []shipmentbillingports.ShipmentBillingValidation,
	prefix string,
) bool {
	for _, failure := range failures {
		if len(failure.Code) >= len(prefix) && failure.Code[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

func HasShipmentRequirementIssues(
	readiness *shipmentbillingports.ShipmentBillingReadiness,
) bool {
	if readiness == nil {
		return false
	}

	if len(readiness.MissingRequirements) > 0 {
		return true
	}

	for _, failure := range readiness.ValidationFailures {
		if len(failure.Code) < len("rate_") || failure.Code[:len("rate_")] != "rate_" {
			return true
		}
	}

	return false
}

func HasRateIssues(
	readiness *shipmentbillingports.ShipmentBillingReadiness,
) bool {
	if readiness == nil {
		return false
	}

	return hasValidationCodePrefix(readiness.ValidationFailures, "rate_")
}

func canProceedManually(
	policy shipmentbillingports.ShipmentBillingReadinessPolicy,
	requirementIssues bool,
	rateIssues bool,
) bool {
	if policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock &&
		requirementIssues {
		return false
	}

	if policy.RateValidationEnforcement == tenant.EnforcementLevelBlock && rateIssues {
		return false
	}

	if policy.BillingExceptionDisposition == billingcontrol.BillingExceptionDispositionReturnToOperations {
		if policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelRequireReview &&
			requirementIssues {
			return false
		}

		if policy.RateValidationEnforcement == tenant.EnforcementLevelRequireReview && rateIssues {
			return false
		}
	}

	return true
}

func canAutoProgress(
	policy shipmentbillingports.ShipmentBillingReadinessPolicy,
	requirementIssues bool,
	rateIssues bool,
) bool {
	if policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock &&
		requirementIssues {
		return false
	}
	if policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelRequireReview &&
		requirementIssues {
		return false
	}
	if policy.RateValidationEnforcement == tenant.EnforcementLevelBlock && rateIssues {
		return false
	}
	if policy.RateValidationEnforcement == tenant.EnforcementLevelRequireReview && rateIssues {
		return false
	}

	return true
}
