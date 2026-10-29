package ediconnectiondraft

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediconnectionfield"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/x12field"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
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
	input  *ediconnectiondraftports.DraftMappingInput
	source ediconnectiondraftports.DraftSource
	status ediconnectiondraftports.ValueStatus
	reason string
}

func (p *planner) buildMappings(ctx context.Context) error {
	targets := make(map[edi.MappingEntityType][]pulid.ID)
	for _, input := range p.req.Mappings {
		if input.TargetID.IsNotNil() &&
			!slices.Contains(targets[input.EntityType], input.TargetID) {
			targets[input.EntityType] = append(targets[input.EntityType], input.TargetID)
		}
	}
	labels := map[edi.MappingEntityType]map[pulid.ID]string{}
	if len(targets) > 0 {
		var err error
		labels, err = p.s.repo.ResolveMappingTargets(
			ctx,
			&ediconnectiondraftports.ResolveEDIMappingTargetsRequest{
				TenantInfo: p.req.TenantInfo,
				Targets:    targets,
			},
		)
		if err != nil {
			return err
		}
	}

	grouped := make(map[mappingKey][]mappingCandidate)
	order := make([]mappingKey, 0, len(p.req.Mappings))
	for inputIndex := range p.req.Mappings {
		input := &p.req.Mappings[inputIndex]
		key := mappingKey{
			entityType: input.EntityType,
			sourceID:   edi.MappingSourceID(input.SourceValue),
		}
		if _, seen := grouped[key]; !seen {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], p.readMapping(input, labels))
	}

	for _, key := range order {
		p.draft.Mappings = append(p.draft.Mappings, p.settleMapping(key, grouped[key], labels))
	}
	if p.current.Setup != nil {
		for _, item := range p.current.Setup.MappingItems {
			key := mappingKey{entityType: item.EntityType, sourceID: item.SourceID}
			if _, mentioned := grouped[key]; mentioned {
				continue
			}
			p.draft.Kept = append(p.draft.Kept, ediconnectiondraftports.DraftKept{
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
	input *ediconnectiondraftports.DraftMappingInput,
	labels map[edi.MappingEntityType]map[pulid.ID]string,
) mappingCandidate {
	check := p.pages.Locate(input.Citation)
	page := check.Page
	if page == 0 {
		page = input.Citation.Page
	}
	candidate := mappingCandidate{
		input:  input,
		source: ediconnectionfield.DocumentSource(page, input.Citation, check.Problem),
		status: ediconnectiondraftports.ValueStatusAccepted,
	}
	_, targetFound := labels[input.EntityType][input.TargetID]
	switch {
	case !check.OK:
		candidate.status = ediconnectiondraftports.ValueStatusUnverified
		candidate.reason = check.Problem
	case strings.TrimSpace(input.SourceValue) == "":
		candidate.status = ediconnectiondraftports.ValueStatusInvalid
		candidate.reason = "The partner's code is blank"
	case !ediconnectionfield.ContainsToken(ediconnectionfield.NormalizeForMatch(input.Citation.Quote), ediconnectionfield.NormalizeForMatch(input.SourceValue)):
		candidate.status = ediconnectiondraftports.ValueStatusUnverified
		candidate.reason = "The quote does not contain the partner's code"
	case input.TargetID.IsNil():
		candidate.status = ediconnectiondraftports.ValueStatusMissing
		candidate.reason = "Name the Trenova record this code maps to"
	case !targetFound:
		candidate.status = ediconnectiondraftports.ValueStatusInvalid
		candidate.reason = fmt.Sprintf(
			"No %s with that ID exists in this organization",
			input.EntityType,
		)
	case input.Confidence < ediconnectionfield.ConfidenceFloor:
		candidate.status = ediconnectiondraftports.ValueStatusHeld
		candidate.reason = fmt.Sprintf(
			"Confidence %.2f is below %.2f",
			input.Confidence,
			ediconnectionfield.ConfidenceFloor,
		)
	}
	return candidate
}

func (p *planner) settleMapping(
	key mappingKey,
	candidates []mappingCandidate,
	labels map[edi.MappingEntityType]map[pulid.ID]string,
) ediconnectiondraftports.DraftMapping {
	first := candidates[0]
	out := ediconnectiondraftports.DraftMapping{
		EntityType:  key.entityType,
		SourceID:    key.sourceID,
		SourceLabel: strings.TrimSpace(first.input.SourceLabel),
		Status:      first.status,
		Confidence:  first.input.Confidence,
		Source:      first.source,
		Reason:      first.reason,
		Change:      ediconnectiondraftports.ChangeKindNone,
	}
	targetsSeen := make([]pulid.ID, 0, len(candidates))
	for candidateIndex := range candidates {
		candidate := &candidates[candidateIndex]
		if candidate.status == ediconnectiondraftports.ValueStatusAccepted &&
			!slices.Contains(targetsSeen, candidate.input.TargetID) {
			targetsSeen = append(targetsSeen, candidate.input.TargetID)
		}
		if candidate.status == ediconnectiondraftports.ValueStatusAccepted &&
			out.Status != ediconnectiondraftports.ValueStatusAccepted {
			out.Status = candidate.status
			out.Source = candidate.source
			out.Confidence = candidate.input.Confidence
			out.Reason = ""
		}
	}
	if len(targetsSeen) > 1 {
		out.Status = ediconnectiondraftports.ValueStatusHeld
		out.Reason = "The document maps this code to more than one record"
		targetsSeen = nil
	}

	current := p.current.Mapping(key.entityType, key.sourceID)
	if current != nil {
		out.CurrentTargetID = current.TargetID
	}
	if out.Status != ediconnectiondraftports.ValueStatusAccepted || len(targetsSeen) == 0 {
		return out
	}

	out.TargetID = targetsSeen[0]
	out.TargetLabel = labels[key.entityType][out.TargetID]
	switch {
	case current == nil:
		out.Change = ediconnectiondraftports.ChangeKindAdd
	case current.TargetID == out.TargetID:
		out.Change = ediconnectiondraftports.ChangeKindSame
	default:
		out.Change = ediconnectiondraftports.ChangeKindChange
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
		check := p.pages.Locate(input.Citation)
		page := check.Page
		if page == 0 {
			page = input.Citation.Page
		}
		gap := ediconnectiondraftports.DraftGap{
			Kind: input.Kind,
			Summary: ediconnectionfield.ClampQuote(
				x12field.RedactSecrets(strings.TrimSpace(input.Summary)),
			),
			Source: ediconnectionfield.DocumentSource(page, input.Citation, ""),
		}
		if !check.OK {
			gap.Problem = check.Problem
		}
		p.draft.Gaps = append(p.draft.Gaps, gap)
	}
}

func (p *planner) buildSamples(ctx context.Context) {
	mapped := p.draftedMappings()
	for _, sample := range p.samples {
		out := ediconnectiondraftports.DraftSample{
			Page:           sample.Page,
			TransactionSet: sample.TransactionSet(),
			Direction:      sample.Direction,
			Parsed:         sample.Err == nil,
		}
		if sample.Err != nil {
			out.Error = x12field.RedactSecrets(sample.Err.Error())
			p.draft.Samples = append(p.draft.Samples, out)
			continue
		}
		overview := sample.Parsed.Inspection
		overview.Diagnostics = slices.Clone(overview.Diagnostics)
		for index := range overview.Diagnostics {
			overview.Diagnostics[index].Message = x12field.RedactSecrets(
				overview.Diagnostics[index].Message,
			)
		}
		out.Overview = &overview
		if sample.Identifying {
			out.Checks = p.sampleChecks(sample)
		}
		out.Unmapped = p.sampleUnmapped(ctx, sample, mapped)
		p.draft.Samples = append(p.draft.Samples, out)
	}
}

func (p *planner) sampleChecks(
	sample *ediconnectionfield.Sample,
) []ediconnectiondraftports.DraftSampleCheck {
	found := sample.EnvelopeValues()
	checks := make([]ediconnectiondraftports.DraftSampleCheck, 0, len(found))
	for _, key := range ediconnectionfield.SampleFieldOrder {
		value, ok := found[key]
		if !ok || value == "" {
			continue
		}
		spec := ediconnectionfield.Catalog[key]
		if normalized, err := spec.Normalize(value); err == nil {
			value = normalized
		}
		expected, _ := p.usable(key)
		checks = append(checks, ediconnectiondraftports.DraftSampleCheck{
			Field:    key,
			Label:    spec.Label,
			Expected: expected,
			Found:    value,
			Matches:  expected != "" && expected == value,
		})
	}
	return checks
}

func (p *planner) draftedMappings() map[mappingKey]bool {
	mapped := make(map[mappingKey]bool)
	if p.current.Setup != nil {
		for _, item := range p.current.Setup.MappingItems {
			mapped[mappingKey{entityType: item.EntityType, sourceID: item.SourceID}] = true
		}
	}
	for mappingIndex := range p.draft.Mappings {
		mapping := &p.draft.Mappings[mappingIndex]
		if mapping.Status == ediconnectiondraftports.ValueStatusAccepted &&
			mapping.TargetID.IsNotNil() {
			mapped[mappingKey{entityType: mapping.EntityType, sourceID: mapping.SourceID}] = true
		}
	}
	return mapped
}

func (p *planner) settle() {
	draft := p.draft
	draft.Decisions = p.decisions()
	draft.Blockers = blockers(draft)
	draft.Changes = countChanges(draft)
	if draft.Changes == 0 && len(draft.Blockers) == 0 {
		draft.Blockers = append(
			draft.Blockers,
			"Nothing would change: the partner already matches this spec",
		)
	}
	draft.Ready = len(draft.Blockers) == 0
	fillEmpty(draft)
}

func (p *planner) decisionSections() [][]ediconnectiondraftports.DraftValue {
	draft := p.draft
	sections := [][]ediconnectiondraftports.DraftValue{draft.Partner.Fields, draft.Interchange}
	if draft.Transport != nil {
		sections = append(sections, draft.Transport.Fields)
	}
	for index := range draft.Sets {
		sections = append(sections, draft.Sets[index].Fields)
	}
	loose := sortedSetSupportKeys()
	for _, kind := range ediconnectionfield.ControlNumberKinds {
		loose = append(loose, ediconnectionfield.ControlNumberField(kind))
	}
	if draft.Transport == nil {
		loose = append(loose, ediconnectionfield.TransportMethod)
	}
	for _, key := range loose {
		if value, ok := p.values[key]; ok && !value.Status.Usable() {
			sections = append(sections, []ediconnectiondraftports.DraftValue{value})
		}
	}
	return sections
}

func (p *planner) decisions() []ediconnectiondraftports.DraftValue {
	seen := make(map[string]bool)
	decisions := make([]ediconnectiondraftports.DraftValue, 0)
	for _, section := range p.decisionSections() {
		for index := range section {
			value := &section[index]
			if seen[value.Field] || !needsDecision(value) {
				continue
			}
			seen[value.Field] = true
			decisions = append(decisions, *value)
		}
	}
	return decisions
}

func blockers(draft *ediconnectiondraftports.ConnectionDraft) []string {
	out := make([]string, 0)
	if draft.Partner.Match == ediconnectiondraftports.PartnerMatchAmbiguous {
		out = append(out, "Name which existing partner this spec is for")
	}
	for index := range draft.Decisions {
		if draft.Decisions[index].Required {
			out = append(out, "Decide "+lowerFirst(draft.Decisions[index].Label))
		}
	}
	for index := range draft.Mappings {
		mapping := &draft.Mappings[index]
		if mapping.Status != ediconnectiondraftports.ValueStatusAccepted {
			out = append(out, fmt.Sprintf(
				"Settle the %s mapping for %s",
				mapping.EntityType,
				mapping.SourceID,
			))
		}
	}
	if len(draft.Failures) > 0 {
		out = append(out, fmt.Sprintf(
			"Fix the %d validation %s",
			len(draft.Failures),
			plural(len(draft.Failures), "failure", "failures"),
		))
	}
	if len(draft.Sets) == 0 {
		out = append(out, "The document names no 204, 210, 214 or 990 that Trenova would exchange")
	}
	return out
}

func countChanges(draft *ediconnectiondraftports.ConnectionDraft) int {
	changes := 0
	count := func(change ediconnectiondraftports.ChangeKind) {
		if change.Writes() {
			changes++
		}
	}
	count(draft.Partner.Change)
	if draft.Transport != nil {
		count(draft.Transport.Change)
	}
	if draft.Project != nil {
		count(draft.Project.Change)
	}
	for index := range draft.Sets {
		set := &draft.Sets[index]
		for controlIndex := range set.ControlNumbers {
			count(set.ControlNumbers[controlIndex].Change)
		}
	}
	for index := range draft.Mappings {
		count(draft.Mappings[index].Change)
	}
	return changes
}

func fillEmpty(draft *ediconnectiondraftports.ConnectionDraft) {
	if draft.Samples == nil {
		draft.Samples = []ediconnectiondraftports.DraftSample{}
	}
	if draft.Gaps == nil {
		draft.Gaps = []ediconnectiondraftports.DraftGap{}
	}
	if draft.Mappings == nil {
		draft.Mappings = []ediconnectiondraftports.DraftMapping{}
	}
	if draft.Failures == nil {
		draft.Failures = []ediconnectiondraftports.DraftFailure{}
	}
	if draft.Sets == nil {
		draft.Sets = []ediconnectiondraftports.DraftSet{}
	}
	if draft.Kept == nil {
		draft.Kept = []ediconnectiondraftports.DraftKept{}
	}
}

func needsDecision(value *ediconnectiondraftports.DraftValue) bool {
	switch value.Status {
	case ediconnectiondraftports.ValueStatusHeld,
		ediconnectiondraftports.ValueStatusUnverified,
		ediconnectiondraftports.ValueStatusInvalid:
		return true
	case ediconnectiondraftports.ValueStatusMissing:
		return value.Required
	case ediconnectiondraftports.ValueStatusAccepted,
		ediconnectiondraftports.ValueStatusDefaulted,
		ediconnectiondraftports.ValueStatusKept:
		return false
	default:
		return false
	}
}

func sortedSetSupportKeys() []string {
	keys := make(
		[]string,
		0,
		len(ediconnectionfield.SupportedSets)*len(ediconnectionfield.Directions),
	)
	for _, set := range ediconnectionfield.SupportedSets {
		for _, direction := range ediconnectionfield.Directions {
			keys = append(
				keys,
				ediconnectionfield.SetField(set, direction, ediconnectionfield.SetAttrSupported),
			)
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
