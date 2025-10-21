package variable

import (
	"fmt"
	"strings"
)

type FormatSQLValidator struct {
	allowedFunctions map[string]bool
}

func NewFormatSQLValidator() *FormatSQLValidator {
	return &FormatSQLValidator{
		allowedFunctions: map[string]bool{
			"upper": true, "lower": true, "concat": true, "substring": true,
			"trim": true, "ltrim": true, "rtrim": true, "length": true,
			"replace": true, "split_part": true, "regexp_replace": true,
			"initcap": true, "chr": true, "ascii": true, "quote_ident": true,
			"quote_literal": true, "quote_nullable": true, "format": true,
			"left": true, "right": true, "reverse": true, "repeat": true,
			"lpad": true, "rpad": true, "overlay": true, "position": true,
			"to_char": true, "to_date": true, "to_timestamp": true,
			"date_part": true, "extract": true, "date_trunc": true,
			"age": true, "now": true, "current_date": true,
			"current_timestamp": true, "localtimestamp": true,
			"make_date": true, "make_time": true, "make_timestamp": true,
			"abs": true, "round": true, "ceil": true, "floor": true,
			"trunc": true, "power": true, "sqrt": true, "mod": true,
			"sign": true, "random": true, "exp": true, "ln": true,
			"log": true, "log10": true, "pi": true, "radians": true,
			"degrees": true, "sin": true, "cos": true, "tan": true,
			"cast": true,
			"case": true, "coalesce": true, "nullif": true,
			"greatest": true, "least": true,
			"jsonb_build_object": true, "json_build_object": true,
			"to_json": true, "to_jsonb": true,
			"jsonb_pretty": true, "json_extract_path_text": true,
			"array_to_string": true, "string_to_array": true,
			"array_length": true, "array_upper": true, "array_lower": true,
		},
	}
}

func (v *FormatSQLValidator) Validate(sql string) error {
	if sql == "" {
		return ErrFormatSQLCannotBeEmpty
	}

	if strings.Contains(sql, ";") {
		return ErrFormatSemicolonsNotAllowed
	}

	if strings.Contains(sql, "--") || strings.Contains(sql, "/*") || strings.Contains(sql, "*/") {
		return ErrFormatCommentsNotAllowed
	}

	upperSQL := strings.ToUpper(sql)

	if strings.Contains(upperSQL, "(SELECT") {
		return ErrSubqueriesNotAllowed
	}

	forbiddenKeywords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER",
		"TRUNCATE", "GRANT", "REVOKE", "FROM", "JOIN", "WHERE",
		"GROUP BY", "HAVING", "ORDER BY", "UNION", "INTERSECT", "EXCEPT",
		"WITH", "RETURNING", "INTO", "BEGIN", "COMMIT", "ROLLBACK",
		"SAVEPOINT", "SET", "DECLARE", "CALL", "MERGE",
	}

	for _, keyword := range forbiddenKeywords {
		if strings.Contains(upperSQL, keyword) {
			return fmt.Errorf(
				"format SQL cannot contain %s - it should be a single expression, not a query",
				keyword,
			)
		}
	}

	dangerousFuncs := []string{
		"pg_sleep", "pg_notify", "pg_terminate_backend",
		"pg_cancel_backend", "pg_reload_conf",
		"pg_read_file", "pg_ls_dir", "pg_stat_file",
		"pg_read_binary_file", "pg_ls_logdir", "pg_ls_waldir",
		"dblink", "file_fdw", "postgres_fdw",
		"set_config", "current_setting", "pg_settings",
		"pg_database_size", "pg_relation_size", "pg_total_relation_size",
		"pg_table_size", "pg_indexes_size",
		"current_user", "session_user", "user",
		"pg_has_role", "pg_backend_pid",
		"pg_current_xact_id", "pg_current_snapshot",
		"txid_current", "txid_snapshot",
		"inet_client_addr", "inet_client_port",
		"inet_server_addr", "inet_server_port",
	}

	lowerSQL := strings.ToLower(sql)
	for _, fn := range dangerousFuncs {
		if strings.Contains(lowerSQL, fn) {
			return fmt.Errorf("dangerous function detected in format SQL: %s", fn)
		}
	}

	openCount := strings.Count(sql, "(")
	closeCount := strings.Count(sql, ")")
	if openCount != closeCount {
		return ErrUnbalancedParentheses
	}

	if !strings.Contains(sql, ":value") {
		return ErrFormatValuePlaceholderNotAllowed
	}

	return nil
}

func (v *FormatSQLValidator) IsAllowedFunction(funcName string) bool {
	return v.allowedFunctions[strings.ToLower(funcName)]
}

func (v *FormatSQLValidator) AddAllowedFunction(funcName string) {
	v.allowedFunctions[strings.ToLower(funcName)] = true
}

func (v *FormatSQLValidator) RemoveAllowedFunction(funcName string) {
	delete(v.allowedFunctions, strings.ToLower(funcName))
}
