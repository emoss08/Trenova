package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/infrastructure/database/structparse"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/stringutils"
)

type builder struct {
	manifest          *Manifest
	models            map[string]*structparse.Model
	modelsByTable     map[string]*structparse.Model
	enums             map[string]map[string][]string
	structToEntityKey map[string]string
	permissions       *permission.Registry
	warnings          []string
}

func newBuilder(manifest *Manifest, domainDir, domaintypesDir string) (*builder, error) {
	b := &builder{
		manifest:          manifest,
		models:            make(map[string]*structparse.Model),
		modelsByTable:     make(map[string]*structparse.Model),
		enums:             make(map[string]map[string][]string),
		structToEntityKey: make(map[string]string),
		permissions:       permission.NewRegistry(),
	}

	entries, err := os.ReadDir(domainDir)
	if err != nil {
		return nil, fmt.Errorf("read domain directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if err = b.indexPackage(filepath.Join(domainDir, entry.Name())); err != nil {
			return nil, err
		}
	}

	if domaintypesDir != "" {
		if err = b.indexPackage(domaintypesDir); err != nil {
			return nil, err
		}
	}

	for key, em := range manifest.Entities {
		b.structToEntityKey[em.Struct] = key
	}

	return b, nil
}

func (b *builder) indexPackage(dir string) error {
	result, err := structparse.ParsePackage(dir)
	if err != nil {
		return fmt.Errorf("parse package %s: %w", dir, err)
	}

	pkgName := filepath.Base(dir)

	for i := range result.Models {
		model := &result.Models[i]
		b.models[model.PackageName+"."+model.StructName] = model
		b.modelsByTable[model.TableName] = model
	}

	if len(result.Enums) > 0 {
		if result.Models != nil && len(result.Models) > 0 {
			pkgName = result.Models[0].PackageName
		}
		enumSet := b.enums[pkgName]
		if enumSet == nil {
			enumSet = make(map[string][]string)
			b.enums[pkgName] = enumSet
		}
		for _, enum := range result.Enums {
			enumSet[enum.TypeName] = enum.Values
		}
	}

	return nil
}

func (b *builder) Build() (*reportcatalog.Catalog, error) {
	catalog := &reportcatalog.Catalog{
		Entities: make([]reportcatalog.Entity, 0, len(b.manifest.Entities)),
	}

	for _, key := range b.manifest.SortedEntityKeys() {
		entity, err := b.buildEntity(key, b.manifest.Entities[key])
		if err != nil {
			return nil, fmt.Errorf("entity %q: %w", key, err)
		}
		catalog.Entities = append(catalog.Entities, *entity)
	}

	return catalog, nil
}

func (b *builder) buildEntity(key string, em *EntityManifest) (*reportcatalog.Entity, error) {
	if em.Struct == "" {
		return nil, fmt.Errorf("missing struct reference")
	}
	model, ok := b.models[em.Struct]
	if !ok {
		return nil, fmt.Errorf("struct %q not found in domain packages", em.Struct)
	}

	if em.Resource == "" {
		return nil, fmt.Errorf("missing resource")
	}
	if !b.permissions.HasResource(em.Resource) {
		return nil, fmt.Errorf("resource %q is not registered in the permission registry", em.Resource)
	}

	tenant, err := tenantColumns(model)
	if err != nil {
		return nil, err
	}

	if em.OwnershipColumn != "" && !hasColumn(model, em.OwnershipColumn) {
		return nil, fmt.Errorf("ownershipColumn %q does not exist on %s", em.OwnershipColumn, em.Struct)
	}

	fieldsByJSON := make(map[string]*structparse.Field, len(model.Fields))
	for i := range model.Fields {
		if model.Fields[i].JSONName != "" {
			fieldsByJSON[model.Fields[i].JSONName] = &model.Fields[i]
		}
	}

	for manifestField := range em.Fields {
		if _, exists := fieldsByJSON[manifestField]; !exists {
			return nil, fmt.Errorf("fields entry %q does not match any JSON field on %s", manifestField, em.Struct)
		}
	}
	excluded := make(map[string]bool, len(em.ExcludeFields))
	for _, name := range em.ExcludeFields {
		if _, exists := fieldsByJSON[name]; !exists {
			return nil, fmt.Errorf("excludeFields entry %q does not match any JSON field on %s", name, em.Struct)
		}
		excluded[name] = true
	}

	label := em.Label
	if label == "" {
		label = stringutils.HumanizeCamelCase(model.StructName)
	}
	pluralLabel := em.PluralLabel
	if pluralLabel == "" {
		pluralLabel = label + "s"
	}

	entity := &reportcatalog.Entity{
		Key:      key,
		Resource: permission.Resource(em.Resource),
		Table: buncolgen.TableInfo{
			Name:       model.TableName,
			Alias:      model.Alias,
			PrimaryKey: model.PKColumns(),
		},
		Label:           label,
		PluralLabel:     pluralLabel,
		Description:     em.Description,
		Category:        em.Category,
		Tenant:          tenant,
		OwnershipColumn: em.OwnershipColumn,
	}

	resourceDef, _ := b.permissions.Get(em.Resource)

	for i := range model.Fields {
		field := &model.Fields[i]
		if field.JSONName == "" || field.IsScanOnly || excluded[field.JSONName] {
			continue
		}

		built, buildErr := b.buildField(model, field, em.Fields[field.JSONName])
		if buildErr != nil {
			return nil, fmt.Errorf("field %q: %w", field.JSONName, buildErr)
		}
		b.warnUnclassifiedSensitivity(resourceDef, em.Resource, field.JSONName)
		entity.Fields = append(entity.Fields, *built)
	}

	if len(entity.Fields) == 0 {
		return nil, fmt.Errorf("no reportable fields after exclusions")
	}

	for _, edgeName := range em.SortedEdgeNames() {
		edge, edgeErr := b.buildEdge(key, model, edgeName, em.Edges[edgeName])
		if edgeErr != nil {
			return nil, fmt.Errorf("edge %q: %w", edgeName, edgeErr)
		}
		entity.Edges = append(entity.Edges, *edge)
	}

	return entity, nil
}

func (b *builder) buildField(
	model *structparse.Model,
	field *structparse.Field,
	fm *FieldManifest,
) (*reportcatalog.Field, error) {
	fieldType, enumValues := b.inferFieldType(model.PackageName, field)

	if fm != nil && fm.Type != "" {
		override := reportcatalog.FieldType(fm.Type)
		if !override.IsValid() {
			return nil, fmt.Errorf("invalid type override %q", fm.Type)
		}
		if override == reportcatalog.FieldEnum && len(enumValues) == 0 {
			return nil, fmt.Errorf("type forced to enum but no enum values resolvable for Go type %q", field.GoType)
		}
		fieldType = override
	}

	label := ""
	description := ""
	format := reportcatalog.FormatNone
	var enumLabels map[string]string
	if fm != nil {
		label = fm.Label
		description = fm.Description
		format = reportcatalog.FormatHint(fm.Format)
		if !isValidFormat(format) {
			return nil, fmt.Errorf("invalid format %q", fm.Format)
		}
		enumLabels = fm.EnumLabels
	}
	if label == "" {
		label = stringutils.HumanizeCamelCase(field.JSONName)
	}

	for value := range enumLabels {
		if !containsString(enumValues, value) {
			return nil, fmt.Errorf("enumLabels entry %q is not a value of %s", value, field.GoType)
		}
	}

	var enums []reportcatalog.EnumValue
	if fieldType == reportcatalog.FieldEnum {
		enums = make([]reportcatalog.EnumValue, 0, len(enumValues))
		for _, value := range enumValues {
			enumLabel := enumLabels[value]
			if enumLabel == "" {
				enumLabel = stringutils.HumanizeCamelCase(value)
			}
			enums = append(enums, reportcatalog.EnumValue{Value: value, Label: enumLabel})
		}
	}

	aggregations := defaultAggregations(fieldType)
	if fm != nil && len(fm.Aggregations) > 0 {
		restricted := make([]reportcatalog.Aggregation, 0, len(fm.Aggregations))
		for _, raw := range fm.Aggregations {
			agg := reportcatalog.Aggregation(raw)
			if !agg.IsValid() {
				return nil, fmt.Errorf("invalid aggregation %q", raw)
			}
			if !containsAggregation(aggregations, agg) {
				return nil, fmt.Errorf(
					"aggregation %q is not legal for field type %q (manifest may only restrict)",
					raw, fieldType,
				)
			}
			restricted = append(restricted, agg)
		}
		aggregations = restricted
	}

	filterable := fieldType != reportcatalog.FieldJSON
	groupable := fieldType != reportcatalog.FieldJSON && fieldType != reportcatalog.FieldDecimal
	if fm != nil {
		if fm.Filterable != nil {
			filterable = *fm.Filterable
		}
		if fm.Groupable != nil {
			groupable = *fm.Groupable
		}
	}
	if fieldType == reportcatalog.FieldJSON && (filterable || groupable) {
		return nil, fmt.Errorf("json fields cannot be filterable or groupable")
	}

	return &reportcatalog.Field{
		Key:          field.JSONName,
		Column:       buncolgen.NewColumn(field.ColumnName, model.Alias),
		Label:        label,
		Description:  description,
		Type:         fieldType,
		Format:       format,
		Nullable:     !(field.IsNotNull || field.IsPK),
		EnumValues:   enums,
		Aggregations: aggregations,
		Filterable:   filterable,
		Groupable:    groupable,
	}, nil
}

func (b *builder) inferFieldType(
	pkgName string,
	field *structparse.Field,
) (reportcatalog.FieldType, []string) {
	goType := strings.TrimPrefix(field.GoType, "*")

	if field.IsArray || strings.HasPrefix(goType, "[]") || strings.HasPrefix(goType, "map[") ||
		strings.EqualFold(field.SQLType, "JSONB") {
		return reportcatalog.FieldJSON, nil
	}

	switch goType {
	case "pulid.ID":
		return reportcatalog.FieldRef, nil
	case "bool":
		return reportcatalog.FieldBool, nil
	case "decimal.Decimal", "decimal.NullDecimal":
		return reportcatalog.FieldDecimal, nil
	case "string":
		return reportcatalog.FieldString, nil
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		if looksLikeEpoch(field) {
			return reportcatalog.FieldEpoch, nil
		}
		return reportcatalog.FieldInt, nil
	case "float32", "float64":
		return reportcatalog.FieldDecimal, nil
	}

	if strings.HasPrefix(field.SQLType, "NUMERIC") {
		return reportcatalog.FieldDecimal, nil
	}

	if values := b.resolveEnumValues(pkgName, goType); len(values) > 0 {
		return reportcatalog.FieldEnum, values
	}

	if strings.HasSuffix(field.SQLType, "_enum") {
		b.warnings = append(b.warnings, fmt.Sprintf(
			"field %s.%s has SQL enum type %q but Go type %q has no resolvable values — treating as string",
			pkgName, field.GoName, field.SQLType, field.GoType,
		))
	}

	return reportcatalog.FieldString, nil
}

func (b *builder) resolveEnumValues(pkgName, goType string) []string {
	if pkg, typeName, found := strings.Cut(goType, "."); found {
		return b.enums[pkg][typeName]
	}
	return b.enums[pkgName][goType]
}

func looksLikeEpoch(field *structparse.Field) bool {
	for _, suffix := range []string{"_at", "_date", "_expiry", "_timestamp", "_time"} {
		if strings.HasSuffix(field.ColumnName, suffix) {
			return true
		}
	}
	return false
}

func (b *builder) buildEdge(
	sourceKey string,
	model *structparse.Model,
	edgeName string,
	em *EdgeManifest,
) (*reportcatalog.Edge, error) {
	if rel := relationByJSONName(model, edgeName); rel != nil {
		return b.buildRelationEdge(sourceKey, model.PackageName, edgeName, rel, em)
	}
	if m2m := m2mByJSONName(model, edgeName); m2m != nil {
		return b.buildM2MEdge(sourceKey, model.PackageName, edgeName, m2m, em)
	}
	return nil, fmt.Errorf("no relation with JSON name %q on %s", edgeName, model.StructName)
}

func (b *builder) buildRelationEdge(
	sourceKey, sourcePkg, edgeName string,
	rel *structparse.Relation,
	em *EdgeManifest,
) (*reportcatalog.Edge, error) {
	targetKey, err := b.resolveTargetEntity(sourcePkg, rel.GoType)
	if err != nil {
		return nil, err
	}

	cardinality := reportcatalog.CardinalityOne
	if rel.Kind == structparse.RelationHasMany {
		cardinality = reportcatalog.CardinalityMany
	}

	if len(rel.JoinPairs) == 0 {
		return nil, fmt.Errorf("relation %q has no join pairs", edgeName)
	}

	join := make([]reportcatalog.JoinPair, 0, len(rel.JoinPairs))
	for _, jp := range rel.JoinPairs {
		join = append(join, reportcatalog.JoinPair{Local: jp.Local, Remote: jp.Remote})
	}

	label := ""
	traversable := true
	if em != nil {
		label = em.Label
		traversable = em.IsTraversable()
	}
	if label == "" {
		label = stringutils.HumanizeCamelCase(edgeName)
	}

	return &reportcatalog.Edge{
		Name:        edgeName,
		Label:       label,
		Source:      sourceKey,
		Target:      targetKey,
		Cardinality: cardinality,
		Join:        join,
		Traversable: traversable,
	}, nil
}

func (b *builder) buildM2MEdge(
	sourceKey, sourcePkg, edgeName string,
	m2m *structparse.M2MRelation,
	em *EdgeManifest,
) (*reportcatalog.Edge, error) {
	targetKey, err := b.resolveTargetEntity(sourcePkg, m2m.GoType)
	if err != nil {
		return nil, err
	}

	through, ok := b.modelsByTable[m2m.ThroughTable]
	if !ok {
		return nil, fmt.Errorf("m2m through table %q has no parsed model", m2m.ThroughTable)
	}

	sourceRelName, targetRelName, found := strings.Cut(m2m.JoinSpec, "=")
	if !found {
		return nil, fmt.Errorf("m2m join spec %q is not in Source=Target form", m2m.JoinSpec)
	}

	sourceRel, ok := through.Relation(sourceRelName)
	if !ok {
		return nil, fmt.Errorf("through model %q has no relation %q", through.StructName, sourceRelName)
	}
	targetRel, ok := through.Relation(targetRelName)
	if !ok {
		return nil, fmt.Errorf("through model %q has no relation %q", through.StructName, targetRelName)
	}

	sourceJoin := make([]reportcatalog.JoinPair, 0, len(sourceRel.JoinPairs))
	for _, jp := range sourceRel.JoinPairs {
		sourceJoin = append(sourceJoin, reportcatalog.JoinPair{Local: jp.Remote, Remote: jp.Local})
	}
	targetJoin := make([]reportcatalog.JoinPair, 0, len(targetRel.JoinPairs))
	for _, jp := range targetRel.JoinPairs {
		targetJoin = append(targetJoin, reportcatalog.JoinPair{Local: jp.Local, Remote: jp.Remote})
	}

	throughTenant, err := tenantColumns(through)
	if err != nil {
		return nil, fmt.Errorf("through table %q: %w", m2m.ThroughTable, err)
	}

	label := ""
	traversable := true
	if em != nil {
		label = em.Label
		traversable = em.IsTraversable()
	}
	if label == "" {
		label = stringutils.HumanizeCamelCase(edgeName)
	}

	return &reportcatalog.Edge{
		Name:        edgeName,
		Label:       label,
		Source:      sourceKey,
		Target:      targetKey,
		Cardinality: reportcatalog.CardinalityM2M,
		Through: &reportcatalog.ThroughJoin{
			Table: buncolgen.TableInfo{
				Name:       through.TableName,
				Alias:      through.Alias,
				PrimaryKey: through.PKColumns(),
			},
			SourceJoin: sourceJoin,
			TargetJoin: targetJoin,
			Tenant:     throughTenant,
		},
		Traversable: traversable,
	}, nil
}

func (b *builder) resolveTargetEntity(sourcePkg, goType string) (string, error) {
	base := strings.TrimPrefix(goType, "[]")
	base = strings.TrimPrefix(base, "*")

	structRef := base
	if !strings.Contains(base, ".") {
		structRef = sourcePkg + "." + base
	}

	targetKey, ok := b.structToEntityKey[structRef]
	if !ok {
		return "", fmt.Errorf(
			"relation targets %q which is not declared in the manifest — declare it or omit the edge",
			structRef,
		)
	}
	return targetKey, nil
}

func (b *builder) warnUnclassifiedSensitivity(
	def *permission.ResourceDefinition,
	resource, fieldKey string,
) {
	if def == nil {
		return
	}
	if def.DefaultSensitivity.Level() < permission.SensitivityRestricted.Level() {
		return
	}
	if _, classified := def.FieldSensitivities[fieldKey]; !classified {
		b.warnings = append(b.warnings, fmt.Sprintf(
			"resource %q has default sensitivity %q but field %q has no explicit FieldSensitivities entry",
			resource, def.DefaultSensitivity, fieldKey,
		))
	}
}

func tenantColumns(model *structparse.Model) (reportcatalog.TenantColumns, error) {
	var tenant reportcatalog.TenantColumns
	for i := range model.Fields {
		switch model.Fields[i].GoName {
		case "OrganizationID":
			tenant.OrganizationID = model.Fields[i].ColumnName
		case "BusinessUnitID":
			tenant.BusinessUnitID = model.Fields[i].ColumnName
		}
	}

	if (tenant.OrganizationID == "") != (tenant.BusinessUnitID == "") {
		return tenant, fmt.Errorf(
			"%s is half-tenanted (has exactly one of organization_id/business_unit_id)",
			model.StructName,
		)
	}

	return tenant, nil
}

func hasColumn(model *structparse.Model, columnName string) bool {
	for i := range model.Fields {
		if model.Fields[i].ColumnName == columnName {
			return true
		}
	}
	return false
}

func relationByJSONName(model *structparse.Model, jsonName string) *structparse.Relation {
	for i := range model.Relations {
		if model.Relations[i].JSONName == jsonName {
			return &model.Relations[i]
		}
	}
	return nil
}

func m2mByJSONName(model *structparse.Model, jsonName string) *structparse.M2MRelation {
	for i := range model.M2MRelations {
		if model.M2MRelations[i].JSONName == jsonName {
			return &model.M2MRelations[i]
		}
	}
	return nil
}

func defaultAggregations(fieldType reportcatalog.FieldType) []reportcatalog.Aggregation {
	switch fieldType {
	case reportcatalog.FieldDecimal, reportcatalog.FieldInt:
		return []reportcatalog.Aggregation{
			reportcatalog.AggCount, reportcatalog.AggCountDistinct,
			reportcatalog.AggSum, reportcatalog.AggAvg,
			reportcatalog.AggMin, reportcatalog.AggMax,
		}
	case reportcatalog.FieldEpoch:
		return []reportcatalog.Aggregation{
			reportcatalog.AggCount, reportcatalog.AggCountDistinct,
			reportcatalog.AggMin, reportcatalog.AggMax,
		}
	case reportcatalog.FieldJSON:
		return nil
	default:
		return []reportcatalog.Aggregation{reportcatalog.AggCount, reportcatalog.AggCountDistinct}
	}
}

func isValidFormat(format reportcatalog.FormatHint) bool {
	switch format {
	case reportcatalog.FormatNone, reportcatalog.FormatMoney, reportcatalog.FormatWeight,
		reportcatalog.FormatPercent, reportcatalog.FormatDuration, reportcatalog.FormatDistance,
		reportcatalog.FormatCount:
		return true
	default:
		return false
	}
}

func containsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func containsAggregation(aggs []reportcatalog.Aggregation, agg reportcatalog.Aggregation) bool {
	for _, a := range aggs {
		if a == agg {
			return true
		}
	}
	return false
}
