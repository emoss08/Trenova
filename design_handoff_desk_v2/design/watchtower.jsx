const WT_KINDS = { run: "Failed agent run", service: "Service failure", cred: "Expiring credential", edi: "Quarantined EDI", late: "Late load", detention: "Detention", insurance: "Carrier insurance", billing: "Blocked invoice" };
const WT_SEV = { Critical: { label: "Critical", icon: "alert" }, Warning: { label: "Warning", icon: "info" }, Info: { label: "Info", icon: "info" } };
const ag = id => AGENTS.find(a => a.id === id);
const WT_ITEMS0 = [
  { id: "w1", sev: "Critical", kind: "late", title: "SEED-SHP-007 will miss its delivery window", summary: "Stuck behind the I-35 storm near Ames. Minneapolis dock closes at 7 PM; the driver is 3h 10m out.", ago: "4m", subj: "SEED-SHP-007", subjKind: "Shipment", facts: [["Customer", "Northline Foods"], ["Appointment", "Today 6:00 PM"], ["New ETA", "Today 9:10 PM"], ["Driver", "Will Grant"]], suggest: "ag_monitor", unseen: true },
  { id: "w2", sev: "Critical", kind: "insurance", title: "Lakeshore Logistics' cargo insurance lapses in 9 days", summary: "They're on 4 tendered loads next week. Their certificate on file expires Oct 11.", ago: "22m", subj: "Lakeshore Logistics", subjKind: "Carrier", facts: [["MC", "771203"], ["Expires", "Oct 11, 2026"], ["Loads affected", "4"]], suggest: "ag_risk", unseen: true,
    agent: { id: "ag_risk", status: "working", step: 1, steps: ["Read the certificate on file", "Checked loads tendered to Lakeshore", "Drafting a renewal request to their broker", "Flag loads for re-tender if no reply in 48h"] } },
  { id: "w3", sev: "Critical", kind: "run", title: "Cash Application stopped partway through a lockbox file", summary: "Matched 212 of 340 payments, then lost access to the bank feed.", ago: "38m", subj: "Lockbox · Oct 2", subjKind: "Batch", facts: [["Matched", "212 of 340"], ["Error", "Bank feed token expired"], ["Run", "run_01J9…4QX"]], suggest: "ag_cash", unseen: true },
  { id: "w4", sev: "Warning", kind: "detention", title: "Detention clock passed 2 hours at Acme Plant 2", summary: "SEED-SHP-001 has been at the dock since 1:40 PM. Billable after 2 free hours at $65/h.", ago: "51m", subj: "SEED-SHP-001", subjKind: "Shipment", facts: [["Arrived", "1:40 PM"], ["Free time", "2h"], ["Accrued", "$32.50"]], suggest: "ag_detention",
    agent: { id: "ag_detention", status: "working", step: 2, steps: ["Read the stop events", "Confirmed free time from the Acme contract", "Requesting the out-time from the driver", "Draft the detention charge"] } },
  { id: "w5", sev: "Warning", kind: "edi", title: "3 load tenders quarantined from Bluewater", summary: "The 204s reference a location code we don't have mapped (BW-DC-17).", ago: "1h", subj: "Bluewater Retail", subjKind: "Customer", facts: [["Messages", "3 × 204"], ["Unknown code", "BW-DC-17"]], suggest: "ag_edi", unseen: true },
  { id: "w6", sev: "Warning", kind: "cred", title: "Dana Ortiz's medical card expires in 12 days", summary: "She's assigned to SEED-SHP-004 next week.", ago: "2h", subj: "Dana Ortiz", subjKind: "Driver", facts: [["Expires", "Oct 14, 2026"], ["Next load", "SEED-SHP-004"]], suggest: "ag_cred",
    agent: { id: "ag_cred", status: "needs", step: 2, steps: ["Checked the credential on file", "Drafted a reminder to Dana and her dispatcher"], note: "Waiting for you to approve the reminder" } },
  { id: "w7", sev: "Warning", kind: "billing", title: "11 invoices can't post without a biller", summary: "Ready for review, but nobody is assigned as biller.", ago: "2h", subj: "Billing queue", subjKind: "Queue", facts: [["Invoices", "11"], ["Amount", "$34,120.50"]], suggest: "ag_billing" },
  { id: "w8", sev: "Warning", kind: "service", title: "Two late deliveries for Harbor Supply this week", summary: "A second service failure in 5 days. Their contract has a 95% on-time clause.", ago: "3h", subj: "Harbor Supply Co.", subjKind: "Customer", facts: [["On-time this month", "91%"], ["Contract minimum", "95%"]], suggest: "ag_updates" },
  { id: "w9", sev: "Info", kind: "late", title: "SEED-SHP-002 running about 2 hours late", summary: "Weather near Des Moines. Still inside the delivery window.", ago: "3h", subj: "SEED-SHP-002", subjKind: "Shipment", facts: [["New ETA", "6:45 PM"], ["Window", "until 8:00 PM"]], suggest: "ag_monitor",
    agent: { id: "ag_monitor", status: "done", step: 3, steps: ["Re-checked the ETA", "Told Bluewater's dock", "Set a watch for 30 min out"], note: "Customer notified · watching" } },
  { id: "w10", sev: "Info", kind: "cred", title: "Tom Reyes' hazmat endorsement renews next month", summary: "No loads need hazmat before then.", ago: "5h", subj: "Tom Reyes", subjKind: "Driver", facts: [["Expires", "Nov 18, 2026"]], suggest: "ag_cred" },
  { id: "w11", sev: "Info", kind: "edi", title: "Acme sent 6 status updates we didn't expect", summary: "214s for loads already delivered. Probably a resend.", ago: "6h", subj: "Acme Manufacturing", subjKind: "Customer", facts: [["Messages", "6 × 214"]], suggest: "ag_edi" },
];
const WT_HANDLED = [
  { id: "h1", sev: "Warning", kind: "detention", title: "Detention billed for SEED-SHP-006", by: "ag_detention", how: "Charged $97.50 · approved by you", ago: "Yesterday" },
  { id: "h2", sev: "Critical", kind: "run", title: "Settlements run retried and finished", by: "ag_settle", how: "Re-ran after the bank feed recovered", ago: "Yesterday" },
  { id: "h3", sev: "Info", kind: "edi", title: "2 Acme tenders mapped and accepted", by: "ag_edi", how: "Mapped ACME-DC-4 · accepted both", ago: "Sep 30" },
  { id: "h4", sev: "Warning", kind: "cred", title: "Marcus Hill's CDL renewal uploaded", by: null, how: "Dismissed by Jordan Pike", ago: "Sep 29" },
];

function WtGlyph({ kind, s = 13 }) {
  const map = { run: "bot", service: "headset", cred: "clipboard", edi: "file", late: "truck", detention: "gauge", insurance: "shield", billing: "receipt" };
  return <svg width={s} height={s} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">{AG_ICON[map[kind]]}</svg>;
}
function AgentRing({ a, s = 22 }) {
  const pct = a.status === "done" ? 1 : a.step / a.steps.length;
  const r = s / 2 + 2.5, c = 2 * Math.PI * r;
  return (
    <span className={"wt-ar st-" + a.status} style={{ width: s, height: s }}>
      <AgentTile agent={ag(a.id)} size="xs" />
      <svg width={s + 8} height={s + 8} viewBox={`0 0 ${s + 8} ${s + 8}`}><circle cx={s / 2 + 4} cy={s / 2 + 4} r={r} fill="none" stroke="currentColor" strokeOpacity=".18" strokeWidth="1.6"></circle><circle cx={s / 2 + 4} cy={s / 2 + 4} r={r} fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeDasharray={c} strokeDashoffset={c * (1 - pct)} transform={`rotate(-90 ${s / 2 + 4} ${s / 2 + 4})`} style={{ transition: "stroke-dashoffset 600ms" }}></circle></svg>
    </span>
  );
}

const WT_DUE = { w1: [190, "Minneapolis dock closes"], w2: [9 * 1440, "Insurance lapses"], w3: [0, "Payments are unmatched now"], w4: [-51, "Detention is accruing"], w5: [120, "Tenders expire"], w6: [12 * 1440, "Medical card expires"], w7: [360, "Billing cutoff"], w9: [105, "Delivery window closes"], w10: [46 * 1440, "Endorsement renews"] };
const WT_IMPACT = { w1: ["Northline Foods", "1 load", "Late fee $250"], w2: ["4 tendered loads", "$18,400 freight"], w3: ["128 payments", "$61,240 unapplied"], w4: ["$32.50 so far", "$65/h"], w5: ["3 loads", "$7,900"], w6: ["SEED-SHP-004"], w7: ["11 invoices", "$34,120.50"], w8: ["Harbor Supply", "95% on-time clause"], w9: ["Bluewater dock notified"], w10: [], w11: [] };
const WT_TRAIL = { w1: ["Picked up at Ames · 11:40 AM", "Storm warning on I-35 · 1:15 PM", "Speed under 20 mph for 40 min", "ETA moved past 6:00 PM appointment"], w4: ["Arrived at Acme Plant 2 · 1:40 PM", "Free time ended · 3:40 PM", "Still checked in"], w2: ["Certificate on file expires Oct 11", "4 loads tendered for Oct 6–10"], w3: ["Lockbox file received · 1:02 PM", "212 of 340 matched", "Bank feed token expired"] };
const fmtDue = m => { if (m == null) return null; if (m <= 0) return m < 0 ? Math.abs(m) + "m running" : "Now"; if (m < 60) return m + "m"; if (m < 1440) return Math.floor(m / 60) + "h " + (m % 60) + "m"; return Math.round(m / 1440) + " days"; };
const lane = m => m == null ? "fyi" : m <= 240 ? "now" : m <= 1440 ? "today" : "later";
const LANES = [["now", "Act now", "Inside 4 hours"], ["today", "Today", "Before end of day"], ["later", "Coming up", "Days out"], ["fyi", "For your information", "No deadline"]];

function Watchtower({ onAsk }) {
  const [items, setItems] = React.useState(WT_ITEMS0.map(x => ({ ...x, due: WT_DUE[x.id] ? WT_DUE[x.id][0] : null, dueLabel: WT_DUE[x.id] ? WT_DUE[x.id][1] : null })));
  const [cleared, setCleared] = React.useState(0);
  const [cur, setCur] = React.useState("w1");
  const [out, setOut] = React.useState(null);
  const [menu, setMenu] = React.useState(false);
  const [tick, setTick] = React.useState(0);
  React.useEffect(() => { const iv = setInterval(() => { setTick(t => t + 1); setItems(xs => xs.map(x => { let nx = x; if (x.due != null && x.due > 0 && x.due < 1440) nx = { ...nx, due: x.due - 1 }; if (x.agent && x.agent.status === "working" && Math.random() < .35) { const step = x.agent.step + 1; nx = { ...nx, agent: step >= x.agent.steps.length ? { ...x.agent, step, status: "needs", note: "Ready for you" } : { ...x.agent, step } }; } return nx; })); }, 3000); return () => clearInterval(iv); }, []);
  const order = [...items].sort((a, b) => (a.due == null ? 1e9 : a.due) - (b.due == null ? 1e9 : b.due));
  const c = items.find(x => x.id === cur) || order[0];
  const total = items.length + cleared;
  const go = d => { const k = order.findIndex(x => x.id === (c && c.id)); const n = order[(k + d + order.length) % order.length]; if (n) setCur(n.id); };
  const clear = (how) => { if (!c) return; setOut(how); setMenu(false); setTimeout(() => { const k = order.findIndex(x => x.id === c.id); const rest = order.filter(x => x.id !== c.id); setItems(xs => xs.filter(x => x.id !== c.id)); setCleared(n => n + 1); setOut(null); if (rest.length) setCur(rest[Math.min(k, rest.length - 1)].id); }, 300); };
  const handOff = id => { setItems(xs => xs.map(x => x.id === c.id ? { ...x, unseen: false, agent: { id: id || x.suggest, status: "working", step: 0, steps: ["Reading the " + x.subjKind.toLowerCase(), "Working out what to do", "Drafting the fix", "Asking you to approve"] } } : x)); setMenu(false); };
  React.useEffect(() => {
    const k = e => { if (e.target.closest && e.target.closest("input,textarea,select")) return; if (e.metaKey || e.ctrlKey || e.altKey) return;
      if (e.key === "j" || e.key === "ArrowDown") { e.preventDefault(); go(1); } else if (e.key === "k" || e.key === "ArrowUp") { e.preventDefault(); go(-1); }
      else if (e.key === "h" && c && !c.agent) { e.preventDefault(); handOff(); } else if (e.key === "e") { e.preventDefault(); clear("done"); } else if (e.key === "s") { e.preventDefault(); clear("snooze"); } };
    window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k);
  });
  React.useEffect(() => { if (c) setItems(xs => xs.map(x => x.id === c.id && x.unseen ? { ...x, unseen: false } : x)); }, [c && c.id]);
  const working = items.filter(x => x.agent && x.agent.status === "working");
  const a = c && c.agent;
  const L = c ? lane(c.due) : null;
  return (
    <div className="w3">
      <aside className="w3-q">
        <div className="dc2-qh"><b>Watchtower</b><span>{cleared + " of " + total + " cleared"}</span></div>
        <div className="dc2-prog"><i style={{ width: (total ? cleared / total * 100 : 0) + "%" }}></i></div>
        {working.length > 0 && <div className="w3-on"><span className="w3-onl">On it</span>{working.map(x => <button key={x.id} title={ag(x.agent.id).name + " · " + x.title} onClick={() => setCur(x.id)}><AgentRing a={x.agent} s={18} /></button>)}<span className="w3-onn">{working.length + " working"}</span></div>}
        <div className="dc2-ql">
          {LANES.map(([k, l, sub]) => { const xs = order.filter(x => lane(x.due) === k); if (!xs.length) return null; return (
            <div key={k} className={"dc2-g w3-l-" + k}>
              <div className="dc2-gh"><b className="w3-lt">{l}</b><i>{xs.length}</i></div>
              {xs.map(x => (
                <button key={x.id} className={"dc2-qi w3-qi" + (c && c.id === x.id ? " on" : "")} onClick={() => setCur(x.id)}>
                  <span className={"w3-ic s-" + x.sev}>{x.agent ? <AgentTile agent={ag(x.agent.id)} size="xs" /> : <WtGlyph kind={x.kind} s={12} />}</span>
                  <span><b>{x.title}</b><em>{x.agent ? (x.agent.status === "needs" ? ag(x.agent.id).name + " needs you" : ag(x.agent.id).name + " is on it") : x.subj}</em></span>
                  {x.due != null && x.due < 1440 ? <span className={"w3-due" + (x.due <= 60 ? " hot" : "")}>{fmtDue(x.due)}</span> : x.unseen ? <span className="dc2-new"></span> : null}
                </button>
              ))}
            </div>); })}
          {!items.length && <div className="dc2-qe">All clear</div>}
        </div>
      </aside>
      <main className="dc2-main">
        {c ? <div className={"dc2-card w3-card" + (out ? " out-" + (out === "done" ? "ok" : "no") : "")} key={c.id}>
          {c.due != null && <div className={"w3-clock l-" + L}>
            <span className="w3-cr"><svg width="44" height="44" viewBox="0 0 44 44"><circle cx="22" cy="22" r="19" fill="none" stroke="currentColor" strokeOpacity=".18" strokeWidth="3"></circle><circle cx="22" cy="22" r="19" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeDasharray={119.4} strokeDashoffset={119.4 * Math.min(1, Math.max(0, c.due) / 240)} transform="rotate(-90 22 22)" style={{ transition: "stroke-dashoffset 1s linear" }}></circle></svg></span>
            <span><b>{c.due > 0 ? fmtDue(c.due) : fmtDue(c.due)}</b><em>{c.due > 0 ? c.dueLabel + " in" : c.dueLabel}</em></span>
          </div>}
          <div className="w3-kind"><span className={"wt-sev s-" + c.sev}></span>{WT_KINDS[c.kind]}<span>{" · " + c.ago + " ago"}</span></div>
          <h1>{c.title}</h1>
          <p className="dc2-why">{c.summary}</p>
          {(WT_IMPACT[c.id] || []).length > 0 && <div className="w3-imp"><em className="w3-impl">At stake</em>{WT_IMPACT[c.id].map(t => <span key={t}>{t}</span>)}</div>}
          <button className="w3-rec"><WtGlyph kind={c.kind} s={14} /><span><em>{c.subjKind}</em><b>{c.subj}</b></span><span className="w3-recf">{c.facts.slice(0, 2).map(f => f[0] + " " + f[1]).join(" · ")}</span><Ic n="ext" s={12} /></button>
          {WT_TRAIL[c.id] && <ol className="w3-trail">{WT_TRAIL[c.id].map((t, k) => <li key={k} className={k === WT_TRAIL[c.id].length - 1 ? "last" : ""}><i></i>{t}</li>)}</ol>}
          {a ? <div className={"w3-ag st-" + a.status}>
            <div className="w3-agh"><AgentRing a={a} s={22} /><span><b>{ag(a.id).name}</b><em>{a.status === "working" ? "Step " + Math.min(a.step + 1, a.steps.length) + " of " + a.steps.length + " · " + a.steps[Math.min(a.step, a.steps.length - 1)] : a.note}</em></span>{a.status === "needs" && <button className="dc2-btn ink sm" onClick={() => clear("done")}>Review & approve</button>}</div>
            <div className="w3-agbar">{a.steps.map((s, k) => <i key={k} className={k < a.step ? "d" : k === a.step && a.status === "working" ? "n" : ""} title={s}></i>)}</div>
          </div> : <div className="w3-sug"><AgentTile agent={ag(c.suggest)} size="sm" /><span><b>{ag(c.suggest).name + " can take this"}</b><em>{ag(c.suggest).desc}</em></span></div>}
        </div> : <div className="dc2-clear"><span className="dc2-cic"><Ic n="check" s={22} w={2.2} /></span><b>All clear</b><span>{"You cleared " + cleared + " today. New signals show up here the moment they happen."}</span></div>}
        {c && <div className="dc2-bar">
          <button className="dc2-nav" onClick={() => go(-1)} title="Previous (K)"><Ic n="chevR" s={12} w={2.4} /></button>
          <button className="dc2-nav dn" onClick={() => go(1)} title="Next (J)"><Ic n="chevR" s={12} w={2.4} /></button>
          <button className="dc2-nav w3-ask" style={{ marginLeft: 6 }} onClick={() => onAsk && onAsk(c)} title="Ask about this"><Ic n="chat" s={13} /><span>Ask about this</span></button>
          <span style={{ flex: 1 }}></span>
          <button className="dc2-btn" onClick={() => clear("snooze")}>Snooze<span className="kbd">S</span></button>
          <button className="dc2-btn" onClick={() => clear("done")}>Done<span className="kbd">E</span></button>
          {!a && <span className="wt-hm"><button className="dc2-btn ink" onClick={() => handOff()}>{"Hand off"}<span className="kbd">H</span></button><button className="dc2-btn ink more" onClick={() => setMenu(m => !m)}><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round"><path d="M6 15l6-6 6 6"></path></svg></button>
            {menu && <div className="wt-hp">{AGENTS.slice(0, 8).filter(x => x.id !== c.suggest).map(x => <button key={x.id} onClick={() => handOff(x.id)}><AgentTile agent={x} size="sm" /><span><b>{x.name}</b><em>{x.desc}</em></span></button>)}</div>}</span>}
        </div>}
      </main>
    </div>
  );
}

Object.assign(window, { Watchtower, WT_ITEMS0 });
