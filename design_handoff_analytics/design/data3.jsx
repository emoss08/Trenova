const EGRESS = { N: ["Reads only", 250, 0.02], P: ["Own records", 182, 0.09], I: ["Internal", 275, 0.11], C: ["Customer", 230, 0.11], D: ["Driver", 160, 0.11], X: ["Outside recipient", 300, 0.13], M: ["Money", 75, 0.13] };
const EG_ORDER = ["N", "P", "I", "C", "D", "X", "M"];
const OUTSIDE = "CDXM";
const TIER_L = { Propose: "Propose", ActWithApproval: "Ask first", AutoExecute: "Automatic" };
const TIER_O = ["Propose", "ActWithApproval", "AutoExecute"];
const ANS = { RUNS: ["Runs on its own", "r"], COND: ["Depends on the call", "c"], APPROVAL: ["Needs approval", "w"], PROPOSE: ["Proposes only", "p"], SIM: ["Simulated", "s"] };
const HELD_L = { agent_ceiling: "Agent ceiling", tool_max: "Tool maximum", tainted: "Read outside text", shadow_mode: "Shadow mode", simulation_mode: "Simulation", personal_exemption: "Own records, person present" };
const T = (l, n, e, tier, needs, ext, kind, src) => ({ l, n, e, tier, needs, ext: ext || "Never", kind: kind || (e === "N" ? "Query" : "Action"), src });
const TOOLS = [
  T("Find in Trenova", "find_in_trenova", "N", "AutoExecute", null), T("Get shipment", "get_shipment", "N", "AutoExecute", "Shipment · read"), T("Search shipments", "search_shipments", "N", "AutoExecute", "Shipment · read"),
  T("List billing queue items", "list_billing_queue_items", "N", "AutoExecute", "Billing Queue · read"), T("Get billing queue item", "get_billing_queue_item", "N", "AutoExecute", "Billing Queue · read"), T("Get accounting sync status", "get_accounting_sync_status", "N", "AutoExecute", "Accounting · read"),
  T("Recall memory", "recall_memory", "N", "AutoExecute", null, "Marked", "Query", "memory"), T("Open page", "open_page", "N", "AutoExecute", null), T("Read inbound email", "read_inbound_message", "N", "AutoExecute", "Inbound Message · read", "Always", "Query", "inbound messages"),
  T("Read document", "read_document", "N", "AutoExecute", "Document · read", "Always", "Query", "documents"), T("Search the web", "web_search", "N", "AutoExecute", null, "Always", "Query", "the web"),
  T("Record for later", "record_memory", "P", "AutoExecute", null), T("Add home widget", "add_home_widget", "P", "AutoExecute", null), T("Add dashboard tile", "add_dashboard_tile", "I", "AutoExecute", "Dashboard · update"),
  T("Acknowledge carrier intel event", "acknowledge_carrier_intel_event", "I", "AutoExecute", "Carrier Intelligence · update"), T("Flag for manual review", "flag_for_manual_review", "I", "AutoExecute", "Billing Queue · update"),
  T("Move item to in review", "transition_billing_item", "I", "ActWithApproval", "Billing Queue · update"), T("Hold billing queue item", "hold_billing_queue_item", "I", "ActWithApproval", "Billing Queue · update"),
  T("Attach document to shipment", "attach_document_to_shipment", "I", "AutoExecute", "Document · create"), T("Draft IFTA return", "draft_ifta_return", "I", "AutoExecute", "IFTA · create"), T("Update driver qualification", "update_qualification", "I", "ActWithApproval", "Worker · update"),
  T("Assign driver", "assign_driver", "D", "ActWithApproval", "Assignment · create"), T("Message driver", "send_driver_message", "D", "ActWithApproval", "Driver Message · create"),
  T("Reply to customer", "send_customer_reply", "C", "ActWithApproval", "Inbound Message · reply"), T("Request missing documents", "request_missing_documents", "C", "ActWithApproval", "Document Request · create"),
  T("Accept EDI tender", "accept_edi_tender", "X", "Propose", "EDI · update"), T("Tender to carrier", "tender_to_carrier", "X", "ActWithApproval", "Carrier Tender · create"), T("Email a scheduled report", "schedule_report_email", "X", "Propose", "Report Schedule · create"),
  T("Accept carrier invoice match", "accept_carrier_invoice_match", "M", "Propose", "Carrier Invoice Match · approve"), T("Add driver settlement adjustment", "add_driver_settlement_adjustment", "M", "ActWithApproval", "Driver Settlement · update"),
  T("Correct charge code", "correct_charge_code", "M", "ActWithApproval", "Invoice · update"), T("Release billing hold", "release_billing_hold", "M", "ActWithApproval", "Billing Queue · release"),
  T("Write off short-pay", "write_off_short_pay", "M", "Propose", "Receivable · adjust"), T("Apply customer payment", "apply_customer_payment", "M", "ActWithApproval", "Receivable · apply"),
  T("Test accounting connection", "test_accounting_connection", "N", "AutoExecute", "Accounting · read", "Never", "Runtime"), T("Hand off to another agent", "delegate_to_agent", "N", "AutoExecute", null, "Never", "Runtime"),
];
const HOLD = {
  billing: "find_in_trenova get_shipment search_shipments list_billing_queue_items get_billing_queue_item get_accounting_sync_status recall_memory open_page read_document record_memory flag_for_manual_review transition_billing_item hold_billing_queue_item request_missing_documents correct_charge_code release_billing_hold write_off_short_pay test_accounting_connection delegate_to_agent",
  compliance: "find_in_trenova update_qualification read_document send_driver_message recall_memory",
  dispatch: "find_in_trenova get_shipment search_shipments recall_memory open_page read_inbound_message assign_driver send_driver_message tender_to_carrier attach_document_to_shipment record_memory delegate_to_agent",
  fuel: "find_in_trenova draft_ifta_return read_document", help: "open_page",
  mds: "find_in_trenova search_shipments read_document attach_document_to_shipment acknowledge_carrier_intel_event",
  recv: "find_in_trenova search_shipments get_accounting_sync_status apply_customer_payment write_off_short_pay send_customer_reply recall_memory",
  report: "find_in_trenova search_shipments open_page add_home_widget add_dashboard_tile schedule_report_email recall_memory",
  settle: "find_in_trenova search_shipments accept_carrier_invoice_match add_driver_settlement_adjustment read_document",
  workforce: "find_in_trenova update_qualification send_driver_message", cs: "find_in_trenova get_shipment send_customer_reply",
  digest: "find_in_trenova search_shipments schedule_report_email",
  billev: "list_billing_queue_items get_billing_queue_item flag_for_manual_review hold_billing_queue_item release_billing_hold",
  coverage: "search_shipments assign_driver tender_to_carrier",
  inbox: "read_inbound_message read_document attach_document_to_shipment send_customer_reply accept_edi_tender tender_to_carrier",
};
const CEIL = { report: "AutoExecute", cs: "Propose", coverage: "Propose" };
const ceilOf = a => a.ceil || CEIL[a.id] || "ActWithApproval";
const toolsOf = a => (HOLD[a.id] || "").split(" ").filter(Boolean).map(n => TOOLS.find(t => t.n === n)).filter(Boolean);
function answerOf(a, t) {
  if (t.kind !== "Action") return { b: "RUNS", a: t.ext !== "Never" ? "RUNS" : "RUNS", held: [] };
  if (a.sim) return { b: "SIM", a: "SIM", held: ["simulation_mode"] };
  if (a.shadow) return { b: "SIM", a: "SIM", held: ["shadow_mode"] };
  const ci = TIER_O.indexOf(ceilOf(a)), ti = TIER_O.indexOf(t.tier), eff = Math.min(ci, ti), held = [];
  if (ci < ti) held.push("agent_ceiling"); else if (ti < 2) held.push("tool_max");
  const b = eff === 2 ? (t.e === "P" ? "RUNS" : "RUNS") : eff === 1 ? "APPROVAL" : "PROPOSE";
  let af = b; if (b === "RUNS" && OUTSIDE.includes(t.e)) { af = "APPROVAL"; held.push("tainted"); }
  if (b === "RUNS" && t.e === "P") held.push("personal_exemption");
  return { b, a: af, held };
}
function safetyFigures(agents) {
  const on = agents.filter(a => a.on);
  const runs = new Set(); on.forEach(a => toolsOf(a).forEach(t => { if (t.kind === "Action" && answerOf(a, t).b === "RUNS") runs.add(t.n); }));
  const leave = TOOLS.filter(t => t.kind === "Action" && OUTSIDE.includes(t.e)).length;
  const open = on.filter(a => a.access === "Everyone" && toolsOf(a).some(t => t.kind === "Action" && OUTSIDE.includes(t.e)));
  return { runs: [...runs], leave, open };
}
Object.assign(window, { EGRESS, EG_ORDER, OUTSIDE, TIER_L, TIER_O, ANS, HELD_L, TOOLS, HOLD, ceilOf, toolsOf, answerOf, safetyFigures });
