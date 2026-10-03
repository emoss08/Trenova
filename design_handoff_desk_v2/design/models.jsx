const LOGO = {
  anthropic: "M17.3041 3.541h-3.6718l6.696 16.918H24Zm-10.6082 0L0 20.459h3.7442l1.3693-3.5527h7.0052l1.3693 3.5528h3.7442L10.5363 3.5409Zm-.3712 10.2232 2.2914-5.9456 2.2914 5.9456Z",
  openai: "M22.2819 9.8211a5.9847 5.9847 0 0 0-.5157-4.9108 6.0462 6.0462 0 0 0-6.5098-2.9A6.0651 6.0651 0 0 0 4.9807 4.1818a5.9847 5.9847 0 0 0-3.9977 2.9 6.0462 6.0462 0 0 0 .7427 7.0966 5.98 5.98 0 0 0 .511 4.9107 6.051 6.051 0 0 0 6.5146 2.9001A5.9847 5.9847 0 0 0 13.2599 24a6.0557 6.0557 0 0 0 5.7718-4.2058 5.9894 5.9894 0 0 0 3.9977-2.9001 6.0557 6.0557 0 0 0-.7475-7.0729zm-9.022 12.6081a4.4755 4.4755 0 0 1-2.8764-1.0408l.1419-.0804 4.7783-2.7582a.7948.7948 0 0 0 .3927-.6813v-6.7369l2.02 1.1686a.071.071 0 0 1 .038.052v5.5826a4.504 4.504 0 0 1-4.4945 4.4944zm-9.6607-4.1254a4.4708 4.4708 0 0 1-.5346-3.0137l.142.0852 4.783 2.7582a.7712.7712 0 0 0 .7806 0l5.8428-3.3685v2.3324a.0804.0804 0 0 1-.0332.0615L9.74 19.9502a4.4992 4.4992 0 0 1-6.1408-1.6464zM2.3408 7.8956a4.485 4.485 0 0 1 2.3655-1.9728V11.6a.7664.7664 0 0 0 .3879.6765l5.8144 3.3543-2.0201 1.1685a.0757.0757 0 0 1-.071 0l-4.8303-2.7865A4.504 4.504 0 0 1 2.3408 7.872zm16.5963 3.8558L13.1038 8.364 15.1192 7.2a.0757.0757 0 0 1 .071 0l4.8303 2.7913a4.4944 4.4944 0 0 1-.6765 8.1042v-5.6772a.79.79 0 0 0-.407-.667zm2.0107-3.0231l-.142-.0852-4.7735-2.7818a.7759.7759 0 0 0-.7854 0L9.409 9.2297V6.8974a.0662.0662 0 0 1 .0284-.0615l4.8303-2.7866a4.4992 4.4992 0 0 1 6.6802 4.66zM8.3065 12.863l-2.02-1.1638a.0804.0804 0 0 1-.038-.0567V6.0742a4.4992 4.4992 0 0 1 7.3757-3.4537l-.142.0805L8.704 5.459a.7948.7948 0 0 0-.3927.6813zm1.0976-2.3654l2.602-1.4998 2.6069 1.4998v2.9994l-2.5974 1.4997-2.6067-1.4997Z",
  gemini: "M11.04 19.32Q12 21.51 12 24q0-2.49.93-4.68.96-2.19 2.58-3.81t3.81-2.55Q21.51 12 24 12q-2.49 0-4.68-.93a12.3 12.3 0 0 1-3.81-2.58 12.3 12.3 0 0 1-2.58-3.81Q12 2.49 12 0q0 2.49-.96 4.68-.93 2.19-2.55 3.81a12.3 12.3 0 0 1-3.81 2.58Q2.49 12 0 12q2.49 0 4.68.96 2.19.93 3.81 2.55t2.55 3.81",
  mistral: "M17.143 3.429v3.428h-3.429v3.429h-3.428V6.857H6.857V3.43H3.43v13.714H0v3.428h10.286v-3.428H6.857v-3.429h3.429v3.429h3.429v-3.429h3.428v3.429h-3.428v3.428H24v-3.428h-3.43V3.429z",
  groq: "m18.445 4.406-9.468 13.74 7.341.665-1.69 9.578 9.469-13.74-7.342-.664 1.69-9.579Z",
};
const LOGO_VB = { groq: "0.54 0.39 32 32" };

function BrandMark({ id, s = 14 }) {
  if (id === "auto") return (
    <svg width={s} height={s} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinejoin="round" className="mk-auto">
      <path d="M2.7 10.3a2.41 2.41 0 0 0 0 3.41l7.59 7.59a2.41 2.41 0 0 0 3.41 0l7.59-7.59a2.41 2.41 0 0 0 0-3.41l-7.59-7.59a2.41 2.41 0 0 0-3.41 0Z"></path>
      <circle cx="12" cy="12" r="2" fill="currentColor" stroke="none"></circle>
    </svg>
  );
  return <svg width={s} height={s} viewBox={LOGO_VB[id] || "0 0 24 24"} fill="currentColor" className={"mk mk-" + id}><path d={LOGO[id]}></path></svg>;
}

// The org's configured endpoints, in its priority order (Auto walks this list).
const MODEL_OPTIONS = [
  { id: "aiprv_claude_sonnet", p: "anthropic", model: "Claude Sonnet 4.5", name: "Anthropic · primary", tags: ["Balanced"] },
  { id: "aiprv_claude_opus", p: "anthropic", model: "Claude Opus 4.1", name: "Anthropic", tags: ["Deep reasoning"] },
  { id: "aiprv_gpt5", p: "openai", model: "GPT-5", name: "OpenAI", tags: ["Deep reasoning"] },
  { id: "aiprv_gpt5_mini", p: "openai", model: "GPT-5 mini", name: "OpenAI", tags: ["Fast"] },
  { id: "aiprv_gemini", p: "gemini", model: "Gemini 2.5 Pro", name: "Google", tags: ["Long context"] },
  { id: "aiprv_groq", p: "groq", model: "Llama 3.3 70B", name: "Groq", tags: ["Fastest"] },
  { id: "aiprv_mistral", p: "mistral", model: "Mistral Large", name: "Mistral · EU", tags: ["Unavailable"], down: true },
];
const PROV_NAME = { anthropic: "Anthropic", openai: "OpenAI", gemini: "Google", groq: "Groq", mistral: "Mistral" };

const RAIN_P = ["anthropic", "openai", "gemini", "groq", "mistral"];
function GemDefs() {
  return <svg width="0" height="0" style={{ position: "absolute" }} aria-hidden="true"><defs><linearGradient id="gemGrad" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stopColor="#4f8cff"></stop><stop offset=".55" stopColor="#a77bff"></stop><stop offset="1" stopColor="#ff7a9a"></stop></linearGradient></defs></svg>;
}

function LogoRain() {
  const cols = React.useMemo(() => Array.from({ length: 18 }, (_, c) => {
    const r = n => { const x = Math.sin(c * 12.9898 + n * 78.233) * 43758.5453; return x - Math.floor(x); };
    const cells = Array.from({ length: 5 }, (_, k) => RAIN_P[Math.floor(r(k + 3) * 5)]);
    return { cells, dur: 1.6 + r(1) * 1.8, delay: -r(2) * 3, op: 0.35 + r(9) * 0.5 };
  }), []);
  return (
    <span className="ai-rain" aria-hidden="true">
      {cols.map((col, c) => (
        <span key={c} className="ai-col" style={{ left: c * 18 + 4 + "px", animationDuration: col.dur + "s", animationDelay: col.delay + "s", opacity: col.op }}>
          {[...col.cells, ...col.cells].map((p, k) => <span key={k} className={"ai-drop p-" + p + (k % 5 === 4 ? " head" : "")}><BrandMark id={p} s={8} /></span>)}
        </span>
      ))}
    </span>
  );
}

function ModelPicker({ value, onChange, hasReplies }) {
  const [open, setOpen] = React.useState(false);
  const [q, setQ] = React.useState("");
  const [hi, setHi] = React.useState(0);
  const [spin, setSpin] = React.useState(0);
  const root = React.useRef(null);
  const inp = React.useRef(null);
  const sel = MODEL_OPTIONS.find(o => o.id === value) || null;
  const chain = [...new Set(MODEL_OPTIONS.filter(o => !o.down).map(o => o.p))].slice(0, 3);

  React.useEffect(() => {
    if (!open) return;
    setQ(""); setHi(0);
    const t = setTimeout(() => inp.current && inp.current.focus(), 30);
    const off = e => { if (root.current && !root.current.contains(e.target)) setOpen(false); };
    document.addEventListener("mousedown", off);
    return () => { clearTimeout(t); document.removeEventListener("mousedown", off); };
  }, [open]);

  const ql = q.trim().toLowerCase();
  const showAuto = !ql || "auto".includes(ql);
  const opts = MODEL_OPTIONS.filter(o => !ql || (o.model + " " + o.name + " " + PROV_NAME[o.p]).toLowerCase().includes(ql));
  const rows = [...(showAuto ? [{ id: "", auto: true }] : []), ...opts];
  const pick = r => { if (!r || r.down) return; if (r.id !== value) setSpin(x => x + 1); onChange(r.id); setOpen(false); };
  const onKey = e => {
    if (e.key === "ArrowDown") { e.preventDefault(); setHi(h => Math.min(rows.length - 1, h + 1)); }
    else if (e.key === "ArrowUp") { e.preventDefault(); setHi(h => Math.max(0, h - 1)); }
    else if (e.key === "Enter") { e.preventDefault(); pick(rows[hi]); }
    else if (e.key === "Escape") { e.preventDefault(); e.stopPropagation(); setOpen(false); }
  };
  React.useEffect(() => { setHi(0); }, [q]);

  let lastP = null, i = -1;
  return (
    <div className="mp2" ref={root}>
      <GemDefs />
      <button className={"mp2-b" + (open ? " on" : "") + (sel ? "" : " auto")} onClick={() => setOpen(o => !o)} aria-label="Choose which model answers">
        {sel ? <span className="mp2-bi" key={spin}><BrandMark id={sel.p} s={13} /></span> : (
          <span className="mp2-bi tr-reel" key={spin}>
            <span className="tr-strip">{["auto", "anthropic", "openai", "gemini", "groq", "auto"].map((p, k) => <span key={k} className={"tr-cell p-" + p}><BrandMark id={p} s={p === "auto" ? 13 : 11} /></span>)}</span>
          </span>
        )}
        <span className="mp2-bl">{sel ? sel.model : <span className="tr-word">Auto</span>}</span>
        {!sel && <span className="tr-ring" aria-hidden="true"></span>}
        <svg className="mp2-cv" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round"><path d="M6 9l6 6 6-6"></path></svg>
      </button>
      {open && (
        <div className="mp2-pop" onKeyDown={onKey}>
          <div className="mp2-s"><Ic n="search" s={13} /><input ref={inp} value={q} onChange={e => setQ(e.target.value)} placeholder="Search models…" /></div>
          <div className="mp2-l">
            {rows.map((r, k) => {
              i += 1; const idx = i;
              if (r.auto) return (
                <button key="auto" className={"mp2-auto" + (hi === idx ? " hi" : "") + (!value ? " sel" : "")} onMouseMove={() => hi !== idx && setHi(idx)} onClick={() => pick(r)} style={{ animationDelay: "0ms" }}>
                  <LogoRain />
                  <span className="mp2-ai">
                    <span className="ai-reel">{["anthropic", "openai", "gemini", "groq", "mistral"].map(p => <span key={p} className={"ai-cell p-" + p}><BrandMark id={p} s={14} /></span>)}<span className="ai-cell ai-home"><BrandMark id="auto" s={17} /></span></span>
                    <span className="ai-orbit">{["anthropic", "openai", "gemini"].map((p, k) => <span key={p} className={"ai-sat p-" + p} style={{ "--k": k }}><BrandMark id={p} s={8} /></span>)}</span>
                  </span>
                  <span className="mp2-tx"><b><span className="ai-word">Auto</span><em>Recommended</em></b></span>
                  {!value && <span className="mp2-ck"><Ic n="check" s={13} w={2.4} /></span>}
                </button>
              );
              const head = r.p !== lastP; lastP = r.p;
              return (
                <React.Fragment key={r.id}>
                  {head && <div className="mp2-gh"><BrandMark id={r.p} s={10} />{PROV_NAME[r.p]}</div>}
                  <button className={"mp2-r" + (hi === idx ? " hi" : "") + (value === r.id ? " sel" : "") + (r.down ? " down" : "")} onMouseMove={() => hi !== idx && setHi(idx)} onClick={() => pick(r)} style={{ animationDelay: Math.min(k, 8) * 22 + "ms" }} title={r.down ? "This endpoint failed its last health check" : ""}>
                    <span className={"mp2-ri p-" + r.p}><BrandMark id={r.p} s={13} /></span>
                    <span className="mp2-tx"><b>{r.model}</b><span>{r.name}</span></span>
                    <span className={"mp2-tag" + (r.down ? " bad" : "")}>{r.tags[0]}</span>
                    {value === r.id && <span className="mp2-ck"><Ic n="check" s={13} w={2.4} /></span>}
                  </button>
                </React.Fragment>
              );
            })}
            {!rows.length && <div className="mp2-empty">No models match “{q}”</div>}
          </div>
          <div className="mp2-f">{hasReplies ? <><Ic n="info" s={12} />Switching re-reads this conversation before the next reply.</> : <>Your organization's admins choose which models are available.</>}</div>
        </div>
      )}
    </div>
  );
}

Object.assign(window, { ModelPicker, BrandMark, MODEL_OPTIONS });
