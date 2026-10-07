const MODELS = {
  "Anthropic Messages": [["claude-sonnet-5", "200k context", "$3 / $15"], ["claude-haiku-5", "200k context", "$0.80 / $4"], ["claude-opus-5", "200k context", "$15 / $75"]],
  "OpenAI Responses": [["gpt-5.1", "400k context", "$1.25 / $10"], ["gpt-5.1-mini", "400k context", "$0.25 / $2"], ["text-embedding-3-large", "Embedding", "$0.13"]],
  "Azure OpenAI": [["gpt-5.1", "Deployment · eastus", "$1.25 / $10"], ["gpt-5.1-mini", "Deployment · eastus", "$0.25 / $2"]],
  Gemini: [["gemini-3-pro", "1M context", "$2 / $12"], ["gemini-3-flash", "1M context", "$0.30 / $2.50"]],
  Ollama: [["qwen2.5:14b", "9.0 GB · loaded", "Local"], ["llama3.3:70b", "43 GB", "Local"], ["nomic-embed-text", "Embedding · 274 MB", "Local"]],
  "OpenAI-compatible": [["meta-llama/Llama-3.3-70B-Instruct", "128k context", "Local"], ["Qwen/Qwen2.5-32B-Instruct", "32k context", "Local"]],
};
const KINDS = Object.keys(MODELS).concat(["Mistral", "Bedrock"]);
const isLocalUrl = u => /localhost|127\.0\.0\.1|192\.168\.|10\.\d|\.local\b/.test(u || "");
function provDraft(p, pre) {
  const x = p || pre;
  return { n: p ? p.n : pre.local ? (pre.k === "ollama" ? "Local Ollama" : "My model server") : pre.n, kind: x.kind, base: x.base || "", model: x.model, key: "", rotate: true, tasks: p ? p.tasks : [], trusted: !!x.trusted, priv: p ? p.priv : !!pre.local, timeout: 60, conc: 8, cap: p && !p.local ? 200 : "", capThen: "next", pin: p && p.local ? "" : "3", pout: p && p.local ? "" : "15" };
}
function RouteImpact({ before, after }) {
  const rows = TASKS.map(t => ({ t, a: routeOf(t, before).first, b: routeOf(t, after).first })).filter(x => (x.a && x.a.id) !== (x.b && x.b.id));
  if (!rows.length) return <p className="imp-n"><Ic n="check" s={12} w={2.2}></Ic>No task changes where it goes.</p>;
  return <div className="imp"><div className="imp-h">When you save</div>{rows.map(({ t, a, b }) => <div key={t.k} className={"imp-r" + (!b ? " lost" : !a ? " won" : "")}><span>{t.l}</span><span className="imp-v">{a ? <><Mark p={a} s={14}></Mark>{a.n}</> : <em>Nowhere</em>}<Ic n="arrowR" s={11}></Ic>{b ? <><Mark p={b} s={14}></Mark><b>{b.n}</b></> : <em className="t-w">Nowhere · {t.fb.toLowerCase()}</em>}</span></div>)}</div>;
}
function ProviderEditor({ p, preset, S, A, onClose, demo }) {
  const d = useDraft(provDraft(p, preset)); const v = d.v, set = d.set;
  const [models, setModels] = React.useState(p ? MODELS[v.kind] || null : null); const [fetching, setFetching] = React.useState(false);
  const [test, setTest] = React.useState(null); const [rm, setRm] = React.useState(false); const [conf, setConf] = React.useState(demo === "conflict" && !!p);
  const local = p ? p.local : preset.local, hasKey = p && p.key;
  React.useEffect(() => { if (demo === "unsaved" && p) d.patch({ timeout: 90, tasks: [...v.tasks, "Embedding"].filter((x, i, a) => a.indexOf(x) === i) }); if (!p && local) fetchModels(); }, []);
  const fetchModels = () => { setFetching(true); setTimeout(() => { setModels(MODELS[v.kind] || []); setFetching(false); }, 900); };
  const runTest = () => { setTest({ s: "run" }); setTimeout(() => setTest(/:8000/.test(v.base) ? { s: "fail", m: "Connection refused", d: "dial tcp 127.0.0.1:8000: connect: connection refused", hint: "Is the server running? vLLM listens on :8000 by default." } : !local && !hasKey && !v.key ? { s: "fail", m: "No API key", d: "401 · missing credentials", hint: "Add a key below, then test again." } : { s: "ok", m: "Connected · " + (local ? 386 : 241) + " ms", d: v.model + " answered a 12-token probe" }), 1200); };
  const urlErr = isLocalUrl(v.base) && !v.priv ? "This address is on your own network. Turn on Private network so Trenova can reach it." : v.base && !/^https?:\/\//.test(v.base) && v.kind !== "Bedrock" ? "Start with http:// or https://" : null;
  const draftP = { ...(p || { id: "__new", m: preset.m, h: preset.h, c: preset.c, local }), n: v.n, model: v.model, tasks: v.tasks, trusted: v.trusted, on: p ? p.on : true, key: true };
  const after = p ? S.providers.map(x => x.id === p.id ? draftP : x) : [...S.providers, draftP];
  const needKeyErr = !p && !local && !v.key ? "Paste an API key" : null;
  const invalid = !v.n.trim() ? "Give it a name" : urlErr ? "Fix the base URL" : needKeyErr;
  const fields = { n: { l: "Name" }, kind: { l: "Kind" }, base: { l: "Base URL" }, model: { l: "Model" }, key: { l: "API key", fmt: x => x ? "New key ••••" + x.slice(-4) : "Unchanged" }, rotate: { l: "Keep old key 24h" }, tasks: { l: "Handles", item: k => TASK_L[k] }, trusted: { l: "Trusted" }, priv: { l: "Private network" }, timeout: { l: "Timeout", fmt: x => x + "s" }, conc: { l: "Concurrent calls" }, cap: { l: "Monthly cap", fmt: x => x === "" ? "None" : "$" + x }, capThen: { l: "At the cap" }, pin: { l: "Input price" }, pout: { l: "Output price" } };
  const mk = p || preset;
  const sections = [
    { id: "conn", l: "Connection", ic: "plug", keys: ["n", "kind", "base"], warn: urlErr, body: <>
      <div className="f-grid"><F l="Name"><Txt v={v.n} on={x => set("n", x)} auto={!p}></Txt></F><F l="Kind"><Sel v={v.kind} on={x => { set("kind", x); setModels(null); }} opts={KINDS}></Sel></F></div>
      <F l="Base URL" err={urlErr} hint={v.base ? null : "Leave empty to use " + v.kind + "'s default endpoint."} aside={urlErr && <button className="lnk" onClick={() => set("priv", true)}>Turn on Private network</button>}><Txt v={v.base} on={x => set("base", x)} mono ph="https://api.example.com/v1"></Txt></F>
    </> },
    { id: "model", l: "Model", ic: "brain", keys: ["model"], r: <button className="btn sm" disabled={fetching} onClick={fetchModels}>{fetching ? <><i className="spn"></i>Asking {v.kind}…</> : <><Ic n="refresh" s={12}></Ic>{models ? "Refresh" : "Fetch models"}</>}</button>, body: <>
      {models ? <div className="mdl">{models.map(([id, meta, price]) => <button key={id} className={"mdl-r" + (v.model === id ? " on" : "")} onClick={() => set("model", id)}><span className="mdl-o"></span><span className="mdl-t"><b className="mono">{id}</b><em>{meta}</em></span><span className="mdl-p mono">{price}</span></button>)}</div>
        : <F l="Model ID" hint="Or fetch the list this provider actually serves."><Txt v={v.model} on={x => set("model", x)} mono></Txt></F>}
    </> },
    ...(!local ? [{ id: "key", l: "API key", ic: "key", keys: ["key", "rotate"], body: <>
      {hasKey && !v.key && <div className="kst"><Ic n="key" s={13}></Ic><span className="mono">{v.kind === "Anthropic Messages" ? "sk-ant-" : "sk-"}••••••••4f2a</span><span className="kst-m">Added Sep 12 by Sarah Alvarez · last used 2 min ago</span></div>}
      <F l={hasKey ? "Replace key" : "Key"} hint="Stored encrypted. Nobody, including you, can read it back."><Txt type="password" v={v.key} on={x => set("key", x)} mono ph={(PRESETS.find(x => x.kind === v.kind) || {}).ph || "API key"}></Txt></F>
      {hasKey && v.key && <SwRow l="Keep the old key working for 24 hours" s="So nothing fails while other systems switch over." v={v.rotate} on={x => set("rotate", x)}></SwRow>}
    </> }] : []),
    { id: "tasks", l: "What it handles", ic: "route", keys: ["tasks"], note: "Each task goes to the first provider in order that takes it.", body: <>
      <Chips v={v.tasks} on={x => set("tasks", x)} opts={TASKS.map(t => [t.k, t.l, t.trust ? <Ic n="shield" s={10}></Ic> : null])}></Chips>
      <RouteImpact before={S.providers} after={after}></RouteImpact>
    </> },
    { id: "access", l: "Access", ic: "shield", keys: ["trusted", "priv"], body: <>
      <SwRow l="Trusted" s="May take tasks that read sensitive records, like billing diagnosis." v={v.trusted} on={x => set("trusted", x)}></SwRow>
      <SwRow l="Private network" s="May reach a server on your own network." v={v.priv} on={x => set("priv", x)}></SwRow>
      {v.tasks.some(k => TASKS.find(t => t.k === k).trust) && !v.trusted && <Callout tone="w" act={<button className="btn sm" onClick={() => set("trusted", true)}>Trust it</button>}>Billing diagnosis needs a trusted provider. It's assigned here but will be skipped.</Callout>}
    </> },
    { id: "limits", l: "Limits and price", ic: "gauge", keys: ["timeout", "conc", "cap", "capThen", "pin", "pout"], body: <>
      <div className="f-grid"><F l="Timeout"><Txt type="number" v={v.timeout} on={x => set("timeout", x)} suf="seconds" mono></Txt></F><F l="Concurrent calls"><Txt type="number" v={v.conc} on={x => set("conc", x)} suf="at once" mono></Txt></F></div>
      <div className="f-grid"><F l="Monthly spend cap" hint={v.cap ? "$" + Math.round(v.cap * 0.38) + " spent this month" : "No cap"}><Txt type="number" v={v.cap} on={x => set("cap", x)} pre="$" mono ph="None"></Txt></F><F l="At the cap"><Seg v={v.capThen} opts={[["next", "Hand to next"], ["stop", "Stop"]]} onChange={x => set("capThen", x)}></Seg></F></div>
      <div className="f-grid"><F l="Input price" hint="Per million tokens"><Txt v={v.pin} on={x => set("pin", x)} pre="$" mono ph="—"></Txt></F><F l="Output price" hint="Per million tokens"><Txt v={v.pout} on={x => set("pout", x)} pre="$" mono ph="—"></Txt></F></div>
    </> },
    ...(p ? [{ id: "danger", l: "Remove", ic: "trash", body: <div className="dz"><div><b>Remove {p.n}</b><span>{p.tasks.length ? "Its " + p.tasks.length + " tasks fall to the next provider in line." : "It handles no tasks."}</span></div><button className="btn sm dng" onClick={() => setRm(true)}>Remove provider</button></div> }] : []),
  ];
  const tr = test && <span className={"es-tr " + test.s} title={test.d}>{test.s === "run" ? <><i className="spn"></i>Testing…</> : <><Ic n={test.s === "ok" ? "check" : "x"} s={11} w={2.4}></Ic>{test.m}</>}</span>;
  return <>
    <EditSheet d={d} fields={fields} sections={sections} create={!p} invalid={invalid}
      icon={<Mark p={mk} s={36}></Mark>} title={p ? "Edit " + p.n : "Add " + v.n} sub={p ? "Priority " + (S.providers.indexOf(p) + 1) + " of " + S.providers.length : local ? "On your network · no key" : "Hosted · needs an API key"}
      conflict={conf && <ConflictBar who="Marcus Reed" when="just now" what={["Changed the model to " + ((MODELS[v.kind] || [[v.model]])[1] || [v.model])[0]]} onReview={() => setConf(false)} onOverwrite={() => setConf(false)}></ConflictBar>}
      banner={test && test.s === "fail" && <div className="es-bn d"><Ic n="alert" s={13}></Ic><div><b>{test.m}</b><span className="mono">{test.d}</span>{test.hint && <span>{test.hint}</span>}</div></div>}
      footL={<><button className="btn sm" disabled={test && test.s === "run"} onClick={runTest}><Ic n="plug" s={12}></Ic>Test draft</button>{tr}</>}
      saveLabel={p ? "Save changes" : test && test.s === "ok" ? "Add and turn on" : "Add provider"}
      onSave={x => p ? A.saveProvFull(p.id, x) : A.createProv(x, preset, test && test.s === "ok")} onClose={onClose}></EditSheet>
    {rm && <ConfirmDialog title={"Remove " + p.n + "?"} body={(p.tasks.length ? "Its tasks fall to the next provider in line. " : "") + "The stored key is deleted."} danger confirm="Remove provider" onConfirm={() => { A.removeProv(p); onClose(); }} onClose={() => setRm(false)}></ConfirmDialog>}
  </>;
}
function ToolRuleEditor({ t, S, A, onClose, demo }) {
  const d = useDraft({ tier: t.tier, ext: t.ext, why: "" }); const v = d.v, set = d.set;
  React.useEffect(() => { if (demo === "unsaved") d.patch({ tier: t.kind === "Action" ? "Propose" : t.tier }); }, []);
  const holders = S.agents.filter(a => toolsOf(a).includes(t));
  const eff = (a, tier) => a.sim ? "Simulated" : a.shadow ? "Recorded" : TIER_L[tierCap(tier, ceilOf(a))];
  const moved = holders.filter(a => eff(a, t.tier) !== eff(a, v.tier));
  const sections = [
    { id: "rule", l: "Rule", ic: "shield", keys: ["tier", "ext"], body: <>
      <div className="kvs"><span><em>Reaches</em><b>{EGRESS[t.e][0]}</b></span><span><em>Needs</em><b>{t.needs || "No permission"}</b></span><span><em>Kind</em><b>{t.kind}</b></span></div>
      {t.kind === "Action" ? <F l="Most freedom any agent gets" hint="An agent's own ceiling can hold it lower, never higher."><Seg v={v.tier} opts={TIER_O.map(x => [x, TIER_L[x]])} onChange={x => set("tier", x)}></Seg></F> : <p className="es-note">Reads never change records, so they always run.</p>}
      <F l="Treat what it returns as outside text" hint="Outside text can't trigger an automatic action in the same run."><Seg v={v.ext} opts={[["Never", "Never"], ["Marked", "When marked"], ["Always", "Always"]]} onChange={x => set("ext", x)}></Seg></F>
      <F l="Reason" hint="Saved to the audit trail with the change."><Txt v={v.why} on={x => set("why", x)} ph="Two wrong releases last week"></Txt></F>
    </> },
    { id: "who", l: "Who's affected", ic: "users", r: <span className="es-tk">{moved.length} of {holders.length} change</span>, body: holders.length ? <div className="aff">{holders.map(a => { const b = eff(a, t.tier), n = eff(a, v.tier); return <div key={a.id} className={"aff-r" + (b !== n ? " ch" : "")}><Tile a={a} s={22}></Tile><b>{a.n}</b><span className="sp"></span>{b !== n ? <span className="cr-v"><s>{b}</s><Ic n="arrowR" s={11}></Ic><b>{n}</b></span> : <span className="dim">{b}</span>}</div>; })}</div> : <p className="es-note">No agent holds this tool.</p> },
  ];
  return <EditSheet d={d} sections={sections} fields={{ tier: { l: "Maximum", fmt: x => TIER_L[x] }, ext: { l: "Outside text" }, why: { l: "Reason" } }} invalid={d.dirty && (v.tier !== t.tier) && !v.why.trim() ? "Add a reason for the audit trail" : null}
    icon={<span className="src-i"><Ic n="tool" s={15}></Ic></span>} title={t.l} sub={<span className="mono">{t.n}</span>}
    onSave={x => { t.tier = x.tier; t.ext = x.ext; A.toast(t.l + " rule saved" + (moved.length ? " · " + moved.length + " agents changed" : "")); return false; }} onClose={onClose}></EditSheet>;
}
function Editors({ ed, S, A, onClose, demo }) {
  if (!ed) return null;
  const k = JSON.stringify(ed) + demo;
  if (ed.k === "agent") { const a = ed.id && S.agents.find(x => x.id === ed.id); return <AgentEditor key={k} a={a} tpl={ed.tpl} S={S} A={A} onClose={onClose} demo={demo}></AgentEditor>; }
  if (ed.k === "prov") { const p = ed.id && S.providers.find(x => x.id === ed.id); if (ed.id && !p) return null; return <ProviderEditor key={k} p={p} preset={ed.preset || PRESETS[0]} S={S} A={A} onClose={onClose} demo={demo}></ProviderEditor>; }
  if (ed.k === "policy") return <PolicyEditor key={k} S={S} A={A} onClose={onClose} demo={demo}></PolicyEditor>;
  if (ed.k === "tool") { const t = TOOLS.find(x => x.n === ed.n); return t ? <ToolRuleEditor key={k} t={t} S={S} A={A} onClose={onClose} demo={demo}></ToolRuleEditor> : null; }
  return null;
}
Object.assign(window, { Editors, ProviderEditor, ToolRuleEditor });
