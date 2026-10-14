function AnHead({ groups }) {
  return <>{groups.map(g => <span key={g.key} className="an-hh c" title={g.label + " · " + g.sub}><AnSw g={g}></AnSw><b>{g.short}</b></span>)}</>;
}
function AnTools({ groups, onCell }) {
  const { hl } = React.useContext(AnCtx);
  const tools = AN_TOOLS.map((t, i) => ({ t, i, n: groups.reduce((s, g) => s + (g.m.tools[i] ? g.m.tools[i].n : 0), 0), f: groups.reduce((s, g) => s + (g.m.tools[i] ? Object.values(g.m.tools[i].f).reduce((a, b) => a + b, 0) : 0), 0) })).filter(x => x.n).sort((a, b) => b.f / b.n - a.f / a.n);
  return (
    <AnCard t="Where tool calls fail" d="Share of each tool's calls that failed. Click a cell to see real examples." cls="an-tl" r={<span className="an-hk"><i style={{ "--i": 0.1 }}></i><i style={{ "--i": 0.4 }}></i><i style={{ "--i": 0.7 }}></i><i style={{ "--i": 1 }}></i><span>0 → 12%+</span></span>}>
      <div className="an-hmw"><div className="an-hm" style={{ "--c": groups.length }}>
        <span className="an-hh"></span><AnHead groups={groups}></AnHead>
        {tools.map(({ t, i }, ri) => <React.Fragment key={t[0]}>
          <span className="an-tn" title={t[1]}><b>{t[1]}</b><span className="mono">{t[0]}</span></span>
          {groups.map((g, ci) => { const x = g.m.tools[i]; if (!x || !x.n) return <span key={g.key} className={"an-c z" + anDim(hl, g.key)}>—</span>; const f = Object.values(x.f).reduce((a, b) => a + b, 0), r = f / x.n, few = x.n < 20;
            return <button key={g.key} className={"an-c" + (few ? " few" : "") + anDim(hl, g.key)} style={{ "--i": few ? 0 : Math.min(1, r / 0.12), animationDelay: (ri * 30 + ci * 20) + "ms" }} title={`${g.label} · ${f} of ${x.n} calls failed`} onClick={() => onCell(g, i)}><span className="mono">{few ? "few" : r === 0 ? "0" : r < 0.001 ? "<0.1%" : anP(r, r < 0.1 ? 1 : 0)}</span></button>; })}
        </React.Fragment>)}
      </div></div>
    </AnCard>
  );
}
function AnTasks({ groups, deps, S, A }) {
  const { hl } = React.useContext(AnCtx);
  const tks = TASKS.filter(t => groups.some(g => g.m.tasks[t.k]));
  return (
    <AnCard t="Which model fits which task" d="Completed runs per task, counting only the tasks each model was routed to." cls="an-tf" r={<div className="an-lgd sm"><span className="an-rtk"><i></i>Goes here now</span><span><Ic n="check" s={10} w={2.6}></Ic>Best for the task</span></div>}>
      <div className="an-hmw"><div className="an-hm fit" style={{ "--c": groups.length }}>
        <span className="an-hh"></span><AnHead groups={groups}></AnHead>
        {tks.map(t => { const route = S.providers.length ? routeOf(t, S.providers).first : null;
          const rate = x => x.m.tasks[t.k].c / x.m.tasks[t.k].n;
          const el = deps.filter(d => d.m.tasks[t.k] && d.m.tasks[t.k].n >= 30), bd = el.length ? el.reduce((a, b) => rate(b) > rate(a) ? b : a) : null;
          const rd = route && deps.find(d => d.pids[0] === route.id && d.m.tasks[t.k]);
          const gain = bd && rd && bd !== rd ? rate(bd) - rate(rd) : 0;
          const ge = groups.filter(g => g.m.tasks[t.k] && g.m.tasks[t.k].n >= 30), bestG = ge.length > 1 ? ge.reduce((a, b) => rate(b) > rate(a) ? b : a) : null;
          return <React.Fragment key={t.k}>
            <span className="an-tn" title={t.l}><b>{t.l}</b>{gain >= 0.05 ? <button className="an-rto" onClick={() => { A.go("providers", null, "routing"); A.toast("Put " + bd.label + " first for " + t.l + " in routing"); }} title={"Route " + t.l + " to " + bd.label}><Ic n="arrowR" s={10}></Ic>Use {AN_SHORT.p[bd.pids[0]]}<em className="mono t-k">+{Math.round(gain * 100)}</em></button> : route && <span>Goes to {route.n}</span>}</span>
            {groups.map(g => { const x = g.m.tasks[t.k]; if (!x) return <span key={g.key} className={"an-c z" + anDim(hl, g.key)}>—</span>; const r = x.c / x.n, rt = route && g.pids.includes(route.id);
              return <span key={g.key} className={"an-c ok" + (x.n < 30 ? " few" : "") + (rt ? " rt" : "") + (bestG === g ? " is-best" : "") + anDim(hl, g.key)} style={{ "--i": x.n < 30 ? 0 : Math.max(0, (r - 0.6) / 0.38) }} title={Math.round(x.c) + " of " + x.n + " runs completed" + (rt ? " · routed here now" : "")}>{bestG === g && <Ic n="check" s={10} w={2.6}></Ic>}<span className="mono">{x.n < 30 ? "few" : anP(r)}</span></span>; })}
          </React.Fragment>; })}
      </div></div>
    </AnCard>
  );
}
function AnDecide({ groups, onOpen }) {
  const { hl, setHl } = React.useContext(AnCtx);
  const rows = groups.filter(g => anNP(g.m)).sort((a, b) => b.m.prop.a / anNP(b.m) - a.m.prop.a / anNP(a.m));
  return (
    <AnCard t="What people did with its proposals" d="A read on judgment, not just completion." cls="an-dc">
      {rows.map((g, i) => { const p = g.m.prop, n = anNP(g.m); return <button key={g.key} className={"an-dcr" + anDim(hl, g.key)} onMouseEnter={() => setHl(g.key)} onMouseLeave={() => setHl(null)} onClick={() => onOpen(g)}>
        <span className="an-dcn"><AnSw g={g}></AnSw><b>{g.label}</b><em className="mono">{n}</em></span>
        <span className="an-stk">{[["a", p.a], ["m", p.m], ["r", p.r]].map(([k, v]) => v > 0 && <i key={k} className={"p-" + k} style={{ flexGrow: v, animationDelay: i * 50 + "ms" }}></i>)}</span>
        <span className="an-dcv mono">{anP(p.a / n)}</span>
      </button>; })}
      <div className="an-lgd"><span><i className="p-a"></i>Approved as proposed</span><span><i className="p-m"></i>With changes</span><span><i className="p-r"></i>Rejected</span></div>
    </AnCard>
  );
}
function AnHealth({ deps, A }) {
  const mx = Math.max(0.0001, ...deps.map(d => anPE(d.m) / d.m.mc));
  return (
    <AnCard t="Provider health" d="Errors that happened before the model could answer. Always shown by provider." cls="an-ph">
      <div className="an-phh"><span>Provider</span><span>Errors</span><span className="r">Rate</span><span className="r">Fell to next</span><span className="r">First token</span></div>
      {deps.map((d, i) => { const pe = anPE(d.m), r = pe / d.m.mc; return <button key={d.key} className="an-phr" onClick={() => A.go("providers", d.pids[0])}>
        <span className="an-phn"><Mark p={d.provs[0]} s={22}></Mark><span className="tl"><b>{d.label}</b><span>{d.sub}{d.until ? " · " + anUntil(d.until) : ""}</span></span></span>
        <span className="an-pe"><span style={{ width: Math.max(pe ? 3 : 0, r / mx * 100) + "%" }}>{AN_PERR.map(([k, l]) => d.m.perr[k] > 0 && <i key={k} className={"e-" + k} style={{ flexGrow: d.m.perr[k], animationDelay: i * 50 + "ms" }} title={l + ": " + d.m.perr[k]}></i>)}</span>{!pe && <em className="dim">None</em>}</span>
        <span className={"mono r" + (r > 0.05 ? " t-d" : "")}>{pe ? anP(r, 1) : "—"}</span>
        <span className={"mono r" + (d.m.fb > 50 ? " t-w" : d.m.fb ? "" : " dim")}>{d.m.fb ? d.m.fb.toLocaleString() : "—"}</span>
        <span className="mono r">{anMs(d.m.ttft / 1000)}</span>
      </button>; })}
      <div className="an-lgd">{AN_PERR.map(([k, l]) => <span key={k}><i className={"e-" + k}></i>{l}</span>)}</div>
    </AnCard>
  );
}
function AnSheet({ open, onClose, A }) {
  if (open.t === "cell") {
    const { g, i } = open, x = g.m.tools[i], t = AN_TOOLS[i], f = Object.values(x.f).reduce((a, b) => a + b, 0);
    const top = AN_WHY.filter(([r]) => x.f[r] > 0).sort((a, b) => x.f[b[0]] - x.f[a[0]]);
    const ags = AGENTS0.filter(a => a.tools > 0 && !a.off);
    return <Sheet onClose={onClose} head={<><AnMarks provs={g.provs} s={30}></AnMarks><div className="sh-t"><b>{t[1]} · {g.label}</b><span>{f} of {x.n.toLocaleString()} calls failed</span></div></>}>
      <div className="sh-p"><h4 className="sh-k">Why</h4>{top.length ? top.map(([r, l, who]) => <div key={r} className="an-wr">
        <span className="an-wl"><b>{l}</b><span className="mono">{x.f[r]} · {who === "m" ? "the model" : "the tool or a rule"}</span></span>
        <span className="an-stk"><i className={"r-" + r} style={{ flexGrow: x.f[r] }}></i><i className="z" style={{ flexGrow: f - x.f[r] }}></i></span></div>) : <p className="ad-h">No failures in this range.</p>}</div>
      {top.length > 0 && <div className="sh-p"><h4 className="sh-k">Recent failures</h4>{top.slice(0, 3).map(([r], k) => { const [n, msg] = AN_SAMPLE[r](t[0]); const a = ags[Math.floor(anH(g.key + t[0] + r) * ags.length)]; return <div key={r} className="an-ex"><code className="mono">{n}</code><span>{msg}</span><em>{a.n} · {["12 min ago", "Today 9:40 AM", "Yesterday"][k]}</em></div>; })}</div>}
      <div className="ad-bar sh-f"><button className="xa" onClick={() => A.go("safety")}><Ic n="shield" s={13}></Ic>Tool rules</button><button className="xa" onClick={() => A.go("activity")}><Ic n="timeline" s={13}></Ic>Failed runs</button></div>
    </Sheet>;
  }
  const g = open.g, m = g.m, np = anNP(m);
  const tl = AN_TOOLS.map((t, i) => ({ t, x: m.tools[i] })).filter(o => o.x && o.x.n >= 20).map(o => ({ ...o, r: Object.values(o.x.f).reduce((a, b) => a + b, 0) / o.x.n })).sort((a, b) => b.r - a.r).slice(0, 4);
  return <Sheet onClose={onClose} head={<><AnMarks provs={g.provs} s={30}></AnMarks><div className="sh-t"><b>{g.label}</b><span>{g.sub}{g.until ? " · " + anUntil(g.until) : ""}</span></div></>}>
    <KV items={[["Runs", m.runs.toLocaleString()], ["Completed", anP(m.out.c / m.runs)], ["Tool calls failed", anP(anTF(m) / (m.calls || 1), 1) + " · " + (anTF(m) ? anP(anTFm(m) / anTF(m)) : "0%") + " the model's mistake"], ["Approved as proposed", np ? anP(m.prop.a / np) + " of " + np : "—"], ["Rated helpful", anP(m.rated)], ["Cost", m.local ? "Local" : "$" + m.cost.toFixed(2) + " · $" + (m.cost / m.runs).toFixed(3) + " a run"], ["First token", anMs(m.ttft / 1000) + " median"]]}></KV>
    <div className="sh-p"><h4 className="sh-k">Run time by outcome</h4><table className="ut an-mini"><thead><tr><th>Outcome</th><th className="r">Runs</th><th className="r">Median</th><th className="r">Slowest 5%</th></tr></thead><tbody>{AN_OUT.map(([o, l]) => <tr key={o}><td><span className="an-ol"><i className={"o-" + o}></i>{l}</span></td><td className="r mono">{m.out[o]}</td><td className="r mono">{m.out[o] ? anS(m.dur[o][1]) : "—"}</td><td className="r mono">{m.out[o] ? anS(m.dur[o][3]) : "—"}</td></tr>)}</tbody></table></div>
    {tl.length > 0 && <div className="sh-p"><h4 className="sh-k">Tools it gets wrong most</h4>{tl.map(o => <div key={o.t[0]} className="an-wr"><span className="an-wl"><b>{o.t[1]}</b><span className="mono">{anP(o.r, 1)} of {o.x.n}</span></span><span className="ubar"><i className="f" style={{ width: Math.min(100, o.r / 0.15 * 100) + "%" }}></i></span></div>)}</div>}
    <div className="ad-bar sh-f">{g.pids.map(pid => <button key={pid} className="xa" onClick={() => A.go("providers", pid)}><Ic n="server" s={13}></Ic>{AN_PROV[pid].n}</button>)}<span className="sp"></span><button className="xa" onClick={() => A.go("activity")}>Its runs</button></div>
  </Sheet>;
}
function AnalyticsTab({ S, A }) {
  const [rg, setRg] = React.useState("d30"), [by, setBy] = React.useState("model"), [ag, setAg] = React.useState("all"), [tk, setTk] = React.useState("all"), [open, setOpen] = React.useState(null);
  const [hl, setHl] = React.useState(null), [hidden, setHidden] = React.useState([]);
  React.useEffect(() => setHidden([]), [by]);
  const { groups: all0, deps: deps0, all } = React.useMemo(() => anGroups(rg, ag, tk, by), [rg, ag, tk, by]);
  const groups = all0.filter(g => !hidden.includes(g.key)), deps = deps0.filter(d => by === "provider" ? !hidden.includes(d.pids[0]) : !hidden.includes(d.model));
  if (!S.providers.length) return <div className="tabp"><div className="empty tall"><b>Nothing to measure yet</b><span>Analytics fills in once a provider is connected and agents start running.</span><button className="btn" onClick={() => A.go("providers")}><Ic n="plus" s={12}></Ic>Connect a provider</button></div></div>;
  const rgl = { d1: "the last 24 hours", d7: "the last 7 days", d30: "the last 30 days" }[rg];
  const onOpen = g => setOpen({ t: "g", g });
  const scope = <div className="an-scope">
    <Seg v={rg} opts={[["d1", "24 hours"], ["d7", "7 days"], ["d30", "30 days"]]} onChange={setRg}></Seg>
    <span className="an-by"><span>Compare by</span><Seg v={by} opts={[["model", "Model"], ["provider", "Provider"]]} onChange={setBy}></Seg></span>
    <span className="sp"></span>
    <AnSel l="Agent" v={ag} opts={[["all", "Every agent"], ...AGENTS0.filter(a => !a.off && a.last).map(a => [a.id, a.n])]} onChange={setAg}></AnSel>
    <AnSel l="Task" v={tk} opts={[["all", "Every task"], ...TASKS.filter(t => AN_DEPS.some(d => t.k in d.tasks)).map(t => [t.k, t.l])]} onChange={setTk}></AnSel>
    {(ag !== "all" || tk !== "all") && <button className="lnk" onClick={() => { setAg("all"); setTk("all"); }}>Clear</button>}
  </div>;
  if (!all) return <div className="tabp">{scope}<div className="empty"><b>No runs match</b><span>Nothing ran for this agent and task in {rgl}.</span></div></div>;
  const el = all0.filter(g => g.m.runs >= 30), best = el.length ? el.reduce((a, b) => b.m.out.c / b.m.runs > a.m.out.c / a.m.runs ? b : a) : null;
  const byM = {}; deps0.forEach(d => (byM[d.model] = byM[d.model] || []).push(d));
  const shared = Object.values(byM).find(xs => xs.filter(d => d.m.runs >= 30).length > 1);
  const sh = shared && shared.filter(d => d.m.runs >= 30).sort((a, b) => b.m.out.c / b.m.runs - a.m.out.c / a.m.runs);
  const tfr = g => anTF(g.m) / (g.m.calls || 1), worst = el.length > 2 ? el.reduce((a, b) => tfr(b) > tfr(a) ? b : a) : null;
  const rest = worst && el.filter(g => g !== worst), restR = rest && rest.reduce((t, g) => t + anTF(g.m), 0) / rest.reduce((t, g) => t + g.m.calls, 0);
  const wr = worst && AN_WHY.reduce((a, b) => worst.m.tf[b[0]] > worst.m.tf[a[0]] ? b : a);
  const tot = anTF(all);
  return (
    <AnCtx.Provider value={{ hl, setHl }}>
    <div className="tabp anx">
      <Nova ctx="Analytics" ctl={<><button className="btn" onClick={() => A.toast("Exported " + rgl + " as CSV")}><Ic n="download" s={13}></Ic>Export</button><span>Updated 2 min ago</span></>}>
        {best && <>Over {rgl}, <span className="ref" onClick={() => onOpen(best)}>{best.label}</span> completed the most of its runs — <b>{anP(best.m.out.c / best.m.runs)}</b>. </>}
        {sh && <><b>{sh[0].sub}</b> completed {anP(sh[0].m.out.c / sh[0].m.runs)} on {sh[0].label} but {anP(sh[1].m.out.c / sh[1].m.runs)} on <span className="ref d" onClick={() => A.go("providers", sh[1].pids[0])}>{sh[1].label}</span>. It's the same model, so the gap comes from the connection. </>}
        {worst && restR > 0 && tfr(worst) / restR >= 2 && <><span className="ref" onClick={() => onOpen(worst)}>{worst.label}</span> gets tool calls wrong <b>{(tfr(worst) / restR).toFixed(1)}×</b> as often as the rest, mostly {wr[1].toLowerCase()}.</>}
      </Nova>
      {scope}
      <Figs items={[
        ["Runs", <AnNum v={all.runs} f={x => Math.round(x).toLocaleString()}></AnNum>, all.mc.toLocaleString() + " model calls"],
        ["Completed", <AnNum v={all.out.c / all.runs * 100} f={x => x.toFixed(1) + "%"}></AnNum>, all.out.w + " handed to a person · " + all.out.f + " failed"],
        ["Median run", <AnNum v={all.dur.c[1]} f={x => anS(x)}></AnNum>, "Slowest 5% took " + anS(all.dur.c[3]) + " or more"],
        ["Tool calls failed", <AnNum v={tot / all.calls * 100} f={x => x.toFixed(1) + "%"}></AnNum>, tot.toLocaleString() + " of " + all.calls.toLocaleString() + " · " + anP(anTFm(all) / (tot || 1)) + " the model's mistake", tot / all.calls > 0.03 ? "t-d" : ""],
        ["Spend", <AnNum v={all.cost} f={x => "$" + x.toFixed(2)}></AnNum>, "$" + (all.cost / all.out.c).toFixed(3) + " per completed run"],
      ]}></Figs>
      <AnField groups={all0} hidden={hidden} setHidden={setHidden} by={by}></AnField>
      <h2 className="an-h">Leaderboard</h2>
      <div className="an-g3">{["c", "med", "tf", "cost", "ap", "ttft"].map(k => <AnBoard key={k + by} k={k} groups={groups} onOpen={onOpen}></AnBoard>)}</div>
      <h2 className="an-h">Trade-offs</h2>
      <div className={"an-g2" + (by === "model" && sh ? "" : " is-one")}><AnScatter groups={groups} onOpen={onOpen}></AnScatter>{by === "model" && <AnSame deps={deps} S={S} A={A}></AnSame>}</div>
      <h2 className="an-h">Run time</h2>
      <AnRunTime groups={groups} deps={deps} onOpen={onOpen}></AnRunTime>
      <AnTrend key={by + rg} groups={groups} rg={rg} ag={ag}></AnTrend>
      <h2 className="an-h">Tool calls</h2>
      <div className="an-g2 is-tools"><AnTools groups={groups} onCell={(g, i) => setOpen({ t: "cell", g, i })}></AnTools><AnWhose groups={groups}></AnWhose></div>
      <h2 className="an-h">Routing and judgment</h2>
      {tk === "all" && <AnTasks groups={groups} deps={deps} S={S} A={A}></AnTasks>}
      <div className="an-g2 is-health"><AnDecide groups={groups} onOpen={onOpen}></AnDecide><AnHealth deps={deps} A={A}></AnHealth></div>
      {open && <AnSheet open={open} onClose={() => setOpen(null)} A={A}></AnSheet>}
    </div>
    </AnCtx.Provider>
  );
}
Object.assign(window, { AnalyticsTab });
