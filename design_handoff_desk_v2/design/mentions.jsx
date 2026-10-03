const MENTION_TYPES = {
  shipment: { label: "Shipments", icon: "truck" },
  customer: { label: "Customers", icon: "headset" },
  invoice: { label: "Invoices", icon: "receipt" },
  driver: { label: "Drivers", icon: "route" },
  carrier: { label: "Carriers", icon: "shield" },
};
const MENTION_INDEX = [
  ...["Chicago, IL → Columbus, OH · In transit", "Des Moines, IA → Omaha, NE · Late", "Joliet, IL → Indianapolis, IN · Needs driver", "Gary, IN → Detroit, MI · Needs driver", "Cedar Rapids, IA → Madison, WI · Late", "St. Louis, MO → Memphis, TN · Delivered", "Ames, IA → Minneapolis, MN · Late", "Chicago, IL → Columbus, OH · In transit"].map((s, i) => ({ type: "shipment", id: "shp_" + (i + 1), label: "SEED-SHP-00" + (i + 1), sub: s })),
  ...[["Acme Manufacturing", "Net 30 · 3 past 60 days"], ["Bluewater Retail", "Net 45 · on credit hold"], ["Granite Building", "Net 30"], ["Harbor Supply Co.", "Net 30 · 1 past 60 days"], ["Northline Foods", "Net 15"], ["Peak Distributing", "Net 30"]].map(([l, s], i) => ({ type: "customer", id: "cus_" + i, label: l, sub: s })),
  ...Array.from({ length: 6 }, (_, i) => ({ type: "invoice", id: "inv_" + i, label: "BQ-2410" + (i + 1), sub: ["Acme Manufacturing", "Bluewater Retail", "Granite Building", "Harbor Supply Co.", "Northline Foods", "Peak Distributing"][i] + " · Ready for review" })),
  ...[["Marcus Hill", "6h left today · Joliet"], ["Dana Ortiz", "4h left today · Gary"], ["Lena Park", "On SEED-SHP-002"], ["Tom Reyes", "On SEED-SHP-005"]].map(([l, s], i) => ({ type: "driver", id: "drv_" + i, label: l, sub: s })),
  ...[["Midwest Freightways", "MC 482190 · insured to Mar 2027"], ["Lakeshore Logistics", "MC 771203 · insurance expires in 9 days"]].map(([l, s], i) => ({ type: "carrier", id: "car_" + i, label: l, sub: s })),
];
const MENTION_RE = new RegExp("@(" + MENTION_INDEX.map(m => m.label.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")).sort((a, b) => b.length - a.length).join("|") + ")", "g");

function mentionSearch(q, type) {
  const ql = q.toLowerCase();
  return MENTION_INDEX.filter(m => (type === "all" || m.type === type) && (!ql || (m.label + " " + m.sub).toLowerCase().includes(ql))).slice(0, 8);
}

function MentionGlyph({ type, s = 13 }) {
  return <svg width={s} height={s} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">{AG_ICON[MENTION_TYPES[type].icon]}</svg>;
}

function MnTabs({ types, type, setType }) {
  const ref = React.useRef(null);
  const [edge, setEdge] = React.useState({ l: false, r: false });
  const upd = () => { const el = ref.current; if (!el) return; setEdge({ l: el.scrollLeft > 2, r: el.scrollLeft + el.clientWidth < el.scrollWidth - 2 }); };
  React.useEffect(() => { upd(); }, []);
  React.useEffect(() => { const el = ref.current && ref.current.querySelector(".on"); if (el) { const p = ref.current; const l = el.offsetLeft - 8, r = el.offsetLeft + el.offsetWidth + 8; if (l < p.scrollLeft) p.scrollTo({ left: l, behavior: "smooth" }); else if (r > p.scrollLeft + p.clientWidth) p.scrollTo({ left: r - p.clientWidth, behavior: "smooth" }); } }, [type]);
  const nudge = d => ref.current && ref.current.scrollBy({ left: d * 120, behavior: "smooth" });
  return (
    <div className={"mn-fw" + (edge.l ? " l" : "") + (edge.r ? " r" : "")}>
      {edge.l && <button className="mn-ar l" onClick={() => nudge(-1)} tabIndex={-1}><Ic n="chevL" s={11} w={2.4} /></button>}
      <div className="mn-f" ref={ref} onScroll={upd} onWheel={e => { if (Math.abs(e.deltaY) > Math.abs(e.deltaX)) { ref.current.scrollLeft += e.deltaY; } }}>
        {types.map(t => <button key={t} className={type === t ? "on" : ""} onClick={() => setType(t)}>{t === "all" ? "All" : MENTION_TYPES[t].label}</button>)}
      </div>
      {edge.r && <button className="mn-ar r" onClick={() => nudge(1)} tabIndex={-1}><Ic n="chevR" s={11} w={2.4} /></button>}
    </div>
  );
}

function MentionPicker({ q, type, setType, results, hi, setHi, onPick, loading }) {
  const types = ["all", ...Object.keys(MENTION_TYPES)];
  let last = null;
  return (
    <div className="mn" onMouseDown={e => e.preventDefault()}>
      <MnTabs types={types} type={type} setType={setType} />
      <div className="mn-l">
        {loading && <div className="mn-sk" aria-busy="true">{[0, 1, 2, 3].map(i => <div key={i} className="mn-skr" style={{ animationDelay: i * 80 + "ms" }}><i className="a"></i><span><i className="b" style={{ width: [46, 58, 38, 52][i] + "%" }}></i><i className="c" style={{ width: [70, 62, 76, 55][i] + "%" }}></i></span></div>)}</div>}
        {!loading && results.map((m, i) => {
          const head = type === "all" && m.type !== last; last = m.type;
          return (
            <React.Fragment key={m.id}>
              {head && <div className="mn-h">{MENTION_TYPES[m.type].label}</div>}
              <button className={"mn-r" + (hi === i ? " hi" : "")} onMouseMove={() => hi !== i && setHi(i)} onClick={() => onPick(m)}>
                <span className="mn-ic"><MentionGlyph type={m.type} /></span>
                <span className="mn-t"><b>{m.label}</b><em>{m.sub}</em></span>
                {hi === i && <span className="kbd">↵</span>}
              </button>
            </React.Fragment>
          );
        })}
        {!loading && !results.length && <div className="mn-empty">{q ? <>No records match “{q}”</> : "Start typing a load, customer, invoice or driver"}</div>}
      </div>
      <div className="mn-ft"><span><span className="kbd">↑</span><span className="kbd">↓</span>move</span><span><span className="kbd">↵</span>insert</span><span><span className="kbd">Esc</span>close</span></div>
    </div>
  );
}

function MentionMirror({ value }) {
  const parts = []; let at = 0; let m;
  MENTION_RE.lastIndex = 0;
  while ((m = MENTION_RE.exec(value))) { if (m.index > at) parts.push(value.slice(at, m.index)); parts.push(<mark key={m.index}>{m[0]}</mark>); at = m.index + m[0].length; }
  parts.push(value.slice(at) + "\u200b");
  return <div className="mn-mirror" aria-hidden="true">{parts}</div>;
}

function MentionText({ text }) {
  const parts = []; let at = 0; let m;
  MENTION_RE.lastIndex = 0;
  while ((m = MENTION_RE.exec(text))) {
    if (m.index > at) parts.push(text.slice(at, m.index));
    const rec = MENTION_INDEX.find(x => x.label === m[1]);
    parts.push(<span key={m.index} className="mn-chip" title={rec.sub}><MentionGlyph type={rec.type} s={12} />{rec.label}</span>);
    at = m.index + m[0].length;
  }
  parts.push(text.slice(at));
  return parts;
}

function useMentions(value, setValue, taRef) {
  const [st, setSt] = React.useState(null); // {start, q}
  const [type, setType] = React.useState("all");
  const [hi, setHi] = React.useState(0);
  const [loading, setLoading] = React.useState(false);
  const results = st ? mentionSearch(st.q, type) : [];
  React.useEffect(() => {
    if (!st) return;
    const off = e => { if (!e.target.closest(".mn") && e.target !== taRef.current && !e.target.closest('[title^="Mention"]')) setSt(null); };
    document.addEventListener("mousedown", off); return () => document.removeEventListener("mousedown", off);
  }, [!!st]);
  React.useEffect(() => { setHi(0); if (!st) return; setLoading(true); const t = setTimeout(() => setLoading(false), 160); return () => clearTimeout(t); }, [st && st.q, type]);
  const detect = (v, caret) => {
    const before = v.slice(0, caret);
    const m = /(^|\s)@([^\s@]{0,30})$/.exec(before);
    if (m) setSt({ start: caret - m[2].length - 1, q: m[2] }); else setSt(null);
  };
  const pick = rec => {
    if (!st) return;
    const ta = taRef.current;
    const caret = ta ? ta.selectionStart : value.length;
    const ins = "@" + rec.label + " ";
    const next = value.slice(0, st.start) + ins + value.slice(caret);
    setValue(next); setSt(null); setType("all");
    requestAnimationFrame(() => { if (ta) { ta.focus(); const p = st.start + ins.length; ta.setSelectionRange(p, p); } });
  };
  const open = () => { const ta = taRef.current; const caret = ta ? ta.selectionStart : value.length; const pre = value.slice(0, caret); const add = (pre && !/\s$/.test(pre) ? " " : "") + "@"; const next = pre + add + value.slice(caret); setValue(next); requestAnimationFrame(() => { if (ta) { ta.focus(); const p = caret + add.length; ta.setSelectionRange(p, p); detect(next, p); } }); };
  const onKey = e => {
    if (!st) return false;
    if (e.key === "ArrowDown") { e.preventDefault(); setHi(h => Math.min(results.length - 1, h + 1)); return true; }
    if (e.key === "ArrowUp") { e.preventDefault(); setHi(h => Math.max(0, h - 1)); return true; }
    if ((e.key === "Enter" || e.key === "Tab") && results[hi] && !loading) { e.preventDefault(); pick(results[hi]); return true; }
    if (e.key === "Escape") { e.preventDefault(); setSt(null); return true; }
    return false;
  };
  return { st, type, setType, hi, setHi, results, loading, detect, pick, open, onKey, close: () => setSt(null) };
}

Object.assign(window, { MentionPicker, MentionMirror, MentionText, useMentions, MENTION_INDEX });
