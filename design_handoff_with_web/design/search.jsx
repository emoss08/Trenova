const SEARCH_RECENT = ["missing biller", "storm warning", "posted this week"];

function flatText(def) {
  return def.text.map(p => typeof p === "string" ? p.replace(/[#*`>|_~-]+/g, " ").replace(/\s+/g, " ") : p.map(s => typeof s === "string" ? s : s.b || s.t || "").join("")).join(" ");
}

function buildIndex() {
  const out = [];
  CONVOS.forEach(g => g.items.forEach(c => out.push({ kind: "chat", id: "c:" + c.t, title: c.t, meta: c.a + " · " + c.w, agent: c.k, on: c.on, hay: c.t + " " + c.a })));
  Object.values(TURNS).forEach((def, i) => {
    const txt = flatText(def);
    out.push({ kind: "msg", id: "m" + i, title: txt, meta: "Billing Specialist · " + def.time, hay: txt, body: txt });
  });
  Object.keys(ART2).forEach(id => { const a = artMeta(id); out.push({ kind: "art", id: "a:" + id, art: id, title: a.title, meta: KIND[a.kind].label + " · " + a.src, hay: a.title + " " + a.src + " " + KIND[a.kind].label }); });
  Object.values(DECISIONS).forEach(d => out.push({ kind: "dec", id: "d:" + d.id, title: d.title + " · " + d.scope, meta: d.meta.join(" · "), hay: d.title + " " + d.scope + " " + d.meta.join(" ") }));
  return out;
}

function snippet(text, q) {
  if (!q) return text;
  const i = text.toLowerCase().indexOf(q.toLowerCase());
  if (i < 0) return text;
  const s = Math.max(0, i - 36);
  return (s > 0 ? "…" : "") + text.slice(s);
}

function Hl({ text, q }) {
  if (!q) return text;
  const lo = text.toLowerCase(), ql = q.toLowerCase();
  const parts = []; let at = 0, i;
  while ((i = lo.indexOf(ql, at)) >= 0 && parts.length < 12) {
    if (i > at) parts.push(text.slice(at, i));
    parts.push(<mark key={i}>{text.slice(i, i + q.length)}</mark>);
    at = i + q.length;
  }
  parts.push(text.slice(at));
  return parts;
}

const SK = { chat: ["Conversations", "chat"], msg: ["Messages", "copy"], art: ["Artifacts", "table"], dec: ["Decisions", "inbox"] };
const SFILTERS = [["all", "All"], ["chat", "Chats"], ["msg", "Messages"], ["art", "Artifacts"], ["dec", "Decisions"]];

function SearchPalette({ onClose, onOpenThread, onOpenArt, onOpenDecisions }) {
  const [q, setQ] = React.useState("");
  const [f, setF] = React.useState("all");
  const [sel, setSel] = React.useState(0);
  const [closing, setClosing] = React.useState(false);
  const idx = React.useMemo(buildIndex, []);
  const inp = React.useRef(null);
  const list = React.useRef(null);
  React.useEffect(() => { inp.current && inp.current.focus(); }, []);

  const query = q.trim();
  const results = React.useMemo(() => {
    let xs = idx.filter(x => f === "all" || x.kind === f);
    if (query) xs = xs.filter(x => x.hay.toLowerCase().includes(query.toLowerCase()));
    else xs = xs.filter(x => x.kind === "chat" || x.kind === "art").slice(0, 7);
    return xs.slice(0, 14);
  }, [idx, f, query]);
  const groups = [];
  results.forEach((r, i) => { let g = groups.find(g => g.k === r.kind); if (!g) groups.push(g = { k: r.kind, items: [] }); g.items.push([r, i]); });
  const flat = groups.flatMap(g => g.items.map(([r]) => r));

  React.useEffect(() => { setSel(0); }, [q, f]);
  React.useEffect(() => {
    const el = list.current && list.current.querySelector('[data-i="' + sel + '"]');
    if (el) { const p = list.current; const t = el.offsetTop, b = t + el.offsetHeight; if (t < p.scrollTop + 32) p.scrollTop = t - 32; else if (b > p.scrollTop + p.clientHeight) p.scrollTop = b - p.clientHeight + 8; }
  }, [sel]);

  const close = () => { setClosing(true); setTimeout(onClose, 160); };
  const pick = r => {
    if (!r) return;
    if (r.kind === "art") onOpenArt(r.art);
    else if (r.kind === "dec") onOpenDecisions();
    else if (r.on || r.kind === "msg") onOpenThread();
    close();
  };
  const onKey = e => {
    e.stopPropagation();
    if (e.key === "Escape") { e.preventDefault(); close(); }
    else if (e.key === "ArrowDown") { e.preventDefault(); setSel(s => Math.min(flat.length - 1, s + 1)); }
    else if (e.key === "ArrowUp") { e.preventDefault(); setSel(s => Math.max(0, s - 1)); }
    else if (e.key === "Enter") { e.preventDefault(); pick(flat[sel]); }
    else if (e.key === "Tab") { e.preventDefault(); const k = SFILTERS.findIndex(x => x[0] === f); setF(SFILTERS[(k + (e.shiftKey ? SFILTERS.length - 1 : 1)) % SFILTERS.length][0]); }
  };
  let n = -1;
  return (
    <div className={"srch-wrap" + (closing ? " out" : "")} onMouseDown={e => { if (e.target === e.currentTarget) close(); }}>
      <div className="srch" onKeyDown={onKey}>
        <div className="srch-in">
          <Ic n="search" s={16} />
          <input ref={inp} value={q} onChange={e => setQ(e.target.value)} placeholder="Search chats, messages, artifacts…" />
          {q ? <button className="srch-clr" onClick={() => { setQ(""); inp.current.focus(); }}>Clear</button> : <span className="kbd">Esc</span>}
        </div>
        <div className="srch-f">
          {SFILTERS.map(([k, l]) => <button key={k} className={f === k ? "on" : ""} onClick={() => { setF(k); inp.current.focus(); }}>{l}</button>)}
        </div>
        <div className="srch-l" ref={list}>
          {!query && f === "all" && (
            <div className="srch-rec">
              <div className="srch-gh">Recent searches</div>
              <div className="srch-chips">{SEARCH_RECENT.map(r => <button key={r} onClick={() => setQ(r)}><Ic n="replay" s={11} />{r}</button>)}</div>
            </div>
          )}
          {groups.map(g => (
            <div key={g.k}>
              <div className="srch-gh">{query ? SK[g.k][0] : g.k === "chat" ? "Recent conversations" : "Recent artifacts"}<i>{g.items.length}</i></div>
              {g.items.map(([r]) => {
                n += 1; const i = n;
                return (
                  <button key={r.id} data-i={i} className={"srch-r k-" + r.kind + (sel === i ? " on" : "")} onMouseMove={() => sel !== i && setSel(i)} onClick={() => pick(r)}>
                    <span className="srch-ic">{r.kind === "chat" ? <span className={"sb-dot" + (r.agent === "dispatch" ? " v" : "")}><i></i></span> : <Ic n={SK[r.kind][1]} s={13} />}</span>
                    <span className="srch-tx">
                      <span className="srch-t"><Hl text={r.kind === "msg" ? snippet(r.title, query) : r.title} q={query} /></span>
                      <span className="srch-m">{r.meta}</span>
                    </span>
                    <span className="srch-go"><Ic n="enter" s={12} /></span>
                  </button>
                );
              })}
            </div>
          ))}
          {query && !results.length && (
            <div className="srch-empty">
              <b>Nothing matches “{query}”</b>
              <span>Try a load number, customer or invoice ID.</span>
            </div>
          )}
        </div>
        <div className="srch-ft">
          <span><span className="kbd">↑</span><span className="kbd">↓</span>to move</span>
          <span><span className="kbd">↵</span>to open</span>
          <span><span className="kbd">Tab</span>to filter</span>
          <span style={{ flex: 1 }}></span>
          <span>{query ? results.length + " results" : ""}</span>
        </div>
      </div>
    </div>
  );
}

Object.assign(window, { SearchPalette });
