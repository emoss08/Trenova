const AX_ICONS = {
  "a-table": <><rect x="3.5" y="5" width="17" height="14" rx="2"></rect><path d="M3.5 10h17M9 10v9"></path></>,
  "a-record": <><rect x="3.5" y="5" width="17" height="14" rx="2"></rect><circle cx="9" cy="11" r="2"></circle><path d="M6 16c.6-1.4 1.7-2 3-2s2.4.6 3 2M14.5 10h3M14.5 13.5h3"></path></>,
  "a-rate": <><path d="M6 3.5h12v17l-3-1.8-3 1.8-3-1.8-3 1.8z"></path><path d="M9.5 9.5h5M9.5 13h5"></path></>,
  "a-mail": <><rect x="3.5" y="5.5" width="17" height="13" rx="2"></rect><path d="M4 7l8 6 8-6"></path></>,
  "a-plan": <><path d="M9 6.5h11M9 12h11M9 17.5h11"></path><path d="M3.8 6.5l1.2 1.2 2-2.2M3.8 12l1.2 1.2 2-2.2"></path><circle cx="5.2" cy="17.5" r="1.3"></circle></>,
  "a-report": <><path d="M4 20V4M4 20h16"></path><path d="M8 16v-4M12 16V8M16 16v-6"></path></>,
  "a-diff": <><path d="M7 4v10M3.5 9.5L7 14l3.5-4.5"></path><path d="M17 20V10M13.5 14.5L17 10l3.5 4.5"></path></>,
  "a-doc": <><path d="M6 3.5h8l4 4v13H6z"></path><path d="M14 3.5v4h4M9 12h6M9 15.5h6"></path></>,
  "a-view": <><path d="M4 5h16l-6 7.5V19l-4-2v-4.5z"></path></>,
  "a-decision": <><path d="M12 3.5l7 2.5v5.5c0 4.4-3 7.7-7 9-4-1.3-7-4.6-7-9V6z"></path><path d="M9 12l2 2 4-4"></path></>,
  "a-pin": <path d="M9 4h6l-1 5 3 3H7l3-3zM12 12v8"></path>,
  "a-dl": <path d="M12 4v11M7.5 10.5L12 15l4.5-4.5M5 19.5h14"></path>,
  "a-ext": <path d="M14 4.5h5.5V10M19.5 4.5L11 13M17 14v4.5a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V8a1 1 0 0 1 1-1h4.5"></path>,
  "a-x": <path d="M6.5 6.5l11 11M17.5 6.5l-11 11"></path>,
  "a-up": <path d="M6 15l6-6 6 6"></path>,
  "a-down": <path d="M6 9l6 6 6-6"></path>,
  "a-search": <><circle cx="11" cy="11" r="6"></circle><path d="M20 20l-4.5-4.5"></path></>,
  "a-check": <path d="M5 12.5l4.5 4.5L19 7"></path>,
  "a-truck": <><path d="M3.5 6.5h10v9h-10zM13.5 9.5h4l3 3v3h-7"></path><circle cx="7" cy="17" r="1.6"></circle><circle cx="17" cy="17" r="1.6"></circle></>,
  "a-warn": <><path d="M12 4l9 15.5H3z"></path><path d="M12 10v4M12 17v.01"></path></>,
  "a-scan": <><path d="M4 8V5.5A1.5 1.5 0 0 1 5.5 4H8M16 4h2.5A1.5 1.5 0 0 1 20 5.5V8M20 16v2.5a1.5 1.5 0 0 1-1.5 1.5H16M8 20H5.5A1.5 1.5 0 0 1 4 18.5V16"></path><path d="M8 10h8M8 14h5"></path></>,
  "a-bill": <><path d="M6 3.5h12v17l-2-1.3-2 1.3-2-1.3-2 1.3-2-1.3-2 1.3z"></path><path d="M9.5 8.5h5M9.5 12h5M12 14.5v2"></path></>,
  "a-copy": <><rect x="8" y="8" width="11" height="11" rx="2"></rect><path d="M5 15V6a1 1 0 0 1 1-1h9"></path></>,
};
function AI({ n, s = 14, w = 1.8 }) {
  return <svg width={s} height={s} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={w} strokeLinecap="round" strokeLinejoin="round">{AX_ICONS[n]}</svg>;
}
const $ = n => (n < 0 ? "−$" : "$") + Math.abs(n).toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
const $0 = n => "$" + Math.round(n).toLocaleString("en-US");
const STL = { ReadyForReview: "Ready for review", Approved: "Approved", Posted: "Posted" };
const PILL = { "On hold": "warn", "Ready for review": "blue", Approved: "green", Posted: "ink", "In transit": "blue", Late: "warn", Delivered: "green", "Needs driver": "warn" };

/* ---------- stack switcher ---------- */
function ArtStack({ ids, active, newest, onPick, onFan, fanH, total, onAll, flags }) {
  const [fan, setFan] = React.useState(false);
  const t = React.useRef(null);
  const order = [active, ...ids.filter(x => x !== active)];
  const n = order.length;
  const gap = Math.max(34, Math.min(54, (fanH - 40) / (n + 1)));
  const root = React.useRef(null);
  const open = () => { setFan(true); onFan(true); };
  const close = () => { setFan(false); onFan(false); };
  React.useEffect(() => { if (!fan) return; const off = e => root.current && !root.current.contains(e.target) && close(); const esc = e => e.key === "Escape" && close(); document.addEventListener("mousedown", off); document.addEventListener("keydown", esc); return () => { document.removeEventListener("mousedown", off); document.removeEventListener("keydown", esc); }; }, [fan]);
  return (
    <div ref={root} className={"ax-stack" + (fan ? " fan" : "")} style={{ "--fanh": n * gap + 52 + "px" }}>
      {order.map((id, k) => {
        const a = artMeta(id);
        const ty = fan ? k * gap : Math.min(k, 2) * 6;
        const sc = fan ? 1 : 1 - Math.min(k, 2) * 0.035;
        const rot = 0;
        return (
          <button key={id} className={"ax-card" + (k === 0 ? " front" : "") + (id === newest ? " nw" : "")} onClick={() => { if (k === 0 && !fan) { open(); return; } onPick(id); close(); }} title={k === 0 && !fan ? "Switch artifact" : undefined}
            style={{ zIndex: n - k, transform: `translateY(${ty}px) scale(${sc}) rotate(${rot}deg)`, opacity: !fan && k > 2 ? 0 : 1, transitionDelay: (fan ? k * 16 : 0) + "ms" }}>
            <span className={"ax-ki k-" + a.kind}><AI n={KIND[a.kind].icon} s={15} /></span>
            <span className="ax-ct"><b>{a.title}</b><span>{KIND[a.kind].label} · {a.at}</span></span>
            {id === newest && <span className="ax-new">New</span>}
            {a.versions && <span className="ax-ver">v{a.versions.filter(v => !v.needs || flags[v.needs]).length}</span>}
            {k === 0 && !fan && <span className="ax-cnt">{total}<AI n="a-down" s={11} w={2.2} /></span>}
          </button>
        );
      })}
      <button className="ax-card ax-all" onClick={() => { onAll(); close(); }} style={{ zIndex: 0, transform: `translateY(${fan ? n * gap : 12}px) scale(${fan ? 1 : 0.93})`, opacity: fan ? 1 : 0, transitionDelay: (fan ? n * 16 : 0) + "ms" }}>
        <span className="ax-ki"><AI n="a-table" s={15} /></span>
        <span className="ax-ct"><b>All {total} artifacts</b><span>Search everything this conversation made</span></span>
        <span className="kbd">⌘J</span>
      </button>
    </div>
  );
}

/* ---------- provenance ---------- */
function Prov({ a, ctx }) {
  const n = a.kind === "table" ? (a.total || a.rows(ctx).length) + " rows" : null;
  return (
    <div className="ax-prov sm">
      <code>{a.src}</code>{n && <span>{n}</span>}{a.calls ? <span>{a.calls + " calls"}</span> : null}
      {a.kind === "table" && a.id !== "board" && ctx.assigned && <span className="ax-chg">{ctx.posted ? "Posted" : "11 changed"}</span>}
    </div>
  );
}

/* ---------- bodies ---------- */
function VersionStrip({ a, ctx, ver, setVer }) {
  const [open, setOpen] = React.useState(false);
  const root = React.useRef(null);
  React.useEffect(() => { if (!open) return; const off = e => root.current && !root.current.contains(e.target) && setOpen(false); document.addEventListener("mousedown", off); return () => document.removeEventListener("mousedown", off); }, [open]);
  const vs = a.versions.filter(v => !v.needs || ctx[v.needs]);
  const cur = ver == null ? vs.length - 1 : ver;
  return (
    <span className="axv" ref={root}>
      <button className={"axv-b" + (open ? " on" : "") + (cur !== vs.length - 1 ? " old" : "")} onClick={() => setOpen(o => !o)} title="Versions">
        {"v" + vs[cur].v}<svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round"><path d="M6 9l6 6 6-6"></path></svg>
      </button>
      {open && <div className="axv-pop">{[...vs].reverse().map(v => { const i = vs.indexOf(v); return (
        <button key={v.v} className={"axv-r" + (i === cur ? " on" : "")} onClick={() => { setVer(i === vs.length - 1 ? null : i); setOpen(false); }}>
          <b>{"v" + v.v}</b><span><em>{v.note}</em><i>{v.at + (i === vs.length - 1 ? " · Latest" : "")}</i></span>
          {i === cur && <Ic n="check" s={12} w={2.4} />}
        </button>); })}</div>}
    </span>
  );
}

function TableBody({ a, ctx: liveCtx }) {
  const [q, setQ] = React.useState("");
  const [sort, setSort] = React.useState(null);
  const [ver, setVer] = React.useState(null);
  const [sel, setSel] = React.useState([]);
  const [undo, setUndo] = React.useState(null);
  (window.useBQ || (() => {}))();
  let ctx = liveCtx;
  if (a.versions && ver != null) { const v = a.versions[ver]; ctx = { ...liveCtx, assigned: v.v >= 3, posted: v.v >= 4, flash: "v" + ver }; }
  let rows = a.rows(ctx);
  const bq = window.bqGet && rows.length > 0 && String(rows[0].id).startsWith("BQ-") && ver == null;
  if (bq) { BQ.ctx = { assigned: ctx.assigned, posted: ctx.posted }; rows = rows.map(r => { const g = bqGet(r.id); return { ...r, biller: g.biller, status: g.hold ? "On hold" : BI_STEPS[g.st], amt: g.total, _g: g }; }); }
  if (q) rows = rows.filter(r => a.cols.some(c => String(r[c.k] ?? c.none ?? "").toLowerCase().includes(q.toLowerCase())));
  if (sort) rows = [...rows].sort((x, y) => { const u = x[sort.k], v = y[sort.k]; if (u == null) return 1; if (v == null) return -1; return (typeof u === "number" ? u - v : String(u).localeCompare(String(v))) * sort.d; });
  const money = a.cols.find(c => c.t === "money");
  const total = money ? rows.reduce((s, r) => s + r[money.k], 0) : 0;
  const selRows = rows.filter(r => sel.includes(r.id));
  const ready = selRows.filter(r => r._g && r._g.ready), needs = selRows.filter(r => r._g && r._g.st === 0 && !r._g.ready);
  const allOn = bq && rows.length > 0 && rows.every(r => sel.includes(r.id));
  const tog = id => setSel(xs => xs.includes(id) ? xs.filter(x => x !== id) : [...xs, id]);
  React.useEffect(() => { if (!undo) return; if (undo.n <= 0) { setUndo(null); return; } const h = setTimeout(() => setUndo(u => u && { ...u, n: u.n - 1 }), 1000); return () => clearTimeout(h); }, [undo]);
  const approve = () => { const ids = ready.map(r => r.id); bqApproveMany(ids); setSel([]); setUndo({ ids, n: 6 }); };
  const openItem = id => { bqOpen(id); ctx.open("bqi"); };
  return (
    <div className="ax-tbl">
      <div className="ax-tools">
        <label className="ax-q"><AI n="a-search" s={13} /><input value={q} onChange={e => setQ(e.target.value)} placeholder={"Filter " + rows.length + " rows"} /></label>
        {a.versions && <VersionStrip a={a} ctx={liveCtx} ver={ver} setVer={setVer} />}
      </div>
      <div className="ax-scroll">
        <table className={bq ? "sel" : ""}>
          <thead><tr>{bq && <th className="cb"><button className={"ax-cb" + (allOn ? " on" : sel.length ? " mid" : "")} onClick={() => setSel(allOn ? [] : rows.map(r => r.id))} title="Select all"><AI n="a-check" s={10} w={3} /></button></th>}{a.cols.map(c => (
            <th key={c.k} className={c.t === "money" ? "ar" : ""} onClick={() => setSort(s => s && s.k === c.k ? (s.d === 1 ? { k: c.k, d: -1 } : null) : { k: c.k, d: 1 })}>
              <span>{c.l}{sort && sort.k === c.k && <AI n={sort.d === 1 ? "a-up" : "a-down"} s={10} w={2.4} />}</span>
            </th>))}<th className="go"></th></tr></thead>
          <tbody>
            {rows.map((r, i) => (
              <tr key={r.id + ":" + ctx.flash} className={sel.includes(r.id) ? "on" : ""} style={{ animationDelay: Math.min(i, 14) * 18 + "ms", cursor: bq ? "pointer" : undefined }} onClick={bq ? () => openItem(r.id) : undefined}>
                {bq && <td className="cb" onClick={e => { e.stopPropagation(); tog(r.id); }}><span className={"ax-cb" + (sel.includes(r.id) ? " on" : "")}><AI n="a-check" s={10} w={3} /></span></td>}
                {a.cols.map(c => {
                  const v = r[c.k]; const chg = r._chg && r._chg[c.k];
                  let cell;
                  if (c.t === "id") cell = <span className="ax-id">{v}</span>;
                  else if (c.t === "money") cell = <span className="ax-num">{$(v)}</span>;
                  else if (c.t === "mono") cell = <span className="ax-mono">{v}</span>;
                  else if (c.t === "warnText") cell = <span className="ax-none">{v}</span>;
                  else if (c.t === "opt") cell = v ? v : <span className="ax-none">{c.none}</span>;
                  else if (c.t === "pill") { const lv = STL[v] || v; cell = <span className={"ax-pill " + (PILL[lv] || "")}><i></i>{lv}{r._g && r._g.st === 0 && r._g.needs > 0 && !r._g.hold && <em className="ax-nd">{r._g.needs}</em>}</span>; }
                  else cell = v;
                  return <td key={c.k} className={(c.t === "money" ? "ar" : "") + (chg ? " chg" : "")}>{cell}</td>;
                })}
                <td className="go"><AI n="a-ext" s={12} /></td>
              </tr>
            ))}
          </tbody>
          {money && <tfoot><tr>{bq && <td></td>}<td colSpan={a.cols.length - 2}>{rows.length} items</td><td className="ar"><span className="ax-num">{$(total)}</span></td><td colSpan={2}></td></tr></tfoot>}
        </table>
        {a.total && a.total > rows.length && !q && <div className="ax-trunc">Showing {rows.length} of {a.total}. Ask for the rest, or open it as a view.</div>}
      </div>
      {bq && (sel.length > 0 || undo) && (
        <div className="ax-bulk" key={undo ? "u" : "s"}>
          {undo ? <>
            <span className="ax-bk-ok"><AI n="a-check" s={12} w={2.6} /></span>
            <span className="ax-bk-t"><b>Approved {undo.ids.length} {undo.ids.length === 1 ? "invoice" : "invoices"}</b></span>
            <button className="ax-btn" onClick={() => { bqUnapprove(undo.ids); setUndo(null); }}><AI n="a-up" s={0} />Undo <em className="ax-bk-n">{undo.n}</em></button>
          </> : <>
            <span className="ax-bk-t"><b>{sel.length} selected</b><span>{ready.length} ready{needs.length ? " · " + needs.length + " need you" : ""}{selRows.length - ready.length - needs.length > 0 ? " · " + (selRows.length - ready.length - needs.length) + " already done or held" : ""}</span></span>
            <button className="ax-btn ghost" onClick={() => setSel([])}>Clear</button>
            {needs.length > 0 && <button className="ax-btn" onClick={() => openItem(needs[0].id)}>Review {needs.length}</button>}
            <button className="ax-btn ink" disabled={!ready.length} onClick={approve}>Approve {ready.length || ""}</button>
          </>}
        </div>
      )}
    </div>
  );
}

function RecordBody({ a }) {
  const r = a.rec;
  return (
    <div className="ax-pad ax-rec">
      <div className="ax-rec-h">
        <div><div className="ax-big">{a.title}</div><div className="ax-sub">{r.customer}</div></div>
        <span className={"ax-pill lg " + PILL[r.status]}><i></i>{r.status}</span>
      </div>
      <div className="ax-route">
        <div className="ax-stop"><i></i><b>{r.from.city}</b><span>{r.from.place}</span><em>{r.from.time}</em></div>
        <div className="ax-line"><span className="ax-done" style={{ width: r.progress * 100 + "%" }}></span><span className="ax-trk" style={{ left: r.progress * 100 + "%" }}><AI n="a-truck" s={13} /></span></div>
        <div className="ax-stop end"><i></i><b>{r.to.city}</b><span>{r.to.place}</span><em>{r.to.time}</em></div>
      </div>
      <dl className="ax-fields">{r.fields.map(([k, v]) => <div key={k}><dt>{k}</dt><dd>{v}</dd></div>)}</dl>
      <button className="ax-btn"><AI n="a-ext" s={13} />Open shipment</button>
    </div>
  );
}

function RateBody({ a }) {
  const [rej, setRej] = React.useState(false);
  const lines = [...a.components, a.discount];
  let run = 0;
  const total = lines.reduce((s, l) => s + l[2], 0);
  return (
    <div className="ax-pad ax-rate">
      <div className="ax-win"><span className="ax-wck"><AI n="a-check" s={12} w={2.6} /></span><div><b>{a.winner.name}</b><span>{a.winner.rule} · <code>{a.winner.code}</code></span></div></div>
      <p className="ax-note">{a.tie}</p>
      <div className="ax-ledger">
        {lines.map(([l, b, amt], i) => { run += amt; return (
          <div key={l} className={"ax-lr" + (amt < 0 ? " neg" : "")} style={{ animationDelay: i * 60 + "ms" }}>
            <span className="ax-ll"><b>{l}</b><span>{b}</span></span>
            <span className="ax-bar"><i style={{ width: (run / total) * 100 + "%" }}></i></span>
            <span className="ax-num">{$(amt)}</span>
          </div>); })}
        <div className="ax-guard"><AI n="a-check" s={12} w={2.4} />{a.guard.kind} {$0(a.guard.bound)} · {a.guard.result}</div>
        <div className="ax-total"><span>Total</span><b>{$(total)}</b></div>
      </div>
      {a.warnings.map(w => <div key={w} className="ax-warn"><AI n="a-warn" s={13} />{w}</div>)}
      <button className="ax-more" onClick={() => setRej(x => !x)}><AI n={rej ? "a-up" : "a-down"} s={11} w={2.2} />{a.rejected.length} agreements didn't apply</button>
      {rej && <div className="ax-rej">{a.rejected.map(([c, r, why]) => <div key={c}><code>{c}</code><span>{r}</span><em>{why}</em></div>)}</div>}
    </div>
  );
}

function EmailBody({ a }) {
  const [body, setBody] = React.useState(a.body);
  const [subj, setSubj] = React.useState(a.subject);
  const [sent, setSent] = React.useState(false);
  return (
    <div className="ax-mail">
      <div className="ax-mrow"><span>To</span><div className="ax-to">{a.to.map(t => <span key={t}>{t}</span>)}</div></div>
      <div className="ax-mrow"><span>Subject</span><input value={subj} onChange={e => setSubj(e.target.value)} /></div>
      <textarea value={body} onChange={e => setBody(e.target.value)} spellCheck="false"></textarea>
      <div className="ax-why"><b>Why this wording</b>{a.why}</div>
      <div className="ax-acts">
        <button className="ax-btn ghost"><AI n="a-copy" s={13} />Copy</button>
        <span style={{ flex: 1 }}></span>
        {sent ? <span className="ax-sent"><AI n="a-check" s={13} w={2.4} />Sent for approval</span> : <button className="ax-btn ink" onClick={() => setSent(true)}>Send for approval</button>}
      </div>
    </div>
  );
}

function PlanBody({ a }) {
  const done = a.steps.filter(s => s.st === "done").length;
  return (
    <div className="ax-pad ax-plan">
      <p className="ax-lead">{a.summary}</p>
      <div className="ax-prog"><span style={{ width: (done / a.steps.length) * 100 + "%" }}></span></div>
      <div className="ax-prog-l">{done} of {a.steps.length} steps done</div>
      <ol>{a.steps.map((s, i) => (
        <li key={i} className={"st-" + s.st} style={{ animationDelay: i * 70 + "ms" }}>
          <span className="ax-sn">{s.st === "done" ? <AI n="a-check" s={11} w={2.8} /> : i + 1}</span>
          <div><b>{s.text}</b><span><code>{s.tool}</code>{s.st === "wait" ? " · waiting on your approval" : s.st === "next" ? " · up next" : ""}</span></div>
        </li>))}</ol>
    </div>
  );
}

function ReportBody({ a }) {
  const max = Math.max(...a.rows.map(r => r[1]));
  const th = a.rows.reduce((s, r) => s + r[1], 0), tc = a.rows.reduce((s, r) => s + r[2], 0);
  return (
    <div className="ax-pad ax-rep">
      <div className="ax-repm"><span>{a.dataset}</span><span>{a.rowCount.toLocaleString()} stops · top {a.rows.length} customers</span></div>
      <div className="ax-bars">{a.rows.map(([c, h, $c], i) => (
        <div key={c} className="ax-br" style={{ animationDelay: i * 60 + "ms" }}>
          <span className="ax-bn">{c}</span>
          <span className="ax-bt"><i style={{ width: (h / max) * 100 + "%", animationDelay: 120 + i * 60 + "ms" }}></i></span>
          <span className="ax-num">{h.toFixed(1)} h</span>
          <span className="ax-num mut">{$0($c)}</span>
        </div>))}
        <div className="ax-br tot"><span className="ax-bn">Total</span><span></span><span className="ax-num">{th.toFixed(1)} h</span><span className="ax-num">{$0(tc)}</span></div>
      </div>
      <div className="ax-acts"><button className="ax-btn ghost"><AI n="a-dl" s={13} />Download CSV</button><button className="ax-btn ghost"><AI n="a-ext" s={13} />Open full report</button></div>
    </div>
  );
}

function DiffBody({ a }) {
  const sign = { added: "+", removed: "−", changed: "~" };
  return (
    <div className="ax-pad ax-diff">
      <div className="ax-dh"><span>{a.before}</span><i>→</i><span>{a.after}</span></div>
      <div className="ax-dc">
        <span className="add">{`+${a.counts.added} added`}</span><span className="rem">{`−${a.counts.removed} removed`}</span><span className="chg">{`~${a.counts.changed} changed`}</span><span>{`${a.counts.unchanged} unchanged`}</span>
      </div>
      <div className="ax-dt">{a.totals.map(([l, b, n]) => { const d = n - b; return (
        <div key={l}><span>{l}</span><b className="ax-num">{$0(n)}</b><em className={d > 0 ? "up" : d < 0 ? "dn" : ""}>{d === 0 ? "no change" : (d > 0 ? "+" : "−") + $0(Math.abs(d)).slice(1)}</em></div>); })}</div>
      <div className="ax-dl">{a.changes.map(([k, cust, bucket, b, n], i) => (
        <div key={cust + bucket} className={"ax-dr " + k} style={{ animationDelay: i * 50 + "ms" }}>
          <span className="ax-dk">{sign[k]}</span><span className="ax-dn"><b>{cust}</b><span>{bucket}</span></span>
          <span className="ax-num mut">{b ? $0(b) : "—"}</span><i>→</i><span className="ax-num">{n ? $0(n) : "—"}</span>
        </div>))}</div>
    </div>
  );
}

function ViewBody({ a }) {
  return (
    <div className="ax-pad ax-view">
      <p className="ax-vx">{a.explanation.map((s, i) => typeof s === "string" ? s : <span key={i} className="ax-term">{s.t}</span>)}</p>
      <div className="ax-vprev">
        <div className="ax-vbar"><span>{a.entity}</span><em>{a.filters} filters</em><b>{a.count} results</b></div>
        {BOARD.filter(r => r.st === "Late").map(r => <div key={r.id} className="ax-vrow"><span className="ax-id">{r.id}</span><span>{r.lane}</span><span className="ax-pill warn"><i></i>Late</span></div>)}
      </div>
      {a.unresolved.map(([p, why]) => <div key={p} className="ax-warn"><AI n="a-warn" s={13} /><span>Left out <b>“{p}”</b>. {why}.</span></div>)}
      <button className="ax-btn ink wide"><AI n="a-ext" s={13} />Open in {a.entity}</button>
    </div>
  );
}

function DecisionBody({ a, ctx }) {
  const d = DECISIONS[a.d];
  const st = ctx.pending.includes(a.d) ? "pending" : (ctx.resolved.find(r => r[0] === a.d) || [])[1] || "pending";
  const rows = a.d === "d1" ? QUEUE.filter(r => !r.biller) : QUEUE;
  return (
    <div className={"ax-pad ax-dec s-" + st}>
      <div className="ax-dech"><b>{d.title}</b><span className={"ax-dst " + st}>{st === "pending" ? "Waiting on you" : st === "approved" ? "Approved" : "Set aside"}</span></div>
      <p className="ax-note">{d.scope} · {d.meta.join(" · ")}</p>
      <div className="ax-decl">{rows.slice(0, 8).map(r => (
        <div key={r.id}><span className="ax-id">{r.id}</span><span>{r.cust}</span><s>{a.d === "d1" ? "No biller" : STL[r.status] || r.status}</s><i>→</i><em>{d.to}</em></div>))}
        {rows.length > 8 && <div className="ax-decm">+ {rows.length - 8} more</div>}
      </div>
      {st === "pending" && <div className="ax-acts"><button className="ax-btn ghost" onClick={() => ctx.onDismiss(a.d)}>Not now</button><span style={{ flex: 1 }}></span><button className="ax-btn ink" onClick={() => ctx.onApprove(a.d)}>Approve {rows.length} changes</button></div>}
    </div>
  );
}

const BODIES = { table: TableBody, record: RecordBody, rate: RateBody, email: EmailBody, plan: PlanBody, report: ReportBody, diff: DiffBody, view: ViewBody, decision: DecisionBody };

/* ---------- pane ---------- */
function ArtifactPane({ s, set, arts, history = ART_HISTORY, newest, pending, resolved, assigned, posted, flash, onApprove, onDismiss, onCopy }) {
  (window.useBQ || (() => {}))();
  const convIds = [...arts].reverse();
  const allIds = React.useMemo(() => [...convIds, ...history], [arts, history]);
  const active = allIds.includes(s.art) && ART2[s.art] ? s.art : allIds[0];
  const a = { ...(ART2[active] || {}), id: active };
  const [fan, setFan] = React.useState(false);
  const [pinned, setPinned] = React.useState({ queue: true });
  const [recent, setRecent] = React.useState([]);
  React.useEffect(() => { setRecent(r => [active, ...r.filter(x => x !== active)].slice(0, 6)); }, [active]);
  const pinIds = Object.keys(pinned).filter(k => pinned[k]);
  const ids = [...new Set([active, ...convIds.slice(0, 3), ...pinIds, ...recent])].filter(id => ART2[id]).slice(0, 8);
  const ref = React.useRef(null);
  const [h, setH] = React.useState(700);
  React.useLayoutEffect(() => { const el = ref.current; if (!el) return; const ro = new ResizeObserver(() => setH(el.clientHeight)); ro.observe(el); return () => ro.disconnect(); }, []);
  const i = allIds.indexOf(active);
  const go = d => set({ art: allIds[(i + d + allIds.length) % allIds.length] });
  const ctx = { assigned, posted, flash, pending, resolved, onApprove, onDismiss, open: id => set({ art: id, browse: false }) };
  const Body = BODIES[a.kind] || window["Body_" + a.kind];
  if (!allIds.length) return (
    <aside className="sheet">
      <div className="apx axe">
        <div className="axe-top"><button className="ax-ib" title="Close" onClick={() => set({ open: false })}><AI n="a-x" s={13} w={2.2} /></button></div>
        <div className="axe-c">
          <div className="axe-stack" aria-hidden="true">{["report", "record", "table"].map(k => <span key={k}><span className={"ax-ki k-" + k}><AI n={(KIND[k] || KIND.table).icon} s={14} /></span><i></i><i></i></span>)}</div>
          <b>No artifacts yet</b>
          <p>When the agent pulls a table, opens a record or drafts an email, it lands here so you can check it, pin it or export it.</p>
          <span className="axe-k"><span className="kbd">⌘J</span>opens this panel</span>
        </div>
      </div>
    </aside>
  );
  return (
    <aside className="sheet">
      <div className={"apx" + (fan ? " fanning" : "") + (s.browse ? " browsing" : "")} ref={ref}>
        {s.browse ? <ArtBrowser ids={allIds} active={active} pinned={pinned} flags={{ assigned, posted }} onPick={id => set({ art: id, browse: false })} onBack={() => set({ browse: false })} onClose={() => set({ open: false, browse: false })} /> : <>
        <div className="ax-top">
          <ArtStack ids={ids} active={active} newest={newest} onPick={id => set({ art: id })} onFan={setFan} fanH={h} total={allIds.length} onAll={() => set({ browse: true })} flags={{ assigned, posted }} />
          <div className="ax-nav">
            <button className="ax-ib" title="Previous" onClick={() => go(-1)}><AI n="a-up" s={13} w={2.2} /></button>
            <button className="ax-ib" title="Next" onClick={() => go(1)}><AI n="a-down" s={13} w={2.2} /></button>
            <button className="ax-ib" title="Close" onClick={() => set({ open: false })}><AI n="a-x" s={13} w={2.2} /></button>
          </div>
        </div>
        <div className="ax-body" key={active}>
          {a.old && <div className="ax-oldnote">From {a.day.toLowerCase() === "yesterday" ? "yesterday" : a.day} · {a.turn.split(" · ")[1]}</div>}
          <Prov a={a} ctx={ctx} />
          <Body a={a} ctx={ctx} />
        </div>
        <div className="ax-foot">
          <button className="ax-link" onClick={onCopy} title="Copy link"><AI n="a-copy" s={12} /><span>desk/c/8f2k/a/<b>{a.slug}</b></span></button>
          <span style={{ flex: 1 }}></span>
          <button className={"ax-ib" + (pinned[active] ? " on" : "")} title={pinned[active] ? "Unpin" : "Pin to conversation"} onClick={() => setPinned(p => ({ ...p, [active]: !p[active] }))}><AI n="a-pin" s={14} /></button>
          {(a.kind === "table" || a.kind === "report") && <button className="ax-ib" title="Export CSV"><AI n="a-dl" s={14} /></button>}
          <button className="ax-ib" title="Open on its own page" onClick={onCopy}><AI n="a-ext" s={14} /></button>
        </div>
        </>}
      </div>
    </aside>
  );
}

function ArtKindIcon({ id, s = 11 }) { const a = artMeta(id); return <AI n={KIND[a.kind].icon} s={s} w={2} />; }

Object.assign(window, { ArtifactPane, ArtKindIcon });
