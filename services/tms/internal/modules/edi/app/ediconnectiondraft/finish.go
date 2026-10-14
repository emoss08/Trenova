package ediconnectiondraft

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	defaultMappingKey  = "DEFAULT"
	mappingRecordLabel = "Mapping"
)

type mappingKey struct {
	entityType edi.MappingEntityType
	sourceID   pulid.ID
}

type mappingCandidate struct {
	input  ediports.DraftMappingInput
	source ediports.DraftSource
	status ediports.ValueStatus
	reason string
}

func (p *planner) buildMappings(ctx context.Context) error {
	targets := make(map[edi.MappingEntityType][]pulid.ID)
	for _, input := range p.req.Mappings {
		if input.TargetID.IsNotNil() && !slices.Contains(targets[input.EntityType], input.TargetID) {
			targets[input.EntityType] = append(targets[input.EntityType], input.TargetID)
		}
	}
	labels := map[edi.MappingEntityType]map[pulid.ID]string{}
	if len(targets) > 0 {
		var err error
		labels, err = p.s.repo.ResolveMappingTargets(ctx, &ediports.ResolveEDIMappingTargetsRequest{
			TenantInfo: p.req.TenantInfo,
			Targets:    targets,
		})
		if err != nil {
			return err
		}
	}

	grouped := make(map[mappingKey][]mappingCandidate)
	order := make([]mappingKey, 0, len(p.req.Mappings))
	for _, input := range p.req.Mappings {
		key := mappingKey{entityType: input.EntityType, sourceID: edi.MappingSourceID(input.SourceValue)}
		if _, seen := grouped[key]; !seen {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], p.readMapping(input, labels))
	}

	for _, key := range order {
		p.draft.Mappings = append(p.draft.Mappings, p.settleMapping(key, grouped[key], labels))
	}
	if p.current.setup != nil {
		for _, item := range p.current.setup.MappingItems {
			key := mappingKey{entityType: item.EntityType, sourceID: item.SourceID}
			if _, mentioned := grouped[key]; mentioned {
				continue
			}
			p.draft.Kept = append(p.draft.Kept, ediports.DraftKept{
				Record: mappingRecordLabel,
				Label: fmt.Sprintf(
					"%s %s → %s is not in this spec; kept",
					item.EntityType,
					item.SourceID,
					firstNonEmpty(item.TargetLabel, item.TargetID.String()),
				),
			})
		}
	}
	return nil
}

func (p *planner) readMapping(
	input ediports.DraftMappingInput,
	labels map[edi.MappingEntityType]map[pulid.ID]string,
) mappingCandidate {
	check := p.pages.locate(input.Citation)
	page := check.page
	if page == 0 {
		page = input.Citation.Page
	}
	candidate := mappingCandidate{
		input:  input,
		source: documentSource(page, input.Citation, check.problem),
		status: ediports.ValueStatusAccepted,
	}
	_, targetFound := labels[input.EntityType][input.TargetID]
	switch {
	case !check.ok:
		candidate.status = ediports.ValueStatusUnverified
		candidate.reason = check.problem
	case strings.TrimSpace(input.SourceValue) == "":
		candidate.status = ediports.ValueStatusInvalid
		candidate.reason = "The partner's code is blank"
	case !containsToken(normalizeForMatch(input.Citation.Quote), normalizeForMatch(input.SourceValue)):
		candidate.status = ediports.ValueStatusUnverified
		candidate.reason = "The quote does not contain the partner's code"
	case input.TargetID.IsNil():
		candidate.status = ediports.ValueStatusMissing
		candidate.reason = "Name the Trenova record this code maps to"
	case !targetFound:
		candidate.status = ediports.ValueStatusInvalid
		candidate.reason = fmt.Sprintf("No %s with that ID exists in this organization", input.EntityType)
	case input.Confidence < confidenceFloor:
		candidate.status = ediports.ValueStatusHeld
		candidate.reason = fmt.Sprintf("Confidence %.2f is below %.2f", input.Confidence, confidenceFloor)
	}
	return candidate
}

func (p *planner) settleMapping(
	key mappingKey,
	candidates []mappingCandidate,
	labels map[edi.MappingEntityType]map[pulid.ID]string,
) ediports.DraftMapping {
	first := candidates[0]
	out := ediports.DraftMapping{
		EntityType:  key.entityType,
		SourceID:    key.sourceID,
		SourceLabel: strings.TrimSpace(first.input.SourceLabel),
		Status:      first.status,
		Confidence:  first.input.Confidence,
		Source:      first.source,
		Reason:      first.reason,
		Change:      ediports.ChangeKindNone,
	}
	targetsSeen := make([]pulid.ID, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.status == ediports.ValueStatusAccepted &&
			!slices.Contains(targetsSeen, candidate.input.TargetID) {
			targetsSeen = append(targetsSeen, candidate.input.TargetID)
		}
		if candidate.status == ediports.ValueStatusAccepted && out.Status != ediports.ValueStatusAccepted {
			out.Status = candidate.status
			out.Source = candidate.source
			out.Confidence = candidate.input.Confidence
			out.Reason = ""
		}
	}
	if len(targetsSeen) > 1 {
		out.Status = ediports.ValueStatusHeld
		out.Reason = "The document maps this code to more than one record"
		targetsSeen = nil
	}

	current := p.current.mapping(key.entityType, key.sourceID)
	if current != nil {
		out.CurrentTargetID = current.TargetID
	}
	if out.Status != ediports.ValueStatusAccepted || len(targetsSeen) == 0 {
		return out
	}

	out.TargetID = targetsSeen[0]
	out.TargetLabel = labels[key.entityType][out.TargetID]
	switch {
	case current == nil:
		out.Change = ediports.ChangeKindAdd
	case current.TargetID == out.TargetID:
		out.Change = ediports.ChangeKindSame
	default:
		out.Change = ediports.ChangeKindChange
	}
	if out.Change.Writes() {
		p.writes.mappings = append(p.writes.mappings, &edi.EDIMappingProfileItem{
			EntityType:  key.entityType,
			SourceID:    key.sourceID,
			SourceLabel: out.SourceLabel,
			TargetID:    out.TargetID,
			TargetLabel: out.TargetLabel,
		})
	}
	return out
}

func (p *planner) buildGaps() {
	for _, input := range p.req.Gaps {
		check := p.pages.locate(input.Citation)
		page := check.page
		if page == 0 {
			page = input.Citation.Page
		}
		gap := ediports.DraftGap{
			Kind:    input.Kind,
			Summary: clampQuote(redact(strings.TrimSpace(input.Summary))),
			Source:  documentSource(page, input.Citation, ""),
		}
		if !check.ok {
			gap.Problem = check.problem
		}
		p.draft.Gaps = append(p.draft.Gaps, gap)
	}
}

func (p *planner) buildSamples() {
	mapped := p.draftedMappings()
	for _, sample := range p.samples {
		out := ediports.DraftSample{
			Page:           sample.page,
			TransactionSet: sample.transactionSet(),
			Direction:      sample.direction,
			Parsed:         sample.err == nil,
		}
		if sample.err != nil {
			out.Error = redact(sample.err.Error())
			p.draft.Samples = append(p.draft.Samples, out)
			continue
		}
		overview := sample.parsed.Inspection
		overview.Diagnostics = slices.Clone(overview.Diagnostics)
		for index := range overview.Diagnostics {
			overview.Diagnostics[index].Message = redact(overview.Diagnostics[index].Message)
		}
		out.Overview = &overview
		if sample.identifying {
			out.Checks = p.sampleChecks(sample)
		}
		out.Unmapped = unmappedCodes(sample, mapped)
		p.draft.Samples = append(p.draft.Samples, out)
	}
}

func (p *planner) sampleChecks(sample *specSample) []ediports.DraftSampleCheck {
	partnerQualifier, partnerID, ourQualifier, ourID, partnerGS, ourGS := sample.parties()
	found := map[string]string{
		fieldPartnerQualifier: partnerQualifier,
		fieldPartnerID:        partnerID,
		fieldOurQualifier:     ourQualifier,
		fieldOurID:            ourID,
		fieldPartnerGSID:      partnerGS,
		fieldOurGSID:          ourGS,
		fieldUsageIndicator:   sample.parsed.UsageIndicator,
		fieldX12Version:       sample.parsed.GroupVersion,
		fieldElementSep:       sample.parsed.Separators.Element,
		fieldSegmentSep:       sample.parsed.Separators.Segment,
		fieldComponentSep:     sample.parsed.Separators.Component,
	}
	checks := make([]ediports.DraftSampleCheck, 0, len(found))
	for _, key := range sampleFieldOrder {
		value, ok := found[key]
		if !ok || value == "" {
			continue
		}
		spec := fieldCatalog[key]
		if normalized, err := spec.normalize(value); err == nil {
			value = normalized
		}
		expected, _ := p.usable(key)
		checks = append(checks, ediports.DraftSampleCheck{
			Field:    key,
			Label:    spec.label,
			Expected: expected,
			Found:    value,
			Matches:  expected != "" && expected == value,
		})
	}
	if set := sample.transactionSet(); set != "" && slices.Contains(supportedSets, set) {
		group := setField(set, sample.direction, setAttrFunctionalGroupID)
		if expected, ok := p.usable(group); ok && sample.parsed.FunctionalGroupID != "" {
			checks = append(checks, ediports.DraftSampleCheck{
				Field:    group,
				Label:    fieldCatalog[group].label,
				Expected: expected,
				Found:    sample.parsed.FunctionalGroupID,
				Matches:  expected == sample.parsed.FunctionalGroupID,
			})
		}
	}
	return checks
}

func (p *planner) draftedMappings() map[mappingKey]bool {
	mapped := make(map[mappingKey]bool)
	if p.current.setup != nil {
		for _, item := range p.current.setup.MappingItems {
			mapped[mappingKey{entityType: item.EntityType, sourceID: item.SourceID}] = true
		}
	}
	for _, mapping := range p.draft.Mappings {
		if mapping.Status == ediports.ValueStatusAccepted && mapping.TargetID.IsNotNil() {
			mapped[mappingKey{entityType: mapping.EntityType, sourceID: mapping.SourceID}] = true
		}
	}
	return mapped
}

func unmappedCodes(sample *specSample, mapped map[mappingKey]bool) []ediports.DraftUnmappedCode {
	out := make([]ediports.DraftUnmappedCode, 0)
	for _, transaction := range sample.parsed.Transactions {
		tender := transaction.Payload.LoadTender
		if tender == nil {
			continue
		}
		entityTypes := make([]edi.MappingEntityType, 0, len(tender.RequiredMappingEntityIDs))
		for entityType := range tender.RequiredMappingEntityIDs {
			entityTypes = append(entityTypes, entityType)
		}
		slices.Sort(entityTypes)
		for _, entityType := range entityTypes {
			ids := slices.Clone(tender.RequiredMappingEntityIDs[entityType])
			slices.Sort(ids)
			for _, id := range ids {
				key := mappingKey{entityType: entityType, sourceID: id}
				if mapped[key] || slices.ContainsFunc(out, func(code ediports.DraftUnmappedCode) bool {
					return code.EntityType == entityType && code.SourceID == id
				}) {
					continue
				}
				code := ediports.DraftUnmappedCode{EntityType: entityType, SourceID: id}
				if id == defaultMappingKey {
					code.SourceLabel = "Default for every inbound tender"
				}
				out = append(out, code)
			}
		}
	}
	return out
}

func (p *planner) settle() {
	draft := p.draft
	sections := [][]ediports.DraftValue{draft.Partner.Fields, draft.Interchange}
	if draft.Transport != nil {
		sections = append(sections, draft.Transport.Fields)
	}
	for _, profile := range draft.DocumentProfiles {
		sections = append(sections, profile.Fields)
	}
	for _, key := range sortedSetSupportKeys() {
		if value, ok := p.values[key]; ok && !value.Status.Usable() {
			sections = append(sections, []ediports.DraftValue{value})
		}
	}
	for _, kind := range controlNumberKinds {
		if value, ok := p.values[controlNumberField(kind)]; ok && !value.Status.Usable() {
			sections = append(sections, []ediports.DraftValue{value})
		}
	}
	if value, ok := p.values[fieldTransportMethod]; ok && draft.Transport == nil {
		sections = append(sections, []ediports.DraftValue{value})
	}

	seen := make(map[string]bool)
	draft.Decisions = make([]ediports.DraftValue, 0)
	for _, section := range sections {
		for _, value := range section {
			if seen[value.Field] || !needsDecision(value) {
				continue
			}
			seen[value.Field] = true
			draft.Decisions = append(draft.Decisions, value)
		}
	}

	draft.Blockers = make([]string, 0)
	if draft.Partner.Match == ediports.PartnerMatchAmbiguous {
		draft.Blockers = append(draft.Blockers, "Name which existing partner this spec is for")
	}
	for _, decision := range draft.Decisions {
		draft.Blockers = append(draft.Blockers, "Decide "+lowerFirst(decision.Label))
	}
	for _, mapping := range draft.Mappings {
		if mapping.Status != ediports.ValueStatusAccepted {
			draft.Blockers = append(draft.Blockers, fmt.Sprintf(
				"Settle the %s mapping for %s",
				mapping.EntityType,
				mapping.SourceID,
			))
		}
	}
	if len(draft.Failures) > 0 {
		draft.Blockers = append(draft.Blockers, fmt.Sprintf(
			"Fix the %d validation %s",
			len(draft.Failures),
			plural(len(draft.Failures), "failure", "failures"),
		))
	}
	if len(draft.DocumentProfiles) == 0 {
		draft.Blockers = append(
			draft.Blockers,
			"The document names no 204, 210, 214 or 990 that Trenova would exchange",
		)
	}

	changes := 0
	if draft.Partner.Change.Writes() {
		changes++
	}
	if draft.Transport != nil && draft.Transport.Change.Writes() {
		changes++
	}
	for _, profile := range draft.DocumentProfiles {
		if profile.Change.Writes() {
			changes++
		}
		for _, control := range profile.ControlNumbers {
			if control.Change.Writes() {
				changes++
			}
		}
	}
	for _, mapping := range draft.Mappings {
		if mapping.Change.Writes() {
			changes++
		}
	}
	draft.Changes = changes
	if changes == 0 && len(draft.Blockers) == 0 {
		draft.Blockers = append(draft.Blockers, "Nothing would change: the partner already matches this spec")
	}
	draft.Ready = len(draft.Blockers) == 0
	if draft.Samples == nil {
		draft.Samples = []ediports.DraftSample{}
	}
	if draft.Gaps == nil {
		draft.Gaps = []ediports.DraftGap{}
	}
	if draft.Mappings == nil {
		draft.Mappings = []ediports.DraftMapping{}
	}
	if draft.Failures == nil {
		draft.Failures = []ediports.DraftFailure{}
	}
	if draft.DocumentProfiles == nil {
		draft.DocumentProfiles = []ediports.DraftDocumentProfile{}
	}
}

func needsDecision(value ediports.DraftValue) bool {
	switch value.Status {
	case ediports.ValueStatusHeld, ediports.ValueStatusUnverified, ediports.ValueStatusInvalid:
		return true
	case ediports.ValueStatusMissing:
		return value.Required
	default:
		return false
	}
}

func sortedSetSupportKeys() []string {
	keys := make([]string, 0, len(supportedSets)*len(directions))
	for _, set := range supportedSets {
		for _, direction := range directions {
			keys = append(keys, setField(set, direction, setAttrSupported))
		}
	}
	return keys
}

func lowerFirst(value string) string {
	if value == "" {
		return value
	}
	if len(value) > 1 && strings.ToUpper(value[:2]) == value[:2] {
		return value
	}
	return strings.ToLower(value[:1]) + value[1:]
}

func plural(count int, one, many string) string {
	if count == 1 {
		return one
	}
	return many
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
