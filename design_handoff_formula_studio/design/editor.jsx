const { useState: eS, useRef: eR, useEffect: eE, useMemo: eM, useLayoutEffect: eL, forwardRef, useImperativeHandle } = React;
const ED_LH = 20, ED_PT = 12, ED_PL = 14;

function useCharWidth() {
  const [w, setW] = eS(7.8);
  eE(() => {
    const c = document.createElement("canvas").getContext("2d");
    const set = () => { c.font = "13px 'Geist Mono'"; setW(c.measureText("0000000000").width / 10); };
    set();
    document.fonts && document.fonts.ready.then(set);
  }, []);
  return w;
}

function edCandidates(an, ctx) {
  const out = [];
  (an.lets || []).forEach((l) => out.push({ label: l.name, kind: "let", type: "let", detail: "Defined above in this formula" }));
  ctx.vars.forEach((v) => out.push({ label: v.name, kind: v.custom ? "input" : "var", type: v.type, detail: v.description || (v.custom ? "This template's input" : "") }));
  FS_FUNCS.forEach((f) => out.push({ label: f.name, kind: "fn", type: "fn", detail: f.signature }));
  const seen = new Set();
  return out.filter((c) => (seen.has(c.label) ? false : seen.add(c.label)));
}

const ExprEditor = forwardRef(function ExprEditor({ value, onChange, ctx, an, mini, placeholder, minLines = 5, focus, onCursor, label }, ref) {
  const ta = eR(), hl = eR(), box = eR();
  const cw = useCharWidth();
  const [ac, setAc] = eS(null);
  const cands = eM(() => edCandidates(an, ctx), [an, ctx]);

  const setVal = (v, caret) => {
    onChange(v);
    requestAnimationFrame(() => { if (ta.current && caret != null) { ta.current.focus(); ta.current.setSelectionRange(caret, caret); } });
  };
  useImperativeHandle(ref, () => ({
    insert(text, off) {
      const el = ta.current, s = el ? el.selectionStart : value.length, e = el ? el.selectionEnd : value.length;
      setVal(value.slice(0, s) + text + value.slice(e), s + (off ?? text.length));
    },
    focus() { ta.current && ta.current.focus(); },
  }), [value]);

  function refreshAc(auto, v = value, pos = ta.current ? ta.current.selectionStart : 0, force) {
    const before = v.slice(0, pos);
    const m = /[A-Za-z_][\w.]*$/.exec(before);
    const word = m ? m[0] : "";
    const inStr = (before.split("\n").pop().match(/"/g) || []).length % 2 === 1;
    if ((!word && !force) || inStr || (auto && !word)) { setAc(null); return; }
    const w = word.toLowerCase();
    let items = cands.filter((c) => c.label.toLowerCase().startsWith(w));
    items = items.concat(cands.filter((c) => !c.label.toLowerCase().startsWith(w) && c.label.toLowerCase().includes(w)));
    if (items.length === 1 && items[0].label === word) items = [];
    items = items.slice(0, 8);
    if (!items.length) { setAc(null); return; }
    const from = pos - word.length;
    const pre = v.slice(0, from);
    const ln = pre.split("\n").length - 1, col = from - pre.lastIndexOf("\n") - 1;
    const el = ta.current;
    setAc({ items, idx: 0, from, to: pos, top: ED_PT + (ln + 1) * ED_LH - (el ? el.scrollTop : 0) + 4, left: Math.max(4, (mini ? 10 : ED_PL) + col * cw - (el ? el.scrollLeft : 0) - 4) });
  }
  function accept(item) {
    const isFn = item.kind === "fn";
    const ins = item.label + (isFn ? "()" : "");
    const v = value.slice(0, ac.from) + ins + value.slice(ac.to);
    setAc(null);
    onChange(v);
    const caret = ac.from + ins.length - (isFn ? 1 : 0);
    requestAnimationFrame(() => { ta.current.focus(); ta.current.setSelectionRange(caret, caret); });
  }
  function onKey(e) {
    if (ac) {
      if (e.key === "ArrowDown") { e.preventDefault(); setAc({ ...ac, idx: (ac.idx + 1) % ac.items.length }); return; }
      if (e.key === "ArrowUp") { e.preventDefault(); setAc({ ...ac, idx: (ac.idx - 1 + ac.items.length) % ac.items.length }); return; }
      if (e.key === "Enter" || e.key === "Tab") { e.preventDefault(); accept(ac.items[ac.idx]); return; }
      if (e.key === "Escape") { e.preventDefault(); setAc(null); return; }
    }
    if (e.key === " " && (e.ctrlKey || e.metaKey)) { e.preventDefault(); refreshAc(false, value, ta.current.selectionStart, true); return; }
    if (e.key === "Tab" && !mini) { e.preventDefault(); const s = ta.current.selectionStart; setVal(value.slice(0, s) + "  " + value.slice(ta.current.selectionEnd), s + 2); }
    if (e.key === "Enter" && mini) e.preventDefault();
  }
  const reportCursor = () => {
    if (!onCursor || !ta.current) return;
    const pre = value.slice(0, ta.current.selectionStart);
    onCursor({ ln: pre.split("\n").length, col: pre.length - pre.lastIndexOf("\n") });
  };
  const sync = () => { if (hl.current && ta.current) { hl.current.scrollLeft = ta.current.scrollLeft; hl.current.scrollTop = ta.current.scrollTop; } };

  const marks = eM(() => an.toks.map((tk) => {
    const hit = an.issues.find((i) => i.s < tk.e && i.e > tk.s && i.e > i.s);
    return hit ? hit.sev : null;
  }), [an]);
  const lines = Math.max((value || "").split("\n").length, mini ? 1 : minLines);

  return (
    <div ref={box} className={"ed" + (mini ? " mini" : "")} onBlur={(e) => { if (!box.current.contains(e.relatedTarget)) setAc(null); }}>
      {!mini && <div className="ed-gut" aria-hidden="true">{Array.from({ length: lines }, (_, i) => <span key={i}>{i + 1}</span>)}</div>}
      <div className="ed-area" style={{ height: lines * ED_LH + (mini ? 12 : ED_PT * 2) }}>
        <pre ref={hl} className="ed-hl" aria-hidden="true">
          {an.toks.map((tk, i) => {
            const c = an.cls[i];
            const inFocus = focus && tk.s >= focus.s && tk.e <= focus.e && tk.t !== "ws";
            return <span key={i} className={"tk-" + c + (marks[i] ? " sq-" + marks[i] : "") + (inFocus ? " fo" : "")}>{tk.v}</span>;
          })}
          {"\n "}
        </pre>
        {!value && <div className="ed-ph">{placeholder}</div>}
        <textarea ref={ta} value={value} spellCheck="false" autoCapitalize="off" autoComplete="off" aria-label={label || "Expression"}
          onChange={(e) => { onChange(e.target.value); const p = e.target.selectionStart, v = e.target.value; requestAnimationFrame(() => refreshAc(true, v, p)); }}
          onKeyDown={onKey} onKeyUp={reportCursor} onClick={() => { reportCursor(); setAc(null); }} onScroll={sync} wrap="off"></textarea>
        {ac && (
          <div className="ac" style={{ top: ac.top, left: ac.left }} role="listbox">
            {ac.items.map((it, i) => (
              <button key={it.label} type="button" tabIndex={-1} className={"ac-o" + (i === ac.idx ? " on" : "")} onMouseDown={(e) => { e.preventDefault(); accept(it); }} onMouseEnter={() => setAc({ ...ac, idx: i })}>
                <i className={"ac-k " + it.kind}>{it.kind === "fn" ? "ƒ" : it.kind === "let" ? "=" : it.kind === "input" ? "in" : "x"}</i>
                <b className="mono">{it.label}</b>
                <em>{it.kind === "fn" ? it.detail : it.type}</em>
              </button>
            ))}
            {ac.items[ac.idx] && ac.items[ac.idx].kind !== "fn" && ac.items[ac.idx].detail && <div className="ac-d">{ac.items[ac.idx].detail}</div>}
          </div>
        )}
      </div>
    </div>
  );
});

Object.assign(window, { ExprEditor });
