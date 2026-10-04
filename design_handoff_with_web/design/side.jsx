function Rail({ view, noChats, onNew, onThread, working, pending, onSearch, onSettings, onWatch, onDecisions, onToday, onMemory }) {
  const listRef = React.useRef(null);
  const [knob, setKnob] = React.useState(null);
  const [decN, setDecN] = React.useState(window.DEC_WAITING != null ? window.DEC_WAITING : (window.DEC_ITEMS0 || []).length);
  React.useEffect(() => { const h = () => setDecN(window.DEC_WAITING); window.addEventListener("desk:dec-count", h); return () => window.removeEventListener("desk:dec-count", h); }, []);
  const active = view === "thread" || view === "agent" ? "c:active" : view === "watch" ? "n:watch" : view === "decisions" ? "n:dec" : view === "memory" ? "n:mem" : view === "home" ? "n:today" : "";
  const [convos, setConvos] = React.useState(() => noChats ? [] : CONVOS.flatMap(g => g.items.map(c => ({ ...c, g: g.g, pin: g.g === "Pinned", home: g.g === "Pinned" ? "Today" : g.g }))));
  const [confirm, setConfirm] = React.useState(null);
  const [leaving, setLeaving] = React.useState(null);
  React.useEffect(() => { if (view === "thread" && !convos.some(c => c.on)) { const c = CONVOS[0].items[0]; setConvos(cs => [{ ...c, g: "Today", pin: false, home: "Today" }, ...cs]); } }, [view]);
  const [editing, setEditing] = React.useState(null);
  const rename = (t, v) => { v = v.trim(); setEditing(null); if (!v || v === t) return; setConvos(cs => cs.map(c => c.t === t ? { ...c, t: v } : c)); if (convos.find(c => c.t === t && c.on)) { window.DESK_TITLE = v; window.dispatchEvent(new Event("desk:title")); } };
  const togglePin = t => setConvos(cs => cs.map(c => c.t === t ? { ...c, pin: !c.pin } : c));
  const del = t => { if (confirm !== t) { setConfirm(t); return; } setConfirm(null); setLeaving(t); setTimeout(() => { setConvos(cs => cs.filter(c => c.t !== t)); setLeaving(null); }, 260); };
  const groups = [["Pinned", convos.filter(c => c.pin)], ...["Today", "Yesterday", "Previous 7 days"].map(g => [g, convos.filter(c => !c.pin && c.home === g)])].filter(([, xs]) => xs.length);
  React.useLayoutEffect(() => {
    const root = listRef.current; if (!root) return;
    const el = root.querySelector('[data-k="' + active + '"]');
    if (!el) return setKnob(null);
    const r = root.getBoundingClientRect(), e = el.getBoundingClientRect();
    setKnob({ y: e.top - r.top + root.scrollTop, h: e.height });
  }, [active, convos]);
  let n = -1;
  return (
    <aside className="sb">
      <div className="sb-top">
        <img className="sb-logo" src="logo.png" alt="Trenova" />
        <span style={{ flex: 1 }}></span>
        <button className="ib" title="Search  ⌘K" onClick={onSearch}><Ic n="search" s={15} /></button>
        <button className="ib sb-new" title="New conversation  ⌘N" onClick={onNew}><Ic n="plus" s={15} w={2} /></button>
      </div>
      <div className="sb-list" ref={listRef}>
        {knob && <span className="sb-knob" style={{ transform: `translateY(${knob.y}px)`, height: knob.h }}></span>}
        <button data-k="n:today" className={"sb-i" + (active === "n:today" ? " on" : "")} onClick={onNew}><Ic n="home" s={14} /><span>Today</span></button>
        <button data-k="n:watch" className={"sb-i" + (active === "n:watch" ? " on" : "")} onClick={onWatch}><Ic n="radar" s={14} /><span>Watchtower</span><em className="sb-ct">{(window.WT_ITEMS0 || []).length}</em></button>
        <button data-k="n:dec" className={"sb-i" + (active === "n:dec" ? " on" : "")} onClick={onDecisions}><Ic n="inbox" s={14} /><span>Decisions</span><em className={"sb-ct" + (pending ? " w" : "")}>{decN + pending}</em></button>
        <button data-k="n:mem" className={"sb-i" + (active === "n:mem" ? " on" : "")} onClick={onMemory}><Ic n="memory" s={14} /><span>Memory</span></button>
        {groups.map(([gname, xs]) => (
          <React.Fragment key={gname}>
            <div className="sb-gh">{gname}</div>
            {xs.map(c => {
              n += 1;
              const k = c.on ? "c:active" : "c:" + c.t;
              const st = c.on ? (working ? "work" : pending ? "wait" : null) : c.s;
              const STL = { work: "Working", wait: "Needs your approval", error: "Last reply failed", new: "New reply" };
              return (
                <div role="button" tabIndex={0} key={c.t} data-k={k} className={"sb-i sb-c" + (active === k ? " on" : "") + (leaving === c.t ? " out" : "") + (confirm === c.t ? " cf" : "") + (editing === c.t ? " ed" : "")} onClick={() => editing !== c.t && c.on && onThread()} onDoubleClick={() => { setConfirm(null); setEditing(c.t); }} onMouseLeave={() => confirm === c.t && setConfirm(null)} title={c.a + (st ? " · " + STL[st] : "") + " · " + c.w}>
                  <span className={"sb-dot" + (c.k === "dispatch" ? " v" : "") + (st ? " s-" + st : "")}>{st === "wait" ? <Ic n="info" s={13} w={2} /> : st === "error" ? <Ic n="alert" s={13} w={2} /> : <i></i>}</span>
                  {editing === c.t ? <input className="sb-in" autoFocus defaultValue={c.t} onFocus={e => e.target.select()} onClick={e => e.stopPropagation()} onBlur={e => rename(c.t, e.target.value)} onKeyDown={e => { e.stopPropagation(); if (e.key === "Enter") rename(c.t, e.target.value); if (e.key === "Escape") setEditing(null); }} /> : <span className="sb-t">{c.t}</span>}
                  <span className="sb-acts" onClick={e => e.stopPropagation()}>
                    {confirm === c.t ? <button className="sb-a del cf" onClick={() => del(c.t)}>Delete</button> : <>
                      <button className={"sb-a" + (c.pin ? " pinned" : "")} data-tip={c.pin ? "Unpin" : "Pin"} onClick={() => togglePin(c.t)}><Ic n="pin" s={13} /></button>
                      <button className="sb-a del" data-tip="Delete" onClick={() => del(c.t)}><Ic n="trash" s={13} /></button>
                    </>}
                  </span>
                </div>
              );
            })}
          </React.Fragment>
        ))}
        {!convos.length && <><div className="sb-gh">Conversations</div><div className="sb-empty">Your chats will show up here. Press <span className="kbd">⌘N</span> to start one.</div></>}
      </div>
      <div className="sb-me"><span className="me">AL</span><b>Avery Lane</b><span style={{ flex: 1 }}></span><button className="ib" title="Settings" onClick={onSettings}><Ic n="gear" s={15} /></button><button className="ib sb-back" title="Back to Trenova"><span className="sb-bk-c"><Ic n="chevL" s={14} /></span><img className="sb-bk-l" src="logo.png" alt="" /></button></div>
    </aside>
  );
}

function TopBar({ view, open, setOpen, pending, newArt, nArts, onReplay, onAgent, onHandoff }) {
  const [title, setTitle] = React.useState(window.DESK_TITLE || "Check and see what shipments are eligible to be transferred");
  React.useEffect(() => { const h = () => setTitle(window.DESK_TITLE); window.addEventListener("desk:title", h); return () => window.removeEventListener("desk:title", h); }, []);
  return (
    <header className="top">
      <div className="ttl">
        {view === "decisions" ? <b>Decisions</b> : view === "watch" ? <b>Watchtower</b> : view === "home" ? <b>Today</b> : view === "memory" ? <b>Memory</b> : view === "agent" ? <><AgentMark s={10} /><b>Billing Specialist</b><span className="sl">/</span><span className="ttl-a">What it can do</span></> : <><AgentMark s={10} /><button className="ttl-a ttl-ag" title="What this agent can do" onClick={onAgent}>Billing Specialist</button><span className="sl">/</span><b>{title}</b></>}
      </div>
      <span className="sp"></span>
      {view === "thread" && <>
        <button className="ib" title="Replay" onClick={onReplay}><Ic n="replay" s={14} /></button>
        <HandoffMenu onPick={onHandoff} />
        <button className="ib" title="What this agent can do" onClick={onAgent}><Ic n="shield" s={14} /></button>
        <button className={"wsb" + (open ? " on" : "")} onClick={() => setOpen(!open)}>
          <Ic n="panel" s={15} />Workspace<span className="n">{nArts}</span>
          {!open && (newArt || pending) ? <span className="dot" style={pending ? { background: "var(--warn)" } : null}></span> : null}
        </button>
      </>}
    </header>
  );
}

function Dictate({ setValue }) {
  const [rec, setRec] = React.useState(false);
  const [t, setT] = React.useState(0);
  React.useEffect(() => {
    if (!rec) return;
    const st = performance.now();
    const iv = setInterval(() => setT((performance.now() - st) / 1000), 100);
    return () => clearInterval(iv);
  }, [rec]);
  const stop = () => { setRec(false); setValue(v => (v ? v + " " : "") + "And send me the totals by customer when it's done."); };
  if (!rec) return <button className="ib" title="Dictate" onClick={() => { setT(0); setRec(true); }}><Ic n="mic" s={15} /></button>;
  return (
    <button className="dict" onClick={stop} title="Stop dictating">
      <span className="wave">{[0, 1, 2, 3, 4].map(k => <i key={k} style={{ animationDelay: k * 110 + "ms" }}></i>)}</span>
      <span className="mono">0:{String(Math.floor(t)).padStart(2, "0")}</span>
      <span className="dict-x"></span>
    </button>
  );
}

function useTypewriter(list, active) {
  const [st, setSt] = React.useState({ i: 0, n: 0, del: false });
  React.useEffect(() => {
    if (!active) return;
    const full = list[st.i];
    let ms = st.del ? 18 : 38;
    if (!st.del && st.n === full.length) ms = 2200;
    if (st.del && st.n === 0) ms = 300;
    const h = setTimeout(() => setSt(p => {
      const f = list[p.i];
      if (!p.del && p.n < f.length) return { ...p, n: p.n + 1 };
      if (!p.del) return { ...p, del: true };
      if (p.n > 0) return { ...p, n: p.n - 1 };
      return { i: (p.i + 1) % list.length, n: 0, del: false };
    }), ms);
    return () => clearTimeout(h);
  }, [st, active, list]);
  return { text: list[st.i].slice(0, st.n), full: list[st.i] };
}

function Composer({ value, setValue, onSend, busy, status, home, placeholder, presets, sendMod, phase, off, cool, up, drop, pageKey, pageSel, setPageSel, cfg = {}, ctx, onCompact, compacting, onCancelCompact }) {
  const [menu, setMenu] = React.useState(false);
  React.useEffect(() => { const h = e => setMenu(e.detail || "ok"); window.addEventListener("desk:capture", h); return () => window.removeEventListener("desk:capture", h); }, []);
  const [agent, setAgent] = React.useState(AGENTS.find(a => a.id === cfg.agent) || AGENTS[0]);
  React.useEffect(() => { const a = AGENTS.find(x => x.id === cfg.agent); if (a) setAgent(a); }, [cfg.agent]);
  const slashOn = cfg.slash !== "off", mentionOn = cfg.mentions !== "off";
  const taRef = React.useRef(null);
  const mn = useMentions(value, setValue, taRef);
  const sl = useSlash(slashOn ? value : "", setValue, taRef, txt => onSend(txt));
  const uploading = up && up.atts.some(a => a.status === "uploading" || a.status === "scanning");
  const hasReady = up && up.atts.some(a => a.status === "ready");
  const canSend = (value.trim() || hasReady) && !uploading;
  const [model, setModel] = React.useState("");
  const empty = !value;
  const tw = useTypewriter(presets || [""], !!presets && empty);
  const onKey = e => {
    if (mn.onKey(e)) return;
    if (sl.onKey(e)) return;
    if (presets && empty && e.key === "Tab") { e.preventDefault(); setValue(tw.full); return; }
    if (presets && (e.metaKey || e.ctrlKey) && /^[1-9]$/.test(e.key) && presets[+e.key - 1]) { e.preventDefault(); onSend(presets[+e.key - 1]); return; }
    const mod = e.metaKey || e.ctrlKey;
    if (e.key === "Enter" && !e.shiftKey && (sendMod ? mod : !mod)) { e.preventDefault(); if (canSend && !busy) onSend(); }
  };
  return (
    <div className={"cmp" + (busy ? " busy" : "") + (home ? " hm" : "") + (phase ? " ec r-" + phase : "") + (off ? " ec off" : "") + (compacting ? " cmpg" : "") + (drop && drop.on && !off ? " drop-on" + (drop.hot ? " drop-hot" : "") : "")}>
      {drop && drop.on && !off && <span className={"drop-tag" + (up && up.atts.length >= MAX_ATTACHMENTS ? " full" : "")}>{up && up.atts.length >= MAX_ATTACHMENTS ? "Message full" : drop.hot ? "Release to attach" : "Drop here"}</span>}
      <span className="cmp-ring" aria-hidden="true"></span>
      {busy && status && (
        <div className="cmp-st ec-st" key={status.t}>
          {status.p === "check" ? <span className="ec-shield"><Ic n="shield" s={13} w={2} /></span> : status.p === "retry" ? <Countdown from={status.secs || 3} /> : status.p === "recall" || status.p === "remember" ? <span className="mem-sti"><MemIc s={13} /></span> : <span className="cmp-st-d"></span>}
          <span className={status.p === "retry" ? "" : "shim"}>{status.t}</span>
          {status.extra && <><span className="ec-sp"></span><span className="ec-att">{status.extra}</span></>}
          {status.p === "retry" && <button className="ec-link">Switch model</button>}
        </div>
      )}
      {compacting && <div className="cmp-st cx-st"><CtxDrain from={compacting.before} to={compacting.after} ms={2600} /><span className="shim cx-stt">{compacting.auto ? "Context is nearly full · compacting" : "Compacting the conversation"}…</span><span className="ec-sp"></span><button className="ec-link" onClick={onCancelCompact}>Cancel</button></div>}
      {!busy && cool > 0 && <div className="cmp-st ec-st"><Countdown from={cool} /><span>You're sending messages quickly · send again in a moment</span></div>}
      {up && !off && <AttachRow up={up} />}
      {drop && drop.on && !off && (() => { const room = Math.max(0, MAX_ATTACHMENTS - (up ? up.atts.length : 0)); const n = Math.min(Math.max(1, drop.count), room);
        return <div className="ar ghosts"><div className="ar-l">{Array.from({ length: n }, (_, i) => <span key={i} className="fp ghost" style={{ animationDelay: i * 50 + "ms" }}><span className="fi" style={{ width: 18, height: 18 }}></span><span className="fp-n"></span></span>)}
          {room === 0 ? <span className="ar-note">{`This message is full · ${MAX_ATTACHMENTS} files max`}</span> : drop.count > room ? <span className="ar-note">{`Only ${room} more will fit · up to ${MAX_ATTACHMENTS} per message`}</span> : null}</div></div>; })()}
      {off ? <div className="ec-offmsg">{off}</div> : <div className="cmp-ta">
        {mn.st && <MentionPicker q={mn.st.q} type={mn.type} setType={mn.setType} results={mn.results} hi={mn.hi} setHi={mn.setHi} onPick={mn.pick} loading={mn.loading} />}
        <SlashMenu sl={sl} />
        {sl.parsed && sl.parsed.started ? <SlashMirror sl={sl} /> : <MentionMirror value={value} />}
        <textarea ref={taRef} className={sl.parsed && sl.parsed.started ? "sl-on" : ""} onSelect={e => mentionOn && mn.detect(e.target.value, e.target.selectionStart)} rows={2} value={value} disabled={!!compacting} placeholder={presets ? "" : home ? placeholder : compacting ? "You can reply once compacting finishes" : "Reply to " + agent.name + "…"} onChange={e => { setValue(e.target.value); mentionOn && mn.detect(e.target.value, e.target.selectionStart); }} onKeyDown={onKey}></textarea>
        {presets && empty && <div className="tw" aria-hidden="true"><span>{tw.text}</span><i className="tw-c"></i>{tw.text === tw.full && <span className="tw-tab"><span className="kbd">Tab</span> to use · <span className="kbd">⌘{presets.indexOf(tw.full) + 1}</span> to ask</span>}</div>}
      </div>}
      <div className="cmp-b">
        <span style={{ position: "relative" }}>
          <button className={"ib" + (menu ? " on" : "")} title="Attach files" onClick={() => up && setMenu(m => !m)} disabled={!!off || !!compacting || (up && up.atts.length >= MAX_ATTACHMENTS)}><Ic n="plus" s={16} /></button>
          {menu && up && <AttachMenu key={typeof menu === "string" ? menu : "m"} up={up} startCapture={typeof menu === "string" ? menu : null} onClose={() => setMenu(false)} />}
        </span>
        <AgentPicker agent={agent} onSelect={setAgent} />
        {setPageSel && <PageChip currentKey={pageKey} sel={pageSel} setSel={setPageSel} onExplain={() => setValue("Explain what's on this page")} />}
        <span style={{ flex: 1 }}></span>
        <ModelPicker value={model} onChange={setModel} hasReplies={!home} />
        {ctx && !home && <ContextMeter ctx={ctx} onCompact={onCompact} compacting={compacting} />}
        {cfg.mic !== "off" && <Dictate setValue={setValue} />}
        {busy ? <button className="send stop" title="Stop"><span className="sq"></span></button>
          : cool > 0 ? <button className="send" disabled><Countdown from={cool} size={18} tone="ink" /></button>
          : <button className="send" disabled={!canSend || !!off || !!compacting} onClick={onSend} title={uploading ? "Waiting for the files to finish uploading" : "Send"}>{uploading ? <span className="send-wait"></span> : <Ic n="up" s={15} w={2.2} />}</button>}
      </div>
    </div>
  );
}

function DecisionCard({ pending, confirming, onApprove, onDismiss, onReview }) {
  if (confirming) {
    const d = DECISIONS[confirming];
    return (
      <div className="dcx ok" key={"ok" + d.id}>
        <span className="okr"><svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3.2" strokeLinecap="round" strokeLinejoin="round"><path d="M5 12.5l4.5 4.5L19 7"></path></svg></span>
        <span className="dcx-t"><b>Approved</b><span>{d.id === "d1" ? "Assigning you as biller on 11 items…" : "Posting 15 invoices…"}</span></span>
        <span className="shine"></span>
      </div>
    );
  }
  const d = pending[0] && DECISIONS[pending[0]];
  if (!d) return null;
  const n = d.id === "d1" ? 11 : 15;
  return (
    <div className="dcx" key={d.id}>
      <span className="dcx-i"><Ic n="info" s={15} w={2} /></span>
      <span className="dcx-t">
        <b>{d.title} <span className="dcx-s">on {n} {d.id === "d1" ? "items" : "invoices"}</span></b>
        <span className="dcx-d">{d.field} <s>{d.from}</s> → <em>{d.to}</em><span className="dcx-m">· {d.hard ? "can't be undone" : "reversible"}</span></span>
      </span>
      <button className="bt sm" onClick={() => onReview(d.id)}>Review</button>
      <button className="bt sm" onClick={() => onDismiss(d.id)}>Not now</button>
      <button className="apv-b" onClick={() => onApprove(d.id)}>Approve<span className="kbd">⌘↵</span></button>
    </div>
  );
}

const STATUS = { ReadyForReview: ["Ready for review", "rv"], Approved: ["Approved", "ap"], Posted: ["Posted", "po"] };

function DataTable({ art, assigned, posted, flash }) {
  const rows = QUEUE.filter(ARTIFACTS[art].filter);
  return (
    <table className="tbl">
      <thead><tr><th>Item</th><th>Customer</th><th>Biller</th><th style={{ textAlign: "right" }}>Amount</th><th>Status</th></tr></thead>
      <tbody>
        {rows.map((r, i) => {
          const biller = r.biller || (assigned ? "Avery Lane" : null);
          const st = posted ? "Posted" : r.status;
          return (
            <tr key={r.id + ":" + flash} style={{ animationDelay: i * 22 + "ms" }}>
              <td className="id">{r.id}</td>
              <td>{r.cust}</td>
              <td className={(biller ? "" : "none") + (flash && !r.biller && assigned && !posted ? " chg" : "")}>{biller || "No biller"}</td>
              <td className="num">{money(r.amt)}</td>
              <td className={flash && posted ? "chg" : ""}><span className={"st " + STATUS[st][1]}><i></i>{STATUS[st][0]}</span></td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

function Panel({ s, set, arts, newest, pending, resolved, activity, assigned, posted, flash, onApprove, onDismiss, onCopy }) {
  const a = ARTIFACTS[s.art] || ARTIFACTS[arts[0]];
  const tabs = ["data", "changes", "activity"];
  return (
    <aside className="sheet">
      <div className="panel">
        <div className="p-h">
          <div>
            <div className="p-k"><span>TABLE</span><span style={{ color: "var(--faint)" }}>·</span><span>{a.sub.split("from ")[1]}</span></div>
            <div className="p-t">{a.title}</div>
          </div>
          <span className="sp"></span>
          <button className="ib" title="Open on its own page" onClick={onCopy}><Ic n="ext" s={14} /></button>
          <button className="ib" title="Close" onClick={() => set({ open: false })}><Ic n="x" s={14} /></button>
        </div>
        <div className="film">
          {arts.map(id => <button key={id} className={"tile" + (a.id === id ? " on" : "") + (id === newest ? " nw" : "")} onClick={() => set({ art: id, tab: "data" })}><Ic n="table" s={12} />{ARTIFACTS[id].title}</button>)}
        </div>
        <div className="segw">
          <div className="seg">
            <span className="kn" style={{ transform: `translateX(${tabs.indexOf(s.tab) * 100}%)` }}></span>
            <button className={s.tab === "data" ? "on" : ""} onClick={() => set({ tab: "data" })}>Data</button>
            <button className={s.tab === "changes" ? "on" : ""} onClick={() => set({ tab: "changes" })}>Changes{pending.length > 0 && <span className="wd"></span>}</button>
            <button className={s.tab === "activity" ? "on" : ""} onClick={() => set({ tab: "activity" })}>Activity</button>
          </div>
        </div>
        <div className="p-b">
          {s.tab === "data" && <DataTable art={a.id} assigned={assigned} posted={posted} flash={flash} />}
          {s.tab === "changes" && <>
            {[...pending.map(id => [id, "pending"]), ...resolved].map(([id, st]) => {
              const d = DECISIONS[id];
              const rows = QUEUE.filter(ARTIFACTS[d.art].filter);
              return (
                <div className={"chg-i st-" + st} key={id}>
                  <h5>{d.title} · {d.scope}<span className={"bdg" + (st === "approved" ? " ok" : st === "dismissed" ? " no" : "")}>{st === "pending" ? "Waiting" : st === "approved" ? "Approved" : "Set aside"}</span></h5>
                  <p>{d.meta.join(" · ")}</p>
                  {st === "pending" && rows.slice(0, 7).map(r => <div className="chg-r" key={r.id}><span className="mono" style={{ fontSize: 11.5, color: "var(--muted)" }}>{r.id}</span><span className="o">{d.id === "d1" ? "No biller" : STATUS[r.status][0]}</span><Ic n="chevR" s={11} /><span className="nv">{d.to}</span></div>)}
                  {st === "pending" && rows.length > 7 && <p style={{ marginTop: 8 }}>+ {rows.length - 7} more</p>}
                  {st === "pending" && <div style={{ display: "flex", gap: 6, marginTop: 12 }}><button className="apv-b" onClick={() => onApprove(id)}>{`Approve ${rows.length} changes`}</button><button className="bt" onClick={() => onDismiss(id)}>Not now</button></div>}
                </div>
              );
            })}
            {!pending.length && !resolved.length && <div className="actv" style={{ color: "var(--subtle)", fontSize: 12.5 }}>Nothing has been proposed in this conversation yet.</div>}
          </>}
          {s.tab === "activity" && <div className="actv">{[...activity].reverse().map((e, i) => <div className={"av " + (e.c || "")} key={activity.length - i}><span className="t">{e.t}</span><span className="d"><i className={e.c || ""}></i></span><span className="x">{e.x}</span></div>)}</div>}
        </div>
        <div className="p-f"><Ic n="link" s={12} /><span>trenova.app/desk/c/8f2k/a/<b>{a.slug}</b></span><button className="bt" onClick={onCopy}>Copy link</button></div>
      </div>
    </aside>
  );
}

const HOME_PRESETS = ["What's blocking the billing queue?", "Which loads are at risk from the storm today?", "Draft this week's AR summary for Acme"];

const TERMS_KEY = "trenova-desk-terms-seen";
function TermsNote() {
  const [show, setShow] = React.useState(() => { try { return !localStorage.getItem(TERMS_KEY); } catch (e) { return true; } });
  const [out, setOut] = React.useState(false);
  const close = () => { try { localStorage.setItem(TERMS_KEY, "1"); } catch (e) {} setOut(true); setTimeout(() => setShow(false), 220); };
  React.useEffect(() => { if (!show) return; window.addEventListener("desk:sent", close); return () => window.removeEventListener("desk:sent", close); }, [show]);
  if (!show) return null;
  return (
    <div className="tnote-w">
      <div className={"tnote" + (out ? " out" : "")}>
        <span>Make sure you agree to our <a href="#" onClick={e => e.preventDefault()}>terms</a> and our <a href="#" onClick={e => e.preventDefault()}>privacy policy</a>.</span>
        <button className="tnote-x" onClick={close} aria-label="Dismiss"><Ic n="x" s={12} w={2.2} /></button>
      </div>
    </div>
  );
}

function Home({ value, setValue, onSend, up, drop, cfg = {} }) {
  return (
    <div className="home">
      <div className="home-in">
        <div className="date">Friday, October 2</div>
        <h1 className="greet">Good morning, Avery</h1>
        <p className="headline">Fifteen items wait on a biller and three loads sit under a storm warning.</p>
        <TermsNote />
        <Composer value={value} setValue={setValue} onSend={onSend} home placeholder="Ask the Desk anything…" presets={cfg.presets === "off" || cfg.motion === "reduced" ? null : HOME_PRESETS} placeholder={cfg.presets === "off" || cfg.motion === "reduced" ? "Ask the Desk anything…" : undefined} up={up} drop={drop} cfg={cfg} />
      </div>
    </div>
  );
}

Object.assign(window, { Rail, TopBar, Composer, DecisionCard, Panel, Home });
