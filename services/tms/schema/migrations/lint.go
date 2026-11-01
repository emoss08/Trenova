package migrations

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/emoss08/keel/migrate"
)

const (
	RuleReconcile      = "reconcile-automatic"
	RuleNoTxIdempotent = "notx-idempotent"
	RuleTenantRLS      = "tenant-rls"
	RuleTableRewrite   = "table-rewrite"
)

var (
	reconcileCall = regexp.MustCompile(`(?i)\bselect\s+trenova_rls\.reconcile\s*\(\s*\)`)
	createIndexCc = regexp.MustCompile(`(?i)\bcreate\s+(?:unique\s+)?index\s+concurrently\b`)
	dropIndexCc   = regexp.MustCompile(`(?i)\bdrop\s+index\s+concurrently\b`)
	ifNotExists   = regexp.MustCompile(`(?i)^\s*if\s+not\s+exists\b`)
	ifExists      = regexp.MustCompile(`(?i)^\s*if\s+exists\b`)
	createTable   = regexp.MustCompile(
		`(?i)\bcreate\s+(?:unlogged\s+)?table\s+(?:if\s+not\s+exists\s+)?((?:"?[a-z0-9_]+"?\.)?"?[a-z0-9_]+"?)\s*\(`,
	)
)

func LintRules() []migrate.LintRule {
	return []migrate.LintRule{
		{Name: RuleReconcile, Check: byPointer(checkReconcile)},
		{Name: RuleNoTxIdempotent, Check: byPointer(checkNoTxIdempotent)},
		{Name: RuleTenantRLS, Check: byPointer(checkTenantRLS)},
		{Name: RuleTableRewrite, Check: byPointer(checkTableRewrite)},
	}
}

func byPointer(
	check func(*migrate.Migration) []migrate.Finding,
) func(migrate.Migration) []migrate.Finding {
	return func(m migrate.Migration) []migrate.Finding {
		return check(&m)
	}
}

func checkReconcile(m *migrate.Migration) []migrate.Finding {
	code := codeOnly(m.SQL)
	var out []migrate.Finding
	for _, loc := range reconcileCall.FindAllStringIndex(code, -1) {
		out = append(out, migrate.Finding{
			Line:     lineOf(code, loc[0]),
			Severity: migrate.SeverityWarning,
			Message: "trenova-db runs SELECT trenova_rls.reconcile() after every run that " +
				"applies a migration; drop this call",
		})
	}

	return out
}

func checkNoTxIdempotent(m *migrate.Migration) []migrate.Finding {
	if !m.Directives.NoTx {
		return nil
	}

	code := codeOnly(m.SQL)
	var out []migrate.Finding
	check := func(re, guard *regexp.Regexp, want string) {
		for _, loc := range re.FindAllStringIndex(code, -1) {
			if guard.MatchString(code[loc[1]:]) {
				continue
			}
			out = append(out, migrate.Finding{
				Line:     lineOf(code, loc[0]),
				Severity: migrate.SeverityError,
				Message: fmt.Sprintf(
					"a notx migration that fails part-way keeps the statements before it; "+
						"write %s so running it again succeeds",
					want,
				),
			})
		}
	}
	check(createIndexCc, ifNotExists, "CREATE INDEX CONCURRENTLY IF NOT EXISTS")
	check(dropIndexCc, ifExists, "DROP INDEX CONCURRENTLY IF EXISTS")

	return out
}

func checkTenantRLS(m *migrate.Migration) []migrate.Finding {
	code := codeOnly(m.SQL)
	lower := strings.ToLower(code)
	var out []migrate.Finding
	for _, loc := range createTable.FindAllStringSubmatchIndex(code, -1) {
		schema, table := splitTableName(code[loc[2]:loc[3]])
		if schema != "" && schema != "public" {
			continue
		}

		body := tableBody(code, loc[1]-1)
		if hasStandardTenantColumns(body) || mentionsPolicy(lower, table) {
			continue
		}

		out = append(out, migrate.Finding{
			Line:     lineOf(code, loc[0]),
			Severity: migrate.SeverityWarning,
			Message: fmt.Sprintf(
				"%s has no non-null organization_id and business_unit_id, so reconcile() cannot "+
					"protect it: call trenova_rls.apply_policy for it or add it to "+
					"trenova_rls.global_tables (docs/engineering/row-level-security.md)",
				table,
			),
		})
	}

	return out
}

var (
	alterTable = regexp.MustCompile(
		`(?i)\balter\s+table\s+(?:if\s+exists\s+)?(?:only\s+)?((?:"?[a-z0-9_]+"?\.)?"?[a-z0-9_]+"?)`,
	)
	addColumn = regexp.MustCompile(
		`(?is)^\s*add\s+(?:column\s+)?(?:if\s+not\s+exists\s+)?("?[a-z0-9_]+"?)\s+(.*)$`,
	)
	addOther = regexp.MustCompile(
		`(?i)^\s*add\s+(?:constraint|primary|unique|check|foreign|exclude)\b`,
	)
	serialType     = regexp.MustCompile(`(?i)^(?:small|big)?serial[248]?\b`)
	identityColumn = regexp.MustCompile(
		`(?i)\bgenerated\s+(?:always|by\s+default)\s+as\s+identity\b`,
	)
	storedColumn    = regexp.MustCompile(`(?i)\bgenerated\s+always\s+as\b[\s\S]*\bstored\b`)
	volatileDefault = regexp.MustCompile(
		`(?i)\bdefault\b[\s\S]*\b(gen_random_uuid|uuid_generate_v1(?:mc)?|uuid_generate_v4|uuidv4|uuidv7|random|clock_timestamp|timeofday|nextval)\s*\(`,
	)
)

type sqlSpan struct {
	text  string
	start int
}

func checkTableRewrite(m *migrate.Migration) []migrate.Finding {
	code := codeOnly(m.SQL)
	created := map[string]int{}
	for _, loc := range createTable.FindAllStringSubmatchIndex(code, -1) {
		name := normalTable(code[loc[2]:loc[3]])
		if _, ok := created[name]; !ok {
			created[name] = loc[0]
		}
	}

	var out []migrate.Finding
	for _, loc := range alterTable.FindAllStringSubmatchIndex(code, -1) {
		table := code[loc[2]:loc[3]]
		if at, ok := created[normalTable(table)]; ok && at < loc[0] {
			continue
		}

		for _, action := range splitTopLevel(code, loc[1]) {
			column, why, ok := rewritingColumn(action.text)
			if !ok {
				continue
			}
			out = append(out, migrate.Finding{
				Line:     lineOf(code, action.start),
				Severity: migrate.SeverityWarning,
				Message: fmt.Sprintf(
					"adding %s to %s with %s rewrites every row under an ACCESS EXCLUSIVE lock; "+
						"add a plain nullable column, backfill it in batches and keep it "+
						"current from the write path or a trigger",
					column, table, why,
				),
			})
		}
	}

	return out
}

func rewritingColumn(action string) (column, why string, ok bool) {
	if addOther.MatchString(action) {
		return "", "", false
	}
	match := addColumn.FindStringSubmatch(action)
	if match == nil {
		return "", "", false
	}

	def := match[2]
	switch {
	case serialType.MatchString(def):
		return match[1], "a serial type", true
	case identityColumn.MatchString(def):
		return match[1], "an identity", true
	case storedColumn.MatchString(def):
		return match[1], "a stored generated expression", true
	}
	if fn := volatileDefault.FindStringSubmatch(def); fn != nil {
		return match[1], fmt.Sprintf("the volatile default %s()", strings.ToLower(fn[1])), true
	}

	return "", "", false
}

func splitTopLevel(code string, from int) []sqlSpan {
	var out []sqlSpan
	depth, start, inString := 0, from, false
	for i := from; i < len(code); i++ {
		c := code[i]
		switch {
		case inString:
			if c == '\'' {
				inString = false
			}
		case c == '\'':
			inString = true
		case c == '(':
			depth++
		case c == ')':
			depth--
		case depth == 0 && (c == ',' || c == ';'):
			out = append(out, trimmedSpan(code, start, i))
			if c == ';' {
				return out
			}
			start = i + 1
		}
	}

	return append(out, trimmedSpan(code, start, len(code)))
}

func trimmedSpan(code string, start, end int) sqlSpan {
	text := strings.TrimLeft(code[start:end], " \t\r\n")

	return sqlSpan{text: text, start: end - len(text)}
}

func normalTable(name string) string {
	_, table := splitTableName(name)

	return strings.ToLower(table)
}

var (
	orgColumn = regexp.MustCompile(`(?im)^\s*"?organization_id"?\s+[^,]*\bnot\s+null\b`)
	buColumn  = regexp.MustCompile(`(?im)^\s*"?business_unit_id"?\s+`)
)

func hasStandardTenantColumns(body string) bool {
	return orgColumn.MatchString(body) && buColumn.MatchString(body)
}

func mentionsPolicy(lowerSQL, table string) bool {
	name := strings.ToLower(table)
	quoted := regexp.QuoteMeta(name)
	global := regexp.MustCompile(`global_tables[\s\S]*'` + quoted + `'`)
	policy := regexp.MustCompile(`apply_policy\s*\(\s*'(?:public\.)?"?` + quoted + `"?'`)

	return global.MatchString(lowerSQL) || policy.MatchString(lowerSQL)
}

func splitTableName(name string) (schema, table string) {
	name = strings.ReplaceAll(name, `"`, "")
	if before, after, ok := strings.Cut(name, "."); ok {
		return strings.ToLower(before), after
	}

	return "", name
}

func tableBody(code string, open int) string {
	depth := 0
	for i := open; i < len(code); i++ {
		switch code[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return code[open+1 : i]
			}
		}
	}

	return code[open+1:]
}

func lineOf(text string, offset int) int {
	return strings.Count(text[:offset], "\n") + 1
}

func codeOnly(sql string) string {
	out := []byte(sql)
	inString := false
	for i := 0; i < len(out); i++ {
		switch {
		case inString:
			if out[i] == '\'' {
				inString = false
			}
		case out[i] == '\'':
			inString = true
		case out[i] == '-' && i+1 < len(out) && out[i+1] == '-':
			for i < len(out) && out[i] != '\n' {
				out[i] = ' '
				i++
			}
		case out[i] == '/' && i+1 < len(out) && out[i+1] == '*':
			for i < len(out) && (out[i] != '*' || i+1 >= len(out) || out[i+1] != '/') {
				if out[i] != '\n' {
					out[i] = ' '
				}
				i++
			}
			if i+1 < len(out) {
				out[i], out[i+1] = ' ', ' '
				i++
			}
		}
	}

	return string(out)
}
