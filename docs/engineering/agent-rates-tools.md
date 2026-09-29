# Agent tools for fuel, IFTA, rates and reports

What an agent can do to fuel purchases and fuel cards, IFTA returns and jurisdiction miles,
rate agreements, rate sheets, fuel index prices and report administration, how far each step
may run without a person, and which of these writes no agent makes.

Read [agent-runtime.md](agent-runtime.md) for tiers, egress classes and taint, and
[proposal-previews.md](proposal-previews.md) for how a proposal is previewed and approved.
The generated [ai-tool-safety.md](ai-tool-safety.md) is the authority on each policy, and
[agent-write-coverage.md](agent-write-coverage.md) on which write each tool covers; this
page explains them.

## The rule

Every write calls the service its page calls — `fuelpurchaseservice`,
`fuelsurchargeservice`, `iftaservice`, `distancecalculationservice`,
`rateagreementservice`, `rateimportservice`, `ratesimulationservice`, `reporting` — and
never writes around it. Each service exposes the plan its write runs on, and the write is the
plan followed by a save:

| Service | Plans |
| --- | --- |
| `fuelpurchaseservice` | `PlanCreatePurchase`, `PlanUpdatePurchase`, `PlanDeletePurchase`, `PlanAssignCard`, `PlanCommit`, `PlanResolveRows`, `PlanDiscard` |
| `fuelsurchargeservice` | `PlanAddManualPrice`, `PlanUpdateManualPrice` |
| `iftaservice` | `PlanGenerate`, `PlanRecompute`, `PlanAmend`, `PlanDelete`, `PlanCreateMileageEntry`, `PlanUpdateMileageEntry`, `PlanDeleteMileageEntry` |
| `distancecalculationservice` | `PlanMoveJurisdictionMiles`; the backfill previews with its own dry run |
| `rateagreementservice` | `PlanCreate`, `PlanUpdate`, `PlanDuplicate`, `PlanReview`, `PlanAmendRules`, `PlanApplyRateIncrease` |
| `rateimportservice` | `PlanCommit`, `PlanDiscard` |
| `ratesimulationservice` | `PlanCreate` |
| `reporting` | `PlanCancelRun`, `PlanDeleteDefinition`, `PlanResetCannedFork`, `PlanDeleteDashboard`, `PlanUpdateSchedule`, `PlanDeleteSchedule` |

A tool's preview renders that plan and its `Validate` runs the same plan, so a proposal shows
what the write decides and is refused for the reasons the write would refuse it. Rate
agreement review moves go through `pkg/approvalworkflow`, whose `Plan` stamps the transition
on a copy and whose `Apply` stamps the loaded entity, so a preview and the save take the same
transition. Every update and delete sends the version of the record it read, which the
approved preview pins; `recordversionrepository` looks each of these records up by its id
prefix, so a fuel card, an import batch and a purchase sharing one permission resource are
told apart.

Anything that changes what a customer is charged or a carrier is paid is money-class, stops
at Propose, and runs only from a person's approval (`ToolExecuteParams.ApprovedFromProposal()`),
as that person. Dates are `YYYY-MM-DD`, an instant is RFC 3339 with its offset, and amounts,
quantities and rates are decimal strings. Every enum a tool offers is a named source beside
its domain type (`ifta.fuelType`, `fuelPurchase.quantityUnit`, `rateAgreement.partyType`,
`rateAgreement.agreementType`, `rateAgreement.direction`, `rateAgreement.laneScopeType`,
`rateSimulation.partyType`).

## Fuel purchases and fuel cards

```
list_fuel_cards ──► assign_fuel_card ──► resolve_fuel_purchase_import_rows
list_fuel_purchase_imports ──► commit_fuel_purchase_import | discard_fuel_purchase_import
list_ifta_jurisdictions ──► record_fuel_purchase
list_fuel_purchases ──► correct_fuel_purchase | delete_fuel_purchase
```

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `list_fuel_purchases`, `list_fuel_cards`, `list_fuel_purchase_imports`, `list_ifta_jurisdictions` | fuel purchase, fuel card, fuel purchase import / read | reads only | automatic | — |
| `record_fuel_purchase` | fuel purchase / create | inside | Propose → Ask first (from outside content: Propose) | no |
| `correct_fuel_purchase` | fuel purchase / update | inside | Propose → Ask first (from outside content: Propose) | no |
| `delete_fuel_purchase` | fuel purchase / delete | inside | Propose → Propose | no |
| `assign_fuel_card` | fuel card / update | inside | Propose → Ask first | no |
| `commit_fuel_purchase_import` | fuel purchase / import | inside | Propose → Ask first | no |
| `resolve_fuel_purchase_import_rows` | fuel purchase import / import | inside | Propose → Ask first | no |
| `discard_fuel_purchase_import` | fuel purchase / cancel | inside | Propose → Ask first | no |

A purchase is a tax record the quarter's IFTA return is computed from, so one read from a
receipt or statement someone outside sent is proposed. The preview shows the gallons the
quantity converts to, the jurisdiction by code and name, and the total in its currency; a
transaction reference already on file is refused before anything is proposed. A correction
changes only the fields it is given. Deleting stops at Propose because nothing brings a
purchase back but entering it again. Creating, editing and cancelling a fuel card is the
card provider's instrument and configuration; uploading and staging a statement moves file
bytes and is infrastructure, as is the feed sync, which runs on its schedule.

## IFTA returns and jurisdiction miles

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `list_ifta_returns`, `list_ifta_mileage_entries` | IFTA return, jurisdiction mileage / read | reads only | automatic | — |
| `generate_ifta_return` | IFTA return / create | inside | Propose → Ask first | no |
| `recompute_ifta_return` | IFTA return / update | inside | Ask first → Automatic | no |
| `amend_ifta_return` | IFTA return / create | inside | Propose → Propose | no |
| `delete_ifta_return` | IFTA return / delete | inside | Propose → Ask first | no |
| `record_ifta_mileage_entry` | jurisdiction mileage / create | inside | Propose → Ask first (from outside content: Propose) | no |
| `correct_ifta_mileage_entry` | jurisdiction mileage / update | inside | Propose → Ask first (from outside content: Propose) | no |
| `delete_ifta_mileage_entry` | jurisdiction mileage / delete | inside | Propose → Propose | no |
| `recalculate_move_jurisdiction_miles` | shipment move / update | inside | Propose → Ask first | no |
| `backfill_jurisdiction_miles` | IFTA return / manage | inside | Propose → Ask first | no |

Generating and recomputing compute a draft from what is on file and file nothing; the
preview gives the miles, gallons, net due and how many problems block finalizing. Recompute
only ever re-derives a draft, and running it twice gives the same figures, so it may run on
its own. Amending opens a new draft for a filed quarter with the auditor's reason and is
always a person's call. Finalizing, marking filed and reopening a return are the filer's
sign-off and are attestation; the jurisdictions' tax rates are configuration.
`backfill_jurisdiction_miles` takes the IFTA quarter, previews with the service's dry run
(how many completed moves have no jurisdiction miles and how many miles they carry) and
starts the background job only on execute; each move is a billable distance request.

## Rate agreements

```
list_customers / list_carriers / list_locations
        │
        ▼
draft_rate_agreement ─┐
duplicate_rate_agreement ─┴─► revise_rate_agreement_draft ─► run_rate_simulation
        ─► submit_rate_agreement ─► (reviewer) approve_rate_agreement | reject_rate_agreement
active ─► amend_rate_agreement_rules | apply_rate_increase
       ─► suspend_rate_agreement ─► resume_rate_agreement
       ─► archive_rate_agreement
list_rate_imports ─► commit_rate_import | discard_rate_import
```

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `list_rate_imports` | rate agreement / read | reads only | automatic | — |
| `draft_rate_agreement` | rate agreement / create | inside | Propose → Ask first (from outside content: Propose) | no |
| `revise_rate_agreement_draft` | rate agreement / update | inside | Propose → Ask first (from outside content: Propose) | no |
| `duplicate_rate_agreement` | rate agreement / duplicate | inside | Propose → Ask first (from outside content: Propose) | no |
| `submit_rate_agreement` | rate agreement / submit | inside | Propose → Propose | no |
| `reject_rate_agreement` | rate agreement / reject | inside | Propose → Propose | no |
| `approve_rate_agreement` | rate agreement / approve | money | Propose → Propose | yes; refused to an agent principal |
| `suspend_rate_agreement`, `resume_rate_agreement` | rate agreement / update | money | Propose → Propose | yes |
| `archive_rate_agreement` | rate agreement / archive | money | Propose → Propose | yes |
| `amend_rate_agreement_rules` | rate agreement / update | money | Propose → Propose | yes |
| `apply_rate_increase` | rate agreement / update | money | Propose → Propose | yes |
| `commit_rate_import` | rate agreement / update | money | Propose → Propose | yes |
| `discard_rate_import` | rate agreement / update | inside | Propose → Ask first | no |
| `run_rate_simulation` | rate simulation / create | inside | Ask first → Automatic | no |

A draft prices nothing until it is approved, so drafting, revising and copying are inside
the organization; one drafted from a rate sheet or a message someone outside sent is still
proposed. Lanes are a closed schema: an origin and a destination by scope type (Any, Country,
State, CityState, Zip3, Zip5, Zone, Location — Radius needs coordinates an agent does not
have), a two-letter state code the tool resolves to the state on file, a direction, a formula
template or a rate matrix, and decimal rate, minimum and maximum. `revise_rate_agreement_draft`
revises only a draft and keeps each lane it names by `ruleId`, changing only what it gives.

A negotiated rate is confidential: the rate agreement resource's default sensitivity is
Confidential, so a preview never carries a rate. Amendments, rate sheets and rate increases
show which lanes close out and which are added, by label, and the rate increase names the
percent or flat change it applies; the person approving reads the rates on the agreement.

Approving is the one move `guardPreview` refuses an agent principal outright, and neither
approve nor reject is on any template: the reviewer decides. `run_rate_simulation` replays
past shipments against an agreement in the background and changes no shipment, rate or
invoice; its last day is inclusive. Uploading a rate sheet moves file bytes and is
infrastructure. Rate matrices, rate zones and distance overrides are pricing structure an
administrator builds and are configuration.

## Fuel index prices

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `list_fuel_index_prices` | fuel surcharge program / read | reads only | automatic | — |
| `record_fuel_index_price` | fuel surcharge program / update | money | Propose → Propose | yes |
| `correct_fuel_index_price` | fuel surcharge program / update | money | Propose → Propose | yes |

A manual price on a custom index is what every surcharge following that index prices from,
so it is money-class. EIA prices are fetched on their own and take no manual price. Deleting a
price is curation an administrator does; fuel indexes and surcharge programs stay
configuration.

## Report administration

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `list_report_schedules` | report / read | reads only | automatic | — |
| `cancel_report_run` | report / read | inside | Ask first → Automatic | no |
| `delete_report` | report / delete | inside | Propose → Propose | no |
| `reset_report_fork` | report / update | inside | Propose → Ask first | no |
| `delete_dashboard` | dashboard / delete | inside | Propose → Propose | no |
| `update_report_schedule` | report / export | sent outside | Propose → Ask first | no |
| `delete_report_schedule` | report / export | inside | Propose → Propose | no |

Cancelling a run is gated on read, as the page gates it: the service lets only the person who
started a run cancel it. Only a schedule's owner can change or remove it, and a changed
schedule may email addresses outside the organization, so it is sent-outside and held once
the run read outside content. Deleting a report, a dashboard or a schedule stops at Propose
because nothing brings them back. Saved report views are one person's preference.
`run_report`, `create_report`, `update_report`, `fork_report`, `create_dashboard`,
`add_dashboard_tile` and `schedule_report` already covered the rest.

## Who holds them

| Template | Runs | Holds |
| --- | --- | --- |
| Formula assistant | chat, at a Propose ceiling | `list_customers`, `list_carriers`, `list_locations`, `list_rate_imports`, `list_fuel_index_prices`, `draft_rate_agreement`, `revise_rate_agreement_draft`, `duplicate_rate_agreement`, `submit_rate_agreement`, `run_rate_simulation`, `amend_rate_agreement_rules`, `apply_rate_increase`, `suspend_rate_agreement`, `resume_rate_agreement`, `archive_rate_agreement`, `commit_rate_import`, `discard_rate_import`, `record_fuel_index_price`, `correct_fuel_index_price` |

The formula assistant is the organization's pricing agent and holds 29 of its 56 tools. Its
Propose ceiling means every write it makes waits for a person, so it needs no new permission
ceiling entry.

The fuel purchase, fuel card, fuel import, IFTA, jurisdiction mileage and report
administration tools, and `approve_rate_agreement` and `reject_rate_agreement`, are
registered and on no template. The billing assistant has no room left under the eight-tool
headroom; the compliance assistant, where fuel tax would belong, has none once the
workforce tools join it; and the books keeper is an unattended accounting desk. Approving and rejecting stay off every
template so the agent that drafts an agreement is never the one that reviews it. An
organization adds them to an agent it builds in AI control.

## Known limits

- An agent made from the formula assistant template before these tools existed keeps the
  tools it was saved with; an administrator adds them in AI control.
- A lane scoped by Radius cannot be drafted by an agent; a person draws it on the page.
- `apply_rate_increase` takes at most 50 agreements by id, and a rate matrix's cells are not
  moved by it; the preview counts the matrix-priced lanes it leaves alone.
- `backfill_jurisdiction_miles` routes at most 5,000 moves in one run.
