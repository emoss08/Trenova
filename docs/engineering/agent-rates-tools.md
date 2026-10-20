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
fuel_list_cards ──► fuel_assign_card ──► fuel_resolve_purchase_import_rows
fuel_list_purchase_imports ──► fuel_commit_purchase_import | fuel_discard_purchase_import
fuel_list_ifta_jurisdictions ──► fuel_record_purchase
fuel_list_purchases ──► fuel_correct_purchase | fuel_delete_purchase
```

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `fuel_list_purchases`, `fuel_list_cards`, `fuel_list_purchase_imports`, `fuel_list_ifta_jurisdictions` | fuel purchase, fuel card, fuel purchase import / read | reads only | automatic | — |
| `fuel_record_purchase` | fuel purchase / create | inside | Propose → Ask first (from outside content: Propose) | no |
| `fuel_correct_purchase` | fuel purchase / update | inside | Propose → Ask first (from outside content: Propose) | no |
| `fuel_delete_purchase` | fuel purchase / delete | inside | Propose → Propose | no |
| `fuel_assign_card` | fuel card / update | inside | Propose → Ask first | no |
| `fuel_commit_purchase_import` | fuel purchase / import | inside | Propose → Ask first | no |
| `fuel_resolve_purchase_import_rows` | fuel purchase import / import | inside | Propose → Ask first | no |
| `fuel_discard_purchase_import` | fuel purchase / cancel | inside | Propose → Ask first | no |

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
| `fuel_list_ifta_returns`, `fuel_list_ifta_mileage_entries` | IFTA return, jurisdiction mileage / read | reads only | automatic | — |
| `fuel_generate_ifta_return` | IFTA return / create | inside | Propose → Ask first | no |
| `fuel_recalculate_ifta_return` | IFTA return / update | inside | Ask first → Automatic | no |
| `fuel_amend_ifta_return` | IFTA return / create | inside | Propose → Propose | no |
| `fuel_delete_ifta_return` | IFTA return / delete | inside | Propose → Ask first | no |
| `fuel_record_ifta_mileage_entry` | jurisdiction mileage / create | inside | Propose → Ask first (from outside content: Propose) | no |
| `fuel_correct_ifta_mileage_entry` | jurisdiction mileage / update | inside | Propose → Ask first (from outside content: Propose) | no |
| `fuel_delete_ifta_mileage_entry` | jurisdiction mileage / delete | inside | Propose → Propose | no |
| `fuel_recalculate_jurisdiction_miles` | shipment move / update | inside | Propose → Ask first | no |
| `fuel_backfill_jurisdiction_miles` | IFTA return / manage | inside | Propose → Ask first | no |

Generating and recomputing compute a draft from what is on file and file nothing; the
preview gives the miles, gallons, net due and how many problems block finalizing. Recompute
only ever re-derives a draft, and running it twice gives the same figures, so it may run on
its own. Amending opens a new draft for a filed quarter with the auditor's reason and is
always a person's call. Finalizing, marking filed and reopening a return are the filer's
sign-off and are attestation; the jurisdictions' tax rates are configuration.
`fuel_backfill_jurisdiction_miles` takes the IFTA quarter, previews with the service's dry run
(how many completed moves have no jurisdiction miles and how many miles they carry) and
starts the background job only on execute; each move is a billable distance request.

## Rate agreements

```
customer_list / carrier_list / location_list
        │
        ▼
rate_create_agreement ─┐
rate_duplicate_agreement ─┴─► rate_update_agreement_draft ─► rate_run_simulation
        ─► rate_submit_agreement ─► (reviewer) rate_approve_agreement | rate_reject_agreement
active ─► rate_amend_agreement_rules | rate_apply_increase
       ─► rate_suspend_agreement ─► rate_resume_agreement
       ─► rate_archive_agreement
rate_list_imports ─► rate_commit_import | rate_discard_import
```

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `rate_list_imports` | rate agreement / read | reads only | automatic | — |
| `rate_create_agreement` | rate agreement / create | inside | Propose → Ask first (from outside content: Propose) | no |
| `rate_update_agreement_draft` | rate agreement / update | inside | Propose → Ask first (from outside content: Propose) | no |
| `rate_duplicate_agreement` | rate agreement / duplicate | inside | Propose → Ask first (from outside content: Propose) | no |
| `rate_submit_agreement` | rate agreement / submit | inside | Propose → Propose | no |
| `rate_reject_agreement` | rate agreement / reject | inside | Propose → Propose | no |
| `rate_approve_agreement` | rate agreement / approve | money | Propose → Propose | yes; refused to an agent principal |
| `rate_suspend_agreement`, `rate_resume_agreement` | rate agreement / update | money | Propose → Propose | yes |
| `rate_archive_agreement` | rate agreement / archive | money | Propose → Propose | yes |
| `rate_amend_agreement_rules` | rate agreement / update | money | Propose → Propose | yes |
| `rate_apply_increase` | rate agreement / update | money | Propose → Propose | yes |
| `rate_commit_import` | rate agreement / update | money | Propose → Propose | yes |
| `rate_discard_import` | rate agreement / update | inside | Propose → Ask first | no |
| `rate_run_simulation` | rate simulation / create | inside | Ask first → Automatic | no |

A draft prices nothing until it is approved, so drafting, revising and copying are inside
the organization; one drafted from a rate sheet or a message someone outside sent is still
proposed. Lanes are a closed schema: an origin and a destination by scope type (Any, Country,
State, CityState, Zip3, Zip5, Zone, Location — Radius needs coordinates an agent does not
have), a two-letter state code the tool resolves to the state on file, a direction, a formula
template or a rate matrix, and decimal rate, minimum and maximum. `rate_update_agreement_draft`
revises only a draft and keeps each lane it names by `ruleId`, changing only what it gives.

A negotiated rate is confidential: the rate agreement resource's default sensitivity is
Confidential, so a preview never carries a rate. Amendments, rate sheets and rate increases
show which lanes close out and which are added, by label, and the rate increase names the
percent or flat change it applies; the person approving reads the rates on the agreement.

Approving is the one move `guardPreview` refuses an agent principal outright, and neither
approve nor reject is on any template: the reviewer decides. `rate_run_simulation` replays
past shipments against an agreement in the background and changes no shipment, rate or
invoice; its last day is inclusive. Uploading a rate sheet moves file bytes and is
infrastructure. Rate matrices, rate zones and distance overrides are pricing structure an
administrator builds and are configuration.

## Fuel index prices

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `rate_list_fuel_index_prices` | fuel surcharge program / read | reads only | automatic | — |
| `rate_record_fuel_index_price` | fuel surcharge program / update | money | Propose → Propose | yes |
| `rate_correct_fuel_index_price` | fuel surcharge program / update | money | Propose → Propose | yes |

A manual price on a custom index is what every surcharge following that index prices from,
so it is money-class. EIA prices are fetched on their own and take no manual price. Deleting a
price is curation an administrator does; fuel indexes and surcharge programs stay
configuration.

## Report administration

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `report_list_schedules` | report / read | reads only | automatic | — |
| `report_cancel_run` | report / read | inside | Ask first → Automatic | no |
| `report_delete` | report / delete | inside | Propose → Propose | no |
| `report_reset_fork` | report / update | inside | Propose → Ask first | no |
| `report_delete_dashboard` | dashboard / delete | inside | Propose → Propose | no |
| `report_update_schedule` | report / export | sent outside | Propose → Ask first | no |
| `report_delete_schedule` | report / export | inside | Propose → Propose | no |

Cancelling a run is gated on read, as the page gates it: the service lets only the person who
started a run cancel it. Only a schedule's owner can change or remove it, and a changed
schedule may email addresses outside the organization, so it is sent-outside and held once
the run read outside content. Deleting a report, a dashboard or a schedule stops at Propose
because nothing brings them back. Saved report views are one person's preference.
`report_run`, `report_create`, `report_update`, `report_fork`, `report_create_dashboard`,
`report_add_dashboard_tile` and `report_create_schedule` already covered the rest.

## Who holds them

| Template | Runs | Holds |
| --- | --- | --- |
| Fuel and IFTA clerk | chat, at a Propose ceiling, Internal data access | `fuel_list_purchases`, `fuel_record_purchase`, `fuel_correct_purchase`, `fuel_delete_purchase`, `fuel_list_cards`, `fuel_assign_card`, `fuel_list_purchase_imports`, `fuel_resolve_purchase_import_rows`, `fuel_commit_purchase_import`, `fuel_discard_purchase_import`, `fuel_list_ifta_jurisdictions`, `fuel_list_ifta_mileage_entries`, `fuel_record_ifta_mileage_entry`, `fuel_correct_ifta_mileage_entry`, `fuel_delete_ifta_mileage_entry`, `fuel_recalculate_jurisdiction_miles`, `fuel_backfill_jurisdiction_miles`, `fuel_list_ifta_returns`, `fuel_generate_ifta_return`, `fuel_recalculate_ifta_return`, `fuel_delete_ifta_return`, `fuel_amend_ifta_return`, `rate_list_fuel_index_prices`, `rate_record_fuel_index_price`, `rate_correct_fuel_index_price` |
| Report analyst | chat, at a Propose ceiling, Internal data access | `report_list`, `report_get`, `report_preview`, `report_run`, `report_get_run`, `report_list_runs`, `report_compare_runs`, `report_cancel_run`, `report_list_datasets`, `report_get_dataset`, `report_create`, `report_update`, `report_fork`, `report_reset_fork`, `report_delete`, `report_list_schedules`, `report_create_schedule`, `report_update_schedule`, `report_delete_schedule`, `report_list_dashboards`, `report_create_dashboard`, `report_add_dashboard_tile`, `report_delete_dashboard` |
| Formula assistant | chat, at a Propose ceiling | `customer_list`, `carrier_list`, `location_list`, `rate_list_imports`, `rate_list_fuel_index_prices`, `rate_create_agreement`, `rate_update_agreement_draft`, `rate_duplicate_agreement`, `rate_submit_agreement`, `rate_run_simulation`, `rate_amend_agreement_rules`, `rate_apply_increase`, `rate_suspend_agreement`, `rate_resume_agreement`, `rate_archive_agreement`, `rate_commit_import`, `rate_discard_import`, `rate_record_fuel_index_price`, `rate_correct_fuel_index_price` |

The fuel and IFTA clerk keeps the fuel tax record and holds 33 of its 56 tools: every fuel
purchase, fuel card, fuel import, jurisdiction mileage and IFTA tool on this page, the fuel
index prices it shares with the formula assistant, `rate_list_fuel_surcharge_programs`, and the tractor,
driver, shipment and document reads that find what a purchase or a mile belongs to. Fuel
purchases and returns default to Internal sensitivity, so its data access is Internal. The
formula assistant is the organization's pricing agent and holds 30.

`formula_get_template` reads one formula template whole: its formula, its own variables with
their defaults, its breakdown lines, its minimum, maximum and rounding, and its status and
version, so a formula is read before it is explained or changed. The formula assistant, the
shipment intake desk and the import assistant hold it beside `formula_list_templates`, which
yields its id. A person sees the template as its own card on the Desk.

The report analyst holds every report administration tool on this page and the report and
dashboard builders they sit beside, 23 tools in all. It is a chat agent acting as the person, so
the service still lets only a run's starter cancel it and only a schedule's owner change it, and
`report_create_schedule` and `report_update_schedule` stay sent-outside: held once the turn read
outside content and never past Ask first. Its data access is Internal, so a report column above
it is left out and named. The insight analyst, an unattended desk, keeps its report reads and
holds none of the report writes.

All three run at a Propose ceiling, so every write they make waits for a person, and none needs
a new agent permission ceiling entry, since that ceiling binds only unattended desks; a fuel
index price still runs only from that person's approval.

`rate_approve_agreement` and `rate_reject_agreement` stay off every template so the agent that
drafts an agreement is never the one that reviews it. An organization adds them to an agent it
builds in AI control.

## Known limits

- An agent made from the formula assistant template before these tools existed keeps the
  tools it was saved with; an administrator adds them in AI control. An organization that
  wants the fuel and IFTA clerk or the report analyst creates it from its starter.
- A lane scoped by Radius cannot be drafted by an agent; a person draws it on the page.
- `rate_apply_increase` takes at most 50 agreements by id, and a rate matrix's cells are not
  moved by it; the preview counts the matrix-priced lanes it leaves alone.
- `fuel_backfill_jurisdiction_miles` routes at most 5,000 moves in one run.
