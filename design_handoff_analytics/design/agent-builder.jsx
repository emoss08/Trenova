const COLS = { Propose: ["Proposes", "A person does it"], ActWithApproval: ["Asks first", "Runs once a person approves"], AutoExecute: ["On its own", "Runs without asking"] };
const TRIGS = [["Chat", "chat", "Someone asks", "In Desk or the assistant"], ["Scheduled", "calendar", "On a schedule", "On a timetable"], ["Event", "bolt", "Something happens", "On events in Trenova"], ["Continuous", "refresh", "Keeps watch", "Every few minutes"]];
const INTERVALS = [[60, "1 min"], [300, "5 min"], [900, "15 min"], [3600, "1 hour"]];
const CTX = [["Organization", "Your organization"], ["Person", "The person asking"], ["Today", "Today's operations"], ["Exceptions", "Open exceptions"], ["Memory", "Memories about the record"]];
const EXTRA = a => ({ guard: a ? ["Promise a customer a delivery date or a rate", "Release a hold over $5,000"] : [], data: "Internal", expire: 86400, interval: 300, conc: 1, endsAt: "", timeout: 600, limits: a ? { release_billing_hold: 25 } : {}, memTok: "", ctx: [], out: "Conversational", enabled: a ? a.on : true });
const MODES = [["live", "Live", "Offers proposals; automatic tools run"], ["shadow", "Shadow", "Runs for real, records proposals instead of offering them"], ["sim", "Simulation", "Previews every write, never makes one"]];
function hl(t) { return t.split(/(\{\{[^}]+\}\})/g).map((x, i) => /^\{\{/.test(x) ? <mark key={i}>{x}</mark> : x); }
function Prog({ v, s = 30 }) { const r = s / 2 - 3, c = 2 * Math.PI * r; return <svg width={s} height={s} className="prog"><circle cx={s / 2} cy={s / 2} r={r}></circle><circle cx={s / 2} cy={s / 2} r={r} className="f" strokeDasharray={c} strokeDashoffset={c * (1 - v)} transform={`rotate(-90 ${s / 2} ${s / 2})`}></circle></svg>; }
function Blk({ id, t, s, r, fresh, children }) {
  return <section id={"ab-" + id} className={"blk" + (fresh ? " fresh" : "")}><header className="blk-h"><div><h2>{t}</h2>{s && <p>{s}</p>}</div>{r}</header>{children}</section>;
}
function Pop({ onClose, cls, children }) {
  React.useEffect(() => { const c = e => !e.target.closest(".pop") && onClose(); const k = e => { if (e.key === "Escape") { e.preventDefault(); onClose(); } }; setTimeout(() => document.addEventListener("mousedown", c)); window.addEventListener("keydown", k, true); return () => { document.removeEventListener("mousedown", c); window.removeEventListener("keydown", k, true); }; }, []);
  return <div className={"pop " + (cls || "")}>{children}</div>;
}
function AgentBuilder({ a, tpl, S, A, onClose, demo }) {
  const d = useDraft({ ...EXTRA(a), ...agentDraft(a, tpl), ...(a ? {} : { budget: "" }) });
  const [gIn, setGIn] = React.useState(""); const v = d.v, set = d.set;
  const [intro, setIntro] = React.useState(!a);
  const [fresh, setFresh] = React.useState([]);
  const [tryOpen, setTryOpen] = React.useState(S.tryOpen);
  const [tested, setTested] = React.useState(null);
  const [act, setAct] = React.useState("who");
  const [pop, setPop] = React.useState(null);
  const [drag, setDrag] = React.useState(null); const [over, setOver] = React.useState(null); const [deny, setDeny] = React.useState(null);
  const [conf, setConf] = React.useState(demo === "conflict" && !!a);
  const [rm, setRm] = React.useState(false); const [tight, setTight] = React.useState(false);
  const canvas = React.useRef(), ta = React.useRef();
  React.useEffect(() => { if (demo === "unsaved" && a) d.patch({ ceil: "AutoExecute", mode: v.mode === "shadow" ? "live" : v.mode, budget: 60 }); }, []);
  React.useLayoutEffect(() => { const el = ta.current; if (el) { el.style.height = "auto"; el.style.height = el.scrollHeight + "px"; } }, [v.instr, intro]);
  const held = Object.keys(v.tools).map(n => TOOLS.find(t => t.n === n)).filter(Boolean);
  const acts = held.filter(t => t.kind === "Action"), reads = held.filter(t => t.kind !== "Action");
  const eff = t => tierCap(v.tools[t.n], v.ceil);
  const counts = TIER_O.map(tr => acts.filter(t => eff(t) === tr).length);
  const lints = LINT.filter(([re, need]) => re.test(v.instr) && !need.some(n => n in v.tools));
  const outside = acts.filter(t => OUTSIDE.includes(t.e));
  const openRisk = !v.access.length && outside.length > 0 && v.trig === "Chat";
  const cron = CRON.find(c => c[0] === v.cron);
  const trigBad = v.trig === "Event" && !v.events.length ? "Pick at least one event" : v.trig === "Scheduled" && !v.cron.trim() ? "Set a schedule" : v.trig === "Continuous" && v.interval < 60 ? "Run at most once a minute" : null;
  const invalid = intro ? "Describe it or pick a start" : !v.n.trim() ? "Give it a name" : trigBad;
  const snap = JSON.stringify(v);
  const f = useEditFlow({ d, create: !a, invalid, onClose, onSave: x => a ? A.saveAgent(a.id, x) : A.createAgent(x) });
  const fields = { n: { l: "Name" }, d: { l: "Description" }, ic: { l: "Icon" }, h: { l: "Color", fmt: x => "Hue " + x }, instr: { l: "Instructions", fmt: x => x.split(/\s+/).filter(Boolean).length + " words" }, trig: { l: "Starts when" }, access: { l: "Who can use it" }, cron: { l: "Schedule", fmt: x => (CRON.find(c => c[0] === x) || [0, x])[1] }, tz: { l: "Time zone" }, events: { l: "Wakes on" }, ceil: { l: "Ceiling", fmt: x => TIER_L[x] }, mode: { l: "Mode", fmt: x => MODES.find(m => m[0] === x)[1] }, runsDay: { l: "Runs per day", fmt: x => x ? x : "No cap" }, budget: { l: "Monthly budget", fmt: x => x === "" || x == null ? "No cap" : "$" + x }, steps: { l: "Tool calls per run" }, route: { l: "Preferred provider" }, guard: { l: "Never" }, data: { l: "Data access" }, expire: { l: "Proposals expire", fmt: x => (EXPIRE.find(e => e[0] === x) || [0, x + "s"])[1] }, interval: { l: "Every", fmt: x => x + "s" }, conc: { l: "At most" }, endsAt: { l: "Stop after" }, timeout: { l: "Run timeout", fmt: x => Math.round(x / 60) + " min" }, limits: { l: "Daily tool limits", fmt: x => Object.values(x).filter(Boolean).length + " set" }, memTok: { l: "Memory in prompt", fmt: x => x ? x + " tokens" : "Default" }, ctx: { l: "Tell it about" }, out: { l: "Replies as" }, enabled: { l: "Enabled" }, learn: { l: "Learns" }, handoff: { l: "Hands work to", item: id => (S.agents.find(x => x.id === id) || {}).n },
    tools: { l: "Tools", diff: (x, y) => { const L = n => (TOOLS.find(t => t.n === n) || {}).l; return <span className="cr-v">{Object.keys(y).filter(k => !(k in x)).map(n => <em key={n} className="ad">+ {L(n)}</em>)}{Object.keys(x).filter(k => !(k in y)).map(n => <em key={n} className="rm">− {L(n)}</em>)}{Object.keys(y).filter(k => k in x && x[k] !== y[k]).map(n => <em key={n}>{L(n)}: {TIER_L[x[n]]} → {TIER_L[y[n]]}</em>)}</span>; } } };
  const jump = id => { const c = canvas.current, el = c && c.querySelector("#ab-" + id); if (el) c.scrollTo({ top: el.offsetTop - 24, behavior: "smooth" }); setAct(id); };
  const onScroll = () => { const c = canvas.current; if (!c) return; let cur = "who"; c.querySelectorAll(".blk,.hero2").forEach(el => { if (el.offsetTop - c.scrollTop < 140) cur = el.id.replace("ab-", ""); }); if (c.scrollTop + c.clientHeight >= c.scrollHeight - 4) cur = a ? "record" : "team"; setAct(cur); };
  const start = (patch, ai) => { d.patch(patch); setIntro(false); if (ai) { setFresh(["who", "instr", "trig", "tools", "limits"]); setTimeout(() => setFresh([]), 2200); } };
  const insert = s => { const el = ta.current; if (!el) return; const i = el.selectionStart, j = el.selectionEnd; set("instr", x => x.slice(0, i) + s + x.slice(j)); setTimeout(() => { el.focus(); el.selectionStart = el.selectionEnd = i + s.length; }); };
  const giveTool = n => { const t = TOOLS.find(x => x.n === n); set("tools", o => ({ ...o, [n]: tierCap(t.tier, v.ceil) })); };
  const drop = tr => { setOver(null); if (!drag) return; const t = TOOLS.find(x => x.n === drag); if (tierIx(tr) > tierIx(t.tier)) { setDeny({ n: t.n, col: tr, msg: t.l + " can't go past " + TIER_L[t.tier] }); setTimeout(() => setDeny(null), 2200); } else set("tools", o => ({ ...o, [t.n]: tr })); setDrag(null); };
  const nudge = (t, dir) => { const i = tierIx(v.tools[t.n]) + dir; if (i < 0 || i > 2) return; if (i > tierIx(t.tier)) { setDeny({ n: t.n, col: TIER_O[i], msg: t.l + " can't go past " + TIER_L[t.tier] }); setTimeout(() => setDeny(null), 2200); return; } set("tools", o => ({ ...o, [t.n]: TIER_O[i] })); };
  const tighten = () => { setTight(true); setTimeout(() => { set("instr", x => x.replace(/\n{3,}/g, "\n\n").trim() + (/cite/i.test(x) ? "" : "\n\nAlways name the record you're talking about, like SHP-48302 or INV-20931.")); setTight(false); }, 1100); };
  const ai = S.providers.some(p => p.on);
  const status = {
    who: v.n.trim() && v.d.trim() ? "ok" : "todo", instr: lints.length ? "warn" : v.instr.trim().length > 40 ? "ok" : "todo", trig: trigBad ? "todo" : "ok",
    tools: openRisk ? "warn" : held.length ? "ok" : "todo", limits: "ok", team: "ok", test: tested ? (tested === snap ? "ok" : "warn") : "todo" };
  const words = v.instr.split(/\s+/).filter(Boolean).length;
  const trigTxt = v.trig === "Chat" ? (v.access.length ? v.access.length + " roles can ask" : "Anyone can ask") : v.trig === "Scheduled" ? (cron ? cron[1] : "Custom schedule") : v.trig === "Continuous" ? "Every " + (INTERVALS.find(x => x[0] === v.interval) || [0, v.interval + "s"])[1] : v.events.length ? v.events.length + " event" + (v.events.length > 1 ? "s" : "") : "No events yet";
  const RAIL = [["who", "Identity", v.n || "Unnamed"], ["instr", "Instructions", words ? words + " words" + (lints.length ? " · " + lints.length + " gap" : "") : "Not written"], ["trig", "When it runs", trigTxt], ["tools", "Tools and autonomy", held.length + " tools · " + counts[1] + " ask first"], ["limits", "Budget and model", (v.budget === "" || v.budget == null ? "No cap" : "$" + v.budget + "/mo") + " · " + (v.out === "Report" ? "reports" : "conversational")], ["team", "Memory and handoffs", (v.learn && S.pol.learn ? "Learns" : "Doesn't learn") + (v.handoff.length ? " · asks " + v.handoff.length : "")], ...(a ? [["record", "How it's doing", a.rec ? a.rec.ap + " approved" : "No decisions yet"]] : [])];
  const req = ["who", "instr", "trig", "tools", "test"], ready = req.filter(k => status[k] === "ok").length;
  const mode = MODES.find(m => m[0] === v.mode);
  const lower = s => s.charAt(0).toLowerCase() + s.slice(1);
  const sentence = <>
    <span className="sj" onClick={() => jump("trig")}>{v.trig === "Chat" ? <>Answers <b>{v.access.length ? v.access.join(", ") : "anyone"}</b> in Desk</> : v.trig === "Scheduled" ? <>Runs <b>{cron ? lower(cron[2]) : "on a custom schedule"}</b></> : <>Wakes when <b>{v.events.length ? v.events.slice(0, 2).map(lower).join(" or ") : "an event you pick"}</b></>}</span>,{" "}
    <span className="sj" onClick={() => jump("tools")}>reads with <b>{reads.length} tools</b>{counts[2] ? <>, does <b>{counts[2]} things on its own</b></> : ""}{counts[1] ? <>, asks a person before <b>{counts[1]}</b></> : ""}{counts[0] ? <> and only proposes <b>{counts[0]}</b></> : ""}</span>.{" "}
    <span className="sj" onClick={() => setPop("mode")}>{v.mode === "shadow" ? "It runs in shadow, so nothing is offered yet." : v.mode === "sim" ? "It's in simulation, so nothing is written." : "It's live."}</span>
  </>;
  return ReactDOM.createPortal(<>
    <div className={"ab" + (tryOpen && !intro ? " ab-try" : "") + (intro ? " intro" : "")} role="dialog" aria-modal="true" aria-label={a ? "Edit " + a.n : "New agent"}>
      <header className={"ab-h" + (f.confirm ? " cf" : "")}>
        <button className="ab-bk" onClick={f.tryClose}><Ic n="chevR" s={13}></Ic>Agents</button>
        <span className="ab-sl">/</span>
        <Tile a={{ ic: v.ic, h: v.h }} s={22}></Tile><b className="ab-n">{v.n || "New agent"}</b>
        {!intro && <div className="rel"><button className={"mp " + v.mode} onClick={() => setPop(pop === "mode" ? null : "mode")}><i></i>{mode[1]}<Ic n="chevD" s={11}></Ic></button>
          {pop === "mode" && <Pop cls="mp-m" onClose={() => setPop(null)}>{MODES.map(([k, l, s]) => <button key={k} className={"mp-o" + (v.mode === k ? " on" : "")} onClick={() => { set("mode", k); setPop(null); }}><span className={"mp-d " + k}></span><span><b>{l}</b><em>{s}</em></span>{v.mode === k && <Ic n="check" s={13} w={2.2}></Ic>}</button>)}</Pop>}</div>}
        {a && a.sys && <span className="tg"><Ic n="lock" s={10}></Ic>System</span>}{!intro && <label className="ab-en" title="Disabled agents can't be talked to, scheduled or started by events"><Switch on={v.enabled} label="Enabled" onChange={x => set("enabled", x)}></Switch><span>{v.enabled ? "Enabled" : "Disabled"}</span></label>}
        <span className="sp"></span>
        {!intro && <>
          <button className={"ab-tb" + (tryOpen ? " on" : "")} onClick={() => setTryOpen(o => !o)}><Ic n="flask" s={13}></Ic>Try it</button>
          {a && <div className="rel"><button className="ib" onClick={() => setPop(pop === "more" ? null : "more")}><Ic n="more" s={15}></Ic></button>
            {pop === "more" && <Pop cls="mn right" onClose={() => setPop(null)}><button className="mn-i" onClick={() => { setPop(null); A.toast("Duplicated as “" + a.n + " copy” · opening it"); }}><Ic n="copy" s={14}></Ic><span className="mn-l"><b>Duplicate</b></span></button><button className="mn-i" onClick={() => { setPop(null); A.toast("Exported " + a.id + ".agent.json"); }}><Ic n="download" s={14}></Ic><span className="mn-l"><b>Export as JSON</b><em>Import it into another organization</em></span></button><button className="mn-i" onClick={() => { setPop(null); A.go("activity"); onClose(); }}><Ic n="timeline" s={14}></Ic><span className="mn-l"><b>Runs and proposals</b></span></button><span className="mn-sep"></span><button className="mn-i" disabled={a.sys} onClick={() => { setPop(null); setRm(true); }}><Ic n={a.sys ? "lock" : "trash"} s={14}></Ic><span className="mn-l"><b className={a.sys ? "" : "t-d"}>{a.sys ? "System agent" : "Remove agent"}</b>{a.sys && <em>Started by Trenova; turn it off instead</em>}</span></button></Pop>}</div>}
          <span className="ab-vr"></span>
        </>}
        <div className="ab-sv"><SaveBar f={f} d={d} fields={fields} create={!a} saveLabel={a ? "Save" : v.mode === "live" ? "Create" : "Create in " + mode[1].toLowerCase()} invalid={!intro && invalid}></SaveBar></div>
        <button className="ib" title="Close (Esc)" onClick={f.tryClose}><Ic n="x" s={14}></Ic></button>
      </header>
      {conf && <ConflictBar who="Sarah Alvarez" when="2 min ago" what={["Raised Release billing hold to Automatic", "Added the role Billing clerk"]} onReview={() => { d.patch({ ceil: "AutoExecute" }); setConf(false); }} onOverwrite={() => setConf(false)}></ConflictBar>}
      {intro ? <CreateIntro S={S} onStart={start}></CreateIntro> : <div className="ab-m">
        <nav className="ab-r">
          <div className="ab-rh">Build</div>
          {RAIL.map(([k, l, sub]) => <button key={k} className={"ri" + (act === k ? " on" : "")} onClick={() => jump(k)}><span className={"ri-s " + status[k]}>{status[k] === "ok" ? <Ic n="check" s={10} w={3}></Ic> : status[k] === "warn" ? "!" : null}</span><span className="ri-t"><b>{l}</b><em>{sub}</em></span></button>)}
          <button className="ri" onClick={() => setTryOpen(true)}><span className={"ri-s " + status.test}>{status.test === "ok" ? <Ic n="check" s={10} w={3}></Ic> : status.test === "warn" ? "!" : null}</span><span className="ri-t"><b>Try it</b><em>{status.test === "ok" ? "Tried with this draft" : status.test === "warn" ? "Draft changed since" : "Not tried yet"}</em></span></button>
          <div className="ab-rf">
            <div className="rdy"><Prog v={ready / req.length}></Prog><span><b>{ready === req.length ? "Ready to go live" : req.length - ready + " left before going live"}</b><em>{v.mode === "live" ? "It's set to go live when you save" : "It stays in " + mode[1].toLowerCase() + " until you switch it"}</em></span></div>
            {a && <div className="rel"><button className="ab-ver" onClick={() => setPop(pop === "hist" ? null : "hist")}><Ic n="timeline" s={12}></Ic><span>v7 · Oct 6 · Sarah Alvarez</span></button>
              {pop === "hist" && <Pop cls="hist" onClose={() => setPop(null)}><div className="mn-h">Versions</div>{VERSIONS.map(([n, who, when, what, cur]) => <div key={n} className={"vh-r" + (cur ? " cur" : "")}><span className="vh-d"></span><div><b>{what}</b><span>{who} · {when}</span></div>{cur ? <span className="tg">Current</span> : <button className="btn sm" onClick={() => { d.patch({ ceil: "ActWithApproval" }); setPop(null); A.toast("Loaded " + n + " into the draft · save to restore it"); }}>Restore</button>}</div>)}</Pop>}</div>}
          </div>
        </nav>
        <div className="ab-c" ref={canvas} onScroll={onScroll}><div className="ab-in">
          {d.base.mode === "shadow" && v.mode === "live" && <div className="srb"><Ic n="eyeOff" s={14}></Ic><div><b>Going live after {(a && { billev: 18, coverage: 31, inbox: 12 }[a.id]) || 24} shadow proposals</b><span>87% matched what people did · 2 would have been rejected · none would have failed</span></div><button className="btn sm" onClick={() => A.go("activity")}>Read them</button></div>}
          <div id="ab-who" className={"hero2" + (fresh.includes("who") ? " fresh" : "")}>
            <div className="rel"><button className="hero2-t" title="Change icon and color" onClick={() => setPop(pop === "look" ? null : "look")}><Tile a={{ ic: v.ic, h: v.h }} s={64}></Tile><span className="hero2-e"><Ic n="edit" s={10}></Ic></span></button>
              {pop === "look" && <Pop cls="look" onClose={() => setPop(null)}><div className="eb-ics">{AG_ICONS.map(ic => <button key={ic} className={v.ic === ic ? "on" : ""} onClick={() => set("ic", ic)}><Ic n={ic} s={14}></Ic></button>)}</div><div className="eb-hs">{AG_HUES.map(h => <button key={h} className={v.h === h ? "on" : ""} style={{ "--h": h }} onClick={() => set("h", h)}></button>)}</div></Pop>}</div>
            <div className="hero2-f"><input className="hero2-n" value={v.n} placeholder="Name your agent" onChange={e => set("n", e.target.value)}></input><input className="hero2-d" value={v.d} placeholder="Say what it does in one line" onChange={e => set("d", e.target.value)}></input></div>
          </div>
          <p className="sent">{sentence}</p>
          <Blk id="instr" fresh={fresh.includes("instr")} t="Instructions" s="How it should think and talk. Write it the way you'd brief a new hire.">
            <div className={"ins" + (tight ? " busy" : "")}>
              <div className="ins-tb">{VARS.map(([k, l]) => <button key={k} className="ins-v" onClick={() => insert(k)} title={"Insert " + k}><Ic n="plus" s={10} w={2.4}></Ic>{l}</button>)}<span className="sp"></span><span className="mono ins-n">~{Math.ceil(v.instr.length / 4)} tokens</span>{ai && <button className="ins-ai" disabled={tight || !v.instr.trim()} onClick={tighten}>{tight ? <i className="spn"></i> : <Ic n="sparkle" s={12}></Ic>}{tight ? "Tightening…" : "Tighten"}</button>}</div>
              <div className="ins-ed"><div className="ins-bk" aria-hidden="true">{hl(v.instr)}{"\n"}</div><textarea ref={ta} value={v.instr} spellCheck="false" placeholder={"You are the detention desk for {{organization}}.\nWhen a truck has waited more than two hours…"} onChange={e => set("instr", e.target.value)}></textarea></div>
            </div>
            {lints.map(([, need, msg]) => <div key={msg} className="lint"><Ic n="warn" s={13}></Ic><span>{msg}</span><button className="btn sm" onClick={() => giveTool(need[0])}><Ic n="plus" s={11}></Ic>Give it {TOOLS.find(t => t.n === need[0]).l}</button></div>)}
            <div className="nv2"><div className="nv2-h"><Ic n="ban" s={13}></Ic><b>Never</b><span>Hard lines it won't cross, whatever it's asked.</span></div>
              {v.guard.map((g, i) => <div key={i} className="nv2-r"><span className="nv2-n mono">{i + 1}</span><input value={g} onChange={e => set("guard", x => x.map((y, j) => j === i ? e.target.value : y))}></input><button className="ib xs" title="Remove" onClick={() => set("guard", x => x.filter((_, j) => j !== i))}><Ic n="x" s={11}></Ic></button></div>)}
              <div className="nv2-r add"><span className="nv2-n"><Ic n="plus" s={11}></Ic></span><input value={gIn} placeholder="Add a line, like “Email a customer outside business hours”" onChange={e => setGIn(e.target.value)} onKeyDown={e => { if (e.key === "Enter" && gIn.trim()) { e.preventDefault(); set("guard", x => [...x, gIn.trim()]); setGIn(""); } }}></input>{gIn.trim() && <span className="kbd">↵</span>}</div>
            </div>
          </Blk>
          <Blk id="trig" fresh={fresh.includes("trig")} t="When it runs" s="What wakes it up, and who's allowed to.">
            <div className="tg3">{TRIGS.map(([k, ic, l, s]) => <button key={k} className={"tgc" + (v.trig === k ? " on" : "")} onClick={() => set("trig", k)}><span className="tgc-i"><Ic n={ic} s={16}></Ic></span><b>{l}</b><span>{s}</span><i className="tgc-ck"><Ic n="check" s={10} w={3}></Ic></i></button>)}</div>
            {v.trig === "Scheduled" && <div className="sch">
              <div className="tks">{CRON.map(c => <button key={c[0]} className={"tkb" + (v.cron === c[0] ? " on first" : "")} onClick={() => set("cron", c[0])}>{c[1]}</button>)}</div>
              <WeekStrip cron={v.cron}></WeekStrip>
              <div className="sch-r"><Txt v={v.cron} on={x => set("cron", x)} mono w={160}></Txt><Sel v={v.tz} on={x => set("tz", x)} opts={TZS}></Sel><span className="sch-h">{cron ? cron[2] + " · next " + cron[3][0] : "Custom · checked when you save"}</span></div>
            </div>}
            {v.trig === "Event" && <div className="sch"><Chips v={v.events} on={x => set("events", x)} opts={EVENTS}></Chips><p className="es-note">Each event starts a run about the record it concerns. Repeats inside five minutes are merged.</p></div>}
            {v.trig === "Continuous" && <div className="sch"><div className="kv2"><div><span>Every</span><div className="tks">{INTERVALS.map(([s2, l]) => <button key={s2} className={"tkb" + (v.interval === s2 ? " on first" : "")} onClick={() => set("interval", s2)}>{l}</button>)}</div></div><div><span>At most</span><div className="stp2"><button onClick={() => set("conc", x => Math.max(1, x - 1))}>−</button><b className="mono">{v.conc}</b><button onClick={() => set("conc", x => Math.min(10, x + 1))}>+</button><em>run{v.conc > 1 ? "s" : ""} at once</em></div></div></div><p className="es-note">About {Math.round(86400 / v.interval * 7).toLocaleString()} checks a week. Each one counts against its runs per day.</p></div>}
            {(v.trig === "Scheduled" || v.trig === "Continuous") && <div className="acc"><div className="acc-l"><b>Stop after</b><span>{v.endsAt ? "Switches itself off on " + v.endsAt : "Optional. It switches itself off after this date."}</span></div><label className="inx" style={{ width: 180 }}><input type="date" value={v.endsAt} onChange={e => set("endsAt", e.target.value)}></input></label></div>}
            <div className="acc"><div className="acc-l"><b>{v.trig === "Chat" ? "Who can ask it" : "Who can run it by hand"}</b><span>{v.access.length ? v.access.length + " roles" : "Everyone who can use the assistant"}</span></div><Seg v={v.access.length ? "roles" : "all"} opts={[["all", "Everyone"], ["roles", "Specific roles"]]} onChange={x => set("access", x === "all" ? [] : ["Owner"])}></Seg></div>
            {v.access.length > 0 && <Chips v={v.access} on={x => set("access", x.length ? x : ["Owner"])} opts={ROLES}></Chips>}
            {openRisk && <Callout tone="w" act={<button className="btn sm" onClick={() => set("access", ["Owner", "Dispatch lead"])}>Limit to roles</button>}>Anyone can ask it, and it holds {outside.length} tool{outside.length > 1 ? "s" : ""} that reach outside the company — {outside.map(t => t.l).slice(0, 2).join(", ")}{outside.length > 2 ? "…" : ""}.</Callout>}
          </Blk>
          <Blk id="tools" fresh={fresh.includes("tools")} t="Tools and autonomy" s="What it can read and change, and how much freedom each change gets.">
            <ToolBench v={v} set={set}></ToolBench>
          </Blk>
          <Blk id="limits" fresh={fresh.includes("limits")} t="Budget and model" s="Where it stops, however well it's going, and how it answers.">
            <div className="gr">{[["budget", "Monthly budget", "$", 5, v.budget === "" || v.budget == null ? "Empty means no cap" : "$" + Math.round(v.budget * 0.41) + " spent this month · across every run", "No cap"], ["runsDay", "Runs per day", "runs", 10, v.runsDay ? "Then it stops for the day and says so" : "Zero means no cap"], ["timeout", "Run timeout", "min", 1, "A run that takes longer is stopped"], ["steps", "Tool calls per run", "calls", 1, "What one run may spend looking things up and acting"]].map(([k, l, u, step, h, ph]) => { const val = k === "timeout" ? Math.round(v.timeout / 60) : v[k]; const put = x => set(k, k === "timeout" ? Math.max(1, x) * 60 : x); return (
              <div key={k} className="gr-c"><span className="gr-l">{l}</span><div className="gr-v">{u === "$" && <span className="gr-u">$</span>}<input className="mono" value={val ?? ""} placeholder={ph || "0"} onChange={e => { const r = e.target.value.replace(/\D/g, ""); put(r === "" && k === "budget" ? "" : +r || 0); }}></input>{u !== "$" && <span className="gr-u">{u}</span>}<span className="gr-st"><button onClick={() => put(Math.max(0, (+val || 0) - step))}>−</button><button onClick={() => put((+val || 0) + step)}>+</button></span></div>{k === "budget" && v.budget !== "" && v.budget != null && <span className="gr-bar"><i style={{ width: "41%" }}></i></span>}<span className="gr-h">{h}</span></div>); })}</div>
            <div className="mdr">
              <div className="mdr-c"><span className="gr-l">Preferred provider</span><Sel v={v.route} on={x => set("route", x)} opts={[["auto", "Automatic"], ...S.providers.map(p => [p.id, p.n + " · " + p.model])]}></Sel><span className="gr-h">Tried first. Automatic follows the routing on the Providers tab.</span></div>
              <div className="mdr-c"><span className="gr-l">Replies as</span><div className="ro">{[["Conversational", "chat", "A conversation", "Answers in prose"], ["Report", "receipt", "A report", "Answers in sections"]].map(([k, ic, l, sub]) => <button key={k} className={"ro-o" + (v.out === k ? " on" : "")} onClick={() => set("out", k)}><Ic n={ic} s={14}></Ic><span><b>{l}</b><em>{sub}</em></span></button>)}</div></div>
            </div>
          </Blk>
          <Blk id="team" t="Memory and handoffs" s="What it keeps, what it's told up front, and who it can ask.">
            <SwRow l="Learns from its work" s="Keeps a lesson when a conversation settles. Lessons shared beyond one person wait for approval." v={v.learn && S.pol.learn} disabled={!S.pol.learn} why="Off for the whole organization" on={x => set("learn", x)}></SwRow>
            <div className="swr"><span><b>Memory in the prompt</b><em>How much remembered context goes in with each question. Empty uses the default.</em></span><label className="inx mono" style={{ width: 170 }}><input value={v.memTok} placeholder="Default" onChange={e => set("memTok", e.target.value.replace(/\D/g, ""))}></input><span className="inx-a">tokens</span></label></div>
            {v.memTok !== "" && (+v.memTok < 1000 || +v.memTok > 16000) && <p className="f-h t-w">Between 1,000 and 16,000 tokens.</p>}
            <div className="tm-s"><div className="tm-h"><b>Tell it about</b><span>{v.ctx.length ? v.ctx.length + " of " + CTX.length : "Everything — leave all unchecked to include all of it"}</span></div><Chips v={v.ctx} on={x => set("ctx", x)} opts={CTX}></Chips></div>
            <div className="tm-s"><div className="tm-h"><b>Can ask</b><span>{v.trig !== "Chat" ? "Only agents people talk to can ask others" : v.handoff.length + " of 8 · it passes the conversation along with what it found"}</span></div>
              {v.trig === "Chat" ? <div className="hof">{S.agents.filter(x => !a || x.id !== a.id).map(x => { const on = v.handoff.includes(x.id), full = !on && v.handoff.length >= 8; return <button key={x.id} disabled={full} className={"hof-i" + (on ? " on" : "")} onClick={() => set("handoff", h => on ? h.filter(y => y !== x.id) : [...h, x.id])}><Tile a={x} s={24}></Tile><span>{x.n}</span><i><Ic n="check" s={9} w={3}></Ic></i></button>; })}</div> : <p className="es-note">Switch it to "Someone asks" to let it hand work to other agents.</p>}</div>
          </Blk>
          {a && <Blk id="record" t="How it's doing" s="The last 30 days.">
            <div className="sc">{[["Runs", (a.runs || []).reduce((t, x) => t + x, 0) * 2], ["Approved", a.rec ? Math.round(a.rec.ap / (a.rec.ap + a.rec.ch + a.rec.rj) * 100) + "%" : "—"], ["Cost", "$" + (a.id.length * 3.1).toFixed(2)], ["Time saved", a.rec ? Math.round(a.rec.ap * 0.3) + " h" : "—"]].map(([l, x]) => <div key={l}><span>{l}</span><b className="mono">{x}</b></div>)}</div>
            {a.rec && <div className="streak"><div className="streak-d">{Array.from({ length: 10 }).map((_, i) => <i key={i} className={i < a.rec.streak ? "on" : ""}></i>)}</div><span>{a.rec.streak} clean approvals in a row on <b>{a.rec.tool}</b>{S.pol.earned ? "" : " · earned autonomy is off for the organization"}</span></div>}
          </Blk>}
        </div></div>
        {tryOpen && <TryPanel a={a} v={v} S={S} onTested={setTested} onClose={() => setTryOpen(false)}></TryPanel>}
      </div>}
    </div>
    {rm && <ConfirmDialog title={"Remove " + a.n + "?"} body="Its runs and audit trail are kept. Open proposals are withdrawn and people can no longer pick it in Desk." typed={a.n} danger confirm="Remove agent" onConfirm={() => { A.removeAgent(a); onClose(); }} onClose={() => setRm(false)}></ConfirmDialog>}
  </>, document.querySelector(".tv"));
}
Object.assign(window, { AgentBuilder, AgentEditor: AgentBuilder });
