package billingqueueservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// podCode is the document type code a proof of delivery is filed under.
const podCode = "POD"

// driverNotifier is the driver portal's notification path, which is how the
// item asks a driver for paperwork.
type driverNotifier interface {
	Notify(ctx context.Context, req *drivernotificationservice.DriverNotification)
}

func notifierOrNil(svc *drivernotificationservice.Service) driverNotifier {
	if svc == nil {
		return nil
	}

	return svc
}

// attachReview fills the item's review: the charges against the rate con, the
// paperwork, the terms and the invoice, the issues the checks raised, and the
// five checks themselves. A part that cannot be read is logged and left out
// rather than failing the read, the way the shipment details already are.
func (s *service) attachReview(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
) {
	if item == nil {
		return
	}

	facts := s.reviewFacts(ctx, item, tenantInfo)
	issues := s.storedIssues(ctx, item, tenantInfo, facts)
	markLedger(facts.charges, issues)
	facts.charges.Totals()

	review := billingqueue.BuildReadiness(item, issues, billingqueue.CheckFacts{
		BillerName:   facts.billerName,
		Charges:      facts.charges,
		POD:          facts.pod,
		Terms:        facts.terms,
		ShipmentRef:  shipmentRef(item.Shipment),
		CustomerName: facts.customerName,
	})
	review.Invoice = facts.invoice
	review.Documents = facts.documents
	review.Duplicates = facts.duplicates
	for _, issue := range issues {
		issue.Undoable = issue.Settled() && billingqueue.IsReviewable(item.Status) &&
			issueEffectKind(issue) != billingqueue.EffectRequest
	}
	item.Review = review
}

type reviewFacts struct {
	billerName   string
	customerName string
	charges      *billingqueue.ChargeReview
	pod          *billingqueue.PODState
	terms        *billingqueue.BillingTerms
	invoice      *billingqueue.InvoiceRef
	documents    []*billingqueue.DocumentTile
	duplicates   []*billingqueue.DuplicateRef
	dwells       []*billingqueue.DetentionDwell
	findings     billingqueue.FindingInput
}

func (s *service) reviewFacts(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
) *reviewFacts {
	facts := &reviewFacts{}
	if item.AssignedBiller != nil {
		facts.billerName = item.AssignedBiller.Name
	}
	if item.BillToCustomer != nil {
		facts.customerName = item.BillToCustomer.Name
	}

	occurrences := s.shipmentOccurrences(ctx, item, tenantInfo)
	facts.charges = s.chargeReview(ctx, item, tenantInfo, occurrences)
	facts.dwells = detentionDwells(item, occurrences)
	facts.invoice = s.invoiceRef(ctx, item, tenantInfo)
	facts.terms = termsFor(item.BillToCustomer, facts.invoice)

	docs := s.shipmentDocuments(ctx, item, tenantInfo)
	required := requiredDocumentTypes(item.BillToCustomer)
	facts.pod = podState(item.Shipment, docs, required)
	facts.documents = documentTiles(docs, required)

	if s.reviewRepo != nil && item.ShipmentID.IsNotNil() {
		invoiceID := pulid.Nil
		if facts.invoice != nil {
			invoiceID = facts.invoice.ID
		}
		dups, err := s.reviewRepo.FindDuplicates(ctx, &repositories.FindBillingQueueDuplicatesRequest{
			TenantInfo:       tenantInfo,
			ItemID:           item.ID,
			ShipmentID:       item.ShipmentID,
			BillToCustomerID: item.BillToCustomerID,
			BillType:         item.BillType,
			InvoiceID:        invoiceID,
		})
		if err != nil {
			s.l.Warn("failed to look for duplicate bills", zap.String("billingQueueItemId", item.ID.String()), zap.Error(err))
		}
		facts.duplicates = dups
	}

	creditHold, holdReason := creditHoldOf(item.BillToCustomer)
	shareError := ""
	if item.PayerShare != nil {
		shareError = item.PayerShare.ResolutionError
	}
	facts.findings = billingqueue.FindingInput{
		CustomerName:     facts.customerName,
		ShipmentRef:      shipmentRef(item.Shipment),
		Charges:          facts.charges,
		POD:              facts.pod,
		Detention:        facts.dwells,
		DetentionHolds:   item.DetentionHolds,
		Duplicates:       facts.duplicates,
		CreditHold:       creditHold,
		CreditHoldReason: holdReason,
		PayerShareError:  shareError,
	}

	return facts
}

// storedIssues syncs what the checks find now onto the stored issues while
// the item is still under review, then returns all of them. Once the item is
// past review its issues are history and are only read.
func (s *service) storedIssues(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
	facts *reviewFacts,
) []*billingqueue.Issue {
	if s.reviewRepo == nil {
		return []*billingqueue.Issue{}
	}
	if !billingqueue.IsReviewable(item.Status) && item.Status != billingqueue.StatusOnHold {
		issues, err := s.reviewRepo.ListIssues(ctx, tenantInfo, item.ID)
		if err != nil {
			s.l.Warn("failed to read billing queue issues", zap.String("billingQueueItemId", item.ID.String()), zap.Error(err))
			return []*billingqueue.Issue{}
		}
		return issues
	}

	result, err := s.reviewRepo.SyncIssues(ctx, &repositories.SyncBillingQueueIssuesRequest{
		TenantInfo: tenantInfo,
		ItemID:     item.ID,
		Findings:   billingqueue.Findings(facts.findings),
	})
	if err != nil {
		s.l.Warn("failed to sync billing queue issues", zap.String("billingQueueItemId", item.ID.String()), zap.Error(err))
		return []*billingqueue.Issue{}
	}

	events := make([]*billingqueue.ItemEvent, 0, len(result.Raised)+len(result.Cleared))
	for _, issue := range append(result.Raised, result.Reopened...) {
		events = append(events, s.newEvent(ctx, item, billingqueue.EventIssueRaised,
			"Flagged: "+issue.Summary, nil, map[string]any{"issueId": issue.ID.String()}))
	}
	for _, issue := range result.Cleared {
		events = append(events, s.newEvent(ctx, item, billingqueue.EventIssueCleared,
			"Cleared: "+issue.Summary, nil, map[string]any{"issueId": issue.ID.String()}))
	}
	if len(events) > 0 {
		if err = s.reviewRepo.CreateEvents(ctx, events...); err != nil {
			s.l.Warn("failed to record billing queue issue activity", zap.Error(err))
		}
	}

	return result.Issues
}

// syncReview reads an item the way the review does, which is what raises its
// issues. Transfers and charge edits call it so the queue's badges are right
// before anybody opens the item.
func (s *service) syncReview(ctx context.Context, itemID pulid.ID, tenantInfo pagination.TenantInfo) {
	if s.reviewRepo == nil {
		return
	}
	if _, err := s.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:                itemID,
		TenantInfo:            tenantInfo,
		ExpandShipmentDetails: true,
	}); err != nil {
		s.l.Warn("failed to check billing queue item", zap.String("billingQueueItemId", itemID.String()), zap.Error(err))
	}
}

func (s *service) guardOpenIssues(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
) error {
	if s.reviewRepo == nil || item.BillType == billingqueue.BillTypeCreditMemo {
		return nil
	}
	open, err := s.reviewRepo.CountOpenIssues(ctx, tenantInfo, item.ID)
	if err != nil {
		return err
	}
	if open == 0 {
		return nil
	}

	return errortypes.NewValidationError(
		"status",
		errortypes.ErrInvalidOperation,
		"Settle the flagged check first",
	)
}

func (s *service) shipmentOccurrences(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
) []*detention.DetentionOccurrence {
	if s.occurrenceRepo == nil || item.ShipmentID.IsNil() {
		return nil
	}
	occurrences, err := s.occurrenceRepo.GetByShipment(ctx, &repositories.GetOccurrencesByShipmentRequest{
		ShipmentID:      item.ShipmentID,
		TenantInfo:      tenantInfo,
		IncludeEvidence: true,
	})
	if err != nil {
		s.l.Warn("failed to read detention for billing review", zap.String("billingQueueItemId", item.ID.String()), zap.Error(err))
		return nil
	}

	return occurrences
}

// chargeReview sets each line the item bills against what priced it: the
// rating for freight, the contract's accessorial schedule, the fuel program,
// the detention policy. A line nothing on the rate con explains expects
// nothing, which is what flags it.
func (s *service) chargeReview(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
	occurrences []*detention.DetentionOccurrence,
) *billingqueue.ChargeReview {
	review := &billingqueue.ChargeReview{Lines: []*billingqueue.ChargeLine{}}
	if item.PayerShare == nil || item.Shipment == nil {
		return review
	}
	shp := item.Shipment
	charges := make(map[pulid.ID]*shipment.AdditionalCharge, len(shp.AdditionalCharges))
	for _, charge := range shp.AdditionalCharges {
		if charge != nil {
			charges[charge.ID] = charge
		}
	}
	byCharge := make(map[pulid.ID]*detention.DetentionOccurrence, len(occurrences))
	for _, occurrence := range occurrences {
		if occurrence != nil && occurrence.AdditionalChargeID != nil {
			byCharge[*occurrence.AdditionalChargeID] = occurrence
		}
	}
	agreement := s.ratingAgreement(ctx, shp, tenantInfo)
	freight := shp.FreightChargeAmount.Decimal

	for idx, share := range item.PayerShare.Lines {
		line := &billingqueue.ChargeLine{
			Label:  share.Description,
			Billed: share.Amount,
			Source: billingqueue.ChargeSourceNone,
		}
		scale := func(expected decimal.Decimal) decimal.NullDecimal {
			if share.Partial && share.ChargeTotal.IsPositive() {
				expected = expected.Mul(share.Amount).Div(share.ChargeTotal).Round(2)
			}
			return decimal.NewNullDecimal(expected)
		}

		if share.AdditionalChargeID.IsNil() {
			line.Key = fmt.Sprintf("freight-%d", idx)
			if shp.RatingDetail != nil && shp.RatingDetail.Result > 0 {
				line.Label = "Linehaul"
				line.Source = billingqueue.ChargeSourceRating
				line.Expected = scale(decimal.NewFromFloat(shp.RatingDetail.Result).Round(2))
			}
			line.Basis = freightBasis(shp)
			review.Lines = append(review.Lines, line)
			continue
		}

		line.Key = share.AdditionalChargeID.String()
		line.AdditionalChargeID = share.AdditionalChargeID
		charge := charges[share.AdditionalChargeID]
		if charge == nil {
			review.Lines = append(review.Lines, line)
			continue
		}

		switch {
		case charge.IsDetention || byCharge[charge.ID] != nil:
			line.Source = billingqueue.ChargeSourceDetention
			occurrence := byCharge[charge.ID]
			if occurrence != nil {
				line.Expected = scale(occurrence.BillableAmount)
				line.Basis = detentionBasis(occurrence)
			} else {
				line.Expected = scale(charge.Total(freight))
			}
		case charge.FuelSurchargeProgramID != nil || charge.FuelSurchargeDetail != nil:
			line.Source = billingqueue.ChargeSourceFuel
			line.Expected = scale(charge.Total(freight))
			line.Basis = fuelBasis(charge.FuelSurchargeDetail)
		default:
			if priced := agreementPrice(agreement, charge, freight); priced.Valid {
				line.Source = billingqueue.ChargeSourceAgreement
				line.Expected = scale(priced.Decimal)
				line.Basis = "Per rate con"
			} else if charge.RateQuoteID != nil {
				line.Source = billingqueue.ChargeSourceQuote
				line.Expected = scale(charge.Total(freight))
				line.Basis = "Per quote"
			} else {
				line.Basis = manualBasis(charge)
			}
		}
		review.Lines = append(review.Lines, line)
	}
	review.Totals()

	return review
}

func (s *service) ratingAgreement(
	ctx context.Context,
	shp *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
) *rateagreement.RateAgreement {
	if s.agreementRepo == nil || shp.RatingDetail == nil || shp.RatingDetail.AgreementID == "" {
		return nil
	}
	agreementID, err := pulid.Parse(shp.RatingDetail.AgreementID)
	if err != nil {
		return nil
	}
	agreement, err := s.agreementRepo.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: agreementID,
		TenantInfo:      tenantInfo,
		IncludeChildren: true,
		AsOf:            shp.RatingDetail.RatedAt,
	})
	if err != nil {
		s.l.Warn("failed to read the agreement that rated the shipment", zap.String("shipmentId", shp.ID.String()), zap.Error(err))
		return nil
	}

	return agreement
}

// agreementPrice is what the contract's accessorial schedule charges for the
// charge: the accessorial the charge was generated from, or else the one the
// contract prices for the same accessorial.
func agreementPrice(
	agreement *rateagreement.RateAgreement,
	charge *shipment.AdditionalCharge,
	freight decimal.Decimal,
) decimal.NullDecimal {
	if agreement == nil {
		return decimal.NullDecimal{}
	}
	for _, acc := range agreement.Accessorials {
		if acc == nil {
			continue
		}
		matches := (charge.RateAgreementAccessorialID != nil && acc.ID == *charge.RateAgreementAccessorialID) ||
			acc.AccessorialChargeID == charge.AccessorialChargeID
		if !matches {
			continue
		}
		priced := &shipment.AdditionalCharge{
			Method: acc.Method,
			Amount: acc.PricedAmount(),
			Unit:   max(acc.BillableUnits(charge.Unit), 0),
		}
		return decimal.NewNullDecimal(priced.Total(freight).Round(2))
	}

	return decimal.NullDecimal{}
}

func freightBasis(shp *shipment.Shipment) string {
	miles := 0.0
	for _, move := range shp.Moves {
		if move != nil && move.Loaded && move.Distance != nil {
			miles += *move.Distance
		}
	}
	detail := shp.RatingDetail
	parts := make([]string, 0, 2)
	if miles > 0 && shp.FreightChargeAmount.Decimal.IsPositive() {
		rate := shp.FreightChargeAmount.Decimal.Div(decimal.NewFromFloat(miles)).Round(2)
		parts = append(parts, fmt.Sprintf("%.0f mi × %s", miles, billingqueue.FormatMoney(rate)))
	}
	if detail != nil {
		switch {
		case detail.AgreementName != "":
			parts = append(parts, detail.AgreementName)
		case detail.FormulaTemplateName != "":
			parts = append(parts, detail.FormulaTemplateName)
		}
	}
	if len(parts) == 0 {
		return "Freight"
	}

	return strings.Join(parts, " · ")
}

func fuelBasis(detail *shipment.FuelSurchargeDetail) string {
	if detail == nil {
		return "Fuel program"
	}
	parts := make([]string, 0, 2)
	if detail.Price > 0 {
		index := detail.IndexCode
		if index == "" {
			index = "Index"
		}
		parts = append(parts, fmt.Sprintf("%s $%.2f", index, detail.Price))
	}
	if detail.ProgramName != "" {
		parts = append(parts, detail.ProgramName)
	}
	if len(parts) == 0 {
		return "Fuel program"
	}

	return strings.Join(parts, " · ")
}

func detentionBasis(occurrence *detention.DetentionOccurrence) string {
	billable := occurrence.RoundedMinutes
	if billable == 0 {
		billable = occurrence.BillableMinutes
	}
	rate := hourlyRate(occurrence)
	text := billingqueue.FormatHours(billable)
	if rate.IsPositive() {
		text += " × " + billingqueue.FormatMoney(rate)
	}
	if occurrence.FreeMinutesGranted > 0 {
		text += " after " + billingqueue.FormatHours(occurrence.FreeMinutesGranted) + " free"
	}

	return text
}

func manualBasis(charge *shipment.AdditionalCharge) string {
	if charge.Unit > 1 {
		return fmt.Sprintf("%d × %s · added by hand", charge.Unit, billingqueue.FormatMoney(charge.Amount))
	}

	return "Added by hand"
}

func hourlyRate(occurrence *detention.DetentionOccurrence) decimal.Decimal {
	minutes := occurrence.RoundedMinutes
	if minutes == 0 {
		minutes = occurrence.BillableMinutes
	}
	if minutes <= 0 || !occurrence.BillableAmount.IsPositive() {
		if occurrence.PolicySnapshot != nil {
			return occurrence.PolicySnapshot.FlatRate
		}
		return decimal.Zero
	}

	return occurrence.BillableAmount.Mul(decimal.NewFromInt(60)).
		Div(decimal.NewFromInt32(minutes)).Round(2)
}

// detentionDwells sets each billed detention wait against the truck's own
// clock. The ELD's dwell is read from the telematics and geofence evidence for
// the stop's arrival and departure; a wait with no such evidence has nothing
// to compare and is left alone.
func detentionDwells(
	item *billingqueue.BillingQueueItem,
	occurrences []*detention.DetentionOccurrence,
) []*billingqueue.DetentionDwell {
	if item.PayerShare == nil {
		return nil
	}
	billed := make(map[pulid.ID]decimal.Decimal, len(item.PayerShare.Lines))
	for _, line := range item.PayerShare.Lines {
		if line.AdditionalChargeID.IsNotNil() {
			billed[line.AdditionalChargeID] = line.Amount
		}
	}

	out := make([]*billingqueue.DetentionDwell, 0, len(occurrences))
	for _, occurrence := range occurrences {
		if occurrence == nil || occurrence.AdditionalChargeID == nil {
			continue
		}
		amount, ok := billed[*occurrence.AdditionalChargeID]
		if !ok {
			continue
		}
		arrived, departed := eldTimes(occurrence.Evidence)
		if arrived == 0 || departed <= arrived {
			continue
		}
		eldMinutes := int32((departed - arrived) / 60)
		rate := hourlyRate(occurrence)
		eldBillable := max(eldMinutes-occurrence.FreeMinutesGranted, 0)
		if occurrence.PolicySnapshot != nil && occurrence.PolicySnapshot.BillingIncrementMinutes > 0 {
			step := int32(occurrence.PolicySnapshot.BillingIncrementMinutes)
			eldBillable = (eldBillable + step - 1) / step * step
		}
		out = append(out, &billingqueue.DetentionDwell{
			OccurrenceID:  occurrence.ID,
			ChargeID:      *occurrence.AdditionalChargeID,
			LocationName:  occurrence.LocationName,
			LoggedMinutes: occurrence.RawDwellMinutes,
			ELDMinutes:    eldMinutes,
			FreeMinutes:   occurrence.FreeMinutesGranted,
			HourlyRate:    rate,
			BilledAmount:  amount,
			ELDAmount: rate.Mul(decimal.NewFromInt32(eldBillable)).
				Div(decimal.NewFromInt(60)).Round(2),
		})
	}

	return out
}

func eldTimes(evidence []*detention.DetentionEvidence) (arrived, departed int64) {
	for _, entry := range evidence {
		if entry == nil || (entry.Source != detention.EvidenceSourceTelematics &&
			entry.Source != detention.EvidenceSourceGeofence) {
			continue
		}
		switch entry.Kind { //nolint:exhaustive // only the dock times matter here
		case detention.EvidenceKindArrival:
			if arrived == 0 || entry.ObservedAt < arrived {
				arrived = entry.ObservedAt
			}
		case detention.EvidenceKindDeparture:
			if entry.ObservedAt > departed {
				departed = entry.ObservedAt
			}
		}
	}

	return arrived, departed
}

func (s *service) invoiceRef(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
) *billingqueue.InvoiceRef {
	if s.invoiceRepo == nil {
		return nil
	}
	if item.Status != billingqueue.StatusApproved && item.Status != billingqueue.StatusPosted {
		return nil
	}
	inv, err := s.invoiceRepo.GetByBillingQueueItemID(ctx, repositories.GetInvoiceByBillingQueueItemIDRequest{
		BillingQueueItemID: item.ID,
		TenantInfo:         tenantInfo,
	})
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			s.l.Warn("failed to read the item's invoice", zap.String("billingQueueItemId", item.ID.String()), zap.Error(err))
		}
		return nil
	}

	posted := inv.Status == "Posted"
	return &billingqueue.InvoiceRef{
		ID:          inv.ID,
		Number:      inv.Number,
		DraftNumber: billingqueue.DraftInvoiceNumber(inv.Number),
		Status:      string(inv.Status),
		InvoiceDate: inv.InvoiceDate,
		DueDate:     inv.DueDate,
		PostedAt:    inv.PostedAt,
		Posted:      posted,
	}
}

// termsFor is how the bill-to pays: its payment term, and when this invoice
// falls due — the invoice's own date once there is one, otherwise as if it
// were invoiced today.
func termsFor(cus *customer.Customer, inv *billingqueue.InvoiceRef) *billingqueue.BillingTerms {
	terms := &billingqueue.BillingTerms{Recipients: []string{}}
	if cus == nil {
		return terms
	}
	if cus.EmailProfile != nil {
		terms.Recipients = billingqueue.ParseRecipients(cus.EmailProfile.ToRecipients)
	}
	if cus.BillingProfile == nil {
		return terms
	}
	terms.PaymentTerm = string(cus.BillingProfile.PaymentTerm)
	terms.NetDays = billingqueue.NetDaysFor(terms.PaymentTerm)
	terms.CreditHold, _ = creditHoldOf(cus)
	switch {
	case inv != nil && inv.DueDate != nil:
		terms.DueDate = inv.DueDate
	case terms.PaymentTerm != "":
		due := timeutils.NowUnix() + int64(terms.NetDays)*int64((24*time.Hour)/time.Second)
		terms.DueDate = &due
	}

	return terms
}

func creditHoldOf(cus *customer.Customer) (bool, string) {
	if cus == nil || cus.BillingProfile == nil {
		return false, ""
	}
	profile := cus.BillingProfile
	held := profile.EnforceCreditLimit &&
		(profile.CreditStatus == customer.CreditStatusHold ||
			profile.CreditStatus == customer.CreditStatusSuspended)

	return held, profile.CreditHoldReason
}

func (s *service) shipmentDocuments(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
) []*document.Document {
	if s.documentRepo == nil || item.ShipmentID.IsNil() {
		return nil
	}
	docs, err := s.documentRepo.GetByResourceID(ctx, &repositories.GetDocumentsByResourceRequest{
		TenantInfo:          tenantInfo,
		ResourceID:          item.ShipmentID.String(),
		ResourceType:        "shipment",
		IncludeDocumentType: true,
	})
	if err != nil {
		s.l.Warn("failed to read the shipment's documents", zap.String("billingQueueItemId", item.ID.String()), zap.Error(err))
		return nil
	}
	current := make([]*document.Document, 0, len(docs))
	for _, doc := range docs {
		if doc != nil && doc.IsCurrentVersion && doc.Status != document.StatusRejected &&
			doc.Status != document.StatusArchived {
			current = append(current, doc)
		}
	}

	return current
}

func requiredDocumentTypes(cus *customer.Customer) []*documenttype.DocumentType {
	if cus == nil || cus.BillingProfile == nil {
		return nil
	}

	return cus.BillingProfile.DocumentTypes
}

// podState reads the proof of delivery: whether the bill-to requires one,
// whether it is on file, and whether it is signed. A POD whose signature
// nobody has read yet counts as on file; only one known to be unsigned is
// flagged.
func podState(
	shp *shipment.Shipment,
	docs []*document.Document,
	required []*documenttype.DocumentType,
) *billingqueue.PODState {
	pod := &billingqueue.PODState{DocumentType: podCode}
	for _, docType := range required {
		if docType != nil && docType.Code == podCode {
			pod.Required = true
			pod.DocumentType = docType.Code
		}
	}

	for _, doc := range docs {
		if doc.DocumentType == nil || doc.DocumentType.Code != podCode {
			continue
		}
		signed := doc.SignatureStatus == document.SignatureStatusSigned
		// A signed copy wins over an unsigned one filed earlier.
		if pod.Present && pod.Signed && !signed {
			continue
		}
		pod.Present = true
		pod.DocumentID = doc.ID
		pod.Signed = signed
		pod.Unsigned = doc.DocumentType.RequiresSignature &&
			doc.SignatureStatus == document.SignatureStatusUnsigned
		pod.SignedAt = doc.SignedAt
	}

	if shp != nil {
		if stop := shp.DeliveryStop(); stop != nil {
			pod.DeliveredAt = stop.ActualArrival
		}
		for _, move := range shp.Moves {
			if move == nil || move.Assignment == nil || move.Assignment.PrimaryWorker == nil {
				continue
			}
			wrk := move.Assignment.PrimaryWorker
			pod.DriverID = wrk.ID
			pod.DriverName = strings.TrimSpace(wrk.FirstName + " " + wrk.LastName)
		}
	}

	return pod
}

// documentTiles is the paperwork row under the shipment: every document the
// bill-to requires, missing or not, and then the rest on file.
func documentTiles(
	docs []*document.Document,
	required []*documenttype.DocumentType,
) []*billingqueue.DocumentTile {
	tiles := make([]*billingqueue.DocumentTile, 0, len(required)+len(docs))
	seen := make(map[pulid.ID]struct{}, len(docs))
	byType := make(map[pulid.ID][]*document.Document, len(docs))
	for _, doc := range docs {
		if doc.DocumentTypeID != nil {
			byType[*doc.DocumentTypeID] = append(byType[*doc.DocumentTypeID], doc)
		}
	}

	tile := func(doc *document.Document, docType *documenttype.DocumentType, isRequired bool) *billingqueue.DocumentTile {
		t := &billingqueue.DocumentTile{
			Required: isRequired,
			State:    billingqueue.DocumentStateOK,
		}
		if docType != nil {
			t.Code = docType.Code
			t.Name = docType.Name
		}
		if doc == nil {
			t.State = billingqueue.DocumentStateMissing
			return t
		}
		t.DocumentID = doc.ID
		t.FileName = doc.OriginalName
		t.Signed = doc.SignatureStatus == document.SignatureStatusSigned
		if docType != nil && docType.RequiresSignature &&
			doc.SignatureStatus == document.SignatureStatusUnsigned {
			t.State = billingqueue.DocumentStateUnsigned
		}
		if t.Name == "" {
			t.Name = doc.OriginalName
		}
		return t
	}

	for _, docType := range required {
		if docType == nil {
			continue
		}
		matches := byType[docType.ID]
		if len(matches) == 0 {
			tiles = append(tiles, tile(nil, docType, true))
			continue
		}
		best := matches[0]
		for _, doc := range matches {
			if doc.SignatureStatus == document.SignatureStatusSigned {
				best = doc
			}
		}
		for _, doc := range matches {
			seen[doc.ID] = struct{}{}
		}
		tiles = append(tiles, tile(best, docType, true))
	}
	for _, doc := range docs {
		if _, ok := seen[doc.ID]; ok {
			continue
		}
		tiles = append(tiles, tile(doc, doc.DocumentType, false))
		if len(tiles) >= 8 {
			break
		}
	}

	return tiles
}

// markLedger lays the issues over the ledger: a line an open issue points at
// is flagged, a line a settled issue repriced shows its new basis, and a line
// a settled issue removed comes back struck through so it can be undone.
func markLedger(review *billingqueue.ChargeReview, issues []*billingqueue.Issue) {
	byCharge := make(map[pulid.ID]*billingqueue.ChargeLine, len(review.Lines))
	for _, line := range review.Lines {
		if line.AdditionalChargeID.IsNotNil() {
			byCharge[line.AdditionalChargeID] = line
		}
	}

	for _, issue := range issues {
		if issue.FlaggedChargeID == nil {
			continue
		}
		line := byCharge[*issue.FlaggedChargeID]
		if issue.IsOpen() {
			if line != nil {
				line.Flagged = true
			}
			continue
		}
		if !issue.Settled() {
			continue
		}
		option := issue.Option(*issue.ResolutionKey)
		if option == nil {
			continue
		}
		switch option.Effect.Kind { //nolint:exhaustive // only charge edits show in the ledger
		case billingqueue.EffectSet:
			if line == nil && issue.EffectSnapshot != nil && issue.EffectSnapshot.RestoredChargeID.IsNotNil() {
				line = byCharge[issue.EffectSnapshot.RestoredChargeID]
			}
			if line != nil {
				line.Adjusted = true
				line.IssueID = issue.ID
				if option.Effect.Basis != "" {
					line.Basis = option.Effect.Basis
				}
			}
		case billingqueue.EffectDrop:
			if line != nil || issue.EffectSnapshot == nil || issue.EffectSnapshot.Charge == nil {
				continue
			}
			snap := issue.EffectSnapshot.Charge
			removed := &billingqueue.ChargeLine{
				Key:                snap.AdditionalChargeID.String(),
				AdditionalChargeID: snap.AdditionalChargeID,
				Label:              snap.Label,
				Basis:              snap.Basis,
				Billed:             snap.Billed,
				Source:             billingqueue.ChargeSourceNone,
				Removed:            true,
				IssueID:            issue.ID,
			}
			if snap.Expected != nil {
				if expected, err := decimal.NewFromString(*snap.Expected); err == nil {
					removed.Expected = decimal.NewNullDecimal(expected)
				}
			}
			review.Lines = append(review.Lines, removed)
		}
	}
}

func issueEffectKind(issue *billingqueue.Issue) billingqueue.EffectKind {
	if issue.ResolutionKey == nil {
		return ""
	}
	if option := issue.Option(*issue.ResolutionKey); option != nil {
		return option.Effect.Kind
	}

	return ""
}

func shipmentRef(shp *shipment.Shipment) string {
	if shp == nil {
		return "this shipment"
	}
	switch {
	case shp.ProNumber != "":
		return shp.ProNumber
	case shp.BOL != "":
		return shp.BOL
	default:
		return "this shipment"
	}
}

func userName(user *tenant.User) string {
	if user == nil || user.Name == "" {
		return "a biller"
	}

	return user.Name
}
