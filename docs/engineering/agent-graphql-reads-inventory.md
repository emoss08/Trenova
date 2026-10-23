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
| `billing_list_transfer_candidates` | partial | shipments (billingTransferEligible: true) | BillingTransferCandidates | cursor | bespoke ShipmentsInput (DataTable + typed) | Tool calls ShipmentService.ListBillingTransferCandidates, which returns a per-row transfer decision (PlanBillingTransfers). shipments(billingTransferEligible) returns the same rows without the decision; markCompletedReadyToInvoice has no equivalent; status/customerId/deliveredFrom/To -> status + fieldFilters. Left on the repository path by the billing PR: the per-row transfer decision needs a field on the shipment (or a root of its own) that the billing transfer screen reads too, a change to that screen rather than to this read. |

## carrier

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|


## dispatch

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `dispatch_get_board` | op | dispatchBoard | DispatchBoard | limit only | bespoke DispatchBoardInput | Same DispatchConsoleService.GetBoard. hoursAhead -> windowStart/windowEnd, uncoveredOnly -> includeCovered:false, query, limit. Stays on the repository path (Operations PR): the board is one object of moves, drivers and a summary read by a bespoke input, which neither the list nor the get path reads, and its move and driver rows are types no kind claims, so the field guard would not gate them. Converting it needs an object-read tool shape and `@kind` on `DispatchBoardMove` and `DispatchBoardDriver`. |

## document

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `document_get_summary` | none | - | - | - | - | Document content and extraction summary (DocumentContentService) have no Query root; documents are read through REST. |
| `document_list` | none | - | - | - | - | Semantic/vector retrieval (RetrievalSearcher.SearchDocuments); no GraphQL document search. |

## edi

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `edi_list_tender_changes` | none | - | - | - | - | EDI tender change review queue (ListTenderChanges) has no Query root. |
| `edi_list_transfer_changes` | none | - | - | - | - | EDI transfer change review queue (ListTransferChanges) has no Query root. |

## equipment

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|

## exception

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `exception_list_watchtower_items` | partial | watchtowerItems | WatchtowerFeed | cursor | bespoke WatchtowerItemsInput | Same WatchtowerService.List. kinds, severities, assignedToMe, unassigned, overdueOnly, orderByDue, includeResolved (-> unresolvedOnly:false) map; unseenOnly has no input (connection returns seenAt, so it can only be filtered client-side). Stays on the repository path (Operations PR): each item carries taint from its own source (inbound mail, an EDI file, a weather alert, an agent run, the run behind a proposal or plan), worked out per item from the run records, which the GraphQL path's one record kind per binding cannot mark; and unseenOnly needs an input. |

## home

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
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
| `invoice_get_run` | none | - | - | - | - | Invoice runs are REST-only (no invoiceRun root). |
| `invoice_list_adjustments` | partial | invoiceAdjustmentApprovals | InvoiceAdjustmentApprovals | cursor | bespoke InvoiceAdjustmentApprovalsInput | Same ListApprovals for the approval queue; the per-invoice lineage (GetLineage by invoiceId) has no Query. Left on the repository path by the billing PR: with invoiceId it reads the invoice's correction lineage, which no root field reads; that needs a lineage root the invoice page also moves to. |
| `invoice_list_open_statements` | none | - | - | - | - | Open statements come from invoice runs (InvoiceRunService.ListOpenStatements); REST-only. |
| `invoice_list_runs` | none | - | - | - | - | Invoice runs are REST-only. |
| `invoice_list_share_candidates` | none | - | - | - | - | Computed share-recipient candidates (ListShareCandidates); no Query. |

## ledger

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `ledger_get_fiscal_close_blockers` | none | - | - | - | - | Close blockers are REST-only: `fiscalPeriods` lists the periods, but no Query computes what stops one closing (FiscalPeriodService.GetCloseBlockers). |
| `ledger_list_accounts` | none | - | - | - | - | GL accounts are REST-only (no glAccounts root). |

## location

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|

## masterdata

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|

## rate

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `rate_list_confirmations` | none | - | - | - | - | Rate confirmations by move are REST-only. |
| `rate_list_imports` | none | - | - | - | - | Rate import batches are REST-only. |

## receivable

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `receivable_get_aging` | op | arAgingSummary | ArAgingSummary | unpaged (object) | asOfDate | Same GetAgingSummary; tool filters customerId/overdueOnly and pages rows in memory. Left on the repository path by the billing PR: a get keyed by a date, not an id, which the generator's get shape (it sends only `$id`) cannot bind; the customer and overdue filters and the paging are the tool's own. |
| `receivable_get_bank_receipt` | none | - | - | - | - | Bank receipts and their suggested matches are REST-only. |
| `receivable_get_statement` | op | arCustomerStatement | ArCustomerStatement | n/a | customerId, startDate, asOfDate | Same GetCustomerStatement. Left on the repository path by the billing PR: a get keyed by a customer and dates, not an id, which the generator's get shape (it sends only `$id`) cannot bind. |
| `receivable_list_bank_receipt_exceptions` | none | - | - | - | - | Bank receipt exceptions and work items are REST-only. |
| `receivable_list_collection_worklist` | partial | arCollectionsWorklist | ArCollectionsWorklist | limit only | limit | Same GetCollectionsWorklist, but the resolver passes no AsOfDate; tool's asOf is missing. severity is filtered in memory. Left on the repository path by the billing PR. |
| `receivable_list_credit_memo_applications` | op | invoice (creditApplications) | DeskInvoiceCard | n/a | id | No list root; Invoice.creditApplications is selected by DeskInvoiceCard (and InvoiceArContext). One call per invoiceId. Left on the repository path by the billing PR: a list over many invoices needs a root that reads applications by invoice ids. |
| `receivable_list_open_items` | op | arOpenItems | ArOpenItems | unpaged list | customerId, asOfDate | Same ListOpenItems; disputedOnly/minDaysPastDue/paging done in memory by the tool. Left on the repository path by the billing PR: disputedOnly, minDaysPastDue and the paging are the tool's own, so the root needs those arguments and limit/offset first. |

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
| `settlement_list_payroll_exports` | op | payrollExports | PayrollExports | limit only | limit | Stays on the repository path: payrollExports checks timesheet export, where the tool reads with timesheet read, so converting would refuse a reader who may read timesheets but not export them. Which permission a payroll run list takes is the owner's call; the generator refuses the binding until the two agree. |

## shipment

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `shipment_get` | partial | shipment (+ shipments, shipmentComments) | ShipmentDetail (+ ShipmentComments) | n/a | id | Same ShipmentService.Get. proNumber lookup -> shipments fieldFilter proNumber. Shipment holds (ShipmentHoldRepository) are not on the GraphQL Shipment type (only hold events via shipmentEvents). Stays on the repository path (Operations PR): it needs the active holds on `Shipment`, a `DeskShipmentCard` sized for the Desk (ShipmentDetail selects the whole page), the newest comments read beside the card (a get has no reads alongside yet), the PRO number lookup and the summary/full detail switch. |
| `shipment_get_tracking` | none | - | - | - | - | Composite live read: board moves + stored stop ETAs + live vehicle positions + HOS. No single root; pieces exist as shipment.eta (ShipmentDetail), vehiclePosition (DeskTractorPosition), workerHosState (DeskWorkerHos). |
| `shipment_list` | op | shipments | DeskRecordShipments (alt ShipmentBoardTable) | cursor | bespoke ShipmentsInput (DataTable + typed) | Same ShipmentRepository list (tool offset, resolver cursor). status -> input.status. Tool resolves party names (customer/location/worker/commodity/type) to ids first; GraphQL wants ids. Desk op picked per rule; ShipmentBoardTable selects the full row. Stays on the repository path (Operations PR): its words match a customer, a stop's place, a driver, a commodity or a shipment type, resolved to ids and combined per word, and `shipments(query:)` matches only the PRO number and BOL; ShipmentsInput also has no customer, commodity or shipment type ids. Converting it without that search would lose the tool's main use. |
| `shipment_list_permits` | none | - | - | - | - | Permits and permit requirements have no Query root. |

## web

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|
| `web_list_portal_sites` | none | - | - | - | - | Agent portal-automation sites (agent extension config); no GraphQL. |
| `web_list_watches` | none | - | - | - | - | Agent web watches (Exa Monitors), agent-scoped state; no GraphQL. |

## worker

| tool | class | root field | operation (or proposed new op name) | paging | filter shape | notes (missing params / reason) |
|---|---|---|---|---|---|---|

## Counts per class

| class | tools |
|---|---|
| op, not yet converted | 28 |
| op, converted (rows removed) | 91 |
| field-no-op, not yet converted | 0 |
| field-no-op, converted (rows removed) | 6 |
| partial, not yet converted | 12 |
| partial, converted (rows removed) | 18 |
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
- **Not yet:** 20 tools, each with the generator's reason below. Most need `entity` named in
  `reads.yml`, because their type binds to a generated model (`gqlmodel`) rather than to the
  record the resolver filters. A few have a shape no list or get tool reads (no arguments at all,
  a catalog, two root fields), and are decided in their module's PR.

| Tool | Named operation | Generator says |
|---|---|---|
| `dispatch_get_board` | `DispatchBoard` | DispatchBoard binds to no Go type; name the entity in reads.yml |
| `factoring_get_submission` | `FactoringSubmissionDetail` | factoring_get_submission is a get tool, but FactoringSubmissionDetail reads factoringSubmission, which is not one |
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
| `settlement_list_payroll_exports` | `PayrollExports` | the resolver of payrollExports checks no read permission, so the tool's own could not be held to it (it checks timesheet export; see the settlement section) |
| `shipment_list` | `DeskRecordShipments` | Shipment binds to no Go type; name the entity in reads.yml |
| `worker_get_hos` | `DeskWorkerHos` | worker_get_hos names DeskWorkerHos, which reads 2 root fields; a tool reads one |
| `worker_get_schedule` | `WorkerShiftAssignments` | worker_get_schedule is a get tool, but WorkerShiftAssignments reads workerShiftAssignments, which is not one |

The workforce PR bound both: `worker_get_hos` reads `WorkerHosState`, the telematics page's
single-root read of the clocks (the tool never returned violations), sent by the worker's id; and
a get tool may now read a keyed whole list (`worker_get_schedule` reads `WorkerShiftAssignments`
with the availability and open swaps alongside). See "Gets" in agent-graphql-reads.md.
