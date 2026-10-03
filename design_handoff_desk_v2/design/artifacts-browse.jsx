function ArtBrowser({ ids, active, pinned, onPick, onBack, onClose, flags = {} }) {
  const [q, setQ] = React.useState("");
  const [kind, setKind] = React.useState("all");
  const [onlyPinned, setOnlyPinned] = React.useState(false);
  const [limit, setLimit] = React.useState(60);
  const inp = React.useRef(null);
  const sent = React.useRef(null);
  const list = React.useRef(null);
  React.useEffect(() => { const t = setTimeout(() => inp.current && inp.current.focus(), 60); return () => clearTimeout(t); }, []);
  React.useEffect(() => { setLimit(60); if (list.current) list.current.scrollTop = 0; }, [q, kind, onlyPinned]);

  const all = React.useMemo(() => ids.map(artMeta), [ids]);
  const ql = q.trim().toLowerCase();
  const base = all.filter(a => (!onlyPinned || pinned[a.id]) && (!ql || (a.title + " " + a.src + " " + (a.turn || "") + " " + KIND[a.kind].label).toLowerCase().includes(ql)));
  const counts = {}; base.forEach(a => { counts[a.kind] = (counts[a.kind] || 0) + 1; });
  const shown = base.filter(a => kind === "all" || a.kind === kind);
  const page = shown.slice(0, limit);

  React.useEffect(() => {
    const el = sent.current; if (!el) return;
    const io = new IntersectionObserver(es => { if (es[0].isIntersecting) setLimit(l => l + 60); }, { root: list.current, rootMargin: "200px" });
    io.observe(el); return () => io.disconnect();
  }, [page.length, shown.length]);

  const groups = [];
  page.forEach(a => {
    let d = groups[groups.length - 1];
    if (!d || d.day !== a.day) groups.push(d = { day: a.day, turns: [] });
    let t = d.turns[d.turns.length - 1];
    if (!t || t.turn !== a.turn) d.turns.push(t = { turn: a.turn, items: [] });
    t.items.push(a);
  });
  const hl = s => { if (!ql) return s; const i = s.toLowerCase().indexOf(ql); return i < 0 ? s : <>{s.slice(0, i)}<mark>{s.slice(i, i + ql.length)}</mark>{s.slice(i + ql.length)}</>; };
  const kinds = Object.keys(KIND).filter(k => counts[k]);

  return (
    <div className="axb">
      <div className="axb-top">
        <button className="ax-ib" onClick={onBack} title="Back"><AI n="a-up" s={13} w={2.2} /></button>
        <b>All artifacts</b><span className="axb-n">{ids.length}</span>
        <span style={{ flex: 1 }}></span>
        <button className="ax-ib" onClick={onClose} title="Close"><AI n="a-x" s={13} w={2.2} /></button>
      </div>
      <div className="axb-s"><AI n="a-search" s={14} /><input ref={inp} value={q} onChange={e => setQ(e.target.value)} placeholder="Search titles, tools, turns…" onKeyDown={e => { if (e.key === "Escape") { e.stopPropagation(); q ? setQ("") : onBack(); } if (e.key === "Enter" && shown[0]) onPick(shown[0].id); }} /><span className="kbd">Esc</span></div>
      <div className="axb-f">
        <button className={kind === "all" && !onlyPinned ? "on" : ""} onClick={() => { setKind("all"); setOnlyPinned(false); }}>All <i>{base.length}</i></button>
        <button className={onlyPinned ? "on" : ""} onClick={() => setOnlyPinned(p => !p)}><AI n="a-pin" s={11} />Pinned <i>{Object.values(pinned).filter(Boolean).length}</i></button>
        {kinds.map(k => <button key={k} className={kind === k ? "on" : ""} onClick={() => setKind(x => x === k ? "all" : k)}><span className={"ax-ki k-" + k}><AI n={KIND[k].icon} s={11} w={2} /></span>{KIND[k].label}{k === "table" || k === "record" || k === "report" ? "s" : ""} <i>{counts[k]}</i></button>)}
      </div>
      <div className="axb-l" ref={list}>
        {groups.map(g => (
          <section key={g.day}>
            <div className="axb-day">{g.day}</div>
            {g.turns.map(t => (
              <div key={t.turn} className="axb-turn">
                <div className="axb-th"><AgentMark s={8} /><span>{t.turn.split(" · ")[1]}</span><span className="axb-tt">{t.turn.split(" · ")[0]}</span></div>
                {t.items.map(a => (
                  <button key={a.id} className={"axb-r" + (a.id === active ? " on" : "")} onClick={() => onPick(a.id)}>
                    <span className={"ax-ki k-" + a.kind}><AI n={KIND[a.kind].icon} s={14} /></span>
                    <span className="axb-rt"><b>{hl(a.title)}</b><span>{KIND[a.kind].label} · {a.count}</span></span>
                    {a.versions && <span className="axb-v">v{a.versions.filter(v => !v.needs || flags[v.needs]).length}</span>}
                    {pinned[a.id] && <span className="axb-pin"><AI n="a-pin" s={11} /></span>}
                    <span className="axb-go"><AI n="a-ext" s={12} /></span>
                  </button>
                ))}
              </div>
            ))}
          </section>
        ))}
        {page.length < shown.length && <div ref={sent} className="axb-more"><span></span>Loading {Math.min(60, shown.length - page.length)} more…</div>}
        {!shown.length && <div className="axb-empty"><b>No artifacts match “{q}”</b><span>Try a tool name like <code>billing.queue.list</code> or a load ID.</span></div>}
      </div>
    </div>
  );
}

Object.assign(window, { ArtBrowser });
