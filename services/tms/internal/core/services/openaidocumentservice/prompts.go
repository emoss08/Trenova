package openaidocumentservice

import (
	"fmt"
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
)

func buildRoutePrompt(req *serviceports.AIRouteRequest) string {
	var b strings.Builder
	b.WriteString(
		"Classify this transportation document into one of: RateConfirmation, BillOfLading, ProofOfDelivery, Other.\n",
	)
	b.WriteString(
		"Use the extracted text, feature summary, and any provider fingerprint hint. Return strict JSON only.\n",
	)
	b.WriteString(
		"Set shouldExtract=true only when the documentKind is RateConfirmation and the evidence is strong enough for structured extraction.\n",
	)
	b.WriteString("Filename: " + strings.TrimSpace(req.FileName) + "\n")
	if req.Fingerprint != nil {
		fmt.Fprintf(&b,
			"Provider fingerprint hint: provider=%s kindHint=%s confidence=%.2f signals=%s\n",
			req.Fingerprint.Provider,
			req.Fingerprint.KindHint,
			req.Fingerprint.Confidence,
			strings.Join(req.Fingerprint.Signals, ", "),
		)
	}
	if req.Features != nil {
		b.WriteString("Normalized features:\n")
		fmt.Fprintf(&b, "Titles: %s\n", strings.Join(req.Features.TitleCandidates, " | "))
		fmt.Fprintf(&b, "Section labels: %s\n", strings.Join(req.Features.SectionLabels, " | "))
		fmt.Fprintf(&b, "Party labels: %s\n", strings.Join(req.Features.PartyLabels, " | "))
		fmt.Fprintf(&b, "Reference labels: %s\n", strings.Join(req.Features.ReferenceLabels, " | "))
		fmt.Fprintf(&b, "Money signals: %s\n", strings.Join(req.Features.MoneySignals, " | "))
		fmt.Fprintf(&b, "Stop signals: %s\n", strings.Join(req.Features.StopSignals, " | "))
		fmt.Fprintf(&b, "Terms signals: %s\n", strings.Join(req.Features.TermsSignals, " | "))
		fmt.Fprintf(
			&b,
			"Signature signals: %s\n",
			strings.Join(req.Features.SignatureSignals, " | "),
		)
	}
	b.WriteString("Document text excerpt:\n")
	b.WriteString(stringutils.Truncate(req.Text, 4000))
	b.WriteString("\nPage summaries:\n")
	for _, page := range req.Pages {
		fmt.Fprintf(&b, "Page %d: %s\n", page.PageNumber, stringutils.Truncate(page.Text, 800))
	}
	return b.String()
}

func buildExtractPrompt(req *serviceports.AIExtractRequest) string {
	var b strings.Builder
	b.WriteString("Extract structured rate confirmation data for a TMS.\n")
	b.WriteString(
		"Return only compact canonical fields and stop data needed for shipment creation/review.\n",
	)
	b.WriteString(
		"Do not emit extra broker-specific or descriptive fields beyond the canonical key set.\n",
	)
	b.WriteString(
		"Use page-local evidence. Keep evidence excerpts short and specific. Mark conflicts and low-confidence fields instead of guessing.\n",
	)
	b.WriteString(
		"Canonical field keys: loadNumber, referenceNumber, shipper, consignee, rate, equipmentType, commodity, pickupDate, deliveryDate, pickupWindow, deliveryWindow, pickupNumber, deliveryNumber, appointmentNumber, bol, poNumber, scac, proNumber, paymentTerms, billTo, carrierName, carrierContact, containerNumber, trailerNumber, tractorNumber, fuelSurcharge, serviceType.\n",
	)
	b.WriteString("Filename: " + strings.TrimSpace(req.FileName) + "\n")
	for _, page := range req.Pages {
		fmt.Fprintf(&b, "\n[Page %d]\n%s\n", page.PageNumber, stringutils.Truncate(page.Text, 2500))
	}
	return b.String()
}
