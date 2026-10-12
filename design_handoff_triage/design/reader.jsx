function Kind({ k }) { const x = KIND[k]; return <span className={"kd" + (x.h == null ? " n" : "")} style={{ "--h": x.h }}><i></i>{x.l}</span>; }
const CONF_BAR = m => { const mb = MAILBOXES.find(b => b.id === m.mb); return mb && mb.bar; };

function Pop({ btn, children, cls = "", align = "right" }) {
  const [open, setOpen] = React.useState(false);
  const ref = React.useRef();
  React.useEffect(() => { if (!open) return; const h = e => !ref.current.contains(e.target) && setOpen(false); const k = e => e.key === "Escape" && setOpen(false); document.addEventListener("mousedown", h); window.addEventListener("keydown", k); return () => { document.removeEventListener("mousedown", h); window.removeEventListener("keydown", k); }; }, [open]);
  return <span className={"popw " + cls} ref={ref}>{btn(open, () => setOpen(v => !v))}{open && <div className={"mn " + align} onClick={e => e.target.closest(".mn-i") && setOpen(false)}>{children}</div>}</span>;
}
function Av({ n, s = 22 }) { return <span className="av" style={{ width: s, height: s, fontSize: s * 0.38, borderRadius: "50%" }}>{initials(n)}</span>; }

function Body({ m, marks }) {
  const [hov, setHov] = React.useState(null);
  const [quoted, setQuoted] = React.useState(false);
  const parts = p => { const out = []; let last = 0, i = 0; p.replace(/\{\{([^|}]+)\|([^}]+)\}\}/g, (all, t, l, off) => { out.push(p.slice(last, off)); const key = i++ + p; out.push(marks ? <span key={key} className={"hl" + (hov === key ? " on" : "")} onMouseEnter={() => setHov(key)} onMouseLeave={() => setHov(null)}>{t}{hov === key && <span className="hl-tip"><span className="dm xs"></span>{l}</span>}</span> : t); last = off + all.length; return all; }); out.push(p.slice(last)); return out; };
  const raw = m.from[1].startsWith("edi@");
  return (
    <section className={"msg" + (m.status === "Quarantined" ? " held" : "")}>
      {m.body.map((p, i) => <p key={i} className={raw ? "raw" : ""}>{parts(p)}</p>)}
      {m.quoted && <><button className="qt" onClick={() => setQuoted(v => !v)}>{quoted ? "Hide earlier thread" : "··· Earlier thread"}</button>{quoted && <pre>{m.quoted}</pre>}</>}
    </section>
  );
}

const TEACH = { file: "File on the matched shipment", load: "Create the load as a draft", reply: "Draft a reply for review", stops: "Draft a reply for review" };

function ActionBar({ m, api, start, openAct, onSheet }) {
  const A = actsOf(m), next = A.find(a => !a.done), doneN = A.filter(a => a.done).length;
  const [why, setWhy] = React.useState(false);
  const [noting, setNoting] = React.useState(false);
  const [note, setNote] = React.useState("");
  const [teach, setTeach] = React.useState(false);
  const ref = React.useRef();
  React.useEffect(() => { if (!why) return; const h = e => !ref.current.contains(e.target) && setWhy(false); document.addEventListener("mousedown", h); return () => document.removeEventListener("mousedown", h); }, [why]);
  const held = m.status === "Quarantined", bar = CONF_BAR(m), dom = m.from[1].split("@")[1];
  const teachAs = A.length && TEACH[A[0].k];
  const done = (s, fb) => api.decide(s, note || fb || "", teach && teachAs);
  const R = m.made.reply, L = m.made.load, busy = next && ((next.k === "load" && L && L.streaming) || (next.k === "reply" && R && R.streaming));
  let lead, main;
  const sheetBtn = A.length > 0 && <button className="ab-sh2" onClick={onSheet}><span className="mono">{doneN + "/" + A.length}</span>Actions</button>;
  if (m.await && !noting) {
    lead = <><span className="ab-wt"><Ic n="send" s={11}></Ic></span><span className="ab-t">{"Waiting on " + senderOf(m).split(" ")[0] + " · back " + m.await.split(" · ")[0].toLowerCase() + " if no reply"}</span></>;
    main = <><button className="btn ghost sm" onClick={() => api.wait(null)}>Stop waiting</button><button className="btn ink ab-p" onClick={() => done("Actioned")}>Mark handled<span className="kbd">E</span></button></>;
  } else if (noting) {
    main = <><input autoFocus className="ab-in" value={note} onChange={e => setNote(e.target.value)} onKeyDown={e => { if (e.key === "Enter") { e.stopPropagation(); done("Actioned"); } if (e.key === "Escape") { e.stopPropagation(); setNoting(false); } }} placeholder="What you did, so nobody does it twice"></input><button className="btn sm ghost" onClick={() => setNoting(false)}>Cancel</button><button className="btn sm ink ab-p" onClick={() => done("Actioned")}>Mark handled</button></>;
  } else if (held && next) {
    lead = <><span className="ab-sh"><Ic n="shield" s={11}></Ic></span><span className="ab-t">Held back — looks like phishing</span><button className={"ab-i" + (why ? " on" : "")} onClick={() => setWhy(v => !v)}><Ic n="help" s={13}></Ic></button></>;
    main = <><button className="btn ghost sm" onClick={api.link}>Not phishing</button><button className="btn ink ab-p" onClick={() => done("Ignored", "Phishing — sender not on allow list.")}>Ignore<span className="kbd">↵</span></button></>;
  } else if (next) {
    lead = <><span className={"ab-n mono" + (busy ? " pulse" : "")}>{doneN + 1}<i>/</i>{A.length}</span><span className={"ab-t" + (busy ? " shim" : "")}>{busy ? "Nova is drafting…" : next.t}</span>{doneN === 0 && <button className={"ab-i" + (why ? " on" : "")} title="Why it's waiting" onClick={() => setWhy(v => !v)}><Ic n="help" s={13}></Ic></button>}</>;
    main = <><button className={"btn ab-p " + (next.desk ? "teal" : "ink")} disabled={busy} onClick={() => { const g = openAct === next.id && document.querySelector(".ax.open .ab-go:not(:disabled)"); g ? g.click() : start(next); }}>{next.desk ? "Have Nova do it" : next.k === "reply" && R ? "Review reply" : next.k === "load" && L ? "Review draft" : "Do it"}<span className="kbd">↵</span></button><button className="btn ghost" title="Mark handled (E)" onClick={() => done("Actioned")}><Ic n="check" s={13} w={2}></Ic><span className="lbl">Done</span></button></>;
  } else {
    lead = <><span className="ab-ok"><Ic n="check" s={12} w={2.6}></Ic></span><span className="ab-t">{A.length ? (A.length > 1 ? "All " + A.length + " done" : A[0].sub) : "Nothing to do"}</span>{teachAs && <button className={"teach" + (teach ? " on" : "")} onClick={() => setTeach(v => !v)} title={"Rule: " + KIND[m.kind].l + " from " + dom + " → " + teachAs.toLowerCase()}><span className="cbx2">{teach && <Ic n="check" s={9} w={3}></Ic>}</span>Do this on its own next time</button>}</>;
    main = <>{R && R.sent && <button className="btn ghost" onClick={() => api.wait("Tomorrow morning · Thu 08:00")}><Ic n="send" s={12}></Ic><span className="lbl">Wait for reply</span></button>}<button className="btn ink ab-p" onClick={() => done("Actioned")}>Mark handled<span className="kbd">E</span></button></>;
  }
  return (
    <div className="ab-w" ref={ref}>
      {why && <div className="pop ab-why">
        {m.fail && <div className="pw-r"><Ic n={held ? "shield" : "warn"} s={13}></Ic><span>{m.fail}</span></div>}
        {!m.fail && bar != null && <div className="pw-r"><span className="mtr"><i className={m.conf < bar ? "w" : ""} style={{ width: m.conf * 100 + "%" }}></i><em style={{ left: bar * 100 + "%" }}></em></span><span><b>{Math.round(m.conf * 100)}% sure</b>{m.conf < bar ? " — under this mailbox's " + Math.round(bar * 100) + "% bar" : ""}</span></div>}
        {m.note && <div className="pw-r"><Ic n="hand" s={13}></Ic><span>{m.note}</span></div>}
        {m.reason && <div className="pw-r"><Ic n="link" s={13}></Ic><span>{m.reason}</span></div>}
      </div>}
      <div className={"ab" + (noting ? " noting" : "")} key={(m.await ? "w" : next ? next.id : "all") + (noting ? "n" : "")}>
        {!noting && sheetBtn}
        {lead && <div className="ab-l">{lead}</div>}
        <div className="ab-r">{main}{!noting && <Pop align="up" btn={(o, t) => <button className={"ib" + (o ? " on" : "")} onClick={t}><Ic n="more" s={15}></Ic></button>}>
          <button className="mn-i" onClick={() => setNoting(true)}><Ic n="edit" s={13}></Ic><span>Handled, with a note</span></button>
          <button className="mn-i" onClick={() => done("Ignored")}><Ic n="archive" s={13}></Ic><span>Ignore</span><span className="kbd">#</span></button>
          {R && R.sent && !m.await && <button className="mn-i" onClick={() => api.wait("Tomorrow morning · Thu 08:00")}><Ic n="send" s={13}></Ic><span>Wait for their reply</span></button>}
        </Pop>}</div>
      </div>
    </div>
  );
}

function Reader({ m, ov, setOv, older, pos, total, msgs, marks, pane, setPane, onPrev, onNext, onClose, api }) {
  const [sheet, setSheet] = React.useState(false);
  const [exp, setExp] = React.useState(null);
  React.useEffect(() => { setSheet(false); setExp(null); }, [m.id]);
  const scroller = React.useRef();
  React.useEffect(() => { if (scroller.current) scroller.current.scrollTop = 0; }, [m.id]);
  const need = isNeed(m), s = m.shp && SHP[m.shp], R = m.made && m.made.reply;
  const A = actsOf(m), left = A.filter(a => !a.done).length;
  const setOpen = id => setPane(p => ({ ...p, act: id, rail: true }));
  const start = a => {
    const L = m.made.load;
    if (a.k === "load" && !L) return api.draftLoad(m.id === "m1");
    if (a.k === "reply" && !R) return a.desk ? api.deskReply() : api.startReply();
    if (a.k === "link") return api.link();
    setOpen(pane.act === a.id ? null : a.id);
  };
  return (
    <section className={"rd" + (pane.rail ? " wr" : "")}>
      <div className="rd-c">
        <div className="rd-tb">
          <button className="ib" title="Close (Esc)" onClick={onClose}><Ic n="x" s={14}></Ic></button>
          <button className="ib" title="Previous (K)" disabled={pos <= 0} onClick={onPrev}><Ic n="up" s={14}></Ic></button>
          <button className="ib" title="Next (J)" disabled={pos < 0 || pos >= total - 1} onClick={onNext}><Ic n="down" s={14}></Ic></button>
          {pos >= 0 && <span className="pos">{pos + 1}<i>/</i>{total}</span>}
          <span className="sp"></span>
          {VIEWING[m.id] && <span className="pres" title={VIEWING[m.id][0] + " is " + VIEWING[m.id][1]}><span className="pres-av"><Av n={VIEWING[m.id][0]} s={20}></Av><i></i></span><span className="pres-t">{VIEWING[m.id][0].split(" ")[0] + " is " + VIEWING[m.id][1]}</span></span>}
          {need && <Pop btn={(o, t) => <button className={"own" + (o ? " on" : "")} onClick={t} title="Assign">{m.owner ? <><Av n={m.owner} s={20}></Av><span>{m.owner === ME ? "You" : m.owner.split(" ")[0]}</span></> : <><span className="own-e"><Ic n="user" s={11}></Ic></span><span>Assign</span></>}</button>}>
            <div className="mn-h">Assign to</div>
            {TEAM.map(([n, r]) => <button key={n} className="mn-i" onClick={() => api.assign(n)}><Av n={n} s={20}></Av><span>{n === ME ? "Me" : n}<em>{n === ME ? "" : r}</em></span>{m.owner === n && <Ic n="check" s={12} w={2.4}></Ic>}</button>)}
            {m.owner && <><span className="mn-sep"></span><button className="mn-i" onClick={() => api.assign(null)}><Ic n="x" s={13}></Ic><span>Unassign</span></button></>}
          </Pop>}
          {need && <Pop btn={(o, t) => <button className={"ib" + (o || m.snooze ? " on" : "")} onClick={t} title={m.snooze ? "Snoozed until " + m.snooze : "Snooze (S)"}><Ic n="clock" s={14}></Ic></button>}>
            <div className="mn-h">{m.snooze ? "Snoozed until " + m.snooze : "Snooze until"}</div>
            {SNOOZES.map(([l, t]) => <button key={l} className="mn-i" onClick={() => api.snooze(l + " · " + t)}><span>{l}</span><span className="mono mn-m">{t}</span></button>)}
            {EVENT_SNOOZE(m).length > 0 && <><span className="mn-sep"></span><div className="mn-h">Or until something happens</div>{EVENT_SNOOZE(m).map(([l]) => <button key={l} className="mn-i" onClick={() => api.snooze(l + " · event")}><Ic n="bolt" s={12}></Ic><span>{l}</span></button>)}</>}
            {m.snooze && <><span className="mn-sep"></span><button className="mn-i" onClick={() => api.snooze(null)}><Ic n="x" s={13}></Ic><span>Wake it now</span></button></>}
          </Pop>}
          <button className={"tb-art" + (pane.rail ? " on" : "")} title="Actions and information (I)" onClick={() => setPane(p => ({ ...p, rail: !p.rail }))}><Ic n="panel" s={14}></Ic>{need && left > 0 && <span className="mono">{left}</span>}</button>
          <Pop btn={(o, t) => <button className={"ib" + (o ? " on" : "")} onClick={t}><Ic n="more" s={15}></Ic></button>}>
            <button className="mn-i" onClick={api.reply}><Ic n="enter" s={13}></Ic><span>Reply</span><span className="kbd">R</span></button>
            <button className="mn-i" onClick={api.link}><Ic n="link" s={13}></Ic><span>Link by hand</span><span className="kbd">L</span></button>
            {need && <button className="mn-i" onClick={() => api.decide("Ignored", "")}><Ic n="archive" s={13}></Ic><span>Ignore</span><span className="kbd">#</span></button>}
            <span className="mn-sep"></span>
            <button className="mn-i" onClick={() => api.act("Link copied")}><Ic n="copy" s={13}></Ic><span>Copy link</span></button>
            <button className="mn-i" onClick={() => api.act("Downloading original .eml")}><Ic n="download" s={13}></Ic><span>Download original</span></button>
          </Pop>
        </div>
        <div className="rd-s" ref={scroller}>
          <div className={"rv-g" + (pane.rail ? "" : " norail")} key={m.id}>
            <article className="rv">
              <header className="rv-h">
                <h1 className={m.subject ? "" : "empty"}>{m.subject || "(no subject)"}</h1>
                {older.length > 0 && <div className="thr">{older.map(o => <div key={o.id} className={"thr-r" + (exp === o.id ? " on" : "")}>
                  <button className="thr-h" onClick={() => setExp(exp === o.id ? null : o.id)}><span className="av">{initials(senderOf(o))}</span><b>{senderOf(o)}</b><span className="thr-p">{exp === o.id ? "" : o.body[0].replace(/\{\{([^|}]+)\|[^}]+\}\}/g, "$1")}</span><span className="rv-t">{stamp(o.min)}</span></button>
                  {exp === o.id && <div className="thr-b">{o.body.map((p, i) => <p key={i}>{p.replace(/\{\{([^|}]+)\|[^}]+\}\}/g, "$1")}</p>)}{o.done && <span className="thr-d"><Ic n="check" s={11} w={2.4}></Ic>{(o.by === ME ? "You" : o.by || "Nova") + ": " + o.done}</span>}</div>}
                </div>)}</div>}
                <div className="rv-f"><span className="av">{initials(senderOf(m))}</span><Sender m={m} msgs={msgs} api={api}></Sender><span className="rv-a">{m.from[0] ? m.from[1] : ""}</span><span className="sp"></span><span className="rv-t">{hhmm(at(m.min)) + " · " + ago(m.min) + " ago"}</span></div>
              </header>
              <div className="chips">
                {m.unread ? <Kind k={m.kind}></Kind> : <Pop align="left" btn={(o, t) => <button className={"chip kc" + (o ? " on" : "")} onClick={t} title="Read wrong? Correct it"><Kind k={m.kind}></Kind><Ic n="chevD" s={10}></Ic></button>}>
                  <div className="mn-h">Nova read this as {KIND[m.kind].l.toLowerCase()}. It's actually…</div>
                  {KINDS.filter(([k]) => k !== m.kind).map(([k]) => <button key={k} className="mn-i" onClick={() => api.correct(k)}><Kind k={k}></Kind></button>)}
                </Pop>}
                {m.corrected && <span className="chip cr"><Ic n="check" s={11} w={2.4}></Ic><span>Corrected — Nova will remember</span></span>}
                {!s && !m.cust && m.status !== "Quarantined" && <button className="chip dash" onClick={api.link}><Ic n="link" s={12}></Ic><span>Not linked</span></button>}
                {m.owner && need && <span className="chip"><Av n={m.owner} s={16}></Av><span>{m.owner === ME ? "Assigned to you" : m.owner}</span></span>}
                {m.snooze && <span className="chip sn"><Ic n="clock" s={12}></Ic><span>{m.snooze}</span></span>}
                {m.await && <span className="chip aw"><Ic n="send" s={12}></Ic><span>Waiting on them</span></span>}
              </div>
              {!need && <div className="rcpt"><Ic n="check" s={12} w={2.4}></Ic><span>{(m.status === "Actioned" ? "Handled" : "Ignored") + " by " + (m.by === ME ? "you" : m.by || "Nova") + (m.byMin != null ? " · " + stamp(m.byMin) : "")}{m.done && <em>{" — " + m.done}</em>}</span><button className="lnk" onClick={api.reopen}>Reopen</button></div>}
              <Body m={m} marks={marks}></Body>
              {m.atts.length > 0 && <div className="files">{m.atts.map(a => { const ext = a.n.split(".").pop().toLowerCase(); return (
                <button key={a.n} className={"file" + (a.fail ? " bad" : "")} title={a.fail || a.n} onClick={() => !a.fail && (a.filed ? api.ov({ k: "doc", a }) : setOpen("file:" + a.n))} onContextMenu={e => !a.fail && ctx(e, { title: a.n, sub: a.size + (a.filed ? " · filed on " + SHP[a.filed].pro : ""), items: [
                  { l: "Preview", ic: "eye", on: () => api.ov({ k: "doc", a }) },
                  { l: "Download", ic: "download", on: () => api.act("Downloading " + a.n) },
                  { sep: 1 },
                  { l: "File on", ic: "truck", sub: [m.shp && { h: "Matched" }, m.shp && { l: SHP[m.shp].pro, hint: SHP[m.shp].o.split(",")[0] + " → " + SHP[m.shp].d.split(",")[0], on: () => api.file(a.n, m.shp, (DOCREAD[a.doc] || {}).type) }, { h: "Other shipments" }, ...SHIPMENTS.filter(s => s.id !== m.shp).map(s => ({ l: s.pro, hint: s.o.split(",")[0] + " → " + s.d.split(",")[0], on: () => api.file(a.n, s.id, (DOCREAD[a.doc] || {}).type) }))].filter(Boolean) },
                  { l: "File as", ic: "file", dis: !m.shp, sub: ["Rate confirmation", "Proof of delivery", "Bill of lading", "Invoice", "Tender", "Other"].map(t => ({ l: t, on: () => api.file(a.n, m.shp, t) })) },
                  PAGES[a.n] && { l: "Split pages", ic: "layers", hint: PAGES[a.n].length + " pages", on: () => setOpen("file:" + a.n) },
                  { sep: 1 },
                  { l: "Copy what Nova read", ic: "copy", dis: !DOCREAD[a.doc], on: () => api.act("Copied " + ((DOCREAD[a.doc] || {}).fields || []).length + " fields") },
                  a.filed && { l: "Unfile", ic: "x", d: 1, on: () => api.unfile(a.n) },
                ] })}>
                  <span className={"ft " + (ext === "pdf" ? "pdf" : "img")}>{ext}</span>
                  <span className="file-t"><b>{a.n}</b><span className={a.fail ? "d" : a.filed ? "k" : ""}>{a.fail ? "Refused · over 25 MB" : a.filed ? "Filed on " + SHP[a.filed].pro : a.size}</span></span>
                </button>); })}</div>}
              {R && R.sent && <div className="sent"><span className="sent-h"><Av n={ME} s={20}></Av><b>You replied</b><span className="mono">{R.sentAt}</span>{R.desk && <span className="sent-d"><span className="dm xs"></span>drafted by Nova</span>}</span><span className="sent-b">{R.body}</span></div>}
              {(m.log || []).length > 0 && <div className="trace">{m.log.map(([l, t], i) => <div key={i} className="tr-r" style={{ animationDelay: i * 40 + "ms" }}><span className="tr-d"></span><span>{l}</span><em className="mono">{t}</em></div>)}</div>}
            </article>
            {pane.rail && <Rail m={m} api={{ ...api, startReply: api.startReply }} open={pane.act} setOpen={setOpen} start={start}></Rail>}
          </div>
        </div>
        {need && <ActionBar key={m.id} m={m} api={api} start={start} openAct={pane.act} onSheet={() => setSheet(true)}></ActionBar>}
        {ov && ov.k === "desk" && <DeskPeek m={m} onClose={() => setOv(null)} api={api} start={start}></DeskPeek>}
        {sheet && <div className="bs" onMouseDown={e => e.target === e.currentTarget && setSheet(false)}><div className="bs-in"><span className="bs-g"></span><Rail m={m} api={api} open={pane.act} setOpen={setOpen} start={start}></Rail></div></div>}
      </div>
    </section>
  );
}

function Sender({ m, msgs, api }) {
  const p = profileOf(m, msgs), rs = (window.__rules || []).filter(r => r.from === p.dom);
  return <Pop align="left" btn={(o, t) => <button className={"snd-b" + (o ? " on" : "")} onClick={t}>{senderOf(m)}</button>}>
    <div className="pf">
      <div className="pf-h"><span className="av">{initials(senderOf(m))}</span><div><b>{senderOf(m)}</b><span>{m.from[1]}</span></div></div>
      <div className="pf-tags"><span className="dchip">{p.role}</span>{p.since && <span className="dchip">{p.since}</span>}</div>
      <div className="pf-st"><div><b className="mono">{p.n}</b><span>messages in 90 days</span></div><div><b className="mono">{p.resp}m</b><span>your usual reply time</span></div></div>
      {p.kinds.length > 0 && <><div className="pf-l">Usually sends</div><div className="pf-k">{p.kinds.map(([k, n]) => <Kind key={k} k={k}></Kind>)}</div></>}
      <div className="pf-l">Rules for {p.dom}</div>
      {rs.length ? rs.map(r => <div key={r.id} className="pf-r"><Ic n="bolt" s={11}></Ic>{r.act}</div>) : <div className="pf-r none">None yet</div>}
      {p.recent.length > 0 && <><div className="pf-l">Recent</div>{p.recent.map(x => <button key={x.id} className="mn-i" onClick={() => api.open(x.id)}><span>{x.subject || "(no subject)"}</span><span className="mono mn-m">{ago(x.min).split(" ")[0]}</span></button>)}</>}
    </div>
  </Pop>;
}

function Idle({ waiting, onStart }) {
  if (!waiting.length) return <div className="idle"><span className="ck"><Ic n="check" s={18} w={2.2}></Ic></span><div className="idle-t">You're all caught up</div><div className="idle-s">New mail shows up here when Nova needs you.</div></div>;
  return (
    <div className="idle">
      <div className="idle-n">{waiting.length}</div>
      <div className="idle-t">waiting on you</div>
      <div className="idle-s">{"Nova handled the rest. Oldest has waited " + ago(Math.max(...waiting.map(m => m.min))) + "."}</div>
      <button className="btn ink" onClick={onStart}>Start<span className="kbd">J</span></button>
    </div>
  );
}

function LinkDialog({ m, onClose, onSave }) {
  const [shp, setShp] = React.useState(m.shp || "");
  const [cust, setCust] = React.useState(m.cust || "");
  const [why, setWhy] = React.useState("");
  const [err, setErr] = React.useState(null);
  React.useEffect(() => { const k = e => e.key === "Escape" && onClose(); window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k); }, []);
  const save = () => { if (!shp && !cust) return setErr("rec"); if (!why.trim()) return setErr("why"); onSave({ shp: shp || null, cust: cust || null, why: why.trim() }); };
  return ReactDOM.createPortal(
    <div className="scm" onMouseDown={e => e.target === e.currentTarget && onClose()}>
      <div className="scm-in" role="dialog" aria-modal="true">
        <div className="scm-h"><b>Link by hand</b><span>Which shipment or customer is this about?</span><button className="ib" onClick={onClose}><Ic n="x" s={14}></Ic></button></div>
        <div className="scm-b">
          <label className="fld"><span>Shipment</span><select value={shp} onChange={e => { setShp(e.target.value); setErr(null); }}><option value="">None</option>{SHIPMENTS.map(s => <option key={s.id} value={s.id}>{s.pro + " · " + s.o + " → " + s.d}</option>)}</select></label>
          <label className="fld"><span>Customer</span><select value={cust} onChange={e => { setCust(e.target.value); setErr(null); }}><option value="">None</option>{CUSTOMERS.map(c => <option key={c}>{c}</option>)}</select>{err === "rec" && <span className="err">Choose a shipment, a customer, or both</span>}</label>
          <label className="fld"><span>Why<em>Saved with the link</em></span><textarea autoFocus value={why} onChange={e => { setWhy(e.target.value); setErr(null); }} placeholder="The PRO in the subject matches"></textarea>{err === "why" && <span className="err">Say why this is the right record</span>}</label>
        </div>
        <div className="scm-f"><button className="btn ghost" onClick={onClose}>Cancel</button><button className="btn ink" onClick={save}>Link</button></div>
      </div>
    </div>, document.querySelector(".tv"));
}

function Keys({ onClose }) {
  React.useEffect(() => { const k = e => (e.key === "Escape" || e.key === "?") && onClose(); window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k); }, []);
  const G = [["Move", [["Briefing", "B"], ["Next", "J"], ["Previous", "K"], ["Close", "Esc"], ["Search", "/"], ["Actions panel", "I"], ["Shortcuts", "?"]]], ["Act", [["Do the next step", "↵"], ["Mark handled", "E"], ["Ignore", "#"], ["Reply", "R"], ["Snooze until tomorrow", "S"], ["Assign to me", "M"], ["Select", "X"], ["Link by hand", "L"], ["Ask an agent in Desk", "A"]]]];
  return ReactDOM.createPortal(
    <div className="scm" onMouseDown={e => e.target === e.currentTarget && onClose()}>
      <div className="scm-in"><div className="scm-h"><b>Keyboard shortcuts</b><span>Moving is opening.</span><button className="ib" onClick={onClose}><Ic n="x" s={14}></Ic></button></div>
        <div style={{ padding: "4px 8px 12px", display: "grid", gridTemplateColumns: "1fr 1fr", gap: "0 12px" }}>{G.map(([g, xs]) => <div key={g}><div className="scm-g">{g}</div>{xs.map(([l, k]) => <div key={l} className="scm-r"><span>{l}</span><span className="kbd">{k}</span></div>)}</div>)}</div>
      </div>
    </div>, document.querySelector(".tv"));
}
Object.assign(window, { Kind, Av, Pop, Reader, Idle, LinkDialog, Keys });
