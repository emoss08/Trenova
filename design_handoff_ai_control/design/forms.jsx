// Shared edit-panel kit. One shell (EditSheet) + one draft hook (useDraft) + field primitives.
// Every editor (agent, provider, tool rule, memory…) composes these; none owns its own chrome.
const eq = (a, b) => JSON.stringify(a) === JSON.stringify(b);
function useDraft(init) {
  const [base, setBase] = React.useState(init);
  const [v, setV] = React.useState(init);
  const set = (k, x) => setV(o => ({ ...o, [k]: typeof x === "function" ? x(o[k]) : x }));
  const changed = Object.keys(v).filter(k => !eq(v[k], base[k]));
  return { v, base, set, changed, dirty: changed.length > 0, reset: () => setV(base), commit: () => setBase(v), patch: p => setV(o => ({ ...o, ...p })) };
}
function fmtVal(x, f) {
  if (f) return f(x);
  if (typeof x === "boolean") return x ? "On" : "Off";
  if (Array.isArray(x)) return x.length + " selected";
  if (x == null || x === "") return "Empty";
  const s = String(x); return s.length > 42 ? s.slice(0, 40) + "…" : s;
}
function arrDiff(a, b) { const add = b.filter(x => !a.includes(x)), rm = a.filter(x => !b.includes(x)); return { add, rm }; }
function ChangeRow({ k, d, field, onUndo }) {
  const f = field || {}, a = d.base[k], b = d.v[k];
  let body;
  if (f.diff) body = f.diff(a, b);
  else if (Array.isArray(a) && Array.isArray(b) && a.every(x => typeof x !== "object")) { const { add, rm } = arrDiff(a, b); body = <span className="cr-v">{add.map(x => <em key={"+" + x} className="ad">+ {f.item ? f.item(x) : x}</em>)}{rm.map(x => <em key={"-" + x} className="rm">− {f.item ? f.item(x) : x}</em>)}</span>; }
  else body = <span className="cr-v"><s>{fmtVal(a, f.fmt)}</s><Ic n="arrowR" s={11}></Ic><b>{fmtVal(b, f.fmt)}</b></span>;
  return <div className="cr"><span className="cr-l">{f.l || k}</span>{body}<button className="ib xs" title="Undo this change" onClick={onUndo}><Ic n="undo" s={11}></Ic></button></div>;
}
// Save/close behaviour shared by every editor surface (side sheet or full builder).
function useEditFlow({ d, onSave, onClose, create, invalid }) {
  const [confirm, setConfirm] = React.useState(false);
  const [review, setReview] = React.useState(false);
  const [saving, setSaving] = React.useState(false);
  const [saved, setSaved] = React.useState(false);
  const canSave = (d.dirty || create) && !invalid && !saving;
  const tryClose = () => (d.dirty ? setConfirm(true) : onClose());
  const save = () => { if (!canSave) return; setSaving(true); setReview(false); setTimeout(() => { const keep = onSave(d.v); d.commit(); setSaving(false); setSaved(true); setTimeout(() => setSaved(false), 1800); if (create && keep !== false) onClose(); }, 650); };
  React.useEffect(() => {
    const k = e => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "s") { e.preventDefault(); save(); }
      else if (e.key === "Escape" && !e.defaultPrevented && !document.querySelector(".mdx")) { e.preventDefault(); confirm ? setConfirm(false) : review ? setReview(false) : tryClose(); }
    };
    window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k);
  });
  return { confirm, setConfirm, review, setReview, saving, saved, canSave, tryClose, save, onClose };
}
function SaveBar({ f, d, fields = {}, create, saveLabel = "Save changes", invalid, footL }) {
  const n = d.changed.length;
  return <>
    {f.review && d.dirty && <div className="es-rv"><div className="es-rvh"><b>Review {n} change{n > 1 ? "s" : ""}</b><span>Undo any one before saving</span></div>{d.changed.map(k => <ChangeRow key={k} k={k} d={d} field={fields[k]} onUndo={() => d.set(k, d.base[k])}></ChangeRow>)}</div>}
    {f.confirm ? <><Ic n="warn" s={14}></Ic><span className="es-cft">Discard {n} unsaved change{n > 1 ? "s" : ""}?</span><span className="sp"></span><button className="btn sm" autoFocus onClick={() => f.setConfirm(false)}>Keep editing</button><button className="btn sm dng" onClick={f.onClose}>Discard and close</button></>
      : <>
        {d.dirty ? <button className={"es-chg" + (f.review ? " on" : "")} onClick={() => f.setReview(r => !r)}><i></i>{n} unsaved change{n > 1 ? "s" : ""}<Ic n="chevD" s={12}></Ic></button>
          : f.saved ? <span className="es-ok"><Ic n="check" s={12} w={2.4}></Ic>Saved</span>
          : <span className="es-idle">{create ? "Nothing is created until you save" : "No changes"}</span>}
        {invalid && <span className="es-inv"><Ic n="alert" s={12}></Ic>{invalid}</span>}
        {footL}
        <span className="sp"></span>
        {d.dirty && !create && <button className="btn sm" onClick={d.reset}>Discard</button>}
        <button className="btn ink" disabled={!f.canSave} onClick={f.save}>{f.saving ? <><i className="spn"></i>Saving</> : saveLabel}<span className="kbd">{MOD} S</span></button>
      </>}
  </>;
}
function EditSheet({ icon, title, sub, headR, sections, d, fields = {}, onSave, onClose, create, saveLabel, aside, banner, footL, invalid, width, conflict }) {
  const [act, setAct] = React.useState(sections[0].id);
  const f = useEditFlow({ d, onSave, onClose, create, invalid });
  const body = React.useRef();
  const onScroll = () => { const b = body.current; if (!b) return; let cur = sections[0].id; for (const s of sections) { const el = b.querySelector("#es-" + s.id); if (el && el.offsetTop - b.scrollTop < 90) cur = s.id; } if (b.scrollTop + b.clientHeight >= b.scrollHeight - 4) cur = sections[sections.length - 1].id; setAct(cur); };
  const jump = id => { const b = body.current, el = b && b.querySelector("#es-" + id); if (el) b.scrollTo({ top: el.offsetTop - 8, behavior: "smooth" }); setAct(id); };
  return ReactDOM.createPortal(
    <div className="shx" onMouseDown={e => e.target === e.currentTarget && f.tryClose()}>
      <aside className={"sheet es es2" + (aside ? " has-aside" : "") + (width ? " w-" + width : "")} role="dialog" aria-modal="true" aria-label={title}>
        <header className="sh-h">{icon}<div className="sh-t"><b>{title}</b>{sub && <span>{sub}</span>}</div>{headR}<button className="ib" title="Close (Esc)" onClick={f.tryClose}><Ic n="x" s={14}></Ic></button></header>
        {conflict}
        {banner}
        <div className="es-m">
          <div className="es-b" ref={body} onScroll={onScroll}>
            {sections.map(s => <section key={s.id} id={"es-" + s.id} className={"es-s" + (s.keys && s.keys.some(k => d.changed.includes(k)) ? " dirty" : "")}><header className="es-sh"><h3>{s.l}{s.warn && <i className="es-wn" title={s.warn}></i>}</h3>{s.r}</header>{s.note && <p className="es-note">{s.note}</p>}{s.body}</section>)}
          </div>
          {aside && <div className="es-aside">{aside}</div>}
        </div>
        <footer className={"es-f" + (f.confirm ? " cf" : "") + (d.dirty || f.confirm || f.saved || create || footL ? " in" : "")}><SaveBar f={f} d={d} fields={fields} create={create} saveLabel={saveLabel} invalid={invalid} footL={footL}></SaveBar></footer>
      </aside>
    </div>, document.querySelector(".tv"));
}
function F({ l, hint, err, aside, children, row, cls = "" }) {
  return <div className={"f" + (row ? " row" : "") + (err ? " err" : "") + " " + cls}><div className="f-l"><label>{l}</label>{aside}</div>{row ? <div className="f-c">{children}</div> : children}{(err || hint) && <p className="f-h">{err || hint}</p>}</div>;
}
function Txt({ v, on, ph, mono, pre, suf, type = "text", auto, w }) {
  return <label className={"inx" + (mono ? " mono" : "")} style={w ? { width: w } : null}>{pre && <span className="inx-a">{pre}</span>}<input type={type} value={v ?? ""} autoFocus={auto} onChange={e => on(type === "number" ? (e.target.value === "" ? "" : +e.target.value) : e.target.value)} placeholder={ph}></input>{suf && <span className="inx-a">{suf}</span>}</label>;
}
function Area({ v, on, ph, rows = 6, max, mono, taRef }) {
  return <div className="ara"><textarea ref={taRef} className={mono ? "mono" : ""} rows={rows} value={v} placeholder={ph} onChange={e => on(e.target.value)}></textarea>{max && <span className={"ara-c mono" + (v.length > max ? " t-d" : "")}>{v.length.toLocaleString()} / {max.toLocaleString()}</span>}</div>;
}
function Sel({ v, on, opts, w }) {
  return <label className="inx sel" style={w ? { width: w } : null}><select value={v} onChange={e => on(e.target.value)}>{opts.map(o => Array.isArray(o) ? <option key={o[0]} value={o[0]}>{o[1]}</option> : <option key={o} value={o}>{o}</option>)}</select><Ic n="chevD" s={12}></Ic></label>;
}
function SwRow({ l, s, v, on, disabled, why }) {
  return <div className={"swr" + (disabled ? " dis" : "")}><span><b>{l}</b>{s && <em>{s}</em>}{disabled && why && <em className="t-w">{why}</em>}</span><Switch on={v} disabled={disabled} label={l} onChange={on}></Switch></div>;
}
function Chips({ v, on, opts, icon }) {
  return <div className="tks">{opts.map(o => { const [k, l, x] = Array.isArray(o) ? o : [o, o]; const sel = v.includes(k); return <button key={k} type="button" className={"tkb" + (sel ? " on" : "")} onClick={() => on(sel ? v.filter(y => y !== k) : [...v, k])}><Ic n={sel ? "check" : "plus"} s={11} w={2.2}></Ic>{l}{x}</button>; })}</div>;
}
function Callout({ tone = "i", ic, children, act }) {
  return <div className={"cal " + tone}><Ic n={ic || (tone === "w" ? "warn" : tone === "d" ? "alert" : tone === "k" ? "check" : "sparkle")} s={13}></Ic><div>{children}</div>{act}</div>;
}
function ConflictBar({ who, when, what, onReview, onOverwrite }) {
  const [open, setOpen] = React.useState(false);
  return <div className="es-cx"><div className="es-cxr"><span className="me sm">{who.split(" ").map(x => x[0]).join("")}</span><span><b>{who}</b> saved changes {when} while you were editing.</span><span className="sp"></span><button className="btn sm" onClick={() => setOpen(o => !o)}>{open ? "Hide" : "See theirs"}</button><button className="btn sm" onClick={onReview}>Load theirs</button><button className="btn sm ink" onClick={onOverwrite}>Keep mine</button></div>{open && <ul className="es-cxl">{what.map(w => <li key={w}>{w}</li>)}</ul>}</div>;
}
function ConfirmDialog({ title, body, confirm = "Confirm", typed, danger, onConfirm, onClose }) {
  const [v, setV] = React.useState("");
  const ok = !typed || v === typed;
  return <Modal onClose={onClose} title={title} sub={body} foot={<><span className="sp"></span><button className="btn sm" onClick={onClose}>Cancel</button><button className={"btn sm " + (danger ? "dng" : "ink")} disabled={!ok} onClick={() => { onConfirm(); onClose(); }}>{confirm}</button></>}>
    {typed && <F l={<>Type <b className="mono">{typed}</b> to confirm</>}><Txt v={v} on={setV} auto mono></Txt></F>}
  </Modal>;
}
Object.assign(window, { useDraft, useEditFlow, SaveBar, ChangeRow, EditSheet, F, Txt, Area, Sel, SwRow, Chips, Callout, ConflictBar, ConfirmDialog, arrDiff, eq });
