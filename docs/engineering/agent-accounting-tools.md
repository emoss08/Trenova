# Agent tools for the ledger and the accounting system

What an agent can do to the general ledger, the fiscal calendar and the connection to the
organization's accounting system, how far each step may run without a person, and which
accounting writes no agent makes.

Read [agent-runtime.md](agent-runtime.md) for tiers, egress classes and taint, and
[proposal-previews.md](proposal-previews.md) for how a proposal is previewed and approved.
The generated [ai-tool-safety.md](ai-tool-safety.md) is the authority on each policy, and
[agent-write-coverage.md](agent-write-coverage.md) on which write each tool covers; this
page explains them.

## The rule

Every write calls the service its page calls — `manualjournalservice`,
`journalreversalservice`, `fiscalperiodservice`, `accountingmappingservice`,
`accountingsyncservice`, `bankreceiptworkitemservice` — and never writes around it. Each
service exposes the plan its write runs on (`PlanCreateDraft`, `PlanUpdateDraft`,
`PlanSubmit`, `PlanCancel` and `PlanPost` on manual journals; `PlanCreate`, `PlanCancel` and
`PlanPost` on reversals; `PlanTransition` on fiscal periods; `PlanConfirm` and `PlanReject`
on mappings; `PlanAssign` and `PlanStartReview` on work items). A tool's preview renders that
plan and its `Validate` runs the same plan, so a proposal shows what the write decides and
is refused for the reasons the write would refuse it. Anything that books to the ledger,
moves a fiscal period, reverses an entry or sends to the accounting system is money-class,
stops at Propose, and runs only from a person's approval
(`ToolExecuteParams.ApprovedFromProposal()`), as that person.

## Manual journals and reversals

```
list_gl_accounts ──► draft_manual_journal ──► revise_manual_journal_draft
                           │
                           ▼
                  submit_manual_journal ──► (a person approves) ──► post_manual_journal
                           │
                           └─► cancel_manual_journal (before it posts)

list_journal_entries / get_journal_entry ──► request_journal_reversal
                           ──► (a person approves) ──► post_journal_reversal
                           └─► cancel_journal_reversal (before it posts)
```

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `list_manual_journals`, `get_manual_journal`, `list_journal_reversals` | manual journal, journal reversal / read | reads only | automatic | — |
| `draft_manual_journal` | manual journal / create | inside | Propose → Ask first (from outside content: Propose) | no |
| `revise_manual_journal_draft` | manual journal / update | inside | Propose → Ask first (from outside content: Propose) | no |
| `submit_manual_journal` | manual journal / submit | inside | Propose → Ask first | no |
| `cancel_manual_journal` | manual journal / cancel | inside | Propose → Ask first (from outside content: Propose) | no |
| `post_manual_journal` | manual journal / approve | money | Propose → Propose | yes |
| `request_journal_reversal` | journal reversal / create | money | Propose → Propose | yes |
| `cancel_journal_reversal` | journal reversal / cancel | inside | Propose → Ask first (from outside content: Propose) | no |
| `post_journal_reversal` | journal reversal / approve | money | Propose → Propose | yes |

A draft books nothing, and the preview shows its lines by account, the debit and credit
totals and whether it balances. Submitting an agent's journal where the organization has
manual journal approval turned off would approve it on submission, so the tool refuses an
agent actor that plan: a person submits it. Approving and rejecting a journal or a reversal
is the second person's sign-off and is exempt as attestation; an agent prepares and never
stands in as the approver. `request_journal_reversal` is money-class because, where approval
is off, the request is approved at once; its preview shows the original entry's lines with
debits and credits swapped, in the functional currency, and the period it will book in.

Dates are `YYYY-MM-DD`. Amounts are decimal strings. Journal lines are a closed schema: a GL
account from `list_gl_accounts`, a description, exactly one of debit or credit, and an
optional customer or location.

## Fiscal periods

| Tool | Operation | Class | Most it may do | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `open_fiscal_period` | activate | money | Propose | yes |
| `lock_fiscal_period` | lock | money | Propose | yes |
| `unlock_fiscal_period` | unlock | money | Propose | yes |
| `close_fiscal_period` | close | money | Propose | yes |
| `reopen_fiscal_period` | reopen (with a reason) | money | Propose | yes |

Each is one move through `fiscalperiodservice.Transition`; `PlanTransition` refuses what the
move refuses (periods open and close in order, a period in a closed year never reopens, a
close with blockers is refused). `list_fiscal_periods` and `get_fiscal_close_blockers` pick the
period and say whether it can close. Creating, editing and deleting periods, and every fiscal
year write, lay out the calendar and are configuration; closing and reopening a fiscal year
book and withdraw the year-end closing entries and are attestation.

## The accounting system

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `confirm_accounting_mapping_proposals` | accounting integration / update | inside | Propose → Ask first | no |
| `reject_accounting_mapping_proposal` | accounting integration / update | inside | Propose → Ask first | no |
| `release_accounting_sync` | accounting sync / update | money | Propose → Propose | yes |
| `change_accounting_backfill` | accounting integration / manage | inside | Ask first → Ask first | no |

Confirming takes each proposal as `list_accounting_mapping_gaps` showed it, mapping and
external id together, and a proposal that changed since it was read is refused. Releasing sends
records a review policy held as `AwaitingApproval`; the record subset lets the person approving
untick some. Changing a backfill pauses, resumes or cancels the one in progress and refuses a
run with no person behind it. Connecting, disconnecting and backfilling setup stay
configuration or security as they were.

## Bank receipts

`receivable_triage_bank_receipt_work_item` assigns a reconciliation work item to a person or marks it
under review (inside, Ask first → Automatic). It matches and closes nothing;
`receivable_match_bank_receipt` and `receivable_resolve_bank_receipt_work_item` do that. Importing a bank receipt
or a batch records what the bank reported and is infrastructure: an agent matches receipts,
it never writes one.

## Factoring

`factoringservice.LedgerService` books what a factor does with each invoice on a schedule:
`PlanFunding`/`RecordFunding`, `PlanRelease`/`ReleaseReserve`, `PlanChargeback`/
`RecordChargeback` and `PlanChargebackReversal`/`ReverseChargeback`. Each plan returns the
journal's lines by account role and account, the period it books in (the first open day when
the asked day falls in a closed period) and whether it waits for journal approval; the tools
preview that plan and validate with it.

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `get_factoring_reserve_balance` | factoring reserve / read | reads only | automatic | — |
| `record_factoring_funding` | factoring reserve / create | money | Propose → Propose | yes |
| `release_factoring_reserve` | factoring reserve / create | money | Propose → Propose | yes |
| `record_factoring_chargeback` | factoring reserve / create | money | Propose → Propose | yes |
| `reverse_factoring_chargeback` | factoring reserve / create | money | Propose → Propose | yes |

Amounts are decimal strings and dates `YYYY-MM-DD`. A funding item takes the advance and fee
the factor actually paid, defaulting to the expected ones; the fee is the figure that
balances a release or a chargeback, so whatever the factor kept beyond the expected reserve or
repurchase price is booked as its fee. The ledger service refuses an actor with no person
behind it (an API key), because the journal and customer ledger rows name who booked them.
The factoring clerk holds the balance read, funding and release; chargebacks and their
reversals are a person's and sit on no template.

`factoringservice.RemittanceService` reads a factor's funding or reserve report (a CSV or
spreadsheet by its headers, or a PDF through the `RemittanceExtraction` model task), matches
each line to the factor's schedule items, and books the matched lines through the same
`LedgerService` in one transaction: funding and rejection lines become one `RecordFunding`
per schedule, reserve releases and chargebacks one movement each, and every reserve entry
records the remittance line it came from. `PlanApply` returns each line's planned movement, so
the apply tool's preview is the journal.

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `list_factoring_remittances` | factoring remittance / read | reads only | automatic | — |
| `get_factoring_remittance` | factoring remittance / read | reads only | automatic | — |
| `resolve_factoring_remittance_line` | factoring remittance / update | internal | Propose → ActWithApproval | no |
| `rematch_factoring_remittance` | factoring remittance / update | internal | Propose → ActWithApproval | no |
| `apply_factoring_remittance` | factoring remittance / import (Apply) | money | Propose → Propose | yes |
| `discard_factoring_remittance` | factoring remittance / cancel | internal | Propose → ActWithApproval | no |

Resolving a line books nothing: it matches the line to a schedule item, unmatches it, ignores
it with a reason, restores it, or changes what it records. The clerk resolves only lines whose
best suggestion names the same invoice and amount, proposes ignoring fees and adjustments, and
then proposes the apply. Discarding is a person's and sits on no template.

## Collections and statements

The collections writes call the services the collections panel and the statement page call:
`collectionservice` (`LogActivity`, `AssignCollector`, `ScheduleFollowUp`),
`paymentreminderservice.SendNow` and `customerstatementservice.Send`. Each exposes the plan its
write runs on (`PlanLogActivity`, `PlanAssignCollector`, `PlanScheduleFollowUp`,
`PreviewSendNow`, `PreviewSend`), sharing the write's validation and arithmetic, so a proposal
is refused for exactly what the write would refuse. A plan reads the account from the
worklist row rather than taking it `FOR UPDATE`, because previews run in a read-only
transaction.

| Tool | Resource / operation | Class | Default → most | Taint |
| --- | --- | --- | --- | --- |
| `receivable_record_collection_activity` | accounts receivable / update | inside | Propose → Automatic | held: proposed once the run read outside text |
| `receivable_assign_collector` | accounts receivable / update | inside | Propose → Ask first | — |
| `receivable_schedule_follow_up` | accounts receivable / update | inside | Propose → Automatic | held |
| `receivable_send_payment_reminder` | accounts receivable / update | outside | Propose → Ask first | held (outside) |
| `receivable_send_statement` | accounts receivable / update | outside | Propose → Ask first | held (outside) |

A logged promise to pay becomes the account's standing promise: the scheduled reminder sweep
skips the customer until it falls due, and the follow-up moves to the day after unless one is
given. That is why logging is held once a run has read what a customer wrote: a promise
invented in an email would quietly stop the dunning. The reminder and statement previews are
the email the send renders, with every recipient on the customer's email profile; the
statement preview leaves the PDF out (it is named as the attachment) because rendering it would
outrun the preview's time budget, and a statement dated today is digested by day.

## Who holds them

| Template | Runs | Holds |
| --- | --- | --- |
| Books keeper | unattended, on accounting sync events and the weekly reconciliation, starting in shadow | `confirm_accounting_mapping_proposals`, `reject_accounting_mapping_proposal`, `release_accounting_sync`, `pause_accounting_sync`, `resume_accounting_sync`, `create_accounting_reference_record`, `refresh_accounting_reference_data`, and for period end `list_fiscal_periods`, `get_fiscal_close_blockers`, `lock_fiscal_period`, `close_fiscal_period` |
| Cash application agent | unattended, on bank receipt exceptions | `receivable_triage_bank_receipt_work_item`, to put an item it leaves for a person under review |
| Receivables assistant | in chat, for the person working receivables | `receivable_record_collection_activity`, `receivable_assign_collector`, `receivable_schedule_follow_up`, `receivable_send_payment_reminder`, `receivable_send_statement`, beside the worklist and statement reads |
| Factoring clerk | unattended, on factoring readiness, schedule, remittance and chargeback-window events | `get_factoring_reserve_balance`, `record_factoring_funding`, `release_factoring_reserve`, `list_factoring_remittances`, `get_factoring_remittance`, `resolve_factoring_remittance_line`, `rematch_factoring_remittance`, `apply_factoring_remittance` |

The agent permission ceiling (`permission/agent.go`) gains lock and close on fiscal periods
for the books keeper; both tools stop at a proposal and run as the approver. Mapping review and
releasing sync use grants the books keeper already had.

The books keeper also holds the four sync and mapping writes that were on no template. Its
Ask first ceiling holds each to a proposal a person approves, whatever its own policy allows:

| Tool | Default → most | Why the books keeper holds it |
| --- | --- | --- |
| `pause_accounting_sync` | Automatic → Automatic | It wakes when the connection degrades; pausing holds what waits, loses nothing, and its prompt proposes it only while the connection is failing, with the reason. |
| `resume_accounting_sync` | Ask first → Ask first | Everything held goes out at once, so it proposes resuming only once `get_accounting_sync_status` shows the connection answering and the reason for the pause is over. |
| `create_accounting_reference_record` | Propose → Ask first | A missing item, customer or vendor is the commonest mapping gap; it proposes creating one only when `get_accounting_mapping`'s search finds no match. Accounts, terms and payment methods stay the bookkeeper's. |
| `refresh_accounting_reference_data` | Automatic → Automatic | It rereads the books after a person says they added or renamed something there, and writes nothing to them. |

Pause and resume act on the whole connection, not one document, which is why neither runs on
the books keeper's own say: an organization that raises its ceiling past Ask first lets an
outage pause sending unattended, and resuming still waits for a person.

The manual journal and reversal tools, `open_fiscal_period`, `unlock_fiscal_period`,
`reopen_fiscal_period`, `change_accounting_backfill`, `request_accounting_backfill`,
`clear_accounting_mapping` and `redate_accounting_sync` are deliberately on no template, and a
template test keeps them off every one. Each is privileged: it starts from a person's request,
and none belongs to an unattended desk, since posting and approving are refused to an agent
principal outright and drafting a journal on its own judgement is not the books keeper's job.
No chat template takes them either. The billing assistant, the one chat template with
accounting work, has no room left under the eight-tool headroom; the receivables assistant
works customer cash; and the fuel and IFTA clerk, the master data steward and the workforce
coordinator keep tax, master and worker records, never the ledger. An organization adds them
to an agent it builds in AI control.

## Known limits

- An agent made from the books keeper or cash application template before these tools existed
  keeps the tools it was saved with; an administrator adds them in AI control.
- A manual journal preview labels at most the accounts its lines name; a line whose account
  was deleted since shows its id.
- `release_accounting_sync` takes at most 50 records per proposal and
  `confirm_accounting_mapping_proposals` at most 50 proposals.
