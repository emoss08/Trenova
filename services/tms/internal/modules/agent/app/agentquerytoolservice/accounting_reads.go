package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/typeutils"
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
	fieldMappingTargetType = "targetType"
	fieldMappingState      = "state"
	fieldRequiredOnly      = "requiredOnly"

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
	accountingStatusField     = "status"
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

// accountingReadSpecs are the accounting sync lists, each read for one
// accounting system through the operation the Desk reads it with.
func accountingReadSpecs() []graphQLListSpec {
	return []graphQLListSpec{
		driftListSpec(), inboundListSpec(), syncRecordsListSpec(), mappingGapsListSpec(),
	}
}

// driftListSpec reads the differences with the accounting system, with the
// tolerance they are judged by.
func driftListSpec() graphQLListSpec {
	return graphQLListSpec{
		name:         "accounting_list_drift_findings",
		entityPlural: "drift findings",
		summary: "List drift: documents Trenova sent to the accounting system that differ " +
			"there now, newest first. Each row has both values in minor units, who " +
			"changed it there, withinTolerance, and an open finding's fixes " +
			"(directions). Pass an Open row's id to accounting_resolve_drift_finding or " +
			"accounting_dismiss_drift_finding.",
		searchTerms: []string{
			"drift",
			"changed after it was sent",
			"edited in the accounting system",
			"reconciliation differences",
		},
		queryNote: "Words to find in the document number, the party name or who " +
			"changed it.",
		names:    accountingSystemName,
		resource: permission.ResourceAccountingSync,
		alongside: []alongsideSpec{{
			key:       keyAccountingOverview,
			operation: "DeskAccountingDriftOverview",
			what:      "drift overview",
			withheld:  overviewWithheldReason,
		}},
		combine: withinTolerance,
	}
}

// inboundListSpec reads the payments the accounting system recorded.
func inboundListSpec() graphQLListSpec {
	return graphQLListSpec{
		name:         "accounting_list_inbound_changes",
		entityPlural: "inbound changes",
		summary: "List inbound changes: payments the accounting system recorded against " +
			"invoices or bills Trenova sent, newest first. Each says what it pays and " +
			"whether Trenova applied it, and if not why (reason), with " +
			"canApply and canIgnore. Pass a Proposed row's id to " +
			"accounting_apply_inbound_change or accounting_ignore_inbound_change.",
		searchTerms: []string{"inbound changes", "payments the accounting system recorded"},
		queryNote: "Words to find in the payment's number or the customer or vendor " +
			"name.",
		names:    accountingSystemName,
		resource: permission.ResourceAccountingSync,
		shape:    inboundActions,
	}
}

// syncRecordsListSpec reads the sync ledger, or where named documents stand.
func syncRecordsListSpec() graphQLListSpec {
	return graphQLListSpec{
		name:         "accounting_list_sync_records",
		entityPlural: "sync records",
		summary: "List the documents Trenova sends to the accounting system and where " +
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
		searchTerms: []string{
			"failed", "blocked", "ledger", "not synced", "synced", "did it sync",
		},
		queryNote: "Words to find in Trenova or accounting system document numbers.",
		names: map[string]string{
			argIntegrationType: paramAccountingSystem,
			"statuses":         accountingStatusField,
			"objectTypes":      "documentType",
			"errorCategories":  fieldSyncErrorCategory,
			argObjectIDs:       paramDocumentIDs,
		},
		notes: map[string]string{
			paramAccountingSystem: "Required to list records; with documentIds, only " +
				"that system's records answer, and without it every system that syncs " +
				"does.",
			accountingStatusField: "Blocked and DeadLettered need a person; AwaitingApproval waits " +
				"to be released.",
			paramDocumentIDs: "The filters list records instead and are left out " +
				"with it.",
		},
		references: map[string]idReference{paramDocumentIDs: {
			kinds:  syncDocumentKinds,
			what:   "Trenova document ids, or the record on screen, to say whether each reached the accounting system",
			source: syncDocumentIDSources,
			most:   syncStateMaxDocuments,
		}},
		resource: permission.ResourceAccountingSync,
		shape:    syncRecordActions,
		instead: []alongsideSpec{{
			key:       keyAccountingStates,
			operation: "DeskAccountingSyncObjectStates",
			what:      "where each document stands",
			shape:     syncStateMeaning,
			note:      untrackedNote,
		}},
	}
}

// mappingGapsListSpec reads what still has to be matched before anything syncs.
func mappingGapsListSpec() graphQLListSpec {
	return graphQLListSpec{
		name:         "accounting_list_mapping_gaps",
		entityPlural: "mapping gaps",
		summary: "List what still has to be matched to the accounting system before " +
			"anything can sync. It covers account roles, line types, accessorial charges, " +
			"customers, carriers, terms and payment methods that are unmatched or only " +
			"proposed; summary says how many required mappings are confirmed and whether " +
			"setup can finish. Each gap carries its id and the candidates Trenova " +
			"considered; pass them to accounting_get_mapping or accounting_set_mapping.",
		searchTerms: []string{
			"mapping", "unmapped", "setup", "match", "gaps", "still needs mapping",
		},
		queryNote: "Words to find in the Trenova label or the chosen record's name.",
		names: map[string]string{
			argIntegrationType: paramAccountingSystem,
			"targetTypes":      fieldMappingTargetType,
			"states":           fieldMappingState,
		},
		notes: map[string]string{
			fieldMappingState: "Unmatched and Proposed unless you name others.",
			fieldRequiredOnly: "Only the mappings setup cannot finish without.",
		},
		defaults: map[string]any{fieldMappingState: []any{
			string(accountingsync.MappingStateUnmatched),
			string(accountingsync.MappingStateProposed),
		}},
		resource: permission.ResourceAccountingIntegration,
		shape:    mappingActions,
		alongside: []alongsideSpec{{
			key:       keyAccountingSummary,
			operation: "DeskAccountingMappingSummary",
			what:      "mapping summary",
			shape:     mappingSetupPath,
		}},
	}
}

// accountingGetSpecs are the accounting gets, each read through the
// operation the Desk reads it with.
func accountingGetSpecs() []getSpec {
	return []getSpec{
		{
			name:   "accounting_get_connection",
			entity: "accounting connection",
			summary: "Get whether the accounting system is connected, to which company, and " +
				"why it last failed. It also says when it last answered, when the " +
				"authorization must be renewed, whether it was connected by sign-in or with " +
				"a certificate and when that certificate expires, and whether documents are " +
				"being sent: sending is NotStarted, Paused, Automatic, or OnRelease when " +
				"each document waits for a person to release it, with the start date. queue " +
				"counts how many documents are in each sync status and groups those needing " +
				"attention by cause. Use it before explaining why something did not reach " +
				"the books. It returns status and dates, never credentials.",
			searchTerms: []string{"sync working", "sync health", "sync queue", "connected"},
			resource:    permission.ResourceAccountingIntegration,
			names:       accountingSystemName,
			shape:       connectionMeaning,
			alongside: []alongsideSpec{{
				key:       keyAccountingQueue,
				operation: "DeskAccountingSyncSummary",
				what:      "sync queue",
				withheld:  queueWithheldReason,
			}},
		},
		{
			name:   "accounting_get_sync_record",
			entity: "sync record",
			summary: "Get one accounting sync record with its status, why it last failed and " +
				"every try. It names the document it sends, gives the plain-language " +
				"resolution, and the accounting system's document number and link once " +
				"synced. Use it to explain why a document did not reach the books before " +
				"retrying or skipping it. It never returns what was sent or any credential.",
			searchTerms: []string{"attempts", "tries", "sync error"},
			resource:    permission.ResourceAccountingSync,
			kinds:       []permission.RecordKind{permission.KindAccountingSyncRecord},
			paramName:   paramSyncRecordID,
			idSource:    "from accounting_list_sync_records or the run's subject",
			shape: func(record map[string]any) {
				syncRecordActions(record, nil, clock{})
				record[accountingMeaningField] = syncMeaning(record, "the accounting system")
			},
			alongside: []alongsideSpec{{
				key:       keyAccountingAttempts,
				operation: "DeskAccountingSyncAttempts",
				what:      "tries",
				withheld:  attemptsWithheldReason,
			}},
		},
		{
			name:   "accounting_get_mapping",
			entity: "accounting mapping",
			summary: "Get which accounting system record one Trenova record or setting is " +
				"sent as. Name it by mappingId, or by targetType with recordId for a " +
				"customer, carrier or accessorial charge, or with key for an account role, " +
				"line type, term or payment method. With query, options holds up to ten " +
				"usable accounting records of the right kind whose names match, so you can " +
				"choose one for accounting_set_mapping.",
			searchTerms: []string{"account", "post", "posts", "revenue", "mapped", "item", "vendor"},
			resource:    permission.ResourceAccountingIntegration,
			kinds:       []permission.RecordKind{permission.KindAccountingMapping},
			paramName:   paramMappingID,
			idSource:    "from accounting_list_mapping_gaps",
			names: map[string]string{
				argIntegrationType: paramAccountingSystem,
				"optionsQuery":     paramQuery,
			},
			notes: map[string]string{
				argMappingKey: "Its trenovaKey from accounting_list_mapping_gaps, for the other " +
					"target types. Examples: \"ARAccount\", \"RevenueAccount\", " +
					"\"Freight\", \"Net30\", \"ACH\".",
			},
			references: map[string]idReference{argMappingRecordID: {
				kinds: []permission.RecordKind{
					permission.RecordKind(permission.ResourceCustomer),
					permission.RecordKind(permission.ResourceCarrier),
					permission.RecordKind(permission.ResourceAccessorialCharge),
				},
				what:   "The Trenova customer, carrier or accessorial charge id",
				source: "customer_list, carrier_list or masterdata_list_accessorial_charges",
			}},
			shape: func(record map[string]any) {
				if mapping, ok := record["mapping"].(map[string]any); ok {
					mappingActions(mapping, nil, clock{})
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
		within := known && accountingsync.DriftKind(kind).IsMoney()
		if difference < 0 {
			difference = -difference
		}
		item["withinTolerance"] = within && difference <= tolerance
	}
}

// inboundActions says what can be done with an inbound change, as the
// domain decides it: applied while proposed for a reason a person can act
// on, ignored while open.
func inboundActions(row map[string]any, _ map[string]struct{}, _ clock) {
	status, _ := row[accountingStatusField].(string)
	reason, _ := row["reason"].(string)
	open := accountingsync.InboundChangeStatus(status).IsOpen()
	row["canApply"] = open && status == string(accountingsync.InboundStatusProposed) &&
		accountingsync.InboundChangeReason(reason).Applicable()
	row["canIgnore"] = open
}

// syncRecordActions says what can be done with a sync record, and drops the
// next try's time unless one is still due.
func syncRecordActions(row map[string]any, _ map[string]struct{}, _ clock) {
	status := accountingsync.SyncStatus(stringOf(row[accountingStatusField]))
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
	syncRecordActions(record, nil, clock{})
	row[accountingMeaningField] = syncMeaning(record, stringOf(row[accountingProviderField]))
}

// syncMeaning is what a sync record's status means, in plain words.
func syncMeaning(record map[string]any, provider string) string {
	switch accountingsync.SyncStatus(stringOf(record[accountingStatusField])) {
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
func mappingActions(row map[string]any, _ map[string]struct{}, _ clock) {
	kind := accountingsync.ReferenceKind(stringOf(row["providerKind"]))
	row["canCreateInAccountingSystem"] = kind.Creatable() &&
		stringOf(row[fieldMappingState]) != string(accountingsync.MappingStateConfirmed)
}

// mappingSetupPath names the page where a person finishes setup.
func mappingSetupPath(summary map[string]any) {
	summary["setupPath"] = accountingsync.SetupPath(
		integration.Type(stringOf(summary["integrationType"])))
}

// connectionMeaning says what the connection's state means and whether
// documents are being sent, and replaces a failure the accounting system
// worded itself with a pointer to the connection page, as the domain words
// it for an agent.
func connectionMeaning(record map[string]any) {
	system := integration.Type(stringOf(record[argIntegrationType]))
	provider := stringOf(record[accountingProviderField])
	available, _ := record["available"].(bool)
	record["setupPath"] = accountingsync.SetupPath(system)
	conn, connected := record[accountingConnectionField].(map[string]any)
	if !connected {
		record[accountingStatusField] = "NeverConnected"
		record["sending"] = sendingNotStarted
		record[accountingMeaningField] = connectionStatusMeaning(provider, "", available)
		return
	}
	status := accountingsync.ConnectionStatus(stringOf(conn[accountingStatusField]))
	conn["sending"] = connectionSending(conn, status)
	record[accountingMeaningField] = connectionStatusMeaning(provider, status, available)
	category := accountingsync.ErrorCategory(stringOf(conn["lastErrorCategory"]))
	if category == "" {
		delete(conn, "lastErrorMessage")
	} else if category == accountingsync.ErrorCategoryUnknown {
		conn["lastErrorMessage"] = "The provider rejected the last call. Its own message " +
			"is on the connection page."
	}
}

// connectionSending is whether and how documents are sent, as the domain
// decides it from the connection's state.
func connectionSending(conn map[string]any, status accountingsync.ConnectionStatus) string {
	active := status != accountingsync.ConnectionStatusDisconnected &&
		status != accountingsync.ConnectionStatusRevoked
	syncing := active &&
		stringOf(conn["setupStep"]) == string(accountingsync.SetupStepComplete) &&
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
	status accountingsync.ConnectionStatus,
	available bool,
) string {
	switch status {
	case accountingsync.ConnectionStatusConnected:
		return provider + " is connected and answering."
	case accountingsync.ConnectionStatusDegraded:
		return "The last check failed; Trenova retries automatically."
	case accountingsync.ConnectionStatusFailing:
		return "Several checks failed in a row; nothing reaches the books until it recovers."
	case accountingsync.ConnectionStatusRevoked:
		return "The authorization was revoked or expired. A person must reconnect " +
			provider + "."
	case accountingsync.ConnectionStatusDisconnected:
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

func stringOf(value any) string {
	text, _ := value.(string)

	return text
}
