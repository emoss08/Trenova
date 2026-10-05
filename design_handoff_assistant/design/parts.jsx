const AsCtx = React.createContext({});

const AGENTS = Object.fromEntries(DAGENTS.map(a => [a.id, { ...a, h: AG_ACCENT[a.accent] }]));
const PRESETS = {
  billing: ["Which billing items are stuck the longest?", "Which payments came in this week?"],
  dispatch: ["Which loads are running late today?", "Who has hours left for a reload?"],
  web: ["Explain EBITDA and how carriers report it", "What's the national diesel average?"],
};
const fav = d => `https://www.google.com/s2/favicons?domain=${d}&sz=32`;
const EBITDA_SRC = [
  { d: "investopedia.com", t: "EBITDA: Definition, Calculation Formulas, History, and Criticisms", s: "How EBITDA is calculated and why analysts use it to compare companies.", a: "Jun 2", u: "https://www.investopedia.com/terms/e/ebitda.asp" },
  { d: "corporatefinanceinstitute.com", t: "EBITDA — Definition, Formula & Calculation", s: "Net income with interest, taxes, depreciation and amortization added back.", a: "Mar 14", u: "https://corporatefinanceinstitute.com/resources/valuation/what-is-ebitda/" },
  { d: "wikipedia.org", t: "Earnings before interest, taxes, depreciation and amortization", s: "A non-GAAP measure; definitions vary between companies.", a: "Sep 30", u: "https://en.wikipedia.org/wiki/Earnings_before_interest,_taxes,_depreciation_and_amortization" },
];

const SCRIPTS = {
  stuck: {
    trace: [
      { live: "Reading the billing queue", done: "Read the billing queue", k: "billing.queue.list", v: "all", r: "10 rows", t: "0.6s" },
      { live: "Checking blockers on 4 items", done: "Checked blockers", k: "billing.queue.inspect", v: "blocked", r: "4 items", t: "0.9s" },
      { live: "Comparing rate confirmations", done: "Compared rate confirmations", k: "rating.compare", v: "PRO 40102", r: "1 mismatch", t: "0.7s" },
    ],
    dur: "2.2s",
    paras: [
      ["Three items have been blocked for five days or more, and all three are paperwork, not disputes.", { fn: 1 }],
      ["PRO 40118 for Halvorsen Foods has waited ", { b: "9 days" }, " on a signed POD.", { fn: 2 }, " PRO 40102 for Ridgeline Steel is ", { b: "7 days" }, " old because the invoice says $1,180 but the signed rate con says $1,240.", { fn: 3 }, " PRO 40131 for Cascade Paper is ", { b: "5 days" }, " old, waiting on approval for $185 of detention."],
      ["Ridgeline is the quickest fix. I can correct the linehaul so it matches the rate con."],
    ],
    decision: { id: "rate40102", title: "Correct linehaul", on: "on PRO 40102", field: "Linehaul", from: "$1,180.00", to: "$1,240.00", rev: [["Customer", "Ridgeline Steel"], ["Why", "Matches signed rate con, Sep 22"], ["Then", "Moves to Ready to bill"]], doing: "Correcting the linehaul on PRO 40102", event: "Linehaul corrected on PRO 40102" },
  },
  payments: {
    trace: [
      { live: "Reading payments received", done: "Read payments received", k: "payments.list", v: "since Monday", r: "5 rows", t: "0.8s" },
      { live: "Matching payments to invoices", done: "Matched payments to invoices", k: "payments.match", v: "5 payments", r: "2 short", t: "1.1s" },
    ],
    dur: "1.9s",
    paras: [
      [{ b: "$14,920" }, " came in across 5 payments since Monday. Copperline and Summit paid in full, and Ironbridge cleared their August balance.", { fn: 1 }],
      ["Two were short-paid. Blue Mesa paid ", { b: "$120 under" }, " and Northwind ", { b: "$45 under" }, ". Both look like they skipped the fuel surcharge.", { fn: 2 }],
    ],
  },
  ebitda: {
    web: true,
    trace: [{ live: "Searching the web", done: "Searched the web", k: "web.search", v: "EBITDA definition · trucking carriers", r: "3 sources", t: "2.1s", web: true }],
    dur: "2.1s",
    sources: EBITDA_SRC,
    paras: [
      [{ b: "EBITDA" }, " is earnings before interest, taxes, depreciation, and amortization. It strips out financing and accounting choices so you can compare how two businesses actually operate.", { c: [1] }],
      ["You get it by adding interest, taxes, depreciation, and amortization back onto net income.", { c: [2] }, " It isn't a GAAP measure, so companies define it a little differently.", { c: [3] }],
      ["For carriers it matters because trucks depreciate fast. Two fleets hauling the same freight can show very different net income just from the age of their equipment."],
    ],
  },
  generic: {
    trace: [{ live: "Reading the billing queue", done: "Read the billing queue", k: "billing.queue.list", v: "all", r: "10 rows", t: "0.7s" }],
    dur: "0.7s",
    paras: [["I can see the billing queue: 10 items, 4 of them blocked.", { fn: 1 }, " Ask about a PRO number, a customer, or a status and I'll dig in."]],
  },
};

const RECENTS = [
  { id: "r1", title: "Which billing items are stuck the longest?", agent: "billing", ago: "13h", g: "Today", script: "stuck" },
  { id: "r2", title: "Can you explain EBITDA to me, and how carriers usually report it?", agent: "web", ago: "13h", g: "Today", script: "ebitda" },
  { id: "r3", title: "Which payments came in this week?", agent: "billing", ago: "1d", g: "Yesterday", script: "payments" },
  { id: "r4", title: "Why is the Ridgeline invoice on hold?", agent: "billing", ago: "1d", g: "Yesterday", script: "generic" },
  { id: "r5", title: "Which loads are running late into Reno?", agent: "dispatch", ago: "3d", g: "This week", script: "generic" },
  { id: "r6", title: "Summarize detention charges for September", agent: "billing", ago: "4d", g: "This week", script: "generic" },
];

function AgentTile({ a, s = 20 }) {
  return <DAgentTile agent={a} size={s >= 24 ? "sm" : "xs"}></DAgentTile>;
}

function tokenize(paras) {
  const out = [];
  paras.forEach((segs, p) => segs.forEach(seg => {
    if (seg.c) { out.push({ p, c: seg.c }); return; }
    if (seg.fn) { out.push({ p, fn: seg.fn }); return; }
    const s = typeof seg === "string" ? { t: seg } : seg.b ? { t: seg.b, b: true } : seg;
    (s.t.match(/\S+\s*|\s+/g) || []).forEach(w => out.push({ p, w, b: s.b, r: s.r }));
  }));
  return out;
}

function Ref({ id, children }) {
  const c = React.useContext(AsCtx);
  return <span className="ref" onMouseEnter={() => c.setHot(id)} onMouseLeave={() => c.setHot(null)}>{children}</span>;
}

function Prose({ toks, shown, streaming, sources, trace, mid }) {
  const [hot, setHot] = React.useState(null);
  const vis = toks.slice(0, shown);
  const paras = [];
  vis.forEach((t, i) => (paras[t.p] = paras[t.p] || []).push([t, i]));
  return (
    <div className="prose">
      {paras.map((ws, p) => {
        const nodes = []; let k = 0;
        while (k < ws.length) {
          const [t, i] = ws[k];
          if (t.c) { nodes.push(<WebCite key={i} ids={t.c} sources={sources} live={streaming}></WebCite>); k++; continue; }
          if (t.fn) { nodes.push(<FnPop key={i} n={t.fn} step={trace[t.fn - 1]} open={hot === t.fn} onEnter={() => setHot(t.fn)} onLeave={() => setHot(null)}></FnPop>); k++; continue; }
          nodes.push(<span key={i} className={streaming ? "w" : undefined}>{t.b ? <strong>{t.w}</strong> : t.w}</span>); k++;
        }
        return <p key={p}>{nodes}{streaming && p === paras.length - 1 && <span className="caret"></span>}</p>;
      })}
    </div>
  );
}

const STEP_WHY = {
  "billing.queue.list": ["The full billing queue, 10 items", "You asked about the queue, so it read all of it instead of a filtered slice.", "Showing only blocked items — would have hidden what's close to blocking."],
  "billing.queue.inspect": ["The blocker on each of the 4 blocked items", "Each blocker has a different fix, so it checked them one by one.", "Trusting the status column, which doesn't say why."],
  "rating.compare": ["The invoice vs Ridgeline's signed rate con", "Anything billed that doesn't match the rate con gets flagged before billing.", "Skipping the check because the total looked normal."],
  "payments.list": ["Payments posted since Monday", "You asked about this week, so it started at Monday.", "Reading the last 7 days, which overlaps last week."],
  "payments.match": ["Each payment against its open invoices", "Matching shows short-pays that a payment total hides.", "Reporting the total only."],
};
function StepWhy({ step }) {
  const w = STEP_WHY[step.k] || [step.v + " → " + step.r, "This tool answers the question directly with the least data.", "Asking you to narrow it down first."];
  return <span className="why"><span><em>Saw</em>{w[0]}</span><span><em>Because</em>{w[1]}</span><span><em>Instead of</em>{w[2]}</span></span>;
}
function FnPop({ n, step, open, onEnter, onLeave }) {
  const [why, setWhy] = React.useState(false);
  const ref = React.useRef();
  const [pos, setPos] = React.useState(null);
  React.useLayoutEffect(() => {
    if (!open) { setPos(null); return; }
    const box = ref.current.closest(".scroll") || document.body;
    const b = box.getBoundingClientRect(), f = ref.current.getBoundingClientRect();
    const W = 250, pad = 10, cx = f.left + f.width / 2;
    const left = Math.min(Math.max(cx - W / 2, b.left + pad), b.right - pad - W) - cx;
    setPos({ dx: left, below: f.top - b.top < 150 });
  }, [open]);
  return (
    <span className="fnw" ref={ref} onMouseEnter={onEnter} onMouseLeave={onLeave}>
      <span className={"fn" + (open ? " hot" : "")}>{n}</span>
      {open && step && <span className={"fnp" + (pos && pos.below ? " below" : "")} role="tooltip" style={pos ? { transform: `translateX(${pos.dx}px)`, left: "50%", visibility: "visible" } : { visibility: "hidden" }}><span className="fnp-in">
        <span className="fnp-h"><span>{String(n).padStart(2, "0")}</span><span>{step.k}</span><span className="t">{step.t}</span></span>
        <span className="fnp-b">{step.done}</span>
        <span className="fnp-q"><span>{step.v}</span><Ic n="chevR" s={10}></Ic><b>{step.r}</b></span>
        <button className={"fnp-why" + (why ? " on" : "")} onClick={e => { e.stopPropagation(); setWhy(w => !w); }}><Ic n="why" s={12}></Ic>{why ? "Hide reasoning" : "Why this step?"}</button>
        {why && <StepWhy step={step}></StepWhy>}
      </span></span>}
    </span>
  );
}

const webHue = d => { let h = 0; for (let i = 0; i < d.length; i++) h = (h * 31 + d.charCodeAt(i)) % 360; return h; };
function Fav({ d, s = 14 }) {
  return <span className="wf" style={{ width: s, height: s, fontSize: s * 0.6, "--wh": webHue(d) }}>{d[0].toUpperCase()}</span>;
}
function WebCite({ ids, sources, live }) {
  const [open, setOpen] = React.useState(false);
  const list = ids.map(i => sources[i - 1]).filter(Boolean);
  if (!list.length) return null;
  const first = list[0];
  return (
    <span className="wc-w" onMouseEnter={() => setOpen(true)} onMouseLeave={() => setOpen(false)}>
      <a className={"wc" + (live ? " w" : "")} href={first.u} target="_blank" rel="noreferrer">{first.d.replace(/\.(com|gov|org)$/, "")}{list.length > 1 && <em>+{list.length - 1}</em>}</a>
      {open && <span className="wc-pop"><span className="wc-in">
        {list.map(s => <a key={s.d} className="wc-it" href={s.u} target="_blank" rel="noreferrer"><span className="wc-d"><Fav d={s.d} s={12}></Fav>{s.d}<i>{s.a}</i></span><b>{s.t}</b><span className="wc-s">{s.s}</span></a>)}
      </span></span>}
    </span>
  );
}
function WebLive({ step, sources }) {
  return (
    <div className="wl">
      <div className="wl-h"><Ic n="globe" s={13}></Ic><span className="wl-l">{step.live}</span></div>
      <div className="wl-q">{step.v}</div>
      <div className="wl-s">{sources.map((s, i) => <span key={s.d} className="wl-it" style={{ animationDelay: 350 + i * 420 + "ms" }}><Fav d={s.d} s={12}></Fav>{s.d}</span>)}</div>
    </div>
  );
}
function WebSources({ s }) {
  const [open, setOpen] = React.useState(false);
  const src = s.sources;
  const q = (s.trace.find(x => x.web) || {}).v;
  return (
    <div className={"ws" + (open ? " open" : "")}>
      <button className="ws-b" onClick={() => setOpen(o => !o)}><span className="ws-st">{src.slice(0, 4).map(x => <Fav key={x.d} d={x.d} s={14}></Fav>)}</span><span>{src.length} sources</span><Ic n="chevR" s={10}></Ic></button>
      <div className="ws-x"><div>
        {q && <div className="ws-q"><Ic n="search" s={11}></Ic>{q}</div>}
        <ol className="ws-l">{src.map((x, i) => <li key={x.d}><a href={x.u} target="_blank" rel="noreferrer"><span className="ws-n">{i + 1}</span><Fav d={x.d} s={14}></Fav><span className="ws-t">{x.t}</span><span className="ws-d">{x.d} · {x.a}</span></a></li>)}</ol>
      </div></div>
    </div>
  );
}

function Narr({ s, phase, step, el, stopped }) {
  const [open, setOpen] = React.useState(false);
  if (phase !== "done") {
    const past = s.trace.slice(0, step).slice(-2);
    const label = stopped ? "Stopped" : phase === "work" ? s.trace[step].live : "Writing the answer";
    return (
      <div className="nar"><div className="nar-live">
        {past.map(x => <div className="nl past" key={x.done}><span className="ck"><Ic n="check" s={12} w={2.2}></Ic></span><span>{x.done}</span></div>)}
        <div className={"nl " + (stopped ? "stop" : "cur")} key={"c" + step + phase}><span className="ck">{stopped ? <Ic n="stop" s={11}></Ic> : <span className="spark"></span>}</span><span className="tx">{label}</span><span className="el">{el.toFixed(1)}s</span></div>
      </div></div>
    );
  }
  return (
    <div className="nar">
      <button className={"nsum" + (open ? " open" : "")} onClick={() => setOpen(!open)}>
        <span className="chev"><Ic n="chevR" s={12}></Ic></span><span>Worked through {s.trace.length} step{s.trace.length > 1 ? "s" : ""} in {s.dur}</span>
      </button>
      <div className={"nexp" + (open ? " open" : "")}><div><div className="nlist">
        {s.trace.map((x, i) => <div className="ni" key={i}><span>{x.done} <em>· {x.r}</em></span><span className="t">{x.t}</span></div>)}
      </div></div></div>
    </div>
  );
}

function MsgActions() {
  const [copied, setCopied] = React.useState(false);
  const [reading, setReading] = React.useState(false);
  React.useEffect(() => { if (!copied) return; const h = setTimeout(() => setCopied(false), 1400); return () => clearTimeout(h); }, [copied]);
  return (
    <div className={"acts" + (copied || reading ? " stay" : "")}>
      <button className={"act" + (copied ? " ok" : "")} data-tip={copied ? "Copied" : "Copy"} onClick={() => setCopied(true)}>{copied ? <Ic n="check" s={14} w={2.2}></Ic> : <Ic n="copy" s={14}></Ic>}</button>
      <button className="act" data-tip={reading ? "Stop reading" : "Read aloud"} onClick={() => setReading(!reading)}>{reading ? <span className="eq">{[0, 1, 2, 3].map(k => <i key={k} style={{ animationDelay: k * 120 + "ms" }}></i>)}</span> : <Ic n="speaker" s={14}></Ic>}</button>
    </div>
  );
}

function Reply({ m, last }) {
  const c = React.useContext(AsCtx);
  const s = SCRIPTS[m.script];
  const toks = React.useMemo(() => tokenize(s.paras), [m.script]);
  const stepMs = s.web ? 1900 : 950;
  const [st, setSt] = React.useState(m.live ? { phase: "work", step: 0, shown: 0, el: 0 } : { phase: "done", step: s.trace.length, shown: toks.length, el: 0 });
  React.useEffect(() => {
    if (st.phase === "done" || m.stopped) return;
    const t0 = performance.now() - st.el * 1000;
    const id = setInterval(() => {
      setSt(p => {
        const el = (performance.now() - t0) / 1000;
        if (p.phase === "work") { const step = Math.floor(el * 1000 / stepMs); return step >= s.trace.length ? { ...p, phase: "write", step: s.trace.length, el } : { ...p, step, el }; }
        const workEnd = s.trace.length * stepMs / 1000;
        const shown = Math.max(1, Math.floor((el - workEnd) * 38));
        return shown >= toks.length ? { ...p, phase: "done", shown: toks.length, el } : { ...p, shown, el };
      });
    }, 50);
    return () => clearInterval(id);
  }, [st.phase, m.stopped]);
  React.useEffect(() => {
    if (!m.live) return;
    if (m.stopped || st.phase === "done") { c.finish(m.id, s, !!m.stopped, m.tid); return; }
    c.status({ t: st.phase === "work" ? s.trace[st.step].live : "Writing the answer", el: st.el });
  }, [st.phase, st.step, Math.floor(st.el * 10), m.stopped]);
  const done = st.phase === "done";
  const cur = st.phase === "work" && !m.stopped && s.trace[st.step];
  return (
    <div className="msg">
      {cur && cur.web && s.sources && <WebLive step={cur} sources={s.sources}></WebLive>}
      {st.shown > 0 && <Prose toks={toks} shown={st.shown} streaming={!done && !m.stopped} sources={s.sources || []} trace={s.trace}></Prose>}
      {done && s.sources && <WebSources s={s}></WebSources>}
      {done && <MsgActions></MsgActions>}
    </div>
  );
}

function useTypewriter(list, active) {
  const [st, setSt] = React.useState({ i: 0, n: 0 });
  React.useEffect(() => {
    if (!active) return;
    const full = list[st.i];
    const t = st.n < full.length ? setTimeout(() => setSt({ ...st, n: st.n + 1 }), 32) : setTimeout(() => setSt({ i: (st.i + 1) % list.length, n: 0 }), 2600);
    return () => clearTimeout(t);
  }, [st, active, list]);
  React.useEffect(() => setSt({ i: 0, n: 0 }), [list]);
  const full = list[st.i] || "";
  return { text: full.slice(0, st.n), full, idx: st.i };
}

function Dictate({ setValue }) {
  const [rec, setRec] = React.useState(false);
  const [t, setT] = React.useState(0);
  React.useEffect(() => {
    if (!rec) return;
    const st = performance.now();
    const iv = setInterval(() => setT((performance.now() - st) / 1000), 100);
    return () => clearInterval(iv);
  }, [rec]);
  const stop = () => { setRec(false); setValue(v => (v ? v + " " : "") + "And send me the totals by customer when it's done."); };
  if (!rec) return <button className="ib" title="Dictate" onClick={() => { setT(0); setRec(true); }}><Ic n="mic" s={15}></Ic></button>;
  return (
    <button className="dict" onClick={stop} title="Stop dictating">
      <span className="wave">{[0, 1, 2, 3, 4].map(k => <i key={k} style={{ animationDelay: k * 110 + "ms" }}></i>)}</span>
      <span className="mono">0:{String(Math.floor(t)).padStart(2, "0")}</span>
      <span className="dict-x"></span>
    </button>
  );
}

function AttachMenu({ onClose }) {
  const inp = React.useRef(null);
  const root = React.useRef(null);
  React.useEffect(() => {
    const k = e => e.key === "Escape" && onClose();
    const off = e => root.current && !root.current.contains(e.target) && !e.target.closest(".cmp-b .ib") && onClose();
    window.addEventListener("keydown", k); document.addEventListener("mousedown", off);
    return () => { window.removeEventListener("keydown", k); document.removeEventListener("mousedown", off); };
  }, []);
  return (
    <div className="am" ref={root}>
      <input ref={inp} type="file" multiple hidden onChange={e => { e.target.value = ""; onClose(); }}></input>
      <button onClick={() => inp.current.click()}><Ic n="plus" s={14}></Ic><span><b>Upload from computer</b><em>PDF, images, CSV, Excel · up to 5 files, 25 MB each</em></span></button>
      <button onClick={onClose}><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M4 15h16v4H4zM6 15V9l4-4h8v10"></path><path d="M8 18h.01"></path></svg><span><b>Scan from Capture</b><em>Scan paper straight into this message</em></span></button>
      <div className="am-f"><span className="kbd">⌘U</span> or drop files anywhere</div>
    </div>
  );
}

function Composer({ agent, setAgent, busy, status, onSend, onStop, home, pageSel, setPageSel, ctx, onCompact, compacting, onCancelCompact }) {
  const [v, setV] = React.useState("");
  const [menu, setMenu] = React.useState(false);
  const [model, setModel] = React.useState("");
  const ta = React.useRef();
  const presets = PRESETS[agent] || PRESETS.billing;
  const empty = !v;
  const tw = useTypewriter(presets, home && empty);
  React.useEffect(() => { const t = ta.current; t.style.height = "auto"; t.style.height = Math.min(t.scrollHeight, 140) + "px"; }, [v]);
  React.useEffect(() => { ta.current.focus(); }, []);
  const go = txt => { const s = (txt ?? v).trim(); if (!s || busy) return; onSend(s); setV(""); };
  const onKey = e => {
    if (home && empty && e.key === "Tab") { e.preventDefault(); setV(tw.full); return; }
    if (home && (e.metaKey || e.ctrlKey) && /^[1-9]$/.test(e.key) && presets[+e.key - 1]) { e.preventDefault(); go(presets[+e.key - 1]); return; }
    if (e.key === "Enter" && !e.shiftKey && !e.metaKey && !e.ctrlKey) { e.preventDefault(); go(); }
  };
  return (
    <div className={"cmp" + (busy ? " busy" : "") + (home && !busy ? " hm" : "") + (compacting ? " cmpg" : "")}>
      <span className="cmp-ring" aria-hidden="true"></span>
      {compacting && <div className="cmp-st cx-st"><CtxDrain from={compacting.before} to={compacting.after} ms={2600}></CtxDrain><span className="shim cx-stt">{compacting.auto ? "Context is nearly full · compacting" : "Compacting the conversation"}…</span><span className="ec-sp"></span><button className="ec-link" onClick={onCancelCompact}>Cancel</button></div>}
      {busy && status && <div className="cmp-st" key={status.t}><span className="spark"></span><span className="shim">{status.t}</span><span className="el">{status.el.toFixed(1)}s</span></div>}
      <div className="cmp-ta">
        <textarea ref={ta} rows={1} value={v} disabled={!!compacting} placeholder={home ? "" : compacting ? "You can reply once compacting finishes" : `Reply to ${AGENTS[agent].name}…`} onChange={e => setV(e.target.value)} onKeyDown={onKey}></textarea>
        {home && empty && <div className="tw" aria-hidden="true"><span>{tw.text}</span><i className="tw-c"></i>{tw.text === tw.full && <span className="tw-tab"><span className="kbd">Tab</span><em>to use</em><span className="kbd">⌘{tw.idx + 1}</span><em>to ask</em></span>}</div>}
      </div>
      <div className="cmp-b">
        <span className="amw">
          <button className={"ib" + (menu ? " on" : "")} title="Attach files" onClick={() => setMenu(m => !m)}><Ic n="plus" s={16}></Ic></button>
          {menu && <AttachMenu onClose={() => setMenu(false)}></AttachMenu>}
        </span>
        <DAgentPicker agent={AGENTS[agent]} onSelect={a => setAgent(a.id)}></DAgentPicker>
        <PageChip currentKey="billing" sel={pageSel} setSel={setPageSel} onExplain={() => setV("Explain what's on this page")}></PageChip>
        <span className="sp"></span>
        <ModelPicker value={model} onChange={setModel} hasReplies={!home}></ModelPicker>
        {!home && ctx && <ContextMeter ctx={ctx} onCompact={onCompact} compacting={!!compacting}></ContextMeter>}
        <Dictate setValue={setV}></Dictate>
        {busy ? <button className="send stop" title="Stop" onClick={onStop}><span className="sq"></span></button>
          : <button className="send" disabled={!v.trim() || !!compacting} onClick={() => go()} title="Send"><Ic n="up" s={15} w={2.2}></Ic></button>}
      </div>
    </div>
  );
}

function DecisionDock({ d, phase, n, onApprove, onDismiss, onUndo, onNow }) {
  const [rv, setRv] = React.useState(false);
  if (phase === "ok") return (
    <div className="dcx ok"><span className="okr"><svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3.2" strokeLinecap="round" strokeLinejoin="round"><path d="M5 12.5l4.5 4.5L19 7"></path></svg></span>
      <span className="dcx-t"><b>Approved</b><span>{d.doing}…</span></span><span className="shine"></span></div>
  );
  if (phase === "undo") return (
    <div className="udo">
      <svg className="udo-ring" width="22" height="22" viewBox="0 0 22 22"><circle cx="11" cy="11" r="9" className="bg"></circle><circle cx="11" cy="11" r="9" className="fg" style={{ strokeDashoffset: 56.5 * (1 - n / 5) }}></circle><text x="11" y="14.5" textAnchor="middle">{n}</text></svg>
      <span className="udo-t"><b>Approved</b><span>{d.doing} in {n}s</span></span>
      <button className="bt" onClick={onUndo}><Ic n="undo" s={12} w={2}></Ic>Undo</button>
      <button className="bt ghost" onClick={onNow}>Now</button>
    </div>
  );
  return (
    <div className={"dcx" + (rv ? " wide" : "")}>
      <span className="dcx-i"><Ic n="info" s={15} w={2}></Ic></span>
      <span className="dcx-t"><b>{d.title} <span className="dcx-s">{d.on}</span></b><span className="dcx-d">{d.field} <s>{d.from}</s> → <em>{d.to}</em><span className="dcx-m">· reversible</span></span></span>
      <span className="dcx-a">
        <button className={"bt" + (rv ? " on" : "")} onClick={() => setRv(!rv)}>{rv ? "Hide" : "Review"}</button>
        <button className="bt" onClick={onDismiss}>Not now</button>
        <button className="apv-b" onClick={onApprove}>Approve<span className="kbd">⌘↵</span></button>
      </span>
      {rv && <div className="dcx-rv">{d.rev.map(([k, v]) => <React.Fragment key={k}><span>{k}</span><b>{v}</b></React.Fragment>)}</div>}
    </div>
  );
}

Object.assign(window, { AsCtx, AGENTS, SCRIPTS, RECENTS, AgentTile, Reply, Composer, DecisionDock });
