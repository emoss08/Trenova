const { useState: rS, useMemo: rM, useEffect: rE } = React;

const SRC_TONE = { shipment: "info", computed: "violet", input: "success", override: "warn", default: "muted", sample: "muted" };

function ValueIn({ name, type, value, onChange }) {
  if (type === "Boolean") return <span className="pv-b"><Sw sm on={!!value} onChange={onChange} label={name} /><em>{value ? "true" : "false"}</em></span>;
  if (type === "String") return <input className="pv-in mono" value={value ?? ""} placeholder="empty" onChange={(e) => onChange(e.target.value)} />;
  return <input className="pv-in mono" inputMode="decimal" value={value ?? ""} placeholder="empty" onChange={(e) => { const v = e.target.value; onChange(v === "" ? null : isNaN(Number(v)) || v.endsWith(".") ? v : Number(v)); }} />;
}

function PreviewPane({ d, an, ctx, run, src, setSrc, shipId, setShipId, overrides, setOverrides, onPin, pinned }) {
  const [showAll, setShowAll] = rS(false);
  const [receipt, setReceipt] = rS(false);
  const ship = FS_SHIPMENTS.find((s) => s.id === shipId);
  const values = run ? run.values : fsValues(ship ? ship.values : null, d.variables, overrides);
  const read = [...an.used.customs, ...an.used.vars];
  const rest = FS_VARS.map((v) => v.name).filter((n) => !read.includes(n));
  const shown = showAll ? [...read, ...rest] : read;
  const docOf = (n) => ctx.vars.get(n) || { type: "Number" };
  const sourceOf = (n) => (n in overrides ? (src === "real" ? "override" : "input") : d.variables.some((v) => v.name === n) ? "default" : src === "real" ? (FS_FIELD_SOURCE.has(n) ? "shipment" : "computed") : "sample");
  const pr = run && run.ok && run.policy;
  return (
    <div className="pane">
      <div className="pv-src">
        <Seg value={src} onChange={(v) => { setSrc(v); if (v === "sample") setShipId(null); else if (!shipId) setShipId(FS_SHIPMENTS[0].id); setOverrides({}); }} options={[{ value: "sample", label: "Sample load" }, { value: "real", label: "Real shipment" }]} />
        <span className="sp"></span>
        {run && run.ms != null && <span className="pv-ms mono">{run.ms < 1 ? "<1" : run.ms.toFixed(1)} ms</span>}
      </div>
      {src === "real" && (
        <div className="pv-ships">{FS_SHIPMENTS.map((s) => (
          <button key={s.id} className={"pv-ship" + (s.id === shipId ? " on" : "")} onClick={() => { setShipId(s.id); setOverrides({}); }}>
            <b className="mono">{s.id}</b><span>{s.lane}</span><em>{s.meta}</em>
          </button>))}
        </div>
      )}

      <div className={"res" + (!run || run.empty ? " idle" : run.ok ? " ok" : " bad")}>
        {!run || run.empty ? (
          <div className="res-idle"><FI n="beaker" s={20} /><span>Write an expression and the charge appears here as you type.</span></div>
        ) : run.ok && pr ? (
          <>
            <div className="res-top"><span>Billed</span><span className="sp"></span>
              <button className="tb tb-ghost tb-s-xs" onClick={onPin} disabled={pinned}><FI n={pinned ? "check" : "pin"} s={12} />{pinned ? "Pinned" : "Pin as scenario"}</button></div>
            <div className="res-amt mono">{fsMoney(pr.amount, d.policy.mode === "None" ? 4 : d.policy.precision)}</div>
            <div className="res-notes">
              {pr.clamp ? <span className="rn warn"><FI n="shield" s={12} />Formula produced {fsMoney(pr.raw)}; raised to the {fsMoney(pr.clamp.limit)} {pr.clamp.bound}</span> : <span className="rn ok"><FI n="shield" s={12} />Within the charge policy</span>}
              <span className="rn">{d.policy.mode === "None" ? "Not rounded" : pr.roundedChange ? `Rounded ${FS_ROUNDING.find((r) => r.value === d.policy.mode).label.toLowerCase()} from ${fsFmt(pr.clamped)}` : "Rounding made no change"}</span>
            </div>
            {run.lines && run.lines.length > 0 && (
              <div className="res-lines">{run.lines.map((l) => <div key={l.id}><span>{l.label || l.name || "Untitled line"}</span><b className={"mono" + (l.ok ? "" : " bad")}>{l.ok ? fsMoney(l.amount) : "Fails"}</b></div>)}
                {run.reconcile && <div className={"rl-f" + (run.reconcile.balanced && !run.reconcile.clampMismatch ? "" : " warn")}><span>{run.reconcile.clampMismatch ? "Lines don't match the billed minimum" : run.reconcile.balanced ? "Lines explain the total" : "Lines don't add up"}</span><b className="mono">{fsMoney(run.reconcile.sum)}</b></div>}
              </div>
            )}
            <button className="res-rc" onClick={() => setReceipt(!receipt)}><FI n={receipt ? "chevD" : "chevR"} s={12} />Receipt · {run.receipt.length} values read</button>
            {receipt && <ScrollArea className="rc-sa" viewportClassName="rc" maskVariant="card">{run.receipt.map((r) => <div key={r.name}><span className="mono">{r.name}</span><b className="mono">{fsFmt(r.value)}</b><i className={"src " + SRC_TONE[r.source]}>{r.source}</i></div>)}</ScrollArea>}
          </>
        ) : run.ok ? (
          <div className="res-err"><b>Not a charge</b><p>The formula returned {fsFmt(run.value)}. It needs to return a number.</p></div>
        ) : (
          <div className="res-err"><FI n="xc" s={15} /><div><b>{an.issues.some((i) => i.sev === "error") ? "The expression has errors" : "Fails on this load"}</b><p className="mono">{run.error}</p></div></div>
        )}
      </div>

      <div className="pv-h"><b>{showAll ? "All variables" : "Values this formula reads"}</b><span className="sp"></span>
        {Object.keys(overrides).length > 0 && <button className="tb tb-link tb-s-xs" onClick={() => setOverrides({})}>Reset {Object.keys(overrides).length}</button>}
      </div>
      {shown.length === 0 && <p className="pv-none">Nothing yet. Values appear here as the expression names them.</p>}
      <div className="pv-vals">
        {shown.map((n) => {
          const doc = docOf(n), s = sourceOf(n);
          return (
            <div key={n} className={"pv-v" + (n in overrides ? " ov" : "")}>
              <span className="pv-n mono" title={doc.description}>{n}</span>
              <ValueIn name={n} type={doc.type} value={values[n]} onChange={(v) => setOverrides({ ...overrides, [n]: v })} />
              <i className={"src " + SRC_TONE[s]}>{s}</i>
            </div>
          );
        })}
      </div>
      <button className="pv-all" onClick={() => setShowAll(!showAll)}>{showAll ? "Show only what the formula reads" : `Show all ${FS_VARS.length} shipment variables`}</button>
    </div>
  );
}

function ScenarioWhy({ r, d, saved, onAccept, onAsk, onClose }) {
  const dg = rM(() => fsDiagnose(d, saved, r), [d, saved, r]);
  const [ans, setAns] = rS(null);
  rE(() => { let live = true; setAns(null); novaWhy(dg, r, d).then((a) => live && setAns(a)); return () => { live = false; }; }, [r.actual, r.expected, d.expression]);
  const fmt = (row, v) => (v == null ? "—" : row.kind === "input" ? fsFmt(v) : fsMoney(v));
  const rows = [...dg.rows.filter((x) => x.kind === "input"), ...dg.rows.filter((x) => x.kind === "term")].slice(0, 5);
  const fixPrompt = `Scenario "${r.name}" expects ${fsMoney(r.expected)} but the formula now bills ${fsMoney(r.actual)}. ${ans ? ans.why : ""} Fix the formula so this scenario passes again without breaking the others.`;
  const delta = r.actual != null ? r.actual - r.expected : null;
  return (
    <>
      <div className="sw-k"><span className={"dm" + (ans ? "" : " spin")}></span>{ans ? <span className={"sw-v " + ans.verdict}>{ans.verdict === "intended" ? "Looks intended" : ans.verdict === "bug" ? "Looks like a regression" : "Needs a look"}</span> : <span>Why it broke</span>}<span className="sp"></span>{delta != null && <span className="sw-d mono">{delta >= 0 ? "+" : "−"}{fsMoney(Math.abs(delta))}</span>}</div>
      {ans ? <p className="sw-p">{ans.why.split(/(`[^`]+`)/).map((s, i) => s.startsWith("`") && s.endsWith("`") && s.length > 1 ? <code key={i}>{s.slice(1, -1)}</code> : s)}</p> : <p className="sw-p xp-shim">Comparing {dg.label} with the editor…</p>}
      {rows.length > 0 && <div className="sw-tr">{rows.slice(0, 3).map((x) => <div key={x.kind + x.name} className="sw-r"><code>{x.name}</code><span className="mono">{fmt(x, x.before)}</span><FI n="arrowR" s={10} /><span className="mono now">{fmt(x, x.after)}</span></div>)}</div>}
      <div className="sw-a">
        {(!ans || ans.verdict !== "intended") && <button className="tb tb-ghost tb-s-xs nv-btn" onClick={() => onAsk(fixPrompt)} disabled={!ans}><span className="dm"></span>Fix with Nova</button>}
        {r.actual !== null && <button className={"tb tb-s-xs " + (ans && ans.verdict === "intended" ? "tb-default" : "tb-quiet")} onClick={() => onAccept(r.id, r.actual)}>Expect {fsMoney(r.actual)}</button>}
        {ans && ans.verdict === "intended" && <button className="tb tb-quiet tb-s-xs" onClick={() => onAsk(fixPrompt)}>Fix instead</button>}
        <span className="sp"></span>
        <em className="sw-by" title={dg.recordedOn ? `Recorded by ${dg.recordedOn.by}` : ""}>vs {dg.label}</em>
      </div>
    </>
  );
}

function ScenariosPane({ d, saved, onAsk, results, onAdd, onRemove, onAccept, onSuggest, suggestion, canSave, addedNames, overrides, run }) {
  const [why, setWhy] = rS(null);
  const [adding, setAdding] = rS(false);
  const [name, setName] = rS("");
  const passed = results.filter((r) => r.pass).length;
  const cur = run && run.ok && run.policy ? run.policy.amount : null;
  return (
    <div className="pane">
      <div className="sc-sum">
        <div className={"sc-big " + (results.length === 0 ? "" : passed === results.length ? "ok" : "bad")}>
          <b className="mono">{passed}<em>/{results.length}</em></b><span>{results.length === 0 ? "No scenarios yet" : passed === results.length ? "Every scenario passes" : `${results.length - passed} failing · approval is blocked`}</span>
        </div>
        <span className="sp"></span>
        <button className="tb tb-ghost tb-s-xs nv-btn" onClick={onSuggest} disabled={!d.expression.trim() || suggestion === "busy"}><span className="dm"></span>{suggestion === "busy" ? "Thinking…" : "Suggest edge cases"}</button>
        <button className="tb tb-outline tb-s-xs" onClick={() => setAdding(true)} disabled={cur === null}><FI n="plus" s={12} />Add</button>
      </div>
      {!canSave && <p className="sc-note"><FI n="info" s={12} />Scenarios you add here are saved with the template when you create it.</p>}
      {adding && (
        <div className="sc-add">
          <input className="ui-field ti" autoFocus value={name} placeholder="Scenario name, e.g. Short haul hits the minimum" onChange={(e) => setName(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter" && name.trim()) { onAdd([{ name: name.trim(), inputs: { ...overrides }, expected: cur }]); setName(""); setAdding(false); } }} />
          <div className="sc-add-m"><span>Uses the preview's values{Object.keys(overrides).length ? ` (${Object.keys(overrides).length} changed)` : ""} and expects</span><b className="mono">{fsMoney(cur)}</b></div>
          <div className="sc-add-a"><button className="tb tb-quiet tb-s-xs" onClick={() => setAdding(false)}>Cancel</button><button className="tb tb-default tb-s-xs" disabled={!name.trim()} onClick={() => { onAdd([{ name: name.trim(), inputs: { ...overrides }, expected: cur }]); setName(""); setAdding(false); }}>Save scenario</button></div>
        </div>
      )}
      {suggestion && suggestion !== "busy" && <ScenarioPicks list={suggestion} draft={d} onAdd={onAdd} added={addedNames} canSave={canSave} />}
      <div className="sc-list">
        {results.map((r) => (
          <div key={r.id} className={"sc-r" + (r.pass ? " pass" : " fail") + (r.fresh ? " fresh" : "")}>
            <i className="sc-i"><FI n={r.pass ? "check" : "x"} s={11} w={2.4} /></i>
            <div className="sc-t">
              <b>{r.name}</b>
              <em className="mono">{Object.keys(r.inputs).length ? Object.entries(r.inputs).map(([k, v]) => `${k} ${fsFmt(v)}`).join(" · ") : "Sample load"}</em>
              {!r.pass && r.error && <span className="sc-fail">{r.error}</span>}
            </div>
            <div className="sc-amt">{r.pass ? <b className="mono">{fsMoney(r.actual)}</b> : <><b className="mono now">{r.actual === null ? "—" : fsMoney(r.actual)}</b><s className="mono">{fsMoney(r.expected)}</s></>}
              <span className="sc-act">{!r.pass && <Popover open={why === r.id} onOpenChange={(o) => setWhy(o ? r.id : null)} align="end" className="sw" style={{ width: 340 }}
                trigger={(p) => <button className={"sc-why" + (why === r.id ? " on" : "")} {...p}><span className="dm"></span>Why?</button>}>
                <ScenarioWhy r={r} d={d} saved={saved} onAccept={(id, a) => { onAccept(id, a); setWhy(null); }} onAsk={(t) => { setWhy(null); onAsk(t); }} onClose={() => setWhy(null)} />
              </Popover>}<button className="ib sm" aria-label="Delete scenario" onClick={() => onRemove(r.id)}><FI n="trash" s={12} /></button></span>
            </div>
          </div>
        ))}
      </div>
      {results.length > 0 && <p className="sc-foot">Scenarios re-run against the editor on every change, saved or not. A failing scenario blocks approval until it passes or is deleted.</p>}
    </div>
  );
}

function ReferencePane({ ctx, onInsert }) {
  const [q, setQ] = rS("");
  const [tab, setTab] = rS("vars");
  const ql = q.trim().toLowerCase();
  const vars = rM(() => [...ctx.vars.values()].filter((v) => !ql || v.name.toLowerCase().includes(ql) || (v.description || "").toLowerCase().includes(ql)), [ctx, ql]);
  const fns = FS_FUNCS.filter((f) => !ql || f.name.toLowerCase().includes(ql) || f.description.toLowerCase().includes(ql));
  const groups = tab === "vars"
    ? Object.keys(FS_CATS).map((c) => [FS_CATS[c], vars.filter((v) => (v.category || "shipment") === c)]).filter((g) => g[1].length)
    : [...new Set(FS_FUNCS.map((f) => f.category))].map((c) => [c, fns.filter((f) => f.category === c)]).filter((g) => g[1].length);
  return (
    <div className="pane ref">
      <div className="ref-top">
        <label className="ref-s"><FI n="search" s={13} /><input id="ref-search" value={q} placeholder="Search variables and functions" onChange={(e) => setQ(e.target.value)} /><span className="kbd">/</span></label>
        <Seg value={tab} onChange={setTab} options={[{ value: "vars", label: `Variables ${vars.length}` }, { value: "fns", label: `Functions ${fns.length}` }]} />
      </div>
      {groups.map(([g, items]) => (
        <div key={g} className="ref-g">
          <div className="ref-gh">{g}</div>
          {items.map((it) => (
            <button key={it.name} className="ref-r" onClick={() => tab === "vars" ? onInsert(it.name) : onInsert(it.name + "()", it.name.length + 1)}>
              <span className="ref-l"><b className="mono">{tab === "vars" ? it.name : it.signature}</b>{tab === "vars" && <em>{it.type}{it.nullable ? " · can be empty" : ""}</em>}</span>
              <span className="ref-d">{it.description}</span>
              {tab === "fns" && it.example && <code className="ref-ex mono">{it.example}</code>}
              <FI n="plus" s={12} />
            </button>
          ))}
        </div>
      ))}
      {groups.length === 0 && <p className="pv-none">Nothing matches “{q}”.</p>}
    </div>
  );
}

Object.assign(window, { PreviewPane, ScenariosPane, ReferencePane });
