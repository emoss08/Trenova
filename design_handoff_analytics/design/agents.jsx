const SHELVES = [["Chat", "chat", "Chat", "Answer people in the assistant"], ["Scheduled", "calendar", "Scheduled", "Run on a timetable"], ["Event", "bolt", "On an event", "Run when something happens"]];
const MODE_NOTE = { live: "Proposals are offered to a person for a decision.", shadow: "Runs, but its proposals are recorded rather than offered.", sim: "Its writes are previewed and recorded, never made." };
const TEMPLATES = [["chat", "Desk agent", "Answers people in the assistant"], ["calendar", "Scheduled report", "Runs on a timetable and sends a summary"], ["bolt", "Event watcher", "Wakes when something happens"], ["sparkle", "Start blank", "Pick tools and instructions yourself"]];
function Runs({ d, on }) {
  const max = Math.max(1, ...d);
  return <span className={"runs" + (on ? "" : " off")} title="Runs, last 14 days">{d.map((v, i) => <i key={i} style={{ height: v ? 3 + (v / max) * 15 : 2 }} className={v ? "" : "z"}></i>)}</span>;
}
function AgentDetail({ a, S, A }) {
  const mode = a.sim ? "sim" : a.shadow ? "shadow" : "live";
  const thr = S.pol.threshold, rec = a.rec;
  return (
    <div className="ad">
      <div className="ad-g">
        <div className="ad-s">
          <h4>Mode</h4>
          <Seg v={mode} opts={[["live", "Live"], ["shadow", "Shadow"], ["sim", "Simulation"]]} onChange={k => A.setMode(a.id, k)} cls="sm"></Seg>
          <p className="ad-h">{MODE_NOTE[mode]}</p>
          <h4>Who can use it</h4>
          <p className="ad-v"><Ic n="users" s={12}></Ic>{a.access === "Everyone" ? "Everyone who can use the assistant" : "Dispatch lead, Billing manager, Owner"}</p>
          {a.trig === "Scheduled" && <><h4>Schedule</h4><p className="ad-v mono">{a.cron}<span className="dim"> · {a.tz}</span></p><p className="ad-h">Next run {a.next}</p></>}
          {a.trig === "Event" && <><h4>Wakes on</h4><div className="chips">{a.events.map(e => <span key={e} className="tg">{e}</span>)}</div></>}
        </div>
        <div className="ad-s">
          <h4>Tools<em className="mono">{a.tools || ""}</em></h4>
          {a.tools ? <><div className="tier">{TIERS.map((t, i) => a.tiers[i] ? <i key={t} className={"t" + i} style={{ flex: a.tiers[i] }}></i> : null)}</div>
            <div className="tier-l">{TIERS.map((t, i) => <span key={t}><i className={"t" + i}></i><b className="mono">{a.tiers[i]}</b>{t}</span>)}</div></>
            : <p className="ad-h">No task tools. It explains how things work and can't read or change records.</p>}
          {a.can && <><h4>Can hand work to</h4><p className="ad-v"><Ic n="swap" s={12}></Ic>{a.can}</p></>}
        </div>
        <div className="ad-s">
          <h4>Track record</h4>
          {rec ? <>
            <div className="rec"><span><b className="mono">{rec.ap}</b>approved</span><span><b className="mono">{rec.ch}</b>changed</span><span><b className="mono">{rec.rj}</b>rejected</span><span><b className={"mono" + (rec.fl ? " t-d" : "")}>{rec.fl}</b>failed</span></div>
            {S.pol.earned ? <div className="streak"><div className="streak-d">{Array.from({ length: Math.min(thr, 25) }).map((_, i) => <i key={i} className={i < rec.streak ? "on" : ""}></i>)}</div><span>{rec.streak >= thr ? <><b>{rec.tool}</b> earned its next tier</> : <>{thr - rec.streak} more clean approvals and <b>{rec.tool}</b> moves up a tier</>}</span></div>
              : <p className="ad-h">A streak of {rec.streak} clean approvals on {rec.tool}. Earned autonomy is off for the organization, so it changes nothing. <button className="lnk" onClick={() => A.setPol("earned", true)}>Turn it on</button></p>}
          </> : <p className="ad-h">No decisions yet. The record starts with the first proposal someone decides on.</p>}
        </div>
      </div>
      <div className="ad-bar">
        <button className="xa" onClick={() => A.edit({ k: "agent", id: a.id })}><Ic n="edit" s={13}></Ic>Edit agent<span className="kbd">E</span></button>
        {a.trig !== "Chat" && <button className="xa" disabled={!a.on} onClick={() => A.run(a)}><Ic n="play" s={12}></Ic>Run now<span className="kbd">R</span></button>}
        <button className="xa" onClick={() => A.go("activity")}><Ic n="timeline" s={13}></Ic>Runs and proposals</button>
        {a.trig === "Chat" && <button className="xa" onClick={() => A.toast("Opening Desk with " + a.n)}><Ic n="chat" s={13}></Ic>Ask in Desk</button>}
        <span className="sp"></span>
        <button className="xa d" disabled={a.sys} title={a.sys ? "Started by Trenova itself; cannot be removed" : ""} onClick={() => A.removeAgent(a)}><Ic n={a.sys ? "lock" : "trash"} s={13}></Ic>{a.sys ? "System agent" : "Remove"}</button>
      </div>
    </div>
  );
}
const LIVE = { dispatch: "Proposing a driver for SHP-48302", inbox: "Filing a rate confirmation", coverage: "Reviewing 6 uncovered moves" };
const SHADOW_SEEN = { billev: 18, coverage: 31, inbox: 12 };
function liveOf(a, S) { return !a.on || !S.providers.length || S.paused ? null : LIVE[a.id] || null; }
function approval(a) { if (!a.rec) return null; const d = a.rec.ap + a.rec.ch + a.rec.rj; return d ? (a.rec.ap + a.rec.ch) / d : null; }
function AgentRow({ a, open, onOpen, S, A }) {
  const none = !S.providers.length, lv = liveOf(a, S), ap = approval(a), tot = a.tiers.reduce((t, x) => t + x, 0), runs = a.runs.reduce((t, x) => t + x, 0);
  const mode = a.sim ? <span className="tg v">Simulation</span> : a.shadow ? <span className="tg"><Ic n="eyeOff" s={10}></Ic>Shadow</span> : null;
  const meta = a.trig === "Scheduled" ? <><Ic n="clock" s={10}></Ic>Next Thu 6:00 AM</> : a.trig === "Event" ? <><Ic n="bolt" s={10}></Ic>{a.events[0]}{a.events.length > 1 ? " +" + (a.events.length - 1) : ""}</> : a.access !== "Everyone" ? <><Ic n="users" s={10}></Ic>{a.access}</> : null;
  return (
    <div className={"al" + (open ? " sel" : "") + (a.on ? "" : " dim")} id={"ag-" + a.id} onClick={onOpen} tabIndex={0} onKeyDown={e => e.key === "Enter" && e.target === e.currentTarget && onOpen()}>
      <span className="pc-mk"><Tile a={a} s={32}></Tile></span>
      <span className="al-t">
        <span className="al-n"><b>{a.n}</b>{mode}{a.sys && <span className="tg"><Ic n="lock" s={10}></Ic>System</span>}</span>
        {lv ? <span className="al-d shm">{lv}…</span> : <span className="al-d">{meta && <span className="al-m">{meta}</span>}{a.d}</span>}
      </span>
      <span className="al-c al-act">{none && a.on ? <span className="t-w al-s">Waiting for a provider</span> : <><Runs d={a.runs} on={a.on && !none}></Runs><span className="mono al-s">{runs ? runs + " runs" : a.on ? "Not run yet" : "—"}</span></>}</span>
      <span className="al-c al-ap">{a.shadow && SHADOW_SEEN[a.id] ? <span className="al-s" title="Proposals recorded in shadow"><b className="mono">{SHADOW_SEEN[a.id]}</b> recorded</span> : ap != null ? <><Ring v={ap} s={18} cls={ap < 0.85 ? "w" : ""}></Ring><span className="mono">{Math.round(ap * 100)}%</span></> : <span className="dim">—</span>}</span>
      <span className="al-c al-w">{a.on && a.pend && !none ? <button className="pill w" onClick={e => { e.stopPropagation(); A.toast("Opening Desk decisions for " + a.n); }}><Ic n="inbox" s={11}></Ic>{a.pend}</button> : <span className="dim">—</span>}</span>
      <span className="al-c al-tl" title={a.tools ? TIERS.map((t, i) => a.tiers[i] + " " + t.toLowerCase()).join(" · ") : "No task tools"}>{a.tools ? <><span className="tier sm">{TIERS.map((t, i) => a.tiers[i] ? <i key={t} className={"t" + i} style={{ flex: a.tiers[i] }}></i> : null)}</span><span className="mono">{a.tools}</span></> : <span className="dim">—</span>}</span>
      <span className="al-x" onClick={e => e.stopPropagation()}>
        {a.trig !== "Chat" ? <button className="ib" title="Run now" disabled={!a.on} onClick={() => A.run(a)}><Ic n="play" s={12}></Ic></button> : <button className="ib" title="Ask in Desk" disabled={!a.on} onClick={() => A.toast("Opening Desk with " + a.n)}><Ic n="chat" s={13}></Ic></button>}
        <Switch on={a.on} label={a.on ? "Disable agent" : "Enable agent"} onChange={v => A.toggleAgent(a.id, v)}></Switch>
      </span>
    </div>
  );
}
function AgentsTab({ S, A, searchRef }) {
  const [q, setQ] = React.useState(""); const [f, setF] = React.useState("all"); const [open, setOpen] = React.useState(S.focus); const [menu, setMenu] = React.useState(false);
  React.useEffect(() => { if (S.focus) { setOpen(S.focus); setTimeout(() => { const el = document.getElementById("ag-" + S.focus), sc = document.querySelector(".ws"); if (el && sc) sc.scrollTo({ top: el.offsetTop - 140, behavior: "smooth" }); }, 60); } }, [S.focus]);
  React.useEffect(() => { const n = () => setMenu(true); window.addEventListener("aic-new", n); return () => window.removeEventListener("aic-new", n); }, []);
  React.useEffect(() => { const k = e => { if (!open || /INPUT|TEXTAREA/.test(e.target.tagName) || e.metaKey || e.ctrlKey) return; const a = S.agents.find(x => x.id === open); if (!a) return; if (e.key === "e") A.edit({ k: "agent", id: a.id }); if (e.key === "r" && a.trig !== "Chat" && a.on) A.run(a); if (e.key === "Escape") setOpen(null); }; window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k); }, [open, S.agents]);
  const none = !S.providers.length;
  const c = { all: S.agents.length, wait: S.agents.filter(a => a.on && a.pend).length, shadow: S.agents.filter(a => a.shadow).length, off: S.agents.filter(a => !a.on).length };
  const list = S.agents.filter(a => (f === "all" || (f === "wait" && a.on && a.pend) || (f === "shadow" && a.shadow) || (f === "off" && !a.on)) && (a.n + " " + a.d).toLowerCase().includes(q.toLowerCase()));
  return (
    <div className="tabp">
      {!none && (() => { const on = S.agents.filter(x => x.on), pend = on.reduce((t, x) => t + x.pend, 0), waitAg = on.filter(x => x.pend), sh = on.filter(x => x.shadow), shN = sh.reduce((t, x) => t + (SHADOW_SEEN[x.id] || 0), 0), live = on.filter(x => liveOf(x, S));
        return <>
          <Nova ctx="Agents" spin={live.length > 0} ctl={pend > 0 && <><button className="btn ink lg" onClick={() => A.toast("Opening Desk decisions")}><Ic n="inbox" s={13}></Ic>Review {pend} in Desk</button><span>{waitAg.map(x => x.n).join(" and ")}</span></>}>
            <b>{on.length} of {S.agents.length}</b> agents are on{live.length ? <>, and <b>{live.length}</b> {live.length > 1 ? "are" : "is"} working right now</> : ""}. <span className="ref w" onClick={() => setF("wait")}>{pend} proposals</span> wait on a person.{sh.length ? <> {sh.length} agents run in <span className="ref" onClick={() => setF("shadow")}>shadow</span> and have recorded {shN} proposals nobody has seen — worth a look before you let them go live.</> : ""}
          </Nova>
          {live.length > 0 && <div className="lv">{live.map(x => <button key={x.id} className="lv-c" onClick={() => setOpen(x.id)}><span className="pc-mk"><Tile a={x} s={22}></Tile><i className="pc-dot run sm"></i></span><span className="lv-t"><b>{x.n}</b><span className="shm">{liveOf(x, S)}</span></span></button>)}</div>}
        </>; })()}
      {none && <div className="bnr"><Ic n="plug" s={14}></Ic><span><b>Agents can't answer until a provider is connected.</b> Their settings are kept; they start as soon as one is on.</span><button className="btn sm ink" onClick={() => A.go("providers")}>Connect a provider</button></div>}
      {S.paused && !none && <div className="bnr w"><Ic n="pause" s={13} w={2.4}></Ic><span><b>All agents are paused organization-wide.</b> Each one below keeps its own switch for when you resume.</span><button className="btn sm" onClick={() => A.setPaused(false)}>Resume</button></div>}
      <div className="tb">
        <Search q={q} setQ={setQ} ph="Search agents" inputRef={searchRef}></Search>
        <div className="seg">{[["all", "All"], ["wait", "Waiting"], ["shadow", "Shadow"], ["off", "Off"]].map(([k, l]) => <button key={k} className={f === k ? "on" : ""} onClick={() => setF(k)}>{l}<em className="mono">{c[k]}</em></button>)}</div>
        <span className="sp"></span>
        <span className="tb-ct mono">{S.agents.filter(a => a.on).length} of {S.agents.length} on</span>
        <div className="rel"><button className="btn ink" onClick={() => setMenu(m => !m)}><Ic n="plus" s={13}></Ic>New agent<span className="kbd">N</span></button>
          {menu && <Menu right onClose={() => setMenu(false)} items={[{ h: "Start from" }, ...TEMPLATES.map(([ic, l, s]) => ({ icon: <Ic n={ic} s={14}></Ic>, l, s, on: () => A.edit({ k: "agent", tpl: l }) }))]}></Menu>}
        </div>
      </div>
      {SHELVES.map(([k, ic, l, note]) => { const xs = list.filter(a => a.trig === k); if (!xs.length) return null; return (
        <section key={k} className="shelf">
          <header className="shelf-h"><Ic n={ic} s={13}></Ic><b>{l}</b><span>{note}</span><em className="mono">{xs.length}</em></header>
          <div className="al-hd"><span></span><span>Agent</span><span>Last 14 days</span><span>Approved</span><span>Waiting</span><span>Tools</span><span></span></div>
          <div className="rows">{xs.map(a => <AgentRow key={a.id} a={a} open={open === a.id} onOpen={() => setOpen(open === a.id ? null : a.id)} S={S} A={A}></AgentRow>)}</div>
        </section>); })}
      {!list.length && <div className="empty"><b>No agents match</b><span>Try another word, or clear the filter.</span><button className="btn sm" onClick={() => { setQ(""); setF("all"); }}>Clear</button></div>}
      {(() => { const a = S.agents.find(x => x.id === open); if (!a) return null; return (
        <Sheet onClose={() => setOpen(null)} head={<><Tile a={a} s={36}></Tile><div className="sh-t"><b>{a.n}</b><span>{SHELVES.find(s => s[0] === a.trig)[2]} · {a.tools ? a.tools + " tools" : "no task tools"}{a.sys ? " · system" : ""}</span></div><button className="btn sm" onClick={() => A.edit({ k: "agent", id: a.id })}><Ic n="edit" s={12}></Ic>Edit<span className="kbd">E</span></button><Switch on={a.on} label={a.on ? "Disable agent" : "Enable agent"} onChange={v => A.toggleAgent(a.id, v)}></Switch></>}>
          <p className="sh-d">{a.d}</p>
          <AgentDetail a={a} S={S} A={A}></AgentDetail>
        </Sheet>); })()}
    </div>
  );
}
Object.assign(window, { AgentsTab });
