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
