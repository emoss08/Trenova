package compiler

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/uptrace/bun"
)

type sqlExpr struct {
	text string
	args []any
}

type emitter struct {
	c    *Compiler
	v    *validatedDef
	plan *joinPlan
	az   *authzResult
	req  *services.ReportCompileRequest
	loc  *time.Location
}

type outputColumn struct {
	id     string
	expr   sqlExpr
	column services.ReportResultColumn
	isDim  bool
}

func requestLocation(orgTimezone string) (*time.Location, error) {
	if orgTimezone == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(orgTimezone)
	if err != nil {
		return nil, fmt.Errorf("invalid organization timezone %q: %w", orgTimezone, err)
	}
	return loc, nil
}

func (c *Compiler) emit(
	req *services.ReportCompileRequest,
	v *validatedDef,
	plan *joinPlan,
	az *authzResult,
) (*services.CompiledReportQuery, error) {
	loc, err := requestLocation(req.OrgTimezone)
	if err != nil {
		return nil, err
	}

	e := &emitter{c: c, v: v, plan: plan, az: az, req: req, loc: loc}

	if err = e.checkOwnScopeSupport(); err != nil {
		return nil, err
	}

	outputs, err := e.buildOutputColumns()
	if err != nil {
		return nil, err
	}

	asm := &sqlAssembler{}

	if err = e.writeSelectFrom(asm, outputs, plan); err != nil {
		return nil, err
	}
	if err = e.writeWhereGroupHaving(asm, outputs); err != nil {
		return nil, err
	}

	orderBy, err := e.orderBySQL(outputs)
	if err != nil {
		return nil, err
	}
	asm.write(orderBy)

	limit := e.v.def.Limit
	if limit <= 0 || limit > e.c.limits.maxLimit {
		limit = e.c.limits.maxLimit
	}
	asm.write(" LIMIT ")
	asm.write(strconv.Itoa(limit))

	return c.buildResult(v, plan, outputs, asm.sb.String(), asm.args, limit), nil
}

type sqlAssembler struct {
	sb   strings.Builder
	args []any
}

func (a *sqlAssembler) write(s string) {
	a.sb.WriteString(s)
}

func (a *sqlAssembler) writeExpr(expr sqlExpr) {
	a.sb.WriteString(expr.text)
	a.args = append(a.args, expr.args...)
}

func (e *emitter) writeSelectFrom(
	asm *sqlAssembler,
	outputs []outputColumn,
	plan *joinPlan,
) error {
	asm.write("SELECT ")
	for i := range outputs {
		if i > 0 {
			asm.write(", ")
		}
		asm.writeExpr(outputs[i].expr)
		asm.write(" AS c")
		asm.write(strconv.Itoa(i))
	}

	asm.write(" FROM ")
	asm.write(e.v.entity.Table.As("t0"))

	for _, join := range plan.joins {
		asm.write(" ")
		asm.writeExpr(e.joinClause(&join))
	}

	for _, key := range plan.lateralOrder {
		lateralExpr, latErr := e.lateralClause(plan.laterals[key])
		if latErr != nil {
			return latErr
		}
		asm.write(" ")
		asm.writeExpr(lateralExpr)
	}

	return nil
}

func (e *emitter) writeWhereGroupHaving(asm *sqlAssembler, outputs []outputColumn) error {
	whereExpr, err := e.whereClause()
	if err != nil {
		return err
	}
	asm.write(" WHERE ")
	asm.writeExpr(whereExpr)

	for i, expr := range e.groupByExprs(outputs) {
		if i == 0 {
			asm.write(" GROUP BY ")
		} else {
			asm.write(", ")
		}
		asm.writeExpr(expr)
	}

	if !e.v.def.Having.IsEmpty() {
		havingExpr, havingErr := e.filterGroupExpr(e.v.def.Having, true)
		if havingErr != nil {
			return havingErr
		}
		asm.write(" HAVING ")
		asm.writeExpr(havingExpr)
	}

	return nil
}

func (c *Compiler) buildResult(
	v *validatedDef,
	plan *joinPlan,
	outputs []outputColumn,
	sql string,
	args []any,
	limit int,
) *services.CompiledReportQuery {
	columns := make([]services.ReportResultColumn, 0, len(outputs))
	for i := range outputs {
		columns = append(columns, outputs[i].column)
	}

	// Through-tables carry query-relevant rows of their own (m2m links), so
	// they must participate in the result cache's data-version vector too.
	tableSet := make(map[string]bool, len(v.entityKeys))
	for _, key := range v.entityKeys {
		if entity, ok := c.catalog.Entity(key); ok {
			tableSet[entity.Table.Name] = true
		}
	}
	for _, ref := range v.refs {
		for _, step := range ref.path.Steps {
			if step.Edge.Through != nil {
				tableSet[step.Edge.Through.Table.Name] = true
			}
		}
	}
	tables := make([]string, 0, len(tableSet))
	for table := range tableSet {
		tables = append(tables, table)
	}
	sort.Strings(tables)

	return &services.CompiledReportQuery{
		SQL:                sql,
		Args:               args,
		Columns:            columns,
		Complexity:         c.complexity(v, plan),
		ReferencedEntities: v.entityKeys,
		ReferencedTables:   tables,
		Limit:              limit,
	}
}

func (e *emitter) groupByExprs(outputs []outputColumn) []sqlExpr {
	if !e.v.def.HasMeasures() {
		return nil
	}

	groupExprs := make([]sqlExpr, 0, len(outputs))
	for i := range outputs {
		if outputs[i].isDim {
			groupExprs = append(groupExprs, outputs[i].expr)
		}
	}
	return groupExprs
}

func (e *emitter) orderBySQL(outputs []outputColumn) (string, error) {
	if len(e.v.def.Sort) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString(" ORDER BY ")
	for i, sortSpec := range e.v.def.Sort {
		if i > 0 {
			sb.WriteString(", ")
		}
		idx := outputIndexByColumnID(outputs, sortSpec.ColumnID)
		if idx < 0 {
			return "", fmt.Errorf("sort references column %q with no output", sortSpec.ColumnID)
		}
		sb.WriteString("c")
		sb.WriteString(strconv.Itoa(idx))
		if sortSpec.Direction == dbtype.SortDirectionDesc {
			sb.WriteString(" DESC")
		} else {
			sb.WriteString(" ASC")
		}
	}

	return sb.String(), nil
}

func (e *emitter) checkOwnScopeSupport() error {
	for entityKey, scope := range e.az.scopes {
		if scope != permission.DataScopeOwn {
			continue
		}
		entity, ok := e.c.catalog.Entity(entityKey)
		if !ok {
			continue
		}
		if entity.OwnershipColumn == "" {
			return errortypes.NewAuthorizationError(fmt.Sprintf(
				"your access to %s is limited to your own records, but %s does not support per-user scoping in reports",
				entity.PluralLabel,
				entity.PluralLabel,
			))
		}
	}
	return nil
}

func (e *emitter) scopeConds(entity *reportcatalog.Entity, alias string) sqlExpr {
	var conds []string
	var args []any

	if entity.Tenant.IsTenanted() {
		conds = append(conds,
			alias+"."+entity.Tenant.OrganizationID+" = ?",
			alias+"."+entity.Tenant.BusinessUnitID+" = ?",
		)
		args = append(args, e.req.Tenant.OrgID, e.req.Tenant.BuID)
	}

	if e.az.scopes[entity.Key] == permission.DataScopeOwn && entity.OwnershipColumn != "" {
		conds = append(conds, alias+"."+entity.OwnershipColumn+" = ?")
		args = append(args, e.req.Tenant.UserID)
	}

	return sqlExpr{text: strings.Join(conds, " AND "), args: args}
}

func (e *emitter) joinClause(join *joinedEntity) sqlExpr {
	var sb strings.Builder
	var args []any

	sb.WriteString("LEFT JOIN ")
	sb.WriteString(join.entity.Table.Name)
	sb.WriteString(" AS ")
	sb.WriteString(join.alias)
	sb.WriteString(" ON ")

	for i, pair := range join.edge.Join {
		if i > 0 {
			sb.WriteString(" AND ")
		}
		sb.WriteString(join.alias + "." + pair.Remote + " = " + join.parentAlias + "." + pair.Local)
	}

	scope := e.scopeConds(join.entity, join.alias)
	if scope.text != "" {
		sb.WriteString(" AND ")
		sb.WriteString(scope.text)
		args = append(args, scope.args...)
	}

	return sqlExpr{text: sb.String(), args: args}
}

type chainStep struct {
	alias  string
	entity *reportcatalog.Entity
	edge   *reportcatalog.Edge
}

func (e *emitter) buildChain(
	path reportcatalog.ResolvedPath,
	firstIdx int,
	aliasPrefix string,
) []chainStep {
	steps := make([]chainStep, 0, len(path.Steps)-firstIdx)
	for i := firstIdx; i < len(path.Steps); i++ {
		steps = append(steps, chainStep{
			alias:  fmt.Sprintf("%s%d", aliasPrefix, i-firstIdx),
			entity: path.Steps[i].Entity,
			edge:   path.Steps[i].Edge,
		})
	}
	return steps
}

func (e *emitter) chainCorrelation(path []string, firstIdx int) string {
	prefixKey := lateralPrefixKey(path, firstIdx)
	return e.plan.aliasFor(prefixKey)
}

type chainClause struct {
	fromSQL    string
	fromArgs   []any
	whereConds []string
	whereArgs  []any
}

func (cc *chainClause) render(
	sb *strings.Builder,
	args *[]any,
	extraConds []string,
	extraArgs []any,
) {
	sb.WriteString(cc.fromSQL)
	*args = append(*args, cc.fromArgs...)

	conds := make([]string, 0, len(cc.whereConds)+len(extraConds))
	conds = append(conds, cc.whereConds...)
	conds = append(conds, extraConds...)
	if len(conds) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(conds, " AND "))
		*args = append(*args, cc.whereArgs...)
		*args = append(*args, extraArgs...)
	}
}

func (e *emitter) chainClause(steps []chainStep, correlationAlias string) (*chainClause, error) {
	cc := &chainClause{}
	var from strings.Builder

	for i, step := range steps {
		parentAlias := correlationAlias
		if i > 0 {
			parentAlias = steps[i-1].alias
		}

		if step.edge.Cardinality == reportcatalog.CardinalityM2M {
			if err := e.m2mChainStep(cc, &from, &step, parentAlias, i == 0); err != nil {
				return nil, err
			}
			continue
		}

		conds := make([]string, 0, len(step.edge.Join)+2)
		for _, pair := range step.edge.Join {
			conds = append(conds, step.alias+"."+pair.Remote+" = "+parentAlias+"."+pair.Local)
		}
		scope := e.scopeConds(step.entity, step.alias)
		if scope.text != "" {
			conds = append(conds, scope.text)
		}

		if i == 0 {
			from.WriteString(" FROM ")
			from.WriteString(step.entity.Table.Name + " AS " + step.alias)
			cc.whereConds = append(cc.whereConds, conds...)
			cc.whereArgs = append(cc.whereArgs, scope.args...)
		} else {
			from.WriteString(" LEFT JOIN ")
			from.WriteString(step.entity.Table.Name + " AS " + step.alias)
			from.WriteString(" ON ")
			from.WriteString(strings.Join(conds, " AND "))
			cc.fromArgs = append(cc.fromArgs, scope.args...)
		}
	}

	cc.fromSQL = from.String()
	return cc, nil
}

func (e *emitter) m2mChainStep(
	cc *chainClause,
	from *strings.Builder,
	step *chainStep,
	parentAlias string,
	isFirst bool,
) error {
	if step.edge.Through == nil {
		return fmt.Errorf("m2m edge %q has no through join", step.edge.Name)
	}
	through := step.edge.Through
	throughAlias := step.alias + "j"

	throughConds := make([]string, 0, len(through.SourceJoin)+2)
	var throughArgs []any
	for _, pair := range through.SourceJoin {
		throughConds = append(throughConds,
			throughAlias+"."+pair.Remote+" = "+parentAlias+"."+pair.Local)
	}
	if through.Tenant.IsTenanted() {
		throughConds = append(throughConds,
			throughAlias+"."+through.Tenant.OrganizationID+" = ?",
			throughAlias+"."+through.Tenant.BusinessUnitID+" = ?",
		)
		throughArgs = append(throughArgs, e.req.Tenant.OrgID, e.req.Tenant.BuID)
	}

	if isFirst {
		from.WriteString(" FROM ")
		from.WriteString(through.Table.Name + " AS " + throughAlias)
		cc.whereConds = append(cc.whereConds, throughConds...)
		cc.whereArgs = append(cc.whereArgs, throughArgs...)
	} else {
		from.WriteString(" JOIN ")
		from.WriteString(through.Table.Name + " AS " + throughAlias)
		from.WriteString(" ON ")
		from.WriteString(strings.Join(throughConds, " AND "))
		cc.fromArgs = append(cc.fromArgs, throughArgs...)
	}

	targetConds := make([]string, 0, len(through.TargetJoin)+1)
	for _, pair := range through.TargetJoin {
		targetConds = append(targetConds,
			step.alias+"."+pair.Remote+" = "+throughAlias+"."+pair.Local)
	}
	scope := e.scopeConds(step.entity, step.alias)
	if scope.text != "" {
		targetConds = append(targetConds, scope.text)
	}
	from.WriteString(" JOIN ")
	from.WriteString(step.entity.Table.Name + " AS " + step.alias)
	from.WriteString(" ON ")
	from.WriteString(strings.Join(targetConds, " AND "))
	cc.fromArgs = append(cc.fromArgs, scope.args...)

	return nil
}

func (e *emitter) lateralClause(lateral *lateralPlan) (sqlExpr, error) {
	firstIdx := firstToManyIndex(lateral.path)
	pathEdges := lateral.measures[0].column.ref.ref.Path
	correlation := e.chainCorrelation(pathEdges, firstIdx)
	steps := e.buildChain(lateral.path, firstIdx, "w")
	terminalAlias := steps[len(steps)-1].alias

	var sb strings.Builder
	var args []any

	sb.WriteString("LEFT JOIN LATERAL (SELECT ")

	for i, measure := range lateral.measures {
		if i > 0 {
			sb.WriteString(", ")
		}
		col := measure.column
		fieldExpr := terminalAlias + "." + col.ref.field.Column.Name

		//nolint:exhaustive // count_distinct is rejected at validation for to-many paths
		switch col.spec.Agg {
		case reportcatalog.AggAvg:
			fmt.Fprintf(&sb, "SUM(%s) AS %s_sum, COUNT(%s) AS %s_cnt",
				fieldExpr, measure.innerName, fieldExpr, measure.innerName)
		case reportcatalog.AggCount:
			fmt.Fprintf(&sb, "COUNT(%s) AS %s", fieldExpr, measure.innerName)
		case reportcatalog.AggSum:
			fmt.Fprintf(&sb, "SUM(%s) AS %s", fieldExpr, measure.innerName)
		case reportcatalog.AggMin:
			fmt.Fprintf(&sb, "MIN(%s) AS %s", fieldExpr, measure.innerName)
		case reportcatalog.AggMax:
			fmt.Fprintf(&sb, "MAX(%s) AS %s", fieldExpr, measure.innerName)
		default:
			return sqlExpr{}, fmt.Errorf(
				"aggregation %q is not supported across to-many relationships", col.spec.Agg,
			)
		}
	}

	cc, err := e.chainClause(steps, correlation)
	if err != nil {
		return sqlExpr{}, err
	}
	cc.render(&sb, &args, nil, nil)

	sb.WriteString(") AS ")
	sb.WriteString(lateral.alias)
	sb.WriteString(" ON TRUE")

	return sqlExpr{text: sb.String(), args: args}, nil
}

func (e *emitter) whereClause() (sqlExpr, error) {
	var conds []string
	var args []any

	base := e.scopeConds(e.v.entity, "t0")
	conds = append(conds, base.text)
	args = append(args, base.args...)

	if !e.v.def.Filters.IsEmpty() {
		filterExpr, err := e.filterGroupExpr(e.v.def.Filters, false)
		if err != nil {
			return sqlExpr{}, err
		}
		conds = append(conds, "("+filterExpr.text+")")
		args = append(args, filterExpr.args...)
	}

	return sqlExpr{text: strings.Join(conds, " AND "), args: args}, nil
}

func (e *emitter) filterGroupExpr(group *report.FilterGroup, having bool) (sqlExpr, error) {
	var parts []string
	var args []any

	joiner := " AND "
	if group.Op == report.BoolOpOr {
		joiner = " OR "
	}

	for i := range group.Filters {
		expr, err := e.filterExpr(&group.Filters[i], having)
		if err != nil {
			return sqlExpr{}, err
		}
		parts = append(parts, expr.text)
		args = append(args, expr.args...)
	}

	for i := range group.Groups {
		if group.Groups[i].IsEmpty() {
			continue
		}
		expr, err := e.filterGroupExpr(&group.Groups[i], having)
		if err != nil {
			return sqlExpr{}, err
		}
		parts = append(parts, "("+expr.text+")")
		args = append(args, expr.args...)
	}

	return sqlExpr{text: strings.Join(parts, joiner), args: args}, nil
}

func (e *emitter) filterExpr(filter *report.FieldFilter, having bool) (sqlExpr, error) {
	ref := e.v.refs[filter.Ref.String()]
	if ref == nil {
		return sqlExpr{}, fmt.Errorf("unresolved filter reference %q", filter.Ref.String())
	}

	if having {
		measureExpr, err := e.aggregateExpr(ref, filter.Agg, sqlExpr{})
		if err != nil {
			return sqlExpr{}, err
		}
		return e.comparisonExpr(measureExpr, filter, ref)
	}

	if ref.toMany {
		return e.existsExpr(filter, ref)
	}

	fieldExpr := sqlExpr{text: e.plan.aliasFor(ref.pathKey) + "." + ref.field.Column.Name}
	return e.predicateExpr(fieldExpr, filter, ref)
}

func (e *emitter) existsExpr(filter *report.FieldFilter, ref *resolvedRef) (sqlExpr, error) {
	firstIdx := firstToManyIndex(ref.path)
	correlation := e.chainCorrelation(filter.Ref.Path, firstIdx)
	steps := e.buildChain(ref.path, firstIdx, "e")
	terminalAlias := steps[len(steps)-1].alias

	var sb strings.Builder
	var args []any

	sb.WriteString("EXISTS (SELECT 1")

	cc, err := e.chainClause(steps, correlation)
	if err != nil {
		return sqlExpr{}, err
	}

	fieldExpr := sqlExpr{text: terminalAlias + "." + ref.field.Column.Name}
	predicate, err := e.predicateExpr(fieldExpr, filter, ref)
	if err != nil {
		return sqlExpr{}, err
	}

	cc.render(&sb, &args, []string{predicate.text}, predicate.args)
	sb.WriteString(")")

	return sqlExpr{text: sb.String(), args: args}, nil
}

func (e *emitter) resolveFilterValue(filter *report.FieldFilter) (any, error) {
	if filter.Param != "" {
		value, ok := e.v.params[filter.Param]
		if !ok {
			return nil, fmt.Errorf("parameter %q has no value", filter.Param)
		}
		return value, nil
	}
	return filter.Value, nil
}

func (e *emitter) predicateExpr(
	fieldExpr sqlExpr,
	filter *report.FieldFilter,
	ref *resolvedRef,
) (sqlExpr, error) {
	col := fieldExpr.text

	//nolint:exhaustive // value-carrying operators are handled below
	switch filter.Operator {
	case dbtype.OpIsNull:
		return sqlExpr{text: col + " IS NULL", args: fieldExpr.args}, nil
	case dbtype.OpIsNotNull:
		return sqlExpr{text: col + " IS NOT NULL", args: fieldExpr.args}, nil
	case dbtype.OpToday, dbtype.OpYesterday, dbtype.OpTomorrow:
		start, end := e.relativeDayRange(filter.Operator)
		return sqlExpr{
			text: col + " >= ? AND " + col + " < ?",
			args: appendArgs(fieldExpr.args, start, end),
		}, nil
	case dbtype.OpThisWeek, dbtype.OpLastWeek, dbtype.OpThisMonth, dbtype.OpLastMonth,
		dbtype.OpThisQuarter, dbtype.OpLastQuarter, dbtype.OpThisYear, dbtype.OpLastYear:
		start, end := e.relativePeriodRange(filter.Operator)
		return sqlExpr{
			text: col + " >= ? AND " + col + " < ?",
			args: appendArgs(fieldExpr.args, start, end),
		}, nil
	}

	value, err := e.resolveFilterValue(filter)
	if err != nil {
		return sqlExpr{}, err
	}

	//nolint:exhaustive // valueless operators returned above; unknown operators fail in default
	switch filter.Operator {
	case dbtype.OpEqual, dbtype.OpNotEqual, dbtype.OpGreaterThan,
		dbtype.OpGreaterThanOrEqual, dbtype.OpLessThan, dbtype.OpLessThanOrEqual:
		coerced, cErr := coerceScalar(ref.field.Type, ref.field, value)
		if cErr != nil {
			return sqlExpr{}, cErr
		}
		return sqlExpr{
			text: col + " " + comparisonSQL(filter.Operator) + " ?",
			args: appendArgs(fieldExpr.args, coerced),
		}, nil
	case dbtype.OpContains, dbtype.OpStartsWith, dbtype.OpEndsWith:
		return e.patternPredicate(fieldExpr, filter.Operator, value)
	case dbtype.OpLike:
		return sqlExpr{text: col + " LIKE ?", args: appendArgs(fieldExpr.args, value)}, nil
	case dbtype.OpILike:
		return sqlExpr{text: col + " ILIKE ?", args: appendArgs(fieldExpr.args, value)}, nil
	case dbtype.OpIn, dbtype.OpNotIn:
		values, lErr := coerceList(ref.field, value)
		if lErr != nil {
			return sqlExpr{}, lErr
		}
		op := " IN (?)"
		if filter.Operator == dbtype.OpNotIn {
			op = " NOT IN (?)"
		}
		return sqlExpr{text: col + op, args: appendArgs(fieldExpr.args, bun.List(values))}, nil
	case dbtype.OpDateRange:
		start, end, rErr := coerceDateRange(value, e.loc)
		if rErr != nil {
			return sqlExpr{}, rErr
		}
		return sqlExpr{
			text: col + " >= ? AND " + col + " < ?",
			args: appendArgs(fieldExpr.args, start, end),
		}, nil
	case dbtype.OpLastNDays, dbtype.OpNextNDays:
		n, nErr := coerceInteger(value)
		if nErr != nil {
			return sqlExpr{}, nErr
		}
		start, end := e.relativeNDayRange(filter.Operator, n)
		return sqlExpr{
			text: col + " >= ? AND " + col + " < ?",
			args: appendArgs(fieldExpr.args, start, end),
		}, nil
	default:
		return sqlExpr{}, fmt.Errorf("operator %q is not supported", filter.Operator)
	}
}

func (e *emitter) patternPredicate(
	fieldExpr sqlExpr,
	op dbtype.Operator,
	value any,
) (sqlExpr, error) {
	s, ok := value.(string)
	if !ok {
		return sqlExpr{}, fmt.Errorf("expected a string pattern, got %T", value)
	}

	pattern := escapeLikePattern(s)
	//nolint:exhaustive // only pattern operators reach this helper
	switch op {
	case dbtype.OpContains:
		pattern = "%" + pattern + "%"
	case dbtype.OpStartsWith:
		pattern += "%"
	default:
		pattern = "%" + pattern
	}

	return sqlExpr{
		text: fieldExpr.text + " ILIKE ?",
		args: appendArgs(fieldExpr.args, pattern),
	}, nil
}

func (e *emitter) comparisonExpr(
	left sqlExpr,
	filter *report.FieldFilter,
	_ *resolvedRef,
) (sqlExpr, error) {
	value, err := e.resolveFilterValue(filter)
	if err != nil {
		return sqlExpr{}, err
	}

	coerced, err := coerceDecimal(value)
	if err != nil {
		return sqlExpr{}, err
	}

	return sqlExpr{
		text: left.text + " " + comparisonSQL(filter.Operator) + " ?",
		args: appendArgs(left.args, coerced),
	}, nil
}

func comparisonSQL(op dbtype.Operator) string {
	//nolint:exhaustive // only comparison operators reach this helper
	switch op {
	case dbtype.OpEqual:
		return "="
	case dbtype.OpNotEqual:
		return "<>"
	case dbtype.OpGreaterThan:
		return ">"
	case dbtype.OpGreaterThanOrEqual:
		return ">="
	case dbtype.OpLessThan:
		return "<"
	case dbtype.OpLessThanOrEqual:
		return "<="
	default:
		return "="
	}
}

func (e *emitter) now() time.Time {
	if e.req.NowUnix > 0 {
		return time.Unix(e.req.NowUnix, 0).In(e.loc)
	}
	return time.Now().In(e.loc)
}

func (e *emitter) relativeDayRange(op dbtype.Operator) (start, end int64) {
	now := e.now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, e.loc)

	//nolint:exhaustive // only single-day operators reach this helper
	switch op {
	case dbtype.OpYesterday:
		return today.AddDate(0, 0, -1).Unix(), today.Unix()
	case dbtype.OpTomorrow:
		return today.AddDate(0, 0, 1).Unix(), today.AddDate(0, 0, 2).Unix()
	default:
		return today.Unix(), today.AddDate(0, 0, 1).Unix()
	}
}

// relativePeriodRange resolves calendar-period operators to [start, end)
// epoch bounds in the organization's timezone. Weeks start on Monday
// (ISO 8601); quarters are calendar quarters.
func (e *emitter) relativePeriodRange(op dbtype.Operator) (start, end int64) {
	now := e.now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, e.loc)

	//nolint:exhaustive // only period operators reach this helper
	switch op {
	case dbtype.OpThisWeek, dbtype.OpLastWeek:
		weekday := (int(today.Weekday()) + 6) % 7
		weekStart := today.AddDate(0, 0, -weekday)
		if op == dbtype.OpLastWeek {
			weekStart = weekStart.AddDate(0, 0, -7)
		}
		return weekStart.Unix(), weekStart.AddDate(0, 0, 7).Unix()
	case dbtype.OpThisMonth, dbtype.OpLastMonth:
		monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, e.loc)
		if op == dbtype.OpLastMonth {
			monthStart = monthStart.AddDate(0, -1, 0)
		}
		return monthStart.Unix(), monthStart.AddDate(0, 1, 0).Unix()
	case dbtype.OpThisQuarter, dbtype.OpLastQuarter:
		quarterMonth := time.Month((int(today.Month())-1)/3*3 + 1)
		quarterStart := time.Date(today.Year(), quarterMonth, 1, 0, 0, 0, 0, e.loc)
		if op == dbtype.OpLastQuarter {
			quarterStart = quarterStart.AddDate(0, -3, 0)
		}
		return quarterStart.Unix(), quarterStart.AddDate(0, 3, 0).Unix()
	default:
		yearStart := time.Date(today.Year(), time.January, 1, 0, 0, 0, 0, e.loc)
		if op == dbtype.OpLastYear {
			yearStart = yearStart.AddDate(-1, 0, 0)
		}
		return yearStart.Unix(), yearStart.AddDate(1, 0, 0).Unix()
	}
}

func (e *emitter) relativeNDayRange(op dbtype.Operator, n int64) (start, end int64) {
	now := e.now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, e.loc)

	if op == dbtype.OpNextNDays {
		return today.Unix(), today.AddDate(0, 0, int(n)+1).Unix()
	}
	return today.AddDate(0, 0, -int(n)).Unix(), today.AddDate(0, 0, 1).Unix()
}

func appendArgs(existing []any, extra ...any) []any {
	result := make([]any, 0, len(existing)+len(extra))
	result = append(result, existing...)
	result = append(result, extra...)
	return result
}

func outputIndexByColumnID(outputs []outputColumn, columnID string) int {
	for i := range outputs {
		if outputs[i].id == columnID {
			return i
		}
	}
	return -1
}
