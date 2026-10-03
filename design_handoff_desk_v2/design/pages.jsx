// Today, Memory and Agent pages + shared schedule store.
window.DESK_SCHED = window.DESK_SCHED || [
  { id: "s1", when: "Every weekday · 7:30 AM", next: "Mon, Oct 5 · 7:30 AM", prompt: "What's blocking the billing queue?", on: true, last: "Today 7:30 AM" },
  { id: "s2", when: "Every Monday · 8:00 AM", next: "Mon, Oct 5 · 8:00 AM", prompt: "Summarize last week's detention by customer", on: true, last: "Sep 28" },
];
const schedEmit = () => window.dispatchEvent(new Event("desk:sched"));
function useSched() { const [, f] = React.useState(0); React.useEffect(() => { const h = () => f(x => x + 1); window.addEventListener("desk:sched", h); return () => window.removeEventListener("desk:sched", h); }, []); return window.DESK_SCHED; }
function schedAdd(prompt) {
  const m = prompt.match(/^(?:\/schedule\s+)?(every|each)\s+(weekday|day|morning|monday|tuesday|wednesday|thursday|friday|week)\w*(?:\s+at\s+([\d:]+\s*(?:am|pm)?))?[,:]?\s*(.*)$/i);
  const unit = m ? m[2].toLowerCase() : "weekday", at = m && m[3] ? m[3].toUpperCase().replace(/\s*(AM|PM)/, " $1") : "8:00 AM";
  const atN = /:/.test(at) ? at : at.replace(/^(\d+)/, "$1:00");
  const label = unit === "day" || unit === "morning" ? "Every day" : unit === "weekday" ? "Every weekday" : unit === "week" ? "Every Monday" : "Every " + unit[0].toUpperCase() + unit.slice(1);
  const body = (m && m[4] ? m[4] : prompt).replace(/^(?:,|\s)*(?:please\s+)?/i, "");
  const it = { id: "s" + Date.now(), when: label + " · " + atN, next: (unit === "day" || unit === "morning" ? "Tomorrow" : "Mon, Oct 5") + " · " + atN, prompt: body[0].toUpperCase() + body.slice(1), on: true, last: null, fresh: true };
  window.DESK_SCHED = [it, ...window.DESK_SCHED]; schedEmit(); return it;
}
function schedToggle(id) { window.DESK_SCHED = window.DESK_SCHED.map(s => s.id === id ? { ...s, on: !s.on } : s); schedEmit(); }
function schedRemove(id) { window.DESK_SCHED = window.DESK_SCHED.filter(s => s.id !== id); schedEmit(); }

function SchedRow({ s, onRun }) {
  return (
    <div className={"sch-r" + (s.on ? "" : " off") + (s.fresh ? " fresh" : "")}>
      <span className="sch-i"><Ic n="clock" s={14} /></span>
      <span className="sch-t"><b>{s.prompt}</b><em>{s.when} · {s.on ? "next " + s.next : "paused"}</em></span>
      {onRun && <button className="ib" title="Run now" onClick={() => onRun(s.prompt)}><Ic n="play" s={13} /></button>}
      <button className="ib" title={s.on ? "Pause" : "Resume"} onClick={() => schedToggle(s.id)}><Ic n={s.on ? "pause" : "play"} s={13} /></button>
      <button className="ib" title="Delete" onClick={() => schedRemove(s.id)}><Ic n="trash" s={13} /></button>
    </div>
  );
}

/* ---------- Today ---------- */
function TodayPage({ onAsk, onWatch, onDecisions, onThread, onQueueItem }) {
  useSched(); (window.useBQ || (() => {}))();
  const [q, setQ] = React.useState("");
  const ids = window.BQ ? Object.keys(BQ.items) : [];
  const gs = ids.map(id => [id, bqGet(id)]);
  const need = gs.filter(([, g]) => g.st === 0 && !g.hold && g.needs > 0), ready = gs.filter(([, g]) => g.ready), done = gs.filter(([, g]) => g.st > 0);
  const total = gs.reduce((s, [, g]) => s + g.total, 0);
  const dec = window.DEC_WAITING != null ? window.DEC_WAITING : (window.DEC_ITEMS0 || []).length;
  const crit = (typeof WT_ITEMS0 !== "undefined" ? WT_ITEMS0 : []).filter(w => w.sev === "Critical");
  const handled = typeof WT_HANDLED !== "undefined" ? WT_HANDLED : [];
  const nNeed = dec + need.length + crit.length;
  return (
    <div className="pg"><div className="pg-in">
      <header className="pg-h">
        <span className="pg-k">Friday, October 3</span>
        <h1>Good evening, Avery</h1>
        <p>{nNeed} things need you. Agents handled {handled.length} on their own since yesterday.</p>
      </header>
      <form className="td-ask" onSubmit={e => { e.preventDefault(); if (q.trim()) onAsk(q.trim()); }}>
        <Ic n="chat" s={15} /><input value={q} onChange={e => setQ(e.target.value)} placeholder="Ask Billing Specialist anything…" />
        <button className="td-go" disabled={!q.trim()}><Ic n="up" s={14} w={2.2} /></button>
      </form>
      <div className="td-grid">
        <section className="pgc span2">
          <div className="pgc-h"><h2>Waiting on you</h2></div>
          <div className="td-wait">
            <button onClick={onDecisions}><span className="td-n">{dec}</span><span><b>Decisions to review</b><em>Changes agents drafted and are holding for you</em></span><Ic n="chevR" s={13} /></button>
            <button onClick={() => need[0] && onQueueItem(need[0][0])} disabled={!need.length}><span className="td-n">{need.length}</span><span><b>Billing items need you</b><em>{need.length ? need.slice(0, 3).map(([id]) => id).join(", ") + (need.length > 3 ? " and " + (need.length - 3) + " more" : "") : "All clear"}</em></span><Ic n="chevR" s={13} /></button>
            <button onClick={onWatch}><span className="td-n warn">{crit.length}</span><span><b>Critical in Watchtower</b><em>{crit[0] ? crit[0].title : "Nothing critical"}</em></span><Ic n="chevR" s={13} /></button>
          </div>
        </section>
        <section className="pgc">
          <div className="pgc-h"><h2>Billing queue</h2><span>${Math.round(total).toLocaleString()}</span></div>
          <div className="td-bar">{[[ready.length, "ok"], [need.length, "warn"], [done.length, "ink"]].map(([n, k]) => n ? <i key={k} className={k} style={{ flex: n }}></i> : null)}</div>
          <div className="td-leg"><span><i className="ok"></i>{ready.length} ready</span><span><i className="warn"></i>{need.length} need you</span><span><i className="ink"></i>{done.length} approved</span></div>
          <button className="pgc-link" onClick={() => onQueueItem(null)}>Open the queue<Ic n="chevR" s={12} /></button>
        </section>
        <section className="pgc">
          <div className="pgc-h"><h2>Handled since yesterday</h2></div>
          <div className="td-log">{handled.map(h => <div key={h.id}><span className="td-ck"><Ic n="check" s={10} w={2.8} /></span><span><b>{h.title}</b><em>{h.how} · {h.ago}</em></span></div>)}</div>
        </section>
        <section className="pgc">
          <div className="pgc-h"><h2>Scheduled</h2><span>{window.DESK_SCHED.filter(s => s.on).length} active</span></div>
          <div className="sch-l">{window.DESK_SCHED.map(s => <SchedRow key={s.id} s={s} onRun={onAsk} />)}</div>
          <p className="pgc-tip">Start a message with “every Monday at 8…” to schedule it.</p>
        </section>
        <section className="pgc">
          <div className="pgc-h"><h2>Pick up where you left off</h2></div>
          <div className="td-cv">{CONVOS.flatMap(g => g.items).slice(0, 4).map(c => <button key={c.t} onClick={onThread}><b>{c.t}</b><em>{c.a} · {c.w}</em></button>)}</div>
        </section>
      </div>
    </div></div>
  );
}

/* ---------- Memory ---------- */
Object.assign(MEMORIES, {
  m3: { text: "Harbor Supply Co. wants PODs attached to every invoice, even when the load delivered on time.", scope: "Billing team", when: "Sep 12", src: "Harbor dispute follow-up" },
  m4: { text: "Lumper fees on Acme loads are passed through when there's a receipt (contract §5.3).", scope: "Organization", when: "Aug 29", src: "Acme contract review" },
  m5: { text: "Avery approves invoices in batches after 4 PM. Don't nudge before then.", scope: "Just you", when: "Sep 22", src: "Billing queue · Sep 22" },
  m6: { text: "Bluewater's dock code BW-DC-17 maps to their Plainfield, IN warehouse.", scope: "Organization", when: "Oct 1", src: "EDI quarantine fix" },
  m7: { text: "Use the DOE weekly average from Monday for fuel surcharges, never the daily price.", scope: "Organization", when: "Jul 14", src: "Fuel surcharge setup" },
});
const MEM_USE = { m1: [14, "Today"], m2: [6, "Today"], m3: [9, "Yesterday"], m4: [3, "Today"], m5: [11, "Oct 1"], m6: [2, "Oct 1"], m7: [27, "Today"] };
const MEM_SCOPES = ["Just you", "Billing team", "Organization"];

function MemoryPage({ memAsk }) {
  const [list, setList] = React.useState(() => Object.keys(MEMORIES).map(id => ({ id, ...MEMORIES[id], used: (MEM_USE[id] || [0])[0], lastUsed: (MEM_USE[id] || [0, "Never"])[1], paused: false, gone: false })));
  const [f, setF] = React.useState("All");
  const [q, setQ] = React.useState("");
  const [edit, setEdit] = React.useState(null);
  const [draft, setDraft] = React.useState("");
  const [add, setAdd] = React.useState("");
  const [addScope, setAddScope] = React.useState("Just you");
  const [mode, setMode] = React.useState(memAsk ? "ask" : "auto");
  const up = (id, p) => setList(l => l.map(m => m.id === id ? { ...m, ...p } : m));
  const shown = list.filter(m => (f === "All" || m.scope === f) && (!q || m.text.toLowerCase().includes(q.toLowerCase())));
  const save = id => { up(id, { text: draft }); MEMORIES[id] && (MEMORIES[id].text = draft); setEdit(null); };
  const create = e => { e.preventDefault(); if (!add.trim()) return; const id = "m" + Date.now(); MEMORIES[id] = { text: add.trim(), scope: addScope, when: "Today", src: "Added by you" }; setList(l => [{ id, ...MEMORIES[id], used: 0, lastUsed: "Never", paused: false, gone: false, fresh: true }, ...l]); setAdd(""); };
  return (
    <div className="pg"><div className="pg-in narrow">
      <header className="pg-h">
        <span className="pg-k">Memory</span>
        <h1>What Desk remembers</h1>
        <p>Agents use these to answer the way your team works. Edit anything that's wrong, pause what you're unsure about, or forget it.</p>
      </header>
      <div className="mm-set">
        <span><b>Saving new memories</b><em>{mode === "auto" ? "Agents save useful facts and tell you in the conversation" : "Agents ask before saving anything"}</em></span>
        <div className="mm-seg">{[["auto", "Automatically"], ["ask", "Ask me first"]].map(([k, l]) => <button key={k} className={mode === k ? "on" : ""} onClick={() => setMode(k)}>{l}</button>)}</div>
      </div>
      <form className="mm-add" onSubmit={create}>
        <Ic n="plus" s={14} /><input value={add} onChange={e => setAdd(e.target.value)} placeholder="Teach Desk something, e.g. “Granite pays by ACH only”" />
        <select value={addScope} onChange={e => setAddScope(e.target.value)}>{MEM_SCOPES.map(s => <option key={s}>{s}</option>)}</select>
        <button className="ax-btn ink" disabled={!add.trim()}>Save</button>
      </form>
      <div className="mm-tools">
        <div className="mm-f">{["All", ...MEM_SCOPES].map(s => <button key={s} className={f === s ? "on" : ""} onClick={() => setF(s)}>{s}<em>{s === "All" ? list.filter(m => !m.gone).length : list.filter(m => !m.gone && m.scope === s).length}</em></button>)}</div>
        <label className="ax-q"><Ic n="search" s={13} /><input value={q} onChange={e => setQ(e.target.value)} placeholder="Search memories" /></label>
      </div>
      <div className="mm-l">
        {shown.map(m => m.gone ? (
          <div key={m.id} className="mm-r gone"><span>Forgotten. Agents won't use this again.</span><button onClick={() => up(m.id, { gone: false })}>Undo</button></div>
        ) : (
          <div key={m.id} className={"mm-r" + (m.paused ? " paused" : "") + (m.fresh ? " fresh" : "")}>
            {edit === m.id ? (
              <div className="mm-ed"><textarea autoFocus value={draft} onChange={e => setDraft(e.target.value)} rows={2}></textarea>
                <div><button className="ax-btn ghost" onClick={() => setEdit(null)}>Cancel</button><button className="ax-btn ink" onClick={() => save(m.id)} disabled={!draft.trim()}>Save</button></div></div>
            ) : <p>{m.text}</p>}
            <div className="mm-m">
              <select className="mm-scope" value={m.scope} onChange={e => up(m.id, { scope: e.target.value })}>{MEM_SCOPES.map(s => <option key={s}>{s}</option>)}</select>
              <span>Saved {m.when} from “{m.src}”</span>
              <span>{m.paused ? "Paused" : m.used ? "Used " + m.used + "× · last " + m.lastUsed.toLowerCase() : "Not used yet"}</span>
              <span className="mm-acts">
                <button className="ib" title="Edit" onClick={() => { setEdit(m.id); setDraft(m.text); }}><Ic n="edit" s={13} /></button>
                <button className="ib" title={m.paused ? "Resume" : "Pause"} onClick={() => up(m.id, { paused: !m.paused })}><Ic n={m.paused ? "play" : "pause"} s={13} /></button>
                <button className="ib" title="Forget" onClick={() => up(m.id, { gone: true })}><Ic n="trash" s={13} /></button>
              </span>
            </div>
          </div>
        ))}
        {!shown.length && <div className="mm-empty">No memories match.</div>}
      </div>
    </div></div>
  );
}

/* ---------- Agent capabilities ---------- */
const AG_CAPS = [
  ["Look things up", [["billing.queue.list", "Read the billing queue", "allow"], ["invoices.search", "Search invoices and aging", "allow"], ["shipments.get", "Open shipments and stops", "allow"], ["documents.read", "Read rate cons, PODs and BOLs", "allow"]]],
  ["Make changes", [["billing.assign_biller", "Assign billers", "ask"], ["billing.post_invoices", "Post invoices", "ask", "Always asks · posting can't be undone"], ["messages.send", "Email customers", "ask"], ["invoices.void", "Void invoices", "off"]]],
];
const AG_HANDS = [["Pay rates, people records", "Workforce Coordinator"], ["Driver and load assignments", "Dispatch desk"], ["Applying customer payments", "Cash Application"]];
const AG_MODES = [["allow", "Allowed"], ["ask", "Ask first"], ["off", "Off"]];

function AgentPage({ onBack }) {
  const ag = AGENTS[0];
  const [caps, setCaps] = React.useState(() => Object.fromEntries(AG_CAPS.flatMap(([, xs]) => xs.map(x => [x[0], x[2]]))));
  const [on, setOn] = React.useState(true);
  const [hours, setHours] = React.useState(false);
  return (
    <div className="pg"><div className="pg-in narrow">
      <button className="pg-back" onClick={onBack}><Ic n="chevL" s={13} />Back to conversation</button>
      <header className="ag-h">
        <AgentTile agent={ag} size="lg" />
        <div><h1>{ag.name}</h1><p>{ag.desc} · Claude Sonnet 4.5 · set up by Jordan Pike</p></div>
        <button className={"ag-on" + (on ? " on" : "")} onClick={() => setOn(o => !o)}><i></i>{on ? "On" : "Off"}</button>
      </header>
      {AG_CAPS.map(([g, xs]) => (
        <section key={g} className="pgc">
          <div className="pgc-h"><h2>{g}</h2><span>{g === "Make changes" ? "What it can change, and when it asks you" : "What it can read"}</span></div>
          <div className="ag-l">{xs.map(([k, l, , lock]) => (
            <div key={k} className="ag-r">
              <span className="ag-t"><b>{l}</b><code>{k}</code>{lock && <em><Ic n="lock" s={11} />{lock}</em>}</span>
              <div className={"mm-seg sm" + (lock ? " locked" : "")}>{AG_MODES.filter(([m]) => g === "Make changes" || m !== "ask").map(([m, ml]) => <button key={m} disabled={!!lock} className={caps[k] === m ? "on " + m : ""} onClick={() => setCaps(c => ({ ...c, [k]: m }))}>{ml}</button>)}</div>
            </div>))}</div>
        </section>
      ))}
      <section className="pgc">
        <div className="pgc-h"><h2>Hands off to</h2><span>Questions outside billing go to these agents</span></div>
        <div className="ag-l">{AG_HANDS.map(([w, a]) => <div key={a} className="ag-r"><span className="ag-t"><b>{w}</b></span><span className="ag-to"><Ic n="handoff" s={13} />{a}</span></div>)}</div>
      </section>
      <section className="pgc">
        <div className="pgc-h"><h2>Limits</h2></div>
        <div className="ag-lim">
          {[["Requests today", 143, 200, "Resets at midnight"], ["October budget", 460, 500, "$460 of $500 · resets Nov 1"]].map(([l, v, mx, sub]) => (
            <div key={l}><span><b>{l}</b><em>{sub}</em></span><span className="ag-n">{l.includes("budget") ? "$" + v : v}<i>/ {l.includes("budget") ? "$" + mx : mx}</i></span><span className={"ag-bar" + (v / mx > 0.85 ? " hi" : "")}><i style={{ width: (v / mx) * 100 + "%" }}></i></span></div>
          ))}
          <div className="ag-r"><span className="ag-t"><b>Largest single change</b><em>Bigger batches are split and approved separately</em></span><span className="ag-n">500<i> items</i></span></div>
          <div className="ag-r"><span className="ag-t"><b>Only change things during business hours</b><em>7 AM – 6 PM Central</em></span><button className={"ag-on sm" + (hours ? " on" : "")} onClick={() => setHours(h => !h)}><i></i></button></div>
        </div>
      </section>
    </div></div>
  );
}

Object.assign(window, { TodayPage, MemoryPage, AgentPage, schedAdd, useSched, SchedRow });
