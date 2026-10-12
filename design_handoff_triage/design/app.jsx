const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "theme": "dark",
  "density": "comfortable",
  "rows": "next step",
  "marks": false,
  "demo": "live",
  "device": "desktop",
  "time": "afternoon"
}/*EDITMODE-END*/;

const DRAFTS = {
  m1: { rows: [["Customer", "Northwind Foods"], ["Origin", "Chicago DC, 2400 S Halsted"], ["Destination", "", "Street address"], ["Pickup", "Thu Oct 8, 08:00"], ["Delivery", "", "Date and time"], ["Equipment", "Dry van"], ["Weight", "42,000 lb"], ["Rate", "", "Rate con to follow"]], src: "the email text" },
  m8: { rows: [["Customer", "Acme Manufacturing"], ["Loads", "3 × dry van"], ["Week of", "Oct 12"], ["Origin", "Acme Plant 2, Toledo OH"], ["Destinations", "Fort Wayne, Lansing, Dayton"], ["Respond by", "Thu Oct 8"]], src: "tender_4471.pdf" },
  m7: { text: "Hi — PO 88213 is on S-104198, Dallas to Memphis. The truck is running about two hours behind; current ETA is 16:10, ahead of your 17:00 receiving cutoff. We'll tell you if that moves.\n\nThanks,\nTrenova" },
};
let PRO_N = 60;
window.DRAFTS = DRAFTS;

function App({ t }) {
  const seed = () => t.demo === "live" || t.demo === "desk off" ? MESSAGES.map(m => ({ ...m, ...(SEED[m.id] || {}), atts: m.atts.map(a => ({ ...a, filed: a.filed ? m.shp : null })), made: {}, log: [], flags: {} })) : [];
  const [raw, setMsgs] = React.useState(seed);
  React.useEffect(() => { setMsgs(seed()); }, [t.demo]);
  const [views, setViews] = React.useState(VIEWS0);
  const [sort, setSort] = React.useState("due");
  const [loading, setLoading] = React.useState(false);
  const [, setTick] = React.useState(0);
  const [ov, setOv] = React.useState(null);
  const [corr, setCorr] = React.useState(3);
  const [renaming, setRenaming] = React.useState(null);
  const [fd, setFd] = React.useState({ stage: 0, read: 0, done: false });
  React.useEffect(() => setFd({ stage: 0, read: 0, done: false }), [t.demo]);
  const firstDay = t.demo === "first day" && !fd.done;
  const live0 = () => MESSAGES.map(m => ({ ...m, ...(SEED[m.id] || {}), atts: m.atts.map(a => ({ ...a, filed: a.filed ? m.shp : null })), made: {}, log: [], flags: {} }));
  const startReading = () => {
    setFd({ stage: 1, read: 0, done: false }); const t0 = Date.now();
    const iv = setInterval(() => { const k = Math.min(1, (Date.now() - t0) / 3200); setFd(f => ({ ...f, read: 38 * (1 - Math.pow(1 - k, 2)) })); if (k >= 1) { clearInterval(iv); setFd(f => ({ ...f, stage: 2 })); setTimeout(() => { setMsgs(live0()); setFd(f => ({ ...f, done: true })); }, 1100); } }, 40);
  };
  const poolI = React.useRef(0);
  React.useEffect(() => {
    if (t.demo !== "live") return;
    const iv = setInterval(() => {
      const p = LIVE_POOL[poolI.current]; if (!p) return clearInterval(iv); poolI.current++;
      const id = "L" + poolI.current;
      setMsgs(xs => [{ id, to: ["intake@dev.trenova.app"], min: 0, status: "InReview", run: "run_live", atts: [], made: {}, log: [], flags: {}, isNew: true, ...p }, ...xs]);
      say("New from " + p.from[0]);
      setTimeout(() => setMsgs(xs => xs.map(x => x.id === id ? { ...x, isNew: false } : x)), 4000);
    }, 26000);
    return () => clearInterval(iv);
  }, [t.demo]);
  const [deskOn, setDeskOn] = React.useState(false);
  React.useEffect(() => setDeskOn(false), [t.demo]);
  const deskOff = t.demo === "desk off" && !deskOn;
  const msgs = deskOff ? raw.map(x => isNeed(x) ? { ...x, unread: true, kind: "Other", shp: null, cust: null, reason: "", note: "" } : x) : raw;
  const [folder, setFolder] = React.useState({ t: "brief", v: "" });
  const [openId, setOpenId] = React.useState(null);
  const [q, setQ] = React.useState("");
  const [side, setSide] = React.useState(true);
  const [leaving, setLeaving] = React.useState([]);
  const [toast, setToast] = React.useState(null);
  const [linkOpen, setLinkOpen] = React.useState(false);
  const [keys, setKeys] = React.useState(false);
  const [pane, setPane] = React.useState({ rail: true, act: null });
  const [sel, setSel] = React.useState([]);
  const [rules, setRules] = React.useState(RULES0);
  const searchRef = React.useRef(), tt = React.useRef();
  window.__rules = rules;
  React.useEffect(() => { openId ? localStorage.setItem("tv-inbox-open", openId) : localStorage.removeItem("tv-inbox-open"); }, [openId]);
  const say = (text, undo) => { clearTimeout(tt.current); setToast({ text, undo, k: Date.now() }); tt.current = setTimeout(() => setToast(null), undo ? 5000 : 2400); };

  const ql = q.trim().toLowerCase();
  const threads = {}; msgs.forEach(x => { const k = x.th || x.id; threads[k] = (threads[k] || 0) + 1; });
  const inF = msgs.filter(x => inFolder(x, folder, views) && matchQ(x, ql));
  const latest = inF.filter(x => !x.th || !inF.some(y => y.th === x.th && y.min < x.min));
  const needSort = sort === "due" && latest.some(isNeed);
  const list = folder.t === "rules" || folder.t === "brief" ? [] : latest.sort((a, b) => needSort ? ((isNeed(b) - isNeed(a)) || (dueOf(a) - dueOf(b))) : a.min - b.min);
  const ids = list.map(m => m.id);
  const m = msgs.find(x => x.id === openId);
  const pos = ids.indexOf(openId);
  const wsRef = React.useRef();
  const firstId = list[0] && list[0].id;
  React.useEffect(() => {
    if (m || !firstId || folder.t === "brief" || folder.t === "rules" || loading) return;
    const el = wsRef.current; if (el && el.clientWidth <= 900) return;
    open(firstId);
  }, [m, firstId, folder.t, folder.v, loading]);
  const waiting = msgs.filter(x => isNeed(x) && !x.snooze && !x.await);
  const older = m && m.th ? msgs.filter(x => x.th === m.th && x.min > m.min).sort((a, b) => b.min - a.min) : [];

  const upd = (id, fn) => setMsgs(xs => xs.map(x => x.id === id ? { ...x, ...fn(x) } : x));
  const open = id => { setOpenId(id); setPane(p => ({ ...p, act: null })); };
  const pick = f => { setFolder(f); setOpenId(null); setSel([]); if (f.t !== "rules" && f.t !== "brief") { setLoading(true); setTimeout(() => setLoading(false), 380); } };
  const step = d => { if (!ids.length) return; const i = pos === -1 ? (d > 0 ? 0 : ids.length - 1) : Math.min(ids.length - 1, Math.max(0, pos + d)); open(ids[i]); };
  const openArt = art => setPane(p => ({ ...p, rail: true, act: art && art.startsWith("doc:") ? null : art }));
  const logIt = (x, l) => [...(x.log || []), [l, hhmm(NOW)]];

  const leave = (list_, apply, label, extra) => {
    const prev = msgs.filter(x => list_.includes(x.id)).map(x => ({ ...x }));
    const stays = !LANES.find(l => folder.t === "lane" && l[0] === folder.v) || true;
    const after = ids.filter(i => !list_.includes(i));
    const nxt = list_.includes(openId) ? (ids.slice(pos + 1).find(i => !list_.includes(i)) || after[after.length - 1] || null) : openId;
    setLeaving(list_);
    setTimeout(() => {
      setMsgs(xs => xs.map(x => list_.includes(x.id) ? { ...x, ...apply(x) } : x));
      setLeaving([]);
      const nowIn = id => { const x = msgs.find(y => y.id === id); return x && inFolder({ ...x, ...apply(x) }, folder, views); };
      if (list_.includes(openId) && !nowIn(openId)) open(nxt);
    }, 240);
    say(label, () => { setMsgs(xs => xs.map(x => prev.find(p => p.id === x.id) || x)); if (extra) extra(); setToast(null); });
  };
  const decide = (idsIn, status, note, teach) => {
    const target = idsIn.filter(i => { const x = msgs.find(y => y.id === i); return x && isNeed(x); });
    if (!target.length) return;
    let ruleId = null;
    if (teach && target.length === 1) {
      const x = msgs.find(y => y.id === target[0]);
      ruleId = "r" + Date.now();
      setRules(rs => [{ id: ruleId, kind: x.kind, from: x.from[1].split("@")[1], mb: x.mb, act: teach, n: 0, on: true, by: ME, fresh: true }, ...rs]);
    }
    const n = target.length;
    leave(target, x => ({ status, by: ME, byMin: 0, done: note || "", snooze: null, await: null, log: logIt(x, status === "Actioned" ? "Marked handled" : "Ignored") }), (status === "Actioned" ? "Marked handled" : "Ignored") + (n > 1 ? " · " + n + " messages" : "") + (ruleId ? " · rule added" : ""), ruleId ? () => setRules(rs => rs.filter(r => r.id !== ruleId)) : null);
  };
  const snoozeMany = (target, label) => leave(target, x => ({ snooze: label, log: logIt(x, "Snoozed until " + label) }), "Snoozed until " + label.split(" · ")[0].toLowerCase() + (target.length > 1 ? " · " + target.length + " messages" : ""));
  const quick = (target, kind) => {
    if (kind === "snooze") return snoozeMany(target, "Tomorrow morning · Thu 08:00");
    if (kind === "mine") { target.forEach(id => upd(id, x => ({ owner: ME, log: logIt(x, "Assigned to you") }))); return say("Assigned to you · " + target.length); }
    decide(target, kind, "");
  };

  const api = m ? {
    act: say,
    ov: setOv,
    unlink: w => { upd(m.id, x => ({ ...(w === "cust" ? { cust: null } : { shp: null }), log: logIt(x, "Unlinked " + (w === "cust" ? x.cust : SHP[x.shp].pro)) })); say("Unlinked"); },
    factWrong: (k, v) => { setCorr(c => c + 1); upd(m.id, x => ({ log: logIt(x, "Flagged “" + k + "” as wrong") })); say("Thanks — Nova will re-check " + k.toLowerCase() + " on messages like this"); },
    correct: k => { const dom = m.from[1].split("@")[1], was = KIND[m.kind].l; upd(m.id, x => ({ kind: k, corrected: true, log: logIt(x, "Corrected: " + was + " → " + KIND[k].l) })); setCorr(c => c + 1); say("Got it — I'll read these from " + dom + " as " + KIND[k].l.toLowerCase() + " from now on"); },
    open,
    openArt,
    link: () => setLinkOpen(true),
    decide: (s, note, teach) => decide([m.id], s, note, teach),
    assign: n => { upd(m.id, x => ({ owner: n, log: logIt(x, n ? "Assigned to " + (n === ME ? "you" : n) : "Unassigned") })); say(n ? "Assigned to " + (n === ME ? "you" : n) : "Unassigned"); },
    snooze: label => label ? snoozeMany([m.id], label) : (upd(m.id, x => ({ snooze: null, log: logIt(x, "Woken") })), say("Back in Waiting on you")),
    reopen: () => { upd(m.id, x => ({ status: "InReview", by: null, byMin: null, done: "", log: logIt(x, "Reopened") })); say("Reopened"); },
    file: (n, shp, type) => { upd(m.id, x => ({ atts: x.atts.map(a => a.n === n ? { ...a, filed: shp, ftype: type } : a), shp: x.shp || shp, log: logIt(x, "Filed " + n + " on " + SHP[shp].pro) })); say("Filed on " + SHP[shp].pro); setPane(p => ({ ...p, act: null })); },
    unfile: n => upd(m.id, x => ({ atts: x.atts.map(a => a.n === n ? { ...a, filed: null } : a) })),
    draftLoad: stream => {
      const D = DRAFTS[m.id] || { rows: [["Customer", m.cust || ""], ["Origin", "", "City, state"], ["Destination", "", "City, state"], ["Pickup", "", "Date and time"]], src: "the email" }, id = m.id;
      upd(id, x => ({ made: { ...x.made, load: { rows: D.rows, src: D.src, streaming: stream, shown: stream ? 0 : null } } }));
      openArt("load");
      if (stream) { let k = 0; const iv = setInterval(() => { k++; upd(id, x => x.made.load ? ({ made: { ...x.made, load: { ...x.made.load, shown: k, streaming: k < D.rows.length } } }) : {}); if (k >= D.rows.length) clearInterval(iv); }, 160); }
    },
    createLoad: f => { const pro = "S-1042" + PRO_N++; upd(m.id, x => ({ made: { ...x.made, load: { ...x.made.load, pro, vals: f } }, log: logIt(x, "Created " + pro) })); say(pro + " created as a draft"); setPane(p => ({ ...p, act: null })); },
    discardLoad: () => { upd(m.id, x => { const { load, ...rest } = x.made; return { made: rest }; }); setPane(p => ({ ...p, act: null })); },
    deskReply: () => {
      const id = m.id;
      upd(id, x => ({ made: { ...x.made, reply: { desk: true, subject: "Re: " + x.subject, body: (DRAFTS[id] || {}).text || "", streaming: true } } }));
      openArt("reply");
      setTimeout(() => upd(id, x => x.made.reply ? ({ made: { ...x.made, reply: { ...x.made.reply, streaming: false } } }) : {}), 1300);
    },
    draftReply: o => { upd(m.id, x => ({ made: { ...x.made, reply: o } })); openArt("reply"); },
    flag: (k, v, draft) => { upd(m.id, x => ({ flags: { ...x.flags, [k]: v }, made: draft ? { ...x.made, reply: draft } : x.made, log: logIt(x, k === "stops" ? v : k === "osd" ? "Logged 2 pallets short on " + (x.shp ? SHP[x.shp].pro : "") : k === "blocked" ? "Blocked " + x.from[1].split("@")[1] : String(v)) })); if (draft) openArt("reply"); else setPane(p => ({ ...p, act: null })); },
    startReply: () => { upd(m.id, x => ({ made: { ...x.made, reply: { subject: "Re: " + (x.subject || "your email"), body: "" } } })); openArt("reply"); },
    reply: () => { if (!m.made.reply) upd(m.id, x => ({ made: { ...x.made, reply: { subject: "Re: " + (x.subject || "your email"), body: "" } } })); openArt("reply"); },
    wait: label => {
      if (!label) { upd(m.id, x => ({ await: null, log: logIt(x, "Stopped waiting") })); return say("Back in Waiting on you"); }
      const id = m.id, th = m.th || m.id, src = m;
      leave([id], x => ({ await: label, th, log: logIt(x, "Waiting on " + senderOf(x).split(" ")[0] + " to reply") }), "Parked in Waiting on them");
      setTimeout(() => {
        const nid = "m" + Date.now();
        setMsgs(xs => xs.find(x => x.id === id && x.await) ? [...xs.map(x => x.id === id ? { ...x, await: null, status: "Actioned", by: ME, byMin: 0, done: "They replied.", log: logIt(x, "They replied") } : x), { id: nid, th, from: src.from, to: src.to, subject: "Re: " + (src.subject || "your email"), min: 0, status: "InReview", kind: src.kind, conf: 0.9, mb: src.mb, shp: src.shp, cust: src.cust, reason: "A reply in a thread you were waiting on.", body: [src.kind === "StatusRequest" ? "Perfect, thanks — we'll hold the dock until 17:30 just in case." : src.kind === "DetentionDispute" ? "Thanks for looking into it. Corrected invoice received." : "Thanks — got it."], atts: [], run: "run_9001", made: {}, log: [], flags: {} }] : xs);
        say(senderOf(src).split(" ")[0] + " replied");
      }, 6500);
    },
    sendReply: (subject, body, alt) => {
      if (m.from[1].startsWith("edi@") && !alt) { upd(m.id, x => ({ made: { ...x.made, reply: { ...x.made.reply, subject, body, err: m.from[1] + " doesn't accept replies — their server returned 550.", alt: "ops@partnercarrier.example" } } })); return; }
      upd(m.id, x => ({ made: { ...x.made, reply: { ...x.made.reply, subject, body, sent: true, sentAt: hhmm(NOW), to: alt || null, err: null } }, flags: { ...x.flags }, log: logIt(logIt(x, "Replied from " + MAILBOXES.find(b => b.id === x.mb).addr), x.made.reply.waive ? "Waived $185 detention on INV-20931" : null).filter(l => l[0]) })); say("Sent"); setPane(p => ({ ...p, act: null })); },
    discardReply: () => { upd(m.id, x => { const { reply, ...rest } = x.made; return { made: rest }; }); setPane(p => ({ ...p, act: null })); },
  } : null;
  const rowCtx = (e, x) => {
    const many = sel.includes(x.id) && sel.length > 1, tg = many ? sel : [x.id], need = isNeed(x), dom = x.from[1].split("@")[1];
    const asg = n => { tg.forEach(id => upd(id, y => ({ owner: n, log: logIt(y, "Assigned to " + (n === ME ? "you" : n)) }))); say("Assigned to " + (n === ME ? "you" : n.split(" ")[0]) + (many ? " · " + tg.length : "")); };
    ctx(e, { title: many ? tg.length + " selected" : senderOf(x), sub: many ? null : x.subject || "(no subject)", items: [
      !many && { l: "Open", ic: "arrowR", k: "↵", on: () => open(x.id) },
      !many && { l: "Open in a new tab", ic: "ext", on: () => say("Opened in a new tab") },
      { sep: 1 },
      need && { l: "Mark handled", ic: "check", k: "E", on: () => decide(tg, "Actioned", "") },
      need && { l: "Ignore", ic: "archive", k: "#", on: () => decide(tg, "Ignored", "") },
      need && { l: "Snooze", ic: "clock", k: "S", sub: [...SNOOZES.map(([l, t]) => ({ l, hint: t, on: () => snoozeMany(tg, l + " · " + t) })), ...(!many && EVENT_SNOOZE(x).length ? [{ sep: 1 }, { h: "Until something happens" }, ...EVENT_SNOOZE(x).map(([l]) => ({ l, ic: "bolt", on: () => snoozeMany(tg, l + " · event") }))] : [])] },
      need && !many && x.made && x.made.reply && x.made.reply.sent && { l: "Wait on them", ic: "send", on: () => { open(x.id); setTimeout(() => document.querySelector(".ab .btn.ghost") && document.querySelector(".ab .btn.ghost").click(), 50); } },
      !need && { l: "Reopen", ic: "repeat", on: () => tg.forEach(id => upd(id, y => ({ status: "InReview", by: null, byMin: null, done: "", log: logIt(y, "Reopened") }))) },
      { sep: 1 },
      need && { l: "Assign", ic: "user", k: "M", sub: TEAM.map(([n, r]) => ({ l: n === ME ? "Me" : n, hint: n === ME ? "" : r, icon: <Av n={n} s={16}></Av>, on: () => asg(n) })) },
      !many && { l: "Correct the kind", ic: "edit", sub: KINDS.filter(([k]) => k !== x.kind).map(([k]) => ({ icon: <i className={"kdot" + (KIND[k].h == null ? " n" : "")} style={{ "--h": KIND[k].h }}></i>, l: KIND[k].l, on: () => { upd(x.id, y => ({ kind: k, corrected: true, log: logIt(y, "Corrected: " + KIND[y.kind].l + " → " + KIND[k].l) })); setCorr(c => c + 1); say("Got it — I'll read these from " + dom + " as " + KIND[k].l.toLowerCase()); } })) },
      !many && { l: "Link to a record…", ic: "link", k: "L", on: () => { open(x.id); setLinkOpen(true); } },
      !many && { l: "Ask an agent in Desk", ic: "chat", k: "A", on: () => { open(x.id); setOv({ k: "desk" }); } },
      { sep: 1 },
      !many && { l: "Copy link", ic: "copy", on: () => say("Link copied") },
      !many && { l: "Copy sender address", ic: "copy", hint: x.from[1], on: () => say("Copied " + x.from[1]) },
      !many && { l: "Make a rule from this sender", ic: "bolt", sub: [{ h: "Nova should…" }, ...Object.keys(ACT_SAY).slice(0, 6).map(ac => ({ l: ACT_SAY[ac].charAt(0).toUpperCase() + ACT_SAY[ac].slice(1), on: () => { const id = "r" + Date.now(); setRules(rs => [{ id, kind: x.kind, from: dom, mb: x.mb, act: ac, n: 0, on: false, by: ME, fresh: true }, ...rs]); say("Rule drafted — paused until you turn it on", () => setRules(rs => rs.filter(r => r.id !== id))); } }))] },
    ] });
  };
  const folderCtx = (e, f) => {
    const inF = msgs.filter(x => inFolder(x, f, views) && isNeed(x) && !x.snooze).map(x => x.id), name = folderName(f, views);
    const mb = f.t === "mb" && MAILBOXES.find(b => b.id === f.v);
    ctx(e, { title: name, sub: inF.length + " waiting", items: [
      { l: "Open", ic: "arrowR", on: () => pick(f) },
      { sep: 1 },
      { l: "Mark all handled", ic: "check", dis: !inF.length, hint: inF.length ? inF.length + "" : "", on: () => decide(inF, "Actioned", "") },
      { l: "Snooze all", ic: "clock", dis: !inF.length, sub: SNOOZES.map(([l, t]) => ({ l, hint: t, on: () => snoozeMany(inF, l + " · " + t) })) },
      f.t === "view" && { sep: 1 },
      f.t === "view" && { l: "Rename", ic: "edit", on: () => setRenaming(f.v) },
      f.t === "view" && { l: "Delete view", ic: "x", d: 1, on: () => { const prev = views; setViews(vs => vs.filter(v => v.id !== f.v)); if (sameF(folder, f)) pick({ t: "brief", v: "" }); say("View deleted", () => setViews(prev)); } },
      mb && { sep: 1 },
      mb && { l: "Send a test email", ic: "send", hint: mb.addr, on: () => { say("Sending a test to " + mb.addr + "…"); setTimeout(() => { mb.ok = true; setTick(n => n + 1); say("Test arrived on " + mb.name); }, 1400); } },
      mb && { l: "Source settings", ic: "gear", on: () => say("Opening " + mb.name + " settings") },
      f.t === "kind" && { sep: 1 },
      f.t === "kind" && { l: "Make a rule for " + KIND[f.v].l.toLowerCase(), ic: "bolt", on: () => pick({ t: "rules", v: "" }) },
    ] });
  };
  const saveView = qq => { const id = "v" + Date.now(); setViews(vs => [...vs, { id, name: qq.charAt(0).toUpperCase() + qq.slice(1), q: qq }]); setQ(""); pick({ t: "view", v: id }); say("Saved as a view"); };
  const fixMb = id => { const b = MAILBOXES.find(x => x.id === id); b.ok = true; setTick(n => n + 1); say("Mail is arriving on " + b.name + " again"); };
  const testMail = () => { setMsgs(xs => [{ id: "mt", from: ["You", "admin@trenova.app"], to: ["intake@dev.trenova.app"], subject: "Test — does this arrive?", min: 0, status: "InReview", kind: "Other", conf: 0.99, mb: "mb1", reason: "A test you sent.", body: ["Testing the intake mailbox."], atts: [], run: "run_test", made: {}, log: [], flags: {} }, ...xs]); say("Test email arrived"); };
  const link = ({ shp, cust, why }) => {
    upd(m.id, x => ({ shp, cust, flags: { ...x.flags, linked: true }, fail: x.status === "Quarantined" ? x.fail : "", reason: "Linked by hand: " + why, log: logIt(x, "Linked to " + (shp ? SHP[shp].pro : cust)) }));
    setLinkOpen(false); say("Linked to " + (shp ? SHP[shp].pro : cust));
  };

  const onKey = React.useRef();
  onKey.current = e => {
    if (e.metaKey || e.ctrlKey || e.altKey || linkOpen || keys || ov) return;
    const tg = e.target; if (tg.isContentEditable || /INPUT|TEXTAREA|SELECT/.test(tg.tagName)) { if (e.key === "Escape") tg.blur(); return; }
    const k = e.key;
    if (k === "/") { e.preventDefault(); searchRef.current && searchRef.current.focus(); }
    else if (k === "?") setKeys(true);
    else if (k === "b") pick({ t: "brief", v: "" });
    else if (k === "j" || k === "ArrowDown") { e.preventDefault(); step(1); }
    else if (k === "k" || k === "ArrowUp") { e.preventDefault(); step(-1); }
    else if (k === "Escape") { if (sel.length) setSel([]); else if (pane.act) setPane(p => ({ ...p, act: null })); else setOpenId(null); }
    else if (!m) return;
    else if (k === "e") decide(sel.length ? sel : [m.id], "Actioned", "");
    else if (k === "#") decide(sel.length ? sel : [m.id], "Ignored", "");
    else if (k === "s" && isNeed(m)) snoozeMany(sel.length ? sel : [m.id], "Tomorrow morning · Thu 08:00");
    else if (k === "m" && isNeed(m)) api.assign(ME);
    else if (k === "x") setSel(xs => xs.includes(m.id) ? xs.filter(x => x !== m.id) : [...xs, m.id]);
    else if (k === "r") { e.preventDefault(); api.reply(); }
    else if (k === "i") setPane(p => ({ ...p, rail: !p.rail }));
    else if (k === "l") { e.preventDefault(); setLinkOpen(true); }
    else if (k === "a") setOv({ k: "desk" });
    else if (k === "Enter") { const g = document.querySelector(".ax.open .ab-go:not(:disabled)"), b = g || document.querySelector(".ab-p:not(:disabled)"); b && b.click(); }
  };
  React.useEffect(() => { const h = e => onKey.current(e); document.addEventListener("keydown", h); return () => document.removeEventListener("keydown", h); }, []);

  return (
    <div className={"tv" + (t.device === "phone" ? " ph" : "") + (t.theme === "dark" ? " dk" : "") + (t.density === "compact" ? " cmp" : "") + (side && t.device !== "phone" ? "" : " sbc")}>
      <Top onToggleSide={() => setSide(v => !v)}></Top>
      <Side onFCtx={folderCtx} renaming={renaming} setRenaming={setRenaming} onConnect={() => setOv({ k: "connect" })} msgs={msgs} folder={folder} setFolder={pick} onKeys={() => setKeys(true)} rules={rules} views={views} setViews={setViews} noMb={t.demo === "no mailbox"}></Side>
      {folder.t === "brief" ? <main className="ws one"><Briefing corr={corr} eod={t.time === "end of day"} firstDay={firstDay} fdStage={fd.stage} fdRead={fd.read} onConnect={() => setOv({ k: "connect" })} msgs={msgs} rules={rules} onOpen={id => { setFolder({ t: "lane", v: "waiting" }); open(id); }} onLane={f => pick(f)} api={{
        fileOn: (id, n, shp) => shp && SHP[shp] && upd(id, x => ({ atts: x.atts.map(a => a.n === n ? { ...a, filed: shp } : a), log: logIt(x, "Filed " + n + " on " + SHP[shp].pro + " from the briefing") })),
        draftFor: (x, a) => {
          if (a.k === "load") { const D = DRAFTS[x.id] || { rows: [["Customer", x.cust || ""], ["Origin", "", "City, state"], ["Destination", "", "City, state"], ["Pickup", "", "Date and time"]], src: "the message" }; upd(x.id, y => ({ made: { ...y.made, load: { rows: D.rows, src: D.src, streaming: false, shown: null } }, log: logIt(y, "Nova drafted the load") })); }
          else if (a.k === "reply") upd(x.id, y => ({ made: { ...y.made, reply: { desk: true, subject: "Re: " + y.subject, body: (DRAFTS[x.id] || {}).text || "Hi — thanks for this. We're on it and will confirm shortly.\n\nThanks,\nTrenova" } }, log: logIt(y, "Nova drafted a reply") }));
        },
        addRule: r => { setRules(rs => [{ id: "r" + Date.now(), ...r, n: 0, on: true, by: ME, fresh: true }, ...rs]); say("Rule on"); },
        preview: r => setOv({ k: "rules", r }),
        act: say,
      }}></Briefing></main> : folder.t === "rules" ? <main className="ws one"><RulesView rules={rules} setRules={setRules} act={say} onSee={r => setOv({ k: "rules", r })}></RulesView></main> :
      <main ref={wsRef} className={"ws" + (m ? " open" : "")}>
        <List onRowCtx={rowCtx} deskOff={deskOff} onDeskOn={() => { setDeskOn(true); say("Nova read " + raw.filter(isNeed).length + " messages"); }} onConnect={() => setOv({ k: "connect" })} list={list} threads={threads} views={views} sort={sort} setSort={setSort} loading={loading} onSaveView={saveView} onFix={fixMb} demo={t.demo} onTest={testMail} folder={folder} openId={openId} onOpen={open} onQuick={quick} q={q} setQ={setQ} searchRef={searchRef} mode={t.rows === "next step" ? "next" : "preview"} leaving={leaving} total={{ handled: msgs.filter(x => x.status === "Actioned").length }} sel={sel} setSel={setSel}></List>
        {m ? <Reader ov={ov} setOv={setOv} m={m} older={older} pos={pos} total={ids.length} msgs={msgs} marks={t.marks} pane={pane} setPane={setPane} onPrev={() => step(-1)} onNext={() => step(1)} onClose={() => setOpenId(null)} api={api}></Reader>
          : <section className="rd idle-w">{list.length || loading ? null : <div className="idle"><span className="ck"><Ic n="check" s={18} w={2.2}></Ic></span><div className="idle-t">{folder.v === "waiting" ? "You're all caught up" : "Nothing in " + folderName(folder, views).toLowerCase()}</div><div className="idle-s">{folder.v === "waiting" ? "Nova brings the next one here the moment it lands." : "Messages show up here as they match."}</div><button className="btn sm" onClick={() => pick({ t: "brief", v: "" })}><span className="dm xs"></span>Back to the briefing</button></div>}</section>}
      </main>}
      {linkOpen && m && <LinkDialog m={m} onClose={() => setLinkOpen(false)} onSave={link}></LinkDialog>}
      {keys && <Keys onClose={() => setKeys(false)}></Keys>}
      <CtxHost></CtxHost>
      {ov && ov.k === "doc" && m && <DocViewer a={m.atts.find(x => x.n === ov.a.n) || ov.a} m={m} api={api} onClose={() => setOv(null)}></DocViewer>}
      {ov && ov.k === "rec" && m && m.shp && <RecPeek m={m} api={api} onClose={() => setOv(null)}></RecPeek>}
      {ov && ov.k === "cust" && m && m.cust && <CustPeek m={m} msgs={msgs} api={api} onClose={() => setOv(null)}></CustPeek>}
      {ov && ov.k === "rules" && <RuleMatches r={ov.r} onClose={() => setOv(null)}></RuleMatches>}
      {ov && ov.k === "connect" && <ConnectMb onClose={() => setOv(null)} onDone={n => { setOv(null); say(n + " connected"); if (firstDay) startReading(); }}></ConnectMb>}
      {toast && <div className="toast" key={toast.k}><Ic n="check" s={13} w={2.2}></Ic>{toast.text}{toast.undo && <button onClick={toast.undo}>Undo</button>}</div>}
    </div>
  );
}

function Root() {
  const [t, setTweak] = useTweaks(TWEAK_DEFAULTS);
  return (
    <>
      {t.device === "phone" ? <div className="phw"><div className="phf"><App t={t}></App></div></div> : <div style={{ width: "100vw", height: "100vh" }}><App t={t}></App></div>}
      <TweaksPanel>
        <TweakSection label="Reader"></TweakSection>
        <TweakToggle label="Underline what Nova read" value={t.marks} onChange={v => setTweak("marks", v)}></TweakToggle>
        <TweakSection label="List"></TweakSection>
        <TweakRadio label="Rows show" value={t.rows} options={["next step", "preview"]} onChange={v => setTweak("rows", v)}></TweakRadio>
        <TweakRadio label="Density" value={t.density} options={["comfortable", "compact"]} onChange={v => setTweak("density", v)}></TweakRadio>
        <TweakSection label="State"></TweakSection>
        <TweakRadio label="Inbox" value={t.demo} options={["live", "first day", "no mailbox", "desk off"]} onChange={v => setTweak("demo", v)}></TweakRadio>
        <TweakRadio label="Time of day" value={t.time} options={["afternoon", "end of day"]} onChange={v => setTweak("time", v)}></TweakRadio>
        <TweakRadio label="Device" value={t.device} options={["desktop", "phone"]} onChange={v => setTweak("device", v)}></TweakRadio>
        <TweakSection label="Appearance"></TweakSection>
        <TweakRadio label="Theme" value={t.theme} options={["dark", "light"]} onChange={v => setTweak("theme", v)}></TweakRadio>
      </TweaksPanel>
    </>
  );
}
ReactDOM.createRoot(document.getElementById("root")).render(<Root></Root>);
