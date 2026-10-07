function tuneUps(S, A) {
  const xs = [], ag = id => S.agents.find(a => a.id === id), pv = id => S.providers.find(p => p.id === id);
  const d = ag("dispatch");
  if (d && d.on && d.rec) xs.push({ id: "earn", ic: "award", tone: "k", a: d, t: <>Let <b>{d.rec.tool}</b> run on its own for {d.n}</>, why: "Approved unchanged " + d.rec.streak + " times in a row · no rejections in 30 days", gain: "~2 h a week of approvals", act: S.pol.earned ? "Raise it" : "Raise it", go: () => A.toast(d.rec.tool + " now runs on its own for " + d.n) });
  const v = pv("vllm"), o = pv("ollama");
  if (v && o && v.on && S.providers.indexOf(v) < S.providers.indexOf(o) && v.last && !v.last.ok) xs.push({ id: "order", ic: "swap", tone: "w", p: o, t: <>Put <b>{o.n}</b> ahead of {v.n}</>, why: v.n + " failed 24 times overnight; " + o.n + " answered every call it got", gain: "Fewer retries on 5 tasks", act: "Reorder", go: () => { A.moveTo(o.id, v.id); A.toast(o.n + " now goes first"); } });
  const c = ag("coverage");
  if (c && c.on && c.shadow) xs.push({ id: "live", ic: "eyeOff", tone: "b", a: c, t: <>Take <b>{c.n}</b> out of shadow</>, why: "31 recorded proposals · 87% match what dispatchers did · none would have failed", gain: "Proposals start reaching Desk", act: "Go live", go: () => { A.setMode(c.id, "live"); A.toast(c.n + " is live"); } });
  const emb = TASKS.find(t => t.k === "Embedding");
  if (S.providers.length && !routeOf(emb, S.providers).first && o) xs.push({ id: "emb", ic: "search", tone: "w", p: o, t: <>Give <b>Embedding</b> to {o.n}</>, why: "Nothing handles it, so search matches words only. " + o.n + " already serves nomic-embed-text", gain: "Search by meaning across 2,160 records", act: "Assign", go: () => { A.toggleTask(o.id, "Embedding"); A.toast("Embedding now goes to " + o.n + " · indexing starts"); } });
  const m = ag("mds");
  if (m && m.on) xs.push({ id: "idle", ic: "clock", tone: "n", a: m, t: <><b>{m.n}</b> hasn't run in 14 days</>, why: "Nobody has asked it anything since Sep 23 · it holds 55 tools", gain: "Fewer agents to choose from in Desk", act: "Turn off", go: () => A.toggleAgent(m.id, false) });
  return xs;
}
function TuneUps({ S, A }) {
  const [hid, setHid] = React.useState([]); const [gone, setGone] = React.useState(null);
  const all = tuneUps(S, A).filter(x => !hid.includes(x.id));
  const done = (x, apply) => { setGone(x.id); setTimeout(() => { setHid(h => [...h, x.id]); setGone(null); if (apply) x.go(); else A.toast("Dismissed · it won't be suggested again for 30 days", () => setHid(h => h.filter(y => y !== x.id))); }, 260); };
  return (
    <section className="sec" id="tune">
      <SecH t="Tune-ups" n={all.length || null} r={<span className="sh2-n">From the last 30 days of runs</span>}></SecH>
      {!all.length ? <div className="clr"><span className="clr-i"><Ic n="check" s={16} w={2.4}></Ic></span><div><b>Nothing to tune</b><span>Routing, autonomy and the roster all look right for how your agents are being used.</span></div></div>
        : <div className="nds">{all.map(x => <div key={x.id} className={"nd-w" + (gone === x.id ? " out" : "")}><div className="nd tu">
          {x.a ? <Tile a={x.a} s={30}></Tile> : x.p ? <Mark p={x.p} s={30}></Mark> : null}
          <div className="nd-b"><b className="tu-t"><Ic n={x.ic} s={12}></Ic>{x.t}</b><span>{x.why}</span></div>
          <span className={"tu-g " + x.tone}>{x.gain}</span>
          <div className="nd-a"><button className="ib" title="Dismiss" onClick={() => done(x, false)}><Ic n="x" s={13}></Ic></button><button className="btn sm" onClick={() => done(x, true)}>{x.act}</button></div>
        </div></div>)}</div>}
    </section>
  );
}
function OvFigs({ S, A }) {
  const tot = WEEK.reduce((t, d) => t + d[1] + d[2], 0), f = WEEK.reduce((t, d) => t + d[2], 0), max = Math.max(...WEEK.map(d => d[1] + d[2]));
  const bars = <span className="fbars">{WEEK.map(([d, ok, x], i) => <i key={d} title={d + ": " + (ok + x) + " calls, " + x + " failed"} className={i === WEEK.length - 1 ? "now" : ""} style={{ height: 3 + (ok + x) / max * 17 }}>{x > 0 && <u style={{ height: Math.max(20, x / (ok + x) * 100) + "%" }}></u>}</i>)}</span>;
  const priced = S.providers.some(p => p.pin);
  return <Figs items={[
    ["Model calls · 7 days", tot.toLocaleString(), <span className="fsub">{bars}<span><span className="t-d">{f} failed</span> · {(100 - f / tot * 100).toFixed(1)}% ok</span></span>],
    ["Median response", <>0.84<small>s</small></>, "Slowest 5% took 4.2s or more"],
    ["Tokens", <>3.1<small>M</small></>, "2.6M in · 0.5M out"],
    ["Spend", priced ? "$41.20" : <button className="fset" onClick={() => A.edit({ k: "prov", id: (S.providers.find(p => p.on) || S.providers[0]).id })}><Ic n="plus" s={12}></Ic>Set prices</button>, priced ? "Across every provider" : "No provider has a price yet", priced ? "" : "dim"],
  ]}></Figs>;
}
function Usage() {
  const max = Math.max(...FEATURES.map(f => f[1]));
  const rows = FEATURES.map(([l, c, f, tok, ms]) => ({ id: l, l, c, f, tok, ms }));
  return (
    <section className="sec">
      <SecH t="Usage by feature" n="7 days"></SecH>
      <DT rows={rows} name="feature" ph="Search features" per={10} find={r => r.l}
        filters={[["all", "All", () => true], ["failed", "With failures", r => r.f > 0]]}
        cols={[{ k: "l", l: "Feature", w: "30%", c: r => <b className="rg">{r.l}</b>, sv: r => r.l },
          { k: "c", l: "Calls", c: r => <span className="ucl"><span className="mono">{r.c}</span><span className="ubar"><i style={{ width: (r.c / max * 100) + "%" }}></i>{r.f > 0 && <i className="f" style={{ width: Math.max(3, r.f / max * 100) + "%" }}></i>}</span></span>, sv: r => r.c },
          { k: "f", l: "Failed", r: true, c: r => r.f ? <span className="mono t-d">{r.f}</span> : <span className="dim">—</span>, sv: r => r.f },
          { k: "tok", l: "Tokens", r: true, c: r => <span className="mono">{r.tok}</span>, sv: r => parseFloat(r.tok) },
          { k: "sp", l: "Spend", r: true, hide: 2, c: () => <span className="dim">—</span> },
          { k: "ms", l: "Median", r: true, c: r => <span className="mono">{r.ms >= 1000 ? (r.ms / 1000).toFixed(1) + "s" : r.ms + "ms"}</span>, sv: r => r.ms }]}></DT>
    </section>
  );
}
function WorkingNow({ S, A }) {
  const live = S.agents.filter(a => liveOf(a, S));
  const on = S.agents.filter(a => a.on), sh = on.filter(a => a.shadow).length, off = S.agents.length - on.length;
  return (
    <section className="sec">
      <SecH t="Agents" n={on.length + " of " + S.agents.length + " on"} r={<button className="lnk" onClick={() => A.go("agents")}>Roster<Ic n="arrowR" s={11}></Ic></button>}></SecH>
      {live.length > 0 ? <div className="wn">{live.map(a => <button key={a.id} className="wn-r" onClick={() => A.go("agents", a.id)}><span className="pc-mk"><Tile a={a} s={26}></Tile><i className="pc-dot run sm"></i></span><span className="wn-t"><b>{a.n}</b><span className="shm">{liveOf(a, S)}…</span></span></button>)}</div>
        : <p className="wn-e">{!S.providers.length ? "Nothing can run until a provider is connected." : S.paused ? "Paused. Nothing is running right now." : "Nothing is running right now."}</p>}
      <div className="ms">{S.agents.map(a => <button key={a.id} className={"ms-i" + (a.on ? "" : " off") + (a.shadow ? " sh" : "")} title={a.n + (a.on ? a.shadow ? " · shadow" : "" : " · off")} onClick={() => A.go("agents", a.id)}><Tile a={a} s={28}></Tile>{a.on && a.pend > 0 && <i className="ms-b mono">{a.pend}</i>}</button>)}</div>
      <div className="lgd"><span><i className="k"></i>{on.length - sh} live</span><span><i className="s"></i>{sh} shadow</span><span><i className="o"></i>{off} off</span></div>
    </section>
  );
}
function OrgState({ S, A }) {
  const cov = TASKS.filter(t => routeOf(t, S.providers).first).length;
  const rows = [["award", "Earned autonomy", S.pol.earned ? "On · after " + S.pol.threshold + " clean" : "Off", S.pol.earned], ["brain", "Learn from their work", S.pol.learn ? "On" : "Off", S.pol.learn], ["gauge", "Monthly allowance", S.pol.allowance ? S.pol.allowance.toLocaleString() + " per person" : "Unlimited", !!S.pol.allowance], ["database", "Share corrections", S.pol.share ? "On" : "Off", S.pol.share]];
  return (
    <section className="sec">
      <SecH t="Organization-wide" r={<button className="btn sm" onClick={() => A.edit({ k: "policy" })}><Ic n="edit" s={12}></Ic>Edit</button>}></SecH>
      <div className="os">{rows.map(([ic, l, v, on]) => <button key={l} className="os-r" onClick={() => A.edit({ k: "policy" })}><Ic n={ic} s={13}></Ic><span>{l}</span><em className={on ? "on" : ""}>{v}</em></button>)}
        {S.providers.length > 0 && <button className="os-r" onClick={() => A.go("providers", null, "routing")}><Ic n="route" s={13}></Ic><span>Routing</span><em className={cov === TASKS.length ? "on" : "w"}>{cov} of {TASKS.length} covered</em></button>}
      </div>
    </section>
  );
}
const POL_IMPACT = th => AGENTS0.filter(a => a.rec && a.rec.streak >= th);
function PolicyEditor({ S, A, onClose, demo }) {
  const d = useDraft({ ...S.pol }); const v = d.v, set = d.set;
  React.useEffect(() => { if (demo === "unsaved") d.patch({ earned: true, threshold: 5 }); }, []);
  const up = v.earned ? POL_IMPACT(v.threshold) : [];
  const sections = POLICIES.map(p => ({ id: p.k, l: p.t, keys: [p.k, p.opt && p.opt.k].filter(Boolean), r: !p.noSwitch && <Switch on={v[p.k]} label={p.t} onChange={x => set(p.k, x)}></Switch>, body: <>
    <p className="es-note">{p.s} {p.more}</p>
    {p.opt && (p.noSwitch || v[p.k]) && <F l={p.opt.l}><Seg v={v[p.opt.k]} opts={p.opt.v.map(x => [x, x === 0 ? "Unlimited" : x.toLocaleString()])} onChange={x => set(p.opt.k, x)}></Seg></F>}
    {p.k === "earned" && v.earned && (up.length ? <Callout tone="i">{up.map(a => a.rec.tool + " on " + a.n).join(" and ")} {up.length > 1 ? "have" : "has"} enough clean approvals already, so {up.length > 1 ? "they move" : "it moves"} up a tier when you save.</Callout> : <Callout tone="k">No tool has a long enough streak yet. Nothing changes when you save.</Callout>)}
    {p.k === "learn" && !v.learn && d.base.learn && <Callout tone="w">Every agent stops keeping lessons, whatever its own switch says. 6 memories already kept stay.</Callout>}
    {p.foot && <p className="es-note">{p.foot}</p>}
  </> }));
  const fields = { earned: { l: "Earned autonomy" }, threshold: { l: "Clean approvals" }, learn: { l: "Learning" }, allowance: { l: "Allowance", fmt: x => x ? x.toLocaleString() : "Unlimited" }, share: { l: "Share corrections" } };
  return <EditSheet d={d} fields={fields} sections={sections} icon={<span className="src-i"><Ic n="cog" s={15}></Ic></span>} title="Organization-wide" sub="Applies to every agent and overrides their own settings" onSave={x => { A.setPolAll(x); A.toast("Saved" + (up.length ? " · " + up.length + " tool" + (up.length > 1 ? "s" : "") + " moved up a tier" : "")); return false; }} onClose={onClose}></EditSheet>;
}
function Setup({ A }) {
  return (
    <section className="setup">
      <ol className="steps">
        <li className="cur"><span className="st-n">1</span><div><b>Connect a model provider</b><p>A hosted model with an API key, or one running on your own network.</p><div className="pre">{PRESETS.map(p => <button key={p.k} className="pre-b" onClick={() => A.addPreset(p)}><Mark p={p} s={20}></Mark>{p.n}</button>)}</div></div></li>
        <li><span className="st-n">2</span><div><b>Choose what it handles</b><p>Each task goes to the first provider in order that takes it.</p></div></li>
        <li><span className="st-n">3</span><div><b>Agents start on their own</b><p>All {AGENTS0.length} are set up already. Nothing else to switch on.</p></div></li>
      </ol>
    </section>
  );
}
function Overview({ S, A }) {
  const none = !S.providers.length;
  const [hideBad, setHideBad] = React.useState([]);
  const on = S.agents.filter(a => a.on).length, pend = S.agents.reduce((t, a) => t + (a.on ? a.pend : 0), 0), live = S.agents.filter(a => liveOf(a, S)).length;
  const bad = S.providers.find(p => p.on && (S.tests[p.id] ? S.tests[p.id].state === "fail" : p.last && !p.last.ok));
  const unc = TASKS.filter(t => !routeOf(t, S.providers).first).length;
  let say, ctl;
  if (none) { say = <>Nothing can answer yet. Your <b>{S.agents.length} agents</b> are set up and waiting for a model provider — connect one and they start on their own.</>; ctl = <><button className="btn ink lg" onClick={() => A.go("providers")}><Ic n="plus" s={13}></Ic>Connect a provider</button><span>Takes about a minute</span></>; }
  else if (S.paused) { say = <>Every agent is <b className="t-w">paused</b>. They keep running and recording what they would do, but nothing is offered or executed — <span className="ref" onClick={() => A.toast("Opening Watchtower")}>{pend} proposals</span> are held until you resume.</>; ctl = <><button className="btn ink lg" onClick={() => A.setPaused(false)}><Ic n="play" s={12}></Ic>Resume agents</button><span>Paused by you · just now</span></>; }
  else { say = <><b>{on} agents</b> are on{live ? <>, <b>{live}</b> working right now</> : ""}. <span className="ref" onClick={() => A.toast("Opening Watchtower")}>{pend} proposals</span> wait on a person in Watchtower{bad ? <>, and <span className="ref d" onClick={() => A.go("providers", bad.id)}>{bad.n}</span> can't connect</> : ""}.{unc ? <> <span className="ref w" onClick={() => A.go("providers", null, "routing")}>{unc} task{unc > 1 ? "s have" : " has"}</span> nowhere to go.</> : ""}</>; ctl = <><Hold label="Hold to pause all agents" onDone={() => A.setPaused(true)}></Hold><span>They keep running in shadow</span></>; }
  const t = bad && S.tests[bad.id];
  return (
    <div className="tabp ov2">
      <Nova ctx="AI control" spin={live > 0 && !S.paused && !none} ctl={ctl}>{say}</Nova>
      {none ? <Setup A={A}></Setup> : <OvFigs S={S} A={A}></OvFigs>}
      {!none && bad && !hideBad.includes(bad.id) && <div className="ovb" role="status">
        <span className="ovb-i"><Mark p={bad} s={28}></Mark><i></i></span>
        <div className="ovb-t"><b>{bad.n} can't connect</b><span>{t && t.state === "fail" ? "Still refused just now" : "24 failed calls overnight"} · its tasks fall to the next provider in line</span></div>
        <div className="ovb-a"><button className="btn sm" disabled={t && t.state === "run"} onClick={() => A.test(bad.id)}>{t && t.state === "run" ? <><i className="spn"></i>Testing</> : <><Ic n="plug" s={12}></Ic>Test again</>}</button><button className="btn sm" onClick={() => A.edit({ k: "prov", id: bad.id })}>Edit connection</button><button className="ib" title="Dismiss until it fails again" onClick={() => { const id = bad.id; setHideBad(h => [...h, id]); A.toast("Hidden until " + bad.n + " fails again", () => setHideBad(h => h.filter(x => x !== id))); }}><Ic n="x" s={13}></Ic></button></div>
      </div>}
      <div className="ov">
        <div className="ov-m">
          {!none && <TuneUps S={S} A={A}></TuneUps>}
          {!none && <Usage></Usage>}
        </div>
        <aside className="ov-a">
          <WorkingNow S={S} A={A}></WorkingNow>
          <OrgState S={S} A={A}></OrgState>
        </aside>
      </div>
    </div>
  );
}
Object.assign(window, { Overview, PolicyEditor });
