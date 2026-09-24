# AI tool safety

<!-- Generated from the tool policies in code by running, in services/tms:
     go generate ./internal/core/services/agenttoolpolicy/safetydoc/...
     Do not edit by hand. -->

What every tool an agent can call may do without a person, read from the
policies the tools declare in code. The runtime decides each call from the same
policies, so this page cannot drift from what runs: CI regenerates it and fails
when it differs. Each tool is listed once, under the furthest class its work
can reach.

Tools listed: 137.

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

## Classes

| Class | Means | Runs at most | Held once tainted | Tools that reach it |
| --- | --- | --- | --- | --- |
| Reads only | Looks something up. Nothing changes and nothing is sent. | Automatic | No | 84 |
| The caller's own records | Changes only the records of the person using the agent. | Automatic | No | 6 |
| Inside the organization | Changes records only people inside the organization see. | Automatic | No | 31 |
| Seen by a customer | Changes something a customer can see. | Ask first | Yes | 1 |
| Seen by a driver | Changes something a driver can see. | Ask first | Yes | 5 |
| Sent outside the organization | Sends to someone outside the organization. | Ask first | Yes | 10 |
| Money | Moves or commits money. | Automatic | Yes | 4 |

## Reads only

Looks something up. Nothing changes and nothing is sent.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Ask user (`ask_user`) | Reads only | Automatic | — | — | Asks the person in the conversation a question; nothing is saved or sent. |
| Compare report runs (`compare_report_runs`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Compose table view (`compose_table_view`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Delegate task (`delegate_task`) | Reads only | Automatic | — | — | Hands a task to another agent of the organization, which runs as the same person under its own tiers. |
| Describe report (`describe_report`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Describe report dataset (`describe_report_dataset`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Explain rate (`explain_rate`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Find in trenova (`find_in_trenova`) | Reads only | Automatic | — | — | Searches the product guide for the caller; nothing changes and nothing is sent. |
| Find tools (`find_tools`) | Reads only | Automatic | — | — | Loads more of the agent's own tools into the turn; it reads the catalog and changes nothing. |
| Get agent run (`get_agent_run`) | Reads only | Automatic | — | When the record is marked, from run record | Reads a run's own record, whose summary may repeat outside text the run read. |
| Get bank receipt (`get_bank_receipt`) | Reads only | Automatic | — | Always, from bank receipt | Reads a bank receipt whose memo the payer wrote; nothing changes and nothing is sent. |
| Get carrier (`get_carrier`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get carrier intel event (`get_carrier_intel_event`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get customer (`get_customer`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get customer update preferences (`get_customer_update_preferences`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get daily briefing (`get_daily_briefing`) | Reads only | Automatic | — | — | Reads the morning's computed figures, and only the sections the caller may read; it changes nothing, not even whether the briefing was read. |
| Get detention occurrence (`get_detention_occurrence`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get dispatch board (`get_dispatch_board`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get document summary (`get_document_summary`) | Reads only | Automatic | — | Always, from document | Reads text extracted from a document someone outside sent; nothing changes and nothing is sent. |
| Get inbound message (`get_inbound_message`) | Reads only | Automatic | — | Always, from inbound message | Reads mail an outsider wrote; nothing changes and nothing is sent. |
| Get insight (`get_insight`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get invoice (`get_invoice`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get my home layout (`get_my_home_layout`) | Reads only | Automatic | — | — | Reads the caller's own home page; nothing changes and nothing is sent. |
| Get report run (`get_report_run`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get service failure (`get_service_failure`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get shipment (`get_shipment`) | Reads only | Automatic | — | When the record is marked, from record note | Reads a shipment with its newest comments, some of which a driver, a trading partner or another system outside the organization wrote; nothing changes and nothing is sent. |
| Get shipment draft (`get_shipment_draft`) | Reads only | Automatic | — | Always, from document | Reads a shipment drafted from a document someone outside sent; nothing changes and nothing is sent. |
| Get shipment tracking (`get_shipment_tracking`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get tractor (`get_tractor`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get trailer (`get_trailer`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get worker (`get_worker`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get worker credential (`get_worker_credential`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Get worker HOS (`get_worker_hos`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List accessorial charges (`list_accessorial_charges`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List agent runs (`list_agent_runs`) | Reads only | Automatic | — | When the record is marked, from run record | Lists agent runs, whose summaries may repeat outside text a run read; a run on a conversation is listed only to the person who owns it. |
| List bank receipt exceptions (`list_bank_receipt_exceptions`) | Reads only | Automatic | — | Always, from bank receipt | Lists bank receipts whose memos the payers wrote; nothing changes and nothing is sent. |
| List carriers (`list_carriers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List commodities (`list_commodities`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List customer payments (`list_customer_payments`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List customers (`list_customers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List dashboards (`list_dashboards`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List detention desk (`list_detention_desk`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List document types (`list_document_types`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List email profiles (`list_email_profiles`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List equipment types (`list_equipment_types`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List expiring credentials (`list_expiring_credentials`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List fleet codes (`list_fleet_codes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List hazardous materials (`list_hazardous_materials`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List hold reasons (`list_hold_reasons`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List home widgets (`list_home_widgets`) | Reads only | Automatic | — | — | Lists the widgets the caller's home page can show; nothing changes and nothing is sent. |
| List inbound messages (`list_inbound_messages`) | Reads only | Automatic | — | Always, from inbound message | Lists mail outsiders wrote, subjects and senders included; nothing changes and nothing is sent. |
| List insights (`list_insights`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List invoices (`list_invoices`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List location categories (`list_location_categories`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List locations (`list_locations`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List report datasets (`list_report_datasets`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List report runs (`list_report_runs`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List reports (`list_reports`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List service failure reason codes (`list_service_failure_reason_codes`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List service failures (`list_service_failures`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| List service types (`list_service_types`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
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
| Quote shipment (`quote_shipment`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Rank move candidates (`rank_move_candidates`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Recall memory (`recall_memory`) | Reads only | Automatic | — | When the record is marked, from memory | Reads memories earlier runs saved, which carry the taint of the run that wrote them. |
| Run report (`run_report`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Search documents (`search_documents`) | Reads only | Automatic | — | Always, from document | Searches text extracted from documents, many of which someone outside wrote; each record returned is checked against what the caller may read, and nothing changes or is sent. |
| Search inbound messages (`search_inbound_messages`) | Reads only | Automatic | — | Always, from inbound message | Searches mail outsiders wrote, subjects, senders and bodies included; nothing changes and nothing is sent. |
| Search shipments (`search_shipments`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Search worker (`search_worker`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
| Shop carriers (`shop_carriers`) | Reads only | Automatic | — | — | Reads records the caller may already open; it changes nothing and sends nothing. |
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
| Assign move (`assign_move`) | Inside the organization | Automatic | — | — | Assigns a driver and tractor to a move; the driver sees the assignment but no text the model wrote. |
| Attach document to shipment (`attach_document_to_shipment`) | Inside the organization | Automatic | — | — | Files a document already in Trenova against a shipment; nobody outside is told. |
| Create dashboard (`create_dashboard`) | Inside the organization | Automatic | — | — | Saves a report dashboard colleagues can open; nothing leaves the organization. |
| Create report (`create_report`) | The caller's own records, inside the organization | Automatic | Each call is classified by what it reaches. A call on the caller's own records runs unasked while they are present. | — | A private report is a saved query on the caller's own list; a shared one appears on every colleague's Reports page and waits for approval. |
| Create shipment (`create_shipment`) | Inside the organization | Ask first | — | — | A new load commits a customer's freight and the money that follows, so no desk books one unattended. |
| Create table change alert (`create_table_change_alert`) | Inside the organization | Automatic | — | — | Creates an alert whose notices go to people inside the organization. |
| Dismiss insight (`dismiss_insight`) | Inside the organization | Automatic | — | — | Dismisses an insight inside Trenova; it can be restored. |
| Escalate detention (`escalate_detention`) | Inside the organization | Automatic | — | — | Hands a detention clock to a person inside the organization. |
| Evaluate service failures (`evaluate_service_failures`) | Inside the organization | Automatic | — | — | Opens service failures from stop actuals inside Trenova; an open failure sends nothing. |
| Flag for manual review (`flag_for_manual_review`) | Inside the organization | Automatic | — | — | Records an exception on the run for a person inside the organization to work. |
| Forget memory (`forget_memory`) | Inside the organization | Automatic | — | — | Retires an agent memory inside Trenova. |
| Fork report (`fork_report`) | Inside the organization | Automatic | — | — | Saves a copy of a report inside Trenova; nothing leaves the organization. |
| Link inbound message (`link_inbound_message`) | Inside the organization | Automatic | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Links an inbound message to a record inside Trenova; the mailbox decides how far it may run. |
| Mark inbound message (`mark_inbound_message`) | Inside the organization | Automatic | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Settles an inbound message inside Trenova; the mailbox decides how far it may run. |
| Place shipment hold (`place_shipment_hold`) | Inside the organization | Automatic | — | — | Places a hold on a shipment inside Trenova; no customer or EDI notice is sent. |
| Place worker dispatch hold (`place_worker_dispatch_hold`) | Inside the organization | Automatic | — | — | Keeps a driver off new freight inside Trenova; the driver is not messaged. |
| Raise exception (`raise_exception`) | Inside the organization | Automatic | — | — | Records an exception against the run itself for a person inside the organization. |
| Record stop actual (`record_stop_actual`) | Inside the organization | Automatic | — | — | Records arrival and departure times on a stop; no model-written text leaves the organization. |
| Release shipment hold (`release_shipment_hold`) | Inside the organization | Automatic | — | — | Releases a hold on a shipment inside Trenova; no customer or EDI notice is sent. |
| Remember (`remember`) | Inside the organization | Automatic | An Instruction or a Correction recorded after the run read text from outside the organization waits for a person's approval; a Fact is recorded and stays marked as drawn from outside text. | Carries outside text into later runs | Saves a memory later runs read, so it keeps the taint of the run that wrote it. |
| Resolve bank receipt work item (`resolve_bank_receipt_work_item`) | Inside the organization | Automatic | — | — | Closes a reconciliation work item without moving money; the note is read inside the organization. |
| Resolve carrier intel event (`resolve_carrier_intel_event`) | Inside the organization | Automatic | — | — | Closes a carrier finding inside Trenova. |
| Save table view (`save_table_view`) | The caller's own records, inside the organization | Automatic | Each call is classified by what it reaches. A call on the caller's own records runs unasked while they are present. | — | A private view is the caller's own picker entry; a shared one appears for every colleague, and nothing leaves the organization. |
| Transition item to in review (`transition_item_to_in_review`) | Inside the organization | Automatic | — | — | Moves the run's billing queue item into review inside Trenova; nothing is sent anywhere. |
| Update report (`update_report`) | Inside the organization | Automatic | — | — | Changes a saved report colleagues may open; nothing leaves the organization. |
| Update tractor status (`update_tractor_status`) | Inside the organization | Automatic | — | — | Changes a tractor's status inside Trenova. |
| Update trailer status (`update_trailer_status`) | Inside the organization | Automatic | — | — | Changes a trailer's status inside Trenova. |

## Seen by a driver

Changes something a driver can see.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Add shipment comment (`add_shipment_comment`) | Inside the organization, seen by a customer, seen by a driver | Automatic | Each call is classified by what it reaches. | Carries outside text into later runs | An internal note stays inside the organization; a customer or driver note is read outside it, so its visibility argument decides. A note written on its own after the run read outside text is marked as drawn from it. |
| Cancel worker PTO (`cancel_worker_pto`) | Seen by a driver | Ask first | — | — | The worker is sent the cancellation reason by push notice and text message. |
| Notify driver (`notify_driver`) | Seen by a driver | Ask first | — | — | Sends a driver a message the model wrote to their phone. |
| Reject worker PTO (`reject_worker_pto`) | Seen by a driver | Ask first | — | — | The worker is sent the rejection reason by push notice and text message. |
| Request credential renewal (`request_credential_renewal`) | Seen by a driver | Ask first | — | — | Sends a driver a renewal request the model wrote. |

## Sent outside the organization

Sends to someone outside the organization.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Cancel shipment (`cancel_shipment`) | Sent outside the organization | Ask first | — | — | Cancelling withdraws live tenders from carriers and sends the model's cancel reason to a linked partner over EDI. |
| Email customer (`email_customer`) | Sent outside the organization | Ask first | — | — | Emails the customer's contacts a message the model wrote. |
| Reply to inbound message (`reply_to_inbound_message`) | Sent outside the organization | Ask first | A call on an inbound message runs only as far as its mailbox allows: a classified message the mailbox handles without review may run on its own, and anything held, quarantined, settled or unreadable waits for a person. | — | Replies to whoever wrote in with text the model composed. |
| Request missing docs (`request_missing_docs`) | Sent outside the organization | Ask first | — | — | Emails an outside party a request for paperwork in words the model wrote. |
| Resolve service failure (`resolve_service_failure`) | Sent outside the organization | Ask first | — | — | Resolving a failure generates an EDI 214 to the customer's trading partner carrying the reason chosen. |
| Schedule report (`schedule_report`) | Sent outside the organization | Ask first | — | — | Emails a report on a schedule to whatever addresses the call names, which may be outside the organization. |
| Send detention notice (`send_detention_notice`) | Sent outside the organization | Ask first | — | — | Sends the customer a detention notice that starts a charge. |
| Tender move to carriers (`tender_move_to_carriers`) | Sent outside the organization | Ask first | — | — | Offers the load to carriers outside the organization. |
| Tender move to routing guide (`tender_move_to_routing_guide`) | Sent outside the organization | Ask first | — | — | Offers the load to carriers outside the organization. |
| Update shipment (`update_shipment`) | Sent outside the organization | Ask first | — | — | A changed shipment is sent as an EDI tender change to the trading partners it was tendered to. |

## Money

Moves or commits money.

| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |
| --- | --- | --- | --- | --- | --- |
| Correct charge code (`correct_charge_code`) | Money | Automatic | — | — | Rewrites the accessorial charges a customer will be invoiced, so it moves money. |
| Match bank receipt (`match_bank_receipt`) | Money | Automatic | — | — | Matches a bank receipt to a posted payment, closing its reconciliation. |
| Post customer payment (`post_customer_payment`) | Money | Automatic | — | — | Records a customer payment and applies it to invoices. |
| Waive detention (`waive_detention`) | Money | Automatic | — | — | Gives up detention revenue the organization would otherwise bill. |

