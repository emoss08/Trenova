package carrierintel

import (
	"slices"
	"sort"
	"strings"

	"github.com/bytedance/sonic"
)

type FieldChange struct {
	Path     string   `json:"path"`
	Section  Section  `json:"section"`
	Label    string   `json:"label"`
	Severity Severity `json:"severity"`
	Prior    any      `json:"prior"`
	Current  any      `json:"current"`
}

var sectionPrefixes = map[string]Section{
	"identity":      SectionIdentity,
	"authority":     SectionAuthority,
	"insurance":     SectionInsurance,
	"safety":        SectionSafety,
	"basics":        SectionBasics,
	"inspections":   SectionInspections,
	"crashes":       SectionCrashes,
	"fleet":         SectionFleet,
	"equipment":     SectionEquipment,
	"contacts":      SectionContacts,
	"operations":    SectionOperations,
	"changeHistory": SectionChangeHistory,
	"network":       SectionNetwork,
	"lanes":         SectionLanes,
	"benchmarks":    SectionBenchmarks,
}

type fieldMeta struct {
	Label    string
	Severity Severity
}

var fieldCatalog = map[string]fieldMeta{
	"identity.usdotStatus":     {"USDOT status", SeverityCritical},
	"identity.legalName":       {"Legal name", SeverityHigh},
	"identity.dbaName":         {"DBA name", SeverityMedium},
	"identity.ein":             {"EIN", SeverityHigh},
	"identity.docketNumber":    {"Docket number", SeverityHigh},
	"identity.physicalAddress": {"Physical address", SeverityMedium},
	"identity.mailingAddress":  {"Mailing address", SeverityMedium},
	"authority.common.status":  {"Common authority", SeverityCritical},
	"authority.common.pending": {"Common authority pending", SeverityMedium},
	"authority.common.revocationPending": {
		"Common authority revocation pending",
		SeverityCritical,
	},
	"authority.contract.status": {"Contract authority", SeverityCritical},
	"authority.contract.revocationPending": {
		"Contract authority revocation pending",
		SeverityCritical,
	},
	"authority.broker.status": {"Broker authority", SeverityCritical},
	"authority.broker.revocationPending": {
		"Broker authority revocation pending",
		SeverityCritical,
	},
	"authority.totalRevocations":   {"Total revocations", SeverityHigh},
	"insurance.bipdOnFile":         {"BIPD coverage on file", SeverityCritical},
	"insurance.bipdRequired":       {"BIPD coverage required", SeverityMedium},
	"insurance.cargoOnFile":        {"Cargo coverage on file", SeverityHigh},
	"insurance.cargoRequired":      {"Cargo coverage required", SeverityMedium},
	"insurance.bondOnFile":         {"Bond on file", SeverityHigh},
	"insurance.pendingCancelAt":    {"Pending insurance cancellation", SeverityCritical},
	"insurance.lastCanceledAt":     {"Last insurance cancellation", SeverityHigh},
	"insurance.cancelCount":        {"Insurance cancellations", SeverityMedium},
	"safety.rating":                {"Safety rating", SeverityHigh},
	"safety.ratingDate":            {"Safety rating date", SeverityLow},
	"safety.issValue":              {"ISS score", SeverityMedium},
	"safety.riskScore":             {"Risk score", SeverityMedium},
	"safety.outOfServiceOrder":     {"Out-of-service order", SeverityCritical},
	"safety.latestReviewAt":        {"Latest compliance review", SeverityLow},
	"crashes.fatal":                {"Fatal crashes", SeverityHigh},
	"crashes.total":                {"Crashes", SeverityMedium},
	"crashes.lastCrashAt":          {"Last crash", SeverityMedium},
	"fleet.powerUnits":             {"Power units", SeverityLow},
	"fleet.drivers":                {"Drivers", SeverityLow},
	"contacts.phone":               {"Phone", SeverityMedium},
	"contacts.email":               {"Email", SeverityMedium},
	"operations.mcs150At":          {"MCS-150 filing date", SeverityLow},
	"operations.hazmatCarrier":     {"Hazmat carrier", SeverityLow},
	"network.sharedAddresses":      {"Shared addresses", SeverityMedium},
	"network.sharedPhones":         {"Shared phones", SeverityMedium},
	"network.sharedEmails":         {"Shared emails", SeverityMedium},
	"network.sharedEins":           {"Shared EINs", SeverityHigh},
	"network.sharedEquipment":      {"Shared equipment", SeverityHigh},
	"changeHistory.nameChanges":    {"Name changes", SeverityMedium},
	"changeHistory.phoneChanges":   {"Phone changes", SeverityMedium},
	"changeHistory.emailChanges":   {"Email changes", SeverityMedium},
	"changeHistory.addressChanges": {"Address changes", SeverityMedium},
}

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

func SectionForPath(path string) Section {
	head, _, _ := strings.Cut(path, ".")
	if section, ok := sectionPrefixes[head]; ok {
		return section
	}
	return ""
}

func DescribeField(path string) (label string, severity Severity) {
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
			severity = SeverityLow
			if parts[2] == "alert" {
				severity = SeverityHigh
			}
			return parts[1] + " " + parts[2], severity
		}
	}
	return path, SeverityInfo
}

func isIgnoredPath(path string) bool {
	for _, ignored := range ignoredDiffPaths {
		if path == ignored || strings.HasPrefix(path, ignored+".") {
			return true
		}
	}
	return false
}

func FlattenProfile(p *Profile) (map[string]any, error) {
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
			if field == "basic" {
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

func DiffProfiles(prior, current *Profile) ([]FieldChange, error) {
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
