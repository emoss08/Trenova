const CTX_MAX = 200000;
const CTX_AUTO_AT = 0.85;
const CTX_PARTS = [["conv", "Messages", "c1"], ["tools", "Tool results & artifacts", "c2"], ["files", "Files", "c3"], ["sys", "Agent instructions", "c4"]];
const CTX0 = { sys: 6200, conv: 41800, tools: 79500, files: 12300 };
const CTX_FRESH = { sys: 6200, conv: 0, tools: 0, files: 0 };
const ctxTotal = c => c.sys + c.conv + c.tools + c.files;
const kfmt = n => n >= 1000 ? (n / 1000).toFixed(n >= 100000 ? 0 : 1).replace(/\.0$/, "") + "k" : String(n);
const CMP_PATH = <path d="M12 3v6M9 6l3 3 3-3M12 21v-6M9 18l3-3 3 3M5 12h14"></path>;

const USAGE = [
  { t: "Today · you", r: "Resets in 6 hr 12 min", p: 18 },
  { t: "October · Billing Manager", r: "Resets Nov 1", p: 64, d: "$318 / $500" },
  { t: "October · organization", r: "Resets Nov 1", p: 41 },
];

function CtxRing({ p, s = 16 }) {
  const r = (s - 3) / 2, c = 2 * Math.PI * r;
  return (
    <svg width={s} height={s} viewBox={`0 0 ${s} ${s}`} className="cx-ring">
      <circle cx={s / 2} cy={s / 2} r={r} fill="none" strokeWidth="2" className="cx-rt"></circle>
      <circle cx={s / 2} cy={s / 2} r={r} fill="none" strokeWidth="2" strokeLinecap="round" strokeDasharray={c} strokeDashoffset={c * (1 - Math.min(1, p))} transform={`rotate(-90 ${s / 2} ${s / 2})`} className="cx-rf"></circle>
    </svg>
  );
}

function ContextMeter({ ctx, onCompact, compacting }) {
  const [open, setOpen] = React.useState(false);
  const [more, setMore] = React.useState(false);
  const ref = React.useRef(null);
  React.useEffect(() => {
    if (!open) return;
    const h = e => { if (ref.current && !ref.current.contains(e.target)) setOpen(false); };
    const k = e => { if (e.key === "Escape") setOpen(false); };
    document.addEventListener("mousedown", h); document.addEventListener("keydown", k);
    return () => { document.removeEventListener("mousedown", h); document.removeEventListener("keydown", k); };
  }, [open]);
  const used = ctxTotal(ctx.parts), p = used / CTX_MAX, pct = Math.round(p * 100);
  const tone = p >= 0.9 ? " hot" : p >= 0.75 ? " warm" : "";
  const frees = used - ctx.parts.sys - Math.min(ctx.parts.files, 4000) - 8000;
  return (
    <span className={"cx" + tone} ref={ref}>
      <button className={"cx-b" + (open ? " on" : "") + (compacting ? " busy" : "")} title={compacting ? "Compacting…" : `Context ${pct}% used`} onClick={() => setOpen(o => !o)}>
        <CtxRing p={p} />{p >= 0.75 && <span className="cx-pc">{pct}%</span>}
      </button>
      {open && (
        <div className="cx-pop" role="dialog">
          <button className="cx-row cx-h" onClick={() => setMore(m => !m)}>
            <span>Context window</span>
            <span className="cx-v">{kfmt(used)} / {kfmt(CTX_MAX)} ({pct}%)</span>
            <span className={"cx-cv" + (more ? " on" : "")}><Ic n="chevR" s={12} w={2} /></span>
          </button>
          <div className="cx-bar">
            {CTX_PARTS.map(([k, , c]) => ctx.parts[k] > 0 && <i key={k} className={c} style={{ width: (ctx.parts[k] / CTX_MAX * 100) + "%" }}></i>)}
            <b style={{ left: CTX_AUTO_AT * 100 + "%" }} title="Auto-compacts here"></b>
          </div>
          {more && <div className="cx-leg">{CTX_PARTS.map(([k, l, c]) => <div key={k}><i className={c}></i><span>{l}</span><span className="cx-v">{kfmt(ctx.parts[k])}</span></div>)}</div>}
          <div className="cx-cmp">
            <div className="cx-ct">
              <b>Compact conversation</b>
              <span>{frees > 4000 ? <>Summarizes older turns and frees about <em>{kfmt(frees)}</em>. The shared page and pending approvals stay in full.</> : "Nothing to compact yet."}</span>
            </div>
            <button className="cx-go" disabled={compacting || frees <= 4000} onClick={() => { setOpen(false); onCompact(false); }}>{compacting ? "Compacting…" : "Compact"}</button>
          </div>
          <label className="cx-auto">
            <span>Compact automatically at {Math.round(CTX_AUTO_AT * 100)}%</span>
            <button role="switch" aria-checked={ctx.auto} className={"cx-sw" + (ctx.auto ? " on" : "")} onClick={() => ctx.setAuto(!ctx.auto)}><i></i></button>
          </label>
          <div className="cx-sep"></div>
          <div className="cx-row cx-sub"><span>Usage limits</span><Ic n="ext" s={12} /></div>
          {USAGE.map(u => (
            <div className="cx-u" key={u.t}>
              <div className="cx-row"><b>{u.t}</b><span className="cx-r">{u.r}</span><span className="cx-v">{u.p}%</span></div>
              <div className="cx-bar thin"><i className={u.p >= 90 ? "c-hot" : u.p >= 75 ? "c-warm" : "c1"} style={{ width: u.p + "%" }}></i></div>
            </div>
          ))}
          <div className="cx-f"><button className="cx-fb">See detailed breakdown</button></div>
        </div>
      )}
    </span>
  );
}

function CtxDrain({ from, to, ms }) {
  const [v, setV] = React.useState(from);
  React.useEffect(() => {
    let raf, t0 = performance.now();
    const ease = x => 1 - Math.pow(1 - x, 3);
    const step = now => { const k = Math.min(1, (now - t0) / ms); setV(from + (to - from) * ease(k)); if (k < 1) raf = requestAnimationFrame(step); };
    raf = requestAnimationFrame(step); return () => cancelAnimationFrame(raf);
  }, [from, to, ms]);
  return <span className="cx-drain"><CtxRing p={v / CTX_MAX} s={14} /><span className="cx-dn">{kfmt(Math.round(v / 100) * 100)}</span></span>;
}

function CompactMark({ it }) {
  const [open, setOpen] = React.useState(false);
  return (
    <div className={"cmk" + (open ? " open" : "") + (it.live ? " live" : "")}>
      <div className="cmk-l">
        <span className="cmk-rule"></span>
        <button className="cmk-p" onClick={() => !it.live && setOpen(o => !o)} disabled={it.live}>
          <span className="cmk-ic"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">{CMP_PATH}</svg></span>
          {it.live ? <span className="shim">Compacting the conversation…</span> : <>
            <span>{it.auto ? "Auto-compacted" : "Compacted"} · {it.turns} earlier messages summarized</span>
            <span className="cmk-n">{kfmt(it.before)} → {kfmt(it.after)}</span>
            <span className="cmk-cv"><Ic n="chevR" s={11} w={2} /></span>
          </>}
        </button>
        <span className="cmk-rule"></span>
      </div>
      {open && (
        <div className="cmk-sum">
          <div className="cmk-h">What the agent carries forward</div>
          <ul>
            <li>You're working the billing queue: 10 items, 4 blocked on paperwork.</li>
            <li>Ridgeline's PRO 40102 was invoiced at <b>$1,180</b>; the signed rate con says <b>$1,240</b>.</li>
            <li>Halvorsen is waiting on a signed POD and Cascade on detention approval.</li>
          </ul>
          <div className="cmk-h">Kept in full</div>
          <div className="cmk-keep"><span><Ic n="eye" s={11} />Shared page</span><span><Ic n="inbox" s={11} />Pending approvals</span></div>
        </div>
      )}
    </div>
  );
}

Object.assign(window, { CtxDrain, kfmt, CMP_PATH, ContextMeter, CompactMark, CTX0, CTX_FRESH, CTX_MAX, CTX_AUTO_AT, ctxTotal });
