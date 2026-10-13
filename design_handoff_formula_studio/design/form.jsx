const { useState: fS, useRef: fR, useEffect: fE, useMemo: fM } = React;

function SecHead({ id, icon, title, sub, children }) {
  return (
    <div className="sh" id={"sec-" + id}>
      <div className="sh-t"><h2><FI n={icon} s={15} />{title}</h2>{sub && <p>{sub}</p>}</div>
      {children && <div className="sh-a">{children}</div>}
    </div>
  );
}
function Seg({ value, options, onChange, size }) {
  return <div className={"seg" + (size ? " " + size : "")}>{options.map((o) => <button key={o.value} type="button" className={value === o.value ? "on" : ""} onClick={() => onChange(o.value)}><span className="seg-l">{o.label}</span></button>)}</div>;
}
function Sw({ on, onChange, sm, label }) {
  return <button type="button" role="switch" aria-checked={on} aria-label={label} className={"tsw" + (on ? " on" : "") + (sm ? " sm" : "")} onClick={() => onChange(!on)}><i></i></button>;
}
function AutoText({ value, onChange, placeholder, className, minRows = 1 }) {
  const r = fR();
  fE(() => { if (r.current) { r.current.style.height = "auto"; r.current.style.height = r.current.scrollHeight + "px"; } }, [value]);
  return <textarea ref={r} rows={minRows} className={className} value={value} placeholder={placeholder} onChange={(e) => onChange(e.target.value)}></textarea>;
}
function NumIn({ value, onChange, placeholder, prefix, step = "any", className = "" }) {
  return (
    <div className="ti-wrap">
      {prefix && <span className="ti-l">{prefix}</span>}
      <input className={"ui-field ti mono " + (prefix ? "pl " : "") + className} inputMode="decimal" value={value ?? ""} placeholder={placeholder}
        onChange={(e) => { const v = e.target.value.replace(/[^\d.\-]/g, ""); onChange(v === "" ? null : v.endsWith(".") || v === "-" ? v : Number(v)); }} />
    </div>
  );
}

function DetailsHero({ d, set, issues }) {
  return (
    <section className="hero" id="sec-details">
      <input className={"hero-name" + (issues.name ? " bad" : "")} value={d.name} placeholder="Untitled rating method" onChange={(e) => set({ name: e.target.value })} aria-label="Template name" />
      <AutoText className={"hero-desc" + (issues.description ? " bad" : "")} value={d.description} onChange={(v) => set({ description: v })} placeholder="When this method applies and how it prices. Reviewers and rate agreement owners read this." />
      <div className="hero-meta">
        <span className="hm-l">Charges</span>
        <Seg value={d.type} onChange={(v) => set({ type: v })} options={[{ value: "FreightCharge", label: "Freight" }, { value: "AccessorialCharge", label: "Accessorial" }]} />
        <span className="hm-l">Schema</span>
        <span className="bdg"><FI n="truck" s={11} />Shipment</span>
      </div>
    </section>
  );
}

function StartCard({ onDescribe, onStandard, busy }) {
  const [q, setQ] = fS("");
  const [tab, setTab] = fS("std");
  return (
    <div className="start">
      <div className={"start-nv" + (busy ? " busy" : "")}>
        <NovaMark s={16} busy={busy} />
        <input value={q} placeholder="Describe the method: “per mile, 18% fuel, $250 minimum, $50 a stop after two”" onChange={(e) => setQ(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter" && q.trim()) { onDescribe(q.trim()); setQ(""); } }} />
        <button className="tb tb-default tb-s-sm" disabled={!q.trim() || busy} onClick={() => { onDescribe(q.trim()); setQ(""); }}>Draft it</button>
      </div>
      <div className="start-or"><span>or start from</span><Seg size="sm" value={tab} onChange={setTab} options={[{ value: "std", label: "Standard library" }, { value: "org", label: "Your templates" }]} /></div>
      <div className="start-g">
        {(tab === "std" ? FS_STANDARDS : FS_ORG_TEMPLATES).map((s) => (
          <button key={s.name} className="start-o" onClick={() => onStandard(s, tab)}>
            <b>{s.name}</b>
            <code className="mono">{s.expression}</code>
            <em>{tab === "std" ? (s.type === "AccessorialCharge" ? "Accessorial" : "Freight") + (s.variables.length ? ` · ${s.variables.length} input${s.variables.length > 1 ? "s" : ""}` : "") : s.meta}</em>
          </button>
        ))}
      </div>
    </div>
  );
}

function Problems({ an, run, onAddInput, onFix, onNovaFix }) {
  const runtime = !an.issues.some((i) => i.sev === "error") && run && !run.ok && !run.empty && run.error;
  if (!an.issues.length && !runtime) return null;
  return (
    <div className="probs">
      {an.issues.map((i, k) => (
        <div key={k} className={"prob " + i.sev}>
          <FI n={i.sev === "error" ? "xc" : "alert"} s={13} />
          <span>{i.msg}</span>
          {i.kind === "var" && <button className="tb tb-outline tb-s-xs" onClick={() => onAddInput(i.name)}><FI n="plus" s={11} />Add <b className="mono">{i.name}</b> as input</button>}
          {i.kind === "nullable" && <button className="tb tb-outline tb-s-xs mono" onClick={() => onFix(i)}>Use {i.fix}</button>}
          {i.sev === "error" && i.kind !== "var" && <button className="tb tb-ghost tb-s-xs nv-btn" onClick={() => onNovaFix(i.msg)}><span className="dm"></span>Fix with Nova</button>}
        </div>
      ))}
      {runtime && <div className="prob error"><FI n="xc" s={13} /><span>On the preview load: {run.error}</span><button className="tb tb-ghost tb-s-xs nv-btn" onClick={() => onNovaFix(run.error)}><span className="dm"></span>Fix with Nova</button></div>}
    </div>
  );
}

function ExplainPanel({ items, busy, onHover, onClose, onAsk, run }) {
  const pr = run && run.ok && run.policy;
  const lead = pr ? `On the preview load the formula comes to ${fsMoney(pr.raw)}${pr.clamp ? `, under the ${pr.clamp.bound}, so it bills ${fsMoney(pr.amount)}` : ` and bills ${fsMoney(pr.amount)}`}. Here's how each line gets there.` : "Here's what each line of the formula does.";
  const steps = items;
  return (
    <div className="xp">
      <div className="xp-b">
        <div className="xp-k"><NovaMark s={13} busy={busy} /><span>What this charges</span><span className="sp"></span>
          {!busy && <button className="xp-wh" onClick={onAsk}>Ask a follow-up<FI n="arrowR" s={11} /></button>}
          <button className="ib sm" onClick={onClose} aria-label="Close explanation"><FI n="x" s={13} /></button></div>
        {busy ? <p className="xp-shim">Reading the formula and its inputs…</p>
          : <p>{lead}</p>}
      </div>
      {!busy && steps.length > 0 && (
        <div className="xp-why">
          {steps.map((it, i) => (
            <div key={i} className="xp-r" onMouseEnter={() => onHover({ s: it.s, e: it.e })} onMouseLeave={() => onHover(null)}>
              <span className={"xp-n mono" + (i === steps.length - 1 && !it.code.startsWith("let") ? " fin" : "")}>{i === steps.length - 1 && !it.code.startsWith("let") ? "=" : i + 1}</span>
              <div><code className="mono">{it.code.length > 70 ? it.code.slice(0, 68) + "…" : it.code}</code><span>{it.says}</span></div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ExpressionSection(p) {
  const { d, set, an, ctx, edRef, focus, setFocus, cursor, setCursor, run, mode, explain, onExplain, onCloseExplain, onAsk, onDescribe, onStandard, novaBusy, onAddInput, onFix, materialNote } = p;
  const errs = an.issues.filter((i) => i.sev === "error").length, warns = an.issues.filter((i) => i.sev === "warn").length;
  const empty = !d.expression.trim();
  return (
    <section className="sec">
      <SecHead id="expression" icon="code" title="Expression" sub="The formula that turns shipment facts into the charge. Charge policy applies after it.">
        <button className="tb tb-outline tb-s-xs" disabled={empty} onClick={onExplain}><FI n="book" s={12} />Explain</button>
        <button className="tb tb-outline tb-s-xs nv-btn" onClick={() => onAsk()}><span className="dm"></span>Ask Nova</button>
      </SecHead>
      {materialNote}
      {mode === "create" && empty && <StartCard onDescribe={onDescribe} onStandard={onStandard} busy={novaBusy} />}
      <div className={"ed-card" + (errs ? " has-err" : "")}>
        <ExprEditor ref={edRef} value={d.expression} onChange={(v) => set({ expression: v })} ctx={ctx} an={an} focus={focus} onCursor={setCursor} placeholder="e.g. baseRate * totalDistance" minLines={mode === "create" && empty ? 3 : 5} />
        <div className="ed-bar">
          {empty ? <span className="eb-st"><i className="dot"></i>Empty</span> : errs ? <span className="eb-st bad"><i className="dot"></i>{errs} error{errs > 1 ? "s" : ""}</span> : run && !run.ok ? <span className="eb-st bad"><i className="dot"></i>Fails on the preview load</span> : <span className="eb-st ok"><i className="dot"></i>Valid{warns ? ` · ${warns} warning${warns > 1 ? "s" : ""}` : ""}</span>}
          <span className="eb-m">{an.used.vars.size} schema · {an.used.customs.size} inputs · {an.used.funcs.size} functions</span>
          <span className="sp"></span>
          <span className="eb-m"><span className="kbd">Ctrl</span><span className="kbd">Space</span> suggest</span>
          <span className="eb-m mono">Ln {cursor.ln}, Col {cursor.col}</span>
        </div>
      </div>
      <Problems an={an} run={run} onAddInput={onAddInput} onFix={onFix} onNovaFix={(msg) => onAsk(`Fix this: ${msg}`)} />
      {explain && <ExplainPanel items={explain.items} busy={explain.busy} onHover={setFocus} onClose={onCloseExplain} onAsk={() => onAsk("Explain this formula, term by term")} run={run} />}
    </section>
  );
}

function InputsSection({ d, set, used, onDescribe, describing }) {
  const upd = (id, patch) => set({ variables: d.variables.map((v) => (v.id === id ? { ...v, ...patch } : v)) });
  const add = () => set({ variables: [...d.variables, { id: fsId("v"), name: "", type: "Number", defaultValue: null, required: false, description: "" }] });
  const missing = d.variables.filter((v) => v.name && !v.description).length;
  return (
    <section className="sec">
      <SecHead id="inputs" icon="input" title="Inputs" sub="Values this template adds to the schema. Lanes and rate agreements can override each default.">
        {missing > 0 && <button className="tb tb-ghost tb-s-xs nv-btn" disabled={describing} onClick={onDescribe}><span className="dm"></span>{describing ? "Writing…" : `Describe ${missing}`}</button>}
        <button className="tb tb-outline tb-s-xs" onClick={add}><FI n="plus" s={12} />Add input</button>
      </SecHead>
      {d.variables.length === 0 ? (
        <div className="empty-row"><span>No inputs. Shipment variables cover most methods; add one for a fee or percentage lanes should set.</span><button className="tb tb-ghost tb-s-xs" onClick={add}><FI n="plus" s={12} />Add input</button></div>
      ) : (
        <div className="card vtab">
          <div className="vt-h"><span>Name</span><span>Type</span><span>Default</span><span>Required</span><span></span></div>
          {d.variables.map((v) => {
            const unused = v.name && !used.has(v.name);
            const dup = v.name && (d.variables.filter((x) => x.name === v.name).length > 1 || FS_VARS.some((x) => x.name === v.name));
            return (
              <div key={v.id} className={"vt-r" + (v.fresh ? " fresh" : "")}>
                <div className="vt-n">
                  <input className={"ui-field ti mono" + (dup ? " bad" : "")} value={v.name} placeholder="fuelSurchargePercent" onChange={(e) => upd(v.id, { name: e.target.value.replace(/[^\w]/g, "") })} />
                  {unused && <span className="tag">unused</span>}
                  {dup && <span className="tag bad">name taken</span>}
                </div>
                <select className="ui-field ti" value={v.type} onChange={(e) => upd(v.id, { type: e.target.value, defaultValue: e.target.value === "Boolean" ? false : e.target.value === "String" ? "" : null })}>
                  <option>Number</option><option>String</option><option>Boolean</option>
                </select>
                {v.type === "Boolean" ? <div className="vt-b"><Sw on={!!v.defaultValue} onChange={(x) => upd(v.id, { defaultValue: x })} label="Default" /><span>{v.defaultValue ? "true" : "false"}</span></div>
                  : v.type === "String" ? <input className="ui-field ti mono" value={v.defaultValue ?? ""} onChange={(e) => upd(v.id, { defaultValue: e.target.value })} />
                  : <NumIn value={v.defaultValue} onChange={(x) => upd(v.id, { defaultValue: x })} placeholder="—" />}
                <div className="vt-b"><Sw on={v.required} onChange={(x) => upd(v.id, { required: x })} sm label="Required" /></div>
                <button className="ib sm" aria-label="Remove input" onClick={() => set({ variables: d.variables.filter((x) => x.id !== v.id) })}><FI n="trash" s={13} /></button>
                <input className={"vt-d" + (v.described ? " nv-fill" : "")} value={v.description} placeholder="Description lanes see when they override it" onChange={(e) => upd(v.id, { description: e.target.value, described: false })} />
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}

function PolicySection({ d, set, run }) {
  const pol = d.policy, setP = (patch) => set({ policy: { ...pol, ...patch } });
  const bad = pol.min != null && pol.max != null && Number(pol.min) > Number(pol.max);
  const pr = run && run.ok && run.policy;
  return (
    <section className="sec">
      <SecHead id="policy" icon="shield" title="Charge policy" sub="Clamp the formula's result to a range, then round it to what gets billed. Preview, scenarios and production all apply it the same way." />
      <div className="card pol">
        <div className="pol-g">
          <label className="fw"><span className="f-label">Minimum charge</span><NumIn prefix="$" value={pol.min} onChange={(v) => setP({ min: v })} placeholder="No minimum" /></label>
          <label className="fw"><span className="f-label">Maximum charge</span><NumIn prefix="$" value={pol.max} onChange={(v) => setP({ max: v })} placeholder="No maximum" className={bad ? "bad" : ""} />{bad && <span className="f-err">Lower than the minimum</span>}</label>
          <div className="fw"><span className="f-label">Rounding</span><Seg value={pol.mode} onChange={(v) => setP({ mode: v })} options={FS_ROUNDING} /></div>
          <div className="fw"><span className="f-label">Decimal places</span>
            <div className="step"><button className="ib sm" disabled={pol.mode === "None" || pol.precision <= 0} onClick={() => setP({ precision: pol.precision - 1 })}><FI n="minus" s={12} /></button><b className="mono">{pol.mode === "None" ? "—" : pol.precision}</b><button className="ib sm" disabled={pol.mode === "None" || pol.precision >= 6} onClick={() => setP({ precision: pol.precision + 1 })}><FI n="plus" s={12} /></button></div>
          </div>
        </div>
        {pr && (
          <div className="pipe">
            <span className="pipe-l">On the preview load</span>
            <span className="pipe-s"><em>Formula</em><b className="mono">{fsMoney(pr.raw, 4)}</b></span><FI n="arrowR" s={12} />
            <span className={"pipe-s" + (pr.clamp ? " hit" : "")}><em>{pr.clamp ? `Raised to ${pr.clamp.bound}` : "Within range"}</em><b className="mono">{fsMoney(pr.clamped, pr.clamp ? 2 : 4)}</b></span><FI n="arrowR" s={12} />
            <span className="pipe-s bill"><em>Billed</em><b className="mono">{fsMoney(pr.amount, pol.mode === "None" ? 4 : Math.max(pol.precision, 0))}</b></span>
          </div>
        )}
      </div>
    </section>
  );
}

function BreakdownSection({ d, set, ctx, run }) {
  const upd = (id, patch) => set({ breakdown: d.breakdown.map((b) => (b.id === id ? { ...b, ...patch } : b)) });
  const add = () => set({ breakdown: [...d.breakdown, { id: fsId("b"), name: "", label: "", expression: "" }] });
  const lines = run && run.lines ? Object.fromEntries(run.lines.map((l) => [l.id, l])) : {};
  const rec = run && run.reconcile;
  return (
    <section className="sec">
      <SecHead id="breakdown" icon="layers" title="Breakdown" sub="Itemize the total into named lines for invoices and disputes. Lines should add up to the formula's result.">
        <button className="tb tb-outline tb-s-xs" disabled={d.breakdown.length >= 20} onClick={add}><FI n="plus" s={12} />Add line</button>
      </SecHead>
      {d.breakdown.length === 0 ? (
        <div className="empty-row"><span>No lines. The invoice shows one amount for this charge.</span><button className="tb tb-ghost tb-s-xs" onClick={add}><FI n="plus" s={12} />Add line</button></div>
      ) : (
        <div className="card bd">
          {d.breakdown.map((b) => {
            const an = fsAnalyze(b.expression, ctx), l = lines[b.id];
            return (
              <div key={b.id} className="bd-r">
                <input className="ui-field ti" value={b.label} placeholder="Fuel surcharge" onChange={(e) => upd(b.id, { label: e.target.value, name: b.nameTouched ? b.name : e.target.value.replace(/[^\w ]/g, "").replace(/ (\w)/g, (_, c) => c.toUpperCase()).replace(/^./, (c) => c.toLowerCase()) })} aria-label="Line label" />
                <ExprEditor mini value={b.expression} onChange={(v) => upd(b.id, { expression: v })} ctx={ctx} an={an} placeholder="baseRate * totalDistance" label="Line expression" />
                <span className={"bd-amt mono" + (l && !l.ok ? " bad" : "")} title={l && l.error ? l.error : ""}>{l ? (l.ok ? fsMoney(l.amount) : "Fails") : "—"}</span>
                <button className="ib sm" aria-label="Remove line" onClick={() => set({ breakdown: d.breakdown.filter((x) => x.id !== b.id) })}><FI n="trash" s={13} /></button>
                <input className="bd-key mono" value={b.name} onChange={(e) => upd(b.id, { name: e.target.value.replace(/[^\w]/g, ""), nameTouched: true })} aria-label="Line key" placeholder="key" />
              </div>
            );
          })}
          {rec && (
            <div className={"bd-f" + (rec.balanced && !rec.clampMismatch ? " ok" : " warn")}>
              <FI n={rec.balanced && !rec.clampMismatch ? "okc" : "alert"} s={13} />
              <span>{rec.clampMismatch ? `Lines add up to the formula's ${fsMoney(rec.sum)}, but the ${run.policy.clamp.bound} bills ${fsMoney(run.policy.amount)}. An invoice built from these lines won't match the charge.` : rec.balanced ? "Lines explain the total" : rec.failed ? `${rec.failed} line${rec.failed > 1 ? "s" : ""} fail on the preview load` : rec.residual > 0 ? `${fsMoney(rec.residual)} of the total isn't on any line` : `Lines exceed the total by ${fsMoney(-rec.residual)}`}</span>
              <b className="mono">{fsMoney(rec.sum)}</b>
            </div>
          )}
        </div>
      )}
    </section>
  );
}

Object.assign(window, { SecHead, Seg, Sw, NumIn, AutoText, DetailsHero, ExpressionSection, InputsSection, PolicySection, BreakdownSection });
