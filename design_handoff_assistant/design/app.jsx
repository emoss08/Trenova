const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "dark": true,
  "startOpen": false,
  "layout": "compact"
}/*EDITMODE-END*/;

const now = () => new Date().toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
function scriptFor(text, agent) {
  const t = text.toLowerCase();
  if (t.includes("stuck") || t.includes("blocked")) return "stuck";
  if (t.includes("payment")) return "payments";
  if (t.includes("ebitda") || agent === "web") return "ebitda";
  return "generic";
}

function History({ threads, activeId, onOpen, onNew, autoFocus, pendingThread }) {
  const [q, setQ] = React.useState("");
  const mine = threads.filter(t => !RECENTS.find(r => r.id === t.id)).map(t => ({ ...t, g: "Today", ago: "now" }));
  const items = [...mine, ...RECENTS].filter(r => r.title.toLowerCase().includes(q.toLowerCase()));
  const groups = ["Today", "Yesterday", "This week"].map(g => [g, items.filter(i => i.g === g)]).filter(([, l]) => l.length);
  return (
    <div className="hl">
      <div className="hs">
        <label className="hsi"><Ic n="search" s={13}></Ic><input autoFocus={autoFocus} value={q} onChange={e => setQ(e.target.value)} placeholder="Search conversations"></input></label>
        <button className="ib" title="New conversation" onClick={onNew}><Ic n="plus" s={15}></Ic></button>
      </div>
      <div className="hlist">
        {groups.map(([g, l]) => <React.Fragment key={g}>
          <div className="hg">{g}</div>
          {l.map(r => <button key={r.id} className={"hli" + (r.id === activeId ? " on" : "")} onClick={() => onOpen(r)}>
            <AgentTile a={AGENTS[r.agent]} s={18}></AgentTile><span><b>{r.title}</b><em>{AGENTS[r.agent].name} · {r.ago === "now" ? "just now" : r.ago + " ago"}</em></span>
            {pendingThread === r.id && <i className="wd" title="Waiting on your approval"></i>}
          </button>)}
        </React.Fragment>)}
        {!groups.length && <div className="hempty">No conversations match “{q}”</div>}
      </div>
    </div>
  );
}

function Thread({ thread, events }) {
  const sc = React.useRef(), col = React.useRef(), stick = React.useRef(true);
  const toBottom = () => requestAnimationFrame(() => { if (stick.current && sc.current) sc.current.scrollTop = sc.current.scrollHeight; });
  React.useEffect(() => {
    const ro = new ResizeObserver(toBottom); ro.observe(col.current);
    const mo = new MutationObserver(toBottom); mo.observe(col.current, { childList: true, subtree: true, characterData: true });
    return () => { ro.disconnect(); mo.disconnect(); };
  }, []);
  React.useEffect(() => { stick.current = true; toBottom(); }, [thread.msgs.length]);
  return (
    <div className="scroll" ref={sc} onWheel={e => { if (e.deltaY < 0) stick.current = false; }} onScroll={e => { const el = e.currentTarget; if (el.scrollHeight - el.scrollTop - el.clientHeight < 40) stick.current = true; }}>
      <div className="col" ref={col}>
        {thread.msgs.map((m, i) => m.role === "user" ? <div key={m.id} className={"q" + (m.text.length > 90 ? " long" : m.text.length > 48 ? " mid" : "")}>{m.text}</div>
          : m.role === "cmp" ? <div key={m.id} className="cmpd"><CompactMark it={m}></CompactMark></div>
          : m.role === "event" ? <div key={m.id} className="evl"><span className="tk"><Ic n="check" s={11} w={2.6}></Ic></span><span><b>Approved</b> · {m.text}</span></div>
          : <Reply key={m.id} m={m} last={i === thread.msgs.length - 1}></Reply>)}
      </div>
    </div>
  );
}

function Home({ layout, openRecent, showAll }) {
  const full = layout === "full";
  return (
    <div className="scroll"><div className="col">
      <div className="hm-h"><h2>Ask about anything you can see</h2><p>Reads what's on your screen. Changes wait for your approval.</p></div>
      {!full && <>
        <div className="rc-h"><span>Recent</span><button onClick={showAll}>All conversations</button></div>
        {RECENTS.slice(0, 3).map((r, i) => <button key={r.id} className="rc-i" style={{ animationDelay: 60 + i * 40 + "ms" }} onClick={() => openRecent(r)}>
          <AgentTile a={AGENTS[r.agent]} s={18}></AgentTile><span>{r.title}</span><em>{r.ago}</em></button>)}
      </>}
    </div></div>
  );
}

function Beacon({ corner, setCorner, hidden, busy, status, waiting, ready, agent, onOpen }) {
  const [d, setD] = React.useState(null);
  const st = React.useRef(null);
  const down = e => { st.current = { x: e.clientX, y: e.clientY, moved: false }; e.currentTarget.setPointerCapture(e.pointerId); };
  const move = e => { const s = st.current; if (!s) return; const dx = e.clientX - s.x, dy = e.clientY - s.y; if (!s.moved && Math.hypot(dx, dy) < 5) return; s.moved = true; setD({ dx, dy }); };
  const up = e => {
    const s = st.current; st.current = null; setD(null); if (!s) return;
    if (!s.moved) return onOpen();
    const r = e.currentTarget.parentElement.getBoundingClientRect();
    setCorner(e.clientX - r.left < r.width / 2 ? "bl" : "br");
  };
  const state = busy ? "busy" : waiting ? "wait" : ready ? "ready" : "";
  return (
    <div role="button" tabIndex={0} aria-label="Open assistant" className={`bc c-${corner} ${state}` + (hidden ? " hide" : "") + (d ? " drag" : "")}
      style={d ? { transform: `translate(${d.dx}px,${d.dy}px)` } : null} onPointerDown={down} onPointerMove={move} onPointerUp={up}
      onKeyDown={e => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); onOpen(); } }}>
      {corner === "br" && label()}
      <span className="bc-orb"></span>
      {corner === "bl" && label()}
    </div>
  );
  function label() {
    return <span className="bc-l"><div><span className="bc-in" style={corner === "bl" ? { padding: "0 4px 0 2px" } : null}>
      {d ? <span>Drop on either side</span>
        : busy ? <><span className="shim">{status ? status.t : "Working"}</span><em>{status ? status.el.toFixed(1) + "s" : ""}</em></>
        : waiting ? <><span><b>1 change</b> needs your approval</span><span className="apv">Review</span></>
        : ready ? <span><b>{AGENTS[agent].name}</b> replied</span>
        : <><span>Ask {AGENTS[agent].name}</span><span className="kb"><span className="kbd">⌘</span><span className="kbd">J</span></span></>}
    </span></div></span>;
  }
}

function App() {
  const [t, setTweak] = useTweaks(TWEAK_DEFAULTS);
  const [open, setOpen] = React.useState(t.startOpen);
  const [layout, setLayout] = React.useState(t.layout);
  const [view, setView] = React.useState("home");
  const [corner, setCorner] = React.useState("br");
  const [agent, setAgent] = React.useState("billing");
  const [threads, setThreads] = React.useState([]);
  const [activeId, setActiveId] = React.useState(null);
  const [busy, setBusy] = React.useState(false);
  const [status, setStatus] = React.useState(null);
  const [ready, setReady] = React.useState(false);
  const [hot, setHot] = React.useState(null);
  const [peek, setPeek] = React.useState(false);
  const [pageSel, setPageSel] = React.useState("billing");
  const [rows, setRows] = React.useState(QROWS);
  const [flash, setFlash] = React.useState(null);
  const [pending, setPending] = React.useState(null); // {d, threadId}
  const [dphase, setDphase] = React.useState("ask");
  const [undoN, setUndoN] = React.useState(5);
  const [resolved, setResolved] = React.useState({});
  const [sbOpen, setSbOpen] = React.useState(true);
  const [ctxs, setCtxs] = React.useState({});
  const [ctxAuto, setCtxAuto] = React.useState(true);
  const [compacting, setCompacting] = React.useState(null);
  const cmpTimer = React.useRef(null);
  const searchRef = React.useRef(null);
  const openRef = React.useRef(open); openRef.current = open;
  const layoutRef = React.useRef(layout); layoutRef.current = layout;
  const uid = React.useRef(0); const nid = () => "m" + (++uid.current);

  React.useEffect(() => { setLayout(t.layout); }, [t.layout]);
  const thread = threads.find(x => x.id === activeId);
  const patch = (id, fn) => setThreads(ts => ts.map(x => x.id === id ? fn(x) : x));

  const send = text => {
    if (busy) return;
    const tid = thread && view === "thread" ? thread.id : "t" + nid();
    const ai = { id: nid(), tid, role: "ai", agent, script: scriptFor(text, agent), live: true, time: now() };
    const u = { id: nid(), role: "user", text };
    if (thread && view === "thread") patch(thread.id, x => ({ ...x, msgs: [...x.msgs, u, ai] }));
    else { setThreads(ts => [{ id: tid, title: text, agent, msgs: [u, ai] }, ...ts]); setActiveId(tid); }
    setView("thread"); setBusy(true); setStatus(null); setPeek(false);
  };
  const openRecent = r => {
    setView("thread"); setActiveId(r.id);
    if (threads.find(x => x.id === r.id)) return;
    setThreads(ts => [...ts, { id: r.id, title: r.title, agent: r.agent, msgs: [{ id: nid(), role: "user", text: r.title }, { id: nid(), role: "ai", agent: r.agent, script: r.script, live: false, time: r.ago + " ago" }] }]);
    setAgent(r.agent);
  };
  const stop = () => setThreads(ts => ts.map(x => ({ ...x, msgs: x.msgs.map(m => m.live ? { ...m, stopped: true } : m) })));
  const finish = (id, s, stopped, tid) => {
    setBusy(false); setStatus(null);
    setThreads(ts => ts.map(x => x.msgs.some(m => m.id === id) ? { ...x, msgs: x.msgs.map(m => m.id === id ? { ...m, live: false } : m) } : x));
    if (!stopped && s.decision && !resolved[s.decision.id]) { setPending({ d: s.decision, threadId: tid }); setDphase("ask"); }
    if (!openRef.current) setReady(true);
  };
  const approve = () => {
    if (!pending || dphase !== "ask") return;
    setDphase("ok");
    setTimeout(() => { setDphase("undo"); setUndoN(5); }, 1000);
  };
  const commit = React.useCallback(() => {
    if (!pending) return;
    const { d, threadId } = pending;
    setResolved(r => ({ ...r, [d.id]: true }));
    setRows(rs => rs.map(r => r.id === "40102" ? { ...r, st: "Ready", why: "", amt: "1,240.00" } : r));
    setFlash("40102"); setTimeout(() => setFlash(null), 1900);
    patch(threadId, x => ({ ...x, msgs: [...x.msgs, { id: nid(), role: "event", text: d.event }] }));
    setPending(null); setDphase("ask");
  }, [pending]);
  React.useEffect(() => {
    if (dphase !== "undo") return;
    if (undoN <= 0) { commit(); return; }
    const h = setTimeout(() => setUndoN(n => n - 1), 1000); return () => clearTimeout(h);
  }, [dphase, undoN]);
  const dismiss = () => { setResolved(r => ({ ...r, [pending.d.id]: true })); setPending(null); };

  React.useEffect(() => {
    const k = e => {
      const mod = e.metaKey || e.ctrlKey;
      if (mod && e.key.toLowerCase() === "j") { e.preventDefault(); setOpen(o => !o); setReady(false); }
      else if (mod && e.key === "Enter" && openRef.current && pending && dphase === "ask") { e.preventDefault(); approve(); }
      else if (mod && e.key.toLowerCase() === "k" && openRef.current && layoutRef.current === "full") { e.preventDefault(); setSbOpen(true); searchRef.current && searchRef.current(); }
      else if (mod && e.key === "\\" && openRef.current && layoutRef.current === "full") { e.preventDefault(); setSbOpen(o => !o); }
      else if (e.key === "Escape" && openRef.current) setOpen(false);
    };
    window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k);
  }, [pending, dphase]);

  const full = layout === "full";
  const v = full && view === "history" ? (activeId ? "thread" : "home") : view;
  const tall = v !== "home" || layout !== "compact";
  const title = v === "thread" && thread ? thread.title : v === "history" ? "Conversations" : null;
  const showDock = v !== "history";
  const dockDecision = pending && thread && pending.threadId === thread.id && v === "thread";
  const actx = { setHot, send, finish, status: setStatus, pending };
  const newChat = () => { setView("home"); setActiveId(null); };
  const ctxFor = id => ctxs[id] || CTX0;
  const ctx = thread ? { parts: ctxFor(thread.id), auto: ctxAuto, setAuto: setCtxAuto } : null;
  const compact = auto => {
    if (!thread || busy || compacting) return;
    const tid = thread.id, before = ctxTotal(ctxFor(tid)), after = { sys: 6200, conv: 9400, tools: 2600, files: 0 };
    const id = nid(), turns = thread.msgs.filter(m => m.role === "user").length * 2 + 14;
    setCompacting({ before, after: ctxTotal(after), auto, tid, id });
    patch(tid, x => ({ ...x, msgs: [...x.msgs, { id, role: "cmp", live: true, auto, turns, before, after: ctxTotal(after) }] }));
    cmpTimer.current = setTimeout(() => {
      patch(tid, x => ({ ...x, msgs: x.msgs.map(m => m.id === id ? { ...m, live: false } : m) }));
      setCtxs(c => ({ ...c, [tid]: after })); setCompacting(null);
    }, 2700);
  };
  const cancelCompact = () => {
    if (!compacting) return; clearTimeout(cmpTimer.current);
    const { tid, id } = compacting;
    patch(tid, x => ({ ...x, msgs: x.msgs.filter(m => m.id !== id) })); setCompacting(null);
  };
  const openPanel = () => { setOpen(true); setReady(false); if (pending && !full) { setActiveId(pending.threadId); setView("thread"); } };

  return (
    <AsCtx.Provider value={actx}>
      <div className={"asx" + (t.dark ? " dk" : "")}>
        <BgApp rows={rows} hot={hot} peek={peek} flash={flash} docked={open && layout === "side"}></BgApp>
        {open && full && <div className="as-scrim" onClick={() => setOpen(false)}></div>}
        {open && full && <div className={"pn full" + (sbOpen ? "" : " sbx")} key="full" role="dialog" aria-label="Assistant">
          <Sidebar threads={threads} activeId={activeId} onOpen={openRecent} onNew={newChat} pendingThread={pending && pending.threadId} searchRef={searchRef}></Sidebar>
          <div className="pn-m">
            <div className="fh">
              <button className={"ib" + (sbOpen ? "" : " on")} title={(sbOpen ? "Hide" : "Show") + " conversations · ⌘\\"} onClick={() => setSbOpen(!sbOpen)}><Ic n="rail" s={15}></Ic></button>
              {!sbOpen && <button className="ib" title="New conversation" onClick={newChat}><Ic n="plus" s={15}></Ic></button>}
              <div className="fh-t">{v === "thread" && thread && <b key={thread.id}>{thread.title}</b>}</div>
              <button className="ib" title="Shrink" onClick={() => setLayout("compact")}><Ic n="collapse" s={14}></Ic></button>
              <button className="ib" title="Close · Esc" onClick={() => setOpen(false)}><Ic n="x" s={15}></Ic></button>
            </div>
            {v === "thread" && thread ? <>
              <Thread key={thread.id} thread={thread}></Thread>
              <div className="dock">
                {dockDecision && <DecisionDock d={pending.d} phase={dphase} n={undoN} onApprove={approve} onDismiss={dismiss} onUndo={() => setDphase("ask")} onNow={commit}></DecisionDock>}
                <Composer key={"f" + thread.id} agent={agent} setAgent={setAgent} busy={busy} status={status} onSend={send} onStop={stop} pageSel={pageSel} setPageSel={setPageSel} ctx={ctx} onCompact={compact} compacting={compacting && thread && compacting.tid === thread.id ? compacting : null} onCancelCompact={cancelCompact}></Composer>
              </div>
            </> : <div className="fhome"><div className="fhome-in">
              <div className="fh-date"><span className="pulse"></span>{new Date().toLocaleDateString([], { weekday: "long", month: "long", day: "numeric" })}</div>
              <h2>Ask about anything you can see</h2>
              <p>Reads what's on your screen. Changes wait for your approval.</p>
              <Composer key="fhome" agent={agent} setAgent={setAgent} busy={busy} status={status} onSend={send} onStop={stop} home={true} pageSel={pageSel} setPageSel={setPageSel} ctx={ctx} onCompact={compact} compacting={compacting && thread && compacting.tid === thread.id ? compacting : null} onCancelCompact={cancelCompact}></Composer>
            </div></div>}
          </div>
        </div>}
        {open && !full && <div className={`pn ${layout} c-${corner}` + (tall ? " tall" : "")} key={layout} role="dialog" aria-label="Assistant">
          <div className="pn-h">
            <div className="pn-t">
              {v !== "home" && !full && <button className="ib bk" title="Back" onClick={() => setView(v === "history" && activeId ? "thread" : "home")}><Ic n="chevL" s={15}></Ic></button>}
              {title ? <b key={title}>{title}</b> : <b key="as">Assistant</b>}
            </div>
            {!full && <button className={"ib" + (v === "history" ? " on" : "")} title="Conversations" onClick={() => setView(v === "history" ? (activeId ? "thread" : "home") : "history")}><Ic n="history" s={15}></Ic></button>}
            <button className="ib" title="New conversation" onClick={() => { setView("home"); setActiveId(null); }}><Ic n="plus" s={15}></Ic></button>
            {!full && <button className={"ib" + (layout === "side" ? " on" : "")} title={layout === "side" ? "Float" : "Dock to side"} onClick={() => setLayout(layout === "side" ? "compact" : "side")}><Ic n="panel" s={15}></Ic></button>}
            <button className="ib" title={full ? "Shrink" : "Full screen"} onClick={() => setLayout(full ? "compact" : "full")}><Ic n={full ? "collapse" : "expand"} s={14}></Ic></button>
            <button className="ib" title="Close · Esc" onClick={() => setOpen(false)}><Ic n="x" s={15}></Ic></button>
          </div>
          <div className="pn-b">
            {full && <History threads={threads} activeId={activeId} onOpen={openRecent} onNew={() => { setView("home"); setActiveId(null); }} pendingThread={pending && pending.threadId}></History>}
            <div className="pn-m">
              {v === "history" ? <History threads={threads} activeId={activeId} onOpen={openRecent} onNew={() => { setView("home"); setActiveId(null); }} autoFocus={true} pendingThread={pending && pending.threadId}></History>
                : v === "thread" && thread ? <Thread key={thread.id} thread={thread}></Thread>
                : <Home layout={layout} openRecent={openRecent} showAll={() => setView("history")}></Home>}
              {showDock && <div className="dock">
                {dockDecision && <DecisionDock d={pending.d} phase={dphase} n={undoN} onApprove={approve} onDismiss={dismiss} onUndo={() => { setDphase("ask"); }} onNow={commit}></DecisionDock>}
                <Composer key={v + (thread ? thread.id : "")} agent={agent} setAgent={setAgent} busy={busy} status={status} onSend={send} onStop={stop} home={v === "home"} pageSel={pageSel} setPageSel={setPageSel} ctx={ctx} onCompact={compact} compacting={compacting && thread && compacting.tid === thread.id ? compacting : null} onCancelCompact={cancelCompact}></Composer>
              </div>}
            </div>
          </div>
        </div>}
        <Beacon corner={corner} setCorner={setCorner} hidden={open} busy={busy} status={status} waiting={!!pending && dphase === "ask"} ready={ready} agent={agent} onOpen={openPanel}></Beacon>
        <TweaksPanel>
          <TweakSection label="Assistant"></TweakSection>
          <TweakRadio label="Layout" value={layout} options={["compact", "side", "full"]} onChange={x => { setLayout(x); setTweak("layout", x); setOpen(true); }}></TweakRadio>
          <TweakToggle label="Dark mode" value={t.dark} onChange={x => setTweak("dark", x)}></TweakToggle>
          <TweakButton label={open ? "Close assistant" : "Open assistant"} onClick={() => setOpen(!open)}></TweakButton>
        </TweaksPanel>
      </div>
    </AsCtx.Provider>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<App></App>);
