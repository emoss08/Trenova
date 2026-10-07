function Top({ onToggleSide }) {
  return (
    <header className="top">
      <button className="ib" onClick={onToggleSide} title="Toggle sidebar"><Ic n="sidebar" s={15}></Ic></button>
      <button className="org"><img src="logo.png" alt=""></img><b>Trenova Logistics</b><Ic n="upDown" s={12}></Ic></button>
      <span className="vr"></span>
      <button className="mod"><Ic n="cog" s={14}></Ic>Settings<Ic n="chevD" s={12}></Ic></button>
      <span className="sp"></span>
      <button className="jump"><Ic n="search" s={13}></Ic><span>Search or jump to…</span><span className="kbd">{MOD + " K"}</span></button>
      <span className="sp"></span>
      <button className="ib bell"><Ic n="bell" s={15}></Ic><i></i></button>
      <span className="me">SA</span>
    </header>
  );
}
const SIDE = [["Organization", ["Organization settings", "Billing controls", "Dispatch controls", "Shipment controls", "Users & roles"]], ["AI & Automation", ["AI control", "Document intelligence"]], ["Documents", ["Scanning and printing", "Parsing rules"]], ["Data & Integrations", ["Integrations", "Inbound mailboxes", "API keys", "Custom fields"]]];
function Side() {
  return (
    <nav className="side"><div className="side-in">
      {SIDE.map(([g, xs]) => <React.Fragment key={g}><div className="side-gh">{g}</div>{xs.map(l => <button key={l} className={"nv" + (l === "AI control" ? " on" : "")}><span>{l}</span></button>)}</React.Fragment>)}
    </div></nav>
  );
}
const TABS = [["overview", "Overview"], ["agents", "Agents"], ["providers", "Providers"], ["extensions", "Extensions"], ["memory", "Memory"], ["retrieval", "Retrieval"], ["safety", "Safety"], ["quality", "Quality"], ["activity", "Activity"], ["audit", "Audit trail"]];
function PageHead({ tab, setTab, counts, state }) {
  const bar = React.useRef(); const [ind, setInd] = React.useState(null);
  React.useLayoutEffect(() => { const el = bar.current && bar.current.querySelector(".tab.on"); if (el) setInd({ l: el.offsetLeft, w: el.offsetWidth }); }, [tab, counts.agents, counts.providers, counts.memory, counts.retrieval, counts.extensions, counts.safety, counts.activity, counts.quality]);
  const pill = state === "none" ? <span className="stp n"><i></i>Not set up</span> : state === "paused" ? <span className="stp w"><Ic n="pause" s={11} w={2.4}></Ic>Paused</span> : null;
  return (
    <div className="ph">
      <div className="ph-r">
        <div className="ph-t"><span className="crumb">Settings<Ic n="chevR" s={11}></Ic>AI &amp; Automation</span><h1>AI control{pill}</h1></div>
      </div>
      <div className="tabs" ref={bar} role="tablist">
        {TABS.map(([k, l], i) => <button key={k} role="tab" className={"tab" + (tab === k ? " on" : "") + ""} onClick={() => setTab(k)}>{l}{counts[k] ? <em className={"mono" + (k === "retrieval" || k === "safety" || k === "activity" ? " w" : k === "quality" ? " d" : "")}>{counts[k]}</em> : null}{i < 9 && <span className="kbd">{i + 1}</span>}</button>)}
        {ind && <span className="tab-ind" style={{ left: ind.l, width: ind.w }}></span>}
      </div>
    </div>
  );
}
function Tile({ a, s = 30 }) {
  return <span className={"tile" + (a.h ? "" : " n")} style={{ "--h": a.h, width: s, height: s, borderRadius: Math.round(s * 0.28) }}><Ic n={a.ic} s={Math.round(s * 0.5)}></Ic></span>;
}
function Mark({ p, s = 28 }) {
  return <span className={"mk" + (p.c ? "" : " n")} style={{ "--h": p.h, "--c": p.c, width: s, height: s, fontSize: Math.max(9, Math.round(s * (p.m.length > 1 ? 0.36 : 0.46))), borderRadius: Math.round(s * 0.26) }}>{p.m}</span>;
}
function Switch({ on, onChange, disabled, label }) {
  return <button role="switch" aria-checked={on} aria-label={label} disabled={disabled} className={"swt" + (on ? " on" : "")} onClick={e => { e.stopPropagation(); onChange(!on); }}><i></i></button>;
}
function SecH({ t, n, r, ic }) {
  return <header className="sh2">{ic && <Ic n={ic} s={13}></Ic>}<h3>{t}</h3>{n != null && <em className="mono">{n}</em>}<span className="sp"></span>{r}</header>;
}
function Seg({ v, opts, onChange, cls = "" }) {
  return <div className={"seg " + cls}>{opts.map(([k, l]) => <button key={k} className={v === k ? "on" : ""} onClick={e => { e.stopPropagation(); onChange(k); }}>{l}</button>)}</div>;
}
function Search({ q, setQ, ph, inputRef }) {
  return (
    <label className="srch"><Ic n="search" s={13}></Ic><input ref={inputRef} value={q} onChange={e => setQ(e.target.value)} onKeyDown={e => e.key === "Escape" && (setQ(""), e.target.blur())} placeholder={ph}></input>{q ? <button className="ib xs" onClick={() => setQ("")}><Ic n="x" s={11}></Ic></button> : <span className="kbd">/</span>}</label>
  );
}
function Hold({ label, onDone, ms = 900 }) {
  const [p, setP] = React.useState(0); const raf = React.useRef(); const t0 = React.useRef();
  const start = () => { cancelAnimationFrame(raf.current); t0.current = performance.now(); const tick = now => { const v = Math.min(1, (now - t0.current) / ms); setP(v); if (v < 1) raf.current = requestAnimationFrame(tick); else { setP(0); onDone(); } }; raf.current = requestAnimationFrame(tick); };
  const stop = () => { cancelAnimationFrame(raf.current); setP(0); };
  return (
    <button className={"hold" + (p > 0 ? " ing" : "")} style={{ "--p": p }} onPointerDown={start} onPointerUp={stop} onPointerLeave={stop} onKeyDown={e => { if ((e.key === " " || e.key === "Enter") && !e.repeat) { e.preventDefault(); start(); } }} onKeyUp={stop}>
      <span className="hold-f"></span><span className="hold-t"><Ic n="pause" s={13} w={2.2}></Ic>{label}</span>
    </button>
  );
}
function Menu({ items, onClose, right }) {
  React.useEffect(() => { const c = e => !e.target.closest(".mn") && onClose(); const k = e => e.key === "Escape" && onClose(); setTimeout(() => document.addEventListener("mousedown", c)); window.addEventListener("keydown", k); return () => { document.removeEventListener("mousedown", c); window.removeEventListener("keydown", k); }; }, []);
  return <div className={"mn" + (right ? " right" : "")}>{items.map((x, i) => x === "-" ? <span key={i} className="mn-sep"></span> : x.h ? <div key={i} className="mn-h">{x.h}</div> : <button key={i} className="mn-i" onClick={() => { x.on(); onClose(); }}>{x.icon}<span className="mn-l"><b>{x.l}</b>{x.s && <em>{x.s}</em>}</span></button>)}</div>;
}
function Sheet({ onClose, head, children }) {
  React.useEffect(() => { const k = e => e.key === "Escape" && !document.querySelector(".sheet.es, .mdx") && onClose(); window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k); }, [onClose]);
  return ReactDOM.createPortal(
    <div className="shx" onMouseDown={e => e.target === e.currentTarget && onClose()}>
      <aside className="sheet" role="dialog" aria-modal="true">
        <header className="sh-h">{head}<button className="ib" title="Close (Esc)" onClick={onClose}><Ic n="x" s={14}></Ic></button></header>
        <div className="sh-b">{children}</div>
      </aside>
    </div>, document.querySelector(".tv"));
}
function Modal({ onClose, title, sub, foot, children }) {
  React.useEffect(() => { const k = e => e.key === "Escape" && onClose(); window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k); }, [onClose]);
  return ReactDOM.createPortal(
    <div className="shx mdx" onMouseDown={e => e.target === e.currentTarget && onClose()}>
      <div className="modal" role="dialog" aria-modal="true">
        <header className="md-h"><div><b>{title}</b>{sub && <span>{sub}</span>}</div><button className="ib" onClick={onClose}><Ic n="x" s={14}></Ic></button></header>
        <div className="md-b">{children}</div>
        {foot && <footer className="md-f">{foot}</footer>}
      </div>
    </div>, document.querySelector(".tv"));
}
Object.assign(window, { Top, Side, PageHead, TABS, Tile, Mark, Switch, SecH, Seg, Search, Hold, Menu, Sheet, Modal });
