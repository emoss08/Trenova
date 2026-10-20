# Agent tools for workforce, safety and driver pay records

What an agent can do to a worker's safety, compliance and HR records, to shipment permits and
to the driver pay decisions that come in from the driver portal, how far each step may run
without a person, and which of those writes no agent makes.

Read [agent-runtime.md](agent-runtime.md) for tiers, egress classes and taint, and
[proposal-previews.md](proposal-previews.md) for how a proposal is previewed and approved.
The generated [ai-tool-safety.md](ai-tool-safety.md) is the authority on each policy, and
[agent-write-coverage.md](agent-write-coverage.md) on which write each tool covers; this page
explains them.

## The rule

Every write calls the service its resolver or handler calls — `workersafetyservice`,
`workerdrugalcoholservice`, `workerleaveservice`, `workertrainingservice`,
`workerchecklistservice`, `performancereviewservice`, `workerdqfservice`,
`workercredentialservice`, `workerinjuryservice`, `workerptoservice`, `ptoledgerservice`,
`permitservice`, `driversettlementservice` and `timesheetservice` — and its policy names the
permission resource and operation that resolver checks. Each service exposes the plan its
write runs on (`PlanCreateEvent`, `PlanRecordTest`, `PlanRunDraw`, `PlanOpenCase`,
`PlanAssign`, `PlanStart`, `PlanUpdateReview`, `PlanRecordVerification`, `PlanCreate`,
`PlanRecordInjury`, `PlanAdjust`, `PlanUpdatePermit`, `PlanResolveDispute`,
`PlanGenerateExport` and the rest), returning the record as it stands and as the write would
leave it (`kernelports.RecordChange[T]`). A tool's preview renders that plan and its `Validate`
runs the same plan, so a proposal shows what the write decides and is refused for the reasons
the write refuses it. `UpdatePermit` runs `PlanUpdatePermit` itself, so the REST route and the
tool both refuse a permit that belongs to another shipment.

Four lines decide where a write lands:

- **What the driver writes or signs is theirs.** Portal submissions, acknowledgements and
  elections are `counterparty` or `attestation`; an agent acting for the organization never
  acts as the driver.
- **What a person certifies is theirs.** Verifying a credential, entering a test result,
  recording a Clearinghouse query or a drug and alcohol violation, deciding or certifying
  leave, signing a review, certifying the OSHA 300A, disciplining, hiring and changing
  employment status are `attestation`. An agent drafts the record the person signs.
- **Policy is configuration.** Random testing pools, PTO policy assignments and similar
  editors belong to an administrator.
- **Anything that reaches the driver or moves pay is at most Ask first, and a decision on
  pay is Propose, run only from a person's approval** (`ToolExecuteParams.ApprovedFromProposal()`),
  as that person.

Drug and alcohol, injury, leave, qualification and pay records keep the permission their
resolver checks: a read tool is gated on the same resource's read, and its free text
(descriptions, findings, notes, medical detail) follows the field sensitivity the person, or
the agent's data access, may see. An agent at Internal sees the case, never the body part.

## Safety events, violations and recognition

| Tool | Resource / operation | Class | Default → most | Person's approval |
| --- | --- | --- | --- | --- |
| `worker_list_safety_events` | worker safety event / read | reads only | automatic | — |
| `worker_open_safety_event` | worker safety event / create | inside | Propose → Ask first | no |
| `worker_update_safety_event` | worker safety event / update | inside | Propose → Ask first | no |
| `worker_set_safety_event_status` | worker safety event / close | inside | Propose → Ask first | no |
| `worker_delete_safety_event` | worker safety event / delete | inside | Propose → Propose | no |
| `worker_record_safety_violation`, `worker_update_safety_violation` | worker safety event / create, update | inside | Propose → Ask first | no |
| `worker_delete_safety_violation` | worker safety event / delete | inside | Propose → Propose | no |
| `worker_record_recognition` | worker recognition / create | driver | Propose → Ask first | no |
| `worker_delete_recognition` | worker recognition / delete | inside | Propose → Propose | no |

An event's points default from the house scale for its kind, severity, preventability and
inspection result; closing needs a resolution. Issuing and rescinding discipline are
attestation: the manager decides and the driver acknowledges it in the portal.

## Drug and alcohol testing

| Tool | Resource / operation | Class | Default → most | Person's approval |
| --- | --- | --- | --- | --- |
| `worker_list_dot_tests`, `worker_list_dot_random_draws`, `worker_get_dot_random_draw` | DOT test, random pool / read | reads only | automatic | — |
| `worker_schedule_dot_test` | worker DOT test / create | inside | Propose → Ask first | no |
| `worker_cancel_dot_test` | worker DOT test / cancel | inside | Propose → Propose | no |
| `worker_run_dot_random_draw` | DOT random pool / manage | inside | Propose → Ask first | no |
| `worker_update_dot_random_selection` | DOT random pool / manage | inside | Propose → Ask first | no |
| `worker_cancel_dot_random_draw` | DOT random pool / manage | inside | Propose → Propose | no |
| `worker_finalize_dot_random_draw` | DOT random pool / manage | inside | Propose → Propose | yes |

`worker_schedule_dot_test` records an order and is forced to `Scheduled` with a pending result.
`worker_run_dot_random_draw` previews the pool size and each substance's target and names nobody: the
names are chosen only when the round is drawn, under a seed nobody sees first, so a proposal
cannot be used to shop for a different selection. A draft round stands only once a person
finalizes it. Results, violations, Clearinghouse queries and return-to-duty steps reach FMCSA
and are attestation; pools are configuration.

## Leave

| Tool | Resource / operation | Class | Default → most | Person's approval |
| --- | --- | --- | --- | --- |
| `worker_list_leave_cases` | worker leave / read | reads only | automatic | — |
| `worker_open_leave_case`, `worker_update_leave_case` | worker leave / create, update | inside | Propose → Ask first | no |
| `worker_request_leave_certification` | worker leave / update | inside | Propose → Ask first | no |
| `worker_record_leave_day`, `worker_update_leave_day` | worker leave / create, update | inside | Propose → Ask first | no |
| `worker_delete_leave_day` | worker leave / update | inside | Propose → Propose | no |
| `worker_close_leave_case` | worker leave / approve | inside | Propose → Propose | yes |

Deciding a case, and designating it FMLA, is the employer's notice under 29 CFR 825.300, and
recording the provider's certification is the person who received it: both attestation.

## Training, checklists and reviews

| Tool | Resource / operation | Class | Default → most | Person's approval |
| --- | --- | --- | --- | --- |
| `worker_list_training`, `worker_list_training_courses` | worker training, training course / read | reads only | automatic | — |
| `worker_assign_training` | worker training / assign | driver | Propose → Ask first | no |
| `worker_assign_required_training` | worker training / assign | driver | Propose → Ask first | no |
| `worker_record_training_completion`, `worker_attach_training_document` | worker training / update | inside | Propose → Ask first | no |
| `worker_close_training` | worker training / cancel | inside | Propose → Propose | no |
| `worker_list_checklists` | worker checklist / read | reads only | automatic | — |
| `worker_start_checklist`, `worker_update_checklist_item` | worker checklist / create, update | inside | Propose → Ask first | no |
| `worker_cancel_checklist` | worker checklist / cancel | inside | Propose → Propose | no |
| `worker_list_performance_reviews` | performance review / read | reads only | automatic | — |
| `worker_start_performance_review`, `worker_update_performance_review` | performance review / create, update | inside | Propose → Ask first | no |
| `worker_delete_performance_review` | performance review / delete | inside | Propose → Propose | no |

`worker_assign_training` takes up to 50 workers and 10 courses; one pair goes through
`Assign`, more through `BulkAssign`, and a run that would assign nothing is refused.
`worker_update_checklist_item` completes, skips, marks not applicable or reopens one item.
`worker_update_performance_review` merges scores by rating key, refuses a key the review does not
have, and replaces the goals only when it is given them; submitting, reopening and closing a
review are the reviewer's sign-off.

## Qualification file, credentials and injuries

| Tool | Resource / operation | Class | Default → most | Person's approval |
| --- | --- | --- | --- | --- |
| `worker_list_employment_verifications` | qualification / read | reads only | automatic | — |
| `worker_record_employment_verification`, `worker_update_employment_verification` | qualification / create, update | inside | Propose → Ask first | no |
| `worker_record_employment_verification_request` | qualification / update | inside | Propose → Ask first | no |
| `worker_delete_employment_verification` | qualification / delete | inside | Propose → Propose | no |
| `worker_list_credentials` | worker credential / read | reads only | automatic | — |
| `worker_record_credential`, `worker_update_credential` | worker credential / create, update | inside | Propose → Ask first | no |
| `worker_attach_credential_document` | worker credential / update | inside | Propose → Ask first | no |
| `worker_archive_credential` | worker credential / archive | inside | Propose → Propose | no |
| `worker_list_injuries` | worker injury / read | reads only | automatic | — |
| `worker_record_injury`, `worker_update_injury` | worker injury / create, update | inside | Propose → Ask first | no |
| `worker_delete_injury` | worker injury / delete | inside | Propose → Propose | no |

A second active credential of a type is refused unless the call renews it, and the preview
shows the one a renewal archives. Changing a fact a person verified clears the verification,
and the preview says so. An injury takes the next case number for its year; its
classification follows the treatment unless a person gave one.

## Time off, permits, driver pay and payroll

| Tool | Resource / operation | Class | Default → most | Person's approval |
| --- | --- | --- | --- | --- |
| `worker_request_pto`, `worker_update_pto` | worker PTO / create, update | driver | Propose → Ask first | no |
| `worker_adjust_pto_balance` | worker PTO / manage | money | Propose → Propose | yes |
| `shipment_list_permits` | permit / read | reads only | automatic | — |
| `shipment_record_permit`, `shipment_update_permit` | permit / create, update | inside | Propose → Ask first | no |
| `settlement_start_dispute_review` | settlement dispute / update | driver | Propose → Ask first | no |
| `settlement_resolve_dispute` | settlement dispute / approve | driver or money, per call | Propose → Propose | yes |
| `settlement_list_driver_expenses` | driver expense / read | reads only, marked | automatic | — |
| `settlement_review_driver_expense` | driver expense / approve | driver or money, per call | Propose → Propose | yes |
| `settlement_list_payroll_exports` | timesheet / read | reads only | automatic | — |
| `settlement_generate_payroll_export`, `settlement_void_payroll_export` | timesheet / export | money | Propose → Propose | yes |

A time-off request under a policy that needs no approval is booked and the driver told at
once, which is why it is driver-visible; the preview says when that will happen. A dispute
decision with an adjustment, and an approved expense, classify as money; a denial or a
rejection is driver-visible. `settlement_list_driver_expenses` marks a row whose description the driver
wrote, so a turn that read it proposes every later write. Waiving a permit requirement is a
named person accepting the compliance risk and is attestation. Time clock entries are the wage
record and are attestation; bulk PTO decisions duplicate the single-request tools.

## Who holds them

| Template | Holds |
| --- | --- |
| Workforce coordinator (chat, Propose, Restricted data access) | `worker_list_time_off`, `worker_request_pto`, `worker_update_pto`, `worker_approve_pto`, `worker_reject_pto`, `worker_cancel_pto`, `worker_adjust_pto_balance`, `worker_list_leave_cases`, `worker_open_leave_case`, `worker_update_leave_case`, `worker_request_leave_certification`, `worker_record_leave_day`, `worker_update_leave_day`, `worker_delete_leave_day`, `worker_close_leave_case`, `worker_list_injuries`, `worker_record_injury`, `worker_update_injury`, `worker_delete_injury`, `worker_list_performance_reviews`, `worker_start_performance_review`, `worker_update_performance_review`, `worker_delete_performance_review`, `worker_record_recognition`, `worker_delete_recognition`, `worker_list_safety_events`, `worker_update_safety_event`, `worker_delete_safety_event`, `worker_update_safety_violation`, `worker_delete_safety_violation`, `worker_list_dot_tests`, `worker_cancel_dot_test`, `worker_list_dot_random_draws`, `worker_get_dot_random_draw`, `worker_run_dot_random_draw`, `worker_finalize_dot_random_draw`, `worker_cancel_dot_random_draw`, `worker_list_checklists`, `worker_start_checklist`, `worker_update_checklist_item`, `worker_cancel_checklist`, `worker_list_training`, `worker_attach_training_document`, `worker_close_training`, `worker_list_credentials`, `worker_archive_credential`, `worker_list_employment_verifications`, `worker_delete_employment_verification`, `shipment_list_permits`, `shipment_record_permit`, `shipment_update_permit` |
| Compliance assistant (chat, Ask first) | `worker_list_credentials`, `worker_record_credential`, `worker_update_credential`, `worker_attach_credential_document`, `worker_list_training`, `worker_list_training_courses`, `worker_assign_training`, `worker_assign_required_training`, `worker_record_training_completion`, `worker_list_employment_verifications`, `worker_record_employment_verification`, `worker_update_employment_verification`, `worker_record_employment_verification_request`, `worker_list_checklists`, `worker_update_checklist_item`, `worker_list_safety_events`, `worker_open_safety_event`, `worker_record_safety_violation`, `worker_set_safety_event_status`, `worker_list_dot_tests`, `worker_get_dot_random_draw`, `worker_schedule_dot_test`, `worker_update_dot_random_selection`, `worker_list_leave_cases`, `worker_get_hos`, `worker_request_credential_renewal`, `worker_place_dispatch_hold` |
| Settlements clerk (chat, Ask first) | `settlement_start_dispute_review`, `settlement_resolve_dispute`, `settlement_approve_driver`, `settlement_post_driver`, `settlement_approve_carrier`, `settlement_post_carrier` |

The workforce coordinator is the HR and safety desk the shipped templates lacked. It holds 55
of its 56 tools: every workforce write the compliance assistant does not, with `worker_search`,
`worker_list`, `worker_get` and `shipment_list` to find the worker or load. It runs at a
Propose ceiling, so every write waits for a person; `worker_adjust_pto_balance`,
`worker_finalize_dot_random_draw` and `worker_close_leave_case` still run only from that person's approval.
Its data access is Restricted, so it reads the Restricted detail on injury, leave and time-off
records, never past the role of the person talking to it, and Confidential fields never reach
it. Its prompt names the compliance assistant as
the holder of credential, training, test and safety event recording, and payroll and the
settlements clerk as the holders of pay.

The settlements clerk approves and posts settlements through `settlement_approve_driver`,
`settlement_post_driver`, `settlement_approve_carrier` and `settlement_post_carrier`, one
proposal for every settlement that is ready, from one to 50. Each keeps one settlement's
policy (person-only, Propose at most, money, and for a driver settlement driver-visible,
classified as money per call), runs each settlement exactly as one settlement's approval or
posting would, lists the settlements it would refuse first, and reports "N of M approved" or
"posted" with each refusal; the person approving may untick any of them, and a call naming one
settlement is that settlement's own proposal, pinned to its version; see
[proposal-previews.md](proposal-previews.md#tools-that-take-one-record-or-many).

The nine pay setup tools stay on no template: `settlement_assign_pay_profile`,
`settlement_end_pay_assignment`, `settlement_create_recurring_deduction`, `settlement_update_recurring_deduction`,
`settlement_create_recurring_earning`, `settlement_update_recurring_earning`, `settlement_open_escrow_account`,
`settlement_update_escrow_account` and `settlement_close_escrow_account`. A pay rate, a standing deduction or
earning and escrow terms are set apart from the settlements that apply them, so the one who
processes pay is not the one who sets it: the settlements clerk reads a driver's pay profile,
recurring pay and escrow accounts and changes none of them, and the workforce coordinator never
touches pay.

`settlement_generate_payroll_export`, `settlement_void_payroll_export` and `settlement_review_driver_expense` stay on no
template. Each moves pay, which the workforce coordinator never touches, and the settlements
clerk, where they would belong, holds all 56 of its tools. An organization adds them to an
agent it builds in AI control, as it adds the pay setup tools. A template test
(`withheldFromEveryTemplate`) keeps all twelve off every template. No template is unattended,
so the agent permission ceiling (`permission/agent.go`) is unchanged.

## Known limits

- An agent made from the Compliance assistant or settlements clerk template before these tools
  existed keeps the tools it was saved with; an administrator adds them in AI control. An
  organization that wants the workforce coordinator creates it from its starter.
- `worker_list_dot_random_draws` returns the 24 newest rounds and the active pools;
  `settlement_list_payroll_exports` the 26 newest runs.
- `worker_assign_training` takes at most 50 workers and 10 courses per proposal.
- `shipment_list_permits` shows a state's id and abbreviation; there is no separate state
  lookup, so a permit is recorded against a state its shipment's requirements name.
