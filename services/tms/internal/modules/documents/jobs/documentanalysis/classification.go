package documentanalysis

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

func ClassifyDocumentWithControl(
	name, text string,
	control *documentcontrol.DocumentControl,
	features *DocumentFeatureSet,
	fingerprint *ProviderFingerprint,
) *ClassificationResult {
	if control == nil || !control.EnableAutoClassification {
		return &ClassificationResult{
			Kind:           KindOther,
			Confidence:     0,
			Signals:        []string{"auto classification disabled"},
			ReviewRequired: true,
			Source:         "disabled",
			Reason:         "automatic classification disabled by document controls",
		}
	}
	return classifyDocumentWithFeatures(name, text, features, fingerprint)
}

func classifyDocumentWithFeatures(
	name, text string,
	features *DocumentFeatureSet,
	fingerprint *ProviderFingerprint,
) *ClassificationResult {
	corpus := strings.ToLower(name + "\n" + text)

	candidates := []*ClassificationResult{
		scoreRateConfirmation(corpus, features, fingerprint),
		scoreBillOfLading(corpus, features, fingerprint),
		scoreProofOfDelivery(corpus, features, fingerprint),
	}

	best := &ClassificationResult{
		Kind:           KindOther,
		Confidence:     defaultLowConfidence,
		Signals:        []string{"no strong classification signals"},
		ReviewRequired: true,
		Source:         "deterministic",
		Reason:         "no strong document-kind evidence detected",
	}

	for _, candidate := range candidates {
		if candidate.Confidence > best.Confidence {
			best = candidate
		}
	}

	if best.Kind == KindOther || best.Confidence < classificationMinConfidence {
		return &ClassificationResult{
			Kind:                KindOther,
			Confidence:          ClampConfidence(best.Confidence),
			Signals:             best.Signals,
			ReviewRequired:      true,
			Source:              "deterministic",
			ProviderFingerprint: ProviderName(fingerprint),
			Reason:              best.Reason,
		}
	}

	best.Confidence = ClampConfidence(best.Confidence)
	best.ReviewRequired = best.Confidence < reviewRequiredConfidenceFloor
	if best.Source == "" {
		best.Source = "deterministic"
	}
	if best.ProviderFingerprint == "" {
		best.ProviderFingerprint = ProviderName(fingerprint)
	}

	return best
}

func scoreRateConfirmation(
	corpus string,
	features *DocumentFeatureSet,
	fingerprint *ProviderFingerprint,
) *ClassificationResult {
	score := 0.0
	signals := make([]string, 0, 10)

	if stringutils.ContainsAny(
		corpus,
		"rate confirmation",
		"load confirmation",
		"carrier load confirmation",
		"contract addendum and carrier load confirmation",
		"ratecon",
	) {
		score += 0.5
		signals = append(signals, "rate confirmation phrase")
	}
	if rateRegex.MatchString(corpus) {
		score += 0.15
		signals = append(signals, "rate amount")
	}
	if pickupRegex.MatchString(corpus) {
		score += 0.1
		signals = append(signals, "pickup details")
	}
	if deliveryRegex.MatchString(corpus) {
		score += 0.1
		signals = append(signals, "delivery details")
	}
	if equipmentRegex.MatchString(corpus) {
		score += 0.075
		signals = append(signals, "equipment type")
	}
	if referenceRegex.MatchString(corpus) {
		score += 0.075
		signals = append(signals, "reference number")
	}
	if stringutils.ContainsAny(
		corpus,
		"line haul",
		"flat rate",
		"fuel surcharge",
		"quick pay",
		"cash advance",
	) {
		score += 0.15
		signals = append(signals, "carrier rate terms")
	}
	if stringutils.ContainsAny(corpus, "service for load #", "load #", "carrier load number") {
		score += 0.1
		signals = append(signals, "load number")
	}
	if stringutils.ContainsAny(
		corpus,
		"load confirmation is subject to the terms",
		"this load confirmation is",
	) {
		score += 0.1
		signals = append(signals, "load confirmation terms")
	}
	if len(features.MoneySignals) > 0 {
		score += 0.1
		signals = append(signals, "money signals")
	}
	if len(features.StopSignals) > 0 {
		score += 0.05
		signals = append(signals, "stop signals")
	}
	if fingerprint != nil && strings.EqualFold(fingerprint.KindHint, KindRateConfirmation) {
		score += fingerprint.Confidence * 0.2
		signals = append(signals, fingerprint.Signals...)
	}

	return &ClassificationResult{
		Kind:                KindRateConfirmation,
		Confidence:          score,
		Signals:             sliceutils.DedupeStrings(signals),
		Source:              "deterministic",
		ProviderFingerprint: ProviderName(fingerprint),
		Reason:              "rate and load-confirmation evidence detected",
	}
}

func scoreBillOfLading(
	corpus string,
	features *DocumentFeatureSet,
	fingerprint *ProviderFingerprint,
) *ClassificationResult {
	score := 0.0
	signals := make([]string, 0, 4)

	if stringutils.ContainsAny(corpus, "bill of lading", "straight bill") {
		score += 0.65
		signals = append(signals, "bill of lading phrase")
	}
	if stringutils.ContainsAny(corpus, "shipper", "consignee") {
		score += 0.05
		signals = append(signals, "shipper/consignee labels")
	}
	if stringutils.ContainsAny(corpus, "bol", "pickup number") {
		score += 0.1
		signals = append(signals, "bol reference")
	}
	if stringutils.ContainsAny(
		corpus,
		"rate confirmation",
		"load confirmation",
		"carrier load confirmation",
	) {
		score -= 0.25
	}
	if len(features.SignatureSignals) > 0 {
		score += 0.08
		signals = append(signals, "signature signals")
	}
	if fingerprint != nil && strings.EqualFold(fingerprint.KindHint, KindBillOfLading) {
		score += fingerprint.Confidence * 0.2
		signals = append(signals, fingerprint.Signals...)
	}

	return &ClassificationResult{
		Kind:                KindBillOfLading,
		Confidence:          score,
		Signals:             sliceutils.DedupeStrings(signals),
		Source:              "deterministic",
		ProviderFingerprint: ProviderName(fingerprint),
		Reason:              "bill-of-lading shipping evidence detected",
	}
}

func scoreProofOfDelivery(
	corpus string,
	features *DocumentFeatureSet,
	fingerprint *ProviderFingerprint,
) *ClassificationResult {
	score := 0.0
	signals := make([]string, 0, 4)

	if stringutils.ContainsAny(
		corpus,
		"proof of delivery",
		"delivery receipt",
		"received in good order",
	) {
		score += 0.7
		signals = append(signals, "proof of delivery phrase")
	}
	if stringutils.ContainsAny(corpus, "delivered", "consignee signature", "receiver signature") {
		score += 0.15
		signals = append(signals, "delivery confirmation language")
	}
	if len(features.SignatureSignals) > 0 {
		score += 0.08
		signals = append(signals, "signature signals")
	}
	if fingerprint != nil && strings.EqualFold(fingerprint.KindHint, KindProofOfDelivery) {
		score += fingerprint.Confidence * 0.2
		signals = append(signals, fingerprint.Signals...)
	}

	return &ClassificationResult{
		Kind:                KindProofOfDelivery,
		Confidence:          score,
		Signals:             sliceutils.DedupeStrings(signals),
		Source:              "deterministic",
		ProviderFingerprint: ProviderName(fingerprint),
		Reason:              "delivery completion evidence detected",
	}
}

func ExtractDocumentFeatures(
	name string,
	pages []*PageExtractionResult,
	text string,
) *DocumentFeatureSet {
	corpus := strings.ToLower(name + "\n" + text)
	lines := splitNormalizedLines(text)
	features := &DocumentFeatureSet{
		TitleCandidates:  make([]string, 0, 4),
		SectionLabels:    make([]string, 0, 12),
		PartyLabels:      make([]string, 0, 8),
		ReferenceLabels:  make([]string, 0, 8),
		MoneySignals:     make([]string, 0, 8),
		StopSignals:      make([]string, 0, 8),
		TermsSignals:     make([]string, 0, 8),
		SignatureSignals: make([]string, 0, 6),
	}

	for _, page := range pages {
		pageLines := splitNormalizedLines(page.Text)
		for i, line := range pageLines {
			if i < 3 && looksLikeTitle(line) {
				features.TitleCandidates = append(features.TitleCandidates, line)
			}
			recordLineFeatures(features, line)
		}
	}

	if len(features.TitleCandidates) == 0 {
		for i, line := range lines {
			if i >= 6 {
				break
			}
			if looksLikeTitle(line) {
				features.TitleCandidates = append(features.TitleCandidates, line)
			}
		}
	}

	if stringutils.ContainsAny(
		corpus,
		"line haul",
		"flat rate",
		"fuel surcharge",
		"amount due",
		"total due",
	) {
		features.MoneySignals = append(features.MoneySignals, "billing terms")
	}
	if stringutils.ContainsAny(corpus, "pickup", "delivery", "shipper", "receiver", "consignee") {
		features.StopSignals = append(features.StopSignals, "stop/party labels")
	}
	if stringutils.ContainsAny(corpus, "signature", "received in good order", "proof of delivery") {
		features.SignatureSignals = append(features.SignatureSignals, "signature language")
	}
	if stringutils.ContainsAny(
		corpus,
		"load confirmation",
		"subject to the terms",
		"contract addendum",
	) {
		features.TermsSignals = append(features.TermsSignals, "carrier contract terms")
	}

	features.TitleCandidates = sliceutils.DedupeStrings(features.TitleCandidates)
	features.SectionLabels = sliceutils.DedupeStrings(features.SectionLabels)
	features.PartyLabels = sliceutils.DedupeStrings(features.PartyLabels)
	features.ReferenceLabels = sliceutils.DedupeStrings(features.ReferenceLabels)
	features.MoneySignals = sliceutils.DedupeStrings(features.MoneySignals)
	features.StopSignals = sliceutils.DedupeStrings(features.StopSignals)
	features.TermsSignals = sliceutils.DedupeStrings(features.TermsSignals)
	features.SignatureSignals = sliceutils.DedupeStrings(features.SignatureSignals)

	return features
}

func recordLineFeatures(features *DocumentFeatureSet, line string) {
	normalized := strings.ToLower(strings.TrimSpace(line))
	if normalized == "" {
		return
	}
	if strings.Contains(normalized, ":") || strings.HasSuffix(normalized, "#") {
		if looksLikeSectionLabel(normalized) {
			features.SectionLabels = append(features.SectionLabels, normalized)
		}
	}
	switch {
	case stringutils.ContainsAny(normalized, "shipper", "consignee", "receiver", "bill to"):
		features.PartyLabels = append(features.PartyLabels, normalized)
	case stringutils.ContainsAny(normalized, "load #", "ref #", "reference", "confirmation", "invoice #", "bol"):
		features.ReferenceLabels = append(features.ReferenceLabels, normalized)
	case stringutils.ContainsAny(normalized, "rate", "line haul", "fuel surcharge", "amount due", "total"):
		features.MoneySignals = append(features.MoneySignals, normalized)
	case stringutils.ContainsAny(normalized, "pickup", "delivery", "scheduled delivery", "pick up date", "delivery date"):
		features.StopSignals = append(features.StopSignals, normalized)
	case stringutils.ContainsAny(normalized, "signature", "received", "proof of delivery"):
		features.SignatureSignals = append(features.SignatureSignals, normalized)
	case stringutils.ContainsAny(normalized, "load confirmation", "subject to the terms", "agreement", "contract addendum"):
		features.TermsSignals = append(features.TermsSignals, normalized)
	}
}

func DetectProviderFingerprint(
	name, text string,
	features *DocumentFeatureSet,
) *ProviderFingerprint {
	corpus := strings.ToLower(name + "\n" + text)
	registry := []ProviderFingerprint{
		{
			Provider:   "CHRobinson",
			KindHint:   KindRateConfirmation,
			Confidence: 0.95,
			Signals:    []string{"ch robinson fingerprint", "carrier load confirmation format"},
		},
		{
			Provider:   "TQL",
			KindHint:   KindRateConfirmation,
			Confidence: 0.9,
			Signals:    []string{"tql fingerprint"},
		},
		{
			Provider:   "Echo",
			KindHint:   KindRateConfirmation,
			Confidence: 0.9,
			Signals:    []string{"echo fingerprint"},
		},
		{
			Provider:   "UberFreight",
			KindHint:   KindRateConfirmation,
			Confidence: 0.9,
			Signals:    []string{"uber freight fingerprint"},
		},
	}

	for _, candidate := range registry {
		switch candidate.Provider {
		case "CHRobinson":
			if stringutils.ContainsAny(
				corpus,
				"c.h. robinson",
				"ch robinson",
				"navispherecarrier",
				"carrier load confirmation",
				"contract addendum and carrier load confirmation",
			) {
				return &candidate
			}
		case "TQL":
			if stringutils.ContainsAny(corpus, "tql", "total quality logistics") {
				return &candidate
			}
		case "Echo":
			if stringutils.ContainsAny(corpus, "echo global logistics", "echo logistics") {
				return &candidate
			}
		case "UberFreight":
			if stringutils.ContainsAny(corpus, "uber freight") {
				return &candidate
			}
		}
	}

	if len(features.TermsSignals) > 0 &&
		stringutils.ContainsAny(corpus, "load confirmation", "carrier load") {
		return &ProviderFingerprint{
			Provider:   "GenericBrokerLoadConfirmation",
			KindHint:   KindRateConfirmation,
			Confidence: 0.7,
			Signals:    []string{"generic broker load confirmation fingerprint"},
		}
	}

	return nil
}

func ProviderName(fingerprint *ProviderFingerprint) string {
	if fingerprint == nil {
		return ""
	}
	return fingerprint.Provider
}

func looksLikeTitle(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || len(trimmed) > 120 {
		return false
	}
	lower := strings.ToLower(trimmed)
	return stringutils.ContainsAny(lower,
		"rate confirmation",
		"load confirmation",
		"bill of lading",
		"proof of delivery",
		"invoice",
		"carrier load confirmation",
	)
}

func looksLikeSectionLabel(line string) bool {
	return stringutils.ContainsAny(line,
		"shipper",
		"receiver",
		"consignee",
		"pickup",
		"delivery",
		"rate",
		"reference",
		"commodity",
		"instructions",
		"invoice",
	)
}

func BuildStructuredData(
	intelligence *DocumentIntelligenceAnalysis,
	aiDiagnostics *AIDiagnostics,
) map[string]any {
	return map[string]any{
		"schemaVersion": 6,
		"intelligence":  intelligence.ToMap(),
		"aiDiagnostics": aiDiagnostics.ToMap(),
	}
}

func IsPlainTextType(contentType, ext string) bool {
	return strings.HasPrefix(contentType, "text/") ||
		ext == ".txt" ||
		ext == ".csv" ||
		ext == ".json" ||
		ext == ".xml" ||
		ext == ".html"
}

func IsPDFType(contentType, ext string) bool {
	return contentType == "application/pdf" || ext == ".pdf"
}
