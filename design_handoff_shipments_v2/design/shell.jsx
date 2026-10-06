function Top({ onToggleSide }) {
  return (
    <header className="top">
      <button className="ib" onClick={onToggleSide} title="Toggle sidebar (Ctrl B)"><Ic n="sidebar" s={15}></Ic></button>
      <button className="org"><img src="logo.png" alt=""></img><b>Trenova Logistics</b><Ic n="upDown" s={12}></Ic></button>
      <span className="vr"></span>
      <button className="mod"><Ic n="truck" s={14}></Ic>Shipments<Ic n="chevD" s={12}></Ic></button>
      <span className="sp"></span>
      <button className="jump"><Ic n="search" s={13}></Ic><span>Search or jump to…</span><span className="kbd">{(/Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent) ? "⌘" : "Ctrl") + " K"}</span></button>
      <span className="sp"></span>
      <button className="ib bell"><Ic n="bell" s={15}></Ic><i></i></button>
      <span className="me">SA</span>
    </header>
  );
}

function Side({ total, late }) {
  return (
    <nav className="side"><div className="side-in">
      <button className="nv on"><Ic n="box" s={14}></Ic><span>Shipments</span><em>{total}</em></button>
      <button className="nv"><Ic n="repeat" s={14}></Ic><span>Recurring shipments</span></button>
      <button className="nv"><Ic n="receipt" s={14}></Ic><span>Orders</span></button>
      <button className="nv"><Ic n="alert" s={14}></Ic><span>Service failures</span>{late ? <em className="d">{late}</em> : null}</button>
      <div className="side-gh">Configuration</div>
      {["Shipment types", "Service types", "Hazardous materials", "Commodities"].map(l => <button key={l} className="nv"><span>{l}</span></button>)}
    </div></nav>
  );
}

function PageHead({ count, ago, onRefresh, spinning, onNew }) {
  return (
    <div className="ph">
      <h1>Shipments</h1>
      <span className="sp"></span>
      <button className={"ib" + (spinning ? " spin" : "")} onClick={onRefresh} title="Refresh"><Ic n="refresh" s={14}></Ic></button>
      <button className="btn ink" onClick={onNew}><Ic n="plus" s={13}></Ic>New shipment</button>
    </div>
  );
}

const CITIES = [...new Set(SHIPMENTS.flatMap(s => [s.o[1].split(",")[0], s.d[1].split(",")[0]]))];
const STAGES = [["risk", "Late", "var(--danger)"], ["open", "Needs coverage", "var(--warn)"], ["move", "Moving", "var(--brand)"], ["sched", "Scheduled", "var(--teal)"], ["done", "Delivered", "var(--ok)"]];
const stageOf = s => s.st === "delayed" ? "risk" : !s.drv ? "open" : s.st === "transit" ? "move" : s.st === "assigned" ? "sched" : "done";
function StageBar({ list, filters, setFilters }) {
  const [hov, setHov] = React.useState(null);
  const act = (filters.find(f => f.k === "stage") || {}).v;
  const total = list.length || 1;
  const data = STAGES.map(([k, l, c]) => { const xs = list.filter(s => stageOf(s) === k); return { k, l, c, n: xs.length, rev: xs.reduce((t, s) => t + s.rev, 0) }; }).filter(d => d.n);
  const pick = d => setFilters(f => act === d.k ? f.filter(x => x.k !== "stage") : [...f.filter(x => x.k !== "stage" && x.k !== "seg"), { k: "stage", v: d.k, label: d.l }]);
  const money = n => n >= 10000 ? "$" + (n / 1000).toFixed(n >= 100000 ? 0 : 1) + "k" : "$" + n.toLocaleString();
  return (
    <div className={"stb" + (act || hov ? " foc" : "")} onMouseLeave={() => setHov(null)}>
      {data.map(d => (
        <button key={d.k} className={"stb-s" + ((act || hov) === d.k ? " on" : "")} style={{ flexGrow: Math.max(d.n, total * 0.13), "--c": d.c }} onMouseEnter={() => setHov(d.k)} onClick={() => pick(d)}>
          <span className="stb-l"><span className="stb-n mono">{d.n.toLocaleString()}</span><span className="stb-t">{d.l}</span></span>
          <span className="stb-bar"></span>
          <span className="stb-r mono">{money(d.rev)}</span>
        </button>
      ))}
    </div>
  );
}
function Capacity({ list, pool, org, hos, onAssign, onAct, onFilter, onTender }) {
  const [tab, setTab] = React.useState(org === "brokerage" ? "carriers" : "drivers");
  const [used, setUsed] = React.useState([]);
  const [hov, setHov] = React.useState(null);
  const [pos, setPos] = React.useState(null);
  const [leaving, setLeaving] = React.useState(null);
  const [tendered, setTendered] = React.useState(false);
  const dock = React.useRef();
  React.useEffect(() => setTab(org === "brokerage" ? "carriers" : "drivers"), [org]);
  const mode = org === "asset" ? "drivers" : org === "brokerage" ? "carriers" : tab;
  const drivers = pool.filter(d => !used.includes(d.n));
  const ready = drivers.filter(d => d.free === 0).sort((a, b) => a.mi - b.mi);
  const soon = drivers.filter(d => d.free > 0 && d.free <= 2).sort((a, b) => a.free - b.free);
  const carriers = CARRIERS.filter(c => !used.includes(c.n));
  const posting = carriers.filter(c => c.trucks > 0).sort((a, b) => b.acc - a.acc);
  const usual = carriers.filter(c => c.trucks === 0).sort((a, b) => b.acc - a.acc);
  const sent = list.filter(s => !s.drv && s.tenderTo).length;
  const unc = list.filter(s => !s.drv && !s.tenderTo && HOURS[s.id]).sort((a, b) => HOURS[a.id][0] - HOURS[b.id][0]);
  const short = Math.max(0, unc.length - ready.length - soon.length);
  const items = mode === "drivers" ? [["Ready now", ready, false], ["Within 2h", soon, true]] : [["Trucks posted", posting, false], ["Usually accept", usual, true]];
  const all = items.flatMap(g => g[1]);
  const match = (d, k) => unc[(all.indexOf(d) * 3 + k) % Math.max(1, unc.length)];
  const fmt = h => { const x = ((h % 24) + 24) % 24; return String(Math.floor(x)).padStart(2, "0") + ":" + String(Math.floor((x % 1) * 60)).padStart(2, "0"); };
  const assign = (d, s) => { setLeaving(d.n); setHov(null); setTimeout(() => { if (!d.carrier || d.trucks <= 1) setUsed(u => [...u, d.n]); setLeaving(null); }, 320); onAssign(s.id, d); };
  React.useEffect(() => { if (!hov) return; const c = e => { if (!e.target.closest(".dv-pop") && !e.target.closest(".dv-b")) setHov(null); }; const k = e => e.key === "Escape" && setHov(null); const sc = () => setHov(null); const el = dock.current; document.addEventListener("mousedown", c); window.addEventListener("keydown", k); el && el.addEventListener("scroll", sc); return () => { document.removeEventListener("mousedown", c); window.removeEventListener("keydown", k); el && el.removeEventListener("scroll", sc); }; }, [hov]);
  const [edge, setEdge] = React.useState("at-s");
  React.useEffect(() => { const el = dock.current; if (!el) return; const f = () => setEdge((el.scrollLeft <= 2 ? "at-s " : "") + (el.scrollLeft + el.clientWidth >= el.scrollWidth - 2 ? "at-e" : "")); f(); el.addEventListener("scroll", f, { passive: true }); const ro = new ResizeObserver(f); ro.observe(el); return () => { el.removeEventListener("scroll", f); ro.disconnect(); }; }, [mode, used.length]);
  const ring = (p, lo) => { const r = 17, c = 2 * Math.PI * r; return <svg className="ring" viewBox="0 0 40 40"><circle cx="20" cy="20" r={r} className="ring-bg"></circle><circle cx="20" cy="20" r={r} className={"ring-fg" + (lo ? " lo" : "")} strokeDasharray={c} strokeDashoffset={c * (1 - p)}></circle></svg>; };
  const chip = (d, dim) => {
    const isC = !!d.carrier, showRing = isC || hos;
    return (
      <div key={d.n} className={"dv" + (dim ? " soon" : "") + (leaving === d.n ? " out" : "") + (hov === d.n ? " on" : "")}>
        <button className="dv-b" onClick={e => { const r = e.currentTarget.getBoundingClientRect(), root = e.currentTarget.closest(".tv").getBoundingClientRect(); setPos({ x: Math.min(r.left - root.left, root.width - 356), y: r.bottom - root.top + 4 }); setHov(h => h === d.n ? null : d.n); }}>
          <span className="dv-av">{showRing && ring(isC ? d.acc / 100 : Math.min(1, d.hosN / 11), isC ? d.acc < 80 : d.hosN < 4)}<span className={"av" + (isC ? " sq" : "")} style={{ "--h": d.h }}>{d.i}</span>{!dim && <i className="dv-ok">{isC && d.trucks > 1 ? d.trucks : ""}</i>}</span>
          <span className="dv-n">{isC ? d.n.split(" ")[0].replace(/[.,]/g, "") : d.n.split(" ")[0]}</span>
          <span className="dv-c mono">{isC ? "$" + d.rate.toFixed(2) + "/mi" : dim ? "free " + fmt(NOW + d.free) : d.city.split(",")[0]}</span>
        </button>
        {hov === d.n && pos && match(d, 0) && ReactDOM.createPortal(
          <div className="dv-pop" style={{ left: pos.x, top: pos.y }} onMouseDown={e => e.stopPropagation()}>
            <div className="dv-ph"><b>{d.n}</b><span className="mono">{isC ? d.unit + " · " + d.acc + "% accept · " + (d.trucks ? d.trucks + " truck" + (d.trucks > 1 ? "s" : "") + " posted" : "no trucks posted") : d.unit + (hos ? " · " + d.hos + " HOS" : "") + " · " + d.city}</span></div>
            <div className="dv-pl">{isC ? "Best loads to tender" : "Best load for " + d.n.split(" ")[0]}</div>
            {[0, 1].map(k => { const s = match(d, k); if (!s || (k && s === match(d, 0))) return null; const q = isC ? Math.round(s.mi * d.rate / 10) * 10 : 0; return (
              <div key={k} className={"dv-m" + (k ? " alt" : "")}>
                <div><b>{s.o[1].split(",")[0] + " → " + s.d[1].split(",")[0]}</b><span className="mono">{"pickup " + fmt(HOURS[s.id][0]) + (isC ? " · quote $" + q.toLocaleString() + " · margin " + Math.round((1 - q / s.rev) * 100) + "%" : " · " + (d.mi + k * 9) + " mi out · $" + s.rev.toLocaleString())}</span></div>
                <button className={"btn sm" + (k ? "" : " ink")} onClick={() => assign(d, s)}>{isC ? "Tender" : "Assign"}</button>
              </div>); })}
          </div>, document.querySelector(".tv"))}
      </div>
    );
  };
  const sum = mode === "drivers" ? (
    <>
      <div className="cap-big"><b className="mono">{ready.length}</b><span>{"drivers ready for " + unc.length + " uncovered loads"}</span></div>
      <div className="cap-bar"><i className="r" style={{ flex: ready.length }}></i><i className="s" style={{ flex: soon.length }}></i><i className="x" style={{ flex: short }}></i></div>
      <div className="cap-lg"><span><i className="r"></i>{ready.length + " ready"}</span><span><i className="s"></i>{soon.length + " within 2h"}</span>{short > 0 && <span><i className="x"></i>{short + " short"}</span>}</div>
      {short > 0 && (org === "both"
        ? (tendered ? <span className="cap-done"><Ic n="check" s={12} w={2.4}></Ic>{short + " loads sent to carriers"}</span> : <button className="btn sm" onClick={() => { const ids = unc.slice(-short).map(x => x.id); onTender(ids); setTendered(true); setTab("carriers"); onAct("Tendered " + short + " loads to your carrier network"); }}>{"Tender " + short + " to carriers"}</button>)
        : <button className="btn sm" onClick={() => onFilter([{ k: "seg", v: "unassigned", label: "Uncovered" }])}>{"Review " + short + " you can't cover"}</button>)}
    </>
  ) : (
    <>
      <div className="cap-big"><b className="mono">{posting.length}</b><span>{"carriers posting trucks for " + unc.length + " untendered loads"}</span></div>
      <div className="cap-bar"><i className="s" style={{ flex: sent }}></i><i className="x" style={{ flex: unc.length }}></i></div>
      <div className="cap-lg">{sent > 0 && <span><i className="s"></i>{sent + " awaiting acceptance"}</span>}{unc.length > 0 && <span><i className="x"></i>{unc.length + " not tendered"}</span>}<span>{"avg $" + (posting.reduce((t, c) => t + c.rate, 0) / Math.max(1, posting.length)).toFixed(2) + "/mi"}</span></div>
      {unc.length > 0 ? <button className="btn sm" onClick={() => { onTender(unc.map(x => x.id)); onAct("Tendered " + unc.length + " loads to best-match carriers"); }}>{"Tender all " + unc.length + " to best matches"}</button> : sent > 0 ? <span className="cap-done"><Ic n="check" s={12} w={2.4}></Ic>Everything is tendered</span> : null}
    </>
  );
  return (
    <div className="cap">
      <div className="cap-sum">
        {org === "both" ? <div className="cap-tabs">{[["drivers", "Drivers", ready.length], ["carriers", "Carriers", posting.length]].map(([k, l, n]) => <button key={k} className={tab === k ? "on" : ""} onClick={() => { setTab(k); setHov(null); }}>{l}<em className="mono">{n}</em></button>)}</div> : <span className="cap-k">{mode === "drivers" ? "Driver capacity" : "Carrier capacity"}</span>}
        {sum}
      </div>
      <div className="cap-dock">
        <div className={"dock " + edge} ref={dock} key={mode}>
          {items.map(([g, xs, dim], gi) => xs.length ? <React.Fragment key={g}>{gi > 0 && <span className="dock-sep"></span>}<span className="dock-g">{g}</span>{xs.map(d => chip(d, dim))}</React.Fragment> : null)}
        </div>
        <div className="dock-f">
          <span>{mode === "carriers" ? "Ring shows acceptance rate on your lanes · click a carrier to tender" : hos ? "Ring shows hours of service left · click a driver to see their best load" : "Click a driver to see their best load"}</span>
          {mode === "drivers" && !hos && <button className="lnk3"><Ic n="plug" s={11}></Ic>Connect ELD for hours of service</button>}
          <span className="sp"></span>
          <button className="ib" onClick={() => dock.current.scrollBy({ left: -320, behavior: "smooth" })}><Ic n="chevL" s={13}></Ic></button>
          <button className="ib" onClick={() => dock.current.scrollBy({ left: 320, behavior: "smooth" })}><Ic n="chevR" s={13}></Ic></button>
        </div>
      </div>
    </div>
  );
}
function parseQuery(q) {
  const t = q.toLowerCase(), f = [];
  if (/late|delay|behind|risk|storm|weather|miss|appointment/.test(t)) f.push({ k: "seg", v: "risk", label: "Late" });
  else if (/uncover|no driver|unassigned|need.*driver|without/.test(t)) f.push({ k: "seg", v: "unassigned", label: "Uncovered" });
  else if (/moving|transit|rolling/.test(t)) f.push({ k: "seg", v: "transit", label: "Moving" });
  else if (/deliver.*today|today/.test(t)) f.push({ k: "seg", v: "today", label: "Delivering today" });
  if (/reefer|cold|frozen/.test(t)) f.push({ k: "equip", v: "Reefer", label: "Reefer" });
  if (/dry/.test(t)) f.push({ k: "equip", v: "Dry", label: "Dry van" });
  if (/low margin|margin/.test(t)) f.push({ k: "margin", v: 15, label: "Margin < 15%" });
  if (/midwest/.test(t)) f.push({ k: "region", v: "Midwest", label: "Midwest" });
  CITIES.forEach(c => { if (t.includes(c.toLowerCase())) f.push({ k: "city", v: c, label: c }); });
  SHIPMENTS.forEach(s => { const w = s.cust.split(" ")[0]; if (w.length > 3 && t.includes(w.toLowerCase()) && !f.find(x => x.v === s.cust)) f.push({ k: "cust", v: s.cust, label: s.cust }); });
  if (!f.length && q.trim()) f.push({ k: "text", v: q.trim(), label: `“${q.trim()}”` });
  return f;
}
function applyFilters(list, f) {
  return list.filter(s => f.every(x => x.k === "stage" ? stageOf(s) === x.v : x.k === "hour" ? (s.today && s.st !== "new" && Math.floor((HOURS[s.id][2] || HOURS[s.id][1])) === x.v) : x.k === "pwin" ? (!s.drv && HOURS[s.id][0] - NOW >= x.v[0] && HOURS[s.id][0] - NOW < x.v[1]) : x.k === "bill" ? (s.st === "done" && (billOf(s) || [])[0] === "Ready to bill") : x.k === "det" ? (!!s.det && !!s.drv && s.st !== "done" && !(window.__detBilled || []).includes(s.id)) : x.k === "seg" ? segFilter(x.v, s) : x.k === "equip" ? s.equip.startsWith(x.v) : x.k === "margin" ? s.mg < x.v : x.k === "region" ? s.reg.includes(x.v) : x.k === "city" ? (s.o[1] + s.d[1]).includes(x.v) : x.k === "cust" ? s.cust === x.v : (s.id + s.bol + s.cust + s.o[1] + s.d[1] + (s.drv ? s.drv.n : "")).toLowerCase().includes(x.v.toLowerCase())));
}
function segFilter(k, s) { return k === "all" ? true : k === "transit" ? s.st === "transit" : k === "risk" ? s.st === "delayed" : k === "unassigned" ? !s.drv : s.today && s.st !== "new"; }

const AI_EXAMPLES = ["Which loads will miss their appointment?", "Uncovered reefers picking up today", "Everything going into Chicago", "Low-margin loads this week"];
const PLAIN_FILTERS = [["Late", "late", "var(--danger)"], ["Uncovered", "no driver", "var(--warn)"], ["Moving", "moving", "var(--brand)"], ["Delivering today", "today", "var(--ok)"], ["Reefer", "reefer", "var(--faint)"], ["Low margin", "margin", "var(--faint)"]];

function FilterField({ filters, setFilters, list, inputRef }) {
  const [q, setQ] = React.useState("");
  const [focus, setFocus] = React.useState(false);
  const add = text => { const f = parseQuery(text); setFilters(x => [...x.filter(a => !f.find(b => b.k === a.k)), ...f]); setQ(""); };
  return (
    <div className="ff">
      <Ic n="filter" s={13}></Ic>
      {filters.map((f, i) => <span key={f.k + f.v} className="tok">{f.label}<button onClick={() => setFilters(x => x.filter((_, j) => j !== i))}><Ic n="x" s={10}></Ic></button></span>)}
      <input ref={inputRef} value={q} onChange={e => setQ(e.target.value)} onFocus={() => setFocus(true)} onBlur={() => setTimeout(() => setFocus(false), 120)}
        onKeyDown={e => { if (e.key === "Enter" && q.trim()) add(q); if (e.key === "Backspace" && !q && filters.length) setFilters(x => x.slice(0, -1)); if (e.key === "Escape") e.target.blur(); }}
        placeholder={filters.length ? "Add filter" : "Search shipments…"}></input>
      {filters.length > 0 ? <button className="ib" style={{ width: 22, height: 22 }} onClick={() => setFilters([])} title="Clear"><Ic n="x" s={11}></Ic></button> : <span className="kbd">/</span>}
      {focus && !q && (
        <div className="sugs">
          <div className="sugs-h">Quick filters</div>
          {PLAIN_FILTERS.map(([l, k, c]) => <button key={l} className="sg" onMouseDown={e => e.preventDefault()} onClick={() => add(k)}><i style={{ width: 7, height: 7, borderRadius: 4, background: c }}></i><span>{l}</span><em>{applyFilters(list, parseQuery(k)).length}</em></button>)}
        </div>
      )}
    </div>
  );
}
Object.assign(window, { Capacity, StageBar, stageOf, Top, Side, PageHead, FilterField, applyFilters, segFilter, parseQuery, AI_EXAMPLES });
