# Agent read tools and their GraphQL operations: the starting inventory

The survey every module's conversion starts from, made on 2026-10-20 against `master` for
[agent-graphql-reads.md](agent-graphql-reads.md). Once a module converts, its rows are generated
by `agentread/gen` and this page is no longer their source: the generated bindings and
`agentread.NoGraphQL` are. A converted module's rows are removed from this page in the same pull
request.

Each of the 176 agent read tools mapped to the Query root field that reads the same records, and to the persisted operation the web client already sends on it.

**Classes:** `op` = a persisted operation exists. `field-no-op` = the root field exists but no persisted query uses it, so the client needs a new operation (name proposed). `partial` = the root field covers the records but lacks a filter or parameter the tool has (named in notes). `none` = no GraphQL equivalent (reason in notes).

**Paging:** "cursor" = Connection with pageInfo (first/after). "limit/offset", "limit only" and "unpaged" are plain lists. **Filter shape:** "DataTable fieldFilters only" = `DataTableConnectionInput` (query, fieldFilters, filterGroups, sort) with no typed filters. "bespoke X" = a dedicated input type with typed filters. "DataTable + typed ..." = DataTableConnectionInput plus typed root arguments.

**How the comparison was done:** each tool's implementation in `internal/modules/agent/app/agentquerytoolservice/` was matched to the service or repository method it calls, and that method to the method the root field's resolver calls (`internal/api/graphql/resolver/*resolver/*.resolvers.go`). The list-catalog tools call `repo.List` (limit/offset). The resolvers call `repo.ListConnection` (cursor) through the service, with the same `querybuilder` field configuration, so they count as the same read.

**Picking an operation:** a `Desk*` operation on the root field is preferred, then `<Type>Table`, then the only operation on the field. Where a Desk operation is a record-scoped view (DeskRecordFailures, DeskRecordShipments, DeskCustomerRateAgreements, DeskCarrierSettlements, DeskEdiPartnerMessages), the register operation is listed as the alternative.

## accounting

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `accounting_get_connection` | op | accountingSyncStatus + accountingSyncSummary | AccountingSyncStatus (+ AccountingSyncSummary) | n/a (object) | integrationType arg | Same AccountingConnectionService.Status + AccountingSyncService.Summary. system -> integrationType. Two ops, both needed. |
| `accounting_get_mapping` | partial | accountingMappings + accountingReferenceObjects | AccountingMappings (+ AccountingReferenceObjects) | cursor (first/after) | bespoke AccountingMappingFilterInput (targetTypes, states, requiredOnly, search) | Tool calls AccountingMappingService.FindMapping (lookup one mapping by key / mappingId / recordId + targetType); no Query reads a single mapping. Reachable only by listing with filter.search/targetTypes. SearchReference = accountingReferenceObjects (same method). |
| `accounting_get_sync_record` | op | accountingSyncRecord + accountingSyncAttempts | AccountingSyncRecord (+ AccountingSyncAttempts) | n/a | id | Same AccountingSyncService.GetRecord / ListAttempts. |
| `accounting_list_drift_findings` | op | accountingDriftFindingTable (+ accountingDriftOverview) | AccountingDriftFindingTable (+ AccountingDriftOverview) | cursor | DataTable fieldFilters only | Same AccountingDriftService.List/Overview. Tool sets typed Statuses/Kinds/ObjectTypes/Search; GraphQL has none of these typed, so status/kind/objectType must go as fieldFilters, and tool query (ILIKE number/party) becomes input.query (querybuilder search; verify same columns). |
| `accounting_list_inbound_changes` | op | accountingInboundChangeTable | AccountingInboundChangeTable | cursor | DataTable fieldFilters only | Same AccountingInboundService.List. status/kind/reason only as fieldFilters (typed Statuses/Kinds/Reasons not exposed); query -> input.query. |
| `accounting_list_mapping_gaps` | op | accountingMappings + accountingMappingSummary | AccountingMappings (+ AccountingMappingSummary) | cursor (first/after) | bespoke AccountingMappingFilterInput | Same ListMappings/Summary. requiredOnly, targetType -> filter.requiredOnly/targetTypes; gaps = states filter. |
| `accounting_list_sync_records` | op | accountingSyncRecords (+ accountingSyncObjectStates) | AccountingSyncRecords (+ AccountingSyncObjectStates for documentIds) | cursor (first/after) | bespoke AccountingSyncRecordFilterInput (statuses, objectTypes, errorCategories, objectId, search) | Same ListRecords/ObjectStates. Picked AccountingSyncRecords over AccountingSyncRecordTable: its typed filter matches the tool's status/documentType/errorCategory/query exactly; the Table op only takes DataTable fieldFilters. |

## agent

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `agent_get_run` | op | agentRun (+ agentProposals) | AgentRunDetail (+ AgentProposalTable filtered by run) | n/a | id | Same AgentRunService.GetByID. includeEvents ~ AgentRun.transcript (AgentRunDetail selects it); includeProposals -> agentProposals fieldFilter on run id. Tool also applies thread-ownership visibility. |
| `agent_list_memories` | partial | agentMemories (alt deskMemories) | AgentMemoryTable (alt DeskMemories) | cursor | DataTable fieldFilters only (deskMemories: bespoke DeskMemoriesInput scope/roleId/query) | Tool calls AgentMemoryService.Recall: query is a semantic/similarity recall, plus RecordUse side effect. AgentMemoryTable lists the same rows with kind/subjectType/subjectId/toolName as fieldFilters but has no similarity ranking. |
| `agent_list_runs` | op | agentRuns | AgentRunTable | cursor | DataTable fieldFilters only | Tool repo.List (offset) vs resolver AgentRunService.ListConnection (cursor); same filter config. mine -> fieldFilter agentDefinitionId = calling agent; tool additionally filters by thread ownership. |

## billing

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `billing_get_queue_item` | none | - | - | - | - | Billing queue has only Mutations in GraphQL (update/assign/post/release); item read and readiness are REST-only. |
| `billing_list_queue_items` | none | - | - | - | - | No billingQueue Query root field; the queue list is REST-only. |
| `billing_list_transfer_candidates` | partial | shipments (billingTransferEligible: true) | BillingTransferCandidates | cursor | bespoke ShipmentsInput (DataTable + typed) | Tool calls ShipmentService.ListBillingTransferCandidates, which returns a per-row transfer decision (PlanBillingTransfers). shipments(billingTransferEligible) returns the same rows without the decision; markCompletedReadyToInvoice has no equivalent; status/customerId/deliveredFrom/To -> status + fieldFilters. |

## carrier

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `carrier_list_capacity_postings` | field-no-op | carrierCapacityPostings | propose DeskCarrierCapacityPostings | cursor | DataTable + typed carrierId, openOnly | Root field reads the same CarrierCapacityPostingRepository; no persisted op selects it. |
| `carrier_list_intelligence_events` | op | carrierIntelEvents | CarrierIntelEventTable (alt CarrierIntelEventInbox) | cursor | DataTable + bespoke CarrierIntelEventFilterInput (statuses, severities, categories, carrierId, openOnly) | Same CarrierIntelService.ListEvents. fromDate/toDate (DetectedFrom/To) not typed: send as fieldFilters on detectedAt. Picked Table (no Desk op); Inbox is the triage view. |
| `carrier_list_invoice_matches` | op | carrierInvoiceMatches | CarrierInvoiceMatches | limit/offset | typed args status, carrierId | Same CarrierSettlementService.ListInvoiceMatches. |

## customer

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `customer_get_update_preferences` | field-no-op | customer | propose DeskCustomerUpdatePreferences | n/a | id | customer exposes statusUpdatePreference/statusUpdateRecipients, but no persisted op selects them (DeskCustomerCard does not); fallback recipients need emailProfile.toRecipients. |

## detention

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `detention_get_occurrence` | op | detentionOccurrence | DeskDetentionCard | n/a | id | Resolver DetentionService.GetOccurrenceDetail (priced detail) is a superset of tool repo.GetByID + PriceOpenClock. |
| `detention_list_occurrences` | partial | detentionDesk (+ shipmentDetention) | DetentionDesk (+ ShipmentDetention for one load) | unpaged list | no args | Same DetentionService.ListDesk; tool filters urgency/proNumber/shipmentId in memory (client can too). Missing: the billing-hold entries from ListAwaitingApproval, which no Query exposes. |

## dispatch

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `dispatch_get_board` | op | dispatchBoard | DispatchBoard | limit only | bespoke DispatchBoardInput | Same DispatchConsoleService.GetBoard. hoursAhead -> windowStart/windowEnd, uncoveredOnly -> includeCovered:false, query, limit. |
| `dispatch_list_tenders` | op | tendersByShipment (+ liveTenderByMove) | TendersByShipment (+ LiveTenderByMove for shipmentMoveId) | unpaged list | shipmentId arg | Same TenderService.ListByShipment. |

## document

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `document_get_summary` | none | - | - | - | - | Document content and extraction summary (DocumentContentService) have no Query root; documents are read through REST. |
| `document_list` | none | - | - | - | - | Semantic/vector retrieval (RetrievalSearcher.SearchDocuments); no GraphQL document search. |

## edi

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `edi_get_ack_health` | partial | ediPartnerScorecards (+ ediSummary, ediPartnerCard) | EdiPartnerScorecards (+ EdiSummary) | unpaged | sinceHours only | Tool calls EdiService.GetEDIAckHealth, a separate computation. Scorecards/summary carry per-partner and org overdue-ack counts and DeskEdiPartnerCard lists overdue acks per partner, but there is no overdueAfterSeconds, transactionSet or partner filter. |
| `edi_get_inbound_file` | partial | ediInboundFiles | EdiInboundFileTable | cursor | DataTable + typed status, partnerId | No single ediInboundFile(id) Query; only via list with fieldFilter id. includeRaw (raw payload) and includeMessages not exposed. |
| `edi_get_message` | partial | ediMessages | DeskEdiPartnerMessages (alt EdiMessageTable) | cursor | DataTable + typed transactionSet, direction, partnerId | No single ediMessage(id) Query; only via list with fieldFilter id. includeRaw not exposed. |
| `edi_get_partner` | op | ediPartnerCard (+ ediPartnerReadiness) | DeskEdiPartnerCard (+ EdiPartnerReadiness) | n/a | id | Tool GetPartner + GetPartnerReadiness; resolver GetPartnerCard is a superset; readiness is its own op. |
| `edi_get_transfer` | partial | ediTransfers | EdiTransferTable | cursor | DataTable + required direction | No single ediTransfer(id) Query; MappingPreview (how the transfer maps) not exposed. |
| `edi_list_carrier_invoices` | op | ediCarrierInvoices (+ suggestCarrierForEdiInvoice) | EdiCarrierInvoices (+ SuggestCarrierForEdiInvoice) | limit/offset | typed reconciliationStatus | Same ListUnmatchedEDIInvoices / SuggestCarrierForInvoice. |
| `edi_list_inbound_files` | op | ediInboundFiles | EdiInboundFileTable | cursor | DataTable + typed status, partnerId | Same ListInboundFilesCursor. |
| `edi_list_messages` | partial | ediMessages | DeskEdiPartnerMessages (alt EdiMessageTable) | cursor | DataTable + typed transactionSet, direction, partnerId | Same ListMessagesCursor, but resolver leaves typed ShipmentID, InvoiceID, AckStatus, CreatedAfter/Before unset. ackStatus/deliveryStatus/created window can go as fieldFilters; shipmentId/invoiceId (linked-record filters) cannot. |
| `edi_list_partners` | op | ediPartners | EdiPartnerTable | cursor | DataTable fieldFilters only | Same ListPartnersCursor; customerId/status as fieldFilters (typed CustomerID not exposed). |
| `edi_list_tender_changes` | none | - | - | - | - | EDI tender change review queue (ListTenderChanges) has no Query root. |
| `edi_list_transfer_changes` | none | - | - | - | - | EDI transfer change review queue (ListTransferChanges) has no Query root. |
| `edi_list_transfers` | op | ediTransfers | EdiTransferTable | cursor | DataTable + required direction | Same List{In,Out}boundTransfersCursor. Tool's direction is optional (GraphQL requires it: two calls for both); inboundFileId/status as fieldFilters (typed InboundFileID not exposed). |

## equipment

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|

## exception

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `exception_get_service_failure` | op | serviceFailure | DeskServiceFailureCard | n/a | id | Same repository GetByID via ServiceFailureService. |
| `exception_list_service_failure_reason_codes` | op | serviceFailureReasonCodes | ServiceFailureReasonCodeTable | cursor | DataTable fieldFilters only | Tool uses SelectOptions (same table); appliesTo -> fieldFilter, query -> input.query. |
| `exception_list_service_failures` | op | serviceFailures | DeskRecordFailures (alt ServiceFailureTable) | cursor | DataTable + typed shipmentId, locationId | Tool ServiceFailureService.List (offset) vs ListConnection. Desk op picked per rule; it is the record-scoped view, ServiceFailureTable is the unscoped register. |
| `exception_list_watchtower_items` | partial | watchtowerItems | WatchtowerFeed | cursor | bespoke WatchtowerItemsInput | Same WatchtowerService.List. kinds, severities, assignedToMe, unassigned, overdueOnly, orderByDue, includeResolved (-> unresolvedOnly:false) map; unseenOnly has no input (connection returns seenAt, so it can only be filtered client-side). |

## factoring

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `factoring_get_agreement` | op | factoringAgreements | FactoringAgreements | unpaged list | factoringCompanyId arg | Same AgreementService.List. |
| `factoring_get_readiness` | op | invoice (factoringState) [root factoringReadiness has no op] | InvoiceFactoring | n/a | id | factoringReadiness(invoiceId) calls the same ReadinessService.GetReadiness but no persisted op uses it; InvoiceFactoring selects invoice.factoringState. |
| `factoring_get_remittance` | op | factoringRemittance (+ factoringRemittanceLines) | FactoringRemittanceDetail (+ FactoringRemittanceLines) | lines: cursor | lines: bespoke FactoringRemittanceLineListInput | Same RemittanceService.Get; matchStatuses -> line list filter. |
| `factoring_get_reserve_balance` | op | factoringReserveBalance | FactoringReserveBalance | n/a | factoringCompanyId arg | Same LedgerService.GetBalance. |
| `factoring_get_submission` | op | factoringSubmission | FactoringSubmissionDetail | n/a | id | Same SubmissionService.Get. |
| `factoring_list_noas` | op | factoringNoas | FactoringNoas | cursor | bespoke FactoringNoaListInput (table + customerId, factoringCompanyId, statuses) | Same NoaService.List. |
| `factoring_list_readiness` | op | factoringInvoiceStates (+ factoringSummary) | FactoringInvoiceStateTable (+ FactoringSummary) | cursor | bespoke FactoringInvoiceStateListInput (table + verdicts, factoringCompanyId, customerId) | Same ReadinessService.ListStates/Summary. |
| `factoring_list_remittances` | op | factoringRemittances | FactoringRemittanceTable (alt FactoringRemittances) | cursor | bespoke FactoringRemittanceListInput (table + factoringCompanyId, inboundMessageId, statuses) | Same RemittanceService.ListConnection. Two ops on the root; Table picked per rule. |
| `factoring_list_submissions` | op | factoringSubmissions | FactoringSubmissionTable | cursor | bespoke FactoringSubmissionListInput (table + factoringCompanyId, statuses) | Same SubmissionService.List. |

## formula

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `formula_get_template` | field-no-op | formulaTemplate | propose DeskFormulaTemplateCard | n/a | id | Same FormulaTemplateRepository.GetByID via service; no persisted op reads formulaTemplate(id). |
| `formula_list_templates` | op | formulaTemplates | FormulaTemplateTable | cursor | DataTable fieldFilters only | Same repository list. |

## fuel

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `fuel_list_cards` | op | fuelCards | FuelCardTable | cursor | bespoke FuelCardsInput (DataTable + typed) | Same FuelPurchase repository ListCards via service. |
| `fuel_list_ifta_jurisdictions` | partial | iftaJurisdictions | IftaJurisdictions | unpaged list | typed membersOnly | Same IFTA ListJurisdictions; countryCode filter missing (client-side only). |
| `fuel_list_ifta_mileage_entries` | op | iftaMileageEntries | IftaMileageEntryTable | cursor | bespoke IFTAMileageEntriesInput (DataTable + typed) | Same ListMileageEntries. |
| `fuel_list_ifta_returns` | op | iftaReturns | IftaReturnTable | cursor | bespoke IFTAReturnsInput (DataTable + year, statuses) | Same ListReturns. |
| `fuel_list_purchase_imports` | op | fuelPurchaseImports | FuelPurchaseImportTable | cursor | bespoke FuelPurchaseImportsInput (DataTable + typed) | Same ListImportBatches. |
| `fuel_list_purchases` | op | fuelPurchases | FuelPurchaseTable | cursor | bespoke FuelPurchasesInput (DataTable + typed) | Same ListPurchases. |

## home

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `home_get_layout` | op | homeLayout | HomeLayout | n/a | none (self-scoped) | Same HomeLayoutService.GetEffective. |
| `home_list_widgets` | op | homeWidgetCatalog | HomeWidgetCatalog | unpaged | none | Same GetCatalog; category filtered client-side. |

## inbox

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `inbox_get_inbound_message` | partial | inboundMessage | InboundMessage | n/a | id | Same InboundMessageService.GetByID; the pending reply drafts the tool attaches (InboundReplyRepository.DraftsByMessageIDs) are not on InboundMessage. |
| `inbox_list_capture_batches` | op | captureBatches (+ captureBatch) | CaptureBatches (+ CaptureBatch for captureBatchId) | cursor | bespoke CaptureBatchesInput | Same CaptureService.ListBatches/GetBatch. |
| `inbox_list_email_profiles` | op | emailProfiles | EmailProfileTable | cursor | DataTable fieldFilters only | Tool uses SelectProfileOptions (same profiles, lighter); query -> input.query. |
| `inbox_list_inbound_messages` | partial | inboundMessages | InboundMessages | cursor | bespoke InboundMessagesInput | Same InboundMessageService.List for the filters (classification, mailboxId, status). When query is given the tool uses semantic RetrievalSearcher.SearchInboundMessages; GraphQL query is lexical. |
| `inbox_list_triage_assignees` | op | triageAssignees (+ triageSettings) | TriageAssignees (+ TriageSettings) | unpaged | none | Same Assignees/Settings. |

## insight

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `insight_get` | partial | watchtowerItem | WatchtowerItemDetail | n/a | id | No insight Query root; insights reach GraphQL only as watchtower items of kind Insight. InsightService.GetDetail (evidence, detector detail) is not exposed. |
| `insight_get_daily_briefing` | op | todaysBriefing | TodaysBriefing | n/a | bespoke TodaysBriefingInput (roleKey, briefingDate) | Same BriefingService.Today; role -> roleKey, date -> briefingDate. |
| `insight_list` | partial | watchtowerItems (kinds:[Insight]) | WatchtowerFeed | cursor | bespoke WatchtowerItemsInput | No insight list root; InsightService.List (category/status browse) differs. severity maps to severities; category and insight status do not. |
| `insight_list_weather_alerts` | partial | watchtowerItems (kinds:[WeatherAlert]) | WatchtowerFeed | cursor | bespoke WatchtowerItemsInput | No weather root; WeatherAlertService.GetActiveAlerts is a feed read. severity maps; query does not. |

## intake

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `intake_get_draft` | none | - | - | - | - | Shipment draft extracted from a document (DocumentContentService.GetShipmentDraft) has no Query root. |

## invoice

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `invoice_get` | op | invoice | DeskInvoiceCard | n/a | id | Same InvoiceService.GetByID; invoiceIds (many) -> one call per id or invoices with an id fieldFilter. |
| `invoice_get_adjustment` | op | invoiceAdjustment | InvoiceAdjustmentApprovalDetail | n/a | id | Same InvoiceAdjustmentService.GetDetail. |
| `invoice_get_run` | none | - | - | - | - | Invoice runs are REST-only (no invoiceRun root). |
| `invoice_list` | op | invoices | InvoiceTable (alt InvoiceRegister, ShipmentInvoices) | cursor | DataTable fieldFilters only | repo.List vs InvoiceService.ListConnection. Table picked per rule. |
| `invoice_list_adjustments` | partial | invoiceAdjustmentApprovals | InvoiceAdjustmentApprovals | cursor | bespoke InvoiceAdjustmentApprovalsInput | Same ListApprovals for the approval queue; the per-invoice lineage (GetLineage by invoiceId) has no Query. |
| `invoice_list_disputes` | field-no-op | invoiceDisputes | propose DeskInvoiceDisputes | unpaged list | invoiceId arg (one) | Same InvoiceDisputeRepository.ListByInvoiceIDs; no persisted op; takes one invoiceId, tool takes many. |
| `invoice_list_open_statements` | none | - | - | - | - | Open statements come from invoice runs (InvoiceRunService.ListOpenStatements); REST-only. |
| `invoice_list_runs` | none | - | - | - | - | Invoice runs are REST-only. |
| `invoice_list_share_candidates` | none | - | - | - | - | Computed share-recipient candidates (ListShareCandidates); no Query. |

## ledger

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `ledger_get_fiscal_close_blockers` | none | - | - | - | - | Fiscal periods and close blockers are REST-only (fiscal_period.graphqls defines types but no Query). |
| `ledger_get_journal_entry` | op | journalEntryCard (alt journalEntry) | DeskJournalEntryCard (alt JournalEntryDetail) | n/a | id | journalEntry calls JournalEntryService.GetEntry (same records); journalEntryCard is the Desk card (superset, carries account names the tool looks up). Desk picked per rule. |
| `ledger_get_manual_journal` | op | manualJournalCard (alt manualJournal) | DeskManualJournalCard | n/a | id | manualJournal(id) has no op; the Desk card reads the same journal with its entries. |
| `ledger_list_accounts` | none | - | - | - | - | GL accounts are REST-only (no glAccounts root). |
| `ledger_list_fiscal_periods` | partial | fiscalYears (periods) | propose DeskFiscalPeriods (FiscalYearTable does not select periods) | cursor (over years) | DataTable fieldFilters only | Periods are reachable only nested under FiscalYear.periods; no period-level filter/sort/paging and no op selects them. |
| `ledger_list_journal_entries` | partial | journalReviewTable | JournalReviewTable | cursor | DataTable fieldFilters only | Resolver uses JournalReviewService.List, tool JournalEntryService.List: same rows, different service. fiscalPeriodId/status/referenceType/date window only as fieldFilters; verify review scope does not narrow rows. |
| `ledger_list_journal_reversals` | op | journalReversals | JournalReversalTable | cursor | DataTable fieldFilters only | Same repository. |
| `ledger_list_manual_journals` | op | manualJournals | ManualJournalTable | cursor | DataTable fieldFilters only | Same repository. |

## location

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|

## masterdata

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|

## order

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `order_get` | op | order | DeskOrderCard | n/a | id | Same OrderService.Get. |
| `order_list` | op | orders | OrderTable | cursor | DataTable fieldFilters only | repo.List vs ListConnection. |

## rate

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `rate_get_matrix` | partial | rateMatrix | DeskRateMatrixCard | n/a | id, cellOffset | rateMatrix(id), its dimensions and its cells(offset, limit) are in the schema and DeskRateMatrixCard reads them; the tool converts once a get sends its operation's other variables (cellOffset). |
| `rate_list_confirmations` | none | - | - | - | - | Rate confirmations by move are REST-only. |
| `rate_list_imports` | none | - | - | - | - | Rate import batches are REST-only. |

## receivable

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `receivable_get_aging` | op | arAgingSummary | ArAgingSummary | unpaged (object) | asOfDate | Same GetAgingSummary; tool filters customerId/overdueOnly and pages rows in memory. |
| `receivable_get_bank_receipt` | none | - | - | - | - | Bank receipts and their suggested matches are REST-only. |
| `receivable_get_statement` | op | arCustomerStatement | ArCustomerStatement | n/a | customerId, startDate, asOfDate | Same GetCustomerStatement. |
| `receivable_list_bank_receipt_exceptions` | none | - | - | - | - | Bank receipt exceptions and work items are REST-only. |
| `receivable_list_collection_worklist` | partial | arCollectionsWorklist | ArCollectionsWorklist | limit only | limit | Same GetCollectionsWorklist, but the resolver passes no AsOfDate; tool's asOf is missing. severity is filtered in memory. |
| `receivable_list_credit_memo_applications` | op | invoice (creditApplications) | DeskInvoiceCard | n/a | id | No list root; Invoice.creditApplications is selected by DeskInvoiceCard (and InvoiceArContext). One call per invoiceId. |
| `receivable_list_customer_payments` | op | customerPayments | CustomerPaymentTable | cursor | DataTable fieldFilters only | Tool CustomerPaymentService.List vs resolver repo.ListConnection. customerId/status as fieldFilters. |
| `receivable_list_open_items` | op | arOpenItems | ArOpenItems | unpaged list | customerId, asOfDate | Same ListOpenItems; disputedOnly/minDaysPastDue/paging done in memory by the tool. |

## report

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `report_get` | op | reportDefinition (+ cannedReports) | ReportDefinitionById (+ CannedReports for reportKey) | n/a | id | Same ReportingService.GetDefinition; canned reports by key from cannedReports. |
| `report_get_dataset` | op | reportCatalog | ReportCatalog | unpaged | none | Catalog metadata: one entity of ReportCatalog.entities; no per-dataset arg, the client picks the entity. |
| `report_get_run` | op | reportRun | ReportRunById | n/a | id | Same GetRun. Row reads (ReadRunRows) and the analysis path of the tool are not GraphQL. |
| `report_list` | op | reportDefinitions (+ cannedReports) | ReportDefinitionsTable (+ CannedReports) | cursor | DataTable fieldFilters only | Tool merges ListCanned + ListDefinitions; category as fieldFilter. |
| `report_list_dashboards` | op | reportDashboards | ReportDashboards | unpaged | none | Same ListDashboards; query/paging client-side. |
| `report_list_datasets` | op | reportCatalog | ReportCatalog | unpaged | none | Catalog metadata with no record kind; category/query filtered client-side. |
| `report_list_runs` | op | reportRuns | ReportRunsTable | cursor | DataTable + bespoke ReportRunsFilterInput (definitionId, statuses) | Same ReportingService runs list. |
| `report_list_schedules` | op | reportSchedules | ReportSchedules | unpaged list | typed definitionId | Same ListSchedules; enabledOnly filtered client-side. |
| `report_list_table_change_alerts` | op | tcaSubscriptions | TCASubscriptionTable | cursor | DataTable fieldFilters only | Same TCA subscription repository. |

## settlement

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `settlement_get_carrier` | op | carrierSettlement | DeskCarrierSettlementCard | n/a | id | Same CarrierSettlementService.Get. |
| `settlement_get_dispute` | op | settlementDispute | SettlementDisputeDetail | n/a | id | Same GetDispute. |
| `settlement_get_driver` | op | driverSettlement (+ settlementDisputes) | DeskDriverSettlementCard (+ SettlementDisputeTable by workerId) | n/a | id | Same DriverSettlementService.Get; the open disputes for the worker via settlementDisputes fieldFilters. |
| `settlement_get_driver_earnings` | op | workerEarningsSummary | WorkerEarningsSummary | n/a | workerId | Same GetWorkerEarningsSummary. |
| `settlement_list_advances` | op | payAdvances | PayAdvanceTable | cursor | DataTable fieldFilters only | Same PayAdvanceRepository; status/workerId as fieldFilters. |
| `settlement_list_carrier` | op | carrierSettlements | DeskCarrierSettlements (alt CarrierSettlementTable) | cursor | DataTable fieldFilters only | Tool service.List vs resolver repo.ListConnection; carrierId/status as fieldFilters. Desk op picked per rule. |
| `settlement_list_driver` | op | driverSettlements | DriverSettlementTable | cursor | DataTable fieldFilters only | workerId/status/hasExceptions (has_exceptions column)/open as fieldFilters. |
| `settlement_list_driver_expenses` | op | driverExpenses | DriverExpenseTable | cursor | DataTable fieldFilters only | Same ListExpensesConnection; tool already sends fieldFilters (workerId, pendingOnly -> status). |
| `settlement_list_escrow_accounts` | op | escrowAccounts | EscrowAccountTable | cursor | DataTable fieldFilters only | Same EscrowAccountRepository. |
| `settlement_list_pay_assignments` | op | workerPayAssignments | WorkerPayAssignments | unpaged list | workerId | Same DriverPayService.ListWorkerAssignments. |
| `settlement_list_pay_codes` | field-no-op | payCodeOptions | propose DeskPayCodeOptions (alt PayCodeTable) | unpaged list | typed direction | payCodeOptions calls the same ListActivePayCodes but no op uses it; PayCodeTable (payCodes) reads all codes through ListPayCodesConnection. |
| `settlement_list_pay_events` | op | driverPayEvents | DriverPayEventTable | cursor | DataTable fieldFilters only | workerId/shipmentId/status as fieldFilters. |
| `settlement_list_pay_profiles` | op | payProfiles | PayProfileTable | cursor | DataTable fieldFilters only | classification as fieldFilter. |
| `settlement_list_payroll_exports` | op | payrollExports | PayrollExports | limit only | limit | Same TimesheetService.ListExports. |
| `settlement_list_recurring_deductions` | op | recurringDeductions | RecurringDeductionTable | cursor | DataTable fieldFilters only | status/workerId as fieldFilters. |
| `settlement_list_recurring_earnings` | op | recurringEarnings | RecurringEarningTable | cursor | DataTable fieldFilters only | status/workerId as fieldFilters. |

## shipment

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `shipment_get` | partial | shipment (+ shipments, shipmentComments) | ShipmentDetail (+ ShipmentComments) | n/a | id | Same ShipmentService.Get. proNumber lookup -> shipments fieldFilter proNumber. Shipment holds (ShipmentHoldRepository) are not on the GraphQL Shipment type (only hold events via shipmentEvents). |
| `shipment_get_tracking` | none | - | - | - | - | Composite live read: board moves + stored stop ETAs + live vehicle positions + HOS. No single root; pieces exist as shipment.eta (ShipmentDetail), vehiclePosition (DeskTractorPosition), workerHosState (DeskWorkerHos). |
| `shipment_list` | op | shipments | DeskRecordShipments (alt ShipmentBoardTable) | cursor | bespoke ShipmentsInput (DataTable + typed) | Same ShipmentRepository list (tool offset, resolver cursor). status -> input.status. Tool resolves party names (customer/location/worker/commodity/type) to ids first; GraphQL wants ids. Desk op picked per rule; ShipmentBoardTable selects the full row. |
| `shipment_list_permits` | none | - | - | - | - | Permits and permit requirements have no Query root. |
| `shipment_list_recurring` | op | recurringShipments | RecurringShipmentTable | cursor | DataTable fieldFilters only | Same repository list. |

## web

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `web_list_portal_sites` | none | - | - | - | - | Agent portal-automation sites (agent extension config); no GraphQL. |
| `web_list_watches` | none | - | - | - | - | Agent web watches (Exa Monitors), agent-scoped state; no GraphQL. |

## worker

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `worker_get` | op | worker | DeskWorkerCard | n/a | id | Same WorkerRepository.GetByID. |
| `worker_get_credential` | field-no-op | workerCredential | propose DeskWorkerCredential | n/a | id | Same WorkerCredentialRepository via service Get; no op uses workerCredential(id) (DeskWorkerCredentials reads the summary). |
| `worker_get_dot_random_draw` | op | dotRandomDraw | DotRandomDraw | n/a | id | Same GetDraw; entries are selected by DotRandomDraw. |
| `worker_get_hos` | op | workerHosState | DeskWorkerHos (alt WorkerHosState) | n/a | workerId | Same TelematicsService.GetWorkerHOSState (telematics read). |
| `worker_get_schedule` | op | workerShiftAssignments (+ workerAvailabilityPreferences, shiftSwapRequests) | WorkerShiftAssignments (+ WorkerAvailabilityPreferences, ShiftSwapRequests) | unpaged lists | workerId | Same SchedulingService ListAssignments/ListPreferences/ListSwaps; tool's team-scope guard is server-side. |
| `worker_list_dot_random_draws` | op | dotRandomDraws (+ dotRandomPools) | DotRandomDraws (+ DotRandomPools) | unpaged / cursor | poolId / DOTRandomPoolsInput | Same ListDraws/ListPools. |
| `worker_list_shift_templates` | op | shiftTemplates | ShiftTemplates | limit only | activeOnly, limit | Same ListTemplates. |

## Counts per class

| class | tools |
|---|---|
| op, not yet converted | 88 |
| op, converted (rows removed) | 31 |
| field-no-op | 6 |
| partial, not yet converted | 23 |
| partial, converted (rows removed) | 7 |
| none | 21 |
| **total** | **176** |

## Binding trial (2026-10-20)

Each of the 119 `op` tools was run alone through `agentread/gen`. A tool binds either by the rule
(root field, record kind, tool name, then the Desk's, the table's or the only operation) or by a
named operation in `reads.yml`, which the generator proves: the operation reads one root field,
the field's shape matches the tool's verb, and the field's resolver checks the tool's own resource
for read.

- **By the rule:** 49 tools.
- **By a named operation** (the one this inventory picked): 46 tools.
- **Not yet:** 22 tools, each with the generator's reason below. Most need `entity` named in
  `reads.yml`, because their type binds to a generated model (`gqlmodel`) rather than to the
  record the resolver filters. A few have a shape no list or get tool reads (no arguments at all,
  a catalog, two root fields), and are decided in their module's PR.

| Tool | Named operation | Generator says |
|---|---|---|
| `detention_get_occurrence` | `DeskDetentionCard` | DetentionOccurrenceDetail binds to no Go type; name the entity in reads.yml |
| `dispatch_get_board` | `DispatchBoard` | DispatchBoard binds to no Go type; name the entity in reads.yml |
| `edi_get_partner` | `DeskEdiPartnerCard` | EdiPartnerCard binds to no Go type; name the entity in reads.yml |
| `factoring_get_submission` | `FactoringSubmissionDetail` | factoring_get_submission is a get tool, but FactoringSubmissionDetail reads factoringSubmission, which is not one |
| `home_get_layout` | `HomeLayout` | home_get_layout is a get tool, but HomeLayout reads homeLayout, which is not one |
| `home_list_widgets` | `HomeWidgetCatalog` | home_list_widgets is a list tool, but HomeWidgetCatalog reads homeWidgetCatalog, which is not one |
| `receivable_list_credit_memo_applications` | `DeskInvoiceCard` | receivable_list_credit_memo_applications is a list tool, but DeskInvoiceCard reads invoice, which is not one |
| `report_get` | `ReportDefinitionById` | ReportDefinition binds to no Go type; name the entity in reads.yml |
| `report_get_dataset` | `ReportCatalog` | report_get_dataset is a get tool, but ReportCatalog reads reportCatalog, which is not one |
| `report_get_run` | `ReportRunById` | ReportRun binds to no Go type; name the entity in reads.yml |
| `report_list` | `ReportDefinitionsTable` | ReportDefinition binds to no Go type; name the entity in reads.yml |
| `report_list_dashboards` | `ReportDashboards` | ReportDashboard binds to no Go type; name the entity in reads.yml |
| `report_list_datasets` | `ReportCatalog` | report_list_datasets is a list tool, but ReportCatalog reads reportCatalog, which is not one |
| `report_list_runs` | `ReportRunsTable` | ReportRun binds to no Go type; name the entity in reads.yml |
| `report_list_schedules` | `ReportSchedules` | ReportSchedule binds to no Go type; name the entity in reads.yml |
| `settlement_list_payroll_exports` | `PayrollExports` | the resolver of payrollExports checks no read permission, so the tool's own could not be held to it |
| `shipment_list` | `DeskRecordShipments` | Shipment binds to no Go type; name the entity in reads.yml |
| `worker_get_hos` | `DeskWorkerHos` | worker_get_hos names DeskWorkerHos, which reads 2 root fields; a tool reads one |
| `worker_get_schedule` | `WorkerShiftAssignments` | worker_get_schedule is a get tool, but WorkerShiftAssignments reads workerShiftAssignments, which is not one |
