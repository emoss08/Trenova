package recordversionrepository

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rateimport"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
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

	dir := filepath.Join("..", "..", "..", "..", "core", "services", "agenttoolservice")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	target := regexp.MustCompile(
		`(?:targetOf\([^)]*|ToolTarget\{Resource:\s*)(?:permission|serviceports)\.((?:Resource|Record)\w+)`,
	)
	resources := resourceNames(t)

	missing := make(map[string]struct{})
	found := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, readErr := os.ReadFile(filepath.Join(dir, name))
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

	source, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "..", "core", "domain", "permission", "resource_gen.go",
	))
	require.NoError(t, err)

	out := make(map[string]permission.Resource)
	pattern := regexp.MustCompile(`(Resource\w+)\s+Resource = "([^"]+)"`)
	for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
		out[match[1]] = permission.Resource(match[2])
	}
	require.NotEmpty(t, out)
	out["RecordInvoiceAdjustment"] = services.RecordInvoiceAdjustment
	out["RecordCreditMemoApplication"] = services.RecordCreditMemoApplication

	return out
}

func TestEDIRecordsAToolTargetsAreToldApartByTheirIDs(t *testing.T) {
	t.Parallel()

	insert := (*bun.InsertQuery)(nil)
	transfer := new(edi.EDITransfer)
	tenderChange := new(edi.TenderChange)
	transferChange := new(edi.TransferChange)
	message := new(edi.EDIMessage)
	file := new(edi.EDIInboundFile)
	for _, record := range []bun.BeforeAppendModelHook{
		transfer, tenderChange, transferChange, message, file,
	} {
		require.NoError(t, record.BeforeAppendModel(t.Context(), insert))
	}

	kinds := lookups[permission.ResourceEDI].kinds
	for name, id := range map[string]pulid.ID{
		"transfer":        transfer.ID,
		"tender change":   tenderChange.ID,
		"transfer change": transferChange.ID,
		"message":         message.ID,
		"inbound file":    file.ID,
	} {
		entry, ok := kinds[id.Prefix()]
		require.Truef(t, ok, "the %s id %s has no version lookup", name, id)
		require.NotNil(t, entry.model, name)
		require.NotNil(t, entry.scope, name)
		require.NotEmpty(t, entry.idEq, name)
	}
}

func TestRecordsSharingAResourceAreToldApartByTheirIDs(t *testing.T) {
	t.Parallel()

	insert := (*bun.InsertQuery)(nil)
	purchase := new(fuelpurchase.FuelPurchase)
	fuelBatch := new(fuelpurchase.ImportBatch)
	card := new(fuelpurchase.FuelCard)
	iftaReturn := new(ifta.Return)
	entry := new(ifta.JurisdictionMileageEntry)
	agreement := new(rateagreement.RateAgreement)
	rateBatch := new(rateimport.RateImportBatch)
	definition := new(report.ReportDefinition)
	schedule := new(report.ReportSchedule)
	run := new(report.ReportRun)
	for _, record := range []bun.BeforeAppendModelHook{
		purchase, fuelBatch, card, iftaReturn, entry, agreement, rateBatch,
		definition, schedule, run,
	} {
		require.NoError(t, record.BeforeAppendModel(t.Context(), insert))
	}

	for name, target := range map[string]services.ToolTarget{
		"fuel purchase":          {Resource: permission.ResourceFuelPurchase, ID: purchase.ID},
		"fuel purchase import":   {Resource: permission.ResourceFuelPurchase, ID: fuelBatch.ID},
		"fuel import to resolve": {Resource: permission.ResourceFuelPurchaseImport, ID: fuelBatch.ID},
		"fuel card":              {Resource: permission.ResourceFuelCard, ID: card.ID},
		"IFTA return":            {Resource: permission.ResourceIFTAReturn, ID: iftaReturn.ID},
		"IFTA mileage entry":     {Resource: permission.ResourceIFTAJurisdictionMileage, ID: entry.ID},
		"rate agreement":         {Resource: permission.ResourceRateAgreement, ID: agreement.ID},
		"rate import":            {Resource: permission.ResourceRateAgreement, ID: rateBatch.ID},
		"report":                 {Resource: permission.ResourceReport, ID: definition.ID},
		"report schedule":        {Resource: permission.ResourceReport, ID: schedule.ID},
		"report run":             {Resource: permission.ResourceReport, ID: run.ID},
	} {
		lookup, ok := lookups[target.Resource]
		require.Truef(t, ok, "%s: %s has no version lookup", name, target.Resource)
		if lookup.kinds != nil {
			lookup, ok = lookup.kinds[target.ID.Prefix()]
			require.Truef(t, ok, "the %s id %s has no version lookup", name, target.ID)
		}
		require.NotNil(t, lookup.model, name)
		require.NotNil(t, lookup.scope, name)
		require.NotEmpty(t, lookup.idEq, name)
		require.NotNil(t, lookup.version, name)
	}
}
