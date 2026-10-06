function Kpis({ quiet }) {
  const Dl = ({ v, d = "up", good }) => <span className={"dl " + (good ? "up" : d)}>{d !== "fl" && <Ic n={d === "up" ? "up" : "down"} s={9} w={2.2}></Ic>}{v}</span>;
  if (quiet) return (
    <div className="kp">
      <div className="k"><div className="k-l">Revenue today</div><div className="k-v">$0</div><div className="k-s"><span>No deliveries yet</span></div></div>
      <div className="k"><div className="k-l">Active</div><div className="k-v">0<small>/ 2</small></div><div className="k-s"><span>2 need a driver</span></div></div>
      <div className="k"><div className="k-l">On-time</div><div className="k-v">—</div><div className="k-s"><span>After first delivery</span></div></div>
      <div className="k"><div className="k-l">Tender accept<Dl v="0.4pp"></Dl></div><div className="k-v">94.1<small>%</small></div><div className="k-s"><div className="tgt"><i className="w" style={{ width: "94.1%" }}></i><b style={{ left: "95%" }}></b></div></div></div>
    </div>
  );
  return (
    <div className="kp">
      <div className="k"><div className="k-l">Revenue today<Dl v="8.2%"></Dl></div><div className="k-v">$24,890</div><div className="k-s"><span>RPM $2.71</span><Spark d={[8, 9, 7, 11, 10, 13, 12, 15, 16]}></Spark></div></div>
      <div className="k"><div className="k-l">Active<Dl v="3"></Dl></div><div className="k-v">8<small>/ 14</small></div><div className="k-s"><div className="mix"><i style={{ flex: 5, background: "var(--brand)" }}></i><i style={{ flex: 3, background: "var(--danger)" }}></i><i style={{ flex: 2, background: "var(--teal)" }}></i><i style={{ flex: 2, background: "var(--warn)" }}></i><i style={{ flex: 2, background: "var(--ok)" }}></i></div></div></div>
      <div className="k"><div className="k-l">On-time<Dl v="1.8pp" d="dn"></Dl></div><div className="k-v">86.4<small>%</small></div><div className="k-s"><div className="tgt"><i className="w" style={{ width: "86.4%" }}></i><b style={{ left: "92%" }}></b></div><span>target 92</span></div></div>
      <div className="k"><div className="k-l">Empty miles<Dl v="0.6pp" d="dn" good></Dl></div><div className="k-v">11.2<small>%</small></div><div className="k-s"><span>goal &lt;10%</span><Spark d={[13, 12.5, 12, 12.4, 11.8, 11.5, 11.2]} cls="ok"></Spark></div></div>
      <div className="k full"><div className="k-l">Tender acceptance<Dl v="0.4pp"></Dl></div><div className="k-s" style={{ gap: 12 }}><div className="k-v" style={{ fontSize: 15 }}>94.1<small>%</small></div><div className="tgt" style={{ flex: 1, width: "auto" }}><i style={{ width: "94.1%" }}></i><b style={{ left: "95%" }}></b></div><span style={{ flex: "none" }}>23 accepted · 1 declined</span></div></div>
    </div>
  );
}

function Hold({ label, done, onDone }) {
  const [p, setP] = React.useState(0);
  const t = React.useRef();
  const start = () => { const t0 = Date.now(); t.current = setInterval(() => { const v = Math.min(1, (Date.now() - t0) / 900); setP(v); if (v >= 1) { clearInterval(t.current); onDone(); } }, 16); };
  const stop = () => { clearInterval(t.current); if (!done) setP(0); };
  return (
    <button className={"hold" + (done ? " ok" : "")} onPointerDown={done ? null : start} onPointerUp={stop} onPointerLeave={stop} style={{ "--p": done ? 1 : p }}>
      <span className="hold-f"></span>
      <span className="hold-t">{done ? <><Ic n="check" s={13} w={2.4}></Ic>Transferred to billing</> : <>{label}<em>{p > 0 ? "keep holding" : "hold"}</em></>}</span>
    </button>
  );
}

function Watchlist({ list, onRef, onFilter, onAct, onOpenRow }) {
  const [sec, setSec] = React.useState(0);
  const [billed, setBilled] = React.useState(false);
  const [detBilled, setDetBilled] = React.useState([]);
  const [hov, setHov] = React.useState(null);
  React.useEffect(() => { const t = setInterval(() => setSec(x => x + 1), 1000); return () => clearInterval(t); }, []);
  const now = NOW + sec / 3600;
  const money = n => n >= 10000 ? "$" + (n / 1000).toFixed(n >= 100000 ? 0 : 1) + "k" : "$" + Math.round(n).toLocaleString();
  const n = v => v.toLocaleString();
  // deliveries histogram
  const A = 6, B = 24, hours = Array.from({ length: B - A }, (_, i) => A + i);
  const today = list.filter(s => s.today && s.st !== "new" && HOURS[s.id]);
  const bk = hours.map(h => { const xs = today.filter(s => Math.floor(HOURS[s.id][2] || HOURS[s.id][1]) === h); return { h, done: xs.filter(s => s.st === "done").length, late: xs.filter(s => s.late).length, ok: xs.filter(s => s.st !== "done" && !s.late).length, all: xs.length }; });
  const max = Math.max(1, ...bk.map(b => b.all));
  const late = today.filter(s => s.late).sort((x, y) => (HOURS[y.id][2] - HOURS[y.id][1]) - (HOURS[x.id][2] - HOURS[x.id][1]));
  const hb = hov != null ? bk.find(b => b.h === hov) : null;
  // uncovered
  const unc = list.filter(s => !s.drv && HOURS[s.id]).sort((x, y) => HOURS[x.id][0] - HOURS[y.id][0]);
  const W = [["< 2h", 0, 2], ["2–6h", 2, 6], ["Later today", 6, 24 - NOW], ["Tomorrow+", 24 - NOW, 999]];
  const wins = W.map(([l, a, b]) => { const xs = unc.filter(s => HOURS[s.id][0] - NOW >= a && HOURS[s.id][0] - NOW < b); return { l, a, b, c: xs.length, rev: xs.reduce((t, s) => t + s.rev, 0) }; });
  const first = unc[0];
  const left = first ? Math.max(0, Math.round((HOURS[first.id][0] - NOW) * 3600) - sec) : 0;
  const hh = Math.floor(left / 3600), mm = Math.floor(left % 3600 / 60), ss = left % 60;
  // detention
  window.__detBilled = detBilled;
  const dets = list.filter(s => s.det && s.drv && s.st !== "done" && !detBilled.includes(s.id)).map(s => ({ s, m: s.det + sec / 60 })).sort((x, y) => y.m - x.m);
  const detAmt = dets.reduce((t, d) => t + Math.max(0, d.m - 120) / 60 * 75, 0);
  // billing
  const bill = list.filter(s => s.st === "done" && (billOf(s) || [])[0] === "Ready to bill");
  const billAmt = bill.reduce((t, s) => t + s.rev, 0);
  const byCust = Object.entries(bill.reduce((m, s) => (m[s.cust] = m[s.cust] || [0, 0], m[s.cust][0]++, m[s.cust][1] += s.rev, m), {})).sort((x, y) => y[1][1] - x[1][1]);
  const cmax = byCust.length ? byCust[0][1][1] : 1;
  return (
    <div className="wv">
      {today.length > 0 && (
        <section className="wv-s">
          <div className="wv-h"><span>Today's deliveries</span><b className="mono">{n(today.length - late.length)}<em>{" / " + n(today.length) + " on time"}</em></b></div>
          <div className="hg" onMouseLeave={() => setHov(null)}>
            <div className="hg-bars">
              {bk.map(b => <button key={b.h} className={"hg-c" + (hov === b.h ? " on" : "") + (b.h < Math.floor(now) ? " past" : "")} onMouseEnter={() => setHov(b.h)} onClick={() => b.all && onFilter([{ k: "hour", v: b.h, label: String(b.h).padStart(2, "0") + ":00–" + String(b.h + 1).padStart(2, "0") + ":00" }])} title="">
                <span className="hg-st" style={{ height: (b.all / max) * 100 + "%" }}>
                  {b.late > 0 && <i className="hl" style={{ flex: b.late }}></i>}
                  {b.ok > 0 && <i className="ho" style={{ flex: b.ok }}></i>}
                  {b.done > 0 && <i className="hd" style={{ flex: b.done }}></i>}
                </span>
              </button>)}
              <span className="hg-now" style={{ left: ((now - A) / (B - A)) * 100 + "%" }}></span>
            </div>
            <div className="hg-ax">{[6, 9, 12, 15, 18, 21].map(h => <span key={h} style={{ left: ((h - A) / (B - A)) * 100 + "%" }}>{String(h).padStart(2, "0")}</span>)}</div>
          </div>
          <div className="rib-cap">{hb ? <><b>{String(hb.h).padStart(2, "0") + ":00–" + String(hb.h + 1).padStart(2, "0") + ":00"}</b>{" · " + n(hb.all) + " deliver" + (hb.all === 1 ? "y" : "ies")}{hb.late ? <span className="t-d">{" · " + hb.late + " late"}</span> : null}{hb.done ? <span className="t-k">{" · " + hb.done + " done"}</span> : null}</> : <><span className="lg"><i className="hd"></i>Delivered</span><span className="lg"><i className="ho"></i>Scheduled</span><span className="lg"><i className="hl"></i>Late</span></>}</div>
          {late.length > 0 && (
            <div className="wst">
              {late.slice(0, 3).map(s => <button key={s.id} className="wst-r" onClick={() => onOpenRow(s.id)}><span className="mono t-d">{s.delta}</span><b>{s.d[1].split(",")[0]}</b><em>{s.cust}</em></button>)}
              <button className="wv-go" onClick={() => onRef("late")}>{late.length > 3 ? "Review all " + n(late.length) + " late" : "Review late"}<Ic n="arrowR" s={11}></Ic></button>
            </div>
          )}
        </section>
      )}
      {unc.length > 0 && (
        <section className="wv-s">
          <div className="wv-h"><span>Uncovered pickups</span><b className="mono t-w">{money(unc.reduce((t, s) => t + s.rev, 0))}<em>{" · " + n(unc.length) + " loads"}</em></b></div>
          <div className="uw">{wins.map((w, i) => <button key={w.l} className={"uw-c u" + i + (w.c ? "" : " z")} disabled={!w.c} onClick={() => onFilter([{ k: "pwin", v: [w.a, w.b], label: "Pickup " + w.l }])}><b className="mono">{n(w.c)}</b><span>{w.l}</span><em className="mono">{w.c ? money(w.rev) : "—"}</em></button>)}</div>
          {first && <div className="nxt"><span>Next pickup in</span><b className={"mono " + (left < 3600 ? "t-d" : "t-w")}>{hh + ":" + String(mm).padStart(2, "0")}<small>{":" + String(ss).padStart(2, "0")}</small></b><em>{first.o[1].split(",")[0] + " → " + first.d[1].split(",")[0]}</em><button className="btn sm" onClick={() => onOpenRow(first.id)}>Cover</button></div>}
        </section>
      )}
      {dets.length > 0 && (
        <section className="wv-s">
          <div className="wv-h"><span>Detention accruing</span><b className="mono t-b">{"$" + detAmt.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}<em>{" · " + dets.length + " stop" + (dets.length > 1 ? "s" : "")}</em></b></div>
          <div className="dtl">{dets.slice(0, 3).map(({ s, m }) => <div key={s.id} className="dtl-r"><span className="dtl-b"><i style={{ width: Math.min(100, (m - 120) / 180 * 100) + "%" }}></i></span><b>{(() => { const cv = covOf(s, window.__ORG); return cv && cv.carrier ? cv.carrier.n : s.drv.n.split(" ").map((x, i) => i ? x : x[0] + ".").join(" "); })()}</b><em className="mono">{Math.floor(m / 60) + "h " + String(Math.floor(m % 60)).padStart(2, "0") + "m"}</em><span className="mono">{"$" + (Math.max(0, m - 120) / 60 * 75).toFixed(0)}</span><button className="lnk2" onClick={() => { setDetBilled(x => [...x, s.id]); onAct("Detention added to " + s.id); }}>Bill</button></div>)}</div>
          {dets.length > 3 ? <button className="wv-go" onClick={() => onFilter([{ k: "det", v: 1, label: "Detention" }])}>{"Review all " + dets.length}<Ic n="arrowR" s={11}></Ic></button> : null}
        </section>
      )}
      {bill.length > 0 && (
        <section className="wv-s">
          <div className="wv-h"><span>Ready to bill</span><b className="mono t-k">{money(billAmt)}<em>{" · " + n(bill.length) + " loads"}</em></b></div>
          {!billed && <div className="bc">{byCust.slice(0, 4).map(([c, [k, v]]) => <div key={c} className="bc-r"><span className="bc-b"><i style={{ width: (v / cmax) * 100 + "%" }}></i></span><b>{c}</b><em className="mono">{k}</em><span className="mono">{money(v)}</span></div>)}{byCust.length > 4 && <div className="bc-more">{"+" + (byCust.length - 4) + " more customer" + (byCust.length - 4 > 1 ? "s" : "")}</div>}</div>}
          <Hold label={"Transfer " + n(bill.length) + " to billing"} done={billed} onDone={() => { setBilled(true); onAct(n(bill.length) + " shipment" + (bill.length > 1 ? "s" : "") + " transferred to billing"); }}></Hold>
          {!billed && bill.length > 1 && <button className="wv-go" style={{ marginTop: 8 }} onClick={() => onFilter([{ k: "bill", v: 1, label: "Ready to bill" }])}>Review first<Ic n="arrowR" s={11}></Ic></button>}
        </section>
      )}
    </div>
  );
}

function Queue({ items, ai, done, onApprove, onReview, onUndo }) {
  const [order, setOrder] = React.useState(items.map(i => i.id));
  const [pick, setPick] = React.useState(null);
  const [last, setLast] = React.useState(null);
  const ids = items.map(i => i.id);
  React.useEffect(() => setOrder(ids), [ids.join()]);
  const byId = id => items.find(i => i.id === id);
  const ord = [...order.filter(id => byId(id)), ...ids.filter(id => !order.includes(id))];
  const open = ord.filter(id => !done[id]);
  const cur = pick && !done[pick] ? byId(pick) : byId(open[0]);
  const nDone = items.length - open.length;
  const approve = it => { setLast(it); setPick(null); ai && it.approve ? onApprove(it) : onReview(it); };
  const later = it => { setOrder(o => [...o.filter(x => x !== it.id), it.id]); setPick(null); };
  React.useEffect(() => {
    const k = e => {
      if (!cur || e.target.closest("input,textarea")) return;
      if ((e.metaKey || e.ctrlKey) && e.key === "Enter") { e.preventDefault(); approve(cur); }
      else if (e.altKey && e.key.toLowerCase() === "l") { e.preventDefault(); later(cur); }
    };
    window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k);
  }, [cur && cur.id, ai]);
  return (
    <div className="aq">
      <div className="aq-h">
        <span>{ai ? "Suggested actions" : "Exceptions"}</span>
        <span className="aq-n mono">{open.length ? (nDone + 1) + " of " + items.length : "done"}</span>
      </div>
      <div className="aq-seg">{ord.map(id => <i key={id} className={done[id] ? "sg-k" : cur && id === cur.id ? "sg-c" : ""}></i>)}</div>
      {cur ? (
        <div className="aq-cur" key={cur.id}>
          <div className="aq-k"><span className={"aq-dot " + cur.tone}></span>{cur.k}<span className="sp"></span><span className="mono">{cur.due}</span></div>
          <b className="aq-t">{ai ? cur.t : cur.pt}</b>
          <p className="aq-s">{ai ? cur.s : cur.ps}</p>
          {cur.imp && <div className="aq-imp">{cur.imp.map(x => <span key={x}>{x}</span>)}</div>}
          <div className="aq-a">
            <button className={"btn sm " + (ai && cur.approve ? "teal" : "ink")} onClick={() => approve(cur)}>{ai && cur.approve ? cur.approve : cur.manual}<span className="kg"><span className="kbd">{MOD}</span><span className="kbd">↵</span></span></button>
            {ai && cur.approve && <button className="btn ghost sm" onClick={() => onReview(cur)}>{cur.review || "Review"}</button>}
            <span className="sp"></span>
            {open.length > 1 && <button className="btn ghost sm" onClick={() => later(cur)}>Later<span className="kg"><span className="kbd">Alt</span><span className="kbd">L</span></span></button>}
          </div>
        </div>
      ) : (
        <div className="aq-zero"><span className="aq-ck"><Ic n="check" s={14} w={2.4}></Ic></span><div><b>All caught up</b><span>{nDone + " handled this shift"}</span></div></div>
      )}
      {open.length > 1 && (
        <div className="aq-l">
          <div className="aq-lh">Up next</div>
          {open.filter(id => !cur || id !== cur.id).map(id => { const it = byId(id); return <button key={id} className="aq-r" onClick={() => setPick(id)}><span className={"aq-dot " + it.tone}></span><span className="aq-rt">{ai ? it.t : it.pt}</span><span className="mono">{it.due}</span></button>; })}
        </div>
      )}
      {last && done[last.id] && <div className="aq-undo"><Ic n="check" s={12} w={2.4}></Ic><span>{last.okText || "Done"}</span><button onClick={() => { onUndo(last); setLast(null); }}>Undo</button></div>}
    </div>
  );
}

const QUEUE = [
  { id: "q1", k: "Coverage", due: "13:00", imp: ["$2,900", "11 mi out", "96% fit"], tone: "w", ic: "user", ship: "S2610-0431", drv: 0, t: "Assign Marcus Bell to San Antonio → Dallas", s: "11 mi from pickup, 10:10 HOS, ran this lane 4× · 96% fit. Pickup at 13:00.", approve: "Assign", okText: "Marcus Bell assigned to S2610-0431", pt: "San Antonio → Dallas has no driver", ps: "Pickup today at 13:00 · $2,900", manual: "Assign driver" },
  { id: "q2", k: "Coverage", due: "tmrw 08:00", imp: ["$1,920", "18 mi out", "92% fit"], tone: "w", ic: "user", ship: "S2610-0430", drv: 0, t: "Assign Kai Whitehorse to Fort Worth → Houston", s: "Free after Omaha unload, 18 mi out tomorrow 06:00 · 92% fit.", approve: "Assign", okText: "Kai Whitehorse assigned to S2610-0430", pt: "Fort Worth → Houston has no driver", ps: "Pickup tomorrow 08:00 · $1,920", manual: "Assign driver" },
  { id: "q3", k: "Delay notice", due: "now", imp: ["2 customers", "3 loads", "up to 2h 40m"], tone: "d", ic: "send", ship: "S2610-0412", t: "Send delay notices to Hy-Vee and Cargill", s: "Drafted with new ETAs 18:40 and 21:10, plus a request to rebook appointments.", approve: "Send both", okText: "Delay notices sent to 2 customers", pt: "3 loads will miss delivery appointments", ps: "S2610-0412, 0398, 0421 · up to 2h 40m late", manual: "Open late loads", review: "Read drafts" },
  { id: "q4", k: "Hours of service", due: "2:15 left", imp: ["60 mi short", "relay at Davenport"], tone: "b", ic: "route", ship: "S2610-0412", t: "Relay Jaime Park's load at Davenport", s: "He runs out of hours 60 mi short of Chicago. Luis Mendez can take it from there.", review: "See plan", okText: "", pt: "Jaime Park has 2:15 of drive time left", ps: "S2610-0412 · 172 mi to go", manual: "Open shipment" },
  { id: "q5", k: "Tender", due: "1h ago", imp: ["$2.48/mi", "3 similar this week"], tone: "t", ic: "receipt", t: "Re-tender S2610-0433 to Schneider", s: "Werner declined an hour ago. Schneider took 3 similar loads this week at $2.48/mi.", approve: "Re-tender", okText: "Tender sent to Schneider", pt: "Werner declined S2610-0433", ps: "Declined 1h ago · no backup carrier set", manual: "Pick carrier" },
];

function Brief({ ai, quiet, list, items, done, onApprove, onReview, onUndo, onRef, onFilter, onAct, onOpenRow, aiNote, setAiNote }) {
  const late = list.filter(s => s.late).length, open = list.filter(s => !s.drv).length;
  const left = items.filter(a => !done[a.id]).length;
  const R = ({ k, c, children }) => <span className={"ref " + (c || "")} onClick={() => onRef(k)}>{children}</span>;
  return (
    <div className="rl-p">
      <div>
        <Queue items={items} ai={ai} done={done} onApprove={onApprove} onReview={onReview} onUndo={onUndo}></Queue>
      </div>
      <Watchlist list={list} onRef={onRef} onFilter={onFilter} onAct={onAct} onOpenRow={onOpenRow}></Watchlist>
      {!ai && aiNote && (
        <div className="note"><Ic n="diamond" s={13}></Ic><div><b>AI suggestions are off for this workspace</b>Connect an AI provider to get shift briefs, driver matches and drafted customer updates. Everything else on this page works without it.<div style={{ display: "flex", gap: 4 }}><button className="btn sm"><Ic n="plug" s={12}></Ic>Open integrations</button><button className="btn ghost sm" onClick={() => setAiNote(false)}>Hide</button></div></div></div>
      )}
    </div>
  );
}

function Activity({ quiet }) {
  const act = quiet ? [{ t: <><b>Eric Moss</b> created S2610-0431</>, at: "3h" }, { t: <><b>Eric Moss</b> created S2610-0430</>, at: "3h" }] : ACTIVITY;
  return <div className="rl-p"><ul className="feed">{act.map((a, i) => <li key={i} className={a.c || ""}><span>{a.t}</span><em>{a.at}</em></li>)}</ul></div>;
}

function Detail({ s, ai, onBack, onAssign, onNotify, assignedNow, onAsk: p_ask }) {
  const sug = SUGGEST[s.id];
  const moving = s.st === "transit" || s.st === "delayed";
  return (
    <div className="rl-p dt" key={s.id}>
      <div className="dt-h"><button className="ib" onClick={onBack} title="Back (Esc)"><Ic n="chevL" s={14}></Ic></button><span className="mono">{s.id}</span><span className="sp"></span><button className="ib" title="Open record"><Ic n="ext" s={13}></Ic></button><button className="ib"><Ic n="more" s={14}></Ic></button></div>
      <div>
        <span className={"st " + s.st}><i></i>{ST[s.st]}</span>
        <h2 style={{ marginTop: 10 }}>{s.o[1]}<Ic n="arrowR" s={14}></Ic>{s.d[1]}</h2>
        <div className="dt-sub">{s.cust} · {s.equip} · {s.mi} mi</div>
        <div className="dt-pg"><span>{s.o[0]}</span><span className="bar"><i className={s.st === "delayed" ? "d" : s.st === "done" ? "k" : ""} style={{ width: s.pct + "%" }}></i></span><span>{s.d[0]}</span></div>
      </div>
      {assignedNow && <div className="ai-b ok"><span className="sweep"></span><div className="who"><Ic n="check" s={13} w={2.4}></Ic><b style={{ color: "inherit" }}>Assigned to {s.drv.n}</b></div><p>Dispatch sheet sent to {s.drv.unit}. Tender to carrier queued.</p></div>}
      {!s.drv && sug && (ai ? (
        <div className="ai-b">
          <div className="who"><span className="at"><Ic n="diamond" s={10} w={2}></Ic></span><b>Assistant</b>suggests</div>
          <p><b>{sug[0].n}</b> is {sug[0].mi} mi from pickup with {sug[0].hos} hours left and has run this lane 4 times.</p>
          <div className="opt">{sug.map((d, i) => <button key={d.n} className="op" onClick={() => onAssign(s.id, d)}><span className="av" style={{ "--h": d.h }}>{d.i}</span><div className="drv-t"><b>{d.n}</b><span>{d.unit} · {d.mi} mi · {d.hos} HOS</span></div><span className="fit">{i ? "81" : "96"}% fit</span></button>)}</div>
        </div>
      ) : (
        <div className="plain">
          <div className="sh">Nearest available drivers</div>
          <div className="opt" style={{ marginTop: 0 }}>{sug.map(d => <button key={d.n} className="op" onClick={() => onAssign(s.id, d)}><span className="av" style={{ "--h": d.h }}>{d.i}</span><div className="drv-t"><b>{d.n}</b><span>{d.unit} · {d.hos} HOS</span></div><span className="dist">{d.mi} mi</span><Ic n="plus" s={12}></Ic></button>)}</div>
          <button className="btn ghost sm" style={{ marginTop: 6 }}>Browse all drivers</button>
        </div>
      ))}
      {s.late && (ai ? (
        <div className="ai-b">
          <div className="who"><span className="at"><Ic n="diamond" s={10} w={2}></Ic></span><b>Assistant</b>drafted a customer update</div>
          <p>“{s.risk} has delayed {s.id}. New ETA is <b>{s.eta}</b>, {s.delta} past the appointment. We'll confirm a new delivery window within the hour.”</p>
          <div className="si-a"><button className="btn sm teal" onClick={() => onNotify(s)}><Ic n="send" s={12}></Ic>Send to {s.cust.split(" ")[0]}</button><button className="btn ghost sm">Edit</button></div>
        </div>
      ) : (
        <div className="note" style={{ borderStyle: "solid", borderColor: "color-mix(in oklch,var(--danger) 30%,transparent)" }}><Ic n="clock" s={14}></Ic><div><b>Running {s.delta} late</b>{s.risk}. The delivery appointment will be missed.<div><button className="btn sm" onClick={() => onNotify(s)}><Ic n="send" s={12}></Ic>Notify customer</button></div></div></div>
      ))}
      <div>
        <div className="sh">Stops</div>
        <ul className="trail">
          <li className={s.pct > 0 ? "d" : ""}><i></i><div><b>Pickup · {s.o[1]}</b><span>{s.pct > 0 ? "Loaded, BOL signed" : s.etaD}</span></div><em>{s.pct > 0 ? "06:12" : "—"}</em></li>
          {moving && <li className={s.late ? "late" : "now"}><i></i><div><b>{s.late ? s.risk : "Rolling"}</b><span>{Math.round(s.mi * (1 - s.pct / 100))} mi to go · ping 3m ago</span></div><em>{s.late ? "+" + s.delta.replace("+", "") : "now"}</em></li>}
          <li className={s.st === "done" ? "d" : ""}><i></i><div><b>Delivery · {s.d[1]}</b><span>{s.st === "done" ? "POD signed" : "Appointment " + s.eta}</span></div><em>{s.eta.split(" ").pop()}</em></li>
        </ul>
      </div>
      <dl className="facts">
        <div><dt>Order</dt><dd className="mono">{s.order}</dd></div>
        <div><dt>BOL</dt><dd className="mono">{s.bol}</dd></div>
        <div><dt>Revenue</dt><dd className="mono">${s.rev.toLocaleString()}</dd></div>
        <div><dt>Margin</dt><dd className="mono">{s.mg}%</dd></div>
        <div><dt>Weight</dt><dd className="mono">{s.wt.toLocaleString()} lb</dd></div>
        <div><dt>Tender</dt><dd>{s.tender}</dd></div>
      </dl>
      <div className="dt-f"><button className="btn sm"><Ic n="chat" s={12}></Ic>Message driver</button><button className="btn sm"><Ic n="download" s={12}></Ic>Documents</button>{ai && <button className="btn sm ghost" onClick={() => p_ask && p_ask(s)}><Ic n="diamond" s={11}></Ic>Ask assistant</button>}</div>
    </div>
  );
}

function Rail({ tab, setTab, sel, ai, quiet, list, ...p }) {
  const left = p.items.filter(a => !p.done[a.id]).length;
  return (
    <aside className="rl">
      <div className="rl-h">
        {[["brief", ai ? "Brief" : "Overview", left], ["activity", "Activity"]].map(([k, l, n]) => <button key={k} className={"rt" + (tab === k && !sel ? " on" : "")} onClick={() => { setTab(k); p.onBack(); }}>{k === "chat" && <Ic n="diamond" s={11} w={2}></Ic>}{l}{n ? <em>{n}</em> : null}</button>)}
        <span className="sp"></span>
        <button className="ib" onClick={p.onClose} title="Close panel"><Ic n="x" s={13}></Ic></button>
      </div>
      <div className={"rl-s" + (!sel && tab === "chat" ? " flat" : "")}>
        {sel ? <Detail s={sel} ai={ai} onBack={p.onBack} onAssign={p.onAssign} onNotify={p.onNotify} assignedNow={p.assignedNow === sel.id} onAsk={p.onAsk}></Detail>
          : tab === "brief" || tab === "chat" ? <Brief ai={ai} quiet={quiet} list={list} {...p}></Brief>
          : <Activity quiet={quiet}></Activity>}
      </div>
    </aside>
  );
}
function TopBrief({ list, quiet, onRef }) {
  const late = list.filter(s => s.late).length, open = list.filter(s => !s.drv).length;
  const moving = list.filter(s => s.st === "transit").length;
  let segs;
  if (quiet) segs = [["Quiet start. "], [list.length + " loads", null], [" on the board, both picking up soon, "], open ? ["and "] : ["and both are covered."], open ? [(open > 1 ? "neither is" : "one isn't") + " covered yet", "no driver"] : null, open ? [". I've lined up a match for each."] : null];
  else segs = [[list.length + " loads today. "], moving ? [moving + " are moving on schedule", "moving"] : null, moving ? [", "] : null, late ? [late + (late > 1 ? " are" : " is") + " running late", "late"] : ["nothing is running late"], late ? [" behind the storm over Iowa"] : null, open ? [", and "] : ["."], open ? [open + " still need" + (open > 1 ? "" : "s") + (window.__ORG === "brokerage" ? " a carrier" : window.__ORG === "both" ? " coverage" : " a driver"), "no driver"] : null, open ? [". "] : [" "], late || open ? ["I've drafted fixes in the brief."] : ["Nothing needs you."]];
  segs = segs.filter(Boolean);
  const total = segs.reduce((n, s) => n + s[0].length, 0);
  const [n, setN] = React.useState(0);
  const key = segs.map(s => s[0]).join("");
  React.useEffect(() => { setN(0); let i = 0; const t = setInterval(() => { i += 2 + Math.round(Math.random() * 2); setN(Math.min(i, total)); if (i >= total) clearInterval(t); }, 28); return () => clearInterval(t); }, [key]);
  let left = n;
  const out = segs.map(([txt, ref], i) => {
    if (left <= 0) return null;
    const vis = txt.slice(0, left); left -= txt.length;
    const words = vis.split(/(\s+)/).map((w, j) => <span key={j} className="sw">{w}</span>);
    return ref ? <span key={i} className="ref" onClick={() => onRef(ref)}>{words}</span> : <React.Fragment key={i}>{words}</React.Fragment>;
  });
  return <div className="tbr"><p>{out}{n < total && <span className="car"></span>}</p></div>;
}
class SafePanel extends React.Component {
  constructor(p) { super(p); this.state = { err: null }; }
  static getDerivedStateFromError(err) { return { err }; }
  componentDidUpdate(pp) { if (pp.resetKey !== this.props.resetKey && this.state.err) this.setState({ err: null }); }
  render() { return this.state.err ? <div className="rl-p"><div className="note"><Ic n="alert" s={14}></Ic><div><b>This panel hit a problem</b>The table still works.<div><button className="btn sm" onClick={() => this.setState({ err: null })}>Try again</button></div></div></div></div> : this.props.children; }
}
Object.assign(window, { SafePanel, Rail, QUEUE, TopBrief });
