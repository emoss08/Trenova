//go:build integration

package schemalint

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// domainDir is resolved from this file rather than the working directory:
// SetupTestDB chdirs to find the migrations, so a relative path would break
// depending on when it is read.
func domainDir(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "cannot locate the schemalint source directory")

	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "core", "domain")
}

// knownUnfixed is the backlog this lint ratchets down.
//
// Every entry is a boolean column that defaults to TRUE in the schema, backed
// by a Go field that declares a bun default — so false cannot be stored when
// the row is created. Removing a tag is only safe once the code that creates
// the row sets the value itself, which is why these are listed rather than
// fixed: each is a tenant control provisioned from a struct that leaves the
// field alone and leans on the substitution to come out true. Removing the tag
// without naming the value would switch the flag off for every new tenant.
//
// Fix one by naming the value at the site that builds the row, then deleting
// the line here. Nothing may be added.
var knownUnfixed = map[string]struct{}{
	"accounting_controls.notify_on_reconciliation_exception": {},
	"accounting_controls.require_manual_je_approval":         {},
	"accounting_controls.require_period_close_approval":      {},
	"agent_controls.shadow_mode":                             {},
	"billing_controls.notify_on_billing_exceptions":          {},
	"billing_controls.require_rate_override_reason":          {},
	"billing_controls.show_balance_due_on_invoice":           {},
	"billing_controls.show_due_date_on_invoice":              {},
	"costing_controls.include_deadhead_miles":                {},
	"costing_controls.use_live_fuel_price":                   {},
	"dash_controls.allow_contact_info_edit":                  {},
	"dash_controls.allow_expense_submission":                 {},
	"dash_controls.allow_load_comments":                      {},
	"dash_controls.allow_load_document_upload":               {},
	"dash_controls.allow_load_refusals":                      {},
	"dash_controls.allow_profile_document_upload":            {},
	"dash_controls.allow_pto_requests":                       {},
	"dash_controls.allow_settlement_disputes":                {},
	"dash_controls.allow_stop_actions":                       {},
	"dash_controls.enable_detention_alerts":                  {},
	"dash_controls.require_load_acknowledgment":              {},
	"dash_controls.send_credential_reminders":                {},
	"dash_controls.show_load_pay":                            {},
	"dash_controls.show_pay_estimates":                       {},
	"document_controls.enable_auto_classification":           {},
	"document_controls.enable_auto_create_document_types":    {},
	"document_controls.enable_auto_document_type_associate":  {},
	"document_controls.enable_document_intelligence":         {},
	"document_controls.enable_full_text_indexing":            {},
	"document_controls.enable_ocr":                           {},
	"document_controls.enable_shipment_draft_extraction":     {},
	"organizations.asset_operations_enabled":                 {},
	"organizations.brokerage_enabled":                        {},
	"settlement_controls.allow_negative_net":                 {},
	"settlement_controls.auto_attach_accruals":               {},
	"shipment_controls.check_for_duplicate_bols":             {},
	"shipment_controls.check_hazmat_segregation":             {},
}

// TestBooleanDefaultsAreNotSubstituted fails when a boolean column that
// defaults to TRUE is backed by a Go field declaring a bun default. See the
// package comment for why that combination silently discards false.
func TestBooleanDefaultsAreNotSubstituted(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	fields, err := BoolFieldsWithDefaultTag(domainDir(t))
	require.NoError(t, err)
	require.NotEmpty(t, fields, "found no bun tags to check - has the domain moved?")

	trueByDefault := columnsDefaultingTrue(t, ctx, db)
	require.NotEmpty(t, trueByDefault, "found no boolean columns - did migrations run?")

	offenders := make([]string, 0)
	stale := make(map[string]struct{}, len(knownUnfixed))
	for key := range knownUnfixed {
		stale[key] = struct{}{}
	}

	for i := range fields {
		field := &fields[i]
		if _, defaultsTrue := trueByDefault[field.Key()]; !defaultsTrue {
			continue
		}
		delete(stale, field.Key())
		if _, allowed := knownUnfixed[field.Key()]; allowed {
			continue
		}
		offenders = append(offenders, fmt.Sprintf("  %s  declared at %s", field.Key(), field.Location()))
	}

	sort.Strings(offenders)
	require.Empty(t, offenders, "these boolean columns default to TRUE and carry a bun default: tag, "+
		"so false is discarded when the row is created. Drop `default:` from the tag and set the "+
		"value where the row is built:\n%s", joinLines(offenders))

	// Keep the backlog honest: an entry whose tag is gone must not linger.
	left := make([]string, 0, len(stale))
	for key := range stale {
		left = append(left, "  "+key)
	}
	sort.Strings(left)
	require.Empty(t, left, "these are listed in knownUnfixed but no longer need to be - "+
		"delete them from the list:\n%s", joinLines(left))
}

func joinLines(lines []string) string {
	out := ""
	for _, line := range lines {
		out += line + "\n"
	}

	return out
}

// columnsDefaultingTrue returns the "table.column" of every boolean column in
// the migrated schema whose column default is TRUE.
func columnsDefaultingTrue(t *testing.T, ctx context.Context, db *bun.DB) map[string]struct{} {
	t.Helper()

	var rows []struct {
		Table  string `bun:"table_name"`
		Column string `bun:"column_name"`
	}
	require.NoError(t, db.NewSelect().
		TableExpr("information_schema.columns").
		Column("table_name", "column_name").
		Where("table_schema = ?", "public").
		Where("data_type = ?", "boolean").
		Where("lower(column_default) LIKE ?", "true%").
		Scan(ctx, &rows))

	out := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		out[row.Table+"."+row.Column] = struct{}{}
	}

	return out
}
