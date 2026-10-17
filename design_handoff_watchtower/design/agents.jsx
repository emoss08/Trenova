const AG_ICON = {
  bot: <><rect x="5" y="8" width="14" height="10" rx="2.5"></rect><path d="M12 4.5V8"></path><circle cx="9.5" cy="13" r="1" fill="currentColor"></circle><circle cx="14.5" cy="13" r="1" fill="currentColor"></circle></>,
  truck: <><path d="M3.5 6.5h10v9h-10zM13.5 9.5h4l3 3v3h-7"></path><circle cx="7" cy="17" r="1.6"></circle><circle cx="17" cy="17" r="1.6"></circle></>,
  route: <><circle cx="6" cy="18" r="2"></circle><circle cx="18" cy="6" r="2"></circle><path d="M8 18h7.5a3 3 0 0 0 0-6h-7a3 3 0 0 1 0-6H16"></path></>,
  receipt: <><path d="M6 3.5h12v17l-3-1.8-3 1.8-3-1.8-3 1.8z"></path><path d="M9.5 9h5M9.5 12.5h5M9.5 16h3"></path></>,
  wallet: <><rect x="3.5" y="6.5" width="17" height="12" rx="2"></rect><path d="M3.5 10h17M16 14.5h1.5"></path></>,
  shield: <><path d="M12 3.5l7 2.5v5.5c0 4.4-3 7.7-7 9-4-1.3-7-4.6-7-9V6z"></path><path d="M9 12l2 2 4-4"></path></>,
  headset: <><path d="M4.5 14v-2a7.5 7.5 0 0 1 15 0v2"></path><rect x="4" y="13.5" width="4" height="5.5" rx="1.5"></rect><rect x="16" y="13.5" width="4" height="5.5" rx="1.5"></rect></>,
  clipboard: <><rect x="6" y="5" width="12" height="15.5" rx="2"></rect><path d="M9.5 3.5h5v3h-5zM9.5 13l2 2 3.5-3.5"></path></>,
  compass: <><circle cx="12" cy="12" r="8.5"></circle><path d="M15.5 8.5l-2 5-5 2 2-5z"></path></>,
  radar: <><circle cx="12" cy="12" r="8.5"></circle><circle cx="12" cy="12" r="4.5"></circle><path d="M12 12l5.5-5.5"></path></>,
  gauge: <><path d="M4.5 16.5a8 8 0 1 1 15 0"></path><path d="M12 15l3.5-4.5"></path></>,
  package: <><path d="M12 3.5l8 4.5v8L12 20.5 4 16V8z"></path><path d="M4 8l8 4.5L20 8M12 12.5v8"></path></>,
  file: <><path d="M6 3.5h8l4 4v13H6z"></path><path d="M14 3.5v4h4M9 13h6M9 16.5h4"></path></>,
  search: <><circle cx="11" cy="11" r="6.5"></circle><path d="M20 20l-4.3-4.3"></path></>,
  bell: <><path d="M6 16.5V11a6 6 0 0 1 12 0v5.5l1.5 2h-15z"></path><path d="M10 20.5h4"></path></>,
  sparkle: <><path d="M2.7 10.3a2.41 2.41 0 0 0 0 3.41l7.59 7.59a2.41 2.41 0 0 0 3.41 0l7.59-7.59a2.41 2.41 0 0 0 0-3.41l-7.59-7.59a2.41 2.41 0 0 0-3.41 0Z"></path><circle cx="12" cy="12" r="2" fill="currentColor" stroke="none"></circle></>,
  inbox: <><path d="M4 13l2.5-7h11l2.5 7v5.5H4z"></path><path d="M4 13h4.5l1.5 2h4l1.5-2H20"></path></>,
  banknote: <><rect x="3" y="7" width="18" height="10" rx="2"></rect><circle cx="12" cy="12" r="2.5"></circle></>,
  coins: <><circle cx="9" cy="9.5" r="5"></circle><path d="M13.6 7.6A5 5 0 1 1 9.4 16.4"></path></>,
};
const AG_ACCENT = { indigo: 272, teal: 190, amber: 72, rose: 12, emerald: 158, sky: 232, violet: 302, slate: 255 };
const TEMPLATE_ICON = { DispatchAssistant: "truck", BillingAssistant: "receipt", ComplianceAssistant: "shield", CustomerAssistant: "headset", GeneralAssistant: "bot", BillingException: "receipt", DispatchAssignment: "route", ImportAssistant: "file", LoadMonitor: "radar", ShipmentIntake: "package", CashApplication: "wallet", DetentionDesk: "gauge", CredentialDesk: "clipboard", CustomerUpdateDesk: "bell", CarrierRiskDesk: "search", IntakeDesk: "inbox", InsightAnalyst: "compass", EDIDesk: "file", FormulaAssistant: "sparkle", BooksKeeper: "receipt", SettlementsClerk: "banknote", Receivables: "coins", WorkforceCoordinator: "clipboard", ReportAnalyst: "compass" };

function agHash(v) { let r = 0x811c9dc5; for (let i = 0; i < v.length; i++) { r ^= v.charCodeAt(i); r = Math.imul(r, 0x01000193) >>> 0; } return r >>> 0; }
function agIdentity(a) {
  const chosen = a.icon || TEMPLATE_ICON[a.template];
  const h = agHash(a.id || a.name);
  return { icon: chosen || "bot", chosen: !!chosen, accent: a.accent || Object.keys(AG_ACCENT)[h % 8], rot: [0, 45, 90, 135, 180, 225, 270, 315][h % 8], len: [14, 22, 30][Math.floor(h / 8) % 3] };
}
const agMono = n => { const w = (n || "").trim().split(/\s+/); return w.length > 1 ? (w[0][0] + w[1][0]).toUpperCase() : (w[0] || "").slice(0, 2).toUpperCase(); };

function AgentTile({ agent, size = "md", className = "" }) {
  const id = agIdentity(agent || {});
  const px = { xs: 20, sm: 24, md: 28, lg: 36 }[size];
  const g = { xs: 11, sm: 13, md: 15, lg: 18 }[size];
  return (
    <span className={"at at-" + size + (id.accent === "slate" ? " slate" : "") + " " + className} style={{ "--ah": AG_ACCENT[id.accent], width: px, height: px }}>
      {(size === "md" || size === "lg") && <svg className="at-sig" viewBox="0 0 32 32" fill="none"><circle cx="16" cy="16" r="14.25" pathLength="100" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeDasharray={`${id.len} ${100 - id.len}`} transform={`rotate(${id.rot - 90} 16 16)`}></circle></svg>}
      {id.chosen ? <svg width={g} height={g} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">{AG_ICON[id.icon]}</svg> : <b>{agMono(agent && agent.name)}</b>}
    </span>
  );
}

const AGENTS = [
  { id: "ag_billing", name: "Billing Specialist", template: "BillingAssistant", accent: "teal", desc: "Billing queue, invoices and posting", used: "now" },
  { id: "ag_dispatch", name: "Dispatch desk", template: "DispatchAssistant", accent: "indigo", desc: "Assigns drivers and tracks loads", used: "1h" },
  { id: "ag_bex", name: "Billing exceptions", template: "BillingException", accent: "rose", desc: "Invoices that can't post on their own", used: "2d" },
  { id: "ag_monitor", name: "Load Monitor", template: "LoadMonitor", accent: "sky", desc: "Watches loads for late or missed stops" },
  { id: "ag_detention", name: "Detention Desk", template: "DetentionDesk", accent: "amber", desc: "Detention time, claims and charges" },
  { id: "ag_cash", name: "Cash Application", template: "CashApplication", accent: "emerald", desc: "Matches payments to open invoices" },
  { id: "ag_receivables", name: "Receivables", template: "Receivables", accent: "emerald", desc: "Aging, collections and dunning" },
  { id: "ag_settle", name: "Settlements Clerk", template: "SettlementsClerk", accent: "teal", desc: "Carrier and driver settlements" },
  { id: "ag_cred", name: "Credential Desk", template: "CredentialDesk", accent: "violet", desc: "Driver and carrier credentials" },
  { id: "ag_risk", name: "Carrier Risk", template: "CarrierRiskDesk", accent: "rose", desc: "Insurance, authority and safety scores" },
  { id: "ag_updates", name: "Customer Updates", template: "CustomerUpdateDesk", accent: "sky", desc: "Status notes and delay notices" },
  { id: "ag_intake", name: "Shipment Intake", template: "ShipmentIntake", accent: "amber", desc: "Turns rate cons and tenders into loads" },
  { id: "ag_edi", name: "EDI Desk", template: "EDIDesk", accent: "slate", desc: "204s, 214s and 210s" },
  { id: "ag_insight", name: "Insight Analyst", template: "InsightAnalyst", accent: "violet", desc: "Spots trends across lanes and customers" },
  { id: "ag_report", name: "Report Analyst", template: "ReportAnalyst", accent: "indigo", desc: "Builds and explains reports" },
  { id: "ag_workforce", name: "Workforce Coordinator", template: "WorkforceCoordinator", accent: "teal", desc: "Schedules, pay and time off" },
  { id: "ag_formula", name: "Formula Assistant", template: "FormulaAssistant", accent: "violet", desc: "Writes rating and billing formulas" },
  { id: "ag_general", name: "General Assistant", template: "GeneralAssistant", accent: "slate", desc: "Anything across Trenova" },
  { id: "ag_huddle", name: "Ops Huddle", accent: "amber", desc: "Morning stand-up summary for the ops team" },
];

function AgentPicker({ agent, onSelect }) {
  const [open, setOpen] = React.useState(false);
  const [q, setQ] = React.useState("");
  const [hi, setHi] = React.useState(0);
  const root = React.useRef(null), inp = React.useRef(null), list = React.useRef(null);
  React.useEffect(() => {
    if (!open) return;
    setQ(""); setHi(0);
    const t = setTimeout(() => inp.current && inp.current.focus(), 30);
    const off = e => root.current && !root.current.contains(e.target) && setOpen(false);
    document.addEventListener("mousedown", off);
    return () => { clearTimeout(t); document.removeEventListener("mousedown", off); };
  }, [open]);
  const ql = q.trim().toLowerCase();
  const match = a => !ql || (a.name + " " + a.desc).toLowerCase().includes(ql);
  const recent = AGENTS.filter(a => a.used && match(a));
  const rest = AGENTS.filter(a => !a.used && match(a));
  const flat = [...recent, ...rest];
  React.useEffect(() => { setHi(0); }, [q]);
  React.useEffect(() => { const el = list.current && list.current.querySelector('[data-i="' + hi + '"]'); if (el) { const p = list.current; if (el.offsetTop < p.scrollTop) p.scrollTop = el.offsetTop - 30; else if (el.offsetTop + el.offsetHeight > p.scrollTop + p.clientHeight) p.scrollTop = el.offsetTop + el.offsetHeight - p.clientHeight + 4; } }, [hi]);
  const pick = a => { if (!a) return; onSelect(a); setOpen(false); };
  const onKey = e => {
    if (e.key === "ArrowDown") { e.preventDefault(); setHi(h => Math.min(flat.length - 1, h + 1)); }
    else if (e.key === "ArrowUp") { e.preventDefault(); setHi(h => Math.max(0, h - 1)); }
    else if (e.key === "Home") { e.preventDefault(); setHi(0); }
    else if (e.key === "End") { e.preventDefault(); setHi(flat.length - 1); }
    else if (e.key === "Enter") { e.preventDefault(); pick(flat[hi]); }
    else if (e.key === "Escape") { e.preventDefault(); e.stopPropagation(); setOpen(false); }
  };
  let n = -1;
  const row = a => { n += 1; const i = n; const sel = agent.id === a.id; return (
    <button key={a.id} data-i={i} className={"ap-r" + (hi === i ? " hi" : "") + (sel ? " sel" : "")} onMouseMove={() => hi !== i && setHi(i)} onClick={() => pick(a)}>
      <AgentTile agent={a} size="sm" />
      <span className="ap-t"><b>{a.name}</b><span>{a.desc}</span></span>
      {a.used && !sel && <span className="ap-ago">{a.used}</span>}
      {sel && <span className="ap-ck"><Ic n="check" s={13} w={2.4} /></span>}
    </button>); };
  return (
    <span className="ap" ref={root}>
      <button className={"ap-b" + (open ? " on" : "")} onClick={() => setOpen(o => !o)} aria-label={"Asking " + agent.name + ". Choose another agent"}>
        <AgentTile key={agent.id} agent={agent} size="xs" className="at-confirm" />
        <span className="ap-bn">{agent.name}</span>
        <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" className="ap-cv"><path d="M8 9l4-4 4 4M8 15l4 4 4-4"></path></svg>
      </button>
      {open && (
        <div className="ap-pop" onKeyDown={onKey}>
          <div className="ap-s"><Ic n="search" s={13} /><input ref={inp} value={q} onChange={e => setQ(e.target.value)} placeholder="Search agents" role="combobox" aria-expanded="true" /></div>
          <div className="ap-l" ref={list} role="listbox">
            {recent.length > 0 && <div className="ap-h">Recent</div>}
            {recent.map(row)}
            {recent.length > 0 && rest.length > 0 && <div className="ap-h">All agents</div>}
            {rest.map(row)}
            {!flat.length && <div className="ap-empty">No agents match “{q}”</div>}
          </div>
        </div>
      )}
    </span>
  );
}

Object.assign(window, { AgentTile, AgentPicker, AGENTS, agIdentity, AG_ICON });
