package accountingquerytools

import (
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingdrift"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountinginbound"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/typeutils"
)

// The fields the accounting lists' bespoke searches match.
const (
	fieldSyncObjectNumber      = "objectNumber"
	fieldSyncExternalDocNumber = "externalDocNumber"
	fieldMappingTargetLabel    = "targetLabel"
)

const (
	paramAccountingSystem = "system"
	paramSyncRecordID     = "syncRecordId"
	paramMappingID        = "mappingId"
	paramDocumentIDs      = "documentIds"

	argIntegrationType = "integrationType"
	argObjectIDs       = "objectIds"
	argMappingKey      = "key"
	argMappingRecordID = "recordId"

	fieldSyncErrorCategory = "errorCategory"
	fieldMappingState      = "state"
	fieldRequiredOnly      = "requiredOnly"
	fieldMapping           = "mapping"

	syncStateMaxDocuments = 50
	syncDocumentIDSources = "invoice_list, invoice_get, receivable_list_customer_payments, " +
		"settlement_list_carrier or settlement_list_driver"

	keyAccountingQueue    = "queue"
	keyAccountingOverview = "overview"
	keyAccountingSummary  = "summary"
	keyAccountingAttempts = "attempts"
	keyAccountingStates   = "states"

	accountingConnectionField = "connection"
	accountingRecordField     = "record"
	accountingMeaningField    = "whatThisMeans"
	accountingProviderField   = "providerName"

	sendingNotStarted = "NotStarted"
	sendingPaused     = "Paused"
	sendingAutomatic  = "Automatic"
	sendingOnRelease  = "OnRelease"

	queueWithheldReason = "The queue is not shown: it needs permission to read the " +
		"accounting sync ledger."
	attemptsWithheldReason = "The tries are not shown: they need permission to read the " +
		"accounting sync ledger."
	overviewWithheldReason = "The reconciliation tolerance is not shown, so withinTolerance " +
		"is left out."
	untrackedNote = "A document missing from states was never sent: sending has not started, " +
		"it is dated before the start date, or it is not a document that syncs."
)

// syncDocumentKinds are the records a sync record can stand for: the invoices,
// payments, applications and settlements sent as documents, the customers,
// carriers and workers sent as the accounts they post to, and the journal
// entries posted one by one. A journal summary is keyed by its day, not by a
// record id, so it is not among them.
var syncDocumentKinds = []permission.RecordKind{
	permission.RecordKind(permission.ResourceInvoice),
	permission.RecordKind(permission.ResourceCustomerPayment),
	permission.KindCreditMemoApplication,
	permission.RecordKind(permission.ResourceCarrierSettlement),
	permission.RecordKind(permission.ResourceDriverSettlement),
	permission.RecordKind(permission.ResourceCustomer),
	permission.RecordKind(permission.ResourceCarrier),
	permission.RecordKind(permission.ResourceWorker),
	permission.RecordKind(permission.ResourceJournalEntry),
}

// accountingSystemName is what every accounting read and write calls the
// accounting system its call is about.
var accountingSystemName = map[string]string{argIntegrationType: paramAccountingSystem}

// ReadSpecs are the accounting sync lists, each read for one
// accounting system through the operation the Desk reads it with.
func ReadSpecs() []readspec.List {
	return []readspec.List{
		driftListSpec(), inboundListSpec(), syncRecordsListSpec(), mappingGapsListSpec(),
	}
}

// driftListSpec reads the differences with the accounting system, with the
// tolerance they are judged by.
func driftListSpec() readspec.List {
	return readspec.List{
		Name:         "accounting_list_drift_findings",
		EntityPlural: "drift findings",
		Summary: "List drift: documents Trenova sent to the accounting system that differ " +
			"there now, newest first. Each row has both values in minor units, who " +
			"changed it there, withinTolerance, and an open finding's fixes " +
			"(directions). Pass an Open row's id to accounting_resolve_drift_finding or " +
			"accounting_dismiss_drift_finding.",
		SearchTerms: []string{
			"drift",
			"changed after it was sent",
			"edited in the accounting system",
			"reconciliation differences",
		},
		QueryNote: "Words to find in the document number, the party name or who " +
			"changed it.",
		Names:    accountingSystemName,
		Resource: permission.ResourceAccountingSync,
		Alongside: []readspec.Alongside{{
			Key:       keyAccountingOverview,
			Operation: "DeskAccountingDriftOverview",
			What:      "drift overview",
			Withheld:  overviewWithheldReason,
		}},
		Combine: withinTolerance,
	}
}

// inboundListSpec reads the payments the accounting system recorded.
func inboundListSpec() readspec.List {
	return readspec.List{
		Name:         "accounting_list_inbound_changes",
		EntityPlural: "inbound changes",
		Summary: "List inbound changes: payments the accounting system recorded against " +
			"invoices or bills Trenova sent, newest first. Each says what it pays and " +
			"whether Trenova applied it, and if not why (reason), with " +
			"canApply and canIgnore. Pass a Proposed row's id to " +
			"accounting_apply_inbound_change or accounting_ignore_inbound_change.",
		SearchTerms: []string{"inbound changes", "payments the accounting system recorded"},
		QueryNote: "Words to find in the payment's number or the customer or vendor " +
			"name.",
		Names:    accountingSystemName,
		Resource: permission.ResourceAccountingSync,
		Shape:    inboundActions,
	}
}

// syncRecordsListSpec reads the sync ledger, or where named documents stand.
func syncRecordsListSpec() readspec.List {
	return readspec.List{
		Name:         "accounting_list_sync_records",
		EntityPlural: "sync records",
		Searches:     []string{fieldSyncObjectNumber, fieldSyncExternalDocNumber},
		Summary: "List the documents Trenova sends to the accounting system and where " +
			"each stands, or say whether particular Trenova documents or parties reached " +
			"it. List one system's records, newest first, by status (Blocked and " +
			"DeadLettered did not reach the books), document type or error category. " +
			"Or pass documentIds of invoices, memos, payments, customers, settlements, " +
			"carriers or drivers: states holds each one's current record and what that " +
			"means, across every syncing system unless you name one; a settlement shows " +
			"its bill or bill payment, whichever needs attention, else the latest, and a " +
			"document with no state was never sent. A record has its status, error " +
			"category, resolution, tries and the accounting system's number once synced; " +
			"pass its id to accounting_get_sync_record for the tries or " +
			"accounting_retry_sync_record once the cause is fixed.",
		SearchTerms: []string{
			"failed", "blocked", "ledger", "not synced", "synced", "did it sync",
		},
		QueryNote: "Words to find in Trenova or accounting system document numbers.",
		Names: map[string]string{
			argIntegrationType:    paramAccountingSystem,
			querytool.ArgStatuses: querytool.AccountingStatusField,
			"objectTypes":         "documentType",
			"errorCategories":     fieldSyncErrorCategory,
			argObjectIDs:          paramDocumentIDs,
		},
		Notes: map[string]string{
			paramAccountingSystem: "Required to list records; with documentIds, only " +
				"that system's records answer, and without it every system that syncs " +
				"does.",
			querytool.AccountingStatusField: "Blocked and DeadLettered need a person; " +
				"AwaitingApproval waits to be released.",
			paramDocumentIDs: "The filters list records instead and are left out " +
				"with it.",
		},
		References: map[string]readspec.Reference{paramDocumentIDs: {
			Kinds: syncDocumentKinds,
			What: "Trenova document ids, or the record on screen, to say whether each " +
				"reached the accounting system",
			Source: syncDocumentIDSources,
			Most:   syncStateMaxDocuments,
		}},
		Resource: permission.ResourceAccountingSync,
		Shape:    syncRecordActions,
		Instead: []readspec.Alongside{{
			Key:       keyAccountingStates,
			Operation: "DeskAccountingSyncObjectStates",
			What:      "where each document stands",
			Shape:     syncStateMeaning,
			Note:      untrackedNote,
		}},
	}
}

// mappingGapsListSpec reads what still has to be matched before anything syncs.
func mappingGapsListSpec() readspec.List {
	return readspec.List{
		Name:         "accounting_list_mapping_gaps",
		EntityPlural: "mapping gaps",
		Searches:     []string{fieldMappingTargetLabel},
		Summary: "List what still has to be matched to the accounting system before " +
			"anything can sync. It covers account roles, line types, accessorial charges, " +
			"customers, carriers, terms and payment methods that are unmatched or only " +
			"proposed; summary says how many required mappings are confirmed and whether " +
			"setup can finish. Each gap carries its id and the candidates Trenova " +
			"considered; pass them to accounting_get_mapping or accounting_set_mapping.",
		SearchTerms: []string{
			fieldMapping, "unmapped", "setup", "match", "gaps", "still needs mapping",
		},
		QueryNote: "Words to find in the Trenova label or the chosen record's name.",
		Names: map[string]string{
			argIntegrationType: paramAccountingSystem,
			"targetTypes":      querytool.FieldTargetType,
			"states":           fieldMappingState,
		},
		Notes: map[string]string{
			fieldMappingState: "Unmatched and Proposed unless you name others.",
			fieldRequiredOnly: "Only the mappings setup cannot finish without.",
		},
		Defaults: map[string]any{fieldMappingState: []any{
			string(accountingmapping.MappingStateUnmatched),
			string(accountingmapping.MappingStateProposed),
		}},
		Resource: permission.ResourceAccountingIntegration,
		Shape:    mappingActions,
		Alongside: []readspec.Alongside{{
			Key:       keyAccountingSummary,
			Operation: "DeskAccountingMappingSummary",
			What:      "mapping summary",
			Shape:     mappingSetupPath,
		}},
	}
}

// GetSpecs are the accounting gets, each read through the
// operation the Desk reads it with.
func GetSpecs() []readspec.Get {
	return []readspec.Get{
		{
			Name:   "accounting_get_connection",
			Entity: "accounting connection",
			Summary: "Get whether the accounting system is connected, to which company, and " +
				"why it last failed. It also says when it last answered, when the " +
				"authorization must be renewed, whether it was connected by sign-in or with " +
				"a certificate and when that certificate expires, and whether documents are " +
				"being sent: sending is NotStarted, Paused, Automatic, or OnRelease when " +
				"each document waits for a person to release it, with the start date. queue " +
				"counts how many documents are in each sync status and groups those needing " +
				"attention by cause. Use it before explaining why something did not reach " +
				"the books. It returns status and dates, never credentials.",
			SearchTerms: []string{"sync working", "sync health", "sync queue", "connected"},
			Resource:    permission.ResourceAccountingIntegration,
			Names:       accountingSystemName,
			Shape:       connectionMeaning,
			Alongside: []readspec.Alongside{{
				Key:       keyAccountingQueue,
				Operation: "DeskAccountingSyncSummary",
				What:      "sync queue",
				Withheld:  queueWithheldReason,
			}},
		},
		{
			Name:   "accounting_get_sync_record",
			Entity: "sync record",
			Summary: "Get one accounting sync record with its status, why it last failed and " +
				"every try. It names the document it sends, gives the plain-language " +
				"resolution, and the accounting system's document number and link once " +
				"synced. Use it to explain why a document did not reach the books before " +
				"retrying or skipping it. It never returns what was sent or any credential.",
			SearchTerms: []string{"attempts", "tries", "sync error"},
			Resource:    permission.ResourceAccountingSync,
			Kinds:       []permission.RecordKind{permission.KindAccountingSyncRecord},
			ParamName:   paramSyncRecordID,
			IDSource:    "from accounting_list_sync_records or the run's subject",
			Shape: func(record map[string]any, withheld map[string]struct{}, clk filtercatalog.Clock) {
				syncRecordActions(record, withheld, clk)
				record[accountingMeaningField] = syncMeaning(record, "the accounting system")
			},
			Alongside: []readspec.Alongside{{
				Key:       keyAccountingAttempts,
				Operation: "DeskAccountingSyncAttempts",
				What:      "tries",
				Withheld:  attemptsWithheldReason,
			}},
		},
		{
			Name:   "accounting_get_mapping",
			Entity: "accounting mapping",
			Summary: "Get which accounting system record one Trenova record or setting is " +
				"sent as. Name it by mappingId, or by targetType with recordId for a " +
				"customer, carrier or accessorial charge, or with key for an account role, " +
				"line type, term or payment method. With query, options holds up to ten " +
				"usable accounting records of the right kind whose names match, so you can " +
				"choose one for accounting_set_mapping.",
			SearchTerms: []string{
				"account", "post", "posts", "revenue", "mapped", "item", "vendor",
			},
			Resource:  permission.ResourceAccountingIntegration,
			Kinds:     []permission.RecordKind{permission.KindAccountingMapping},
			ParamName: paramMappingID,
			IDSource:  "from accounting_list_mapping_gaps",
			Names: map[string]string{
				argIntegrationType: paramAccountingSystem,
				"optionsQuery":     querytool.ParamQuery,
			},
			Notes: map[string]string{
				argMappingKey: "Its trenovaKey from accounting_list_mapping_gaps, for the other " +
					"target types. Examples: \"ARAccount\", \"RevenueAccount\", " +
					"\"Freight\", \"Net30\", \"ACH\".",
			},
			References: map[string]readspec.Reference{argMappingRecordID: {
				Kinds: []permission.RecordKind{
					permission.RecordKind(permission.ResourceCustomer),
					permission.RecordKind(permission.ResourceCarrier),
					permission.RecordKind(permission.ResourceAccessorialCharge),
				},
				What:   "The Trenova customer, carrier or accessorial charge id",
				Source: "customer_list, carrier_list or masterdata_list_accessorial_charges",
			}},
			Shape: func(record map[string]any, withheld map[string]struct{}, clk filtercatalog.Clock) {
				if mapping, ok := record[fieldMapping].(map[string]any); ok {
					mappingActions(mapping, withheld, clk)
				}
			},
		},
	}
}

// withinTolerance marks each money difference with whether it is within the
// reconciliation tolerance the overview read alongside carries, as an agent
// may dismiss only those.
func withinTolerance(items []map[string]any, beside map[string]any) {
	overview, ok := beside[keyAccountingOverview].(map[string]any)
	if !ok {
		return
	}
	tolerance, ok := typeutils.IntOf(overview["toleranceMinor"])
	if !ok {
		return
	}
	for _, item := range items {
		kind, _ := item["kind"].(string)
		difference, known := typeutils.IntOf(item["differenceMinor"])
		within := known && accountingdrift.DriftKind(kind).IsMoney()
		if difference < 0 {
			difference = -difference
		}
		item["withinTolerance"] = within && difference <= tolerance
	}
}

// inboundActions says what can be done with an inbound change, as the
// domain decides it: applied while proposed for a reason a person can act
// on, ignored while open.
func inboundActions(row map[string]any, _ map[string]struct{}, _ filtercatalog.Clock) {
	status, _ := row[querytool.AccountingStatusField].(string)
	reason, _ := row["reason"].(string)
	open := accountinginbound.InboundChangeStatus(status).IsOpen()
	row["canApply"] = open && status == string(accountinginbound.InboundStatusProposed) &&
		accountinginbound.InboundChangeReason(reason).Applicable()
	row["canIgnore"] = open
}

// syncRecordActions says what can be done with a sync record, and drops the
// next try's time unless one is still due.
func syncRecordActions(row map[string]any, _ map[string]struct{}, _ filtercatalog.Clock) {
	status := accountingsync.SyncStatus(querytool.StringOf(row[querytool.AccountingStatusField]))
	row["canRetry"] = status.Retryable()
	row["canSkip"] = !status.IsFinal() && status != accountingsync.SyncStatusInFlight
	if !status.Dispatchable() {
		delete(row, "nextAttemptAt")
	}
}

// syncStateMeaning shapes one document's state: its record's actions, and
// what its status means in that accounting system.
func syncStateMeaning(row map[string]any) {
	record, ok := row[accountingRecordField].(map[string]any)
	if !ok {
		return
	}
	syncRecordActions(record, nil, filtercatalog.Clock{})
	row[accountingMeaningField] = syncMeaning(
		record,
		querytool.StringOf(row[accountingProviderField]),
	)
}

// syncMeaning is what a sync record's status means, in plain words.
func syncMeaning(record map[string]any, provider string) string {
	switch accountingsync.SyncStatus(querytool.StringOf(record[querytool.AccountingStatusField])) {
	case accountingsync.SyncStatusSynced:
		return "It is in " + provider + "."
	case accountingsync.SyncStatusQueued, accountingsync.SyncStatusInFlight:
		return "It is on its way to " + provider + "."
	case accountingsync.SyncStatusRetrying:
		return "The last try failed for a reason that usually passes; Trenova tries again " +
			"on its own."
	case accountingsync.SyncStatusAwaitingApproval:
		return "Sending is not automatic, so it waits for a person to release it."
	case accountingsync.SyncStatusBlocked:
		return provider + " refused it and it will not be tried again until the cause is " +
			"fixed and it is retried. The resolution says what to fix."
	case accountingsync.SyncStatusDeadLettered:
		return "Trenova gave up after repeated temporary failures. Retry it once " +
			provider + " answers again."
	case accountingsync.SyncStatusSkipped:
		return "A person chose not to send it to " + provider + "."
	case accountingsync.SyncStatusSuperseded:
		return "A later version of the document replaced this record."
	default:
		return "Its state is not known."
	}
}

// mappingActions says whether a mapping's record can be created in the
// accounting system rather than chosen.
func mappingActions(row map[string]any, _ map[string]struct{}, _ filtercatalog.Clock) {
	kind := accountingprovider.ReferenceKind(querytool.StringOf(row["providerKind"]))
	row["canCreateInAccountingSystem"] = kind.Creatable() &&
		querytool.StringOf(
			row[fieldMappingState],
		) != string(
			accountingmapping.MappingStateConfirmed,
		)
}

// mappingSetupPath names the page where a person finishes setup.
func mappingSetupPath(summary map[string]any) {
	summary["setupPath"] = accountingprovider.SetupPath(
		integration.Type(querytool.StringOf(summary["integrationType"])))
}

// connectionMeaning says what the connection's state means and whether
// documents are being sent, and replaces a failure the accounting system
// worded itself with a pointer to the connection page, as the domain words
// it for an agent.
func connectionMeaning(record map[string]any, _ map[string]struct{}, _ filtercatalog.Clock) {
	system := integration.Type(querytool.StringOf(record[argIntegrationType]))
	provider := querytool.StringOf(record[accountingProviderField])
	available, _ := record["available"].(bool)
	record["setupPath"] = accountingprovider.SetupPath(system)
	conn, connected := record[accountingConnectionField].(map[string]any)
	if !connected {
		record[querytool.AccountingStatusField] = "NeverConnected"
		record["sending"] = sendingNotStarted
		record[accountingMeaningField] = connectionStatusMeaning(provider, "", available)
		return
	}
	status := accountingconnection.ConnectionStatus(
		querytool.StringOf(conn[querytool.AccountingStatusField]),
	)
	conn["sending"] = connectionSending(conn, status)
	record[accountingMeaningField] = connectionStatusMeaning(provider, status, available)
	category := accountingsync.ErrorCategory(querytool.StringOf(conn["lastErrorCategory"]))
	if category == "" {
		delete(conn, "lastErrorMessage")
		return
	}
	if category == accountingsync.ErrorCategoryUnknown {
		conn["lastErrorMessage"] = "The provider rejected the last call. Its own message " +
			"is on the connection page."
	}
}

// connectionSending is whether and how documents are sent, as the domain
// decides it from the connection's state.
func connectionSending(conn map[string]any, status accountingconnection.ConnectionStatus) string {
	active := status != accountingconnection.ConnectionStatusDisconnected &&
		status != accountingconnection.ConnectionStatusRevoked
	syncing := active &&
		querytool.StringOf(conn["setupStep"]) == string(accountingconnection.SetupStepComplete) &&
		conn["syncStartDate"] != nil && conn["syncEnabledAt"] != nil
	automatic, _ := conn["autoSync"].(bool)
	switch {
	case !syncing:
		return sendingNotStarted
	case conn["pausedAt"] != nil:
		return sendingPaused
	case automatic:
		return sendingAutomatic
	default:
		return sendingOnRelease
	}
}

func connectionStatusMeaning(
	provider string,
	status accountingconnection.ConnectionStatus,
	available bool,
) string {
	switch status {
	case accountingconnection.ConnectionStatusConnected:
		return provider + " is connected and answering."
	case accountingconnection.ConnectionStatusDegraded:
		return "The last check failed; Trenova retries automatically."
	case accountingconnection.ConnectionStatusFailing:
		return "Several checks failed in a row; nothing reaches the books until it recovers."
	case accountingconnection.ConnectionStatusRevoked:
		return "The authorization was revoked or expired. A person must reconnect " +
			provider + "."
	case accountingconnection.ConnectionStatusDisconnected:
		return "Someone disconnected " + provider + " on purpose."
	default:
		if !available {
			return provider + " is not set up on this Trenova instance; an administrator " +
				"must add the app credentials."
		}
		return provider + " has never been connected. A person connects it from the " +
			"integrations page."
	}
}
