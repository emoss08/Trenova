const agOf = id => AGENTS0.find(a => a.id === id);
function AgCell({ id, sub }) { const a = agOf(id); return <span className="agc"><Tile a={a} s={22}></Tile><span className="tl"><b>{a.n}</b>{sub && <span>{sub}</span>}</span></span>; }
function Bd({ m }) { return <span className={"bdg " + (m[1] || "")}>{m[1] === "b" && <i className="spn"></i>}{m[0]}</span>; }
function Line({ d, w = 64, h = 18 }) { const mx = Math.max(...d), mn = Math.min(...d), r = mx - mn || 1; const pts = d.map((v, i) => `${(i / (d.length - 1)) * (w - 2) + 1},${h - 2 - ((v - mn) / r) * (h - 4)}`).join(" "); return <svg className="ln" width={w} height={h} viewBox={`0 0 ${w} ${h}`} fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><polyline points={pts}></polyline></svg>; }
function Pts({ v }) { if (v == null) return <span className="dim">—</span>; return <span className={"mono pts " + (v > 0 ? "t-k" : v < 0 ? "t-d" : "dim")}>{v > 0 ? "+" : v < 0 ? "−" : "±"}{Math.abs(v)} pts</span>; }
function Nova({ ctx, spin, children, ctl }) {
  return <section className="hero rh"><div className="hero-s"><span className="who"><span className={"dm" + (spin ? " spin" : "")}></span><b>Nova</b><span>{ctx}</span></span><p className="say">{children}</p></div>{ctl && <div className="hc">{ctl}</div>}</section>;
}
function Figs({ items }) { return <section className="figs" style={{ "--n": items.length }}>{items.map(([l, v, s, c]) => <div key={l} className="fg"><span className="lbl">{l}</span><b className={"big mono " + (c || "")}>{v}</b><span className="sub">{s}</span></div>)}</section>; }
function DT({ rows, cols, find, filters, onOpen, sel, per: per0 = 10, ph = "Search...", searchRef, tools, empty = "No results.", name = "row" }) {
  const [q, setQ] = React.useState(""); const [f, setF] = React.useState(filters ? filters[0][0] : null); const [so, setSo] = React.useState(null); const [pg, setPg] = React.useState(0); const [per, setPer] = React.useState(per0); const [hid, setHid] = React.useState([]); const [mn, setMn] = React.useState(null);
  React.useEffect(() => setPg(0), [q, f, so, per]);
  const vis = cols.filter(c => !hid.includes(c.k));
  let xs = rows.filter(r => (!q || find(r).toLowerCase().includes(q.toLowerCase())) && (!filters || filters.find(x => x[0] === f)[2](r)));
  if (so) { const c = cols.find(c => c.k === so.k); xs = [...xs].sort((a, b) => { const A = c.sv(a), B = c.sv(b); return (A > B ? 1 : A < B ? -1 : 0) * so.d; }); }
  const pages = Math.max(1, Math.ceil(xs.length / per)), page = xs.slice(pg * per, pg * per + per);
  const fAct = filters && f !== filters[0][0];
  const toggleSort = c => setSo(s => s && s.k === c.k ? (s.d === 1 ? { k: c.k, d: -1 } : null) : { k: c.k, d: 1 });
  return (
    <div className="dtx">
      <div className="dtx-tb">
        <label className="dtx-s"><Ic n="search" s={13}></Ic><input ref={searchRef} value={q} onChange={e => setQ(e.target.value)} onKeyDown={e => e.key === "Escape" && (setQ(""), e.target.blur())} placeholder={ph}></input><Ic n="help" s={13}></Ic></label>
        <div className="rel"><button className={"btn" + (fAct ? " act" : "")} onClick={() => setMn(mn === "f" ? null : "f")}><Ic n="filter" s={13}></Ic>Filter{fAct && <em className="mono">1</em>}</button>
          {mn === "f" && <Menu onClose={() => setMn(null)} items={filters ? [{ h: "Show" }, ...filters.map(([k, l, p]) => ({ icon: <span className={"cbx" + (f === k ? " on" : "")}>{f === k && <Ic n="check" s={10} w={3}></Ic>}</span>, l, s: rows.filter(p).length + " " + name + "s", on: () => setF(k) }))] : [{ h: "No filters for this table" }]}></Menu>}</div>
        <div className="rel"><button className={"btn" + (so ? " act" : "")} onClick={() => setMn(mn === "s" ? null : "s")}><Ic n="upDown" s={13}></Ic>Sort{so && <em className="mono">1</em>}</button>
          {mn === "s" && <Menu onClose={() => setMn(null)} items={[{ h: "Sort by" }, ...cols.filter(c => c.sv).map(c => ({ icon: <Ic n={so && so.k === c.k ? (so.d === 1 ? "up" : "down") : "upDown"} s={13}></Ic>, l: c.l, s: so && so.k === c.k ? (so.d === 1 ? "Ascending" : "Descending") : null, on: () => toggleSort(c) })), ...(so ? ["-", { icon: <Ic n="x" s={13}></Ic>, l: "Clear sort", on: () => setSo(null) }] : [])]}></Menu>}</div>
        <span className="sp"></span>
        <div className="rel"><button className="btn" onClick={() => setMn(mn === "d" ? null : "d")}><Ic n="gear" s={13}></Ic>Display</button>
          {mn === "d" && <Menu right onClose={() => setMn(null)} items={[{ h: "Columns" }, ...cols.filter(c => c.l).map(c => ({ icon: <span className={"cbx" + (hid.includes(c.k) ? "" : " on")}>{!hid.includes(c.k) && <Ic n="check" s={10} w={3}></Ic>}</span>, l: c.l, on: () => setHid(h => h.includes(c.k) ? h.filter(x => x !== c.k) : [...h, c.k]) }))]}></Menu>}</div>
        <div className="rel"><button className="btn" onClick={() => setMn(mn === "v" ? null : "v")}><Ic n="bookmark" s={13}></Ic>Views</button>
          {mn === "v" && <Menu right onClose={() => setMn(null)} items={[{ h: "Saved views" }, { icon: <Ic n="check" s={13}></Ic>, l: "Default", on: () => {} }, "-", { icon: <Ic n="plus" s={13}></Ic>, l: "Save current view", on: () => {} }]}></Menu>}</div>
        {tools}
      </div>
      <div className="dtx-f"><table className="dtx-t"><thead><tr>{vis.map(c => <th key={c.k} className={(c.r ? "r " : "") + (c.sv ? "so" : "") + (so && so.k === c.k ? " on" : "") + (c.hide ? " h" + c.hide : "")} style={{ width: c.w }} onClick={() => c.sv && toggleSort(c)}><span className="th-in">{c.l}{c.sv && <Ic n={so && so.k === c.k ? (so.d === 1 ? "up" : "down") : "upDown"} s={11}></Ic>}</span></th>)}</tr></thead>
        <tbody>{page.map((r, i) => <tr key={r.id || r.seq || r.n || i} className={sel === r ? "sel" : ""} onClick={() => onOpen && onOpen(r)}>{vis.map(c => <td key={c.k} className={(c.r ? "r " : "") + (c.hide ? "h" + c.hide : "")}>{c.c(r)}</td>)}</tr>)}</tbody>
      </table>{!xs.length && <div className="dtx-e"><b>{empty}</b><span>{q || fAct ? "Try another search, or clear the filter." : "Nothing has been recorded here yet."}</span></div>}</div>
      <div className="dtx-ft"><span>Showing <b>{xs.length ? pg * per + 1 : 0}</b> to <b>{Math.min(xs.length, pg * per + per)}</b> of <b>{xs.length}</b> results</span><span className="sp"></span>
        <span className="dtx-pp">Rows per page<select value={per} onChange={e => setPer(+e.target.value)}>{[10, 20, 50].map(n => <option key={n} value={n}>{n}</option>)}</select></span>
        <span className="dtx-pg"><button className="btn icon" disabled={pg === 0} onClick={() => setPg(pg - 1)}><Ic n="chevL" s={13}></Ic></button>Page <b>{pg + 1}</b> of <b>{pages}</b><button className="btn icon" disabled={pg >= pages - 1} onClick={() => setPg(pg + 1)}><Ic n="chevR" s={13}></Ic></button></span>
      </div>
    </div>
  );
}
function KV({ items }) { return <dl className="sfx">{items.filter(Boolean).map(([k, v]) => <div key={k}><dt>{k}</dt><dd>{v}</dd></div>)}</dl>; }
Object.assign(window, { agOf, AgCell, Bd, Line, Pts, Nova, Figs, DT, KV });
