package carrierinteldiff

import (
	"slices"
	"sort"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
)

type FieldChange struct {
	Path     string                `json:"path"`
	Section  carrierintel.Section  `json:"section"`
	Label    string                `json:"label"`
	Severity carrierintel.Severity `json:"severity"`
	Prior    any                   `json:"prior"`
	Current  any                   `json:"current"`
}

var sectionPrefixes = map[string]carrierintel.Section{
	"identity":      carrierintel.SectionIdentity,
	"authority":     carrierintel.SectionAuthority,
	"insurance":     carrierintel.SectionInsurance,
	"safety":        carrierintel.SectionSafety,
	"basics":        carrierintel.SectionBasics,
	"inspections":   carrierintel.SectionInspections,
	"crashes":       carrierintel.SectionCrashes,
	"fleet":         carrierintel.SectionFleet,
	"equipment":     carrierintel.SectionEquipment,
	"contacts":      carrierintel.SectionContacts,
	"operations":    carrierintel.SectionOperations,
	"changeHistory": carrierintel.SectionChangeHistory,
	"network":       carrierintel.SectionNetwork,
	"lanes":         carrierintel.SectionLanes,
	"benchmarks":    carrierintel.SectionBenchmarks,
}

type fieldMeta struct {
	Label    string
	Severity carrierintel.Severity
}

var fieldCatalog = map[string]fieldMeta{
	"identity.usdotStatus":     {"USDOT status", carrierintel.SeverityCritical},
	"identity.legalName":       {"Legal name", carrierintel.SeverityHigh},
	"identity.dbaName":         {"DBA name", carrierintel.SeverityMedium},
	"identity.ein":             {"EIN", carrierintel.SeverityHigh},
	"identity.docketNumber":    {"Docket number", carrierintel.SeverityHigh},
	"identity.physicalAddress": {"Physical address", carrierintel.SeverityMedium},
	"identity.mailingAddress":  {"Mailing address", carrierintel.SeverityMedium},
	"authority.common.status":  {"Common authority", carrierintel.SeverityCritical},
	"authority.common.pending": {"Common authority pending", carrierintel.SeverityMedium},
	"authority.common.revocationPending": {
		"Common authority revocation pending",
		carrierintel.SeverityCritical,
	},
	"authority.contract.status": {"Contract authority", carrierintel.SeverityCritical},
	"authority.contract.revocationPending": {
		"Contract authority revocation pending",
		carrierintel.SeverityCritical,
	},
	"authority.broker.status": {"Broker authority", carrierintel.SeverityCritical},
	"authority.broker.revocationPending": {
		"Broker authority revocation pending",
		carrierintel.SeverityCritical,
	},
	"authority.totalRevocations": {"Total revocations", carrierintel.SeverityHigh},
	"insurance.bipdOnFile":       {"BIPD coverage on file", carrierintel.SeverityCritical},
	"insurance.bipdRequired":     {"BIPD coverage required", carrierintel.SeverityMedium},
	"insurance.cargoOnFile":      {"Cargo coverage on file", carrierintel.SeverityHigh},
	"insurance.cargoRequired":    {"Cargo coverage required", carrierintel.SeverityMedium},
	"insurance.bondOnFile":       {"Bond on file", carrierintel.SeverityHigh},
	"insurance.pendingCancelAt": {
		"Pending insurance cancellation",
		carrierintel.SeverityCritical,
	},
	"insurance.lastCanceledAt":     {"Last insurance cancellation", carrierintel.SeverityHigh},
	"insurance.cancelCount":        {"Insurance cancellations", carrierintel.SeverityMedium},
	"safety.rating":                {"Safety rating", carrierintel.SeverityHigh},
	"safety.ratingDate":            {"Safety rating date", carrierintel.SeverityLow},
	"safety.issValue":              {"ISS score", carrierintel.SeverityMedium},
	"safety.riskScore":             {"Risk score", carrierintel.SeverityMedium},
	"safety.outOfServiceOrder":     {"Out-of-service order", carrierintel.SeverityCritical},
	"safety.latestReviewAt":        {"Latest compliance review", carrierintel.SeverityLow},
	"crashes.fatal":                {"Fatal crashes", carrierintel.SeverityHigh},
	"crashes.total":                {"Crashes", carrierintel.SeverityMedium},
	"crashes.lastCrashAt":          {"Last crash", carrierintel.SeverityMedium},
	"fleet.powerUnits":             {"Power units", carrierintel.SeverityLow},
	"fleet.drivers":                {"Drivers", carrierintel.SeverityLow},
	"contacts.phone":               {"Phone", carrierintel.SeverityMedium},
	"contacts.email":               {"Email", carrierintel.SeverityMedium},
	"operations.mcs150At":          {"MCS-150 filing date", carrierintel.SeverityLow},
	"operations.hazmatCarrier":     {"Hazmat carrier", carrierintel.SeverityLow},
	"network.sharedAddresses":      {"Shared addresses", carrierintel.SeverityMedium},
	"network.sharedPhones":         {"Shared phones", carrierintel.SeverityMedium},
	"network.sharedEmails":         {"Shared emails", carrierintel.SeverityMedium},
	"network.sharedEins":           {"Shared EINs", carrierintel.SeverityHigh},
	"network.sharedEquipment":      {"Shared equipment", carrierintel.SeverityHigh},
	"changeHistory.nameChanges":    {"Name changes", carrierintel.SeverityMedium},
	"changeHistory.phoneChanges":   {"Phone changes", carrierintel.SeverityMedium},
	"changeHistory.emailChanges":   {"Email changes", carrierintel.SeverityMedium},
	"changeHistory.addressChanges": {"Address changes", carrierintel.SeverityMedium},
}

var ignoredBasicFields = []string{"measuredAt"}

var ignoredDiffPaths = []string{
	"identity.dotAgeDays",
	"authority.common.ageDays",
	"authority.contract.ageDays",
	"authority.broker.ageDays",
	"authority.history",
	"insurance.filings",
	"equipment",
	"network.links",
	"lanes",
	"coverage",
}

func SectionForPath(path string) carrierintel.Section {
	head, _, _ := strings.Cut(path, ".")
	if section, ok := sectionPrefixes[head]; ok {
		return section
	}
	return ""
}

func DescribeField(path string) (label string, severity carrierintel.Severity) {
	if meta, ok := fieldCatalog[path]; ok {
		return meta.Label, meta.Severity
	}
	for prefix, meta := range fieldCatalog {
		if strings.HasPrefix(path, prefix+".") {
			return meta.Label, meta.Severity
		}
	}
	if strings.HasPrefix(path, "basics.") {
		parts := strings.Split(path, ".")
		if len(parts) >= 3 {
			severity = carrierintel.SeverityLow
			if parts[2] == "alert" {
				severity = carrierintel.SeverityHigh
			}
			return parts[1] + " " + parts[2], severity
		}
	}
	return path, carrierintel.SeverityInfo
}

func isIgnoredPath(path string) bool {
	for _, ignored := range ignoredDiffPaths {
		if path == ignored || strings.HasPrefix(path, ignored+".") {
			return true
		}
	}
	return false
}

func FlattenProfile(p *carrierintel.Profile) (map[string]any, error) {
	if p == nil {
		return map[string]any{}, nil
	}
	raw, err := sonic.Marshal(p)
	if err != nil {
		return nil, err
	}
	var tree map[string]any
	if err = sonic.Unmarshal(raw, &tree); err != nil {
		return nil, err
	}

	flat := make(map[string]any, 128)
	for key, value := range tree {
		if key == "basics" {
			flattenBasics(flat, value)
			continue
		}
		flattenValue(flat, key, value)
	}
	return flat, nil
}

func flattenBasics(flat map[string]any, value any) {
	items, ok := value.([]any)
	if !ok {
		return
	}
	for _, item := range items {
		entry, isMap := item.(map[string]any)
		if !isMap {
			continue
		}
		basic, _ := entry["basic"].(string)
		if basic == "" {
			continue
		}
		for field, fieldValue := range entry {
			if field == "basic" || slices.Contains(ignoredBasicFields, field) {
				continue
			}
			flat["basics."+basic+"."+field] = fieldValue
		}
	}
}

func flattenValue(flat map[string]any, prefix string, value any) {
	if isIgnoredPath(prefix) {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		if isAddressPath(prefix) {
			flat[prefix] = addressString(typed)
			return
		}
		for key, nested := range typed {
			flattenValue(flat, prefix+"."+key, nested)
		}
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				parts = append(parts, s)
			}
		}
		sort.Strings(parts)
		flat[prefix] = strings.Join(parts, ", ")
	default:
		flat[prefix] = typed
	}
}

func isAddressPath(path string) bool {
	return strings.HasSuffix(path, "Address")
}

func addressString(m map[string]any) string {
	parts := make([]string, 0, 4)
	for _, key := range []string{"line1", "city", "state", "postalCode"} {
		if v, ok := m[key].(string); ok && v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, ", ")
}

type DiffOptions struct {
	IgnoreMissing bool
}

func DiffOptionsForDepths(prior, current carrierintel.LookupDepth) DiffOptions {
	return DiffOptions{IgnoreMissing: prior != current}
}

func DiffProfiles(prior, current *carrierintel.Profile, opts DiffOptions) ([]FieldChange, error) {
	if prior == nil || current == nil {
		return nil, nil
	}
	priorFlat, err := FlattenProfile(prior)
	if err != nil {
		return nil, err
	}
	currentFlat, err := FlattenProfile(current)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(priorFlat)+len(currentFlat))
	for path := range priorFlat {
		paths = append(paths, path)
	}
	for path := range currentFlat {
		if _, ok := priorFlat[path]; !ok {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)

	changes := make([]FieldChange, 0, 8)
	for _, path := range paths {
		section := SectionForPath(path)
		if section == "" || !prior.Covers(section) || !current.Covers(section) {
			continue
		}
		before, hadBefore := priorFlat[path]
		after, hasAfter := currentFlat[path]
		if opts.IgnoreMissing && (isMissing(before, hadBefore) || isMissing(after, hasAfter)) {
			continue
		}
		if valuesEqual(before, hadBefore, after, hasAfter) {
			continue
		}
		label, severity := DescribeField(path)
		changes = append(changes, FieldChange{
			Path:     path,
			Section:  section,
			Label:    label,
			Severity: severity,
			Prior:    nilIfMissing(before, hadBefore),
			Current:  nilIfMissing(after, hasAfter),
		})
	}
	return changes, nil
}

func nilIfMissing(v any, present bool) any {
	if !present {
		return nil
	}
	return v
}

func isMissing(v any, present bool) bool {
	return !present || v == nil
}

func isEmptyValue(v any, present bool) bool {
	if !present || v == nil {
		return true
	}
	switch typed := v.(type) {
	case string:
		return typed == ""
	case bool:
		return !typed
	}
	return false
}

func valuesEqual(before any, hadBefore bool, after any, hasAfter bool) bool {
	if isEmptyValue(before, hadBefore) && isEmptyValue(after, hasAfter) {
		return true
	}
	if hadBefore != hasAfter {
		return false
	}
	beforeRaw, errBefore := sonic.Marshal(before)
	afterRaw, errAfter := sonic.Marshal(after)
	if errBefore != nil || errAfter != nil {
		return false
	}
	return slices.Equal(beforeRaw, afterRaw)
}
