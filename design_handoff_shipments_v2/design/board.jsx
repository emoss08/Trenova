const GROUPS = [["risk", "Needs attention", "var(--danger)", s => s.st === "delayed"], ["open", "Needs coverage", "var(--warn)", s => !s.drv], ["move", "Moving", "var(--brand)", s => s.st === "transit"], ["sched", "Scheduled", "var(--teal)", s => s.st === "assigned"], ["done", "Delivered", "var(--ok)", s => s.st === "done"]];
const $ = n => "$" + n.toLocaleString();
const city = c => c.split(",")[0];
const groupOf = s => GROUPS.findIndex(g => g[3](s));
const FACETS = [["status", "Status", s => ST[s.st]], ["equip", "Equipment", s => s.equip.replace(" 53′", "")], ["tender", "Tender", s => s.tender], ["cust", "Customer", s => s.cust]];

function SRow({ s, i, cols, on, kb, chk, flash, onOpen, onCheck, onAct }) {
  return (
    <tr className={"rw" + (on ? " on" : "") + (kb ? " kb" : "") + (flash ? " flash" : "")} style={{ animationDelay: Math.min(i, 14) * 18 + "ms" }} onClick={() => onOpen(s.id)}>
      <td className="cc"><span className={"cbx" + (chk ? " on" : "")} onClick={e => { e.stopPropagation(); onCheck(s.id); }}>{chk && <Ic n="check" s={10} w={3}></Ic>}</span></td>
      {cols.map(c => <td key={c.id} className={c.r ? "r" : ""}><Cell id={c.id} s={s}></Cell></td>)}
      <td className="r act"><span className={"xc" + (on ? " on" : "")}><Ic n="chevD" s={13}></Ic></span><RowMenu s={s} onAct={onAct}></RowMenu></td>
    </tr>
  );
}

function Timeline({ rows, openId, onOpen }) {
  const A = 4, B = 24, pct = h => ((Math.max(A, Math.min(B, h)) - A) / (B - A)) * 100;
  const ticks = [4, 8, 12, 16, 20];
  const sorted = [...rows].sort((a, b) => (HOURS[a.id] || [99])[0] - (HOURS[b.id] || [99])[0]);
  return (
    <div className="tl">
      <div className="tl-h"><div>Shipment</div><div className="tl-ax">{ticks.map(t => <span key={t} style={{ left: pct(t) + "%" }}>{String(t).padStart(2, "0")}:00</span>)}</div></div>
      {sorted.map((s, i) => {
        const [p, d, x] = HOURS[s.id] || [0, 0];
        return (
          <div key={s.id} className={"tl-r" + (openId === s.id ? " on" : "")} style={{ animationDelay: i * 22 + "ms" }} onClick={() => onOpen(s.id)}>
            <div className="tl-n"><b>{city(s.o[1])} → {city(s.d[1])}</b><span>{s.id} · {s.drv ? s.drv.n : "no driver"}</span></div>
            <div className="tl-t">
              {ticks.map(t => <span key={t} className="tl-grid" style={{ left: pct(t) + "%" }}></span>)}
              {p >= B ? <span className="tl-b new" style={{ right: 6 }}>pickup tomorrow →</span> : <>
                <span className={"tl-b " + s.st + (d > B ? " open" : "")} style={{ left: pct(p) + "%", width: `calc(${pct(d) - pct(p)}% - 2px)` }}>{d > B ? "→ " + s.eta : s.st === "done" ? "✓ " + s.eta : s.eta}</span>
                {x && <span className="tl-x" style={{ left: pct(d) + "%", width: pct(x) - pct(d) + "%" }}></span>}
              </>}
            </div>
          </div>
        );
      })}
      <div className="tl-now" style={{ left: `calc(220px + (100% - 220px) * ${pct(NOW) / 100})` }}><span>11:30</span></div>
    </div>
  );
}

function Pop({ btnCls, onClose, children, style }) {
  const ref = React.useRef();
  React.useEffect(() => { const h = e => { if (ref.current && !ref.current.contains(e.target) && !e.target.closest("." + btnCls)) onClose(); }; document.addEventListener("mousedown", h); return () => document.removeEventListener("mousedown", h); }, []);
  return <div className="fm" ref={ref} style={style}>{children}</div>;
}

function FacetMenu({ list, facets, setFacets, onClose }) {
  const toggle = (k, v) => setFacets(f => { const cur = f[k] || []; return { ...f, [k]: cur.includes(v) ? cur.filter(x => x !== v) : [...cur, v] }; });
  const n = Object.values(facets).flat().length;
  return (
    <Pop btnCls="fbtn" onClose={onClose}>
      {FACETS.map(([k, l, fn]) => <div key={k} className="fm-g"><div className="fm-h">{l}</div>
        {[...new Set(list.map(fn))].map(v => { const on = (facets[k] || []).includes(v); return <button key={v} className="fm-i" onClick={() => toggle(k, v)}><span className={"cbx" + (on ? " on" : "")}>{on && <Ic n="check" s={10} w={3}></Ic>}</span><span className="fm-l">{v}</span><em>{list.filter(s => fn(s) === v).length}</em></button>; })}</div>)}
      <div className="fm-f"><button className="btn ghost sm" disabled={!n} onClick={() => setFacets({})}>Clear all</button><span className="sp"></span><button className="btn sm" onClick={onClose}>Done</button></div>
    </Pop>
  );
}

function SortMenu({ sort, setSort, onClose }) {
  return (
    <Pop btnCls="sbtn" onClose={onClose} style={{ width: 220 }}>
      <div className="fm-h">Sort by</div>
      {COLS.map(c => { const on = sort && sort.k === c.id; return <button key={c.id} className="fm-i" onClick={() => setSort(on ? { k: c.id, d: sort.d === "asc" ? "desc" : "asc" } : { k: c.id, d: "asc" })}><span className="fm-l">{c.label}</span>{on && <em style={{ display: "inline-flex", alignItems: "center", gap: 4, color: "var(--fg)" }}><Ic n={sort.d === "asc" ? "up" : "down"} s={11} w={2}></Ic>{sort.d === "asc" ? "Asc" : "Desc"}</em>}</button>; })}
      <div className="fm-f"><button className="btn ghost sm" disabled={!sort} onClick={() => setSort(null)}>Reset</button><span className="sp"></span><button className="btn sm" onClick={onClose}>Done</button></div>
    </Pop>
  );
}

function Board({ filters, setFilters, filterRef, list, rows: rows0, view, setView, group, setGroup, openId, kbId, onOpen, checked, onCheck, onCheckAll, flashId, railOpen, setRailOpen, onClearFilters, ai, onAssign, onNotify, onAct, onAsk, assignedNow }) {
  const [shut, setShut] = React.useState([]);
  const [facets, setFacets] = React.useState({});
  const [menu, setMenu] = React.useState(null);
  const [sort, setSort] = React.useState(null);
  const [page, setPage] = React.useState(0);
  const [per, setPer] = React.useState(10);
  const [hidden, setHidden] = React.useState(DEFAULT_HIDDEN);
  const cols = COLS.filter(c => !hidden.includes(c.id));
  const span = cols.length + 2;
  const nF = Object.values(facets).flat().length;
  const rows = rows0.filter(s => FACETS.every(([k, , fn]) => !(facets[k] || []).length || facets[k].includes(fn(s))));
  React.useEffect(() => setPage(0), [rows.length, per, group]);
  let flat = [...rows];
  if (sort) { const fn = COLS.find(c => c.id === sort.k).sort; flat.sort((a, b) => { const x = fn(a), y = fn(b); return (x > y ? 1 : x < y ? -1 : 0) * (sort.d === "asc" ? 1 : -1); }); }
  if (group && view === "table") flat = flat.filter(s => !shut.includes(GROUPS[groupOf(s)][0])).sort((a, b) => groupOf(a) - groupOf(b));
  const pages = Math.max(1, Math.ceil(flat.length / per)), pg = Math.min(page, pages - 1);
  const pageRows = flat.slice(pg * per, pg * per + per);
  window.__rows = pageRows.map(s => s.id);
  React.useEffect(() => { if (openId && !pageRows.find(s => s.id === openId)) { const i = flat.findIndex(s => s.id === openId); if (i >= 0) setPage(Math.floor(i / per)); } }, [openId]);
  const allChk = pageRows.length > 0 && pageRows.every(r => checked.includes(r.id));
  const cycle = k => setSort(s => !s || s.k !== k ? { k, d: "asc" } : s.d === "asc" ? { k, d: "desc" } : null);
  const bds = React.useRef();
  const row = (s, i) => {
    const out = [<SRow key={s.id} s={s} i={i} cols={cols} on={openId === s.id} kb={kbId === s.id} chk={checked.includes(s.id)} flash={flashId === s.id} onOpen={onOpen} onCheck={onCheck} onAct={onAct}></SRow>];
    if (openId === s.id) out.push(<tr key={s.id + "x"} className="xrow"><td colSpan={span}><div className="xpw x2w"><Expand s={s} ai={ai} onAssign={onAssign} onNotify={onNotify} onAct={onAct} onAsk={onAsk} assignedNow={assignedNow === s.id} onCollapse={() => onOpen(s.id)}></Expand></div></td></tr>);
    return out;
  };
  const body = [];
  let n = 0;
  if (group) GROUPS.forEach(([k, l, c], gi) => {
    const all = rows.filter(s => groupOf(s) === gi);
    if (!all.length) return;
    const onPage = pageRows.filter(s => groupOf(s) === gi), isShut = shut.includes(k);
    if (!onPage.length && !(isShut && pg === 0)) return;
    body.push(<tr key={"g" + k} className={"gh" + (isShut ? " shut" : "")} onClick={() => setShut(x => x.includes(k) ? x.filter(y => y !== k) : [...x, k])}><td colSpan={span}><div className="gh-in"><span className="chev" style={{ display: "grid" }}><Ic n="chevD" s={13}></Ic></span><i style={{ background: c }}></i>{l}<em>{all.length}</em><span className="sp"></span><span className="agg">{$(all.reduce((a, s) => a + s.rev, 0))}</span></div></td></tr>);
    onPage.forEach(s => body.push(...row(s, n++)));
  });
  else pageRows.forEach(s => body.push(...row(s, n++)));
  const tog = m => setMenu(x => x === m ? null : m);
  return (
    <section className="bd">
      <div className="bd-h">
        <FilterField filters={filters} setFilters={setFilters} list={list} inputRef={filterRef}></FilterField>
        <div style={{ position: "relative" }}>
          <button className={"btn fbtn" + (nF ? " act" : "")} onClick={() => tog("f")}><Ic n="filter" s={13}></Ic>Filter{nF ? <em>{nF}</em> : null}</button>
          {menu === "f" && <FacetMenu list={list} facets={facets} setFacets={setFacets} onClose={() => setMenu(null)}></FacetMenu>}
        </div>
        <div style={{ position: "relative" }}>
          <button className={"btn sbtn" + (sort ? " act" : "")} onClick={() => tog("s")}><Ic n="upDown" s={13}></Ic>{sort ? COLS.find(c => c.id === sort.k).label : "Sort"}{sort && <Ic n={sort.d === "asc" ? "up" : "down"} s={11} w={2}></Ic>}</button>
          {menu === "s" && <SortMenu sort={sort} setSort={setSort} onClose={() => setMenu(null)}></SortMenu>}
        </div>
        <span className="sp"></span>
        <div className="vt">
          <button className={view === "table" ? "on" : ""} onClick={() => setView("table")} title="Table"><Ic n="table" s={13}></Ic><span className="vl">Table</span></button>
          <button className={view === "timeline" ? "on" : ""} onClick={() => setView("timeline")} title="Timeline"><Ic n="timeline" s={13}></Ic><span className="vl">Timeline</span></button>
          <button className={view === "map" ? "on" : ""} onClick={() => setView("map")} title="Map"><Ic n="map" s={13}></Ic><span className="vl">Map</span></button>
        </div>
        {view === "table" && <button className={"btn" + (group ? " act" : "")} onClick={() => setGroup(!group)} title="Group by stage"><Ic n="layers" s={13}></Ic><span className="vl">Group</span></button>}
        <div style={{ position: "relative" }} className="dsp">
          <button className={"btn cbtn" + (hidden.join() !== DEFAULT_HIDDEN.join() ? " act" : "")} onClick={() => tog("c")}><Ic n="columns" s={13}></Ic><span className="vl">Columns</span></button>
          {menu === "c" && <ColMenu hidden={hidden} setHidden={setHidden} onClose={() => setMenu(null)}></ColMenu>}
        </div>
        <button className="btn icon" title="Export"><Ic n="download" s={13}></Ic></button>
        <button className="btn ghostb"><Ic n="bookmark" s={13}></Ic><span className="vl">Views</span></button>
        <button className={"btn icon" + (railOpen ? " act" : "")} onClick={() => setRailOpen(!railOpen)} title="Toggle side panel"><Ic n="panel" s={13}></Ic></button>
      </div>
      <div className="bd-s" ref={bds}>
        {view === "map" ? (
          <div className="mp">
            <div className="mp-l"><span><i style={{ background: "var(--brand)" }}></i>Moving {rows.filter(s => s.st === "transit").length}</span><span><i style={{ background: "var(--danger)" }}></i>Late {rows.filter(s => s.late).length}</span><span><i style={{ background: "var(--warn)" }}></i>Uncovered {rows.filter(s => !s.drv).length}</span></div>
            <div className="mp-c"><span className="mono">live map</span><b>Google Maps isn't connected for this workspace</b><span>Connect it to see trucks, lanes and geofences here. Table and Timeline work without it.</span><button className="btn sm"><Ic n="plug" s={12}></Ic>Connect Google Maps</button></div>
          </div>
        ) : !rows.length ? (
          <div className="empty"><b>No shipments match</b><span>Try removing a filter.</span><button className="btn sm" onClick={() => { onClearFilters(); setFacets({}); }}>Clear filters</button></div>
        ) : view === "timeline" ? <Timeline rows={rows} openId={openId} onOpen={onOpen}></Timeline> : (
          <div className="tfr"><table className="tbl">
            <thead><tr>
              <th className="cc"><span className={"cbx" + (allChk ? " on" : "")} onClick={() => onCheckAll(pageRows.map(r => r.id), allChk)}>{allChk && <Ic n="check" s={10} w={3}></Ic>}</span></th>
              {cols.map(c => <th key={c.id} className={(c.r ? "r " : "") + "so" + (sort && sort.k === c.id ? " on" : "")} onClick={() => cycle(c.id)}><span className="th-in">{c.label}<Ic n={sort && sort.k === c.id ? (sort.d === "asc" ? "up" : "down") : "upDown"} s={11} w={2}></Ic></span></th>)}
              <th style={{ width: 64 }}></th>
            </tr></thead>
            <tbody>{body}</tbody>
          </table></div>
        )}
      </div>
      {view === "table" && rows.length > 0 && (
        <div className="bd-f">
          <button className="ib" title={"Keyboard shortcuts (" + MOD + " /)"} onClick={() => window.dispatchEvent(new Event("tv:keys"))}><Ic n="keyboard" s={14}></Ic></button><span>Showing <b>{flat.length ? pg * per + 1 : 0}</b> to <b>{Math.min(flat.length, pg * per + per)}</b> of <b>{flat.length}</b> results</span>
          <span className="sp"></span>
          <label className="per">Rows per page<select value={per} onChange={e => setPer(+e.target.value)}>{[10, 25, 50].map(v => <option key={v} value={v}>{v}</option>)}</select></label>
          <div className="pgn">
            <button className="btn icon" disabled={pg === 0} onClick={() => setPage(pg - 1)}><Ic n="chevL" s={13}></Ic></button>
            <span>Page <b>{pg + 1}</b> of <b>{pages}</b></span>
            <button className="btn icon" disabled={pg >= pages - 1} onClick={() => setPage(pg + 1)}><Ic n="chevR" s={13}></Ic></button>
          </div>
        </div>
      )}
    </section>
  );
}
Object.assign(window, { Board });
