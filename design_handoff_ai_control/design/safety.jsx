function Eg({ e }) { const [l, h, c] = EGRESS[e]; return <span className="eg" style={{ "--h": h, "--c": c }}>{l}</span>; }
function Ans({ v }) { const [l, k] = ANS[v]; return <span className={"an " + k}>{l}</span>; }
function Held({ xs }) { return xs.length ? <span className="hb">{xs.map(x => <span key={x} className="tg">{HELD_L[x]}</span>)}</span> : <span className="dim">—</span>; }
function readsOut(t) { return t.ext === "Never" ? null : (t.ext === "Always" ? "Always" : "When marked") + (t.src ? ", from " + t.src : ""); }
function ToolSheet({ t, S, A, onClose, focus }) {
  const holders = S.agents.filter(a => toolsOf(a).includes(t));
  return (
    <Sheet onClose={onClose} head={<><span className="src-i"><Ic n={t.kind === "Query" ? "search" : t.kind === "Runtime" ? "cog" : "tool"} s={15}></Ic></span><div className="sh-t"><b>{t.l}</b><span className="mono">{t.n}</span></div></>}>
      <div className="sh-m"><Eg e={t.e}></Eg><span>{t.kind === "Query" ? "Reads" : t.kind === "Action" ? "Changes" : "Runtime"}</span></div>
      <dl className="sfx">
        <div><dt>Most it may do</dt><dd>{TIER_L[t.tier]}</dd></div>
        <div><dt>Needs</dt><dd>{t.needs || "Nothing: own records only"}</dd></div>
        <div><dt>Reads outside content</dt><dd>{readsOut(t) || "Never"}</dd></div>
        <div><dt>Who sees its work</dt><dd>{OUTSIDE.includes(t.e) ? "Leaves the organization — never past approval" : t.e === "N" ? "Nobody; it only reads" : "Stays inside the organization"}</dd></div>
      </dl>
      <div className="sh-p"><h4 className="sh-k">Held by {holders.length} agent{holders.length === 1 ? "" : "s"}</h4>
        {holders.length ? <div className="hl-l">{holders.map(a => { const r = answerOf(a, t); return (
          <div key={a.id} className={"hl-r" + (focus === a.id ? " on" : "") + (a.on ? "" : " dim")}>
            <Tile a={a} s={24}></Tile><div className="hl-n"><b>{a.n}</b><span>Ceiling {TIER_L[ceilOf(a)]}{a.on ? "" : " · off"}</span></div>
            <div className="ba"><Ans v={r.b}></Ans>{r.a !== r.b && <><Ic n="arrowR" s={11}></Ic><Ans v={r.a}></Ans></>}</div>
          </div>); })}</div> : <p className="ad-h">No agent holds this tool.</p>}
      </div>
      <div className="ad-bar sh-f"><button className="xa" onClick={() => { onClose(); A.edit({ k: "tool", n: t.n }); }}><Ic n="edit" s={13}></Ic>Change tool rule</button><span className="sp"></span><button className="xa" onClick={() => A.go("audit")}><Ic n="receipt" s={13}></Ic>Audit trail</button></div>
    </Sheet>
  );
}
function EgressMap({ act, onPick }) {
  const acts = TOOLS.filter(t => t.kind === "Action"), tot = acts.length;
  const data = EG_ORDER.filter(e => e !== "N").map(e => ({ e, n: acts.filter(t => t.e === e).length })).filter(d => d.n);
  return (
    <div className={"egm" + (act ? " foc" : "")}>
      {data.map(d => { const [l, h, c] = EGRESS[d.e]; return (
        <button key={d.e} className={"egm-s" + (act === d.e ? " on" : "")} style={{ flexGrow: Math.max(d.n, tot * 0.09), "--h": h, "--c": c }} onClick={() => onPick(act === d.e ? null : d.e)}>
          <span className="egm-l"><b className="mono">{d.n}</b><span>{l}</span></span><span className="egm-b"></span>
        </button>); })}
    </div>
  );
}
function Rules({ S, A, searchRef, eg, setEg }) {
  const [open, setOpen] = React.useState(null);
  const rows = TOOLS.filter(t => !eg || t.e === eg);
  return (
    <>
      <DT key={eg || "all"} rows={rows} name="tool" searchRef={searchRef} find={t => t.l + t.n + (t.needs || "")} onOpen={setOpen} sel={open}
        filters={[["all", "All tools", () => true], ["Query", "Reads", t => t.kind === "Query"], ["Action", "Changes", t => t.kind === "Action"], ["Runtime", "Runtime", t => t.kind === "Runtime"]]}
        tools={eg && <span className="tok">{EGRESS[eg][0]}<button onClick={() => setEg(null)}><Ic n="x" s={10}></Ic></button></span>}
        cols={[{ k: "t", l: "Tool", c: t => <div className="tl"><b>{t.l}</b><span className="mono">{t.n}</span></div>, sv: t => t.l }, { k: "e", l: "Who sees it", c: t => <Eg e={t.e}></Eg>, sv: t => EG_ORDER.indexOf(t.e) }, { k: "m", l: "Max tier", c: t => TIER_L[t.tier], sv: t => TIER_O.indexOf(t.tier) }, { k: "n", l: "Needs", c: t => <span className="mu">{t.needs || "Nothing: own records only"}</span>, sv: t => t.needs || "", hide: 1 }, { k: "o", l: "Reads outside content", c: t => readsOut(t) || <span className="dim">—</span>, sv: t => t.ext, hide: 1 }, { k: "a", l: "Agents", r: true, c: t => { const hs = S.agents.filter(a => toolsOf(a).includes(t)); return <span className="stk">{hs.slice(0, 4).map(a => <Tile key={a.id} a={a} s={20}></Tile>)}{hs.length > 4 && <em className="mono">+{hs.length - 4}</em>}{!hs.length && <span className="dim">—</span>}</span>; }, sv: t => S.agents.filter(a => toolsOf(a).includes(t)).length }]}></DT>
      {open && <ToolSheet t={open} S={S} A={A} onClose={() => setOpen(null)}></ToolSheet>}
    </>
  );
}
function ByAgent({ S, A, picked, setPicked }) {
  const [menu, setMenu] = React.useState(false); const [open, setOpen] = React.useState(null); const [warn, setWarn] = React.useState(null);
  const ags = picked.map(id => S.agents.find(a => a.id === id)).filter(Boolean);
  const rows = ags.flatMap(a => toolsOf(a).map(t => ({ a, t, r: answerOf(a, t) })));
  return (
    <>
      <div className="tb">
        <div className="pk">{ags.map(a => <span key={a.id} className="pk-c"><Tile a={a} s={18}></Tile>{a.n}<button onClick={() => setPicked(picked.filter(x => x !== a.id))}><Ic n="x" s={10}></Ic></button></span>)}
          {picked.length < 4 && <div className="rel"><button className="btn sm" onClick={() => setMenu(!menu)}><Ic n="plus" s={12}></Ic>{picked.length ? "Compare another" : "Pick an agent"}</button>
            {menu && <Menu onClose={() => setMenu(false)} items={[{ h: "Up to 4 agents" }, ...S.agents.filter(a => !picked.includes(a.id)).map(a => ({ icon: <Tile a={a} s={18}></Tile>, l: a.n, s: toolsOf(a).length + " tools · ceiling " + TIER_L[ceilOf(a)], on: () => setPicked([...picked, a.id]) }))]}></Menu>}</div>}
        </div>
        <span className="sp"></span>
      </div>
      {!ags.length ? (
        <div className="pk-e"><b>Pick an agent to see what it can do without a person</b><span>Each tool is answered twice: for a run that has read only your own records, and for one that has read an email, a document or another message written outside the organization.</span>
          <div className="ms">{S.agents.map(a => <button key={a.id} className={"ms-i" + (a.on ? "" : " off")} title={a.n} onClick={() => setPicked([a.id])}><Tile a={a} s={32}></Tile></button>)}</div></div>
      ) : <>
        <div className="ahs">{ags.map(a => { const sens = toolsOf(a).filter(t => t.kind === "Action" && OUTSIDE.includes(t.e)); const isOpen = a.access === "Everyone" && sens.length > 0; return (
          <div key={a.id} className="ah">
            <div className="ah-h"><Tile a={a} s={28}></Tile><div className="sh-t"><b>{a.n}{!a.on && <span className="tg">Off</span>}{a.shadow && <span className="tg"><Ic n="eyeOff" s={10}></Ic>Shadow</span>}</b><span>{a.access === "Everyone" ? "Everyone who can use the assistant" : a.access} · ceiling {TIER_L[ceilOf(a)]}</span></div></div>
            {isOpen && <div className="ah-w"><Ic n="warn" s={13}></Ic><span>Open to everyone, and holds <button className="lnk" onClick={() => setWarn(warn === a.id ? null : a.id)}>{sens.length} tool{sens.length > 1 ? "s" : ""} that leave the organization</button>.{warn === a.id && <em> {sens.map(t => t.l).join(", ")}.</em>}</span><button className="btn sm" onClick={() => A.toast("Opening access for " + a.n)}>Limit to roles</button></div>}
          </div>); })}</div>
        <DT key={picked.join()} rows={rows} name="tool" find={x => x.t.l + x.t.n + x.a.n} onOpen={x => setOpen({ a: x.a, t: x.t })} sel={open && rows.find(x => x.a === open.a && x.t === open.t)}
          filters={[["all", "All tools", () => true], ["chg", "Changes", x => x.t.kind === "Action"], ["strict", "Stricter after outside text", x => x.r.a !== x.r.b]]} empty="Nothing here"
          cols={[...(ags.length > 1 ? [{ k: "ag", l: "Agent", c: x => <span className="ag"><Tile a={x.a} s={18}></Tile>{x.a.n}</span>, sv: x => x.a.n }] : []), { k: "t", l: "Tool", c: x => <div className="tl"><b>{x.t.l}</b><span className="mono">{x.t.n}</span></div>, sv: x => x.t.l }, { k: "e", l: "Who sees it", c: x => <Eg e={x.t.e}></Eg>, sv: x => EG_ORDER.indexOf(x.t.e), hide: 1 }, { k: "b", l: "Before outside text", c: x => <Ans v={x.r.b}></Ans>, sv: x => x.r.b }, { k: "af", l: "After outside text", c: x => <span className="ba">{x.r.a !== x.r.b && <Ic n="arrowR" s={11}></Ic>}<Ans v={x.r.a}></Ans></span>, sv: x => x.r.a }, { k: "h", l: "Held by", c: x => <Held xs={x.r.held}></Held> }]}></DT>
      </>}
      {open && <ToolSheet t={open.t} S={S} A={A} focus={open.a.id} onClose={() => setOpen(null)}></ToolSheet>}
    </>
  );
}
function SafetyTab({ S, A, searchRef }) {
  const [view, setView] = React.useState("rules"); const [picked, setPicked] = React.useState([]); const [eg, setEg] = React.useState(null);
  const F = safetyFigures(S.agents);
  const review = () => { setPicked(F.open.slice(0, 3).map(a => a.id)); setView("agents"); };
  return (
    <div className="tabp">
      <section className="hero rh">
        <div className="hero-s"><span className="who"><span className="dm"></span><b>Nova</b><span>Safety</span></span>
          <p className="say">{F.runs.length ? <><span className="ref" onClick={() => { setView("rules"); }}>{F.runs.length} tool{F.runs.length > 1 ? "s run" : " runs"}</span> without a person, all inside the organization.</> : <>No tool runs without a person.</>} <b>{F.leave} tools</b> can send outside the organization, and every one waits for approval.{F.open.length ? <> <span className="ref w" onClick={review}>{F.open.length} agents</span> everyone can use hold some of them.</> : ""}</p></div>
        {F.open.length > 0 && <div className="hc"><button className="btn ink lg" onClick={review}>Review open agents</button><span>Compare what they can do</span></div>}
      </section>
      <section className="wk w3">
        <div className="wk-c"><span className="lbl">Tools that run without a person</span><b className={"big mono" + (F.runs.length ? " t-b" : "")}>{F.runs.length}</b><span className="sub">on at least one agent{F.runs.length ? " · " + F.runs.map(n => TOOLS.find(t => t.n === n).l).join(", ") : ""}</span></div>
        <div className="wk-c"><span className="lbl">Tools that send outside the organization</span><b className="big mono">{F.leave}</b><span className="sub">never past approval</span></div>
        <div className="wk-c"><span className="lbl">Open agents with sensitive tools</span><b className={"big mono" + (F.open.length ? " t-w" : "")}>{F.open.length}</b><span className="sub">usable by everyone</span></div>
      </section>
      <section className="sec">
        <SecH t="Who sees the work" n={TOOLS.filter(t => t.kind === "Action").length + " tools that change things"} r={eg ? <button className="lnk" onClick={() => setEg(null)}>Clear</button> : null}></SecH>
        <EgressMap act={eg} onPick={e => { setEg(e); setView("rules"); }}></EgressMap>
      </section>
      <div className="vw"><Seg v={view} opts={[["rules", "Tool rules"], ["agents", "By agent" + (picked.length ? " · " + picked.length : "")]]} onChange={setView}></Seg></div>
      {view === "rules" ? <Rules S={S} A={A} searchRef={searchRef} eg={eg} setEg={setEg}></Rules> : <ByAgent S={S} A={A} picked={picked} setPicked={setPicked}></ByAgent>}
    </div>
  );
}
Object.assign(window, { SafetyTab });
