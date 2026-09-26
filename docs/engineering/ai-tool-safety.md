# AI tool safety

<!-- Generated from the tool policies in code by running, in services/tms:
     go generate ./internal/core/services/agenttoolpolicy/safetydoc/...
     Do not edit by hand. -->

What every tool an agent can call may do without a person, read from the
policies the tools declare in code. The runtime decides each call from the same
policies, so this page cannot drift from what runs: CI regenerates it and fails
when it differs. Each tool is listed once, under the furthest class its work
can reach.

Tools listed: 270.

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

**Data access.** A read shows a field only when the reader's data access
reaches it. An unattended agent reads at its own data access setting, Internal
unless someone whose role reaches Restricted raises it; a run a person is in
reads at the lower of that setting and the person's own role. Amounts, pay,
memos and raw EDI above that tier are left out and named in withheldByAccess,
and Confidential fields never reach a model at all.

## Classes

| Class | Means | Runs at most | Held once tainted | Tools that reach it |
| --- | --- | --- | --- | --- |
| Reads only | Looks something up. Nothing changes and nothing is sent. | Automatic | No | 141 |
| The caller's own records | Changes only the records of the person using the agent. | Automatic | No | 6 |
| Inside the organization | Changes records only people inside the organization see. | Automatic | No | 66 |
| Seen by a customer | Changes something a customer can see. | Ask first | Yes | 1 |
| Seen by a driver | Changes something a driver can see. | Ask first | Yes | 10 |
| Sent outside the organization | Sends to someone outside the organization. | Ask first | Yes | 14 |
| Money | Moves or commits money. | Automatic | Yes | 41 |

## Reads only

Looks something up. Nothing changes and nothing is sent.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Accept all confident (`accept_all_confident`) | Reads only | Automatic | — | — | Hands a change to the shipment the person is building on their own page; nothing is saved and nothing is sent. |
| Accept field (`accept_field`) | Reads only | Automatic | — | — | Hands a change to the shipment the person is building on their own page; nothing is saved and nothing is sent. |
| Ask user (`ask_user`) | Reads only | Automatic | — | — | Asks the person in the conversation a question; nothing is saved or sent. |
| Compare report runs (`compare_report_runs`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Compose table view (`compose_table_view`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Delegate task (`delegate_task`) | Reads only | Automatic | — | — | Hands a task to another agent of the organization, which runs as the same person under its own tiers. |
| Describe formula schema (`describe_formula_schema`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Describe report (`describe_report`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Describe report dataset (`describe_report_dataset`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Explain rate (`explain_rate`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Find in trenova (`find_in_trenova`) | Reads only | Automatic | — | — | Searches the product guide for the caller; nothing changes and nothing is sent. |
| Find tools (`find_tools`) | Reads only | Automatic | — | — | Loads more of the agent's own tools into the turn; it reads the catalog and changes nothing. |
| Get accounting mapping (`get_accounting_mapping`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get accounting sync record (`get_accounting_sync_record`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get accounting sync status (`get_accounting_sync_status`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get agent run (`get_agent_run`) | Reads only | Automatic | — | When the record is marked, from run record | Reads a run's own record, whose summary may repeat outside text the run read. |
| Get ar aging (`get_ar_aging`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get bank receipt (`get_bank_receipt`) | Reads only | Automatic | — | Always, from bank receipt | Reads a bank receipt whose memo the payer wrote; nothing changes and nothing is sent. |
| Get billing queue item (`get_billing_queue_item`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get carrier (`get_carrier`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get carrier intel event (`get_carrier_intel_event`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get carrier settlement (`get_carrier_settlement`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get customer (`get_customer`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get customer statement (`get_customer_statement`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get customer update preferences (`get_customer_update_preferences`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get daily briefing (`get_daily_briefing`) | Reads only | Automatic | — | — | Reads the morning's computed figures, and only the sections the caller may read; it changes nothing, not even whether the briefing was read. |
| Get detention occurrence (`get_detention_occurrence`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get dispatch board (`get_dispatch_board`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get document summary (`get_document_summary`) | Reads only | Automatic | — | Always, from document | Reads text extracted from a document someone outside sent; nothing changes and nothing is sent. |
| Get driver settlement (`get_driver_settlement`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get EDI inbound file (`get_edi_inbound_file`) | Reads only | Automatic | — | Always, from EDI | Reads an EDI file a trading partner sent, raw X12 included on request; nothing changes and nothing is sent. |
| Get EDI partner (`get_edi_partner`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get fiscal close blockers (`get_fiscal_close_blockers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get fuel surcharge rates (`get_fuel_surcharge_rates`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get inbound message (`get_inbound_message`) | Reads only | Automatic | — | Always, from inbound message | Reads mail an outsider wrote; nothing changes and nothing is sent. |
| Get insight (`get_insight`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get invoice (`get_invoice`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get journal entry (`get_journal_entry`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get my home layout (`get_my_home_layout`) | Reads only | Automatic | — | — | Reads the caller's own home page; nothing changes and nothing is sent. |
| Get order (`get_order`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get rate agreement (`get_rate_agreement`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get rate matrix (`get_rate_matrix`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get record accounting sync state (`get_record_accounting_sync_state`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get report run (`get_report_run`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get service failure (`get_service_failure`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get settlement dispute (`get_settlement_dispute`) | Reads only | Automatic | — | When the record is marked, from record note | Reads a pay dispute whose description a driver wrote in the driver portal; nothing changes and nothing is sent. |
| Get shipment (`get_shipment`) | Reads only | Automatic | — | When the record is marked, from record note | Reads a shipment with its newest comments, some of which a driver, a trading partner or another system outside the organization wrote; nothing changes and nothing is sent. |
| Get shipment draft (`get_shipment_draft`) | Reads only | Automatic | — | Always, from document | Reads a shipment drafted from a document someone outside sent; nothing changes and nothing is sent. |
| Get shipment tracking (`get_shipment_tracking`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get tractor (`get_tractor`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get trailer (`get_trailer`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get worker (`get_worker`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get worker credential (`get_worker_credential`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get worker earnings summary (`get_worker_earnings_summary`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get worker HOS (`get_worker_hos`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List accessorial charges (`list_accessorial_charges`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List accounting inbound changes (`list_accounting_inbound_changes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List accounting mapping gaps (`list_accounting_mapping_gaps`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List accounting sync records (`list_accounting_sync_records`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List agent runs (`list_agent_runs`) | Reads only | Automatic | — | When the record is marked, from run record | Lists agent runs, whose summaries may repeat outside text a run read; a run on a conversation is listed only to the person who owns it. |
| List ar open items (`list_ar_open_items`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List bank receipt exceptions (`list_bank_receipt_exceptions`) | Reads only | Automatic | — | Always, from bank receipt | Lists bank receipts whose memos the payers wrote; nothing changes and nothing is sent. |
| List billing queue items (`list_billing_queue_items`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List billing transfer candidates (`list_billing_transfer_candidates`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List carrier invoice matches (`list_carrier_invoice_matches`) | Reads only | Automatic | — | When the record is marked, from EDI | Lists carrier invoices, some of which a carrier sent over EDI with its own invoice text; nothing changes and nothing is sent. |
| List carrier settlements (`list_carrier_settlements`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List carriers (`list_carriers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List collections worklist (`list_collections_worklist`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List commodities (`list_commodities`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List customer payments (`list_customer_payments`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List customers (`list_customers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List dashboards (`list_dashboards`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List detention desk (`list_detention_desk`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List document types (`list_document_types`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List driver pay events (`list_driver_pay_events`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List driver settlements (`list_driver_settlements`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List EDI carrier invoices (`list_edi_carrier_invoices`) | Reads only | Automatic | — | Always, from EDI | Lists carrier invoices a carrier sent over EDI with its own invoice text; nothing changes and nothing is sent. |
| List EDI inbound files (`list_edi_inbound_files`) | Reads only | Automatic | — | Always, from EDI | Lists EDI files trading partners sent, whose names and failure reasons repeat the partner's text; nothing changes and nothing is sent. |
| List EDI transfers (`list_edi_transfers`) | Reads only | Automatic | — | Always, from EDI | Lists load tenders trading partners sent, whose contents the partner wrote; nothing changes and nothing is sent. |
| List email profiles (`list_email_profiles`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List equipment types (`list_equipment_types`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List escrow accounts (`list_escrow_accounts`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List expiring credentials (`list_expiring_credentials`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List fiscal periods (`list_fiscal_periods`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List fleet codes (`list_fleet_codes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List formula templates (`list_formula_templates`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List GL accounts (`list_gl_accounts`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List hazardous materials (`list_hazardous_materials`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List hold reasons (`list_hold_reasons`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List home widgets (`list_home_widgets`) | Reads only | Automatic | — | — | Lists the widgets the caller's home page can show; nothing changes and nothing is sent. |
| List inbound messages (`list_inbound_messages`) | Reads only | Automatic | — | Always, from inbound message | Lists mail outsiders wrote, subjects and senders included; nothing changes and nothing is sent. |
| List insights (`list_insights`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List invoices (`list_invoices`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List journal entries (`list_journal_entries`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List location categories (`list_location_categories`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List locations (`list_locations`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List orders (`list_orders`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List pay advances (`list_pay_advances`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List pay assignments (`list_pay_assignments`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List pay codes (`list_pay_codes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List pay profiles (`list_pay_profiles`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List rate agreements (`list_rate_agreements`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List rate confirmations (`list_rate_confirmations`) | Reads only | Automatic | — | — | Reads a move's rate confirmation revisions; the name a carrier typed when signing is left out, so nothing written outside the organization is read. |
| List recurring deductions (`list_recurring_deductions`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List recurring earnings (`list_recurring_earnings`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List report datasets (`list_report_datasets`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List report runs (`list_report_runs`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List reports (`list_reports`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List service failure reason codes (`list_service_failure_reason_codes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List service failures (`list_service_failures`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List service types (`list_service_types`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List shipment tenders (`list_shipment_tenders`) | Reads only | Automatic | — | — | Reads a shipment's tenders and their offers; a carrier's own words, such as a decline reason, are left out, so nothing written outside the organization is read. |
| List shipment types (`list_shipment_types`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List shipments (`list_shipments`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List time off (`list_time_off`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List tractors (`list_tractors`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List trailers (`list_trailers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List vehicle positions (`list_vehicle_positions`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List watchtower items (`list_watchtower_items`) | Reads only | Automatic | — | When the record is marked, from inbound message, EDI, weather or run record | Reads the watchtower feed, whose headlines can quote an inbound email, an EDI file, a weather alert or a run that read outside text; it changes nothing, not even what the person has seen. |
| List weather alerts (`list_weather_alerts`) | Reads only | Automatic | — | Always, from weather | Reads National Weather Service alert text; nothing changes and nothing is sent. |
| List workers (`list_workers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Open page (`open_page`) | Reads only | Automatic | — | — | Opens a page in the caller's own browser; nothing changes and nothing is sent. |
| Plan dispatch (`plan_dispatch`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Preview report (`preview_report`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Propose formula (`propose_formula`) | Reads only | Automatic | — | — | Hands a formula to the person's own editor for them to insert, test and save; nothing is saved and nothing is sent. |
| Quote shipment (`quote_shipment`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Rank move candidates (`rank_move_candidates`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Recall memory (`recall_memory`) | Reads only | Automatic | — | When the record is marked, from memory | Reads memories earlier runs saved, which carry the taint of the run that wrote them. |
| Run report (`run_report`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Search documents (`search_documents`) | Reads only | Automatic | — | Always, from document | Searches text extracted from documents, many of which someone outside wrote; each record returned is checked against what the caller may read, and nothing changes or is sent. |
| Search inbound messages (`search_inbound_messages`) | Reads only | Automatic | — | Always, from inbound message | Searches mail outsiders wrote, subjects, senders and bodies included; nothing changes and nothing is sent. |
| Search shipments (`search_shipments`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Search worker (`search_worker`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Set field value (`set_field_value`) | Reads only | Automatic | — | — | Hands a change to the shipment the person is building on their own page; nothing is saved and nothing is sent. |
| Set required field (`set_required_field`) | Reads only | Automatic | — | — | Hands a change to the shipment the person is building on their own page; nothing is saved and nothing is sent. |
| Set stop location (`set_stop_location`) | Reads only | Automatic | — | — | Hands a change to the shipment the person is building on their own page; nothing is saved and nothing is sent. |
| Set stop schedule (`set_stop_schedule`) | Reads only | Automatic | — | — | Hands a change to the shipment the person is building on their own page; nothing is saved and nothing is sent. |
| Shop carriers (`shop_carriers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Test formula expression (`test_formula_expression`) | Reads only | Automatic | — | — | Prices an expression with the formula engine, for sample values or a shipment the caller may read; nothing is saved and nothing is sent. |
| Web read (`web_read`) | Reads only | Automatic | — | Always, from web | Reads a page a web search returned; it changes nothing in Trenova and returns text written outside it. |
| Web search (`web_search`) | Reads only | Automatic | — | Always, from web | Searches the public web through the organization's extension; it changes nothing in Trenova, sends only the query, and returns text written outside it. |

## The caller's own records

Changes only the records of the person using the agent.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Add home widget (`add_home_widget`) | The caller's own records | Automatic | — | — | Changes only the caller's own home page. |
| Arrange home layout (`arrange_home_layout`) | The caller's own records | Automatic | — | — | Changes only the caller's own home page. |
| Publish artifact (`publish_artifact`) | The caller's own records | Automatic | — | — | Publishes a document into the caller's own conversation, where only they read it. |
| Remove home widget (`remove_home_widget`) | The caller's own records | Automatic | — | — | Changes only the caller's own home page. |

## Inside the organization

Changes records only people inside the organization see.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Acknowledge carrier intel event (`acknowledge_carrier_intel_event`) | Inside the organization | Automatic | — | — | Acknowledges a carrier finding inside Trenova. |
| Add dashboard tile (`add_dashboard_tile`) | Inside the organization | Automatic | — | — | Adds a tile to a saved dashboard inside Trenova. |
| Approve worker PTO (`approve_worker_pto`) | Inside the organization | Automatic | — | — | Books approved time off; the worker is told it was approved but reads no text the model wrote. |
| Assign billing queue biller (`assign_billing_queue_biller`) | Inside the organization | Automatic | — | — | Names who reviews an item inside Trenova; it creates no money and is changed by assigning someone else. |
| Assign move (`assign_move`) | Inside the organization | Automatic | — | — | Assigns a driver and tractor to a move; the driver sees the assignment but no text the model wrote. |
| Attach document to shipment (`attach_document_to_shipment`) | Inside the organization | Automatic | — | — | Files a document already in Trenova against a shipment; nobody outside is told. |
| Attach pay events to settlement (`attach_pay_events_to_settlement`) | Inside the organization | Automatic | — | — | Moves pay a driver already earned between the pool and a draft settlement inside Trenova; nothing is paid until a person approves it. |
| Check accounting connection (`check_accounting_connection`) | Inside the organization | Automatic | — | — | Asks the accounting system whether it answers and records the result in Trenova; it writes nothing to the books. |
| Clear accounting mapping (`clear_accounting_mapping`) | Inside the organization | Ask first | — | — | Unmatches a record so it cannot sync until someone maps it again; mapping it again undoes it. |
| Create accounting reference record (`create_accounting_reference_record`) | Inside the organization | Ask first | — | — | Creates a record in the organization's own accounting system; Trenova cannot delete it again, so a person approves it. |
| Create carrier invoice match (`create_carrier_invoice_match`) | Inside the organization | Automatic | — | — | Records how a carrier's invoice compares with the expected cost inside Trenova; nothing is paid until a person accepts it. |
| Create dashboard (`create_dashboard`) | Inside the organization | Automatic | — | — | Saves a report dashboard colleagues can open; nothing leaves the organization. |
| Create location (`create_location`) | Inside the organization | Ask first | — | — | A new location is where colleagues will book freight, and its address is usually read from a document someone outside sent, so a person approves it first. |
| Create report (`create_report`) | The caller's own records, inside the organization | Automatic | Each call is classified by what it reaches. A call on the caller's own records runs unasked while they are present. | — | A private report is a saved query on the caller's own list; a shared one appears on every colleague's Reports page and waits for approval. |
| Create shipment (`create_shipment`) | Inside the organization | Ask first | — | — | A new load commits a customer's freight and the money that follows, so no desk books one unattended. |
| Create table change alert (`create_table_change_alert`) | Inside the organization | Automatic | — | — | Creates an alert whose notices go to people inside the organization. |
| Detach pay event from settlement (`detach_pay_event_from_settlement`) | Inside the organization | Automatic | — | — | Moves pay a driver already earned between the pool and a draft settlement inside Trenova; nothing is paid until a person approves it. |
| Dismiss insight (`dismiss_insight`) | Inside the organization | Automatic | — | — | Dismisses an insight inside Trenova; it can be restored. |
| Escalate detention (`escalate_detention`) | Inside the organization | Automatic | — | — | Hands a detention clock to a person inside the organization. |
| Evaluate service failures (`evaluate_service_failures`) | Inside the organization | Automatic | — | — | Opens service failures from stop actuals inside Trenova; an open failure sends nothing. |
| Flag for manual review (`flag_for_manual_review`) | Inside the organization | Automatic | — | — | Records an exception on the run for a person inside the organization to work. |
| Forget memory (`forget_memory`) | Inside the organization | Automatic | — | — | Retires an agent memory inside Trenova. |
| Fork report (`fork_report`) | Inside the organization | Automatic | — | — | Saves a copy of a report inside Trenova; nothing leaves the organization. |
| Generate carrier settlement batch (`generate_carrier_settlement_batch`) | Inside the organization | Automatic | — | — | Drafts the period's carrier settlements from cost already accrued inside Trenova; nothing is paid until a person approves and posts them. |
| Generate driver settlement (`generate_driver_settlement`) | Inside the organization | Automatic | — | — | Drafts a settlement from pay already accrued inside Trenova; nothing is paid until a person approves it, and hands-off approval is only the settlement control's clean-settlement rule. |
| Generate driver settlement batch (`generate_driver_settlement_batch`) | Inside the organization | Automatic | — | — | Drafts the period's settlements from pay already accrued inside Trenova; nothing is paid until a person approves them, and hands-off approval is only the settlement control's clean-settlement rule. |
| Generate rate confirmation (`generate_rate_confirmation`) | Inside the organization | Automatic | A first revision is generated as far as the agent allows; one that would void a revision already standing, which the carrier may hold or have signed, is a proposal a person decides, as is one the service would refuse or that cannot be read. | — | Renders the agreement and files it on the shipment inside Trenova; nothing reaches the carrier until it is sent, and voiding the revision undoes it. |
| Hold billing queue item (`hold_billing_queue_item`) | Inside the organization | Automatic | Its notes are what a biller or operations reads next, so a run that has read outside text proposes it rather than writing it. | — | Parks an item inside Trenova with a note; it creates no money and is undone by moving the item on. |
| Ignore accounting inbound change (`ignore_accounting_inbound_change`) | Inside the organization | Ask first | — | — | Keeps a payment the books hold out of Trenova for good, so a person approves it. |
| Link EDI carrier invoice to carrier (`link_edi_carrier_invoice_to_carrier`) | Inside the organization | Automatic | — | — | Names which carrier an inbound invoice belongs to inside Trenova; nothing is paid and it is relinked the same way. |
| Link inbound message (`link_inbound_message`) | Inside the organization | Automatic | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Links an inbound message to a record inside Trenova; the mailbox decides how far it may run. |
| Mark inbound message (`mark_inbound_message`) | Inside the organization | Automatic | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Settles an inbound message inside Trenova; the mailbox decides how far it may run. |
| Move billing item to exception (`move_billing_item_to_exception`) | Inside the organization | Automatic | Its notes are what a biller or operations reads next, so a run that has read outside text proposes it rather than writing it. | — | Marks an item for a biller to resolve inside Trenova; it creates no money and a biller moves it back when it is resolved. |
| Pause accounting sync (`pause_accounting_sync`) | Inside the organization | Automatic | — | — | Holds what is waiting to go to the books; nothing is sent, changed or lost, and resuming sends it on. |
| Place shipment hold (`place_shipment_hold`) | Inside the organization | Automatic | — | — | Places a hold on a shipment inside Trenova; no customer or EDI notice is sent. |
| Place worker dispatch hold (`place_worker_dispatch_hold`) | Inside the organization | Automatic | — | — | Keeps a driver off new freight inside Trenova; the driver is not messaged. |
| Raise exception (`raise_exception`) | Inside the organization | Automatic | — | — | Records an exception against the run itself for a person inside the organization. |
| Recalculate carrier settlement (`recalculate_carrier_settlement`) | Inside the organization | Automatic | — | — | Rebuilds a draft from the records it is computed from inside Trenova; nothing is paid until a person approves it. |
| Recalculate driver settlement (`recalculate_driver_settlement`) | Inside the organization | Automatic | — | — | Rebuilds a draft from the records it is computed from inside Trenova; nothing is paid until a person approves it. |
| Record stop actual (`record_stop_actual`) | Inside the organization | Automatic | — | — | Records arrival and departure times on a stop; no model-written text leaves the organization. |
| Refresh accounting reference data (`refresh_accounting_reference_data`) | Inside the organization | Automatic | — | — | Reads the accounting system and refreshes Trenova's suggestions; it writes nothing to the books and leaves confirmed mappings alone. |
| Reject carrier invoice match (`reject_carrier_invoice_match`) | Inside the organization | Automatic | Its note is what accounts payable reads about the carrier's invoice, so a run that has read outside text proposes it. | — | Marks a carrier's invoice as not payable inside Trenova; nothing is paid and a corrected invoice is matched afresh. |
| Reject carrier settlement (`reject_carrier_settlement`) | Inside the organization | Automatic | Its text is what payroll or the payee reads next, so a run that has read outside text proposes it rather than writing it. | — | Returns a settlement to draft with a note inside Trenova; nothing is paid and it is submitted again once fixed. |
| Reject driver settlement (`reject_driver_settlement`) | Inside the organization | Automatic | Its text is what payroll or the payee reads next, so a run that has read outside text proposes it rather than writing it. | — | Returns a settlement to draft with a note inside Trenova; nothing is paid and it is submitted again once fixed. |
| Release driver pay event (`release_driver_pay_event`) | Inside the organization | Automatic | — | — | Returns held pay to the settlement pool inside Trenova; nothing is paid until a settlement is approved, and the pay is held again the same way. |
| Release shipment hold (`release_shipment_hold`) | Inside the organization | Automatic | — | — | Releases a hold on a shipment inside Trenova; no customer or EDI notice is sent. |
| Remember (`remember`) | Inside the organization | Automatic | An Instruction or a Correction recorded after the run read text from outside the organization waits for a person's approval; a Fact is recorded and stays marked as drawn from outside text. | Carries outside text into later runs | Saves a memory later runs read, so it keeps the taint of the run that wrote it. |
| Request accounting backfill (`request_accounting_backfill`) | Inside the organization | Ask first | — | — | Sends historical documents to the organization's books, some of which may already be there by hand, and cannot be called back; a person who manages the integration approves it. |
| Resolve bank receipt work item (`resolve_bank_receipt_work_item`) | Inside the organization | Automatic | — | — | Closes a reconciliation work item without moving money; the note is read inside the organization. |
| Resolve carrier intel event (`resolve_carrier_intel_event`) | Inside the organization | Automatic | — | — | Closes a carrier finding inside Trenova. |
| Resume accounting sync (`resume_accounting_sync`) | Inside the organization | Ask first | — | — | Releases everything held to the organization's books at once, and a person paused it for a reason, so a person approves it; what is sent cannot be called back. |
| Retry accounting sync (`retry_accounting_sync`) | Inside the organization | Automatic | — | — | Sends again, to the organization's own books, documents Trenova already decided to send; the accounting system recognizes a repeat by its request id, so nothing is entered twice, and a document held for release stays held. |
| Save table view (`save_table_view`) | The caller's own records, inside the organization | Automatic | Each call is classified by what it reaches. A call on the caller's own records runs unasked while they are present. | — | A private view is the caller's own picker entry; a shared one appears for every colleague, and nothing leaves the organization. |
| Send billing item back to ops (`send_billing_item_back_to_ops`) | Inside the organization | Automatic | Its notes are what a biller or operations reads next, so a run that has read outside text proposes it rather than writing it. | — | Returns an item to operations with a note on its shipment inside Trenova; it creates no money and the item comes back when operations fixes it. |
| Set accounting mapping (`set_accounting_mapping`) | Inside the organization | Ask first | — | — | Decides which account or record Trenova's invoices, payments and bills will post to, so a person approves it; clearing or changing it undoes it. |
| Skip accounting sync (`skip_accounting_sync`) | Inside the organization | Ask first | — | — | Leaves a document out of the organization's books for good; nothing sends it again, so a person approves it. |
| Submit carrier settlement (`submit_carrier_settlement`) | Inside the organization | Automatic | — | — | Moves a draft into the approval queue inside Trenova; nothing is paid and a reviewer sends it back to draft. |
| Submit driver settlement (`submit_driver_settlement`) | Inside the organization | Automatic | — | — | Moves a draft into the approval queue inside Trenova; nothing is paid and a reviewer sends it back to draft. |
| Transfer to billing (`transfer_to_billing`) | Inside the organization | Automatic | — | — | Hands delivered shipments to the billing queue inside Trenova, by the checks the transfer dialog makes; a biller, or the organization's own auto-approve rule, still decides every item. |
| Transition item to in review (`transition_item_to_in_review`) | Inside the organization | Automatic | An item on hold was held there by a person or a rule, so moving one into review is a proposal a person decides; an item in any other state moves as far as the agent allows, and one that cannot be read waits for a person. | — | Moves the run's billing queue item into review inside Trenova; nothing is sent anywhere. |
| Unassign moves (`unassign_moves`) | Inside the organization | Automatic | Only one move at a time, still freshly assigned, is taken off its driver without a decision; several moves, a move the service would refuse and a move that cannot be read each wait for a person. | — | Takes the driver off a move that has not started, inside Trenova; the driver is told the load was taken off them but sees no text the model wrote, and assigning the move again undoes it. |
| Update move status (`update_move_status`) | Inside the organization | Ask first | Canceling a move ends it for good, so a cancellation is a proposal a person decides; any other status runs once a person approves it. | — | Moves a move's status forward inside Trenova, which re-derives its shipment's status and, on completion, releases its equipment; a move never moves back, so it is not undone by running it again. |
| Update report (`update_report`) | Inside the organization | Automatic | — | — | Changes a saved report colleagues may open; nothing leaves the organization. |
| Update tractor status (`update_tractor_status`) | Inside the organization | Automatic | — | — | Changes a tractor's status inside Trenova. |
| Update trailer status (`update_trailer_status`) | Inside the organization | Automatic | — | — | Changes a trailer's status inside Trenova. |

## Seen by a driver

Changes something a driver can see.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Add shipment comment (`add_shipment_comment`) | Inside the organization, seen by a customer, seen by a driver | Automatic | Each call is classified by what it reaches. | Carries outside text into later runs | An internal note stays inside the organization; a customer or driver note is read outside it, so its visibility argument decides. A note written on its own after the run read outside text is marked as drawn from it. |
| Cancel worker PTO (`cancel_worker_pto`) | Seen by a driver | Ask first | — | — | The worker is sent the cancellation reason by push notice and text message. |
| Hold driver pay event (`hold_driver_pay_event`) | Seen by a driver | Ask first | — | — | Defers a driver's pay and tells the driver why in the driver portal; it is released the same way. |
| Notify driver (`notify_driver`) | Seen by a driver | Ask first | — | — | Sends a driver a message the model wrote to their phone. |
| Reject worker PTO (`reject_worker_pto`) | Seen by a driver | Ask first | — | — | The worker is sent the rejection reason by push notice and text message. |
| Request credential renewal (`request_credential_renewal`) | Seen by a driver | Ask first | — | — | Sends a driver a renewal request the model wrote. |

## Sent outside the organization

Sends to someone outside the organization.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Cancel shipment (`cancel_shipment`) | Sent outside the organization | Ask first | — | — | Cancelling withdraws live tenders from carriers and sends the model's cancel reason to a linked partner over EDI. |
| Cancel tender (`cancel_tender`) | Sent outside the organization | Ask first | A tender waiting on review holds a carrier's acceptance, so withdrawing it is a proposal a person decides, as is one the service would refuse or that cannot be read; an active tender is withdrawn once a person approves it. | — | Withdraws the offers carriers outside the organization hold, whose answer links stop working; a withdrawn tender cannot be reopened, only tendered afresh. |
| Email customer (`email_customer`) | Sent outside the organization | Ask first | — | — | Emails the customer's contacts a message the model wrote. |
| Reply to inbound message (`reply_to_inbound_message`) | Sent outside the organization | Ask first | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Replies to whoever wrote in with text the model composed. |
| Request missing docs (`request_missing_docs`) | Sent outside the organization | Ask first | — | — | Emails an outside party a request for paperwork in words the model wrote. |
| Resolve service failure (`resolve_service_failure`) | Sent outside the organization | Ask first | — | — | Resolving a failure generates an EDI 214 to the customer's trading partner carrying the reason chosen. |
| Schedule report (`schedule_report`) | Sent outside the organization | Ask first | — | — | Emails a report on a schedule to whatever addresses the call names, which may be outside the organization. |
| Send detention notice (`send_detention_notice`) | Sent outside the organization | Ask first | — | — | Sends the customer a detention notice that starts a charge. |
| Send invoice (`send_invoice`) | Sent outside the organization | Propose | — | — | Emails the customer their invoice; only a person sends it, to the recipients the customer's billing profile names. |
| Send rate confirmation (`send_rate_confirmation`) | Sent outside the organization | Propose | — | — | Emails a binding agreement, with a link to sign it, to a carrier outside the organization; the recipients come from the carrier's record, and a sent email cannot be recalled, so a person decides every send. |
| Tender move to carriers (`tender_move_to_carriers`) | Sent outside the organization | Ask first | — | — | Offers the load to carriers outside the organization. |
| Tender move to routing guide (`tender_move_to_routing_guide`) | Sent outside the organization | Ask first | — | — | Offers the load to carriers outside the organization. |
| Update shipment (`update_shipment`) | Sent outside the organization | Ask first | — | — | A changed shipment is sent as an EDI tender change to the trading partners it was tendered to. |

## Money

Moves or commits money.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Accept carrier invoice match (`accept_carrier_invoice_match`) | Money | Propose | — | — | Clears a carrier's invoice for payment; only a person approves what the organization pays. |
| Accept carrier invoice match with variance (`accept_carrier_invoice_match_with_variance`) | Money | Propose | — | — | Agrees to pay a carrier more or less than the load was expected to cost; only a person approves it. |
| Add carrier settlement adjustment (`add_carrier_settlement_adjustment`) | Money | Automatic | Its text is what payroll or the payee reads next, so a run that has read outside text proposes it rather than writing it. | — | Changes what the carrier will be paid on a settlement a person still approves, so it moves money; the line is removed the same way. |
| Add driver settlement adjustment (`add_driver_settlement_adjustment`) | Money | Automatic | Its text is what payroll or the payee reads next, so a run that has read outside text proposes it rather than writing it. | — | Changes what the driver will be paid on a settlement a person still approves, so it moves money; the line is removed the same way. |
| Adjust escrow account (`adjust_escrow_account`) | Money | Propose | — | — | Moves money held in escrow for an owner-operator; only a person moves it. |
| Apply accounting inbound change (`apply_accounting_inbound_change`) | Money | Ask first | — | — | Posts money in Trenova: a customer payment or a settlement payment, so a person approves each one. |
| Approve billing queue item (`approve_billing_queue_item`) | Money | Propose | — | — | Approving creates the invoice a customer is billed on, so only a person approves; the agent proposes it with what it checked. |
| Approve carrier settlement (`approve_carrier_settlement`) | Money | Propose | — | — | Commits the organization to what the carrier is paid; only a person approves, and hands-off approval is the settlement control's rule. |
| Approve detention (`approve_detention`) | Money | Propose | — | — | Releases a held detention charge onto the customer's invoice; only a person approves it. |
| Approve driver settlement (`approve_driver_settlement`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Commits the organization to what the driver is paid; only a person approves, and hands-off approval is the settlement control's rule. |
| Assign move to carrier (`assign_move_to_carrier`) | Money | Ask first | Replacing a carrier already on the move, or overriding the new carrier's insurance warning, is a proposal a person decides; any other assignment runs once a person approves it. | — | Commits the organization to pay an outside carrier the rate it names for the move; nothing is sent to the carrier, and canceling the assignment releases the move. |
| Assign pay profile (`assign_pay_profile`) | Money | Propose | — | — | Sets how a driver is paid from a date on; only a person decides someone's pay. |
| Cancel billing queue item (`cancel_billing_queue_item`) | Money | Propose | — | — | Drops a charge from billing for good, so only a person decides; the agent proposes it. |
| Cancel carrier assignment (`cancel_carrier_assignment`) | Money | Ask first | Taking off a carrier who has confirmed the rate breaks an executed agreement, so it is a proposal a person decides, as is one the service would refuse or that cannot be read; a carrier who has not confirmed comes off once a person approves it. | — | Releases the organization's commitment to pay an outside carrier and voids the carrier's rate confirmation, whose sign link stops working; covering the move again makes a new agreement the carrier has to confirm afresh. |
| Close escrow account (`close_escrow_account`) | Money | Propose | — | — | Refunds an owner-operator's escrow balance and closes the account for good; only a person closes it. |
| Correct charge code (`correct_charge_code`) | Money | Automatic | — | — | Rewrites the accessorial charges a customer will be invoiced, so it moves money. |
| Create recurring deduction (`create_recurring_deduction`) | Money | Propose | — | — | Sets money taken from or added to a driver's pay every settlement; only a person agrees it. |
| Create recurring earning (`create_recurring_earning`) | Money | Propose | — | — | Sets money taken from or added to a driver's pay every settlement; only a person agrees it. |
| End pay assignment (`end_pay_assignment`) | Money | Propose | — | — | Stops how a driver is paid from a date; only a person decides someone's pay. |
| Issue pay advance (`issue_pay_advance`) | Money | Propose | — | — | Records money handed to a driver that their pay then repays; only a person records it. |
| Match bank receipt (`match_bank_receipt`) | Money | Automatic | — | — | Matches a bank receipt to a posted payment, closing its reconciliation. |
| Open escrow account (`open_escrow_account`) | Money | Propose | — | — | Sets escrow terms under an owner-operator's lease that their pay is then withheld against; only a person agrees them. |
| Pay driver now (`pay_driver_now`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Approves, posts and records a payment to a driver in one step; only a person pays someone. |
| Post carrier settlement (`post_carrier_settlement`) | Money | Propose | — | — | Books the settlement's payable to the ledger and queues it for the accounting system; only a person posts. |
| Post customer payment (`post_customer_payment`) | Money | Automatic | — | — | Records a customer payment and applies it to invoices. |
| Post driver settlement (`post_driver_settlement`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Books the settlement's payable to the ledger and queues it for the accounting system; only a person posts. |
| Post invoice (`post_invoice`) | Money | Propose | — | — | Books a receivable to the ledger and queues it for the accounting system and the customer's EDI; only a person posts, and hands-off posting is the billing-control auto-post setting. |
| Record carrier settlement payment (`record_carrier_settlement_payment`) | Money | Propose | — | — | Records a payment to the carrier and queues it for the accounting system; only a person says money went out. |
| Record driver settlement payment (`record_driver_settlement_payment`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Records a payment to the driver and queues it for the accounting system; only a person says money went out. |
| Record rate confirmation confirmed (`record_rate_confirmation_confirmed`) | Money | Ask first | — | — | Makes the revision the executed agreement to pay the carrier and confirms their assignment, on the word of someone outside the organization; voiding it afterwards withdraws the agreement rather than restoring it. |
| Record tender response (`record_tender_response`) | Sent outside the organization, money | Ask first | Each call is classified by what it reaches. Recording an acceptance commits the load to the carrier at the offered rate, so it is a proposal a person decides; a decline runs once a person approves it. | — | An acceptance commits the organization to pay the carrier and sends them the rate confirmation; a decline sends the next carrier its offer. Neither is undone by recording another answer. |
| Remove carrier settlement adjustment (`remove_carrier_settlement_adjustment`) | Money | Automatic | — | — | Changes what the carrier will be paid on a settlement a person still approves, so it moves money; the line is added back the same way. |
| Remove driver settlement adjustment (`remove_driver_settlement_adjustment`) | Money | Automatic | — | — | Changes what the driver will be paid on a settlement a person still approves, so it moves money; the line is added back the same way. |
| Update escrow account (`update_escrow_account`) | Money | Propose | — | — | Changes the escrow terms of an owner-operator's lease; only a person agrees them. |
| Update recurring deduction (`update_recurring_deduction`) | Money | Propose | — | — | Changes money taken from or added to a driver's pay every settlement; only a person agrees it. |
| Update recurring earning (`update_recurring_earning`) | Money | Propose | — | — | Changes money taken from or added to a driver's pay every settlement; only a person agrees it. |
| Void carrier settlement (`void_carrier_settlement`) | Money | Propose | — | — | Cancels a settlement for good and reverses any posting; only a person voids. |
| Void driver settlement (`void_driver_settlement`) | Money | Propose | — | — | Cancels a settlement for good and reverses any posting; only a person voids. |
| Void rate confirmation (`void_rate_confirmation`) | Money | Ask first | A revision the carrier has been sent or has signed is voided only as a proposal a person decides, as is one that cannot be read; one never sent is voided once a person approves it. | — | Withdraws the organization's written agreement to pay a carrier, whose sign link stops working, and undoes a confirmation it carried; a voided revision cannot be restored, only replaced. |
| Waive detention (`waive_detention`) | Money | Propose | — | — | Gives up detention revenue the organization would otherwise bill; only a person approves it. |
| Write off pay advance (`write_off_pay_advance`) | Money | Propose | — | — | Forgives money a driver owes, which the organization absorbs; only a person writes it off. |

