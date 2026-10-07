function Health({ p, t }) {
  if (t && t.state === "run") return <span className="hl run"><i className="spn"></i>Testing connection…</span>;
  const r = t || (p.last && { state: p.last.ok ? "ok" : "fail", msg: p.last.msg, when: p.last.when });
  return (
    <span className="hl-w">
      {r ? <span className={"hl " + (r.state === "ok" ? "ok" : "bad")}><i></i>{r.msg}<em>{r.when || "just now"}</em></span> : <span className="hl nv"><i></i>Never tested</span>}
      {p.wk && <span className="hl-s mono">{p.wk.calls.toLocaleString()} calls · {(p.wk.f / p.wk.calls * 100).toFixed(1)}% failed</span>}
    </span>
  );
}
function ProviderDetail({ p, i, n, S, A }) {
  const needKey = !p.key && !p.local; const [k, setK] = React.useState("");
  const pre = PRESETS.find(x => x.n === p.n || x.kind === p.kind) || {};
  return (
    <div className="pd">
      {needKey && <div className="pd-key">
        <div><b>Add an API key to turn {p.n} on</b><span>Stored encrypted and never shown again. {p.tasks.length ? "It will take " + p.tasks.map(t => TASK_L[t]).join(" and ") + "." : ""}</span></div>
        <div className="pd-kf"><Ic n="key" s={13}></Ic><input type="password" autoFocus placeholder={pre.ph || "API key"} value={k} onChange={e => setK(e.target.value)} onKeyDown={e => e.key === "Enter" && k && A.saveKey(p.id)}></input><button className="btn ink sm" disabled={!k} onClick={() => A.saveKey(p.id)}>Save and test</button></div>
      </div>}
      <div className="pd-g">
        <div className="pd-s wide"><h4>Handles<em className="mono">{p.tasks.length}</em></h4>
          <div className="tks">{TASKS.map(t => { const on = p.tasks.includes(t.k), r = routeOf(t, S.providers), first = r.first && r.first.id === p.id; return <button key={t.k} className={"tkb" + (on ? " on" : "") + (first ? " first" : "")} title={first ? "This provider takes it first" : on ? "Assigned; an earlier provider takes it first" : "Not assigned"} onClick={() => A.toggleTask(p.id, t.k)}><Ic n={on ? "check" : "plus"} s={11} w={2.2}></Ic>{t.l}{t.trust && <Ic n="shield" s={10}></Ic>}</button>; })}</div>
        </div>
        <div className="pd-s"><h4>Access</h4>
          <div className="sw-r"><span><b>Trusted</b><em>May take tasks that read sensitive records</em></span><Switch on={p.trusted} onChange={v => A.setProv(p.id, { trusted: v })}></Switch></div>
          <div className="sw-r"><span><b>Private network</b><em>May reach a server on your own network</em></span><Switch on={p.priv} onChange={v => A.setProv(p.id, { priv: v })}></Switch></div>
        </div>
        <div className="pd-s"><h4>Price per million tokens</h4>
          <div className="price"><label><span>Input</span><i>$</i><input className="mono" placeholder="—"></input></label><label><span>Output</span><i>$</i><input className="mono" placeholder="—"></input></label></div>
          <p className="ad-h">{p.local ? "Leave empty for your own hardware; spend shows as —." : "Used to show spend on the overview."}</p>
        </div>
        <div className="pd-s"><h4>Last 7 days</h4>
          {p.wk ? <dl className="st4"><div><dt>Calls</dt><dd className="mono">{p.wk.calls}</dd></div><div><dt>Failed</dt><dd className={"mono" + (p.wk.f ? " t-d" : "")}>{p.wk.f}</dd></div><div><dt>Median</dt><dd className="mono">{p.wk.ms >= 1000 ? (p.wk.ms / 1000).toFixed(1) + "s" : p.wk.ms + "ms"}</dd></div><div><dt>Tokens</dt><dd className="mono">{p.wk.tok}</dd></div></dl> : <p className="ad-h">No calls yet.</p>}
        </div>
      </div>
      <div className="ad-bar">
        <button className="xa" onClick={() => A.edit({ k: "prov", id: p.id })}><Ic n="edit" s={13}></Ic>Edit connection</button>
        <button className="xa" disabled={i === 0} onClick={() => A.moveBy(p.id, -1)}><Ic n="up" s={13}></Ic>Move up</button>
        <button className="xa" disabled={i === n - 1} onClick={() => A.moveBy(p.id, 1)}><Ic n="down" s={13}></Ic>Move down</button>
        <span className="sp"></span>
        <button className="xa d" onClick={() => A.removeProv(p)}><Ic n="trash" s={13}></Ic>Remove</button>
      </div>
    </div>
  );
}
function daysOf(p) { if (!p.wk) return null; let x = p.id.length * 7; const r = () => (x = (x * 9301 + 49297) % 233280) / 233280; const base = p.wk.calls / 7; return WEEK.map(([d], i) => { const c = Math.round(base * (0.6 + r() * 0.8)); const f = p.id === "vllm" ? (i === 5 ? 16 : i === 6 ? 8 : 0) : (i === 4 ? 3 : i === 5 ? 2 : 0); return [d, c, f]; }); }
function Ring({ v, s = 38, cls = "" }) { const r = s / 2 - 3, c = 2 * Math.PI * r; return <svg className={"rng " + cls} width={s} height={s} viewBox={`0 0 ${s} ${s}`}><circle cx={s / 2} cy={s / 2} r={r} className="rng-b"></circle><circle cx={s / 2} cy={s / 2} r={r} className="rng-f" strokeDasharray={c} strokeDashoffset={c * (1 - v)} transform={`rotate(-90 ${s / 2} ${s / 2})`}></circle></svg>; }
function ProviderLine({ p, i, open, onOpen, S, A, drag, setDrag }) {
  const needKey = !p.key && !p.local, t = S.tests[p.id], days = daysOf(p), ok = p.wk ? 1 - p.wk.f / p.wk.calls : null, mx = days ? Math.max(...days.map(d => d[1])) : 1;
  const live = t && t.state === "run" ? "run" : t ? t.state : p.last ? (p.last.ok ? "ok" : "fail") : "none";
  const firsts = TASKS.filter(x => { const r = routeOf(x, S.providers); return r.first && r.first.id === p.id; }).length;
  return (
    <li className={"pl" + (open ? " sel" : "") + (p.on ? "" : " dim") + (drag === p.id ? " drg" : "")} id={"pv-" + p.id} draggable onDragStart={e => { setDrag(p.id); e.dataTransfer.effectAllowed = "move"; }} onDragOver={e => { e.preventDefault(); if (drag && drag !== p.id) A.moveTo(drag, p.id); }} onDragEnd={() => setDrag(null)} onClick={() => onOpen()}>
      <span className="pl-n mono"><Ic n="grip" s={11}></Ic>{String(i + 1).padStart(2, "0")}</span>
      <span className="pc-mk"><Mark p={p} s={26}></Mark><i className={"pc-dot sm " + live}></i></span>
      <span className="pl-t"><b>{p.n}</b><span className="mono">{p.model}</span></span>
      <span className="pl-tg">{needKey ? <span className="tg w">Needs a key</span> : !p.on ? <span className="tg">Off</span> : firsts ? <span className="pl-f">first for {firsts}</span> : <span className="pl-f dim">backup</span>}{p.trusted && <span className="tg b"><Ic n="shield" s={10}></Ic></span>}</span>
      <span className="pl-st">{p.wk ? <><span className="pc-bars sm">{days.map(([dd, c, f], k) => <i key={k} style={{ height: 3 + c / mx * 13 }}>{f > 0 && <u style={{ height: Math.max(3, f / c * 100) + "%" }}></u>}</i>)}</span><span className={"mono pl-ok" + (ok < 0.98 ? " t-w" : "")}>{(ok * 100).toFixed(1)}%</span><span className="mono pl-ms">{p.wk.ms >= 1000 ? (p.wk.ms / 1000).toFixed(1) + "s" : p.wk.ms + "ms"}</span></> : <span className="pl-none">No calls this week</span>}</span>
      <span className="pc-c" onClick={e => e.stopPropagation()}>{needKey ? <button className="btn sm" onClick={() => onOpen(true)}><Ic n="key" s={12}></Ic>Add key</button> : <button className="ib" title="Test connection" disabled={live === "run"} onClick={() => A.test(p.id)}>{live === "run" ? <i className="spn"></i> : <Ic n="plug" s={13}></Ic>}</button>}<Switch on={p.on} disabled={needKey} onChange={v => A.setProv(p.id, { on: v })}></Switch></span>
    </li>
  );
}
function ProviderCard({ p, i, n, open, onOpen, S, A, drag, setDrag, next }) {
  const needKey = !p.key && !p.local, t = S.tests[p.id], days = daysOf(p);
  const firsts = TASKS.filter(x => { const r = routeOf(x, S.providers); return r.first && r.first.id === p.id; });
  const backs = p.tasks.filter(k => !firsts.find(f => f.k === k));
  const ok = p.wk ? 1 - p.wk.f / p.wk.calls : null, mx = days ? Math.max(...days.map(d => d[1])) : 1;
  const live = t && t.state === "run" ? "run" : t ? t.state : p.last ? (p.last.ok ? "ok" : "fail") : "none";
  const flow = p.on && next && next.on;
  return (
    <li className={"pc-w" + (drag === p.id ? " drg" : "")} id={"pv-" + p.id} draggable onDragStart={e => { setDrag(p.id); e.dataTransfer.effectAllowed = "move"; }} onDragOver={e => { e.preventDefault(); if (drag && drag !== p.id) A.moveTo(drag, p.id); }} onDragEnd={() => setDrag(null)}>
      <span className="pc-n mono">{String(i + 1).padStart(2, "0")}</span>
      <Magic as="div" className={"pc" + (open ? " sel" : "") + (p.on ? "" : " dim")} onClick={() => onOpen()} from={p.c ? `oklch(0.66 0.16 ${p.h})` : "var(--brand)"} to={p.c ? `oklch(0.62 0.15 ${(p.h + 50) % 360})` : "oklch(0.7 0.12 230)"}>
        <div className="pc-g">
          <div className="pc-id">
            <span className="pc-mk"><Mark p={p} s={44}></Mark><i className={"pc-dot " + live}></i></span>
            <div className="pc-t">
              <div className="pc-h"><b>{p.n}</b>{needKey ? <span className="tg w">Needs a key</span> : !p.on ? <span className="tg">Off</span> : <span className="tg k"><i></i>On</span>}{p.trusted && <span className="tg b"><Ic n="shield" s={10}></Ic>Trusted</span>}</div>
              <span className="pc-m mono">{p.model}</span>
              <span className="pc-s">{p.kind}{p.priv ? " · your network" : ""}{p.key ? " · key stored" : ""}</span>
            </div>
          </div>
          <div className="pc-st">
            {p.wk ? <>
              <div className="pc-k"><Ring v={ok} cls={ok < 0.98 ? "w" : ""}></Ring><span><b className="mono">{(ok * 100).toFixed(1)}%</b><em>succeeded</em></span></div>
              <div className="pc-k"><span className="pc-bars">{days.map(([d, c, f], k) => <i key={k} title={d + ": " + c + " calls" + (f ? ", " + f + " failed" : "")} style={{ height: 4 + c / mx * 22 }}>{f > 0 && <u style={{ height: Math.max(3, f / c * 100) + "%" }}></u>}</i>)}</span><span><b className="mono">{p.wk.calls.toLocaleString()}</b><em>calls · 7 days</em></span></div>
              <div className="pc-k"><span><b className="mono">{p.wk.ms >= 1000 ? (p.wk.ms / 1000).toFixed(1) + "s" : p.wk.ms + "ms"}</b><em>median</em></span></div>
            </> : <span className="pc-none">{needKey ? "Add a key and it starts taking " + (p.tasks.length ? p.tasks.map(k => TASK_L[k]).join(" and ") : "work") + "." : "No calls yet."}</span>}
          </div>
          <div className="pc-c" onClick={e => e.stopPropagation()}>
            {needKey ? <button className="btn sm ink" onClick={() => onOpen(true)}><Ic n="key" s={12}></Ic>Add key</button>
              : <button className={"btn sm" + (live === "run" ? " busy" : "")} disabled={live === "run"} onClick={() => A.test(p.id)}>{live === "run" ? <i className="spn"></i> : <Ic n="plug" s={12}></Ic>}{live === "run" ? "Testing" : "Test"}</button>}
            <Switch on={p.on} disabled={needKey} label={p.on ? "Turn off" : "Turn on"} onChange={v => A.setProv(p.id, { on: v })}></Switch>
          </div>
        </div>
        {(t && t.state !== "run") && <div className={"pc-tr " + (t.state === "ok" ? "ok" : "bad")}><Ic n={t.state === "ok" ? "check" : "x"} s={11} w={2.4}></Ic>{t.msg}<em>just now</em></div>}
        <div className="pc-tk">
          {firsts.length > 0 && <span className="pc-tl">Takes first</span>}
          {firsts.map(x => <span key={x.k} className="tk first">{x.l}</span>)}
          {backs.length > 0 && <span className="pc-tl">Backup for</span>}
          {backs.map(k => <span key={k} className="tk">{TASK_L[k]}</span>)}
          {!p.tasks.length && <span className="tk none">No tasks assigned</span>}
        </div>
      </Magic>
      {next && <span className={"pc-flow" + (flow ? " on" : "")}><i></i><i></i><i></i><em>{flow ? "hands on what it can't take" : next.on ? "skipped while off" : "next is off"}</em></span>}
    </li>
  );
}
function Routing({ S, A }) {
  const ps = S.providers, cov = TASKS.filter(t => routeOf(t, ps).first).length;
  return (
    <section className="sec" id="routing">
      <SecH t="Routing" n={cov + " of " + TASKS.length + " covered"}></SecH>
      <p className="lead">Each task goes to the first provider, in order, that's on, assigned and — where marked <Ic n="shield" s={11}></Ic> — trusted. Click a cell to assign or unassign.</p>
      <div className="mx-w"><div className="mx" style={{ "--cols": ps.length }}>
        <div className="mx-h"><span>Task</span>{ps.map((p, i) => <span key={p.id} className={"mx-ph" + (p.on ? "" : " dim")}><em className="mono">{i + 1}</em><Mark p={p} s={16}></Mark><b>{p.n}</b></span>)}<span>Goes to</span></div>
        {TASKS.map(t => { const r = routeOf(t, ps); return (
          <div key={t.k} className={"mx-r" + (r.first ? "" : " un")}>
            <span className="mx-t">{t.l}{t.trust && <Ic n="shield" s={11}></Ic>}</span>
            {ps.map(p => { const as = p.tasks.includes(t.k), first = r.first && r.first.id === p.id, ok = p.on && (!t.trust || p.trusted), why = !p.on ? (p.key || p.local ? "off" : "needs key") : !ok ? "not trusted" : "backup"; return (
              <button key={p.id} className={"mx-c" + (as ? " as" : "") + (first ? " first" : "") + (as && !first ? " skip" : "")} onClick={() => A.toggleTask(p.id, t.k)} title={as ? (first ? p.n + " takes it" : p.n + ": " + why) : "Assign to " + p.n}>
                <i></i>{as && !first && <em>{why}</em>}
              </button>); })}
            <span className="mx-g">{r.first ? <><b>{r.first.n}</b>{r.next && <em>then {r.next.n}</em>}</> : <span className="t-w">{t.fb}</span>}</span>
          </div>); })}
      </div></div>
    </section>
  );
}
function EmptyProviders({ A }) {
  return (
    <div className="tabp"><div className="ep">
      <span className="ep-k mono">No providers yet</span>
      <h2>Connect a model to wake your agents</h2>
      <p>Agents, the assistant and document reading all route through a provider. Start with a hosted model, or point Trenova at one on your own network.</p>
      <div className="ep-g">{PRESETS.map(p => <button key={p.k} className="ep-b" onClick={() => A.addPreset(p)}><Mark p={p} s={30}></Mark><span><b>{p.n}</b><em>{p.local ? "Your network · no key" : "Hosted · API key"}</em></span><Ic n="arrowR" s={13}></Ic></button>)}</div>
      <div className="ep-f"><b>Until then</b><span>Search uses words only</span><span>Scope checks use built-in rules</span><span>Documents wait for manual entry</span></div>
    </div></div>
  );
}
function ProvidersTab({ S, A, searchRef }) {
  const [q, setQ] = React.useState(""); const [open, setOpen] = React.useState(S.focus); const [menu, setMenu] = React.useState(false); const [drag, setDrag] = React.useState(null); const [mode, setMode] = React.useState(null); const [showOff, setShowOff] = React.useState(false);
  React.useEffect(() => { if (!S.focus) return; if (S.focus !== "routing") setOpen(S.focus); setTimeout(() => { const el = document.getElementById(S.focus === "routing" ? "routing" : "pv-" + S.focus), sc = document.querySelector(".ws"); if (el && sc) sc.scrollTo({ top: el.offsetTop - 130, behavior: "smooth" }); }, 60); }, [S.focus, S.fseq]);
  React.useEffect(() => { const n = () => setMenu(true); window.addEventListener("aic-new", n); return () => window.removeEventListener("aic-new", n); }, []);
  if (!S.providers.length) return <EmptyProviders A={A}></EmptyProviders>;
  const list = S.providers.filter(p => (p.n + p.model + p.kind).toLowerCase().includes(q.toLowerCase()));
  const view = "compact", offN = S.providers.filter(p => !p.on).length;
  return (
    <div className="tabp">
      {(() => { const on = S.providers.filter(p => p.on), bad = S.providers.find(p => p.on && ((S.tests[p.id] && S.tests[p.id].state === "fail") || (!S.tests[p.id] && p.last && !p.last.ok))), nk = S.providers.find(p => !p.key && !p.local), unc = TASKS.filter(t => !routeOf(t, S.providers).first), calls = S.providers.reduce((t, p) => t + (p.wk ? p.wk.calls : 0), 0);
        return <Nova ctx="Providers" ctl={bad ? <><button className="btn ink lg" onClick={() => A.test(bad.id)}><Ic n="plug" s={13}></Ic>Test {bad.n}</button><span>{bad.base}</span></> : nk ? <><button className="btn ink lg" onClick={() => setOpen(nk.id)}><Ic n="key" s={13}></Ic>Add {nk.n} key</button><span>Takes {nk.tasks.map(k => TASK_L[k]).join(" and ")}</span></> : null}>
          <b>{on.length} of {S.providers.length}</b> providers are taking work — {calls.toLocaleString()} calls this week.{bad ? <> <span className="ref d" onClick={() => setOpen(bad.id)}>{bad.n}</span> is failing to connect, so its tasks fall through to the next in line.</> : ""}{nk ? <> <span className="ref w" onClick={() => setOpen(nk.id)}>{nk.n}</span> is waiting for a key.</> : ""}{unc.length ? <> <span className="ref" onClick={() => A.scrollTo("routing")}>{unc.length} task{unc.length > 1 ? "s have" : " has"}</span> nowhere to go.</> : ""}
        </Nova>; })()}
      <div className="tb">
        <Search q={q} setQ={setQ} ph="Search providers" inputRef={searchRef}></Search>
        <span className="sp"></span>
        <span className="tb-ct mono">{S.providers.filter(p => p.on).length} of {S.providers.length} on</span>
        <div className="rel"><button className="btn ink" onClick={() => setMenu(m => !m)}><Ic n="plus" s={13}></Ic>New provider<span className="kbd">N</span></button>
          {menu && <Menu right onClose={() => setMenu(false)} items={[{ h: "Hosted" }, ...PRESETS.filter(p => !p.local).map(p => ({ icon: <Mark p={p} s={18}></Mark>, l: p.n, s: p.model, on: () => A.addPreset(p) })), "-", { h: "On your network" }, ...PRESETS.filter(p => p.local).map(p => ({ icon: <Mark p={p} s={18}></Mark>, l: p.n, s: p.base, on: () => A.addPreset(p) }))]}></Menu>}
        </div>
      </div>
      <section className="sec chain-s"><SecH t="The chain" n={S.providers.length + " providers · top to bottom"} r={<span className="sh2-n">Drag to reorder</span>}></SecH>
      {view === "compact" ? <ol className="pls">{list.filter(p => p.on || showOff || q).map(p => <ProviderLine key={p.id} p={p} i={S.providers.indexOf(p)} open={open === p.id} onOpen={force => setOpen(force === true ? p.id : open === p.id ? null : p.id)} S={S} A={A} drag={drag} setDrag={setDrag}></ProviderLine>)}
        {!q && offN > 0 && <li className="pl-off" onClick={() => setShowOff(!showOff)}><Ic n={showOff ? "chevD" : "chevR"} s={12}></Ic><span>{showOff ? "Hide" : "Show"} {offN} off</span><span className="pl-om">{S.providers.filter(p => !p.on).map(p => <Mark key={p.id} p={p} s={18}></Mark>)}</span><em>They keep their place in line and are skipped until turned on.</em></li>}
        <li className="pl-add" onClick={() => setMenu(true)}><Ic n="plus" s={13}></Ic>Add a provider</li></ol> :
      <ol className="chain">{list.map((p, i) => { const idx = S.providers.indexOf(p); return <ProviderCard key={p.id} p={p} i={idx} n={S.providers.length} next={S.providers[idx + 1]} open={open === p.id} onOpen={force => setOpen(force === true ? p.id : open === p.id ? null : p.id)} S={S} A={A} drag={drag} setDrag={setDrag}></ProviderCard>; })}
        <li className="pc-w add"><span className="pc-n mono">+</span><button className="pc-add" onClick={() => setMenu(true)}><span className="pc-add-m">{PRESETS.slice(0, 5).map(x => <Mark key={x.k} p={x} s={24}></Mark>)}</span><span><b>Add a provider</b><em>Hosted with a key, or a model on your own network</em></span><Ic n="plus" s={14}></Ic></button></li>
      </ol>}</section>
      <Routing S={S} A={A}></Routing>
      {(() => { const p = S.providers.find(x => x.id === open); if (!p) return null; const i = S.providers.indexOf(p); return (
        <Sheet onClose={() => setOpen(null)} head={<><Mark p={p} s={36}></Mark><div className="sh-t"><b>{p.n}</b><span className="mono">{p.model}</span></div><Switch on={p.on} disabled={!p.key && !p.local} label={p.on ? "Turn off" : "Turn on"} onChange={v => A.setProv(p.id, { on: v })}></Switch></>}>
          <div className="sh-act"><button className="btn sm" onClick={() => A.edit({ k: "prov", id: p.id })}><Ic n="edit" s={12}></Ic>Edit connection</button></div>
          <div className="sh-m"><span>Priority {i + 1}</span><span>{p.kind}</span><span className="mono">{p.base || "Provider default"}</span></div>
          <div className="sh-hl"><Health p={p} t={S.tests[p.id]}></Health>{(p.key || p.local) && <button className="btn sm" disabled={S.tests[p.id] && S.tests[p.id].state === "run"} onClick={() => A.test(p.id)}><Ic n="plug" s={12}></Ic>Test</button>}</div>
          <ProviderDetail p={p} i={i} n={S.providers.length} S={S} A={A}></ProviderDetail>
        </Sheet>); })()}
    </div>
  );
}
Object.assign(window, { ProvidersTab });
