package recordlabelrepository

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/recordnumberports"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const migrationsDir = "../../../../../../schema/migrations"

var numberIndexes = map[recordkind.Kind]string{
	recordkind.Kind(permission.ResourceShipment):            "uq_pro_number",
	recordkind.Kind(permission.ResourceOrder):               "idx_orders_order_number",
	recordkind.Kind(permission.ResourceInvoice):             "uk_invoices_number",
	recordkind.BillingQueueItem:                             "uq_billing_queue_items_number_tenant",
	recordkind.Kind(permission.ResourceFactoringSubmission): "uq_factoring_submissions_number",
	recordkind.Kind(permission.ResourceServiceFailure):      "ux_sf_tenant_number",
	recordkind.Kind(permission.ResourceJournalEntry):        "uq_journal_entries_entry_number",
	recordkind.Kind(permission.ResourceDriverSettlement):    "uq_driver_settlements_number",
	recordkind.Kind(permission.ResourceCarrierSettlement):   "uq_carrier_settlements_number",
	recordkind.Kind(permission.ResourceTractor):             "idx_tractors_code",
	recordkind.Kind(permission.ResourceTrailer):             "idx_trailers_code",
}

func stringsArg(args []any, want []string) bool {
	for _, arg := range args {
		if values, ok := arg.([]string); ok && slices.Equal(values, want) {
			return true
		}
	}

	return false
}

func TestNumberSources_AreKindsOfRecordTheRegistryKnows(t *testing.T) {
	t.Parallel()

	for kind, src := range numberSources {
		assert.True(t, kind.Known(), kind)
		assert.NotEmpty(t, src.numbering.Field, kind)
		assert.NotEmpty(t, src.numbering.Name, kind)
	}
}

// Every number is read through a unique index, which is what makes the read
// exact and cheap: a kind whose number has none is not listed.
func TestNumberSources_EachReadsANumberBehindAUniqueIndex(t *testing.T) {
	t.Parallel()

	kinds := make([]recordkind.Kind, 0, len(numberSources))
	for kind := range numberSources {
		kinds = append(kinds, kind)
	}
	indexed := make([]recordkind.Kind, 0, len(numberIndexes))
	for kind := range numberIndexes {
		indexed = append(indexed, kind)
	}
	slices.Sort(kinds)
	slices.Sort(indexed)
	require.Equal(t, indexed, kinds)

	entries, err := os.ReadDir(migrationsDir)
	require.NoError(t, err)
	var schema strings.Builder
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
		require.NoError(t, readErr)
		schema.Write(raw)
		schema.WriteByte('\n')
	}
	for kind, index := range numberIndexes {
		unique := regexp.MustCompile(`(?i)(UNIQUE INDEX (CONCURRENTLY )?(IF NOT EXISTS )?"?` +
			regexp.QuoteMeta(index) + `"?\s)|(CONSTRAINT "?` + regexp.QuoteMeta(index) +
			`"? UNIQUE)`)
		assert.Regexp(t, unique, schema.String(), "%s reads through %s", kind, index)
		dropped := regexp.MustCompile(`(?i)DROP (INDEX|CONSTRAINT) (CONCURRENTLY )?(IF EXISTS )?"?` +
			regexp.QuoteMeta(index) + `"?\b`)
		assert.NotRegexp(t, dropped, schema.String(), "%s is dropped", index)
	}
}

func TestNumberQueries_AreScopedToTheTenantAndMatchExactly(t *testing.T) {
	t.Parallel()

	for kind, src := range numberSources {
		asked := []string{"s-100", "s-101"}
		sql, args := keeltest.Render(t, src.query(asked))

		assert.Contains(t, args, keeltest.TestOrganizationID.String(), kind)
		assert.Contains(t, args, keeltest.TestBusinessUnitID.String(), kind)
		assert.True(t, stringsArg(args, asked), "%s: %v", kind, args)
		assert.Contains(t, sql, "LIMIT", kind)
		assert.NotContains(t, strings.ToUpper(sql), " LIKE ", kind)
		if src.foldsCase {
			assert.Contains(t, sql, "lower(", kind)
		} else {
			assert.NotContains(t, sql, "lower(", kind)
		}
	}
}

func TestNumberSources_NameFieldsReadableAtInternal(t *testing.T) {
	t.Parallel()

	for kind, src := range numberSources {
		level := fieldsensitivity.Level(
			fieldsensitivity.DefaultRegistry(), kind.Resource(), src.numbering.Field,
		)
		assert.True(t, fieldsensitivity.VisibleAt(level, permission.SensitivityInternal),
			"%s's %s is %s", kind, src.numbering.Field, level)
	}
}

func TestDistinctNumbers_FoldsOnlyWhereTheIndexDoes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"S-1", "s-1"}, distinctNumbers([]string{"S-1", " s-1", "S-1", ""}, false))
	assert.Equal(t, []string{"t-1"}, distinctNumbers([]string{"T-1", "t-1"}, true))
}

func TestMatchNumbers_KeysEachRecordByTheNumberAsked(t *testing.T) {
	t.Parallel()

	tractor := pulid.MustNew("trac_")
	rows := []numberRow{{ID: tractor, Number: "T-1"}}

	assert.Equal(t, map[string][]recordnumberports.NumberedRecord{
		"t-1": {{ID: tractor, Number: "T-1"}},
		"T-1": {{ID: tractor, Number: "T-1"}},
	}, matchNumbers([]string{"t-1", "T-1", "t-1"}, rows, true))
	assert.Empty(t, matchNumbers([]string{"t-1"}, rows, false))
}
