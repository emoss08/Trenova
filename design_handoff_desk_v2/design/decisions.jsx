const DEC_ITEMS0 = [
  { id: "p1", agent: "ag_billing", title: "Assign you as biller", n: 11, noun: "invoices", field: "Biller", from: "None", to: "Avery Lane", rev: true, ago: "3m", why: "These are ready for review but can't post without a biller.", rows: ["BQ-24101 · Acme Manufacturing", "BQ-24103 · Granite Building", "BQ-24104 · Harbor Supply Co.", "BQ-24106 · Peak Distributing", "BQ-24108 · Bluewater Retail"], unseen: true },
  { id: "p2", agent: "ag_billing", title: "Post invoices", n: 15, noun: "invoices", field: "Status", from: "Ready for review", to: "Posted", rev: false, ago: "6m", why: "Totals $47,030.50 across 6 customers.", rows: ["BQ-24101 · $3,150.00", "BQ-24102 · $2,875.50", "BQ-24103 · $4,120.00", "BQ-24104 · $1,980.00", "BQ-24105 · $3,640.00"], fail: 3, unseen: true },
  { id: "p3", agent: "ag_dispatch", title: "Assign drivers to open loads", n: 2, noun: "loads", field: "Driver", from: "Unassigned", to: "Marcus Hill, Dana Ortiz", rev: true, ago: "22m", why: "Both have hours left and are under 20 miles from pickup.", rows: ["SEED-SHP-003 → Marcus Hill", "SEED-SHP-004 → Dana Ortiz"] },
  { id: "p4", agent: "ag_detention", title: "Bill detention", n: 1, noun: "charge", field: "Accessorial", from: "—", to: "$97.50 detention", rev: true, ago: "1h", why: "1.5 billable hours at Acme Plant 2 under the 2026 contract.", rows: ["SEED-SHP-001 · 1.5 h × $65"] },
  { id: "p5", agent: "ag_updates", title: "Send delay notices", n: 3, noun: "customers", field: "Message", from: "—", to: "Delay notice", rev: false, ago: "2h", why: "Storm delays on I-80 and I-35.", rows: ["Northline Foods · SEED-SHP-007", "Bluewater Retail · SEED-SHP-002", "Peak Distributing · SEED-SHP-005"] },
  { id: "p6", agent: "ag_cred", title: "Remind Dana Ortiz to renew her medical card", n: 1, noun: "message", field: "Message", from: "—", to: "Reminder", rev: true, ago: "2h", why: "It expires Oct 14 and she's on SEED-SHP-004 next week.", rows: ["Dana Ortiz · cc dispatcher"] },
];
const DEC_DONE0 = [
  { id: "d1", agent: "ag_settle", title: "Pay 14 carrier settlements", how: "Approved by you", ago: "Yesterday", ok: true },
  { id: "d2", agent: "ag_edi", title: "Map BW-DC-17 to Bluewater DC 17", how: "Approved by Jordan Pike", ago: "Yesterday", ok: true },
  { id: "d3", agent: "ag_billing", title: "Write off 2 short-paid invoices", how: "Declined · amounts too high", ago: "Sep 30", ok: false },
];

const DEC_TOOL = { p1: "billing.assign_biller", p2: "billing.post_invoices", p3: "shipments.assign_driver", p4: "billing.add_accessorial", p5: "messages.send", p6: "messages.send" };
const DEC_SIM = { p1: "All 11 would update. Nothing else changes.", p2: "12 would post. 3 would be refused: missing bill-to address.", p3: "Both drivers stay within their hours. Customers get the new ETA.", p4: "Adds $97.50 to invoice BQ-24101 before it posts.", p5: "3 emails send from billing@trenova.app.", p6: "1 email to Dana, copied to dispatch." };

function Decisions({ onAsk }) {
  const [items, setItems] = React.useState(DEC_ITEMS0.map(x => ({ ...x, tool: DEC_TOOL[x.id] })));
  const [done, setDone] = React.useState([]);
  const [cur, setCur] = React.useState(DEC_ITEMS0[0].id);
  const [leaving, setLeaving] = React.useState(null);
  const [batch, setBatch] = React.useState(null);
  const [raw, setRaw] = React.useState(false);
  const [seen, setSeen] = React.useState({});
  const total = items.length + done.length;
  React.useEffect(() => { window.DEC_WAITING = items.length; window.dispatchEvent(new Event("desk:dec-count")); }, [items.length]);
  const c = items.find(x => x.id === cur) || items[0];
  React.useEffect(() => { if (c) setSeen(s => ({ ...s, [c.id]: true })); setRaw(false); }, [c && c.id]);
  const go = dir => { if (!items.length) return; const i = items.findIndex(x => x.id === (c && c.id)); const n = items[(i + dir + items.length) % items.length]; setCur(n.id); };
  const decide = (ids, ok) => {
    setLeaving(ok ? "ok" : "no");
    setTimeout(() => {
      const idx = items.findIndex(x => x.id === ids[0]);
      const rest = items.filter(x => !ids.includes(x.id));
      setDone(dn => [...items.filter(x => ids.includes(x.id)).map(x => ({ ...x, ok })), ...dn]);
      setItems(rest); setBatch(null); setLeaving(null);
      if (rest.length) setCur(rest[Math.min(idx, rest.length - 1)].id);
    }, 320);
  };
  React.useEffect(() => {
    const k = e => {
      if (e.target.closest && e.target.closest("input,textarea,select")) return;
      if (e.metaKey || e.ctrlKey || e.altKey) return;
      if (e.key === "j" || e.key === "ArrowDown") { e.preventDefault(); go(1); }
      else if (e.key === "k" || e.key === "ArrowUp") { e.preventDefault(); go(-1); }
      else if (e.key === "a" && c) { e.preventDefault(); decide(batch || [c.id], true); }
      else if (e.key === "d" && c) { e.preventDefault(); decide(batch || [c.id], false); }
      else if (e.key === "x" && c) { e.preventDefault(); const like = items.filter(x => x.tool === c.tool).map(x => x.id); setBatch(b => b ? null : like.length > 1 ? like : null); }
    };
    window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k);
  });
  const groups = [];
  items.forEach(x => { let g = groups.find(g => g.tool === x.tool); if (!g) groups.push(g = { tool: x.tool, items: [] }); g.items.push(x); });
  const a = c && AGENTS.find(y => y.id === c.agent);
  const like = c ? items.filter(x => x.tool === c.tool) : [];
  return (
    <div className="dc2">
      <aside className="dc2-q">
        <div className="dc2-qh"><b>Queue</b><span>{done.length + " of " + total + " decided"}</span></div>
        <div className="dc2-prog"><i style={{ width: (total ? done.length / total * 100 : 0) + "%" }}></i></div>
        <div className="dc2-ql">
          {groups.map(g => (
            <div key={g.tool} className="dc2-g">
              <div className="dc2-gh"><code>{g.tool}</code><i>{g.items.length}</i></div>
              {g.items.map(x => (
                <button key={x.id} className={"dc2-qi" + (c && c.id === x.id ? " on" : "") + (batch && batch.includes(x.id) ? " in" : "")} onClick={() => setCur(x.id)}>
                  <AgentTile agent={AGENTS.find(y => y.id === x.agent)} size="xs" />
                  <span><b>{x.title}</b><em>{x.n + " " + x.noun + (x.rev ? "" : " · final")}</em></span>
                  {!seen[x.id] && <span className="dc2-new"></span>}
                </button>
              ))}
            </div>
          ))}
          {!items.length && <div className="dc2-qe">Queue clear</div>}
          {done.length > 0 && <div className="dc2-g done"><div className="dc2-gh">Decided today<i>{done.length}</i></div>{done.map(x => <div key={x.id} className="dc2-qd"><span className={x.ok ? "ok" : "no"}><Ic n={x.ok ? "check" : "x"} s={10} w={2.6} /></span>{x.title}</div>)}</div>}
        </div>
      </aside>
      <main className="dc2-main">
        {c ? (
          <div className={"dc2-card" + (leaving ? " out-" + leaving : "")} key={c.id}>
            <div className="dc2-who"><AgentTile agent={a} size="sm" /><span><b>{a.name}</b> wants to</span><span className="dc2-ago">{c.ago + " ago"}</span><button className="ec-link" onClick={onAsk}>See the conversation</button></div>
            <h1>{c.title}<span>{" · " + c.n + " " + c.noun}</span></h1>
            <p className="dc2-why">{c.why}</p>
            <div className="dc2-diff">
              <div className="dc2-dh"><span>{c.field}</span><span className={"dc2-rev" + (c.rev ? "" : " hard")}>{c.rev ? "Reversible" : "Can't be undone"}</span></div>
              {c.rows.map((r, k) => <div key={r} className={"dc2-dr" + (c.fail && k < c.fail ? " bad" : "")}><span className="dc2-dk">{r}</span><s>{c.from}</s><i>→</i><b>{c.fail && k < c.fail ? "Refused" : c.to}</b></div>)}
              {c.n > c.rows.length && <div className="dc2-more">{"+ " + (c.n - c.rows.length) + " more like these"}</div>}
            </div>
            <div className={"dc2-sim" + (c.fail ? " warn" : "")}><span className="dc2-simh"><Ic n={c.fail ? "alert" : "shield"} s={12} w={2} />Dry run</span>{DEC_SIM[c.id]}</div>
            <button className="dc2-raw" onClick={() => setRaw(r => !r)}><Ic n="chevR" s={10} w={2.4} />Arguments the agent sent</button>
            {raw && <pre className="dc2-pre">{JSON.stringify({ tool: c.tool, count: c.n, set: { [c.field.toLowerCase()]: c.to } }, null, 2)}</pre>}
            {like.length > 1 && <div className="dc2-like">{batch ? <><b>{"Deciding " + batch.length + " together"}</b><span>{"All " + c.tool + " · " + batch.filter(id => !seen[id]).length + " not opened yet"}</span><button className="ec-link" onClick={() => setBatch(null)}>Just this one</button></> : <><span>{(like.length - 1) + " more " + c.tool + " waiting"}</span><button className="ec-link" onClick={() => setBatch(like.map(x => x.id))}>Decide all {like.length} together</button><span className="kbd">X</span></>}</div>}
          </div>
        ) : <div className="dc2-clear"><span className="dc2-cic"><Ic n="check" s={22} w={2.2} /></span><b>Queue clear</b><span>{"You decided " + done.length + " things. Agents will ask here before they change anything else."}</span></div>}
        {c && <div className="dc2-bar">
          <button className="dc2-nav" onClick={() => go(-1)} title="Previous (K)"><Ic n="chevR" s={12} w={2.4} /></button>
          <button className="dc2-nav dn" onClick={() => go(1)} title="Next (J)"><Ic n="chevR" s={12} w={2.4} /></button>
          <span className="dc2-pos">{(items.findIndex(x => x.id === c.id) + 1) + " / " + items.length}</span>
          <span style={{ flex: 1 }}></span>
          <button className="dc2-btn" onClick={() => decide(batch || [c.id], false)}>{batch ? "Decline " + batch.length : "Decline"}<span className="kbd">D</span></button>
          <button className="dc2-btn ink" onClick={() => decide(batch || [c.id], true)}>{batch ? "Approve " + batch.length : c.fail ? "Approve " + (c.n - c.fail) + ", skip " + c.fail : "Approve"}<span className="kbd">A</span></button>
        </div>}
      </main>
    </div>
  );
}

Object.assign(window, { Decisions, DEC_ITEMS0 });
