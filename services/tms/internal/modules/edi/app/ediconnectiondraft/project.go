package ediconnectiondraft

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/ladingdoc"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	projectRecord    = "Lading project"
	guideDirectory   = "guides/"
	keptSetLabel     = "The document says this set is not exchanged; the project keeps what it has for it"
	keptGuideLabel   = "The project's guide is kept as it is; add the document's requirements to it in the Lading studio"
	noMapSummary     = "The project has no map for %s; add one in the Lading studio"
	ackConflict      = "The document asks for %s; the project acknowledges every set with %s"
	draftNotes       = "Drafted from %s"
	conditionalNote  = "The partner's map writes it only under a condition"
	maxGuideCodes    = 50
	maxDiagnostics   = 100
	maxGuideAttempts = 3
)

var (
	partnerHeader   = regexp.MustCompile(`(?m)^partner\s+([A-Za-z_][A-Za-z0-9_]*)`)
	interchangeArea = regexp.MustCompile(`(?s)interchange\s*\{[^{}]*\}`)
	zoneLine        = regexp.MustCompile(`(?m)^zone\s+\S+[ \t]*$`)
	baseGuideRef    = regexp.MustCompile(`(?m)^(\s*(?:from|to)\s+)x12\.\d{6}\.(\d{3})([ \t]*)$`)
	segmentID       = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,2}$`)
)

type requirement struct {
	input  *ediconnectiondraftports.DraftRequirementInput
	source ediconnectiondraftports.DraftSource
	ref    string
}

type exchange struct {
	set          edidocument.TransactionSet
	direction    edidocument.DocumentDirection
	index        int
	version      string
	ack          string
	window       int
	requirements []*requirement
}

type projectWrite struct {
	files   []lading.File
	samples []lading.Sample
	notes   string
	change  ediconnectiondraftports.ChangeKind
}

type guideLine struct {
	requirement *requirement
	line        int
}

type guideFile struct {
	set   edidocument.TransactionSet
	name  string
	path  string
	file  lading.File
	lines []guideLine
}

func setKey(set edidocument.TransactionSet, direction edidocument.DocumentDirection) string {
	return string(set) + "/" + string(direction)
}

func guidePath(set edidocument.TransactionSet) string {
	return guideDirectory + string(set) + ".lading"
}

func elementRef(segment string, position int) string {
	if position <= 0 {
		return segment
	}
	return fmt.Sprintf("%s%02d", segment, position)
}

func (p *planner) documentTypeID(
	ctx context.Context,
	set edidocument.TransactionSet,
	direction edidocument.DocumentDirection,
) (pulid.ID, error) {
	key := setKey(set, direction)
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
) ([]*requirement, []ediconnectiondraftports.DraftRequirementCheck) {
	accepted := make([]*requirement, 0)
	unverified := make([]ediconnectiondraftports.DraftRequirementCheck, 0)
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
		segment := strings.ToUpper(strings.TrimSpace(input.SegmentID))
		problem := check.problem
		switch {
		case !check.ok:
		case !segmentID.MatchString(segment):
			problem = fmt.Sprintf("%s is not an X12 segment ID", segment)
		case !containsToken(normalizeForMatch(input.Citation.Quote), strings.ToLower(segment)) &&
			!strings.Contains(strings.ToUpper(input.Citation.Quote), segment):
			problem = fmt.Sprintf("The quote does not mention %s", segment)
		default:
			accepted = append(accepted, &requirement{
				input:  input,
				source: source,
				ref:    elementRef(segment, input.ElementPosition),
			})
			continue
		}
		unverified = append(unverified, requirementCheck(input, segment, source, problem))
	}
	return accepted, unverified
}

func requirementCheck(
	input *ediconnectiondraftports.DraftRequirementInput,
	segment string,
	source ediconnectiondraftports.DraftSource,
	problem string,
) ediconnectiondraftports.DraftRequirementCheck {
	return ediconnectiondraftports.DraftRequirementCheck{
		SegmentID:       segment,
		ElementPosition: input.ElementPosition,
		Usage:           input.Usage,
		Codes:           input.Codes,
		Problem:         problem,
		Source:          source,
	}
}

func (p *planner) buildSets(ctx context.Context) ([]*exchange, error) {
	exchanges := make([]*exchange, 0)
	for _, set := range supportedSets {
		for _, direction := range directions {
			supported := p.values[setField(set, direction, setAttrSupported)]
			current := p.current.exchanged[setKey(set, direction)]
			wanted := supported.Status.Usable() && supported.Value == valueTrue
			if !wanted {
				if current && supported.Status.Usable() && supported.Value == valueFalse &&
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
	interchangeVersion, _ := p.usable(fieldX12Version)
	versionFallback := &fallback{
		value:  interchangeVersion,
		status: ediconnectiondraftports.ValueStatusDefaulted,
		source: ediconnectiondraftports.DraftSource{
			Kind: ediconnectiondraftports.SourceKindDefault,
			Note: versionFromISA,
		},
	}
	if source, ok := p.values[fieldX12Version]; ok && source.Status.Usable() {
		versionFallback.source = source.Source
		versionFallback.source.Note = joinNotes(source.Source.Note, versionFromISA)
	}
	version := p.resolve(setField(set, direction, setAttrVersion), versionFallback, true)
	ack := p.resolve(setField(set, direction, setAttrAck), nil, false)
	window := p.resolve(setField(set, direction, setAttrAckWindowMinutes), nil, false)
	fields := []ediconnectiondraftports.DraftValue{
		p.values[setField(set, direction, setAttrSupported)],
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
	out := make([]ediconnectiondraftports.DraftControlNumber, 0, len(controlNumberKinds))
	for _, kind := range controlNumberKinds {
		value, ok := p.values[controlNumberField(kind)]
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
		if sequence := p.current.sequence(documentTypeID, kind); sequence != nil {
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
	if version := p.current.version; version != nil {
		files, guided = p.adaptProject(version, starter.Files, exchanges)
		kept, keptErr := p.s.projects.Samples(ctx, p.req.TenantInfo, version)
		if keptErr != nil {
			return keptErr
		}
		samples = p.specSamples()
		checked = edilading.MergeSamples(kept, samples)
		project.ProjectID = version.ProjectID
	}
	name := partnerName(files)
	if name == "" {
		name = partnerName(starter.Files)
	}

	var check *lading.Check
	var guides []*guideFile
	for attempt := 1; ; attempt++ {
		guides = p.guides(name, request.Version, exchanges, guided)
		check, err = p.check(ctx, withGuides(files, guides), checked)
		if err != nil {
			return err
		}
		if attempt == maxGuideAttempts || !p.dropFailingGuideLines(check, guides, exchanges) {
			break
		}
	}
	files = withGuides(files, guides)

	p.unmet(check, exchanges)
	p.projectFailures(check, guides)
	project.Files = projectFiles(check, files, p.current.version)
	project.Errors = check.Errors
	project.Warnings = check.Warnings
	project.Diagnostics = projectDiagnostics(check)
	if p.current.version != nil {
		change = ediconnectiondraftports.ChangeKindChange
		if sameFiles(p.current.version.Files, files) {
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
	if version, ok := p.usable(fieldX12Version); ok {
		request.Version = version
	}
	request.Them = p.pair(fieldPartnerQualifier, fieldPartnerID)
	request.Us = p.pair(fieldOurQualifier, fieldOurID)
	request.Group = p.pair(fieldPartnerGSID, fieldOurGSID)
	separators := make([]string, 0, 4)
	for _, key := range []string{fieldElementSep, fieldSegmentSep, fieldComponentSep, fieldRepetitionSep} {
		if value, ok := p.usable(key); ok {
			separators = append(separators, value)
		}
	}
	if len(separators) == 4 {
		request.Separators = separators
	}
	request.Usage = valueTest
	if usage, ok := p.usable(fieldUsageIndicator); ok && usage == "P" {
		request.Usage = valueProduction
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
	if chosen == ackNone {
		return lading.AcknowledgeNone, window
	}
	return chosen, window
}

func partnerName(files []lading.File) string {
	for _, file := range files {
		if match := partnerHeader.FindStringSubmatch(file.Source); match != nil {
			return match[1]
		}
	}
	return ""
}

func (p *planner) adaptProject(
	version *ladingproject.LadingVersion,
	starter []lading.File,
	exchanges []*exchange,
) (files []lading.File, guided map[string]string) {
	files = slices.Clone(version.Files)
	var interchange, zone string
	for _, file := range starter {
		if partnerHeader.MatchString(file.Source) {
			interchange = interchangeArea.FindString(file.Source)
			zone = zoneLine.FindString(file.Source)
		}
	}
	for index := range files {
		source := files[index].Source
		if !partnerHeader.MatchString(source) {
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
			guide.Path != guidePath(edidocument.TransactionSet(guide.Set)) {
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
		if !mapsSet(manifest, exchanged.set, exchanged.direction) {
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

func mapsSet(
	manifest *lading.Check,
	set edidocument.TransactionSet,
	direction edidocument.DocumentDirection,
) bool {
	wanted := lading.DirectionOutbound
	if direction == edidocument.DocumentDirectionInbound {
		wanted = lading.DirectionInbound
	}
	for index := range manifest.Maps {
		if item := &manifest.Maps[index]; item.Set == string(set) && item.Direction == wanted {
			return true
		}
	}
	return false
}

func (p *planner) guides(
	name, version string,
	exchanges []*exchange,
	kept map[string]string,
) []*guideFile {
	bySet := map[edidocument.TransactionSet]*guideFile{}
	order := make([]edidocument.TransactionSet, 0)
	label := firstNonEmpty(p.plannedPartner().Name, name)
	for _, exchanged := range exchanges {
		if _, ok := kept[string(exchanged.set)]; ok {
			continue
		}
		guide, ok := bySet[exchanged.set]
		if !ok {
			base := firstNonEmpty(exchanged.version, version, "004010")
			guide = &guideFile{
				set:  exchanged.set,
				name: name + "." + string(exchanged.set),
				path: guidePath(exchanged.set),
			}
			guide.file = lading.File{Path: guide.path, Source: fmt.Sprintf(
				"guide %s from x12.%s.%s %s\n",
				guide.name,
				base,
				exchanged.set,
				quoteLading(
					fmt.Sprintf("%s %s, from %s", label, exchanged.set, p.req.Document.FileName),
				),
			)}
			bySet[exchanged.set] = guide
			order = append(order, exchanged.set)
		}
		for _, item := range exchanged.requirements {
			line, found := guideRule(item.input)
			if !found {
				continue
			}
			doc := fmt.Sprintf("/// p.%d: %s\n", item.source.Page, oneLine(item.source.Quote))
			guide.file.Source += "\n" + doc
			guide.lines = append(guide.lines, guideLine{
				requirement: item,
				line:        strings.Count(guide.file.Source, "\n") + 1,
			})
			guide.file.Source += line + "\n"
		}
	}
	out := make([]*guideFile, 0, len(order))
	for _, set := range order {
		out = append(out, bySet[set])
	}
	return out
}

func guideRule(input *ediconnectiondraftports.DraftRequirementInput) (string, bool) {
	segment := strings.ToUpper(strings.TrimSpace(input.SegmentID))
	usage := ""
	switch input.Usage {
	case ediconnectiondraftports.RequirementUsageMandatory:
		usage = "required"
	case ediconnectiondraftports.RequirementUsageNotUsed:
		usage = "unused"
	case ediconnectiondraftports.RequirementUsageOptional,
		ediconnectiondraftports.RequirementUsageConditional:
	}
	if input.ElementPosition <= 0 {
		if usage == "" {
			return "", false
		}
		return segment + " " + usage, true
	}
	rules := make([]string, 0, 2)
	if usage != "" {
		rules = append(rules, usage)
	}
	if codes := guideCodes(
		input.Codes,
	); codes != "" &&
		input.Usage != ediconnectiondraftports.RequirementUsageNotUsed {
		rules = append(rules, "codes "+codes)
	}
	if len(rules) == 0 {
		return "", false
	}
	return fmt.Sprintf(
		"%s { %02d %s }",
		segment,
		input.ElementPosition,
		strings.Join(rules, " "),
	), true
}

func guideCodes(codes []string) string {
	quoted := make([]string, 0, len(codes))
	seen := map[string]bool{}
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" || seen[code] || len(quoted) == maxGuideCodes {
			continue
		}
		seen[code] = true
		quoted = append(quoted, quoteLading(code))
	}
	return strings.Join(quoted, ", ")
}

func quoteLading(text string) string {
	replacer := strings.NewReplacer(
		`\`,
		`\\`,
		`"`,
		`\"`,
		"{",
		"{{",
		"}",
		"}}",
		"\n",
		" ",
		"\r",
		" ",
	)
	return `"` + replacer.Replace(text) + `"`
}

func oneLine(text string) string {
	return strings.Join(strings.Fields(clampQuote(text)), " ")
}

func withGuides(files []lading.File, guides []*guideFile) []lading.File {
	replaced := make(map[string]string, len(guides))
	paths := make(map[string]bool, len(guides))
	for _, guide := range guides {
		replaced[string(guide.set)] = guide.name
		paths[guide.path] = true
	}
	out := make([]lading.File, 0, len(files)+len(guides))
	for _, file := range files {
		if paths[file.Path] {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(file.Source), "map ") {
			file.Source = baseGuideRef.ReplaceAllStringFunc(file.Source, func(line string) string {
				match := baseGuideRef.FindStringSubmatch(line)
				name, ok := replaced[match[2]]
				if !ok {
					return line
				}
				return match[1] + name + match[3]
			})
		}
		out = append(out, file)
	}
	for _, guide := range guides {
		out = append(out, guide.file)
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
	guides []*guideFile,
	exchanges []*exchange,
) bool {
	dropped := false
	for _, guide := range guides {
		checked := checkedFile(check, guide.path)
		if checked == nil {
			continue
		}
		for index := range checked.Diagnostics {
			diagnostic := &checked.Diagnostics[index]
			if diagnostic.Severity != lading.SeverityError {
				continue
			}
			line := lineAt(guide.file.Source, diagnostic.Range.From)
			for _, item := range guide.lines {
				if item.line != line {
					continue
				}
				p.unverify(exchanges, item.requirement, diagnostic.Message.String())
				dropped = true
			}
		}
	}
	return dropped
}

func (p *planner) unverify(exchanges []*exchange, item *requirement, problem string) {
	for _, exchanged := range exchanges {
		index := slices.Index(exchanged.requirements, item)
		if index < 0 {
			continue
		}
		exchanged.requirements = slices.Delete(exchanged.requirements, index, index+1)
		set := &p.draft.Sets[exchanged.index]
		set.Requirements--
		set.Unverified = append(set.Unverified, requirementCheck(
			item.input,
			strings.ToUpper(strings.TrimSpace(item.input.SegmentID)),
			item.source,
			problem,
		))
	}
}

func checkedFile(check *lading.Check, path string) *lading.CheckedFile {
	for index := range check.Files {
		if check.Files[index].Path == path {
			return &check.Files[index]
		}
	}
	return nil
}

func lineAt(source string, offset int) int {
	if offset > len(source) {
		offset = len(source)
	}
	if offset < 0 {
		offset = 0
	}
	return strings.Count(source[:offset], "\n") + 1
}

func (p *planner) unmet(check *lading.Check, exchanges []*exchange) {
	unwritten := unwrittenBySet(check)
	for _, exchanged := range exchanges {
		if exchanged.direction != edidocument.DocumentDirectionOutbound {
			continue
		}
		names := unwritten[string(exchanged.set)]
		set := &p.draft.Sets[exchanged.index]
		for _, item := range exchanged.requirements {
			if item.input.Usage != ediconnectiondraftports.RequirementUsageMandatory {
				continue
			}
			problem, ok := names[item.ref]
			if !ok {
				continue
			}
			set.Unmet = append(set.Unmet, requirementCheck(
				item.input,
				strings.ToUpper(strings.TrimSpace(item.input.SegmentID)),
				item.source,
				problem,
			))
		}
	}
}

func unwrittenBySet(check *lading.Check) map[string]map[string]string {
	unwritten := map[string]map[string]string{}
	for index := range check.Maps {
		item := &check.Maps[index]
		if item.Direction != lading.DirectionOutbound || item.Set == "" {
			continue
		}
		checked := checkedFile(check, item.Path)
		if checked == nil {
			continue
		}
		for diagnosticIndex := range checked.Diagnostics {
			diagnostic := &checked.Diagnostics[diagnosticIndex]
			problem, unmet := unmetProblem(diagnostic)
			if !unmet {
				continue
			}
			if unwritten[item.Set] == nil {
				unwritten[item.Set] = map[string]string{}
			}
			for _, name := range unwrittenNames(diagnostic.Message) {
				if _, seen := unwritten[item.Set][name]; !seen {
					unwritten[item.Set][name] = problem
				}
			}
		}
	}
	return unwritten
}

func unmetProblem(diagnostic *lading.Diagnostic) (problem string, unmet bool) {
	switch diagnostic.Code {
	case "x12-mandatory-unwritten":
		return diagnostic.Message.String(), true
	case "x12-mandatory-conditional":
		return conditionalNote, true
	default:
		return "", false
	}
}

func unwrittenNames(message wasmmodule.Message) []string {
	if len(message.Args) == 0 {
		return nil
	}
	text, ok := message.Args[0].(string)
	if !ok {
		return nil
	}
	names := make([]string, 0)
	for name := range strings.SplitSeq(text, ",") {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func (p *planner) projectFailures(check *lading.Check, guides []*guideFile) {
	owned := map[string]bool{}
	for _, guide := range guides {
		owned[guide.path] = true
	}
	for index := range check.Files {
		file := &check.Files[index]
		if file.Kind != "partner" && !owned[file.Path] && p.current.version != nil {
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

func projectFiles(
	check *lading.Check,
	files []lading.File,
	before *ladingproject.LadingVersion,
) []ediconnectiondraftports.DraftProjectFile {
	previous := map[string]string{}
	if before != nil {
		for _, file := range before.Files {
			previous[file.Path] = file.Source
		}
	}
	out := make([]ediconnectiondraftports.DraftProjectFile, 0, len(files))
	for _, file := range files {
		change := ediconnectiondraftports.ChangeKindAdd
		if source, ok := previous[file.Path]; ok {
			change = ediconnectiondraftports.ChangeKindChange
			if source == file.Source {
				change = ediconnectiondraftports.ChangeKindSame
			}
		}
		kind := ""
		if checked := checkedFile(check, file.Path); checked != nil {
			kind = checked.Kind
		}
		out = append(
			out,
			ediconnectiondraftports.DraftProjectFile{Path: file.Path, Kind: kind, Change: change},
		)
	}
	return out
}

func projectDiagnostics(check *lading.Check) []ediconnectiondraftports.DraftDiagnostic {
	out := make([]ediconnectiondraftports.DraftDiagnostic, 0)
	for fileIndex := range check.Files {
		file := &check.Files[fileIndex]
		for index := range file.Diagnostics {
			diagnostic := &file.Diagnostics[index]
			if diagnostic.Severity == lading.SeverityInfo || len(out) == maxDiagnostics {
				continue
			}
			out = append(out, ediconnectiondraftports.DraftDiagnostic{
				Path:     file.Path,
				Severity: diagnostic.Severity,
				Code:     diagnostic.Code,
				Message:  diagnostic.Message.String(),
			})
		}
	}
	return out
}

// specSamples are the spec's sample interchanges, kept with the project for its tests.
func (p *planner) specSamples() []lading.Sample {
	samples := make([]lading.Sample, 0, len(p.samples))
	for _, sample := range p.samples {
		if sample.err != nil || strings.TrimSpace(sample.raw) == "" {
			continue
		}
		samples = append(samples, lading.Sample{
			Path:    fmt.Sprintf("samples/spec-page-%d.x12", sample.page),
			Content: sample.raw,
		})
	}
	return samples
}

// sampleUnmapped is what the drafted project needs mapped to take an inbound sample tender: it
// reads the sample, answers every lookup as unmapped, and reports the codes the tender then
// requires.
func (p *planner) sampleUnmapped(
	ctx context.Context,
	sample *specSample,
	mapped map[mappingKey]bool,
) []ediconnectiondraftports.DraftUnmappedCode {
	out := make([]ediconnectiondraftports.DraftUnmappedCode, 0)
	if p.encoded == nil || sample.direction != edidocument.DocumentDirectionInbound ||
		sample.transactionSet() != edidocument.TransactionSet204 {
		return out
	}
	_, clause := p.manifest.Receiving(string(edidocument.TransactionSet204))
	if clause == nil {
		return out
	}
	request := &lading.RunRequest{Map: clause.Map, Input: lading.X12Input(sample.raw)}
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

func sameFiles(before, after []lading.File) bool {
	if len(before) != len(after) {
		return false
	}
	sources := make(map[string]string, len(before))
	for _, file := range before {
		sources[file.Path] = file.Source
	}
	for _, file := range after {
		if source, ok := sources[file.Path]; !ok || source != file.Source {
			return false
		}
	}
	return true
}
