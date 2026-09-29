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
`watchtowerservice`, `captureservice` — and never writes around it. Each service exposes the
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
| `captureservice` | `PlanFileItem`, `PlanDiscardItem`, `PlanDiscardBatch` |

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
`capture.targetType`). The shared field kit is `agenttoolservice/masterdata_kit.go`, and
`statuschange.Plan` is the bulk status plan every status service uses.

A master record read from a document or a message someone outside sent is proposed, whatever
the tier, because colleagues book and pay against it.

## Carriers, customers, commodities and hazardous materials

```
list_carriers ──► update_carrier | update_carrier_status        create_carrier
list_customers ──► update_customer | update_customer_status     create_customer
list_commodities ──► update_commodity | update_commodity_status create_commodity
list_hazardous_materials ──► update_hazardous_material | update_hazardous_material_status
                                                                create_hazardous_material
```

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `create_carrier`, `update_carrier` | carrier / create, update | inside; money when payment or remit-to is set | Propose → Ask first (money or outside content: Propose) | only a money call |
| `update_carrier_status` | carrier / update | inside | Propose → Ask first | no |
| `create_customer`, `update_customer` | customer / create, update | inside; sent outside when status email recipients are set | Propose → Ask first (recipients or outside content: Propose) | no |
| `update_customer_status` | customer / update | inside | Propose → Ask first | no |
| `create_commodity`, `update_commodity` | commodity / create, update | inside | Propose → Ask first (from outside content: Propose) | no |
| `update_commodity_status` | commodity / update | inside | Propose → Ask first | no |
| `create_hazardous_material`, `update_hazardous_material` | hazardous material / create, update | inside | Propose → Propose | no |
| `update_hazardous_material_status` | hazardous material / update | inside | Propose → Ask first | no |

A carrier's payment method, terms and remit-to details decide where money goes, so a call that
sets any of them is money-class, stops at Propose and runs only from a person's approval
(`ToolExecuteParams.ApprovedFromProposal()`); a call that touches only its name or contact
fields stays inside. `update_carrier` loads the carrier with its contacts, insurance and EDI
channels and keeps them. A customer's status update preference and recipients decide who later
emails go to, so a call that sets them is sent-outside and is held once the run read outside
content; the customer's billing and email profiles are not edited here. A hazardous material's
class, packing group, placarding and emergency contact decide what a driver carries and posts,
so a person checks every one against the DOT table before it is saved.

## Locations, tractors and trailers

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `list_equipment_manufacturers` | equipment manufacturer / read | reads only | automatic | — |
| `update_location` | location / update | inside | Propose → Ask first (from outside content: Propose) | no |
| `update_location_status` | location / update | inside | Propose → Ask first | no |
| `create_tractor`, `update_tractor` | tractor / create, update | inside | Propose → Ask first (from outside content: Propose) | no |
| `create_trailer`, `update_trailer` | trailer / create, update | inside | Propose → Ask first (from outside content: Propose) | no |
| `locate_tractor` | tractor / update | inside | Propose → Ask first | no |
| `locate_trailer` | trailer / update | money | Propose → Propose | yes |

Creating a location was already `create_location`. `locate_tractor` moves where Trenova thinks
a tractor is and locating it again moves it back. `locate_trailer` adds an empty move to the
trailer's last shipment and re-rates it, which can change what that customer is charged, so
the preview shows the shipment's total before and after and it runs only on a person's
approval. Equipment types, manufacturers and fleet codes are named by id from their list tools
and stay configuration.

## Documents, insights, change alerts and the watchtower feed

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `list_table_change_alerts` | table change alert / read | reads only | automatic | — |
| `delete_documents` | document / delete | inside | Propose → Propose | no |
| `restore_document_version` | document / update | inside | Propose → Ask first | no |
| `restore_insight` | insight / update | inside | Ask first → Automatic | no |
| `update_table_change_alert` | table change alert / update | inside | Propose → Automatic | no |
| `set_table_change_alert_status` | table change alert / update | inside | Ask first → Automatic | no |
| `delete_table_change_alert` | table change alert / delete | inside | Propose → Ask first | no |
| `dismiss_watchtower_item` | watchtower / update | inside | Propose → Ask first | no |

`delete_documents` takes at most 25 ids; one goes through the page's single delete and more
through its bulk delete, and every stored version of each goes with it, so it is always
proposed. `restore_document_version` takes any version of the document and the version number
to make current; the newer version is kept, so a later restore brings it back.
`restore_insight` refuses an insight that is not dismissed and one the caller cannot see.

A change alert belongs to the person who set it up, and its notices go only to them: the
service now scopes an update to its owner in the repository as well as in the plan, and runs
the table allowlist again on every change. Dismissing a watchtower item takes it off the shared
attention feed and leaves the record behind it alone; an item that oversees agents' own work,
and one already resolved, is refused.

## Capture intake

```
list_capture_batches ──► file_capture_items
                     ──► discard_capture_item | discard_capture_batch
```

| Tool | Resource / operation | Class | Default → most | Runs only from a person's approval |
| --- | --- | --- | --- | --- |
| `list_capture_batches` | capture batch / read | reads only | automatic | — |
| `file_capture_items` | capture batch / update | inside | Propose → Ask first | no |
| `discard_capture_item` | capture batch / delete | inside | Propose → Ask first | no |
| `discard_capture_batch` | capture batch / delete | inside | Propose → Propose | no |

`list_capture_batches` shows the open stacks (ready, partly filed or failed) and the documents
the split proposed in each. `file_capture_items` files at most 25 of them, each onto a record
by `capture.targetType` and id as a document type, as the person it acts for; one goes through
the page's single filing and more through its batch filing, and any that fail are named in the
error. A misfiled document can be deleted. Discarding a batch drops every unfiled page with
nothing to bring them back, so it is always proposed.

## What no agent does

| Write | Category | Why |
| --- | --- | --- |
| Start a scan or catch a print on a paired computer | security | It drives one of the person's own computers under the access they granted that machine, and a scan needs the person at the scanner. |
| Cancel a scan or print request | security | It withdraws an instruction the person gave their own paired computer. |
| Print capture cover sheets | infrastructure | The routing codes exist only to be printed and laid on paper; `file_capture_items` files what they route. |
| Split, merge, reorder or rotate a scanned stack's pages | user-preference | The person arranges pages looking at their images; pages carry no text an agent can read. |
| Re-extract a shipment draft from a document | infrastructure | A repair that re-runs the extraction pipeline and replaces the draft being reviewed. |
| Hand off a watchtower item | agent-administration | Agents hand work to one another through `delegate_task`. |

## Who holds them

| Template | Runs | Holds |
| --- | --- | --- |
| Customer assistant | chat, at a Propose ceiling | `create_customer`, `update_customer`, `update_customer_status`, `list_locations`, `update_location`, `update_location_status`, `list_commodities`, `create_commodity`, `update_commodity` |
| Insight analyst | on each new insight, at a Propose ceiling | `restore_insight` |

The customer assistant handles customer records and now holds 53 of its 56 tools. The insight
analyst dismisses insights it finds are noise and can bring one back. Both run at a Propose
ceiling, so every write they make waits for a person and needs no new permission ceiling
entry.

Every other tool on this page is registered and on no template. The dispatch, billing and
compliance assistants each hold 55 of 56 tools and the settlements clerk all 56, so the
carrier, equipment and document tools have no room there; the carrier risk desk says whether a
carrier can be tendered and leaves the decision to stop using one to a person; and capture
filing is a person's paperwork, done on their behalf. An organization adds these tools to an
agent it builds in AI control.

## Known limits

- An agent made from the customer assistant or insight analyst template before these tools
  existed keeps the tools it was saved with; an administrator adds them in AI control.
- A bulk status change, a document delete and a capture filing each take at most 25 records;
  `list_capture_batches` shows at most 10 open stacks.
- An agent cannot edit a customer's billing or email profile, a carrier's contacts, insurance
  or EDI channels, or a location's geofence; a person does that on the page.
- `restore_document_version` names the version by number, so a document with more than 10,000
  versions cannot be restored past that by an agent.
