package variable

import (
	"fmt"
	"strings"

	"github.com/xwb1989/sqlparser"
)

type QueryValidator struct {
	allowedTables      map[string]bool
	allowedParams      map[string]bool
	allowedFunctions   map[string]bool
	forbiddenFunctions map[string]bool
}

func NewQueryValidator() *QueryValidator {
	return &QueryValidator{
		allowedTables: map[string]bool{
			"customers":                 true,
			"invoices":                  true,
			"shipments":                 true,
			"organizations":             true,
			"business_units":            true,
			"users":                     true,
			"locations":                 true,
			"customer_billing_profiles": true,
			"customer_email_profiles":   true,
			"workers":                   true,
			"commodities":               true,
			"tractors":                  true,
			"trailers":                  true,
			"us_states":                 true,
			"variables":                 true,
			"variable_formats":          true,
		},
		allowedParams: map[string]bool{
			"contextId":      true,
			"orgId":          true,
			"buId":           true,
			"userId":         true,
			"customerId":     true,
			"invoiceId":      true,
			"shipmentId":     true,
			"locationId":     true,
			"workerId":       true,
			"organizationId": true,
		},
		allowedFunctions: map[string]bool{
			"upper": true, "lower": true, "concat": true, "substring": true,
			"trim": true, "ltrim": true, "rtrim": true, "length": true,
			"replace": true, "split_part": true, "regexp_replace": true,
			"now": true, "current_date": true, "current_timestamp": true,
			"date_part": true, "extract": true, "to_char": true, "to_date": true,
			"age": true, "date_trunc": true,
			"abs": true, "round": true, "ceil": true, "floor": true,
			"power": true, "sqrt": true, "mod": true,
			"cast": true, "coalesce": true, "nullif": true,
			"count": true, "sum": true, "avg": true, "max": true, "min": true,
		},
		forbiddenFunctions: map[string]bool{
			"pg_sleep": true, "pg_notify": true, "pg_terminate_backend": true,
			"pg_cancel_backend": true, "pg_reload_conf": true,
			"pg_read_file": true, "pg_ls_dir": true, "pg_stat_file": true,
			"dblink": true, "file_fdw": true, "postgres_fdw": true,
			"set_config": true, "current_setting": true,
		},
	}
}

func (v *QueryValidator) Validate(query string) error {
	if query == "" {
		return ErrQueryCannotBeEmpty
	}

	// Check for semicolons first (before parsing, as parser might fail on them)
	if strings.Contains(query, ";") {
		return ErrSemicolonsNotAllowed
	}

	// Check for comments before parsing
	if strings.Contains(query, "--") || strings.Contains(query, "/*") ||
		strings.Contains(query, "*/") {
		return ErrCommentsNotAllowed
	}

	stmt, err := sqlparser.Parse(query)
	if err != nil {
		return fmt.Errorf("failed to parse SQL: %w", err)
	}

	selectStmt, ok := stmt.(*sqlparser.Select)
	if !ok {
		return ErrQueryMustBeSelect
	}

	if err = v.validateSelectStatement(selectStmt); err != nil {
		return err
	}

	return nil
}

func (v *QueryValidator) validateSelectStatement(stmt *sqlparser.Select) error {
	if err := v.validateTableExpr(stmt.From); err != nil {
		return err
	}

	if stmt.Where != nil {
		if err := v.validateExpr(stmt.Where.Expr); err != nil {
			return err
		}
	}

	for _, expr := range stmt.SelectExprs {
		if err := v.validateSelectExpr(expr); err != nil {
			return err
		}
	}

	if stmt.GroupBy != nil {
		for _, expr := range stmt.GroupBy {
			if err := v.validateExpr(expr); err != nil {
				return err
			}
		}
	}

	return nil
}

func (v *QueryValidator) validateTableExpr(tables sqlparser.TableExprs) error {
	for _, table := range tables {
		switch t := table.(type) {
		case *sqlparser.AliasedTableExpr:
			if err := v.validateSimpleTableExpr(t.Expr); err != nil {
				return err
			}
		case *sqlparser.JoinTableExpr:
			if err := v.validateTableExpr([]sqlparser.TableExpr{t.LeftExpr}); err != nil {
				return err
			}
			if err := v.validateTableExpr([]sqlparser.TableExpr{t.RightExpr}); err != nil {
				return err
			}
			if t.Condition.On != nil {
				if err := v.validateExpr(t.Condition.On); err != nil {
					return err
				}
			}
		case *sqlparser.ParenTableExpr:
			if err := v.validateTableExpr(t.Exprs); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported table expression type: %T", t)
		}
	}
	return nil
}

func (v *QueryValidator) validateSimpleTableExpr(expr sqlparser.SimpleTableExpr) error {
	switch t := expr.(type) {
	case sqlparser.TableName:
		tableName := t.Name.String()
		if !v.allowedTables[strings.ToLower(tableName)] {
			return fmt.Errorf("table not allowed: %s", tableName)
		}
	case *sqlparser.Subquery:
		if err := v.validateSelectStatement(t.Select.(*sqlparser.Select)); err != nil { //nolint:errcheck // no need to check error
			return fmt.Errorf("invalid subquery: %w", err)
		}
	default:
		return fmt.Errorf("unsupported table expression: %T", t)
	}
	return nil
}

func (v *QueryValidator) validateSelectExpr(expr sqlparser.SelectExpr) error {
	switch e := expr.(type) {
	case *sqlparser.StarExpr:
		return nil
	case *sqlparser.AliasedExpr:
		return v.validateExpr(e.Expr)
	case sqlparser.Nextval:
		return ErrNEXTVALNotAllowed
	default:
		return fmt.Errorf("unsupported select expression: %T", e)
	}
}

func (v *QueryValidator) validateExpr( //nolint:gocognit,gocyclo,cyclop,funlen // this is a single purpose function it's fine.
	expr sqlparser.Expr,
) error {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *sqlparser.AndExpr:
		if err := v.validateExpr(e.Left); err != nil {
			return err
		}
		return v.validateExpr(e.Right)

	case *sqlparser.OrExpr:
		if err := v.validateExpr(e.Left); err != nil {
			return err
		}
		return v.validateExpr(e.Right)

	case *sqlparser.NotExpr:
		return v.validateExpr(e.Expr)

	case *sqlparser.ComparisonExpr:
		if err := v.validateExpr(e.Left); err != nil {
			return err
		}
		return v.validateExpr(e.Right)

	case *sqlparser.RangeCond:
		if err := v.validateExpr(e.Left); err != nil {
			return err
		}
		if err := v.validateExpr(e.From); err != nil {
			return err
		}
		return v.validateExpr(e.To)

	case *sqlparser.IsExpr:
		return v.validateExpr(e.Expr)

	case *sqlparser.ExistsExpr:
		return v.validateSelectStatement(e.Subquery.Select.(*sqlparser.Select)) //nolint:errcheck // no need to check error

	case *sqlparser.SQLVal:
		if e.Type == sqlparser.ValArg {
			paramName := strings.TrimPrefix(string(e.Val), ":")
			if !v.allowedParams[paramName] {
				return fmt.Errorf("parameter not allowed: %s", paramName)
			}
		}
		return nil

	case *sqlparser.ColName:
		return nil

	case *sqlparser.FuncExpr:
		return v.validateFunction(e)

	case *sqlparser.Subquery:
		return v.validateSelectStatement(e.Select.(*sqlparser.Select)) //nolint:errcheck // no need to check error

	case *sqlparser.CaseExpr:
		if e.Expr != nil {
			if err := v.validateExpr(e.Expr); err != nil {
				return err
			}
		}
		for _, when := range e.Whens {
			if err := v.validateExpr(when.Cond); err != nil {
				return err
			}
			if err := v.validateExpr(when.Val); err != nil {
				return err
			}
		}
		if e.Else != nil {
			return v.validateExpr(e.Else)
		}
		return nil

	case sqlparser.BoolVal, *sqlparser.NullVal:
		return nil

	case *sqlparser.ParenExpr:
		return v.validateExpr(e.Expr)

	case *sqlparser.BinaryExpr:
		if err := v.validateExpr(e.Left); err != nil {
			return err
		}
		return v.validateExpr(e.Right)

	default:
		return fmt.Errorf("unsupported expression type: %T", e)
	}
}

func (v *QueryValidator) validateFunction(fn *sqlparser.FuncExpr) error {
	funcName := strings.ToLower(fn.Name.String())

	if v.forbiddenFunctions[funcName] {
		return fmt.Errorf("forbidden function: %s", funcName)
	}

	if !v.allowedFunctions[funcName] {
		return fmt.Errorf("function not allowed: %s", funcName)
	}

	for _, arg := range fn.Exprs {
		if err := v.validateSelectExpr(arg); err != nil {
			return fmt.Errorf("invalid function argument in %s: %w", funcName, err)
		}
	}

	return nil
}

func (v *QueryValidator) ValidateWithTables(query string) error {
	return v.Validate(query)
}

func (v *QueryValidator) AddAllowedTable(table string) {
	v.allowedTables[strings.ToLower(table)] = true
}

func (v *QueryValidator) RemoveAllowedTable(table string) {
	delete(v.allowedTables, strings.ToLower(table))
}

func (v *QueryValidator) GetAllowedTables() []string {
	tables := make([]string, 0, len(v.allowedTables))
	for table := range v.allowedTables {
		tables = append(tables, table)
	}
	return tables
}

func (v *QueryValidator) AddAllowedParam(param string) {
	v.allowedParams[param] = true
}

func (v *QueryValidator) AddAllowedFunction(fn string) {
	v.allowedFunctions[strings.ToLower(fn)] = true
}
