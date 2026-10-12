function Ov({ onClose, cls = "", children }) {
  React.useEffect(() => { const k = e => e.key === "Escape" && (e.stopPropagation(), onClose()); window.addEventListener("keydown", k, true); return () => window.removeEventListener("keydown", k, true); }, []);
  return ReactDOM.createPortal(<div className={"scm " + cls} onMouseDown={e => e.target === e.currentTarget && onClose()}>{children}</div>, document.querySelector(".tv"));
}

function DocViewer({ a, m, onClose, api }) {
  const r = DOCREAD[a.doc] || { type: "Document", fields: [], pages: 1 };
  const pages = (PAGES[a.n] || Array.from({ length: r.pages || 1 }, (_, i) => ["Page " + (i + 1)]));
  const [p, setP] = React.useState(0);
  const [hl, setHl] = React.useState(null);
  return (
    <Ov onClose={onClose} cls="dv">
      <div className="dv-in">
        <header className="dv-h"><span className={"ft " + (/pdf$/i.test(a.n) ? "pdf" : "img")}>{a.n.split(".").pop()}</span><div><b>{a.n}</b><span>{a.size + " · " + pages.length + (pages.length > 1 ? " pages" : " page") + " · from " + senderOf(m)}</span></div><span className="sp"></span><button className="btn sm ghost" onClick={() => api.act("Downloading " + a.n)}><Ic n="download" s={12}></Ic>Download</button><button className="ib" onClick={onClose}><Ic n="x" s={14}></Ic></button></header>
        <div className="dv-b">
          <div className="dv-th">{pages.map(([l], i) => <button key={i} className={i === p ? "on" : ""} onClick={() => setP(i)}><span className="pg-th mono">{i + 1}</span><em>{l}</em></button>)}</div>
          <div className="dv-pg"><div className="dv-paper" key={p}><span className="mono">{(pages[p] || [])[0] || "Page " + (p + 1)}</span><span>page preview</span>{hl != null && p === 0 && <i className="dv-box" style={{ top: 18 + hl * 9 + "%" }}></i>}</div></div>
          <aside className="dv-r">
            <div className="ar-h">What Nova read{r.conf && <span className="mono">{Math.round(r.conf * 100)}%</span>}</div>
            <dl className="kv">{r.fields.map(([k, v, t], i) => <div key={k} onMouseEnter={() => setHl(i)} onMouseLeave={() => setHl(null)}><dt>{k}</dt><dd className={t || ""}>{v}</dd></div>)}</dl>
            {!r.fields.length && <p className="ap sm">Nothing structured on this file.</p>}
            {a.filed ? <div className="aok"><span className="ab-ok"><Ic n="check" s={11} w={2.6}></Ic></span><span>{"Filed on " + SHP[a.filed].pro}</span></div> : m.shp && <button className="btn ink sm" onClick={() => { api.file(a.n, m.shp, r.type); onClose(); }}>{"File on " + SHP[m.shp].pro}</button>}
          </aside>
        </div>
      </div>
    </Ov>
  );
}

function RecPeek({ m, onClose, api }) {
  const s = SHP[m.shp], x = RECX[m.shp];
  const files = x.docs.concat(m.atts.filter(a => a.filed === m.shp).map(a => (a.ftype || (DOCREAD[a.doc] || {}).type || "Doc")));
  const ev = [["Picked up", s.o, x.p > 0], ["In transit", x.p >= 1 ? "Arrived" : x.eta, x.p > 0.3], [x.p >= 1 ? "Delivered" : "Delivery", s.d, x.p >= 1]];
  return (
    <Ov onClose={onClose} cls="pk">
      <div className="pk-in">
        <header className="pk-h"><div><b className="mono">{s.pro}</b><span>{x.cust + " · " + x.eq}</span></div><span className={"stp " + (x.late ? "w" : x.p >= 1 ? "k" : "b")}><i></i>{x.p >= 1 ? "Delivered" : x.late ? "Running late" : "In transit"}</span><button className="ib" onClick={onClose}><Ic n="x" s={14}></Ic></button></header>
        <div className="rt"><div><b>{s.o.split(",")[0]}</b><span>{s.o.split(", ")[1]}</span></div><div className="rt-l"><i style={{ width: x.p * 100 + "%" }} className={x.late ? "w" : ""}></i>{x.p < 1 && <em style={{ left: x.p * 100 + "%" }}><Ic n="truck" s={11}></Ic></em>}</div><div className="r"><b>{s.d.split(",")[0]}</b><span>{s.d.split(", ")[1]}</span></div></div>
        <ol className="trl">{ev.map(([l, v, d], i) => <li key={i} className={d ? "ok" : "cur"}><span>{l}</span><em>{v}</em></li>)}</ol>
        <dl className="kv two"><div><dt>Driver</dt><dd>{x.drv}</dd></div><div><dt>BOL</dt><dd className="mono">{s.bol.replace("BOL ", "")}</dd></div><div><dt>PO</dt><dd className="mono">{s.po ? s.po.replace("PO ", "") : "—"}</dd></div><div><dt>ETA</dt><dd className={x.late ? "w" : ""}>{x.eta}</dd></div></dl>
        <div className="ar-h">On file</div>
        <div className="dchips">{[...new Set(files)].map(d => <span key={d} className="dchip"><Ic n="check" s={10} w={2.6}></Ic>{d}</span>)}</div>
        <footer className="pk-f"><button className="btn sm ghost" onClick={() => api.act("Link copied")}><Ic n="copy" s={12}></Ic>Copy link</button><span className="sp"></span><button className="btn sm" onClick={() => api.act("Opening " + s.pro + " in Shipments")}>Open in Shipments<Ic n="ext" s={12}></Ic></button></footer>
      </div>
    </Ov>
  );
}

function CustPeek({ m, msgs, onClose, api }) {
  const xs = msgs.filter(x => x.cust === m.cust), open = SHIPMENTS.filter(s => RECX[s.id].cust === m.cust);
  return (
    <Ov onClose={onClose} cls="pk">
      <div className="pk-in">
        <header className="pk-h"><div><b>{m.cust}</b><span>Customer · {profileOf(m, msgs).since || "New"}</span></div><button className="ib" onClick={onClose}><Ic n="x" s={14}></Ic></button></header>
        <div className="pf-st"><div><b className="mono">{open.length}</b><span>shipments this week</span></div><div><b className="mono">{xs.filter(isNeed).length}</b><span>waiting in Triage</span></div></div>
        {open.length > 0 && <><div className="ar-h">Shipments</div>{open.map(s => <button key={s.id} className="mn-i" onClick={() => api.act("Opening " + s.pro)}><Ic n="truck" s={13}></Ic><span>{s.pro + " · " + s.o.split(",")[0] + " → " + s.d.split(",")[0]}</span><span className="mono mn-m">{RECX[s.id].p >= 1 ? "done" : "moving"}</span></button>)}</>}
        <div className="ar-h">Messages</div>{xs.slice(0, 4).map(x => <button key={x.id} className="mn-i" onClick={() => { onClose(); api.open(x.id); }}><Ic n="inbox" s={13}></Ic><span>{x.subject}</span><span className="mono mn-m">{ago(x.min).split(" ")[0]}</span></button>)}
        <footer className="pk-f"><span className="sp"></span><button className="btn sm" onClick={() => api.act("Opening " + m.cust + " in Customers")}>Open customer<Ic n="ext" s={12}></Ic></button></footer>
      </div>
    </Ov>
  );
}

const AGENTS = [{ n: "Dispatch", i: "D", h: 230, d: "Loads, tenders, carriers" }, { n: "Billing", i: "B", h: 150, d: "Invoices, detention, pay" }, { n: "Customer service", i: "C", h: 30, d: "Status, ETAs, replies" }];
const agentFor = m => m.kind === "Tender" || m.kind === "RateConfirmation" ? 0 : m.kind === "Invoice" || m.kind === "DetentionDispute" || m.kind === "ProofOfDelivery" ? 1 : 2;
function AgTile({ a, s = 22 }) { return <span className="agt" style={{ "--h": a.h, width: s, height: s, borderRadius: Math.round(s * 0.3), fontSize: Math.round(s * 0.42) }}>{a.i}</span>; }
function DeskPeek({ m, onClose, api, start }) {
  const A = actsOf(m), open = A.filter(a => !a.done), next = open[0];
  const [ag, setAg] = React.useState(() => agentFor(m)), AG = AGENTS[ag];
  const [ph, setPh] = React.useState(0);
  const [q, setQ] = React.useState("");
  const [extra, setExtra] = React.useState([]);
  React.useEffect(() => { const t = [setTimeout(() => setPh(1), 500), setTimeout(() => setPh(2), 1700)]; return () => t.forEach(clearTimeout); }, []);
  const s = m.shp && SHP[m.shp];
  const say = ph < 2 ? null : next ? <>{"This is a " + KIND[m.kind].l.toLowerCase() + " from " + senderOf(m) + (s ? " about " : "") }{s && <b className="mono">{s.pro}</b>}{". " + open.length + (open.length === 1 ? " thing is" : " things are") + " left. The first is "}<b>{next.t.charAt(0).toLowerCase() + next.t.slice(1)}</b>{" — want me to start there?"}</> : "Everything this message asked for is done. I can mark it handled, or draft a follow-up.";
  const send = () => { if (!q.trim()) return; setExtra(x => [...x, ["u", q], ["a", "On it — I'll keep this in the conversation so the next person can see what happened."]]); setQ(""); };
  return ReactDOM.createPortal(
    <div className="dkp-w" onMouseDown={e => e.target === e.currentTarget && onClose()}>
      <aside className="dkp">
        <header className="dkp-h"><b>Desk</b><Pop align="left" btn={(o, t) => <button className={"agp" + (o ? " on" : "")} onClick={t}><AgTile a={AG} s={18}></AgTile><span>{AG.n}</span><Ic n="chevD" s={11}></Ic></button>}><div className="mn-h">Your agents</div>{AGENTS.map((g, i) => <button key={g.n} className="mn-i" onClick={() => setAg(i)}><AgTile a={g} s={20}></AgTile><span>{g.n}<em>{g.d}</em></span>{i === ag && <Ic n="check" s={12} w={2.4}></Ic>}</button>)}</Pop><span className="sp"></span><button className="btn ghost sm" onClick={() => api.act("Opening the full conversation in Desk")}>Open in Desk<Ic n="ext" s={11}></Ic></button><button className="ib" onClick={onClose}><Ic n="x" s={14}></Ic></button></header>
        <div className="dkp-s">
          <div className="dkp-u"><p>Help me finish this one.</p><div className="dkp-ctx"><span className="dkp-c"><Ic n="inbox" s={11}></Ic>{m.subject || "(no subject)"}</span>{open.map(a => <span key={a.id} className="dkp-c a"><Ic n="check" s={10}></Ic>{a.t}</span>)}</div></div>
          {ph >= 1 && <div className="dkp-a"><AgTile a={AG}></AgTile><div>{ph < 2 ? <p className="shim">Reading the message, {s ? s.pro + ", " : ""}and what's been done…</p> : <><p>{say}</p>{next && <div className="dkp-b"><button className="btn sm ink" onClick={() => { onClose(); start(next); }}>{next.desk ? "Yes, do it" : "Yes, open it"}</button>{open.length > 1 && <button className="btn sm ghost" onClick={() => setExtra(x => [...x, ["a", "Here's the plan: " + open.map((a, i) => (i + 1) + ". " + a.t).join("  ") + ". I'll stop and ask before anything is sent or created."]])}>Show the plan</button>}</div>}{!next && <div className="dkp-b"><button className="btn sm ink" onClick={() => { onClose(); api.decide("Actioned", "Closed from Desk."); }}>Mark handled</button></div>}</>}</div></div>}
          {extra.map(([w, t], i) => w === "u" ? <div key={i} className="dkp-u"><p>{t}</p></div> : <div key={i} className="dkp-a"><AgTile a={AG}></AgTile><div><p>{t}</p></div></div>)}
        </div>
        <div className={"dkp-cmp" + (ph === 1 ? " busy" : "")}><textarea value={q} onChange={e => setQ(e.target.value)} onKeyDown={e => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); send(); } }} placeholder={"Ask " + AG.n + " about this message…"} rows={2}></textarea><div className="dkp-cb"><span className="dkp-c"><Ic n="inbox" s={10}></Ic>{senderOf(m)}</span><span className="sp"></span><button className="snd2" disabled={!q.trim()} onClick={send}><Ic n="up" s={13} w={2.2}></Ic></button></div></div>
      </aside>
    </div>, document.querySelector(".tv"));
}

function RuleMatches({ r, onClose }) {
  const p = previewRule(r), mb = MAILBOXES.find(b => b.id === r.mb);
  const senders = r.from === "any sender" || r.from === "any customer" ? ["northwindfoods.example", "acme.example", "harborline.example", "crestline.example", "blueridgebrokerage.example"] : [r.from];
  const rows = Array.from({ length: Math.min(p.n, 12) }, (_, i) => ({ from: senders[i % senders.length], d: (i * 2.4 + 0.6).toFixed(0) + "d ago", bad: i < p.wrong, sub: (KIND[r.kind] ? KIND[r.kind].l : "Email") + " · " + ["PRO S-1041" + (20 + i), "ref " + (4400 + i * 7), "PO " + (88100 + i * 13)][i % 3] }));
  return (
    <Ov onClose={onClose}>
      <div className="scm-in wide"><div className="scm-h"><b>{"Would have handled " + p.n + " messages"}</b><span>{(KIND[r.kind] ? KIND[r.kind].l : "") + " from " + r.from + " on " + mb.name + " → " + r.act.toLowerCase() + " · last 30 days"}</span><button className="ib" onClick={onClose}><Ic n="x" s={14}></Ic></button></div>
        <div className="rm-l">{rows.map((x, i) => <div key={i} className={"rm" + (x.bad ? " bad" : "")}><span className="mono">{x.d}</span><span><b>{x.from}</b><em>{x.sub}</em></span>{x.bad ? <span className="rm-w"><Ic n="warn" s={11}></Ic>{i % 2 ? "Kind looks wrong" : "No matching shipment"}</span> : <Ic n="check" s={12} w={2.2}></Ic>}</div>)}{p.n > 12 && <div className="rm-more">{"+ " + (p.n - 12) + " more"}</div>}</div>
        <div className="scm-f">{p.wrong > 0 && <span className="ax-n w" style={{ marginRight: "auto" }}>{p.wrong + " would have needed a person — narrow the sender to avoid them"}</span>}<button className="btn ink" onClick={onClose}>Done</button></div>
      </div>
    </Ov>
  );
}

function ConnectMb({ onClose, onDone }) {
  const [st, setSt] = React.useState(-1);
  const [ch, setCh] = React.useState("email");
  const [addr, setAddr] = React.useState("dispatch@trenovalogistics.com");
  const [nm, setNm] = React.useState("Dispatch");
  const fwd = nm.toLowerCase().replace(/\W+/g, "") + "@dev.trenova.app";
  const S = ["Address", "Forward", "Test"];
  return (
    <Ov onClose={onClose}>
      <div className="scm-in"><div className="scm-h"><b>Add a source</b><span>{st < 0 ? "Anywhere freight work arrives. Nova reads all of it the same way." : "Takes a couple of minutes. Nothing changes for the people already using this address."}</span><button className="ib" onClick={onClose}><Ic n="x" s={14}></Ic></button></div>
        {st >= 0 && <div className="cm-st">{S.map((l, i) => <span key={l} className={i < st ? "d" : i === st ? "on" : ""}><i className="mono">{i < st ? "✓" : i + 1}</i>{l}</span>)}</div>}
        <div className="scm-b">
          {st < 0 && <div className="src-g">{SOURCES.map(([k, ic, l, d, soon]) => <button key={k} className={"src" + (ch === k ? " on" : "") + (soon ? " soon" : "")} disabled={soon} onClick={() => setCh(k)}><span className="src-ic"><Ic n={ic} s={15}></Ic></span><span className="src-t"><b>{l}</b><span>{d}</span></span>{soon ? <em>Soon</em> : ch === k && <Ic n="check" s={13} w={2.4}></Ic>}</button>)}</div>}
          {st === 0 && <><label className="fld"><span>Which address should Nova watch?</span><input className="cm-in" value={addr} onChange={e => setAddr(e.target.value)}></input></label><label className="fld"><span>Name it</span><input className="cm-in" value={nm} onChange={e => setNm(e.target.value)}></input></label></>}
          {st === 1 && <><p className="ap sm">In your email provider, forward <b>{addr}</b> to this address. Keep a copy in the original inbox.</p><div className="cm-addr"><span className="mono">{fwd}</span><button className="btn sm ghost" onClick={() => {}}><Ic n="copy" s={12}></Ic>Copy</button></div><div className="cm-prov">{["Google Workspace", "Microsoft 365", "Other"].map(p => <span key={p} className="dchip">{p}</span>)}</div></>}
          {st === 2 && <div className="cm-test">{st === 2 && <TestMail onOk={() => setSt(3)}></TestMail>}</div>}
          {st === 3 && <div className="aok"><span className="ab-ok"><Ic n="check" s={11} w={2.6}></Ic></span><span>{nm + " is connected. New mail on " + addr + " shows up in Waiting on you."}</span></div>}
        </div>
        <div className="scm-f"><button className="btn ghost" onClick={() => st >= 0 && st < 3 ? setSt(st - 1) : onClose()}>{st >= 0 && st < 3 ? "Back" : "Cancel"}</button>{st < 2 && <button className="btn ink" onClick={() => setSt(st + 1)}>Continue</button>}{st === 3 && <button className="btn ink" onClick={() => onDone(nm)}>Done</button>}</div>
      </div>
    </Ov>
  );
}
const SOURCES = [["email", "inbox", "Email", "A shared address — dispatch@, pod@, ap@"], ["edi", "swap", "EDI", "204 tenders, 214 status, 210 invoices", true], ["portal", "ext", "Customer portals", "Loads posted on a shipper's site", true], ["sms", "chat", "Texts", "Drivers texting photos and updates", true]];
function TestMail({ onOk }) {
  const [s, setS] = React.useState(0);
  React.useEffect(() => { if (s !== 1) return; const t = setTimeout(onOk, 1800); return () => clearTimeout(t); }, [s]);
  return s === 0 ? <><p className="ap sm">Send any email to the address. We'll wait for it here.</p><button className="btn sm ink" onClick={() => setS(1)}>I sent it</button></> : <p className="ap sm"><span className="shim">Waiting for it to arrive…</span></p>;
}
Object.assign(window, { DocViewer, RecPeek, CustPeek, DeskPeek, RuleMatches, ConnectMb });
