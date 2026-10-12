/* What an email asks for (actions) and what it tells you (info). Both are derived from the email's state, so they can't drift from it. */
function actsOf(m) {
  if (m.unread) return [];
  const s = m.shp && SHP[m.shp], f = m.flags || {}, R = m.made.reply, L = m.made.load;
  const A = [];
  const file = a => A.push({ id: "file:" + a.n, k: "file", a, t: "File " + a.n.replace(/\.(pdf|jpg)$/i, ""), sub: a.filed ? "Filed on " + SHP[a.filed].pro + " as " + (a.ftype || (DOCREAD[a.doc] || {}).type || "document").toLowerCase() : "as " + ((DOCREAD[a.doc] || {}).type || "document").toLowerCase() + (s ? " on " + s.pro : ""), done: !!a.filed });
  const reply = (t, sub, desk) => A.push({ id: "reply", k: "reply", t, sub: R && R.sent ? "Sent " + R.sentAt : R ? (R.streaming ? "Drafting…" : "Draft ready") : sub, desk: desk && !R, done: !!(R && R.sent) });
  if (A.length && false) {}
  const link = () => A.push({ id: "link", k: "link", t: "Say which record it's about", sub: f.linked ? "Linked to " + (s ? s.pro : m.cust) : "Nova couldn't tell", done: !!f.linked });
  switch (m.id) {
    case "m1": A.push({ id: "load", k: "load", t: "Build the load", sub: L && L.pro ? L.pro + " created" : L ? (L.streaming ? "Drafting…" : "Draft ready to check") : "From the email text", desk: !L, done: !!(L && L.pro) }); reply("Confirm you'll take it", "Northwind is waiting on a yes"); break;
    case "m2": A.push({ id: "stops", k: "stops", t: "Check the stop times", sub: f.stops ? f.stops : "ELD says 40 min late", done: !!f.stops }); reply("Answer Harborline", "Waive it or send the log"); break;
    case "m3": link(); break;
    case "m4": A.push({ id: "block", k: "block", t: "Block the sender", sub: f.blocked ? "Blocked on Intake" : "Sign-in link to a fake domain", done: !!f.blocked }); break;
    case "m5": m.atts.filter(a => !a.fail).forEach(file); reply("Return it signed", "Within 2 hours to hold the truck"); break;
    case "m6": m.atts.filter(a => !a.fail).forEach(file); A.push({ id: "osd", k: "osd", t: "Log 2 pallets short", sub: f.osd ? "Exception on " + s.pro : "Noted on the BOL", done: !!f.osd }); reply("Ask Luis to resend IMG_4411", "Over the 25 MB limit"); break;
    case "m7": reply("Send Crestline the ETA", "Receiving closes at 17:00", true); break;
    case "m8": m.atts.filter(a => !a.fail).forEach(file); A.push({ id: "load", k: "load", t: "Create the 3 loads", sub: L && L.pro ? L.pro + " created" : L ? "Draft ready to check" : "Read from tender_4471.pdf", done: !!(L && L.pro) }); reply("Accept the tender", "Due Thu Oct 8"); break;
    default: m.atts.filter(a => !a.fail).forEach(file);
  }
  return A;
}
function infoOf(m) {
  if (m.unread) return [];
  const s = m.shp && SHP[m.shp], x = m.shp && RECX[m.shp], I = [];
  if (s) I.push({ id: "shp", ic: "truck", l: s.pro, v: s.o.split(",")[0] + " → " + s.d.split(",")[0], meta: x.eta, w: x.late, more: [["Driver", x.drv], ["Equipment", x.eq], [s.bol.split(" ")[0], s.bol.split(" ")[1]], ["PO", s.po ? s.po.replace("PO ", "") : "—"], ["On file", x.docs.join(", ")]] });
  if (m.cust) I.push({ id: "cust", ic: "building", l: m.cust, v: "Customer", meta: m.reason && /since/.test(m.reason) ? "Since 2023" : null });
  const found = { m1: [["Pickup", "Thu Oct 8, 08:00"], ["Weight", "42,000 lb"], ["Equipment", "Dry van"], ["Origin", "Chicago DC"]], m2: [["Disputed", "$185.00"], ["Invoice", "INV-20931"], ["Stop", "Oct 3"]], m7: [["PO", "88213"], ["Cutoff", "17:00 today"]], m3: [["File", "214 · 0441"], ["City", "Columbus OH"]] }[m.id];
  const doc = m.atts.find(a => a.doc && DOCREAD[a.doc]);
  const rows = found || (doc ? DOCREAD[doc.doc].fields.slice(0, 4).map(([k, v, t]) => [k, v, t]) : null);
  if (rows) I.push({ id: "found", ic: "mark", l: "Read from " + (found ? "the message" : doc.n), facts: rows });
  return I;
}

function Composer({ m, api, onDone }) {
  const R = m.made.reply, mb = MAILBOXES.find(b => b.id === m.mb);
  const [body, setBody] = React.useState(R ? R.body : "");
  React.useEffect(() => { if (R) setBody(R.body); }, [R && R.body, R && R.streaming]);
  if (!R) return null;
  if (R.sent) return <div className="ax-res"><p className="ax-q">{R.body}</p></div>;
  const err = R.err, who = VIEWING[m.id], tpls = window.__tpls || TEMPLATES0;
  const [saved, setSaved] = React.useState(false);
  return (
    <div className="ax-x">
      {who && who[1] === "writing a reply" && <div className="pres-w"><span className="pres-av"><Av n={who[0]} s={18}></Av><i></i></span><span><b>{who[0].split(" ")[0]}</b> is writing a reply too</span><span className="pres-dots"><i></i><i></i><i></i></span></div>}
      {R.streaming ? <p className="ax-q shim">Looking it up and drafting…</p> : <textarea className="ax-ta" autoFocus value={body} onChange={e => setBody(e.target.value)} placeholder={"Write to " + m.from[1]}></textarea>}
      {R.att && <span className="dchip"><Ic n="clip" s={10}></Ic>{R.att}</span>}
      {err && <div className="ax-err"><Ic n="warn" s={13}></Ic><span><b>Didn't send.</b>{" " + err}</span></div>}
      {err && <div className="ax-f"><span className="ax-n">{"To " + R.alt}</span><span className="sp"></span><button className="btn sm ab-go" onClick={() => api.sendReply(R.subject, body, R.alt)}>Send to {R.alt.split("@")[0]}@ instead</button></div>}
      <div className="ax-f" style={err ? { display: "none" } : null}><span className="ax-n">{"From " + mb.addr + (R.to ? " to " + R.to : "")}</span><span className="sp"></span>{!R.streaming && <Pop align="left tpm" btn={(o, t) => <button className={"btn ghost sm" + (o ? " on" : "")} onClick={t}><Ic n="layers" s={12}></Ic>Templates</button>}><div className="mn-h">Fill in from this message</div>{tpls.map(t => <button key={t.id} className="mn-i tp" onClick={() => setBody(fillT(t.b, m))}><span>{t.n}<em>{fillT(t.b, m).split("\n")[0].slice(0, 46) + "…"}</em></span></button>)}<span className="mn-sep"></span><button className="mn-i" disabled={!body.trim()} onClick={() => { window.__tpls = [...tpls, { id: "t" + Date.now(), n: "Saved · " + (m.subject || "reply").slice(0, 22), b: body }]; setSaved(true); }}><Ic n="plus" s={12}></Ic><span>{saved ? "Saved for the team" : "Save this as a template"}</span></button></Pop>}{!R.desk && !R.streaming && DRAFTS[m.id] && DRAFTS[m.id].text && <button className="btn ghost sm" onClick={api.deskReply}>Have Nova draft it</button>}<button className="btn ink sm ab-go" disabled={!body.trim() || R.streaming} onClick={() => api.sendReply(R.subject, body)}><Ic n="send" s={11}></Ic>Send</button></div>
    </div>
  );
}
function LoadForm({ m, api }) {
  const L = m.made.load;
  const [f, setF] = React.useState(() => Object.fromEntries(L.rows.map(([k, v]) => [k, v || ""])));
  React.useEffect(() => setF(Object.fromEntries(L.rows.map(([k, v]) => [k, v || ""]))), [L.shown]);
  if (L.pro) return <div className="ax-res"><dl className="ax-kv">{L.rows.slice(0, 4).map(([k]) => <div key={k}><dt>{k}</dt><dd>{L.vals[k] || "—"}</dd></div>)}</dl><button className="lnk" onClick={() => api.act("Opening " + L.pro + " on the board")}>Open {L.pro}<Ic n="arrowR" s={11}></Ic></button></div>;
  const miss = L.rows.filter(([k]) => !f[k]).length;
  return (
    <div className="ax-x">
      <div className="aform">{L.rows.slice(0, L.shown == null ? L.rows.length : L.shown).map(([k, , ms]) => <label key={k} className={"af" + (!f[k] ? " miss" : "")}><span>{k}</span><input value={f[k] || ""} onChange={e => setF(o => ({ ...o, [k]: e.target.value }))} placeholder={ms || ""}></input></label>)}</div>
      {!L.streaming && <div className="ax-f"><span className={"ax-n" + (miss ? " w" : "")}>{miss ? miss + " empty — fine for a draft" : "All filled in"}</span><span className="sp"></span><button className="btn ghost sm" onClick={api.discardLoad}>Discard</button><button className="btn ink sm ab-go" onClick={() => api.createLoad(f)}>Create</button></div>}
    </div>
  );
}
function FileRow({ m, a, api }) {
  const r = DOCREAD[a.doc] || {};
  const [type, setType] = React.useState(r.type || "Other");
  const [shp, setShp] = React.useState(m.shp || "");
  const pg = PAGES[a.n];
  const [split, setSplit] = React.useState(false);
  const [pt, setPt] = React.useState(() => pg ? pg.map(p => p[1]) : []);
  if (a.filed) return <div className="ax-res"><button className="lnk" onClick={() => api.unfile(a.n)}>Undo</button></div>;
  return (
    <div className="ax-x">
      {pg && <button className={"split-t" + (split ? " on" : "")} onClick={() => setSplit(v => !v)}><Ic n="layers" s={12}></Ic>{split ? "File as one document" : "Split into " + pg.length + " documents"}<span className="mono">{pg.length + " pages"}</span></button>}
      {split ? <div className="pgs">{pg.map(([l], i) => <div key={i} className="pg"><span className="pg-th mono">{i + 1}</span><span className="pg-l">{l}</span><select value={pt[i]} onChange={e => setPt(o => o.map((v, j) => j === i ? e.target.value : v))}>{["Rate confirmation", "Proof of delivery", "Bill of lading", "Invoice", "Tender", "Other", "Don't file"].map(t => <option key={t}>{t}</option>)}</select></div>)}</div> :
      <div className="ax-row"><label className="sel"><span>As</span><select value={type} onChange={e => setType(e.target.value)}>{["Rate confirmation", "Proof of delivery", "Bill of lading", "Invoice", "Tender", "Other"].map(x => <option key={x}>{x}</option>)}</select></label><label className="sel"><span>On</span><select value={shp} onChange={e => setShp(e.target.value)}><option value="">Shipment</option>{SHIPMENTS.map(s => <option key={s.id} value={s.id}>{s.pro}</option>)}</select></label></div>}
      {split && <label className="sel"><span>On</span><select value={shp} onChange={e => setShp(e.target.value)}><option value="">Shipment</option>{SHIPMENTS.map(s => <option key={s.id} value={s.id}>{s.pro}</option>)}</select></label>}
      <div className="ax-f"><button className="btn ghost sm" onClick={() => api.ov({ k: "doc", a })}><Ic n="eye" s={12}></Ic>Preview</button><span className="sp"></span><button className="btn ink sm ab-go" disabled={!shp} onClick={() => split ? api.file(a.n, shp, pt.filter(t => t !== "Don't file").length + " documents (" + [...new Set(pt.filter(t => t !== "Don't file"))].join(", ").toLowerCase() + ")") : api.file(a.n, shp, type)}>{split ? "File " + pt.filter(t => t !== "Don't file").length : "File"}</button></div>
    </div>
  );
}
function StopsX({ m, api }) {
  const f = m.flags || {};
  if (f.stops) return null;
  return (
    <div className="ax-x">
      <div className="stops3">{STOPS.rows.slice(0, 4).map(([k, v, t]) => <div key={k} className={t}><span>{k}</span><b className="mono">{v.replace("Oct 3, ", "")}</b></div>)}</div>
      <p className="ax-q sm">{STOPS.read}</p>
      <div className="ax-f col"><button className="btn ink sm ab-go" onClick={() => api.flag("stops", "Waiving $185", { desk: true, subject: "Re: " + m.subject, body: "Hi — you're right. The ELD shows arrival at 10:40, 40 minutes after the 08:00–10:00 window, so we've removed the $185.00 detention from INV-20931. A corrected invoice follows today.\n\nThanks,\nTrenova Billing", waive: true })}>Waive $185</button><button className="btn sm" onClick={() => api.flag("stops", "Standing by the charge", { desk: true, subject: "Re: " + m.subject, body: "Hi — attached is the ELD record and gate log for SEED-PAY-002 on Oct 3. Happy to talk it through.\n\nThanks,\nTrenova Billing", att: "ELD_SEED-PAY-002_Oct3.pdf" })}>Stand by it, send the log</button></div>
    </div>
  );
}
function Confirm({ label, onGo, note }) {
  return <div className="ax-x"><p className="ax-q sm">{note}</p><div className="ax-f"><span className="sp"></span><button className="btn ink sm ab-go" onClick={onGo}>{label}</button></div></div>;
}

function Act({ m, a, n, open, setOpen, api, start: run }) {
  const R = m.made.reply, L = m.made.load;
  const start = () => run(a);
  const was = React.useRef(a.done), [just, setJust] = React.useState(false);
  React.useEffect(() => { if (a.done && !was.current) { setJust(true); const t = setTimeout(() => setJust(false), 1100); was.current = true; return () => clearTimeout(t); } was.current = a.done; }, [a.done]);
  const ref = React.useRef();
  React.useEffect(() => { if (!open || !ref.current) return; const sc = ref.current.closest(".rd-s"); if (!sc) return; const t = setTimeout(() => { const d = ref.current.getBoundingClientRect().top - sc.getBoundingClientRect().top; if (d < 40 || d > sc.clientHeight - 260) sc.scrollTo({ top: sc.scrollTop + d - 80, behavior: "smooth" }); }, 60); return () => clearTimeout(t); }, [open]);
  const busy = (a.k === "load" && L && L.streaming) || (a.k === "reply" && R && R.streaming);
  let x = null;
  if (open) {
    if (a.k === "file") x = <FileRow m={m} a={a.a} api={api}></FileRow>;
    if (a.k === "load" && L) x = <LoadForm m={m} api={api}></LoadForm>;
    if (a.k === "reply" && R) x = <Composer m={m} api={api}></Composer>;
    if (a.k === "stops") x = <StopsX m={m} api={api}></StopsX>;
    if (a.k === "osd") x = !a.done && <Confirm label="Log exception" note={"Adds a shortage exception to " + SHP[m.shp].pro + " so billing holds the invoice until it's settled."} onGo={() => api.flag("osd", "2 pallets short")}></Confirm>;
    if (a.k === "block") x = !a.done && <Confirm label="Block and ignore" note={"Future mail from " + m.from[1].split("@")[1] + " goes straight to Ignored."} onGo={() => { api.flag("blocked", true); api.decide("Ignored", "Blocked sender — phishing."); }}></Confirm>;
  }
  return (
    <li ref={ref} className={"ax" + (just ? " just" : "") + (a.done ? " done" : "") + (open ? " open" : "") + (busy ? " busy" : "")}>
      <button className="ax-h" onClick={start}>
        <span className="ax-c">{a.done ? <Ic n="check" s={10} w={3}></Ic> : busy ? <i className="spin2"></i> : <span className="mono">{n}</span>}</span>
        <span className="ax-t"><b>{a.t}</b><span>{a.sub}</span></span>
        
        <Ic n={open ? "chevD" : "chevR"} s={12}></Ic>
      </button>
      {x}
    </li>
  );
}

function Info({ it, api, m }) {
  const [more, setMore] = React.useState(false);
  if (it.facts) return (
    <div className="in-f"><div className="in-l">{it.l}</div><div className="facts2">{it.facts.map(([k, v, t]) => <button key={k} onContextMenu={e => ctx(e, { title: k, sub: v, items: [{ l: "Copy value", ic: "copy", on: () => api.act("Copied " + v) }, { l: "Copy as “" + k + ": …”", ic: "copy", on: () => api.act("Copied") }, { sep: 1 }, { l: "This is wrong", ic: "warn", hint: "Nova learns from it", on: () => api.factWrong(k, v) }] })} className={"fact" + (t ? " " + t : "")} title="Copy" onClick={() => api.act("Copied " + v)}><span>{k}</span><b>{v}</b></button>)}</div></div>
  );
  return (
    <div className={"in" + (more ? " more" : "")}>
      <button className="in-h" onContextMenu={e => { const s = it.id === "shp" && m.shp && SHP[m.shp]; ctx(e, { title: it.l, sub: it.v, items: [{ l: "Open", ic: "ext", on: () => api.act("Opening " + it.l) }, { l: "Peek", ic: "eye", on: () => api.ov({ k: it.id === "cust" ? "cust" : "rec" }) }, s && { sep: 1 }, s && { l: "Copy PRO", ic: "copy", hint: s.pro, on: () => api.act("Copied " + s.pro) }, s && { l: "Copy BOL", ic: "copy", hint: s.bol.replace("BOL ", ""), on: () => api.act("Copied " + s.bol) }, s && s.po && { l: "Copy PO", ic: "copy", hint: s.po.replace("PO ", ""), on: () => api.act("Copied " + s.po) }, { sep: 1 }, { l: "Unlink", ic: "x", d: 1, on: () => api.unlink(it.id) }] }); }} onClick={() => it.more ? setMore(v => !v) : api.ov({ k: it.id === "cust" ? "cust" : "rec" })}>
        <span className="in-ic"><Ic n={it.ic} s={13}></Ic></span>
        <span className="in-t"><b className={it.ic === "truck" ? "mono" : ""}>{it.l}</b><span>{it.v}{it.meta && <em className={it.w ? "w" : ""}>{" · " + it.meta}</em>}</span></span>
        {it.more ? <Ic n={more ? "chevD" : "chevR"} s={12}></Ic> : <Ic n="ext" s={12}></Ic>}
      </button>
      {more && <><dl className="ax-kv in-kv">{it.more.map(([k, v]) => <div key={k}><dt>{k}</dt><dd>{v}</dd></div>)}</dl><button className="lnk in-go" onClick={() => api.ov({ k: "rec" })}>Open shipment<Ic n="arrowR" s={11}></Ic></button></>}
    </div>
  );
}

function Rail({ m, api, open, setOpen, start }) {
  const A = actsOf(m), I = infoOf(m), left = A.filter(a => !a.done).length, cf = CONFLICTS[m.id], cfd = (m.flags || {}).conflict;
  return (
    <aside className="rail">
      {cf && isNeed(m) && (cfd ? <div className="heads ok"><Ic n="check" s={12} w={2.4}></Ic><span>{cfd}</span><button className="lnk" onClick={() => api.flag("conflict", null)}>Undo</button></div> :
        <div className="heads"><div className="heads-h"><Ic n="warn" s={13}></Ic><b>{cf.t}</b></div><p>{cf.d}</p><div className="heads-a">{cf.acts.map(([l, v], i) => <button key={l} className={"btn sm" + (i ? " ghost" : "")} onClick={() => api.flag("conflict", v)}>{l}</button>)}</div></div>)}
      {A.length > 0 && <section>
        <div className="rl-h"><span>Actions</span><span className="mono">{A.length - left + "/" + A.length}</span></div>
        <div className="rl-bar" key={A.length - left}><i style={{ width: ((A.length - left) / A.length) * 100 + "%" }}></i></div>
        <ol className="axl">{A.map((a, i) => <Act key={a.id} m={m} a={a} n={i + 1} open={open === a.id} setOpen={setOpen} api={api} start={start}></Act>)}</ol>
      </section>}
      {I.length > 0 && <section>
        <div className="rl-h"><span>Information</span></div>
        <div className="inl">{I.map(it => <Info key={it.id} it={it} api={api} m={m}></Info>)}</div>
      </section>}
      {m.unread && <div className="heads"><div className="heads-h"><Ic n="eye" s={13}></Ic><b>Not read yet</b></div><p>Nova is off, so nothing was pulled out of this message. You can still reply, link and handle it by hand.</p></div>}
      <button className="todesk" onClick={() => api.ov({ k: "desk" })}><span className="dsk-i"><Ic n="chat" s={13}></Ic></span><span><b>Work on it in Desk</b><span>Hand the message and its actions to one of your agents</span></span><Ic n="arrowR" s={13}></Ic></button>
    </aside>
  );
}
Object.assign(window, { actsOf, infoOf, Rail });
