package ediconnectiondraft

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	keptProfileLabel = "The document says this set is not exchanged; the profile is kept as it is"
	keptTemplateNote = "Kept: the profile's template has no active or certified version to check " +
		"against the spec's requirements"
)

func hasTemplate(options []templateOption, templateID pulid.ID) bool {
	for _, option := range options {
		if option.templateID == templateID {
			return true
		}
	}
	return false
}

func metCount(requirements []requirement, unmet []ediports.DraftRequirementCheck, note string) int {
	if note != "" {
		return 0
	}
	return len(requirements) - len(unmet)
}

func (p *planner) documentTypeID(
	ctx context.Context,
	set edi.TransactionSet,
	direction edi.DocumentDirection,
) (pulid.ID, error) {
	key := profileKey(set, direction)
	if id, ok := p.docTypes[key]; ok {
		return id, nil
	}
	types, err := p.s.documentTypes.ListDocumentTypes(ctx, ediports.ListEDIDocumentTypesRequest{
		Standard:       edi.EDIStandardX12,
		TransactionSet: set,
		Direction:      direction,
	})
	if err != nil {
		return pulid.Nil, err
	}
	if len(types) == 0 {
		return pulid.Nil, fmt.Errorf("x12 %s %s document type is not seeded", set, direction)
	}
	p.docTypes[key] = types[0].ID
	return types[0].ID, nil
}

func (p *planner) requirementsFor(
	set edi.TransactionSet,
	direction edi.DocumentDirection,
) ([]requirement, []ediports.DraftRequirementCheck) {
	accepted := make([]requirement, 0)
	unverified := make([]ediports.DraftRequirementCheck, 0)
	for inputIndex := range p.req.Requirements {
		input := &p.req.Requirements[inputIndex]
		if input.TransactionSet != set || input.Direction != direction {
			continue
		}
		check := p.pages.locate(input.Citation)
		page := check.page
		if page == 0 {
			page = input.Citation.Page
		}
		source := documentSource(page, input.Citation, check.problem)
		segmentID := strings.ToUpper(strings.TrimSpace(input.SegmentID))
		problem := check.problem
		switch {
		case !check.ok:
		case !containsToken(normalizeForMatch(input.Citation.Quote), strings.ToLower(segmentID)) &&
			!strings.Contains(strings.ToUpper(input.Citation.Quote), segmentID):
			problem = fmt.Sprintf("The quote does not mention %s", segmentID)
		default:
			accepted = append(accepted, requirement{input: input, source: source})
			continue
		}
		unverified = append(unverified, ediports.DraftRequirementCheck{
			SegmentID:       segmentID,
			ElementPosition: input.ElementPosition,
			Usage:           input.Usage,
			Codes:           input.Codes,
			Problem:         problem,
			Source:          source,
		})
	}
	return accepted, unverified
}

func (p *planner) buildProfiles(ctx context.Context) error {
	for _, set := range supportedSets {
		for _, direction := range directions {
			supported := p.values[setField(set, direction, setAttrSupported)]
			existing := p.current.profiles[profileKey(set, direction)]
			wanted := supported.Status.Usable() && supported.Value == "true"
			if !wanted && existing == nil {
				continue
			}
			if !wanted {
				if supported.Status.Usable() && supported.Value == "false" &&
					supported.Change != ediports.ChangeKindSame {
					p.draft.Kept = append(p.draft.Kept, ediports.DraftKept{
						Record: "Document profile " + existing.Name,
						Label:  keptProfileLabel,
					})
				}
				continue
			}
			if err := p.buildProfile(ctx, set, direction, existing); err != nil {
				return err
			}
		}
	}
	return nil
}

type profileFields struct {
	version ediports.DraftValue
	group   ediports.DraftValue
	ack     ediports.DraftValue
	window  ediports.DraftValue
	all     []ediports.DraftValue
}

func (p *planner) buildProfile(
	ctx context.Context,
	set edi.TransactionSet,
	direction edi.DocumentDirection,
	existing *edi.EDIPartnerDocumentProfile,
) error {
	documentTypeID, err := p.documentTypeID(ctx, set, direction)
	if err != nil {
		return err
	}

	fields := p.resolveProfileFields(set, direction)
	request := p.profileRequest(set, direction, existing)
	p.applyProfileFields(request, &fields, existing, direction)

	template, chosen, err := p.chooseProfileTemplate(ctx, set, direction, existing)
	if err != nil {
		return err
	}
	request.TemplateID = chosen.templateID
	request.TemplateVersionID = chosen.versionID
	if existing != nil && chosen.templateID == existing.TemplateID {
		request.TemplateVersionID = existing.TemplateVersionID
	}

	draft := ediports.DraftDocumentProfile{
		TransactionSet: set,
		Direction:      direction,
		DocumentTypeID: documentTypeID,
		ProfileID:      request.ProfileID,
		Version:        request.Version,
		Name:           request.Name,
		Status:         request.Status,
		Fields:         fields.all,
		Template:       template,
		Change:         profileChange(existing, request, template.Change),
	}
	if direction == edi.DocumentDirectionOutbound {
		draft.ControlNumbers = p.controlNumbers(set, direction, documentTypeID)
	}

	p.validateProfile(ctx, &draft, request)
	p.draft.DocumentProfiles = append(p.draft.DocumentProfiles, draft)
	p.writes.profiles = append(p.writes.profiles, &profileWrite{
		set:       set,
		direction: direction,
		request:   request,
		starter:   chosen.starter && chosen.templateID.IsNil(),
		change:    draft.Change,
	})
	return nil
}

func (p *planner) resolveProfileFields(
	set edi.TransactionSet,
	direction edi.DocumentDirection,
) profileFields {
	interchangeVersion, _ := p.usable(fieldX12Version)
	versionFallback := &fallback{
		value:  interchangeVersion,
		status: ediports.ValueStatusDefaulted,
		source: ediports.DraftSource{Kind: ediports.SourceKindDefault, Note: versionFromISA},
	}
	if source, ok := p.values[fieldX12Version]; ok && source.Status.Usable() {
		versionFallback.source = source.Source
		versionFallback.source.Note = joinNotes(source.Source.Note, versionFromISA)
	}

	fields := profileFields{
		version: p.resolve(setField(set, direction, setAttrVersion), versionFallback, true),
		group: p.resolve(setField(set, direction, setAttrFunctionalGroupID), &fallback{
			value:  edi.FunctionalGroupDefault(set),
			status: ediports.ValueStatusDefaulted,
			source: ediports.DraftSource{Kind: ediports.SourceKindDefault, Note: groupDefaultNote},
		}, true),
		ack:    p.resolve(setField(set, direction, setAttrAck), nil, false),
		window: p.resolve(setField(set, direction, setAttrAckWindowMinutes), nil, false),
	}
	fields.all = []ediports.DraftValue{
		p.values[setField(set, direction, setAttrSupported)],
		fields.version,
		fields.group,
		fields.ack,
		fields.window,
	}
	return fields
}

func (p *planner) profileRequest(
	set edi.TransactionSet,
	direction edi.DocumentDirection,
	existing *edi.EDIPartnerDocumentProfile,
) *ediservice.UpsertEDIPartnerDocumentProfileRequest {
	request := &ediservice.UpsertEDIPartnerDocumentProfileRequest{
		TenantInfo:     p.req.TenantInfo,
		EDIPartnerID:   p.match.partnerID(),
		Name:           profileName(p.partnerName(), set, direction),
		Status:         edi.DocumentStatusInactive,
		ValidationMode: edi.ValidationModeStrict,
		Envelope:       p.envelope(direction),
	}
	if existing == nil {
		return request
	}
	request.ProfileID = existing.ID
	request.Name = existing.Name
	request.Status = existing.Status
	request.ValidationMode = existing.ValidationMode
	request.PartnerSettings = maps.Clone(existing.PartnerSettings)
	request.PartnerSettingsSchemaID = existing.PartnerSettingsSchemaID
	request.PartnerSettingsSchemaVersion = existing.PartnerSettingsSchemaVersion
	request.Version = existing.Version
	request.Acknowledgment = existing.Acknowledgment
	request.X12VersionOverride = existing.X12VersionOverride
	request.FunctionalGroupID = existing.FunctionalGroupID
	return request
}

func writableValue(value *ediports.DraftValue, existing bool) (string, bool) {
	if existing && !value.Change.Writes() {
		return "", false
	}
	if !value.Status.Usable() || value.Value == "" {
		return "", false
	}
	return value.Value, true
}

func (p *planner) applyProfileFields(
	request *ediservice.UpsertEDIPartnerDocumentProfileRequest,
	fields *profileFields,
	existing *edi.EDIPartnerDocumentProfile,
	direction edi.DocumentDirection,
) {
	isExisting := existing != nil
	if value, ok := writableValue(&fields.version, isExisting); ok {
		request.X12VersionOverride = value
	}
	if value, ok := writableValue(&fields.group, isExisting); ok {
		request.FunctionalGroupID = value
	}
	if value, ok := writableValue(&fields.ack, isExisting); ok {
		applyAcknowledgment(&request.Acknowledgment, value)
	}
	if value, ok := writableValue(&fields.window, isExisting); ok {
		minutes, _ := strconv.ParseInt(value, 10, 64)
		request.Acknowledgment.SLAInMinutes = minutes
	}
	if direction != edi.DocumentDirectionOutbound {
		return
	}
	scac := p.values[fieldCarrierSCAC]
	if value, ok := writableValue(&scac, isExisting); ok {
		request.PartnerSettings = withCarrierSCAC(request.PartnerSettings, value)
	}
}

func applyAcknowledgment(config *edi.AcknowledgmentConfig, value string) {
	config.Expected = value != ackNone
	config.Type = edi.AcknowledgmentType(value)
	if value == ackNone {
		config.Type = edi.AcknowledgmentTypeNone
	}
	if config.Expected && config.MissingAckSeverity == "" {
		config.MissingAckSeverity = edi.ValidationSeverityWarning
	}
}

func (p *planner) chooseProfileTemplate(
	ctx context.Context,
	set edi.TransactionSet,
	direction edi.DocumentDirection,
	existing *edi.EDIPartnerDocumentProfile,
) (ediports.DraftTemplateChoice, templateOption, error) {
	requirements, unverified := p.requirementsFor(set, direction)
	versions, err := p.s.repo.ListTemplateCandidates(
		ctx,
		&ediports.ListEDITemplateCandidatesRequest{
			TenantInfo:     p.req.TenantInfo,
			TransactionSet: set,
			Direction:      direction,
		},
	)
	if err != nil {
		return ediports.DraftTemplateChoice{}, templateOption{}, err
	}

	var currentTemplateID pulid.ID
	var currentTemplateName string
	if existing != nil {
		currentTemplateID = existing.TemplateID
		if existing.Template != nil {
			currentTemplateName = existing.Template.Name
		}
	}
	options := templateOptions(p.req.TenantInfo, set, direction, versions)
	chosen, unmet, considered := chooseTemplate(options, requirements, currentTemplateID)
	note := ""
	if existing != nil && !hasTemplate(options, existing.TemplateID) {
		chosen = templateOption{
			templateID: existing.TemplateID,
			versionID:  existing.TemplateVersionID,
			name:       firstNonEmpty(currentTemplateName, "Current template"),
		}
		unmet = nil
		note = keptTemplateNote
	}

	template := ediports.DraftTemplateChoice{
		TemplateID: chosen.templateID,
		VersionID:  chosen.versionID,
		Name:       chosen.name,
		Starter:    chosen.starter,
		Current:    currentTemplateName,
		Met:        metCount(requirements, unmet, note),
		Unmet:      unmet,
		Considered: considered,
		Unverified: unverified,
		Note:       note,
		Change:     ediports.ChangeKindAdd,
	}
	if existing != nil {
		template.Change = ediports.ChangeKindSame
		if chosen.templateID != existing.TemplateID {
			template.Change = ediports.ChangeKindChange
		}
	}
	return template, chosen, nil
}

func profileName(
	partnerName string,
	set edi.TransactionSet,
	direction edi.DocumentDirection,
) string {
	name := fmt.Sprintf("%s %s", set, strings.ToLower(string(direction)))
	if partnerName == "" {
		return name
	}
	return partnerName + " " + name
}

func withCarrierSCAC(settings map[string]any, scac string) map[string]any {
	if settings == nil {
		settings = map[string]any{}
	}
	carrier, _ := settings["carrier"].(map[string]any)
	carrier = maps.Clone(carrier)
	if carrier == nil {
		carrier = map[string]any{}
	}
	carrier["scac"] = scac
	settings["carrier"] = carrier
	return settings
}

func profileChange(
	existing *edi.EDIPartnerDocumentProfile,
	request *ediservice.UpsertEDIPartnerDocumentProfileRequest,
	templateChange ediports.ChangeKind,
) ediports.ChangeKind {
	if existing == nil {
		return ediports.ChangeKindAdd
	}
	same := existing.Envelope == request.Envelope &&
		existing.Acknowledgment == request.Acknowledgment &&
		existing.X12VersionOverride == request.X12VersionOverride &&
		existing.FunctionalGroupID == request.FunctionalGroupID &&
		nestedString(existing.PartnerSettings, "carrier", "scac") ==
			nestedString(request.PartnerSettings, "carrier", "scac") &&
		templateChange == ediports.ChangeKindSame
	if same {
		return ediports.ChangeKindSame
	}
	return ediports.ChangeKindChange
}

func (p *planner) controlNumbers(
	set edi.TransactionSet,
	direction edi.DocumentDirection,
	documentTypeID pulid.ID,
) []ediports.DraftControlNumber {
	out := make([]ediports.DraftControlNumber, 0, len(controlNumberKinds))
	for _, kind := range controlNumberKinds {
		value, ok := p.values[controlNumberField(kind)]
		if !ok || !value.Status.Usable() || value.Value == "" {
			continue
		}
		start, _ := strconv.ParseInt(value.Value, 10, 64)
		control := ediports.DraftControlNumber{
			Kind:   kind,
			Start:  start,
			Change: ediports.ChangeKindAdd,
			Source: value.Source,
		}
		if sequence := p.current.sequence(documentTypeID, kind); sequence != nil {
			control.Current = sequence.NextValue
			switch {
			case sequence.NextValue == start:
				control.Change = ediports.ChangeKindSame
			case sequence.NextValue > start:
				control.Change = ediports.ChangeKindSame
				control.Problem = fmt.Sprintf(
					"Kept at %d: starting at %d would reissue numbers the partner has already seen",
					sequence.NextValue,
					start,
				)
			default:
				control.Change = ediports.ChangeKindChange
			}
		}
		if control.Change.Writes() {
			p.writes.controls = append(p.writes.controls, controlWrite{
				set:       set,
				direction: direction,
				kind:      kind,
				next:      start,
			})
		}
		out = append(out, control)
	}
	return out
}

func (p *planner) validateProfile(
	ctx context.Context,
	draft *ediports.DraftDocumentProfile,
	request *ediservice.UpsertEDIPartnerDocumentProfileRequest,
) {
	record := "Document profile " + titleSet(draft.TransactionSet, draft.Direction)
	validation := *request
	if validation.EDIPartnerID.IsNil() {
		validation.EDIPartnerID = p.validateID
	}
	if multiErr := p.s.validator.ValidatePartnerDocumentProfileRequest(
		&validation,
	); multiErr != nil {
		p.addFailures(record, multiErr.Errors)
	}
	if len(request.PartnerSettings) == 0 {
		return
	}
	version := request.X12VersionOverride
	diagnostics, err := p.s.setup.ValidatePartnerSettings(
		ctx,
		&ediservice.ValidatePartnerSettingsRequest{
			TenantInfo:                   p.req.TenantInfo,
			PartnerDocumentProfileID:     request.ProfileID,
			PartnerSettingsSchemaID:      request.PartnerSettingsSchemaID,
			PartnerSettingsSchemaVersion: request.PartnerSettingsSchemaVersion,
			DocumentTypeID:               draft.DocumentTypeID,
			Standard:                     edi.EDIStandardX12,
			TransactionSet:               draft.TransactionSet,
			Direction:                    draft.Direction,
			X12Version:                   version,
			Settings:                     request.PartnerSettings,
		},
	)
	if err != nil {
		p.draft.Failures = append(p.draft.Failures, ediports.DraftFailure{
			Record:  record,
			Field:   "partnerSettings",
			Message: err.Error(),
		})
		return
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != edi.ValidationSeverityError {
			continue
		}
		p.draft.Failures = append(p.draft.Failures, ediports.DraftFailure{
			Record:  record,
			Field:   "partnerSettings." + diagnostic.Path,
			Message: diagnostic.Message,
		})
	}
}
