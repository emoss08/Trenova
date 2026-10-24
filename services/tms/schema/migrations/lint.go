package migrations

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/emoss08/keel/migrate"
)

const (
	RuleBunDirective   = "bun-directive"
	RuleReconcile      = "reconcile-automatic"
	RuleNoTxIdempotent = "notx-idempotent"
	RuleTenantRLS      = "tenant-rls"
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
		{Name: RuleBunDirective, Check: checkBunDirective},
		{Name: RuleReconcile, Check: checkReconcile},
		{Name: RuleNoTxIdempotent, Check: checkNoTxIdempotent},
		{Name: RuleTenantRLS, Check: checkTenantRLS},
	}
}

func checkBunDirective(m migrate.Migration) []migrate.Finding {
	var out []migrate.Finding
	for i, line := range strings.Split(m.SQL, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--bun:") {
			out = append(out, migrate.Finding{
				Line:     i + 1,
				Severity: migrate.SeverityError,
				Message: "keel does not read Bun directives: split a --keel:notx migration " +
					"with --keel:split and leave a transactional one unsplit",
			})
		}
	}

	return out
}

func checkReconcile(m migrate.Migration) []migrate.Finding {
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

func checkNoTxIdempotent(m migrate.Migration) []migrate.Finding {
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

func checkTenantRLS(m migrate.Migration) []migrate.Finding {
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
