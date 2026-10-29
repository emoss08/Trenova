package ediconnectiondraft

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediconnectionfield"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediconnectionguide"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/ladingdoc"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	projectRecord    = "Lading project"
	keptSetLabel     = "The document says this set is not exchanged; the project keeps what it has for it"
	keptGuideLabel   = "The project's guide is kept as it is; add the document's requirements to it in the Lading studio"
	noMapSummary     = "The project has no map for %s; add one in the Lading studio"
	ackConflict      = "The document asks for %s; the project acknowledges every set with %s"
	draftNotes       = "Drafted from %s"
	maxGuideAttempts = 3
)

var (
	interchangeArea = regexp.MustCompile(`(?s)interchange\s*\{[^{}]*\}`)
	zoneLine        = regexp.MustCompile(`(?m)^zone\s+\S+[ \t]*$`)
	segmentID       = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,2}$`)
)

type exchange struct {
	set          edidocument.TransactionSet
	direction    edidocument.DocumentDirection
	index        int
	version      string
	ack          string
	window       int
	requirements []*ediconnectionguide.Requirement
}

type projectWrite struct {
	files   []lading.File
	samples []lading.Sample
	notes   string
	change  ediconnectiondraftports.ChangeKind
}

func (p *planner) documentTypeID(
	ctx context.Context,
	set edidocument.TransactionSet,
	direction edidocument.DocumentDirection,
) (pulid.ID, error) {
	key := ediconnectionfield.SetKey(set, direction)
	if id, ok := p.docTypes[key]; ok {
		return id, nil
	}
	types, err := p.s.documentTypes.ListDocumentTypes(
		ctx,
		edipartnerports.ListEDIDocumentTypesRequest{
			Standard:       edidocument.EDIStandardX12,
			TransactionSet: set,
			Direction:      direction,
		},
	)
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
	set edidocument.TransactionSet,
	direction edidocument.DocumentDirection,
) ([]*ediconnectionguide.Requirement, []ediconnectiondraftports.DraftRequirementCheck) {
	accepted := make([]*ediconnectionguide.Requirement, 0)
	unverified := make([]ediconnectiondraftports.DraftRequirementCheck, 0)
	for inputIndex := range p.req.Requirements {
		input := &p.req.Requirements[inputIndex]
		if input.TransactionSet != set || input.Direction != direction {
			continue
		}
		check := p.pages.Locate(input.Citation)
		page := check.Page
		if page == 0 {
			page = input.Citation.Page
		}
		source := ediconnectionfield.DocumentSource(page, input.Citation, check.Problem)
		segment := strings.ToUpper(strings.TrimSpace(input.SegmentID))
		problem := check.Problem
		switch {
		case !check.OK:
		case !segmentID.MatchString(segment):
			problem = fmt.Sprintf("%s is not an X12 segment ID", segment)
		case !ediconnectionfield.ContainsToken(ediconnectionfield.NormalizeForMatch(input.Citation.Quote), strings.ToLower(segment)) &&
			!strings.Contains(strings.ToUpper(input.Citation.Quote), segment):
			problem = fmt.Sprintf("The quote does not mention %s", segment)
		default:
			accepted = append(accepted, &ediconnectionguide.Requirement{
				Input:  input,
				Source: source,
				Ref:    ediconnectionguide.ElementRef(segment, input.ElementPosition),
			})
			continue
		}
		unverified = append(
			unverified,
			ediconnectionguide.RequirementCheck(input, segment, source, problem),
		)
	}
	return accepted, unverified
}

func (p *planner) buildSets(ctx context.Context) ([]*exchange, error) {
	exchanges := make([]*exchange, 0)
	for _, set := range ediconnectionfield.SupportedSets {
		for _, direction := range ediconnectionfield.Directions {
			supported := p.values[ediconnectionfield.SetField(set, direction, ediconnectionfield.SetAttrSupported)]
			current := p.current.Exchanged[ediconnectionfield.SetKey(set, direction)]
			wanted := supported.Status.Usable() && supported.Value == ediconnectionfield.ValueTrue
			if !wanted {
				if current && supported.Status.Usable() &&
					supported.Value == ediconnectionfield.ValueFalse &&
					supported.Change != ediconnectiondraftports.ChangeKindSame {
					p.draft.Kept = append(p.draft.Kept, ediconnectiondraftports.DraftKept{
						Record: projectRecord + " " + titleSet(set, direction),
						Label:  keptSetLabel,
					})
				}
				continue
			}
			built, err := p.buildSet(ctx, set, direction, current)
			if err != nil {
				return nil, err
			}
			exchanges = append(exchanges, built)
		}
	}
	return exchanges, nil
}

func (p *planner) buildSet(
	ctx context.Context,
	set edidocument.TransactionSet,
	direction edidocument.DocumentDirection,
	current bool,
) (*exchange, error) {
	documentTypeID, err := p.documentTypeID(ctx, set, direction)
	if err != nil {
		return nil, err
	}
	interchangeVersion, _ := p.usable(ediconnectionfield.X12Version)
	versionFallback := &ediconnectionfield.Fallback{
		Value:  interchangeVersion,
		Status: ediconnectiondraftports.ValueStatusDefaulted,
		Source: ediconnectiondraftports.DraftSource{
			Kind: ediconnectiondraftports.SourceKindDefault,
			Note: versionFromISA,
		},
	}
	if source, ok := p.values[ediconnectionfield.X12Version]; ok && source.Status.Usable() {
		versionFallback.Source = source.Source
		versionFallback.Source.Note = ediconnectionfield.JoinNotes(
			source.Source.Note,
			versionFromISA,
		)
	}
	version := p.resolve(
		ediconnectionfield.SetField(set, direction, ediconnectionfield.SetAttrVersion),
		versionFallback,
		true,
	)
	ack := p.resolve(
		ediconnectionfield.SetField(set, direction, ediconnectionfield.SetAttrAck),
		nil,
		false,
	)
	window := p.resolve(
		ediconnectionfield.SetField(set, direction, ediconnectionfield.SetAttrAckWindowMinutes),
		nil,
		false,
	)
	fields := []ediconnectiondraftports.DraftValue{
		p.values[ediconnectionfield.SetField(set, direction, ediconnectionfield.SetAttrSupported)],
		version,
		ack,
		window,
	}

	requirements, unverified := p.requirementsFor(set, direction)
	draft := ediconnectiondraftports.DraftSet{
		TransactionSet: set,
		Direction:      direction,
		DocumentTypeID: documentTypeID,
		Change:         ediconnectiondraftports.ChangeKindAdd,
		Fields:         fields,
		Requirements:   len(requirements),
		Unmet:          []ediconnectiondraftports.DraftRequirementCheck{},
		Unverified:     unverified,
	}
	if current {
		draft.Change = ediconnectiondraftports.ChangeKindSame
		for index := range fields {
			if fields[index].Change.Writes() {
				draft.Change = ediconnectiondraftports.ChangeKindChange
			}
		}
	}
	if direction == edidocument.DocumentDirectionOutbound {
		draft.ControlNumbers = p.controlNumbers(set, direction, documentTypeID)
	}
	p.draft.Sets = append(p.draft.Sets, draft)

	built := &exchange{
		set:          set,
		direction:    direction,
		index:        len(p.draft.Sets) - 1,
		requirements: requirements,
	}
	if value, ok := usableValue(&version); ok {
		built.version = value
	}
	if value, ok := usableValue(&ack); ok {
		built.ack = value
	}
	if value, ok := usableValue(&window); ok {
		built.window, _ = strconv.Atoi(value)
	}
	return built, nil
}

func usableValue(value *ediconnectiondraftports.DraftValue) (string, bool) {
	if !value.Status.Usable() || value.Value == "" {
		return "", false
	}
	return value.Value, true
}

func (p *planner) controlNumbers(
	set edidocument.TransactionSet,
	direction edidocument.DocumentDirection,
	documentTypeID pulid.ID,
) []ediconnectiondraftports.DraftControlNumber {
	out := make(
		[]ediconnectiondraftports.DraftControlNumber,
		0,
		len(ediconnectionfield.ControlNumberKinds),
	)
	for _, kind := range ediconnectionfield.ControlNumberKinds {
		value, ok := p.values[ediconnectionfield.ControlNumberField(kind)]
		if !ok || !value.Status.Usable() || value.Value == "" {
			continue
		}
		start, _ := strconv.ParseInt(value.Value, 10, 64)
		control := ediconnectiondraftports.DraftControlNumber{
			Kind:   kind,
			Start:  start,
			Change: ediconnectiondraftports.ChangeKindAdd,
			Source: value.Source,
		}
		if sequence := p.current.Sequence(documentTypeID, kind); sequence != nil {
			control.Current = sequence.NextValue
			switch {
			case sequence.NextValue == start:
				control.Change = ediconnectiondraftports.ChangeKindSame
			case sequence.NextValue > start:
				control.Change = ediconnectiondraftports.ChangeKindSame
				control.Problem = fmt.Sprintf(
					"Kept at %d: starting at %d would reissue numbers the partner has already seen",
					sequence.NextValue,
					start,
				)
			default:
				control.Change = ediconnectiondraftports.ChangeKindChange
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

func (p *planner) buildProject(ctx context.Context, exchanges []*exchange) error {
	if len(exchanges) == 0 {
		return nil
	}
	request := p.starterRequest(exchanges)
	starter, err := p.s.lading.Client().Starter(ctx, request)
	if err != nil {
		return err
	}

	files := starter.Files
	samples := append(slices.Clone(starter.Samples), p.specSamples()...)
	checked := samples
	change := ediconnectiondraftports.ChangeKindAdd
	project := &ediconnectiondraftports.DraftProject{Role: request.Role}
	var guided map[string]string
	if version := p.current.Version; version != nil {
		files, guided = p.adaptProject(version, starter.Files, exchanges)
		kept, keptErr := p.s.projects.Samples(ctx, p.req.TenantInfo, version)
		if keptErr != nil {
			return keptErr
		}
		samples = p.specSamples()
		checked = edilading.MergeSamples(kept, samples)
		project.ProjectID = version.ProjectID
	}
	name := ediconnectionguide.PartnerName(files)
	if name == "" {
		name = ediconnectionguide.PartnerName(starter.Files)
	}

	var check *lading.Check
	var guides []*ediconnectionguide.GuideFile
	for attempt := 1; ; attempt++ {
		guides = p.guides(name, request.Version, exchanges, guided)
		check, err = p.check(ctx, ediconnectionguide.WithGuides(files, guides), checked)
		if err != nil {
			return err
		}
		if attempt == maxGuideAttempts || !p.dropFailingGuideLines(check, guides, exchanges) {
			break
		}
	}
	files = ediconnectionguide.WithGuides(files, guides)

	p.unmet(check, exchanges)
	p.projectFailures(check, guides)
	project.Files = ediconnectionguide.ProjectFiles(check, files, p.current.Version)
	project.Errors = check.Errors
	project.Warnings = check.Warnings
	project.Diagnostics = ediconnectionguide.ProjectDiagnostics(check)
	if p.current.Version != nil {
		change = ediconnectiondraftports.ChangeKindChange
		if ediconnectionguide.SameFiles(p.current.Version.Files, files) {
			change = ediconnectiondraftports.ChangeKindSame
		}
	}
	project.Change = change
	p.draft.Project = project
	p.writes.project = &projectWrite{
		files:   files,
		samples: samples,
		notes:   fmt.Sprintf(draftNotes, p.req.Document.FileName),
		change:  change,
	}
	return nil
}

func (p *planner) starterRequest(exchanges []*exchange) *lading.StarterRequest {
	partner := p.plannedPartner()
	request := &lading.StarterRequest{
		Name:  firstNonEmpty(strings.ToLower(partner.Code), partner.Name),
		Label: firstNonEmpty(partner.Name, partner.Code),
		Role:  roleFor(exchanges),
		Zone:  partner.Timezone,
	}
	if version, ok := p.usable(ediconnectionfield.X12Version); ok {
		request.Version = version
	}
	request.Them = p.pair(ediconnectionfield.PartnerQualifier, ediconnectionfield.PartnerID)
	request.Us = p.pair(ediconnectionfield.OurQualifier, ediconnectionfield.OurID)
	request.Group = p.pair(ediconnectionfield.PartnerGSID, ediconnectionfield.OurGSID)
	separators := make([]string, 0, 4)
	for _, key := range []string{ediconnectionfield.ElementSep, ediconnectionfield.SegmentSep, ediconnectionfield.ComponentSep, ediconnectionfield.RepetitionSep} {
		if value, ok := p.usable(key); ok {
			separators = append(separators, value)
		}
	}
	if len(separators) == 4 {
		request.Separators = separators
	}
	request.Usage = ediconnectionfield.ValueTest
	if usage, ok := p.usable(ediconnectionfield.UsageIndicator); ok && usage == "P" {
		request.Usage = ediconnectionfield.ValueProduction
	}
	request.Acknowledge, request.AckWithin = p.acknowledgment(exchanges)
	for _, exchanged := range exchanges {
		if !slices.Contains(request.Sets, string(exchanged.set)) {
			request.Sets = append(request.Sets, string(exchanged.set))
		}
	}
	return request
}

func (p *planner) plannedPartner() *edi.EDIPartner {
	if p.writes.partner != nil {
		return p.writes.partner
	}
	return p.partnerEntity()
}

func (p *planner) pair(first, second string) *[2]string {
	left, leftOK := p.usable(first)
	right, rightOK := p.usable(second)
	if !leftOK || !rightOK {
		return nil
	}
	return &[2]string{left, right}
}

func roleFor(exchanges []*exchange) string {
	carrier, broker := 0, 0
	for _, exchanged := range exchanges {
		inbound := exchanged.direction == edidocument.DocumentDirectionInbound
		tender := exchanged.set == edidocument.TransactionSet204
		if inbound == tender {
			carrier++
		} else {
			broker++
		}
	}
	if broker > carrier {
		return lading.RoleBroker
	}
	return lading.RoleCarrier
}

func (p *planner) acknowledgment(exchanges []*exchange) (acknowledge string, window int) {
	counts := map[string]int{}
	for _, exchanged := range exchanges {
		if exchanged.ack != "" {
			counts[exchanged.ack]++
		}
		if exchanged.direction == edidocument.DocumentDirectionOutbound &&
			exchanged.window > window {
			window = exchanged.window
		}
	}
	if len(counts) == 0 {
		return "", window
	}
	asked := make([]string, 0, len(counts))
	for ack := range counts {
		asked = append(asked, ack)
	}
	slices.Sort(asked)
	chosen := asked[0]
	for _, ack := range asked {
		if counts[ack] > counts[chosen] {
			chosen = ack
		}
	}
	if len(asked) > 1 {
		p.draft.Gaps = append(p.draft.Gaps, ediconnectiondraftports.DraftGap{
			Kind:    ediconnectiondraftports.GapKindUnsupported,
			Summary: fmt.Sprintf(ackConflict, strings.Join(asked, " and "), chosen),
			Source: ediconnectiondraftports.DraftSource{
				Kind: ediconnectiondraftports.SourceKindDefault,
			},
		})
	}
	if chosen == ediconnectionfield.AckNone {
		return lading.AcknowledgeNone, window
	}
	return chosen, window
}

func (p *planner) adaptProject(
	version *ladingproject.LadingVersion,
	starter []lading.File,
	exchanges []*exchange,
) (files []lading.File, guided map[string]string) {
	files = slices.Clone(version.Files)
	var interchange, zone string
	for _, file := range starter {
		if ediconnectionguide.PartnerHeader.MatchString(file.Source) {
			interchange = interchangeArea.FindString(file.Source)
			zone = zoneLine.FindString(file.Source)
		}
	}
	for index := range files {
		source := files[index].Source
		if !ediconnectionguide.PartnerHeader.MatchString(source) {
			continue
		}
		if interchange != "" {
			source = interchangeArea.ReplaceAllLiteralString(source, interchange)
		}
		if zone != "" {
			source = zoneLine.ReplaceAllLiteralString(source, zone)
		}
		files[index].Source = source
	}

	manifest := &version.Manifest
	guided = map[string]string{}
	for _, guide := range manifest.Guides {
		if guide.Set != "" && guide.Path != "" &&
			guide.Path != ediconnectionguide.GuidePath(edidocument.TransactionSet(guide.Set)) {
			guided[guide.Set] = guide.Name
		}
	}
	for _, exchanged := range exchanges {
		if name, ok := guided[string(exchanged.set)]; ok {
			p.draft.Kept = append(p.draft.Kept, ediconnectiondraftports.DraftKept{
				Record: "Guide " + name,
				Label:  keptGuideLabel,
			})
		}
		if !ediconnectionguide.MapsSet(manifest, exchanged.set, exchanged.direction) {
			p.draft.Gaps = append(p.draft.Gaps, ediconnectiondraftports.DraftGap{
				Kind:    ediconnectiondraftports.GapKindUnsupported,
				Summary: fmt.Sprintf(noMapSummary, titleSet(exchanged.set, exchanged.direction)),
				Source: ediconnectiondraftports.DraftSource{
					Kind: ediconnectiondraftports.SourceKindConfiguration,
				},
			})
		}
	}
	return files, guided
}

func (p *planner) guides(
	name, version string,
	exchanges []*exchange,
	kept map[string]string,
) []*ediconnectionguide.GuideFile {
	bySet := map[edidocument.TransactionSet]*ediconnectionguide.GuideFile{}
	order := make([]edidocument.TransactionSet, 0)
	label := firstNonEmpty(p.plannedPartner().Name, name)
	for _, exchanged := range exchanges {
		if _, ok := kept[string(exchanged.set)]; ok {
			continue
		}
		guide, ok := bySet[exchanged.set]
		if !ok {
			base := firstNonEmpty(exchanged.version, version, "004010")
			guide = &ediconnectionguide.GuideFile{
				Set:  exchanged.set,
				Name: name + "." + string(exchanged.set),
				Path: ediconnectionguide.GuidePath(exchanged.set),
			}
			guide.File = lading.File{Path: guide.Path, Source: fmt.Sprintf(
				"guide %s from x12.%s.%s %s\n",
				guide.Name,
				base,
				exchanged.set,
				ediconnectionguide.QuoteLading(
					fmt.Sprintf("%s %s, from %s", label, exchanged.set, p.req.Document.FileName),
				),
			)}
			bySet[exchanged.set] = guide
			order = append(order, exchanged.set)
		}
		for _, item := range exchanged.requirements {
			line, found := ediconnectionguide.GuideRule(item.Input)
			if !found {
				continue
			}
			doc := fmt.Sprintf(
				"/// p.%d: %s\n",
				item.Source.Page,
				ediconnectionguide.OneLine(item.Source.Quote),
			)
			guide.File.Source += "\n" + doc
			guide.Lines = append(guide.Lines, ediconnectionguide.GuideLine{
				Requirement: item,
				Line:        strings.Count(guide.File.Source, "\n") + 1,
			})
			guide.File.Source += line + "\n"
		}
	}
	out := make([]*ediconnectionguide.GuideFile, 0, len(order))
	for _, set := range order {
		out = append(out, bySet[set])
	}
	return out
}

func (p *planner) check(
	ctx context.Context,
	files []lading.File,
	samples []lading.Sample,
) (*lading.Check, error) {
	ladingContext, err := p.s.lading.Context(ctx, p.req.TenantInfo)
	if err != nil {
		return nil, err
	}
	encoded, err := lading.Encode(&lading.Project{Files: files, Samples: samples}, ladingContext)
	if err != nil {
		return nil, err
	}
	check, err := p.s.lading.Client().Check(ctx, encoded)
	if err != nil {
		return nil, err
	}
	p.encoded = encoded
	p.manifest = check
	return check, nil
}

func (p *planner) dropFailingGuideLines(
	check *lading.Check,
	guides []*ediconnectionguide.GuideFile,
	exchanges []*exchange,
) bool {
	dropped := false
	for _, guide := range guides {
		checked := ediconnectionguide.CheckedFile(check, guide.Path)
		if checked == nil {
			continue
		}
		for index := range checked.Diagnostics {
			diagnostic := &checked.Diagnostics[index]
			if diagnostic.Severity != lading.SeverityError {
				continue
			}
			line := ediconnectionguide.LineAt(guide.File.Source, diagnostic.Range.From)
			for _, item := range guide.Lines {
				if item.Line != line {
					continue
				}
				p.unverify(exchanges, item.Requirement, diagnostic.Message.String())
				dropped = true
			}
		}
	}
	return dropped
}

func (p *planner) unverify(
	exchanges []*exchange,
	item *ediconnectionguide.Requirement,
	problem string,
) {
	for _, exchanged := range exchanges {
		index := slices.Index(exchanged.requirements, item)
		if index < 0 {
			continue
		}
		exchanged.requirements = slices.Delete(exchanged.requirements, index, index+1)
		set := &p.draft.Sets[exchanged.index]
		set.Requirements--
		set.Unverified = append(set.Unverified, ediconnectionguide.RequirementCheck(
			item.Input,
			strings.ToUpper(strings.TrimSpace(item.Input.SegmentID)),
			item.Source,
			problem,
		))
	}
}

func (p *planner) unmet(check *lading.Check, exchanges []*exchange) {
	unwritten := ediconnectionguide.UnwrittenBySet(check)
	for _, exchanged := range exchanges {
		if exchanged.direction != edidocument.DocumentDirectionOutbound {
			continue
		}
		names := unwritten[string(exchanged.set)]
		set := &p.draft.Sets[exchanged.index]
		for _, item := range exchanged.requirements {
			if item.Input.Usage != ediconnectiondraftports.RequirementUsageMandatory {
				continue
			}
			problem, ok := names[item.Ref]
			if !ok {
				continue
			}
			set.Unmet = append(set.Unmet, ediconnectionguide.RequirementCheck(
				item.Input,
				strings.ToUpper(strings.TrimSpace(item.Input.SegmentID)),
				item.Source,
				problem,
			))
		}
	}
}

func (p *planner) projectFailures(check *lading.Check, guides []*ediconnectionguide.GuideFile) {
	owned := map[string]bool{}
	for _, guide := range guides {
		owned[guide.Path] = true
	}
	for index := range check.Files {
		file := &check.Files[index]
		if file.Kind != "partner" && !owned[file.Path] && p.current.Version != nil {
			continue
		}
		for diagnosticIndex := range file.Diagnostics {
			diagnostic := &file.Diagnostics[diagnosticIndex]
			if diagnostic.Severity != lading.SeverityError ||
				diagnostic.Code == "x12-mandatory-unwritten" {
				continue
			}
			p.draft.Failures = append(p.draft.Failures, ediconnectiondraftports.DraftFailure{
				Record:  projectRecord,
				Field:   file.Path,
				Message: diagnostic.Message.String(),
			})
		}
	}
}

// specSamples are the spec's sample interchanges, kept with the project for its tests.
func (p *planner) specSamples() []lading.Sample {
	samples := make([]lading.Sample, 0, len(p.samples))
	for _, sample := range p.samples {
		if sample.Err != nil || strings.TrimSpace(sample.Raw) == "" {
			continue
		}
		samples = append(samples, lading.Sample{
			Path:    fmt.Sprintf("samples/spec-page-%d.x12", sample.Page),
			Content: sample.Raw,
		})
	}
	return samples
}

// sampleUnmapped is what the drafted project needs mapped to take an inbound sample tender: it
// reads the sample, answers every lookup as unmapped, and reports the codes the tender then
// requires.
func (p *planner) sampleUnmapped(
	ctx context.Context,
	sample *ediconnectionfield.Sample,
	mapped map[mappingKey]bool,
) []ediconnectiondraftports.DraftUnmappedCode {
	out := make([]ediconnectiondraftports.DraftUnmappedCode, 0)
	if p.encoded == nil || sample.Direction != edidocument.DocumentDirectionInbound ||
		sample.TransactionSet() != edidocument.TransactionSet204 {
		return out
	}
	_, clause := p.manifest.Receiving(string(edidocument.TransactionSet204))
	if clause == nil {
		return out
	}
	request := &lading.RunRequest{Map: clause.Map, Input: lading.X12Input(sample.Raw)}
	result, err := p.s.lading.Client().Run(ctx, p.encoded, request)
	if err != nil || result.Error != nil {
		return out
	}
	for _, need := range result.Needs {
		request.Lookups = append(
			request.Lookups,
			lading.LookupAnswer{Kind: need.Kind, Code: need.Code},
		)
	}
	if len(request.Lookups) > 0 {
		if result, err = p.s.lading.Client().Run(ctx, p.encoded, request); err != nil ||
			result.Error != nil {
			return out
		}
	}
	unmapped := ladingdoc.UnmappedOf(result.Unmapped)
	for index := range result.Transactions {
		tender, _, convertErr := ladingdoc.ToLoadTender(
			result.Transactions[index].Document,
			unmapped,
		)
		if convertErr != nil {
			continue
		}
		out = appendUnmapped(out, tender.RequiredMappingEntityIDs, mapped)
	}
	return out
}

func appendUnmapped(
	out []ediconnectiondraftports.DraftUnmappedCode,
	required map[edi.MappingEntityType][]pulid.ID,
	mapped map[mappingKey]bool,
) []ediconnectiondraftports.DraftUnmappedCode {
	entityTypes := make([]edi.MappingEntityType, 0, len(required))
	for entityType := range required {
		entityTypes = append(entityTypes, entityType)
	}
	slices.Sort(entityTypes)
	for _, entityType := range entityTypes {
		ids := slices.Clone(required[entityType])
		slices.Sort(ids)
		for _, id := range ids {
			key := mappingKey{entityType: entityType, sourceID: id}
			if mapped[key] ||
				slices.ContainsFunc(out, func(code ediconnectiondraftports.DraftUnmappedCode) bool {
					return code.EntityType == entityType && code.SourceID == id
				}) {
				continue
			}
			code := ediconnectiondraftports.DraftUnmappedCode{EntityType: entityType, SourceID: id}
			if id == defaultMappingKey {
				code.SourceLabel = "Default for every inbound tender"
			}
			out = append(out, code)
		}
	}
	return out
}
