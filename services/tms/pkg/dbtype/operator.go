package dbtype

type Operator string

const (
	OpEqual              = Operator("eq")
	OpNotEqual           = Operator("ne")
	OpGreaterThan        = Operator("gt")
	OpGreaterThanOrEqual = Operator("gte")
	OpLessThan           = Operator("lt")
	OpLessThanOrEqual    = Operator("lte")
	OpContains           = Operator("contains")
	OpStartsWith         = Operator("startswith")
	OpEndsWith           = Operator("endswith")
	OpLike               = Operator("like")
	OpILike              = Operator("ilike")
	OpIn                 = Operator("in")
	OpNotIn              = Operator("notin")
	OpIsNull             = Operator("isnull")
	OpIsNotNull          = Operator("isnotnull")
	OpDateRange          = Operator("daterange")
	OpLastNDays          = Operator("lastndays")
	OpNextNDays          = Operator("nextndays")
	OpToday              = Operator("today")
	OpYesterday          = Operator("yesterday")
	OpTomorrow           = Operator("tomorrow")
	OpThisWeek           = Operator("thisweek")
	OpLastWeek           = Operator("lastweek")
	OpThisMonth          = Operator("thismonth")
	OpLastMonth          = Operator("lastmonth")
	OpThisQuarter        = Operator("thisquarter")
	OpLastQuarter        = Operator("lastquarter")
	OpThisYear           = Operator("thisyear")
	OpLastYear           = Operator("lastyear")
	OpCountGt            = Operator("countgt")
	OpCountLt            = Operator("countlt")
	OpCountEq            = Operator("counteq")
	OpCountGte           = Operator("countgte")
	OpCountLte           = Operator("countlte")
)

var knownOperators = map[Operator]struct{}{
	OpEqual:              {},
	OpNotEqual:           {},
	OpGreaterThan:        {},
	OpGreaterThanOrEqual: {},
	OpLessThan:           {},
	OpLessThanOrEqual:    {},
	OpContains:           {},
	OpStartsWith:         {},
	OpEndsWith:           {},
	OpLike:               {},
	OpILike:              {},
	OpIn:                 {},
	OpNotIn:              {},
	OpIsNull:             {},
	OpIsNotNull:          {},
	OpDateRange:          {},
	OpLastNDays:          {},
	OpNextNDays:          {},
	OpToday:              {},
	OpYesterday:          {},
	OpTomorrow:           {},
	OpThisWeek:           {},
	OpLastWeek:           {},
	OpThisMonth:          {},
	OpLastMonth:          {},
	OpThisQuarter:        {},
	OpLastQuarter:        {},
	OpThisYear:           {},
	OpLastYear:           {},
	OpCountGt:            {},
	OpCountLt:            {},
	OpCountEq:            {},
	OpCountGte:           {},
	OpCountLte:           {},
}

func (o Operator) IsValid() bool {
	_, ok := knownOperators[o]
	return ok
}

func (o Operator) String() string { return string(o) }

func (d SortDirection) IsValid() bool {
	return d == SortDirectionAsc || d == SortDirectionDesc
}
