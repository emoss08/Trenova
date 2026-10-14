function Suggestions({ S, A }) {
  const [edit, setEdit] = React.useState(null); const [txt, setTxt] = React.useState("");
  if (!S.sugs.length) return null;
  return (
    <section className="sec sg-s">
      <header className="sh2"><span className="dm"></span><h3>Nova suggests</h3><em className="mono">{S.sugs.length}</em><span className="sp"></span><span className="sh2-n">Agents read none of these until you approve</span></header>
      {S.sugs.map(s => (
        <div key={s.id} className="sgm">
          <p className="sgm-c">{s.content}</p>
          <p className="sgm-e">{s.src === "Ratings" ? `Drawn from ${s.ratings} ratings by ${s.people} people · last rated ${s.when}` : `Learned by ${s.agent} looking back over its work · ${s.when}`}</p>
          {s.reason && <p className="sgm-e">{s.reason}</p>}
          {s.quotes && <ul className="qts">{s.quotes.map(q => <li key={q}>“{q}”</li>)}</ul>}
          <div className="sgm-a">
            <button className="btn sm ghost" onClick={() => A.dismissSug(s)}>Dismiss</button><button className="btn sm" onClick={() => { setEdit(s.id); setTxt(s.content); }}><Ic n="edit" s={12}></Ic>Edit</button><button className="btn sm ink" onClick={() => A.approveSug(s, s.content)}><Ic n="check" s={12} w={2.2}></Ic>Approve</button>
          </div>
        </div>
      ))}
      {edit && <Modal onClose={() => setEdit(null)} title="Approve memory" sub="Edit it so it reads as a rule an agent should follow. Once approved, every agent that asks for memory reads it." foot={<><span className="cmp-n mono">{txt.length}/4000</span><span className="sp"></span><button className="btn sm ghost" onClick={() => setEdit(null)}>Cancel</button><button className="btn sm ink" disabled={!txt.trim()} onClick={() => { A.approveSug(S.sugs.find(x => x.id === edit), txt.trim()); setEdit(null); }}>Approve</button></>}>
        <textarea className="mta" value={txt} autoFocus rows={4} onChange={e => setTxt(e.target.value.slice(0, 4000))}></textarea>
      </Modal>}
    </section>
  );
}
const LEARN_CTX = {
  m2: { from: "Reflection", why: "Spotted that 5 POD holds it cleared this month were re-opened because the signature date didn't match delivery.", by: "Sarah Alvarez", run: "Run #4812", since: "No re-opened POD holds since" },
  m3: { from: "Correction", why: "Two Friday pickups it booked at Joliet DC were refused at the dock after 15:00. Dana Ortiz rebooked them.", by: "Dana Ortiz", run: "Run #4677", since: "3 Friday pickups booked before 14:00 since" },
  m6: { from: "Pattern", why: "Ridgeline accepted 11 of 12 Ohio tenders priced in this range and turned down every one above it.", by: "Mike Chen", run: "Run #4390", since: "Tenders to Ridgeline accepted first time since" },
};
const LEARN_POOL = (() => {
  const ag = ["Dispatch desk", "Billing exceptions", "Inbox triage", "Carrier sourcing", "Detention watch"];
  const seed = [
    ["Fact", "Pilot Travel Center in Gary has no overnight truck parking after 22:00.", ["Location", "Pilot Gary"], "Correction", "A driver was turned away at 22:30 and had to reroute 40 miles."],
    ["Procedure", "When a rate confirmation is missing the fuel surcharge, ask the carrier before invoicing.", null, "Reflection", "Three invoices this month were short-paid for missing surcharge lines."],
    ["Instruction", "Send Walgreens appointment requests to the DC scheduler, not the store contact.", ["Customer", "Walgreens"], "Correction", "Two requests went unanswered until a person re-sent them."],
    ["Fact", "Blue Line Carriers doesn't run reefer loads into Canada.", ["Carrier", "Blue Line Carriers"], "Pattern", "Declined every cross-border reefer tender in the last 60 days."],
    ["Correction", "Lumper fees at Meijer Lansing are billed back to the shipper, not absorbed.", ["Customer", "Meijer"], "Correction", "Billing corrected two invoices it had marked as absorbed."],
    ["Fact", "Emails from noreply@project44 are tracking updates, not customer requests.", null, "Pattern", "Every one it routed to a person was closed without action."],
    ["Procedure", "To clear a weight variance: compare scale ticket to BOL, then request a revised rate con if over 5%.", ["Customer", "Every customer"], "Reflection", "Variance holds took 2 days on average when it skipped the scale ticket."],
    ["Fact", "Elena Park can't take loads leaving before 05:00 on Mondays.", ["Worker", "Elena Park"], "Correction", "Dispatch reassigned two early Monday loads it had offered her."],
  ];
  const days = ["Oct 1", "Sep 29", "Sep 27", "Sep 25", "Sep 24", "Sep 22", "Sep 19", "Sep 18", "Sep 16", "Sep 12", "Sep 10", "Sep 8", "Sep 5", "Sep 3", "Aug 29", "Aug 27", "Aug 22", "Aug 20", "Aug 15", "Aug 13", "Aug 8", "Aug 6", "Aug 1", "Jul 30", "Jul 25", "Jul 23", "Jul 18", "Jul 16", "Jul 11", "Jul 9", "Jul 3", "Jul 1"];
  const by = ["Sarah Alvarez", "Dana Ortiz", "Mike Chen"];
  return days.map((d, i) => { const s = seed[i % seed.length]; return { id: "lh" + i, kind: s[0], content: s[1], subj: s[2], src: ag[(i * 3) % ag.length], learned: true, reads: (i * 7) % 23 + 1, last: null, rec: d, ctx: { from: s[3], why: s[4], by: by[i % 3], run: "Run #" + (4300 - i * 37) } }; });
})();
const LEARN_PAGE = 8;
function LearnSkel({ n }) { return Array.from({ length: n }, (_, i) => <div key={i} className="lrn-r sk"><span className="sk-b" style={{ width: 34 }}></span><div className="lrn-m"><span className="sk-b" style={{ width: 90 }}></span><span className="sk-b" style={{ width: (60 + (i * 13) % 30) + "%", height: 10 }}></span><span className="sk-b" style={{ width: "40%" }}></span></div></div>); }
function Learned({ S, onOpen }) {
  const all = React.useMemo(() => [...S.mems.filter(m => m.learned).map(m => ({ ...m, ctx: LEARN_CTX[m.id] || { from: "Reflection", by: "you" } })), ...LEARN_POOL], [S.mems]);
  const agents = [...new Set(all.map(m => m.src))];
  const [who, setWho] = React.useState("all"); const [n, setN] = React.useState(0); const [busy, setBusy] = React.useState(true);
  const box = React.useRef(); const end = React.useRef(); const tm = React.useRef();
  const rows = all.filter(m => who === "all" || m.src === who);
  const more = () => { if (busy) return; setBusy(true); clearTimeout(tm.current); tm.current = setTimeout(() => { setN(x => x + LEARN_PAGE); setBusy(false); }, 650); };
  React.useEffect(() => { setN(0); setBusy(true); clearTimeout(tm.current); tm.current = setTimeout(() => { setN(LEARN_PAGE); setBusy(false); }, 700); box.current && box.current.scrollTo({ top: 0 }); return () => clearTimeout(tm.current); }, [who]);
  React.useEffect(() => { if (!end.current || !box.current) return; const io = new IntersectionObserver(es => { if (es[0].isIntersecting && n < rows.length) more(); }, { root: box.current, rootMargin: "80px" }); io.observe(end.current); return () => io.disconnect(); });
  if (!all.length) return null;
  const shown = rows.slice(0, n), done = n >= rows.length && !busy;
  return (
    <section className="sec lrn">
      <header className="sh2"><Ic n="sparkle" s={14}></Ic><h3>What agents learned</h3><em className="mono">{rows.length}</em><span className="sp"></span><select className="msel sm" value={who} onChange={e => setWho(e.target.value)}><option value="all">All agents</option>{agents.map(a => <option key={a} value={a}>{a}</option>)}</select></header>
      <div className="lrn-g" ref={box}>
        {shown.map(m => { const c = m.ctx; return (
          <div key={m.id} className={"lrn-r" + (m.fresh ? " fresh" : "")} onClick={() => !m.id.startsWith("lh") && onOpen(m.id)}>
            <span className="lrn-d mono">{m.rec}</span>
            <div className="lrn-m">
              <span className="lrn-a">{m.src === "ratings" ? "From ratings" : m.src}</span>
              <p className="lrn-c">{m.content}</p>
              {c.why && <p className="lrn-y">{c.why}</p>}
              <p className="lrn-s">Approved by {c.by} · <span className="mono">read {m.reads}×</span></p>
            </div>
          </div>
        ); })}
        {busy && <LearnSkel n={n ? 3 : 5}></LearnSkel>}
        <div ref={end} className="lrn-end">{done && <span>That's everything agents have learned{who !== "all" ? " · " + who : ""}</span>}</div>
      </div>
      <p className="lrn-l mono">{busy ? "Loading…" : "Showing " + Math.min(n, rows.length) + " of " + rows.length}</p>
    </section>
  );
}
function Composer({ init, onSave, onCancel }) {
  const [kind, setKind] = React.useState(init.kind || "Instruction"); const [txt, setTxt] = React.useState(init.content || "");
  const [st, setSt] = React.useState(init.subj ? init.subj[0] : ""); const [sl, setSl] = React.useState(init.subj ? init.subj[1] : "");
  const ok = txt.trim() && (!st || sl.trim());
  return (
    <div className="cmp-m">
      <textarea className="mta" placeholder="What should the agents know?" value={txt} autoFocus rows={2} onChange={e => setTxt(e.target.value.slice(0, 4000))} onKeyDown={e => { if (e.key === "Enter" && (e.metaKey || e.ctrlKey) && ok) onSave({ kind, content: txt.trim(), subj: st ? [st, sl.trim()] : null }); if (e.key === "Escape") onCancel(); }}></textarea>
      <div className="cmp-r">
        <Seg v={kind} opts={Object.keys(MEM_KIND).map(k => [k, k])} onChange={setKind} cls="sm"></Seg>
        <span className="cmp-l">About</span>
        <select className="msel" value={st} onChange={e => setSt(e.target.value)}><option value="">Every agent</option>{Object.entries(MEM_SUBJ).map(([k, l]) => <option key={k} value={k}>A {l}</option>)}</select>
        {st && <input className="inp" placeholder={"Which " + MEM_SUBJ[st] + "?"} value={sl} onChange={e => setSl(e.target.value)}></input>}
        <span className="sp"></span>
        <span className="cmp-n mono">{txt.length}/4000</span>
        <button className="btn sm ghost" onClick={onCancel}>Cancel</button>
        <button className="btn sm ink" disabled={!ok} onClick={() => onSave({ kind, content: txt.trim(), subj: st ? [st, sl.trim()] : null })}>Save<span className="kbd">{MOD}↵</span></button>
      </div>
    </div>
  );
}
function MemRow({ m, A, editing, setEditing }) {
  const scope = m.subj ? <><Ic n={m.subj[0] === "Worker" ? "user" : m.subj[0] === "Carrier" ? "truck" : m.subj[0] === "Location" ? "map" : "users"} s={11}></Ic>{m.subj[1]}</> : m.tool ? <><Ic n="tool" s={11}></Ic><span className="mono">{m.tool}</span></> : <><Ic n="sparkle" s={11}></Ic>Every agent</>;
  return (
    <div className={"mr" + (m.fresh ? " fresh" : "") + (editing ? " sel" : "")} onClick={() => setEditing(m.id)}>
      <span className={"mk-k " + MEM_KIND[m.kind]}>{m.kind}</span>
      <div className="mr-m">
        <p>{m.content}</p>
        <div className="mr-s"><span>{scope}</span><span>{m.learned ? <><Ic n="sparkle" s={10}></Ic>Learned by {m.src}</> : "Added by " + m.src}</span><span className="mono">{m.reads ? "read " + m.reads + "×" : "not read yet"}{m.last ? " · " + m.last : ""}</span>{m.until && <span>until {m.until}</span>}</div>
      </div>
      <div className="mr-a" onClick={e => e.stopPropagation()}>
        <button className="ib" title="Edit" onClick={() => setEditing(m.id)}><Ic n="edit" s={13}></Ic></button>
        <button className="ib" title="Retire" onClick={() => A.retireMem(m)}><Ic n="trash" s={13}></Ic></button>
      </div>
    </div>
  );
}
function MemoryTab({ S, A, searchRef }) {
  const [comp, setComp] = React.useState(null); const [editing, setEditing] = React.useState(null);
  React.useEffect(() => { const n = () => setComp({}); window.addEventListener("aic-new", n); return () => window.removeEventListener("aic-new", n); }, []);
  const embed = routeOf(TASKS.find(t => t.k === "Embedding"), S.providers).first;
  const save = v => { A.addMem(v); setComp(null); };
  return (
    <div className="tabp">
      <p className="lead">Agents read these before they answer.{!embed && <> They're found by their words until <button className="lnk" onClick={() => A.go("retrieval")}>search by meaning</button> is on.</>}{!S.pol.learn && <> Learning is off, so agents won't suggest new ones.</>}</p>
      {S.learnedOn && <Learned S={S} onOpen={setEditing}></Learned>}
      <Suggestions S={S} A={A}></Suggestions>
      <section className="sec">
                {!S.mems.length ? (
          <div className="mem-e">
            <b>Nothing recorded yet</b>
            <span>Tell agents something once and every one of them remembers it — a customer's rule, a dock's hours, how your team clears a hold.{S.pol.learn ? " With Learn from their work on, they'll also suggest lessons for you to approve." : ""}</span>
            <div className="mem-x">{MEM_EXAMPLES.map(([kk, t]) => <button key={t} onClick={() => setComp({ kind: kk, content: t.replace(" …", " ").replace("…", "") })}><span className={"mk-k " + MEM_KIND[kk]}>{kk}</span>{t}</button>)}</div>
          </div>
        ) : <DT rows={S.mems} name="memory" searchRef={searchRef} find={x => x.content + " " + (x.subj ? x.subj[1] : "") + (x.tool || "") + x.src} onOpen={x => setEditing(x.id)} sel={S.mems.find(x => x.id === editing)}
          filters={[["all", "All kinds", () => true], ...Object.keys(MEM_KIND).map(kk => [kk, kk, x => x.kind === kk])]}
          tools={<button className="btn ink" onClick={() => setComp({})}><Ic n="plus" s={13}></Ic>New memory</button>}
          cols={[{ k: "m", l: "Memory", w: "40%", c: x => <span className="mm">{x.content}</span> }, { k: "k", l: "Kind", c: x => <span className={"mk-k " + MEM_KIND[x.kind]}>{x.kind}</span>, sv: x => x.kind }, { k: "a", l: "About", c: x => <span className="mu">{x.subj ? x.subj[1] + " (" + MEM_SUBJ[x.subj[0]] + ")" : x.tool ? "tool " + x.tool : "Every agent"}</span>, sv: x => x.subj ? x.subj[1] : "", hide: 1 }, { k: "s", l: "Status", c: x => <Bd m={["Active", "k"]}></Bd> }, { k: "r", l: "Recorded by", c: x => <span className="mu">{x.learned ? "Agent · " + x.src : x.src}</span>, sv: x => x.src, hide: 1 }, { k: "rd", l: "Read", r: true, c: x => <span className="mono">{x.reads} {x.reads === 1 ? "time" : "times"}</span>, sv: x => x.reads }, { k: "rc", l: "Recorded", c: x => <span className="mu">{x.rec}</span>, hide: 1 }]}></DT>}
      </section>
      {comp && <Sheet onClose={() => setComp(null)} head={<div className="sh-t"><b>New memory</b><span>Every agent that asks for memory reads it</span></div>}><div className="sh-p"><Composer init={comp} onCancel={() => setComp(null)} onSave={save}></Composer></div></Sheet>}
      {(() => { const m = S.mems.find(x => x.id === editing); if (!m) return null; return <Sheet onClose={() => setEditing(null)} head={<div className="sh-t"><b>Edit memory</b><span>{m.learned ? "Learned by " + m.src : "Added by " + m.src} · {m.rec} · read {m.reads}×</span></div>}><div className="sh-p"><Composer key={m.id} init={m} onCancel={() => setEditing(null)} onSave={v => { A.saveMem({ ...m, ...v }); setEditing(null); }}></Composer></div><div className="ad-bar sh-f"><span className="sp"></span><button className="xa d" onClick={() => { A.retireMem(m); setEditing(null); }}><Ic n="trash" s={13}></Ic>Retire memory</button></div></Sheet>; })()}
    </div>
  );
}
Object.assign(window, { MemoryTab });
