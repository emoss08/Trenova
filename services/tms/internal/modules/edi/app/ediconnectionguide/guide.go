package ediconnectionguide

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediconnectionfield"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/wasmmodule"
)

const (
	GuideDirectory  = "guides/"
	conditionalNote = "The partner's map writes it only under a condition"
	maxGuideCodes   = 50
	maxDiagnostics  = 100
)

var (
	PartnerHeader = regexp.MustCompile(`(?m)^partner\s+([A-Za-z_][A-Za-z0-9_]*)`)
	baseGuideRef  = regexp.MustCompile(`(?m)^(\s*(?:from|to)\s+)x12\.\d{6}\.(\d{3})([ \t]*)$`)
)

type Requirement struct {
	Input  *ediconnectiondraftports.DraftRequirementInput
	Source ediconnectiondraftports.DraftSource
	Ref    string
}

type GuideLine struct {
	Requirement *Requirement
	Line        int
}

type GuideFile struct {
	Set   edidocument.TransactionSet
	Name  string
	Path  string
	File  lading.File
	Lines []GuideLine
}

func GuidePath(set edidocument.TransactionSet) string {
	return GuideDirectory + string(set) + ".lading"
}

func ElementRef(segment string, position int) string {
	if position <= 0 {
		return segment
	}
	return fmt.Sprintf("%s%02d", segment, position)
}

func RequirementCheck(
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

func PartnerName(files []lading.File) string {
	for _, file := range files {
		if match := PartnerHeader.FindStringSubmatch(file.Source); match != nil {
			return match[1]
		}
	}
	return ""
}

func MapsSet(
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

func GuideRule(input *ediconnectiondraftports.DraftRequirementInput) (string, bool) {
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
		quoted = append(quoted, QuoteLading(code))
	}
	return strings.Join(quoted, ", ")
}

func QuoteLading(text string) string {
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

func OneLine(text string) string {
	return strings.Join(strings.Fields(ediconnectionfield.ClampQuote(text)), " ")
}

func WithGuides(files []lading.File, guides []*GuideFile) []lading.File {
	replaced := make(map[string]string, len(guides))
	paths := make(map[string]bool, len(guides))
	for _, guide := range guides {
		replaced[string(guide.Set)] = guide.Name
		paths[guide.Path] = true
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
		out = append(out, guide.File)
	}
	return out
}

func CheckedFile(check *lading.Check, path string) *lading.CheckedFile {
	for index := range check.Files {
		if check.Files[index].Path == path {
			return &check.Files[index]
		}
	}
	return nil
}

func LineAt(source string, offset int) int {
	if offset > len(source) {
		offset = len(source)
	}
	if offset < 0 {
		offset = 0
	}
	return strings.Count(source[:offset], "\n") + 1
}

func UnwrittenBySet(check *lading.Check) map[string]map[string]string {
	unwritten := map[string]map[string]string{}
	for index := range check.Maps {
		item := &check.Maps[index]
		if item.Direction != lading.DirectionOutbound || item.Set == "" {
			continue
		}
		checked := CheckedFile(check, item.Path)
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

func ProjectFiles(
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
		if checked := CheckedFile(check, file.Path); checked != nil {
			kind = checked.Kind
		}
		out = append(
			out,
			ediconnectiondraftports.DraftProjectFile{Path: file.Path, Kind: kind, Change: change},
		)
	}
	return out
}

func ProjectDiagnostics(check *lading.Check) []ediconnectiondraftports.DraftDiagnostic {
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

func SameFiles(before, after []lading.File) bool {
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
