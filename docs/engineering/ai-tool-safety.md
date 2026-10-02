# AI tool safety

<!-- Generated from the tool policies in code by running, in services/tms:
     go generate ./internal/core/services/agenttoolpolicy/safetydoc/...
     Do not edit by hand. -->

What every tool an agent can call may do without a person, read from the
policies the tools declare in code. The runtime decides each call from the same
policies, so this page cannot drift from what runs: CI regenerates it and fails
when it differs. Each tool is listed once, under the furthest class its work
can reach.

Tools listed: 551.

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
| Reads only | Looks something up. Nothing changes and nothing is sent. | Automatic | No | 189 |
| The caller's own records | Changes only the records of the person using the agent. | Automatic | No | 7 |
| Inside the organization | Changes records only people inside the organization see. | Automatic | No | 219 |
| Seen by a customer | Changes something a customer can see. | Ask first | Yes | 3 |
| Seen by a driver | Changes something a driver can see. | Ask first | Yes | 24 |
| Sent outside the organization | Sends to someone outside the organization. | Ask first | Yes | 33 |
| Money | Moves or commits money. | Automatic | Yes | 97 |

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
| Get DOT random draw (`get_dot_random_draw`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get driver settlement (`get_driver_settlement`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get EDI inbound file (`get_edi_inbound_file`) | Reads only | Automatic | — | Always, from EDI | Reads an EDI file a trading partner sent, raw X12 included on request; nothing changes and nothing is sent. |
| Get EDI partner (`get_edi_partner`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get EDI transfer (`get_edi_transfer`) | Reads only | Automatic | — | Always, from EDI | Reads a load tender a trading partner wrote, stops and references included; nothing changes and nothing is sent. |
| Get fiscal close blockers (`get_fiscal_close_blockers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get fuel surcharge rates (`get_fuel_surcharge_rates`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get inbound message (`get_inbound_message`) | Reads only | Automatic | — | Always, from inbound message | Reads mail an outsider wrote; nothing changes and nothing is sent. |
| Get insight (`get_insight`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get invoice (`get_invoice`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get invoice adjustment (`get_invoice_adjustment`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get invoice run (`get_invoice_run`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get invoices (`get_invoices`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get journal entry (`get_journal_entry`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get manual journal (`get_manual_journal`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
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
| Get worker schedule (`get_worker_schedule`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List accessorial charges (`list_accessorial_charges`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List accounting drift findings (`list_accounting_drift_findings`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List accounting inbound changes (`list_accounting_inbound_changes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List accounting mapping gaps (`list_accounting_mapping_gaps`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List accounting sync records (`list_accounting_sync_records`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List agent runs (`list_agent_runs`) | Reads only | Automatic | — | When the record is marked, from run record | Lists agent runs, whose summaries may repeat outside text a run read; a run on a conversation is listed only to the person who owns it. |
| List ar open items (`list_ar_open_items`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List bank receipt exceptions (`list_bank_receipt_exceptions`) | Reads only | Automatic | — | Always, from bank receipt | Lists bank receipts whose memos the payers wrote; nothing changes and nothing is sent. |
| List billing queue items (`list_billing_queue_items`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List billing transfer candidates (`list_billing_transfer_candidates`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List capture batches (`list_capture_batches`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List carrier invoice matches (`list_carrier_invoice_matches`) | Reads only | Automatic | — | When the record is marked, from EDI | Lists carrier invoices, some of which a carrier sent over EDI with its own invoice text; nothing changes and nothing is sent. |
| List carrier settlements (`list_carrier_settlements`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List carriers (`list_carriers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List collections worklist (`list_collections_worklist`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List commodities (`list_commodities`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List credit memo applications (`list_credit_memo_applications`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List customer payments (`list_customer_payments`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List customers (`list_customers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List dashboards (`list_dashboards`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List detention desk (`list_detention_desk`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List document types (`list_document_types`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List DOT random draws (`list_dot_random_draws`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List DOT tests (`list_dot_tests`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List driver expenses (`list_driver_expenses`) | Reads only | Automatic | — | When the record is marked, from record note | Reads expenses whose descriptions drivers wrote in the driver portal; nothing changes and nothing is sent. |
| List driver pay events (`list_driver_pay_events`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List driver settlements (`list_driver_settlements`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List EDI carrier invoices (`list_edi_carrier_invoices`) | Reads only | Automatic | — | Always, from EDI | Lists carrier invoices a carrier sent over EDI with its own invoice text; nothing changes and nothing is sent. |
| List EDI inbound files (`list_edi_inbound_files`) | Reads only | Automatic | — | Always, from EDI | Lists EDI files trading partners sent, whose names and failure reasons repeat the partner's text; nothing changes and nothing is sent. |
| List EDI messages (`list_edi_messages`) | Reads only | Automatic | — | Always, from EDI | Lists EDI documents and the errors partners' systems returned for them; nothing changes and nothing is sent. |
| List EDI partners (`list_edi_partners`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List EDI tender changes (`list_edi_tender_changes`) | Reads only | Automatic | — | Always, from EDI | Lists changes another organization wrote to loads tendered over EDI; nothing changes and nothing is sent. |
| List EDI transfer changes (`list_edi_transfer_changes`) | Reads only | Automatic | — | Always, from EDI | Lists statuses another organization reported on a linked load; nothing changes and nothing is sent. |
| List EDI transfers (`list_edi_transfers`) | Reads only | Automatic | — | Always, from EDI | Lists load tenders trading partners sent, whose contents the partner wrote; nothing changes and nothing is sent. |
| List email profiles (`list_email_profiles`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List employment verifications (`list_employment_verifications`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List equipment manufacturers (`list_equipment_manufacturers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List equipment types (`list_equipment_types`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List escrow accounts (`list_escrow_accounts`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List expiring credentials (`list_expiring_credentials`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List fiscal periods (`list_fiscal_periods`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List fleet codes (`list_fleet_codes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List formula templates (`list_formula_templates`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List fuel cards (`list_fuel_cards`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List fuel index prices (`list_fuel_index_prices`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List fuel purchase imports (`list_fuel_purchase_imports`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List fuel purchases (`list_fuel_purchases`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List GL accounts (`list_gl_accounts`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List hazardous materials (`list_hazardous_materials`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List hold reasons (`list_hold_reasons`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List home widgets (`list_home_widgets`) | Reads only | Automatic | — | — | Lists the widgets the caller's home page can show; nothing changes and nothing is sent. |
| List IFTA jurisdictions (`list_ifta_jurisdictions`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List IFTA mileage entries (`list_ifta_mileage_entries`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List IFTA returns (`list_ifta_returns`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List inbound messages (`list_inbound_messages`) | Reads only | Automatic | — | Always, from inbound message | Lists mail outsiders wrote, subjects and senders included; nothing changes and nothing is sent. |
| List insights (`list_insights`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List invoice adjustments (`list_invoice_adjustments`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List invoice disputes (`list_invoice_disputes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List invoice runs (`list_invoice_runs`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List invoice share candidates (`list_invoice_share_candidates`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List invoices (`list_invoices`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List journal entries (`list_journal_entries`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List journal reversals (`list_journal_reversals`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List location categories (`list_location_categories`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List locations (`list_locations`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List manual journals (`list_manual_journals`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List open statements (`list_open_statements`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List orders (`list_orders`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List pay advances (`list_pay_advances`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List pay assignments (`list_pay_assignments`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List pay codes (`list_pay_codes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List pay profiles (`list_pay_profiles`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List payroll exports (`list_payroll_exports`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List performance reviews (`list_performance_reviews`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List rate agreements (`list_rate_agreements`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List rate confirmations (`list_rate_confirmations`) | Reads only | Automatic | — | — | Reads a move's rate confirmation revisions; the name a carrier typed when signing is left out, so nothing written outside the organization is read. |
| List rate imports (`list_rate_imports`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List recurring deductions (`list_recurring_deductions`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List recurring earnings (`list_recurring_earnings`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List recurring shipments (`list_recurring_shipments`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List report datasets (`list_report_datasets`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List report runs (`list_report_runs`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List report schedules (`list_report_schedules`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List reports (`list_reports`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List service failure reason codes (`list_service_failure_reason_codes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List service failures (`list_service_failures`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List service types (`list_service_types`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List shift templates (`list_shift_templates`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List shipment permits (`list_shipment_permits`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List shipment tenders (`list_shipment_tenders`) | Reads only | Automatic | — | — | Reads a shipment's tenders and their offers; a carrier's own words, such as a decline reason, are left out, so nothing written outside the organization is read. |
| List shipment types (`list_shipment_types`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List shipments (`list_shipments`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List table change alerts (`list_table_change_alerts`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List time off (`list_time_off`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List tractors (`list_tractors`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List trailers (`list_trailers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List training courses (`list_training_courses`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List vehicle positions (`list_vehicle_positions`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List watchtower items (`list_watchtower_items`) | Reads only | Automatic | — | When the record is marked, from inbound message, EDI, weather or run record | Reads the watchtower feed, whose headlines can quote an inbound email, an EDI file, a weather alert or a run that read outside text; it changes nothing, not even what the person has seen. |
| List weather alerts (`list_weather_alerts`) | Reads only | Automatic | — | Always, from weather | Reads National Weather Service alert text; nothing changes and nothing is sent. |
| List worker checklists (`list_worker_checklists`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List worker credentials (`list_worker_credentials`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List worker injuries (`list_worker_injuries`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List worker leave cases (`list_worker_leave_cases`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List worker safety events (`list_worker_safety_events`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List worker training (`list_worker_training`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
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
| Request decision (`request_decision`) | The caller's own records | Automatic | — | — | Opens, in the person's own conversation, the approval box on a proposal already waiting on them; it decides nothing and changes no record. |

## Inside the organization

Changes records only people inside the organization see.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Acknowledge carrier intel event (`acknowledge_carrier_intel_event`) | Inside the organization | Automatic | — | — | Acknowledges a carrier finding inside Trenova. |
| Add dashboard tile (`add_dashboard_tile`) | Inside the organization | Automatic | — | — | Adds a tile to a saved dashboard inside Trenova. |
| Adjust invoice run membership (`adjust_invoice_run_membership`) | Inside the organization | Automatic | An exclusion read from what a customer sent is proposed, since it decides what they are billed this period. | — | Edits a proposal of invoices before anything is invoiced; each change is undone by the opposite edit. |
| Amend IFTA return (`amend_ifta_return`) | Inside the organization | Propose | — | — | Starts a correction of a return already filed with the jurisdictions; it files nothing, but a person decides to reopen a filing. |
| Apply carrier intel suggestions (`apply_carrier_intel_suggestions`) | Inside the organization | Ask first | — | — | Edits the carrier's own profile inside Trenova from what the provider reports; nothing is sent, and the old values are set back by hand. |
| Approve worker PTO (`approve_worker_pto`) | Inside the organization | Automatic | — | — | Books approved time off; the worker is told it was approved but reads no text the model wrote. |
| Archive worker credential (`archive_worker_credential`) | Inside the organization | Propose | — | — | Retires a credential inside Trenova, which can take the worker off dispatch; the archived row stays on file, and a new credential is recorded to replace it. |
| Assign billing queue biller (`assign_billing_queue_biller`) | Inside the organization | Automatic | — | — | Names who reviews an item inside Trenova; it creates no money and is changed by assigning someone else. |
| Assign fuel card (`assign_fuel_card`) | Inside the organization | Ask first | — | — | Changes which tractor a card's purchases count against inside Trenova; nothing is sent, and assigning it again changes it back. |
| Assign move (`assign_move`) | Inside the organization | Automatic | — | — | Assigns a driver and tractor to a move; the driver sees the assignment but no text the model wrote. |
| Assign worker shift (`assign_worker_shift`) | Inside the organization | Ask first | — | — | Changes a worker's standing schedule inside Trenova; nothing is sent, and end_worker_shift_assignment or a new assignment changes it back. |
| Attach document to shipment (`attach_document_to_shipment`) | Inside the organization | Automatic | — | — | Files a document already in Trenova against a shipment; nobody outside is told. |
| Attach order shipments (`attach_order_shipments`) | Inside the organization | Ask first | — | — | Regroups shipments under an order inside Trenova; nothing is sent, and detach_order_shipment moves one back out. |
| Attach pay events to settlement (`attach_pay_events_to_settlement`) | Inside the organization | Automatic | — | — | Moves pay a driver already earned between the pool and a draft settlement inside Trenova; nothing is paid until a person approves it. |
| Attach worker credential document (`attach_worker_credential_document`) | Inside the organization | Ask first | — | — | Links a filed document to a credential inside Trenova; attaching another replaces it. |
| Attach worker training document (`attach_worker_training_document`) | Inside the organization | Ask first | — | — | Links a filed document to a training record inside Trenova; attaching another replaces it. |
| Backfill jurisdiction miles (`backfill_jurisdiction_miles`) | Inside the organization | Ask first | — | — | Routes a quarter's unattributed moves inside Trenova in the background; nothing is sent to a customer or carrier, but each move is a billable request. |
| Build invoice run (`build_invoice_run`) | Inside the organization | Automatic | — | — | Builds a proposal of invoices inside Trenova for a biller to review; it invoices nothing and is discarded with cancel_invoice_run. |
| Cancel DOT random draw (`cancel_dot_random_draw`) | Inside the organization | Propose | — | — | Voids a round so the period is drawn again; the cancelled round and its reason stay on the record, and it cannot be undone. |
| Cancel DOT test (`cancel_dot_test`) | Inside the organization | Propose | — | — | Voids a scheduled collection; the row and its reason stay on the record an auditor reads, and it cannot be undone. |
| Cancel invoice run (`cancel_invoice_run`) | Inside the organization | Ask first | — | — | Discards a proposal of invoices; nothing was invoiced, and a new run is built with build_invoice_run. |
| Cancel journal reversal (`cancel_journal_reversal`) | Inside the organization | Ask first | The reason is what the reversal's requester reads next, so a run that has read outside text proposes it. | — | Withdraws a reversal before it reaches the ledger; nothing is booked, and it is requested again if it was needed after all. |
| Cancel manual journal (`cancel_manual_journal`) | Inside the organization | Ask first | The reason is what the journal's author reads next, so a run that has read outside text proposes it. | — | Withdraws a journal before it reaches the ledger; nothing is booked, and the entry is drafted again if it was needed after all. |
| Cancel report run (`cancel_report_run`) | Inside the organization | Automatic | — | — | Stops a report the person asked for from running; nothing is changed or sent, and running the report again gives the same rows. |
| Cancel worker checklist (`cancel_worker_checklist`) | Inside the organization | Propose | — | — | Cancels a checklist inside Trenova; nothing is sent, and a cancelled checklist is started again from its template. |
| Change accounting backfill (`change_accounting_backfill`) | Inside the organization | Ask first | — | — | Changes how much history reaches the organization's books; a person who manages the integration approves it, and pausing or cancelling sends nothing. |
| Change worker safety event status (`change_worker_safety_event_status`) | Inside the organization | Ask first | — | — | Moves a safety event's status inside Trenova; nothing is sent, and the event is reopened or closed again the same way. |
| Check accounting connection (`check_accounting_connection`) | Inside the organization | Automatic | — | — | Asks the accounting system whether it answers and records the result in Trenova; it writes nothing to the books. |
| Check accounting drift (`check_accounting_drift`) | Inside the organization | Ask first | — | — | Reads every synced document back from the accounting system, which spends the provider's call allowance, so a person approves it. |
| Clear accounting mapping (`clear_accounting_mapping`) | Inside the organization | Ask first | — | — | Unmatches a record so it cannot sync until someone maps it again; mapping it again undoes it. |
| Close leave case (`close_leave_case`) | Inside the organization | Propose | — | — | Closes a leave case under the leave approval grant, so it runs as the person who approves it; the days it drew down stay drawn down. |
| Close order (`close_order`) | Inside the organization | Propose | — | — | Settles an order for good; nothing reopens it, so only a person closes one. |
| Close worker training (`close_worker_training`) | Inside the organization | Propose | — | — | Closes a training assignment inside Trenova; the reason stays on the record, and the course is assigned again the same way. |
| Commit fuel purchase import (`commit_fuel_purchase_import`) | Inside the organization | Ask first | — | — | Adds a statement's purchases inside Trenova, which the IFTA return counts; nothing is sent, but undoing it means deleting each purchase. |
| Confirm accounting mapping proposals (`confirm_accounting_mapping_proposals`) | Inside the organization | Ask first | — | — | Decides which accounts and records Trenova's documents post to in the books, so a person approves it; clearing or changing a mapping undoes it. |
| Correct fuel purchase (`correct_fuel_purchase`) | Inside the organization | Ask first | A purchase read from a receipt or statement someone outside sent is proposed, since it is a tax record the IFTA return is computed from. | — | Changes a fuel purchase inside Trenova; nothing is sent, and a later correction changes it back. |
| Correct IFTA mileage entry (`correct_ifta_mileage_entry`) | Inside the organization | Ask first | Miles read from a log or a message someone outside sent are proposed, since the IFTA return is computed from them. | — | Changes jurisdiction miles inside Trenova; nothing is sent, and a later correction changes them back. |
| Create accounting reference record (`create_accounting_reference_record`) | Inside the organization | Ask first | — | — | Creates a record in the organization's own accounting system; Trenova cannot delete it again, so a person approves it. |
| Create carrier invoice match (`create_carrier_invoice_match`) | Inside the organization | Automatic | — | — | Records how a carrier's invoice compares with the expected cost inside Trenova; nothing is paid until a person accepts it. |
| Create commodity (`create_commodity`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Adds a commodity inside Trenova that shipments can name; nothing is sent and it can be made inactive. |
| Create dashboard (`create_dashboard`) | Inside the organization | Automatic | — | — | Saves a report dashboard colleagues can open; nothing leaves the organization. |
| Create hazardous material (`create_hazardous_material`) | Inside the organization | Propose | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Adds a hazardous material inside Trenova; nothing is sent. Its class, placarding and emergency contact decide what drivers carry and post, so a person checks every one against the DOT table before it is saved. |
| Create location (`create_location`) | Inside the organization | Ask first | — | — | A new location is where colleagues will book freight, and its address is usually read from a document someone outside sent, so a person approves it first. |
| Create order (`create_order`) | Inside the organization | Ask first | — | — | Opens an empty order inside Trenova; nothing is billed or sent, and an order opened in error is canceled. |
| Create recurring shipment (`create_recurring_shipment`) | Inside the organization | Ask first | — | — | Sets up a schedule inside Trenova that creates shipments later; nothing is sent, and pausing the series stops it. |
| Create report (`create_report`) | The caller's own records, inside the organization | Automatic | Each call is classified by what it reaches. A call on the caller's own records runs unasked while they are present. | — | A private report is a saved query on the caller's own list; a shared one appears on every colleague's Reports page and waits for approval. |
| Create shipment (`create_shipment`) | Inside the organization | Ask first | — | — | A new load commits a customer's freight and the money that follows, so no desk books one unattended. |
| Create table change alert (`create_table_change_alert`) | Inside the organization | Automatic | — | — | Creates an alert whose notices go to people inside the organization. |
| Create tractor (`create_tractor`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Adds a tractor inside Trenova that dispatch can then assign; nothing is sent and it can be marked Sold or out of service. |
| Create trailer (`create_trailer`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Adds a trailer inside Trenova that dispatch can then assign; nothing is sent and it can be marked Sold or out of service. |
| Delete dashboard (`delete_dashboard`) | Inside the organization | Propose | — | — | Removes a dashboard colleagues may open and nothing brings it back, so a person always decides. |
| Delete documents (`delete_documents`) | Inside the organization | Propose | — | — | Removes documents and their stored files from Trenova for good; nothing is sent, but nothing restores them either. |
| Delete employment verification (`delete_employment_verification`) | Inside the organization | Propose | — | — | Removes a previous employer from the qualification file; the audit trail keeps what was removed. |
| Delete fuel purchase (`delete_fuel_purchase`) | Inside the organization | Propose | — | — | Removes a tax record the IFTA return is computed from, and nothing brings it back but entering it again, so a person always decides. |
| Delete IFTA mileage entry (`delete_ifta_mileage_entry`) | Inside the organization | Propose | — | — | Removes miles the IFTA return is computed from, and nothing brings them back but entering them again, so a person always decides. |
| Delete IFTA return (`delete_ifta_return`) | Inside the organization | Ask first | — | — | Removes a draft return inside Trenova that nothing was filed from; generating it again recomputes the same figures. |
| Delete leave day (`delete_leave_day`) | Inside the organization | Propose | — | — | Removes a recorded day of leave; the audit trail keeps what was removed, and record_leave_day records it again. |
| Delete performance review (`delete_performance_review`) | Inside the organization | Propose | — | — | Removes a draft review nobody has seen; the audit trail keeps what was removed. |
| Delete report (`delete_report`) | Inside the organization | Propose | — | — | Removes a saved report colleagues may run and nothing brings it back, so a person always decides. |
| Delete report schedule (`delete_report_schedule`) | Inside the organization | Propose | — | — | Stops a report reaching the people it was scheduled for; nothing brings the schedule back but setting it up again, so a person always decides. |
| Delete safety violation (`delete_safety_violation`) | Inside the organization | Propose | — | — | Removes a cited violation inside Trenova; the audit trail keeps what was removed, and it is cited again with record_safety_violation. |
| Delete shipment comment (`delete_shipment_comment`) | Inside the organization | Ask first | — | — | Removes a note from a shipment's own thread inside Trenova; the text is gone, so a person confirms it. |
| Delete table change alert (`delete_table_change_alert`) | Inside the organization | Ask first | — | — | Removes an alert whose notices went to the person who owns it; create_table_change_alert sets it up again. |
| Delete worker injury (`delete_worker_injury`) | Inside the organization | Propose | — | — | Removes a case from the OSHA log; the audit trail keeps what was removed, and the number is not given out again. |
| Delete worker recognition (`delete_worker_recognition`) | Inside the organization | Propose | — | — | Removes a recognition from a driver's record; it is given again the same way. |
| Delete worker safety event (`delete_worker_safety_event`) | Inside the organization | Propose | — | — | Removes an event recorded in error from a driver's safety record; the audit trail keeps what was deleted, and it cannot be put back. |
| Detach order shipment (`detach_order_shipment`) | Inside the organization | Ask first | — | — | Moves a shipment onto a new order inside Trenova; nothing is sent, and attach_order_shipments puts it back. |
| Detach pay event from settlement (`detach_pay_event_from_settlement`) | Inside the organization | Automatic | — | — | Moves pay a driver already earned between the pool and a draft settlement inside Trenova; nothing is paid until a person approves it. |
| Discard capture batch (`discard_capture_batch`) | Inside the organization | Propose | — | — | Closes a scanned stack inside Trenova and drops its unfiled pages; nothing is sent, but nothing brings the pages back, so it is always proposed. |
| Discard capture item (`discard_capture_item`) | Inside the organization | Ask first | — | — | Marks one scanned document not worth filing inside Trenova; nothing is sent, and its pages are removed with the stack's other unfiled pages. |
| Discard fuel purchase import (`discard_fuel_purchase_import`) | Inside the organization | Ask first | — | — | Closes a staged statement inside Trenova without adding any purchase; it can be staged again. |
| Discard rate import (`discard_rate_import`) | Inside the organization | Ask first | — | — | Closes a staged rate sheet inside Trenova without changing any rate; the sheet can be uploaded again. |
| Dismiss accounting drift (`dismiss_accounting_drift`) | Inside the organization | Ask first | — | — | Closes a difference with the books without fixing it, so a person approves it. |
| Dismiss insight (`dismiss_insight`) | Inside the organization | Automatic | — | — | Dismisses an insight inside Trenova; it can be restored. |
| Dismiss watchtower item (`dismiss_watchtower_item`) | Inside the organization | Ask first | — | — | Takes an item off the shared attention feed inside Trenova; nothing is sent and the record behind it is unchanged. |
| Draft manual journal (`draft_manual_journal`) | Inside the organization | Ask first | A journal drafted from what someone outside wrote is proposed, since its lines are what an approver books. | — | Saves a draft journal inside Trenova; nothing reaches the ledger until a person submits, approves and posts it, and a draft is cancelled the same way. |
| Draft performance review (`draft_performance_review`) | Inside the organization | Ask first | — | — | Saves the reviewer's draft inside Trenova; the worker sees nothing until the reviewer submits it, and the draft is edited again the same way. |
| Draft rate agreement (`draft_rate_agreement`) | Inside the organization | Ask first | An agreement drafted from a rate sheet or a message someone outside sent is proposed, since its lanes are what the organization will charge. | — | Saves a draft agreement inside Trenova that prices nothing until a person approves it; archive_rate_agreement retires it. |
| Duplicate rate agreement (`duplicate_rate_agreement`) | Inside the organization | Ask first | An agreement drafted from a rate sheet or a message someone outside sent is proposed, since its lanes are what the organization will charge. | — | Saves a draft copy of an agreement inside Trenova that prices nothing until a person approves it. |
| Duplicate shipment (`duplicate_shipment`) | Inside the organization | Ask first | — | — | Creates new shipments inside Trenova from one already saved; nothing is sent, and a copy made in error is canceled. |
| End worker shift assignment (`end_worker_shift_assignment`) | Inside the organization | Ask first | — | — | Ends a worker's standing schedule inside Trenova; nothing is sent, and assign_worker_shift puts them back on it. |
| Escalate detention (`escalate_detention`) | Inside the organization | Automatic | — | — | Hands a detention clock to a person inside the organization. |
| Evaluate service failures (`evaluate_service_failures`) | Inside the organization | Automatic | — | — | Opens service failures from stop actuals inside Trenova; an open failure sends nothing. |
| File capture items (`file_capture_items`) | Inside the organization | Ask first | — | — | Files scanned pages as documents on records inside Trenova, as the person it acts for; nothing is sent, and a misfiled document can be deleted. |
| Finalize DOT random draw (`finalize_dot_random_draw`) | Inside the organization | Propose | — | — | Locks the round's selections as the record an auditor reads; it cannot be undone, only cancelled, so a person approves it. |
| Flag for manual review (`flag_for_manual_review`) | Inside the organization | Automatic | — | — | Records an exception on the run for a person inside the organization to work. |
| Forget memory (`forget_memory`) | Inside the organization | Automatic | — | — | Retires an agent memory inside Trenova. |
| Fork report (`fork_report`) | Inside the organization | Automatic | — | — | Saves a copy of a report inside Trenova; nothing leaves the organization. |
| Generate carrier settlement batch (`generate_carrier_settlement_batch`) | Inside the organization | Automatic | — | — | Drafts the period's carrier settlements from cost already accrued inside Trenova; nothing is paid until a person approves and posts them. |
| Generate driver settlement (`generate_driver_settlement`) | Inside the organization | Automatic | — | — | Drafts a settlement from pay already accrued inside Trenova; nothing is paid until a person approves it, and hands-off approval is only the settlement control's clean-settlement rule. |
| Generate driver settlement batch (`generate_driver_settlement_batch`) | Inside the organization | Automatic | — | — | Drafts the period's settlements from pay already accrued inside Trenova; nothing is paid until a person approves them, and hands-off approval is only the settlement control's clean-settlement rule. |
| Generate IFTA return (`generate_ifta_return`) | Inside the organization | Ask first | — | — | Computes a draft return inside Trenova; nothing is filed or sent, and delete_ifta_return removes the draft. |
| Generate invoice PDF (`generate_invoice_pdf`) | Inside the organization | Automatic | — | — | Renders the invoice as it stands into its filed PDF; nothing is sent, and it is refused whenever rendering would email the customer. |
| Generate rate confirmation (`generate_rate_confirmation`) | Inside the organization | Automatic | A first revision is generated as far as the agent allows; one that would void a revision already standing, which the carrier may hold or have signed, is a proposal a person decides, as is one the service would refuse or that cannot be read. | — | Renders the agreement and files it on the shipment inside Trenova; nothing reaches the carrier until it is sent, and voiding the revision undoes it. |
| Generate recurring shipment (`generate_recurring_shipment`) | Inside the organization | Ask first | — | — | Creates one shipment inside Trenova from the series' source; nothing is sent, and a shipment made in error is canceled. |
| Hold billing queue item (`hold_billing_queue_item`) | Inside the organization | Automatic | Its notes are what a biller or operations reads next, so a run that has read outside text proposes it rather than writing it. | — | Parks an item inside Trenova with a note; it creates no money and is undone by moving the item on. |
| Ignore accounting inbound change (`ignore_accounting_inbound_change`) | Inside the organization | Ask first | — | — | Keeps a payment the books hold out of Trenova for good, so a person approves it. |
| Import sourced carrier (`import_sourced_carrier`) | Inside the organization | Ask first | — | — | Creates a carrier inside Trenova from the provider's profile; nothing is sent to it, and one added in error is made inactive. |
| Link EDI carrier invoice to carrier (`link_edi_carrier_invoice_to_carrier`) | Inside the organization | Automatic | — | — | Names which carrier an inbound invoice belongs to inside Trenova; nothing is paid and it is relinked the same way. |
| Link inbound message (`link_inbound_message`) | Inside the organization | Automatic | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Links an inbound message to a record inside Trenova; the mailbox decides how far it may run. |
| Locate tractor (`locate_tractor`) | Inside the organization | Ask first | — | — | Moves where Trenova thinks a tractor is; nothing is sent and locating it again moves it back. |
| Log employment verification request (`log_employment_verification_request`) | Inside the organization | Ask first | — | — | Stamps the request date or counts a follow-up inside Trenova; nothing is sent to the employer. |
| Manage billing transfer run (`manage_billing_transfer_run`) | Inside the organization | Ask first | — | — | Stops or reruns a transfer inside Trenova; nothing is invoiced or sent, and a stopped run is retried the same way. |
| Mark carrier intel reviewed (`mark_carrier_intel_reviewed`) | Inside the organization | Propose | — | — | Puts a person's name on a carrier risk review; only that person can say they reviewed it. |
| Mark inbound message (`mark_inbound_message`) | Inside the organization | Automatic | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Settles an inbound message inside Trenova; the mailbox decides how far it may run. |
| Move billing item to exception (`move_billing_item_to_exception`) | Inside the organization | Automatic | Its notes are what a biller or operations reads next, so a run that has read outside text proposes it rather than writing it. | — | Marks an item for a biller to resolve inside Trenova; it creates no money and a biller moves it back when it is resolved. |
| Open invoice dispute (`open_invoice_dispute`) | Inside the organization | Ask first | A dispute opened from what a customer wrote is proposed, since the notes and the amount are what collections acts on. | — | Marks an invoice Disputed inside Trenova with the customer's reason; it moves no money and is withdrawn by withdraw_invoice_dispute. |
| Open leave case (`open_leave_case`) | Inside the organization | Ask first | — | — | Files a pending leave request inside Trenova; nothing is decided or sent, and the case is corrected with update_leave_case. |
| Open worker safety event (`open_worker_safety_event`) | Inside the organization | Ask first | — | — | Adds an event to a driver's safety record inside Trenova; nothing is sent to the driver, and it is corrected or deleted while it is open. |
| Pause accounting sync (`pause_accounting_sync`) | Inside the organization | Automatic | — | — | Holds what is waiting to go to the books; nothing is sent, changed or lost, and resuming sends it on. |
| Pin shipment comment (`pin_shipment_comment`) | Inside the organization | Automatic | — | — | Orders a shipment's own comment thread inside Trenova; unpinning undoes it and nothing is sent. |
| Place shipment hold (`place_shipment_hold`) | Inside the organization | Automatic | — | — | Places a hold on a shipment inside Trenova; no customer or EDI notice is sent. |
| Place worker dispatch hold (`place_worker_dispatch_hold`) | Inside the organization | Automatic | — | — | Keeps a driver off new freight inside Trenova; the driver is not messaged. |
| Propose shift swap (`propose_shift_swap`) | Inside the organization | Ask first | — | — | Opens a request inside Trenova that changes nobody's schedule until it is approved; withdraw_shift_swap takes it back. |
| Raise exception (`raise_exception`) | Inside the organization | Automatic | — | — | Records an exception against the run itself for a person inside the organization. |
| Recalculate carrier settlement (`recalculate_carrier_settlement`) | Inside the organization | Automatic | — | — | Rebuilds a draft from the records it is computed from inside Trenova; nothing is paid until a person approves it. |
| Recalculate driver settlement (`recalculate_driver_settlement`) | Inside the organization | Automatic | — | — | Rebuilds a draft from the records it is computed from inside Trenova; nothing is paid until a person approves it. |
| Recalculate move jurisdiction miles (`recalculate_move_jurisdiction_miles`) | Inside the organization | Ask first | — | — | Re-derives one move's jurisdiction miles inside Trenova from its stops; nothing is sent to a customer or carrier, and running it again gives the same miles. |
| Recalculate shipment distance (`recalculate_shipment_distance`) | Inside the organization | Automatic | — | — | Re-derives the shipment's move distances inside Trenova from its stops; nothing is sent and running it again gives the same miles. |
| Recompute IFTA return (`recompute_ifta_return`) | Inside the organization | Automatic | — | — | Re-derives a draft return's figures inside Trenova from what is on file; nothing is filed, and running it again gives the same figures. |
| Record employment verification (`record_employment_verification`) | Inside the organization | Ask first | — | — | Adds a previous employer to the qualification file inside Trenova; nothing is sent to the employer, and it is corrected or removed the same way. |
| Record fuel purchase (`record_fuel_purchase`) | Inside the organization | Ask first | A purchase read from a receipt or statement someone outside sent is proposed, since it is a tax record the IFTA return is computed from. | — | Adds a fuel purchase inside Trenova that the next IFTA return counts; nothing is sent, and delete_fuel_purchase removes it. |
| Record IFTA mileage entry (`record_ifta_mileage_entry`) | Inside the organization | Ask first | Miles read from a log or a message someone outside sent are proposed, since the IFTA return is computed from them. | — | Adds jurisdiction miles inside Trenova that the next IFTA return counts; nothing is sent, and delete_ifta_mileage_entry removes them. |
| Record leave day (`record_leave_day`) | Inside the organization | Ask first | — | — | Records a day taken against a leave case inside Trenova; nothing is sent, and update_leave_day or delete_leave_day corrects it. |
| Record safety violation (`record_safety_violation`) | Inside the organization | Ask first | — | — | Adds a cited violation to a safety event inside Trenova; nothing is sent, and it is corrected or removed the same way. |
| Record shipment permit (`record_shipment_permit`) | Inside the organization | Ask first | — | — | Records a permit on the shipment inside Trenova, which can release a dispatch hold; nothing is sent to the state, and update_shipment_permit voids it. |
| Record stop actual (`record_stop_actual`) | Inside the organization | Automatic | — | — | Records arrival and departure times on a stop; no model-written text leaves the organization. |
| Record training completion (`record_training_completion`) | Inside the organization | Ask first | — | — | Records a completion on the worker's training record inside Trenova; nothing is sent, and a completion recorded in error is corrected by a person. |
| Record worker credential (`record_worker_credential`) | Inside the organization | Ask first | — | — | Adds a credential to the worker's file inside Trenova, which can change whether they may be dispatched; nothing is sent, and archive_worker_credential retires it. |
| Record worker injury (`record_worker_injury`) | Inside the organization | Ask first | — | — | Adds a case to the organization's OSHA log inside Trenova; nothing is sent, and it is corrected with update_worker_injury. |
| Redate accounting sync (`redate_accounting_sync`) | Inside the organization | Ask first | — | — | Changes the date the organization's books record for a document, so a person approves it. |
| Refresh accounting reference data (`refresh_accounting_reference_data`) | Inside the organization | Automatic | — | — | Reads the accounting system and refreshes Trenova's suggestions; it writes nothing to the books and leaves confirmed mappings alone. |
| Reject accounting mapping proposal (`reject_accounting_mapping_proposal`) | Inside the organization | Ask first | — | — | Unmatches a proposal so it is not used; choosing a record for the mapping undoes it. |
| Reject carrier invoice match (`reject_carrier_invoice_match`) | Inside the organization | Automatic | Its note is what accounts payable reads about the carrier's invoice, so a run that has read outside text proposes it. | — | Marks a carrier's invoice as not payable inside Trenova; nothing is paid and a corrected invoice is matched afresh. |
| Reject carrier settlement (`reject_carrier_settlement`) | Inside the organization | Automatic | Its text is what payroll or the payee reads next, so a run that has read outside text proposes it rather than writing it. | — | Returns a settlement to draft with a note inside Trenova; nothing is paid and it is submitted again once fixed. |
| Reject driver settlement (`reject_driver_settlement`) | Inside the organization | Automatic | Its text is what payroll or the payee reads next, so a run that has read outside text proposes it rather than writing it. | — | Returns a settlement to draft with a note inside Trenova; nothing is paid and it is submitted again once fixed. |
| Reject invoice adjustment (`reject_invoice_adjustment`) | Inside the organization | Propose | — | — | Turns down an adjustment an approver was asked to sign off; the approval decision is a person's. |
| Reject rate agreement (`reject_rate_agreement`) | Inside the organization | Propose | — | — | Returns an agreement to its author with the reviewer's reason; it prices nothing either way, but the review is a person's decision. |
| Release driver pay event (`release_driver_pay_event`) | Inside the organization | Automatic | — | — | Returns held pay to the settlement pool inside Trenova; nothing is paid until a settlement is approved, and the pay is held again the same way. |
| Release shipment hold (`release_shipment_hold`) | Inside the organization | Automatic | — | — | Releases a hold on a shipment inside Trenova; no customer or EDI notice is sent. |
| Remember (`remember`) | Inside the organization | Automatic | An Instruction or a Correction recorded after the run read text from outside the organization waits for a person's approval; a Fact is recorded and stays marked as drawn from outside text. | Carries outside text into later runs | Saves a memory later runs read, so it keeps the taint of the run that wrote it. |
| Request accounting backfill (`request_accounting_backfill`) | Inside the organization | Ask first | — | — | Sends historical documents to the organization's books, some of which may already be there by hand, and cannot be called back; a person who manages the integration approves it. |
| Request leave certification (`request_leave_certification`) | Inside the organization | Ask first | — | — | Starts the certification clock on a leave case inside Trenova; nothing is sent to the worker, and a later request restarts it. |
| Reset report fork (`reset_report_fork`) | Inside the organization | Ask first | — | — | Rewrites a report the person owns from the built-in catalog; nothing is sent, but the person's changes to it are lost. |
| Resolve bank receipt work item (`resolve_bank_receipt_work_item`) | Inside the organization | Automatic | — | — | Closes a reconciliation work item without moving money; the note is read inside the organization. |
| Resolve carrier intel event (`resolve_carrier_intel_event`) | Inside the organization | Automatic | — | — | Closes a carrier finding inside Trenova. |
| Resolve fuel purchase import rows (`resolve_fuel_purchase_import_rows`) | Inside the organization | Ask first | — | — | Re-matches held statement rows inside Trenova and, for a feed, posts the purchases that now resolve; nothing is sent. |
| Resolve invoice dispute (`resolve_invoice_dispute`) | Inside the organization | Propose | — | — | Records the outcome of a customer's dispute and releases the invoice back to collections; the outcome is a biller's call, so the agent proposes it. |
| Resolve shipment comment (`resolve_shipment_comment`) | Inside the organization | Automatic | — | — | Marks a shipment's own comment settled inside Trenova; reopening undoes it and nothing is sent. |
| Restore document version (`restore_document_version`) | Inside the organization | Ask first | — | — | Changes which stored version of a document Trenova shows; nothing is sent or removed. |
| Restore insight (`restore_insight`) | Inside the organization | Automatic | — | — | Returns a finding to the active list inside Trenova; nothing is sent and dismiss_insight takes it off again. |
| Resume accounting sync (`resume_accounting_sync`) | Inside the organization | Ask first | — | — | Releases everything held to the organization's books at once, and a person paused it for a reason, so a person approves it; what is sent cannot be called back. |
| Retry accounting sync (`retry_accounting_sync`) | Inside the organization | Automatic | — | — | Sends again, to the organization's own books, documents Trenova already decided to send; the accounting system recognizes a repeat by its request id, so nothing is entered twice, and a document held for release stays held. |
| Revise manual journal draft (`revise_manual_journal_draft`) | Inside the organization | Ask first | A journal rewritten from what someone outside wrote is proposed, since its lines are what an approver books. | — | Changes a draft journal inside Trenova; nothing reaches the ledger until a person submits, approves and posts it, and a later revision changes it back. |
| Revise rate agreement draft (`revise_rate_agreement_draft`) | Inside the organization | Ask first | An agreement drafted from a rate sheet or a message someone outside sent is proposed, since its lanes are what the organization will charge. | — | Changes a draft agreement inside Trenova that prices nothing yet; a later revision changes it back. |
| Run DOT random draw (`run_dot_random_draw`) | Inside the organization | Ask first | — | — | Draws a draft round inside Trenova; nobody is told and nothing is final until a person finalizes it, and cancel_dot_random_draw withdraws it. |
| Run rate simulation (`run_rate_simulation`) | Inside the organization | Automatic | — | — | Saves a simulation inside Trenova and replays past shipments in the background; no shipment, rate or invoice changes. |
| Save invoice adjustment draft (`save_invoice_adjustment_draft`) | Inside the organization | Automatic | A draft written from what a customer sent is proposed, since its reason and amounts are what a biller submits. | — | Saves a draft adjustment inside Trenova; it credits nothing until a person submits it, and a later save rewrites it. |
| Save table view (`save_table_view`) | The caller's own records, inside the organization | Automatic | Each call is classified by what it reaches. A call on the caller's own records runs unasked while they are present. | — | A private view is the caller's own picker entry; a shared one appears for every colleague, and nothing leaves the organization. |
| Schedule DOT test (`schedule_dot_test`) | Inside the organization | Ask first | — | — | Files a scheduled collection inside Trenova. The driver is sent nothing, no result is recorded, and cancel_dot_test withdraws it. |
| Send billing item back to ops (`send_billing_item_back_to_ops`) | Inside the organization | Automatic | Its notes are what a biller or operations reads next, so a run that has read outside text proposes it rather than writing it. | — | Returns an item to operations with a note on its shipment inside Trenova; it creates no money and the item comes back when operations fixes it. |
| Set accounting mapping (`set_accounting_mapping`) | Inside the organization | Ask first | — | — | Decides which account or record Trenova's invoices, payments and bills will post to, so a person approves it; clearing or changing it undoes it. |
| Set recurring shipment status (`set_recurring_shipment_status`) | Inside the organization | Ask first | — | — | Starts or stops a schedule inside Trenova; nothing is sent, and the status is set back the same way. |
| Set table change alert status (`set_table_change_alert_status`) | Inside the organization | Automatic | — | — | Pauses or resumes an alert whose notices go to the person who owns it; the opposite call undoes it. |
| Set worker availability preference (`set_worker_availability_preference`) | Inside the organization | Ask first | — | — | Records a worker's own stated preference inside Trenova; nothing is sent, and it is set again the same way. |
| Share invoice (`share_invoice`) | Inside the organization | Propose | — | — | Notifies and emails colleagues in the name of the person who shares it, with a note they read as theirs; only that person sends it. |
| Skip accounting sync (`skip_accounting_sync`) | Inside the organization | Ask first | — | — | Leaves a document out of the organization's books for good; nothing sends it again, so a person approves it. |
| Split move at relay (`split_move_at_relay`) | Inside the organization | Propose | — | — | Adds a move and turns a delivery into a relay inside Trenova; the two new windows are times only a person can vouch for, so a person always approves it. |
| Start performance review (`start_performance_review`) | Inside the organization | Ask first | — | — | Opens a draft review inside Trenova; the worker sees nothing, and delete_performance_review removes the draft. |
| Start worker checklist (`start_worker_checklist`) | Inside the organization | Ask first | — | — | Opens a checklist on the worker inside Trenova; nothing is sent, and cancel_worker_checklist cancels it. |
| Submit carrier settlement (`submit_carrier_settlement`) | Inside the organization | Automatic | — | — | Moves a draft into the approval queue inside Trenova; nothing is paid and a reviewer sends it back to draft. |
| Submit driver settlement (`submit_driver_settlement`) | Inside the organization | Automatic | — | — | Moves a draft into the approval queue inside Trenova; nothing is paid and a reviewer sends it back to draft. |
| Submit manual journal (`submit_manual_journal`) | Inside the organization | Ask first | — | — | Moves a draft journal into the approval queue inside Trenova; an approver decides and it books nothing until a person posts it. |
| Submit rate agreement (`submit_rate_agreement`) | Inside the organization | Propose | — | — | Moves a draft agreement into review inside Trenova; it prices nothing until a person approves it. |
| Transfer shipment ownership (`transfer_shipment_ownership`) | Inside the organization | Ask first | — | — | Changes who owns a shipment inside Trenova; nothing is sent, and it is transferred back the same way. |
| Transfer to billing (`transfer_to_billing`) | Inside the organization | Automatic | — | — | Hands delivered shipments to the billing queue inside Trenova, by the checks the transfer dialog makes; a biller, or the organization's own auto-approve rule, still decides every item. |
| Transition item to in review (`transition_item_to_in_review`) | Inside the organization | Automatic | An item on hold was held there by a person or a rule, so moving one into review is a proposal a person decides; an item in any other state moves as far as the agent allows, and one that cannot be read waits for a person. | — | Moves the run's billing queue item into review inside Trenova; nothing is sent anywhere. |
| Triage bank receipt work item (`triage_bank_receipt_work_item`) | Inside the organization | Automatic | — | — | Changes who works a reconciliation item and says it is being looked into; it moves no money and the item is reassigned the same way. |
| Unassign moves (`unassign_moves`) | Inside the organization | Automatic | Only one move at a time, still freshly assigned, is taken off its driver without a decision; several moves, a move the service would refuse and a move that cannot be read each wait for a person. | — | Takes the driver off a move that has not started, inside Trenova; the driver is told the load was taken off them but sees no text the model wrote, and assigning the move again undoes it. |
| Uncancel shipment (`uncancel_shipment`) | Inside the organization | Ask first | — | — | Puts a canceled shipment back to New inside Trenova; nothing is sent, and canceling it again undoes it. |
| Unpin shipment comment (`unpin_shipment_comment`) | Inside the organization | Automatic | — | — | Orders a shipment's own comment thread inside Trenova; pinning again undoes it and nothing is sent. |
| Update carrier status (`update_carrier_status`) | Inside the organization | Ask first | — | — | Changes whether carriers are offered for booking inside Trenova; nothing is sent. |
| Update commodity (`update_commodity`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Changes a commodity inside Trenova; nothing is sent, and a later change puts it back. |
| Update commodity status (`update_commodity_status`) | Inside the organization | Ask first | — | — | Changes whether commodities are offered for booking inside Trenova; nothing is sent. |
| Update customer status (`update_customer_status`) | Inside the organization | Ask first | — | — | Changes whether customers are offered for booking inside Trenova; nothing is sent. |
| Update DOT random selection (`update_dot_random_selection`) | Inside the organization | Ask first | — | — | Updates a selection on a round inside Trenova; nothing is sent to the driver, and the selection is updated again the same way. |
| Update employment verification (`update_employment_verification`) | Inside the organization | Ask first | — | — | Updates the previous employer's record in the qualification file inside Trenova; nothing is sent, and it is corrected again the same way. |
| Update hazardous material (`update_hazardous_material`) | Inside the organization | Propose | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Changes a hazardous material inside Trenova; nothing is sent. What drivers placard and who they call follows it, so a person approves every change. |
| Update hazardous material status (`update_hazardous_material_status`) | Inside the organization | Ask first | — | — | Changes whether hazardous materials are offered inside Trenova; nothing is sent. |
| Update invoice draft (`update_invoice_draft`) | Inside the organization | Ask first | Changing who the invoice is emailed to is a proposal a person decides; any other change runs once a person approves it. Outside content could put its own wording or recipients on an invoice the customer receives. | — | Edits a draft nobody outside has seen; the customer sees it only when a person sends it, but the recipients decide where it goes then. |
| Update leave case (`update_leave_case`) | Inside the organization | Ask first | — | — | Corrects a leave case inside Trenova; nothing is decided or sent, and it is corrected again the same way. |
| Update leave day (`update_leave_day`) | Inside the organization | Ask first | — | — | Corrects a recorded day of leave inside Trenova; nothing is sent, and it is corrected again the same way. |
| Update location (`update_location`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Changes a location inside Trenova; nothing is sent. Its address is often read from a document someone outside sent, so such a change is proposed. |
| Update location status (`update_location_status`) | Inside the organization | Ask first | — | — | Changes whether locations are offered for booking inside Trenova; nothing is sent. |
| Update move status (`update_move_status`) | Inside the organization | Ask first | Canceling a move ends it for good, so a cancellation is a proposal a person decides; any other status runs once a person approves it. | — | Moves a move's status forward inside Trenova, which re-derives its shipment's status and, on completion, releases its equipment; a move never moves back, so it is not undone by running it again. |
| Update recurring shipment (`update_recurring_shipment`) | Inside the organization | Ask first | — | — | Changes when a series creates shipments inside Trenova; nothing is sent, and the old schedule is set back the same way. |
| Update report (`update_report`) | Inside the organization | Automatic | — | — | Changes a saved report colleagues may open; nothing leaves the organization. |
| Update safety violation (`update_safety_violation`) | Inside the organization | Ask first | — | — | Corrects a cited violation inside Trenova; nothing is sent, and it is corrected again the same way. |
| Update service failure (`update_service_failure`) | Inside the organization | Ask first | — | — | Edits a service failure's own notes and codes inside Trenova; the overrides shape a later EDI 214, so a person approves each edit. |
| Update shipment permit (`update_shipment_permit`) | Inside the organization | Ask first | — | — | Changes a permit on the shipment inside Trenova, which can place or lift a dispatch hold; nothing is sent to the state, and it is changed again the same way. |
| Update table change alert (`update_table_change_alert`) | Inside the organization | Automatic | — | — | Changes an alert whose notices go to the person who owns it; nothing else changes. |
| Update tractor (`update_tractor`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Changes a tractor inside Trenova; nothing is sent, and a later change puts it back. |
| Update tractor status (`update_tractor_status`) | Inside the organization | Automatic | — | — | Changes a tractor's status inside Trenova. |
| Update trailer (`update_trailer`) | Inside the organization | Ask first | A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Changes a trailer inside Trenova; nothing is sent, and a later change puts it back. |
| Update trailer status (`update_trailer_status`) | Inside the organization | Automatic | — | — | Changes a trailer's status inside Trenova. |
| Update worker checklist item (`update_worker_checklist_item`) | Inside the organization | Ask first | — | — | Settles or reopens a checklist item inside Trenova; nothing is sent, and a settled item is reopened the same way. |
| Update worker credential (`update_worker_credential`) | Inside the organization | Ask first | — | — | Corrects a credential inside Trenova, which can change whether the worker may be dispatched; nothing is sent, and it is corrected again the same way. |
| Update worker injury (`update_worker_injury`) | Inside the organization | Ask first | — | — | Corrects a case on the OSHA log inside Trenova; nothing is sent, and it is corrected again the same way. |
| Update worker safety event (`update_worker_safety_event`) | Inside the organization | Ask first | — | — | Corrects an event on a driver's safety record inside Trenova; nothing is sent, and the event is corrected again the same way. |
| Verify carrier equipment (`verify_carrier_equipment`) | Inside the organization | Ask first | — | — | Records an equipment check inside Trenova after a provider lookup; nothing is sent to the carrier. |
| Withdraw invoice dispute (`withdraw_invoice_dispute`) | Inside the organization | Ask first | A withdrawal read from what a customer wrote is proposed, since it sends the invoice back to collections. | — | Closes a dispute case inside Trenova with no outcome; it moves no money and a new case can be opened with open_invoice_dispute. |

## Seen by a customer

Changes something a customer can see.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Update shipment hold (`update_shipment_hold`) | Inside the organization, seen by a customer | Automatic | Each call is classified by what it reaches. | — | Changes a hold inside Trenova; one made visible to the customer is shown to them, so that call is held to the customer-visible ceiling. |

## Seen by a driver

Changes something a driver can see.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Add shipment comment (`add_shipment_comment`) | Inside the organization, seen by a customer, seen by a driver | Automatic | Each call is classified by what it reaches. | Carries outside text into later runs | An internal note stays inside the organization; a customer or driver note is read outside it, so its visibility argument decides. A note written on its own after the run read outside text is marked as drawn from it. |
| Approve shift swap (`approve_shift_swap`) | Seen by a driver | Propose | — | — | Changes who works a day; approving is the office's decision, so only a person makes it. |
| Assign required worker training (`assign_required_worker_training`) | Seen by a driver | Ask first | — | — | Assigns the courses the worker's driver type requires; the driver sees them in Dash, and cancel_worker_training withdraws one. |
| Assign worker training (`assign_worker_training`) | Seen by a driver | Ask first | — | — | Assigns courses the driver sees in Dash; cancel_worker_training withdraws one. |
| Cancel worker PTO (`cancel_worker_pto`) | Seen by a driver | Ask first | — | — | The worker is sent the cancellation reason by push notice and text message. |
| Edit shipment comment (`edit_shipment_comment`) | Inside the organization, seen by a customer, seen by a driver | Automatic | Each call is classified by what it reaches. | — | Edits a shipment's own comment inside Trenova; one made visible to a customer or a driver is read by them, so its visibility argument decides. |
| Give worker recognition (`give_worker_recognition`) | Seen by a driver | Ask first | — | — | The driver is shown the recognition and told of it in Dash when it is visible to them. |
| Hold driver pay event (`hold_driver_pay_event`) | Seen by a driver | Ask first | — | — | Defers a driver's pay and tells the driver why in the driver portal; it is released the same way. |
| Notify driver (`notify_driver`) | Seen by a driver | Ask first | — | — | Sends a driver a message the model wrote to their phone. |
| Reject shift swap (`reject_shift_swap`) | Seen by a driver | Ask first | — | — | Refuses a request the workers read in their portal; a person approves each refusal. |
| Reject worker PTO (`reject_worker_pto`) | Seen by a driver | Ask first | — | — | The worker is sent the rejection reason by push notice and text message. |
| Request credential renewal (`request_credential_renewal`) | Seen by a driver | Ask first | — | — | Sends a driver a renewal request the model wrote. |
| Request worker PTO (`request_worker_pto`) | Seen by a driver | Ask first | — | — | Files time off the driver sees in Dash; under a policy with no approval step it is booked and the driver told at once, and cancel_worker_pto withdraws it. |
| Start settlement dispute review (`start_settlement_dispute_review`) | Seen by a driver | Ask first | — | — | Marks the dispute as being worked inside Trenova; the driver sees its status in Dash, and resolving it moves it on. |
| Update worker PTO (`update_worker_pto`) | Seen by a driver | Ask first | — | — | Changes a pending request the driver sees in Dash; it is changed again the same way. |
| Withdraw shift swap (`withdraw_shift_swap`) | Seen by a driver | Ask first | — | — | Takes back a request the workers read in their portal; a person approves each withdrawal. |

## Sent outside the organization

Sends to someone outside the organization.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Accept EDI tender (`accept_edi_tender`) | Sent outside the organization | Propose | — | — | Commits the organization to haul the load, creates the shipment and sends the partner an acceptance they act on; it cannot be taken back. A person always decides. |
| Cancel EDI tender (`cancel_edi_tender`) | Sent outside the organization | Propose | — | — | Withdraws freight offered to another organization, which sees the tender canceled; a withdrawn tender is sent afresh, never reopened. A person always decides. |
| Cancel order (`cancel_order`) | Sent outside the organization | Propose | — | — | Cancels every live shipment on the order, which reaches carriers and trading partners; only a person cancels an order. |
| Cancel shipment (`cancel_shipment`) | Sent outside the organization | Ask first | — | — | Cancelling withdraws live tenders from carriers and sends the model's cancel reason to a linked partner over EDI. |
| Cancel tender (`cancel_tender`) | Sent outside the organization | Ask first | A tender waiting on review holds a carrier's acceptance, so withdrawing it is a proposal a person decides, as is one the service would refuse or that cannot be read; an active tender is withdrawn once a person approves it. | — | Withdraws the offers carriers outside the organization hold, whose answer links stop working; a withdrawn tender cannot be reopened, only tendered afresh. |
| Create customer (`create_customer`) | Inside the organization, sent outside the organization | Ask first | Each call is classified by what it reaches. A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Adds a customer inside Trenova that shipments can be booked for; nothing is sent. Setting who receives status emails decides where later emails go, so that call is held for a person. |
| Decline EDI tender (`decline_edi_tender`) | Sent outside the organization | Propose | — | — | Turns down freight a partner offered and sends them the decline; the partner moves the load elsewhere and it cannot be taken back. A person always decides. |
| Email customer (`email_customer`) | Sent outside the organization | Ask first | — | — | Emails the customer's contacts a message the model wrote. |
| Expire EDI tender (`expire_edi_tender`) | Sent outside the organization | Propose | — | — | Closes a tender on both sides of the trading relationship, which the other side sees; an expired tender is sent afresh, never reopened. A person always decides. |
| Replay EDI message (`replay_edi_message`) | Sent outside the organization | Propose | — | — | Sends a trading partner a document it already has, which it may act on a second time; a document sent cannot be recalled. A person always decides. |
| Reply to inbound message (`reply_to_inbound_message`) | Sent outside the organization | Ask first | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Replies to whoever wrote in with text the model composed. |
| Reprocess EDI inbound files (`reprocess_edi_inbound_files`) | Sent outside the organization | Propose | — | — | Turns what trading partners sent into tenders, status updates and invoices again, and sends each partner acknowledgments they act on. A person always decides. |
| Request missing docs (`request_missing_docs`) | Sent outside the organization | Ask first | — | — | Emails an outside party a request for paperwork in words the model wrote. |
| Resolve service failure (`resolve_service_failure`) | Sent outside the organization | Ask first | — | — | Resolving a failure generates an EDI 214 to the customer's trading partner carrying the reason chosen. |
| Retry EDI message delivery (`retry_edi_message_delivery`) | Sent outside the organization | Propose | — | — | Sends documents to trading partners outside the organization, who act on what they receive; a document sent cannot be recalled. A person always decides. |
| Review EDI tender change (`review_edi_tender_change`) | Sent outside the organization | Propose | — | — | Changes a load this organization committed to on another organization's word, and that organization sees the outcome. A person always decides. |
| Review EDI transfer change (`review_edi_transfer_change`) | Sent outside the organization | Propose | — | — | Changes a shipment on another organization's report, and that organization sees the outcome on its own shipment. A person always decides. |
| Review service failure (`review_service_failure`) | Sent outside the organization | Propose | — | — | Reviewing a failure is a person's attestation and may send an EDI 214 to the customer's trading partner; only a person approves it. |
| Schedule report (`schedule_report`) | Sent outside the organization | Ask first | — | — | Emails a report on a schedule to whatever addresses the call names, which may be outside the organization. |
| Send detention notice (`send_detention_notice`) | Sent outside the organization | Ask first | — | — | Sends the customer a detention notice that starts a charge. |
| Send EDI status update (`send_edi_status_update`) | Sent outside the organization | Propose | — | — | Sends a trading partner a shipment status they act on and pass to their own customers; a document sent cannot be recalled. A person always decides. |
| Send EDI tender (`send_edi_tender`) | Sent outside the organization | Propose | — | — | Offers a load to another organization, which may accept it and haul it on the terms tendered; a tender is withdrawn, never recalled. A person always decides. |
| Send invoice (`send_invoice`) | Sent outside the organization | Propose | — | — | Emails the customer their invoice; only a person sends it, to the recipients the customer's billing profile names. |
| Send invoice EDI (`send_invoice_edi`) | Sent outside the organization | Propose | — | — | Transmits the invoice to the customer's EDI trading partner, which cannot be called back; only a person sends it. |
| Send invoices (`send_invoices`) | Sent outside the organization | Propose | — | — | Emails several customers their invoices at once; only a person sends them, is named as who sent each, and sees every recipient before approving. |
| Send rate confirmation (`send_rate_confirmation`) | Sent outside the organization | Propose | — | — | Emails a binding agreement, with a link to sign it, to a carrier outside the organization; the recipients come from the carrier's record, and a sent email cannot be recalled, so a person decides every send. |
| Tender move to carriers (`tender_move_to_carriers`) | Sent outside the organization | Ask first | — | — | Offers the load to carriers outside the organization. |
| Tender move to routing guide (`tender_move_to_routing_guide`) | Sent outside the organization | Ask first | — | — | Offers the load to carriers outside the organization. |
| Update customer (`update_customer`) | Inside the organization, sent outside the organization | Ask first | Each call is classified by what it reaches. A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Changes a customer inside Trenova and leaves its billing and email profiles alone; nothing is sent. Changing who receives status emails is held for a person. |
| Update report schedule (`update_report_schedule`) | Sent outside the organization | Ask first | — | — | Changes where and when a report is emailed, which may be to addresses outside the organization. |
| Update shipment (`update_shipment`) | Sent outside the organization | Ask first | — | — | A changed shipment is sent as an EDI tender change to the trading partners it was tendered to. |
| Void service failure (`void_service_failure`) | Sent outside the organization | Propose | — | — | Voiding a failure is final and may send an EDI 214 to the customer's trading partner; only a person approves it. |

## Money

Moves or commits money.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Accept carrier invoice match (`accept_carrier_invoice_match`) | Money | Propose | — | — | Clears a carrier's invoice for payment; only a person approves what the organization pays. |
| Accept carrier invoice match with variance (`accept_carrier_invoice_match_with_variance`) | Money | Propose | — | — | Agrees to pay a carrier more or less than the load was expected to cost; only a person approves it. |
| Add carrier settlement adjustment (`add_carrier_settlement_adjustment`) | Money | Automatic | Its text is what payroll or the payee reads next, so a run that has read outside text proposes it rather than writing it. | — | Changes what the carrier will be paid on a settlement a person still approves, so it moves money; the line is removed the same way. |
| Add driver settlement adjustment (`add_driver_settlement_adjustment`) | Money | Automatic | Its text is what payroll or the payee reads next, so a run that has read outside text proposes it rather than writing it. | — | Changes what the driver will be paid on a settlement a person still approves, so it moves money; the line is removed the same way. |
| Add order charge (`add_order_charge`) | Money | Ask first | — | — | Adds to what the customer is billed; nothing is invoiced or sent, and remove_order_charge takes it off until it is invoiced. |
| Adjust escrow account (`adjust_escrow_account`) | Money | Propose | — | — | Moves money held in escrow for an owner-operator; only a person moves it. |
| Adjust worker PTO balance (`adjust_worker_pto_balance`) | Money | Propose | — | — | Changes how many paid days a worker holds, which the organization owes and may pay out, so a person approves it and it runs as them. |
| Amend rate agreement rules (`amend_rate_agreement_rules`) | Money | Propose | — | — | Changes the rates an agreement prices shipments at; only a person amends a live contract. |
| Apply accounting inbound change (`apply_accounting_inbound_change`) | Money | Ask first | — | — | Posts money in Trenova: a customer payment or a settlement payment, so a person approves each one. |
| Apply credit memo (`apply_credit_memo`) | Money | Propose | — | — | Uses a customer's credit to settle what they owe; only a person applies credit. |
| Apply customer payment (`apply_customer_payment`) | Money | Propose | — | — | Moves a customer's cash onto their invoices and books the entry that says so; only a person applies cash. |
| Apply rate increase (`apply_rate_increase`) | Money | Propose | — | — | Changes the rates many agreements charge customers or pay carriers; only a person applies a rate increase. |
| Approve billing queue item (`approve_billing_queue_item`) | Money | Propose | — | — | Approving creates the invoice a customer is billed on, so only a person approves; the agent proposes it with what it checked. |
| Approve billing queue items (`approve_billing_queue_items`) | Money | Propose | — | — | Approving creates the invoices customers are billed on, so only a person approves, item by item exactly as approve_billing_queue_item, and may untick any of them. |
| Approve carrier settlement (`approve_carrier_settlement`) | Money | Propose | — | — | Commits the organization to what the carrier is paid; only a person approves, and hands-off approval is the settlement control's rule. |
| Approve carrier settlements (`approve_carrier_settlements`) | Money | Propose | — | — | Commits the organization to what several carriers are paid at once; only a person approves, settlement by settlement exactly as approve_carrier_settlement, and may untick any of them. |
| Approve detention (`approve_detention`) | Money | Propose | — | — | Releases a held detention charge onto the customer's invoice; only a person approves it. |
| Approve driver settlement (`approve_driver_settlement`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Commits the organization to what the driver is paid; only a person approves, and hands-off approval is the settlement control's rule. |
| Approve driver settlements (`approve_driver_settlements`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Commits the organization to what several drivers are paid at once; only a person approves, settlement by settlement exactly as approve_driver_settlement, and may untick any of them. |
| Approve invoice adjustment (`approve_invoice_adjustment`) | Money | Propose | — | — | Executes a credit, rebill or write-off an approver was asked to sign off; approving is a person's decision. |
| Approve rate agreement (`approve_rate_agreement`) | Money | Propose | — | — | Turns on an agreement that prices what customers are charged or carriers are paid; only a person approves one. |
| Archive rate agreement (`archive_rate_agreement`) | Money | Propose | — | — | Ends an agreement for good, which changes what customers are charged when it was active; only a person archives one. |
| Assess late charges (`assess_late_charges`) | Money | Propose | — | — | Bills customers for paying late; only a person assesses late charges, and the billing control may post the memos as they are raised. |
| Assign move to carrier (`assign_move_to_carrier`) | Money | Ask first | Replacing a carrier already on the move, or overriding the new carrier's insurance warning, is a proposal a person decides; any other assignment runs once a person approves it. | — | Commits the organization to pay an outside carrier the rate it names for the move; nothing is sent to the carrier, and canceling the assignment releases the move. |
| Assign pay profile (`assign_pay_profile`) | Money | Propose | — | — | Sets how a driver is paid from a date on; only a person decides someone's pay. |
| Bill statement now (`bill_statement_now`) | Money | Propose | — | — | Invoices a customer off their agreed cycle; only a person decides to bill early, and the reason stays on the run. |
| Cancel billing queue item (`cancel_billing_queue_item`) | Money | Propose | — | — | Drops a charge from billing for good, so only a person decides; the agent proposes it. |
| Cancel carrier assignment (`cancel_carrier_assignment`) | Money | Ask first | Taking off a carrier who has confirmed the rate breaks an executed agreement, so it is a proposal a person decides, as is one the service would refuse or that cannot be read; a carrier who has not confirmed comes off once a person approves it. | — | Releases the organization's commitment to pay an outside carrier and voids the carrier's rate confirmation, whose sign link stops working; covering the move again makes a new agreement the carrier has to confirm afresh. |
| Close escrow account (`close_escrow_account`) | Money | Propose | — | — | Refunds an owner-operator's escrow balance and closes the account for good; only a person closes it. |
| Close fiscal period (`close_fiscal_period`) | Money | Propose | — | — | Ends posting to a period of the books a person signs off; only a person closes one. |
| Commit invoice run (`commit_invoice_run`) | Money | Propose | — | — | Issues the invoices a run proposes and advances the customers' billing period; only a person commits a run. |
| Commit rate import (`commit_rate_import`) | Money | Propose | — | — | Changes the rates an agreement prices shipments at; only a person applies a rate sheet. |
| Correct charge code (`correct_charge_code`) | Money | Automatic | — | — | Rewrites the accessorial charges a customer will be invoiced, so it moves money. |
| Correct fuel index price (`correct_fuel_index_price`) | Money | Propose | — | — | Moves the price every fuel surcharge on this index is computed from, which changes what customers are charged; only a person sets it. |
| Create carrier (`create_carrier`) | Inside the organization, money | Ask first | Each call is classified by what it reaches. A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Adds a carrier inside Trenova that colleagues can then book; nothing is sent. Payment and remit-to details decide where money goes, so a call that sets them is money and runs only on a person's approval. |
| Create invoice (`create_invoice`) | Money | Propose | — | — | Turns freight into receivables and takes it off the billing queue and any statement; only a person decides what is billed and when. |
| Create invoice memo (`create_invoice_memo`) | Money | Propose | — | — | Raises what a customer owes or is owed outside any shipment; only a person decides it, and it is never posted as it is made. |
| Create recurring deduction (`create_recurring_deduction`) | Money | Propose | — | — | Sets money taken from or added to a driver's pay every settlement; only a person agrees it. |
| Create recurring earning (`create_recurring_earning`) | Money | Propose | — | — | Sets money taken from or added to a driver's pay every settlement; only a person agrees it. |
| Dispute detention (`dispute_detention`) | Money | Ask first | — | — | Marks a detention charge disputed inside Trenova and holds it from billing; nothing is sent, but the hold is not undone by a tool. |
| End pay assignment (`end_pay_assignment`) | Money | Propose | — | — | Stops how a driver is paid from a date; only a person decides someone's pay. |
| Generate payroll export (`generate_payroll_export`) | Money | Propose | — | — | Locks approved hours into a payroll run that pays people, so a person approves it and it runs as them; void_payroll_export takes it back. |
| Issue pay advance (`issue_pay_advance`) | Money | Propose | — | — | Records money handed to a driver that their pay then repays; only a person records it. |
| Locate trailer (`locate_trailer`) | Money | Propose | — | — | Adds an empty move to the trailer's last shipment and re-rates it, which can change what that customer is charged, so it runs only on a person's approval. |
| Lock fiscal period (`lock_fiscal_period`) | Money | Propose | — | — | Changes which dates the ledger takes postings for; only a person locks a period. |
| Match bank receipt (`match_bank_receipt`) | Money | Automatic | — | — | Matches a bank receipt to a posted payment, closing its reconciliation. |
| Open escrow account (`open_escrow_account`) | Money | Propose | — | — | Sets escrow terms under an owner-operator's lease that their pay is then withheld against; only a person agrees them. |
| Open fiscal period (`open_fiscal_period`) | Money | Propose | — | — | Changes which dates the ledger takes postings for; only a person opens a period. |
| Pay driver now (`pay_driver_now`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Approves, posts and records a payment to a driver in one step; only a person pays someone. |
| Post carrier settlement (`post_carrier_settlement`) | Money | Propose | — | — | Books the settlement's payable to the ledger and queues it for the accounting system; only a person posts. |
| Post carrier settlements (`post_carrier_settlements`) | Money | Propose | — | — | Books several settlements' payables to the ledger at once and queues each for the accounting system; only a person posts, settlement by settlement exactly as post_carrier_settlement, and may untick any of them. |
| Post customer payment (`post_customer_payment`) | Money | Automatic | — | — | Records a customer payment and applies it to invoices. |
| Post driver settlement (`post_driver_settlement`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Books the settlement's payable to the ledger and queues it for the accounting system; only a person posts. |
| Post driver settlements (`post_driver_settlements`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Books several settlements' payables to the ledger at once and queues each for the accounting system; only a person posts, settlement by settlement exactly as post_driver_settlement, and may untick any of them. |
| Post invoice (`post_invoice`) | Money | Propose | — | — | Books a receivable to the ledger and queues it for the accounting system and the customer's EDI; only a person posts, and hands-off posting is the billing-control auto-post setting. |
| Post invoices (`post_invoices`) | Money | Propose | — | — | Books several receivables to the ledger at once and queues each for the accounting system and the customer's EDI; only a person posts, invoice by invoice exactly as post_invoice, and may untick any of them. |
| Post journal reversal (`post_journal_reversal`) | Money | Propose | — | — | Books the reversing entry to the general ledger; only a person posts a reversal. |
| Post manual journal (`post_manual_journal`) | Money | Propose | — | — | Books the journal's lines to the general ledger; only a person posts a manual journal. |
| Reassign billing charge (`reassign_billing_charge`) | Money | Propose | — | — | Moves what each customer is billed for a shipment and opens or cancels their queue items; only a person reassigns a charge. |
| Record carrier settlement payment (`record_carrier_settlement_payment`) | Money | Propose | — | — | Records a payment to the carrier and queues it for the accounting system; only a person says money went out. |
| Record driver settlement payment (`record_driver_settlement_payment`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Records a payment to the driver and queues it for the accounting system; only a person says money went out. |
| Record fuel index price (`record_fuel_index_price`) | Money | Propose | — | — | Moves the price every fuel surcharge on this index is computed from, which changes what customers are charged; only a person sets it. |
| Record rate confirmation confirmed (`record_rate_confirmation_confirmed`) | Money | Ask first | — | — | Makes the revision the executed agreement to pay the carrier and confirms their assignment, on the word of someone outside the organization; voiding it afterwards withdraws the agreement rather than restoring it. |
| Record tender response (`record_tender_response`) | Sent outside the organization, money | Ask first | Each call is classified by what it reaches. Recording an acceptance commits the load to the carrier at the offered rate, so it is a proposal a person decides; a decline runs once a person approves it. | — | An acceptance commits the organization to pay the carrier and sends them the rate confirmation; a decline sends the next carrier its offer. Neither is undone by recording another answer. |
| Release accounting sync (`release_accounting_sync`) | Money | Propose | — | — | Sends documents a review policy held for a person to the organization's books; only a person releases them. |
| Remove carrier settlement adjustment (`remove_carrier_settlement_adjustment`) | Money | Automatic | — | — | Changes what the carrier will be paid on a settlement a person still approves, so it moves money; the line is added back the same way. |
| Remove driver settlement adjustment (`remove_driver_settlement_adjustment`) | Money | Automatic | — | — | Changes what the driver will be paid on a settlement a person still approves, so it moves money; the line is added back the same way. |
| Remove order charge (`remove_order_charge`) | Money | Ask first | — | — | Lowers what the customer is billed; nothing is invoiced or sent, and add_order_charge puts it back. |
| Reopen fiscal period (`reopen_fiscal_period`) | Money | Propose | — | — | Lets postings land again in books a person closed; only a person reopens a period. |
| Request journal reversal (`request_journal_reversal`) | Money | Propose | — | — | Commits the organization to taking a posted entry back out of the ledger, approved at once where approval is off; only a person requests a reversal. |
| Rerate shipment (`rerate_shipment`) | Money | Ask first | — | — | Changes what the customer will be billed for the shipment; nothing is invoiced or sent, and a person can edit the rate back. |
| Resolve accounting drift (`resolve_accounting_drift`) | Money | Ask first | — | — | Changes money on one side of the books: it posts a memo, void or reversal in Trenova, or sends a document to the accounting system, so a person approves each fix. |
| Resolve settlement dispute (`resolve_settlement_dispute`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Decides a driver's pay complaint, tells the driver, and can add pay to a settlement, so a person approves it and it runs as them; a decided dispute stays decided. |
| Resume rate agreement (`resume_rate_agreement`) | Money | Propose | — | — | Turns an agreement back on so it prices shipments again; only a person resumes one. |
| Reverse customer payment (`reverse_customer_payment`) | Money | Propose | — | — | Takes cash back off the books and reopens the invoices it paid; only a person reverses a payment. |
| Review driver expense (`review_driver_expense`) | Seen by a driver, money | Propose | Each call is classified by what it reaches. | — | Pays or refuses a driver's out-of-pocket expense and tells the driver, so a person approves it and it runs as them; a reviewed expense stays reviewed. |
| Set carrier monitoring (`set_carrier_monitoring`) | Money | Ask first | — | — | Monitoring is billed per carrier by the provider; turning it off again undoes it, and nothing is sent to the carriers. |
| Set order charge allocations (`set_order_charge_allocations`) | Money | Ask first | — | — | Moves who is billed for a charge; nothing is invoiced or sent, and the split is set back the same way until it is invoiced. |
| Submit invoice adjustment (`submit_invoice_adjustment`) | Money | Propose | — | — | Credits, rebills or writes off receivables and books the entries that say so; only a person submits an adjustment, and the adjustment policy may still hold it for an approver. |
| Suspend rate agreement (`suspend_rate_agreement`) | Money | Propose | — | — | Stops an agreement pricing shipments, which changes what customers are charged; only a person suspends one. |
| Unapply credit memo (`unapply_credit_memo`) | Money | Propose | — | — | Reopens an invoice's balance and restores a customer's credit; only a person moves credit. |
| Unlock fiscal period (`unlock_fiscal_period`) | Money | Propose | — | — | Changes which dates the ledger takes postings for; only a person unlocks a period. |
| Update carrier (`update_carrier`) | Inside the organization, money | Ask first | Each call is classified by what it reaches. A record read from a document or message someone outside sent is proposed, since colleagues book and pay against it. | — | Changes a carrier inside Trenova and keeps its contacts, insurance and EDI channels; nothing is sent. A call that changes payment or remit-to details is money and runs only on a person's approval. |
| Update escrow account (`update_escrow_account`) | Money | Propose | — | — | Changes the escrow terms of an owner-operator's lease; only a person agrees them. |
| Update order (`update_order`) | Inside the organization, money | Ask first | Each call is classified by what it reaches. | — | Edits an order's own fields inside Trenova; nothing is sent, and the old values are set back the same way. A changed amount is a money change. |
| Update order charge (`update_order_charge`) | Money | Ask first | — | — | Changes what the customer is billed; nothing is invoiced or sent, and the old amount is set back the same way until it is invoiced. |
| Update recurring deduction (`update_recurring_deduction`) | Money | Propose | — | — | Changes money taken from or added to a driver's pay every settlement; only a person agrees it. |
| Update recurring earning (`update_recurring_earning`) | Money | Propose | — | — | Changes money taken from or added to a driver's pay every settlement; only a person agrees it. |
| Vet carrier (`vet_carrier`) | Money | Ask first | — | — | Spends the organization's carrier intelligence budget on a lookup and records the result; nothing is sent to the carrier. |
| Vet customer broker (`vet_customer_broker`) | Money | Ask first | — | — | Spends the organization's carrier intelligence budget on a lookup and records the result; nothing is sent to the customer. |
| Void carrier settlement (`void_carrier_settlement`) | Money | Propose | — | — | Cancels a settlement for good and reverses any posting; only a person voids. |
| Void driver settlement (`void_driver_settlement`) | Money | Propose | — | — | Cancels a settlement for good and reverses any posting; only a person voids. |
| Void invoice (`void_invoice`) | Money | Propose | — | — | Takes an invoice out of circulation and reverses what it booked; it cannot be undone, so only a person voids one. |
| Void payroll export (`void_payroll_export`) | Money | Propose | — | — | Takes back a payroll run that payroll may already have paid from, so a person approves it and it runs as them; a voided run stays voided. |
| Void rate confirmation (`void_rate_confirmation`) | Money | Ask first | A revision the carrier has been sent or has signed is voided only as a proposal a person decides, as is one that cannot be read; one never sent is voided once a person approves it. | — | Withdraws the organization's written agreement to pay a carrier, whose sign link stops working, and undoes a confirmation it carried; a voided revision cannot be restored, only replaced. |
| Waive detention (`waive_detention`) | Money | Propose | — | — | Gives up detention revenue the organization would otherwise bill; only a person approves it. |
| Write off pay advance (`write_off_pay_advance`) | Money | Propose | — | — | Forgives money a driver owes, which the organization absorbs; only a person writes it off. |

