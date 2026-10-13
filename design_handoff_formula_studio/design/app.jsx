const { useState: S, useMemo: M, useEffect: E, useRef: R, useCallback: C } = React;

const strip = (d) => ({ ...d, variables: d.variables.map(({ id, fresh, described, ...v }) => v), breakdown: d.breakdown.map(({ id, nameTouched, ...b }) => b) });
const same = (a, b) => JSON.stringify(a) === JSON.stringify(b);
const withIds = (d) => ({ ...d, variables: d.variables.map((v) => ({ id: v.id || fsId("v"), ...v })), breakdown: d.breakdown.map((b) => ({ id: b.id || fsId("b"), ...b })) });
const STATUS = { Draft: { label: "Draft", tone: "muted" }, InReview: { label: "In review", tone: "info" }, Active: { label: "Active", tone: "success" }, Inactive: { label: "Archived", tone: "muted" } };
const REVIEWERS = [{ id: "pr", name: "Priya Raman", role: "Pricing lead" }, { id: "mw", name: "Marcus Webb", role: "Controller" }];

function useToasts() {
  const [list, setList] = S([]);
  const push = C((text, action) => { const id = fsId("t"); setList((l) => [...l.slice(-2), { id, text, action }]); setTimeout(() => setList((l) => l.filter((t) => t.id !== id)), action ? 6000 : 3200); }, []);
  const view = <div className="toasts">{list.map((t) => <div key={t.id} className="toast"><span>{t.text}</span>{t.action && <button onClick={() => { t.action.run(); setList((l) => l.filter((x) => x.id !== t.id)); }}><FI n="undo" s={12} />{t.action.label}</button>}</div>)}</div>;
  return [push, view];
}

function Modal({ title, sub, children, onClose, actions, wide }) {
  E(() => { const k = (e) => e.key === "Escape" && onClose(); window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k); }, [onClose]);
  return (
    <div className="mdl-bk" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className={"mdl" + (wide ? " wide" : "")} role="dialog" aria-modal="true">
        <div className="mdl-h"><div><h3>{title}</h3>{sub && <p>{sub}</p>}</div><button className="ib" onClick={onClose} aria-label="Close"><FI n="x" s={15} /></button></div>
        <ScrollArea className="mdl-b-sa" viewportClassName="mdl-b" maskVariant="card">{children}</ScrollArea>
        {actions && <div className="mdl-f">{actions}</div>}
      </div>
    </div>
  );
}

function Readiness({ checks, compact }) {
  return <ul className={"rd" + (compact ? " compact" : "")}>{checks.map((c) => <li key={c.key} className={c.status}><FI n={c.status === "pass" ? "okc" : c.status === "warn" ? "alert" : "xc"} s={13} /><span><b>{c.label}</b>{c.detail && <em>{c.detail}</em>}</span></li>)}</ul>;
}

function SubmitDialog({ d, saved, baseline, checks, onClose, onSubmit }) {
  const bt = M(() => (baseline ? fsBacktest(baseline, d) : null), [baseline, d]);
  const [note, setNote] = S("");
  const [busy, setBusy] = S(true);
  const [rev, setRev] = S("pr");
  const write = () => { setBusy(true); novaSummary(baseline || saved, d, bt || { n: 0, changed: 0, delta: 0, pct: 0, failed: 0 }).then((t) => { setNote(t); setBusy(false); }); };
  E(write, []);
  const blocked = checks.some((c) => c.status === "fail");
  return (
    <Modal title="Submit for review" sub="The reviewer approves exactly what is saved. Approval makes this version rate shipments." onClose={onClose} wide
      actions={<><span className="mdl-fl">{blocked ? "Fix the blocking checks to submit." : "You'll be notified when it's reviewed."}</span><button className="tb tb-quiet tb-s-sm" onClick={onClose}>Cancel</button><button className="tb tb-default tb-s-sm" disabled={blocked || busy} onClick={() => onSubmit(rev, note)}><FI n="send" s={13} />Submit</button></>}>
      <div className="sub-g">
        <div>
          <div className="mdl-l">Readiness</div>
          <Readiness checks={checks} />
          <div className="mdl-l">Reviewer</div>
          <div className="rv">{REVIEWERS.map((r) => <button key={r.id} className={rev === r.id ? "on" : ""} onClick={() => setRev(r.id)}><span className="av">{r.name.split(" ").map((x) => x[0]).join("")}</span><span><b>{r.name}</b><em>{r.role}</em></span>{rev === r.id && <FI n="check" s={13} />}</button>)}</div>
        </div>
        <div>
          <div className="mdl-l">Note for the reviewer<span className="sp"></span><button className="tb tb-ghost tb-s-xs nv-btn" disabled={busy} onClick={write}><span className="dm"></span>{busy ? "Writing…" : "Rewrite"}</button></div>
          <div className={"sub-note" + (busy ? " busy" : "")}><AutoText value={busy ? "" : note} onChange={setNote} placeholder={busy ? "Nova is reading the change and the backtest…" : ""} minRows={4} /></div>
          <div className="mdl-l">Impact on recent freight</div>
          {bt ? (
            <div className="imp">
              <div className="imp-f"><span><b className="mono">{bt.n}</b><em>re-rated</em></span><span><b className="mono">{bt.changed}</b><em>would change</em></span><span className={bt.delta > 0 ? "up" : bt.delta < 0 ? "dn" : ""}><b className="mono">{bt.delta >= 0 ? "+" : "−"}{fsMoney(Math.abs(bt.delta), 0)}</b><em>{bt.pct >= 0 ? "+" : ""}{bt.pct.toFixed(1)}% revenue</em></span></div>
              {bt.movers.map((m) => <div key={m.id} className="imp-r"><span className="mono">{m.id}</span><em>{m.v.totalDistance} mi · {m.v.totalStops} stops{m.v.hasHazmat ? " · hazmat" : ""}</em><span className="mono">{fsMoney(m.before)} → <b>{fsMoney(m.after)}</b></span></div>)}
              {bt.changed === 0 && <p className="imp-none">No recent shipment would rate differently.</p>}
            </div>
          ) : <p className="imp-none">First version. Nothing rates with this template yet.</p>}
        </div>
      </div>
    </Modal>
  );
}

function BacktestDialog({ baseline, d, onClose }) {
  const bt = M(() => fsBacktest(baseline, d), [baseline, d]);
  return (
    <Modal title="Backtest" sub={`Re-rates ${bt.n} shipments from the last 30 days with the active version and with the editor's.`} onClose={onClose} actions={<button className="tb tb-default tb-s-sm" onClick={onClose}>Done</button>}>
      <div className="imp">
        <div className="imp-f"><span><b className="mono">{bt.changed}</b><em>would change</em></span><span><b className="mono">{bt.failed}</b><em>fail to rate</em></span><span className={bt.delta > 0 ? "up" : bt.delta < 0 ? "dn" : ""}><b className="mono">{bt.delta >= 0 ? "+" : "−"}{fsMoney(Math.abs(bt.delta), 0)}</b><em>{bt.pct >= 0 ? "+" : ""}{bt.pct.toFixed(1)}%</em></span></div>
        {bt.movers.map((m) => <div key={m.id} className="imp-r"><span className="mono">{m.id}</span><em>{m.v.totalDistance} mi · {m.v.totalStops} stops</em><span className="mono">{fsMoney(m.before)} → <b>{fsMoney(m.after)}</b></span></div>)}
      </div>
    </Modal>
  );
}

function Studio({ initMode, initStatus, theme, chrome }) {
  const [mode, setMode] = S(initMode);
  const [status, setStatus] = S(initMode === "edit" ? initStatus : null);
  const [version, setVersion] = S(initMode === "edit" ? 4 : null);
  const start = initMode === "edit" ? withIds(FS_EDIT_TEMPLATE) : withIds(FS_EMPTY_TEMPLATE);
  const [saved, setSaved] = S(start);
  const baseline = initMode === "edit" ? FS_EDIT_TEMPLATE : null;
  const [d, setD] = S(start);
  const [scen, setScen] = S(initMode === "edit" ? FS_EDIT_SCENARIOS : []);
  const [tab, setTab] = S("preview");
  const [src, setSrc] = S("sample");
  const [shipId, setShipId] = S(null);
  const [ov, setOv] = S({});
  const [cursor, setCursor] = S({ ln: 1, col: 1 });
  const [focus, setFocus] = S(null);
  const [explain, setExplain] = S(null);
  const [thread, setThread] = S([]);
  const [busy, setBusy] = S(false);
  const [describing, setDescribing] = S(false);
  const [suggestion, setSuggestion] = S(null);
  const [addedNames, setAddedNames] = S(new Set());
  const [dlg, setDlg] = S(null);
  const [menu, setMenu] = S(false);
  const [usage, setUsage] = S(false);
  const [flash, setFlash] = S(false);
  const [pinKey, setPinKey] = S(null);
  const undoRef = R({});
  const edRef = R(), mainRef = R();
  const [toast, toasts] = useToasts();

  const set = C((patch) => setD((x) => ({ ...x, ...patch })), []);
  const ctx = M(() => fsContext(d.variables), [d.variables]);
  const an = M(() => fsAnalyze(d.expression, ctx), [d.expression, ctx]);
  const ship = FS_SHIPMENTS.find((s) => s.id === shipId);
  const run = M(() => fsRun(d, ship ? ship.values : null, ov, !!ship), [d, ship, ov]);
  const results = M(() => scen.map((sc) => fsScenarioRun(d, sc)), [scen, d]);
  const dirty = !same(strip(d), strip(saved));
  const material = !same([d.expression, strip(d).variables, strip(d).breakdown, d.policy], [saved.expression, strip(saved).variables, strip(saved).breakdown, saved.policy]);
  const errs = an.issues.filter((i) => i.sev === "error").length, warns = an.issues.filter((i) => i.kind === "nullable").length;
  const passed = results.filter((r) => r.pass).length;

  const checks = M(() => {
    const c = [];
    c.push({ key: "details", label: "Name and description", status: d.name.trim() && d.description.trim() ? "pass" : "fail", detail: !d.name.trim() ? "Name is missing" : !d.description.trim() ? "Description is missing" : null });
    c.push({ key: "expr", label: "Expression is valid", status: !d.expression.trim() || errs || !run.ok ? "fail" : "pass", detail: !d.expression.trim() ? "Empty" : errs ? `${errs} error${errs > 1 ? "s" : ""}` : !run.ok ? "Fails on the preview load" : null });
    c.push({ key: "null", label: "Safe on every shipment", status: warns ? "warn" : "pass", detail: warns ? `${warns} value${warns > 1 ? "s" : ""} can be empty` : null });
    c.push({ key: "scen", label: "Scenarios pass", status: !scen.length ? "fail" : passed === scen.length ? "pass" : "fail", detail: !scen.length ? "Add at least one" : `${passed} of ${scen.length}` });
    if (d.breakdown.length) c.push({ key: "bd", label: "Breakdown reconciles", status: run.reconcile && run.reconcile.balanced ? (run.reconcile.clampMismatch ? "warn" : "pass") : "warn", detail: run.reconcile && run.reconcile.clampMismatch ? "Not at the minimum" : run.reconcile && !run.reconcile.balanced ? "Lines don't add up" : null });
    if (dirty && mode === "edit") c.push({ key: "saved", label: "Changes saved", status: "fail", detail: "Review covers what is saved" });
    return c;
  }, [d, errs, warns, run, scen.length, passed, dirty, mode]);

  const doSave = C((force) => {
    if (mode === "create") {
      if (!d.name.trim() || !d.description.trim() || !d.expression.trim() || errs) {
        setFlash(true); setTimeout(() => setFlash(false), 1600);
        mainRef.current.scrollTo({ top: 0, behavior: "smooth" });
        toast(!d.name.trim() ? "Name the template to create it" : !d.description.trim() ? "Add a description to create it" : "The expression needs to be valid first");
        return;
      }
      setSaved(d); setMode("edit"); setStatus("Draft"); setVersion(1);
      toast(`Created “${d.name}” as a draft${scen.length ? ` with ${scen.length} scenario${scen.length > 1 ? "s" : ""}` : ""}`);
      return;
    }
    if (!dirty) return;
    if (!force && material && (status === "Active" || status === "InReview")) { setDlg("confirm"); return; }
    setSaved(d);
    if (material) { setVersion((v) => v + 1); if (status === "Active" || status === "InReview") setStatus("Draft"); }
    toast(material ? `Saved as v${version + 1}${status === "Active" ? " · back to Draft" : ""}` : "Changes saved");
    setDlg(null);
  }, [mode, d, errs, dirty, material, status, version, scen.length]);

  E(() => {
    const k = (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "s") { e.preventDefault(); doSave(); }
      if ((e.metaKey || e.ctrlKey) && e.key === "j") { e.preventDefault(); setTab((t) => (t === "nova" ? "preview" : "nova")); }
      if (e.key === "/" && !/INPUT|TEXTAREA/.test(document.activeElement.tagName)) { e.preventDefault(); setTab("reference"); setTimeout(() => document.getElementById("ref-search")?.focus(), 30); }
    };
    window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k);
  }, [doSave]);

  const askNova = async (text) => {
    text = text || (d.expression.trim() ? "Explain this formula, term by term" : "");
    setTab("nova");
    if (!text) return;
    const id = fsId("m");
    const lines = d.expression.split("\n").filter(Boolean).length;
    setThread((t) => [...t, { id: id + "q", role: "user", text }, { id, role: "nova", at: Date.now(), steps: [{ text: "Read the editor", meta: lines ? `${lines} lines · ${d.variables.length} inputs` : "empty", done: true }, { text: /fix|error/i.test(text) ? "Working out the fix" : /edge|scenario|test/i.test(text) ? "Looking for the formula's edges" : d.expression.trim() ? "Thinking it through" : "Drafting the formula", done: false }] }]);
    setBusy(true);
    const r = await novaAsk(text, d, run);
    const steps = [{ text: "Read the editor", meta: lines ? `${lines} lines · ${d.variables.length} inputs` : "empty", done: true }];
    if (r.proposal) steps.push({ text: d.expression.trim() ? "Drafted a change" : "Drafted a formula", done: true }, { text: "Checked names against the shipment schema", meta: `${FS_VARS.length} variables`, done: true }, { text: "Priced loads with the formula engine", meta: `${FS_SHIPMENTS.length + 1} loads`, done: true });
    else if (r.scenarios) steps.push({ text: "Found edges in the formula", done: true }, { text: "Priced each with the formula engine", meta: `${r.scenarios.length} loads`, done: true });
    else steps.push({ text: "Answered", done: true });
    setThread((t) => t.map((m) => (m.id === id ? { ...m, steps, reply: r.reply, proposal: r.proposal ? { ...r.proposal, variables: r.proposal.variables || [] } : null, scenarios: r.scenarios || null, offline: !r.live } : m)));
    setBusy(false);
  };
  const insertProposal = (mid, next) => {
    undoRef.current[mid] = d;
    setD(next);
    setThread((t) => t.map((m) => (m.id === mid ? { ...m, state: "inserted", proposal: { ...m.proposal, before: d.expression } } : m)));
    toast("Inserted into the editor. Test it, then save.", { label: "Undo", run: () => undoProposal(mid) });
  };
  const undoProposal = (mid) => { const prev = undoRef.current[mid]; if (prev) setD(prev); setThread((t) => t.map((m) => (m.id === mid ? { ...m, state: null } : m))); };
  const addScenarios = (list) => {
    setScen((s) => [...s, ...list.map((x) => ({ id: fsId("sc"), name: x.name, inputs: x.inputs, expected: x.expected, fresh: true }))]);
    setAddedNames((a) => { const n = new Set(a); list.forEach((x) => n.add(x.name)); return n; });
    toast(list.length > 1 ? `${list.length} scenarios added` : `Scenario “${list[0].name}” added`);
  };
  const pin = () => {
    const inputs = ship ? { ...ship.values, ...ov } : { ...ov };
    const name = ship ? `${ship.id} · ${ship.lane}` : Object.keys(ov).length ? `Sample load · ${Object.entries(ov).slice(0, 2).map(([k, v]) => `${k} ${fsFmt(v)}`).join(", ")}` : "Sample load, 742 mi · 3 stops";
    addScenarios([{ name, inputs, expected: run.policy.amount }]);
    setPinKey(JSON.stringify([shipId, ov, run.policy.amount]));
    setTab("scenarios");
  };
  const addInput = (name) => {
    const bool = /^(is|has|requires|uses)[A-Z]/.test(name);
    set({ variables: [...d.variables, { id: fsId("v"), name, type: bool ? "Boolean" : "Number", defaultValue: bool ? false : null, required: true, description: "", fresh: true }] });
    toast(`Added ${name} as an input. Set its default below.`);
  };
  const fixNullable = (i) => set({ expression: d.expression.slice(0, i.s) + i.fix + d.expression.slice(i.e) });
  const pickStart = (s, kind) => {
    const prev = d;
    if (kind === "org") { const full = s.name === FS_EDIT_TEMPLATE.name ? FS_EDIT_TEMPLATE : { ...FS_EMPTY_TEMPLATE, expression: s.expression, description: "" }; setD(withIds({ ...full, name: "Copy of " + s.name, policy: full.policy || d.policy })); }
    else setD(withIds({ ...d, expression: s.expression, type: s.type, variables: s.variables.map((v) => ({ ...v })), description: d.description || s.description, name: d.name || s.name }));
    toast(`Started from “${s.name}”`, { label: "Undo", run: () => setD(prev) });
  };
  const doExplain = async () => { setExplain({ busy: true, items: [] }); const items = await novaExplain(d); setExplain({ busy: false, items }); };
  const doDescribe = async () => {
    setDescribing(true);
    const map = await novaDescribe(d);
    setD((x) => ({ ...x, variables: x.variables.map((v) => (!v.description && map[v.name] ? { ...v, description: map[v.name], described: true } : v)) }));
    setDescribing(false);
  };
  const doSuggest = async () => { setSuggestion("busy"); const r = await novaAsk("Suggest 4 edge-case scenarios to test this formula", d, run); setSuggestion(r.scenarios && r.scenarios.length ? r.scenarios : novaAskFallbackScen()); };
  const novaAskFallbackScen = () => [{ name: "Local move, 0 miles", inputs: { totalDistance: 0, totalStops: 2 } }, { name: "Five stops with hazmat", inputs: { totalStops: 5, hasHazmat: true } }];
  const scrollTo = (id) => { const el = document.getElementById("sec-" + id); if (el && mainRef.current) mainRef.current.scrollTo({ top: el.offsetTop - 24, behavior: "smooth" }); };
  const exportJson = () => {
    const blob = new Blob([JSON.stringify({ format: "trenova.formula-template", version: 1, templates: [{ ...strip(d), status, versionNumber: version, testCases: scen.map(({ fresh, id, ...s }) => s) }] }, null, 2)], { type: "application/json" });
    const a = document.createElement("a"); a.href = URL.createObjectURL(blob); a.download = (d.name || "formula-template").toLowerCase().replace(/[^\w]+/g, "-") + ".json"; a.click();
    toast("Template exported");
  };

  const outline = [
    { id: "details", icon: "file", label: "Details", state: d.name.trim() && d.description.trim() ? "ok" : flash ? "bad" : "todo" },
    { id: "expression", icon: "code", label: "Expression", state: !d.expression.trim() ? "todo" : errs || !run.ok ? "bad" : warns ? "warn" : "ok", count: errs || null },
    { id: "inputs", icon: "input", label: "Inputs", state: d.variables.some((v) => v.name && !an.used.customs.has(v.name)) ? "warn" : d.variables.length ? "ok" : "none", count: d.variables.length || null },
    { id: "policy", icon: "shield", label: "Charge policy", state: d.policy.min != null || d.policy.max != null ? "ok" : "none" },
    { id: "breakdown", icon: "layers", label: "Breakdown", state: !d.breakdown.length ? "none" : run.reconcile && run.reconcile.balanced && !run.reconcile.clampMismatch ? "ok" : "warn", count: d.breakdown.length || null },
  ];
  const st = status && STATUS[status];
  const materialNote = mode === "edit" && material && (status === "Active" || status === "InReview") ? (
    <div className="mat"><FI n="info" s={14} /><span>{status === "Active" ? <>Saving this edit returns the template to <b>Draft</b>. The {FS_USAGE.reduce((s, u) => s + u.count, 0) - 214} lanes and matrices that use it keep rating with v{version} until a new version is approved.</> : <>Saving this edit cancels the review in progress and returns the template to <b>Draft</b>.</>}</span></div>
  ) : null;
  const headActions = (<>
        {mode === "edit" && (status === "Draft" || status === "Inactive") && <button className="tb tb-outline tb-s-sm" title={dirty ? "Save first; review covers what is saved" : ""} disabled={dirty} onClick={() => setDlg("submit")}><FI n="send" s={13} />{status === "Inactive" ? "Reactivate via review" : "Submit for review"}</button>}
        {mode === "edit" && status === "InReview" && <>
          <button className="tb tb-outline tb-s-sm ok-t" disabled={dirty || checks.some((c) => c.status === "fail")} onClick={() => { setStatus("Active"); toast(`Approved · v${version} is active`); }}><FI n="check" s={13} />Approve</button>
          <button className="tb tb-outline tb-s-sm" onClick={() => { setStatus("Draft"); toast("Sent back to the author with your comment"); }}><FI n="msg" s={13} />Request changes</button>
        </>}
        {mode === "edit" && (
          <DropdownMenu align="end" trigger={(p) => <button className="ib" aria-label="More actions" {...p}><FI n="more" s={16} /></button>}>
            <DropdownMenuItem icon="clock" title="Version history" description={`v1 – v${version}`} onClick={() => toast("Version history opens beside the studio")} />
            <DropdownMenuItem icon="rewind" title="Backtest" description="Re-rate recent shipments" disabled={!baseline} onClick={() => setDlg("backtest")} />
            <DropdownMenuSeparator />
            <DropdownMenuItem icon="branch" title="Fork template" onClick={() => toast("Forked as a new draft")} />
            <DropdownMenuItem icon="download" title="Export JSON" onClick={exportJson} />
            <DropdownMenuItem icon="upload" title="Import templates" onClick={() => toast("Choose an exported JSON file")} />
            {status === "InReview" && <><DropdownMenuSeparator /><DropdownMenuItem icon="x" color="danger" title="Reject and archive" onClick={() => { setStatus("Inactive"); toast("Rejected and archived"); }} /></>}
          </DropdownMenu>
        )}
  </>);
  const pinned = pinKey === JSON.stringify([shipId, ov, run.policy && run.policy.amount]);

  return (
    <div className={"us fs" + (theme === "dark" ? " dk" : "") + (chrome === "classic" ? " classic" : "")}>
      {chrome === "classic" ? (
      <header className="top">
        <button className="ib" aria-label="Back to formula templates"><FI n="back" s={16} /></button>
        <div className="crumb"><span>Formula templates</span><FI n="chevR" s={12} /><b>{mode === "create" && !d.name ? "New template" : d.name || "Untitled"}</b></div>
        {st && <span className={"pill " + st.tone}><i></i>{st.label}</span>}
        {version && <span className="bdg mono">v{version}</span>}
        {mode === "edit" && status !== "Draft" && (
          <HoverCard align="start" trigger={(p) => <span className="bdg use-t" {...p}><FI n="users" s={11} />In use · {FS_USAGE[0].count + FS_USAGE[1].count}</span>}><UsageCard version={version} /></HoverCard>
        )}
        <span className="sp"></span>
        <span className={"svs" + (dirty ? " dirty" : "")}><i></i>{mode === "create" ? "Not created yet" : dirty ? "Unsaved changes" : "All changes saved"}</span>
        {headActions}
        <button className="tb tb-default tb-s-sm" disabled={mode === "edit" && !dirty} onClick={() => doSave()}>{mode === "create" ? "Create template" : "Save changes"}<span className="kbd inv">⌘S</span></button>
      </header>
      ) : <StudioTop d={d} mode={mode} status={status} version={version} dirty={dirty} material={material} actions={headActions} onSave={doSave} />}

      <div className="body">
        {chrome === "classic" ? (
        <nav className="ol">
          <div className="ol-g">Template</div>
          {outline.map((o) => <button key={o.id} className="ol-i" onClick={() => scrollTo(o.id)}><FI n={o.icon} s={14} /><span>{o.label}</span>{o.count && <em className="mono">{o.count}</em>}<i className={"st " + o.state}></i></button>)}
          <button className="ol-i" onClick={() => setTab("scenarios")}><FI n="beaker" s={14} /><span>Scenarios</span>{scen.length > 0 && <em className="mono">{passed}/{scen.length}</em>}<i className={"st " + (!scen.length ? "todo" : passed === scen.length ? "ok" : "bad")}></i></button>
          <div className="ol-g">{mode === "create" ? "Before you create" : status === "InReview" ? "Before approval" : "Before review"}</div>
          <Readiness checks={mode === "create" ? checks.filter((c) => c.key !== "scen") : checks} compact />
          <div className="ol-k"><span><span className="kbd">⌘S</span>Save</span><span><span className="kbd">⌘J</span>Nova</span><span><span className="kbd">/</span>Reference</span></div>
        </nav>
        ) : <RateTrace d={d} an={an} run={run} ship={ship} ov={ov} results={results} checks={checks} mode={mode} status={status} mainRef={mainRef} onJump={scrollTo} onTab={setTab} />}

        <ScrollArea as="main" className="main" viewportRef={mainRef} maskVariant="background">
          <div className="col">
            <DetailsHero d={d} set={set} issues={flash ? { name: !d.name.trim(), description: !d.description.trim() } : {}} />
            <ExpressionSection d={d} set={set} an={an} ctx={ctx} edRef={edRef} focus={focus} setFocus={setFocus} cursor={cursor} setCursor={setCursor} run={run} mode={mode}
              explain={explain} onExplain={doExplain} onCloseExplain={() => { setExplain(null); setFocus(null); }} onAsk={askNova} onDescribe={(q) => askNova(`Write a formula template: ${q}`)} onStandard={pickStart} novaBusy={busy}
              onAddInput={addInput} onFix={fixNullable} materialNote={materialNote} />
            <InputsSection d={d} set={set} used={an.used.customs} onDescribe={doDescribe} describing={describing} />
            <PolicySection d={d} set={set} run={run} />
            <BreakdownSection d={d} set={set} ctx={ctx} run={run} />
          </div>
        </ScrollArea>

        <aside className="rail">
          <div className="tabs" role="tablist">
            {[["preview", "Preview"], ["scenarios", "Scenarios"], ["reference", "Reference"], ["nova", "Nova"]].map(([k, l]) => (
              <button key={k} role="tab" aria-selected={tab === k} className={(tab === k ? "on " : "") + (k === "nova" ? "nv-tab" : "")} onClick={() => setTab(k)}>
                {k === "nova" && <span className="dm"></span>}{l}
                {k === "scenarios" && scen.length > 0 && <em className={"mono " + (passed === scen.length ? "ok" : "bad")}>{passed}/{scen.length}</em>}
                {k === "nova" && busy && <span className="spin"></span>}
              </button>
            ))}
          </div>
          <ScrollArea className="rail-b" maskVariant="card">
            {tab === "preview" && <PreviewPane d={d} an={an} ctx={ctx} run={run} src={src} setSrc={setSrc} shipId={shipId} setShipId={setShipId} overrides={ov} setOverrides={setOv} onPin={pin} pinned={pinned} />}
            {tab === "scenarios" && <ScenariosPane d={d} saved={saved} onAsk={askNova} results={results} onAdd={addScenarios} onRemove={(id) => setScen((s) => s.filter((x) => x.id !== id))} onAccept={(id, amt) => setScen((s) => s.map((x) => (x.id === id ? { ...x, expected: amt } : x)))} onSuggest={doSuggest} suggestion={suggestion} canSave={mode === "edit"} addedNames={addedNames} overrides={ship ? { ...ship.values, ...ov } : ov} run={run} />}
            {tab === "reference" && <ReferencePane ctx={ctx} onInsert={(t, off) => edRef.current && edRef.current.insert(t, off)} />}
            {tab === "nova" && <NovaPane thread={thread} busy={busy} draft={d} onAsk={askNova} onInsert={insertProposal} onUndo={undoProposal} onDismiss={(mid) => setThread((t) => t.map((m) => (m.id === mid ? { ...m, state: "dismissed" } : m)))} onAddScenarios={addScenarios} addedScenarios={addedNames} canSave={mode === "edit"} inputs={d.variables.length} />}
          </ScrollArea>
        </aside>
      </div>

      {dlg === "confirm" && (
        <Modal title="Save and return to Draft?" sub={`This changes what the template computes. v${version} keeps rating shipments until the new version is approved.`} onClose={() => setDlg(null)}
          actions={<><button className="tb tb-quiet tb-s-sm" onClick={() => setDlg(null)}>Cancel</button><button className="tb tb-default tb-s-sm" onClick={() => doSave(true)}>Save and return to Draft</button></>}>
          <div className="use-l">{FS_USAGE.map((u) => <div key={u.type}><span>{u.type}</span><b className="mono">{u.count}</b></div>)}</div>
        </Modal>
      )}
      {dlg === "submit" && <SubmitDialog d={d} saved={saved} baseline={baseline} checks={checks} onClose={() => setDlg(null)} onSubmit={(rev) => { setDlg(null); setStatus("InReview"); toast(`Sent to ${REVIEWERS.find((r) => r.id === rev).name} for review`); }} />}
      {dlg === "backtest" && <BacktestDialog baseline={baseline} d={d} onClose={() => setDlg(null)} />}
      {toasts}
    </div>
  );
}

const FS_TWEAKS = /*EDITMODE-BEGIN*/{"theme":"light","mode":"edit","status":"Draft","chrome":"trace"}/*EDITMODE-END*/;
function App() {
  const [t, setTweak] = useTweaks(FS_TWEAKS);
  return (
    <>
      <Studio key={t.mode + t.status} initMode={t.mode} initStatus={t.status} theme={t.theme} chrome={t.chrome} />
      <TweaksPanel>
        <TweakSection label="Formula studio" />
        <TweakRadio label="Form" value={t.mode} options={["create", "edit"]} onChange={(v) => setTweak("mode", v)} />
        <TweakRadio label="Status (edit)" value={t.status} options={["Draft", "InReview", "Active"]} onChange={(v) => setTweak("status", v)} />
        <TweakRadio label="Sidebar + top bar" value={t.chrome} options={["trace", "classic"]} onChange={(v) => setTweak("chrome", v)} />
        <TweakRadio label="Theme" value={t.theme} options={["light", "dark"]} onChange={(v) => setTweak("theme", v)} />
      </TweaksPanel>
    </>
  );
}
ReactDOM.createRoot(document.getElementById("root")).render(<App />);
