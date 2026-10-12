function Top({ onToggleSide }) {
  return (
    <header className="top">
      <button className="ib" onClick={onToggleSide} title="Toggle folders"><Ic n="sidebar" s={15}></Ic></button>
      <button className="org"><img src="logo.png" alt=""></img><b>Trenova Logistics</b><Ic n="upDown" s={12}></Ic></button>
      <span className="vr"></span>
      <button className="mod"><Ic n="filter" s={14}></Ic>Triage<Ic n="chevD" s={12}></Ic></button>
      <span className="sp"></span>
      <button className="jump"><Ic n="search" s={13}></Ic><span>Search or jump to…</span><span className="kbd">Ctrl K</span></button>
      <span className="sp"></span>
      <button className="ib bell"><Ic n="bell" s={15}></Ic><i></i></button>
      <span className="me">SA</span>
    </header>
  );
}
const matchQ = (m, q) => !q || (senderOf(m) + m.from[1] + m.subject + (m.shp ? SHP[m.shp].pro : "") + (m.cust || "")).toLowerCase().includes(q.toLowerCase());
function inFolder(m, f, views) {
  if (f.t === "lane") return LANES.find(x => x[0] === f.v)[2](m);
  if (f.t === "kind") return isNeed(m) && !m.snooze && !m.await && m.kind === f.v;
  if (f.t === "mb") return m.mb === f.v;
  if (f.t === "view") { const v = (views || []).find(x => x.id === f.v); return !!v && matchQ(m, v.q); }
  return false;
}
const sameF = (a, b) => a.t === b.t && a.v === b.v;
function folderName(f, views) { return f.t === "lane" ? LANES.find(x => x[0] === f.v)[1] : f.t === "kind" ? KIND[f.v].l : f.t === "rules" ? "Rules" : f.t === "brief" ? "Briefing" : f.t === "view" ? ((views || []).find(x => x.id === f.v) || { name: "View" }).name : MAILBOXES.find(x => x.id === f.v).name; }

function Side({ msgs, folder, setFolder, onKeys, rules, views, setViews, noMb, onConnect, onFCtx, renaming, setRenaming }) {
  const waiting = msgs.filter(m => isNeed(m) && !m.snooze && !m.await);
  const kinds = KINDS.filter(([k]) => waiting.some(m => m.kind === k));
  return (
    <nav className="side"><div className="side-in">
      <div className="side-h"><h1>Triage</h1></div>
      <button className={"nv" + (folder.t === "brief" ? " on" : "")} onClick={() => setFolder({ t: "brief", v: "" })}><span className="dm md"></span><span>Briefing</span></button>
      {LANES.map(([k, l, p, ic]) => { const n = msgs.filter(p).length; if ((k === "mine" || k === "snoozed" || k === "them") && !n) return null; return (
        <button key={k} className={"nv" + (sameF(folder, { t: "lane", v: k }) ? " on" : "")} onClick={() => setFolder({ t: "lane", v: k })}><Ic n={ic} s={14}></Ic><span>{l}</span>{k === "waiting" ? (n ? <em className="w">{n}</em> : null) : k === "mine" || k === "snoozed" || k === "them" ? <em>{n}</em> : null}</button>); })}
      {kinds.length > 0 && <div className="side-gh">Waiting</div>}
      {kinds.map(([k, l, h]) => (
        <button key={k} className={"nv" + (sameF(folder, { t: "kind", v: k }) ? " on" : "")} onClick={() => setFolder({ t: "kind", v: k })} onContextMenu={e => onFCtx(e, { t: "kind", v: k })}><i className="dot" style={{ background: h == null ? "var(--faint)" : `oklch(var(--kl) 0.13 ${h})` }}></i><span>{l}</span><em>{waiting.filter(m => m.kind === k).length}</em></button>))}
      {views.length > 0 && <div className="side-gh">Views</div>}
      {views.map(v => (
        <div key={v.id} className={"nv vw" + (sameF(folder, { t: "view", v: v.id }) ? " on" : "")} role="button" onClick={() => setFolder({ t: "view", v: v.id })} onContextMenu={e => onFCtx(e, { t: "view", v: v.id })}><Ic n="search" s={13}></Ic>{renaming === v.id ? <input className="vw-in" autoFocus defaultValue={v.name} onClick={e => e.stopPropagation()} onBlur={e => { const n = e.target.value.trim(); if (n) setViews(vs => vs.map(x => x.id === v.id ? { ...x, name: n } : x)); setRenaming(null); }} onKeyDown={e => { if (e.key === "Enter") e.target.blur(); if (e.key === "Escape") setRenaming(null); }}></input> : <span>{v.name}</span>}<em>{msgs.filter(m => matchQ(m, v.q) && isNeed(m)).length || ""}</em><button className="vw-x" title="Remove view" onClick={e => { e.stopPropagation(); setViews(vs => vs.filter(x => x.id !== v.id)); }}><Ic n="x" s={10}></Ic></button></div>))}
      <div className="side-gh">Sources{!noMb && <button onClick={onConnect}>Add</button>}</div>
      {noMb ? <button className="nv add" onClick={onConnect}><Ic n="plus" s={13}></Ic><span>Add a source</span></button> : MAILBOXES.map(b => (
        <button key={b.id} className={"nv" + (sameF(folder, { t: "mb", v: b.id }) ? " on" : "")} onClick={() => setFolder({ t: "mb", v: b.id })} onContextMenu={e => onFCtx(e, { t: "mb", v: b.id })} title={b.addr}><Ic n="inbox" s={13}></Ic><span>{b.name}</span>{!b.ok && <i className="st-d off"></i>}</button>
      ))}
      <div className="side-gh">Automation</div>
      <button className={"nv" + (folder.t === "rules" ? " on" : "")} onClick={() => setFolder({ t: "rules", v: "" })}><Ic n="bolt" s={14}></Ic><span>Rules</span><em>{rules.filter(r => r.on).length}</em></button>
      <span className="sp"></span>
      <button className="side-k" onClick={onKeys}><span className="kbd">?</span>Shortcuts</button>
    </div></nav>
  );
}

function Row({ m, n, on, out, sel, selecting, mode, onOpen, onQuick, onSel, onCtx }) {
  const need = isNeed(m), st = nextStep(m), c = stepCopy(st, m), k = KIND[m.kind];
  const today = dayOf(m.min) === "Today";
  const A = need ? actsOf(m) : [], nx = A.find(a => !a.done), left = A.filter(a => !a.done).length;
  const due = dueOf(m);
  const line = m.unread ? "Not read — Nova is off" : m.status === "Quarantined" ? "Held back — looks like phishing" : m.await ? "Waiting on " + senderOf(m).split(" ")[0] + " to reply" : m.snooze ? "Snoozed · " + m.snooze : nx ? nx.t + (left > 1 ? " · +" + (left - 1) : "") : need && A.length ? "All done · mark handled" : c && c.short;
  const [dx, setDx] = React.useState(0);
  const drag = React.useRef(null);
  const down = e => { if (!need || selecting || e.button !== 0) return; drag.current = { x: e.clientX, y: e.clientY, on: false }; };
  const move = e => { const d = drag.current; if (!d) return; const ddx = e.clientX - d.x; if (!d.on && Math.abs(ddx) > 8 && Math.abs(ddx) > Math.abs(e.clientY - d.y)) { d.on = true; e.currentTarget.setPointerCapture(e.pointerId); } if (d.on) setDx(Math.max(-140, Math.min(140, ddx))); };
  const up = () => { const d = drag.current; drag.current = null; if (!d || !d.on) return; if (dx < -90) onQuick([m.id], "Actioned"); else if (dx > 90) onQuick([m.id], "snooze"); setDx(0); d.swiped = true; window.__swiped = Date.now(); };
  return (
    <div className={"row-w" + (dx < -90 ? " sw-l" : dx > 90 ? " sw-r" : "")}>
      {dx !== 0 && <><span className="sw-bg l"><Ic n="clock" s={14}></Ic>Snooze</span><span className="sw-bg r">Handled<Ic n="check" s={14} w={2.2}></Ic></span></>}
      <div role="button" tabIndex={-1} onContextMenu={e => onCtx && onCtx(e, m)} data-new={m.isNew ? "1" : null} style={dx ? { transform: `translateX(${dx}px)`, transition: "none" } : null} onPointerDown={down} onPointerMove={move} onPointerUp={up} onPointerCancel={up}
        className={"row" + (m.isNew ? " nw" : "") + (on ? " on" : "") + (need ? " need" : "") + (out ? " out" : "") + (sel ? " picked" : "") + (selecting ? " selm" : "")} onClick={e => { if (window.__swiped && Date.now() - window.__swiped < 300) return; (e.shiftKey || e.metaKey || e.ctrlKey || selecting) ? onSel(m.id) : onOpen(m.id); }}>
        <span className="r-f"><span className="r-ck" onClick={e => { e.stopPropagation(); onSel(m.id); }}><i className={"kdot" + (k.h == null ? " n" : "")} style={{ "--h": k.h }} title={k.l}></i><span className="cbx2">{sel && <Ic n="check" s={9} w={3}></Ic>}</span></span><b>{senderOf(m)}</b>{n > 1 && <span className="r-th mono">{n}</span>}{m.atts.length > 0 && <Ic n="clip" s={11}></Ic>}{VIEWING[m.id] && <span className="r-pr" title={VIEWING[m.id][0] + " is " + VIEWING[m.id][1]}>{initials(VIEWING[m.id][0])}<i></i></span>}{m.owner && need && <span className="r-ow" title={"Assigned to " + m.owner}>{initials(m.owner)}</span>}</span>
        <span className={"r-tm" + (need && !m.snooze && !m.await ? (due < 0 ? " d" : due < 30 ? " w" : "") : "")} title={need ? "Response target " + Math.round(SLA[m.kind] / 60) + "h for " + k.l.toLowerCase() + "s" : ""}>{m.snooze ? <Ic n="clock" s={11}></Ic> : m.await ? <Ic n="send" s={11}></Ic> : need ? dueTxt(due) : today ? hhmm(at(m.min)) : at(m.min).toLocaleDateString("en-US", { month: "short", day: "numeric" })}</span>
        <span className="r-s">{m.subject || "(no subject)"}</span>
        <span className="r-x">{mode === "next" && need && line ? <span className="r-nx"><span>{line}</span></span> : <span className="r-p">{(m.body[0] || "").replace(/\{\{([^|}]+)\|[^}]+\}\}/g, "$1")}</span>}</span>
      </div>
    </div>
  );
}
function Skel() { return <div className="skel">{[0, 1, 2, 3, 4, 5].map(i => <div key={i} className="sk-r" style={{ animationDelay: i * 60 + "ms" }}><i style={{ width: 90 + (i * 37) % 70 }}></i><i style={{ width: "78%" }}></i><i style={{ width: 120 + (i * 53) % 90 }}></i></div>)}</div>; }

function Health({ mb, onFix }) {
  const H = MB_HEALTH[mb.id], [st, setSt] = React.useState(null);
  if (!H || mb.ok) return null;
  return (
    <div className="hb">
      <div className="hb-h"><Ic n="warn" s={13}></Ic><b>{H.t}</b></div>
      <p>{H.d}</p>
      {st === "copied" && <p className="hb-s">Ask Harborline to forward to <span className="mono">{H.addr}</span>, then send a test.</p>}
      <div className="hb-a">
        <button className="btn sm ink" disabled={st === "sending"} onClick={() => { setSt("sending"); setTimeout(() => { setSt(null); onFix(mb.id); }, 1600); }}>{st === "sending" ? <span className="shim">Waiting for it to arrive…</span> : "Send a test email"}</button>
        <button className="btn sm ghost" onClick={() => setSt("copied")}><Ic n="copy" s={12}></Ic>Copy address</button>
      </div>
    </div>
  );
}

function DeskOff({ onOn, n }) {
  const [s, setS] = React.useState(0);
  return <div className="hb off"><div className="hb-h"><b>Nova is off</b></div><p>Messages still arrive, but nothing reads them — no kinds, no matches, no suggested actions. Everything waits for a person.</p><div className="hb-a"><button className="btn sm ink" disabled={s} onClick={() => { setS(1); setTimeout(onOn, 1600); }}>{s ? <span className="shim">{"Reading " + n + " emails…"}</span> : "Turn on Nova"}</button><button className="btn sm ghost">AI control</button></div></div>;
}

function List({ onRowCtx, deskOff, onDeskOn, onConnect, list, threads, folder, views, openId, onOpen, onQuick, q, setQ, searchRef, mode, leaving, total, sel, setSel, sort, setSort, loading, onSaveView, onFix, demo, onTest }) {
  const groups = [];
  const due = sort === "due" && folder.t !== "rules";
  if (due) { const o = list.filter(m => isNeed(m) && dueOf(m) < 0), s = list.filter(m => isNeed(m) && dueOf(m) >= 0 && dueOf(m) < 60), r = list.filter(m => !o.includes(m) && !s.includes(m)); [["Over target", o], ["Due within the hour", s], ["Later", r]].forEach(g => g[1].length && groups.push(g)); }
  else list.forEach(m => { const d = dayOf(m.min); const g = groups.find(x => x[0] === d); g ? g[1].push(m) : groups.push([d, [m]]); });
  const tog = id => setSel(xs => xs.includes(id) ? xs.filter(x => x !== id) : [...xs, id]);
  const selecting = sel.length > 0, selNeed = list.filter(m => sel.includes(m.id) && isNeed(m)).map(m => m.id);
  const mb = folder.t === "mb" && MAILBOXES.find(b => b.id === folder.v);
  const needLane = list.some(isNeed);
  return (
    <section className="ls">
      <header className="ls-h">
        <label className="srch"><Ic n="search" s={13}></Ic><input ref={searchRef} value={q} onChange={e => setQ(e.target.value)} onKeyDown={e => e.key === "Escape" && (setQ(""), e.target.blur())} placeholder={"Search " + folderName(folder, views).toLowerCase()}></input>{q ? <><button className="sv" title="Save as a view" onClick={() => onSaveView(q)}>Save</button><button className="ib" style={{ width: 22, height: 22 }} onClick={() => setQ("")}><Ic n="x" s={11}></Ic></button></> : <span className="kbd">/</span>}</label>
        {needLane && <div className="srt">{[["due", "Due first"], ["new", "Newest"]].map(([k, l]) => <button key={k} className={sort === k ? "on" : ""} onClick={() => setSort(k)}>{l}</button>)}</div>}
      </header>
      {mb && <Health mb={mb} onFix={onFix}></Health>}
      {deskOff && <DeskOff onOn={onDeskOn} n={list.length}></DeskOff>}
      <div className="ls-s">
        {loading ? <Skel></Skel> : demo === "no mailbox" ? <div className="ls-e set"><span className="set-ic"><Ic n="inbox" s={18}></Ic></span><b>Connect your first source</b>Start with a shared address — dispatch@, pod@, ap@ — and Nova reads everything that lands there. EDI, portals and texts plug in the same way.<ol className="set-s"><li><span className="mono">1</span>Pick an address to watch</li><li><span className="mono">2</span>Forward it to Trenova</li><li><span className="mono">3</span>Send a test email</li></ol><button className="btn ink sm" onClick={onConnect}>Add a source</button></div>
          : list.length === 0 ? (
          q ? <div className="ls-e"><b>Nothing matches “{q}”</b>Search covers the sender, subject, customer and PRO across every source.</div>
            : demo === "first day" ? <div className="ls-e set"><span className="set-ic"><Ic n="send" s={16}></Ic></span><b>Waiting for your first message</b>Forward anything to <span className="mono">intake@dev.trenova.app</span> and it shows up here, already read.<button className="btn sm" onClick={onTest}>Send a test email</button></div>
            : folder.v === "waiting" ? <div className="ls-e"><b>All caught up</b>{"Nova handled " + total.handled + " on its own."}</div>
              : folder.v === "them" ? <div className="ls-e"><b>Not waiting on anyone</b>After you reply, choose “Wait for reply” and it parks here until they answer.</div>
                : <div className="ls-e"><b>Nothing here</b>Messages appear here as they arrive from a connected source.</div>
        ) : groups.map(([d, ms]) => <React.Fragment key={d}>
          <div className={"dg" + (d === "Over target" ? " d" : "")}>{d}</div>
          {ms.map(m => <Row key={m.id} m={m} n={threads[m.th || m.id] || 1} on={m.id === openId} out={leaving.includes(m.id)} sel={sel.includes(m.id)} selecting={selecting} mode={mode} onOpen={onOpen} onQuick={onQuick} onSel={tog} onCtx={onRowCtx}></Row>)}
        </React.Fragment>)}
      </div>
      {selecting && <div className="bulk">
        <button className="ib" title="Clear" onClick={() => setSel([])}><Ic n="x" s={13}></Ic></button>
        <b>{sel.length} selected</b>
        <span className="sp"></span>
        <button className="ib" title="Select all" onClick={() => setSel(list.map(m => m.id))}><Ic n="layers" s={13}></Ic></button>
        <button className="ib" title="Snooze until tomorrow" disabled={!selNeed.length} onClick={() => { onQuick(selNeed, "snooze"); setSel([]); }}><Ic n="clock" s={13}></Ic></button>
        <button className="ib" title="Assign to me" disabled={!selNeed.length} onClick={() => { onQuick(selNeed, "mine"); setSel([]); }}><Ic n="user" s={13}></Ic></button>
        <button className="ib" title="Ignore" disabled={!selNeed.length} onClick={() => { onQuick(selNeed, "Ignored"); setSel([]); }}><Ic n="archive" s={13}></Ic></button>
        <button className="btn sm ink" disabled={!selNeed.length} onClick={() => { onQuick(selNeed, "Actioned"); setSel([]); }}><Ic n="check" s={12} w={2.2}></Ic>Handled</button>
      </div>}
    </section>
  );
}

const hashOf = s => [...s].reduce((h, c) => (h * 31 + c.charCodeAt(0)) >>> 0, 7);
const ACT_SAY = { "File on the matched shipment": "file it on the shipment", "File and queue for carrier pay": "file it and queue it for pay", "Ignore": "ignore it", "Draft a reply for review": "draft a reply for you to check", "Create the load as a draft": "draft the load", "Open the shipment": "open the shipment", "Link by sender": "link it by sender", "Assign to Dispatch": "hand it to Dispatch", "Assign to Billing": "hand it to Billing" };
const ACTS = Object.keys(ACT_SAY);
const art = l => /^[aeiou]/i.test(l) ? "an" : "a";
const SAMPLE = { ProofOfDelivery: ["IMG_4402 → S-104219", "POD_77114 → S-104207", "IMG_4398 → S-104201"], Invoice: ["KF-5518 → carrier pay", "INV-8841 → carrier pay", "BR-2210 → carrier pay"], Other: ["FreightWeekly · ignored", "DAT Trendlines · ignored", "Webinar invite · ignored"], StatusRequest: ["PO 88190 · reply drafted", "PO 88174 · reply drafted"], RateConfirmation: ["RateCon S-104230 → filed"], Tender: ["Tender 4460 → draft load"], DetentionDispute: [] };
function Blank({ v, set, opts, ph, show }) {
  const cur = opts.find(o => o[0] === v);
  return <label className={"rw-b" + (cur ? "" : " empty")}><select value={v || ""} onChange={e => set(e.target.value)}>{!cur && <option value="">{ph}</option>}{opts.map(([k, l]) => <option key={k} value={k}>{l}</option>)}</select><span>{cur ? (show ? show(cur) : cur[1]) : ph}</span></label>;
}
function Flow({ r, on }) {
  const s = SAMPLE[r.kind] || [];
  return <div className="rw-fl"><span className="rw-n">{!on ? "Paused" : r.n ? <><b>{r.n}</b> this month</> : "Waiting for a first match"}</span>{on && s.length > 0 && <span className="rw-ex">{"Last: " + s[0]}</span>}</div>;
}
function RulesView({ rules, setRules, act, onSee }) {
  const [draft, setDraft] = React.useState(null);
  const [gone, setGone] = React.useState(null);
  const on = rules.filter(r => r.on), total = on.reduce((s, r) => s + r.n, 0);
  const kOpts = KINDS.map(([k, l]) => [k, l.toLowerCase()]), mOpts = MAILBOXES.map(x => [x.id, x.name]), aOpts = ACTS.map(x => [x, ACT_SAY[x]]);
  const fromOpts = [...new Set(["any sender", ...rules.map(r => r.from), ...MESSAGES.map(m => m.from[1].split("@")[1])])].map(x => [x, x === "any sender" ? "anyone" : x]);
  const ready = draft && draft.kind && draft.from && draft.act;
  const pv = ready ? previewRule(draft) : null;
  const shown = useCountR(pv ? pv.n : 0);
  const set = (id, k, v) => setRules(rs => rs.map(x => x.id === id ? { ...x, [k]: v } : x));
  const say = (r, upd, isNew) => (
    <p className="rw-s">
      {"When " + (r.kind ? art(KIND[r.kind].l) : "a") + " "}<Blank v={r.kind} set={v => upd("kind", v)} opts={kOpts} ph="kind of message"></Blank>
      {" arrives from "}<Blank v={r.from} set={v => upd("from", v)} opts={fromOpts} ph="someone"></Blank>
      {" on "}<Blank v={r.mb} set={v => upd("mb", v)} opts={mOpts} ph="a source"></Blank>
      {", Nova should "}<Blank v={r.act} set={v => upd("act", v)} opts={aOpts} ph="do something"></Blank>{"."}
    </p>
  );
  return (
    <section className="rules rw">
      <div className="rw-in">
        <div className="rw-top"><h2>Rules</h2><span className="mono">{on.length + " on · " + total + " handled this month"}</span></div>

        {draft ? <div className={"rw-r new" + (ready ? " ready" : "")}>
          <div className="rw-l"><span className="rw-plus"><Ic n="plus" s={14}></Ic></span></div>
          <div className="rw-m">
            {say(draft, (k, v) => setDraft(d => ({ ...d, [k]: v })), true)}
            <div className="rw-pv">{ready ? <><b className="mono">{shown}</b><span>{" messages last month would have matched"}{pv.wrong ? <> · <em>{pv.wrong} look off</em></> : " · all look right"}</span><button className="lnk" onClick={() => onSee(draft)}>See them</button></> : <span className="rw-hint">Fill in the blanks — it reads like a sentence because it is one.</span>}<span className="sp"></span><button className="btn ghost sm" onClick={() => setDraft(null)}>Cancel</button><button className="btn ink sm" disabled={!ready} onClick={() => { const id = "r" + Date.now(); setRules(rs => [{ id, ...draft, n: 0, on: true, by: ME, fresh: true }, ...rs]); setDraft(null); act("Rule on — Nova starts with the next match"); }}>Turn on</button></div>
          </div>
        </div> : <button className="rw-add" onClick={() => setDraft({ kind: "", from: "any sender", mb: "mb1", act: "" })}><span className="rw-plus"><Ic n="plus" s={14}></Ic></span><span>{"When "}<u>something</u>{" arrives, Nova should "}<u>do something</u>{"…"}</span><em>New rule</em></button>}

        <div className="rw-list">{rules.map((r, i) => (
          <div key={r.id} className={"rw-r" + (r.on ? "" : " off") + (r.fresh ? " fresh" : "") + (gone === r.id ? " gone" : "")} style={{ animationDelay: i * 60 + "ms", "--h": KIND[r.kind].h == null ? 260 : KIND[r.kind].h }}>
            <div className="rw-l"><button role="switch" aria-checked={r.on} className={"rw-sw" + (r.on ? " on" : "")} onClick={() => { set(r.id, "on", !r.on); act(r.on ? "Paused — matches wait for a person" : "Back on"); }}><i></i></button></div>
            <div className="rw-m">
              {say(r, (k, v) => set(r.id, k, v))}
              <Flow r={r} on={r.on}></Flow>
            </div>
            <div className="rw-x">
              <button className="ib" title="See what it matched" onClick={() => onSee(r)}><Ic n="eye" s={14}></Ic></button>
              <button className="ib" title="Delete" onClick={() => { const prev = rules; setGone(r.id); setTimeout(() => { setRules(rs => rs.filter(x => x.id !== r.id)); setGone(null); act("Rule deleted", () => setRules(prev)); }, 280); }}><Ic n="x" s={14}></Ic></button>
            </div>
          </div>))}</div>
      </div>
    </section>
  );
}
function useCountR(to) { const [v, setV] = React.useState(0); const f = React.useRef(0); React.useEffect(() => { const a = f.current, t0 = performance.now(); let raf; const tick = now => { const k = Math.min(1, (now - t0) / 700), e = 1 - Math.pow(1 - k, 3); setV(Math.round(a + (to - a) * e)); if (k < 1) raf = requestAnimationFrame(tick); else f.current = to; }; raf = requestAnimationFrame(tick); return () => cancelAnimationFrame(raf); }, [to]); return v; }
Object.assign(window, { Top, Side, List, RulesView, inFolder, sameF, folderName, matchQ, ACT_SAY });
