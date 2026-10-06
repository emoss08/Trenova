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

## Schema contract

A tool that performs a GraphQL write takes what the mutation's input takes, and
no more. `internal/api/toolcontract` holds each such tool to its input: a
`Binding` in `bindings.go` names the tool, the input type, the parameter
the input sits under (none for the top level) and the input each nested object
parameter is held to (`moves.stops` to `ShipmentStopInput`).
`TestEveryBoundToolFitsItsGraphQLInput` parses the schema with gqlparser and
fails when

- a non-null input field without a default is not a required parameter, unless
  the binding's `Defaulted` says why (a patch asks only for what changes);
- a parameter is not a field of its input, unless `Extra` says why (the
  mutation takes the record's id as its own argument). This is the check a
  made-up field such as `moves[].type` fails;
- a parameter bound to a GraphQL enum does not offer exactly its values, unless
  `Narrowed` says why the tool offers fewer.

`TestEveryWriteToolWithAGraphQLInputIsBound` reads this ledger: every
create or update tool listed against a mutation that takes an input object is
bound, or named in `Unbound` with the reason. A tool whose writes are REST
routes has no input to hold it to and needs neither. An excuse that no longer
matches anything fails too, so the file cannot drift behind the tools.

## Exemption categories

| Category | Means | Writes |
| --- | --- | --- |
| `security` | Sign-in, sessions, passwords, API keys, SSO, identity providers, roles, permissions and every other grant of access. Never agent-operated: an agent that could widen access could widen its own. | 79 |
| `configuration` | Organization-wide settings, controls, lookup tables, templates and integration connections an administrator sets once and every later write depends on. | 260 |
| `user-preference` | A person's own interface state: saved table views, the sidebar, favorites, notification read state, a profile picture. | 27 |
| `infrastructure` | Plumbing a client, a provider or the platform drives rather than a decision a person makes: upload sessions, inbound webhooks, presence signals, the GraphQL transport, repair operations. | 46 |
| `agent-administration` | Defining, configuring, evaluating and overseeing agents, including deciding what they propose. An agent that did this would be grading its own work. | 70 |
| `counterparty` | Done by someone other than the organization's staff acting for themselves: a driver in their own portal, a customer or carrier through a public link. An agent acts for the organization and must not act as them. | 33 |
| `read-only` | Sent as a POST or a mutation but only computes, previews, validates or tests, and changes nothing. | 47 |
| `attestation` | A sign-off a named, accountable person must make: certifying a regulatory summary, filing a return, overriding a failed vetting. | 44 |
| `duplicate` | Another surface for a write listed elsewhere that the analysis could not merge on its own. The reason names the write it duplicates. | 3 |

## Totals

992 writes: 513 GraphQL mutations and 479 REST writes, after merging 72 REST routes into the mutation they duplicate.

| Decision | Writes |
| --- | --- |
| Covered by a tool | 380 |
| Exempt | 609 |
| — Security | 79 |
| — Configuration | 260 |
| — User preference | 27 |
| — Infrastructure | 46 |
| — Agent administration | 70 |
| — Counterparty | 33 |
| — Read-only | 47 |
| — Attestation | 44 |
| — Duplicate | 3 |
| **Pending** | **3** |
| Total | 992 |

Of the 383 writes an agent should be able to make, 380 have a tool (99%).

## Pending

The writes no tool performs yet, and what the tool would do.

| Domain | Write | What the tool would do |
| --- | --- | --- |
| billingqueue | `mutation releaseBillingQueueItem` | Take a billing queue item off hold and back to the status the hold found it in, the counterpart of hold_billing_queue_item. |
| billingqueue | `mutation resolveBillingQueueIssue` | Settle one of a billing queue item's checks with one of the options the check offers. |
| billingqueue | `mutation undoBillingQueueIssue` | Take back how a billing queue item's check was settled and put back any charge the settlement changed. |

## By domain

| Domain | Writes | Covered | Exempt | Pending |
| --- | --- | --- | --- | --- |
| accessorialcharge | 3 | 0 | 3 | 0 |
| accountingcontrol | 1 | 0 | 1 | 0 |
| accountingsync | 31 | 20 | 11 | 0 |
| accountingwebhook | 1 | 0 | 1 | 0 |
| accounttype | 4 | 0 | 4 | 0 |
| agent | 15 | 2 | 13 | 0 |
| agentdefinition | 7 | 0 | 7 | 0 |
| agentextension | 2 | 0 | 2 | 0 |
| agentquality | 6 | 0 | 6 | 0 |
| agentrun | 1 | 0 | 1 | 0 |
| aiaudit | 3 | 0 | 3 | 0 |
| aifeedback | 2 | 0 | 2 | 0 |
| aiprovider | 4 | 0 | 4 | 0 |
| airetrieval | 2 | 0 | 2 | 0 |
| apikey | 4 | 0 | 4 | 0 |
| assignment | 1 | 0 | 1 | 0 |
| assistant | 20 | 0 | 20 | 0 |
| auth | 7 | 0 | 7 | 0 |
| bankreceipt | 2 | 1 | 1 | 0 |
| bankreceiptbatch | 1 | 0 | 1 | 0 |
| bankreceiptworkitem | 4 | 4 | 0 | 0 |
| benefits | 4 | 0 | 4 | 0 |
| billingcontrol | 1 | 0 | 1 | 0 |
| billingqueue | 14 | 7 | 4 | 3 |
| billingtransfer | 3 | 3 | 0 | 0 |
| briefing | 2 | 0 | 2 | 0 |
| capture | 25 | 4 | 21 | 0 |
| carrier | 4 | 4 | 0 | 0 |
| carrierintelligence | 15 | 9 | 6 | 0 |
| carriersettlement | 16 | 15 | 1 | 0 |
| commodity | 4 | 4 | 0 | 0 |
| costing | 2 | 0 | 2 | 0 |
| customer | 4 | 4 | 0 | 0 |
| customerpayment | 5 | 5 | 0 | 0 |
| customfield | 4 | 0 | 4 | 0 |
| databasesession | 1 | 0 | 1 | 0 |
| dataentrycontrol | 1 | 0 | 1 | 0 |
| dataretention | 1 | 0 | 1 | 0 |
| decisions | 1 | 0 | 1 | 0 |
| deskmemory | 6 | 2 | 4 | 0 |
| detention | 8 | 4 | 4 | 0 |
| detentionpolicy | 1 | 0 | 1 | 0 |
| dispatchconsole | 5 | 4 | 1 | 0 |
| dispatchcontrol | 1 | 0 | 1 | 0 |
| distancecontrol | 2 | 0 | 2 | 0 |
| distanceoverride | 4 | 0 | 4 | 0 |
| distanceprofile | 5 | 0 | 5 | 0 |
| document | 15 | 4 | 11 | 0 |
| documentcontrol | 1 | 0 | 1 | 0 |
| documentoperations | 3 | 0 | 3 | 0 |
| documentpacketrule | 3 | 0 | 3 | 0 |
| documentparsingrule | 9 | 0 | 9 | 0 |
| documenttemplate | 12 | 0 | 12 | 0 |
| documenttype | 3 | 0 | 3 | 0 |
| driverportal | 31 | 3 | 28 | 0 |
| driversettlement | 34 | 29 | 5 | 0 |
| edi | 56 | 15 | 41 | 0 |
| email | 9 | 0 | 9 | 0 |
| equipmentmanufacturer | 4 | 0 | 4 | 0 |
| equipmenttype | 4 | 0 | 4 | 0 |
| exchangerate | 2 | 0 | 2 | 0 |
| extractioneval | 5 | 0 | 5 | 0 |
| extractionrollout | 1 | 0 | 1 | 0 |
| extractionshadow | 1 | 0 | 1 | 0 |
| fiscalperiod | 9 | 5 | 4 | 0 |
| fiscalyear | 7 | 0 | 7 | 0 |
| fleetcode | 3 | 0 | 3 | 0 |
| fleetsafety | 3 | 3 | 0 | 0 |
| formulatemplate | 24 | 0 | 24 | 0 |
| fuelpurchase | 13 | 7 | 6 | 0 |
| fuelsurcharge | 9 | 2 | 7 | 0 |
| glaccount | 5 | 0 | 5 | 0 |
| googlemaps | 1 | 0 | 1 | 0 |
| graphql | 1 | 0 | 1 | 0 |
| hazardousmaterial | 4 | 4 | 0 | 0 |
| hazmatsegregationrule | 3 | 0 | 3 | 0 |
| holdreason | 3 | 0 | 3 | 0 |
| homelayout | 5 | 1 | 4 | 0 |
| iam | 14 | 0 | 14 | 0 |
| ifta | 14 | 9 | 5 | 0 |
| inbound | 1 | 0 | 1 | 0 |
| inboundmessage | 7 | 2 | 5 | 0 |
| insight | 2 | 2 | 0 | 0 |
| integration | 5 | 0 | 5 | 0 |
| invoice | 12 | 11 | 1 | 0 |
| invoiceadjustment | 10 | 7 | 3 | 0 |
| invoiceadjustmentcontrol | 1 | 0 | 1 | 0 |
| invoicedispute | 3 | 3 | 0 | 0 |
| invoicerun | 5 | 5 | 0 | 0 |
| invoiceshare | 1 | 1 | 0 | 0 |
| journalentry | 2 | 0 | 2 | 0 |
| journalreversal | 5 | 3 | 2 | 0 |
| jurisdictionrule | 6 | 0 | 6 | 0 |
| latecharge | 1 | 1 | 0 | 0 |
| location | 4 | 4 | 0 | 0 |
| locationcategory | 3 | 0 | 3 | 0 |
| manualjournal | 7 | 5 | 2 | 0 |
| notification | 5 | 0 | 5 | 0 |
| onboarding | 1 | 0 | 1 | 0 |
| order | 10 | 10 | 0 | 0 |
| organization | 4 | 0 | 4 | 0 |
| orgholiday | 3 | 0 | 3 | 0 |
| orgstructure | 6 | 0 | 6 | 0 |
| pagefavorite | 1 | 0 | 1 | 0 |
| performancereview | 11 | 3 | 8 | 0 |
| permission | 1 | 0 | 1 | 0 |
| permit | 3 | 2 | 1 | 0 |
| ptopolicy | 8 | 1 | 7 | 0 |
| push | 2 | 0 | 2 | 0 |
| rateagreement | 12 | 11 | 1 | 0 |
| rateconfirmation | 4 | 4 | 0 | 0 |
| rateconfirmationpublic | 1 | 0 | 1 | 0 |
| rateimport | 3 | 2 | 1 | 0 |
| ratematrix | 4 | 0 | 4 | 0 |
| ratequote | 3 | 0 | 3 | 0 |
| ratesimulation | 1 | 1 | 0 | 0 |
| ratezone | 3 | 0 | 3 | 0 |
| realtime | 3 | 0 | 3 | 0 |
| recurringshipment | 5 | 4 | 1 | 0 |
| report | 17 | 13 | 4 | 0 |
| role | 11 | 0 | 11 | 0 |
| routingguide | 3 | 0 | 3 | 0 |
| scheduling | 7 | 5 | 2 | 0 |
| selfservice | 3 | 0 | 3 | 0 |
| sequenceconfig | 1 | 0 | 1 | 0 |
| servicefailure | 9 | 7 | 2 | 0 |
| servicefailurereasoncode | 6 | 0 | 6 | 0 |
| servicetype | 4 | 0 | 4 | 0 |
| shipment | 33 | 21 | 12 | 0 |
| shipmentcontrol | 1 | 0 | 1 | 0 |
| shipmentmove | 4 | 4 | 0 | 0 |
| shipmenttype | 4 | 0 | 4 | 0 |
| sidebarpreference | 1 | 0 | 1 | 0 |
| storedmileage | 1 | 0 | 1 | 0 |
| tablechangealert | 5 | 5 | 0 | 0 |
| tableconfiguration | 6 | 1 | 5 | 0 |
| tablequery | 1 | 0 | 1 | 0 |
| telematics | 4 | 0 | 4 | 0 |
| tenant | 1 | 0 | 1 | 0 |
| tender | 4 | 4 | 0 | 0 |
| tenderpublic | 2 | 0 | 2 | 0 |
| timesheet | 7 | 2 | 5 | 0 |
| tractor | 5 | 5 | 0 | 0 |
| trailer | 5 | 5 | 0 | 0 |
| user | 16 | 0 | 16 | 0 |
| version | 1 | 0 | 1 | 0 |
| watchtower | 3 | 1 | 2 | 0 |
| worker | 8 | 5 | 3 | 0 |
| workerchecklist | 10 | 6 | 4 | 0 |
| workercredential | 9 | 4 | 5 | 0 |
| workerdqf | 5 | 5 | 0 | 0 |
| workerdrugalcohol | 13 | 6 | 7 | 0 |
| workeremployment | 2 | 0 | 2 | 0 |
| workerinjury | 6 | 3 | 3 | 0 |
| workerleave | 10 | 7 | 3 | 0 |
| workersafety | 11 | 8 | 3 | 0 |
| workertraining | 13 | 7 | 6 | 0 |

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
| `mutation applyAccountingInboundChange` | Tool: `apply_accounting_inbound_change` |
| `mutation changeAccountingBackfill` | Tool: `change_accounting_backfill` |
| `mutation checkAccountingConnection` | Tool: `check_accounting_connection` |
| `mutation checkAccountingDrift` | Tool: `check_accounting_drift` |
| `mutation chooseAccountingCompany` | Exempt, security: Finishes the OAuth sign-in by choosing the company the authorization is for; a person holds the grant, never an agent. |
| `mutation chooseAccountingSyncMode` | Exempt, configuration: Chooses what the organization's accounting connection sends during setup; an administrator owns the connection and it is fixed once sync is enabled. |
| `mutation clearAccountingMapping` | Tool: `clear_accounting_mapping` |
| `mutation completeAccountingAuthorization` | Exempt, security: Handles the OAuth app and its credentials that let Trenova act in the accounting system. |
| `mutation completeAccountingSetup` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |
| `mutation confirmAccountingMappings` | Tool: `confirm_accounting_mapping_proposals` |
| `mutation createAccountingReferenceRecord` | Tool: `create_accounting_reference_record` |
| `mutation disconnectAccountingSystem` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |
| `mutation dismissAccountingDrift` | Tool: `dismiss_accounting_drift` |
| `mutation enableAccountingSync` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |
| `mutation finishAccountingAuthorization` | Exempt, security: Handles the OAuth app and its credentials that let Trenova act in the accounting system. |
| `mutation ignoreAccountingInboundChange` | Tool: `ignore_accounting_inbound_change` |
| `mutation pauseAccountingSync` | Tool: `pause_accounting_sync` |
| `mutation redateAccountingSync` | Tool: `redate_accounting_sync` |
| `mutation refreshAccountingReferenceData` | Tool: `refresh_accounting_reference_data` |
| `mutation rejectAccountingMapping` | Tool: `reject_accounting_mapping_proposal` |
| `mutation releaseAccountingSync` | Tool: `release_accounting_sync` |
| `mutation removeAccountingApp` | Exempt, security: Handles the OAuth app and its credentials that let Trenova act in the accounting system. |
| `mutation requestAccountingBackfill` | Tool: `request_accounting_backfill` |
| `mutation resolveAccountingDrift` | Tool: `resolve_accounting_drift` |
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
| `POST /api/v1/webhooks/accounting/:provider/`<br>accountingwebhookhandler.receive<br>also `POST /api/v1/webhooks/accounting/:provider/:app/` | Exempt, infrastructure: An inbound webhook a provider calls with a signed token, not a person. |

### accounttype

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/account-types/:accountTypeID/`<br>accounttypehandler.patch | Exempt, configuration: The account types the chart of accounts is classified by for the financial statements; an accountant sets them up once and every account depends on them. |
| `POST /api/v1/account-types/`<br>accounttypehandler.create | Exempt, configuration: The account types the chart of accounts is classified by for the financial statements; an accountant sets them up once and every account depends on them. |
| `POST /api/v1/account-types/bulk-update-status/`<br>accounttypehandler.bulkUpdateStatus | Exempt, configuration: The account types the chart of accounts is classified by for the financial statements; an accountant sets them up once and every account depends on them. |
| `PUT /api/v1/account-types/:accountTypeID/`<br>accounttypehandler.update | Exempt, configuration: The account types the chart of accounts is classified by for the financial statements; an accountant sets them up once and every account depends on them. |

### agent

| Write | Decision |
| --- | --- |
| `mutation approveAgentMemorySuggestion` | Exempt, agent-administration: A person curating what agents remember; an agent writes memory through remember and forget_memory. |
| `mutation commitMyDecisionNow` | Exempt, agent-administration: Deciding what an agent proposed is the human check on agents; an agent cannot approve its own work. |
| `mutation createAgentMemory` | Tool: `remember` |
| `mutation decideAgentPlan`<br>twin `POST /api/v1/agent-plans/:planID/resolve/` | Exempt, agent-administration: Deciding what an agent proposed is the human check on agents; an agent cannot approve its own work. |
| `mutation decideAgentProposal`<br>twin `POST /api/v1/agent-proposals/:proposalID/resolve/` | Exempt, agent-administration: Deciding what an agent proposed is the human check on agents; an agent cannot approve its own work. |
| `mutation decideMyPlan` | Exempt, agent-administration: Deciding what an agent proposed is the human check on agents; an agent cannot approve its own work. |
| `mutation decideMyProposal` | Exempt, agent-administration: Deciding what an agent proposed is the human check on agents; an agent cannot approve its own work. |
| `mutation decideMyProposals` | Exempt, agent-administration: Deciding what an agent proposed is the human check on agents; an agent cannot approve its own work. |
| `mutation dismissAgentMemorySuggestion` | Exempt, agent-administration: A person curating what agents remember; an agent writes memory through remember and forget_memory. |
| `mutation replayAgentRun` | Exempt, agent-administration: Configures, evaluates or oversees the agents themselves; an agent doing it would be grading its own work. |
| `mutation resolveAgentException`<br>twin `POST /api/v1/agent-exceptions/:exceptionID/resolve/` | Exempt, agent-administration: An agent exception is an agent handing a case to a person; the person resolves it. |
| `mutation setAgentMemoryStatus` | Tool: `forget_memory` |
| `mutation undoMyDecision` | Exempt, agent-administration: Deciding what an agent proposed is the human check on agents; an agent cannot approve its own work. |
| `mutation updateAgentControl`<br>twin `PUT /api/v1/agent-controls/` | Exempt, agent-administration: The organization-wide switches and ceilings every agent runs under. |
| `mutation updateAgentMemory` | Exempt, agent-administration: A person curating what agents remember; an agent writes memory through remember and forget_memory. |

### agentdefinition

| Write | Decision |
| --- | --- |
| `mutation setAgentAccess` | Exempt, security: Decides which people may use which agents. |
| `mutation setRoleAgentAccess` | Exempt, security: Decides which roles may use which agents. |
| `mutation updateAgentCapabilities` | Exempt, agent-administration: Configures, evaluates or oversees the agents themselves; an agent doing it would be grading its own work. |
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
| `DELETE /api/v1/assistant/schedules/:scheduleID/`<br>assistanthandler.deleteSchedule | Exempt, agent-administration: Scheduling a conversation to run again is a person directing an agent's work; an agent that could schedule itself would set its own workload. |
| `DELETE /api/v1/assistant/threads/:threadID/`<br>assistanthandler.deleteThread | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `PATCH /api/v1/assistant/schedules/:scheduleID/`<br>assistanthandler.updateSchedule | Exempt, agent-administration: Scheduling a conversation to run again is a person directing an agent's work; an agent that could schedule itself would set its own workload. |
| `PATCH /api/v1/assistant/threads/:threadID/`<br>assistanthandler.updateThread | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/assistant/ask/`<br>assistanthandler.ask | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/assistant/schedules/:scheduleID/run/`<br>assistanthandler.runSchedule | Exempt, agent-administration: Scheduling a conversation to run again is a person directing an agent's work; an agent that could schedule itself would set its own workload. |
| `POST /api/v1/assistant/threads/`<br>assistanthandler.startThread | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/assistant/threads/:threadID/artifacts/:artifactID/pin/`<br>assistanthandler.pinArtifact | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/assistant/threads/:threadID/artifacts/:artifactID/restore/`<br>assistanthandler.restoreDocumentVersion | Exempt, agent-administration: A person keeping or restoring their own version of a document an agent drafted in their conversation; the agent drafts through the conversation itself. |
| `POST /api/v1/assistant/threads/:threadID/artifacts/:artifactID/rewrite/`<br>assistanthandler.rewriteDocument | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/assistant/threads/:threadID/artifacts/:artifactID/versions/`<br>assistanthandler.saveDocumentVersion | Exempt, agent-administration: A person keeping or restoring their own version of a document an agent drafted in their conversation; the agent drafts through the conversation itself. |
| `POST /api/v1/assistant/threads/:threadID/compact/`<br>assistanthandler.compactThread | Exempt, agent-administration: Compacting a conversation manages the agent's own context window; the runtime compacts on its own limits and no record changes. |
| `POST /api/v1/assistant/threads/:threadID/handoff/`<br>assistanthandler.handoff | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/assistant/threads/:threadID/messages/`<br>assistanthandler.sendMessage | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/assistant/threads/:threadID/read/`<br>assistanthandler.markThreadRead | Exempt, agent-administration: Marking a conversation read arranges the person's own Desk; it changes no record an agent acts on. |
| `POST /api/v1/assistant/threads/:threadID/requests/`<br>assistanthandler.requestMore | Exempt, agent-administration: Asking an administrator for access to an agent, or for more allowance, budget or daily requests, is a person's request about their own use of the assistant; an agent that could ask for its own limits to be raised would defeat them. |
| `POST /api/v1/assistant/threads/:threadID/schedules/`<br>assistanthandler.createSchedule | Exempt, agent-administration: Scheduling a conversation to run again is a person directing an agent's work; an agent that could schedule itself would set its own workload. |
| `POST /api/v1/assistant/threads/:threadID/turns/`<br>assistanthandler.startTurn | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/assistant/turns/:turnID/stop/`<br>assistanthandler.stopTurn | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `PUT /api/v1/assistant/threads/:threadID/proposals/:proposalID/edits/`<br>assistanthandler.saveProposalEdits | Exempt, agent-administration: Deciding what an agent proposed is the human check on agents; an agent cannot approve its own work. |

### auth

| Write | Decision |
| --- | --- |
| `POST /api/v1/auth/forgot-password`<br>authhandler.forgotPassword | Exempt, security: Signing in and out and resetting a password are a person proving who they are; an agent holds no credentials of its own. |
| `POST /api/v1/auth/login`<br>authhandler.login | Exempt, security: Signing in and out and resetting a password are a person proving who they are; an agent holds no credentials of its own. |
| `POST /api/v1/auth/logout`<br>authhandler.logout | Exempt, security: Signing in and out and resetting a password are a person proving who they are; an agent holds no credentials of its own. |
| `POST /api/v1/auth/mfa/verify`<br>authhandler.verifyMFA | Exempt, security: Answering a sign-in's second-factor challenge is a person proving who they are; an agent holds no credentials of its own. |
| `POST /api/v1/auth/reset-password`<br>authhandler.resetPassword | Exempt, security: Signing in and out and resetting a password are a person proving who they are; an agent holds no credentials of its own. |
| `POST /api/v1/auth/session/roles/activate`<br>authhandler.activateSessionRoles | Exempt, security: Signing in and out and resetting a password are a person proving who they are; an agent holds no credentials of its own. |
| `POST /api/v1/auth/validate-session`<br>authhandler.validateSession | Exempt, security: Signing in and out and resetting a password are a person proving who they are; an agent holds no credentials of its own. |

### bankreceipt

| Write | Decision |
| --- | --- |
| `POST /api/v1/accounting/bank-receipts/`<br>bankreceipthandler.importReceipt | Exempt, infrastructure: Records a line as the bank reported it, from its file or feed; a receipt is the bank's own record of cash that arrived, which an agent only ever matches and never writes. |
| `POST /api/v1/accounting/bank-receipts/:receiptID/match/`<br>bankreceipthandler.match | Tool: `match_bank_receipt` |

### bankreceiptbatch

| Write | Decision |
| --- | --- |
| `POST /api/v1/accounting/bank-receipt-batches/`<br>bankreceiptbatchhandler.importBatch | Exempt, infrastructure: Loads a bank file's receipts as the bank reported them; the bank's record of cash that arrived is imported, never written by an agent. |

### bankreceiptworkitem

| Write | Decision |
| --- | --- |
| `POST /api/v1/accounting/bank-receipt-work-items/:workItemID/assign/`<br>bankreceiptworkitemhandler.assign | Tool: `triage_bank_receipt_work_item` |
| `POST /api/v1/accounting/bank-receipt-work-items/:workItemID/dismiss/`<br>bankreceiptworkitemhandler.dismiss | Tool: `resolve_bank_receipt_work_item` |
| `POST /api/v1/accounting/bank-receipt-work-items/:workItemID/resolve/`<br>bankreceiptworkitemhandler.resolve | Tool: `resolve_bank_receipt_work_item` |
| `POST /api/v1/accounting/bank-receipt-work-items/:workItemID/start-review/`<br>bankreceiptworkitemhandler.startReview | Tool: `triage_bank_receipt_work_item` |

### benefits

| Write | Decision |
| --- | --- |
| `mutation createBenefitPlan` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation endBenefitEnrollment` | Exempt, attestation: Ending a benefit election is the worker's own choice, with the payroll deduction it stops; an agent must not make it for them. |
| `mutation enrollBenefit` | Exempt, attestation: Enrolling in a benefit is the worker's own election, with payroll deductions it starts; an agent must not make it for them. |
| `mutation updateBenefitPlan` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |

### billingcontrol

| Write | Decision |
| --- | --- |
| `PUT /api/v1/billing-controls/`<br>billingcontrolhandler.update | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |

### billingqueue

| Write | Decision |
| --- | --- |
| `mutation assignBillingQueueBiller`<br>twin `PUT /api/v1/billing-queue/:itemID/assign/` | Tool: `assign_billing_queue_biller`, `assign_billing_queue_billers` |
| `mutation postBillingQueueItem`<br>twin `POST /api/v1/billing-queue/:itemID/post/` | Tool: `post_invoice`, `send_invoice` |
| `mutation releaseBillingQueueItem`<br>twin `POST /api/v1/billing-queue/:itemID/release/` | Pending: Take a billing queue item off hold and back to the status the hold found it in, the counterpart of hold_billing_queue_item. |
| `mutation resolveBillingQueueIssue`<br>twin `POST /api/v1/billing-queue/:itemID/issues/:issueID/resolve/` | Pending: Settle one of a billing queue item's checks with one of the options the check offers. |
| `mutation undoBillingQueueIssue`<br>twin `POST /api/v1/billing-queue/:itemID/issues/:issueID/undo/` | Pending: Take back how a billing queue item's check was settled and put back any charge the settlement changed. |
| `mutation updateBillingQueueStatus`<br>twin `PUT /api/v1/billing-queue/:itemID/status/` | Tool: `approve_billing_queue_item`, `approve_billing_queue_items`, `cancel_billing_queue_item`, `hold_billing_queue_item`, `move_billing_item_to_exception`, `send_billing_item_back_to_ops`, `transition_item_to_in_review`, `transition_items_to_in_review` |
| `DELETE /api/v1/billing-queue/filter-presets/:presetId/`<br>billingqueuehandler.deleteFilterPreset | Exempt, user-preference: A saved filter on the billing queue screen. |
| `POST /api/v1/billing-queue/:itemID/reassign-charge/`<br>billingqueuehandler.reassignCharge | Tool: `reassign_billing_charge` |
| `POST /api/v1/billing-queue/bulk-approve/`<br>billingqueuehandler.startBulkApprove | Tool: `approve_billing_queue_items` |
| `POST /api/v1/billing-queue/bulk-approve/:runID/cancel/`<br>billingqueuehandler.cancelBulkApprove | Exempt, infrastructure: Stops a bulk approval run the queue screen started in the background; approve_billing_queue_items approves its items within the one proposal a person decides, so an agent has no run to stop. |
| `POST /api/v1/billing-queue/filter-presets/`<br>billingqueuehandler.createFilterPreset | Exempt, user-preference: A saved filter on the billing queue screen. |
| `POST /api/v1/billing-queue/transfer/`<br>billingqueuehandler.transfer | Tool: `transfer_to_billing` |
| `PUT /api/v1/billing-queue/:itemID/charges/`<br>billingqueuehandler.updateCharges | Tool: `correct_charge_code` |
| `PUT /api/v1/billing-queue/filter-presets/:presetId/`<br>billingqueuehandler.updateFilterPreset | Exempt, user-preference: A saved filter on the billing queue screen. |

### billingtransfer

| Write | Decision |
| --- | --- |
| `mutation cancelBillingTransferRun` | Tool: `manage_billing_transfer_run` |
| `mutation retryBillingTransferRun` | Tool: `manage_billing_transfer_run` |
| `mutation startBillingTransferRun` | Tool: `transfer_to_billing` |

### briefing

| Write | Decision |
| --- | --- |
| `mutation markBriefingRead` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation regenerateBriefing` | Exempt, agent-administration: Asks the briefing agent to write the briefing again. |

### capture

| Write | Decision |
| --- | --- |
| `mutation approveCaptureDevicePairing` | Exempt, security: Approving a pairing grants a computer the person's access; only that person may do it. |
| `mutation cancelCaptureRequest` | Exempt, security: Withdraws an instruction the caller gave one of their own paired computers; those instructions are the person's, given under the access they granted the machine, and an agent never acts through a device pairing. |
| `mutation createCaptureCoverSheets` | Exempt, infrastructure: Issues printable routing codes that exist only to be printed and laid on paper at a scanner, returned once to the person who prints them; an agent has no printer, and file_capture_items files what the sheets route. |
| `mutation createCaptureProfile` | Exempt, configuration: Scan profiles are presets an administrator maintains for the organization's scanners. |
| `mutation createCaptureRequest` | Exempt, security: Drives one of the caller's own paired computers, starting its scanner or catching its next print, under the access the person granted that machine; a scan also needs the person at the scanner with the paper. An agent never acts through a device pairing. |
| `mutation deleteCaptureProfile` | Exempt, configuration: Scan profiles are presets an administrator maintains for the organization's scanners. |
| `mutation denyCaptureDevicePairing` | Exempt, security: Denying a pairing is the person's answer to a request for their access. |
| `mutation discardCaptureBatch` | Tool: `discard_capture_batch` |
| `mutation discardCaptureItem` | Tool: `discard_capture_item` |
| `mutation editCaptureItems` | Exempt, user-preference: The person arranging a scanned stack in the intake editor, looking at the page images, replaces the whole layout of the version they were looking at; pages carry no text an agent can read, so an agent would split documents blind. Filing and discarding what the split proposes are file_capture_items and discard_capture_item. |
| `mutation fileCaptureItem` | Tool: `file_capture_items` |
| `mutation fileCaptureItems` | Tool: `file_capture_items` |
| `mutation revokeCaptureDevice` | Exempt, security: Revoking a paired computer removes a person's access. |
| `mutation revokeMyCaptureDevice` | Exempt, security: A person removing their own paired computer's access. |
| `mutation updateCaptureProfile` | Exempt, configuration: Scan profiles are presets an administrator maintains for the organization's scanners. |
| `DELETE /api/v1/capture/device/`<br>capturehandler.signOut | Exempt, security: A paired computer revoking its own credential when a person signs out in the tray. |
| `POST /api/v1/capture/device/batches/`<br>capturehandler.openBatch | Exempt, infrastructure: The Trenova Capture companion calls this as it scans or prints; it is transport for pages a person captured at their own scanner, not a decision. |
| `POST /api/v1/capture/device/batches/:batchID/seal/`<br>capturehandler.sealBatch | Exempt, infrastructure: The Trenova Capture companion calls this as it scans or prints; it is transport for pages a person captured at their own scanner, not a decision. |
| `POST /api/v1/capture/device/requests/:requestID/status/`<br>capturehandler.reportRequestStatus | Exempt, infrastructure: The companion reporting how a scan request is going; the request itself is the person's write. |
| `POST /api/v1/capture/pair/`<br>capturehandler.startPairing | Exempt, security: Starts pairing a computer to a person: a grant of access that person must approve. |
| `POST /api/v1/capture/pair/token/`<br>capturehandler.exchangePairing | Exempt, security: Exchanges an approved pairing for the device credential. |
| `POST /api/v1/capture/token/refresh/`<br>capturehandler.refreshToken | Exempt, security: Rotates a paired device's credential. |
| `PUT /api/v1/capture/device/batches/:batchID/pages/:sequence/`<br>capturehandler.putPage | Exempt, infrastructure: The Trenova Capture companion calls this as it scans or prints; it is transport for pages a person captured at their own scanner, not a decision. |
| `PUT /api/v1/capture/device/batches/:batchID/print-job/`<br>capturehandler.putPrintJob | Exempt, infrastructure: The Trenova Capture companion calls this as it scans or prints; it is transport for pages a person captured at their own scanner, not a decision. |
| `PUT /api/v1/capture/device/sources/`<br>capturehandler.reportSources | Exempt, infrastructure: The companion reporting which scanners its computer can reach. |

### carrier

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/carriers/:carrierID/`<br>carrierhandler.patch | Tool: `update_carrier` |
| `POST /api/v1/carriers/`<br>carrierhandler.create | Tool: `create_carrier` |
| `POST /api/v1/carriers/bulk-update-status/`<br>carrierhandler.bulkUpdateStatus | Tool: `update_carrier_status` |
| `PUT /api/v1/carriers/:carrierID/`<br>carrierhandler.update | Tool: `update_carrier` |

### carrierintelligence

| Write | Decision |
| --- | --- |
| `mutation acknowledgeCarrierIntelEvents` | Tool: `acknowledge_carrier_intel_event` |
| `mutation applyCarrierIntelSuggestions` | Tool: `apply_carrier_intel_suggestions` |
| `mutation grantCarrierIntelOverride` | Exempt, attestation: A person accepts accountability for using a carrier or equipment that failed vetting. |
| `mutation importSourcedCarrier` | Tool: `import_sourced_carrier` |
| `mutation markCarrierIntelReviewed` | Tool: `mark_carrier_intel_reviewed` |
| `mutation overrideCarrierEquipmentVerification` | Exempt, attestation: A person accepts accountability for using a carrier or equipment that failed vetting. |
| `mutation resolveCarrierIntelEvent` | Tool: `resolve_carrier_intel_event` |
| `mutation resumeCarrierIntelMonitoring` | Exempt, configuration: Turns the organization-wide carrier intelligence feed back on after it was paused; an administrator sets it once and every later lookup depends on it. |
| `mutation revokeCarrierIntelOverride` | Exempt, attestation: A person accepts accountability for using a carrier or equipment that failed vetting. |
| `mutation setCarrierMonitoring` | Tool: `set_carrier_monitoring` |
| `mutation switchCarrierIntelProvider` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |
| `mutation updateCarrierIntelControl` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |
| `mutation verifyCarrierEquipment` | Tool: `verify_carrier_equipment` |
| `mutation vetCarrier` | Tool: `vet_carrier` |
| `mutation vetCustomerBroker` | Tool: `vet_customer_broker` |

### carriersettlement

| Write | Decision |
| --- | --- |
| `mutation acceptCarrierInvoiceMatch` | Tool: `accept_carrier_invoice_match` |
| `mutation acceptCarrierInvoiceMatchWithVariance` | Tool: `accept_carrier_invoice_match_with_variance` |
| `mutation addCarrierSettlementAdjustment` | Tool: `add_carrier_settlement_adjustment` |
| `mutation approveCarrierSettlement` | Tool: `approve_carrier_settlement`, `approve_carrier_settlements` |
| `mutation createCarrierInvoiceMatch` | Tool: `create_carrier_invoice_match` |
| `mutation generateCarrierSettlementBatch` | Tool: `generate_carrier_settlement_batch` |
| `mutation linkEdiCarrierInvoiceToCarrier` | Tool: `link_edi_carrier_invoice_to_carrier` |
| `mutation markCarrierSettlementPaid` | Tool: `record_carrier_settlement_payment` |
| `mutation postCarrierSettlement` | Tool: `post_carrier_settlement`, `post_carrier_settlements` |
| `mutation recalculateCarrierSettlement` | Tool: `recalculate_carrier_settlement` |
| `mutation rejectCarrierInvoiceMatch` | Tool: `reject_carrier_invoice_match` |
| `mutation rejectCarrierSettlement` | Tool: `reject_carrier_settlement` |
| `mutation removeCarrierSettlementAdjustment` | Tool: `remove_carrier_settlement_adjustment` |
| `mutation submitCarrierSettlement` | Tool: `submit_carrier_settlement` |
| `mutation updateCarrierSettlementControl` | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |
| `mutation voidCarrierSettlement` | Tool: `void_carrier_settlement` |

### commodity

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/commodities/:commodityID/`<br>commodityhandler.patch | Tool: `update_commodity` |
| `POST /api/v1/commodities/`<br>commodityhandler.create | Tool: `create_commodity` |
| `POST /api/v1/commodities/bulk-update-status/`<br>commodityhandler.bulkUpdateStatus | Tool: `update_commodity_status` |
| `PUT /api/v1/commodities/:commodityID/`<br>commodityhandler.update | Tool: `update_commodity` |

### costing

| Write | Decision |
| --- | --- |
| `mutation updateCostCategory` | Exempt, configuration: Cost categories define the costing model every margin is computed with. |
| `mutation updateCostingControl` | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |

### customer

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/customers/:customerID/`<br>customerhandler.patch | Tool: `update_customer` |
| `POST /api/v1/customers/`<br>customerhandler.create | Tool: `create_customer` |
| `POST /api/v1/customers/bulk-update-status/`<br>customerhandler.bulkUpdateStatus | Tool: `update_customer_status` |
| `PUT /api/v1/customers/:customerID/`<br>customerhandler.update | Tool: `update_customer` |

### customerpayment

| Write | Decision |
| --- | --- |
| `mutation applyCreditMemo`<br>twin `POST /api/v1/accounting/customer-payments/credit-memo-applications/` | Tool: `apply_credit_memo` |
| `mutation applyUnappliedCustomerPayment`<br>twin `POST /api/v1/accounting/customer-payments/:paymentID/apply/` | Tool: `apply_customer_payment` |
| `mutation postAndApplyCustomerPayment`<br>twin `POST /api/v1/accounting/customer-payments/` | Tool: `post_customer_payment` |
| `mutation reverseCustomerPayment`<br>twin `POST /api/v1/accounting/customer-payments/:paymentID/reverse/` | Tool: `reverse_customer_payment` |
| `mutation unapplyCreditMemoApplication`<br>twin `POST /api/v1/accounting/customer-payments/credit-memo-applications/:applicationID/unapply/` | Tool: `unapply_credit_memo` |

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

### deskmemory

| Write | Decision |
| --- | --- |
| `mutation confirmDeskMemory` | Exempt, agent-administration: A person curating what agents remember; an agent writes memory through remember and forget_memory. |
| `mutation createDeskMemory` | Tool: `remember` |
| `mutation dismissDeskMemory` | Exempt, agent-administration: A person curating what agents remember; an agent writes memory through remember and forget_memory. |
| `mutation reviseDeskMemory` | Exempt, agent-administration: A person curating what agents remember; an agent writes memory through remember and forget_memory. |
| `mutation setDeskMemoryStatus` | Tool: `forget_memory` |
| `mutation setMemorySavingMode` | Exempt, agent-administration: Whether agents keep what they learn in the person's conversations straight away or ask first; an agent that could change it could stop asking. |

### detention

| Write | Decision |
| --- | --- |
| `mutation approveDetentionOccurrence`<br>twin `POST /api/v1/detention/occurrences/:occurrenceID/approve/` | Tool: `approve_detention` |
| `mutation createDetentionPolicy`<br>twin `POST /api/v1/detention-policies/` | Exempt, configuration: Detention policies are the commercial terms each customer agreed to; an administrator maintains them. |
| `mutation deleteDetentionPolicy`<br>twin `DELETE /api/v1/detention-policies/:detentionPolicyID/` | Exempt, configuration: Detention policies are the commercial terms each customer agreed to; an administrator maintains them. |
| `mutation detentionBacktest`<br>twin `POST /api/v1/detention/backtest/` | Exempt, read-only: Replays a detention policy against past stops and saves nothing. |
| `mutation disputeDetentionOccurrence`<br>twin `POST /api/v1/detention/occurrences/:occurrenceID/dispute/` | Tool: `dispute_detention` |
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
| `DELETE /api/v1/distance-overrides/:distanceOverrideID/`<br>distanceoverridehandler.delete | Exempt, configuration: A distance override is standing routing data an administrator keeps for a lane the routing provider gets wrong; every later rating, pay and IFTA run reads it. |
| `PATCH /api/v1/distance-overrides/:distanceOverrideID/`<br>distanceoverridehandler.patch | Exempt, configuration: A distance override is standing routing data an administrator keeps for a lane the routing provider gets wrong; every later rating, pay and IFTA run reads it. |
| `POST /api/v1/distance-overrides/`<br>distanceoverridehandler.create | Exempt, configuration: A distance override is standing routing data an administrator keeps for a lane the routing provider gets wrong; every later rating, pay and IFTA run reads it. |
| `PUT /api/v1/distance-overrides/:distanceOverrideID/`<br>distanceoverridehandler.update | Exempt, configuration: A distance override is standing routing data an administrator keeps for a lane the routing provider gets wrong; every later rating, pay and IFTA run reads it. |

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
| `DELETE /api/v1/documents/:documentID/`<br>documenthandler.delete | Tool: `delete_documents` |
| `POST /api/v1/documents/:documentID/approve/`<br>documenthandler.approve | Exempt, attestation: Accepting a document as proof, such as a proof of delivery that lets a shipment bill, is the reviewer's sign-off on the evidence; agents may never approve. |
| `POST /api/v1/documents/:documentID/attach-to-shipment/`<br>documenthandler.attachToShipment | Tool: `attach_document_to_shipment` |
| `POST /api/v1/documents/:documentID/import-assistant/thread/`<br>documenthandler.openImportAssistantThread | Exempt, agent-administration: Talking to the assistant is how a person reaches an agent; agents hand work to one another through delegate_task instead. |
| `POST /api/v1/documents/:documentID/reject/`<br>documenthandler.reject | Exempt, attestation: Refusing a document as proof takes it out of the shipment's billing requirements and is the reviewer's judgement of the evidence, recorded with their reason; an agent can point out a doubtful document but does not decide it. |
| `POST /api/v1/documents/:documentID/restore/`<br>documenthandler.restoreVersion | Tool: `restore_document_version` |
| `POST /api/v1/documents/:documentID/shipment-draft/reextract/`<br>documenthandler.reextractDocumentContent | Exempt, infrastructure: A repair a person asks for when the page's machine reading of a document went wrong: it runs extraction again, replaces the draft they are reviewing and archives their import assistant conversation. The extraction pipeline, not a decision, produces what replaces it. |
| `POST /api/v1/documents/bulk-delete/`<br>documenthandler.bulkDelete | Tool: `delete_documents` |
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
| `mutation resolveSettlementDispute` | Tool: `resolve_settlement_dispute` |
| `mutation respondToMyAssignment` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation respondToMyShiftSwap` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation restoreMyNotifications` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation reviewDriverExpense` | Tool: `review_driver_expense` |
| `mutation revokeWorkerPortalAccess` | Exempt, security: Removes a driver's sign-in to the driver portal. |
| `mutation setMyAvailability` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation startSettlementDisputeReview` | Tool: `start_settlement_dispute_review` |
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
| `mutation addDriverSettlementAdjustment` | Tool: `add_driver_settlement_adjustment` |
| `mutation adjustEscrowAccount` | Tool: `adjust_escrow_account` |
| `mutation approveDriverSettlement` | Tool: `approve_driver_settlement` |
| `mutation assignPayProfileToWorker` | Tool: `assign_pay_profile` |
| `mutation attachPayEventsToSettlement` | Tool: `attach_pay_events_to_settlement` |
| `mutation bulkDriverSettlementAction` | Tool: `approve_driver_settlements`, `post_driver_settlements`, `record_driver_settlement_payment`, `submit_driver_settlement` |
| `mutation closeEscrowAccount` | Tool: `close_escrow_account` |
| `mutation createPayCode` | Exempt, configuration: Pay codes and pay profiles are the pay rules an administrator authors once; every settlement is computed from them, and assigning one to a driver is its own tool. |
| `mutation createPayProfile` | Exempt, configuration: Pay codes and pay profiles are the pay rules an administrator authors once; every settlement is computed from them, and assigning one to a driver is its own tool. |
| `mutation createRecurringDeduction` | Tool: `create_recurring_deduction` |
| `mutation createRecurringEarning` | Tool: `create_recurring_earning` |
| `mutation detachPayEventFromSettlement` | Tool: `detach_pay_event_from_settlement` |
| `mutation endWorkerPayAssignment` | Tool: `end_pay_assignment` |
| `mutation generateDriverSettlement` | Tool: `generate_driver_settlement` |
| `mutation generateSettlementBatch` | Tool: `generate_driver_settlement_batch` |
| `mutation holdDriverPayEvent` | Tool: `hold_driver_pay_event` |
| `mutation issuePayAdvance` | Tool: `issue_pay_advance` |
| `mutation markDriverSettlementPaid` | Tool: `record_driver_settlement_payment` |
| `mutation openEscrowAccount` | Tool: `open_escrow_account` |
| `mutation payWorkerNow` | Tool: `pay_driver_now` |
| `mutation postDriverSettlement` | Tool: `post_driver_settlement` |
| `mutation recalculateDriverSettlement` | Tool: `recalculate_driver_settlement` |
| `mutation rejectDriverSettlement` | Tool: `reject_driver_settlement` |
| `mutation releaseDriverPayEvent` | Tool: `release_driver_pay_event` |
| `mutation removeDriverSettlementAdjustment` | Tool: `remove_driver_settlement_adjustment` |
| `mutation submitDriverSettlement` | Tool: `submit_driver_settlement` |
| `mutation updateEscrowAccount` | Tool: `update_escrow_account` |
| `mutation updatePayCode` | Exempt, configuration: Pay codes and pay profiles are the pay rules an administrator authors once; every settlement is computed from them, and assigning one to a driver is its own tool. |
| `mutation updatePayProfile` | Exempt, configuration: Pay codes and pay profiles are the pay rules an administrator authors once; every settlement is computed from them, and assigning one to a driver is its own tool. |
| `mutation updateRecurringDeduction` | Tool: `update_recurring_deduction` |
| `mutation updateRecurringEarning` | Tool: `update_recurring_earning` |
| `mutation updateSettlementControl` | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |
| `mutation voidDriverSettlement` | Tool: `void_driver_settlement` |
| `mutation writeOffPayAdvance` | Tool: `write_off_pay_advance` |

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
| `POST /api/v1/edi/documents/generate/`<br>edihandler.generateDocument | Tool: `send_edi_status_update` |
| `POST /api/v1/edi/documents/preview/`<br>edihandler.previewDocument | Exempt, read-only: Validates, inspects, previews or tests EDI setup and saves nothing. |
| `POST /api/v1/edi/inbound-files/:fileID/reprocess/`<br>edihandler.reprocessInboundFile | Tool: `reprocess_edi_inbound_files` |
| `POST /api/v1/edi/inbound-files/bulk-reprocess/`<br>edihandler.bulkReprocessInboundFiles | Tool: `reprocess_edi_inbound_files` |
| `POST /api/v1/edi/load-tenders/`<br>edihandler.submitLoadTender | Tool: `send_edi_tender` |
| `POST /api/v1/edi/messages/:messageID/replay/`<br>edihandler.replayMessageDelivery | Tool: `replay_edi_message` |
| `POST /api/v1/edi/messages/:messageID/retry-delivery/`<br>edihandler.retryMessageDelivery | Tool: `retry_edi_message_delivery` |
| `POST /api/v1/edi/messages/bulk-retry-delivery/`<br>edihandler.bulkRetryMessageDelivery | Tool: `retry_edi_message_delivery` |
| `POST /api/v1/edi/partners/`<br>edihandler.createPartner | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/partners/internal-pairs/`<br>edihandler.createInternalPartnerPair | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/templates/`<br>edihandler.createTemplate | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/templates/:templateID/draft/`<br>edihandler.createDraftVersion | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/templates/:templateID/versions/:versionID/activate/`<br>edihandler.activateTemplateVersion | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/templates/:templateID/versions/:versionID/archive/`<br>edihandler.archiveTemplateVersion | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/templates/:templateID/versions/:versionID/certify/`<br>edihandler.certifyTemplateVersion | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/templates/:templateID/versions/:versionID/rollback/`<br>edihandler.rollbackTemplateVersion | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/templates/:templateID/versions/:versionID/validate/`<br>edihandler.validateTemplateVersion | Exempt, read-only: Validates, inspects, previews or tests EDI setup and saves nothing. |
| `POST /api/v1/edi/tender-changes/:changeID/apply/`<br>edihandler.applyTenderChange | Tool: `review_edi_tender_change` |
| `POST /api/v1/edi/tender-changes/:changeID/reject/`<br>edihandler.rejectTenderChange | Tool: `review_edi_tender_change` |
| `POST /api/v1/edi/test-cases/`<br>edihandler.createTestCase | Exempt, configuration: Trading partner setup: partners, connections, mappings, templates and communication profiles an EDI administrator maintains and certifies. |
| `POST /api/v1/edi/test-cases/:testCaseID/preview/`<br>edihandler.previewTestCase | Exempt, read-only: Validates, inspects, previews or tests EDI setup and saves nothing. |
| `POST /api/v1/edi/transfer-changes/:changeID/apply/`<br>edihandler.applyTransferChange | Tool: `review_edi_transfer_change` |
| `POST /api/v1/edi/transfer-changes/:changeID/reject/`<br>edihandler.rejectTransferChange | Tool: `review_edi_transfer_change` |
| `POST /api/v1/edi/transfers/:transferID/approve/`<br>edihandler.approveTransfer | Tool: `accept_edi_tender` |
| `POST /api/v1/edi/transfers/:transferID/cancel/`<br>edihandler.cancelTransfer | Tool: `cancel_edi_tender` |
| `POST /api/v1/edi/transfers/:transferID/expire/`<br>edihandler.expireTransfer | Tool: `expire_edi_tender` |
| `POST /api/v1/edi/transfers/:transferID/reject/`<br>edihandler.rejectTransfer | Tool: `decline_edi_tender` |
| `POST /api/v1/edi/transfers/bulk-approve/`<br>edihandler.bulkApproveTransfers | Exempt, duplicate: Runs POST /api/v1/edi/transfers/:transferID/approve/ once per selected tender; accept_edi_tender answers one tender per proposal so each is previewed and pinned to its own version. |
| `POST /api/v1/edi/transfers/bulk-reject/`<br>edihandler.bulkRejectTransfers | Exempt, duplicate: Runs POST /api/v1/edi/transfers/:transferID/reject/ once per selected tender; decline_edi_tender answers one tender per proposal so each is previewed and pinned to its own version. |
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
| `POST /api/v1/exchange-rates/settlement-quotes`<br>exchangeratehandler.createSettlementQuote | Exempt, infrastructure: Fetches a live rate from the rate provider and keeps a short-lived quote for the screen that asked; it books nothing, and nothing uses the quote unless a person settles a payment with it. |

### extractioneval

| Write | Decision |
| --- | --- |
| `mutation cancelExtractionEvalRun` | Exempt, agent-administration: Curating and running the evaluation of AI document extraction is oversight of the agents; an agent must not grade or shape its own test set. |
| `mutation deleteExtractionEvalCase` | Exempt, agent-administration: Curating and running the evaluation of AI document extraction is oversight of the agents; an agent must not grade or shape its own test set. |
| `mutation promoteAICorrection` | Exempt, agent-administration: Curating and running the evaluation of AI document extraction is oversight of the agents; an agent must not grade or shape its own test set. |
| `mutation startExtractionEvalRun` | Exempt, agent-administration: Curating and running the evaluation of AI document extraction is oversight of the agents; an agent must not grade or shape its own test set. |
| `mutation updateExtractionEvalCase` | Exempt, agent-administration: Curating and running the evaluation of AI document extraction is oversight of the agents; an agent must not grade or shape its own test set. |

### extractionrollout

| Write | Decision |
| --- | --- |
| `mutation updateExtractionRollout` | Exempt, agent-administration: Choosing which AI provider serves real document extractions, how much of them, and when its guards stop it is oversight of the agents; an agent must not promote or protect the model it is measured against. |

### extractionshadow

| Write | Decision |
| --- | --- |
| `mutation updateExtractionShadowSettings` | Exempt, agent-administration: Choosing which AI provider shadows document extraction, and how much it is sent, is oversight of the agents; an agent must not pick or tune the model it is measured against. |

### fiscalperiod

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/fiscal-periods/:fiscalPeriodID/`<br>fiscalperiodhandler.delete | Exempt, configuration: The fiscal calendar every posting resolves its period from; an accounting administrator lays its periods out with the fiscal year, and they change with the calendar, not with the work. |
| `PATCH /api/v1/fiscal-periods/:fiscalPeriodID/`<br>fiscalperiodhandler.patch | Exempt, configuration: The fiscal calendar every posting resolves its period from; an accounting administrator lays its periods out with the fiscal year, and they change with the calendar, not with the work. |
| `POST /api/v1/fiscal-periods/`<br>fiscalperiodhandler.create | Exempt, configuration: The fiscal calendar every posting resolves its period from; an accounting administrator lays its periods out with the fiscal year, and they change with the calendar, not with the work. |
| `PUT /api/v1/fiscal-periods/:fiscalPeriodID/`<br>fiscalperiodhandler.update | Exempt, configuration: The fiscal calendar every posting resolves its period from; an accounting administrator lays its periods out with the fiscal year, and they change with the calendar, not with the work. |
| `PUT /api/v1/fiscal-periods/:fiscalPeriodID/activate/`<br>fiscalperiodhandler.activate | Tool: `open_fiscal_period` |
| `PUT /api/v1/fiscal-periods/:fiscalPeriodID/close/`<br>fiscalperiodhandler.close | Tool: `close_fiscal_period` |
| `PUT /api/v1/fiscal-periods/:fiscalPeriodID/lock/`<br>fiscalperiodhandler.lock | Tool: `lock_fiscal_period` |
| `PUT /api/v1/fiscal-periods/:fiscalPeriodID/reopen/`<br>fiscalperiodhandler.reopen | Tool: `reopen_fiscal_period` |
| `PUT /api/v1/fiscal-periods/:fiscalPeriodID/unlock/`<br>fiscalperiodhandler.unlock | Tool: `unlock_fiscal_period` |

### fiscalyear

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/fiscal-years/:fiscalYearID/`<br>fiscalyearhandler.delete | Exempt, configuration: The fiscal years the calendar is laid out in, which an accounting administrator sets up once a year and every posting's period hangs from. |
| `PATCH /api/v1/fiscal-years/:fiscalYearID/`<br>fiscalyearhandler.patch | Exempt, configuration: The fiscal years the calendar is laid out in, which an accounting administrator sets up once a year and every posting's period hangs from. |
| `POST /api/v1/fiscal-years/`<br>fiscalyearhandler.create | Exempt, configuration: The fiscal years the calendar is laid out in, which an accounting administrator sets up once a year and every posting's period hangs from. |
| `PUT /api/v1/fiscal-years/:fiscalYearID/`<br>fiscalyearhandler.update | Exempt, configuration: The fiscal years the calendar is laid out in, which an accounting administrator sets up once a year and every posting's period hangs from. |
| `PUT /api/v1/fiscal-years/:fiscalYearID/activate/`<br>fiscalyearhandler.activate | Exempt, configuration: Makes a fiscal year the one the calendar runs on, part of setting up the year that an accounting administrator does once. |
| `PUT /api/v1/fiscal-years/:fiscalYearID/close/`<br>fiscalyearhandler.close | Exempt, attestation: Closing the year books its closing entries into retained earnings and is the year-end sign-off the person accountable for the books makes. |
| `PUT /api/v1/fiscal-years/:fiscalYearID/reopen/`<br>fiscalyearhandler.reopen | Exempt, attestation: Reverses the year's closing entries and withdraws the year-end sign-off, which only the person accountable for the books does. |

### fleetcode

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/fleet-codes/:fleetCodeID`<br>fleetcodehandler.patch | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `POST /api/v1/fleet-codes/`<br>fleetcodehandler.create | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `PUT /api/v1/fleet-codes/:fleetCodeID`<br>fleetcodehandler.update | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |

### fleetsafety

| Write | Decision |
| --- | --- |
| `mutation deleteSafetyViolation` | Tool: `delete_safety_violation` |
| `mutation recordSafetyViolation` | Tool: `record_safety_violation` |
| `mutation updateSafetyViolation` | Tool: `update_safety_violation` |

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
| `mutation assignFuelCard` | Tool: `assign_fuel_card` |
| `mutation cancelFuelCard` | Exempt, configuration: Fuel cards are payment instruments an administrator issues, edits and cancels with the card provider; an agent assigns a card already on file (assign_fuel_card). |
| `mutation commitFuelPurchaseImport` | Tool: `commit_fuel_purchase_import` |
| `mutation createFuelCard` | Exempt, configuration: Fuel cards are payment instruments an administrator issues, edits and cancels with the card provider; an agent assigns a card already on file (assign_fuel_card). |
| `mutation createFuelPurchase` | Tool: `record_fuel_purchase` |
| `mutation createFuelPurchaseImport` | Exempt, infrastructure: Part of uploading a statement file from a browser: the batch holds the upload and staging parses it; an agent works a staged batch with commit_fuel_purchase_import or discard_fuel_purchase_import. |
| `mutation deleteFuelPurchase` | Tool: `delete_fuel_purchase` |
| `mutation discardFuelPurchaseImport` | Tool: `discard_fuel_purchase_import` |
| `mutation resolveFuelPurchaseImportRows` | Tool: `resolve_fuel_purchase_import_rows` |
| `mutation stageFuelPurchaseImport` | Exempt, infrastructure: Part of uploading a statement file from a browser: the batch holds the upload and staging parses it; an agent works a staged batch with commit_fuel_purchase_import or discard_fuel_purchase_import. |
| `mutation syncFuelCardFeed` | Exempt, infrastructure: Reads the card provider's feed on the connection an administrator set up, which runs on its own schedule; resolve_fuel_purchase_import_rows works the rows a run held back. |
| `mutation updateFuelCard` | Exempt, configuration: Fuel cards are payment instruments an administrator issues, edits and cancels with the card provider; an agent assigns a card already on file (assign_fuel_card). |
| `mutation updateFuelPurchase` | Tool: `correct_fuel_purchase` |

### fuelsurcharge

| Write | Decision |
| --- | --- |
| `mutation addFuelIndexPrice` | Tool: `record_fuel_index_price` |
| `mutation createFuelIndex` | Exempt, configuration: Fuel surcharge programs and the indexes they follow are commercial terms an administrator sets up. |
| `mutation createFuelSurchargeProgram` | Exempt, configuration: Fuel surcharge programs and the indexes they follow are commercial terms an administrator sets up. |
| `mutation deleteFuelIndex` | Exempt, configuration: Fuel surcharge programs and the indexes they follow are commercial terms an administrator sets up. |
| `mutation deleteFuelIndexPrice` | Exempt, configuration: Pruning an index's price history is curation an administrator does; a manual price entered wrong is fixed with correct_fuel_index_price. |
| `mutation deleteFuelSurchargeProgram` | Exempt, configuration: Fuel surcharge programs and the indexes they follow are commercial terms an administrator sets up. |
| `mutation updateFuelIndex` | Exempt, configuration: Fuel surcharge programs and the indexes they follow are commercial terms an administrator sets up. |
| `mutation updateFuelIndexPrice` | Tool: `correct_fuel_index_price` |
| `mutation updateFuelSurchargeProgram` | Exempt, configuration: Fuel surcharge programs and the indexes they follow are commercial terms an administrator sets up. |

### glaccount

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/gl-accounts/:glAccountID/`<br>glaccounthandler.delete | Exempt, configuration: The chart of accounts every posting and accounting mapping depends on; an accountant shapes it, and an agent reshaping it would change what every later entry means. |
| `PATCH /api/v1/gl-accounts/:glAccountID/`<br>glaccounthandler.patch | Exempt, configuration: The chart of accounts every posting and accounting mapping depends on; an accountant shapes it, and an agent reshaping it would change what every later entry means. |
| `POST /api/v1/gl-accounts/`<br>glaccounthandler.create | Exempt, configuration: The chart of accounts every posting and accounting mapping depends on; an accountant shapes it, and an agent reshaping it would change what every later entry means. |
| `POST /api/v1/gl-accounts/bulk-update-status/`<br>glaccounthandler.bulkUpdateStatus | Exempt, configuration: The chart of accounts every posting and accounting mapping depends on; an accountant shapes it, and an agent reshaping it would change what every later entry means. |
| `PUT /api/v1/gl-accounts/:glAccountID/`<br>glaccounthandler.update | Exempt, configuration: The chart of accounts every posting and accounting mapping depends on; an accountant shapes it, and an agent reshaping it would change what every later entry means. |

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
| `PATCH /api/v1/hazardous-materials/:hazardousMaterialID/`<br>hazardousmaterialhandler.patch | Tool: `update_hazardous_material` |
| `POST /api/v1/hazardous-materials/`<br>hazardousmaterialhandler.create | Tool: `create_hazardous_material` |
| `POST /api/v1/hazardous-materials/bulk-update-status/`<br>hazardousmaterialhandler.bulkUpdateStatus | Tool: `update_hazardous_material_status` |
| `PUT /api/v1/hazardous-materials/:hazardousMaterialID/`<br>hazardousmaterialhandler.update | Tool: `update_hazardous_material` |

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
| `mutation amendIftaReturn` | Tool: `amend_ifta_return` |
| `mutation backfillJurisdictionMiles` | Tool: `backfill_jurisdiction_miles` |
| `mutation createIftaMileageEntry` | Tool: `record_ifta_mileage_entry` |
| `mutation deleteIftaMileageEntry` | Tool: `delete_ifta_mileage_entry` |
| `mutation deleteIftaReturn` | Tool: `delete_ifta_return` |
| `mutation deleteIftaTaxRate` | Exempt, configuration: IFTA tax rates are the jurisdictions' published rates an administrator loads each quarter; every return is computed from them. |
| `mutation finalizeIftaReturn` | Exempt, attestation: A fuel tax return is signed off by the person accountable for filing it. |
| `mutation generateIftaReturn` | Tool: `generate_ifta_return` |
| `mutation markIftaReturnFiled` | Exempt, attestation: Records that a person filed the return with the jurisdiction. |
| `mutation recalculateMoveJurisdictionMiles` | Tool: `recalculate_move_jurisdiction_miles` |
| `mutation recomputeIftaReturn` | Tool: `recompute_ifta_return` |
| `mutation reopenIftaReturn` | Exempt, attestation: Reopening undoes the sign-off of the person accountable for filing the return, so that person decides. |
| `mutation updateIftaMileageEntry` | Tool: `correct_ifta_mileage_entry` |
| `mutation upsertIftaTaxRates` | Exempt, configuration: IFTA tax rates are the jurisdictions' published rates an administrator loads each quarter; every return is computed from them. |

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
| `mutation setInboundMailboxApiKey` | Exempt, security: Sets the provider credential the mailbox reads forwarded mail's content with. |
| `mutation setInboundMailboxSigningSecret` | Exempt, security: Sets the secret that authenticates mail forwarded into the mailbox. |
| `mutation updateInboundMailbox` | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |

### insight

| Write | Decision |
| --- | --- |
| `POST /api/v1/insights/:insightID/dismiss/`<br>insighthandler.dismiss | Tool: `dismiss_insight` |
| `POST /api/v1/insights/:insightID/restore/`<br>insighthandler.restore | Tool: `restore_insight` |

### integration

| Write | Decision |
| --- | --- |
| `POST /api/v1/integrations/:type/test-connection/`<br>integrationhandler.testConnection | Exempt, configuration: Tests an integration's credentials and records the connection status on its configuration. |
| `POST /api/v1/integrations/samsara/workers/sync/`<br>integrationhandler.startWorkerSync | Exempt, configuration: Creates and links drivers' accounts in the telematics provider from the integration's setup page; connecting that integration and the accounts it holds is an administrator's. |
| `POST /api/v1/integrations/samsara/workers/sync/drift/detect/`<br>integrationhandler.detectWorkerSyncDrift | Exempt, read-only: Compares workers with the telematics provider and reports the drift. |
| `POST /api/v1/integrations/samsara/workers/sync/drift/repair/`<br>integrationhandler.repairWorkerSyncDrift | Exempt, configuration: Rewrites drivers' accounts in the telematics provider to match Trenova from the integration's setup page; an administrator owns that connection and the accounts it holds. |
| `PUT /api/v1/integrations/:type/config/`<br>integrationhandler.updateConfig | Exempt, configuration: Connects the organization to an outside system; an administrator owns the connection and its credentials. |

### invoice

| Write | Decision |
| --- | --- |
| `mutation createInvoiceFromOrder`<br>twin `POST /api/v1/billing/invoices/from-order/` | Tool: `create_invoice` |
| `mutation createInvoiceFromShipments`<br>twin `POST /api/v1/billing/invoices/from-shipments/` | Tool: `create_invoice` |
| `mutation createInvoicesFromOrder` | Tool: `create_invoice` |
| `mutation createInvoicesFromShipments` | Tool: `create_invoice` |
| `mutation createMemo`<br>twin `POST /api/v1/billing/invoices/memos/` | Tool: `create_invoice_memo` |
| `mutation sendInvoiceEdi` | Tool: `send_invoice_edi` |
| `mutation voidInvoice`<br>twin `POST /api/v1/billing/invoices/:invoiceID/void/` | Tool: `void_invoice` |
| `PATCH /api/v1/billing/invoices/:invoiceID/`<br>invoicehandler.updateDraft | Tool: `update_invoice_draft` |
| `POST /api/v1/billing/invoices/:invoiceID/generate-pdf/`<br>invoicehandler.generatePDF | Tool: `generate_invoice_pdf` |
| `POST /api/v1/billing/invoices/:invoiceID/post/`<br>invoicehandler.post | Tool: `post_invoice`, `post_invoices` |
| `POST /api/v1/billing/invoices/:invoiceID/preview/`<br>invoicehandler.preview | Exempt, read-only: Renders an invoice for review and saves nothing. |
| `POST /api/v1/billing/invoices/:invoiceID/send/`<br>invoicehandler.send<br>also `POST /api/v1/billing/invoices/:invoiceID/resend/` | Tool: `send_invoice`, `send_invoices` |

### invoiceadjustment

| Write | Decision |
| --- | --- |
| `mutation approveInvoiceAdjustment`<br>twin `POST /api/v1/billing/invoice-adjustments/:adjustmentID/approve/` | Tool: `approve_invoice_adjustment` |
| `mutation rejectInvoiceAdjustment`<br>twin `POST /api/v1/billing/invoice-adjustments/:adjustmentID/reject/` | Tool: `reject_invoice_adjustment` |
| `PATCH /api/v1/billing/invoice-adjustments/drafts/:adjustmentID/`<br>invoiceadjustmenthandler.updateDraft | Tool: `save_invoice_adjustment_draft` |
| `POST /api/v1/billing/invoice-adjustments/bulk-preview/`<br>invoiceadjustmenthandler.bulkPreview | Exempt, read-only: Previews an invoice adjustment and saves nothing. |
| `POST /api/v1/billing/invoice-adjustments/bulk-submit/`<br>invoiceadjustmenthandler.bulkSubmit | Tool: `submit_invoice_adjustment` |
| `POST /api/v1/billing/invoice-adjustments/drafts/`<br>invoiceadjustmenthandler.createDraft | Tool: `save_invoice_adjustment_draft` |
| `POST /api/v1/billing/invoice-adjustments/drafts/:adjustmentID/preview/`<br>invoiceadjustmenthandler.previewDraft | Exempt, read-only: Previews an invoice adjustment and saves nothing. |
| `POST /api/v1/billing/invoice-adjustments/drafts/:adjustmentID/submit/`<br>invoiceadjustmenthandler.submitDraft | Tool: `submit_invoice_adjustment` |
| `POST /api/v1/billing/invoice-adjustments/preview/`<br>invoiceadjustmenthandler.preview | Exempt, read-only: Previews an invoice adjustment and saves nothing. |
| `POST /api/v1/billing/invoice-adjustments/submit/`<br>invoiceadjustmenthandler.submit | Tool: `submit_invoice_adjustment` |

### invoiceadjustmentcontrol

| Write | Decision |
| --- | --- |
| `PUT /api/v1/invoice-adjustment-controls/`<br>invoiceadjustmentcontrolhandler.update | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |

### invoicedispute

| Write | Decision |
| --- | --- |
| `mutation openInvoiceDispute` | Tool: `open_invoice_dispute` |
| `mutation resolveInvoiceDispute` | Tool: `resolve_invoice_dispute` |
| `mutation withdrawInvoiceDispute` | Tool: `withdraw_invoice_dispute` |

### invoicerun

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/billing/invoice-runs/:runID/membership/`<br>invoicerunhandler.adjustMembership | Tool: `adjust_invoice_run_membership` |
| `POST /api/v1/billing/invoice-runs/:runID/cancel/`<br>invoicerunhandler.cancel | Tool: `cancel_invoice_run` |
| `POST /api/v1/billing/invoice-runs/:runID/commit/`<br>invoicerunhandler.commit | Tool: `commit_invoice_run` |
| `POST /api/v1/billing/invoice-runs/preview/`<br>invoicerunhandler.preview | Tool: `build_invoice_run` |
| `POST /api/v1/billing/statements/:customerID/bill/`<br>invoicerunhandler.billStatement | Tool: `bill_statement_now` |

### invoiceshare

| Write | Decision |
| --- | --- |
| `POST /api/v1/billing/invoices/:invoiceID/shares/`<br>invoicesharehandler.share | Tool: `share_invoice` |

### journalentry

| Write | Decision |
| --- | --- |
| `mutation approveJournalEntries` | Exempt, attestation: Approving a journal for the general ledger is a sign-off an accountable person makes; agents never hold the approve permission. |
| `mutation postJournalEntries` | Exempt, attestation: Posting approved journals changes the general ledger balances a person signs off on; agents never hold the approve permission. |

### journalreversal

| Write | Decision |
| --- | --- |
| `POST /api/v1/accounting/journal-reversals/`<br>journalreversalhandler.create | Tool: `request_journal_reversal` |
| `POST /api/v1/accounting/journal-reversals/:reversalID/approve/`<br>journalreversalhandler.approve | Exempt, attestation: Approving a reversal is the second person's sign-off on taking a posted entry back out of the ledger; an agent requests reversals and never stands in as their approver. |
| `POST /api/v1/accounting/journal-reversals/:reversalID/cancel/`<br>journalreversalhandler.cancel | Tool: `cancel_journal_reversal` |
| `POST /api/v1/accounting/journal-reversals/:reversalID/post/`<br>journalreversalhandler.post | Tool: `post_journal_reversal` |
| `POST /api/v1/accounting/journal-reversals/:reversalID/reject/`<br>journalreversalhandler.reject | Exempt, attestation: Turning a reversal down is the approver's half of the same sign-off, made by the accountable person reviewing it. |

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
| `mutation assessLateCharges` | Tool: `assess_late_charges` |

### location

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/locations/:locationID/`<br>locationhandler.patch | Tool: `update_location` |
| `POST /api/v1/locations/`<br>locationhandler.create | Tool: `create_location` |
| `POST /api/v1/locations/bulk-update-status/`<br>locationhandler.bulkUpdateStatus | Tool: `update_location_status` |
| `PUT /api/v1/locations/:locationID/`<br>locationhandler.update | Tool: `update_location` |

### locationcategory

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/location-categories/:locationCategoryID/`<br>locationcategoryhandler.patch | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `POST /api/v1/location-categories/`<br>locationcategoryhandler.create | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |
| `PUT /api/v1/location-categories/:locationCategoryID/`<br>locationcategoryhandler.update | Exempt, configuration: A lookup list every record points at, maintained by an administrator. |

### manualjournal

| Write | Decision |
| --- | --- |
| `POST /api/v1/accounting/manual-journals/:requestID/approve/`<br>manualjournalhandler.approve | Exempt, attestation: Approving a manual journal is the second person's sign-off on an entry someone else prepared; an agent prepares journals and never stands in as their approver. |
| `POST /api/v1/accounting/manual-journals/:requestID/cancel/`<br>manualjournalhandler.cancel | Tool: `cancel_manual_journal` |
| `POST /api/v1/accounting/manual-journals/:requestID/post/`<br>manualjournalhandler.post | Tool: `post_manual_journal` |
| `POST /api/v1/accounting/manual-journals/:requestID/reject/`<br>manualjournalhandler.reject | Exempt, attestation: Sending a manual journal back is the approver's half of the same sign-off, made by the accountable person reviewing it. |
| `POST /api/v1/accounting/manual-journals/:requestID/submit/`<br>manualjournalhandler.submit | Tool: `submit_manual_journal` |
| `POST /api/v1/accounting/manual-journals/drafts/`<br>manualjournalhandler.createDraft | Tool: `draft_manual_journal` |
| `PUT /api/v1/accounting/manual-journals/drafts/:requestID/`<br>manualjournalhandler.updateDraft | Tool: `revise_manual_journal_draft` |

### notification

| Write | Decision |
| --- | --- |
| `mutation dismissNotifications` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation markAllNotificationsRead` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation markNotificationsRead` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation markNotificationsUnread` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `mutation restoreNotifications` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |

### onboarding

| Write | Decision |
| --- | --- |
| `POST /api/v1/onboarding/complete/`<br>onboardinghandler.complete | Exempt, configuration: The owner's one-time setup of a new organization's profile and operating mode; every later write depends on it and it runs before any agent is configured. |

### order

| Write | Decision |
| --- | --- |
| `mutation addOrderCharge` | Tool: `add_order_charge` |
| `mutation attachOrderShipments` | Tool: `attach_order_shipments` |
| `mutation cancelOrder` | Tool: `cancel_order` |
| `mutation closeOrder` | Tool: `close_order` |
| `mutation createOrder`<br>twin `POST /api/v1/orders/` | Tool: `create_order` |
| `mutation detachOrderShipment` | Tool: `detach_order_shipment` |
| `mutation removeOrderCharge` | Tool: `remove_order_charge` |
| `mutation setOrderChargeAllocations` | Tool: `set_order_charge_allocations` |
| `mutation updateOrder`<br>twin `PATCH /api/v1/orders/:orderID/`<br>twin `PUT /api/v1/orders/:orderID/` | Tool: `update_order` |
| `mutation updateOrderCharge` | Tool: `update_order_charge` |

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
| `mutation assignUserPosition` | Exempt, security: Places a person in the reporting line that time-off and expense approvals route through, which decides whose requests they may approve. |
| `mutation assignWorkerPosition` | Exempt, security: Places a driver in the reporting line their time-off and expense approvals route through, which decides who may approve their requests. |
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
| `mutation closePerformanceReview` | Exempt, attestation: Closing a review is the reviewer's sign-off on a record the worker has acknowledged; an agent drafts, it does not sign. |
| `mutation createPerformanceReview` | Tool: `start_performance_review` |
| `mutation createPerformanceReviewTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation deletePerformanceReview` | Tool: `delete_performance_review` |
| `mutation reopenPerformanceReview` | Exempt, attestation: Reopening a submitted review withdraws the reviewer's signed assessment from the worker; only the reviewer does that. |
| `mutation restorePerformanceReviewTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation submitPerformanceReview` | Exempt, attestation: Submitting is the reviewer signing the assessment and sending it to the worker to acknowledge; an agent drafts, it does not sign. |
| `mutation updatePerformanceReview` | Tool: `draft_performance_review` |
| `mutation updatePerformanceReviewTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |

### permission

| Write | Decision |
| --- | --- |
| `POST /api/v1/me/permissions/check`<br>permissionhandler.checkBatch | Exempt, read-only: Checks which permissions the caller holds. |

### permit

| Write | Decision |
| --- | --- |
| `POST /api/v1/shipments/:shipmentID/permit-requirements/:requirementID/waive/`<br>permithandler.waiveRequirement | Exempt, attestation: Waiving a permit requirement is a named person accepting the compliance risk of moving the load without that state's permit. |
| `POST /api/v1/shipments/:shipmentID/permits/`<br>permithandler.createPermit | Tool: `record_shipment_permit` |
| `PUT /api/v1/shipments/:shipmentID/permits/:permitID/`<br>permithandler.updatePermit | Tool: `update_shipment_permit` |

### ptopolicy

| Write | Decision |
| --- | --- |
| `mutation adjustWorkerPtoBalance` | Tool: `adjust_worker_pto_balance` |
| `mutation archivePtoPolicy` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation assignWorkerPtoPolicy` | Exempt, configuration: Which PTO policy a worker accrues under is an HR setting an administrator assigns; requests and balances follow from it. |
| `mutation createPtoPolicy` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation endWorkerPtoPolicyAssignment` | Exempt, configuration: Which PTO policy a worker accrues under is an HR setting an administrator assigns; requests and balances follow from it. |
| `mutation restorePtoPolicy` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation runPtoAccrual` | Exempt, infrastructure: Accrual runs on its schedule; this re-runs the job for a period, which is repair work on the ledger rather than a decision. |
| `mutation updatePtoPolicy` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |

### push

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/push/subscriptions/`<br>pushhandler.unsubscribe | Exempt, infrastructure: Registers or removes a browser push subscription for the device the person is using. |
| `POST /api/v1/push/subscriptions/`<br>pushhandler.subscribe | Exempt, infrastructure: Registers or removes a browser push subscription for the device the person is using. |

### rateagreement

| Write | Decision |
| --- | --- |
| `POST /api/v1/rate-agreements/`<br>rateagreementhandler.create | Tool: `draft_rate_agreement` |
| `POST /api/v1/rate-agreements/:rateAgreementID/approve/`<br>rateagreementhandler.review | Tool: `approve_rate_agreement` |
| `POST /api/v1/rate-agreements/:rateAgreementID/archive/`<br>rateagreementhandler.review | Tool: `archive_rate_agreement` |
| `POST /api/v1/rate-agreements/:rateAgreementID/duplicate/`<br>rateagreementhandler.duplicate | Tool: `duplicate_rate_agreement` |
| `POST /api/v1/rate-agreements/:rateAgreementID/reject/`<br>rateagreementhandler.review | Tool: `reject_rate_agreement` |
| `POST /api/v1/rate-agreements/:rateAgreementID/resume/`<br>rateagreementhandler.review | Tool: `resume_rate_agreement` |
| `POST /api/v1/rate-agreements/:rateAgreementID/rules/amend/`<br>rateagreementhandler.amendRules | Tool: `amend_rate_agreement_rules` |
| `POST /api/v1/rate-agreements/:rateAgreementID/submit/`<br>rateagreementhandler.review | Tool: `submit_rate_agreement` |
| `POST /api/v1/rate-agreements/:rateAgreementID/suspend/`<br>rateagreementhandler.review | Tool: `suspend_rate_agreement` |
| `POST /api/v1/rate-agreements/rate-increase/apply/`<br>rateagreementhandler.applyRateIncrease | Tool: `apply_rate_increase` |
| `POST /api/v1/rate-agreements/rate-increase/preview/`<br>rateagreementhandler.previewRateIncrease | Exempt, read-only: Plans a general rate increase for review and saves nothing. |
| `PUT /api/v1/rate-agreements/:rateAgreementID/`<br>rateagreementhandler.update | Tool: `revise_rate_agreement_draft` |

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
| `POST /api/v1/rate-imports/:rateImportID/commit/`<br>rateimporthandler.commit | Tool: `commit_rate_import` |
| `POST /api/v1/rate-imports/:rateImportID/discard/`<br>rateimporthandler.discard | Tool: `discard_rate_import` |

### ratematrix

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/rate-matrices/:rateMatrixID/`<br>ratematrixhandler.delete | Exempt, configuration: A rate matrix is a pricing table an administrator builds and agreements point at; an agent drafts and amends agreements against the matrices that exist. |
| `POST /api/v1/rate-matrices/`<br>ratematrixhandler.create | Exempt, configuration: A rate matrix is a pricing table an administrator builds and agreements point at; an agent drafts and amends agreements against the matrices that exist. |
| `PUT /api/v1/rate-matrices/:rateMatrixID/`<br>ratematrixhandler.update | Exempt, configuration: A rate matrix is a pricing table an administrator builds and agreements point at; an agent drafts and amends agreements against the matrices that exist. |
| `PUT /api/v1/rate-matrices/:rateMatrixID/cells/`<br>ratematrixhandler.replaceCells | Exempt, configuration: A rate matrix is a pricing table an administrator builds and agreements point at; an agent drafts and amends agreements against the matrices that exist. |

### ratequote

| Write | Decision |
| --- | --- |
| `POST /api/v1/rate-quotes/quote/`<br>ratequotehandler.quote | Exempt, read-only: Quotes a rate from the rating engine and saves nothing. |
| `POST /api/v1/rate-quotes/shipment/:shipmentID/explain/`<br>ratequotehandler.explain | Exempt, read-only: Explains how a shipment was rated and saves nothing. |
| `POST /api/v1/rate-quotes/shipment/:shipmentID/shop/`<br>ratequotehandler.shop | Exempt, read-only: Compares the rates every agreement would give a shipment and saves nothing. |

### ratesimulation

| Write | Decision |
| --- | --- |
| `POST /api/v1/rate-simulations/`<br>ratesimulationhandler.create | Tool: `run_rate_simulation` |

### ratezone

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/rate-zones/:rateZoneID/`<br>ratezonehandler.delete | Exempt, configuration: Rate zones are the geography agreements price against, which an administrator defines once for every agreement. |
| `POST /api/v1/rate-zones/`<br>ratezonehandler.create | Exempt, configuration: Rate zones are the geography agreements price against, which an administrator defines once for every agreement. |
| `PUT /api/v1/rate-zones/:rateZoneID/`<br>ratezonehandler.update | Exempt, configuration: Rate zones are the geography agreements price against, which an administrator defines once for every agreement. |

### realtime

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/shipments/:shipmentID/comments/presence/`<br>realtimehandler.leaveShipmentComments | Exempt, infrastructure: Presence and typing signals a browser sends while a person is on the page. |
| `POST /api/v1/shipments/:shipmentID/comments/presence/`<br>realtimehandler.joinShipmentComments | Exempt, infrastructure: Presence and typing signals a browser sends while a person is on the page. |
| `POST /api/v1/shipments/:shipmentID/comments/typing/`<br>realtimehandler.typingShipmentComments | Exempt, infrastructure: Presence and typing signals a browser sends while a person is on the page. |

### recurringshipment

| Write | Decision |
| --- | --- |
| `POST /api/v1/recurring-shipments/`<br>recurringshipmenthandler.create | Tool: `create_recurring_shipment` |
| `POST /api/v1/recurring-shipments/:recurringShipmentID/generate/`<br>recurringshipmenthandler.generate | Tool: `generate_recurring_shipment` |
| `POST /api/v1/recurring-shipments/match/`<br>recurringshipmenthandler.match | Exempt, read-only: Finds the recurring shipment a new shipment matches and saves nothing. |
| `PUT /api/v1/recurring-shipments/:recurringShipmentID/`<br>recurringshipmenthandler.update | Tool: `update_recurring_shipment` |
| `PUT /api/v1/recurring-shipments/:recurringShipmentID/status/`<br>recurringshipmenthandler.updateStatus | Tool: `set_recurring_shipment_status` |

### report

| Write | Decision |
| --- | --- |
| `mutation cancelReportRun` | Tool: `cancel_report_run` |
| `mutation createReportDashboard` | Tool: `create_dashboard` |
| `mutation createReportDefinition` | Tool: `create_report` |
| `mutation createReportSchedule` | Tool: `schedule_report` |
| `mutation createReportView` | Exempt, user-preference: A saved view is one person's own columns and filters on a report page. |
| `mutation deleteReportDashboard` | Tool: `delete_dashboard` |
| `mutation deleteReportDefinition` | Tool: `delete_report` |
| `mutation deleteReportSchedule` | Tool: `delete_report_schedule` |
| `mutation deleteReportView` | Exempt, user-preference: A saved view is one person's own columns and filters on a report page. |
| `mutation forkCannedReport` | Tool: `fork_report` |
| `mutation resetCannedFork` | Tool: `reset_report_fork` |
| `mutation runReport` | Tool: `run_report` |
| `mutation updateReportDashboard` | Tool: `add_dashboard_tile` |
| `mutation updateReportDefinition` | Tool: `update_report` |
| `mutation updateReportSchedule` | Tool: `update_report_schedule` |
| `mutation updateReportView` | Exempt, user-preference: A saved view is one person's own columns and filters on a report page. |
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
| `DELETE /api/v1/routing-guides/:guideID/`<br>routingguidehandler.delete | Exempt, configuration: A routing guide is the lane-by-lane carrier waterfall an administrator maintains; tender_move_to_routing_guide uses it. |
| `POST /api/v1/routing-guides/`<br>routingguidehandler.create | Exempt, configuration: A routing guide is the lane-by-lane carrier waterfall an administrator maintains; tender_move_to_routing_guide uses it. |
| `PUT /api/v1/routing-guides/:guideID/`<br>routingguidehandler.update | Exempt, configuration: A routing guide is the lane-by-lane carrier waterfall an administrator maintains; tender_move_to_routing_guide uses it. |

### scheduling

| Write | Decision |
| --- | --- |
| `mutation assignWorkerShift` | Tool: `assign_worker_shift` |
| `mutation createShiftTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation endWorkerShiftAssignment` | Tool: `end_worker_shift_assignment` |
| `mutation proposeShiftSwap` | Tool: `propose_shift_swap` |
| `mutation setWorkerAvailabilityPreference` | Tool: `set_worker_availability_preference` |
| `mutation transitionShiftSwap` | Tool: `approve_shift_swap`, `reject_shift_swap`, `withdraw_shift_swap` |
| `mutation updateShiftTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |

### selfservice

| Write | Decision |
| --- | --- |
| `mutation createWorkerPolicy` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation decideProfileChange` | Exempt, attestation: Approving a driver's own change to their personal or banking details is a person verifying it came from the driver; an agent must not vouch for identity or pay details. |
| `mutation updateWorkerPolicy` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |

### sequenceconfig

| Write | Decision |
| --- | --- |
| `PUT /api/v1/sequence-configs/`<br>sequenceconfighandler.update | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |

### servicefailure

| Write | Decision |
| --- | --- |
| `PATCH /api/v1/service-failures/:serviceFailureID/`<br>servicefailurehandler.update<br>also `PUT /api/v1/service-failures/:serviceFailureID/` | Tool: `update_service_failure` |
| `POST /api/v1/service-failures/`<br>servicefailurehandler.createManual | Exempt, read-only: The service refuses every manual service failure: failures are opened only by evaluation against stop actuals, which evaluate_service_failures runs, so the route changes nothing. |
| `POST /api/v1/service-failures/:serviceFailureID/edi-214-payload/`<br>servicefailurehandler.buildEDI214Payload | Exempt, read-only: Builds the EDI 214 a service failure would send and returns it. |
| `POST /api/v1/service-failures/:serviceFailureID/resolve/`<br>servicefailurehandler.resolve | Tool: `resolve_service_failure` |
| `POST /api/v1/service-failures/:serviceFailureID/review/`<br>servicefailurehandler.review | Tool: `review_service_failure` |
| `POST /api/v1/service-failures/:serviceFailureID/void/`<br>servicefailurehandler.void | Tool: `void_service_failure` |
| `POST /api/v1/service-failures/bulk-evaluate/`<br>servicefailurehandler.bulkEvaluate | Tool: `evaluate_service_failures` |
| `POST /api/v1/service-failures/evaluate-shipment/:shipmentID/`<br>servicefailurehandler.evaluateShipment | Tool: `evaluate_service_failures` |
| `POST /api/v1/service-failures/evaluate-stop/:shipmentID/:stopID/`<br>servicefailurehandler.evaluateStop | Tool: `evaluate_service_failures` |

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
| `mutation acknowledgeShipmentComment` | Exempt, attestation: Acknowledging a comment says that the person it was addressed to has read it; only that person can say so. |
| `mutation autoRateShipment`<br>twin `POST /api/v1/shipments/:shipmentID/auto-rate/` | Tool: `rerate_shipment` |
| `mutation bulkTransferShipmentsToBilling`<br>twin `POST /api/v1/shipments/bulk-transfer-to-billing/` | Tool: `transfer_to_billing` |
| `mutation calculateShipmentDistance`<br>twin `POST /api/v1/shipments/calculate-distance/` | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `mutation calculateShipmentLoadingOptimization`<br>twin `POST /api/v1/shipments/loading-optimization/` | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `mutation calculateShipmentTotals`<br>twin `POST /api/v1/shipments/calculate-totals/` | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `mutation cancelShipment`<br>twin `POST /api/v1/shipments/:shipmentID/cancel/` | Tool: `cancel_shipment` |
| `mutation checkShipmentDuplicateBol` | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `mutation checkShipmentHazmatSegregation` | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `mutation createShipment`<br>twin `POST /api/v1/shipments/` | Tool: `create_shipment` |
| `mutation createShipmentComment`<br>twin `POST /api/v1/shipments/:shipmentID/comments/` | Tool: `add_shipment_comment` |
| `mutation deleteShipmentComment`<br>twin `DELETE /api/v1/shipments/:shipmentID/comments/:commentID/` | Tool: `delete_shipment_comment` |
| `mutation duplicateShipment`<br>twin `POST /api/v1/shipments/duplicate/` | Tool: `duplicate_shipment` |
| `mutation pinShipmentComment` | Tool: `pin_shipment_comment` |
| `mutation previewShipmentContractRate` | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `mutation recalculateShipmentDistance`<br>twin `POST /api/v1/shipments/:shipmentID/recalculate-distance/` | Tool: `recalculate_shipment_distance` |
| `mutation resolveShipmentComment` | Tool: `resolve_shipment_comment` |
| `mutation transferShipmentOwnership`<br>twin `POST /api/v1/shipments/:shipmentID/transfer-ownership/` | Tool: `transfer_shipment_ownership` |
| `mutation transferShipmentToBilling`<br>twin `POST /api/v1/shipments/:shipmentID/transfer-to-billing/` | Tool: `transfer_to_billing` |
| `mutation transferShipmentToBillingItems` | Tool: `transfer_to_billing` |
| `mutation uncancelShipment`<br>twin `POST /api/v1/shipments/:shipmentID/uncancel/` | Tool: `uncancel_shipment` |
| `mutation unpinShipmentComment` | Tool: `unpin_shipment_comment` |
| `mutation unresolveShipmentComment` | Tool: `resolve_shipment_comment` |
| `mutation updateShipment`<br>twin `PUT /api/v1/shipments/:shipmentID/` | Tool: `update_shipment` |
| `mutation updateShipmentComment`<br>twin `PUT /api/v1/shipments/:shipmentID/comments/:commentID/` | Tool: `edit_shipment_comment` |
| `POST /api/v1/shipments/:shipmentID/holds/`<br>shipmenthandler.createHold | Tool: `place_shipment_hold` |
| `POST /api/v1/shipments/:shipmentID/holds/:holdID/release/`<br>shipmenthandler.releaseHold | Tool: `release_shipment_hold` |
| `POST /api/v1/shipments/auto-cancel/`<br>shipmenthandler.autoCancelShipments | Exempt, infrastructure: The sweep the Temporal schedule runs over every organization to cancel shipments past the auto-cancel threshold; a person does not decide which shipments it takes, and cancel_shipment cancels one. |
| `POST /api/v1/shipments/check-for-duplicate-bols/`<br>shipmenthandler.checkForDuplicateBOLs | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `POST /api/v1/shipments/check-hazmat-segregation/`<br>shipmenthandler.checkHazmatSegregation | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `POST /api/v1/shipments/delay/`<br>shipmenthandler.delayShipments | Exempt, infrastructure: The sweep the Temporal schedule runs to mark late shipments delayed from their stop windows; evaluate_service_failures records a late stop for one shipment. |
| `POST /api/v1/shipments/previous-rates/`<br>shipmenthandler.getPreviousRates | Exempt, read-only: Computes a figure or a check for the shipment form and saves nothing. |
| `PUT /api/v1/shipments/:shipmentID/holds/:holdID/`<br>shipmenthandler.updateHold | Tool: `update_shipment_hold` |

### shipmentcontrol

| Write | Decision |
| --- | --- |
| `PUT /api/v1/shipment-controls/`<br>shipmentcontrolhandler.update | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |

### shipmentmove

| Write | Decision |
| --- | --- |
| `POST /api/v1/shipment-moves/:moveID/split/`<br>shipmentmovehandler.splitMove | Tool: `split_move_at_relay` |
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
| `DELETE /api/v1/stored-mileages/:storedMileageID/`<br>storedmileagehandler.delete | Exempt, infrastructure: A stored mileage is a cached routing answer; clearing one only makes the next rating ask the routing provider again. |

### tablechangealert

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/tca/subscriptions/:id`<br>tablechangealerthandler.deleteSubscription | Tool: `delete_table_change_alert` |
| `PATCH /api/v1/tca/subscriptions/:id/pause`<br>tablechangealerthandler.pauseSubscription | Tool: `set_table_change_alert_status` |
| `PATCH /api/v1/tca/subscriptions/:id/resume`<br>tablechangealerthandler.resumeSubscription | Tool: `set_table_change_alert_status` |
| `POST /api/v1/tca/subscriptions/`<br>tablechangealerthandler.createSubscription | Tool: `create_table_change_alert` |
| `PUT /api/v1/tca/subscriptions/:id`<br>tablechangealerthandler.updateSubscription | Tool: `update_table_change_alert` |

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
| `mutation deleteTimeEntry` | Exempt, attestation: Time clock entries are the wage record a worker's pay is computed from; only the worker or a supervisor who saw the time may change them. |
| `mutation generatePayrollExport` | Tool: `generate_payroll_export` |
| `mutation recordTimeEntry` | Exempt, attestation: Time clock entries are the wage record a worker's pay is computed from; only the worker or a supervisor who saw the time may enter them. |
| `mutation transitionTimesheet` | Exempt, attestation: Submitting and approving hours for payroll is a sign-off by the worker and their manager. |
| `mutation voidPayrollExport` | Tool: `void_payroll_export` |

### tractor

| Write | Decision |
| --- | --- |
| `mutation bulkUpdateTractorStatus`<br>twin `POST /api/v1/tractors/bulk-update-status/` | Tool: `update_tractor_status` |
| `mutation createTractor`<br>twin `POST /api/v1/tractors/` | Tool: `create_tractor` |
| `mutation locateTractor` | Tool: `locate_tractor` |
| `mutation patchTractor`<br>twin `PATCH /api/v1/tractors/:tractorID/` | Tool: `update_tractor` |
| `mutation updateTractor`<br>twin `PUT /api/v1/tractors/:tractorID/` | Tool: `update_tractor` |

### trailer

| Write | Decision |
| --- | --- |
| `mutation bulkUpdateTrailerStatus`<br>twin `POST /api/v1/trailers/bulk-update-status/` | Tool: `update_trailer_status` |
| `mutation createTrailer`<br>twin `POST /api/v1/trailers/` | Tool: `create_trailer` |
| `mutation locateTrailer`<br>twin `POST /api/v1/trailers/:trailerID/locate/` | Tool: `locate_trailer` |
| `mutation patchTrailer`<br>twin `PATCH /api/v1/trailers/:trailerID/` | Tool: `update_trailer` |
| `mutation updateTrailer`<br>twin `PUT /api/v1/trailers/:trailerID/` | Tool: `update_trailer` |

### user

| Write | Decision |
| --- | --- |
| `DELETE /api/v1/users/:userID/mfa/`<br>userhandler.resetUserMFA | Exempt, security: Resetting a person's second sign-in factor decides who can sign in as them. |
| `DELETE /api/v1/users/me/profile-picture/`<br>userhandler.deleteProfilePicture | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `PATCH /api/v1/users/:userID/`<br>userhandler.patch | Exempt, security: User accounts, their status, memberships and passwords decide who can sign in and to what. |
| `PATCH /api/v1/users/me/settings/`<br>userhandler.updateMySettings | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |
| `POST /api/v1/users/:userID/permissions/simulate/`<br>userhandler.simulatePermissions | Exempt, read-only: Shows what a user could do with a set of roles and saves nothing. |
| `POST /api/v1/users/:userID/reset-password/`<br>userhandler.sendPasswordReset | Exempt, security: User accounts, their status, memberships and passwords decide who can sign in and to what. |
| `POST /api/v1/users/bulk-update-status/`<br>userhandler.bulkUpdateStatus | Exempt, security: User accounts, their status, memberships and passwords decide who can sign in and to what. |
| `POST /api/v1/users/me/change-password/`<br>userhandler.changeMyPassword | Exempt, security: User accounts, their status, memberships and passwords decide who can sign in and to what. |
| `POST /api/v1/users/me/mfa/recovery-codes/`<br>userhandler.regenerateRecoveryCodes | Exempt, security: A person's second sign-in factor proves who they are; an agent holds no credentials of its own and never enrolls or removes one. |
| `POST /api/v1/users/me/mfa/totp/confirm/`<br>userhandler.confirmTOTPEnrollment | Exempt, security: A person's second sign-in factor proves who they are; an agent holds no credentials of its own and never enrolls or removes one. |
| `POST /api/v1/users/me/mfa/totp/disable/`<br>userhandler.disableTOTP | Exempt, security: A person's second sign-in factor proves who they are; an agent holds no credentials of its own and never enrolls or removes one. |
| `POST /api/v1/users/me/mfa/totp/enroll/`<br>userhandler.beginTOTPEnrollment | Exempt, security: A person's second sign-in factor proves who they are; an agent holds no credentials of its own and never enrolls or removes one. |
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
| `mutation dismissWatchtowerItem` | Tool: `dismiss_watchtower_item` |
| `mutation handOffWatchtowerItem` | Exempt, agent-administration: Starts an agent on the item's subject, or publishes its event to the agents that subscribe; agents hand work to one another through delegate_task instead. |
| `mutation markWatchtowerSeen` | Exempt, user-preference: A person's own interface state; it changes nothing anyone else sees. |

### worker

| Write | Decision |
| --- | --- |
| `mutation approveWorkerPTO` | Tool: `approve_worker_pto` |
| `mutation bulkWorkerPTOAction` | Exempt, duplicate: Approves, rejects or cancels several requests at once; approve_worker_pto, reject_worker_pto and cancel_worker_pto each do one. |
| `mutation cancelWorkerPTO` | Tool: `cancel_worker_pto` |
| `mutation createWorkerPTO` | Tool: `request_worker_pto` |
| `mutation patchWorker`<br>twin `PATCH /api/v1/workers/:workerID/`<br>twin `PUT /api/v1/workers/:workerID/` | Exempt, attestation: Changes a worker's employment status, classification or personal record, which are HR decisions a person makes and signs. |
| `mutation rejectWorkerPTO` | Tool: `reject_worker_pto` |
| `mutation updateWorkerPTO` | Tool: `update_worker_pto` |
| `POST /api/v1/workers/`<br>workerhandler.create | Exempt, attestation: Hiring a worker is an employment decision with its eligibility and qualification checks; a person makes it. |

### workerchecklist

| Write | Decision |
| --- | --- |
| `mutation archiveWorkerChecklistTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation cancelWorkerChecklist` | Tool: `cancel_worker_checklist` |
| `mutation completeWorkerChecklistItem` | Tool: `update_worker_checklist_item` |
| `mutation createWorkerChecklistTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation markWorkerChecklistItemNotApplicable` | Tool: `update_worker_checklist_item` |
| `mutation reopenWorkerChecklistItem` | Tool: `update_worker_checklist_item` |
| `mutation restoreWorkerChecklistTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation skipWorkerChecklistItem` | Tool: `update_worker_checklist_item` |
| `mutation startWorkerChecklist` | Tool: `start_worker_checklist` |
| `mutation updateWorkerChecklistTemplate` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |

### workercredential

| Write | Decision |
| --- | --- |
| `mutation archiveWorkerCredential` | Tool: `archive_worker_credential` |
| `mutation archiveWorkerCredentialType` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation attachWorkerCredentialDocument` | Tool: `attach_worker_credential_document` |
| `mutation createWorkerCredential` | Tool: `record_worker_credential` |
| `mutation createWorkerCredentialType` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation restoreWorkerCredentialType` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation updateWorkerCredential` | Tool: `update_worker_credential` |
| `mutation updateWorkerCredentialType` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation verifyWorkerCredential` | Exempt, attestation: Verifying is a person vouching that they checked the credential against the original; an agent records, it does not vouch. |

### workerdqf

| Write | Decision |
| --- | --- |
| `mutation deleteEmploymentVerification` | Tool: `delete_employment_verification` |
| `mutation markEmploymentVerificationRequested` | Tool: `log_employment_verification_request` |
| `mutation recordEmploymentVerification` | Tool: `record_employment_verification` |
| `mutation recordEmploymentVerificationFollowUp` | Tool: `log_employment_verification_request` |
| `mutation updateEmploymentVerification` | Tool: `update_employment_verification` |

### workerdrugalcohol

| Write | Decision |
| --- | --- |
| `mutation cancelDotRandomDraw` | Tool: `cancel_dot_random_draw` |
| `mutation cancelDotTest` | Tool: `cancel_dot_test` |
| `mutation completeClearinghouseQuery` | Exempt, attestation: A Clearinghouse query's outcome is what FMCSA returned to the person who ran it under the carrier's account; only they record it. |
| `mutation createDotRandomPool` | Exempt, configuration: A random testing pool sets who is tested and at what annual rate under 49 CFR 382.305; the designated employer representative maintains it. |
| `mutation finalizeDotRandomDraw` | Tool: `finalize_dot_random_draw` |
| `mutation recordClearinghouseQuery` | Exempt, attestation: A Clearinghouse query is made under the carrier's FMCSA account with the driver's consent; the person who ran it records it. |
| `mutation recordDotTest` | Tool: `schedule_dot_test` |
| `mutation recordDotTestResult` | Exempt, attestation: A test result is what the lab and medical review officer reported; the person who received it records it, and a positive result reaches the Clearinghouse. |
| `mutation recordDotViolation` | Exempt, attestation: A drug or alcohol violation is reported to the FMCSA Clearinghouse and removes the driver from safety-sensitive work; a person records it. |
| `mutation runDotRandomDraw` | Tool: `run_dot_random_draw` |
| `mutation updateDotRandomDrawEntry` | Tool: `update_dot_random_selection` |
| `mutation updateDotRandomPool` | Exempt, configuration: A random testing pool sets who is tested and at what annual rate under 49 CFR 382.305; the designated employer representative maintains it. |
| `mutation updateDotViolation` | Exempt, attestation: A drug or alcohol violation and its return-to-duty steps are reported to the FMCSA Clearinghouse; a person records them. |

### workeremployment

| Write | Decision |
| --- | --- |
| `mutation amendWorkerEmploymentEvent` | Exempt, attestation: An employment event is a hire, rehire, leave or termination decision on the worker's record; a person makes and amends it. |
| `mutation recordWorkerEmploymentEvent` | Exempt, attestation: An employment event is a hire, rehire, leave or termination decision on the worker's record; a person makes it. |

### workerinjury

| Write | Decision |
| --- | --- |
| `mutation certifyOshaSummary` | Exempt, attestation: OSHA requires a company executive to certify the annual injury summary. |
| `mutation deleteWorkerInjury` | Tool: `delete_worker_injury` |
| `mutation recordWorkerInjury` | Tool: `record_worker_injury` |
| `mutation saveOshaSummary` | Exempt, attestation: The OSHA 300A summary is certified by a company executive and posted for employees; the figures are theirs to sign. |
| `mutation uncertifyOshaSummary` | Exempt, attestation: Withdraws the executive certification of the annual injury summary. |
| `mutation updateWorkerInjury` | Tool: `update_worker_injury` |

### workerleave

| Write | Decision |
| --- | --- |
| `mutation closeLeaveCase` | Tool: `close_leave_case` |
| `mutation decideLeaveCase` | Exempt, attestation: Approving or denying leave, and designating it FMLA, is the employer's notice to the employee under 29 CFR 825.300; a person decides it. |
| `mutation deleteLeaveDay` | Tool: `delete_leave_day` |
| `mutation openLeaveCase` | Tool: `open_leave_case` |
| `mutation recordLeaveCertification` | Exempt, attestation: The certification is the health care provider's statement received from the employee; the person who received it records it. |
| `mutation recordLeaveDay` | Tool: `record_leave_day` |
| `mutation requestLeaveCertification` | Tool: `request_leave_certification` |
| `mutation updateLeaveCase` | Tool: `update_leave_case` |
| `mutation updateLeaveControl` | Exempt, configuration: An organization-wide control an administrator sets once; every later write depends on it. |
| `mutation updateLeaveDay` | Tool: `update_leave_day` |

### workersafety

| Write | Decision |
| --- | --- |
| `mutation acknowledgeMyDisciplinaryAction` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation closeWorkerSafetyEvent` | Tool: `change_worker_safety_event_status` |
| `mutation createWorkerSafetyEvent` | Tool: `open_worker_safety_event` |
| `mutation deleteWorkerRecognition` | Tool: `delete_worker_recognition` |
| `mutation deleteWorkerSafetyEvent` | Tool: `delete_worker_safety_event` |
| `mutation giveWorkerRecognition` | Tool: `give_worker_recognition` |
| `mutation issueDisciplinaryAction` | Exempt, attestation: Discipline is a manager's decision on an employee's record, which the driver acknowledges in the portal; a person issues it. |
| `mutation reopenWorkerSafetyEvent` | Tool: `change_worker_safety_event_status` |
| `mutation rescindDisciplinaryAction` | Exempt, attestation: Rescinding discipline reverses a manager's decision on an employee's record; the person who owns that decision makes it. |
| `mutation reviewWorkerSafetyEvent` | Tool: `change_worker_safety_event_status` |
| `mutation updateWorkerSafetyEvent` | Tool: `update_worker_safety_event` |

### workertraining

| Write | Decision |
| --- | --- |
| `mutation acknowledgeMyTraining` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation archiveTrainingCourse` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation assignRequiredWorkerTraining` | Tool: `assign_required_worker_training` |
| `mutation assignWorkerTraining` | Tool: `assign_worker_training` |
| `mutation attachWorkerTrainingDocument` | Tool: `attach_worker_training_document` |
| `mutation bulkAssignTraining` | Tool: `assign_worker_training` |
| `mutation cancelWorkerTraining` | Tool: `close_worker_training` |
| `mutation completeWorkerTraining` | Tool: `record_training_completion` |
| `mutation createTrainingCourse` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation restoreTrainingCourse` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation startMyTraining` | Exempt, counterparty: The driver doing this for themselves in their own portal; an agent acting for the organization must not act as the driver. |
| `mutation updateTrainingCourse` | Exempt, configuration: Templates and rules an administrator authors, reviews and publishes; they decide how every later record is produced. |
| `mutation waiveWorkerTraining` | Tool: `close_worker_training` |
