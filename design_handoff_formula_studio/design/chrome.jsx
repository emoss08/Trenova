const { useState: cS, useEffect: cE } = React;
const LC = [["Draft", "Draft"], ["InReview", "In review"], ["Active", "Active"]];
const prettyMode = (m) => String(m || "").replace(/([a-z])([A-Z])/g, "$1 $2").toLowerCase();

function Lifecycle({ mode, status, version, material, dirty }) {
  if (mode === "create") return (
    <ol className="lc" aria-label="Lifecycle">
      <li className="lc-s on new"><i></i>New</li><li className="lc-ln"></li>
      {LC.map(([k, l], i) => <React.Fragment key={k}><li className="lc-s"><i></i>{l}</li>{i < 2 && <li className="lc-ln"></li>}</React.Fragment>)}
    </ol>
  );
  if (status === "Inactive") return <ol className="lc"><li className="lc-s on arch"><i></i>Archived · v{version}</li></ol>;
  const cur = LC.findIndex(([k]) => k === status);
  const back = dirty && material && cur > 0;
  const n = FS_USAGE[0].count + FS_USAGE[1].count;
  return (
    <ol className={"lc" + (back ? " back" : "")} aria-label="Lifecycle">
      {LC.map(([k, l], i) => (
        <React.Fragment key={k}>
          <li className={"lc-s " + (i < cur ? "past" : i === cur ? "on t-" + k : "") + (back && i === 0 ? " next" : "")}>
            {i < cur ? <FI n="check" s={11} w={2.4} /> : <i></i>}{l}
            {i === cur && <span className="lc-v mono">v{version}</span>}
            {back && i === 0 && <em>on save</em>}
          </li>
          {i === cur && cur === 2 && (
            <li className="lc-use"><HoverCard align="start" trigger={(p) => <span {...p}>{n} in use</span>}><UsageCard version={version} /></HoverCard></li>
          )}
          {i < 2 && <li className={"lc-ln" + (i < cur ? " past" : "")}></li>}
        </React.Fragment>
      ))}
    </ol>
  );
}

function StudioTop({ d, mode, status, version, dirty, material, actions, onSave }) {
  const name = d.name || (mode === "create" ? "New rating method" : "Untitled");
  return (
    <header className="top2">
      <button className="ib" aria-label="Back to formula templates"><FI n="back" s={16} /></button>
      <div className="t2-id">
        <b className={d.name ? "" : "ph"}>{name}</b>
      </div>
      <span className="t2-div"></span>
      <Lifecycle mode={mode} status={status} version={version} material={material} dirty={dirty} />
      <span className="sp"></span>
      {actions}
      {mode === "edit" && !dirty
        ? <span className="t2-saved"><FI n="check" s={13} w={2.2} />Saved</span>
        : <button className="tb tb-default tb-s-sm t2-save" onClick={() => onSave()}>{mode === "create" ? "Create template" : <><i className="t2-dot"></i>Save changes</>}<span className="kbd inv">⌘S</span></button>}
    </header>
  );
}

function RateTrace({ d, an, run, ship, ov, results, checks, mode, status, mainRef, onJump, onTab }) {
  const [act, setAct] = cS("expression");
  cE(() => {
    const el = mainRef.current; if (!el) return;
    const ids = ["details", "expression", "inputs", "policy", "breakdown"];
    const h = () => { let cur = ids[0]; ids.forEach((id) => { const s = document.getElementById("sec-" + id); if (s && s.offsetTop - 140 <= el.scrollTop) cur = id; }); setAct(cur); };
    h(); el.addEventListener("scroll", h, { passive: true }); return () => el.removeEventListener("scroll", h);
  }, [mainRef]);

  const empty = !d.expression.trim();
  const errs = an.issues.filter((i) => i.sev === "error").length;
  const ok = run.ok && run.policy;
  const reads = ok ? run.receipt : [...an.used.vars, ...an.used.customs].map((name) => ({ name, value: run.values ? run.values[name] : null }));
  const shown = reads.slice(0, 4);
  const p = d.policy;
  const edited = Object.keys(ov).length;
  const srcLabel = ship ? `${ship.id} · ${ship.lane}` : `Sample load · ${FS_SAMPLE.totalDistance} mi · ${FS_SAMPLE.totalStops} stops`;
  const lines = ok ? run.lines.filter((l) => l.ok && l.amount > 0) : [];
  const lsum = lines.reduce((s, l) => s + l.amount, 0);
  const passN = results.filter((r) => r.pass).length;
  const rc = mode === "create" ? checks.filter((c) => c.key !== "scen") : checks;
  const open = rc.filter((c) => c.status !== "pass");
  const fState = empty ? "todo" : errs || !run.ok ? "bad" : an.issues.some((i) => i.kind === "nullable") ? "warn" : "ok";
  const unused = d.variables.filter((v) => v.name && !an.used.customs.has(v.name)).length;
  const goal = mode === "create" ? "Ready to create" : status === "InReview" ? "Ready to approve" : status === "Active" ? "Ready to ship" : "Ready for review";
  const pulse = ok ? run.policy.amount : "x";

  return (
    <ScrollArea as="nav" className="tr" viewportClassName="tr-vp" maskVariant="background" aria-label="Rate trace">
      <div className="tr-h">
        <div className="tr-t"><b>Rate trace</b><span className={"tr-live" + (ok ? "" : " off")}><i></i>{ok ? "live" : empty ? "waiting" : "stopped"}</span></div>
        <button className="tr-src" onClick={() => onTab("preview")} title="Change the load in Preview"><FI n="truck" s={13} /><span>{srcLabel}</span>{edited > 0 && <em>{edited} edited</em>}</button>
      </div>

      <ol className="tr-sp">
        <li className={"tr-n " + (unused ? "warn" : reads.length ? "ok" : "todo") + (act === "inputs" ? " act" : "")}>
          <button className="tr-hd" onClick={() => onJump("inputs")}><i className="tr-m"></i><span>Reads</span><em className="mono">{reads.length ? `${reads.length} value${reads.length > 1 ? "s" : ""}` : "nothing yet"}</em></button>
          {shown.length > 0 && <div className="tr-reads">{shown.map((r) => <div key={r.name}><code>{r.name}</code><span className="mono">{r.value == null ? "—" : typeof r.value === "boolean" ? String(r.value) : fsFmt(r.value)}</span></div>)}{reads.length > 4 && <button className="tr-more" onClick={() => onTab("preview")}>+{reads.length - 4} more in Preview</button>}</div>}
          {unused > 0 && <p className="tr-note warn">{unused} input{unused > 1 ? "s" : ""} defined but never read</p>}
        </li>
        <li className={"tr-n " + fState + (act === "expression" || act === "details" ? " act" : "")}>
          <button className="tr-hd" onClick={() => onJump("expression")}><i className="tr-m"></i><span>Formula</span><em className="mono">{empty ? "empty" : `${an.stmts.length} line${an.stmts.length > 1 ? "s" : ""}`}</em></button>
          {ok ? <div className="tr-val mono" key={run.policy.raw}>{fsMoney(run.policy.raw)}</div>
            : !empty && <p className="tr-note bad">{errs ? `${errs} error${errs > 1 ? "s" : ""} in the expression` : run.error}</p>}
          {empty && <p className="tr-note">Write or draft a formula to start the trace.</p>}
        </li>
        <li className={"tr-n " + (!ok ? "idle" : run.policy.clamp ? "hit" : "ok") + (act === "policy" ? " act" : "")}>
          <button className="tr-hd" onClick={() => onJump("policy")}><i className="tr-m"></i><span>Charge policy</span>{ok && run.policy.clamp && <em className="tr-tag">{run.policy.clamp.bound}</em>}</button>
          <p className="tr-note">{ok && run.policy.clamp ? <>Raised to the <b>{fsMoney(run.policy.clamp.limit)}</b> {run.policy.clamp.bound}</> : p.min != null || p.max != null ? <>Inside {p.min != null ? fsMoney(p.min) : "no minimum"} – {p.max != null ? fsMoney(p.max) : "no maximum"}</> : "No minimum or maximum"}<br />Rounds {prettyMode(p.mode)} to {p.precision} place{p.precision == 1 ? "" : "s"}</p>
        </li>
        {d.breakdown.length > 0 && (
          <li className={"tr-n " + (!ok ? "idle" : run.reconcile && run.reconcile.balanced && !run.reconcile.clampMismatch ? "ok" : "warn") + (act === "breakdown" ? " act" : "")}>
            <button className="tr-hd" onClick={() => onJump("breakdown")}><i className="tr-m"></i><span>Breakdown</span><em className="mono">{d.breakdown.length} line{d.breakdown.length > 1 ? "s" : ""}</em></button>
            {lines.length > 0 && <>
              <div className="tr-bar">{lines.map((l, i) => <i key={l.id || i} className={"c" + (i % 5)} style={{ flexGrow: l.amount }} title={`${l.name} · ${fsMoney(l.amount)}`}></i>)}</div>
              <div className="tr-lines">{lines.slice(0, 4).map((l, i) => <div key={l.id || i}><i className={"c" + (i % 5)}></i><span>{l.name || "Untitled"}</span><b className="mono">{Math.round((l.amount / lsum) * 100)}%</b></div>)}</div>
            </>}
            {ok && run.reconcile && !run.reconcile.balanced && <p className="tr-note warn">Lines don't add up to the formula</p>}
          </li>
        )}
        <li className={"tr-n bill" + (ok ? "" : " idle")}>
          <button className="tr-bill" onClick={() => onTab("preview")}>
            <span>Billed</span>
            <b className="mono" key={pulse}>{ok ? fsMoney(run.policy.amount) : "—"}</b>
            <em>{ok ? (ship ? "on " + ship.id : "on the sample load") : "nothing to bill yet"}</em>
          </button>
        </li>
      </ol>

      <button className={"tr-sc" + (!results.length ? " none" : passN === results.length ? " ok" : " bad")} onClick={() => onTab("scenarios")}>
        <FI n="beaker" s={13} />
        <span>{!results.length ? "No scenarios yet" : passN === results.length ? "Scenarios hold" : `${results.length - passN} scenario${results.length - passN > 1 ? "s" : ""} broke`}</span>
        <span className="tr-dots">{results.slice(0, 10).map((r) => <i key={r.id} className={r.pass ? "p" : "f"}></i>)}</span>
        {results.length > 0 && <em className="mono">{passN}/{results.length}</em>}
      </button>

      <div className="tr-rd">
        <div className="tr-rd-h"><b>{goal}</b><em className="mono">{rc.length - open.length}/{rc.length}</em></div>
        <div className="tr-meter">{rc.map((c) => <i key={c.key} className={c.status} title={c.label}></i>)}</div>
        {open.length === 0 ? <p className="tr-all"><FI n="okc" s={13} />Every check passes</p>
          : <ul>{open.map((c) => <li key={c.key} className={c.status}><FI n={c.status === "warn" ? "alert" : "xc"} s={12} /><span>{c.label}{c.detail && <em>{c.detail}</em>}</span></li>)}</ul>}
      </div>
    </ScrollArea>
  );
}

// @trenova/shared/components/ui/scroll-area — viewport + hover scrollbar + edge masks that fade in only
// when there is more to scroll (30px, transition 300ms), tinted by maskVariant.
function ScrollArea({ as = "div", className = "", viewportClassName = "", viewportRef, maskVariant = "background", maskHeight = 30, type = "hover", children, ...rest }) {
  const vp = React.useRef(), thumb = React.useRef(), hthumb = React.useRef();
  const [m, setM] = cS({ t: 0, b: 0, l: 0, r: 0, v: false, h: false });
  const setRef = (el) => { vp.current = el; if (viewportRef) { if (typeof viewportRef === "function") viewportRef(el); else viewportRef.current = el; } };
  const check = React.useCallback(() => {
    const el = vp.current; if (!el) return;
    const { scrollTop: st, scrollLeft: sl, scrollWidth: sw, clientWidth: cw, scrollHeight: sh, clientHeight: ch } = el;
    const next = { t: st > 0, b: st + ch < sh - 1, l: sl > 0, r: sl + cw < sw - 1, v: sh > ch + 1, h: sw > cw + 1 };
    setM((p) => (p.t === next.t && p.b === next.b && p.l === next.l && p.r === next.r && p.v === next.v && p.h === next.h ? p : next));
    if (thumb.current && sh > ch) { const tr = ch - 8, th = Math.max(24, (ch / sh) * tr); thumb.current.style.height = th + "px"; thumb.current.style.transform = `translateY(${(st / (sh - ch)) * (tr - th)}px)`; }
    if (hthumb.current && sw > cw) { const tr = cw - 8, tw = Math.max(24, (cw / sw) * tr); hthumb.current.style.width = tw + "px"; hthumb.current.style.transform = `translateX(${(sl / (sw - cw)) * (tr - tw)}px)`; }
  }, []);
  cE(() => {
    const el = vp.current; if (!el) return;
    const ro = new ResizeObserver(check); ro.observe(el); [...el.children].forEach((k) => ro.observe(k));
    const mo = new MutationObserver(() => { [...el.children].forEach((k) => ro.observe(k)); check(); }); mo.observe(el, { childList: true, subtree: true });
    el.addEventListener("scroll", check, { passive: true }); window.addEventListener("resize", check);
    check();
    return () => { ro.disconnect(); mo.disconnect(); el.removeEventListener("scroll", check); window.removeEventListener("resize", check); };
  }, [check]);
  const drag = (axis) => (e) => {
    e.preventDefault(); e.stopPropagation();
    const el = vp.current, start = axis === "y" ? e.clientY : e.clientX, s0 = axis === "y" ? el.scrollTop : el.scrollLeft;
    const ratio = axis === "y" ? el.scrollHeight / el.clientHeight : el.scrollWidth / el.clientWidth;
    const mv = (ev) => { const d = ((axis === "y" ? ev.clientY : ev.clientX) - start) * ratio; if (axis === "y") el.scrollTop = s0 + d; else el.scrollLeft = s0 + d; };
    const up = () => { window.removeEventListener("pointermove", mv); window.removeEventListener("pointerup", up); };
    window.addEventListener("pointermove", mv); window.addEventListener("pointerup", up);
  };
  const Tag = as;
  return (
    <Tag data-slot="scroll-area" data-sa="root" data-mask={maskVariant} data-type={type} className={className} {...rest}>
      <div ref={setRef} data-slot="scroll-area-viewport" data-sa="vp" className={viewportClassName} tabIndex={-1}>{children}</div>
      {m.v && <div data-slot="scroll-area-scrollbar" data-sa="bar" data-orientation="vertical"><i ref={thumb} onPointerDown={drag("y")}></i></div>}
      {m.h && <div data-slot="scroll-area-scrollbar" data-sa="bar" data-orientation="horizontal"><i ref={hthumb} onPointerDown={drag("x")}></i></div>}
      {maskHeight > 0 && <>
        <div aria-hidden="true" data-sa="mask" data-axis="y" data-start={m.t ? "" : undefined} data-end={m.b ? "" : undefined} style={{ "--mh": maskHeight + "px" }}></div>
        <div aria-hidden="true" data-sa="mask" data-axis="x" data-start={m.l ? "" : undefined} data-end={m.r ? "" : undefined} style={{ "--mh": maskHeight + "px" }}></div>
      </>}
    </Tag>
  );
}

function useAnchored(open, tRef, pRef, align, side, sideOffset, alignOffset = 0) {
  const [pos, setPos] = cS(null);
  React.useLayoutEffect(() => {
    if (!open) { setPos(null); return; }
    const place = () => {
      if (!tRef.current || !pRef.current) return;
      const t = tRef.current.getBoundingClientRect(), p = pRef.current.getBoundingClientRect();
      const vw = window.innerWidth, vh = window.innerHeight;
      let left = align === "end" ? t.right - p.width - alignOffset : align === "center" ? t.left + t.width / 2 - p.width / 2 : t.left + alignOffset;
      left = Math.max(8, Math.min(left, vw - p.width - 8));
      let s = side, top = side === "top" ? t.top - p.height - sideOffset : t.bottom + sideOffset;
      if (s === "bottom" && top + p.height > vh - 8 && t.top - p.height - sideOffset > 8) { top = t.top - p.height - sideOffset; s = "top"; }
      if (s === "top" && top < 8) { top = t.bottom + sideOffset; s = "bottom"; }
      setPos({ left, top, side: s, origin: `${align === "end" ? "right" : align === "center" ? "center" : "left"} ${s === "top" ? "bottom" : "top"}` });
    };
    place();
    const ro = new ResizeObserver(place); pRef.current && ro.observe(pRef.current);
    document.addEventListener("scroll", place, true);
    return () => { ro.disconnect(); document.removeEventListener("scroll", place, true); };
  }, [open]);
  return pos;
}
function Floating({ pos, side, pRef, slot, className, style, children, ...rest }) {
  return ReactDOM.createPortal(
    <div className="ddm-pos pv-tok" style={pos ? { left: pos.left, top: pos.top } : { left: -9999, top: 0 }}>
      <div ref={pRef} data-slot={slot} data-side={pos ? pos.side : side} className={className + (pos ? " open" : "")} style={{ ...(pos ? { transformOrigin: pos.origin } : null), ...style }} {...rest}>{children}</div>
    </div>, document.body);
}
// @trenova/shared/components/ui/popover — Base UI Popover, inverted (dark) by default, p-2.5 gap-2.5 rounded-lg ring-1.
function Popover({ open, onOpenChange, trigger, align = "center", side = "bottom", sideOffset = 4, className = "", style, children }) {
  const tRef = React.useRef(), pRef = React.useRef();
  const pos = useAnchored(open, tRef, pRef, align, side, sideOffset);
  cE(() => {
    if (!open) return;
    const down = (e) => { if (!pRef.current?.contains(e.target) && !tRef.current?.contains(e.target)) onOpenChange(false); };
    const key = (e) => { if (e.key === "Escape") { onOpenChange(false); tRef.current?.focus(); } };
    document.addEventListener("pointerdown", down); document.addEventListener("keydown", key);
    return () => { document.removeEventListener("pointerdown", down); document.removeEventListener("keydown", key); };
  }, [open]);
  return <>
    {trigger({ ref: tRef, "aria-haspopup": "dialog", "aria-expanded": !!open, "data-popup-open": open ? "" : undefined, onClick: () => onOpenChange(!open) })}
    {open && <Floating pos={pos} side={side} pRef={pRef} slot="popover-content" className={"pvc " + className} style={style} role="dialog">{children}</Floating>}
  </>;
}
// @trenova/shared/components/ui/hover-card — Base UI PreviewCard, always dark, w-64 p-2.5.
function HoverCard({ trigger, align = "center", side = "bottom", sideOffset = 4, className = "", children }) {
  const [open, setOpen] = cS(false);
  const tRef = React.useRef(), pRef = React.useRef(), tm = React.useRef();
  const pos = useAnchored(open, tRef, pRef, align, side, sideOffset, align === "center" ? 0 : 4);
  const enter = () => { clearTimeout(tm.current); tm.current = setTimeout(() => setOpen(true), 250); };
  const leave = () => { clearTimeout(tm.current); tm.current = setTimeout(() => setOpen(false), 120); };
  cE(() => () => clearTimeout(tm.current), []);
  return <>
    {trigger({ ref: tRef, onMouseEnter: enter, onMouseLeave: leave, onFocus: () => setOpen(true), onBlur: () => setOpen(false), tabIndex: 0, "data-popup-open": open ? "" : undefined })}
    {open && <Floating pos={pos} side={side} pRef={pRef} slot="hover-card-content" className={"pvc hvc " + className} onMouseEnter={enter} onMouseLeave={leave}>{children}</Floating>}
  </>;
}
const UsageCard = ({ version }) => <>
  <div className="pvc-h"><b>Where this template is used</b></div>
  <div className="pvc-rows">{FS_USAGE.map((u) => <div key={u.type}><span>{u.type}</span><b className="mono">{u.count}</b></div>)}</div>
  <p className="pvc-d">Editing the formula returns it to Draft; shipments keep rating with v{version} until the new version is approved. Name and description edits are safe.</p>
</>;

// @trenova/shared/components/ui/dropdown-menu — Base UI Menu: portalled, positioned to the trigger
// (side bottom, sideOffset 4), always the dark popover, p-1 list, px-1.5 py-1 items.
const DDCtx = React.createContext(() => {});
function DropdownMenu({ trigger, align = "start", side = "bottom", sideOffset = 4, children }) {
  const [open, setOpen] = cS(false);
  const tRef = React.useRef(), pRef = React.useRef();
  const pos = useAnchored(open, tRef, pRef, align, side, sideOffset);
  cE(() => {
    if (!open) return;
    const down = (e) => { if (!pRef.current?.contains(e.target) && !tRef.current?.contains(e.target)) setOpen(false); };
    const key = (e) => { if (e.key === "Escape") { setOpen(false); tRef.current?.focus(); } };
    const away = () => setOpen(false);
    document.addEventListener("pointerdown", down); document.addEventListener("keydown", key); window.addEventListener("resize", away);
    return () => { document.removeEventListener("pointerdown", down); document.removeEventListener("keydown", key); window.removeEventListener("resize", away); };
  }, [open]);
  return <>
    {trigger({ ref: tRef, "aria-haspopup": "menu", "aria-expanded": open, "data-popup-open": open ? "" : undefined, onClick: () => setOpen((o) => !o) })}
    {open && <Floating pos={pos} side={side} pRef={pRef} slot="dropdown-menu-content" className="ddm" role="menu"><ScrollArea className="ddm-sa" viewportClassName="ddm-l" maskVariant="popover"><DDCtx.Provider value={() => setOpen(false)}>{children}</DDCtx.Provider></ScrollArea></Floating>}
  </>;
}
function DropdownMenuItem({ icon, title, description, color, disabled, onClick, endContent }) {
  const close = React.useContext(DDCtx);
  return (
    <button role="menuitem" data-slot="dropdown-menu-item" data-color={color} className="ddm-i" disabled={disabled} onClick={() => { close(); onClick && onClick(); }}>
      {icon && <span className="ddm-s"><FI n={icon} s={16} /></span>}
      <span className="ddm-t"><span>{title}</span>{description && <span className="ddm-d">{description}</span>}</span>
      {endContent && <span className="ddm-e">{endContent}</span>}
    </button>
  );
}
function DropdownMenuSeparator() { return <div role="separator" className="ddm-sep"></div>; }
function DropdownMenuLabel({ children }) { return <div className="ddm-l">{children}</div>; }

Object.assign(window, { ScrollArea, StudioTop, RateTrace, Popover, HoverCard, UsageCard, DropdownMenu, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuLabel });
