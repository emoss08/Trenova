package archiveplan

import "strings"

const rollupReason = "a rollup, rebuilt for the organization by the rollup refresh from the marks its source tables' triggers write as they load"

var derived = map[string]string{
	"invoice_search_projections":       "written by trg_invoices_search_projection as the invoices load",
	"document_storage_totals":          "written by the documents triggers as the documents load",
	"customer_ledger_invoice_balances": "written by the customer ledger triggers as the entries load",
	"customer_ledger_monthly_balances": "written by the customer ledger triggers as the entries load",
}

var notCarried = map[string]string{
	"instance_bootstraps":        "the install's own first-run record; a dedicated import writes the destination's",
	"support_log_requests":       "the install's log bundle requests, which hold every tenant's lines",
	"support_access_grants":      "Trenova support access, granted against the source install's support keys",
	"work_queue_items":           "in-flight background work; each kind's sweep finds its work again from the records",
	"password_reset_tokens":      "single-use sign-in and invitation links; people ask for new ones",
	"capture_pairings":           "scanner pairing codes in progress, which expire within minutes",
	"document_upload_sessions":   "uploads in progress",
	"ai_analytics_refresh_state": "a refresh cursor; the refresh starts over for the organization",
	"watchtower_cursors":         "projector cursors; the projector catches up from the records",
}

var twoParty = map[string][2]string{
	"edi_load_tender_transfers": {"source_organization_id", "target_organization_id"},
}

var outsideTenancy = map[string]string{
	"cloud_signups": "the retired Cloud sign-up ledger names the organization it provisioned; it is the provider's bookkeeping, not the tenant's data",
}

var globalKeys = map[string][]string{
	"us_states":          {"abbreviation"},
	"ifta_jurisdictions": {"country_code", "code"},
	"edi_document_types": {IDColumn},
	"pretrained_models":  {IDColumn},
}

var exportTables = map[string]string{
	"mfa_recovery_codes": "second-factor recovery codes",
	"mfa_authenticators": "second-factor secrets",
	"scim_tokens":        "directory provisioning tokens",
}

func secretReason(column string) string {
	switch {
	case strings.Contains(column, "password") || strings.Contains(column, "passphrase"):
		return "password hashes"
	case strings.HasSuffix(column, "_hash"):
		return "token and key hashes"
	case strings.Contains(column, "token"):
		return "tokens"
	default:
		return "integration credentials and secrets"
	}
}

func secretColumn(name string) bool {
	if strings.Contains(name, "password") || strings.Contains(name, "passphrase") ||
		strings.Contains(name, "cipher") || strings.Contains(name, "secret") {
		return true
	}
	for _, suffix := range []string{
		"_token", "token_hash", "code_hash", "key_hash", "_private_key", "signing_key",
		"access_key", "encryption_key", "sealed_access", "recovery_code",
	} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return name == "token" || name == "api_key" || name == "private_key"
}

func Derived(table string, rollups []string) (string, bool) {
	if reason, ok := derived[table]; ok {
		return reason, true
	}
	for _, r := range rollups {
		if r == table {
			return rollupReason, true
		}
	}
	return "", false
}

func NotCarried(table string) (string, bool) {
	reason, ok := notCarried[table]
	return reason, ok
}

func GlobalKey(table string) ([]string, bool) {
	key, ok := globalKeys[table]
	return key, ok
}

func TwoParty(table string) ([2]string, bool) {
	cols, ok := twoParty[table]
	return cols, ok
}

func OutsideTenancy(table string) (string, bool) {
	reason, ok := outsideTenancy[table]
	return reason, ok
}
