const CORE_TOOLS = ["find_in_trenova", "recall_memory", "open_page", "delegate_to_agent"];
const PREREQ = { release_billing_hold: ["get_billing_queue_item"], hold_billing_queue_item: ["list_billing_queue_items"], assign_driver: ["search_shipments"], tender_to_carrier: ["search_shipments"], send_customer_reply: ["read_inbound_message"], attach_document_to_shipment: ["read_document"], apply_customer_payment: ["get_accounting_sync_status"], correct_charge_code: ["get_shipment"] };
const resOf = t => t.needs ? t.needs.split(" · ")[0] : t.kind === "Runtime" ? "Runtime" : "General";
const EXPIRE = [[3600, "1 hour"], [14400, "4 hours"], [86400, "1 day"], [259200, "3 days"], [604800, "7 days"]];
function impliedOf(v) {
  const m = {};
  Object.keys(v.tools).forEach(n => (PREREQ[n] || []).forEach(p => { if (!(p in v.tools)) (m[p] = m[p] || []).push(n); }));
  return Object.entries(m).map(([p, by]) => ({ t: TOOLS.find(x => x.n === p), by: by.map(n => TOOLS.find(x => x.n === n)) })).filter(x => x.t);
}
function ToolPickerDialog({ v, set, onClose }) {
  const [q, setQ] = React.useState(""); const [g, setG] = React.useState("all");
  const all = TOOLS.filter(t => !CORE_TOOLS.includes(t.n));
  const groups = [...new Set(all.map(resOf))].sort();
  const needle = q.trim().toLowerCase();
  const vis = all.filter(t => (g === "all" || resOf(t) === g) && (!needle || (t.l + t.n + resOf(t)).toLowerCase().includes(needle)));
  const shown = [...new Set(vis.map(resOf))].sort();
  const n = Object.keys(v.tools).filter(k => !CORE_TOOLS.includes(k)).length;
  const toggle = t => set("tools", o => { const x = { ...o }; if (t.n in x) delete x[t.n]; else x[t.n] = t.kind === "Action" ? tierCap(t.tier, v.ceil) : "AutoExecute"; return x; });
  React.useEffect(() => { const k = e => { if (e.key === "Escape") { e.preventDefault(); e.stopPropagation(); onClose(); } }; window.addEventListener("keydown", k, true); return () => window.removeEventListener("keydown", k, true); }, []);
  return (
    <div className="tpk-x" onMouseDown={e => e.target === e.currentTarget && onClose()}>
      <div className="tpk" role="dialog" aria-label="Add tools">
        <header className="tpk-h"><div><b>Tools</b><span>Pick what this agent can read and change. Tiers are set on the agent.</span></div><button className="ib" onClick={onClose}><Ic n="x" s={14}></Ic></button></header>
        <div className="tpk-s"><label className="srch"><Ic n="search" s={13}></Ic><input autoFocus value={q} onChange={e => setQ(e.target.value)} placeholder={"Search " + all.length + " tools"}></input></label></div>
        <div className="tpk-m">
          <nav className="tpk-n">
            <button className={g === "all" ? "on" : ""} onClick={() => setG("all")}><span>All tools</span><em className="mono">{n}</em></button>
            {groups.map(r => { const c = all.filter(t => resOf(t) === r && t.n in v.tools).length; return <button key={r} className={g === r ? "on" : ""} onClick={() => setG(r)}><span>{r}</span>{c > 0 && <em className="mono">{c}</em>}</button>; })}
          </nav>
          <div className="tpk-l">{shown.map(r => <div key={r} className="tpk-g"><div className="tpk-gh">{r}</div>{vis.filter(t => resOf(t) === r).sort((a, b) => a.kind === b.kind ? 0 : a.kind === "Action" ? 1 : -1).map(t => { const on = t.n in v.tools; return (
            <button key={t.n} className={"tpk-r" + (on ? " on" : "")} onClick={() => toggle(t)}><span className="tpk-c">{on && <Ic n="check" s={11} w={3}></Ic>}</span><span className="tpk-t"><b>{t.l}</b><em>{t.kind === "Action" ? "Changes · " + EGRESS[t.e][0] + " · up to " + TIER_L[t.tier].toLowerCase() : "Reads"}{t.ext !== "Never" ? " · returns outside text" : ""}</em></span><span className="mono tpk-nm">{t.n}</span></button>); })}</div>)}
            {!vis.length && <div className="nil">No tools match “{q}”.</div>}</div>
        </div>
        <footer className="tpk-f"><span><b className="mono">{n}</b> chosen · 4 always on</span><span className="sp"></span><button className="btn ink" onClick={onClose}>Done</button></footer>
      </div>
    </div>
  );
}
function ToolBench({ v, set }) {
  const [q, setQ] = React.useState(""); const [f, setF] = React.useState("all"); const [pick, setPick] = React.useState(false); const [core, setCore] = React.useState(false);
  const held = Object.keys(v.tools).filter(n => !CORE_TOOLS.includes(n)).map(n => TOOLS.find(t => t.n === n)).filter(Boolean);
  const acts = held.filter(t => t.kind === "Action"), reads = held.filter(t => t.kind !== "Action");
  const eff = t => tierCap(v.tools[t.n], v.ceil);
  const counts = TIER_O.map(tr => acts.filter(t => eff(t) === tr).length);
  const imp = impliedOf(v);
  const needle = q.trim().toLowerCase();
  const list = held.filter(t => (f === "all" || (f === "act" ? t.kind === "Action" : t.kind !== "Action")) && (!needle || (t.l + t.n + resOf(t)).toLowerCase().includes(needle)));
  const groups = [...new Set(list.map(resOf))].sort();
  const rm = n => set("tools", o => { const x = { ...o }; delete x[n]; return x; });
  const setLimit = (n, val) => set("limits", o => ({ ...o, [n]: val === "" ? "" : Math.min(10000, +val.replace(/\D/g, "") || 0) }));
  return (
    <div className="bench">
      <div className="bench-l">
        <div className="bench-tb">
          <label className="srch"><Ic n="search" s={13}></Ic><input value={q} onChange={e => setQ(e.target.value)} placeholder="Filter this agent's tools"></input>{q && <button className="ib xs" onClick={() => setQ("")}><Ic n="x" s={11}></Ic></button>}</label>
          <div className="seg sm">{[["all", "All", held.length], ["act", "Changes", acts.length], ["read", "Reads", reads.length]].map(([k, l, c]) => <button key={k} className={f === k ? "on" : ""} onClick={() => setF(k)}>{l}<em className="mono">{c}</em></button>)}</div>
          <button className="btn sm" onClick={() => setPick(true)}><Ic n="plus" s={12}></Ic>Add tools</button>
        </div>
        <div className="bench-hd"><span>Tool</span><span>Freedom</span><span>Daily limit</span><span></span></div>
        <div className="bench-list">
          {groups.map(r => <div key={r} className="bg"><div className="bg-h">{r}<em className="mono">{list.filter(t => resOf(t) === r).length}</em></div>
            {list.filter(t => resOf(t) === r).sort((a, b) => a.kind === b.kind ? a.l.localeCompare(b.l) : a.kind === "Action" ? 1 : -1).map(t => { const isA = t.kind === "Action", own = v.tools[t.n], capped = isA && tierIx(own) > tierIx(v.ceil); return (
              <div key={t.n} className="br">
                <span className="br-t"><span className="eg-d" style={{ "--h": EGRESS[t.e][1], "--c": EGRESS[t.e][2] }} title={EGRESS[t.e][0]}></span><span><b>{t.l}</b><em>{isA ? EGRESS[t.e][0] : "Reads"}{t.ext !== "Never" ? " · outside text" : ""}</em></span></span>
                {isA ? <span className={"t3" + (capped ? " capped" : "")}>{TIER_O.map(tr => { const over = tierIx(tr) > tierIx(t.tier), above = tierIx(tr) > tierIx(v.ceil); return <button key={tr} disabled={over} className={(own === tr ? "on" : "") + (above ? " above" : "")} title={over ? "This tool can't go past " + TIER_L[t.tier] : above ? "Above the ceiling — runs as " + TIER_L[v.ceil] : ""} onClick={() => set("tools", o => ({ ...o, [t.n]: tr }))}>{over ? <Ic n="lock" s={9}></Ic> : null}{TIER_L[tr]}</button>; })}</span> : <span className="br-rd">Always runs</span>}
                {isA ? <label className="br-lim"><input className="mono" value={v.limits[t.n] ?? ""} placeholder="No limit" onChange={e => setLimit(t.n, e.target.value)}></input><span>/day</span></label> : <span className="br-rd dim">—</span>}
                <button className="ib xs br-x" title={"Remove " + t.l} onClick={() => rm(t.n)}><Ic n="x" s={11}></Ic></button>
              </div>); })}</div>)}
          {!held.length && <div className="bench-e"><b>No tools yet</b><span>Without tools it can only explain how Trenova works.</span><button className="btn sm ink" onClick={() => setPick(true)}><Ic n="plus" s={12}></Ic>Add tools</button></div>}
          {held.length > 0 && !list.length && <div className="bench-e"><span>Nothing matches.</span></div>}
        </div>
        <div className={"bench-ft" + (core ? " open" : "")}>
          <button className="bf" onClick={() => setCore(c => !c)} aria-expanded={core}><Ic n="chevR" s={11}></Ic><span>Also held</span><em className="mono">{CORE_TOOLS.length + imp.length}</em><span className="bf-s">{CORE_TOOLS.length} every agent has{imp.length ? " · " + imp.length + " that come with your tools" : ""}</span></button>
          {core && <div className="bf-g">
            <span className="bf-k">Every agent</span><div className="bf-c">{CORE_TOOLS.map(n => <span key={n} className="bfc">{TOOLS.find(t => t.n === n).l}</span>)}</div>
            {imp.length > 0 && <><span className="bf-k">With your tools</span><div className="bf-c">{imp.map(x => <span key={x.t.n} className="bfc" title={"Needed by " + x.by.map(b => b.l).join(", ")}>{x.t.l}<em>for {x.by.map(b => b.l).join(", ")}</em></span>)}</div></>}
          </div>}
        </div>
      </div>
      <aside className="bench-r">
        <div className="bx"><span className="bx-l">Ceiling</span>
          <div className="cl">{TIER_O.map((t, i) => <button key={t} className={"cl-o" + (v.ceil === t ? " on" : "")} onClick={() => set("ceil", t)}><span className="cl-r"></span><span><b>{TIER_L[t]}</b><em>{["Every change is a proposal", "Changes run once a person approves", "Changes may run without asking"][i]}</em></span></button>)}</div>
          <span className="bx-h">No tool goes past this, whatever it's set to.</span></div>
        <div className="bx"><span className="bx-l">How its {acts.length} changes run</span>
          <span className="dist">{counts.map((c, i) => c ? <i key={i} className={"d" + i} style={{ flex: c }}></i> : null)}{!acts.length && <i className="dz0"></i>}</span>
          <div className="dist-l">{TIER_O.map((t, i) => <span key={t}><i className={"d" + i}></i>{TIER_L[t]}<b className="mono">{counts[i]}</b></span>)}</div></div>
        <div className="bx"><span className="bx-l">Data access</span><Seg v={v.data} opts={[["Internal", "Internal"], ["Restricted", "Restricted"]]} onChange={x => set("data", x)} cls="sm"></Seg><span className="bx-h">{v.data === "Restricted" ? "May read restricted records like pay and medical files." : "Reads internal records only. Pay, medical and other restricted data stay out."}</span></div>
        <div className="bx"><span className="bx-l">Proposals expire after</span><Sel v={v.expire} on={x => set("expire", +x)} opts={EXPIRE}></Sel><span className="bx-h">An undecided proposal is withdrawn after this.</span></div>
      </aside>
      {pick && <ToolPickerDialog v={v} set={set} onClose={() => setPick(false)}></ToolPickerDialog>}
    </div>
  );
}
Object.assign(window, { ToolBench, ToolPickerDialog, CORE_TOOLS, EXPIRE, resOf });
