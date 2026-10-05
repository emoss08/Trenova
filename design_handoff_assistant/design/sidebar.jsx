function Sidebar({ threads, activeId, onOpen, onNew, pendingThread, searchRef }) {
  const [q, setQ] = React.useState("");
  const [searching, setSearching] = React.useState(false);
  const list = React.useRef();
  const [knob, setKnob] = React.useState(null);
  const mine = threads.filter(t => !RECENTS.find(r => r.id === t.id)).map(t => ({ ...t, g: "Today", ago: "now" }));
  const all = [...mine, ...RECENTS].filter(r => r.title.toLowerCase().includes(q.toLowerCase()));
  const waiting = all.filter(r => r.id === pendingThread);
  const rest = all.filter(r => r.id !== pendingThread);
  const groups = [["Waiting on you", waiting], ...["Today", "Yesterday", "This week"].map(g => [g, rest.filter(i => i.g === g)])].filter(([, l]) => l.length);
  React.useLayoutEffect(() => {
    const el = list.current && list.current.querySelector(".sb-i.on");
    setKnob(el ? { y: el.offsetTop, h: el.offsetHeight } : null);
  }, [activeId, q, threads.length, pendingThread]);
  React.useEffect(() => { if (searchRef) searchRef.current = () => setSearching(true); }, []);
  const close = () => { setQ(""); setSearching(false); };
  return (
    <div className="sbw"><aside className="sb">
      <div className="sb-top">
        <span className="sb-mk"><Ic n="diamond" s={13} w={1.8}></Ic></span><b>Assistant</b><span className="sp"></span>
        <button className="ib sb-new" title="New conversation" onClick={onNew}><Ic n="plus" s={15}></Ic></button>
      </div>
      <div className="sb-s">
        {searching
          ? <label className="sb-in"><Ic n="search" s={13}></Ic><input autoFocus value={q} onChange={e => setQ(e.target.value)} onKeyDown={e => e.key === "Escape" && (e.stopPropagation(), close())} onBlur={() => !q && close()} placeholder="Search conversations"></input>{q && <button onMouseDown={e => e.preventDefault()} onClick={close}><Ic n="x" s={12}></Ic></button>}</label>
          : <button className="sb-sb" onClick={() => setSearching(true)}><Ic n="search" s={13}></Ic><span>Search</span><span className="kbd">⌘K</span></button>}
      </div>
      <div className="sb-list" ref={list}>
        {knob && <div className="sb-knob" style={{ transform: `translateY(${knob.y}px)`, height: knob.h }}></div>}
        {groups.map(([g, l]) => <React.Fragment key={g}>
          <div className={"sb-gh" + (g === "Waiting on you" ? " w" : "")}>{g}</div>
          {l.map((r, i) => <button key={r.id} className={"sb-i" + (r.id === activeId ? " on" : "")} style={{ animationDelay: i * 25 + "ms" }} onClick={() => onOpen(r)} title={r.title}>
            <span className="sb-dot" title={AGENTS[r.agent].name}><i style={{ "--ah": AGENTS[r.agent].h }}></i></span>
            <span className="tt">{r.title}</span>
            {r.id === pendingThread ? <i className="wd"></i> : <em>{r.ago === "now" ? "now" : r.ago}</em>}
          </button>)}
        </React.Fragment>)}
        {!groups.length && <div className="hempty">Nothing matches “{q}”</div>}
      </div>
      <div className="sb-f"><Ic n="lock" s={11}></Ic>Read-only until you approve</div>
    </aside></div>
  );
}
Object.assign(window, { Sidebar });
