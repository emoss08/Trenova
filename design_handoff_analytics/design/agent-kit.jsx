const ROLES = ["Owner", "Dispatch lead", "Dispatcher", "Billing manager", "Billing clerk", "Safety manager", "Customer service", "Accounting"];
const EVENTS = ["Billing hold placed", "Invoice rejected", "Rate mismatch", "Shipment uncovered", "Driver unassigned", "Email received", "EDI 204 received", "Document uploaded", "Qualification expiring", "Detention started"];
const AG_ICONS = ["clock", "receipt", "shield", "truck", "fuel", "route", "dollar", "wallet", "chart", "users", "headset", "inbox", "sun", "database", "brain", "bolt", "help"];
const AG_HUES = [0, 25, 60, 85, 145, 182, 230, 258, 285, 310];
const CRON = [["0 9 * * 5", "Fridays", "Every Friday at 9:00 AM", ["Fri, Oct 9 · 9:00 AM", "Fri, Oct 16 · 9:00 AM", "Fri, Oct 23 · 9:00 AM"]], ["0 6 * * 1-5", "Weekday mornings", "Every weekday at 6:00 AM", ["Thu, Oct 8 · 6:00 AM", "Fri, Oct 9 · 6:00 AM", "Mon, Oct 12 · 6:00 AM"]], ["0 * * * *", "Every hour", "At the top of every hour", ["Today · 1:00 PM", "Today · 2:00 PM", "Today · 3:00 PM"]], ["0 7 * * *", "Daily", "Every day at 7:00 AM", ["Thu, Oct 8 · 7:00 AM", "Fri, Oct 9 · 7:00 AM", "Sat, Oct 10 · 7:00 AM"]], ["0 8 * * 1", "Mondays", "Every Monday at 8:00 AM", ["Mon, Oct 12 · 8:00 AM", "Mon, Oct 19 · 8:00 AM", "Mon, Oct 26 · 8:00 AM"]], ["0 17 L * *", "Month end", "The last day of each month at 5:00 PM", ["Sat, Oct 31 · 5:00 PM", "Mon, Nov 30 · 5:00 PM", "Thu, Dec 31 · 5:00 PM"]]];
const TZS = ["America/Los_Angeles", "America/Denver", "America/Chicago", "America/New_York"];
const VARS = [["{{organization}}", "Organization"], ["{{user.name}}", "Person asking"], ["{{user.role}}", "Their role"], ["{{today}}", "Today's date"]];
const LINT = [[/\bemail|e-mail\b/i, ["send_customer_reply", "schedule_report_email"], "Mentions email, but it holds no tool that sends email."], [/\bassign|driver\b/i, ["assign_driver", "send_driver_message"], "Mentions drivers, but it can't assign or message one."], [/\btender\b/i, ["tender_to_carrier", "accept_edi_tender"], "Mentions tendering, but it holds no tender tool."], [/\bpayment|pay\b/i, ["apply_customer_payment", "add_driver_settlement_adjustment"], "Mentions payments, but it can't apply one."]];
const VERSIONS = [["v7", "Sarah Alvarez", "Oct 6, 3:12 PM", "Raised Release billing hold to Ask first", true], ["v6", "Marcus Reed", "Sep 29, 9:40 AM", "Added the rule about short-pays under $50"], ["v5", "Sarah Alvarez", "Sep 18, 1:05 PM", "Moved from Shadow to Live after 64 clean proposals"], ["v4", "Trenova", "Sep 2, 6:00 AM", "Created from the Billing exceptions template"]];
const TRY = {
  dispatch: { q: "Who can cover SHP-48302 tomorrow morning?", calls: ["search_shipments", "find_in_trenova", "assign_driver"], a: "Marcus Reed is 14 miles from the pickup with 9 hours of drive time left. He's the closest qualified driver." },
  billing: { q: "Why is INV-20931 still on hold?", calls: ["get_billing_queue_item", "read_document", "release_billing_hold"], a: "It was held for a missing POD. The POD was attached at 9:12 and the signature date matches delivery, so the hold can come off." },
  recv: { q: "Who should we chase first this week?", calls: ["search_shipments", "get_accounting_sync_status", "send_customer_reply"], a: "Acme Foods: $18,420 over 45 days across 6 invoices. They usually pay within a week of a reminder." },
  digest: { q: "Run this morning's digest", calls: ["search_shipments", "find_in_trenova", "schedule_report_email"], a: "4 loads are uncovered for today, 2 are running late, and 1 billing hold is over 3 days old." },
};
const ED_MODE_NOTE = { live: "Proposals are offered to a person; automatic tools run on their own.", shadow: "Runs for real, but proposals are recorded instead of offered. Use it to earn trust before going live.", sim: "Writes are previewed and recorded, never made. Good for testing new instructions." };
const tierIx = t => TIER_O.indexOf(t);
const tierCap = (t, ceil) => TIER_O[Math.min(tierIx(t), tierIx(ceil))];
function outcomeOf(v, n) {
  const t = TOOLS.find(x => x.n === n); if (!t) return ["Unknown", "s"];
  if (!(n in v.tools)) return ["Not held · skipped", "x"];
  if (v.mode === "sim") return ["Simulated", "s"];
  if (t.kind !== "Action") return ["Runs", "r"];
  if (v.mode === "shadow") return ["Recorded, not offered", "p"];
  const tier = tierCap(v.tools[n], v.ceil);
  return tier === "AutoExecute" ? ["Runs on its own", "r"] : tier === "ActWithApproval" ? ["Asks a person first", "w"] : ["Proposes only", "p"];
}
function agentDraft(a, tpl) {
  if (a) { const ceil = ceilOf(a); const tools = {}; toolsOf(a).forEach(t => tools[t.n] = tierCap(t.tier, ceil));
    return { n: a.n, d: a.d, ic: a.ic, h: a.h, instr: `You are ${a.n} for {{organization}}. ${a.d}\n\nBefore proposing a change, check the record it touches and say why in one sentence.\nIf something is unclear, ask {{user.name}} instead of guessing.\nNever promise a customer a date or a rate.`, trig: a.trig, access: a.access === "Everyone" ? [] : ["Dispatch lead", "Billing manager", "Owner"], cron: a.cron || "0 6 * * 1-5", tz: a.tz || TZS[0], events: a.events || [], tools, ceil, mode: a.sim ? "sim" : a.shadow ? "shadow" : "live", runsDay: 200, budget: 40, steps: 12, route: "auto", learn: true, handoff: a.can ? S_AG_IDS(a.can) : [] }; }
  const trig = tpl === "Scheduled report" ? "Scheduled" : tpl === "Event watcher" ? "Event" : "Chat";
  const tools = { find_in_trenova: "AutoExecute", recall_memory: "AutoExecute" }; if (trig === "Scheduled") tools.schedule_report_email = "Propose"; if (trig === "Event") tools.flag_for_manual_review = "AutoExecute";
  return { n: "", d: "", ic: trig === "Scheduled" ? "sun" : trig === "Event" ? "bolt" : "chat", h: 258, instr: "", trig, access: [], cron: "0 6 * * 1-5", tz: TZS[0], events: [], tools, ceil: "ActWithApproval", mode: "shadow", runsDay: 100, budget: 20, steps: 10, route: "auto", learn: true, handoff: [] };
}
const S_AG_IDS = s => AGENTS0.filter(x => s.includes(x.n)).map(x => x.id);
function ToolPicker({ v, set }) {
  const [q, setQ] = React.useState(""); const [open, setOpen] = React.useState(false);
  const free = TOOLS.filter(t => !(t.n in v.tools) && (t.l + t.n).toLowerCase().includes(q.toLowerCase()));
  return <div className="rel">
    <button className="btn sm" onClick={() => setOpen(o => !o)}><Ic n="plus" s={12}></Ic>Add tools</button>
    {open && <div className="mn right tp" onMouseDown={e => e.stopPropagation()}><label className="srch"><Ic n="search" s={12}></Ic><input autoFocus value={q} onChange={e => setQ(e.target.value)} placeholder="Search 37 tools" onKeyDown={e => e.key === "Escape" && (e.preventDefault(), setOpen(false))}></input></label>
      <div className="tp-l">{free.slice(0, 40).map(t => <button key={t.n} className="mn-i" onClick={() => set("tools", o => ({ ...o, [t.n]: t.kind === "Action" ? tierCap(t.tier, v.ceil) : "AutoExecute" }))}><span className="eg-d" style={{ "--h": EGRESS[t.e][1], "--c": EGRESS[t.e][2] }}></span><span className="mn-l"><b>{t.l}</b><em>{EGRESS[t.e][0]}{t.needs ? " · needs " + t.needs : ""}</em></span><Ic n="plus" s={12}></Ic></button>)}{!free.length && <div className="mn-h">No tools match</div>}</div>
      <div className="tp-f"><button className="lnk" onClick={() => setOpen(false)}>Done</button></div></div>}
  </div>;
}
Object.assign(window, { ROLES, EVENTS, AG_ICONS, AG_HUES, CRON, TZS, VARS, LINT, VERSIONS, TRY, ED_MODE_NOTE, tierIx, tierCap, outcomeOf, agentDraft, ToolPicker });
