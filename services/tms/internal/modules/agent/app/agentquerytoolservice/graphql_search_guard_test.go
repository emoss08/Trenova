package agentquerytoolservice

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// searchedElsewhere are the lists whose free text a bespoke resolver search
// matches, outside any search configuration, and that name no searched field
// of their own record, each with why its query reveals nothing withheld.
var searchedElsewhere = map[string]string{
	"inbox_list_capture_batches": "matches linked records only: the scanner and computer " +
		"names, the scanning person's name, a code on a page and the PRO or bill of lading " +
		"number of a shipment the stack was filed to, each Internal on its own kind",
}

var (
	searchVectorDefinition = regexp.MustCompile(
		`(?s)"?search_vector"?\s+tsvector\s+GENERATED\s+ALWAYS\s+AS\s*\(`)
	migrationTable = regexp.MustCompile(
		`(?i)(?:ALTER|CREATE)\s+TABLE\s+(?:IF\s+(?:NOT\s+)?EXISTS\s+)?(?:ONLY\s+)?"?([a-z_][a-z0-9_]*)"?`)
	vectorColumn = regexp.MustCompile(
		`(?i)(?:COALESCE|enum_to_text|array_to_string)\(\s*"?([a-z_][a-z0-9_]*)"?`)
	vectorFunctions = []string{"coalesce", "enum_to_text", "array_to_string"}
)

// searchVectors reads each table's search vector from the migrations, as the
// database builds it: the latest definition of the generated column wins.
func searchVectors(t *testing.T) map[string][]string {
	t.Helper()

	dir := filepath.Join("..", "..", "..", "..", "infrastructure", "postgres", "migrations")
	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "the migrations are read from %s", dir)
	sort.Strings(files)

	vectors := make(map[string][]string)
	for _, file := range files {
		raw, readErr := os.ReadFile(file)
		require.NoError(t, readErr)
		text := string(raw)
		for _, loc := range searchVectorDefinition.FindAllStringIndex(text, -1) {
			tables := migrationTable.FindAllStringSubmatch(text[:loc[0]], -1)
			if len(tables) == 0 {
				continue
			}
			depth, end := 1, loc[1]
			for ; end < len(text) && depth > 0; end++ {
				switch text[end] {
				case '(':
					depth++
				case ')':
					depth--
				}
			}
			columns := make([]string, 0, 4)
			for _, match := range vectorColumn.FindAllStringSubmatch(text[loc[1]:end], -1) {
				column := strings.ToLower(match[1])
				if slices.Contains(vectorFunctions, column) {
					continue
				}
				if !slices.Contains(columns, column) {
					columns = append(columns, column)
				}
			}
			vectors[tables[len(tables)-1][1]] = columns
		}
	}

	return vectors
}

// Every converted list that offers free text guards it: a table input's
// query matches the record's search configuration, which the guard reads as
// the repository's query builder does, so a spec cannot leave a searched
// column out; where the configuration searches through a stored vector, the
// vector the migrations build matches no column the configuration does not
// name; and a bespoke search names the fields it matches, or says why it
// reveals nothing withheld. At Internal each list refuses a query exactly
// when a field it matches is above Internal.
func TestConvertedLists_GuardEveryFieldTheirQueryMatches(t *testing.T) {
	t.Parallel()

	vectors := searchVectors(t)
	registry := fieldsensitivity.DefaultRegistry()
	specs := graphQLListSpecs()
	for idx := range specs {
		spec := &specs[idx]
		built, err := newGraphQLReadTool(spec, nil, &mocks.DenyResourcesPermissionEngine{})
		require.NoError(t, err, spec.name)
		tool, paged := built.(*graphQLListTool)
		if !paged {
			continue
		}
		properties, _ := tool.ParamSchema()["properties"].(map[string]any)
		if _, offered := properties[paramQuery]; !offered {
			continue
		}

		switch {
		case tool.input != "":
			require.NotNil(t, tool.read.Entity, "%s: a table input's query needs the record "+
				"its search configuration belongs to", spec.name)
			entity := tool.read.Entity()
			config := entity.GetPostgresSearchConfig()
			if config.UseSearchVector {
				vector, built := vectors[entity.GetTableName()]
				require.True(t, built, "%s: %s searches a vector no migration builds",
					spec.name, entity.GetTableName())
				named := make([]string, 0, len(config.SearchableFields))
				for _, field := range config.SearchableFields {
					named = append(named, field.Name)
				}
				for _, column := range vector {
					assert.Contains(t, named, column, "%s: the search vector of %s matches %s, "+
						"which its search configuration does not name, so the guard cannot "+
						"judge it", spec.name, entity.GetTableName(), column)
				}
			}
			assert.NotEmpty(t, tool.searched, "%s: its query matches no field", spec.name)
		default:
			_, exempt := searchedElsewhere[spec.name]
			assert.True(t, len(spec.searches) > 0 || exempt, "%s: a bespoke search names the "+
				"fields it matches in searches, or says in searchedElsewhere why it reveals "+
				"nothing withheld", spec.name)
		}

		hidden := slices.ContainsFunc(tool.searched, func(field string) bool {
			resource, name := tool.fieldOwner(field)
			return !fieldsensitivity.Visible(registry, resource, name,
				permission.SensitivityInternal)
		})
		call := keyedCall(map[string]any{paramQuery: "probe"})
		call.DataAccessCeiling = permission.SensitivityInternal
		refused := tool.refuseSearch(t.Context(), call) != nil
		assert.Equal(t, hidden, refused, "%s: at Internal a query is refused exactly when it "+
			"matches a withheld field (%s)", spec.name, strings.Join(tool.searched, ", "))
	}
	for name := range searchedElsewhere {
		assert.True(t, slices.ContainsFunc(specs, func(spec graphQLListSpec) bool {
			return spec.name == name
		}), "searchedElsewhere names %s, which is no converted list", name)
	}
}

// What each list's query matches, and so where it is refused: at Internal
// wherever a matched field is Restricted, and at every access where one is
// Confidential, and never where everything it matches is Internal.
func TestConvertedLists_RefuseAQueryOverWithheldFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tool       string
		withheld   string
		restricted bool
		args       map[string]any
	}{
		{tool: "worker_list_time_off", withheld: "reason", restricted: true},
		{tool: "ledger_list_manual_journals", withheld: "reason", restricted: true},
		{tool: "ledger_list_journal_reversals", withheld: "reasonText", restricted: true},
		{tool: "settlement_list_driver_expenses", withheld: "description", restricted: true},
		{tool: "settlement_list_advances", withheld: "notes", restricted: true},
		{tool: "settlement_list_escrow_accounts", withheld: "balanceMinor", restricted: true},
		{tool: "receivable_list_customer_payments", withheld: "memo", restricted: true},
		{tool: "inbox_list_email_profiles", withheld: "host", restricted: true},
		{tool: "report_list_table_change_alerts", withheld: "name", restricted: true},
		{tool: "factoring_list_noas"},
		{tool: "fuel_list_cards"},
		{tool: "accounting_list_sync_records", args: map[string]any{"system": "Xero"}},
		{tool: "accounting_list_mapping_gaps", args: map[string]any{"system": "Xero"}},
		{tool: "customer_list"},
	}
	for _, c := range cases {
		t.Run(c.tool, func(t *testing.T) {
			t.Parallel()

			spec := keyedSpec(t, c.tool)
			internal, err := newGraphQLReadTool(spec, &keyedReader{},
				&mocks.DenyResourcesPermissionEngine{})
			require.NoError(t, err)
			list, ok := internal.(*graphQLListTool)
			require.True(t, ok)
			if c.withheld != "" {
				assert.Contains(t, list.searched, c.withheld)
			}

			args := map[string]any{paramQuery: "probe"}
			for name, value := range c.args {
				args[name] = value
			}
			call := keyedCall(args)
			call.DataAccessCeiling = permission.SensitivityInternal
			_, err = internal.Query(t.Context(), call)
			if c.withheld == "" {
				require.NoError(t, err, "everything its query matches is Internal")
				return
			}
			require.ErrorContains(t, err, "cannot search by query")

			reached, err := newGraphQLReadTool(spec, &keyedReader{}, &restrictedRole{})
			require.NoError(t, err)
			call = keyedCall(map[string]any{paramQuery: "probe"})
			call.DataAccessCeiling = permission.SensitivityRestricted
			_, err = reached.Query(t.Context(), call)
			if c.restricted {
				require.NoError(t, err, "a reader whose access reaches Restricted may search")
				return
			}
			require.ErrorContains(t, err, "cannot search by query",
				"a Confidential field is never matched for anyone")
		})
	}
}

// A fuel card's query matches its label and last four digits, both Internal,
// and not the card provider's own id, which is Confidential and which the
// card's search vector no longer holds: a query reads at Internal.
func TestFuelCards_SearchTheirLabelAndLastFourOnly(t *testing.T) {
	t.Parallel()

	spec := keyedSpec(t, "fuel_list_cards")
	built, err := newGraphQLReadTool(spec, &keyedReader{}, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)
	list, ok := built.(*graphQLListTool)
	require.True(t, ok)
	assert.Equal(t, []string{"label", "lastFour"}, list.searched)
	registry := fieldsensitivity.DefaultRegistry()
	for _, field := range list.searched {
		assert.True(t, fieldsensitivity.Visible(registry, permission.ResourceFuelCard, field,
			permission.SensitivityInternal), field)
	}
	assert.NotContains(t, list.searched, "externalCardId")
	assert.NotContains(t, searchVectors(t)["fuel_cards"], "external_card_id",
		"the vector the migrations build no longer matches the provider's id")
}
