# AI tool safety

<!-- Generated from the tool policies in code by running, in services/tms:
     go generate ./internal/modules/agent/app/agenttoolpolicy/safetydoc/...
     Do not edit by hand. -->

What every tool an agent can call may do without a person, read from the
policies the tools declare in code. The runtime decides each call from the same
policies, so this page cannot drift from what runs: CI regenerates it and fails
when it differs. Each tool is listed once, under the furthest class its work
can reach.

Tools listed: 634.

## The model

**Classes.** Every tool says where its work can be seen or felt: nowhere (it
only reads), the caller's own records, inside the organization, a customer, a
driver, someone outside the organization, or money. A tool whose calls differ
classifies each call, and the class of that call decides.

**Tiers.** An agent runs a tool at one of three tiers: *Propose* (a person
decides and nothing runs until then), *Ask first* (it runs once a person
approves) or *Automatic* (it runs without asking). The tier a call gets is the
lowest of the tier the agent sets for the tool, the agent's ceiling, the tool's
max tier, its class's ceiling and any condition the call's record must meet.
Work a customer or a driver can see, or that is sent outside the organization,
never runs past Ask first. Earned autonomy moves a tool up one tier after a
streak of clean approvals, and never past those limits.

**Money.** A money write runs at most Ask first, so a new money tool asks a
person by default. It runs on its own only on a tool whose condition proves the
call is within bounds: the records it touches are in the state the tool
requires, and the amount it moves is within the organization's money limits for
agents in AI Control, which are a ceiling on one change and a cap on each run
and on each day in both the number of changes and the total moved. The limits
are amounts in the organization's functional currency, so a change in any other
currency waits for a person, and the caps count only changes in it. The day is
the organization's own and counts every agent, a delegate's changes count
against the run that handed it the task, and a change that failed moved nothing
and counts against nothing. A call outside any bound, or whose bounds, limits or
day cannot be read, is filed as a proposal that names the bound it exceeded. A
run that has read outside text never moves money on its own, with one
exception a tool declares: a bound that reads a record again from the
organization's own records, and takes the amount, the customer and the
references from it rather than from the call, may vouch for having read that
record. Only a bank receipt can be vouched for this way, by the tools that say
so below, and only for a receipt the run itself read and named: a receipt
handed across a delegation, one read without naming which, or reads past what
the run can keep track of hold the change, as does any other outside text the
run read.

**Taint.** Some tools return text written outside the organization: an inbound
message, a document, an EDI transaction, a bank receipt, a weather alert, a
comment a driver or a trading partner left on a shipment. Once a run has read
such text, every call that would leave the organization or move money waits
for approval, whatever its tier, and so does any call a tool's own taint hold
names, so an instruction hidden in that text cannot act on its own. Every call
taint held names it among what held it, whatever else held it too.

**Personal exemption.** A call that changes only the caller's own records runs
without a decision while that person is in the conversation, unless a person
set the tool's tier on the agent. An unattended run never has it.

**Escalation.** Raising an exception, which hands the run to a person for
review, runs whatever the agent's autonomy ceiling, the turn's taint or the run being
unattended, because it only records the run for a person inside the
organization. A run that is unsure has to be able to hand its work to someone
without first waiting for someone. A tier a person set for the tool on the
agent still holds.

**Unattended runs.** A run nobody is in is authorized as the agent, from the
fixed table of what any agent may do, and never from a role: it cannot approve,
and it cannot reach a person's own records. What it writes is attributed to the
instance's system user, so a record's created-by and updated-by, an automatic
write's executor and the audit log's user name that account rather than nobody,
and the audit log's description says which agent ran it ("Ran by Dispatch
Agent"); the account lends its name, not its permissions.

**Data access.** A read shows a field only when the reader's data access
reaches it. An unattended agent reads at its own data access setting, Internal
unless someone whose role reaches Restricted raises it; a run a person is in
reads at the lower of that setting and the person's own role. Amounts, pay,
memos and raw EDI above that tier are left out and named in withheldByAccess,
and Confidential fields never reach a model at all.

## Classes

| Class | Means | Runs at most | Held once tainted | Tools that reach it |
| --- | --- | --- | --- | --- |
| Reads only | Looks something up. Nothing changes and nothing is sent. | Automatic | No | 211 |
| The caller's own records | Changes only the records of the person using the agent. | Automatic | No | 9 |
| Inside the organization | Changes records only people inside the organization see. | Automatic | No | 266 |
| Seen by a customer | Changes something a customer can see. | Ask first | Yes | 5 |
| Seen by a driver | Changes something a driver can see. | Ask first | Yes | 23 |
| Sent outside the organization | Sends to someone outside the organization. | Ask first | Yes | 43 |
| Money | Moves or commits money. A call runs on its own only when its tool proves it is within bounds. | Ask first | Yes | 100 |

## Reads only

Looks something up. Nothing changes and nothing is sent.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Get accounting connection (`accounting_get_connection`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get accounting mapping (`accounting_get_mapping`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get accounting sync record (`accounting_get_sync_record`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List accounting drift findings (`accounting_list_drift_findings`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List accounting inbound changes (`accounting_list_inbound_changes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List accounting mapping gaps (`accounting_list_mapping_gaps`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List accounting sync records (`accounting_list_sync_records`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Find agent page (`agent_find_page`) | Reads only | Automatic | — | — | Searches the product guide for the caller; nothing changes and nothing is sent. |
| Get agent run (`agent_get_run`) | Reads only | Automatic | — | When the record is marked, from run record | Reads a run's own record, whose summary may repeat outside text the run read. |
| List agent memories (`agent_list_memories`) | Reads only | Automatic | — | When the record is marked, from memory | Reads memories earlier runs saved, which carry the taint of the run that wrote them. It returns only what the reader's own prompt already reads, so it is used under the assistant like the Desk's memory page. |
| List agent runs (`agent_list_runs`) | Reads only | Automatic | — | When the record is marked, from run record | Lists agent runs, whose summaries may repeat outside text a run read; a run on a conversation is listed only to the person who owns it. |
| Open agent page (`agent_open_page`) | Reads only | Automatic | — | — | Opens a page in the caller's own browser; nothing changes and nothing is sent. |
| Read agent attachment (`agent_read_attachment`) | Reads only | Automatic | — | Always, from attachment | Reads text from a file the person attached to their own conversation; nothing changes and nothing is sent. |
| Run agent script (`agent_run_script`) | Reads only | Automatic | — | When the record is marked, from record note, inbound message, document, EDI, bank receipt, weather, attachment, memory or run record | Runs a short script in a sandbox that cannot reach the network; every read it makes is one of the agent's own reads, run as the person under the same checks, and the call carries the taint of every read that brought outside text in. It changes nothing. |
| Ask user (`ask_user`) | Reads only | Automatic | — | — | Asks the person in the conversation a question; nothing is saved or sent. |
| Get billing queue item (`billing_get_queue_item`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List billing queue items (`billing_list_queue_items`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List billing transfer candidates (`billing_list_transfer_candidates`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get carrier (`carrier_get`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get carrier intelligence event (`carrier_get_intelligence_event`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List carriers (`carrier_list`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List carrier capacity postings (`carrier_list_capacity_postings`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List carrier intelligence events (`carrier_list_intelligence_events`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List carrier invoice matches (`carrier_list_invoice_matches`) | Reads only | Automatic | — | When the record is marked, from EDI | Lists carrier invoices, some of which a carrier sent over EDI with its own invoice text; nothing changes and nothing is sent. |
| Shop carrier (`carrier_shop`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get customer (`customer_get`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get customer update preferences (`customer_get_update_preferences`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List customers (`customer_list`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Delegate task (`delegate_task`) | Reads only | Automatic | — | — | Hands a task to another agent of the organization, which runs as the same person under its own tiers. |
| Get detention occurrence (`detention_get_occurrence`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List detention occurrences (`detention_list_occurrences`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get dispatch board (`dispatch_get_board`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List tenders (`dispatch_list_tenders`) | Reads only | Automatic | — | — | Reads a shipment's tenders and their offers; a carrier's own words, such as a decline reason, are left out, so nothing written outside the organization is read. |
| Plan dispatch (`dispatch_plan`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Rank dispatch candidates (`dispatch_rank_candidates`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get document summary (`document_get_summary`) | Reads only | Automatic | — | Always, from document | Reads text extracted from a document someone outside sent; nothing changes and nothing is sent. |
| List documents (`document_list`) | Reads only | Automatic | — | Always, from document | Searches text extracted from documents, many of which someone outside wrote; each record returned is checked against what the caller may read, and nothing changes or is sent. |
| List document types (`document_list_types`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Draft EDI connection (`edi_draft_connection`) | Reads only | Automatic | — | Always, from attachment | Reads a trading partner's guide the person attached and plans the connection from it; nothing is stored or sent, and every write after it waits for a person. |
| Get EDI ack health (`edi_get_ack_health`) | Reads only | Automatic | — | Always, from EDI | Reads how partners have acknowledged what was sent, with the delivery errors their systems returned; nothing changes and nothing is sent. |
| Get EDI inbound file (`edi_get_inbound_file`) | Reads only | Automatic | — | Always, from EDI | Reads an EDI file a trading partner sent, raw X12 included on request; nothing changes and nothing is sent. |
| Get EDI message (`edi_get_message`) | Reads only | Automatic | — | Always, from EDI | Reads one EDI document, the errors and acknowledgment a partner's system returned for it, and its raw X12 on request; nothing changes and nothing is sent. |
| Get EDI partner (`edi_get_partner`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get EDI transfer (`edi_get_transfer`) | Reads only | Automatic | — | Always, from EDI | Reads a load tender a trading partner wrote, stops and references included; nothing changes and nothing is sent. |
| Inspect EDI x12 (`edi_inspect_x12`) | Reads only | Automatic | — | Always, from EDI | Parses X12 a person handed the agent, which a trading partner wrote; nothing is stored, changed or sent, and every write after it waits for a person. |
| List EDI carrier invoices (`edi_list_carrier_invoices`) | Reads only | Automatic | — | Always, from EDI | Lists carrier invoices a carrier sent over EDI with its own invoice text; nothing changes and nothing is sent. |
| List EDI inbound files (`edi_list_inbound_files`) | Reads only | Automatic | — | Always, from EDI | Lists EDI files trading partners sent, whose names and failure reasons repeat the partner's text; nothing changes and nothing is sent. |
| List EDI messages (`edi_list_messages`) | Reads only | Automatic | — | Always, from EDI | Lists EDI documents and the errors partners' systems returned for them; nothing changes and nothing is sent. |
| List EDI partners (`edi_list_partners`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List EDI tender changes (`edi_list_tender_changes`) | Reads only | Automatic | — | Always, from EDI | Lists changes another organization wrote to loads tendered over EDI; nothing changes and nothing is sent. |
| List EDI transfer changes (`edi_list_transfer_changes`) | Reads only | Automatic | — | Always, from EDI | Lists statuses another organization reported on a linked load; nothing changes and nothing is sent. |
| List EDI transfers (`edi_list_transfers`) | Reads only | Automatic | — | Always, from EDI | Lists load tenders trading partners sent, whose contents the partner wrote; nothing changes and nothing is sent. |
| Preview EDI x12 (`edi_preview_x12`) | Reads only | Automatic | — | When the record is marked, from EDI | Renders the X12 a send would produce from records already in Trenova, with the partner profile's findings; nothing is saved and nothing is sent. |
| Get tractor (`equipment_get_tractor`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get trailer (`equipment_get_trailer`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List equipment manufacturers (`equipment_list_manufacturers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List equipment positions (`equipment_list_positions`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List tractors (`equipment_list_tractors`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List trailers (`equipment_list_trailers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List equipment types (`equipment_list_types`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get service failure (`exception_get_service_failure`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List service failure reason codes (`exception_list_service_failure_reason_codes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List service failures (`exception_list_service_failures`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List watchtower items (`exception_list_watchtower_items`) | Reads only | Automatic | — | When the record is marked, from inbound message, EDI, weather or run record | Reads the watchtower feed, whose headlines can quote an inbound email, an EDI file, a weather alert or a run that read outside text; it changes nothing, not even what the person has seen. |
| Get factoring agreement (`factoring_get_agreement`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get factoring readiness (`factoring_get_readiness`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get factoring remittance (`factoring_get_remittance`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get factoring reserve balance (`factoring_get_reserve_balance`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get factoring submission (`factoring_get_submission`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List factoring noas (`factoring_list_noas`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List factoring readiness (`factoring_list_readiness`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List factoring remittances (`factoring_list_remittances`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List factoring submissions (`factoring_list_submissions`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Find tools (`find_tools`) | Reads only | Automatic | — | — | Loads more of the agent's own tools into the turn; it reads the catalog and changes nothing. |
| Describe formula schema (`formula_describe_schema`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get formula template (`formula_get_template`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List formula templates (`formula_list_templates`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Propose formula (`formula_propose`) | Reads only | Automatic | — | — | Changes the person's own editor, which they can undo; nothing is saved and nothing is sent. |
| Test formula expression (`formula_test_expression`) | Reads only | Automatic | — | — | Prices an expression with the formula engine, for sample values or a shipment the caller may read; nothing is saved and nothing is sent. |
| List fuel cards (`fuel_list_cards`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List IFTA jurisdictions (`fuel_list_ifta_jurisdictions`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List IFTA mileage entries (`fuel_list_ifta_mileage_entries`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List IFTA returns (`fuel_list_ifta_returns`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List fuel purchase imports (`fuel_list_purchase_imports`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List fuel purchases (`fuel_list_purchases`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get home layout (`home_get_layout`) | Reads only | Automatic | — | — | Reads the caller's own home page; nothing changes and nothing is sent. |
| List home widgets (`home_list_widgets`) | Reads only | Automatic | — | — | Lists the widgets the caller's home page can show; nothing changes and nothing is sent. |
| Get inbound message (`inbox_get_inbound_message`) | Reads only | Automatic | — | Always, from inbound message | Reads mail an outsider wrote; nothing changes and nothing is sent. |
| List capture batches (`inbox_list_capture_batches`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List email profiles (`inbox_list_email_profiles`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List inbound messages (`inbox_list_inbound_messages`) | Reads only | Automatic | — | Always, from inbound message | Lists and searches mail outsiders wrote, subjects, senders and bodies included; nothing changes and nothing is sent. |
| List inbox triage assignees (`inbox_list_triage_assignees`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get insight (`insight_get`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get insight daily briefing (`insight_get_daily_briefing`) | Reads only | Automatic | — | — | Reads the morning's computed figures, and only the sections the caller may read; it changes nothing, not even whether the briefing was read. |
| List insights (`insight_list`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List insight weather alerts (`insight_list_weather_alerts`) | Reads only | Automatic | — | Always, from weather | Reads National Weather Service alert text; nothing changes and nothing is sent. |
| Accept intake confident (`intake_accept_confident`) | Reads only | Automatic | — | — | Hands a change to the shipment the person is building on their own page; nothing is saved and nothing is sent. |
| Accept intake field (`intake_accept_field`) | Reads only | Automatic | — | — | Hands a change to the shipment the person is building on their own page; nothing is saved and nothing is sent. |
| Get intake draft (`intake_get_draft`) | Reads only | Automatic | — | Always, from document | Reads a shipment drafted from a document someone outside sent; nothing changes and nothing is sent. |
| Set intake field (`intake_set_field`) | Reads only | Automatic | — | — | Hands a change to the shipment the person is building on their own page; nothing is saved and nothing is sent. |
| Set intake required field (`intake_set_required_field`) | Reads only | Automatic | — | — | Hands a change to the shipment the person is building on their own page; nothing is saved and nothing is sent. |
| Set intake stop location (`intake_set_stop_location`) | Reads only | Automatic | — | — | Hands a change to the shipment the person is building on their own page; nothing is saved and nothing is sent. |
| Set intake stop schedule (`intake_set_stop_schedule`) | Reads only | Automatic | — | — | Hands a change to the shipment the person is building on their own page; nothing is saved and nothing is sent. |
| Get invoice (`invoice_get`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get invoice adjustment (`invoice_get_adjustment`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get invoice run (`invoice_get_run`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List invoices (`invoice_list`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List invoice adjustments (`invoice_list_adjustments`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List invoice disputes (`invoice_list_disputes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List invoice open statements (`invoice_list_open_statements`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List invoice runs (`invoice_list_runs`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List invoice share candidates (`invoice_list_share_candidates`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get ledger fiscal close blockers (`ledger_get_fiscal_close_blockers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get journal entry (`ledger_get_journal_entry`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get manual journal (`ledger_get_manual_journal`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List ledger accounts (`ledger_list_accounts`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List fiscal periods (`ledger_list_fiscal_periods`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List journal entries (`ledger_list_journal_entries`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List journal reversals (`ledger_list_journal_reversals`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List manual journals (`ledger_list_manual_journals`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get location (`location_get`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List locations (`location_list`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List location categories (`location_list_categories`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List accessorial charges (`masterdata_list_accessorial_charges`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List commodities (`masterdata_list_commodities`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List fleet codes (`masterdata_list_fleet_codes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List hazardous materials (`masterdata_list_hazardous_materials`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List hold reasons (`masterdata_list_hold_reasons`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List service types (`masterdata_list_service_types`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List shipment types (`masterdata_list_shipment_types`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Name plan (`name_plan`) | Reads only | Automatic | — | — | Names the plan the person approves the turn's proposals as. It writes no business record: the name is kept only as the plan's own title, beside the previewed changes the approval rests on. |
| Get order (`order_get`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List orders (`order_list`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Explain rate (`rate_explain`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get rate agreement (`rate_get_agreement`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get rate matrix (`rate_get_matrix`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List rate agreements (`rate_list_agreements`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List rate confirmations (`rate_list_confirmations`) | Reads only | Automatic | — | — | Reads a move's rate confirmation revisions; the name a carrier typed when signing is left out, so nothing written outside the organization is read. |
| List fuel index prices (`rate_list_fuel_index_prices`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List fuel surcharge programs (`rate_list_fuel_surcharge_programs`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List rate imports (`rate_list_imports`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Quote rate (`rate_quote`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get receivable aging (`receivable_get_aging`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get bank receipt (`receivable_get_bank_receipt`) | Reads only | Automatic | — | Always, from bank receipt | Reads a bank receipt whose memo the payer wrote; nothing changes and nothing is sent. |
| Get receivable statement (`receivable_get_statement`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List bank receipt exceptions (`receivable_list_bank_receipt_exceptions`) | Reads only | Automatic | — | Always, from bank receipt | Lists bank receipts whose memos the payers wrote; nothing changes and nothing is sent. |
| List receivable collection worklist (`receivable_list_collection_worklist`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List receivable credit memo applications (`receivable_list_credit_memo_applications`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List customer payments (`receivable_list_customer_payments`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List receivable open items (`receivable_list_open_items`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Analyze report data (`report_analyze_data`) | Reads only | Automatic | — | When the record is marked, from attachment | Runs code over report results and files the person already read, in a sandbox that cannot reach the network; it changes nothing, and it carries the taint of any attached file it reads. |
| Compare report runs (`report_compare_runs`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Compose report table view (`report_compose_table_view`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get report (`report_get`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get report dataset (`report_get_dataset`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get report run (`report_get_run`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List reports (`report_list`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List dashboards (`report_list_dashboards`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List report datasets (`report_list_datasets`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List report runs (`report_list_runs`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List report schedules (`report_list_schedules`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List table change alerts (`report_list_table_change_alerts`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Preview report (`report_preview`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Run report (`report_run`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get carrier settlement (`settlement_get_carrier`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get settlement dispute (`settlement_get_dispute`) | Reads only | Automatic | — | When the record is marked, from record note | Reads a pay dispute whose description a driver wrote in the driver portal; nothing changes and nothing is sent. |
| Get driver settlement (`settlement_get_driver`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get driver settlement earnings (`settlement_get_driver_earnings`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List settlement advances (`settlement_list_advances`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List carrier settlements (`settlement_list_carrier`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List driver settlements (`settlement_list_driver`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List driver expenses (`settlement_list_driver_expenses`) | Reads only | Automatic | — | When the record is marked, from record note | Reads expenses whose descriptions drivers wrote in the driver portal; nothing changes and nothing is sent. |
| List settlement escrow accounts (`settlement_list_escrow_accounts`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List settlement pay assignments (`settlement_list_pay_assignments`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List settlement pay codes (`settlement_list_pay_codes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List settlement pay events (`settlement_list_pay_events`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List settlement pay profiles (`settlement_list_pay_profiles`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List settlement payroll exports (`settlement_list_payroll_exports`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List settlement recurring deductions (`settlement_list_recurring_deductions`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List settlement recurring earnings (`settlement_list_recurring_earnings`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get shipment (`shipment_get`) | Reads only | Automatic | — | When the record is marked, from record note | Reads a shipment with its newest comments, some of which a driver, a trading partner or another system outside the organization wrote; nothing changes and nothing is sent. |
| Get shipment tracking (`shipment_get_tracking`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List shipments (`shipment_list`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List shipment permits (`shipment_list_permits`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List shipment recurring (`shipment_list_recurring`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Match shipment records (`shipment_match_records`) | Reads only | Automatic | — | — | Matches the records a new shipment names against the organization's own; nothing changes and nothing is sent. |
| List portal sites (`web_list_portal_sites`) | Reads only | Automatic | — | — | Lists the portal sites an administrator set up; it changes nothing, sends nothing, and returns only what administrators wrote. |
| List web watches (`web_list_watches`) | Reads only | Automatic | — | — | Lists the web watches set for the person; it changes nothing, reaches no one, and returns only what the agents wrote when they set them. |
| Read web (`web_read`) | Reads only | Automatic | — | Always, from web | Reads a page a web search returned; it changes nothing in Trenova and returns text written outside it. |
| Read web portal (`web_read_portal`) | Reads only | Automatic | — | Always, from web | Reads a third-party portal through the organization's extension; it changes nothing in Trenova, tells the browser to submit nothing, and returns text written outside it. |
| Search web (`web_search`) | Reads only | Automatic | — | Always, from web | Searches the public web through the organization's extension; it changes nothing in Trenova, sends only the query, and returns text written outside it. |
| Withdraw proposal (`withdraw_proposal`) | Reads only | Automatic | — | — | Takes back a proposal the agent filed earlier in the same turn, before anyone has seen it; it changes no record and only narrows what the person is asked. |
| Get worker (`worker_get`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get worker credential (`worker_get_credential`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get worker DOT random draw (`worker_get_dot_random_draw`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get worker HOS (`worker_get_hos`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get worker schedule (`worker_get_schedule`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List workers (`worker_list`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List worker checklists (`worker_list_checklists`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List worker credentials (`worker_list_credentials`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List worker DOT random draws (`worker_list_dot_random_draws`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List worker DOT tests (`worker_list_dot_tests`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List worker employment verifications (`worker_list_employment_verifications`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List worker injuries (`worker_list_injuries`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List worker leave cases (`worker_list_leave_cases`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List performance reviews (`worker_list_performance_reviews`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List worker safety events (`worker_list_safety_events`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List shift templates (`worker_list_shift_templates`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List worker time off (`worker_list_time_off`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List worker training (`worker_list_training`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List training courses (`worker_list_training_courses`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |

## The caller's own records

Changes only the records of the person using the agent.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Add home widget (`home_add_widget`) | The caller's own records | Automatic | — | — | Changes only the caller's own home page. |
| Arrange home layout (`home_arrange_layout`) | The caller's own records | Automatic | — | — | Changes only the caller's own home page. |
| Remove home widget (`home_remove_widget`) | The caller's own records | Automatic | — | — | Changes only the caller's own home page. |
| Publish artifact (`publish_artifact`) | The caller's own records | Automatic | — | — | Publishes a document into the caller's own conversation, where only they read it. |
| Request decision (`request_decision`) | The caller's own records | Automatic | — | — | Opens, in the person's own conversation, the approval box on a proposal already waiting on them; it decides nothing and changes no record. |

## Inside the organization

Changes records only people inside the organization see.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Check accounting connection (`accounting_check_connection`) | Inside the organization | Automatic | — | — | Asks the accounting system whether it answers and records the result in Trenova; it writes nothing to the books. |
| Check accounting drift (`accounting_check_drift`) | Inside the organization | Ask first | — | — | Reads every synced document back from the accounting system, which spends the provider's call allowance, so a person approves it. |
| Clear accounting mapping (`accounting_clear_mapping`) | Inside the organization | Ask first | — | — | Unmatches a record so it cannot sync until someone maps it again; mapping it again undoes it. |
| Confirm accounting mapping (`accounting_confirm_mapping`) | Inside the organization | Ask first | — | — | Decides which accounts and records Trenova's documents post to in the books, so a person approves it; clearing or changing a mapping undoes it. |
| Create accounting reference record (`accounting_create_reference_record`) | Inside the organization | Ask first | — | — | Creates a record in the organization's own accounting system; Trenova cannot delete it again, so a person approves it. |
| Dismiss accounting drift finding (`accounting_dismiss_drift_finding`) | Inside the organization | Ask first | — | — | Closes a difference with the books without fixing it, so a person approves it. |
| Ignore accounting inbound change (`accounting_ignore_inbound_change`) | Inside the organization | Ask first | — | — | Keeps a payment the books hold out of Trenova for good, so a person approves it. |
| Pause accounting sync (`accounting_pause_sync`) | Inside the organization | Automatic | — | — | Holds what is waiting to go to the books; nothing is sent, changed or lost, and resuming sends it on. |
| Redate accounting sync record (`accounting_redate_sync_record`) | Inside the organization | Ask first | — | — | Changes the date the organization's books record for a document, so a person approves it. |
| Refresh accounting reference data (`accounting_refresh_reference_data`) | Inside the organization | Automatic | — | — | Reads the accounting system and refreshes Trenova's suggestions; it writes nothing to the books and leaves confirmed mappings alone. |
| Reject accounting mapping (`accounting_reject_mapping`) | Inside the organization | Ask first | — | — | Unmatches a proposal so it is not used; choosing a record for the mapping undoes it. |
| Request accounting backfill (`accounting_request_backfill`) | Inside the organization | Ask first | — | — | Sends historical documents to the organization's books, some of which may already be there by hand, and cannot be called back; a person who manages the integration approves it. |
| Resume accounting sync (`accounting_resume_sync`) | Inside the organization | Ask first | — | — | Releases everything held to the organization's books at once, and a person paused it for a reason, so a person approves it; what is sent cannot be called back. |
| Retry accounting sync record (`accounting_retry_sync_record`) | Inside the organization | Automatic | — | — | Sends again, to the organization's own books, documents Trenova already decided to send; the accounting system recognizes a repeat by its request id, so nothing is entered twice, and a document held for release stays held. |
| Set accounting mapping (`accounting_set_mapping`) | Inside the organization | Ask first | — | — | Decides which account or record Trenova's invoices, payments and bills will post to, so a person approves it; clearing or changing it undoes it. |
| Skip accounting sync record (`accounting_skip_sync_record`) | Inside the organization | Ask first | — | — | Leaves a document out of the organization's books for good; nothing sends it again, so a person approves it. |
| Update accounting backfill (`accounting_update_backfill`) | Inside the organization | Ask first | — | — | Changes how much history reaches the organization's books; a person who manages the integration approves it, and pausing or cancelling sends nothing. |
| Cancel agent wait (`agent_cancel_wait`) | Inside the organization | Automatic | — | Carries outside text into later runs | Cancels a wait the agent set on its own work; it changes no record and reaches nobody. |
| Create agent memory (`agent_create_memory`) | The caller's own records, inside the organization | Automatic | Each call is classified by what it reaches. An Instruction, a Procedure or a Correction recorded after the run read text from outside the organization waits for a person's approval; a Fact is recorded and stays marked as drawn from outside text. A call on the caller's own records runs unasked while they are present. | Carries outside text into later runs | Saves a memory later runs read, so it keeps the taint of the run that wrote it. One kept for the caller alone is their own record; one for their team or the organization reaches colleagues' conversations. |
| Create agent wait (`agent_create_wait`) | Inside the organization | Automatic | — | Carries outside text into later runs | Parks the agent's own work until something happens and starts it again then; it changes no record and reaches nobody. |
| Guide agent profile (`agent_guide_profile`) | The caller's own records, inside the organization | Automatic | Each call is classified by what it reaches. Guidance recorded after the run read text from outside the organization waits for a person's approval, because every later prompt is written from it. A call on the caller's own records runs unasked while they are present. | Carries outside text into later runs | Adds a line every later write-up of a profile honours. Guidance about the person's own work reaches only their conversations; guidance about the organization reaches everyone's. |
| Raise agent exception (`agent_raise_exception`) | Inside the organization | Automatic | Runs above the agent's ceiling: it only puts the run in front of a person, so it waits for no one unless a person set its tier on the agent. | — | Records an exception against the run itself for a person inside the organization. |
| Retire agent memory (`agent_retire_memory`) | Inside the organization | Automatic | — | — | Retires an agent memory inside Trenova: one kept for the person in the conversation, or one shared with others when that person may manage shared memory. A run with nobody in it retires none. |
| Assign billing biller (`billing_assign_biller`) | Inside the organization | Automatic | — | — | Names who reviews billing queue items inside Trenova; it creates no money and is changed by assigning someone else. |
| Hold billing queue item (`billing_hold_queue_item`) | Inside the organization | Automatic | Its notes are what a biller or operations reads next, so a run that has read outside text proposes it rather than writing it. | — | Parks an item inside Trenova with a note; it creates no money and is undone by moving the item on. |
| Manage billing transfer run (`billing_manage_transfer_run`) | Inside the organization | Ask first | — | — | Stops or reruns a transfer inside Trenova; nothing is invoiced or sent, and a stopped run is retried the same way. |
| Raise billing exception (`billing_raise_exception`) | Inside the organization | Automatic | Its notes are what a biller or operations reads next, so a run that has read outside text proposes it rather than writing it. | — | Marks an item for a biller to resolve inside Trenova; it creates no money and a biller moves it back when it is resolved. |
| Release billing queue item (`billing_release_queue_item`) | Inside the organization | Ask first | What the hold waited on is what outside text would claim has arrived, so a run that has read it proposes the release rather than making it. | — | Moves an item back into the review queue inside Trenova; it creates no money and billing_hold_queue_item puts it back on hold. |
| Return billing to operations (`billing_return_to_operations`) | Inside the organization | Automatic | Its notes are what a biller or operations reads next, so a run that has read outside text proposes it rather than writing it. | — | Returns an item to operations with a note on its shipment inside Trenova; it creates no money and the item comes back when operations fixes it. |
| Start billing review (`billing_start_review`) | Inside the organization | Automatic | An item on hold was held there by a person or a rule, so moving one into review is a proposal a person decides; an item in any other state moves as far as the agent allows, and one that cannot be read waits for a person. A set is held to what its most guarded item allows. | — | Moves billing queue items into review inside Trenova; nothing is sent anywhere. |
| Transfer billing (`billing_transfer`) | Inside the organization | Automatic | — | — | Hands delivered shipments to the billing queue inside Trenova, by the checks the transfer dialog makes; a biller, or the organization's own auto-approve rule, still decides every item. |
| Acknowledge carrier intelligence event (`carrier_acknowledge_intelligence_event`) | Inside the organization | Automatic | — | — | Acknowledges a carrier finding inside Trenova. |
| Apply carrier intelligence suggestions (`carrier_apply_intelligence_suggestions`) | Inside the organization | Ask first | — | — | Edits the carrier's own profile inside Trenova from what the provider reports; nothing is sent, and the old values are set back by hand. |
| Create carrier capacity posting (`carrier_create_capacity_posting`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Adds a capacity posting inside Trenova; nothing is sent to the carrier, and carrier_delete_capacity_posting removes it. |
| Create carrier invoice match (`carrier_create_invoice_match`) | Inside the organization | Automatic | — | — | Records how a carrier's invoice compares with the expected cost inside Trenova; nothing is paid until a person accepts it. |
| Delete carrier capacity posting (`carrier_delete_capacity_posting`) | Inside the organization | Ask first | — | — | Removes trucks a carrier offered from Trenova; nothing is sent, and carrier_create_capacity_posting records them again. |
| Import carrier (`carrier_import`) | Inside the organization | Ask first | — | — | Creates a carrier inside Trenova from the provider's profile; nothing is sent to it, and one added in error is made inactive. |
| Record carrier intelligence review (`carrier_record_intelligence_review`) | Inside the organization | Propose | — | — | Puts a person's name on a carrier risk review; only that person can say they reviewed it. |
| Reject carrier invoice match (`carrier_reject_invoice_match`) | Inside the organization | Automatic | Its note is what accounts payable reads about the carrier's invoice, so a run that has read outside text proposes it. | — | Marks a carrier's invoice as not payable inside Trenova; nothing is paid and a corrected invoice is matched afresh. |
| Resolve carrier intelligence event (`carrier_resolve_intelligence_event`) | Inside the organization | Automatic | — | — | Closes a carrier finding inside Trenova. |
| Set carrier status (`carrier_set_status`) | Inside the organization | Ask first | — | — | Changes whether carriers are offered for booking inside Trenova; nothing is sent. |
| Update carrier capacity posting (`carrier_update_capacity_posting`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Changes a capacity posting inside Trenova; nothing is sent, and a later change puts it back. |
| Verify carrier equipment (`carrier_verify_equipment`) | Inside the organization | Ask first | — | — | Records an equipment check inside Trenova after a provider lookup; nothing is sent to the carrier. |
| Set customer status (`customer_set_status`) | Inside the organization | Ask first | — | — | Changes whether customers are offered for booking inside Trenova; nothing is sent. |
| Escalate detention (`detention_escalate`) | Inside the organization | Automatic | — | — | Hands a detention clock to a person inside the organization. |
| Assign dispatch move (`dispatch_assign_move`) | Inside the organization | Automatic | — | — | Assigns a driver and tractor to a move; the driver sees the assignment but no text the model wrote. |
| Record dispatch check call (`dispatch_record_check_call`) | Inside the organization | Automatic | — | — | Records a check call, and for an arrival or departure the stop's actual time; nothing leaves the organization. |
| Record dispatch stop actual (`dispatch_record_stop_actual`) | Inside the organization | Automatic | — | — | Records arrival and departure times on a stop; no model-written text leaves the organization. |
| Set dispatch move status (`dispatch_set_move_status`) | Inside the organization | Ask first | Canceling a move ends it for good, so a cancellation is a proposal a person decides; any other status runs once a person approves it. | — | Moves a move's status forward inside Trenova, which re-derives its shipment's status and, on completion, releases its equipment; a move never moves back, so it is not undone by running it again. |
| Split dispatch move (`dispatch_split_move`) | Inside the organization | Propose | — | — | Adds a move and turns a delivery into a relay inside Trenova; the two new windows are times only a person can vouch for, so a person always approves it. |
| Unassign dispatch move (`dispatch_unassign_move`) | Inside the organization | Automatic | Only one move at a time, still freshly assigned, is taken off its driver without a decision; several moves, a move the service would refuse and a move that cannot be read each wait for a person. | — | Takes the driver off a move that has not started, inside Trenova; the driver is told the load was taken off them but sees no text the model wrote, and assigning the move again undoes it. |
| Delete document (`document_delete`) | Inside the organization | Propose | — | — | Removes documents and their stored files from Trenova for good; nothing is sent, but nothing restores them either. |
| Restore document version (`document_restore_version`) | Inside the organization | Ask first | — | — | Changes which stored version of a document Trenova shows; nothing is sent or removed. |
| Apply EDI connection draft (`edi_apply_connection_draft`) | Inside the organization | Propose | — | Always, from attachment | Sets up a trading partner from a guide the partner wrote, so a person checks every value against its page before anything is saved A person always decides. |
| Link EDI carrier invoice (`edi_link_carrier_invoice`) | Inside the organization | Automatic | — | — | Names which carrier an inbound invoice belongs to inside Trenova; nothing is paid and it is relinked the same way. |
| Create tractor (`equipment_create_tractor`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Adds a tractor inside Trenova that dispatch can then assign; nothing is sent and it can be marked Sold or out of service. |
| Create trailer (`equipment_create_trailer`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Adds a trailer inside Trenova that dispatch can then assign; nothing is sent and it can be marked Sold or out of service. |
| Set tractor location (`equipment_set_tractor_location`) | Inside the organization | Ask first | — | — | Moves where Trenova thinks a tractor is; nothing is sent and locating it again moves it back. |
| Set tractor status (`equipment_set_tractor_status`) | Inside the organization | Automatic | — | — | Changes a tractor's status inside Trenova. |
| Set trailer status (`equipment_set_trailer_status`) | Inside the organization | Automatic | — | — | Changes a trailer's status inside Trenova. |
| Update tractor (`equipment_update_tractor`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Changes a tractor inside Trenova; nothing is sent, and a later change puts it back. |
| Update trailer (`equipment_update_trailer`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Changes a trailer inside Trenova; nothing is sent, and a later change puts it back. |
| Add exception comment (`exception_add_comment`) | Inside the organization | Ask first | — | — | Adds a line to an exception's history inside Trenova and tells its owner. |
| Assign exception (`exception_assign`) | Inside the organization | Ask first | — | — | Changes who owns an exception inside Trenova and notifies that person; the record behind it is unchanged. |
| Dismiss watchtower item (`exception_dismiss_watchtower_item`) | Inside the organization | Ask first | — | — | Takes an item off the shared attention feed inside Trenova; nothing is sent and the record behind it is unchanged. |
| Evaluate service failures (`exception_evaluate_service_failures`) | Inside the organization | Automatic | — | — | Opens service failures from stop actuals inside Trenova; an open failure sends nothing. |
| Reopen exception (`exception_reopen`) | Inside the organization | Ask first | — | — | Puts an exception back on the open queue inside Trenova; nothing outside changes. |
| Resolve exception (`exception_resolve`) | Inside the organization | Ask first | — | — | Closes an exception inside Trenova with an audited reason; the record behind it is unchanged. |
| Set exception due (`exception_set_due`) | Inside the organization | Ask first | — | — | Changes when an exception escalates inside Trenova; nothing outside changes. |
| Snooze exception (`exception_snooze`) | Inside the organization | Ask first | — | — | Hides an exception from the open queue until a time; nothing outside Trenova changes. |
| Update service failure (`exception_update_service_failure`) | Inside the organization | Ask first | — | — | Edits a service failure's own notes and codes inside Trenova; the overrides shape a later EDI 214, so a person approves each edit. |
| Wake exception (`exception_wake`) | Inside the organization | Ask first | — | — | Puts a snoozed exception back in the open queue inside Trenova; nothing outside changes and it can be snoozed again. |
| Cancel factoring submission (`factoring_cancel_submission`) | Inside the organization | Ask first | — | — | Releases invoices from a schedule; a person approves it. |
| Create factoring noa (`factoring_create_noa`) | Inside the organization | Automatic | — | — | Drafts a letter nobody outside sees until it is sent. |
| Discard factoring remittance (`factoring_discard_remittance`) | Inside the organization | Ask first | — | — | Sets a remittance aside for good before anything on it is booked. |
| Record factoring noa acknowledgement (`factoring_record_noa_acknowledgement`) | Inside the organization | Ask first | — | — | Lets the customer's invoices be sold to the factor; a person confirms the signed copy is the customer's. |
| Release factoring noa (`factoring_release_noa`) | Inside the organization | Ask first | — | — | Ends the customer's assignment to the factor; only the factor can release it, so a person confirms the release. |
| Rematch factoring remittance (`factoring_rematch_remittance`) | Inside the organization | Ask first | — | — | Recomputes suggestions and automatic matches before anything is booked; a person's own matches are kept. |
| Resolve factoring remittance line (`factoring_resolve_remittance_line`) | Inside the organization | Ask first | — | — | Changes which schedule item a remittance line pays before anything is booked; it can be undone until the remittance is applied. |
| Set factoring override (`factoring_set_override`) | Inside the organization | Ask first | — | — | Moves one invoice between the organization's receivables and the factor's; reversible until the invoice is submitted. |
| Amend IFTA return (`fuel_amend_ifta_return`) | Inside the organization | Propose | — | — | Starts a correction of a return already filed with the jurisdictions; it files nothing, but a person decides to reopen a filing. |
| Assign fuel card (`fuel_assign_card`) | Inside the organization | Ask first | — | — | Changes which tractor a card's purchases count against inside Trenova; nothing is sent, and assigning it again changes it back. |
| Backfill fuel jurisdiction miles (`fuel_backfill_jurisdiction_miles`) | Inside the organization | Ask first | — | — | Routes a quarter's unattributed moves inside Trenova in the background; nothing is sent to a customer or carrier, but each move is a billable request. |
| Commit fuel purchase import (`fuel_commit_purchase_import`) | Inside the organization | Ask first | — | — | Adds a statement's purchases inside Trenova, which the IFTA return counts; nothing is sent, but undoing it means deleting each purchase. |
| Correct IFTA mileage entry (`fuel_correct_ifta_mileage_entry`) | Inside the organization | Ask first | Miles read from a log or a message someone outside sent are proposed, since the IFTA return is computed from them. | — | Changes jurisdiction miles inside Trenova; nothing is sent, and a later correction changes them back. |
| Correct fuel purchase (`fuel_correct_purchase`) | Inside the organization | Ask first | A purchase read from a receipt or statement someone outside sent is proposed, since it is a tax record the IFTA return is computed from. | — | Changes a fuel purchase inside Trenova; nothing is sent, and a later correction changes it back. |
| Delete IFTA mileage entry (`fuel_delete_ifta_mileage_entry`) | Inside the organization | Propose | — | — | Removes miles the IFTA return is computed from, and nothing brings them back but entering them again, so a person always decides. |
| Delete IFTA return (`fuel_delete_ifta_return`) | Inside the organization | Ask first | — | — | Removes a draft return inside Trenova that nothing was filed from; generating it again recomputes the same figures. |
| Delete fuel purchase (`fuel_delete_purchase`) | Inside the organization | Propose | — | — | Removes a tax record the IFTA return is computed from, and nothing brings it back but entering it again, so a person always decides. |
| Discard fuel purchase import (`fuel_discard_purchase_import`) | Inside the organization | Ask first | — | — | Closes a staged statement inside Trenova without adding any purchase; it can be staged again. |
| Generate IFTA return (`fuel_generate_ifta_return`) | Inside the organization | Ask first | — | — | Computes a draft return inside Trenova; nothing is filed or sent, and fuel_delete_ifta_return removes the draft. |
| Recalculate IFTA return (`fuel_recalculate_ifta_return`) | Inside the organization | Automatic | — | — | Re-derives a draft return's figures inside Trenova from what is on file; nothing is filed, and running it again gives the same figures. |
| Recalculate fuel jurisdiction miles (`fuel_recalculate_jurisdiction_miles`) | Inside the organization | Ask first | — | — | Re-derives one move's jurisdiction miles inside Trenova from its stops; nothing is sent to a customer or carrier, and running it again gives the same miles. |
| Record IFTA mileage entry (`fuel_record_ifta_mileage_entry`) | Inside the organization | Ask first | Miles read from a log or a message someone outside sent are proposed, since the IFTA return is computed from them. | — | Adds jurisdiction miles inside Trenova that the next IFTA return counts; nothing is sent, and fuel_delete_ifta_mileage_entry removes them. |
| Record fuel purchase (`fuel_record_purchase`) | Inside the organization | Ask first | A purchase read from a receipt or statement someone outside sent is proposed, since it is a tax record the IFTA return is computed from. | — | Adds a fuel purchase inside Trenova that the next IFTA return counts; nothing is sent, and fuel_delete_purchase removes it. |
| Resolve fuel purchase import rows (`fuel_resolve_purchase_import_rows`) | Inside the organization | Ask first | — | — | Re-matches held statement rows inside Trenova and, for a feed, posts the purchases that now resolve; nothing is sent. |
| Assign inbound message (`inbox_assign_inbound_message`) | Inside the organization | Ask first | — | — | Changes who owns messages in the Triage queue inside Trenova; nothing is sent and the messages themselves are unchanged. |
| Assign inbox queue (`inbox_assign_queue`) | Inside the organization | Ask first | — | — | Moves the whole waiting queue to one person inside Trenova; each message can be reassigned, and nothing is sent. |
| Await inbox reply (`inbox_await_reply`) | Inside the organization | Ask first | — | — | Hides a message from the Triage queue until its sender replies or a time comes; nothing is sent. |
| Block inbox sender (`inbox_block_sender`) | Inside the organization | Ask first | — | — | Silently drops an outside sender's future mail, which can hide a customer or carrier from everyone, so a person always approves it. |
| Create inbox load (`inbox_create_load`) | Inside the organization | Ask first | — | — | A new load commits a customer's freight and the money that follows, so no desk books one unattended. |
| Discard capture batch (`inbox_discard_capture_batch`) | Inside the organization | Propose | — | — | Closes a scanned stack inside Trenova and drops its unfiled pages; nothing is sent, but nothing brings the pages back, so it is always proposed. |
| Discard inbox capture item (`inbox_discard_capture_item`) | Inside the organization | Ask first | — | — | Marks one scanned document not worth filing inside Trenova; nothing is sent, and its pages are removed with the stack's other unfiled pages. |
| Discard inbox load (`inbox_discard_load`) | Inside the organization | Automatic | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Removes a draft from the message inside Trenova; it can be read again and nothing is booked or sent. |
| Discard inbox reply draft (`inbox_discard_reply_draft`) | Inside the organization | Ask first | — | — | Deletes a draft that may be a person's own unsent words, with nothing to bring it back. |
| Draft inbox load (`inbox_draft_load`) | Inside the organization | Automatic | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Writes a draft on the message only people inside the organization see; nothing is booked. |
| Draft inbox reply (`inbox_draft_reply`) | Inside the organization | Automatic | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Writes a draft only people inside the organization see; sending it stays with a person. |
| File inbox attachment (`inbox_file_attachment`) | Inside the organization | Automatic | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Files a document already in Trenova on a shipment; nobody outside is told and it can be unfiled. |
| File inbox capture item (`inbox_file_capture_item`) | Inside the organization | Ask first | — | — | Files scanned pages as documents on records inside Trenova, as the person it acts for; nothing is sent, and a misfiled document can be deleted. |
| Link inbound message (`inbox_link_inbound_message`) | Inside the organization | Automatic | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Links an inbound message to a record inside Trenova; the mailbox decides how far it may run. |
| Mark inbound message (`inbox_mark_inbound_message`) | Inside the organization | Automatic | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Settles an inbound message inside Trenova; the mailbox decides how far it may run. |
| Record inbox discrepancy (`inbox_record_discrepancy`) | Inside the organization | Automatic | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Places a billing hold on a shipment inside Trenova; nothing is sent and a person can release it. |
| Reopen inbox conflict (`inbox_reopen_conflict`) | Inside the organization | Automatic | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Takes back how a conflict on an inbound message was settled; nothing filed or booked changes. |
| Reopen inbound message (`inbox_reopen_inbound_message`) | Inside the organization | Ask first | — | — | Puts settled messages back in the queue inside Trenova; nothing is sent. |
| Resolve inbox conflict (`inbox_resolve_conflict`) | Inside the organization | Ask first | — | — | Decides whether a document on file is replaced or a load is a duplicate, which later filing and booking follow, so a person approves it. |
| Snooze inbound message (`inbox_snooze_inbound_message`) | Inside the organization | Ask first | — | — | Hides messages from the Triage queue until a time or an event; they return on their own and nothing is sent. |
| Unblock inbox sender (`inbox_unblock_sender`) | Inside the organization | Ask first | — | — | Lets an outside sender's mail reach people again; it changes which mail the organization reads, so a person approves it. |
| Unfile inbox attachment (`inbox_unfile_attachment`) | Inside the organization | Ask first | — | — | Takes a document off a shipment inside Trenova and deletes the documents a split made, so a person approves it. |
| Unlink inbound message (`inbox_unlink_inbound_message`) | Inside the organization | Automatic | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Removes a link from an inbound message inside Trenova; the mailbox decides how far it may run. |
| Wake inbound message (`inbox_wake_inbound_message`) | Inside the organization | Automatic | — | — | Puts a parked message back in front of the people working Triage; nothing is sent. |
| Dismiss insight (`insight_dismiss`) | Inside the organization | Automatic | — | — | Dismisses an insight inside Trenova; it can be restored. |
| Restore insight (`insight_restore`) | Inside the organization | Automatic | — | — | Returns a finding to the active list inside Trenova; nothing is sent and insight_dismiss takes it off again. |
| Adjust invoice run (`invoice_adjust_run`) | Inside the organization | Automatic | An exclusion read from what a customer sent is proposed, since it decides what they are billed this period. | — | Edits a proposal of invoices before anything is invoiced; each change is undone by the opposite edit. |
| Build invoice run (`invoice_build_run`) | Inside the organization | Automatic | — | — | Builds a proposal of invoices inside Trenova for a biller to review; it invoices nothing and is discarded with invoice_cancel_run. |
| Cancel invoice run (`invoice_cancel_run`) | Inside the organization | Ask first | — | — | Discards a proposal of invoices; nothing was invoiced, and a new run is built with invoice_build_run. |
| Generate invoice PDF (`invoice_generate_pdf`) | Inside the organization | Automatic | — | — | Renders the invoice as it stands into its filed PDF; nothing is sent, and it is refused whenever rendering would email the customer. |
| Open invoice dispute (`invoice_open_dispute`) | Inside the organization | Ask first | A dispute opened from what a customer wrote is proposed, since the notes and the amount are what collections acts on. | — | Marks an invoice Disputed inside Trenova with the customer's reason; it moves no money and is withdrawn by invoice_withdraw_dispute. |
| Reject invoice adjustment (`invoice_reject_adjustment`) | Inside the organization | Propose | — | — | Turns down an adjustment an approver was asked to sign off; the approval decision is a person's. |
| Resolve invoice dispute (`invoice_resolve_dispute`) | Inside the organization | Propose | — | — | Records the outcome of a customer's dispute and releases the invoice back to collections; the outcome is a biller's call, so the agent proposes it. |
| Save invoice adjustment draft (`invoice_save_adjustment_draft`) | Inside the organization | Automatic | A draft written from what a customer sent is proposed, since its reason and amounts are what a biller submits. | — | Saves a draft adjustment inside Trenova; it credits nothing until a person submits it, and a later save rewrites it. |
| Share invoice (`invoice_share`) | Inside the organization | Propose | — | — | Notifies and emails colleagues in the name of the person who shares it, with a note they read as theirs; only that person sends it. |
| Update invoice draft (`invoice_update_draft`) | Inside the organization | Ask first | Changing who the invoice is emailed to is a proposal a person decides; any other change runs once a person approves it. Outside content could put its own wording or recipients on an invoice the customer receives. | — | Edits a draft nobody outside has seen; the customer sees it only when a person sends it, but the recipients decide where it goes then. |
| Withdraw invoice dispute (`invoice_withdraw_dispute`) | Inside the organization | Ask first | A withdrawal read from what a customer wrote is proposed, since it sends the invoice back to collections. | — | Closes a dispute case inside Trenova with no outcome; it moves no money and a new case can be opened with invoice_open_dispute. |
| Cancel journal reversal (`ledger_cancel_journal_reversal`) | Inside the organization | Ask first | The reason is what the reversal's requester reads next, so a run that has read outside text proposes it. | — | Withdraws a reversal before it reaches the ledger; nothing is booked, and it is requested again if it was needed after all. |
| Cancel manual journal (`ledger_cancel_manual_journal`) | Inside the organization | Ask first | The reason is what the journal's author reads next, so a run that has read outside text proposes it. | — | Withdraws a journal before it reaches the ledger; nothing is booked, and the entry is drafted again if it was needed after all. |
| Create manual journal (`ledger_create_manual_journal`) | Inside the organization | Ask first | A journal drafted from what someone outside wrote is proposed, since its lines are what an approver books. | — | Saves a draft journal inside Trenova; nothing reaches the ledger until a person submits, approves and posts it, and a draft is cancelled the same way. |
| Submit manual journal (`ledger_submit_manual_journal`) | Inside the organization | Ask first | — | — | Moves a draft journal into the approval queue inside Trenova; an approver decides and it books nothing until a person posts it. |
| Update manual journal (`ledger_update_manual_journal`) | Inside the organization | Ask first | A journal rewritten from what someone outside wrote is proposed, since its lines are what an approver books. | — | Changes a draft journal inside Trenova; nothing reaches the ledger until a person submits, approves and posts it, and a later revision changes it back. |
| Create location (`location_create`) | Inside the organization | Ask first | — | — | A new location is where colleagues will book freight, and its address is usually read from a document someone outside sent, so a person approves it first. |
| Set location status (`location_set_status`) | Inside the organization | Ask first | — | — | Changes whether locations are offered for booking inside Trenova; nothing is sent. |
| Update location (`location_update`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Changes a location inside Trenova; nothing is sent. Its address is often read from a document someone outside sent, so such a change is proposed. |
| Create commodity (`masterdata_create_commodity`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Adds a commodity inside Trenova that shipments can name; nothing is sent and it can be made inactive. |
| Create hazardous material (`masterdata_create_hazardous_material`) | Inside the organization | Propose | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Adds a hazardous material inside Trenova; nothing is sent. Its class, placarding and emergency contact decide what drivers carry and post, so a person checks every one against the DOT table before it is saved. |
| Set commodity status (`masterdata_set_commodity_status`) | Inside the organization | Ask first | — | — | Changes whether commodities are offered for booking inside Trenova; nothing is sent. |
| Set hazardous material status (`masterdata_set_hazardous_material_status`) | Inside the organization | Ask first | — | — | Changes whether hazardous materials are offered inside Trenova; nothing is sent. |
| Update commodity (`masterdata_update_commodity`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Changes a commodity inside Trenova; nothing is sent, and a later change puts it back. |
| Update hazardous material (`masterdata_update_hazardous_material`) | Inside the organization | Propose | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Changes a hazardous material inside Trenova; nothing is sent. What drivers placard and who they call follows it, so a person approves every change. |
| Attach order shipments (`order_attach_shipments`) | Inside the organization | Ask first | — | — | Regroups shipments under an order inside Trenova; nothing is sent, and order_detach_shipment moves one back out. |
| Close order (`order_close`) | Inside the organization | Propose | — | — | Settles an order for good; nothing reopens it, so only a person closes one. |
| Create order (`order_create`) | Inside the organization | Ask first | — | — | Opens an empty order inside Trenova; nothing is billed or sent, and an order opened in error is canceled. |
| Detach order shipment (`order_detach_shipment`) | Inside the organization | Ask first | — | — | Moves a shipment onto a new order inside Trenova; nothing is sent, and order_attach_shipments puts it back. |
| Create rate agreement (`rate_create_agreement`) | Inside the organization | Ask first | An agreement drafted from a rate sheet or a message someone outside sent is proposed, since its lanes are what the organization will charge. | — | Saves a draft agreement inside Trenova that prices nothing until a person approves it; rate_archive_agreement retires it. |
| Discard rate import (`rate_discard_import`) | Inside the organization | Ask first | — | — | Closes a staged rate sheet inside Trenova without changing any rate; the sheet can be uploaded again. |
| Duplicate rate agreement (`rate_duplicate_agreement`) | Inside the organization | Ask first | An agreement drafted from a rate sheet or a message someone outside sent is proposed, since its lanes are what the organization will charge. | — | Saves a draft copy of an agreement inside Trenova that prices nothing until a person approves it. |
| Generate rate confirmation (`rate_generate_confirmation`) | Inside the organization | Automatic | A first revision is generated as far as the agent allows; one that would void a revision already standing, which the carrier may hold or have signed, is a proposal a person decides, as is one the service would refuse or that cannot be read. | — | Renders the agreement and files it on the shipment inside Trenova; nothing reaches the carrier until it is sent, and voiding the revision undoes it. |
| Reject rate agreement (`rate_reject_agreement`) | Inside the organization | Propose | — | — | Returns an agreement to its author with the reviewer's reason; it prices nothing either way, but the review is a person's decision. |
| Run rate simulation (`rate_run_simulation`) | Inside the organization | Automatic | — | — | Saves a simulation inside Trenova and replays past shipments in the background; no shipment, rate or invoice changes. |
| Submit rate agreement (`rate_submit_agreement`) | Inside the organization | Propose | — | — | Moves a draft agreement into review inside Trenova; it prices nothing until a person approves it. |
| Update rate agreement draft (`rate_update_agreement_draft`) | Inside the organization | Ask first | An agreement drafted from a rate sheet or a message someone outside sent is proposed, since its lanes are what the organization will charge. | — | Changes a draft agreement inside Trenova that prices nothing yet; a later revision changes it back. |
| Assign receivable collector (`receivable_assign_collector`) | Inside the organization | Ask first | — | — | Changes who chases a customer inside Trenova and tells the new collector; nothing is sent to the customer and assigning someone else changes it back. |
| Record receivable collection activity (`receivable_record_collection_activity`) | Inside the organization | Automatic | What a customer said is what a collector acts on next, and a promise to pay holds back reminders and moves the follow-up, so a run that has read outside text proposes it rather than writing it. | — | Adds an entry to a customer's collections history inside Trenova; a promise moves the follow-up and holds back reminders until its date, and nothing is sent. |
| Resolve bank receipt work item (`receivable_resolve_bank_receipt_work_item`) | Inside the organization | Automatic | — | — | Closes a reconciliation work item without moving money; the note is read inside the organization. |
| Schedule receivable follow up (`receivable_schedule_follow_up`) | Inside the organization | Automatic | What a customer said is what a collector acts on next, and a promise to pay holds back reminders and moves the follow-up, so a run that has read outside text proposes it rather than writing it. | — | Changes when a customer is next chased inside Trenova and notes why; nothing is sent and another follow-up replaces it. |
| Triage bank receipt work item (`receivable_triage_bank_receipt_work_item`) | Inside the organization | Automatic | — | — | Changes who works a reconciliation item and says it is being looked into; it moves no money and the item is reassigned the same way. |
| Add dashboard tile (`report_add_dashboard_tile`) | Inside the organization | Automatic | — | — | Adds a tile to a saved dashboard inside Trenova. |
| Cancel report run (`report_cancel_run`) | Inside the organization | Automatic | — | — | Stops a report the person asked for from running; nothing is changed or sent, and running the report again gives the same rows. |
| Create report (`report_create`) | The caller's own records, inside the organization | Automatic | Each call is classified by what it reaches. A call on the caller's own records runs unasked while they are present. | — | A private report is a saved query on the caller's own list; a shared one appears on every colleague's Reports page and waits for approval. |
| Create dashboard (`report_create_dashboard`) | Inside the organization | Automatic | — | — | Saves a report dashboard colleagues can open; nothing leaves the organization. |
| Create table change alert (`report_create_table_change_alert`) | Inside the organization | Automatic | — | — | Creates an alert whose notices go to people inside the organization. |
| Delete report (`report_delete`) | Inside the organization | Propose | — | — | Removes a saved report colleagues may run and nothing brings it back, so a person always decides. |
| Delete dashboard (`report_delete_dashboard`) | Inside the organization | Propose | — | — | Removes a dashboard colleagues may open and nothing brings it back, so a person always decides. |
| Delete report schedule (`report_delete_schedule`) | Inside the organization | Propose | — | — | Stops a report reaching the people it was scheduled for; nothing brings the schedule back but setting it up again, so a person always decides. |
| Delete table change alert (`report_delete_table_change_alert`) | Inside the organization | Ask first | — | — | Removes an alert whose notices went to the person who owns it; report_create_table_change_alert sets it up again. |
| Fork report (`report_fork`) | Inside the organization | Automatic | — | — | Saves a copy of a report inside Trenova; nothing leaves the organization. |
| Reset report fork (`report_reset_fork`) | Inside the organization | Ask first | — | — | Rewrites a report the person owns from the built-in catalog; nothing is sent, but the person's changes to it are lost. |
| Save report table view (`report_save_table_view`) | The caller's own records, inside the organization | Automatic | Each call is classified by what it reaches. A call on the caller's own records runs unasked while they are present. | — | A private view is the caller's own picker entry; a shared one appears for every colleague, and nothing leaves the organization. |
| Set table change alert status (`report_set_table_change_alert_status`) | Inside the organization | Automatic | — | — | Pauses or resumes an alert whose notices go to the person who owns it; the opposite call undoes it. |
| Update report (`report_update`) | Inside the organization | Automatic | — | — | Changes a saved report colleagues may open; nothing leaves the organization. |
| Update table change alert (`report_update_table_change_alert`) | Inside the organization | Automatic | — | — | Changes an alert whose notices go to the person who owns it; nothing else changes. |
| Attach settlement pay events (`settlement_attach_pay_events`) | Inside the organization | Automatic | — | — | Moves pay a driver already earned between the pool and a draft settlement inside Trenova; nothing is paid until a person approves it. |
| Detach settlement pay event (`settlement_detach_pay_event`) | Inside the organization | Automatic | — | — | Moves pay a driver already earned between the pool and a draft settlement inside Trenova; nothing is paid until a person approves it. |
| Generate carrier settlement batch (`settlement_generate_carrier_batch`) | Inside the organization | Automatic | — | — | Drafts the period's carrier settlements from cost already accrued inside Trenova; nothing is paid until a person approves and posts them. |
| Generate driver settlement (`settlement_generate_driver`) | Inside the organization | Automatic | — | — | Drafts a settlement from pay already accrued inside Trenova; nothing is paid until a person approves it, and hands-off approval is only the settlement control's clean-settlement rule. |
| Generate driver settlement batch (`settlement_generate_driver_batch`) | Inside the organization | Automatic | — | — | Drafts the period's settlements from pay already accrued inside Trenova; nothing is paid until a person approves them, and hands-off approval is only the settlement control's clean-settlement rule. |
| Recalculate carrier settlement (`settlement_recalculate_carrier`) | Inside the organization | Automatic | — | — | Rebuilds a draft from the records it is computed from inside Trenova; nothing is paid until a person approves it. |
| Recalculate driver settlement (`settlement_recalculate_driver`) | Inside the organization | Automatic | — | — | Rebuilds a draft from the records it is computed from inside Trenova; nothing is paid until a person approves it. |
| Reject carrier settlement (`settlement_reject_carrier`) | Inside the organization | Automatic | Its text is what payroll or the payee reads next, so a run that has read outside text proposes it rather than writing it. | — | Returns a settlement to draft with a note inside Trenova; nothing is paid and it is submitted again once fixed. |
| Reject driver settlement (`settlement_reject_driver`) | Inside the organization | Automatic | Its text is what payroll or the payee reads next, so a run that has read outside text proposes it rather than writing it. | — | Returns a settlement to draft with a note inside Trenova; nothing is paid and it is submitted again once fixed. |
| Release settlement pay event (`settlement_release_pay_event`) | Inside the organization | Automatic | — | — | Returns held pay to the settlement pool inside Trenova; nothing is paid until a settlement is approved, and the pay is held again the same way. |
| Submit carrier settlement (`settlement_submit_carrier`) | Inside the organization | Automatic | — | — | Moves a draft into the approval queue inside Trenova; nothing is paid and a reviewer sends it back to draft. |
| Submit driver settlement (`settlement_submit_driver`) | Inside the organization | Automatic | — | — | Moves a draft into the approval queue inside Trenova; nothing is paid and a reviewer sends it back to draft. |
| Attach shipment document (`shipment_attach_document`) | Inside the organization | Automatic | — | — | Files a document already in Trenova against a shipment; nobody outside is told. |
| Create shipment (`shipment_create`) | Inside the organization | Ask first | — | — | A new load commits a customer's freight and the money that follows, so no desk books one unattended. |
| Create shipment recurring (`shipment_create_recurring`) | Inside the organization | Ask first | — | — | Sets up a schedule inside Trenova that creates shipments later; nothing is sent, and pausing the series stops it. |
| Delete shipment comment (`shipment_delete_comment`) | Inside the organization | Ask first | — | — | Removes a note from a shipment's own thread inside Trenova; the text is gone, so a person confirms it. |
| Duplicate shipment (`shipment_duplicate`) | Inside the organization | Ask first | — | — | Creates new shipments inside Trenova from one already saved; nothing is sent, and a copy made in error is canceled. |
| Generate shipment recurring (`shipment_generate_recurring`) | Inside the organization | Ask first | — | — | Creates one shipment inside Trenova from the series' source; nothing is sent, and a shipment made in error is canceled. |
| Pin shipment comment (`shipment_pin_comment`) | Inside the organization | Automatic | — | — | Orders a shipment's own comment thread inside Trenova; unpinning undoes it and nothing is sent. |
| Place shipment hold (`shipment_place_hold`) | Inside the organization | Automatic | — | — | Places a hold on a shipment inside Trenova; no customer or EDI notice is sent. |
| Recalculate shipment distance (`shipment_recalculate_distance`) | Inside the organization | Automatic | — | — | Re-derives the shipment's move distances inside Trenova from its stops; nothing is sent and running it again gives the same miles. |
| Record shipment permit (`shipment_record_permit`) | Inside the organization | Ask first | — | — | Records a permit on the shipment inside Trenova, which can release a dispatch hold; nothing is sent to the state, and shipment_update_permit voids it. |
| Release shipment hold (`shipment_release_hold`) | Inside the organization | Automatic | — | — | Releases a hold on a shipment inside Trenova; no customer or EDI notice is sent. |
| Resolve shipment comment (`shipment_resolve_comment`) | Inside the organization | Automatic | — | — | Marks a shipment's own comment settled inside Trenova; reopening undoes it and nothing is sent. |
| Set shipment recurring status (`shipment_set_recurring_status`) | Inside the organization | Ask first | — | — | Starts or stops a schedule inside Trenova; nothing is sent, and the status is set back the same way. |
| Transfer shipment ownership (`shipment_transfer_ownership`) | Inside the organization | Ask first | — | — | Changes who owns a shipment inside Trenova; nothing is sent, and it is transferred back the same way. |
| Uncancel shipment (`shipment_uncancel`) | Inside the organization | Ask first | — | — | Puts a canceled shipment back to New inside Trenova; nothing is sent, and canceling it again undoes it. |
| Unpin shipment comment (`shipment_unpin_comment`) | Inside the organization | Automatic | — | — | Orders a shipment's own comment thread inside Trenova; pinning again undoes it and nothing is sent. |
| Update shipment permit (`shipment_update_permit`) | Inside the organization | Ask first | — | — | Changes a permit on the shipment inside Trenova, which can place or lift a dispatch hold; nothing is sent to the state, and it is changed again the same way. |
| Update shipment recurring (`shipment_update_recurring`) | Inside the organization | Ask first | — | — | Changes when a series creates shipments inside Trenova; nothing is sent, and the old schedule is set back the same way. |
| Stop web watch (`web_stop_watch`) | Inside the organization | Automatic | — | Carries outside text into later runs | Stops a scheduled search the person set; it changes no record and reaches nobody. |
| Watch web (`web_watch`) | Inside the organization | Automatic | — | Carries outside text into later runs | Sets a scheduled search on the organization's web research vendor for the person; it changes no record, reaches nobody, and ends on its own or when stopped. |
| Approve worker PTO (`worker_approve_pto`) | Inside the organization | Automatic | — | — | Books approved time off; the worker is told it was approved but reads no text the model wrote. |
| Archive worker credential (`worker_archive_credential`) | Inside the organization | Propose | — | — | Retires a credential inside Trenova, which can take the worker off dispatch; the archived row stays on file, and a new credential is recorded to replace it. |
| Assign worker shift (`worker_assign_shift`) | Inside the organization | Ask first | — | — | Changes a worker's standing schedule inside Trenova; nothing is sent, and worker_end_shift_assignment or a new assignment changes it back. |
| Attach worker credential document (`worker_attach_credential_document`) | Inside the organization | Ask first | — | — | Links a filed document to a credential inside Trenova; attaching another replaces it. |
| Attach worker training document (`worker_attach_training_document`) | Inside the organization | Ask first | — | — | Links a filed document to a training record inside Trenova; attaching another replaces it. |
| Cancel worker checklist (`worker_cancel_checklist`) | Inside the organization | Propose | — | — | Cancels a checklist inside Trenova; nothing is sent, and a cancelled checklist is started again from its template. |
| Cancel worker DOT random draw (`worker_cancel_dot_random_draw`) | Inside the organization | Propose | — | — | Voids a round so the period is drawn again; the cancelled round and its reason stay on the record, and it cannot be undone. |
| Cancel worker DOT test (`worker_cancel_dot_test`) | Inside the organization | Propose | — | — | Voids a scheduled collection; the row and its reason stay on the record an auditor reads, and it cannot be undone. |
| Close worker leave case (`worker_close_leave_case`) | Inside the organization | Propose | — | — | Closes a leave case under the leave approval grant, so it runs as the person who approves it; the days it drew down stay drawn down. |
| Close worker training (`worker_close_training`) | Inside the organization | Propose | — | — | Closes a training assignment inside Trenova; the reason stays on the record, and the course is assigned again the same way. |
| Delete worker employment verification (`worker_delete_employment_verification`) | Inside the organization | Propose | — | — | Removes a previous employer from the qualification file; the audit trail keeps what was removed. |
| Delete worker injury (`worker_delete_injury`) | Inside the organization | Propose | — | — | Removes a case from the OSHA log; the audit trail keeps what was removed, and the number is not given out again. |
| Delete worker leave day (`worker_delete_leave_day`) | Inside the organization | Propose | — | — | Removes a recorded day of leave; the audit trail keeps what was removed, and worker_record_leave_day records it again. |
| Delete performance review (`worker_delete_performance_review`) | Inside the organization | Propose | — | — | Removes a draft review nobody has seen; the audit trail keeps what was removed. |
| Delete worker recognition (`worker_delete_recognition`) | Inside the organization | Propose | — | — | Removes a recognition from a driver's record; it is given again the same way. |
| Delete worker safety event (`worker_delete_safety_event`) | Inside the organization | Propose | — | — | Removes an event recorded in error from a driver's safety record; the audit trail keeps what was deleted, and it cannot be put back. |
| Delete worker safety violation (`worker_delete_safety_violation`) | Inside the organization | Propose | — | — | Removes a cited violation inside Trenova; the audit trail keeps what was removed, and it is cited again with worker_record_safety_violation. |
| End worker shift assignment (`worker_end_shift_assignment`) | Inside the organization | Ask first | — | — | Ends a worker's standing schedule inside Trenova; nothing is sent, and worker_assign_shift puts them back on it. |
| Finalize worker DOT random draw (`worker_finalize_dot_random_draw`) | Inside the organization | Propose | — | — | Locks the round's selections as the record an auditor reads; it cannot be undone, only cancelled, so a person approves it. |
| Open worker leave case (`worker_open_leave_case`) | Inside the organization | Ask first | — | — | Files a pending leave request inside Trenova; nothing is decided or sent, and the case is corrected with worker_update_leave_case. |
| Open worker safety event (`worker_open_safety_event`) | Inside the organization | Ask first | — | — | Adds an event to a driver's safety record inside Trenova; nothing is sent to the driver, and it is corrected or deleted while it is open. |
| Place worker dispatch hold (`worker_place_dispatch_hold`) | Inside the organization | Automatic | — | — | Keeps a driver off new freight inside Trenova; the driver is not messaged. |
| Record worker credential (`worker_record_credential`) | Inside the organization | Ask first | — | — | Adds a credential to the worker's file inside Trenova, which can change whether they may be dispatched; nothing is sent, and worker_archive_credential retires it. |
| Record worker employment verification (`worker_record_employment_verification`) | Inside the organization | Ask first | — | — | Adds a previous employer to the qualification file inside Trenova; nothing is sent to the employer, and it is corrected or removed the same way. |
| Record worker employment verification request (`worker_record_employment_verification_request`) | Inside the organization | Ask first | — | — | Stamps the request date or counts a follow-up inside Trenova; nothing is sent to the employer. |
| Record worker injury (`worker_record_injury`) | Inside the organization | Ask first | — | — | Adds a case to the organization's OSHA log inside Trenova; nothing is sent, and it is corrected with worker_update_injury. |
| Record worker leave day (`worker_record_leave_day`) | Inside the organization | Ask first | — | — | Records a day taken against a leave case inside Trenova; nothing is sent, and worker_update_leave_day or worker_delete_leave_day corrects it. |
| Record worker safety violation (`worker_record_safety_violation`) | Inside the organization | Ask first | — | — | Adds a cited violation to a safety event inside Trenova; nothing is sent, and it is corrected or removed the same way. |
| Record worker training completion (`worker_record_training_completion`) | Inside the organization | Ask first | — | — | Records a completion on the worker's training record inside Trenova; nothing is sent, and a completion recorded in error is corrected by a person. |
| Request worker leave certification (`worker_request_leave_certification`) | Inside the organization | Ask first | — | — | Starts the certification clock on a leave case inside Trenova; nothing is sent to the worker, and a later request restarts it. |
| Request shift swap (`worker_request_shift_swap`) | Inside the organization | Ask first | — | — | Opens a request inside Trenova that changes nobody's schedule until it is approved; worker_withdraw_shift_swap takes it back. |
| Run worker DOT random draw (`worker_run_dot_random_draw`) | Inside the organization | Ask first | — | — | Draws a draft round inside Trenova; nobody is told and nothing is final until a person finalizes it, and worker_cancel_dot_random_draw withdraws it. |
| Schedule worker DOT test (`worker_schedule_dot_test`) | Inside the organization | Ask first | — | — | Files a scheduled collection inside Trenova. The driver is sent nothing, no result is recorded, and worker_cancel_dot_test withdraws it. |
| Set worker availability preference (`worker_set_availability_preference`) | Inside the organization | Ask first | — | — | Records a worker's own stated preference inside Trenova; nothing is sent, and it is set again the same way. |
| Set worker safety event status (`worker_set_safety_event_status`) | Inside the organization | Ask first | — | — | Moves a safety event's status inside Trenova; nothing is sent, and the event is reopened or closed again the same way. |
| Start worker checklist (`worker_start_checklist`) | Inside the organization | Ask first | — | — | Opens a checklist on the worker inside Trenova; nothing is sent, and worker_cancel_checklist cancels it. |
| Start performance review (`worker_start_performance_review`) | Inside the organization | Ask first | — | — | Opens a draft review inside Trenova; the worker sees nothing, and worker_delete_performance_review removes the draft. |
| Update worker checklist item (`worker_update_checklist_item`) | Inside the organization | Ask first | — | — | Settles or reopens a checklist item inside Trenova; nothing is sent, and a settled item is reopened the same way. |
| Update worker credential (`worker_update_credential`) | Inside the organization | Ask first | — | — | Corrects a credential inside Trenova, which can change whether the worker may be dispatched; nothing is sent, and it is corrected again the same way. |
| Update worker DOT random selection (`worker_update_dot_random_selection`) | Inside the organization | Ask first | — | — | Updates a selection on a round inside Trenova; nothing is sent to the driver, and the selection is updated again the same way. |
| Update worker employment verification (`worker_update_employment_verification`) | Inside the organization | Ask first | — | — | Updates the previous employer's record in the qualification file inside Trenova; nothing is sent, and it is corrected again the same way. |
| Update worker injury (`worker_update_injury`) | Inside the organization | Ask first | — | — | Corrects a case on the OSHA log inside Trenova; nothing is sent, and it is corrected again the same way. |
| Update worker leave case (`worker_update_leave_case`) | Inside the organization | Ask first | — | — | Corrects a leave case inside Trenova; nothing is decided or sent, and it is corrected again the same way. |
| Update worker leave day (`worker_update_leave_day`) | Inside the organization | Ask first | — | — | Corrects a recorded day of leave inside Trenova; nothing is sent, and it is corrected again the same way. |
| Update performance review (`worker_update_performance_review`) | Inside the organization | Ask first | — | — | Saves the reviewer's draft inside Trenova; the worker sees nothing until the reviewer submits it, and the draft is edited again the same way. |
| Update worker safety event (`worker_update_safety_event`) | Inside the organization | Ask first | — | — | Corrects an event on a driver's safety record inside Trenova; nothing is sent, and the event is corrected again the same way. |
| Update worker safety violation (`worker_update_safety_violation`) | Inside the organization | Ask first | — | — | Corrects a cited violation inside Trenova; nothing is sent, and it is corrected again the same way. |

## Seen by a customer

Changes something a customer can see.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Revoke shipment tracking link (`shipment_revoke_tracking_link`) | Seen by a customer | Ask first | — | — | Ends the customer's access to the shipment's tracking page; a new link can be shared later. |
| Update shipment hold (`shipment_update_hold`) | Inside the organization, seen by a customer | Automatic | Each call is classified by what it reaches. | — | Changes a hold inside Trenova; one made visible to the customer is shown to them, so that call is held to the customer-visible ceiling. |

## Seen by a driver

Changes something a driver can see.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Hold settlement pay event (`settlement_hold_pay_event`) | Seen by a driver | Ask first | — | — | Defers a driver's pay and tells the driver why in the driver portal; it is released the same way. |
| Start settlement dispute review (`settlement_start_dispute_review`) | Seen by a driver | Ask first | — | — | Marks the dispute as being worked inside Trenova; the driver sees its status in Dash, and resolving it moves it on. |
| Add shipment comment (`shipment_add_comment`) | Inside the organization, seen by a customer, seen by a driver | Automatic | Each call is classified by what it reaches. | Carries outside text into later runs | An internal note stays inside the organization; a customer or driver note is read outside it, so its visibility argument decides. A note written on its own after the run read outside text is marked as drawn from it. |
| Update shipment comment (`shipment_update_comment`) | Inside the organization, seen by a customer, seen by a driver | Automatic | Each call is classified by what it reaches. | — | Edits a shipment's own comment inside Trenova; one made visible to a customer or a driver is read by them, so its visibility argument decides. |
| Approve shift swap (`worker_approve_shift_swap`) | Seen by a driver | Propose | — | — | Changes who works a day; approving is the office's decision, so only a person makes it. |
| Assign worker required training (`worker_assign_required_training`) | Seen by a driver | Ask first | — | — | Assigns the courses the worker's driver type requires; the driver sees them in Dash, and cancel_worker_training withdraws one. |
| Assign worker training (`worker_assign_training`) | Seen by a driver | Ask first | — | — | Assigns courses the driver sees in Dash; cancel_worker_training withdraws one. |
| Cancel worker PTO (`worker_cancel_pto`) | Seen by a driver | Ask first | — | — | The worker is sent the cancellation reason by push notice and text message. |
| Record worker recognition (`worker_record_recognition`) | Seen by a driver | Ask first | — | — | The driver is shown the recognition and told of it in Dash when it is visible to them. |
| Reject worker PTO (`worker_reject_pto`) | Seen by a driver | Ask first | — | — | The worker is sent the rejection reason by push notice and text message. |
| Reject shift swap (`worker_reject_shift_swap`) | Seen by a driver | Ask first | — | — | Refuses a request the workers read in their portal; a person approves each refusal. |
| Request worker credential renewal (`worker_request_credential_renewal`) | Seen by a driver | Ask first | — | — | Sends a driver a renewal request the model wrote. |
| Request worker PTO (`worker_request_pto`) | Seen by a driver | Ask first | — | — | Files time off the driver sees in Dash; under a policy with no approval step it is booked and the driver told at once, and worker_cancel_pto withdraws it. |
| Send worker message (`worker_send_message`) | Seen by a driver | Ask first | — | — | Sends a driver a message the model wrote to their phone. |
| Update worker PTO (`worker_update_pto`) | Seen by a driver | Ask first | — | — | Changes a pending request the driver sees in Dash; it is changed again the same way. |
| Withdraw shift swap (`worker_withdraw_shift_swap`) | Seen by a driver | Ask first | — | — | Takes back a request the workers read in their portal; a person approves each withdrawal. |

## Sent outside the organization

Sends to someone outside the organization.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Create customer (`customer_create`) | Inside the organization, sent outside the organization | Ask first | Each call is classified by what it reaches. A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Adds a customer inside Trenova that shipments can be booked for; nothing is sent. Setting who receives status emails decides where later emails go, so that call is held for a person. |
| Send customer email (`customer_send_email`) | Sent outside the organization | Ask first | — | — | Emails the customer's contacts a message the model wrote. |
| Update customer (`customer_update`) | Inside the organization, sent outside the organization | Ask first | Each call is classified by what it reaches. A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Changes a customer inside Trenova and leaves its billing and email profiles alone; nothing is sent. Changing who receives status emails is held for a person. |
| Send detention notice (`detention_send_notice`) | Sent outside the organization | Ask first | — | — | Sends the customer a detention notice that starts a charge. |
| Cancel tender (`dispatch_cancel_tender`) | Sent outside the organization | Ask first | A tender waiting on review holds a carrier's acceptance, so withdrawing it is a proposal a person decides, as is one the service would refuse or that cannot be read; an active tender is withdrawn once a person approves it. | — | Withdraws the offers carriers outside the organization hold, whose answer links stop working; a withdrawn tender cannot be reopened, only tendered afresh. |
| Create tender (`dispatch_create_tender`) | Sent outside the organization | Ask first | — | — | Offers the load to carriers outside the organization. |
| Create tender from routing guide (`dispatch_create_tender_from_routing_guide`) | Sent outside the organization | Ask first | — | — | Offers the load to carriers outside the organization. |
| Revoke dispatch tracking link (`dispatch_revoke_tracking_link`) | Sent outside the organization | Ask first | — | — | Ends a link someone outside the organization holds, so the carrier's page stops working; a new link can be issued later. |
| Share dispatch tracking link (`dispatch_share_tracking_link`) | Sent outside the organization | Ask first | — | — | Emails a link outside the organization that lets the carrier post to the load, and ends any link the carrier holds now; dispatch_revoke_tracking_link ends it. |
| Accept EDI tender (`edi_accept_tender`) | Sent outside the organization | Propose | — | — | Commits the organization to haul the load, creates the shipment and sends the partner an acceptance they act on; it cannot be taken back. A person always decides. |
| Cancel EDI tender (`edi_cancel_tender`) | Sent outside the organization | Propose | — | — | Withdraws freight offered to another organization, which sees the tender canceled; a withdrawn tender is sent afresh, never reopened. A person always decides. |
| Decline EDI tender (`edi_decline_tender`) | Sent outside the organization | Propose | — | — | Turns down freight a partner offered and sends them the decline; the partner moves the load elsewhere and it cannot be taken back. A person always decides. |
| Expire EDI tender (`edi_expire_tender`) | Sent outside the organization | Propose | — | — | Closes a tender on both sides of the trading relationship, which the other side sees; an expired tender is sent afresh, never reopened. A person always decides. |
| Replay EDI message (`edi_replay_message`) | Sent outside the organization | Propose | — | — | Sends a trading partner a document it already has, which it may act on a second time; a document sent cannot be recalled. A person always decides. |
| Reprocess EDI inbound file (`edi_reprocess_inbound_file`) | Sent outside the organization | Propose | — | — | Turns what trading partners sent into tenders, status updates and invoices again, and sends each partner acknowledgments they act on. A person always decides. |
| Retry EDI message (`edi_retry_message`) | Sent outside the organization | Propose | — | — | Sends documents to trading partners outside the organization, who act on what they receive; a document sent cannot be recalled. A person always decides. |
| Review EDI tender change (`edi_review_tender_change`) | Sent outside the organization | Propose | — | — | Changes a load this organization committed to on another organization's word, and that organization sees the outcome. A person always decides. |
| Review EDI transfer change (`edi_review_transfer_change`) | Sent outside the organization | Propose | — | — | Changes a shipment on another organization's report, and that organization sees the outcome on its own shipment. A person always decides. |
| Send EDI status update (`edi_send_status_update`) | Sent outside the organization | Propose | — | — | Sends a trading partner a shipment status they act on and pass to their own customers; a document sent cannot be recalled. A person always decides. |
| Send EDI tender (`edi_send_tender`) | Sent outside the organization | Propose | — | — | Offers a load to another organization, which may accept it and haul it on the terms tendered; a tender is withdrawn, never recalled. A person always decides. |
| Resolve service failure (`exception_resolve_service_failure`) | Sent outside the organization | Ask first | — | — | Resolving a failure generates an EDI 214 to the customer's trading partner carrying the reason chosen. |
| Review service failure (`exception_review_service_failure`) | Sent outside the organization | Propose | — | — | Reviewing a failure is a person's attestation and may send an EDI 214 to the customer's trading partner; only a person approves it. |
| Void service failure (`exception_void_service_failure`) | Sent outside the organization | Propose | — | — | Voiding a failure is final and may send an EDI 214 to the customer's trading partner; only a person approves it. |
| Create factoring submission (`factoring_create_submission`) | Sent outside the organization | Propose | — | — | Sells receivables to a factor and sends them out of the organization; a person approves every schedule. |
| Request factoring document (`factoring_request_document`) | Sent outside the organization | Propose | — | — | Emails a driver, carrier or customer outside the organization. |
| Retry factoring submission (`factoring_retry_submission`) | Sent outside the organization | Ask first | — | — | Sends receivables out of the organization again; a person approves it. |
| Send factoring noa (`factoring_send_noa`) | Sent outside the organization | Propose | — | — | Tells a customer, in writing, to pay someone else; a person decides every one. |
| Send inbox reply (`inbox_send_reply`) | Sent outside the organization | Ask first | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Replies to whoever wrote in with text the model composed. |
| Send invoice (`invoice_send`) | Sent outside the organization | Propose | — | — | Emails customers their invoices; only a person sends them, is named as who sent each, and sees every recipient before approving. |
| Send invoice EDI (`invoice_send_edi`) | Sent outside the organization | Propose | — | — | Transmits the invoice to the customer's EDI trading partner, which cannot be called back; only a person sends it. |
| Cancel order (`order_cancel`) | Sent outside the organization | Propose | — | — | Cancels every live shipment on the order, which reaches carriers and trading partners; only a person cancels an order. |
| Send rate confirmation (`rate_send_confirmation`) | Sent outside the organization | Propose | — | — | Emails a binding agreement, with a link to sign it, to a carrier outside the organization; the recipients come from the carrier's record, and a sent email cannot be recalled, so a person decides every send. |
| Send receivable payment reminder (`receivable_send_payment_reminder`) | Sent outside the organization | Ask first | — | — | Emails the customer about what it owes, which cannot be called back; a person approves every reminder and sees the recipients and the email first. |
| Send receivable statement (`receivable_send_statement`) | Sent outside the organization | Ask first | — | — | Emails the customer its balance and open items, which cannot be called back; a person approves every statement and sees the recipients and the email first. |
| Create report schedule (`report_create_schedule`) | Sent outside the organization | Ask first | — | — | Emails a report on a schedule to whatever addresses the call names, which may be outside the organization. |
| Update report schedule (`report_update_schedule`) | Sent outside the organization | Ask first | — | — | Changes where and when a report is emailed, which may be to addresses outside the organization. |
| Cancel shipment (`shipment_cancel`) | Sent outside the organization | Ask first | — | — | Cancelling withdraws live tenders from carriers and sends the model's cancel reason to a linked partner over EDI. |
| Request shipment documents (`shipment_request_documents`) | Sent outside the organization | Ask first | — | — | Emails an outside party a request for paperwork in words the model wrote. |
| Reschedule shipment stop (`shipment_reschedule_stop`) | Sent outside the organization | Ask first | — | — | A changed appointment is sent as an EDI tender change to the trading partners the load was tendered to. |
| Share shipment tracking link (`shipment_share_tracking_link`) | Seen by a customer, sent outside the organization | Ask first | Each call is classified by what it reaches. | — | Gives the customer a link to the shipment's tracking page and can email it outside the organization; the link can be revoked. |
| Update shipment (`shipment_update`) | Sent outside the organization | Ask first | — | — | A changed shipment is sent as an EDI tender change to the trading partners it was tendered to. |
| Use web portal (`web_use_portal`) | Sent outside the organization | Propose | — | — | Acts on a third-party portal in the organization's name, which another company then relies on; what it books or submits is undone, if at all, on that portal. A person always decides. |

## Money

Moves or commits money. A call runs on its own only when its tool proves it is within bounds.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Apply accounting inbound change (`accounting_apply_inbound_change`) | Money | Ask first | — | — | Posts money in Trenova: a customer payment or a settlement payment, so a person approves each one. |
| Release accounting sync record (`accounting_release_sync_record`) | Money | Propose | — | — | Sends documents a review policy held for a person to the organization's books; only a person releases them. |
| Resolve accounting drift finding (`accounting_resolve_drift_finding`) | Money | Ask first | — | — | Changes money on one side of the books: it posts a memo, void or reversal in Trenova, or sends a document to the accounting system, so a person approves each fix. |
| Approve billing queue item (`billing_approve_queue_item`) | Money | Propose | — | — | Approving creates the invoices customers are billed on, so only a person approves, and may untick any of them. |
| Cancel billing queue item (`billing_cancel_queue_item`) | Money | Propose | — | — | Drops a charge from billing for good, so only a person decides; the agent proposes it. |
| Correct billing charge codes (`billing_correct_charge_codes`) | Money | Automatic | Runs on its own only while nothing has been invoiced from the billing queue item and every charge it adds, removes or changes, counted in full, is within the organization's money limits for agents; any other call waits for a person. | — | Rewrites the accessorial charges a customer will be invoiced, so it moves money. |
| Reassign billing charge (`billing_reassign_charge`) | Money | Propose | — | — | Moves what each customer is billed for a shipment and opens or cancels their queue items; only a person reassigns a charge. |
| Resolve billing issue (`billing_resolve_issue`) | Money, seen by a driver | Propose | Each call is classified by what it reaches. | — | Decides what a customer is billed for a flagged charge, or messages the driver for paperwork, so only a person settles it; billing_undo_issue takes it back. |
| Undo billing issue (`billing_undo_issue`) | Money | Propose | — | — | Puts back a charge a settlement removed or repriced, which changes what the customer is billed, so only a person decides. |
| Accept carrier invoice match (`carrier_accept_invoice_match`) | Money | Propose | — | — | Clears a carrier's invoice for payment; only a person approves what the organization pays. |
| Accept carrier invoice match with variance (`carrier_accept_invoice_match_with_variance`) | Money | Propose | — | — | Agrees to pay a carrier more or less than the load was expected to cost; only a person approves it. |
| Create carrier (`carrier_create`) | Inside the organization, money | Ask first | Each call is classified by what it reaches. A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Adds a carrier inside Trenova that colleagues can then book; nothing is sent. Payment and remit-to details decide where money goes, so a call that sets them is money and runs only on a person's approval. |
| Set carrier monitoring (`carrier_set_monitoring`) | Money | Ask first | — | — | Monitoring is billed per carrier by the provider; turning it off again undoes it, and nothing is sent to the carriers. |
| Update carrier (`carrier_update`) | Inside the organization, money | Ask first | Each call is classified by what it reaches. A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Changes a carrier inside Trenova and keeps its contacts, insurance and EDI channels; nothing is sent. A call that changes payment or remit-to details is money and runs only on a person's approval. |
| Vet carrier (`carrier_vet`) | Money | Ask first | — | — | Spends the organization's carrier intelligence budget on a lookup and records the result; nothing is sent to the carrier. |
| Vet customer broker (`customer_vet_broker`) | Money | Ask first | — | — | Spends the organization's carrier intelligence budget on a lookup and records the result; nothing is sent to the customer. |
| Approve detention (`detention_approve`) | Money | Propose | — | — | Releases a held detention charge onto the customer's invoice; only a person approves it. |
| Open detention dispute (`detention_open_dispute`) | Money | Ask first | — | — | Marks a detention charge disputed inside Trenova and holds it from billing; nothing is sent, but the hold is not undone by a tool. |
| Resolve detention dispute (`detention_resolve_dispute`) | Money | Propose | — | — | Gives up or keeps detention revenue the customer disputes, and may credit an invoice; only a person approves it. |
| Waive detention (`detention_waive`) | Money | Propose | — | — | Gives up detention revenue the organization would otherwise bill; only a person approves it. |
| Cancel carrier assignment (`dispatch_cancel_carrier_assignment`) | Money | Ask first | Taking off a carrier who has confirmed the rate breaks an executed agreement, so it is a proposal a person decides, as is one the service would refuse or that cannot be read; a carrier who has not confirmed comes off once a person approves it. | — | Releases the organization's commitment to pay an outside carrier and voids the carrier's rate confirmation, whose sign link stops working; covering the move again makes a new agreement the carrier has to confirm afresh. |
| Create carrier assignment (`dispatch_create_carrier_assignment`) | Money | Ask first | Replacing a carrier already on the move, or overriding the new carrier's insurance warning, is a proposal a person decides; any other assignment runs once a person approves it. | — | Commits the organization to pay an outside carrier the rate it names for the move; nothing is sent to the carrier, and canceling the assignment releases the move. |
| Record tender response (`dispatch_record_tender_response`) | Sent outside the organization, money | Ask first | Each call is classified by what it reaches. Recording an acceptance, a counter or a carrier backing out moves money or the carrier off the move, so it is a proposal a person decides; a decline runs once a person approves it. | — | An acceptance commits the organization to pay the carrier and sends them the rate confirmation; a decline sends the next carrier its offer; a carrier backing out voids its rate confirmation and offers the load to others. None is undone by recording another answer. |
| Set trailer location (`equipment_set_trailer_location`) | Money | Propose | — | — | Adds an empty move to the trailer's last shipment and re-rates it, which can change what that customer is charged, so it runs only on a person's approval. |
| Apply factoring remittance (`factoring_apply_remittance`) | Money | Propose | — | — | Books the factor's advances, fees, reserve releases and chargebacks to the general ledger; only a person's approval runs it. |
| Record factoring chargeback (`factoring_record_chargeback`) | Money | Propose | — | — | Books a repayment to the factor and reopens an invoice; only a person's approval runs it. |
| Record factoring funding (`factoring_record_funding`) | Money | Propose | — | — | Books the factor's advances and fees to the general ledger and settles invoices; only a person's approval runs it. |
| Release factoring reserve (`factoring_release_reserve`) | Money | Propose | — | — | Books the factor's reserve remittance to the general ledger; only a person's approval runs it. |
| Reverse factoring chargeback (`factoring_reverse_chargeback`) | Money | Propose | — | — | Books the opposite of a chargeback to the general ledger; only a person's approval runs it. |
| Approve invoice adjustment (`invoice_approve_adjustment`) | Money | Propose | — | — | Executes a credit, rebill or write-off an approver was asked to sign off; approving is a person's decision. |
| Bill invoice statement (`invoice_bill_statement`) | Money | Propose | — | — | Invoices a customer off their agreed cycle; only a person decides to bill early, and the reason stays on the run. |
| Commit invoice run (`invoice_commit_run`) | Money | Propose | — | — | Issues the invoices a run proposes and advances the customers' billing period; only a person commits a run. |
| Create invoice (`invoice_create`) | Money | Propose | — | — | Turns freight into receivables and takes it off the billing queue and any statement; only a person decides what is billed and when. |
| Create invoice memo (`invoice_create_memo`) | Money | Propose | — | — | Raises what a customer owes or is owed outside any shipment; only a person decides it, and it is never posted as it is made. |
| Post invoice (`invoice_post`) | Money | Propose | — | — | Books receivables to the ledger and queues each for the accounting system and the customer's EDI; only a person posts, and may untick any of them, and hands-off posting is the billing-control auto-post setting. |
| Submit invoice adjustment (`invoice_submit_adjustment`) | Money | Propose | — | — | Credits, rebills or writes off receivables and books the entries that say so; only a person submits an adjustment, and the adjustment policy may still hold it for an approver. |
| Void invoice (`invoice_void`) | Money | Propose | — | — | Takes an invoice out of circulation and reverses what it booked; it cannot be undone, so only a person voids one. |
| Close fiscal period (`ledger_close_fiscal_period`) | Money | Propose | — | — | Ends posting to a period of the books a person signs off; only a person closes one. |
| Create journal reversal (`ledger_create_journal_reversal`) | Money | Propose | — | — | Commits the organization to taking a posted entry back out of the ledger, approved at once where approval is off; only a person requests a reversal. |
| Lock fiscal period (`ledger_lock_fiscal_period`) | Money | Propose | — | — | Changes which dates the ledger takes postings for; only a person locks a period. |
| Open fiscal period (`ledger_open_fiscal_period`) | Money | Propose | — | — | Changes which dates the ledger takes postings for; only a person opens a period. |
| Post journal reversal (`ledger_post_journal_reversal`) | Money | Propose | — | — | Books the reversing entry to the general ledger; only a person posts a reversal. |
| Post manual journal (`ledger_post_manual_journal`) | Money | Propose | — | — | Books the journal's lines to the general ledger; only a person posts a manual journal. |
| Reopen fiscal period (`ledger_reopen_fiscal_period`) | Money | Propose | — | — | Lets postings land again in books a person closed; only a person reopens a period. |
| Unlock fiscal period (`ledger_unlock_fiscal_period`) | Money | Propose | — | — | Changes which dates the ledger takes postings for; only a person unlocks a period. |
| Add order charge (`order_add_charge`) | Money | Ask first | — | — | Adds to what the customer is billed; nothing is invoiced or sent, and order_remove_charge takes it off until it is invoiced. |
| Remove order charge (`order_remove_charge`) | Money | Ask first | — | — | Lowers what the customer is billed; nothing is invoiced or sent, and order_add_charge puts it back. |
| Set order charge allocations (`order_set_charge_allocations`) | Money | Ask first | — | — | Moves who is billed for a charge; nothing is invoiced or sent, and the split is set back the same way until it is invoiced. |
| Update order (`order_update`) | Inside the organization, money | Ask first | Each call is classified by what it reaches. | — | Edits an order's own fields inside Trenova; nothing is sent, and the old values are set back the same way. A changed amount is a money change. |
| Update order charge (`order_update_charge`) | Money | Ask first | — | — | Changes what the customer is billed; nothing is invoiced or sent, and the old amount is set back the same way until it is invoiced. |
| Amend rate agreement rules (`rate_amend_agreement_rules`) | Money | Propose | — | — | Changes the rates an agreement prices shipments at; only a person amends a live contract. |
| Apply rate increase (`rate_apply_increase`) | Money | Propose | — | — | Changes the rates many agreements charge customers or pay carriers; only a person applies a rate increase. |
| Approve rate agreement (`rate_approve_agreement`) | Money | Propose | — | — | Turns on an agreement that prices what customers are charged or carriers are paid; only a person approves one. |
| Archive rate agreement (`rate_archive_agreement`) | Money | Propose | — | — | Ends an agreement for good, which changes what customers are charged when it was active; only a person archives one. |
| Commit rate import (`rate_commit_import`) | Money | Propose | — | — | Changes the rates an agreement prices shipments at; only a person applies a rate sheet. |
| Correct fuel index price (`rate_correct_fuel_index_price`) | Money | Propose | — | — | Moves the price every fuel surcharge on this index is computed from, which changes what customers are charged; only a person sets it. |
| Record rate confirmation (`rate_record_confirmation`) | Money | Ask first | — | — | Makes the revision the executed agreement to pay the carrier and confirms their assignment, on the word of someone outside the organization; voiding it afterwards withdraws the agreement rather than restoring it. |
| Record fuel index price (`rate_record_fuel_index_price`) | Money | Propose | — | — | Moves the price every fuel surcharge on this index is computed from, which changes what customers are charged; only a person sets it. |
| Resume rate agreement (`rate_resume_agreement`) | Money | Propose | — | — | Turns an agreement back on so it prices shipments again; only a person resumes one. |
| Suspend rate agreement (`rate_suspend_agreement`) | Money | Propose | — | — | Stops an agreement pricing shipments, which changes what customers are charged; only a person suspends one. |
| Void rate confirmation (`rate_void_confirmation`) | Money | Ask first | A revision the carrier has been sent or has signed is voided only as a proposal a person decides, as is one that cannot be read; one never sent is voided once a person approves it. | — | Withdraws the organization's written agreement to pay a carrier, whose sign link stops working, and undoes a confirmation it carried; a voided revision cannot be restored, only replaced. |
| Apply receivable credit memo (`receivable_apply_credit_memo`) | Money | Propose | — | — | Uses a customer's credit to settle what they owe; only a person applies credit. |
| Apply customer payment (`receivable_apply_customer_payment`) | Money | Propose | — | — | Moves a customer's cash onto their invoices and books the entry that says so; only a person applies cash. |
| Assess receivable late charges (`receivable_assess_late_charges`) | Money | Propose | — | — | Bills customers for paying late; only a person assesses late charges, and the billing control may post the memos as they are raised. |
| Match bank receipt (`receivable_match_bank_receipt`) | Money | Automatic | Runs on its own only when the payment is the one posted payment with the receipt's exact amount and reference, and the amount is within the organization's money limits for agents; any other call waits for a person. The receipt it matches, if the run read it, does not hold it, because the receipt is read again from the organization's records; any other outside content does. | — | Matches a bank receipt to a posted payment, closing its reconciliation. |
| Post customer payment (`receivable_post_customer_payment`) | Money | Automatic | Runs on its own only when it records an unmatched bank receipt as the receipt has it (its day, reference and memo, by ACH), whose reference or memo names every invoice it pays, each an open invoice of the same customer paid to its exact balance, the whole payment is applied with nothing short-paid or left unapplied, and the amount is within the organization's money limits for agents; any other call waits for a person. The receipt it records, if the run read it, does not hold it, because the receipt is read again from the organization's records; any other outside content does. | — | Records a customer payment and applies it to invoices. |
| Reverse customer payment (`receivable_reverse_customer_payment`) | Money | Propose | — | — | Takes cash back off the books and reopens the invoices it paid; only a person reverses a payment. |
| Unapply receivable credit memo (`receivable_unapply_credit_memo`) | Money | Propose | — | — | Reopens an invoice's balance and restores a customer's credit; only a person moves credit. |
| Add carrier settlement adjustment (`settlement_add_carrier_adjustment`) | Money | Automatic | Runs on its own only while the settlement is not yet approved and the line is within the organization's money limits for agents; any other call waits for a person. Its text is what payroll or the payee reads next, so a run that has read outside text proposes it rather than writing it. | — | Changes what the carrier will be paid on a settlement a person still approves, so it moves money; the line is removed the same way. |
| Add driver settlement adjustment (`settlement_add_driver_adjustment`) | Money | Automatic | Runs on its own only while the settlement is not yet approved and the line is within the organization's money limits for agents; any other call waits for a person. Its text is what payroll or the payee reads next, so a run that has read outside text proposes it rather than writing it. | — | Changes what the driver will be paid on a settlement a person still approves, so it moves money; the line is removed the same way. |
| Adjust settlement escrow account (`settlement_adjust_escrow_account`) | Money | Propose | — | — | Moves money held in escrow for an owner-operator; only a person moves it. |
| Approve carrier settlement (`settlement_approve_carrier`) | Money | Propose | — | — | Commits the organization to what carriers are paid; only a person approves, and may untick any of them, and hands-off approval is the settlement control's rule. |
| Approve driver settlement (`settlement_approve_driver`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Commits the organization to what drivers are paid; only a person approves, and may untick any of them, and hands-off approval is the settlement control's rule. |
| Assign settlement pay profile (`settlement_assign_pay_profile`) | Money | Propose | — | — | Sets how a driver is paid from a date on; only a person decides someone's pay. |
| Close settlement escrow account (`settlement_close_escrow_account`) | Money | Propose | — | — | Refunds an owner-operator's escrow balance and closes the account for good; only a person closes it. |
| Create settlement recurring deduction (`settlement_create_recurring_deduction`) | Money | Propose | — | — | Sets money taken from or added to a driver's pay every settlement; only a person agrees it. |
| Create settlement recurring earning (`settlement_create_recurring_earning`) | Money | Propose | — | — | Sets money taken from or added to a driver's pay every settlement; only a person agrees it. |
| End settlement pay assignment (`settlement_end_pay_assignment`) | Money | Propose | — | — | Stops how a driver is paid from a date; only a person decides someone's pay. |
| Forgive settlement advance (`settlement_forgive_advance`) | Money | Propose | — | — | Forgives money a driver owes, which the organization absorbs; only a person writes it off. |
| Generate settlement payroll export (`settlement_generate_payroll_export`) | Money | Propose | — | — | Locks approved hours into a payroll run that pays people, so a person approves it and it runs as them; settlement_void_payroll_export takes it back. |
| Issue settlement advance (`settlement_issue_advance`) | Money | Propose | — | — | Records money handed to a driver that their pay then repays; only a person records it. |
| Open settlement escrow account (`settlement_open_escrow_account`) | Money | Propose | — | — | Sets escrow terms under an owner-operator's lease that their pay is then withheld against; only a person agrees them. |
| Pay settlement now (`settlement_pay_now`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Approves, posts and records a payment to a driver in one step; only a person pays someone. |
| Post carrier settlement (`settlement_post_carrier`) | Money | Propose | — | — | Books settlements' payables to the ledger and queues each for the accounting system; only a person posts, and may untick any of them. |
| Post driver settlement (`settlement_post_driver`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Books settlements' payables to the ledger and queues each for the accounting system; only a person posts, and may untick any of them. |
| Record carrier settlement payment (`settlement_record_carrier_payment`) | Money | Propose | — | — | Records a payment to the carrier and queues it for the accounting system; only a person says money went out. |
| Record driver settlement payment (`settlement_record_driver_payment`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Records a payment to the driver and queues it for the accounting system; only a person says money went out. |
| Remove carrier settlement adjustment (`settlement_remove_carrier_adjustment`) | Money | Automatic | Runs on its own only when an agent added the line, the settlement is not yet approved and the line is within the organization's money limits for agents; any other call waits for a person. | — | Changes what the carrier will be paid on a settlement a person still approves, so it moves money; the line is added back the same way. |
| Remove driver settlement adjustment (`settlement_remove_driver_adjustment`) | Money | Automatic | Runs on its own only when an agent added the line, the settlement is not yet approved and the line is within the organization's money limits for agents; any other call waits for a person. | — | Changes what the driver will be paid on a settlement a person still approves, so it moves money; the line is added back the same way. |
| Resolve settlement dispute (`settlement_resolve_dispute`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Decides a driver's pay complaint, tells the driver, and can add pay to a settlement, so a person approves it and it runs as them; a decided dispute stays decided. |
| Review driver expense (`settlement_review_driver_expense`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Pays or refuses a driver's out-of-pocket expense and tells the driver, so a person approves it and it runs as them; a reviewed expense stays reviewed. |
| Update settlement escrow account (`settlement_update_escrow_account`) | Money | Propose | — | — | Changes the escrow terms of an owner-operator's lease; only a person agrees them. |
| Update settlement recurring deduction (`settlement_update_recurring_deduction`) | Money | Propose | — | — | Changes money taken from or added to a driver's pay every settlement; only a person agrees it. |
| Update settlement recurring earning (`settlement_update_recurring_earning`) | Money | Propose | — | — | Changes money taken from or added to a driver's pay every settlement; only a person agrees it. |
| Void carrier settlement (`settlement_void_carrier`) | Money | Propose | — | — | Cancels a settlement for good and reverses any posting; only a person voids. |
| Void driver settlement (`settlement_void_driver`) | Money | Propose | — | — | Cancels a settlement for good and reverses any posting; only a person voids. |
| Void settlement payroll export (`settlement_void_payroll_export`) | Money | Propose | — | — | Takes back a payroll run that payroll may already have paid from, so a person approves it and it runs as them; a voided run stays voided. |
| Add shipment charge (`shipment_add_charge`) | Money | Ask first | — | — | Adds to what the customer is billed for the shipment. |
| Rerate shipment (`shipment_rerate`) | Money | Ask first | — | — | Changes what the customer will be billed for the shipment; nothing is invoiced or sent, and a person can edit the rate back. |
| Adjust worker PTO balance (`worker_adjust_pto_balance`) | Money | Propose | — | — | Changes how many paid days a worker holds, which the organization owes and may pay out, so a person approves it and it runs as them. |

