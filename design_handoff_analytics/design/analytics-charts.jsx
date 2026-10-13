const AnCtx = React.createContext({ hl: null, setHl: () => {} });
function useAnW() {
  const ref = React.useRef(); const [w, setW] = React.useState(0);
  React.useLayoutEffect(() => { const el = ref.current; if (!el) return; setW(el.clientWidth); const ro = new ResizeObserver(([e]) => setW(Math.round(e.contentRect.width))); ro.observe(el); return () => ro.disconnect(); }, []);
  return [ref, w];
}
function useAnCount(to, ms = 900) {
  const [v, setV] = React.useState(0); const from = React.useRef(0);
  React.useEffect(() => { const a = from.current, t0 = performance.now(); let raf; const tick = now => { const k = Math.min(1, (now - t0) / ms), e = 1 - Math.pow(1 - k, 3); setV(a + (to - a) * e); if (k < 1) raf = requestAnimationFrame(tick); else from.current = to; }; raf = requestAnimationFrame(tick); return () => cancelAnimationFrame(raf); }, [to]);
  return v;
}
function AnNum({ v, f }) { const x = useAnCount(v); return <>{f(x)}</>; }
const anOneIn = r => r <= 0 ? "none" : "1 in " + Math.max(2, Math.round(1 / r));
function anSay(k, b, w) {
  const B = <b>{b.g.short}</b>, W = w && w !== b ? <b>{w.g.short}</b> : null, f = AN_MET[k].f;
  if (k === "c") return <>{B} finishes {Math.round(b.v * 100)} of every 100 runs{W && <>; {W} finishes {Math.round(w.v * 100)}</>}.</>;
  if (k === "med") return <>{B} usually finishes in {f(b.v)}{W && <>; {W} takes {f(w.v)}</>}.</>;
  if (k === "tf") return <>{B} gets {anOneIn(b.v)} tool calls wrong{W && <>; {W} gets {anOneIn(w.v)} wrong</>}.</>;
  if (k === "cost") return b.v === 0 ? <>{B} runs on your own hardware, so a run costs nothing{W && <>; {W} costs {f(w.v)}</>}.</> : <>{B} costs {f(b.v)} per finished run{W && <>; {W} costs {f(w.v)}</>}.</>;
  if (k === "ap") return <>People accept {Math.round(b.v * 100)}% of {B}'s proposals as written{W && <>, {Math.round(w.v * 100)}% of {W}'s</>}.</>;
  if (k === "ttft") return <>{B} starts answering in {f(b.v)}{W && <>; {W} in {f(w.v)}</>}.</>;
  return null;
}
const anDim = (hl, k) => hl ? (hl === k ? " is-hot" : " is-dim") : "";
function AnMarks({ provs, s = 22 }) { return <span className="an-mk">{provs.map(p => <Mark key={p.id} p={p} s={s}></Mark>)}</span>; }
function AnSw({ g }) { return <i className="an-sw" style={{ "--col": g.col }}></i>; }
function AnCard({ t, d, r, cls = "", children }) {
  return <section className={"an-card " + cls}><header className="an-ch"><div className="an-ct"><h3>{t}</h3>{d && <p>{d}</p>}</div>{r && <div className="an-cr">{r}</div>}</header>{children}</section>;
}
function AnBetter({ hi }) { return <span className="an-bt"><Ic n={hi ? "up" : "down"} s={10} w={2.4}></Ic>{hi ? "Higher is better" : "Lower is better"}</span>; }
function AnTip({ x, y, w = 220, box, children, flip }) {
  const left = box ? Math.max(w / 2 + 4, Math.min(box - w / 2 - 4, x)) : x;
  return <div className={"an-tip" + (flip ? " is-flip" : "")} style={{ left, top: y, "--tw": w + "px" }}>{children}</div>;
}
const anTicks = (mx, n = 5) => { const raw = mx / n, p = Math.pow(10, Math.floor(Math.log10(raw || 1))), s = [1, 2, 2.5, 5, 10].map(k => k * p).find(k => k >= raw) || p * 10, top = Math.ceil(mx / s) * s; const out = []; for (let v = 0; v <= top + s / 1e6; v += s) out.push(+v.toFixed(6)); return { top, out }; };

function AnField({ groups, hidden, setHidden, by }) {
  const { hl, setHl } = React.useContext(AnCtx);
  return (
    <div className="an-field">
      <span className="an-fl">Comparing <b>{groups.length - hidden.length}</b> {by === "model" ? "models" : "providers"}</span>
      {groups.map(g => { const off = hidden.includes(g.key); return <button key={g.key} className={"an-fc" + (off ? " off" : "") + anDim(hl, g.key)} onMouseEnter={() => !off && setHl(g.key)} onMouseLeave={() => setHl(null)} onClick={() => setHidden(h => off ? h.filter(x => x !== g.key) : h.length >= groups.length - 1 ? h : [...h, g.key])} title={off ? "Show in charts" : "Hide from charts"}>
        <AnSw g={g}></AnSw><b>{g.label}</b><span className="mono">{g.m.runs.toLocaleString()}</span>
      </button>; })}
      {hidden.length > 0 && <button className="lnk" onClick={() => setHidden([])}>Show all</button>}
    </div>
  );
}

function AnBoard({ k, groups, onOpen }) {
  const M = AN_MET[k], { hl, setHl } = React.useContext(AnCtx);
  const xs = groups.map(g => ({ g, v: M.v(g.m), few: g.m.runs < 30 })).filter(x => x.v != null).sort((a, b) => (a.few - b.few) || (M.hi ? b.v - a.v : a.v - b.v));
  const mx = Math.max(...xs.map(x => x.v)) || 1;
  const rk = xs.filter(x => !x.few), b = rk[0], wv = rk[rk.length - 1];
  return (
    <AnCard t={M.l} d={M.d} r={<AnBetter hi={M.hi}></AnBetter>} cls="an-bd">
      {b && <p className="an-say">{anSay(k, b, wv)}</p>}
      <div className="an-vb">
        {xs.map((x, i) => <button key={x.g.key} className={"an-vc" + anDim(hl, x.g.key) + (x.few ? " few" : "") + (i === 0 && !x.few && xs.length > 1 ? " is-best" : "")} onMouseEnter={() => setHl(x.g.key)} onMouseLeave={() => setHl(null)} onClick={() => onOpen(x.g)} title={x.g.label + " · " + M.f(x.v) + (x.few ? " · too few runs to rank" : "")}>
          <span className="an-vv mono">{M.f(x.v)}</span>
          <span className="an-vt"><i style={{ "--col": x.g.col, height: (x.v === 0 ? 1.5 : Math.max(3, x.v / mx * 100)) + "%", animationDelay: i * 50 + "ms" }}></i></span>
          <span className="an-vl"><AnMarks provs={x.g.provs} s={16}></AnMarks><em>{x.g.short}</em></span>
        </button>)}
      </div>
    </AnCard>
  );
}

function AnScatter({ groups, onOpen }) {
  const [xk, setXk] = React.useState("med"), [yk, setYk] = React.useState("c"), [tip, setTip] = React.useState(null);
  const [ref, w] = useAnW(), { hl, setHl } = React.useContext(AnCtx);
  const X = AN_MET[xk], Y = AN_MET[yk];
  const pts = groups.map(g => ({ g, x: X.v(g.m), y: Y.v(g.m) })).filter(p => p.y != null);
  const H = 340, P = { l: 48, r: 24, t: 16, b: 40 }, iw = Math.max(10, w - P.l - P.r), ih = H - P.t - P.b;
  const xt = anTicks(Math.max(...pts.map(p => p.x)) * 1.1 || 1);
  const ylo = Y.pct ? Math.max(0, Math.floor((Math.min(...pts.map(p => p.y)) - 0.03) * 20) / 20) : 0, yhi = Y.pct ? 1 : anTicks(Math.max(...pts.map(p => p.y)) * 1.1).top;
  const yt = []; for (let v = ylo; v <= yhi + 1e-6; v += Y.pct ? 0.05 : (yhi - ylo) / 5) yt.push(+v.toFixed(4));
  const sx = v => P.l + v / xt.top * iw, sy = v => P.t + (1 - (v - ylo) / (yhi - ylo)) * ih;
  const med = a => { const s = [...a].sort((p, q) => p - q); return s[Math.floor(s.length / 2)]; };
  const qx = med(pts.map(p => p.x)), qy = med(pts.map(p => p.y)), mr = Math.max(...pts.map(p => p.g.m.runs));
  const qr = { x: X.hi ? sx(qx) : P.l, w: X.hi ? P.l + iw - sx(qx) : sx(qx) - P.l, y: Y.hi ? P.t : sy(qy), h: Y.hi ? sy(qy) - P.t : P.t + ih - sy(qy) };
  const xo = [["med", "Median run time"], ["cost", "Cost per completed run"], ["ttft", "Time to first token"], ["tf", "Tool calls that failed"]], yo = [["c", "Completed"], ["ap", "Approved as proposed"], ["rt", "Rated helpful"]];
  return (
    <AnCard t={Y.l + " against " + X.l.toLowerCase()} d="Every model on one chart. The shaded corner is where you want a model to be." cls="an-scw" r={<div className="an-ax2"><AnSel l="Up" v={yk} opts={yo} onChange={setYk} plain></AnSel><AnSel l="Across" v={xk} opts={xo} onChange={setXk} plain></AnSel></div>}>
      <div className="an-scp" ref={ref} onMouseLeave={() => { setTip(null); setHl(null); }}>
        {w > 0 && <svg width={w} height={H} className="an-svg">
          <rect x={qr.x} y={qr.y} width={Math.max(0, qr.w)} height={Math.max(0, qr.h)} className="an-qr" rx="4"></rect>
          <text x={X.hi ? P.l + iw - 8 : P.l + 8} y={Y.hi ? P.t + 16 : P.t + ih - 8} textAnchor={X.hi ? "end" : "start"} className="an-qt">Best here</text>
          {yt.map(v => <g key={v}><line x1={P.l} x2={P.l + iw} y1={sy(v)} y2={sy(v)} className="an-gl"></line><text x={P.l - 10} y={sy(v) + 3.5} textAnchor="end" className="an-tk">{Y.f(v)}</text></g>)}
          {xt.out.map(v => <text key={v} x={sx(v)} y={P.t + ih + 18} textAnchor="middle" className="an-tk">{v === 0 && xk === "cost" ? "Local" : X.f(v)}</text>)}
          <text x={P.l + iw} y={H - 4} textAnchor="end" className="an-al">{X.l} {X.hi ? "→" : "← better"}</text>
          <text x={P.l - 40} y={P.t - 4} className="an-al">{Y.l} {Y.hi ? "↑" : ""}</text>
          {(() => { const LB = pts.map(p => { const r = 5 + Math.sqrt(p.g.m.runs / mr) * 8, cx = sx(p.x), cy = sy(p.y), lw = p.g.label.length * 6.4 + 4; return { p, r, cx, cy, lw, lf: cx + r + 7 + lw > P.l + iw, ly: cy }; }).sort((a, b) => a.cy - b.cy);
            const box = l => ({ x0: l.lf ? l.cx - l.r - 7 - l.lw : l.cx + l.r + 7, x1: l.lf ? l.cx - l.r - 7 : l.cx + l.r + 7 + l.lw, y0: l.ly - 8, y1: l.ly + 8 });
            const hit = (a, b) => a.x0 < b.x1 && b.x0 < a.x1 && a.y0 < b.y1 && b.y0 < a.y1;
            const dots = LB.map(o => ({ x0: o.cx - o.r - 2, x1: o.cx + o.r + 2, y0: o.cy - o.r - 2, y1: o.cy + o.r + 2 }));
            const bad = (l, i) => LB.slice(0, i).some(o => hit(box(l), box(o))) || dots.some((d, k) => k !== i && hit(box(l), d));
            LB.forEach((l, i) => { if (!bad(l, i)) return; l.lf = !l.lf; if (!bad(l, i)) return; l.lf = !l.lf; for (let t = 0; t < 10 && bad(l, i); t++) l.ly += 9; });
            return LB.map(l => <g key={l.p.g.key} className={"an-pt" + anDim(hl, l.p.g.key)} style={{ "--col": l.p.g.col }} onMouseEnter={() => { setHl(l.p.g.key); setTip({ p: l.p, x: l.cx, y: l.cy - l.r - 8 }); }} onClick={() => onOpen(l.p.g)}>
              <circle cx={l.cx} cy={l.cy} r={l.r} className="an-dot"></circle>
              {Math.abs(l.ly - l.cy) > 6 && <line x1={l.lf ? l.cx - l.r - 1 : l.cx + l.r + 1} y1={l.cy} x2={l.lf ? l.cx - l.r - 6 : l.cx + l.r + 6} y2={l.ly} className="an-ld"></line>}
              <text x={l.lf ? l.cx - l.r - 7 : l.cx + l.r + 7} y={l.ly + 4} textAnchor={l.lf ? "end" : "start"} className="an-pl">{l.p.g.label}</text>
            </g>); })()}
        </svg>}
        {tip && <AnTip x={tip.x} y={Math.max(70, tip.y)} box={w}><div className="an-tth"><AnSw g={tip.p.g}></AnSw><b>{tip.p.g.label}</b></div><span className="an-ttr"><em>{Y.l}</em><b className="mono">{Y.f(tip.p.y)}</b></span><span className="an-ttr"><em>{X.l}</em><b className="mono">{X.f(tip.p.x)}</b></span><span className="an-ttr"><em>Runs</em><b className="mono">{tip.p.g.m.runs.toLocaleString()}</b></span></AnTip>}
      </div>
      <p className="an-cap">Bigger dots ran more. Click a dot for the full breakdown.</p>
    </AnCard>
  );
}

function AnSame({ deps, S, A }) {
  const by = {}; deps.forEach(d => (by[d.model] = by[d.model] || []).push(d));
  const xs = Object.values(by).find(a => a.length > 1 && a.filter(d => d.m.runs >= 30).length > 1);
  if (!xs) return null;
  const [a, b] = [...xs].filter(d => d.m.runs >= 30).sort((p, q) => q.m.out.c / q.m.runs - p.m.out.c / p.m.runs);
  const rows = ["c", "med", "tf", "ttft", "ap"].map(k => { const M = AN_MET[k], va = M.v(a.m), vb = M.v(b.m); if (va == null || vb == null) return null; const mx = Math.max(va, vb) || 1, aw = M.hi ? va >= vb : va <= vb; return { k, M, va, vb, mx, aw }; }).filter(Boolean);
  const wins = rows.filter(r => r.aw).length;
  const canMove = S.providers.find(p => p.id === a.pids[0]) && S.providers.find(p => p.id === b.pids[0]);
  return (
    <AnCard t="Same model, different provider" d={<>{a.sub} runs on both. Since the model is the same, every gap here comes from the connection.</>} cls="an-same">
      <div className="an-bfh"><span><Mark p={a.provs[0]} s={22}></Mark><b>{a.label}</b></span><em className="mono">vs</em><span><b>{b.label}</b><Mark p={b.provs[0]} s={22}></Mark></span></div>
      <div className="an-bf">{rows.map((r, i) => <div key={r.k} className="an-bfr">
        <span className={"an-bfv mono" + (r.aw ? " w" : "")}>{r.M.f(r.va)}</span>
        <span className="an-bfb l"><i className={r.aw ? "w" : ""} style={{ width: r.va / r.mx * 100 + "%", "--col": a.col, animationDelay: i * 60 + "ms" }}></i></span>
        <span className="an-bfl">{r.M.l}</span>
        <span className="an-bfb r"><i className={!r.aw ? "w" : ""} style={{ width: r.vb / r.mx * 100 + "%", "--col": b.col, animationDelay: i * 60 + "ms" }}></i></span>
        <span className={"an-bfv mono" + (!r.aw ? " w" : "")}>{r.M.f(r.vb)}</span>
      </div>)}</div>
      <footer className="an-bff"><span><b>{a.label}</b> is better on {wins} of {rows.length}</span>{canMove && <button className="btn sm" onClick={() => { A.moveTo(a.pids[0], b.pids[0]); A.toast(a.label + " now goes ahead of " + b.label); }}><Ic n="swap" s={12}></Ic>Put {a.label} first</button>}</footer>
    </AnCard>
  );
}

const AN_TICKS = [1, 2, 5, 10, 30, 60, 120];
const anX = s => Math.max(0, Math.min(1, Math.log(Math.max(1, s)) / Math.log(180)));
function AnRunTime({ groups, deps, onOpen }) {
  const [o, setO] = React.useState("c"), { hl, setHl } = React.useContext(AnCtx);
  const rows = [...groups].sort((p, q) => (q.m.out[o] > 0) - (p.m.out[o] > 0) || p.m.dur[o][1] - q.m.dur[o][1]);
  const rank = Object.fromEntries(rows.map((g, i) => [g.key, i])), RH = 46;
  const ins = deps.filter(d => d.m.out.f >= 10 && d.m.out.c >= 30).map(d => ({ d, r: d.m.dur.f[1] / d.m.dur.c[1] })).sort((p, q) => q.r - p.r)[0];
  const ol = Object.fromEntries(AN_OUT);
  return (
    <AnCard t="How long runs take" d="Each bar spans the middle half of runs. The line runs out to the slowest 5%, and the white tick marks the median." cls="an-rt" r={<Seg cls="sm" v={o} opts={AN_OUT.map(([k, l]) => [k, k === "w" ? "Handed off" : k === "s" ? "Stopped" : l])} onChange={setO}></Seg>}>
      {ins && ins.r >= 2 && <p className="an-ins"><Ic n="clock" s={12}></Ic><span>Failed runs on <b>{ins.d.label}</b> take <b>{ins.r.toFixed(1)}×</b> as long as its completed ones{ins.d.key === "wv" ? ". Most of them wait out the 60-second timeout first" : ""}.</span></p>}
      <div className="an-rth"><span></span><div className="an-rtax">{AN_TICKS.map(t => <em key={t} className="mono" style={{ left: anX(t) * 100 + "%" }}>{t >= 60 ? t / 60 + "m" : t + "s"}</em>)}</div><span className="r">Median</span><span>Outcome mix</span></div>
      <div className="an-rtb" style={{ height: rows.length * RH }}>
        {groups.map(g => { const q = g.m.dur[o], n = g.m.out[o]; return <div key={g.key} className={"an-rtr" + anDim(hl, g.key)} style={{ transform: `translateY(${rank[g.key] * RH}px)`, height: RH }} onMouseEnter={() => setHl(g.key)} onMouseLeave={() => setHl(null)} onClick={() => onOpen(g)}>
          <span className="an-rtn"><AnSw g={g}></AnSw><b>{g.label}</b></span>
          <div className="an-rtt">{AN_TICKS.map(t => <u key={t} style={{ left: anX(t) * 100 + "%" }}></u>)}
            {n > 0 ? <span className="an-rg" style={{ "--col": g.col }} title={`p25 ${anS(q[0])} · median ${anS(q[1])} · p75 ${anS(q[2])} · slowest 5% ${anS(q[3])} · ${n} runs`}>
              <i className="rg-w" style={{ left: anX(q[0]) * 100 + "%", width: (anX(q[3]) - anX(q[0])) * 100 + "%" }}></i>
              <i className="rg-b" style={{ left: anX(q[0]) * 100 + "%", width: Math.max(1, (anX(q[2]) - anX(q[0])) * 100) + "%" }}></i>
              <i className="rg-m" style={{ left: anX(q[1]) * 100 + "%" }}></i>
            </span> : <span className="an-none">No {ol[o].toLowerCase()} runs</span>}
          </div>
          <span className="an-rtv mono">{n > 0 ? anS(q[1]) : "—"}<em>{n > 0 ? n.toLocaleString() + " runs" : ""}</em></span>
          <span className="an-mix">{AN_OUT.map(([k]) => g.m.out[k] > 0 && <i key={k} className={"o-" + k + (k === o ? " on" : "")} style={{ flexGrow: g.m.out[k] }}></i>)}</span>
        </div>; })}
      </div>
      <div className="an-lgd">{AN_OUT.map(([k, l]) => <span key={k} className={k === o ? "on" : ""}><i className={"o-" + k}></i>{l}</span>)}</div>
    </AnCard>
  );
}

function AnTrend({ groups, rg, ag }) {
  const [k, setK] = React.useState("c"), [hi, setHi] = React.useState(null), [ref, w] = useAnW(), { hl, setHl } = React.useContext(AnCtx);
  const M = k === "c" ? AN_MET.c : k === "tf" ? AN_MET.tf : AN_MET.med;
  const ser = groups.map(g => ({ g, s: anSeries(g, rg, ag) })), n = ser[0] ? ser[0].s.length : 0, [, span] = AN_BUCKETS[rg];
  const vals = ser.flatMap(x => x.s.filter(Boolean).map(p => p[k]));
  const ev0 = AN_EVENTS.filter(e => e.ago <= (n - 1) * span && groups.some(g => g.pids.includes(e.pid))).sort((a, b) => b.ago - a.ago);
  const iw0 = Math.max(10, w - 64), evx = e => 48 + (n > 1 ? ((n - 1) - e.ago / span) / (n - 1) : 0) * iw0;
  let lastX = -1e9, row = -1; const ev = ev0.map(e => { const x = evx(e); row = x - lastX < 130 ? row + 1 : 0; lastX = x; return { ...e, row }; });
  const rows = ev.length ? Math.max(...ev.map(e => e.row)) + 1 : 0;
  const H = 260 + rows * 14, P = { l: 48, r: 16, t: 18 + rows * 14, b: 28 }, iw = Math.max(10, w - P.l - P.r), ih = H - P.t - P.b;
  const lo = M.pct && k === "c" ? Math.max(0, Math.floor((Math.min(...vals) - 0.02) * 10) / 10) : 0, top = M.pct && k === "c" ? 1 : anTicks(Math.max(...vals) * 1.08, 4).top;
  const yt = k === "c" ? Array.from({ length: Math.round((top - lo) / 0.1) + 1 }, (_, i) => +(lo + i * 0.1).toFixed(2)) : anTicks(top, 4).out;
  const sx = i => P.l + (n > 1 ? i / (n - 1) : 0) * iw, sy = v => P.t + (1 - (v - lo) / (top - lo)) * ih;
  const path = s => { let d = "", pen = false; s.forEach((p, i) => { if (!p) { pen = false; return; } d += (pen ? "L" : "M") + sx(i).toFixed(1) + "," + sy(p[k]).toFixed(1); pen = true; }); return d; };
  const agoL = i => { const ago = (n - 1 - i) * span; if (rg === "d1") return ago === 0 ? "Now" : Math.round(ago * 24) + "h ago"; const dt = new Date(2026, 9, 9 - Math.floor(ago)); return ago === 0 ? "Today" : dt.toLocaleDateString("en-US", { month: "short", day: "numeric" }) + (rg === "d7" ? " " + ["12a", "6a", "12p", "6p"][3 - Math.round((ago % 1) * 4) % 4] : ""); };
  const xl = Array.from({ length: n }, (_, i) => i).filter(i => i % Math.ceil(n / 6) === (n - 1) % Math.ceil(n / 6));
  const move = e => { const r = e.currentTarget.getBoundingClientRect(), x = e.clientX - r.left; const i = Math.round((x - P.l) / iw * (n - 1)); setHi(i >= 0 && i < n ? i : null); };
  return (
    <AnCard t="Over time" d="Catch a model or provider getting worse after a change." cls="an-tr" r={<Seg cls="sm" v={k} opts={[["c", "Completed"], ["tf", "Tool calls failed"], ["med", "Median run time"]]} onChange={setK}></Seg>}>
      <div className="an-trp" ref={ref} onMouseMove={move} onMouseLeave={() => setHi(null)}>
        {w > 0 && n > 0 && <svg width={w} height={H} className="an-svg">
          {yt.map(v => <g key={v}><line x1={P.l} x2={P.l + iw} y1={sy(v)} y2={sy(v)} className="an-gl"></line><text x={P.l - 10} y={sy(v) + 3.5} textAnchor="end" className="an-tk">{M.f(v)}</text></g>)}
          {xl.map(i => <text key={i} x={sx(i)} y={H - 6} textAnchor="middle" className="an-tk">{agoL(i)}</text>)}
          {ev.map(e => { const x = sx((n - 1) - e.ago / span), ly = 12 + e.row * 14, end = x > P.l + iw - 140; return <g key={e.l} className="an-ev"><line x1={x} x2={x} y1={ly + 4} y2={P.t + ih}></line><circle cx={x} cy={ly} r="3.5"></circle><text x={end ? x - 7 : x + 7} y={ly + 3.5} textAnchor={end ? "end" : "start"}>{e.l}</text></g>; })}
          {ser.map(({ g, s }) => <path key={g.key + k + rg} d={path(s)} pathLength="1" className={"an-ln" + anDim(hl, g.key)} style={{ "--col": g.col }} onMouseEnter={() => setHl(g.key)} onMouseLeave={() => setHl(null)}></path>)}
          {hi != null && <g><line x1={sx(hi)} x2={sx(hi)} y1={P.t} y2={P.t + ih} className="an-xh"></line>{ser.map(({ g, s }) => s[hi] && <circle key={g.key} cx={sx(hi)} cy={sy(s[hi][k])} r="4" className="an-xd" style={{ "--col": g.col }}></circle>)}</g>}
        </svg>}
        {hi != null && <AnTip x={sx(hi)} y={P.t} flip={sx(hi) > w - 240}><div className="an-tth"><b>{agoL(hi)}</b></div>{ser.filter(x => x.s[hi]).sort((p, q) => (M.hi ? q.s[hi][k] - p.s[hi][k] : p.s[hi][k] - q.s[hi][k])).map(({ g, s }) => <span key={g.key} className={"an-ttr" + anDim(hl, g.key)}><em><AnSw g={g}></AnSw>{g.label}</em><b className="mono">{M.f(s[hi][k])}</b></span>)}</AnTip>}
      </div>
    </AnCard>
  );
}

function AnWhose({ groups }) {
  const { hl, setHl } = React.useContext(AnCtx);
  const rows = groups.map(g => { const c = g.m.calls || 1; const m = AN_WHY.filter(w => w[2] === "m").map(([r]) => [r, g.m.tf[r] / c]), s = AN_WHY.filter(w => w[2] === "s").map(([r]) => [r, g.m.tf[r] / c]); return { g, m, s, mt: m.reduce((t, x) => t + x[1], 0), st: s.reduce((t, x) => t + x[1], 0) }; }).sort((a, b) => b.mt - a.mt);
  const mx = Math.max(...rows.map(r => Math.max(r.mt, r.st))) || 1;
  return (
    <AnCard t="Whose mistake was it" d="Failed tool calls split by cause, as a share of all calls." cls="an-wh">
      <div className="an-whh"><span>The model</span><span></span><span>The tool or a rule</span></div>
      {rows.map((r, i) => <div key={r.g.key} className={"an-whr" + anDim(hl, r.g.key)} onMouseEnter={() => setHl(r.g.key)} onMouseLeave={() => setHl(null)}>
        <span className="an-whv mono">{anP(r.mt, 1)}</span>
        <span className="an-whb l">{r.m.map(([k, v]) => v > 0 && <i key={k} className={"r-" + k} style={{ width: v / mx * 100 + "%", animationDelay: i * 50 + "ms" }} title={AN_WHY.find(w => w[0] === k)[1] + ": " + anP(v, 2)}></i>)}</span>
        <span className="an-whn"><AnSw g={r.g}></AnSw>{r.g.short}</span>
        <span className="an-whb r">{r.s.map(([k, v]) => v > 0 && <i key={k} className={"r-" + k} style={{ width: v / mx * 100 + "%", animationDelay: i * 50 + "ms" }} title={AN_WHY.find(w => w[0] === k)[1] + ": " + anP(v, 2)}></i>)}</span>
        <span className="an-whv mono r">{anP(r.st, 1)}</span>
      </div>)}
      <div className="an-wk"><div>{AN_WHY.filter(w => w[2] === "m").map(([r, l]) => <span key={r}><i className={"r-" + r}></i>{l}</span>)}</div><div>{AN_WHY.filter(w => w[2] === "s").map(([r, l]) => <span key={r}><i className={"r-" + r}></i>{l}</span>)}</div></div>
    </AnCard>
  );
}
function AnSel({ l, v, opts, onChange, plain }) {
  return <label className={"an-sel" + (!plain && v !== "all" ? " on" : "")}><span>{l}</span><select value={v} onChange={e => onChange(e.target.value)}>{opts.map(([k, t]) => <option key={k} value={k}>{t}</option>)}</select><Ic n="chevD" s={11}></Ic></label>;
}
Object.assign(window, { AnCtx, AnNum, AnMarks, AnSw, AnCard, AnField, AnBoard, AnScatter, AnSame, AnRunTime, AnTrend, AnWhose, AnSel, anDim });
