package recordversionrepository

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement/rateagreementdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateimport/rateimportdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase/fuelpurchasedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta/iftadb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report/reportdb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A tool names the record it changes so a proposal can be pinned to that record's
version, and the pin is only as good as this table: a resource missing here is
reported as unsupported and the proposal runs unpinned, so a person approving a
change to a record somebody else has since edited gets no warning at all. Six
resources were targeted with no lookup before this test existed.

The targets are read out of the tool package's source because a tool's Target
method is only reachable by constructing the tool, and the point is to catch
the one nobody remembered to list anywhere.
*/
func TestEveryResourceAToolTargetsHasAVersionLookup(t *testing.T) {
	t.Parallel()

	files := toolSourceFiles(t,
		filepath.Join("..", "..", "..", "..", "agent", "app", "agenttoolservice"),
		filepath.Join("..", "..", "..", "..", "agent", "app", "agenttools"),
	)

	target := regexp.MustCompile(
		`(?:(?:common\.)?[Tt]argetOf\([^)]*|ToolTarget\{Resource:\s*)(?:permission|\w+ports)\.((?:Resource|Record)\w+)`,
	)
	resources := resourceNames(t)

	missing := make(map[string]struct{})
	found := 0
	for _, path := range files {
		name := filepath.Base(path)
		source, readErr := os.ReadFile(path)
		require.NoError(t, readErr)

		for _, match := range target.FindAllStringSubmatch(string(source), -1) {
			found++
			resource, ok := resources[match[1]]
			require.Truef(t, ok, "%s names %s, which is not a resource", name, match[1])
			if _, ok = lookups[resource]; !ok {
				missing[match[1]] = struct{}{}
			}
		}
	}
	require.NotZero(t, found, "no tool targets were found; the scan is reading the wrong place")

	names := make([]string, 0, len(missing))
	for name := range missing {
		names = append(names, name)
	}
	sort.Strings(names)
	require.Emptyf(t, names,
		"tools target these resources and nothing reads their version, so every "+
			"proposal against them runs unpinned: %v", names,
	)
}

func resourceNames(t *testing.T) map[string]permission.Resource {
	t.Helper()

	source, err := os.ReadFile(
		filepath.Join(
			"..",
			"..",
			"..",
			"..",
			"identity",
			"domain",
			"permission",
			"resource_gen.go",
		),
	)
	require.NoError(t, err)

	out := make(map[string]permission.Resource)
	pattern := regexp.MustCompile(`(Resource\w+)\s+Resource = "([^"]+)"`)
	for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
		out[match[1]] = permission.Resource(match[2])
	}
	require.NotEmpty(t, out)
	out["RecordInvoiceAdjustment"] = agentports.RecordInvoiceAdjustment
	out["RecordCreditMemoApplication"] = agentports.RecordCreditMemoApplication

	return out
}

func newID(t *testing.T, table *schema.Table) pulid.ID {
	t.Helper()

	require.NotEmpty(t, table.IDPrefix, table.Name)
	return pulid.MustNew(table.IDPrefix)
}

func TestEDIRecordsAToolTargetsAreToldApartByTheirIDs(t *testing.T) {
	t.Parallel()

	kinds := lookups[permission.ResourceEDI].kinds
	for name, id := range map[string]pulid.ID{
		"transfer":        newID(t, edidb.EDITransfersSchema()),
		"tender change":   newID(t, edidb.TenderChangesSchema()),
		"transfer change": newID(t, edidb.TransferChangesSchema()),
		"message":         newID(t, edidb.EDIMessagesSchema()),
		"inbound file":    newID(t, edidb.EDIInboundFilesSchema()),
	} {
		entry, ok := kinds[id.Prefix()]
		require.Truef(t, ok, "the %s id %s has no version lookup", name, id)
		require.NotNil(t, entry.read, name)
	}
}

func TestRecordsSharingAResourceAreToldApartByTheirIDs(t *testing.T) {
	t.Parallel()

	fuelBatch := newID(t, fuelpurchasedb.ImportBatchesSchema())
	for name, target := range map[string]agentports.ToolTarget{
		"fuel purchase": {
			Resource: permission.ResourceFuelPurchase,
			ID:       newID(t, fuelpurchasedb.FuelPurchasesSchema()),
		},
		"fuel purchase import":   {Resource: permission.ResourceFuelPurchase, ID: fuelBatch},
		"fuel import to resolve": {Resource: permission.ResourceFuelPurchaseImport, ID: fuelBatch},
		"fuel card": {
			Resource: permission.ResourceFuelCard,
			ID:       newID(t, fuelpurchasedb.FuelCardsSchema()),
		},
		"IFTA return": {
			Resource: permission.ResourceIFTAReturn,
			ID:       newID(t, iftadb.ReturnsSchema()),
		},
		"IFTA mileage entry": {
			Resource: permission.ResourceIFTAJurisdictionMileage,
			ID:       newID(t, iftadb.JurisdictionMileageEntriesSchema()),
		},
		"rate agreement": {
			Resource: permission.ResourceRateAgreement,
			ID:       newID(t, rateagreementdb.RateAgreementsSchema()),
		},
		"rate import": {
			Resource: permission.ResourceRateAgreement,
			ID:       newID(t, rateimportdb.RateImportBatchesSchema()),
		},
		"report": {
			Resource: permission.ResourceReport,
			ID:       newID(t, reportdb.ReportDefinitionsSchema()),
		},
		"report schedule": {
			Resource: permission.ResourceReport,
			ID:       newID(t, reportdb.ReportSchedulesSchema()),
		},
		"report run": {
			Resource: permission.ResourceReport,
			ID:       newID(t, reportdb.ReportRunsSchema()),
		},
	} {
		lookup, ok := lookups[target.Resource]
		require.Truef(t, ok, "%s: %s has no version lookup", name, target.Resource)
		if lookup.kinds != nil {
			lookup, ok = lookup.kinds[target.ID.Prefix()]
			require.Truef(t, ok, "the %s id %s has no version lookup", name, target.ID)
		}
		require.NotNil(t, lookup.read, name)
	}
}

func TestVersionLookup_ReadsTheVersionByID(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, lookups[permission.ResourceOrder].read(pulid.ID("ord_1")))

	assert.Contains(t, sql, `."version" FROM "orders"`)
	assert.Contains(t, sql, `."id" = $`)
	assert.Contains(t, args, "ord_1")
}

func toolSourceFiles(t *testing.T, roots ...string) []string {
	t.Helper()

	files := make([]string, 0, 512)
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			files = append(files, path)
			return nil
		})
		require.NoError(t, err)
	}

	return files
}
