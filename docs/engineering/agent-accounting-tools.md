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

`triage_bank_receipt_work_item` assigns a reconciliation work item to a person or marks it
under review (inside, Ask first → Automatic). It matches and closes nothing;
`match_bank_receipt` and `resolve_bank_receipt_work_item` do that. Importing a bank receipt
or a batch records what the bank reported and is infrastructure: an agent matches receipts,
it never writes one.

## Who holds them

| Template | Runs | Holds |
| --- | --- | --- |
| Books keeper | unattended, on accounting sync events and the weekly reconciliation, starting in shadow | `confirm_accounting_mapping_proposals`, `reject_accounting_mapping_proposal`, `release_accounting_sync`, `pause_accounting_sync`, `resume_accounting_sync`, `create_accounting_reference_record`, `refresh_accounting_reference_data`, and for period end `list_fiscal_periods`, `get_fiscal_close_blockers`, `lock_fiscal_period`, `close_fiscal_period` |
| Cash application agent | unattended, on bank receipt exceptions | `triage_bank_receipt_work_item`, to put an item it leaves for a person under review |

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
