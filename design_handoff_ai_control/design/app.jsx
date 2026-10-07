const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "prov": "configured",
  "paused": false,
  "mem": "recorded",
  "catalog": "today",
  "theme": "dark",
  "panel": "none",
  "edState": "clean",
  "tryOpen": true
}/*EDITMODE-END*/;
function Later({ tab }) {
  const l = (TABS.find(x => x[0] === tab) || [])[1];
  return <div className="tabp"><div className="empty tall"><b>{l}</b><span>Not part of this redesign pass yet. Overview through Safety are live.</span></div></div>;
}
function App({ t, setTweak }) {
  const [tab, setTabS] = React.useState(() => localStorage.getItem("aic-tab") || "overview");
  const [side, setSide] = React.useState(true);
  const [providers, setProviders] = React.useState(t.prov === "none" ? [] : t.prov === "many" ? [PROVIDERS0[1], ...PROVIDERS_MORE.slice(0, 3), PROVIDERS0[2], PROVIDERS0[0], ...PROVIDERS_MORE.slice(3)] : PROVIDERS0);
  const [agents, setAgents] = React.useState(AGENTS0);
  const [pol, setPolS] = React.useState({ earned: false, threshold: 10, learn: true, allowance: 0, share: false });
  const [tests, setTests] = React.useState({});
  const [focus, setFocus] = React.useState(null); const [fseq, setFseq] = React.useState(0);
  const [toast, setToastS] = React.useState(null);
  const [ed, setEd] = React.useState(null);
  React.useEffect(() => { const m = { "Agent editor": { k: "agent", id: "billing" }, "New agent": { k: "agent", tpl: "Desk agent" }, "Scheduled agent": { k: "agent", id: "digest" }, "Provider editor": { k: "prov", id: (providers.find(p => p.id === "vllm") || providers[0] || {}).id }, "New provider": { k: "prov", preset: PRESETS[0] }, "New local provider": { k: "prov", preset: PRESETS[4] }, "Tool rule": { k: "tool", n: "release_billing_hold" }, "Org-wide settings": { k: "policy" } }; setEd(m[t.panel] || null); }, [t.panel]);
  const closeEd = () => { setEd(null); if (t.panel !== "none") setTweak("panel", "none"); };
  const [ext, setExtS] = React.useState(EXT0);
  const [extOv, setExtOvS] = React.useState({});
  const [mems, setMems] = React.useState(t.mem === "empty" ? [] : MEMORIES0);
  const [sugs, setSugs] = React.useState(t.mem === "empty" ? [] : SUGGESTIONS0);
  const [retr, setRetrS] = React.useState(RETR0);
  const memFirst = React.useRef(true);
  React.useEffect(() => { if (memFirst.current) { memFirst.current = false; return; } setMems(t.mem === "empty" ? [] : MEMORIES0); setSugs(t.mem === "empty" ? [] : SUGGESTIONS0); }, [t.mem]);
  React.useEffect(() => { setRetrS(r => ({ ...r, src: r.src.map(s => s.k === "Memory" ? { ...s, total: mems.length, idx: Math.min(s.idx, mems.length) } : s) })); }, [mems.length]);
  const embOn = !!routeOf(TASKS.find(x => x.k === "Embedding"), providers).first;
  React.useEffect(() => {
    if (!embOn || retr.paused) return;
    const id = setInterval(() => setRetrS(r => {
      let moved = 0, fails = r.fails;
      const src = r.src.map(s => { if (!s.on) return s; const w = s.total - s.idx - s.fail - s.skip; if (w <= 0) return s; const step = Math.min(w, Math.max(1, Math.ceil(s.total / 16))); moved += step; let fail = s.fail; if (s.k === "Document" && !fail && s.idx > s.total * 0.45) { fail = 2; fails = FAIL_ITEMS; } return { ...s, idx: s.idx + step - (fail - s.fail), fail }; });
      if (!moved) return r;
      return { ...r, src, fails, spent: Math.min(r.budget, r.spent + moved * 0.00009), last: "just now" };
    }), 450);
    return () => clearInterval(id);
  }, [embOn, retr.paused]);
  const searchRef = React.useRef(); const first = React.useRef(true);
  React.useEffect(() => { if (first.current) { first.current = false; return; } setProviders(t.prov === "none" ? [] : t.prov === "many" ? [PROVIDERS0[1], ...PROVIDERS_MORE.slice(0, 3), PROVIDERS0[2], PROVIDERS0[0], ...PROVIDERS_MORE.slice(3)] : PROVIDERS0); setTests({}); }, [t.prov]);
  const setTab = k => { setTabS(k); localStorage.setItem("aic-tab", k); setFocus(null); const ws = document.querySelector(".ws"); ws && ws.scrollTo({ top: 0 }); };
  const tm = React.useRef();
  const say = (msg, undo) => { clearTimeout(tm.current); setToastS({ msg, undo, k: Date.now() }); tm.current = setTimeout(() => setToastS(null), undo ? 5000 : 2600); };
  const setProv = (id, patch) => setProviders(ps => ps.map(p => p.id === id ? { ...p, ...patch } : p));
  const test = (id, after) => {
    setTests(x => ({ ...x, [id]: { state: "run" } }));
    setTimeout(() => {
      const bad = id === "vllm";
      const ms = 280 + Math.round(Math.random() * 420);
      setTests(x => ({ ...x, [id]: bad ? { state: "fail", msg: "Could not connect · connection refused" } : { state: "ok", msg: "Connected · " + ms + " ms" } }));
      after && after(!bad);
    }, 1300);
  };
  const A = {
    toast: say,
    go: (k, f, anchor) => { setTabS(k); localStorage.setItem("aic-tab", k); setFocus(anchor || f || null); setFseq(n => n + 1); if (!f && !anchor) { const ws = document.querySelector(".ws"); ws && ws.scrollTo({ top: 0 }); } },
    scrollTo: id => { const el = document.getElementById(id), ws = document.querySelector(".ws"); el && ws && ws.scrollTo({ top: el.offsetTop - 120, behavior: "smooth" }); },
    setPaused: v => { setTweak("paused", v); say(v ? "All agents paused · they keep running in shadow" : "Agents resumed · held proposals are back in Desk"); },
    setPol: (k, v) => { setPolS(p => ({ ...p, [k]: v })); },
    test,
    setProv: (id, patch) => { const p = providers.find(x => x.id === id); setProv(id, patch); if ("on" in patch) say(p.n + (patch.on ? " is on" : " is off · its tasks move to the next provider"), patch.on ? null : () => setProv(id, { on: true })); },
    saveKey: id => { const p = providers.find(x => x.id === id); setProv(id, { key: true }); test(id, ok => { if (ok) { setProv(id, { on: true }); const gained = p.tasks.filter(k => !routeOf(TASKS.find(x => x.k === k), providers).first).map(k => TASK_L[k]); say(p.n + " is on" + (gained.length ? " · " + gained.join(", ") + " now covered" : "")); } }); },
    fix: f => { if (f.l === "Add key") A.go("providers", f.p.id); else if (f.l === "Turn on") A.setProv(f.p.id, { on: true }); else { setProv(f.p.id, { trusted: true }); say(f.p.n + " is now trusted"); } },
    edit: x => setEd(x),
    decide: (id, delta) => setAgents(xs => xs.map(x => x.id === id ? { ...x, pend: Math.max(0, x.pend + delta) } : x)),
    setPolAll: v => setPolS(v),
    addPreset: pr => setEd({ k: "prov", preset: pr }),
    createProv: (v, pr, tested) => {
      const id = pr.k + "-" + Date.now().toString(36);
      const np = { id, n: v.n, kind: v.kind, model: v.model, m: pr.m, h: pr.h, c: pr.c, on: !!tested, local: !!pr.local, key: !!v.key, trusted: v.trusted, priv: v.priv, base: v.base, tasks: v.tasks, last: tested ? { ok: true, msg: "Connected", when: "just now" } : null, wk: null };
      setProviders(ps => [...ps, np]); A.go("providers"); say(np.n + (tested ? " added and on" : " added · test it to turn it on"));
    },
    saveProvFull: (id, v) => { setProv(id, { n: v.n, kind: v.kind, model: v.model, base: v.base, tasks: v.tasks, trusted: v.trusted, priv: v.priv, ...(v.key ? { key: true } : {}) }); say("Saved · routing updated"); },
    saveAgent: (id, v) => { const held = Object.keys(v.tools).map(n => TOOLS.find(x => x.n === n)).filter(Boolean); const acts = held.filter(x => x.kind === "Action"); setAgents(xs => xs.map(x => x.id === id ? { ...x, n: v.n, d: v.d, ic: v.ic, h: v.h, shadow: v.mode === "shadow", sim: v.mode === "sim", cron: v.cron, tz: v.tz, events: v.events, access: v.access.length ? v.access.length + " roles" : "Everyone", tools: held.length, tiers: [held.length - acts.length, acts.filter(x => tierCap(v.tools[x.n], v.ceil) !== "AutoExecute").length, acts.filter(x => tierCap(v.tools[x.n], v.ceil) === "AutoExecute").length], ceil: v.ceil } : x)); say(v.n + " saved · version 8"); },
    createAgent: v => { const id = "a" + Date.now().toString(36); const held = Object.keys(v.tools).length; setAgents(xs => [...xs, { id, n: v.n, d: v.d || "New agent", trig: v.trig, ic: v.ic, h: v.h, tools: held, tiers: [held, 0, 0], pend: 0, open: 0, access: v.access.length ? v.access.length + " roles" : "Everyone", on: true, shadow: v.mode === "shadow", sim: v.mode === "sim", last: null, runs: Array(14).fill(0), cron: v.cron, tz: v.tz, next: "Thu, Oct 8 · 6:00 AM", events: v.events, ceil: v.ceil }]); A.go("agents"); say(v.n + " created in " + (v.mode === "live" ? "live" : v.mode === "sim" ? "simulation" : "shadow")); },
    addPresetDirect: pr => {
      const id = pr.k + "-" + Date.now().toString(36);
      const tasks = providers.length ? [] : ["AssistantChat", "General", "DocumentExtraction", "OperationalInsights"];
      const np = { id, n: pr.local ? (pr.k === "ollama" ? "Local Ollama" : "My model server") : pr.n, kind: pr.kind, model: pr.model, m: pr.m, h: pr.h, c: pr.c, on: false, local: !!pr.local, key: false, trusted: !!pr.trusted, priv: !!pr.local, base: pr.base || "", tasks, last: null, wk: null };
      setProviders(ps => [...ps, np]); A.go("providers", id);
      if (pr.local) test(id, ok => { if (ok) { setProv(id, { on: true }); say(np.n + " connected and on"); } });
    },
    toggleTask: (id, k) => setProviders(ps => ps.map(p => p.id === id ? { ...p, tasks: p.tasks.includes(k) ? p.tasks.filter(x => x !== k) : [...p.tasks, k] } : p)),
    moveTo: (from, to) => setProviders(ps => { const a = [...ps], i = a.findIndex(p => p.id === from), j = a.findIndex(p => p.id === to); const [x] = a.splice(i, 1); a.splice(j, 0, x); return a; }),
    moveBy: (id, d) => setProviders(ps => { const a = [...ps], i = a.findIndex(p => p.id === id), j = i + d; if (j < 0 || j >= a.length) return ps; [a[i], a[j]] = [a[j], a[i]]; return a; }),
    removeProv: p => { const snap = providers; setProviders(ps => ps.filter(x => x.id !== p.id)); say(p.n + " removed", () => setProviders(snap)); },
    toggleAgent: (id, v) => { const a = agents.find(x => x.id === id); setAgents(xs => xs.map(x => x.id === id ? { ...x, on: v } : x)); say(a.n + (v ? " is on" : " is off"), v ? null : () => setAgents(xs => xs.map(x => x.id === id ? { ...x, on: true } : x))); },
    setMode: (id, k) => setAgents(xs => xs.map(x => x.id === id ? { ...x, shadow: k === "shadow", sim: k === "sim" } : x)),
    run: a => say("Started a run of " + a.n + (a.shadow ? " · in shadow" : "")),
    setExt: (patch, msg) => { setExtS(e => ({ ...e, ...patch, today: patch.enabled ? 37 : patch.today ?? e.today })); msg && say(msg); },
    setExtOv: (id, patch, msg) => { setExtOvS(o => ({ ...o, [id]: { ...(o[id] || {}), ...patch, ...(patch.enabled ? { today: Math.round(Math.random() * 60) } : {}) } })); msg && say(msg); },
    addMem: v => { const m = { id: "m" + Date.now(), src: "you", reads: 0, last: null, rec: "Today", fresh: true, ...v }; setMems(xs => [m, ...xs]); say("Memory saved · agents read it from their next run"); },
    saveMem: m => { setMems(xs => xs.map(x => x.id === m.id ? m : x)); say("Memory updated"); },
    retireMem: m => { const snap = mems; setMems(xs => xs.filter(x => x.id !== m.id)); say("Memory retired", () => setMems(snap)); },
    approveSug: (s, content) => { setSugs(xs => xs.filter(x => x.id !== s.id)); setMems(xs => [{ id: "m" + Date.now(), kind: s.src === "Ratings" ? "Instruction" : "Procedure", content, src: s.agent || "ratings", learned: true, reads: 0, last: null, rec: "Today", fresh: true }, ...xs]); say("Memory approved · agents read it from their next run"); },
    dismissSug: s => { const snap = sugs; setSugs(xs => xs.filter(x => x.id !== s.id)); say("Dismissed · it won't be suggested again for 30 days", () => setSugs(snap)); },
    setRetr: (patch, msg) => { setRetrS(r => ({ ...r, ...patch })); msg && say(msg); },
    setSrc: (k, patch) => setRetrS(r => ({ ...r, src: r.src.map(s => s.k === k ? { ...s, ...patch } : s) })),
    reindex: k => { setRetrS(r => ({ ...r, src: r.src.map(s => s.k === k ? { ...s, idx: 0, fail: 0 } : s), fails: k === "Document" ? [] : r.fails })); say("Re-indexing " + (RETR0.src.find(s => s.k === k) || {}).n); },
    retryFail: f => { setRetrS(r => ({ ...r, fails: r.fails.filter(x => x !== f), src: r.src.map(s => s.k === "Document" ? { ...s, fail: Math.max(0, s.fail - 1), idx: s.idx + 1 } : s) })); say(f.item + " indexed"); },
    removeAgent: a => { const snap = agents; setAgents(xs => xs.filter(x => x.id !== a.id)); say(a.n + " removed", () => setAgents(snap)); },
  };
  React.useEffect(() => {
    const k = e => {
      if (/INPUT|TEXTAREA|SELECT/.test(e.target.tagName) || e.metaKey || e.ctrlKey || e.altKey) return;
      if (["1", "2", "3", "4", "5", "6", "7", "8", "9"].includes(e.key)) setTab(TABS[+e.key - 1][0]);
      else if (e.key === "/" && searchRef.current) { e.preventDefault(); searchRef.current.focus(); }
      else if (e.key.toLowerCase() === "n" && (tab === "agents" || tab === "providers" || tab === "memory")) { e.preventDefault(); window.dispatchEvent(new Event("aic-new")); }
    };
    window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k);
  }, [tab]);
  const S = { tryOpen: t.tryOpen, providers, agents, pol, tests, paused: t.paused, focus, fseq, ext, mems, sugs, retr, extOv, extCatalog: t.catalog };
  const state = !providers.length ? "none" : t.paused ? "paused" : "live";
  const counts = { agents: agents.filter(a => a.on).length + "/" + agents.length, providers: providers.length ? providers.filter(p => p.on).length + "/" + providers.length : null, extensions: (() => { const n = (ext.enabled && ext.key ? 1 : 0) + Object.values(extOv).filter(o => o.enabled && (o.key || EXT_PREVIEW.find(e => o === extOv[e.id] && e.nokey))).length; return n ? n + " on" : null; })(), memory: mems.length ? mems.length + (sugs.length ? " · " + sugs.length + " new" : "") : null, retrieval: embOn ? null : "words only", activity: PROPS.filter(p => p.st === "Pending").length + " waiting", quality: QUAL.some(q => q.delta <= -5) ? "1 regressed" : null, safety: (() => { const n = safetyFigures(agents).open.length; return n ? n + " open" : null; })() };
  return (
    <div className={"tv" + (t.theme === "dark" ? " dk" : "") + (side ? "" : " sbc")}>
      <Top onToggleSide={() => setSide(s => !s)}></Top>
      <Side></Side>
      <main className="ws">
        <div className="pg">
          <PageHead tab={tab} setTab={setTab} counts={counts} state={state}></PageHead>
          <div key={tab} className="tab-in">
            {tab === "overview" ? <Overview S={S} A={A}></Overview> : tab === "agents" ? <AgentsTab S={S} A={A} searchRef={searchRef}></AgentsTab> : tab === "providers" ? <ProvidersTab S={S} A={A} searchRef={searchRef}></ProvidersTab> : tab === "extensions" ? <ExtensionsTab S={S} A={A} searchRef={searchRef}></ExtensionsTab> : tab === "memory" ? <MemoryTab S={S} A={A} searchRef={searchRef}></MemoryTab> : tab === "retrieval" ? <RetrievalTab S={S} A={A}></RetrievalTab> : tab === "safety" ? <SafetyTab S={S} A={A} searchRef={searchRef}></SafetyTab> : tab === "quality" ? <QualityTab S={S} A={A} searchRef={searchRef}></QualityTab> : tab === "activity" ? <ActivityTab S={S} A={A} searchRef={searchRef}></ActivityTab> : tab === "audit" ? <AuditTab S={S} A={A} searchRef={searchRef}></AuditTab> : <Later tab={tab}></Later>}
          </div>
        </div>
      </main>
      <Editors ed={ed} S={S} A={A} onClose={closeEd} demo={t.edState}></Editors>
      {toast && <div className="toast" key={toast.k}><Ic n="check" s={13} w={2.4}></Ic><span>{toast.msg}</span>{toast.undo && <button onClick={() => { toast.undo(); setToastS(null); }}>Undo</button>}</div>}
    </div>
  );
}
function Root() {
  const [t, setTweak] = useTweaks(TWEAK_DEFAULTS);
  return (
    <>
      <div style={{ width: "100vw", height: "100vh" }}><App t={t} setTweak={setTweak}></App></div>
      <TweaksPanel>
        <TweakSection label="State"></TweakSection>
        <TweakRadio label="Providers" value={t.prov} options={["configured", "many", "none"]} onChange={v => setTweak("prov", v)}></TweakRadio>
        <TweakRadio label="Extensions catalog" value={t.catalog} options={["today", "preview"]} onChange={v => setTweak("catalog", v)}></TweakRadio>
        <TweakRadio label="Memory" value={t.mem} options={["recorded", "empty"]} onChange={v => setTweak("mem", v)}></TweakRadio>
        <TweakToggle label="All agents paused" value={t.paused} onChange={v => setTweak("paused", v)}></TweakToggle>
        <TweakSection label="Panels"></TweakSection>
        <TweakSelect label="Open panel" value={t.panel} options={["none", "Agent editor", "Scheduled agent", "New agent", "Provider editor", "New provider", "New local provider", "Tool rule", "Org-wide settings"]} onChange={v => setTweak("panel", v)}></TweakSelect>
        <TweakRadio label="Editor state" value={t.edState} options={["clean", "unsaved", "conflict"]} onChange={v => setTweak("edState", v)}></TweakRadio>
        <TweakToggle label="Try it panel open" value={t.tryOpen} onChange={v => setTweak("tryOpen", v)}></TweakToggle>
        <TweakSection label="Appearance"></TweakSection>
        <TweakRadio label="Theme" value={t.theme} options={["dark", "light"]} onChange={v => setTweak("theme", v)}></TweakRadio>
      </TweaksPanel>
    </>
  );
}
ReactDOM.createRoot(document.getElementById("root")).render(<Root></Root>);
