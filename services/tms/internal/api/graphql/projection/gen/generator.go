package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/vektah/gqlparser/v2"
	gqlast "github.com/vektah/gqlparser/v2/ast"
	"golang.org/x/mod/modfile"
	"gopkg.in/yaml.v3"
)

const (
	minCoverageFields = 2
	minCoverageRatio  = 0.60
)

const (
	reportCatalogDTO    = "GraphQL report catalog manifest DTO"
	reportPreviewDTO    = "GraphQL report preview DTO"
	jsonbValueObject    = "GraphQL value object stored as JSONB without column projection metadata"
	homeWidgetDTO       = "GraphQL home widget catalog DTO"
	deskMemoryDTO       = "GraphQL Desk memory DTO built by its resolver"
	shipmentWorkflowDTO = "GraphQL shipment workflow DTO"
	journalDTO          = "GraphQL journal DTO without column projection metadata"
	costingDTO          = "GraphQL costing DTO without column projection metadata"
	telematicsFormDTO   = "GraphQL telematics form DTO without column projection metadata"
)

var nonProjectionObjects = map[string]string{
	"AIProviderModelOption":           "GraphQL provider model listing DTO",
	"BulkEditFailure":                 "GraphQL bulk edit progress DTO",
	"BulkEditField":                   "GraphQL bulk edit catalog DTO",
	"BulkEditJob":                     "GraphQL bulk edit progress DTO",
	"BulkEditOption":                  "GraphQL bulk edit catalog DTO",
	"BulkEditPreview":                 "GraphQL bulk edit preview DTO",
	"DataTableAggregateValue":         "GraphQL table totals DTO",
	"DataTableAggregates":             "GraphQL table totals DTO",
	"DataTableFacetBucket":            "GraphQL table value counts DTO",
	"DataTableFacets":                 "GraphQL table value counts DTO",
	"DataTableInsightField":           "GraphQL table counts and totals catalog DTO",
	"DataTableSeries":                 "GraphQL per-row chart DTO",
	"DataTableSeriesGroup":            "GraphQL per-row chart DTO",
	"CannedReport":                    reportCatalogDTO,
	"ReportCatalog":                   reportCatalogDTO,
	"ReportCatalogEntity":             reportCatalogDTO,
	"ReportCatalogEnumValue":          reportCatalogDTO,
	"ReportCatalogField":              reportCatalogDTO,
	"ReportPreview":                   reportPreviewDTO,
	"ReportPreviewColumn":             reportPreviewDTO,
	"ReportPreviewRow":                reportPreviewDTO,
	"ReportRunError":                  "GraphQL report run error DTO",
	"EIASeriesOption":                 "GraphQL fuel surcharge reference DTO",
	"FuelIndexLatestPrice":            "GraphQL fuel dashboard DTO",
	"FuelProgramCurrentRate":          "GraphQL fuel dashboard DTO",
	"GeneratedFuelTableRow":           "GraphQL fuel table generator DTO",
	"GeofenceVertex":                  "GraphQL value object without column projection metadata",
	"ShipmentFreightLine":             jsonbValueObject,
	"ShipmentRatingSkip":              jsonbValueObject,
	"ShipmentRatingReviewApproval":    jsonbValueObject,
	"ShipmentContractRateCharge":      "GraphQL contract rate preview DTO built by its resolver",
	"TariffBill":                      jsonbValueObject,
	"TariffBillLine":                  jsonbValueObject,
	"TariffLineLimit":                 jsonbValueObject,
	"TariffBillExchange":              jsonbValueObject,
	"SelectOption":                    "GraphQL select-option DTO",
	"SidebarPreferences":              "GraphQL sidebar preferences DTO",
	"SidebarSectionOption":            "GraphQL sidebar customization catalog DTO",
	"SidebarAttentionMetricOption":    "GraphQL sidebar customization catalog DTO",
	"SidebarQuickActionOption":        "GraphQL sidebar quick-action DTO",
	"HomeWidgetCategory":              homeWidgetDTO,
	"HomeWidgetOption":                homeWidgetDTO,
	"HomeMetricOption":                homeWidgetDTO,
	"DispatchFinding":                 "GraphQL dispatch console DTO",
	"PendingDecisionAgentCount":       "GraphQL decision queue summary DTO",
	"RecentDecision":                  "GraphQL recent-decisions DTO built by its resolver",
	"DeskMemory":                      deskMemoryDTO,
	"DeskMemoryCount":                 deskMemoryDTO,
	"DeskMemoryLink":                  deskMemoryDTO,
	"DeskMemoryPage":                  deskMemoryDTO,
	"DeskMemoryRole":                  deskMemoryDTO,
	"DeskMemorySettings":              deskMemoryDTO,
	"DocumentTemplateKind":            "GraphQL registry catalog DTO, not a table",
	"DocumentTemplateVariable":        "GraphQL registry catalog DTO, not a table",
	"DocumentTemplateDiagnostic":      "GraphQL template engine finding DTO",
	"DocumentTemplatePreview":         "GraphQL template preview DTO",
	"ShipmentBoardGroup":              "GraphQL shipment board group DTO built by its resolver",
	"ShipmentAxleWeight":              shipmentWorkflowDTO,
	"ShipmentContractRateAccessorial": shipmentWorkflowDTO,
	"ShipmentHazmatZone":              shipmentWorkflowDTO,
	"ShipmentLoadingCommodity":        shipmentWorkflowDTO,
	"ShipmentLoadingRecommendation":   shipmentWorkflowDTO,
	"ShipmentStopDivider":             shipmentWorkflowDTO,
	// Assembled in Go from the documents table rather than stored; the parent
	// field is keel:"-", so there is no column to project.
	"ShipmentCommentAttachment": "GraphQL shipment comment DTO without column projection metadata",
	"JournalEntry":              journalDTO,
	"JournalEntryLine":          journalDTO,
	"JournalBatch":              journalDTO,
	"ResolvedCategoryRate":      costingDTO,
	"FuelCostResolution":        costingDTO,
	"ResolvedCostProfile":       costingDTO,
	"CostCategoryGLAccountLink": costingDTO,
	"TelematicsFormFieldValue":  telematicsFormDTO,
	"TelematicsFormSubmission":  telematicsFormDTO,
	"TelematicsFormMapping":     telematicsFormDTO,
	"TelematicsFormMappingItem": telematicsFormDTO,
	// Assembled in Go by unioning the worker and user rosters, so it maps to no
	// single table and fuzzy matching would otherwise bind it to an unrelated one.
	"PositionHolder": "GraphQL org structure DTO without column projection metadata",
	// Fuel and IFTA value objects: stored as JSONB on their parent, assembled in
	// Go, or served straight from the domain, so none of them own a table.
	"FuelPurchaseImportParsed":        "GraphQL fuel import DTO without column projection metadata",
	"FuelPurchaseImportSummary":       "GraphQL fuel import DTO stored as JSONB without column projection metadata",
	"FuelImportTemplate":              "GraphQL fuel import template DTO",
	"IFTAPeriod":                      "GraphQL IFTA period value object without column projection metadata",
	"IFTAFleetMPG":                    "GraphQL IFTA value object stored as JSONB without column projection metadata",
	"IFTAReturnProblem":               "GraphQL IFTA value object stored as JSONB without column projection metadata",
	"JurisdictionMilesBackfillResult": "GraphQL jurisdiction mileage backfill DTO",
}

func run(opts generatorOptions) error {
	resolved, err := resolveOptions(opts)
	if err != nil {
		return err
	}

	data, err := loadManifest(resolved.ManifestPath)
	if err != nil {
		return err
	}

	schema, err := loadSchema(resolved.SchemaDir)
	if err != nil {
		return err
	}

	gqlgen, err := loadGqlgenConfig(resolved.GqlgenPath)
	if err != nil {
		return err
	}

	modulePath, err := loadModulePath(resolved.GoModPath)
	if err != nil {
		return err
	}

	structs, err := loadGoStructs(resolved.DomainDir, modulePath)
	if err != nil {
		return err
	}

	output, err := generate(discovery{
		Schema:   schema,
		Manifest: data,
		Gqlgen:   gqlgen,
		Structs:  structs,
	})
	if err != nil {
		return err
	}

	//nolint:gosec // G306: generated Go source is meant to be world-readable
	return os.WriteFile(resolved.OutputPath, output, 0o644)
}

func resolveOptions(opts generatorOptions) (generatorOptions, error) {
	resolved := opts
	if resolved.ManifestPath != "" &&
		resolved.SchemaDir != "" &&
		resolved.OutputPath != "" &&
		resolved.GqlgenPath != "" &&
		resolved.DomainDir != "" &&
		resolved.GoModPath != "" {
		return resolved, nil
	}

	root, err := findModuleRoot(".")
	if err != nil {
		return generatorOptions{}, err
	}
	if resolved.ManifestPath == "" {
		resolved.ManifestPath = "projection.yml"
	}
	if resolved.SchemaDir == "" {
		resolved.SchemaDir = filepath.Join(root, "internal/api/graphql/schema")
	}
	if resolved.GqlgenPath == "" {
		resolved.GqlgenPath = filepath.Join(root, "gqlgen.yml")
	}
	if resolved.DomainDir == "" {
		resolved.DomainDir = filepath.Join(root, "internal", "modules", "*", "domain")
	}
	if resolved.GoModPath == "" {
		resolved.GoModPath = filepath.Join(root, "go.mod")
	}

	return resolved, nil
}

func findModuleRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolving current directory: %w", err)
	}

	for {
		if _, statErr := os.Stat(filepath.Join(current, "go.mod")); statErr == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("could not find go.mod from %q", start)
		}
		current = parent
	}
}

func loadManifest(path string) (manifest, error) {
	input, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, fmt.Errorf("reading manifest %q: %w", path, err)
	}
	if len(bytes.TrimSpace(input)) == 0 {
		return manifest{}, nil
	}

	var data manifest
	decoder := yaml.NewDecoder(bytes.NewReader(input))
	decoder.KnownFields(true)
	if err = decoder.Decode(&data); err != nil {
		return manifest{}, fmt.Errorf("decoding manifest %q: %w", path, err)
	}

	return data, nil
}

func loadGqlgenConfig(path string) (gqlgenConfig, error) {
	input, err := os.ReadFile(path)
	if err != nil {
		return gqlgenConfig{}, fmt.Errorf("reading gqlgen config %q: %w", path, err)
	}

	var cfg gqlgenConfig
	if err = yaml.Unmarshal(input, &cfg); err != nil {
		return gqlgenConfig{}, fmt.Errorf("decoding gqlgen config %q: %w", path, err)
	}

	return cfg, nil
}

func loadModulePath(path string) (string, error) {
	input, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading go.mod %q: %w", path, err)
	}
	mod := modfile.ModulePath(input)
	if mod == "" {
		return "", fmt.Errorf("go.mod %q does not declare a module path", path)
	}

	return mod, nil
}

func loadSchema(dir string) (*gqlast.Schema, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.graphqls"))
	if err != nil {
		return nil, fmt.Errorf("globbing schema files: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no GraphQL schema files found in %q", dir)
	}

	sort.Strings(matches)
	sources := make([]*gqlast.Source, 0, len(matches))
	for _, match := range matches {
		input, readErr := os.ReadFile(match)
		if readErr != nil {
			return nil, fmt.Errorf("reading schema file %q: %w", match, readErr)
		}
		sources = append(sources, &gqlast.Source{
			Name:  match,
			Input: string(input),
		})
	}

	schema, err := gqlparser.LoadSchema(sources...)
	if err != nil {
		return nil, fmt.Errorf("loading GraphQL schema: %w", err)
	}

	return schema, nil
}

func generate(data discovery) ([]byte, error) {
	if data.Schema == nil {
		return nil, errors.New("schema is required")
	}
	if data.Manifest.Aliases == nil {
		data.Manifest.Aliases = map[string]map[string]string{}
	}
	if data.Manifest.Virtuals == nil {
		data.Manifest.Virtuals = map[string][]string{}
	}
	if data.Manifest.Specials == nil {
		data.Manifest.Specials = map[string]map[string][]string{}
	}
	if data.Manifest.Gates == nil {
		data.Manifest.Gates = map[string]map[string]string{}
	}
	if data.Manifest.Always == nil {
		data.Manifest.Always = map[string][]string{}
	}
	if data.Manifest.ModelOverrides == nil {
		data.Manifest.ModelOverrides = map[string]string{}
	}

	selections, skipped, err := discoverProjectionTypes(data)
	if err != nil {
		return nil, err
	}
	data.Selections = selections
	data.Skipped = skipped

	if err = validateOverrideTypes(data); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(selections))
	for name := range selections {
		names = append(names, name)
	}
	sort.Strings(names)

	specs := make([]generatedSpec, 0, len(names))
	for _, name := range names {
		spec, buildErr := buildSpec(name, data)
		if buildErr != nil {
			return nil, buildErr
		}
		specs = append(specs, spec)
	}

	var buf bytes.Buffer
	tmpl := template.Must(template.New("specs").Funcs(template.FuncMap{
		"quote": quote,
	}).Parse(specsTemplate))
	if err = tmpl.Execute(&buf, specs); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("formatting generated specs: %w\n%s", err, buf.String())
	}

	return formatted, nil
}

func discoverProjectionTypes(
	data discovery,
) (map[string]typeSelection, map[string]string, error) {
	objectNames := schemaObjectNames(data.Schema)
	selections := make(map[string]typeSelection, len(objectNames))
	skipped := make(map[string]string, len(objectNames))

	for _, name := range objectNames {
		if reason, ok := autoSkipReason(name, data); ok {
			skipped[name] = reason
			continue
		}

		selection, err := selectProjectionType(name, data)
		if err != nil {
			return nil, nil, err
		}
		if selection.TypeName == "" {
			skipped[name] = "no keel table model with columns was discovered"
			continue
		}
		selections[name] = selection
	}

	return selections, skipped, nil
}

func schemaObjectNames(schema *gqlast.Schema) []string {
	names := make([]string, 0, len(schema.Types))
	for name, def := range schema.Types {
		if def.Kind != gqlast.Object || strings.HasPrefix(name, "__") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

func autoSkipReason(name string, data discovery) (string, bool) {
	if reason, ok := nonProjectionObjects[name]; ok {
		return reason, true
	}

	switch name {
	case "Mutation", "PageInfo", "Query":
		return "GraphQL operation or pagination wrapper", true
	}

	suffixes := []string{
		"Analytics",
		"Connection",
		"Context",
		"Counts",
		"DataPoint",
		"Edge",
		"Policy",
		"Readiness",
		"Reference",
		"Requirement",
		"Response",
		"Result",
		"Summary",
		"Validation",
		"Warning",
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			// A wrapper/DTO suffix is only a heuristic. A type with an explicit
			// gqlgen model binding or modelOverride is a real projected entity
			// that merely shares a suffix (e.g. EdiConnection); never skip it.
			if hasExplicitModelBinding(name, data) {
				return "", false
			}
			return "GraphQL wrapper or DTO suffix " + suffix, true
		}
	}

	return "", false
}

func hasExplicitModelBinding(name string, data discovery) bool {
	if data.Manifest.ModelOverrides[name] != "" {
		return true
	}

	return firstModelBinding(data.Gqlgen.Models[name]) != ""
}

func selectProjectionType(typeName string, data discovery) (typeSelection, error) {
	if override := data.Manifest.ModelOverrides[typeName]; override != "" {
		return selectionForModelOverride(typeName, override, data)
	}

	if binding := firstModelBinding(data.Gqlgen.Models[typeName]); binding != "" {
		selection, err := selectionForModelOverride(typeName, binding, data)
		if err != nil {
			if isNonProjectionBindingError(err) {
				return typeSelection{}, nil
			}
			return typeSelection{}, fmt.Errorf("type %q gqlgen model binding: %w", typeName, err)
		}
		return selection, nil
	}

	selection, ok, err := exactStructSelection(typeName, data)
	if err != nil {
		return typeSelection{}, err
	}
	if ok {
		return selection, nil
	}

	return coverageStructSelection(typeName, data)
}

func isNonProjectionBindingError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "was not found in parsed Go structs") ||
		strings.Contains(message, "is not a keel table model")
}

func firstModelBinding(model gqlgenModel) string {
	for _, candidate := range model.Model {
		if isGoStructBinding(candidate) {
			return candidate
		}
	}

	return ""
}

func isGoStructBinding(binding string) bool {
	if binding == "" || strings.HasPrefix(binding, "github.com/99designs/gqlgen/") {
		return false
	}

	last := binding[strings.LastIndex(binding, "/")+1:]
	return strings.Contains(last, ".")
}

func selectionForModelOverride(
	typeName string,
	model string,
	data discovery,
) (typeSelection, error) {
	fullName := normalizeModelName(model)
	st, ok := data.Structs[fullName]
	if !ok {
		return typeSelection{}, fmt.Errorf("model %q was not found in parsed Go structs", model)
	}

	columns, ok := tableColumns(&st)
	if !ok {
		return typeSelection{}, fmt.Errorf("model %q is not a keel table model with columns", model)
	}

	return typeSelection{
		TypeName: typeName,
		Struct:   st,
		Columns:  columns,
	}, nil
}

func tableColumns(st *goStruct) (tableColumnMap, bool) {
	if !st.IsEntity {
		return tableColumnMap{}, false
	}

	values := make(map[string]string, len(st.Fields))
	for _, field := range st.Fields {
		if field.IsColumn {
			values[field.JSONName] = field.ColumnName
		}
	}
	if len(values) == 0 {
		return tableColumnMap{}, false
	}

	return tableColumnMap{Values: values}, true
}

func normalizeModelName(model string) string {
	return strings.TrimPrefix(strings.TrimSpace(model), "*")
}

func exactStructSelection(typeName string, data discovery) (typeSelection, bool, error) {
	matches := make([]typeSelection, 0, 1)
	for _, st := range data.Structs {
		if st.Name != typeName {
			continue
		}
		columns, ok := tableColumns(&st)
		if !ok {
			continue
		}

		matches = append(matches, typeSelection{
			TypeName: typeName,
			Struct:   st,
			Columns:  columns,
		})
	}

	if len(matches) == 0 {
		return typeSelection{}, false, nil
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Struct.FullName < matches[j].Struct.FullName
	})
	if len(matches) > 1 {
		return typeSelection{}, false, fmt.Errorf(
			"type %q has ambiguous exact Go model matches %q and %q; add modelOverrides.%s",
			typeName,
			matches[0].Struct.FullName,
			matches[1].Struct.FullName,
			typeName,
		)
	}

	return matches[0], true, nil
}

func coverageStructSelection(typeName string, data discovery) (typeSelection, error) {
	type candidate struct {
		selection typeSelection
		score     int
		ratio     float64
	}

	def := data.Schema.Types[typeName]
	candidates := []candidate{}
	for _, st := range data.Structs {
		columns, ok := tableColumns(&st)
		if !ok {
			continue
		}

		score, ratio := fieldCoverage(def, st, data.Manifest.Aliases[typeName])
		if score < minCoverageFields || ratio < minCoverageRatio {
			continue
		}

		candidates = append(candidates, candidate{
			selection: typeSelection{
				TypeName: typeName,
				Struct:   st,
				Columns:  columns,
			},
			score: score,
			ratio: ratio,
		})
	}
	if len(candidates) == 0 {
		return typeSelection{}, nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].ratio != candidates[j].ratio {
			return candidates[i].ratio > candidates[j].ratio
		}
		return candidates[i].selection.Struct.FullName < candidates[j].selection.Struct.FullName
	})

	best := candidates[0]
	if len(candidates) > 1 &&
		candidates[1].score == best.score &&
		candidates[1].ratio == best.ratio {
		return typeSelection{}, fmt.Errorf(
			"type %q has ambiguous Go model matches %q and %q; add modelOverrides.%s",
			typeName,
			best.selection.Struct.FullName,
			candidates[1].selection.Struct.FullName,
			typeName,
		)
	}

	return best.selection, nil
}

func fieldCoverage(
	def *gqlast.Definition,
	st goStruct,
	aliases map[string]string,
) (int, float64) {
	if def == nil || len(def.Fields) == 0 {
		return 0, 0
	}

	var score int
	for _, field := range def.Fields {
		if _, ok := st.Fields[field.Name]; ok {
			score++
			continue
		}
		if key := aliases[field.Name]; key != "" {
			if _, ok := st.Fields[key]; ok {
				score++
			}
		}
	}

	return score, float64(score) / float64(len(def.Fields))
}

func validateOverrideTypes(data discovery) error {
	validateTypeMap := func(groupName string, values map[string]map[string]string) error {
		for typeName := range values {
			if _, ok := data.Selections[typeName]; !ok {
				return fmt.Errorf(
					"%s override references non-projection type %q",
					groupName,
					typeName,
				)
			}
		}
		return nil
	}
	validateTypeList := func(groupName string, values map[string][]string) error {
		for typeName := range values {
			if _, ok := data.Selections[typeName]; !ok {
				return fmt.Errorf(
					"%s override references non-projection type %q",
					groupName,
					typeName,
				)
			}
		}
		return nil
	}

	if err := validateTypeMap("aliases", data.Manifest.Aliases); err != nil {
		return err
	}
	if err := validateTypeList("virtuals", data.Manifest.Virtuals); err != nil {
		return err
	}
	if err := validateSpecialOverrideTypes(data); err != nil {
		return err
	}
	if err := validateTypeMap("gates", data.Manifest.Gates); err != nil {
		return err
	}
	for typeName := range data.Manifest.Always {
		if _, ok := data.Selections[typeName]; !ok {
			return fmt.Errorf("always override references non-projection type %q", typeName)
		}
	}
	for typeName := range data.Manifest.ModelOverrides {
		if _, ok := data.Selections[typeName]; !ok {
			return fmt.Errorf("modelOverrides override references non-projection type %q", typeName)
		}
	}

	if err := validateVirtualFields(data); err != nil {
		return err
	}
	if err := validateSpecialFields(data); err != nil {
		return err
	}

	return nil
}

func validateSpecialOverrideTypes(data discovery) error {
	for typeName := range data.Manifest.Specials {
		if _, ok := data.Selections[typeName]; !ok {
			return fmt.Errorf("specials override references non-projection type %q", typeName)
		}
	}

	return nil
}

func validateVirtualFields(data discovery) error {
	for typeName, fields := range data.Manifest.Virtuals {
		def := data.Schema.Types[typeName]
		seen := make(map[string]struct{}, len(fields))
		for _, field := range fields {
			if field == "" {
				return fmt.Errorf("virtuals.%s contains an empty field", typeName)
			}
			if _, ok := seen[field]; ok {
				return fmt.Errorf("virtuals.%s lists field %q more than once", typeName, field)
			}
			seen[field] = struct{}{}
			if !hasSchemaField(def, field) {
				return fmt.Errorf("virtuals.%s field %q does not exist in schema", typeName, field)
			}
		}
	}

	return nil
}

func validateSpecialFields(data discovery) error {
	for typeName, specials := range data.Manifest.Specials {
		def := data.Schema.Types[typeName]
		// Runtime special keys are exclusive: one GraphQL field cannot drive
		// both a same-name virtual and a grouped special handler.
		virtuals := stringSet(data.Manifest.Virtuals[typeName])
		seen := map[string]string{}

		for specialKey, fields := range specials {
			if specialKey == "" {
				return fmt.Errorf("specials.%s contains an empty special key", typeName)
			}
			seenForKey := make(map[string]struct{}, len(fields))
			for _, field := range fields {
				if field == "" {
					return fmt.Errorf(
						"specials.%s.%s contains an empty field",
						typeName,
						specialKey,
					)
				}
				if _, ok := seenForKey[field]; ok {
					return fmt.Errorf(
						"specials.%s.%s lists field %q more than once",
						typeName,
						specialKey,
						field,
					)
				}
				seenForKey[field] = struct{}{}
				if _, ok := virtuals[field]; ok {
					return fmt.Errorf(
						"type %q field %q cannot be declared in both virtuals and specials",
						typeName,
						field,
					)
				}
				if prior, ok := seen[field]; ok {
					return fmt.Errorf(
						"type %q field %q is listed under multiple special keys %q and %q",
						typeName,
						field,
						prior,
						specialKey,
					)
				}
				seen[field] = specialKey
				if !hasSchemaField(def, field) {
					return fmt.Errorf(
						"specials.%s.%s field %q does not exist in schema",
						typeName,
						specialKey,
						field,
					)
				}
			}
		}
	}

	return nil
}

func hasSchemaField(def *gqlast.Definition, name string) bool {
	if def == nil {
		return false
	}
	for _, field := range def.Fields {
		if field.Name == name {
			return true
		}
	}

	return false
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}

	return set
}

func containsString(values []string, target string) bool {
	return slices.Contains(values, target)
}

func buildSpec(typeName string, data discovery) (generatedSpec, error) {
	selection := data.Selections[typeName]
	def := data.Schema.Types[typeName]
	if def == nil {
		return generatedSpec{}, fmt.Errorf("unknown schema type %q", typeName)
	}

	always, err := buildAlwaysColumns(typeName, selection, data.Manifest)
	if err != nil {
		return generatedSpec{}, err
	}

	fields := make([]generatedField, 0, len(def.Fields))
	for _, field := range def.Fields {
		generated, fieldErr := buildField(fieldBuildParams{
			TypeName:  typeName,
			Field:     field,
			Discovery: data,
			Selection: selection,
		})
		if fieldErr != nil {
			return generatedSpec{}, fieldErr
		}
		fields = append(fields, generated)
	}

	return generatedSpec{
		Name:          typeName,
		AlwaysColumns: always,
		Fields:        fields,
	}, nil
}

func buildAlwaysColumns(
	typeName string,
	selection typeSelection,
	data manifest,
) ([]string, error) {
	keys := []string{}
	if _, ok := selection.Columns.Values["id"]; ok {
		keys = append(keys, "id")
	}
	if _, ok := selection.Columns.Values["createdAt"]; ok {
		keys = append(keys, "createdAt")
	}
	keys = append(keys, data.Always[typeName]...)

	seen := make(map[string]struct{}, len(keys))
	columns := make([]string, 0, len(keys))
	for _, field := range keys {
		key := columnKey(field, data.Aliases[typeName])
		column, ok := selection.Columns.Values[key]
		if !ok {
			return nil, fmt.Errorf(
				"type %q always field %q uses missing column key %q",
				typeName,
				field,
				key,
			)
		}
		if _, ok = seen[column]; ok {
			continue
		}
		seen[column] = struct{}{}
		columns = append(columns, column)
	}

	return columns, nil
}

type fieldBuildParams struct {
	TypeName  string
	Field     *gqlast.FieldDefinition
	Discovery discovery
	Selection typeSelection
}

func buildField(params fieldBuildParams) (generatedField, error) {
	field := params.Field
	generated := generatedField{Name: field.Name}
	typeName := params.TypeName
	manifest := params.Discovery.Manifest

	// virtuals are same-name runtime specials; specials can group several
	// GraphQL fields behind one runtime Selection.HasSpecial key.
	if containsString(manifest.Virtuals[typeName], field.Name) {
		generated.Special = field.Name
		return generated, nil
	}

	if special, ok := specialKeyForField(manifest.Specials[typeName], field.Name); ok {
		generated.Special = special
		return generated, nil
	}

	key := columnKey(field.Name, manifest.Aliases[typeName])
	if column, ok := params.Selection.Columns.Values[key]; ok {
		generated.Column = column
		return generated, nil
	}

	if relation, ok := inferRelation(params); ok {
		generated.Relation = &generatedRelation{
			Target: relation.Target,
			Gate:   relation.Gate,
		}
		generated.Column = relation.Column
		return generated, nil
	}

	const unresolvedFieldMessage = "type %q field %q is neither a keel column, inferred relation, " +
		"nor configured projection override; add aliases.%s.%s, virtuals.%s list entry, " +
		"specials.%s.<specialKey> list entry, or modelOverrides.%s"

	return generatedField{}, fmt.Errorf(
		unresolvedFieldMessage,
		typeName,
		field.Name,
		typeName,
		field.Name,
		typeName,
		typeName,
		typeName,
	)
}

func specialKeyForField(specials map[string][]string, fieldName string) (string, bool) {
	for specialKey, fields := range specials {
		if slices.Contains(fields, fieldName) {
			return specialKey, true
		}
	}

	return "", false
}

type inferredRelation struct {
	Target string
	Gate   string
	Column string
}

func inferRelation(params fieldBuildParams) (inferredRelation, bool) {
	fieldName := params.Field.Name
	structField, ok := params.Selection.Struct.Fields[fieldName]
	if !ok || !structField.IsRelation {
		return inferredRelation{}, false
	}

	target := namedType(params.Field.Type)
	if _, ok = params.Discovery.Selections[target]; !ok {
		return inferredRelation{}, false
	}

	column := ""
	switch structField.RelationKind {
	case relationBelongsTo:
		if slices.Contains(
			slices.Collect(maps.Values(params.Selection.Columns.Values)),
			structField.RelationLocal,
		) {
			column = structField.RelationLocal
		}
	case "":
		column = params.Selection.Columns.Values[fieldName+"Id"]
	}

	return inferredRelation{
		Target: target,
		Gate:   params.Discovery.Manifest.Gates[params.TypeName][fieldName],
		Column: column,
	}, true
}

func columnKey(field string, aliases map[string]string) string {
	if key, ok := aliases[field]; ok {
		return key
	}

	return field
}

func namedType(t *gqlast.Type) string {
	if t == nil {
		return ""
	}

	return t.Name()
}

func loadGoStructs(pattern, modulePath string) (map[string]goStruct, error) {
	roots, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("expanding %q: %w", pattern, err)
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("no domain directory matches %q", pattern)
	}

	structs := map[string]goStruct{}
	for _, root := range roots {
		if err = loadGoStructsFrom(root, modulePath, structs); err != nil {
			return nil, err
		}
	}

	return structs, nil
}

type goStructLoader struct {
	fset       *token.FileSet
	root       string
	modulePath string
	structs    map[string]goStruct
}

func loadGoStructsFrom(root, modulePath string, structs map[string]goStruct) error {
	loader := goStructLoader{
		fset:       token.NewFileSet(),
		root:       root,
		modulePath: modulePath,
		structs:    structs,
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !isGoSourceFile(path) {
			return nil
		}

		return loader.loadFile(path)
	})
	if err != nil {
		return fmt.Errorf("walking Go domain structs in %q: %w", root, err)
	}

	return nil
}

func isGoSourceFile(path string) bool {
	return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}

func (l goStructLoader) loadFile(path string) error {
	file, err := parser.ParseFile(l.fset, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parsing %q: %w", path, err)
	}

	packagePath, err := packagePathForFile(l.root, l.modulePath, path)
	if err != nil {
		return err
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			st := buildGoStruct(packagePath, file.Name.Name, typeSpec.Name.Name, structType)
			l.structs[st.FullName] = st
		}
	}

	return nil
}

func buildGoStruct(packagePath, packageName, name string, structType *ast.StructType) goStruct {
	st := goStruct{
		PackagePath: packagePath,
		PackageName: packageName,
		Name:        name,
		FullName:    packagePath + "." + name,
		Fields:      map[string]goField{},
	}
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			if isKeelTableEmbed(field) {
				st.IsEntity = true
			}
			continue
		}
		parsed, ok := parseGoStructField(field)
		if !ok {
			continue
		}
		st.Fields[parsed.JSONName] = parsed
	}

	return st
}

func packagePathForFile(root, modulePath, path string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolving root %q: %w", root, err)
	}
	pathAbs, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return "", fmt.Errorf("resolving path %q: %w", path, err)
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return "", fmt.Errorf("building package path for %q: %w", path, err)
	}

	base := filepath.ToSlash(filepath.Clean(rootAbs))
	domainIndex := strings.LastIndex(base, "/internal/")
	if domainIndex == -1 {
		return "", fmt.Errorf("domain root %q must be inside the module's internal directory", root)
	}
	domainImportRoot := strings.TrimSuffix(modulePath+base[domainIndex:], "/")
	if rel == "." {
		return domainImportRoot, nil
	}

	return domainImportRoot + "/" + filepath.ToSlash(rel), nil
}

func isKeelTableEmbed(field *ast.Field) bool {
	selector, ok := field.Type.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Table" {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)

	return ok && pkg.Name == "keel"
}

const (
	relationBelongsTo = "belongs"
	keelTagSkip       = "-"
)

var keelRelationKinds = []string{relationBelongsTo, "one", "many", "m2m", "pick"}

func parseGoStructField(field *ast.Field) (goField, bool) {
	name := field.Names[0].Name
	if !isExported(name) {
		return goField{}, false
	}

	jsonName := lowerCamel(name)
	var keelTag string
	tagged := false
	if field.Tag != nil {
		tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
		if value := tag.Get("json"); value != "" {
			jsonName = strings.Split(value, ",")[0]
		}
		keelTag, tagged = tag.Lookup("keel")
	}
	head := strings.TrimSpace(strings.Split(keelTag, ",")[0])
	kind, local, isRelation := keelRelation(head)
	if !tagged && isRelationType(field.Type) {
		isRelation = true
	}
	if jsonName == "-" && isRelation {
		jsonName = lowerCamel(name)
	}
	if jsonName == "" || jsonName == "-" {
		return goField{}, false
	}

	parsed := goField{
		JSONName: jsonName,
		GoName:   name,
		TypeName: exprName(field.Type),
	}

	switch {
	case isRelation:
		parsed.IsRelation = true
		parsed.RelationKind = kind
		parsed.RelationLocal = local
	case head != "" && head != keelTagSkip && !strings.Contains(head, ":"):
		parsed.IsColumn = true
		parsed.ColumnName = head
	}

	return parsed, true
}

func isRelationType(expr ast.Expr) bool {
	switch value := expr.(type) {
	case *ast.StarExpr:
		return isNamedStruct(value.X)
	case *ast.ArrayType:
		if star, ok := value.Elt.(*ast.StarExpr); ok {
			return isNamedStruct(star.X)
		}
		return isNamedStruct(value.Elt)
	}

	return false
}

func isNamedStruct(expr ast.Expr) bool {
	switch value := expr.(type) {
	case *ast.SelectorExpr:
		return true
	case *ast.Ident:
		return value.Obj != nil || isExported(value.Name)
	}

	return false
}

func keelRelation(head string) (kind, local string, ok bool) {
	prefix, value, found := strings.Cut(head, ":")
	if !found || !slices.Contains(keelRelationKinds, prefix) {
		return "", "", false
	}
	if prefix != relationBelongsTo {
		return prefix, "", true
	}

	return prefix, strings.TrimSpace(strings.Split(value, "|")[0]), true
}

func exprName(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return exprName(value.X)
	case *ast.ArrayType:
		return exprName(value.Elt)
	case *ast.SelectorExpr:
		return value.Sel.Name
	}

	return ""
}

func lowerCamel(value string) string {
	if value == "" {
		return ""
	}
	if value == strings.ToUpper(value) {
		return strings.ToLower(value)
	}
	runes := []rune(value)
	runes[0] = unicode.ToLower(runes[0])

	return string(runes)
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	r := []rune(name)[0]

	return unicode.IsUpper(r)
}

func quote(value string) string {
	return fmt.Sprintf("%q", value)
}
