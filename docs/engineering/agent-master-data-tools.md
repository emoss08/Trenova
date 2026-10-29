# Agent tools for master data, documents, alerts, the feed and capture

What an agent can do to carriers, customers, commodities, hazardous materials, locations,
tractors and trailers, to stored documents and their versions, to dismissed insights, change
alerts, the watchtower feed and scanned paperwork in capture intake, how far each step may run
without a person, and which of these writes no agent makes.

Read [agent-runtime.md](agent-runtime.md) for tiers, egress classes and taint, and
[proposal-previews.md](proposal-previews.md) for how a proposal is previewed and approved.
The generated [ai-tool-safety.md](ai-tool-safety.md) is the authority on each policy, and
[agent-write-coverage.md](agent-write-coverage.md) on which write each tool covers; this
page explains them.

## The rule

Every write calls the service its page calls — `carrierservice`, `customerservice`,
`commodityservice`, `hazardousmaterialservice`, `locationservice`, `tractorservice`,
`trailerservice`, `documentservice`, `insightservice`, `tablechangealertservice`,
`watchtowerservice`, `capturefiling` — and never writes around it. Each service exposes the
plan its write runs on, and the write is the plan followed by a save:

| Service | Plans |
| --- | --- |
| `carrierservice`, `customerservice`, `commodityservice`, `hazardousmaterialservice` | `PlanCreate`, `PlanUpdate`, `PlanBulkUpdateStatus` |
| `locationservice` | `PlanUpdate`, `PlanBulkUpdateStatus` |
| `tractorservice` | `PlanCreate`, `PlanUpdate`, `PlanLocate` |
| `trailerservice` | `PlanCreate`, `PlanUpdate`, `PlanLocate` |
| `documentservice` | `PlanDelete`, `PlanRestoreVersion` |
| `insightservice` | `PlanRestore` |
| `tablechangealertservice` | `PlanUpdateSubscription`, `PlanSetSubscriptionStatus`, `PlanDeleteSubscription` |
| `watchtowerservice` | `PlanDismiss` |
| `capturefiling` | `PlanFileItem`, `PlanDiscardItem`, `PlanDiscardBatch` |

A tool's preview renders that plan and its `Validate` runs the same plan, so a proposal shows
what the write decides and is refused for the reasons the write would refuse it. A create or
update plan runs the entity's own transform and validator, so a code, DOT number or UN number
already on file is refused before anything is proposed. Every update sends the version of the
record it read, which the approved preview pins; `recordversionrepository` looks up commodity,
hazardous material, location, tractor, trailer, change alert and watchtower item ids for that.

PATCH and PUT on the same record are one `update_<entity>` tool that changes only the fields
it is given. Bulk status changes are one `update_<entity>_status` tool that takes at most 25
ids and one status (`masterData.status`: Active or Inactive); an id that is not on file
refuses the whole change instead of skipping it, in the tool and on the page. Two-letter state
codes are resolved to the state on file, dates are `YYYY-MM-DD`, and every enum is a named
source beside its domain type (`carrier.status`, `carrier.type`, `carrier.paymentMethod`,
`carrier.taxIdType`, `customer.statusUpdatePreference`, `commodity.freightClass`,
`hazardousMaterial.class`, `hazardousMaterial.packingGroup`, `tableChangeAlert.status`,
`capture.targetType`). The shared field kit is `agenttools/masterdatakit`; the tools that use it are in `partytools`
(carriers, customers, capacity postings), `commoditytools`, `equipmenttools` and `locationtools`, and
`statuschange.Plan` is the bulk status plan every status service uses.

A master record read from a document or a message someone outside sent is proposed, whatever
the tier, because colleagues book and pay against it.

## Carriers, customers, commodities and hazardous materials

```
carrier_list ──► carrier_update | carrier_set_status        carrier_create
customer_list ──► customer_update | customer_set_status     customer_create
masterdata_list_commodities ──► masterdata_update_commodity | masterdata_set_commodity_status masterdata_create_commodity
masterdata_list_hazardous_materials ──► masterdata_update_hazardous_material | masterdata_set_hazardous_material_status
                                                                masterdata_create_hazardous_material
```

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `carrier_create`, `carrier_update` | carrier / create, update | inside; money when payment or remit-to is set | Propose → Ask first (money or outside content: Propose) | only a money call |
| `carrier_set_status` | carrier / update | inside | Propose → Ask first | no |
| `customer_create`, `customer_update` | customer / create, update | inside; sent outside when status email recipients are set | Propose → Ask first (recipients or outside content: Propose) | no |
| `customer_set_status` | customer / update | inside | Propose → Ask first | no |
| `masterdata_create_commodity`, `masterdata_update_commodity` | commodity / create, update | inside | Propose → Ask first (from outside content: Propose) | no |
| `masterdata_set_commodity_status` | commodity / update | inside | Propose → Ask first | no |
| `masterdata_create_hazardous_material`, `masterdata_update_hazardous_material` | hazardous material / create, update | inside | Propose → Propose | no |
| `masterdata_set_hazardous_material_status` | hazardous material / update | inside | Propose → Ask first | no |

A carrier's payment method, terms and remit-to details decide where money goes, so a call that
sets any of them is money-class, stops at Propose and runs only from a person's approval
(`ToolExecuteParams.ApprovedFromProposal()`); a call that touches only its name or contact
fields stays inside. `carrier_update` loads the carrier with its contacts, insurance and EDI
channels and keeps them. A customer's status update preference and recipients decide who later
emails go to, so a call that sets them is sent-outside and is held once the run read outside
content; the customer's billing and email profiles are not edited here. A hazardous material's
class, packing group, placarding and emergency contact decide what a driver carries and posts,
so a person checks every one against the DOT table before it is saved.

## Carrier intelligence events

```
carrier_list_intelligence_events ──► carrier_get_intelligence_event ──► carrier_acknowledge_intelligence_event
                                                      ──► carrier_resolve_intelligence_event
```

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `carrier_list_intelligence_events` | carrier intelligence / read | reads only | automatic | — |

`carrier_list_intelligence_events` reads the events monitoring raised through
`carrierintelservice.ListEvents`, the same call the carrier intelligence page's event list
makes, behind the `carrierintelports.CarrierIntelEventReader` port. It filters on what that
list filters on — status (or `openOnly` for Open and Acknowledged), severity, category such as
Authority or Insurance, and a carrier — plus the days an event was detected (`fromDate`,
`toDate`, read in the organization's timezone, the last day included) and free text over the
carrier's name, DOT number and the summary. It returns 25 events newest first unless asked, at
most 50, and a `nextCursor` while more are left. Each row is the carrier, DOT number, category,
severity, what the rule did, the summary, when it was detected and whether it was acknowledged
or resolved; the prior and current values stay on `carrier_get_intelligence_event`, which every row's
id opens, and the same id is what `carrier_acknowledge_intelligence_event` and
`carrier_resolve_intelligence_event` take. Before it existed the get tool's id could only come from the
page or the run's subject, so "which carriers had insurance problems this week" had no answer.

## Locations, tractors and trailers

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `equipment_list_manufacturers` | equipment manufacturer / read | reads only | automatic | — |
| `location_update` | location / update | inside | Propose → Ask first (from outside content: Propose) | no |
| `location_set_status` | location / update | inside | Propose → Ask first | no |
| `equipment_create_tractor`, `equipment_update_tractor` | tractor / create, update | inside | Propose → Ask first (from outside content: Propose) | no |
| `equipment_create_trailer`, `equipment_update_trailer` | trailer / create, update | inside | Propose → Ask first (from outside content: Propose) | no |
| `equipment_set_tractor_location` | tractor / update | inside | Propose → Ask first | no |
| `equipment_set_trailer_location` | trailer / update | money | Propose → Propose | yes |

Creating a location was already `location_create`. It places the address on the map through
the organization's address lookup; without one, or when nothing is found at that state and ZIP,
the location is saved without coordinates and the card says so. `equipment_set_tractor_location` moves where Trenova thinks
a tractor is and locating it again moves it back. `equipment_set_trailer_location` adds an empty move to the
trailer's last shipment and re-rates it, which can change what that customer is charged, so
the preview shows the shipment's total before and after and it runs only on a person's
approval. Equipment types, manufacturers and fleet codes are named by id from their list tools
and stay configuration.

## Documents, insights, change alerts and the watchtower feed

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `report_list_table_change_alerts` | table change alert / read | reads only | automatic | — |
| `document_delete` | document / delete | inside | Propose → Propose | no |
| `document_restore_version` | document / update | inside | Propose → Ask first | no |
| `insight_restore` | insight / update | inside | Ask first → Automatic | no |
| `report_update_table_change_alert` | table change alert / update | inside | Propose → Automatic | no |
| `report_set_table_change_alert_status` | table change alert / update | inside | Ask first → Automatic | no |
| `report_delete_table_change_alert` | table change alert / delete | inside | Propose → Ask first | no |
| `exception_dismiss_watchtower_item` | watchtower / update | inside | Propose → Ask first | no |

`document_delete` takes at most 25 ids; one goes through the page's single delete and more
through its bulk delete, and every stored version of each goes with it, so it is always
proposed. `document_restore_version` takes any version of the document and the version number
to make current; the newer version is kept, so a later restore brings it back.
`insight_restore` refuses an insight that is not dismissed and one the caller cannot see.

A change alert belongs to the person who set it up, and its notices go only to them: the
service now scopes an update to its owner in the repository as well as in the plan, and runs
the table allowlist again on every change. Dismissing a watchtower item takes it off the shared
attention feed and leaves the record behind it alone; an item that oversees agents' own work,
and one already resolved, is refused.

## Capture intake

```
inbox_list_capture_batches ──► inbox_file_capture_item
                     ──► inbox_discard_capture_item | inbox_discard_capture_batch
```

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `inbox_list_capture_batches` | capture batch / read | reads only | automatic | — |
| `inbox_file_capture_item` | capture batch / update | inside | Propose → Ask first | no |
| `inbox_discard_capture_item` | capture batch / delete | inside | Propose → Ask first | no |
| `inbox_discard_capture_batch` | capture batch / delete | inside | Propose → Propose | no |

`inbox_list_capture_batches` shows the open stacks (ready, partly filed or failed) and the documents
the split proposed in each. `inbox_file_capture_item` files at most 25 of them, each onto a record
by `capture.targetType` and id as a document type, as the person it acts for; one goes through
the page's single filing and more through its batch filing, and any that fail are named in the
error. A misfiled document can be deleted. Discarding a batch drops every unfiled page with
nothing to bring them back, so it is always proposed.

## What no agent does

| Write | Category | Why |
| --- | --- | --- |
| Start a scan or catch a print on a paired computer | security | It drives one of the person's own computers under the access they granted that machine, and a scan needs the person at the scanner. |
| Cancel a scan or print request | security | It withdraws an instruction the person gave their own paired computer. |
| Print capture cover sheets | infrastructure | The routing codes exist only to be printed and laid on paper; `inbox_file_capture_item` files what they route. |
| Split, merge, reorder or rotate a scanned stack's pages | user-preference | The person arranges pages looking at their images; pages carry no text an agent can read. |
| Re-extract a shipment draft from a document | infrastructure | A repair that re-runs the extraction pipeline and replaces the draft being reviewed. |
| Hand off a watchtower item | agent-administration | Agents hand work to one another through `delegate_task`. |

## Who holds them

| Template | Runs | Holds |
| --- | --- | --- |
| Master data steward | chat, at a Propose ceiling, Internal data access | `carrier_create`, `carrier_update`, `carrier_set_status`, `customer_create`, `customer_update`, `customer_set_status`, `masterdata_create_commodity`, `masterdata_update_commodity`, `masterdata_set_commodity_status`, `masterdata_create_hazardous_material`, `masterdata_update_hazardous_material`, `masterdata_set_hazardous_material_status`, `location_create`, `location_update`, `location_set_status`, `equipment_create_tractor`, `equipment_update_tractor`, `equipment_set_tractor_status`, `equipment_set_tractor_location`, `equipment_create_trailer`, `equipment_update_trailer`, `equipment_set_trailer_status`, `equipment_set_trailer_location`, `equipment_list_manufacturers`, `document_delete`, `document_restore_version`, `report_list_table_change_alerts`, `report_create_table_change_alert`, `report_update_table_change_alert`, `report_set_table_change_alert_status`, `report_delete_table_change_alert`, `exception_list_watchtower_items`, `exception_dismiss_watchtower_item`, `inbox_list_capture_batches`, `inbox_file_capture_item`, `inbox_discard_capture_item`, `inbox_discard_capture_batch`, and the list and get tools each needs |
| Customer assistant | chat, at a Propose ceiling | `customer_create`, `customer_update`, `customer_set_status`, `location_list`, `location_update`, `location_set_status`, `masterdata_list_commodities`, `masterdata_create_commodity`, `masterdata_update_commodity` |
| Insight analyst | on each new insight, at a Propose ceiling | `insight_restore` |
| Carrier risk desk | on each new carrier finding, at a Propose ceiling | `carrier_list_intelligence_events`, `carrier_get_intelligence_event`, `carrier_acknowledge_intelligence_event`, `carrier_resolve_intelligence_event` |
| Compliance assistant | chat, Ask first | `carrier_list_intelligence_events`, beside its carrier vetting and monitoring tools |
| Any agent holding `shipment_create` | as the agent runs | `customer_create`, `location_create`, `masterdata_create_commodity`, as `shipment_create`'s companions, unless switched off for the agent |

The master data steward is the organization's records keeper and holds 55 of its 56 tools:
every write on this page, the reads that find each record, `document_list` and
`document_get_summary` for the documents it deletes or restores, `document_list_types` for
filing, and `shipment_list` for paperwork that belongs on a load. The customer assistant
handles customer records and holds 53. The insight analyst dismisses insights it finds are
noise and can bring one back. All three run at a Propose ceiling, so every write they make
waits for a person and needs no new permission ceiling entry; `equipment_set_trailer_location`, and a carrier
change that sets payment or remit-to, still run only from that person's approval.

The compliance assistant was at its 56-tool limit, so it gave up `report_preview` (the report
analyst's; it still runs saved and built-in reports with `report_run`) to hold
`carrier_list_intelligence_events`.

No write on this page is left without a holder. The carrier risk desk still only says whether a
carrier can be tendered and leaves the decision to stop using one to a person, and capture
filing stays a proposal the person whose paperwork it is approves. An organization can still
add any of these tools to an agent it builds in AI control.

## Known limits

- An agent made from the customer assistant, insight analyst, carrier risk desk or compliance
  assistant template before these tools existed keeps the tools it was saved with; an
  administrator adds them in AI control. A compliance agent saved earlier also keeps
  `report_preview`.
- `carrier_list_intelligence_events` shows at most 50 events a page; it filters on the day an event
  was detected, not on when the provider says the change happened. An
  organization that wants the master data steward creates it from its starter.
- A bulk status change, a document delete and a capture filing each take at most 25 records;
  `inbox_list_capture_batches` shows at most 10 open stacks.
- An agent cannot edit a customer's billing or email profile, a carrier's contacts, insurance
  or EDI channels, or a location's geofence; a person does that on the page.
- `document_restore_version` names the version by number, so a document with more than 10,000
  versions cannot be restored past that by an agent.
