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
leave it (`services.RecordChange[T]`). A tool's preview renders that plan and its `Validate`
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
| `list_worker_safety_events` | worker safety event / read | reads only | automatic | — |
| `open_worker_safety_event` | worker safety event / create | inside | Propose → Ask first | no |
| `update_worker_safety_event` | worker safety event / update | inside | Propose → Ask first | no |
| `change_worker_safety_event_status` | worker safety event / close | inside | Propose → Ask first | no |
| `delete_worker_safety_event` | worker safety event / delete | inside | Propose → Propose | no |
| `record_safety_violation`, `update_safety_violation` | worker safety event / create, update | inside | Propose → Ask first | no |
| `delete_safety_violation` | worker safety event / delete | inside | Propose → Propose | no |
| `give_worker_recognition` | worker recognition / create | driver | Propose → Ask first | no |
| `delete_worker_recognition` | worker recognition / delete | inside | Propose → Propose | no |

An event's points default from the house scale for its kind, severity, preventability and
inspection result; closing needs a resolution. Issuing and rescinding discipline are
attestation: the manager decides and the driver acknowledges it in the portal.

## Drug and alcohol testing

| Tool | Resource / operation | Class | Default → most | Person's approval |
| --- | --- | --- | --- | --- |
| `list_dot_tests`, `list_dot_random_draws`, `get_dot_random_draw` | DOT test, random pool / read | reads only | automatic | — |
| `schedule_dot_test` | worker DOT test / create | inside | Propose → Ask first | no |
| `cancel_dot_test` | worker DOT test / cancel | inside | Propose → Propose | no |
| `run_dot_random_draw` | DOT random pool / manage | inside | Propose → Ask first | no |
| `update_dot_random_selection` | DOT random pool / manage | inside | Propose → Ask first | no |
| `cancel_dot_random_draw` | DOT random pool / manage | inside | Propose → Propose | no |
| `finalize_dot_random_draw` | DOT random pool / manage | inside | Propose → Propose | yes |

`schedule_dot_test` records an order and is forced to `Scheduled` with a pending result.
`run_dot_random_draw` previews the pool size and each substance's target and names nobody: the
names are chosen only when the round is drawn, under a seed nobody sees first, so a proposal
cannot be used to shop for a different selection. A draft round stands only once a person
finalizes it. Results, violations, Clearinghouse queries and return-to-duty steps reach FMCSA
and are attestation; pools are configuration.

## Leave

| Tool | Resource / operation | Class | Default → most | Person's approval |
| --- | --- | --- | --- | --- |
| `list_worker_leave_cases` | worker leave / read | reads only | automatic | — |
| `open_leave_case`, `update_leave_case` | worker leave / create, update | inside | Propose → Ask first | no |
| `request_leave_certification` | worker leave / update | inside | Propose → Ask first | no |
| `record_leave_day`, `update_leave_day` | worker leave / create, update | inside | Propose → Ask first | no |
| `delete_leave_day` | worker leave / update | inside | Propose → Propose | no |
| `close_leave_case` | worker leave / approve | inside | Propose → Propose | yes |

Deciding a case, and designating it FMLA, is the employer's notice under 29 CFR 825.300, and
recording the provider's certification is the person who received it: both attestation.

## Training, checklists and reviews

| Tool | Resource / operation | Class | Default → most | Person's approval |
| --- | --- | --- | --- | --- |
| `list_worker_training`, `list_training_courses` | worker training, training course / read | reads only | automatic | — |
| `assign_worker_training` | worker training / assign | driver | Propose → Ask first | no |
| `assign_required_worker_training` | worker training / assign | driver | Propose → Ask first | no |
| `record_training_completion`, `attach_worker_training_document` | worker training / update | inside | Propose → Ask first | no |
| `close_worker_training` | worker training / cancel | inside | Propose → Propose | no |
| `list_worker_checklists` | worker checklist / read | reads only | automatic | — |
| `start_worker_checklist`, `update_worker_checklist_item` | worker checklist / create, update | inside | Propose → Ask first | no |
| `cancel_worker_checklist` | worker checklist / cancel | inside | Propose → Propose | no |
| `list_performance_reviews` | performance review / read | reads only | automatic | — |
| `start_performance_review`, `draft_performance_review` | performance review / create, update | inside | Propose → Ask first | no |
| `delete_performance_review` | performance review / delete | inside | Propose → Propose | no |

`assign_worker_training` takes up to 50 workers and 10 courses; one pair goes through
`Assign`, more through `BulkAssign`, and a run that would assign nothing is refused.
`update_worker_checklist_item` completes, skips, marks not applicable or reopens one item.
`draft_performance_review` merges scores by rating key, refuses a key the review does not
have, and replaces the goals only when it is given them; submitting, reopening and closing a
review are the reviewer's sign-off.

## Qualification file, credentials and injuries

| Tool | Resource / operation | Class | Default → most | Person's approval |
| --- | --- | --- | --- | --- |
| `list_employment_verifications` | qualification / read | reads only | automatic | — |
| `record_employment_verification`, `update_employment_verification` | qualification / create, update | inside | Propose → Ask first | no |
| `log_employment_verification_request` | qualification / update | inside | Propose → Ask first | no |
| `delete_employment_verification` | qualification / delete | inside | Propose → Propose | no |
| `list_worker_credentials` | worker credential / read | reads only | automatic | — |
| `record_worker_credential`, `update_worker_credential` | worker credential / create, update | inside | Propose → Ask first | no |
| `attach_worker_credential_document` | worker credential / update | inside | Propose → Ask first | no |
| `archive_worker_credential` | worker credential / archive | inside | Propose → Propose | no |
| `list_worker_injuries` | worker injury / read | reads only | automatic | — |
| `record_worker_injury`, `update_worker_injury` | worker injury / create, update | inside | Propose → Ask first | no |
| `delete_worker_injury` | worker injury / delete | inside | Propose → Propose | no |

A second active credential of a type is refused unless the call renews it, and the preview
shows the one a renewal archives. Changing a fact a person verified clears the verification,
and the preview says so. An injury takes the next case number for its year; its
classification follows the treatment unless a person gave one.

## Time off, permits, driver pay and payroll

| Tool | Resource / operation | Class | Default → most | Person's approval |
| --- | --- | --- | --- | --- |
| `request_worker_pto`, `update_worker_pto` | worker PTO / create, update | driver | Propose → Ask first | no |
| `adjust_worker_pto_balance` | worker PTO / manage | money | Propose → Propose | yes |
| `list_shipment_permits` | permit / read | reads only | automatic | — |
| `record_shipment_permit`, `update_shipment_permit` | permit / create, update | inside | Propose → Ask first | no |
| `start_settlement_dispute_review` | settlement dispute / update | driver | Propose → Ask first | no |
| `resolve_settlement_dispute` | settlement dispute / approve | driver or money, per call | Propose → Propose | yes |
| `list_driver_expenses` | driver expense / read | reads only, marked | automatic | — |
| `review_driver_expense` | driver expense / approve | driver or money, per call | Propose → Propose | yes |
| `list_payroll_exports` | timesheet / read | reads only | automatic | — |
| `generate_payroll_export`, `void_payroll_export` | timesheet / export | money | Propose → Propose | yes |

A time-off request under a policy that needs no approval is booked and the driver told at
once, which is why it is driver-visible; the preview says when that will happen. A dispute
decision with an adjustment, and an approved expense, classify as money; a denial or a
rejection is driver-visible. `list_driver_expenses` marks a row whose description the driver
wrote, so a turn that read it proposes every later write. Waiving a permit requirement is a
named person accepting the compliance risk and is attestation. Time clock entries are the wage
record and are attestation; bulk PTO decisions duplicate the single-request tools.

## Who holds them

| Template | Holds |
| --- | --- |
| Workforce coordinator (chat, Propose, Restricted data access) | `list_time_off`, `request_worker_pto`, `update_worker_pto`, `approve_worker_pto`, `reject_worker_pto`, `cancel_worker_pto`, `adjust_worker_pto_balance`, `list_worker_leave_cases`, `open_leave_case`, `update_leave_case`, `request_leave_certification`, `record_leave_day`, `update_leave_day`, `delete_leave_day`, `close_leave_case`, `list_worker_injuries`, `record_worker_injury`, `update_worker_injury`, `delete_worker_injury`, `list_performance_reviews`, `start_performance_review`, `draft_performance_review`, `delete_performance_review`, `give_worker_recognition`, `delete_worker_recognition`, `list_worker_safety_events`, `update_worker_safety_event`, `delete_worker_safety_event`, `update_safety_violation`, `delete_safety_violation`, `list_dot_tests`, `cancel_dot_test`, `list_dot_random_draws`, `get_dot_random_draw`, `run_dot_random_draw`, `finalize_dot_random_draw`, `cancel_dot_random_draw`, `list_worker_checklists`, `start_worker_checklist`, `update_worker_checklist_item`, `cancel_worker_checklist`, `list_worker_training`, `attach_worker_training_document`, `close_worker_training`, `list_worker_credentials`, `archive_worker_credential`, `list_employment_verifications`, `delete_employment_verification`, `list_shipment_permits`, `record_shipment_permit`, `update_shipment_permit` |
| Compliance assistant (chat, Ask first) | `list_worker_credentials`, `record_worker_credential`, `update_worker_credential`, `attach_worker_credential_document`, `list_worker_training`, `list_training_courses`, `assign_worker_training`, `assign_required_worker_training`, `record_training_completion`, `list_employment_verifications`, `record_employment_verification`, `update_employment_verification`, `log_employment_verification_request`, `list_worker_checklists`, `update_worker_checklist_item`, `list_worker_safety_events`, `open_worker_safety_event`, `record_safety_violation`, `change_worker_safety_event_status`, `list_dot_tests`, `get_dot_random_draw`, `schedule_dot_test`, `update_dot_random_selection`, `list_worker_leave_cases` |
| Settlements clerk (chat, Ask first) | `start_settlement_dispute_review`, `resolve_settlement_dispute` |

The workforce coordinator is the HR and safety desk the shipped templates lacked. It holds 55
of its 56 tools: every workforce write the compliance assistant does not, with `search_worker`,
`list_workers`, `get_worker` and `search_shipments` to find the worker or load. It runs at a
Propose ceiling, so every write waits for a person; `adjust_worker_pto_balance`,
`finalize_dot_random_draw` and `close_leave_case` still run only from that person's approval.
Its data access is Restricted, so it reads the Restricted detail on injury, leave and time-off
records, never past the role of the person talking to it, and Confidential fields never reach
it. Its prompt names the compliance assistant as
the holder of credential, training, test and safety event recording, and payroll and the
settlements clerk as the holders of pay.

`generate_payroll_export`, `void_payroll_export` and `review_driver_expense` stay on no
template. Each moves pay, which the workforce coordinator never touches, and the settlements
clerk, where they would belong, holds all 56 of its tools. An organization adds them to an
agent it builds in AI control. No template is unattended, so the agent permission ceiling
(`permission/agent.go`) is unchanged.

## Known limits

- An agent made from the Compliance assistant or settlements clerk template before these tools
  existed keeps the tools it was saved with; an administrator adds them in AI control. An
  organization that wants the workforce coordinator creates it from its starter.
- `list_dot_random_draws` returns the 24 newest rounds and the active pools;
  `list_payroll_exports` the 26 newest runs.
- `assign_worker_training` takes at most 50 workers and 10 courses per proposal.
- `list_shipment_permits` shows a state's id and abbreviation; there is no separate state
  lookup, so a permit is recorded against a state its shipment's requirements name.
