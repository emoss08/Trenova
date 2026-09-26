# Agent write coverage

<!-- Generated from the GraphQL schema, the REST route table, the registered agent
     tools and services/tms/internal/api/writecoverage/writecoverage.yml
     by running, in services/tms:
     go generate ./internal/api/writecoverage/...
     Do not edit by hand. -->

Every write a person can make in Trenova should either have an agent tool that
performs it or a reasoned exemption. This page is that ledger: each GraphQL
mutation and each POST, PUT, PATCH or DELETE route, the tool that covers it or
the reason none should, and the writes still waiting for a tool.

## The rule

A new mutation or a new write route ships with a decision in
`services/tms/internal/api/writecoverage/writecoverage.yml`.
Each write is one entry under `writes:`, keyed as it appears below
(`mutation createShipment`, `POST /api/v1/shipments/:shipmentID/cancel/`),
and takes exactly one of:

- `tools: [cancel_shipment]`: the agent tools that perform it. A tool
  named here must be registered.
- `exempt: <category>` with `reason: <why no agent should>`: one of
  the categories below. The reason is required.
- `pending: <what the tool would do>`: no tool yet. Pending writes are the
  backlog; they are counted, never failed.

The generator refuses, and `TestCoverageIsCurrent` and the `Agent write coverage`
step of `Codegen Checks` fail, when a write has no entry, an entry names a
write the app no longer exposes, an entry names a tool that does not exist, an
exemption has no reason or an unknown category, or this page differs from what
the generator writes. Each message names the key and the file to edit. After
editing the file, run `task generate-write-coverage` (or the command above) and
commit this page; `task generate-write-coverage-check` runs the CI check.

## How writes are found

- **GraphQL.** Every field of the `Mutation` type in
  `internal/api/graphql/schema/*.graphqls`, parsed with gqlparser. The
  domain is the schema file.
- **REST.** The gin route table itself: `api.RouteTable` registers every
  handler with zero-valued dependencies and reads back what gin holds, so a
  route cannot be missed by a parser. Routes one handler function serves (a
  trailing-slash alias, a PUT and a PATCH on one method) are one write, keyed
  by its shortest route. The domain is the handler package.
- **Twins.** A route is merged into a mutation, and needs no entry of its own,
  when the two reach exactly the same set of service methods (reads such as
  `Get` and `List` are ignored), found by walking the handler and the
  resolver with go/ast through their helper methods. When several mutations
  match, the one named for the handler method wins (`patch` to
  `patchTractor`); when that still leaves several, both are listed.
- **Blind spots.** A handler registered as a closure (`h.review(h.service.Approve)`)
  or one that passes a service method as a value is not merged with its twin;
  a route whose registration depends on configuration would be missed if the
  zero-valued configuration turns it off; a write reached only by a background
  job, an EDI message or an inbound webhook is not a write a person makes and is
  not listed.

## Exemption categories

| Category | Means | Writes |
| --- | --- | --- |
| `security` | Sign-in, sessions, passwords, API keys, SSO, identity providers, roles, permissions and every other grant of access. Never agent-operated: an agent that could widen access could widen its own. | 58 |
| `configuration` | Organization-wide settings, controls, lookup tables, templates and integration connections an administrator sets once and every later write depends on. | 206 |
| `user-preference` | A person's own interface state: saved table views, the sidebar, favorites, notification read state, a profile picture. | 23 |
| `infrastructure` | Plumbing a client, a provider or the platform drives rather than a decision a person makes: upload sessions, inbound webhooks, presence signals, the GraphQL transport, repair operations. | 28 |
| `agent-administration` | Defining, configuring, evaluating and overseeing agents, including deciding what they propose. An agent that did this would be grading its own work. | 42 |
| `counterparty` | Done by someone other than the organization's staff acting for themselves: a driver in their own portal, a customer or carrier through a public link. An agent acts for the organization and must not act as them. | 33 |
| `read-only` | Sent as a POST or a mutation but only computes, previews, validates or tests, and changes nothing. | 46 |
| `attestation` | A sign-off a named, accountable person must make: certifying a regulatory summary, filing a return, overriding a failed vetting. | 8 |
| `duplicate` | Another surface for a write listed elsewhere that the analysis could not merge on its own. The reason names the write it duplicates. | 0 |

## Totals

912 writes: 465 GraphQL mutations and 447 REST writes, after merging 68 REST routes into the mutation they duplicate.

| Decision | Writes |
| --- | --- |
| Covered by a tool | 73 |
| Exempt | 444 |
| — Security | 58 |
| — Configuration | 206 |
| — User preference | 23 |
| — Infrastructure | 28 |
| — Agent administration | 42 |
| — Counterparty | 33 |
| — Read-only | 46 |
| — Attestation | 8 |
| **Pending** | **395** |
| Total | 912 |

Of the 468 writes an agent should be able to make, 73 have a tool (15%).

## Pending

The writes no tool performs yet, and what the tool would do.

| Domain | Write | What the tool would do |
| --- | --- | --- |
| accountingsync | `mutation changeAccountingBackfill` | Change the range or scope of an accounting backfill that has not finished. |
| accountingsync | `mutation confirmAccountingMappings` | Confirm the suggested accounting mappings so sync can use them. |
| accountingsync | `mutation rejectAccountingMapping` | Reject a suggested accounting mapping. |
| accountingsync | `mutation releaseAccountingSync` | Release accounting sync records held for review so they post. |
| accounttype | `PATCH /api/v1/account-types/:accountTypeID/` | Update some fields of an account type. |
| accounttype | `POST /api/v1/account-types/` | Create an account type. |
| accounttype | `POST /api/v1/account-types/bulk-update-status/` | Change the status of several account types at once. |
| accounttype | `PUT /api/v1/account-types/:accountTypeID/` | Update an account type. |
| bankreceipt | `POST /api/v1/accounting/bank-receipts/` | Import a bank receipt line so it can be matched to open invoices. |
| bankreceiptbatch | `POST /api/v1/accounting/bank-receipt-batches/` | Import a batch of bank receipts from a bank file. |
| bankreceiptworkitem | `POST /api/v1/accounting/bank-receipt-work-items/:workItemID/assign/` | Assign a bank receipt work item to a person. |
| bankreceiptworkitem | `POST /api/v1/accounting/bank-receipt-work-items/:workItemID/start-review/` | Mark a bank receipt work item as under review. |
| benefits | `mutation endBenefitEnrollment` | End benefit enrollment. |
| benefits | `mutation enrollBenefit` | Enroll benefit. |
| billingqueue | `POST /api/v1/billing-queue/:itemID/reassign-charge/` | Move a charge from one billing queue item to another. |
| billingtransfer | `mutation cancelBillingTransferRun` | Cancel billing transfer run. |
| billingtransfer | `mutation retryBillingTransferRun` | Retry billing transfer run. |
| carrier | `PATCH /api/v1/carriers/:carrierID/` | Update some fields of a carrier. |
| carrier | `POST /api/v1/carriers/` | Create a carrier. |
| carrier | `POST /api/v1/carriers/bulk-update-status/` | Change the status of several carriers at once. |
| carrier | `PUT /api/v1/carriers/:carrierID/` | Update a carrier. |
| carrierintelligence | `mutation applyCarrierIntelSuggestions` | Apply the carrier profile corrections carrier intelligence suggested. |
| carrierintelligence | `mutation importSourcedCarrier` | Import sourced carrier. |
| carrierintelligence | `mutation markCarrierIntelReviewed` | Mark carrier intel reviewed. |
| carrierintelligence | `mutation resumeCarrierIntelMonitoring` | Resume carrier intel monitoring. |
| carrierintelligence | `mutation setCarrierMonitoring` | Set carrier monitoring. |
| carrierintelligence | `mutation verifyCarrierEquipment` | Verify carrier equipment. |
| carrierintelligence | `mutation vetCarrier` | Vet carrier. |
| carrierintelligence | `mutation vetCustomerBroker` | Vet customer broker. |
| carriersettlement | `mutation acceptCarrierInvoiceMatch` | Accept carrier invoice match. |
| carriersettlement | `mutation acceptCarrierInvoiceMatchWithVariance` | Accept carrier invoice match with variance. |
| carriersettlement | `mutation addCarrierSettlementAdjustment` | Add carrier settlement adjustment. |
| carriersettlement | `mutation approveCarrierSettlement` | Approve carrier settlement. |
| carriersettlement | `mutation createCarrierInvoiceMatch` | Create carrier invoice match. |
| carriersettlement | `mutation generateCarrierSettlementBatch` | Generate carrier settlement batch. |
| carriersettlement | `mutation linkEdiCarrierInvoiceToCarrier` | Link EDI carrier invoice to carrier. |
| carriersettlement | `mutation markCarrierSettlementPaid` | Mark carrier settlement paid. |
| carriersettlement | `mutation postCarrierSettlement` | Post carrier settlement. |
| carriersettlement | `mutation recalculateCarrierSettlement` | Recalculate carrier settlement. |
| carriersettlement | `mutation rejectCarrierInvoiceMatch` | Reject carrier invoice match. |
| carriersettlement | `mutation rejectCarrierSettlement` | Reject carrier settlement. |
| carriersettlement | `mutation removeCarrierSettlementAdjustment` | Remove carrier settlement adjustment. |
| carriersettlement | `mutation submitCarrierSettlement` | Submit carrier settlement. |
| carriersettlement | `mutation voidCarrierSettlement` | Void carrier settlement. |
| commodity | `PATCH /api/v1/commodities/:commodityID/` | Update some fields of a commodity. |
| commodity | `POST /api/v1/commodities/` | Create a commodity. |
| commodity | `POST /api/v1/commodities/bulk-update-status/` | Change the status of several commodities at once. |
| commodity | `PUT /api/v1/commodities/:commodityID/` | Update a commodity. |
| customer | `PATCH /api/v1/customers/:customerID/` | Update some fields of a customer. |
| customer | `POST /api/v1/customers/` | Create a customer. |
| customer | `POST /api/v1/customers/bulk-update-status/` | Change the status of several customers at once. |
| customer | `PUT /api/v1/customers/:customerID/` | Update a customer. |
| customerpayment | `mutation applyCreditMemo` | Apply credit memo. |
| customerpayment | `mutation applyUnappliedCustomerPayment` | Apply unapplied customer payment. |
| customerpayment | `mutation reverseCustomerPayment` | Reverse customer payment. |
| customerpayment | `mutation unapplyCreditMemoApplication` | Unapply credit memo application. |
| detention | `mutation disputeDetentionOccurrence` | Dispute detention occurrence. |
| distanceoverride | `DELETE /api/v1/distance-overrides/:distanceOverrideID/` | Delete a distance override. |
| distanceoverride | `PATCH /api/v1/distance-overrides/:distanceOverrideID/` | Update some fields of a distance override. |
| distanceoverride | `POST /api/v1/distance-overrides/` | Create a distance override. |
| distanceoverride | `PUT /api/v1/distance-overrides/:distanceOverrideID/` | Update a distance override. |
| document | `DELETE /api/v1/documents/:documentID/` | Delete a document. |
| document | `POST /api/v1/documents/:documentID/restore/` | Restore an earlier version of a document. |
| document | `POST /api/v1/documents/:documentID/shipment-draft/reextract/` | Run shipment extraction again on a document and replace the draft. |
| document | `POST /api/v1/documents/bulk-delete/` | Delete several documents at once. |
| driverportal | `mutation resolveSettlementDispute` | Resolve settlement dispute. |
| driverportal | `mutation reviewDriverExpense` | Review driver expense. |
| driverportal | `mutation startSettlementDisputeReview` | Start settlement dispute review. |
| driversettlement | `mutation addDriverSettlementAdjustment` | Add driver settlement adjustment. |
| driversettlement | `mutation adjustEscrowAccount` | Adjust escrow account. |
| driversettlement | `mutation approveDriverSettlement` | Approve driver settlement. |
| driversettlement | `mutation assignPayProfileToWorker` | Assign pay profile to worker. |
| driversettlement | `mutation attachPayEventsToSettlement` | Attach pay events to a driver settlement. |
| driversettlement | `mutation bulkDriverSettlementAction` | Approve, post or void several driver settlements at once. |
| driversettlement | `mutation closeEscrowAccount` | Close escrow account. |
| driversettlement | `mutation createPayCode` | Create pay code. |
| driversettlement | `mutation createPayProfile` | Create pay profile. |
| driversettlement | `mutation createRecurringDeduction` | Create recurring deduction. |
| driversettlement | `mutation createRecurringEarning` | Create recurring earning. |
| driversettlement | `mutation detachPayEventFromSettlement` | Detach a pay event from a driver settlement. |
| driversettlement | `mutation endWorkerPayAssignment` | End worker pay assignment. |
| driversettlement | `mutation generateDriverSettlement` | Generate driver settlement. |
| driversettlement | `mutation generateSettlementBatch` | Generate settlement batch. |
| driversettlement | `mutation holdDriverPayEvent` | Hold driver pay event. |
| driversettlement | `mutation issuePayAdvance` | Issue pay advance. |
| driversettlement | `mutation markDriverSettlementPaid` | Mark driver settlement paid. |
| driversettlement | `mutation openEscrowAccount` | Open escrow account. |
| driversettlement | `mutation payWorkerNow` | Pay a worker off cycle now. |
| driversettlement | `mutation postDriverSettlement` | Post driver settlement. |
| driversettlement | `mutation recalculateDriverSettlement` | Recalculate driver settlement. |
| driversettlement | `mutation rejectDriverSettlement` | Reject driver settlement. |
| driversettlement | `mutation releaseDriverPayEvent` | Release driver pay event. |
| driversettlement | `mutation removeDriverSettlementAdjustment` | Remove driver settlement adjustment. |
| driversettlement | `mutation submitDriverSettlement` | Submit driver settlement. |
| driversettlement | `mutation updateEscrowAccount` | Update escrow account. |
| driversettlement | `mutation updatePayCode` | Update pay code. |
| driversettlement | `mutation updatePayProfile` | Update pay profile. |
| driversettlement | `mutation updateRecurringDeduction` | Update recurring deduction. |
| driversettlement | `mutation updateRecurringEarning` | Update recurring earning. |
| driversettlement | `mutation voidDriverSettlement` | Void driver settlement. |
| driversettlement | `mutation writeOffPayAdvance` | Write off pay advance. |
| edi | `POST /api/v1/edi/documents/generate/` | Generate an outbound EDI document (a 214, a 210) for a record. |
| edi | `POST /api/v1/edi/inbound-files/:fileID/reprocess/` | Process a failed inbound EDI file again. |
| edi | `POST /api/v1/edi/inbound-files/bulk-reprocess/` | Process several failed inbound EDI files again. |
| edi | `POST /api/v1/edi/load-tenders/` | Send a load tender to a trading partner over EDI. |
| edi | `POST /api/v1/edi/messages/:messageID/replay/` | Send an EDI message again to its partner. |
| edi | `POST /api/v1/edi/messages/:messageID/retry-delivery/` | Retry delivery of an EDI message that failed to send. |
| edi | `POST /api/v1/edi/messages/bulk-retry-delivery/` | Retry delivery of several EDI messages that failed to send. |
| edi | `POST /api/v1/edi/tender-changes/:changeID/apply/` | Apply a change a trading partner sent to a tendered load. |
| edi | `POST /api/v1/edi/tender-changes/:changeID/reject/` | Reject a change a trading partner sent to a tendered load. |
| edi | `POST /api/v1/edi/transfer-changes/:changeID/apply/` | Apply a change to an inbound EDI transfer. |
| edi | `POST /api/v1/edi/transfer-changes/:changeID/reject/` | Reject a change to an inbound EDI transfer. |
| edi | `POST /api/v1/edi/transfers/:transferID/approve/` | Accept an inbound EDI load tender and create the shipment. |
| edi | `POST /api/v1/edi/transfers/:transferID/cancel/` | Cancel an inbound EDI transfer. |
| edi | `POST /api/v1/edi/transfers/:transferID/expire/` | Expire an inbound EDI transfer that was not answered in time. |
| edi | `POST /api/v1/edi/transfers/:transferID/reject/` | Decline an inbound EDI load tender. |
| edi | `POST /api/v1/edi/transfers/bulk-approve/` | Accept several inbound EDI load tenders at once. |
| edi | `POST /api/v1/edi/transfers/bulk-reject/` | Decline several inbound EDI load tenders at once. |
| exchangerate | `POST /api/v1/exchange-rates/settlement-quotes` | Lock an exchange rate quote for settling a foreign-currency payment. |
| fiscalperiod | `DELETE /api/v1/fiscal-periods/:fiscalPeriodID/` | Delete a fiscal period. |
| fiscalperiod | `PATCH /api/v1/fiscal-periods/:fiscalPeriodID/` | Update some fields of a fiscal period. |
| fiscalperiod | `POST /api/v1/fiscal-periods/` | Create a fiscal period. |
| fiscalperiod | `PUT /api/v1/fiscal-periods/:fiscalPeriodID/` | Update a fiscal period. |
| fiscalperiod | `PUT /api/v1/fiscal-periods/:fiscalPeriodID/activate/` | Activate a fiscal period. |
| fiscalperiod | `PUT /api/v1/fiscal-periods/:fiscalPeriodID/close/` | Close a fiscal period. |
| fiscalperiod | `PUT /api/v1/fiscal-periods/:fiscalPeriodID/lock/` | Lock a fiscal period. |
| fiscalperiod | `PUT /api/v1/fiscal-periods/:fiscalPeriodID/reopen/` | Reopen a fiscal period. |
| fiscalperiod | `PUT /api/v1/fiscal-periods/:fiscalPeriodID/unlock/` | Unlock a fiscal period. |
| fiscalyear | `DELETE /api/v1/fiscal-years/:fiscalYearID/` | Delete a fiscal year. |
| fiscalyear | `PATCH /api/v1/fiscal-years/:fiscalYearID/` | Update some fields of a fiscal year. |
| fiscalyear | `POST /api/v1/fiscal-years/` | Create a fiscal year. |
| fiscalyear | `PUT /api/v1/fiscal-years/:fiscalYearID/` | Update a fiscal year. |
| fiscalyear | `PUT /api/v1/fiscal-years/:fiscalYearID/activate/` | Activate a fiscal year. |
| fiscalyear | `PUT /api/v1/fiscal-years/:fiscalYearID/close/` | Close a fiscal year. |
| fiscalyear | `PUT /api/v1/fiscal-years/:fiscalYearID/reopen/` | Reopen a fiscal year. |
| fleetsafety | `mutation deleteSafetyViolation` | Delete safety violation. |
| fleetsafety | `mutation recordSafetyViolation` | Record safety violation. |
| fleetsafety | `mutation updateSafetyViolation` | Update safety violation. |
| fuelpurchase | `mutation assignFuelCard` | Assign fuel card. |
| fuelpurchase | `mutation cancelFuelCard` | Cancel fuel card. |
| fuelpurchase | `mutation commitFuelPurchaseImport` | Commit fuel purchase import. |
| fuelpurchase | `mutation createFuelCard` | Create fuel card. |
| fuelpurchase | `mutation createFuelPurchase` | Create fuel purchase. |
| fuelpurchase | `mutation createFuelPurchaseImport` | Create fuel purchase import. |
| fuelpurchase | `mutation deleteFuelPurchase` | Delete fuel purchase. |
| fuelpurchase | `mutation discardFuelPurchaseImport` | Discard fuel purchase import. |
| fuelpurchase | `mutation resolveFuelPurchaseImportRows` | Resolve fuel purchase import rows. |
| fuelpurchase | `mutation stageFuelPurchaseImport` | Stage fuel purchase import. |
| fuelpurchase | `mutation syncFuelCardFeed` | Sync fuel card feed. |
| fuelpurchase | `mutation updateFuelCard` | Update fuel card. |
| fuelpurchase | `mutation updateFuelPurchase` | Update fuel purchase. |
| fuelsurcharge | `mutation addFuelIndexPrice` | Add fuel index price. |
| fuelsurcharge | `mutation deleteFuelIndexPrice` | Delete fuel index price. |
| fuelsurcharge | `mutation updateFuelIndexPrice` | Update fuel index price. |
| glaccount | `DELETE /api/v1/gl-accounts/:glAccountID/` | Delete a gl account. |
| glaccount | `PATCH /api/v1/gl-accounts/:glAccountID/` | Update some fields of a gl account. |
| glaccount | `POST /api/v1/gl-accounts/` | Create a gl account. |
| glaccount | `POST /api/v1/gl-accounts/bulk-update-status/` | Change the status of several gl accounts at once. |
| glaccount | `PUT /api/v1/gl-accounts/:glAccountID/` | Update a gl account. |
| hazardousmaterial | `PATCH /api/v1/hazardous-materials/:hazardousMaterialID/` | Update some fields of a hazardous material. |
| hazardousmaterial | `POST /api/v1/hazardous-materials/` | Create a hazardous material. |
| hazardousmaterial | `POST /api/v1/hazardous-materials/bulk-update-status/` | Change the status of several hazardous materials at once. |
| hazardousmaterial | `PUT /api/v1/hazardous-materials/:hazardousMaterialID/` | Update a hazardous material. |
| ifta | `mutation amendIftaReturn` | Amend IFTA return. |
| ifta | `mutation backfillJurisdictionMiles` | Backfill jurisdiction miles. |
| ifta | `mutation createIftaMileageEntry` | Create IFTA mileage entry. |
| ifta | `mutation deleteIftaMileageEntry` | Delete IFTA mileage entry. |
| ifta | `mutation deleteIftaReturn` | Delete IFTA return. |
| ifta | `mutation deleteIftaTaxRate` | Delete IFTA tax rate. |
| ifta | `mutation generateIftaReturn` | Generate IFTA return. |
| ifta | `mutation recalculateMoveJurisdictionMiles` | Recalculate move jurisdiction miles. |
| ifta | `mutation recomputeIftaReturn` | Recompute IFTA return. |
| ifta | `mutation reopenIftaReturn` | Reopen IFTA return. |
| ifta | `mutation updateIftaMileageEntry` | Update IFTA mileage entry. |
| ifta | `mutation upsertIftaTaxRates` | Upsert IFTA tax rates. |
| insight | `POST /api/v1/insights/:insightID/restore/` | Restore an insight that was dismissed. |
| integration | `POST /api/v1/integrations/samsara/workers/sync/` | Sync workers from the telematics provider now. |
| integration | `POST /api/v1/integrations/samsara/workers/sync/drift/repair/` | Repair the drift found between workers and the telematics provider. |
| invoice | `mutation createInvoiceFromOrder` | Create invoice from order. |
| invoice | `mutation createInvoiceFromShipments` | Create invoice from shipments. |
| invoice | `mutation createInvoicesFromOrder` | Create invoices from order. |
| invoice | `mutation createInvoicesFromShipments` | Create invoices from shipments. |
| invoice | `mutation createMemo` | Create memo. |
| invoice | `mutation sendInvoiceEdi` | Send invoice EDI. |
| invoice | `mutation voidInvoice` | Void invoice. |
| invoice | `PATCH /api/v1/billing/invoices/:invoiceID/` | Update draft (invoice). |
| invoice | `POST /api/v1/billing/invoices/:invoiceID/generate-pdf/` | Generate pdf (invoice). |
| invoiceadjustment | `mutation approveInvoiceAdjustment` | Approve invoice adjustment. |
| invoiceadjustment | `mutation rejectInvoiceAdjustment` | Reject invoice adjustment. |
| invoiceadjustment | `PATCH /api/v1/billing/invoice-adjustments/drafts/:adjustmentID/` | Update draft (invoice adjustment). |
| invoiceadjustment | `POST /api/v1/billing/invoice-adjustments/bulk-submit/` | Bulk submit (invoice adjustment). |
| invoiceadjustment | `POST /api/v1/billing/invoice-adjustments/drafts/` | Create draft (invoice adjustment). |
| invoiceadjustment | `POST /api/v1/billing/invoice-adjustments/drafts/:adjustmentID/submit/` | Submit draft (invoice adjustment). |
| invoiceadjustment | `POST /api/v1/billing/invoice-adjustments/submit/` | Submit an invoice adjustment. |
| invoicedispute | `mutation openInvoiceDispute` | Open invoice dispute. |
| invoicedispute | `mutation resolveInvoiceDispute` | Resolve invoice dispute. |
| invoicedispute | `mutation withdrawInvoiceDispute` | Withdraw invoice dispute. |
| invoicerun | `PATCH /api/v1/billing/invoice-runs/:runID/membership/` | Adjust membership (invoice run). |
| invoicerun | `POST /api/v1/billing/invoice-runs/:runID/cancel/` | Cancel an invoice run. |
| invoicerun | `POST /api/v1/billing/invoice-runs/:runID/commit/` | Commit an invoice run. |
| invoicerun | `POST /api/v1/billing/invoice-runs/preview/` | Create an invoice run in preview for review before committing it. |
| invoicerun | `POST /api/v1/billing/statements/:customerID/bill/` | Bill statement (statement). |
| invoiceshare | `POST /api/v1/billing/invoices/:invoiceID/shares/` | Share an invoice. |
| journalreversal | `POST /api/v1/accounting/journal-reversals/` | Create a journal reversal. |
| journalreversal | `POST /api/v1/accounting/journal-reversals/:reversalID/approve/` | Approve a journal reversal. |
| journalreversal | `POST /api/v1/accounting/journal-reversals/:reversalID/cancel/` | Cancel a journal reversal. |
| journalreversal | `POST /api/v1/accounting/journal-reversals/:reversalID/post/` | Post a journal reversal. |
| journalreversal | `POST /api/v1/accounting/journal-reversals/:reversalID/reject/` | Reject a journal reversal. |
| latecharge | `mutation assessLateCharges` | Assess late charges. |
| location | `PATCH /api/v1/locations/:locationID/` | Update some fields of a location. |
| location | `POST /api/v1/locations/bulk-update-status/` | Change the status of several locations at once. |
| location | `PUT /api/v1/locations/:locationID/` | Update a location. |
| manualjournal | `POST /api/v1/accounting/manual-journals/:requestID/approve/` | Approve a manual journal. |
| manualjournal | `POST /api/v1/accounting/manual-journals/:requestID/cancel/` | Cancel a manual journal. |
| manualjournal | `POST /api/v1/accounting/manual-journals/:requestID/post/` | Post a manual journal. |
| manualjournal | `POST /api/v1/accounting/manual-journals/:requestID/reject/` | Reject a manual journal. |
| manualjournal | `POST /api/v1/accounting/manual-journals/:requestID/submit/` | Submit a manual journal. |
| manualjournal | `POST /api/v1/accounting/manual-journals/drafts/` | Create draft (manual journal). |
| manualjournal | `PUT /api/v1/accounting/manual-journals/drafts/:requestID/` | Update draft (manual journal). |
| order | `mutation addOrderCharge` | Add order charge. |
| order | `mutation attachOrderShipments` | Attach order shipments. |
| order | `mutation cancelOrder` | Cancel order. |
| order | `mutation closeOrder` | Close order. |
| order | `mutation createOrder` | Create order. |
| order | `mutation detachOrderShipment` | Detach order shipment. |
| order | `mutation removeOrderCharge` | Remove order charge. |
| order | `mutation setOrderChargeAllocations` | Set order charge allocations. |
| order | `mutation updateOrder` | Update order. |
| order | `mutation updateOrderCharge` | Update order charge. |
| orgstructure | `mutation assignUserPosition` | Assign user position. |
| orgstructure | `mutation assignWorkerPosition` | Assign worker position. |
| performancereview | `mutation closePerformanceReview` | Close performance review. |
| performancereview | `mutation createPerformanceReview` | Create performance review. |
| performancereview | `mutation deletePerformanceReview` | Delete performance review. |
| performancereview | `mutation reopenPerformanceReview` | Reopen performance review. |
| performancereview | `mutation submitPerformanceReview` | Submit performance review. |
| performancereview | `mutation updatePerformanceReview` | Update performance review. |
| permit | `POST /api/v1/shipments/:shipmentID/permit-requirements/:requirementID/waive/` | Waive requirement (shipment). |
| permit | `POST /api/v1/shipments/:shipmentID/permits/` | Create permit (shipment). |
| permit | `PUT /api/v1/shipments/:shipmentID/permits/:permitID/` | Update permit (shipment). |
| ptopolicy | `mutation adjustWorkerPtoBalance` | Adjust worker PTO balance. |
| ptopolicy | `mutation assignWorkerPtoPolicy` | Assign worker PTO policy. |
| ptopolicy | `mutation endWorkerPtoPolicyAssignment` | End worker PTO policy assignment. |
| ptopolicy | `mutation runPtoAccrual` | Run PTO accrual. |
| rateagreement | `POST /api/v1/rate-agreements/` | Create a rate agreement. |
| rateagreement | `POST /api/v1/rate-agreements/:rateAgreementID/approve/` | Approve a rate agreement. |
| rateagreement | `POST /api/v1/rate-agreements/:rateAgreementID/archive/` | Archive a rate agreement. |
| rateagreement | `POST /api/v1/rate-agreements/:rateAgreementID/duplicate/` | Duplicate a rate agreement. |
| rateagreement | `POST /api/v1/rate-agreements/:rateAgreementID/reject/` | Reject a rate agreement. |
| rateagreement | `POST /api/v1/rate-agreements/:rateAgreementID/resume/` | Resume a rate agreement. |
| rateagreement | `POST /api/v1/rate-agreements/:rateAgreementID/rules/amend/` | Amend the rating rules of an active rate agreement. |
| rateagreement | `POST /api/v1/rate-agreements/:rateAgreementID/submit/` | Submit a rate agreement. |
| rateagreement | `POST /api/v1/rate-agreements/:rateAgreementID/suspend/` | Suspend a rate agreement. |
| rateagreement | `POST /api/v1/rate-agreements/rate-increase/apply/` | Apply rate increase (rate agreement). |
| rateagreement | `PUT /api/v1/rate-agreements/:rateAgreementID/` | Update a rate agreement. |
| rateimport | `POST /api/v1/rate-imports/:rateImportID/commit/` | Commit a rate import. |
| rateimport | `POST /api/v1/rate-imports/:rateImportID/discard/` | Discard a rate import. |
| ratematrix | `DELETE /api/v1/rate-matrices/:rateMatrixID/` | Delete a rate matrice. |
| ratematrix | `POST /api/v1/rate-matrices/` | Create a rate matrice. |
| ratematrix | `PUT /api/v1/rate-matrices/:rateMatrixID/` | Update a rate matrice. |
| ratematrix | `PUT /api/v1/rate-matrices/:rateMatrixID/cells/` | Replace cells (rate matrice). |
| ratesimulation | `POST /api/v1/rate-simulations/` | Run and save a rate simulation across past shipments. |
| ratezone | `DELETE /api/v1/rate-zones/:rateZoneID/` | Delete a rate zone. |
| ratezone | `POST /api/v1/rate-zones/` | Create a rate zone. |
| ratezone | `PUT /api/v1/rate-zones/:rateZoneID/` | Update a rate zone. |
| recurringshipment | `POST /api/v1/recurring-shipments/` | Create a recurring shipment. |
| recurringshipment | `POST /api/v1/recurring-shipments/:recurringShipmentID/generate/` | Generate a recurring shipment. |
| recurringshipment | `PUT /api/v1/recurring-shipments/:recurringShipmentID/` | Update a recurring shipment. |
| recurringshipment | `PUT /api/v1/recurring-shipments/:recurringShipmentID/status/` | Update status (recurring shipment). |
| report | `mutation cancelReportRun` | Cancel report run. |
| report | `mutation createReportView` | Create report view. |
| report | `mutation deleteReportDashboard` | Delete report dashboard. |
| report | `mutation deleteReportDefinition` | Delete report definition. |
| report | `mutation deleteReportSchedule` | Delete report schedule. |
| report | `mutation deleteReportView` | Delete report view. |
| report | `mutation resetCannedFork` | Reset canned fork. |
| report | `mutation updateReportSchedule` | Update report schedule. |
| report | `mutation updateReportView` | Update report view. |
| routingguide | `DELETE /api/v1/routing-guides/:guideID/` | Delete a routing guide. |
| routingguide | `POST /api/v1/routing-guides/` | Create a routing guide. |
| routingguide | `PUT /api/v1/routing-guides/:guideID/` | Update a routing guide. |
| scheduling | `mutation assignWorkerShift` | Assign worker shift. |
| scheduling | `mutation endWorkerShiftAssignment` | End worker shift assignment. |
| scheduling | `mutation proposeShiftSwap` | Propose shift swap. |
| scheduling | `mutation setWorkerAvailabilityPreference` | Set worker availability preference. |
| scheduling | `mutation transitionShiftSwap` | Transition shift swap. |
| selfservice | `mutation decideProfileChange` | Decide profile change. |
| servicefailure | `PATCH /api/v1/service-failures/:serviceFailureID/` | Update a service failure. |
| servicefailure | `POST /api/v1/service-failures/` | Create manual (service failure). |
| servicefailure | `POST /api/v1/service-failures/:serviceFailureID/review/` | Review a service failure. |
| servicefailure | `POST /api/v1/service-failures/:serviceFailureID/void/` | Void a service failure. |
| servicefailure | `POST /api/v1/service-failures/bulk-evaluate/` | Evaluate several shipments for service failures at once. |
| servicefailure | `POST /api/v1/service-failures/evaluate-stop/:shipmentID/:stopID/` | Evaluate one stop for a service failure. |
| shipment | `mutation acknowledgeShipmentComment` | Acknowledge shipment comment. |
| shipment | `mutation autoRateShipment` | Auto rate shipment. |
| shipment | `mutation deleteShipmentComment` | Delete shipment comment. |
| shipment | `mutation duplicateShipment` | Duplicate shipment. |
| shipment | `mutation pinShipmentComment` | Pin shipment comment. |
| shipment | `mutation recalculateShipmentDistance` | Recalculate shipment distance. |
| shipment | `mutation resolveShipmentComment` | Resolve shipment comment. |
| shipment | `mutation transferShipmentOwnership` | Transfer shipment ownership. |
| shipment | `mutation transferShipmentToBillingItems` | Transfer chosen shipment charges to billing as separate items. |
| shipment | `mutation uncancelShipment` | Uncancel shipment. |
| shipment | `mutation unpinShipmentComment` | Unpin shipment comment. |
| shipment | `mutation unresolveShipmentComment` | Unresolve shipment comment. |
| shipment | `mutation updateShipmentComment` | Update shipment comment. |
| shipment | `POST /api/v1/shipments/auto-cancel/` | Cancel shipments that passed the auto-cancel threshold. |
| shipment | `POST /api/v1/shipments/delay/` | Mark shipments as delayed. |
| shipment | `PUT /api/v1/shipments/:shipmentID/holds/:holdID/` | Update hold (shipment). |
| shipmentmove | `POST /api/v1/shipment-moves/:moveID/split/` | Split a two-stop move at a relay point into two moves. Left without a tool: the split needs a relay location and two new scheduled windows the system holds nowhere, so a model would have to invent the times, and the service checks only their order. |
| storedmileage | `DELETE /api/v1/stored-mileages/:storedMileageID/` | Delete a stored mileage. |
| tablechangealert | `DELETE /api/v1/tca/subscriptions/:id` | Delete subscription (table change alert). |
| tablechangealert | `PATCH /api/v1/tca/subscriptions/:id/pause` | Pause subscription (table change alert). |
| tablechangealert | `PATCH /api/v1/tca/subscriptions/:id/resume` | Resume subscription (table change alert). |
| tablechangealert | `PUT /api/v1/tca/subscriptions/:id` | Update subscription (table change alert). |
| timesheet | `mutation deleteTimeEntry` | Delete time entry. |
| timesheet | `mutation generatePayrollExport` | Generate payroll export. |
| timesheet | `mutation recordTimeEntry` | Record time entry. |
| timesheet | `mutation voidPayrollExport` | Void payroll export. |
| tractor | `mutation createTractor` | Create tractor. |
| tractor | `mutation locateTractor` | Locate tractor. |
| tractor | `mutation patchTractor` | Update some fields of a tractor. |
| tractor | `mutation updateTractor` | Update tractor. |
| trailer | `mutation createTrailer` | Create trailer. |
| trailer | `mutation locateTrailer` | Locate trailer. |
| trailer | `mutation patchTrailer` | Update some fields of a trailer. |
| trailer | `mutation updateTrailer` | Update trailer. |
| watchtower | `mutation dismissWatchtowerItem` | Dismiss watchtower item. |
| watchtower | `mutation handOffWatchtowerItem` | Hand off watchtower item. |
| worker | `mutation bulkWorkerPTOAction` | Approve, reject or cancel several PTO requests at once. |
| worker | `mutation createWorkerPTO` | Create worker PTO. |
| worker | `mutation patchWorker` | Update some fields of a worker. |
| worker | `mutation updateWorkerPTO` | Update worker PTO. |
| worker | `POST /api/v1/workers/` | Create a worker. |
| workerchecklist | `mutation cancelWorkerChecklist` | Cancel worker checklist. |
| workerchecklist | `mutation completeWorkerChecklistItem` | Complete worker checklist item. |
| workerchecklist | `mutation markWorkerChecklistItemNotApplicable` | Mark worker checklist item not applicable. |
| workerchecklist | `mutation reopenWorkerChecklistItem` | Reopen worker checklist item. |
| workerchecklist | `mutation skipWorkerChecklistItem` | Skip worker checklist item. |
| workerchecklist | `mutation startWorkerChecklist` | Start worker checklist. |
| workercredential | `mutation archiveWorkerCredential` | Archive worker credential. |
| workercredential | `mutation attachWorkerCredentialDocument` | Attach worker credential document. |
| workercredential | `mutation createWorkerCredential` | Create worker credential. |
| workercredential | `mutation updateWorkerCredential` | Update worker credential. |
| workercredential | `mutation verifyWorkerCredential` | Verify worker credential. |
| workerdqf | `mutation deleteEmploymentVerification` | Delete employment verification. |
| workerdqf | `mutation markEmploymentVerificationRequested` | Mark employment verification requested. |
| workerdqf | `mutation recordEmploymentVerification` | Record employment verification. |
| workerdqf | `mutation recordEmploymentVerificationFollowUp` | Record a follow-up on an employment verification request. |
| workerdqf | `mutation updateEmploymentVerification` | Update employment verification. |
| workerdrugalcohol | `mutation cancelDotRandomDraw` | Cancel DOT random draw. |
| workerdrugalcohol | `mutation cancelDotTest` | Cancel DOT test. |
| workerdrugalcohol | `mutation completeClearinghouseQuery` | Complete clearinghouse query. |
| workerdrugalcohol | `mutation createDotRandomPool` | Create DOT random pool. |
| workerdrugalcohol | `mutation finalizeDotRandomDraw` | Finalize DOT random draw. |
| workerdrugalcohol | `mutation recordClearinghouseQuery` | Record clearinghouse query. |
| workerdrugalcohol | `mutation recordDotTest` | Record DOT test. |
| workerdrugalcohol | `mutation recordDotTestResult` | Record DOT test result. |
| workerdrugalcohol | `mutation recordDotViolation` | Record DOT violation. |
| workerdrugalcohol | `mutation runDotRandomDraw` | Run DOT random draw. |
| workerdrugalcohol | `mutation updateDotRandomDrawEntry` | Update DOT random draw entry. |
| workerdrugalcohol | `mutation updateDotRandomPool` | Update DOT random pool. |
| workerdrugalcohol | `mutation updateDotViolation` | Update DOT violation. |
| workeremployment | `mutation amendWorkerEmploymentEvent` | Amend worker employment event. |
| workeremployment | `mutation recordWorkerEmploymentEvent` | Record worker employment event. |
| workerinjury | `mutation deleteWorkerInjury` | Delete worker injury. |
| workerinjury | `mutation recordWorkerInjury` | Record worker injury. |
| workerinjury | `mutation saveOshaSummary` | Save OSHA summary. |
| workerinjury | `mutation updateWorkerInjury` | Update worker injury. |
| workerleave | `mutation closeLeaveCase` | Close leave case. |
| workerleave | `mutation decideLeaveCase` | Decide leave case. |
| workerleave | `mutation deleteLeaveDay` | Delete leave day. |
| workerleave | `mutation openLeaveCase` | Open leave case. |
| workerleave | `mutation recordLeaveCertification` | Record leave certification. |
| workerleave | `mutation recordLeaveDay` | Record leave day. |
| workerleave | `mutation requestLeaveCertification` | Request leave certification. |
| workerleave | `mutation updateLeaveCase` | Update leave case. |
| workerleave | `mutation updateLeaveDay` | Update leave day. |
| workersafety | `mutation closeWorkerSafetyEvent` | Close worker safety event. |
| workersafety | `mutation createWorkerSafetyEvent` | Create worker safety event. |
| workersafety | `mutation deleteWorkerRecognition` | Delete worker recognition. |
| workersafety | `mutation deleteWorkerSafetyEvent` | Delete worker safety event. |
| workersafety | `mutation giveWorkerRecognition` | Give worker recognition. |
| workersafety | `mutation issueDisciplinaryAction` | Issue disciplinary action. |
| workersafety | `mutation reopenWorkerSafetyEvent` | Reopen worker safety event. |
| workersafety | `mutation rescindDisciplinaryAction` | Rescind disciplinary action. |
| workersafety | `mutation reviewWorkerSafetyEvent` | Review worker safety event. |
| workersafety | `mutation updateWorkerSafetyEvent` | Update worker safety event. |
| workertraining | `mutation assignRequiredWorkerTraining` | Assign required worker training. |
| workertraining | `mutation assignWorkerTraining` | Assign worker training. |
| workertraining | `mutation attachWorkerTrainingDocument` | Attach worker training document. |
| workertraining | `mutation bulkAssignTraining` | Bulk assign training. |
| workertraining | `mutation cancelWorkerTraining` | Cancel worker training. |
| workertraining | `mutation completeWorkerTraining` | Complete worker training. |
| workertraining | `mutation waiveWorkerTraining` | Waive worker training. |

## By domain

| Domain | Writes | Covered | Exempt | Pending |
| --- | --- | --- | --- | --- |
| accessorialcharge | 3 | 0 | 3 | 0 |
| accountingcontrol | 1 | 0 | 1 | 0 |
| accountingsync | 22 | 10 | 8 | 4 |
| accountingwebhook | 1 | 0 | 1 | 0 |
| accounttype | 4 | 0 | 0 | 4 |
| agent | 12 | 2 | 10 | 0 |
| agentdefinition | 6 | 0 | 6 | 0 |
| agentextension | 2 | 0 | 2 | 0 |
| agentquality | 6 | 0 | 6 | 0 |
| agentrun | 1 | 0 | 1 | 0 |
| aiaudit | 3 | 0 | 3 | 0 |
| aifeedback | 2 | 0 | 2 | 0 |
| aiprovider | 4 | 0 | 4 | 0 |
| airetrieval | 2 | 0 | 2 | 0 |
| apikey | 4 | 0 | 4 | 0 |
| assignment | 1 | 0 | 1 | 0 |
| assistant | 8 | 0 | 8 | 0 |
| auth | 6 | 0 | 6 | 0 |
| bankreceipt | 2 | 1 | 0 | 1 |
| bankreceiptbatch | 1 | 0 | 0 | 1 |
| bankreceiptworkitem | 4 | 2 | 0 | 2 |
| benefits | 4 | 0 | 2 | 2 |
| billingcontrol | 1 | 0 | 1 | 0 |
| billingqueue | 8 | 4 | 3 | 1 |
| billingtransfer | 3 | 1 | 0 | 2 |
| briefing | 2 | 0 | 2 | 0 |
| carrier | 4 | 0 | 0 | 4 |
| carrierintelligence | 15 | 2 | 5 | 8 |
| carriersettlement | 16 | 0 | 1 | 15 |
| commodity | 4 | 0 | 0 | 4 |
| controlplaneprovisioning | 1 | 0 | 1 | 0 |
| costing | 2 | 0 | 2 | 0 |
| customer | 4 | 0 | 0 | 4 |
| customerpayment | 5 | 1 | 0 | 4 |
| customfield | 4 | 0 | 4 | 0 |
| databasesession | 1 | 0 | 1 | 0 |
| dataentrycontrol | 1 | 0 | 1 | 0 |
| dataretention | 1 | 0 | 1 | 0 |
| decisions | 1 | 0 | 1 | 0 |
| detention | 8 | 3 | 4 | 1 |
| detentionpolicy | 1 | 0 | 1 | 0 |
| dispatchconsole | 5 | 4 | 1 | 0 |
| dispatchcontrol | 1 | 0 | 1 | 0 |
| distancecontrol | 2 | 0 | 2 | 0 |
| distanceoverride | 4 | 0 | 0 | 4 |
| distanceprofile | 5 | 0 | 5 | 0 |
| document | 13 | 1 | 8 | 4 |
| documentcontrol | 1 | 0 | 1 | 0 |
| documentoperations | 3 | 0 | 3 | 0 |
| documentpacketrule | 3 | 0 | 3 | 0 |
| documentparsingrule | 9 | 0 | 9 | 0 |
| documenttemplate | 12 | 0 | 12 | 0 |
| documenttype | 3 | 0 | 3 | 0 |
| driverportal | 31 | 0 | 28 | 3 |
| driversettlement | 34 | 0 | 1 | 33 |
| edi | 56 | 0 | 39 | 17 |
| email | 9 | 0 | 9 | 0 |
| equipmentmanufacturer | 4 | 0 | 4 | 0 |
| equipmenttype | 4 | 0 | 4 | 0 |
| exchangerate | 2 | 0 | 1 | 1 |
| fiscalperiod | 9 | 0 | 0 | 9 |
| fiscalyear | 7 | 0 | 0 | 7 |
| fleetcode | 3 | 0 | 3 | 0 |
| fleetsafety | 3 | 0 | 0 | 3 |
| formulatemplate | 24 | 0 | 24 | 0 |
| fuelpurchase | 13 | 0 | 0 | 13 |
| fuelsurcharge | 9 | 0 | 6 | 3 |
| glaccount | 5 | 0 | 0 | 5 |
| googlemaps | 1 | 0 | 1 | 0 |
| graphql | 1 | 0 | 1 | 0 |
| hazardousmaterial | 4 | 0 | 0 | 4 |
| hazmatsegregationrule | 3 | 0 | 3 | 0 |
| holdreason | 3 | 0 | 3 | 0 |
| homelayout | 5 | 1 | 4 | 0 |
| iam | 14 | 0 | 14 | 0 |
| ifta | 14 | 0 | 2 | 12 |
| inbound | 1 | 0 | 1 | 0 |
| inboundmessage | 6 | 2 | 4 | 0 |
| insight | 2 | 1 | 0 | 1 |
| integration | 5 | 0 | 3 | 2 |
| invoice | 12 | 2 | 1 | 9 |
| invoiceadjustment | 10 | 0 | 3 | 7 |
| invoiceadjustmentcontrol | 1 | 0 | 1 | 0 |
| invoicedispute | 3 | 0 | 0 | 3 |
| invoicerun | 5 | 0 | 0 | 5 |
| invoiceshare | 1 | 0 | 0 | 1 |
| journalreversal | 5 | 0 | 0 | 5 |
| jurisdictionrule | 6 | 0 | 6 | 0 |
| latecharge | 1 | 0 | 0 | 1 |
| location | 4 | 1 | 0 | 3 |
| locationcategory | 3 | 0 | 3 | 0 |
| manualjournal | 7 | 0 | 0 | 7 |
| notification | 5 | 0 | 5 | 0 |
| order | 10 | 0 | 0 | 10 |
| organization | 4 | 0 | 4 | 0 |
| orgholiday | 3 | 0 | 3 | 0 |
| orgstructure | 6 | 0 | 4 | 2 |
| pagefavorite | 1 | 0 | 1 | 0 |
| performancereview | 11 | 0 | 5 | 6 |
| permission | 1 | 0 | 1 | 0 |
| permit | 3 | 0 | 0 | 3 |
| ptopolicy | 8 | 0 | 4 | 4 |
| push | 2 | 0 | 2 | 0 |
| rateagreement | 12 | 0 | 1 | 11 |
| rateconfirmation | 4 | 4 | 0 | 0 |
| rateconfirmationpublic | 1 | 0 | 1 | 0 |
| rateimport | 3 | 0 | 1 | 2 |
| ratematrix | 4 | 0 | 0 | 4 |
| ratequote | 3 | 0 | 3 | 0 |
| ratesimulation | 1 | 0 | 0 | 1 |
| ratezone | 3 | 0 | 0 | 3 |
| realtime | 3 | 0 | 3 | 0 |
| recurringshipment | 5 | 0 | 1 | 4 |
| report | 17 | 7 | 1 | 9 |
| role | 11 | 0 | 11 | 0 |
| routingguide | 3 | 0 | 0 | 3 |
| scheduling | 7 | 0 | 2 | 5 |
| selfservice | 3 | 0 | 2 | 1 |
| sequenceconfig | 1 | 0 | 1 | 0 |
| servicefailure | 9 | 2 | 1 | 6 |
| servicefailurereasoncode | 6 | 0 | 6 | 0 |
| servicetype | 4 | 0 | 4 | 0 |
| shipment | 33 | 8 | 9 | 16 |
| shipmentcontrol | 1 | 0 | 1 | 0 |
| shipmentmove | 4 | 3 | 0 | 1 |
| shipmenttype | 4 | 0 | 4 | 0 |
| sidebarpreference | 1 | 0 | 1 | 0 |
| storedmileage | 1 | 0 | 0 | 1 |
| tablechangealert | 5 | 1 | 0 | 4 |
| tableconfiguration | 6 | 1 | 5 | 0 |
| tablequery | 1 | 0 | 1 | 0 |
| telematics | 4 | 0 | 4 | 0 |
| tenant | 1 | 0 | 1 | 0 |
| tender | 4 | 4 | 0 | 0 |
| tenderpublic | 2 | 0 | 2 | 0 |
| timesheet | 7 | 0 | 3 | 4 |
| tractor | 5 | 1 | 0 | 4 |
| trailer | 5 | 1 | 0 | 4 |
| user | 11 | 0 | 11 | 0 |
| version | 1 | 0 | 1 | 0 |
| watchtower | 3 | 0 | 1 | 2 |
| worker | 8 | 3 | 0 | 5 |
| workerchecklist | 10 | 0 | 4 | 6 |
| workercredential | 9 | 0 | 4 | 5 |
| workerdqf | 5 | 0 | 0 | 5 |
| workerdrugalcohol | 13 | 0 | 0 | 13 |
| workeremployment | 2 | 0 | 0 | 2 |
| workerinjury | 6 | 0 | 2 | 4 |
| workerleave | 10 | 0 | 1 | 9 |
| workersafety | 11 | 0 | 1 | 10 |
| workertraining | 13 | 0 | 6 | 7 |

## Action tools no write maps to

Tools that change something no person-facing write does, such as sending a message or raising an exception for review.

| Tool | Resource | Operation |
| --- | --- | --- |
| `email_customer` | customer_communication | create |
| `escalate_detention` | detention_policy | update |
| `flag_for_manual_review` | agent_exception | create |
| `notify_driver` | driver_message | create |
| `place_worker_dispatch_hold` | worker_dispatch_hold | create |
| `raise_exception` | agent_exception | create |
| `reply_to_inbound_message` | customer_communication | create |
| `request_credential_renewal` | worker_credential | update |
| `request_missing_docs` | customer_communication | create |

## Every write

### accessorialcharge

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/accessorial-charges/:accessorialChargeID/`<br>accessorialchargehandler.patch | Exempt, configuration: The accessorial charge catalog is a price list an administrator maintains; charges on a shipment use it. |
| `POST /api/v1/accessorial-charges/`<br>accessorialchargehandler.create | Exempt, configuration: The accessorial charge catalog is a price list an administrator maintains; charges on a shipment use it. |
| `PUT /api/v1/accessorial-charges/:accessorialChargeID/`<br>accessorialchargehandler.update | Exempt, configuration: The accessorial charge catalog is a price list an administrator maintains; charges on a shipment use it. |

### accountingcontrol

| Write | Decision |
| --- | --- |
| `PUT /api/v1/accounting-controls/`<br>accountingcontrolhandler.update | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |

### accountingsync

| Write | Decision |
| --- | --- |
| `mutation changeAccountingBackfill` | Pending: Change the range or scope of an accounting backfill that has not finished. |
| `mutation checkAccountingConnection` | Tool: `check_accounting_connection` |
| `mutation clearAccountingMapping` | Tool: `clear_accounting_mapping` |
| `mutation completeAccountingAuthorization` | Exempt, security: Handles the OAuth app and its credentials that let Trenova act in the accounting system. |
| `mutation completeAccountingSetup` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |
| `mutation confirmAccountingMappings` | Pending: Confirm the suggested accounting mappings so sync can use them. |
| `mutation createAccountingReferenceRecord` | Tool: `create_accounting_reference_record` |
| `mutation disconnectAccountingSystem` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |
| `mutation enableAccountingSync` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |
| `mutation pauseAccountingSync` | Tool: `pause_accounting_sync` |
| `mutation refreshAccountingReferenceData` | Tool: `refresh_accounting_reference_data` |
| `mutation rejectAccountingMapping` | Pending: Reject a suggested accounting mapping. |
| `mutation releaseAccountingSync` | Pending: Release accounting sync records held for review so they post. |
| `mutation removeAccountingApp` | Exempt, security: Handles the OAuth app and its credentials that let Trenova act in the accounting system. |
| `mutation requestAccountingBackfill` | Tool: `request_accounting_backfill` |
| `mutation resumeAccountingSync` | Tool: `resume_accounting_sync` |
| `mutation retryAccountingSync` | Tool: `retry_accounting_sync` |
| `mutation saveAccountingApp` | Exempt, security: Handles the OAuth app and its credentials that let Trenova act in the accounting system. |
| `mutation setAccountingMapping` | Tool: `set_accounting_mapping` |
| `mutation skipAccountingSync` | Tool: `skip_accounting_sync` |
| `mutation startAccountingAuthorization` | Exempt, security: Handles the OAuth app and its credentials that let Trenova act in the accounting system. |
| `mutation updateAccountingSyncSettings` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |

### accountingwebhook

| Write | Decision |
| --- | --- |
| `POST /api/v1/webhooks/accounting/:provider/`<br>accountingwebhookhandler.receive | Exempt, infrastructure: An inbound webhook a provider calls with a signed token, not a person. |

### accounttype

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/account-types/:accountTypeID/`<br>accounttypehandler.patch | Pending: Update some fields of an account type. |
| `POST /api/v1/account-types/`<br>accounttypehandler.create | Pending: Create an account type. |
| `POST /api/v1/account-types/bulk-update-status/`<br>accounttypehandler.bulkUpdateStatus | Pending: Change the status of several account types at once. |
| `PUT /api/v1/account-types/:accountTypeID/`<br>accounttypehandler.update | Pending: Update an account type. |

### agent

| Write | Decision |
| --- | --- |
| `mutation approveAgentMemorySuggestion` | Exempt, agent-administration: A person curating what agents remember; an agent writes memory through remember and forget_memory. |
| `mutation createAgentMemory` | Tool: `remember` |
| `mutation decideAgentPlan`<br>twin `POST /api/v1/agent-plans/:planID/resolve/` | Exempt, agent-administration: Deciding what an agent proposed is the human check on agents; an agent cannot approve its own work. |
| `mutation decideAgentProposal`<br>twin `POST /api/v1/agent-proposals/:proposalID/resolve/` | Exempt, agent-administration: Deciding what an agent proposed is the human check on agents; an agent cannot approve its own work. |
| `mutation decideMyPlan` | Exempt, agent-administration: Deciding what an agent proposed is the human check on agents; an agent cannot approve its own work. |
| `mutation decideMyProposal` | Exempt, agent-administration: Deciding what an agent proposed is the human check on agents; an agent cannot approve its own work. |
| `mutation dismissAgentMemorySuggestion` | Exempt, agent-administration: A person curating what agents remember; an agent writes memory through remember and forget_memory. |
| `mutation replayAgentRun` | Exempt, agent-administration: Configures, evaluates or oversees the agents themselves; an agent doing it would be grading its own work. |
| `mutation resolveAgentException`<br>twin `POST /api/v1/agent-exceptions/:exceptionID/resolve/` | Exempt, agent-administration: An agent exception is an agent handing a case to a person; the person resolves it. |
| `mutation setAgentMemoryStatus` | Tool: `forget_memory` |
| `mutation updateAgentControl`<br>twin `PUT /api/v1/agent-controls/` | Exempt, agent-administration: The organization-wide switches and ceilings every agent runs under. |
| `mutation updateAgentMemory` | Exempt, agent-administration: A person curating what agents remember; an agent writes memory through remember and forget_memory. |

### agentdefinition

| Write | Decision |
| --- | --- |
| `mutation setAgentAccess` | Exempt, security: Decides which people may use which agents. |
| `mutation setRoleAgentAccess` | Exempt, security: Decides which roles may use which agents. |
| `DELETE /api/v1/agent-definitions/:agentID/`<br>agentdefinitionhandler.remove | Exempt, agent-administration: Configures, evaluates or oversees the agents themselves; an agent doing it would be grading its own work. |
| `POST /api/v1/agent-definitions/`<br>agentdefinitionhandler.create | Exempt, agent-administration: Configures, evaluates or oversees the agents themselves; an agent doing it would be grading its own work. |
| `POST /api/v1/agent-definitions/preview-prompt/`<br>agentdefinitionhandler.previewPrompt | Exempt, read-only: Renders an agent prompt for review and saves nothing. |
| `PUT /api/v1/agent-definitions/:agentID/`<br>agentdefinitionhandler.update | Exempt, agent-administration: Configures, evaluates or oversees the agents themselves; an agent doing it would be grading its own work. |

### agentextension

| Write | Decision |
| --- | --- |
| `POST /api/v1/agent-extensions/:type/test/`<br>agentextensionhandler.test | Exempt, read-only: Tests the vendor credentials and returns the result. |
| `PUT /api/v1/agent-extensions/:type/config/`<br>agentextensionhandler.updateConfig | Exempt, agent-administration: Configures a vendor account an organization turns on for its agents; it holds the vendor credentials. |

### agentquality

| Write | Decision |
| --- | --- |
| `mutation createAgentEvalCase` | Exempt, agent-administration: Configures, evaluates or oversees the agents themselves; an agent doing it would be grading its own work. |
| `mutation replayAgentEvalCase` | Exempt, agent-administration: Configures, evaluates or oversees the agents themselves; an agent doing it would be grading its own work. |
| `mutation runAgentSuite` | Exempt, agent-administration: Configures, evaluates or oversees the agents themselves; an agent doing it would be grading its own work. |
| `mutation setAgentEvalCaseStatus` | Exempt, agent-administration: Configures, evaluates or oversees the agents themselves; an agent doing it would be grading its own work. |
| `mutation updateAgentEvalCase` | Exempt, agent-administration: Configures, evaluates or oversees the agents themselves; an agent doing it would be grading its own work. |
| `mutation updateAgentQualityControl` | Exempt, agent-administration: Configures, evaluates or oversees the agents themselves; an agent doing it would be grading its own work. |

### agentrun

| Write | Decision |
| --- | --- |
| `POST /api/v1/agent-runs/`<br>agentrunhandler.start | Exempt, agent-administration: Starts an agent on demand; agents hand work to one another through delegate_task instead. |

### aiaudit

| Write | Decision |
| --- | --- |
| `mutation aiAuditExportDownload` | Exempt, read-only: Returns a download link for an export already produced. |
| `mutation requestAIAuditExport` | Exempt, agent-administration: The AI audit trail is how people check what agents did; an agent must not drive its own audit. |
| `mutation verifyAIAuditChain` | Exempt, agent-administration: The AI audit trail is how people check what agents did; an agent must not drive its own audit. |

### aifeedback

| Write | Decision |
| --- | --- |
| `mutation clearMyAIFeedback` | Exempt, agent-administration: A person's rating of an agent's answer; an agent rating itself would be meaningless. |
| `mutation setMyAIFeedback` | Exempt, agent-administration: A person's rating of an agent's answer; an agent rating itself would be meaningless. |

### aiprovider

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/ai-providers/:providerID/`<br>aiproviderhandler.remove | Exempt, agent-administration: Configures the model providers agents run on, including their API keys. |
| `POST /api/v1/ai-providers/`<br>aiproviderhandler.create | Exempt, agent-administration: Configures the model providers agents run on, including their API keys. |
| `POST /api/v1/ai-providers/:providerID/test/`<br>aiproviderhandler.test | Exempt, read-only: Tests the provider credentials and returns the result. |
| `PUT /api/v1/ai-providers/:providerID/`<br>aiproviderhandler.update | Exempt, agent-administration: Configures the model providers agents run on, including their API keys. |

### airetrieval

| Write | Decision |
| --- | --- |
| `mutation reindexAIRetrievalSource` | Exempt, agent-administration: Configures, evaluates or oversees the agents themselves; an agent doing it would be grading its own work. |
| `mutation updateAIRetrievalSettings` | Exempt, agent-administration: Configures, evaluates or oversees the agents themselves; an agent doing it would be grading its own work. |

### apikey

| Write | Decision |
| --- | --- |
| `POST /api/v1/api-keys/`<br>apikeyhandler.create | Exempt, security: API keys grant programmatic access to the organization; only a person issues, rotates or revokes one. |
| `POST /api/v1/api-keys/:apiKeyID/revoke/`<br>apikeyhandler.revoke | Exempt, security: API keys grant programmatic access to the organization; only a person issues, rotates or revokes one. |
| `POST /api/v1/api-keys/:apiKeyID/rotate/`<br>apikeyhandler.rotate | Exempt, security: API keys grant programmatic access to the organization; only a person issues, rotates or revokes one. |
| `PUT /api/v1/api-keys/:apiKeyID/`<br>apikeyhandler.update | Exempt, security: API keys grant programmatic access to the organization; only a person issues, rotates or revokes one. |

### assignment

| Write | Decision |
| --- | --- |
| `POST /api/v1/assignments/check-worker-compliance/`<br>assignmenthandler.checkWorkerCompliance | Exempt, read-only: Checks a worker against compliance rules before an assignment and saves nothing. |

### assistant

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/assistant/threads/:threadID/`<br>assistanthandler.deleteThread | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `PATCH /api/v1/assistant/threads/:threadID/`<br>assistanthandler.updateThread | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/assistant/ask/`<br>assistanthandler.ask | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/assistant/threads/`<br>assistanthandler.startThread | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/assistant/threads/:threadID/artifacts/:artifactID/pin/`<br>assistanthandler.pinArtifact | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/assistant/threads/:threadID/messages/`<br>assistanthandler.sendMessage | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/assistant/threads/:threadID/turns/`<br>assistanthandler.startTurn | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/assistant/turns/:turnID/stop/`<br>assistanthandler.stopTurn | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |

### auth

| Write | Decision |
| --- | --- |
| `POST /api/v1/auth/forgot-password`<br>authhandler.forgotPassword | Exempt, security: Signing in and out and resetting a password are a person proving who they are; an agent holds no credentials of its own. |
| `POST /api/v1/auth/login`<br>authhandler.login | Exempt, security: Signing in and out and resetting a password are a person proving who they are; an agent holds no credentials of its own. |
| `POST /api/v1/auth/logout`<br>authhandler.logout | Exempt, security: Signing in and out and resetting a password are a person proving who they are; an agent holds no credentials of its own. |
| `POST /api/v1/auth/reset-password`<br>authhandler.resetPassword | Exempt, security: Signing in and out and resetting a password are a person proving who they are; an agent holds no credentials of its own. |
| `POST /api/v1/auth/session/roles/activate`<br>authhandler.activateSessionRoles | Exempt, security: Signing in and out and resetting a password are a person proving who they are; an agent holds no credentials of its own. |
| `POST /api/v1/auth/validate-session`<br>authhandler.validateSession | Exempt, security: Signing in and out and resetting a password are a person proving who they are; an agent holds no credentials of its own. |

### bankreceipt

| Write | Decision |
| --- | --- |
| `POST /api/v1/accounting/bank-receipts/`<br>bankreceipthandler.importReceipt | Pending: Import a bank receipt line so it can be matched to open invoices. |
| `POST /api/v1/accounting/bank-receipts/:receiptID/match/`<br>bankreceipthandler.match | Tool: `match_bank_receipt` |

### bankreceiptbatch

| Write | Decision |
| --- | --- |
| `POST /api/v1/accounting/bank-receipt-batches/`<br>bankreceiptbatchhandler.importBatch | Pending: Import a batch of bank receipts from a bank file. |

### bankreceiptworkitem

| Write | Decision |
| --- | --- |
| `POST /api/v1/accounting/bank-receipt-work-items/:workItemID/assign/`<br>bankreceiptworkitemhandler.assign | Pending: Assign a bank receipt work item to a person. |
| `POST /api/v1/accounting/bank-receipt-work-items/:workItemID/dismiss/`<br>bankreceiptworkitemhandler.dismiss | Tool: `resolve_bank_receipt_work_item` |
| `POST /api/v1/accounting/bank-receipt-work-items/:workItemID/resolve/`<br>bankreceiptworkitemhandler.resolve | Tool: `resolve_bank_receipt_work_item` |
| `POST /api/v1/accounting/bank-receipt-work-items/:workItemID/start-review/`<br>bankreceiptworkitemhandler.startReview | Pending: Mark a bank receipt work item as under review. |

### benefits

| Write | Decision |
| --- | --- |
| `mutation createBenefitPlan` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation endBenefitEnrollment` | Pending: End benefit enrollment. |
| `mutation enrollBenefit` | Pending: Enroll benefit. |
| `mutation updateBenefitPlan` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |

### billingcontrol

| Write | Decision |
| --- | --- |
| `PUT /api/v1/billing-controls/`<br>billingcontrolhandler.update | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |

### billingqueue

| Write | Decision |
| --- | --- |
| `mutation assignBillingQueueBiller`<br>twin `PUT /api/v1/billing-queue/:itemID/assign/` | Tool: `assign_billing_queue_biller` |
| `mutation updateBillingQueueStatus`<br>twin `PUT /api/v1/billing-queue/:itemID/status/` | Tool: `approve_billing_queue_item`, `cancel_billing_queue_item`, `hold_billing_queue_item`, `move_billing_item_to_exception`, `send_billing_item_back_to_ops`, `transition_item_to_in_review` |
| `DELETE /api/v1/billing-queue/filter-presets/:presetId/`<br>billingqueuehandler.deleteFilterPreset | Exempt, user-preference: A saved filter on the billing queue screen. |
| `POST /api/v1/billing-queue/:itemID/reassign-charge/`<br>billingqueuehandler.reassignCharge | Pending: Move a charge from one billing queue item to another. |
| `POST /api/v1/billing-queue/filter-presets/`<br>billingqueuehandler.createFilterPreset | Exempt, user-preference: A saved filter on the billing queue screen. |
| `POST /api/v1/billing-queue/transfer/`<br>billingqueuehandler.transfer | Tool: `transfer_to_billing` |
| `PUT /api/v1/billing-queue/:itemID/charges/`<br>billingqueuehandler.updateCharges | Tool: `correct_charge_code` |
| `PUT /api/v1/billing-queue/filter-presets/:presetId/`<br>billingqueuehandler.updateFilterPreset | Exempt, user-preference: A saved filter on the billing queue screen. |

### billingtransfer

| Write | Decision |
| --- | --- |
| `mutation cancelBillingTransferRun` | Pending: Cancel billing transfer run. |
| `mutation retryBillingTransferRun` | Pending: Retry billing transfer run. |
| `mutation startBillingTransferRun` | Tool: `transfer_to_billing` |

### briefing

| Write | Decision |
| --- | --- |
| `mutation markBriefingRead` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation regenerateBriefing` | Exempt, agent-administration: Asks the briefing agent to write the briefing again. |

### carrier

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/carriers/:carrierID/`<br>carrierhandler.patch | Pending: Update some fields of a carrier. |
| `POST /api/v1/carriers/`<br>carrierhandler.create | Pending: Create a carrier. |
| `POST /api/v1/carriers/bulk-update-status/`<br>carrierhandler.bulkUpdateStatus | Pending: Change the status of several carriers at once. |
| `PUT /api/v1/carriers/:carrierID/`<br>carrierhandler.update | Pending: Update a carrier. |

### carrierintelligence

| Write | Decision |
| --- | --- |
| `mutation acknowledgeCarrierIntelEvents` | Tool: `acknowledge_carrier_intel_event` |
| `mutation applyCarrierIntelSuggestions` | Pending: Apply the carrier profile corrections carrier intelligence suggested. |
| `mutation grantCarrierIntelOverride` | Exempt, attestation: A person accepts accountability for using a carrier or equipment that failed vetting. |
| `mutation importSourcedCarrier` | Pending: Import sourced carrier. |
| `mutation markCarrierIntelReviewed` | Pending: Mark carrier intel reviewed. |
| `mutation overrideCarrierEquipmentVerification` | Exempt, attestation: A person accepts accountability for using a carrier or equipment that failed vetting. |
| `mutation resolveCarrierIntelEvent` | Tool: `resolve_carrier_intel_event` |
| `mutation resumeCarrierIntelMonitoring` | Pending: Resume carrier intel monitoring. |
| `mutation revokeCarrierIntelOverride` | Exempt, attestation: A person accepts accountability for using a carrier or equipment that failed vetting. |
| `mutation setCarrierMonitoring` | Pending: Set carrier monitoring. |
| `mutation switchCarrierIntelProvider` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |
| `mutation updateCarrierIntelControl` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |
| `mutation verifyCarrierEquipment` | Pending: Verify carrier equipment. |
| `mutation vetCarrier` | Pending: Vet carrier. |
| `mutation vetCustomerBroker` | Pending: Vet customer broker. |

### carriersettlement

| Write | Decision |
| --- | --- |
| `mutation acceptCarrierInvoiceMatch` | Pending: Accept carrier invoice match. |
| `mutation acceptCarrierInvoiceMatchWithVariance` | Pending: Accept carrier invoice match with variance. |
| `mutation addCarrierSettlementAdjustment` | Pending: Add carrier settlement adjustment. |
| `mutation approveCarrierSettlement` | Pending: Approve carrier settlement. |
| `mutation createCarrierInvoiceMatch` | Pending: Create carrier invoice match. |
| `mutation generateCarrierSettlementBatch` | Pending: Generate carrier settlement batch. |
| `mutation linkEdiCarrierInvoiceToCarrier` | Pending: Link EDI carrier invoice to carrier. |
| `mutation markCarrierSettlementPaid` | Pending: Mark carrier settlement paid. |
| `mutation postCarrierSettlement` | Pending: Post carrier settlement. |
| `mutation recalculateCarrierSettlement` | Pending: Recalculate carrier settlement. |
| `mutation rejectCarrierInvoiceMatch` | Pending: Reject carrier invoice match. |
| `mutation rejectCarrierSettlement` | Pending: Reject carrier settlement. |
| `mutation removeCarrierSettlementAdjustment` | Pending: Remove carrier settlement adjustment. |
| `mutation submitCarrierSettlement` | Pending: Submit carrier settlement. |
| `mutation updateCarrierSettlementControl` | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |
| `mutation voidCarrierSettlement` | Pending: Void carrier settlement. |

### commodity

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/commodities/:commodityID/`<br>commodityhandler.patch | Pending: Update some fields of a commodity. |
| `POST /api/v1/commodities/`<br>commodityhandler.create | Pending: Create a commodity. |
| `POST /api/v1/commodities/bulk-update-status/`<br>commodityhandler.bulkUpdateStatus | Pending: Change the status of several commodities at once. |
| `PUT /api/v1/commodities/:commodityID/`<br>commodityhandler.update | Pending: Update a commodity. |

### controlplaneprovisioning

| Write | Decision |
| --- | --- |
| `POST /api/v1/control-plane/tenants/provision`<br>controlplaneprovisioninghandler.provisionTenant | Exempt, infrastructure: Called by the platform control plane with a service credential to provision a tenant. |

### costing

| Write | Decision |
| --- | --- |
| `mutation updateCostCategory` | Exempt, configuration: Cost categories define the costing model every margin is computed with. |
| `mutation updateCostingControl` | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |

### customer

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/customers/:customerID/`<br>customerhandler.patch | Pending: Update some fields of a customer. |
| `POST /api/v1/customers/`<br>customerhandler.create | Pending: Create a customer. |
| `POST /api/v1/customers/bulk-update-status/`<br>customerhandler.bulkUpdateStatus | Pending: Change the status of several customers at once. |
| `PUT /api/v1/customers/:customerID/`<br>customerhandler.update | Pending: Update a customer. |

### customerpayment

| Write | Decision |
| --- | --- |
| `mutation applyCreditMemo`<br>twin `POST /api/v1/accounting/customer-payments/credit-memo-applications/` | Pending: Apply credit memo. |
| `mutation applyUnappliedCustomerPayment`<br>twin `POST /api/v1/accounting/customer-payments/:paymentID/apply/` | Pending: Apply unapplied customer payment. |
| `mutation postAndApplyCustomerPayment`<br>twin `POST /api/v1/accounting/customer-payments/` | Tool: `post_customer_payment` |
| `mutation reverseCustomerPayment`<br>twin `POST /api/v1/accounting/customer-payments/:paymentID/reverse/` | Pending: Reverse customer payment. |
| `mutation unapplyCreditMemoApplication`<br>twin `POST /api/v1/accounting/customer-payments/credit-memo-applications/:applicationID/unapply/` | Pending: Unapply credit memo application. |

### customfield

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/custom-fields/definitions/:definitionID/`<br>customfieldhandler.delete | Exempt, configuration: Custom field definitions change the shape of records for everyone; an administrator owns them. |
| `PATCH /api/v1/custom-fields/definitions/:definitionID/`<br>customfieldhandler.patch | Exempt, configuration: Custom field definitions change the shape of records for everyone; an administrator owns them. |
| `POST /api/v1/custom-fields/definitions/`<br>customfieldhandler.create | Exempt, configuration: Custom field definitions change the shape of records for everyone; an administrator owns them. |
| `PUT /api/v1/custom-fields/definitions/:definitionID/`<br>customfieldhandler.update | Exempt, configuration: Custom field definitions change the shape of records for everyone; an administrator owns them. |

### databasesession

| Write | Decision |
| --- | --- |
| `POST /api/v1/admin/database-sessions/:pid/terminate/`<br>databasesessionhandler.terminate | Exempt, security: Terminating database sessions is a platform administrator's emergency control. |

### dataentrycontrol

| Write | Decision |
| --- | --- |
| `PUT /api/v1/data-entry-controls/`<br>dataentrycontrolhandler.update | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |

### dataretention

| Write | Decision |
| --- | --- |
| `PUT /api/v1/data-retention/`<br>dataretentionhandler.update | Exempt, configuration: Retention periods decide what the organization deletes; an administrator sets them against its legal obligations. |

### decisions

| Write | Decision |
| --- | --- |
| `mutation decideAgentProposals` | Exempt, agent-administration: Deciding what an agent proposed is the human check on agents; an agent cannot approve its own work. |

### detention

| Write | Decision |
| --- | --- |
| `mutation approveDetentionOccurrence`<br>twin `POST /api/v1/detention/occurrences/:occurrenceID/approve/` | Tool: `approve_detention` |
| `mutation createDetentionPolicy`<br>twin `POST /api/v1/detention-policies/` | Exempt, configuration: Detention policies are the commercial terms each customer agreed to; an administrator maintains them. |
| `mutation deleteDetentionPolicy`<br>twin `DELETE /api/v1/detention-policies/:detentionPolicyID/` | Exempt, configuration: Detention policies are the commercial terms each customer agreed to; an administrator maintains them. |
| `mutation detentionBacktest`<br>twin `POST /api/v1/detention/backtest/` | Exempt, read-only: Replays a detention policy against past stops and saves nothing. |
| `mutation disputeDetentionOccurrence`<br>twin `POST /api/v1/detention/occurrences/:occurrenceID/dispute/` | Pending: Dispute detention occurrence. |
| `mutation sendDetentionNotice` | Tool: `send_detention_notice` |
| `mutation updateDetentionPolicy`<br>twin `PUT /api/v1/detention-policies/:detentionPolicyID/` | Exempt, configuration: Detention policies are the commercial terms each customer agreed to; an administrator maintains them. |
| `mutation waiveDetentionOccurrence`<br>twin `POST /api/v1/detention/occurrences/:occurrenceID/waive/` | Tool: `waive_detention` |

### detentionpolicy

| Write | Decision |
| --- | --- |
| `POST /api/v1/detention-policies/preview/`<br>detentionpolicyhandler.preview | Exempt, read-only: Previews how a detention policy would apply and saves nothing. |

### dispatchconsole

| Write | Decision |
| --- | --- |
| `mutation dispatchAssignMoveToCarrier`<br>twin `POST /api/v1/shipment-moves/:moveID/carrier-assignment/` | Tool: `assign_move_to_carrier` |
| `mutation dispatchAssignMoves`<br>twin `POST /api/v1/shipment-moves/:moveID/assignment/`<br>twin `PUT /api/v1/shipment-moves/:moveID/assignment/` | Tool: `assign_move` |
| `mutation dispatchCancelCarrierAssignment`<br>twin `DELETE /api/v1/shipment-moves/:moveID/carrier-assignment/` | Tool: `cancel_carrier_assignment` |
| `mutation dispatchPlanAutoAssign` | Exempt, read-only: Plans automatic assignments for review and saves nothing. |
| `mutation dispatchUnassignMoves`<br>twin `DELETE /api/v1/shipment-moves/:moveID/assignment/` | Tool: `unassign_moves` |

### dispatchcontrol

| Write | Decision |
| --- | --- |
| `PUT /api/v1/dispatch-controls/`<br>dispatchcontrolhandler.update | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |

### distancecontrol

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/distance-controls/`<br>distancecontrolhandler.patch | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |
| `PUT /api/v1/distance-controls/`<br>distancecontrolhandler.update | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |

### distanceoverride

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/distance-overrides/:distanceOverrideID/`<br>distanceoverridehandler.delete | Pending: Delete a distance override. |
| `PATCH /api/v1/distance-overrides/:distanceOverrideID/`<br>distanceoverridehandler.patch | Pending: Update some fields of a distance override. |
| `POST /api/v1/distance-overrides/`<br>distanceoverridehandler.create | Pending: Create a distance override. |
| `PUT /api/v1/distance-overrides/:distanceOverrideID/`<br>distanceoverridehandler.update | Pending: Update a distance override. |

### distanceprofile

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/distance-profiles/:distanceProfileID/`<br>distanceprofilehandler.delete | Exempt, configuration: Routing profiles decide how every distance is computed; an administrator maintains them. |
| `PATCH /api/v1/distance-profiles/:distanceProfileID/`<br>distanceprofilehandler.patch | Exempt, configuration: Routing profiles decide how every distance is computed; an administrator maintains them. |
| `POST /api/v1/distance-profiles/`<br>distanceprofilehandler.create | Exempt, configuration: Routing profiles decide how every distance is computed; an administrator maintains them. |
| `POST /api/v1/distance-profiles/:distanceProfileID/set-default/`<br>distanceprofilehandler.setDefault | Exempt, configuration: Routing profiles decide how every distance is computed; an administrator maintains them. |
| `PUT /api/v1/distance-profiles/:distanceProfileID/`<br>distanceprofilehandler.update | Exempt, configuration: Routing profiles decide how every distance is computed; an administrator maintains them. |

### document

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/documents/:documentID/`<br>documenthandler.delete | Pending: Delete a document. |
| `POST /api/v1/documents/:documentID/attach-to-shipment/`<br>documenthandler.attachToShipment | Tool: `attach_document_to_shipment` |
| `POST /api/v1/documents/:documentID/import-assistant/thread/`<br>documenthandler.openImportAssistantThread | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/documents/:documentID/restore/`<br>documenthandler.restoreVersion | Pending: Restore an earlier version of a document. |
| `POST /api/v1/documents/:documentID/shipment-draft/reextract/`<br>documenthandler.reextractDocumentContent | Pending: Run shipment extraction again on a document and replace the draft. |
| `POST /api/v1/documents/bulk-delete/`<br>documenthandler.bulkDelete | Pending: Delete several documents at once. |
| `POST /api/v1/documents/upload-bulk/`<br>documenthandler.uploadBulk | Exempt, infrastructure: Moves file bytes from a browser into storage; an agent attaches documents that already exist (attach_document_to_shipment). |
| `POST /api/v1/documents/upload/`<br>documenthandler.upload | Exempt, infrastructure: Moves file bytes from a browser into storage; an agent attaches documents that already exist (attach_document_to_shipment). |
| `POST /api/v1/documents/uploads/`<br>documenthandler.createUploadSession | Exempt, infrastructure: Moves file bytes from a browser into storage; an agent attaches documents that already exist (attach_document_to_shipment). |
| `POST /api/v1/documents/uploads/:uploadSessionID/cancel/`<br>documenthandler.cancelUploadSession | Exempt, infrastructure: Moves file bytes from a browser into storage; an agent attaches documents that already exist (attach_document_to_shipment). |
| `POST /api/v1/documents/uploads/:uploadSessionID/complete/`<br>documenthandler.completeUploadSession | Exempt, infrastructure: Moves file bytes from a browser into storage; an agent attaches documents that already exist (attach_document_to_shipment). |
| `POST /api/v1/documents/uploads/:uploadSessionID/parts/`<br>documenthandler.getUploadPartURLs | Exempt, infrastructure: Moves file bytes from a browser into storage; an agent attaches documents that already exist (attach_document_to_shipment). |
| `PUT /api/v1/documents/uploads/:uploadSessionID/parts/:partNumber/`<br>documenthandler.uploadSessionPart | Exempt, infrastructure: Moves file bytes from a browser into storage; an agent attaches documents that already exist (attach_document_to_shipment). |

### documentcontrol

| Write | Decision |
| --- | --- |
| `PUT /api/v1/document-controls/`<br>documentcontrolhandler.update | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |

### documentoperations

| Write | Decision |
| --- | --- |
| `POST /api/v1/admin/document-operations/:documentID/reextract/`<br>documentoperationshandler.reextract | Exempt, infrastructure: A repair operation for when processing failed; the platform retries on its own. |
| `POST /api/v1/admin/document-operations/:documentID/regenerate-preview/`<br>documentoperationshandler.regeneratePreview | Exempt, infrastructure: A repair operation for when processing failed; the platform retries on its own. |
| `POST /api/v1/admin/document-operations/:documentID/resync-search/`<br>documentoperationshandler.resyncSearch | Exempt, infrastructure: A repair operation for when processing failed; the platform retries on its own. |

### documentpacketrule

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/document-packet-rules/:ruleID/`<br>documentpacketrulehandler.delete | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `POST /api/v1/document-packet-rules/`<br>documentpacketrulehandler.create | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `PUT /api/v1/document-packet-rules/:ruleID/`<br>documentpacketrulehandler.update | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |

### documentparsingrule

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/document-parsing-rules/:ruleSetID/`<br>documentparsingrulehandler.deleteRuleSet | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `DELETE /api/v1/document-parsing-rules/fixtures/:fixtureID/`<br>documentparsingrulehandler.deleteFixture | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `POST /api/v1/document-parsing-rules/`<br>documentparsingrulehandler.createRuleSet | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `POST /api/v1/document-parsing-rules/:ruleSetID/fixtures/`<br>documentparsingrulehandler.saveFixture<br>also `PUT /api/v1/document-parsing-rules/fixtures/:fixtureID/` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `POST /api/v1/document-parsing-rules/:ruleSetID/versions/`<br>documentparsingrulehandler.createVersion | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `POST /api/v1/document-parsing-rules/versions/:versionID/publish/`<br>documentparsingrulehandler.publishVersion | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `POST /api/v1/document-parsing-rules/versions/:versionID/simulate/`<br>documentparsingrulehandler.simulateVersion | Exempt, read-only: Runs a parsing rule version against a sample and saves nothing. |
| `PUT /api/v1/document-parsing-rules/:ruleSetID/`<br>documentparsingrulehandler.updateRuleSet | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `PUT /api/v1/document-parsing-rules/versions/:versionID/`<br>documentparsingrulehandler.updateVersion | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |

### documenttemplate

| Write | Decision |
| --- | --- |
| `mutation archiveDocumentTemplateVersion` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation assignDocumentTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation createDocumentTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation createDocumentTemplateVersion` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation deleteDocumentTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation deleteDocumentTemplateVersion` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation publishDocumentTemplateVersion` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation rollbackDocumentTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation sendTestMessageTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation unassignDocumentTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation updateDocumentTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation updateDocumentTemplateVersion` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |

### documenttype

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/document-types/:docTypeID/`<br>documenttypehandler.patch | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `POST /api/v1/document-types/`<br>documenttypehandler.create | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `PUT /api/v1/document-types/:docTypeID/`<br>documenttypehandler.update | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |

### driverportal

| Write | Decision |
| --- | --- |
| `mutation acknowledgeMyPolicy` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation cancelMyExpense` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation cancelMyPto` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation createMyLoadComment` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation createSettlementDispute` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation dismissMyNotifications` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation inviteWorkerToPortal` | Exempt, security: Gives a driver a sign-in to the driver portal. |
| `mutation markAllMyNotificationsRead` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation markMyNotificationsRead` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation markMyNotificationsUnread` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation proposeMyShiftSwap` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation recordMyStopAction` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation requestMyPto` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation resolveSettlementDispute` | Pending: Resolve settlement dispute. |
| `mutation respondToMyAssignment` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation respondToMyShiftSwap` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation restoreMyNotifications` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation reviewDriverExpense` | Pending: Review driver expense. |
| `mutation revokeWorkerPortalAccess` | Exempt, security: Removes a driver's sign-in to the driver portal. |
| `mutation setMyAvailability` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation startSettlementDisputeReview` | Pending: Start settlement dispute review. |
| `mutation submitMyExpense` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation updateDashControl` | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |
| `mutation updateMyContactInfo` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation withdrawMyProfileChange` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation withdrawSettlementDispute` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `POST /api/v1/portal/credentials/:credentialID/document/`<br>driverportalhandler.uploadCredentialDocument | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `POST /api/v1/portal/expenses/:expenseID/receipt/`<br>driverportalhandler.uploadExpenseReceipt | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `POST /api/v1/portal/invitations/accept`<br>driverportalhandler.acceptInvitation | Exempt, security: A driver accepting a portal invitation creates their own sign-in. |
| `POST /api/v1/portal/loads/:shipmentID/documents/`<br>driverportalhandler.uploadLoadDocument | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `POST /api/v1/portal/profile/documents/`<br>driverportalhandler.uploadProfileDocument | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |

### driversettlement

| Write | Decision |
| --- | --- |
| `mutation addDriverSettlementAdjustment` | Pending: Add driver settlement adjustment. |
| `mutation adjustEscrowAccount` | Pending: Adjust escrow account. |
| `mutation approveDriverSettlement` | Pending: Approve driver settlement. |
| `mutation assignPayProfileToWorker` | Pending: Assign pay profile to worker. |
| `mutation attachPayEventsToSettlement` | Pending: Attach pay events to a driver settlement. |
| `mutation bulkDriverSettlementAction` | Pending: Approve, post or void several driver settlements at once. |
| `mutation closeEscrowAccount` | Pending: Close escrow account. |
| `mutation createPayCode` | Pending: Create pay code. |
| `mutation createPayProfile` | Pending: Create pay profile. |
| `mutation createRecurringDeduction` | Pending: Create recurring deduction. |
| `mutation createRecurringEarning` | Pending: Create recurring earning. |
| `mutation detachPayEventFromSettlement` | Pending: Detach a pay event from a driver settlement. |
| `mutation endWorkerPayAssignment` | Pending: End worker pay assignment. |
| `mutation generateDriverSettlement` | Pending: Generate driver settlement. |
| `mutation generateSettlementBatch` | Pending: Generate settlement batch. |
| `mutation holdDriverPayEvent` | Pending: Hold driver pay event. |
| `mutation issuePayAdvance` | Pending: Issue pay advance. |
| `mutation markDriverSettlementPaid` | Pending: Mark driver settlement paid. |
| `mutation openEscrowAccount` | Pending: Open escrow account. |
| `mutation payWorkerNow` | Pending: Pay a worker off cycle now. |
| `mutation postDriverSettlement` | Pending: Post driver settlement. |
| `mutation recalculateDriverSettlement` | Pending: Recalculate driver settlement. |
| `mutation rejectDriverSettlement` | Pending: Reject driver settlement. |
| `mutation releaseDriverPayEvent` | Pending: Release driver pay event. |
| `mutation removeDriverSettlementAdjustment` | Pending: Remove driver settlement adjustment. |
| `mutation submitDriverSettlement` | Pending: Submit driver settlement. |
| `mutation updateEscrowAccount` | Pending: Update escrow account. |
| `mutation updatePayCode` | Pending: Update pay code. |
| `mutation updatePayProfile` | Pending: Update pay profile. |
| `mutation updateRecurringDeduction` | Pending: Update recurring deduction. |
| `mutation updateRecurringEarning` | Pending: Update recurring earning. |
| `mutation updateSettlementControl` | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |
| `mutation voidDriverSettlement` | Pending: Void driver settlement. |
| `mutation writeOffPayAdvance` | Pending: Write off pay advance. |

### edi

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/edi/mapping-profiles/:profileID/items/:mappingItemID/`<br>edihandler.deleteMappingProfileItem | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `DELETE /api/v1/edi/partners/:partnerID/mapping-profile/items/:mappingItemID/`<br>edihandler.deleteMappingItem | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `DELETE /api/v1/edi/test-cases/:testCaseID/`<br>edihandler.deleteTestCase | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/as2/inbound/`<br>edihandler.receiveAS2Message | Exempt, infrastructure: Receives AS2 messages a trading partner sends; no person makes this call. |
| `POST /api/v1/edi/catalog/partner-settings/validate/`<br>edihandler.validatePartnerSettings | Exempt, read-only: Validates, inspects, previews or tests EDI setup and saves nothing. |
| `POST /api/v1/edi/communication-profiles/`<br>edihandler.createCommunicationProfile | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/communication-profiles/:profileID/poll/`<br>edihandler.pollCommunicationProfile | Exempt, infrastructure: Polls a partner mailbox now; the scheduler polls on its own. |
| `POST /api/v1/edi/communication-profiles/:profileID/test-connection/`<br>edihandler.testCommunicationProfileConnection | Exempt, read-only: Validates, inspects, previews or tests EDI setup and saves nothing. |
| `POST /api/v1/edi/communication-profiles/inspect-certificate/`<br>edihandler.inspectCertificate | Exempt, read-only: Validates, inspects, previews or tests EDI setup and saves nothing. |
| `POST /api/v1/edi/connections/`<br>edihandler.createConnection | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/connections/:connectionID/accept/`<br>edihandler.acceptConnection | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/connections/:connectionID/reject/`<br>edihandler.rejectConnection | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/connections/:connectionID/revoke/`<br>edihandler.revokeConnection | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/connections/:connectionID/suspend/`<br>edihandler.suspendConnection | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/control-numbers/reset/`<br>edihandler.resetControlNumber | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/document-profiles/`<br>edihandler.createPartnerDocumentProfile | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/documents/generate/`<br>edihandler.generateDocument | Pending: Generate an outbound EDI document (a 214, a 210) for a record. |
| `POST /api/v1/edi/documents/preview/`<br>edihandler.previewDocument | Exempt, read-only: Validates, inspects, previews or tests EDI setup and saves nothing. |
| `POST /api/v1/edi/inbound-files/:fileID/reprocess/`<br>edihandler.reprocessInboundFile | Pending: Process a failed inbound EDI file again. |
| `POST /api/v1/edi/inbound-files/bulk-reprocess/`<br>edihandler.bulkReprocessInboundFiles | Pending: Process several failed inbound EDI files again. |
| `POST /api/v1/edi/load-tenders/`<br>edihandler.submitLoadTender | Pending: Send a load tender to a trading partner over EDI. |
| `POST /api/v1/edi/messages/:messageID/replay/`<br>edihandler.replayMessageDelivery | Pending: Send an EDI message again to its partner. |
| `POST /api/v1/edi/messages/:messageID/retry-delivery/`<br>edihandler.retryMessageDelivery | Pending: Retry delivery of an EDI message that failed to send. |
| `POST /api/v1/edi/messages/bulk-retry-delivery/`<br>edihandler.bulkRetryMessageDelivery | Pending: Retry delivery of several EDI messages that failed to send. |
| `POST /api/v1/edi/partners/`<br>edihandler.createPartner | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/partners/internal-pairs/`<br>edihandler.createInternalPartnerPair | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/templates/`<br>edihandler.createTemplate | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/templates/:templateID/draft/`<br>edihandler.createDraftVersion | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/templates/:templateID/versions/:versionID/activate/`<br>edihandler.activateTemplateVersion | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/templates/:templateID/versions/:versionID/archive/`<br>edihandler.archiveTemplateVersion | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/templates/:templateID/versions/:versionID/certify/`<br>edihandler.certifyTemplateVersion | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/templates/:templateID/versions/:versionID/rollback/`<br>edihandler.rollbackTemplateVersion | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/templates/:templateID/versions/:versionID/validate/`<br>edihandler.validateTemplateVersion | Exempt, read-only: Validates, inspects, previews or tests EDI setup and saves nothing. |
| `POST /api/v1/edi/tender-changes/:changeID/apply/`<br>edihandler.applyTenderChange | Pending: Apply a change a trading partner sent to a tendered load. |
| `POST /api/v1/edi/tender-changes/:changeID/reject/`<br>edihandler.rejectTenderChange | Pending: Reject a change a trading partner sent to a tendered load. |
| `POST /api/v1/edi/test-cases/`<br>edihandler.createTestCase | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/test-cases/:testCaseID/preview/`<br>edihandler.previewTestCase | Exempt, read-only: Validates, inspects, previews or tests EDI setup and saves nothing. |
| `POST /api/v1/edi/transfer-changes/:changeID/apply/`<br>edihandler.applyTransferChange | Pending: Apply a change to an inbound EDI transfer. |
| `POST /api/v1/edi/transfer-changes/:changeID/reject/`<br>edihandler.rejectTransferChange | Pending: Reject a change to an inbound EDI transfer. |
| `POST /api/v1/edi/transfers/:transferID/approve/`<br>edihandler.approveTransfer | Pending: Accept an inbound EDI load tender and create the shipment. |
| `POST /api/v1/edi/transfers/:transferID/cancel/`<br>edihandler.cancelTransfer | Pending: Cancel an inbound EDI transfer. |
| `POST /api/v1/edi/transfers/:transferID/expire/`<br>edihandler.expireTransfer | Pending: Expire an inbound EDI transfer that was not answered in time. |
| `POST /api/v1/edi/transfers/:transferID/reject/`<br>edihandler.rejectTransfer | Pending: Decline an inbound EDI load tender. |
| `POST /api/v1/edi/transfers/bulk-approve/`<br>edihandler.bulkApproveTransfers | Pending: Accept several inbound EDI load tenders at once. |
| `POST /api/v1/edi/transfers/bulk-reject/`<br>edihandler.bulkRejectTransfers | Pending: Decline several inbound EDI load tenders at once. |
| `POST /api/v1/edi/x12/inspect/`<br>edihandler.inspectX12 | Exempt, read-only: Validates, inspects, previews or tests EDI setup and saves nothing. |
| `PUT /api/v1/edi/communication-profiles/:profileID/`<br>edihandler.updateCommunicationProfile | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `PUT /api/v1/edi/document-profiles/:profileID/`<br>edihandler.updatePartnerDocumentProfile | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `PUT /api/v1/edi/mapping-profiles/:profileID/items/`<br>edihandler.updateMappingProfileItems | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `PUT /api/v1/edi/partners/:partnerID/`<br>edihandler.updatePartner | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `PUT /api/v1/edi/partners/:partnerID/mapping-profile/`<br>edihandler.updateMappingProfile | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `PUT /api/v1/edi/templates/:templateID/`<br>edihandler.updateTemplate | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `PUT /api/v1/edi/templates/:templateID/versions/:versionID/`<br>edihandler.updateTemplateVersion | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `PUT /api/v1/edi/templates/:templateID/versions/:versionID/script-libraries/`<br>edihandler.replaceTemplateScriptLibraries | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `PUT /api/v1/edi/templates/:templateID/versions/:versionID/segments/`<br>edihandler.replaceTemplateSegments | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `PUT /api/v1/edi/test-cases/:testCaseID/`<br>edihandler.updateTestCase | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |

### email

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/email-profiles/:profileID/`<br>emailhandler.deleteProfile | Exempt, configuration: Sending profiles, suppressions and assignments decide how the organization sends mail; an administrator owns them. |
| `DELETE /api/v1/email-suppressions/:suppressionID/`<br>emailhandler.deleteSuppression | Exempt, configuration: Sending profiles, suppressions and assignments decide how the organization sends mail; an administrator owns them. |
| `POST /api/v1/email-profiles/`<br>emailhandler.createProfile | Exempt, configuration: Sending profiles, suppressions and assignments decide how the organization sends mail; an administrator owns them. |
| `POST /api/v1/email-profiles/:profileID/test-send/`<br>emailhandler.testSend | Exempt, configuration: Sending profiles, suppressions and assignments decide how the organization sends mail; an administrator owns them. |
| `POST /api/v1/email-suppressions/`<br>emailhandler.createSuppression | Exempt, configuration: Sending profiles, suppressions and assignments decide how the organization sends mail; an administrator owns them. |
| `POST /api/v1/webhooks/email/postmark/:webhookToken/`<br>emailhandler.handlePostmarkWebhook | Exempt, infrastructure: An inbound webhook a provider calls with a signed token, not a person. |
| `POST /api/v1/webhooks/email/resend/:webhookToken/`<br>emailhandler.handleResendWebhook | Exempt, infrastructure: An inbound webhook a provider calls with a signed token, not a person. |
| `PUT /api/v1/email-profiles/:profileID/`<br>emailhandler.updateProfile | Exempt, configuration: Sending profiles, suppressions and assignments decide how the organization sends mail; an administrator owns them. |
| `PUT /api/v1/email-profiles/assignments/`<br>emailhandler.updateAssignments | Exempt, configuration: Sending profiles, suppressions and assignments decide how the organization sends mail; an administrator owns them. |

### equipmentmanufacturer

| Write | Decision |
| --- | --- |
| `mutation bulkUpdateEquipmentManufacturerStatus`<br>twin `POST /api/v1/equipment-manufacturers/bulk-update-status/` | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `mutation createEquipmentManufacturer`<br>twin `POST /api/v1/equipment-manufacturers/` | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `mutation patchEquipmentManufacturer`<br>twin `PATCH /api/v1/equipment-manufacturers/:equipManufacturerID/` | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `mutation updateEquipmentManufacturer`<br>twin `PUT /api/v1/equipment-manufacturers/:equipManufacturerID/` | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |

### equipmenttype

| Write | Decision |
| --- | --- |
| `mutation bulkUpdateEquipmentTypeStatus`<br>twin `POST /api/v1/equipment-types/bulk-update-status/` | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `mutation createEquipmentType`<br>twin `POST /api/v1/equipment-types/` | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `mutation patchEquipmentType`<br>twin `PATCH /api/v1/equipment-types/:equipTypeID/` | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `mutation updateEquipmentType`<br>twin `PUT /api/v1/equipment-types/:equipTypeID/` | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |

### exchangerate

| Write | Decision |
| --- | --- |
| `POST /api/v1/exchange-rates/refresh`<br>exchangeratehandler.refresh | Exempt, infrastructure: Fetches exchange rates from the provider now; the scheduler refreshes them on its own. |
| `POST /api/v1/exchange-rates/settlement-quotes`<br>exchangeratehandler.createSettlementQuote | Pending: Lock an exchange rate quote for settling a foreign-currency payment. |

### fiscalperiod

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/fiscal-periods/:fiscalPeriodID/`<br>fiscalperiodhandler.delete | Pending: Delete a fiscal period. |
| `PATCH /api/v1/fiscal-periods/:fiscalPeriodID/`<br>fiscalperiodhandler.patch | Pending: Update some fields of a fiscal period. |
| `POST /api/v1/fiscal-periods/`<br>fiscalperiodhandler.create | Pending: Create a fiscal period. |
| `PUT /api/v1/fiscal-periods/:fiscalPeriodID/`<br>fiscalperiodhandler.update | Pending: Update a fiscal period. |
| `PUT /api/v1/fiscal-periods/:fiscalPeriodID/activate/`<br>fiscalperiodhandler.activate | Pending: Activate a fiscal period. |
| `PUT /api/v1/fiscal-periods/:fiscalPeriodID/close/`<br>fiscalperiodhandler.close | Pending: Close a fiscal period. |
| `PUT /api/v1/fiscal-periods/:fiscalPeriodID/lock/`<br>fiscalperiodhandler.lock | Pending: Lock a fiscal period. |
| `PUT /api/v1/fiscal-periods/:fiscalPeriodID/reopen/`<br>fiscalperiodhandler.reopen | Pending: Reopen a fiscal period. |
| `PUT /api/v1/fiscal-periods/:fiscalPeriodID/unlock/`<br>fiscalperiodhandler.unlock | Pending: Unlock a fiscal period. |

### fiscalyear

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/fiscal-years/:fiscalYearID/`<br>fiscalyearhandler.delete | Pending: Delete a fiscal year. |
| `PATCH /api/v1/fiscal-years/:fiscalYearID/`<br>fiscalyearhandler.patch | Pending: Update some fields of a fiscal year. |
| `POST /api/v1/fiscal-years/`<br>fiscalyearhandler.create | Pending: Create a fiscal year. |
| `PUT /api/v1/fiscal-years/:fiscalYearID/`<br>fiscalyearhandler.update | Pending: Update a fiscal year. |
| `PUT /api/v1/fiscal-years/:fiscalYearID/activate/`<br>fiscalyearhandler.activate | Pending: Activate a fiscal year. |
| `PUT /api/v1/fiscal-years/:fiscalYearID/close/`<br>fiscalyearhandler.close | Pending: Close a fiscal year. |
| `PUT /api/v1/fiscal-years/:fiscalYearID/reopen/`<br>fiscalyearhandler.reopen | Pending: Reopen a fiscal year. |

### fleetcode

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/fleet-codes/:fleetCodeID`<br>fleetcodehandler.patch | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `POST /api/v1/fleet-codes/`<br>fleetcodehandler.create | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `PUT /api/v1/fleet-codes/:fleetCodeID`<br>fleetcodehandler.update | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |

### fleetsafety

| Write | Decision |
| --- | --- |
| `mutation deleteSafetyViolation` | Pending: Delete safety violation. |
| `mutation recordSafetyViolation` | Pending: Record safety violation. |
| `mutation updateSafetyViolation` | Pending: Update safety violation. |

### formulatemplate

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/formula-templates/:templateID/test-cases/:testCaseID`<br>formulatemplatehandler.deleteTestCase | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `PATCH /api/v1/formula-templates/:templateID/`<br>formulatemplatehandler.patch | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `PATCH /api/v1/formula-templates/:templateID/versions/:versionNumber/effective-date`<br>formulatemplatehandler.updateVersionEffectiveDate | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `PATCH /api/v1/formula-templates/:templateID/versions/:versionNumber/tags`<br>formulatemplatehandler.updateVersionTags | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `POST /api/v1/formula-templates/`<br>formulatemplatehandler.create | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `POST /api/v1/formula-templates/:templateID/approve`<br>formulatemplatehandler.approve | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `POST /api/v1/formula-templates/:templateID/backtest`<br>formulatemplatehandler.backtest | Exempt, read-only: Evaluates a pricing formula against samples or past shipments and saves nothing. |
| `POST /api/v1/formula-templates/:templateID/fork`<br>formulatemplatehandler.fork | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `POST /api/v1/formula-templates/:templateID/impact`<br>formulatemplatehandler.approvalImpact | Exempt, read-only: Evaluates a pricing formula against samples or past shipments and saves nothing. |
| `POST /api/v1/formula-templates/:templateID/reject`<br>formulatemplatehandler.reject | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `POST /api/v1/formula-templates/:templateID/request-changes`<br>formulatemplatehandler.requestChanges | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `POST /api/v1/formula-templates/:templateID/rollback`<br>formulatemplatehandler.rollback | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `POST /api/v1/formula-templates/:templateID/submit`<br>formulatemplatehandler.submit | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `POST /api/v1/formula-templates/:templateID/test-cases`<br>formulatemplatehandler.createTestCase | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `POST /api/v1/formula-templates/:templateID/test-cases/run`<br>formulatemplatehandler.runTestCases | Exempt, read-only: Evaluates a pricing formula against samples or past shipments and saves nothing. |
| `POST /api/v1/formula-templates/:templateID/versions`<br>formulatemplatehandler.createVersion | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `POST /api/v1/formula-templates/ai/thread/`<br>formulatemplatehandler.openAssistantThread | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/formula-templates/bulk-update-status`<br>formulatemplatehandler.bulkUpdateStatus | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `POST /api/v1/formula-templates/duplicate`<br>formulatemplatehandler.duplicate | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `POST /api/v1/formula-templates/import`<br>formulatemplatehandler.importTemplates | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `POST /api/v1/formula-templates/install-standards`<br>formulatemplatehandler.installStandards | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `POST /api/v1/formula-templates/test`<br>formulatemplatehandler.testExpression | Exempt, read-only: Evaluates a pricing formula against samples or past shipments and saves nothing. |
| `PUT /api/v1/formula-templates/:templateID/`<br>formulatemplatehandler.update | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |
| `PUT /api/v1/formula-templates/:templateID/test-cases/:testCaseID`<br>formulatemplatehandler.updateTestCase | Exempt, configuration: Pricing formulas are rating logic an administrator authors, tests and approves before any rate uses them. |

### fuelpurchase

| Write | Decision |
| --- | --- |
| `mutation assignFuelCard` | Pending: Assign fuel card. |
| `mutation cancelFuelCard` | Pending: Cancel fuel card. |
| `mutation commitFuelPurchaseImport` | Pending: Commit fuel purchase import. |
| `mutation createFuelCard` | Pending: Create fuel card. |
| `mutation createFuelPurchase` | Pending: Create fuel purchase. |
| `mutation createFuelPurchaseImport` | Pending: Create fuel purchase import. |
| `mutation deleteFuelPurchase` | Pending: Delete fuel purchase. |
| `mutation discardFuelPurchaseImport` | Pending: Discard fuel purchase import. |
| `mutation resolveFuelPurchaseImportRows` | Pending: Resolve fuel purchase import rows. |
| `mutation stageFuelPurchaseImport` | Pending: Stage fuel purchase import. |
| `mutation syncFuelCardFeed` | Pending: Sync fuel card feed. |
| `mutation updateFuelCard` | Pending: Update fuel card. |
| `mutation updateFuelPurchase` | Pending: Update fuel purchase. |

### fuelsurcharge

| Write | Decision |
| --- | --- |
| `mutation addFuelIndexPrice` | Pending: Add fuel index price. |
| `mutation createFuelIndex` | Exempt, configuration: Fuel surcharge programs and the indexes they follow are commercial terms an administrator sets up. |
| `mutation createFuelSurchargeProgram` | Exempt, configuration: Fuel surcharge programs and the indexes they follow are commercial terms an administrator sets up. |
| `mutation deleteFuelIndex` | Exempt, configuration: Fuel surcharge programs and the indexes they follow are commercial terms an administrator sets up. |
| `mutation deleteFuelIndexPrice` | Pending: Delete fuel index price. |
| `mutation deleteFuelSurchargeProgram` | Exempt, configuration: Fuel surcharge programs and the indexes they follow are commercial terms an administrator sets up. |
| `mutation updateFuelIndex` | Exempt, configuration: Fuel surcharge programs and the indexes they follow are commercial terms an administrator sets up. |
| `mutation updateFuelIndexPrice` | Pending: Update fuel index price. |
| `mutation updateFuelSurchargeProgram` | Exempt, configuration: Fuel surcharge programs and the indexes they follow are commercial terms an administrator sets up. |

### glaccount

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/gl-accounts/:glAccountID/`<br>glaccounthandler.delete | Pending: Delete a gl account. |
| `PATCH /api/v1/gl-accounts/:glAccountID/`<br>glaccounthandler.patch | Pending: Update some fields of a gl account. |
| `POST /api/v1/gl-accounts/`<br>glaccounthandler.create | Pending: Create a gl account. |
| `POST /api/v1/gl-accounts/bulk-update-status/`<br>glaccounthandler.bulkUpdateStatus | Pending: Change the status of several gl accounts at once. |
| `PUT /api/v1/gl-accounts/:glAccountID/`<br>glaccounthandler.update | Pending: Update a gl account. |

### googlemaps

| Write | Decision |
| --- | --- |
| `POST /api/v1/google-maps/autocomplete/`<br>googlemapshandler.autocomplete | Exempt, read-only: Returns address suggestions from the maps provider. |

### graphql

| Write | Decision |
| --- | --- |
| `POST /graphql`<br>graphql.handle | Exempt, infrastructure: The GraphQL transport; every mutation it carries is listed as its own write. |

### hazardousmaterial

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/hazardous-materials/:hazardousMaterialID/`<br>hazardousmaterialhandler.patch | Pending: Update some fields of a hazardous material. |
| `POST /api/v1/hazardous-materials/`<br>hazardousmaterialhandler.create | Pending: Create a hazardous material. |
| `POST /api/v1/hazardous-materials/bulk-update-status/`<br>hazardousmaterialhandler.bulkUpdateStatus | Pending: Change the status of several hazardous materials at once. |
| `PUT /api/v1/hazardous-materials/:hazardousMaterialID/`<br>hazardousmaterialhandler.update | Pending: Update a hazardous material. |

### hazmatsegregationrule

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/hazmat-segregation-rules/:hazmatSegregationRuleID/`<br>hazmatsegregationrulehandler.patch | Exempt, configuration: Hazmat segregation rules encode the regulation every load is checked against; an administrator maintains them. |
| `POST /api/v1/hazmat-segregation-rules/`<br>hazmatsegregationrulehandler.create | Exempt, configuration: Hazmat segregation rules encode the regulation every load is checked against; an administrator maintains them. |
| `PUT /api/v1/hazmat-segregation-rules/:hazmatSegregationRuleID/`<br>hazmatsegregationrulehandler.update | Exempt, configuration: Hazmat segregation rules encode the regulation every load is checked against; an administrator maintains them. |

### holdreason

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/hold-reasons/:holdReasonID/`<br>holdreasonhandler.patch | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `POST /api/v1/hold-reasons/`<br>holdreasonhandler.create | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `PUT /api/v1/hold-reasons/:holdReasonID/`<br>holdreasonhandler.update | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |

### homelayout

| Write | Decision |
| --- | --- |
| `mutation createHomeLayoutPreset` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation deleteHomeLayoutPreset` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation resetHomeLayout` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation updateHomeLayout` | Tool: `add_home_widget`, `remove_home_widget`, `arrange_home_layout` |
| `mutation updateHomeLayoutPreset` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |

### iam

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/organizations/:id/iam/access-policies/:policyId`<br>iamhandler.deleteAccessPolicy | Exempt, security: Identity providers, SCIM directories and access policies decide who can sign in and with what access. |
| `DELETE /api/v1/organizations/:id/iam/identity-providers/:providerId`<br>iamhandler.deleteIdentityProvider | Exempt, security: Identity providers, SCIM directories and access policies decide who can sign in and with what access. |
| `DELETE /api/v1/organizations/:id/iam/scim/directories/:directoryId`<br>iamhandler.deleteSCIMDirectory | Exempt, security: Identity providers, SCIM directories and access policies decide who can sign in and with what access. |
| `DELETE /api/v1/organizations/:id/iam/scim/directories/:directoryId/group-role-mappings/:mappingId`<br>iamhandler.deleteSCIMGroupRoleMapping | Exempt, security: Identity providers, SCIM directories and access policies decide who can sign in and with what access. |
| `POST /api/v1/organizations/:id/iam/access-policies`<br>iamhandler.createAccessPolicy | Exempt, security: Identity providers, SCIM directories and access policies decide who can sign in and with what access. |
| `POST /api/v1/organizations/:id/iam/identity-providers`<br>iamhandler.createIdentityProvider | Exempt, security: Identity providers, SCIM directories and access policies decide who can sign in and with what access. |
| `POST /api/v1/organizations/:id/iam/scim/directories`<br>iamhandler.createSCIMDirectory | Exempt, security: Identity providers, SCIM directories and access policies decide who can sign in and with what access. |
| `POST /api/v1/organizations/:id/iam/scim/directories/:directoryId/group-role-mappings`<br>iamhandler.createSCIMGroupRoleMapping | Exempt, security: Identity providers, SCIM directories and access policies decide who can sign in and with what access. |
| `POST /api/v1/organizations/:id/iam/scim/directories/:directoryId/tokens`<br>iamhandler.createSCIMToken | Exempt, security: Identity providers, SCIM directories and access policies decide who can sign in and with what access. |
| `POST /api/v1/organizations/:id/iam/scim/tokens/:tokenId/revoke`<br>iamhandler.revokeSCIMToken | Exempt, security: Identity providers, SCIM directories and access policies decide who can sign in and with what access. |
| `PUT /api/v1/organizations/:id/iam/access-policies/:policyId`<br>iamhandler.updateAccessPolicy | Exempt, security: Identity providers, SCIM directories and access policies decide who can sign in and with what access. |
| `PUT /api/v1/organizations/:id/iam/identity-providers/:providerId`<br>iamhandler.updateIdentityProvider | Exempt, security: Identity providers, SCIM directories and access policies decide who can sign in and with what access. |
| `PUT /api/v1/organizations/:id/iam/scim/directories/:directoryId`<br>iamhandler.updateSCIMDirectory | Exempt, security: Identity providers, SCIM directories and access policies decide who can sign in and with what access. |
| `PUT /api/v1/organizations/:id/iam/scim/directories/:directoryId/group-role-mappings/:mappingId`<br>iamhandler.updateSCIMGroupRoleMapping | Exempt, security: Identity providers, SCIM directories and access policies decide who can sign in and with what access. |

### ifta

| Write | Decision |
| --- | --- |
| `mutation amendIftaReturn` | Pending: Amend IFTA return. |
| `mutation backfillJurisdictionMiles` | Pending: Backfill jurisdiction miles. |
| `mutation createIftaMileageEntry` | Pending: Create IFTA mileage entry. |
| `mutation deleteIftaMileageEntry` | Pending: Delete IFTA mileage entry. |
| `mutation deleteIftaReturn` | Pending: Delete IFTA return. |
| `mutation deleteIftaTaxRate` | Pending: Delete IFTA tax rate. |
| `mutation finalizeIftaReturn` | Exempt, attestation: A fuel tax return is signed off by the person accountable for filing it. |
| `mutation generateIftaReturn` | Pending: Generate IFTA return. |
| `mutation markIftaReturnFiled` | Exempt, attestation: Records that a person filed the return with the jurisdiction. |
| `mutation recalculateMoveJurisdictionMiles` | Pending: Recalculate move jurisdiction miles. |
| `mutation recomputeIftaReturn` | Pending: Recompute IFTA return. |
| `mutation reopenIftaReturn` | Pending: Reopen IFTA return. |
| `mutation updateIftaMileageEntry` | Pending: Update IFTA mileage entry. |
| `mutation upsertIftaTaxRates` | Pending: Upsert IFTA tax rates. |

### inbound

| Write | Decision |
| --- | --- |
| `POST /api/v1/webhooks/inbound-mail/:mailboxToken/`<br>inboundhandler.receive | Exempt, infrastructure: An inbound webhook a provider calls with a signed token, not a person. |

### inboundmessage

| Write | Decision |
| --- | --- |
| `mutation createInboundMailbox` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |
| `mutation linkInboundMessage` | Tool: `link_inbound_message` |
| `mutation reviewInboundMessage` | Tool: `mark_inbound_message` |
| `mutation rotateInboundMailboxToken` | Exempt, security: Replaces the secret that authenticates mail forwarded into the mailbox. |
| `mutation setInboundMailboxSigningSecret` | Exempt, security: Sets the secret that authenticates mail forwarded into the mailbox. |
| `mutation updateInboundMailbox` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |

### insight

| Write | Decision |
| --- | --- |
| `POST /api/v1/insights/:insightID/dismiss/`<br>insighthandler.dismiss | Tool: `dismiss_insight` |
| `POST /api/v1/insights/:insightID/restore/`<br>insighthandler.restore | Pending: Restore an insight that was dismissed. |

### integration

| Write | Decision |
| --- | --- |
| `POST /api/v1/integrations/:type/test-connection/`<br>integrationhandler.testConnection | Exempt, configuration: Tests an integration's credentials and records the connection status on its configuration. |
| `POST /api/v1/integrations/samsara/workers/sync/`<br>integrationhandler.startWorkerSync | Pending: Sync workers from the telematics provider now. |
| `POST /api/v1/integrations/samsara/workers/sync/drift/detect/`<br>integrationhandler.detectWorkerSyncDrift | Exempt, read-only: Compares workers with the telematics provider and reports the drift. |
| `POST /api/v1/integrations/samsara/workers/sync/drift/repair/`<br>integrationhandler.repairWorkerSyncDrift | Pending: Repair the drift found between workers and the telematics provider. |
| `PUT /api/v1/integrations/:type/config/`<br>integrationhandler.updateConfig | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |

### invoice

| Write | Decision |
| --- | --- |
| `mutation createInvoiceFromOrder`<br>twin `POST /api/v1/billing/invoices/from-order/` | Pending: Create invoice from order. |
| `mutation createInvoiceFromShipments`<br>twin `POST /api/v1/billing/invoices/from-shipments/` | Pending: Create invoice from shipments. |
| `mutation createInvoicesFromOrder` | Pending: Create invoices from order. |
| `mutation createInvoicesFromShipments` | Pending: Create invoices from shipments. |
| `mutation createMemo`<br>twin `POST /api/v1/billing/invoices/memos/` | Pending: Create memo. |
| `mutation sendInvoiceEdi` | Pending: Send invoice EDI. |
| `mutation voidInvoice`<br>twin `POST /api/v1/billing/invoices/:invoiceID/void/` | Pending: Void invoice. |
| `PATCH /api/v1/billing/invoices/:invoiceID/`<br>invoicehandler.updateDraft | Pending: Update draft (invoice). |
| `POST /api/v1/billing/invoices/:invoiceID/generate-pdf/`<br>invoicehandler.generatePDF | Pending: Generate pdf (invoice). |
| `POST /api/v1/billing/invoices/:invoiceID/post/`<br>invoicehandler.post | Tool: `post_invoice` |
| `POST /api/v1/billing/invoices/:invoiceID/preview/`<br>invoicehandler.preview | Exempt, read-only: Renders an invoice for review and saves nothing. |
| `POST /api/v1/billing/invoices/:invoiceID/send/`<br>invoicehandler.send<br>also `POST /api/v1/billing/invoices/:invoiceID/resend/` | Tool: `send_invoice` |

### invoiceadjustment

| Write | Decision |
| --- | --- |
| `mutation approveInvoiceAdjustment`<br>twin `POST /api/v1/billing/invoice-adjustments/:adjustmentID/approve/` | Pending: Approve invoice adjustment. |
| `mutation rejectInvoiceAdjustment`<br>twin `POST /api/v1/billing/invoice-adjustments/:adjustmentID/reject/` | Pending: Reject invoice adjustment. |
| `PATCH /api/v1/billing/invoice-adjustments/drafts/:adjustmentID/`<br>invoiceadjustmenthandler.updateDraft | Pending: Update draft (invoice adjustment). |
| `POST /api/v1/billing/invoice-adjustments/bulk-preview/`<br>invoiceadjustmenthandler.bulkPreview | Exempt, read-only: Previews an invoice adjustment and saves nothing. |
| `POST /api/v1/billing/invoice-adjustments/bulk-submit/`<br>invoiceadjustmenthandler.bulkSubmit | Pending: Bulk submit (invoice adjustment). |
| `POST /api/v1/billing/invoice-adjustments/drafts/`<br>invoiceadjustmenthandler.createDraft | Pending: Create draft (invoice adjustment). |
| `POST /api/v1/billing/invoice-adjustments/drafts/:adjustmentID/preview/`<br>invoiceadjustmenthandler.previewDraft | Exempt, read-only: Previews an invoice adjustment and saves nothing. |
| `POST /api/v1/billing/invoice-adjustments/drafts/:adjustmentID/submit/`<br>invoiceadjustmenthandler.submitDraft | Pending: Submit draft (invoice adjustment). |
| `POST /api/v1/billing/invoice-adjustments/preview/`<br>invoiceadjustmenthandler.preview | Exempt, read-only: Previews an invoice adjustment and saves nothing. |
| `POST /api/v1/billing/invoice-adjustments/submit/`<br>invoiceadjustmenthandler.submit | Pending: Submit an invoice adjustment. |

### invoiceadjustmentcontrol

| Write | Decision |
| --- | --- |
| `PUT /api/v1/invoice-adjustment-controls/`<br>invoiceadjustmentcontrolhandler.update | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |

### invoicedispute

| Write | Decision |
| --- | --- |
| `mutation openInvoiceDispute` | Pending: Open invoice dispute. |
| `mutation resolveInvoiceDispute` | Pending: Resolve invoice dispute. |
| `mutation withdrawInvoiceDispute` | Pending: Withdraw invoice dispute. |

### invoicerun

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/billing/invoice-runs/:runID/membership/`<br>invoicerunhandler.adjustMembership | Pending: Adjust membership (invoice run). |
| `POST /api/v1/billing/invoice-runs/:runID/cancel/`<br>invoicerunhandler.cancel | Pending: Cancel an invoice run. |
| `POST /api/v1/billing/invoice-runs/:runID/commit/`<br>invoicerunhandler.commit | Pending: Commit an invoice run. |
| `POST /api/v1/billing/invoice-runs/preview/`<br>invoicerunhandler.preview | Pending: Create an invoice run in preview for review before committing it. |
| `POST /api/v1/billing/statements/:customerID/bill/`<br>invoicerunhandler.billStatement | Pending: Bill statement (statement). |

### invoiceshare

| Write | Decision |
| --- | --- |
| `POST /api/v1/billing/invoices/:invoiceID/shares/`<br>invoicesharehandler.share | Pending: Share an invoice. |

### journalreversal

| Write | Decision |
| --- | --- |
| `POST /api/v1/accounting/journal-reversals/`<br>journalreversalhandler.create | Pending: Create a journal reversal. |
| `POST /api/v1/accounting/journal-reversals/:reversalID/approve/`<br>journalreversalhandler.approve | Pending: Approve a journal reversal. |
| `POST /api/v1/accounting/journal-reversals/:reversalID/cancel/`<br>journalreversalhandler.cancel | Pending: Cancel a journal reversal. |
| `POST /api/v1/accounting/journal-reversals/:reversalID/post/`<br>journalreversalhandler.post | Pending: Post a journal reversal. |
| `POST /api/v1/accounting/journal-reversals/:reversalID/reject/`<br>journalreversalhandler.reject | Pending: Reject a journal reversal. |

### jurisdictionrule

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/jurisdiction-rule-overrides/:overrideID/`<br>jurisdictionrulehandler.deleteOverride | Exempt, configuration: Jurisdiction rules encode permit and tax regulation per state; an administrator maintains and verifies them. |
| `POST /api/v1/jurisdiction-rule-overrides/`<br>jurisdictionrulehandler.createOverride | Exempt, configuration: Jurisdiction rules encode permit and tax regulation per state; an administrator maintains and verifies them. |
| `POST /api/v1/jurisdiction-rules/`<br>jurisdictionrulehandler.create | Exempt, configuration: Jurisdiction rules encode permit and tax regulation per state; an administrator maintains and verifies them. |
| `POST /api/v1/jurisdiction-rules/:ruleID/verify/`<br>jurisdictionrulehandler.verify | Exempt, configuration: Jurisdiction rules encode permit and tax regulation per state; an administrator maintains and verifies them. |
| `PUT /api/v1/jurisdiction-rule-overrides/:overrideID/`<br>jurisdictionrulehandler.updateOverride | Exempt, configuration: Jurisdiction rules encode permit and tax regulation per state; an administrator maintains and verifies them. |
| `PUT /api/v1/jurisdiction-rules/:ruleID/`<br>jurisdictionrulehandler.update | Exempt, configuration: Jurisdiction rules encode permit and tax regulation per state; an administrator maintains and verifies them. |

### latecharge

| Write | Decision |
| --- | --- |
| `mutation assessLateCharges` | Pending: Assess late charges. |

### location

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/locations/:locationID/`<br>locationhandler.patch | Pending: Update some fields of a location. |
| `POST /api/v1/locations/`<br>locationhandler.create | Tool: `create_location` |
| `POST /api/v1/locations/bulk-update-status/`<br>locationhandler.bulkUpdateStatus | Pending: Change the status of several locations at once. |
| `PUT /api/v1/locations/:locationID/`<br>locationhandler.update | Pending: Update a location. |

### locationcategory

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/location-categories/:locationCategoryID/`<br>locationcategoryhandler.patch | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `POST /api/v1/location-categories/`<br>locationcategoryhandler.create | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `PUT /api/v1/location-categories/:locationCategoryID/`<br>locationcategoryhandler.update | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |

### manualjournal

| Write | Decision |
| --- | --- |
| `POST /api/v1/accounting/manual-journals/:requestID/approve/`<br>manualjournalhandler.approve | Pending: Approve a manual journal. |
| `POST /api/v1/accounting/manual-journals/:requestID/cancel/`<br>manualjournalhandler.cancel | Pending: Cancel a manual journal. |
| `POST /api/v1/accounting/manual-journals/:requestID/post/`<br>manualjournalhandler.post | Pending: Post a manual journal. |
| `POST /api/v1/accounting/manual-journals/:requestID/reject/`<br>manualjournalhandler.reject | Pending: Reject a manual journal. |
| `POST /api/v1/accounting/manual-journals/:requestID/submit/`<br>manualjournalhandler.submit | Pending: Submit a manual journal. |
| `POST /api/v1/accounting/manual-journals/drafts/`<br>manualjournalhandler.createDraft | Pending: Create draft (manual journal). |
| `PUT /api/v1/accounting/manual-journals/drafts/:requestID/`<br>manualjournalhandler.updateDraft | Pending: Update draft (manual journal). |

### notification

| Write | Decision |
| --- | --- |
| `mutation dismissNotifications` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation markAllNotificationsRead` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation markNotificationsRead` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation markNotificationsUnread` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation restoreNotifications` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |

### order

| Write | Decision |
| --- | --- |
| `mutation addOrderCharge` | Pending: Add order charge. |
| `mutation attachOrderShipments` | Pending: Attach order shipments. |
| `mutation cancelOrder` | Pending: Cancel order. |
| `mutation closeOrder` | Pending: Close order. |
| `mutation createOrder`<br>twin `POST /api/v1/orders/` | Pending: Create order. |
| `mutation detachOrderShipment` | Pending: Detach order shipment. |
| `mutation removeOrderCharge` | Pending: Remove order charge. |
| `mutation setOrderChargeAllocations` | Pending: Set order charge allocations. |
| `mutation updateOrder`<br>twin `PATCH /api/v1/orders/:orderID/`<br>twin `PUT /api/v1/orders/:orderID/` | Pending: Update order. |
| `mutation updateOrderCharge` | Pending: Update order charge. |

### organization

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/organizations/:id/logo`<br>organizationhandler.deleteLogo | Exempt, configuration: The organization's logo on every document it sends. |
| `POST /api/v1/organizations/:id/logo`<br>organizationhandler.uploadLogo | Exempt, infrastructure: Moves file bytes from a browser into storage; an agent attaches documents that already exist (attach_document_to_shipment). |
| `PUT /api/v1/organizations/:id/microsoft-sso`<br>organizationhandler.upsertMicrosoftSSOConfig | Exempt, security: Single sign-on decides who can sign in to the organization. |
| `PUT /api/v1/organizations/:id/okta-sso`<br>organizationhandler.upsertOktaSSOConfig | Exempt, security: Single sign-on decides who can sign in to the organization. |

### orgholiday

| Write | Decision |
| --- | --- |
| `mutation createOrgHoliday` | Exempt, configuration: The holiday calendar drives scheduling and pay rules for the whole organization. |
| `mutation deleteOrgHoliday` | Exempt, configuration: The holiday calendar drives scheduling and pay rules for the whole organization. |
| `mutation updateOrgHoliday` | Exempt, configuration: The holiday calendar drives scheduling and pay rules for the whole organization. |

### orgstructure

| Write | Decision |
| --- | --- |
| `mutation assignUserPosition` | Pending: Assign user position. |
| `mutation assignWorkerPosition` | Pending: Assign worker position. |
| `mutation createJobPosition` | Exempt, configuration: The organization chart that approvals and scoping follow. |
| `mutation delegateApproval` | Exempt, security: Hands a person's approval authority to someone else. |
| `mutation revokeApprovalDelegation` | Exempt, security: Withdraws approval authority handed to someone else. |
| `mutation updateJobPosition` | Exempt, configuration: The organization chart that approvals and scoping follow. |

### pagefavorite

| Write | Decision |
| --- | --- |
| `POST /api/v1/page-favorites/toggle`<br>pagefavoritehandler.toggle | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |

### performancereview

| Write | Decision |
| --- | --- |
| `mutation acknowledgeMyReview` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation archivePerformanceReviewTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation closePerformanceReview` | Pending: Close performance review. |
| `mutation createPerformanceReview` | Pending: Create performance review. |
| `mutation createPerformanceReviewTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation deletePerformanceReview` | Pending: Delete performance review. |
| `mutation reopenPerformanceReview` | Pending: Reopen performance review. |
| `mutation restorePerformanceReviewTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation submitPerformanceReview` | Pending: Submit performance review. |
| `mutation updatePerformanceReview` | Pending: Update performance review. |
| `mutation updatePerformanceReviewTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |

### permission

| Write | Decision |
| --- | --- |
| `POST /api/v1/me/permissions/check`<br>permissionhandler.checkBatch | Exempt, read-only: Checks which permissions the caller holds. |

### permit

| Write | Decision |
| --- | --- |
| `POST /api/v1/shipments/:shipmentID/permit-requirements/:requirementID/waive/`<br>permithandler.waiveRequirement | Pending: Waive requirement (shipment). |
| `POST /api/v1/shipments/:shipmentID/permits/`<br>permithandler.createPermit | Pending: Create permit (shipment). |
| `PUT /api/v1/shipments/:shipmentID/permits/:permitID/`<br>permithandler.updatePermit | Pending: Update permit (shipment). |

### ptopolicy

| Write | Decision |
| --- | --- |
| `mutation adjustWorkerPtoBalance` | Pending: Adjust worker PTO balance. |
| `mutation archivePtoPolicy` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation assignWorkerPtoPolicy` | Pending: Assign worker PTO policy. |
| `mutation createPtoPolicy` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation endWorkerPtoPolicyAssignment` | Pending: End worker PTO policy assignment. |
| `mutation restorePtoPolicy` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation runPtoAccrual` | Pending: Run PTO accrual. |
| `mutation updatePtoPolicy` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |

### push

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/push/subscriptions/`<br>pushhandler.unsubscribe | Exempt, infrastructure: Registers or removes a browser push subscription for the device the person is using. |
| `POST /api/v1/push/subscriptions/`<br>pushhandler.subscribe | Exempt, infrastructure: Registers or removes a browser push subscription for the device the person is using. |

### rateagreement

| Write | Decision |
| --- | --- |
| `POST /api/v1/rate-agreements/`<br>rateagreementhandler.create | Pending: Create a rate agreement. |
| `POST /api/v1/rate-agreements/:rateAgreementID/approve/`<br>rateagreementhandler.review | Pending: Approve a rate agreement. |
| `POST /api/v1/rate-agreements/:rateAgreementID/archive/`<br>rateagreementhandler.review | Pending: Archive a rate agreement. |
| `POST /api/v1/rate-agreements/:rateAgreementID/duplicate/`<br>rateagreementhandler.duplicate | Pending: Duplicate a rate agreement. |
| `POST /api/v1/rate-agreements/:rateAgreementID/reject/`<br>rateagreementhandler.review | Pending: Reject a rate agreement. |
| `POST /api/v1/rate-agreements/:rateAgreementID/resume/`<br>rateagreementhandler.review | Pending: Resume a rate agreement. |
| `POST /api/v1/rate-agreements/:rateAgreementID/rules/amend/`<br>rateagreementhandler.amendRules | Pending: Amend the rating rules of an active rate agreement. |
| `POST /api/v1/rate-agreements/:rateAgreementID/submit/`<br>rateagreementhandler.review | Pending: Submit a rate agreement. |
| `POST /api/v1/rate-agreements/:rateAgreementID/suspend/`<br>rateagreementhandler.review | Pending: Suspend a rate agreement. |
| `POST /api/v1/rate-agreements/rate-increase/apply/`<br>rateagreementhandler.applyRateIncrease | Pending: Apply rate increase (rate agreement). |
| `POST /api/v1/rate-agreements/rate-increase/preview/`<br>rateagreementhandler.previewRateIncrease | Exempt, read-only: Plans a general rate increase for review and saves nothing. |
| `PUT /api/v1/rate-agreements/:rateAgreementID/`<br>rateagreementhandler.update | Pending: Update a rate agreement. |

### rateconfirmation

| Write | Decision |
| --- | --- |
| `POST /api/v1/rate-confirmations/:rateConfirmationID/confirm/`<br>rateconfirmationhandler.confirm | Tool: `record_rate_confirmation_confirmed` |
| `POST /api/v1/rate-confirmations/:rateConfirmationID/send/`<br>rateconfirmationhandler.send | Tool: `send_rate_confirmation` |
| `POST /api/v1/rate-confirmations/:rateConfirmationID/void/`<br>rateconfirmationhandler.void | Tool: `void_rate_confirmation` |
| `POST /api/v1/shipment-moves/:moveID/rate-confirmations/`<br>rateconfirmationhandler.generate | Tool: `generate_rate_confirmation` |

### rateconfirmationpublic

| Write | Decision |
| --- | --- |
| `POST /api/v1/rate-confirmation-links/:token/confirm/`<br>rateconfirmationpublichandler.confirm | Exempt, counterparty: The outside party answering through a public link; an agent acting for the organization must not answer for them. |

### rateimport

| Write | Decision |
| --- | --- |
| `POST /api/v1/rate-imports/`<br>rateimporthandler.upload | Exempt, infrastructure: Moves file bytes from a browser into storage; an agent attaches documents that already exist (attach_document_to_shipment). |
| `POST /api/v1/rate-imports/:rateImportID/commit/`<br>rateimporthandler.commit | Pending: Commit a rate import. |
| `POST /api/v1/rate-imports/:rateImportID/discard/`<br>rateimporthandler.discard | Pending: Discard a rate import. |

### ratematrix

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/rate-matrices/:rateMatrixID/`<br>ratematrixhandler.delete | Pending: Delete a rate matrice. |
| `POST /api/v1/rate-matrices/`<br>ratematrixhandler.create | Pending: Create a rate matrice. |
| `PUT /api/v1/rate-matrices/:rateMatrixID/`<br>ratematrixhandler.update | Pending: Update a rate matrice. |
| `PUT /api/v1/rate-matrices/:rateMatrixID/cells/`<br>ratematrixhandler.replaceCells | Pending: Replace cells (rate matrice). |

### ratequote

| Write | Decision |
| --- | --- |
| `POST /api/v1/rate-quotes/quote/`<br>ratequotehandler.quote | Exempt, read-only: Quotes a rate from the rating engine and saves nothing. |
| `POST /api/v1/rate-quotes/shipment/:shipmentID/explain/`<br>ratequotehandler.explain | Exempt, read-only: Explains how a shipment was rated and saves nothing. |
| `POST /api/v1/rate-quotes/shipment/:shipmentID/shop/`<br>ratequotehandler.shop | Exempt, read-only: Compares the rates every agreement would give a shipment and saves nothing. |

### ratesimulation

| Write | Decision |
| --- | --- |
| `POST /api/v1/rate-simulations/`<br>ratesimulationhandler.create | Pending: Run and save a rate simulation across past shipments. |

### ratezone

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/rate-zones/:rateZoneID/`<br>ratezonehandler.delete | Pending: Delete a rate zone. |
| `POST /api/v1/rate-zones/`<br>ratezonehandler.create | Pending: Create a rate zone. |
| `PUT /api/v1/rate-zones/:rateZoneID/`<br>ratezonehandler.update | Pending: Update a rate zone. |

### realtime

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/shipments/:shipmentID/comments/presence/`<br>realtimehandler.leaveShipmentComments | Exempt, infrastructure: Presence and typing signals a browser sends while a person is on the page. |
| `POST /api/v1/shipments/:shipmentID/comments/presence/`<br>realtimehandler.joinShipmentComments | Exempt, infrastructure: Presence and typing signals a browser sends while a person is on the page. |
| `POST /api/v1/shipments/:shipmentID/comments/typing/`<br>realtimehandler.typingShipmentComments | Exempt, infrastructure: Presence and typing signals a browser sends while a person is on the page. |

### recurringshipment

| Write | Decision |
| --- | --- |
| `POST /api/v1/recurring-shipments/`<br>recurringshipmenthandler.create | Pending: Create a recurring shipment. |
| `POST /api/v1/recurring-shipments/:recurringShipmentID/generate/`<br>recurringshipmenthandler.generate | Pending: Generate a recurring shipment. |
| `POST /api/v1/recurring-shipments/match/`<br>recurringshipmenthandler.match | Exempt, read-only: Finds the recurring shipment a new shipment matches and saves nothing. |
| `PUT /api/v1/recurring-shipments/:recurringShipmentID/`<br>recurringshipmenthandler.update | Pending: Update a recurring shipment. |
| `PUT /api/v1/recurring-shipments/:recurringShipmentID/status/`<br>recurringshipmenthandler.updateStatus | Pending: Update status (recurring shipment). |

### report

| Write | Decision |
| --- | --- |
| `mutation cancelReportRun` | Pending: Cancel report run. |
| `mutation createReportDashboard` | Tool: `create_dashboard` |
| `mutation createReportDefinition` | Tool: `create_report` |
| `mutation createReportSchedule` | Tool: `schedule_report` |
| `mutation createReportView` | Pending: Create report view. |
| `mutation deleteReportDashboard` | Pending: Delete report dashboard. |
| `mutation deleteReportDefinition` | Pending: Delete report definition. |
| `mutation deleteReportSchedule` | Pending: Delete report schedule. |
| `mutation deleteReportView` | Pending: Delete report view. |
| `mutation forkCannedReport` | Tool: `fork_report` |
| `mutation resetCannedFork` | Pending: Reset canned fork. |
| `mutation runReport` | Tool: `run_report` |
| `mutation updateReportDashboard` | Tool: `add_dashboard_tile` |
| `mutation updateReportDefinition` | Tool: `update_report` |
| `mutation updateReportSchedule` | Pending: Update report schedule. |
| `mutation updateReportView` | Pending: Update report view. |
| `POST /api/v1/reports/dashboards/:dashboardID/export/`<br>reporthandler.exportDashboard | Exempt, read-only: Renders a dashboard to a file for download. |

### role

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/roles/:roleID/permissions/:permID`<br>rolehandler.deletePermission | Exempt, security: Grants or removes access. An agent that could change access could widen its own. |
| `DELETE /api/v1/roles/assignments/:assignmentID`<br>rolehandler.unassignRole | Exempt, security: Grants or removes access. An agent that could change access could widen its own. |
| `DELETE /api/v1/roles/constraints/:constraintID`<br>rolehandler.deleteConstraint | Exempt, security: Grants or removes access. An agent that could change access could widen its own. |
| `DELETE /api/v1/roles/hierarchy/:edgeID`<br>rolehandler.deleteHierarchy | Exempt, security: Grants or removes access. An agent that could change access could widen its own. |
| `POST /api/v1/roles/`<br>rolehandler.create | Exempt, security: Grants or removes access. An agent that could change access could widen its own. |
| `POST /api/v1/roles/:roleID/assignments`<br>rolehandler.assignRole | Exempt, security: Grants or removes access. An agent that could change access could widen its own. |
| `POST /api/v1/roles/:roleID/permissions`<br>rolehandler.addPermission | Exempt, security: Grants or removes access. An agent that could change access could widen its own. |
| `POST /api/v1/roles/constraints`<br>rolehandler.saveConstraint<br>also `PUT /api/v1/roles/constraints/:constraintID` | Exempt, security: Grants or removes access. An agent that could change access could widen its own. |
| `POST /api/v1/roles/hierarchy`<br>rolehandler.upsertHierarchy | Exempt, security: Grants or removes access. An agent that could change access could widen its own. |
| `PUT /api/v1/roles/:roleID`<br>rolehandler.update | Exempt, security: Grants or removes access. An agent that could change access could widen its own. |
| `PUT /api/v1/roles/:roleID/permissions/:permID`<br>rolehandler.updatePermission | Exempt, security: Grants or removes access. An agent that could change access could widen its own. |

### routingguide

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/routing-guides/:guideID/`<br>routingguidehandler.delete | Pending: Delete a routing guide. |
| `POST /api/v1/routing-guides/`<br>routingguidehandler.create | Pending: Create a routing guide. |
| `PUT /api/v1/routing-guides/:guideID/`<br>routingguidehandler.update | Pending: Update a routing guide. |

### scheduling

| Write | Decision |
| --- | --- |
| `mutation assignWorkerShift` | Pending: Assign worker shift. |
| `mutation createShiftTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation endWorkerShiftAssignment` | Pending: End worker shift assignment. |
| `mutation proposeShiftSwap` | Pending: Propose shift swap. |
| `mutation setWorkerAvailabilityPreference` | Pending: Set worker availability preference. |
| `mutation transitionShiftSwap` | Pending: Transition shift swap. |
| `mutation updateShiftTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |

### selfservice

| Write | Decision |
| --- | --- |
| `mutation createWorkerPolicy` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation decideProfileChange` | Pending: Decide profile change. |
| `mutation updateWorkerPolicy` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |

### sequenceconfig

| Write | Decision |
| --- | --- |
| `PUT /api/v1/sequence-configs/`<br>sequenceconfighandler.update | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |

### servicefailure

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/service-failures/:serviceFailureID/`<br>servicefailurehandler.update<br>also `PUT /api/v1/service-failures/:serviceFailureID/` | Pending: Update a service failure. |
| `POST /api/v1/service-failures/`<br>servicefailurehandler.createManual | Pending: Create manual (service failure). |
| `POST /api/v1/service-failures/:serviceFailureID/edi-214-payload/`<br>servicefailurehandler.buildEDI214Payload | Exempt, read-only: Builds the EDI 214 a service failure would send and returns it. |
| `POST /api/v1/service-failures/:serviceFailureID/resolve/`<br>servicefailurehandler.resolve | Tool: `resolve_service_failure` |
| `POST /api/v1/service-failures/:serviceFailureID/review/`<br>servicefailurehandler.review | Pending: Review a service failure. |
| `POST /api/v1/service-failures/:serviceFailureID/void/`<br>servicefailurehandler.void | Pending: Void a service failure. |
| `POST /api/v1/service-failures/bulk-evaluate/`<br>servicefailurehandler.bulkEvaluate | Pending: Evaluate several shipments for service failures at once. |
| `POST /api/v1/service-failures/evaluate-shipment/:shipmentID/`<br>servicefailurehandler.evaluateShipment | Tool: `evaluate_service_failures` |
| `POST /api/v1/service-failures/evaluate-stop/:shipmentID/:stopID/`<br>servicefailurehandler.evaluateStop | Pending: Evaluate one stop for a service failure. |

### servicefailurereasoncode

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/service-failure-reason-codes/:reasonCodeID/`<br>servicefailurereasoncodehandler.patch | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `POST /api/v1/service-failure-reason-codes/`<br>servicefailurereasoncodehandler.create | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `POST /api/v1/service-failure-reason-codes/:reasonCodeID/activate/`<br>servicefailurereasoncodehandler.activate | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `POST /api/v1/service-failure-reason-codes/:reasonCodeID/archive/`<br>servicefailurereasoncodehandler.archive | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `POST /api/v1/service-failure-reason-codes/reorder/`<br>servicefailurereasoncodehandler.reorder | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `PUT /api/v1/service-failure-reason-codes/:reasonCodeID/`<br>servicefailurereasoncodehandler.update | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |

### servicetype

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/service-types/:serviceTypeID/`<br>servicetypehandler.patch | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `POST /api/v1/service-types/`<br>servicetypehandler.create | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `POST /api/v1/service-types/bulk-update-status/`<br>servicetypehandler.bulkUpdateStatus | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `PUT /api/v1/service-types/:serviceTypeID/`<br>servicetypehandler.update | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |

### shipment

| Write | Decision |
| --- | --- |
| `mutation acknowledgeShipmentComment` | Pending: Acknowledge shipment comment. |
| `mutation autoRateShipment`<br>twin `POST /api/v1/shipments/:shipmentID/auto-rate/` | Pending: Auto rate shipment. |
| `mutation bulkTransferShipmentsToBilling`<br>twin `POST /api/v1/shipments/bulk-transfer-to-billing/` | Tool: `transfer_to_billing` |
| `mutation calculateShipmentDistance`<br>twin `POST /api/v1/shipments/calculate-distance/` | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `mutation calculateShipmentLoadingOptimization`<br>twin `POST /api/v1/shipments/loading-optimization/` | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `mutation calculateShipmentTotals`<br>twin `POST /api/v1/shipments/calculate-totals/` | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `mutation cancelShipment`<br>twin `POST /api/v1/shipments/:shipmentID/cancel/` | Tool: `cancel_shipment` |
| `mutation checkShipmentDuplicateBol` | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `mutation checkShipmentHazmatSegregation` | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `mutation createShipment`<br>twin `POST /api/v1/shipments/` | Tool: `create_shipment` |
| `mutation createShipmentComment`<br>twin `POST /api/v1/shipments/:shipmentID/comments/` | Tool: `add_shipment_comment` |
| `mutation deleteShipmentComment`<br>twin `DELETE /api/v1/shipments/:shipmentID/comments/:commentID/` | Pending: Delete shipment comment. |
| `mutation duplicateShipment`<br>twin `POST /api/v1/shipments/duplicate/` | Pending: Duplicate shipment. |
| `mutation pinShipmentComment` | Pending: Pin shipment comment. |
| `mutation previewShipmentContractRate` | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `mutation recalculateShipmentDistance`<br>twin `POST /api/v1/shipments/:shipmentID/recalculate-distance/` | Pending: Recalculate shipment distance. |
| `mutation resolveShipmentComment` | Pending: Resolve shipment comment. |
| `mutation transferShipmentOwnership`<br>twin `POST /api/v1/shipments/:shipmentID/transfer-ownership/` | Pending: Transfer shipment ownership. |
| `mutation transferShipmentToBilling`<br>twin `POST /api/v1/shipments/:shipmentID/transfer-to-billing/` | Tool: `transfer_to_billing` |
| `mutation transferShipmentToBillingItems` | Pending: Transfer chosen shipment charges to billing as separate items. |
| `mutation uncancelShipment`<br>twin `POST /api/v1/shipments/:shipmentID/uncancel/` | Pending: Uncancel shipment. |
| `mutation unpinShipmentComment` | Pending: Unpin shipment comment. |
| `mutation unresolveShipmentComment` | Pending: Unresolve shipment comment. |
| `mutation updateShipment`<br>twin `PUT /api/v1/shipments/:shipmentID/` | Tool: `update_shipment` |
| `mutation updateShipmentComment`<br>twin `PUT /api/v1/shipments/:shipmentID/comments/:commentID/` | Pending: Update shipment comment. |
| `POST /api/v1/shipments/:shipmentID/holds/`<br>shipmenthandler.createHold | Tool: `place_shipment_hold` |
| `POST /api/v1/shipments/:shipmentID/holds/:holdID/release/`<br>shipmenthandler.releaseHold | Tool: `release_shipment_hold` |
| `POST /api/v1/shipments/auto-cancel/`<br>shipmenthandler.autoCancelShipments | Pending: Cancel shipments that passed the auto-cancel threshold. |
| `POST /api/v1/shipments/check-for-duplicate-bols/`<br>shipmenthandler.checkForDuplicateBOLs | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `POST /api/v1/shipments/check-hazmat-segregation/`<br>shipmenthandler.checkHazmatSegregation | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `POST /api/v1/shipments/delay/`<br>shipmenthandler.delayShipments | Pending: Mark shipments as delayed. |
| `POST /api/v1/shipments/previous-rates/`<br>shipmenthandler.getPreviousRates | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `PUT /api/v1/shipments/:shipmentID/holds/:holdID/`<br>shipmenthandler.updateHold | Pending: Update hold (shipment). |

### shipmentcontrol

| Write | Decision |
| --- | --- |
| `PUT /api/v1/shipment-controls/`<br>shipmentcontrolhandler.update | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |

### shipmentmove

| Write | Decision |
| --- | --- |
| `POST /api/v1/shipment-moves/:moveID/split/`<br>shipmentmovehandler.splitMove | Pending: Split a two-stop move at a relay point into two moves. Left without a tool: the split needs a relay location and two new scheduled windows the system holds nowhere, so a model would have to invent the times, and the service checks only their order. |
| `POST /api/v1/shipment-moves/:moveID/stops/:stopID/record-actual/`<br>shipmentmovehandler.recordStopActual | Tool: `record_stop_actual` |
| `POST /api/v1/shipment-moves/:moveID/update-status/`<br>shipmentmovehandler.updateStatus | Tool: `update_move_status` |
| `POST /api/v1/shipment-moves/bulk-update-status/`<br>shipmentmovehandler.bulkUpdateStatus | Tool: `update_move_status` |

### shipmenttype

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/shipment-types/:shipmentTypeID/`<br>shipmenttypehandler.patch | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `POST /api/v1/shipment-types/`<br>shipmenttypehandler.create | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `POST /api/v1/shipment-types/bulk-update-status/`<br>shipmenttypehandler.bulkUpdateStatus | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `PUT /api/v1/shipment-types/:shipmentTypeID/`<br>shipmenttypehandler.update | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |

### sidebarpreference

| Write | Decision |
| --- | --- |
| `mutation updateSidebarPreferences` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |

### storedmileage

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/stored-mileages/:storedMileageID/`<br>storedmileagehandler.delete | Pending: Delete a stored mileage. |

### tablechangealert

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/tca/subscriptions/:id`<br>tablechangealerthandler.deleteSubscription | Pending: Delete subscription (table change alert). |
| `PATCH /api/v1/tca/subscriptions/:id/pause`<br>tablechangealerthandler.pauseSubscription | Pending: Pause subscription (table change alert). |
| `PATCH /api/v1/tca/subscriptions/:id/resume`<br>tablechangealerthandler.resumeSubscription | Pending: Resume subscription (table change alert). |
| `POST /api/v1/tca/subscriptions/`<br>tablechangealerthandler.createSubscription | Tool: `create_table_change_alert` |
| `PUT /api/v1/tca/subscriptions/:id`<br>tablechangealerthandler.updateSubscription | Pending: Update subscription (table change alert). |

### tableconfiguration

| Write | Decision |
| --- | --- |
| `mutation createTableConfiguration` | Tool: `save_table_view` |
| `mutation deleteTableConfiguration` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation patchTableConfiguration` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation setDefaultTableConfiguration` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation setOrgDefaultTableConfiguration` | Exempt, configuration: Sets the table view everyone in the organization starts from. |
| `mutation updateTableConfiguration` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |

### tablequery

| Write | Decision |
| --- | --- |
| `POST /api/v1/tables/:resource/compose/`<br>tablequeryhandler.compose | Exempt, read-only: Composes a table view from a request and returns it; save_table_view saves one. |

### telematics

| Write | Decision |
| --- | --- |
| `mutation deleteTelematicsFormMapping` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |
| `mutation saveTelematicsFormMapping` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |
| `POST /api/v1/webhooks/samsara/:webhookToken/`<br>telematicshandler.handleProviderWebhook | Exempt, infrastructure: An inbound webhook a provider calls with a signed token, not a person. |
| `POST /api/v1/webhooks/telematics/:provider/:webhookToken/`<br>telematicshandler.handleTelematicsWebhook | Exempt, infrastructure: An inbound webhook a provider calls with a signed token, not a person. |

### tenant

| Write | Decision |
| --- | --- |
| `mutation updateOrganization`<br>twin `PUT /api/v1/organizations/:id` | Exempt, configuration: The organization's own profile and settings. |

### tender

| Write | Decision |
| --- | --- |
| `POST /api/v1/tenders/:tenderID/cancel/`<br>tenderhandler.cancel | Tool: `cancel_tender` |
| `POST /api/v1/tenders/offers/:offerID/respond/`<br>tenderhandler.recordResponse | Tool: `record_tender_response` |
| `POST /api/v1/tenders/spot/`<br>tenderhandler.createSpot | Tool: `tender_move_to_carriers` |
| `POST /api/v1/tenders/waterfall/`<br>tenderhandler.createWaterfall | Tool: `tender_move_to_routing_guide` |

### tenderpublic

| Write | Decision |
| --- | --- |
| `POST /api/v1/tender-offers/:token/accept/`<br>tenderpublichandler.accept | Exempt, counterparty: The outside party answering through a public link; an agent acting for the organization must not answer for them. |
| `POST /api/v1/tender-offers/:token/decline/`<br>tenderpublichandler.decline | Exempt, counterparty: The outside party answering through a public link; an agent acting for the organization must not answer for them. |

### timesheet

| Write | Decision |
| --- | --- |
| `mutation clockIn` | Exempt, counterparty: A worker clocking their own time; an agent must not record hours worked for a person. |
| `mutation clockOut` | Exempt, counterparty: A worker clocking their own time; an agent must not record hours worked for a person. |
| `mutation deleteTimeEntry` | Pending: Delete time entry. |
| `mutation generatePayrollExport` | Pending: Generate payroll export. |
| `mutation recordTimeEntry` | Pending: Record time entry. |
| `mutation transitionTimesheet` | Exempt, attestation: Submitting and approving hours for payroll is a sign-off by the worker and their manager. |
| `mutation voidPayrollExport` | Pending: Void payroll export. |

### tractor

| Write | Decision |
| --- | --- |
| `mutation bulkUpdateTractorStatus`<br>twin `POST /api/v1/tractors/bulk-update-status/` | Tool: `update_tractor_status` |
| `mutation createTractor`<br>twin `POST /api/v1/tractors/` | Pending: Create tractor. |
| `mutation locateTractor` | Pending: Locate tractor. |
| `mutation patchTractor`<br>twin `PATCH /api/v1/tractors/:tractorID/` | Pending: Update some fields of a tractor. |
| `mutation updateTractor`<br>twin `PUT /api/v1/tractors/:tractorID/` | Pending: Update tractor. |

### trailer

| Write | Decision |
| --- | --- |
| `mutation bulkUpdateTrailerStatus`<br>twin `POST /api/v1/trailers/bulk-update-status/` | Tool: `update_trailer_status` |
| `mutation createTrailer`<br>twin `POST /api/v1/trailers/` | Pending: Create trailer. |
| `mutation locateTrailer`<br>twin `POST /api/v1/trailers/:trailerID/locate/` | Pending: Locate trailer. |
| `mutation patchTrailer`<br>twin `PATCH /api/v1/trailers/:trailerID/` | Pending: Update some fields of a trailer. |
| `mutation updateTrailer`<br>twin `PUT /api/v1/trailers/:trailerID/` | Pending: Update trailer. |

### user

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/users/me/profile-picture/`<br>userhandler.deleteProfilePicture | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `PATCH /api/v1/users/:userID/`<br>userhandler.patch | Exempt, security: User accounts, their status, memberships and passwords decide who can sign in and to what. |
| `PATCH /api/v1/users/me/settings/`<br>userhandler.updateMySettings | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `POST /api/v1/users/:userID/permissions/simulate/`<br>userhandler.simulatePermissions | Exempt, read-only: Shows what a user could do with a set of roles and saves nothing. |
| `POST /api/v1/users/:userID/reset-password/`<br>userhandler.sendPasswordReset | Exempt, security: User accounts, their status, memberships and passwords decide who can sign in and to what. |
| `POST /api/v1/users/bulk-update-status/`<br>userhandler.bulkUpdateStatus | Exempt, security: User accounts, their status, memberships and passwords decide who can sign in and to what. |
| `POST /api/v1/users/me/change-password/`<br>userhandler.changeMyPassword | Exempt, security: User accounts, their status, memberships and passwords decide who can sign in and to what. |
| `POST /api/v1/users/me/profile-picture/`<br>userhandler.uploadProfilePicture | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `POST /api/v1/users/me/switch-organization/`<br>userhandler.switchOrganization | Exempt, security: User accounts, their status, memberships and passwords decide who can sign in and to what. |
| `PUT /api/v1/users/:userID/`<br>userhandler.update | Exempt, security: User accounts, their status, memberships and passwords decide who can sign in and to what. |
| `PUT /api/v1/users/:userID/organization-memberships/`<br>userhandler.replaceOrganizationMemberships | Exempt, security: User accounts, their status, memberships and passwords decide who can sign in and to what. |

### version

| Write | Decision |
| --- | --- |
| `POST /api/v1/system/check-updates`<br>versionhandler.checkUpdates | Exempt, read-only: Asks the release server whether a newer version exists and changes nothing. |

### watchtower

| Write | Decision |
| --- | --- |
| `mutation dismissWatchtowerItem` | Pending: Dismiss watchtower item. |
| `mutation handOffWatchtowerItem` | Pending: Hand off watchtower item. |
| `mutation markWatchtowerSeen` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |

### worker

| Write | Decision |
| --- | --- |
| `mutation approveWorkerPTO` | Tool: `approve_worker_pto` |
| `mutation bulkWorkerPTOAction` | Pending: Approve, reject or cancel several PTO requests at once. |
| `mutation cancelWorkerPTO` | Tool: `cancel_worker_pto` |
| `mutation createWorkerPTO` | Pending: Create worker PTO. |
| `mutation patchWorker`<br>twin `PATCH /api/v1/workers/:workerID/`<br>twin `PUT /api/v1/workers/:workerID/` | Pending: Update some fields of a worker. |
| `mutation rejectWorkerPTO` | Tool: `reject_worker_pto` |
| `mutation updateWorkerPTO` | Pending: Update worker PTO. |
| `POST /api/v1/workers/`<br>workerhandler.create | Pending: Create a worker. |

### workerchecklist

| Write | Decision |
| --- | --- |
| `mutation archiveWorkerChecklistTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation cancelWorkerChecklist` | Pending: Cancel worker checklist. |
| `mutation completeWorkerChecklistItem` | Pending: Complete worker checklist item. |
| `mutation createWorkerChecklistTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation markWorkerChecklistItemNotApplicable` | Pending: Mark worker checklist item not applicable. |
| `mutation reopenWorkerChecklistItem` | Pending: Reopen worker checklist item. |
| `mutation restoreWorkerChecklistTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation skipWorkerChecklistItem` | Pending: Skip worker checklist item. |
| `mutation startWorkerChecklist` | Pending: Start worker checklist. |
| `mutation updateWorkerChecklistTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |

### workercredential

| Write | Decision |
| --- | --- |
| `mutation archiveWorkerCredential` | Pending: Archive worker credential. |
| `mutation archiveWorkerCredentialType` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation attachWorkerCredentialDocument` | Pending: Attach worker credential document. |
| `mutation createWorkerCredential` | Pending: Create worker credential. |
| `mutation createWorkerCredentialType` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation restoreWorkerCredentialType` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation updateWorkerCredential` | Pending: Update worker credential. |
| `mutation updateWorkerCredentialType` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation verifyWorkerCredential` | Pending: Verify worker credential. |

### workerdqf

| Write | Decision |
| --- | --- |
| `mutation deleteEmploymentVerification` | Pending: Delete employment verification. |
| `mutation markEmploymentVerificationRequested` | Pending: Mark employment verification requested. |
| `mutation recordEmploymentVerification` | Pending: Record employment verification. |
| `mutation recordEmploymentVerificationFollowUp` | Pending: Record a follow-up on an employment verification request. |
| `mutation updateEmploymentVerification` | Pending: Update employment verification. |

### workerdrugalcohol

| Write | Decision |
| --- | --- |
| `mutation cancelDotRandomDraw` | Pending: Cancel DOT random draw. |
| `mutation cancelDotTest` | Pending: Cancel DOT test. |
| `mutation completeClearinghouseQuery` | Pending: Complete clearinghouse query. |
| `mutation createDotRandomPool` | Pending: Create DOT random pool. |
| `mutation finalizeDotRandomDraw` | Pending: Finalize DOT random draw. |
| `mutation recordClearinghouseQuery` | Pending: Record clearinghouse query. |
| `mutation recordDotTest` | Pending: Record DOT test. |
| `mutation recordDotTestResult` | Pending: Record DOT test result. |
| `mutation recordDotViolation` | Pending: Record DOT violation. |
| `mutation runDotRandomDraw` | Pending: Run DOT random draw. |
| `mutation updateDotRandomDrawEntry` | Pending: Update DOT random draw entry. |
| `mutation updateDotRandomPool` | Pending: Update DOT random pool. |
| `mutation updateDotViolation` | Pending: Update DOT violation. |

### workeremployment

| Write | Decision |
| --- | --- |
| `mutation amendWorkerEmploymentEvent` | Pending: Amend worker employment event. |
| `mutation recordWorkerEmploymentEvent` | Pending: Record worker employment event. |

### workerinjury

| Write | Decision |
| --- | --- |
| `mutation certifyOshaSummary` | Exempt, attestation: OSHA requires a company executive to certify the annual injury summary. |
| `mutation deleteWorkerInjury` | Pending: Delete worker injury. |
| `mutation recordWorkerInjury` | Pending: Record worker injury. |
| `mutation saveOshaSummary` | Pending: Save OSHA summary. |
| `mutation uncertifyOshaSummary` | Exempt, attestation: Withdraws the executive certification of the annual injury summary. |
| `mutation updateWorkerInjury` | Pending: Update worker injury. |

### workerleave

| Write | Decision |
| --- | --- |
| `mutation closeLeaveCase` | Pending: Close leave case. |
| `mutation decideLeaveCase` | Pending: Decide leave case. |
| `mutation deleteLeaveDay` | Pending: Delete leave day. |
| `mutation openLeaveCase` | Pending: Open leave case. |
| `mutation recordLeaveCertification` | Pending: Record leave certification. |
| `mutation recordLeaveDay` | Pending: Record leave day. |
| `mutation requestLeaveCertification` | Pending: Request leave certification. |
| `mutation updateLeaveCase` | Pending: Update leave case. |
| `mutation updateLeaveControl` | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |
| `mutation updateLeaveDay` | Pending: Update leave day. |

### workersafety

| Write | Decision |
| --- | --- |
| `mutation acknowledgeMyDisciplinaryAction` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation closeWorkerSafetyEvent` | Pending: Close worker safety event. |
| `mutation createWorkerSafetyEvent` | Pending: Create worker safety event. |
| `mutation deleteWorkerRecognition` | Pending: Delete worker recognition. |
| `mutation deleteWorkerSafetyEvent` | Pending: Delete worker safety event. |
| `mutation giveWorkerRecognition` | Pending: Give worker recognition. |
| `mutation issueDisciplinaryAction` | Pending: Issue disciplinary action. |
| `mutation reopenWorkerSafetyEvent` | Pending: Reopen worker safety event. |
| `mutation rescindDisciplinaryAction` | Pending: Rescind disciplinary action. |
| `mutation reviewWorkerSafetyEvent` | Pending: Review worker safety event. |
| `mutation updateWorkerSafetyEvent` | Pending: Update worker safety event. |

### workertraining

| Write | Decision |
| --- | --- |
| `mutation acknowledgeMyTraining` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation archiveTrainingCourse` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation assignRequiredWorkerTraining` | Pending: Assign required worker training. |
| `mutation assignWorkerTraining` | Pending: Assign worker training. |
| `mutation attachWorkerTrainingDocument` | Pending: Attach worker training document. |
| `mutation bulkAssignTraining` | Pending: Bulk assign training. |
| `mutation cancelWorkerTraining` | Pending: Cancel worker training. |
| `mutation completeWorkerTraining` | Pending: Complete worker training. |
| `mutation createTrainingCourse` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation restoreTrainingCourse` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation startMyTraining` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation updateTrainingCourse` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation waiveWorkerTraining` | Pending: Waive worker training. |
