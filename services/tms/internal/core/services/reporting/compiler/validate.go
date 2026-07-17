package compiler

import (
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

type resolvedRef struct {
	ref     report.FieldRef
	entity  *reportcatalog.Entity
	field   *reportcatalog.Field
	path    reportcatalog.ResolvedPath
	pathKey string
	toMany  bool
}

type validatedColumn struct {
	spec *report.ColumnSpec
	ref  *resolvedRef
}

type validatedDef struct {
	def     *report.Definition
	entity  *reportcatalog.Entity
	columns []validatedColumn
	refs    map[string]*resolvedRef
	params  map[string]any

	entityKeys []string
	pivotRef   *resolvedRef
}

func (v *validatedDef) columnByID(id string) *validatedColumn {
	for i := range v.columns {
		if v.columns[i].spec.ID == id {
			return &v.columns[i]
		}
	}
	return nil
}

type limits struct {
	maxToOneJoins       int
	maxToManySubqueries int
	maxDimensions       int
	maxPivotColumns     int
	maxPathDepth        int
	maxLimit            int
}

func (c *Compiler) validateEnvelope(
	multiErr *errortypes.MultiError,
	def *report.Definition,
) (*reportcatalog.Entity, bool) {
	if def == nil {
		multiErr.Add("definition", errortypes.ErrRequired, "Definition is required")
		return nil, false
	}
	if def.IRVersion != report.CurrentIRVersion {
		multiErr.Add(
			"definition.irVersion",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Unsupported IR version %d (expected %d)",
				def.IRVersion,
				report.CurrentIRVersion,
			),
		)
		return nil, false
	}

	entity, ok := c.catalog.Entity(def.Entity)
	if !ok {
		multiErr.Add(
			"definition.entity",
			errortypes.ErrInvalid,
			fmt.Sprintf("Unknown entity %q", def.Entity),
		)
		return nil, false
	}

	return entity, true
}

func (c *Compiler) validate(
	req *services.ReportCompileRequest,
) (*validatedDef, error) {
	multiErr := errortypes.NewMultiError()
	def := req.Definition

	entity, ok := c.validateEnvelope(multiErr, def)
	if !ok {
		return nil, multiErr
	}

	v := &validatedDef{
		def:    def,
		entity: entity,
		refs:   make(map[string]*resolvedRef),
	}

	if len(def.Columns) == 0 {
		multiErr.Add(
			"definition.columns",
			errortypes.ErrRequired,
			"At least one column is required",
		)
	}

	params, paramErr := c.validateParams(def, req.Params)
	if paramErr != nil {
		multiErr.AddError(errortypes.NewValidationError(
			"definition.parameters", errortypes.ErrInvalid, paramErr.Error(),
		))
	}
	v.params = params

	hasMeasures := def.HasMeasures()
	dimensionCount := c.validateColumns(v, multiErr, hasMeasures)

	if hasMeasures {
		for i := range v.columns {
			if v.columns[i].spec.Kind == report.ColumnKindDimension &&
				!v.columns[i].ref.field.Groupable {
				multiErr.Add(fmt.Sprintf("definition.columns[%d]", i), errortypes.ErrInvalid,
					"Non-groupable dimension in an aggregated report")
			}
		}
	} else if !def.Having.IsEmpty() {
		multiErr.Add("definition.having", errortypes.ErrInvalid,
			"Measure filters require at least one measure column")
	}

	c.validateFilterTree(v, multiErr, def.Filters, "definition.filters", false)
	c.validateFilterTree(v, multiErr, def.Having, "definition.having", true)
	c.validateSort(v, multiErr)
	c.validatePivot(v, multiErr, hasMeasures)

	if def.Limit < 0 {
		multiErr.Add("definition.limit", errortypes.ErrInvalid, "Limit cannot be negative")
	}
	if def.Limit > c.limits.maxLimit {
		multiErr.Add("definition.limit", errortypes.ErrInvalid,
			fmt.Sprintf("Limit exceeds the maximum of %d rows", c.limits.maxLimit))
	}
	// The dimension cap protects GROUP BY width; plain list reports (no
	// measures, no grouping) are only bounded by a looser column backstop.
	dimensionLimit := c.limits.maxDimensions
	if !hasMeasures {
		dimensionLimit *= 4
	}
	if dimensionCount > dimensionLimit {
		multiErr.Add("definition.columns", errortypes.ErrInvalid,
			fmt.Sprintf("Too many dimensions (%d > %d)", dimensionCount, dimensionLimit))
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	v.entityKeys = collectEntityKeys(v)
	return v, nil
}

func (c *Compiler) validateColumns(
	v *validatedDef,
	multiErr *errortypes.MultiError,
	hasMeasures bool,
) (dimensionCount int) {
	def := v.def
	seenIDs := make(map[string]bool, len(def.Columns))

	for i := range def.Columns {
		col := &def.Columns[i]
		fieldPath := fmt.Sprintf("definition.columns[%d]", i)

		if col.ID == "" {
			multiErr.Add(fieldPath+".id", errortypes.ErrRequired, "Column ID is required")
			continue
		}
		if seenIDs[col.ID] {
			multiErr.Add(fieldPath+".id", errortypes.ErrInvalid, "Duplicate column ID")
			continue
		}
		seenIDs[col.ID] = true

		if col.Kind == report.ColumnKindComputed {
			if !validateComputedShape(multiErr, col, fieldPath) {
				continue
			}
			v.columns = append(v.columns, validatedColumn{spec: col})
			continue
		}

		ref, refErr := c.resolveRef(v, col.Ref)
		if refErr != nil {
			multiErr.Add(fieldPath+".ref", errortypes.ErrInvalid, refErr.Error())
			continue
		}

		if col.Kind == report.ColumnKindDimension {
			dimensionCount++
		}
		if !c.validateColumn(multiErr, col, ref, fieldPath, hasMeasures) {
			continue
		}

		v.columns = append(v.columns, validatedColumn{spec: col, ref: ref})
	}

	validateComputedOperands(v, multiErr)

	return dimensionCount
}

func validateComputedShape(
	multiErr *errortypes.MultiError,
	col *report.ColumnSpec,
	fieldPath string,
) bool {
	comp := col.Computed
	if comp == nil {
		multiErr.Add(fieldPath+".computed", errortypes.ErrRequired,
			"Computed columns require a computed expression")
		return false
	}
	if col.Ref.Field != "" || len(col.Ref.Path) > 0 {
		multiErr.Add(fieldPath+".ref", errortypes.ErrInvalid,
			"Computed columns cannot reference a field directly")
		return false
	}
	if col.Agg != "" || col.Bucket != report.DateBucketNone {
		multiErr.Add(fieldPath, errortypes.ErrInvalid,
			"Computed columns cannot carry an aggregation or date bucket")
		return false
	}
	if col.Label == "" {
		multiErr.Add(fieldPath+".label", errortypes.ErrRequired,
			"Computed columns require a label")
		return false
	}
	if !comp.Op.IsValid() {
		multiErr.Add(fieldPath+".computed.op", errortypes.ErrInvalid,
			"Computed operator must be add, subtract, multiply, or divide")
		return false
	}
	if comp.Format != "" && !comp.Format.IsValid() {
		multiErr.Add(fieldPath+".computed.format", errortypes.ErrInvalid,
			fmt.Sprintf("Unknown format hint %q", comp.Format))
		return false
	}
	if comp.LeftID == "" || comp.RightID == "" {
		multiErr.Add(fieldPath+".computed", errortypes.ErrRequired,
			"Computed columns require both operand column IDs")
		return false
	}
	if comp.LeftID == col.ID || comp.RightID == col.ID {
		multiErr.Add(fieldPath+".computed", errortypes.ErrInvalid,
			"A computed column cannot reference itself")
		return false
	}
	return true
}

func validateComputedOperands(v *validatedDef, multiErr *errortypes.MultiError) {
	for i := range v.columns {
		col := &v.columns[i]
		if col.spec.Kind != report.ColumnKindComputed {
			continue
		}
		fieldPath := fmt.Sprintf("definition.columns[%s].computed", col.spec.ID)
		for _, operandID := range []string{col.spec.Computed.LeftID, col.spec.Computed.RightID} {
			operand := v.columnByID(operandID)
			if operand == nil {
				multiErr.Add(fieldPath, errortypes.ErrInvalid, fmt.Sprintf(
					"Computed operand %q does not reference a valid column", operandID))
				continue
			}
			if operand.spec.Kind != report.ColumnKindMeasure {
				multiErr.Add(fieldPath, errortypes.ErrInvalid, fmt.Sprintf(
					"Computed operand %q must be a measure column", operandID))
			}
		}
	}
}

func (c *Compiler) validateColumn(
	multiErr *errortypes.MultiError,
	col *report.ColumnSpec,
	ref *resolvedRef,
	fieldPath string,
	hasMeasures bool,
) bool {
	if col.Computed != nil {
		multiErr.Add(fieldPath+".computed", errortypes.ErrInvalid,
			"Only computed columns may carry a computed expression")
		return false
	}

	//nolint:exhaustive // computed columns are validated before ref resolution
	switch col.Kind {
	case report.ColumnKindDimension:
		return c.validateDimensionColumn(multiErr, col, ref, fieldPath, hasMeasures)
	case report.ColumnKindMeasure:
		return c.validateMeasureColumn(multiErr, col, ref, fieldPath)
	default:
		multiErr.Add(
			fieldPath+".kind",
			errortypes.ErrInvalid,
			"Column kind must be dimension or measure",
		)
		return false
	}
}

func (c *Compiler) validateDimensionColumn(
	multiErr *errortypes.MultiError,
	col *report.ColumnSpec,
	ref *resolvedRef,
	fieldPath string,
	hasMeasures bool,
) bool {
	if ref.toMany {
		multiErr.Add(fieldPath+".ref", errortypes.ErrInvalid, fmt.Sprintf(
			"Dimension %q crosses a to-many relationship; only measures may aggregate across to-many paths",
			col.Ref.String(),
		))
		return false
	}
	if !ref.field.Groupable && hasMeasures {
		multiErr.Add(fieldPath+".ref", errortypes.ErrInvalid, fmt.Sprintf(
			"Field %q cannot be used as a grouping dimension", col.Ref.String(),
		))
		return false
	}
	if col.Agg != "" {
		multiErr.Add(
			fieldPath+".agg",
			errortypes.ErrInvalid,
			"Dimensions cannot have an aggregation",
		)
		return false
	}
	if col.Bucket != report.DateBucketNone {
		if !col.Bucket.IsValid() {
			multiErr.Add(fieldPath+".bucket", errortypes.ErrInvalid, "Invalid date bucket")
			return false
		}
		if ref.field.Type != reportcatalog.FieldEpoch {
			multiErr.Add(fieldPath+".bucket", errortypes.ErrInvalid,
				"Date buckets are only valid on date/timestamp fields")
			return false
		}
	}
	return true
}

func (c *Compiler) validateMeasureColumn(
	multiErr *errortypes.MultiError,
	col *report.ColumnSpec,
	ref *resolvedRef,
	fieldPath string,
) bool {
	if col.Agg == "" {
		multiErr.Add(
			fieldPath+".agg",
			errortypes.ErrRequired,
			"Measures require an aggregation",
		)
		return false
	}
	if !ref.field.SupportsAggregation(col.Agg) {
		multiErr.Add(fieldPath+".agg", errortypes.ErrInvalid, fmt.Sprintf(
			"Aggregation %q is not legal for field %q", col.Agg, col.Ref.String(),
		))
		return false
	}
	if ref.toMany && col.Agg == reportcatalog.AggCountDistinct {
		multiErr.Add(fieldPath+".agg", errortypes.ErrInvalid,
			"count_distinct is not supported across to-many relationships")
		return false
	}
	if col.Bucket != report.DateBucketNone {
		multiErr.Add(
			fieldPath+".bucket",
			errortypes.ErrInvalid,
			"Measures cannot be date-bucketed",
		)
		return false
	}
	return true
}

func (c *Compiler) resolveRef(v *validatedDef, ref report.FieldRef) (*resolvedRef, error) {
	key := ref.String()
	if existing, ok := v.refs[key]; ok {
		return existing, nil
	}

	if len(ref.Path) > c.limits.maxPathDepth {
		return nil, fmt.Errorf(
			"path %q exceeds the maximum depth of %d",
			reportcatalog.PathKey(ref.Path), c.limits.maxPathDepth,
		)
	}

	_, path, err := c.catalog.ResolvePath(v.entity.Key, ref.Path)
	if err != nil {
		return nil, err
	}

	terminal := path.Terminal(v.entity)
	field, ok := terminal.Field(ref.Field)
	if !ok {
		return nil, fmt.Errorf("unknown field %q on entity %q", ref.Field, terminal.Key)
	}

	resolved := &resolvedRef{
		ref:     ref,
		entity:  terminal,
		field:   field,
		path:    path,
		pathKey: reportcatalog.PathKey(ref.Path),
		toMany:  path.CrossesToMany(),
	}
	v.refs[key] = resolved
	return resolved, nil
}

func (c *Compiler) validateFilterTree(
	v *validatedDef,
	multiErr *errortypes.MultiError,
	group *report.FilterGroup,
	fieldPath string,
	having bool,
) {
	if group.IsEmpty() {
		return
	}
	if !group.Op.IsValid() {
		multiErr.Add(
			fieldPath+".op",
			errortypes.ErrInvalid,
			"Filter group operator must be and/or",
		)
		return
	}

	for i := range group.Filters {
		filterPath := fmt.Sprintf("%s.filters[%d]", fieldPath, i)
		c.validateFilter(v, multiErr, &group.Filters[i], filterPath, having)
	}
	for i := range group.Groups {
		c.validateFilterTree(v, multiErr, &group.Groups[i],
			fmt.Sprintf("%s.groups[%d]", fieldPath, i), having)
	}
}

func (c *Compiler) validateFilter(
	v *validatedDef,
	multiErr *errortypes.MultiError,
	filter *report.FieldFilter,
	fieldPath string,
	having bool,
) {
	ref, err := c.resolveRef(v, filter.Ref)
	if err != nil {
		multiErr.Add(fieldPath+".ref", errortypes.ErrInvalid, err.Error())
		return
	}

	if having {
		if !c.validateHavingFilter(multiErr, filter, ref, fieldPath) {
			return
		}
	} else if !c.validateRowFilter(multiErr, filter, ref, fieldPath) {
		return
	}

	if filter.Param != "" && filter.Value != nil {
		multiErr.Add(fieldPath, errortypes.ErrInvalid,
			"A filter may bind either a literal value or a parameter, not both")
		return
	}

	if filter.Param != "" {
		if _, ok := v.def.Parameter(filter.Param); !ok {
			multiErr.Add(fieldPath+".param", errortypes.ErrInvalid,
				fmt.Sprintf("Undeclared parameter %q", filter.Param))
		}
		return
	}

	if operatorRequiresValue(filter.Operator) && filter.Value == nil {
		multiErr.Add(fieldPath+".value", errortypes.ErrRequired, "Filter value is required")
		return
	}
	if filter.Value != nil && !having {
		if err = coerceFilterValue(filter.Operator, ref.field, filter.Value); err != nil {
			multiErr.Add(fieldPath+".value", errortypes.ErrInvalid, err.Error())
		}
	}
}

func (c *Compiler) validateHavingFilter(
	multiErr *errortypes.MultiError,
	filter *report.FieldFilter,
	ref *resolvedRef,
	fieldPath string,
) bool {
	if filter.Agg == "" {
		multiErr.Add(
			fieldPath+".agg",
			errortypes.ErrRequired,
			"Measure filters require an aggregation",
		)
		return false
	}
	if !ref.field.SupportsAggregation(filter.Agg) {
		multiErr.Add(fieldPath+".agg", errortypes.ErrInvalid, fmt.Sprintf(
			"Aggregation %q is not legal for field %q", filter.Agg, filter.Ref.String(),
		))
		return false
	}
	if ref.toMany && filter.Agg == reportcatalog.AggCountDistinct {
		multiErr.Add(fieldPath+".agg", errortypes.ErrInvalid,
			"count_distinct is not supported across to-many relationships")
		return false
	}
	if !isComparisonOperator(filter.Operator) {
		multiErr.Add(fieldPath+".operator", errortypes.ErrInvalid,
			"Measure filters support only comparison operators")
		return false
	}
	return true
}

func (c *Compiler) validateRowFilter(
	multiErr *errortypes.MultiError,
	filter *report.FieldFilter,
	ref *resolvedRef,
	fieldPath string,
) bool {
	if filter.Agg != "" {
		multiErr.Add(fieldPath+".agg", errortypes.ErrInvalid,
			"Row filters cannot have an aggregation; use a measure filter")
		return false
	}
	if !ref.field.Filterable {
		multiErr.Add(fieldPath+".ref", errortypes.ErrInvalid,
			fmt.Sprintf("Field %q is not filterable", filter.Ref.String()))
		return false
	}
	if !operatorLegalForType(filter.Operator, ref.field.Type) {
		multiErr.Add(fieldPath+".operator", errortypes.ErrInvalid, fmt.Sprintf(
			"Operator %q is not valid for %s fields", filter.Operator, ref.field.Type,
		))
		return false
	}
	return true
}

func (c *Compiler) validateSort(v *validatedDef, multiErr *errortypes.MultiError) {
	for i, sortSpec := range v.def.Sort {
		fieldPath := fmt.Sprintf("definition.sort[%d]", i)
		if _, ok := v.def.ColumnByID(sortSpec.ColumnID); !ok {
			multiErr.Add(fieldPath+".columnId", errortypes.ErrInvalid,
				fmt.Sprintf("Sort references unknown column %q", sortSpec.ColumnID))
		}
		if sortSpec.Direction != dbtype.SortDirectionAsc &&
			sortSpec.Direction != dbtype.SortDirectionDesc {
			multiErr.Add(fieldPath+".direction", errortypes.ErrInvalid,
				"Sort direction must be asc or desc")
		}
	}
}

func (c *Compiler) validatePivot(
	v *validatedDef,
	multiErr *errortypes.MultiError,
	hasMeasures bool,
) {
	pivot := v.def.Pivot
	if pivot == nil {
		return
	}

	if !hasMeasures {
		multiErr.Add("definition.pivot", errortypes.ErrInvalid, "Pivots require measure columns")
		return
	}
	if len(pivot.Values) == 0 {
		multiErr.Add(
			"definition.pivot.values",
			errortypes.ErrRequired,
			"Pivot values are required",
		)
		return
	}
	if len(pivot.MeasureIDs) == 0 {
		multiErr.Add("definition.pivot.measureIds", errortypes.ErrRequired,
			"Pivot must reference at least one measure")
		return
	}

	pivotColumns := len(pivot.Values) * len(pivot.MeasureIDs)
	if pivot.IncludeOther {
		pivotColumns += len(pivot.MeasureIDs)
	}
	if pivotColumns > c.limits.maxPivotColumns {
		multiErr.Add("definition.pivot.values", errortypes.ErrInvalid,
			fmt.Sprintf("Pivot produces %d columns, exceeding the maximum of %d",
				pivotColumns, c.limits.maxPivotColumns))
		return
	}

	ref, err := c.resolveRef(v, pivot.Ref)
	if err != nil {
		multiErr.Add("definition.pivot.ref", errortypes.ErrInvalid, err.Error())
		return
	}
	if ref.toMany {
		multiErr.Add("definition.pivot.ref", errortypes.ErrInvalid,
			"Pivot fields cannot cross to-many relationships")
		return
	}
	if !ref.field.Groupable {
		multiErr.Add("definition.pivot.ref", errortypes.ErrInvalid,
			fmt.Sprintf("Field %q cannot be pivoted", pivot.Ref.String()))
		return
	}

	for i, id := range pivot.MeasureIDs {
		col, ok := v.def.ColumnByID(id)
		if !ok ||
			(col.Kind != report.ColumnKindMeasure && col.Kind != report.ColumnKindComputed) {
			multiErr.Add(fmt.Sprintf("definition.pivot.measureIds[%d]", i), errortypes.ErrInvalid,
				fmt.Sprintf("Pivot measure %q does not reference a measure column", id))
		}
	}
	for i := range v.def.Columns {
		col := &v.def.Columns[i]
		if col.Ref.String() == pivot.Ref.String() && col.Kind == report.ColumnKindDimension {
			multiErr.Add("definition.pivot.ref", errortypes.ErrInvalid,
				"The pivot field cannot also be selected as a dimension")
		}
	}

	v.pivotRef = ref
}

func (c *Compiler) validateParams(
	def *report.Definition,
	raw map[string]any,
) (map[string]any, error) {
	resolved := make(map[string]any, len(def.Parameters))

	for i := range def.Parameters {
		param := &def.Parameters[i]
		if !param.Type.IsValid() {
			return nil, fmt.Errorf("parameter %q has invalid type %q", param.Name, param.Type)
		}
		if err := c.validateRefParam(param); err != nil {
			return nil, err
		}

		value, provided := raw[param.Name]
		if !provided || value == nil {
			switch {
			case param.Default != nil:
				value = param.Default
			case param.Required:
				return nil, fmt.Errorf("required parameter %q was not provided", param.Name)
			default:
				continue
			}
		}

		coerced, err := coerceParamValue(param, value)
		if err != nil {
			return nil, fmt.Errorf("parameter %q: %w", param.Name, err)
		}
		resolved[param.Name] = coerced
	}

	for name := range raw {
		if _, ok := def.Parameter(name); !ok {
			return nil, fmt.Errorf("unknown parameter %q", name)
		}
	}

	return resolved, nil
}

func (c *Compiler) validateRefParam(param *report.ParameterDef) error {
	if param.Type != reportcatalog.FieldRef {
		if param.RefEntity != "" {
			return fmt.Errorf(
				"parameter %q declares a reference entity but is not a ref parameter", param.Name,
			)
		}
		return nil
	}

	if param.RefEntity == "" {
		return fmt.Errorf("ref parameter %q must declare a reference entity", param.Name)
	}
	if _, ok := c.catalog.Entity(param.RefEntity); !ok {
		return fmt.Errorf(
			"ref parameter %q references unknown entity %q", param.Name, param.RefEntity,
		)
	}
	if len(param.AllowedValues) > 0 {
		return fmt.Errorf(
			"ref parameter %q cannot carry an allow-list; values come from the referenced entity",
			param.Name,
		)
	}
	return nil
}

func collectEntityKeys(v *validatedDef) []string {
	seen := map[string]bool{v.entity.Key: true}
	for _, ref := range v.refs {
		for _, step := range ref.path.Steps {
			seen[step.Entity.Key] = true
		}
	}

	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func isComparisonOperator(op dbtype.Operator) bool {
	//nolint:exhaustive // non-comparison operators fall through to false
	switch op {
	case dbtype.OpEqual, dbtype.OpNotEqual, dbtype.OpGreaterThan,
		dbtype.OpGreaterThanOrEqual, dbtype.OpLessThan, dbtype.OpLessThanOrEqual:
		return true
	default:
		return false
	}
}

func operatorRequiresValue(op dbtype.Operator) bool {
	//nolint:exhaustive // every operator not listed carries a value
	switch op {
	case dbtype.OpIsNull,
		dbtype.OpIsNotNull,
		dbtype.OpToday,
		dbtype.OpYesterday,
		dbtype.OpTomorrow,
		dbtype.OpThisWeek,
		dbtype.OpLastWeek,
		dbtype.OpThisMonth,
		dbtype.OpLastMonth,
		dbtype.OpThisQuarter,
		dbtype.OpLastQuarter,
		dbtype.OpThisYear,
		dbtype.OpLastYear:
		return false
	default:
		return true
	}
}

func operatorLegalForType(op dbtype.Operator, fieldType reportcatalog.FieldType) bool {
	//nolint:exhaustive // unsupported operators (count ops) fall through to false
	switch op {
	case dbtype.OpEqual, dbtype.OpNotEqual:
		return fieldType != reportcatalog.FieldJSON
	case dbtype.OpGreaterThan, dbtype.OpGreaterThanOrEqual,
		dbtype.OpLessThan, dbtype.OpLessThanOrEqual:
		//nolint:exhaustive // remaining field types are not ordered
		switch fieldType {
		case reportcatalog.FieldInt, reportcatalog.FieldDecimal, reportcatalog.FieldEpoch,
			reportcatalog.FieldString:
			return true
		default:
			return false
		}
	case dbtype.OpContains, dbtype.OpStartsWith, dbtype.OpEndsWith,
		dbtype.OpLike, dbtype.OpILike:
		return fieldType == reportcatalog.FieldString
	case dbtype.OpIn, dbtype.OpNotIn:
		//nolint:exhaustive // remaining field types do not support set membership
		switch fieldType {
		case reportcatalog.FieldString, reportcatalog.FieldEnum, reportcatalog.FieldRef,
			reportcatalog.FieldInt:
			return true
		default:
			return false
		}
	case dbtype.OpIsNull, dbtype.OpIsNotNull:
		return true
	case dbtype.OpDateRange, dbtype.OpLastNDays, dbtype.OpNextNDays,
		dbtype.OpToday, dbtype.OpYesterday, dbtype.OpTomorrow,
		dbtype.OpThisWeek, dbtype.OpLastWeek, dbtype.OpThisMonth, dbtype.OpLastMonth,
		dbtype.OpThisQuarter, dbtype.OpLastQuarter, dbtype.OpThisYear, dbtype.OpLastYear:
		return fieldType == reportcatalog.FieldEpoch
	default:
		return false
	}
}

func escapeLikePattern(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}
