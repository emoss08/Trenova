const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "theme": "dark",
  "ai": true,
  "org": "both",
  "hos": true,
  "density": "compact",
  "board": "high volume"
}/*EDITMODE-END*/;

function App({ t }) {
  window.__ORG = t.org;
  const quiet = t.board === "quiet", hv = t.board === "high volume", ai = t.ai;
  const base = quiet ? QUIET : hv ? HV : SHIPMENTS;
  const [list, setList] = React.useState(base);
  const [filters, setFilters] = React.useState([]);
  const [msgs, setMsgs] = React.useState([]);
  const [view, setView] = React.useState("table");
  const [group, setGroup] = React.useState(true);
  const [openId, setOpenId] = React.useState(null);
  const [kbId, setKbId] = React.useState(null);
  const [checked, setChecked] = React.useState([]);
  const [railOpen, setRailOpen] = React.useState(window.innerWidth >= 900);
  const [tab, setTab] = React.useState("brief");
  const [done, setDone] = React.useState({});
  const [assignedNow, setAssignedNow] = React.useState(null);
  const [flashId, setFlashId] = React.useState(null);
  const [toast, setToast] = React.useState(null);
  const [sbc, setSbc] = React.useState(window.innerWidth < 1280);
  const [aiNote, setAiNote] = React.useState(true);
  const [spinning, setSpinning] = React.useState(false);
  const [keys, setKeys] = React.useState(false);
  React.useEffect(() => { const f = () => setKeys(true); window.addEventListener("tv:keys", f); return () => window.removeEventListener("tv:keys", f); }, []);
  const [ago, setAgo] = React.useState(0);
  const cmdRef = React.useRef();
  React.useEffect(() => { setList(base); setDone({}); setOpenId(null); setFilters([]); setMsgs([]); setChecked([]); }, [t.board]);
  
  
  const items = React.useMemo(() => {
    let q = (quiet ? QUEUE.slice(0, 2) : QUEUE).filter(a => (t.hos && t.org !== "brokerage") || a.id !== "q4");
    if (t.org === "brokerage") q = q.map(a => { if (a.drv === undefined) return a; const c = CARRIERS[hashId(a.ship) % CARRIERS.length], s0 = base.find(x => x.id === a.ship) || {}; const quote = Math.round((s0.mi || 300) * c.rate / 10) * 10; return { ...a, drv: undefined, carrier: c, t: "Tender " + (s0.o ? s0.o[1].split(",")[0] + " → " + s0.d[1].split(",")[0] : a.ship) + " to " + c.n, s: c.acc + "% acceptance on this lane, quoting $" + quote.toLocaleString() + " ($" + c.rate.toFixed(2) + "/mi).", approve: "Tender", okText: "Tendered " + a.ship + " to " + c.n, imp: ["$" + quote.toLocaleString() + " quote", c.acc + "% accept", Math.round((1 - quote / (s0.rev || quote)) * 100) + "% margin"], pt: a.pt, manual: "Find carrier" }; });
    return q;
  }, [quiet, t.org, t.hos, base]);
  const rows = applyFilters(list, filters);
  const say = m => { setToast(m); clearTimeout(window.__tt); window.__tt = setTimeout(() => setToast(null), 2400); };
  const finish = id => { setDone(d => ({ ...d, [id]: "ok" }));  };
  const open = (id) => { setOpenId(o => o === id ? null : id); setKbId(id); };
  const assign = (id, d) => {
    setList(l => l.map(s => s.id === id ? { ...s, drv: { ...d }, st: "assigned", tender: "Accepted", etaD: s.etaD } : s));
    setAssignedNow(id); setFlashId(id); say(d.carrier ? `Tendered ${id} to ${d.n}` : `${d.n} assigned to ${id}`);
    setTimeout(() => { setAssignedNow(a => a === id ? null : a); setFlashId(null); }, 3000);
    items.filter(a => a.ship === id && (a.drv !== undefined || a.carrier)).forEach(q => finish(q.id));
  };
  const approve = a => {
    if (a.carrier) { assign(a.ship, a.carrier); return; }
    if (a.drv !== undefined) { assign(a.ship, SUGGEST[a.ship][a.drv]); return; }
    finish(a.id); say(a.okText);
  };
  const review = a => {
    if (!ai || !a.approve) setDone(d => ({ ...d, [a.id]: "ok" }));
    if (!ai && a.id === "q3") { setFilters(parseQuery("late")); return; }
    if (a.ship) { setOpenId(a.ship); setKbId(a.ship); return; }
    say("Carrier picker opened");
  };
  const undo = a => {
    if (a.ship && a.drv !== undefined) setList(l => l.map(s => s.id === a.ship ? base.find(b => b.id === a.ship) : s));
    setDone(d => { const n = { ...d }; delete n[a.id]; return n; });
  };
  const ask = text => {
    setMsgs(m => [...m, { me: true, text }, { thinking: true }]);
    setTimeout(() => {
      const f = parseQuery(text), rows = applyFilters(list, f).filter(() => true);
      const real = f.length && f[0].k !== "text";
      const r2 = real ? rows : [];
      const late = r2.filter(s => s.late).length, open = r2.filter(s => !s.drv).length, rev = r2.reduce((a, s) => a + s.rev, 0);
      let t;
      if (!real) t = "I couldn't tie that to anything on today's board. Try asking about late loads, uncovered loads, a city or a customer.";
      else if (!r2.length) t = "Nothing on today's board matches that.";
      else { t = `${r2.length} shipment${r2.length > 1 ? "s" : ""} match, worth $${rev.toLocaleString()}.`; if (late) t += ` ${late === r2.length ? "All" : late} running late, mostly behind the storm over Iowa.`; if (open) t += ` ${open} still need${open > 1 ? "" : "s"} a driver. I have matches ready in the brief.`; if (!late && !open) t += " Everything here is covered and on time."; }
      setMsgs(m => [...m.slice(0, -1), { text: t, rows: r2, filters: real ? f : null }]);
    }, 900);
  };
  const applyF = (f, id) => { setFilters(f); if (id) { setOpenId(id); setKbId(id); } };
  const tender = ids => setList(l => l.map(s => ids.includes(s.id) ? { ...s, tender: "Sent", tenderTo: CARRIERS[hashId(s.id) % CARRIERS.length] } : s));
  const askAbout = s => say(`Assistant opened with ${s.id}`);
  const check = id => setChecked(c => c.includes(id) ? c.filter(x => x !== id) : [...c, id]);
  React.useEffect(() => {
    const h = e => {
      const mod = e.metaKey || e.ctrlKey, k = e.key.toLowerCase();
      if (mod && k === "/") { e.preventDefault(); setKeys(x => !x); return; }
      if (mod && k === "b") { e.preventDefault(); setSbc(x => !x); return; }
      if (mod && k === "j") { e.preventDefault(); say("Assistant opened"); return; }
      if (e.target.closest("input,textarea") || keys) return;
      const r = window.__rows || [], cur = kbId || openId;
      if (mod && k === "l" && cur) { e.preventDefault(); say("Link to " + cur + " copied"); return; }
      if (e.altKey && k === "c" && cur) { e.preventDefault(); say("Copied " + cur); return; }
      if (mod || e.altKey) return;
      if (e.key === "/") { e.preventDefault(); cmdRef.current && cmdRef.current.focus(); }
      else if (e.key === "Escape") { if (openId) setOpenId(null); else if (checked.length) setChecked([]); else setKbId(null); }
      else if (["ArrowDown", "ArrowUp", "j", "k"].includes(e.key)) {
        if (!r.length) return; e.preventDefault();
        const i = r.indexOf(cur), dn = e.key === "ArrowDown" || e.key === "j";
        const nx = r[i < 0 ? 0 : Math.max(0, Math.min(r.length - 1, i + (dn ? 1 : -1)))];
        setKbId(nx); if (openId) setOpenId(nx);
      }
      else if (e.key === "Enter" && cur) setOpenId(o => o === cur ? null : cur);
      else if (k === "x" && cur) check(cur);
      else if (k === "e" && cur) say("Editing " + cur);
    };
    window.addEventListener("keydown", h); return () => window.removeEventListener("keydown", h);
  }, [openId, kbId, checked, keys, ai]);
  const sel = list.find(s => s.id === openId);
  return (
    <div className={"tv" + (t.theme === "dark" ? " dk" : "") + (t.density === "compact" ? " cmp" : "") + (t.hos ? "" : " nohos") + (sbc ? " sbc" : "")}>
      <Top onToggleSide={() => setSbc(!sbc)}></Top>
      <Side total={list.length} late={list.filter(s => s.late).length}></Side>
      <main className="ws">
        <PageHead count={list.length} ago={ago < 5 ? "just now" : ago < 60 ? ago + "s ago" : Math.floor(ago / 60) + "m ago"} spinning={spinning} onRefresh={() => { setSpinning(true); setAgo(0); setTimeout(() => setSpinning(false), 720); }} onNew={() => say("New shipment draft opened")}></PageHead>
        <div className="hdx">{ai && <TopBrief list={list} quiet={quiet} onRef={k => setFilters(parseQuery(k))}></TopBrief>}<Capacity key={t.board + t.org} onTender={tender} org={t.org} hos={t.hos} onFilter={setFilters} list={list} pool={quiet ? POOL.slice(0, 4) : hv ? POOL : POOL.slice(0, 9)} onAssign={assign} onAct={say}></Capacity></div>
        <div className={"split" + (railOpen ? "" : " norail")}>
          <Board filters={filters} setFilters={setFilters} filterRef={cmdRef} list={list} rows={rows} view={view} setView={setView} group={group} setGroup={setGroup} openId={openId} kbId={kbId} onOpen={open} checked={checked} onCheck={check} onCheckAll={(ids, all) => setChecked(c => all ? c.filter(x => !ids.includes(x)) : [...new Set([...c, ...ids])])} flashId={flashId} railOpen={railOpen} setRailOpen={setRailOpen} onClearFilters={() => setFilters([])} ai={ai} onAssign={assign} onNotify={s => say(`Update sent to ${s.cust}`)} onAct={say} onAsk={askAbout} assignedNow={assignedNow}></Board>
          {railOpen && <SafePanel resetKey={t.org + t.hos + t.board + t.ai}><Rail onFilter={setFilters} onAct={say} onOpenRow={id => { setOpenId(id); setKbId(id); }} tab={tab} setTab={setTab} sel={null} ai={ai} quiet={quiet} list={list} items={items} done={done} onApprove={approve} onReview={review} onUndo={undo} onRef={k => setFilters(parseQuery(k))} msgs={msgs} onSend={ask} onApply={applyF} onAsk={askAbout} aiNote={aiNote} setAiNote={setAiNote} onBack={() => setOpenId(null)} onClose={() => { setRailOpen(false); setOpenId(null); }} onAssign={assign} onNotify={s => say(`Update sent to ${s.cust}`)} assignedNow={assignedNow}></Rail></SafePanel>}
        </div>
        {checked.length > 0 && !toast && <div className="bulk"><b>{checked.length} selected</b><button className="btn ghost sm" onClick={() => { say(`${checked.length} tendered`); setChecked([]); }}><Ic n="truck" s={12}></Ic>Tender</button><button className="btn ghost sm" onClick={() => { if (t.org !== "asset") { tender(checked); say(checked.length + " tendered to best-match carriers"); setChecked([]); } }}><Ic n="user" s={12}></Ic>{t.org === "brokerage" ? "Tender to carrier" : "Assign"}</button>{ai && <button className="btn ghost sm"><Ic n="diamond" s={11}></Ic>Ask assistant</button>}<button className="btn ghost sm"><Ic n="download" s={12}></Ic>Export</button><span className="vr"></span><button className="ib" onClick={() => setChecked([])}><Ic n="x" s={12}></Ic></button></div>}
        {keys && <Shortcuts onClose={() => setKeys(false)}></Shortcuts>}
        {toast && <div className="toast" key={toast}><Ic n="check" s={13} w={2.4}></Ic>{toast}</div>}
      </main>
    </div>
  );
}

function Root() {
  const [t, setTweak] = useTweaks(TWEAK_DEFAULTS);
  return (
    <>
      <div style={{ width: "100vw", height: "100vh" }}><App t={t}></App></div>
      <TweaksPanel>
        <TweakSection label="Workspace"></TweakSection>
        <TweakToggle label="AI provider configured" value={t.ai} onChange={v => setTweak("ai", v)}></TweakToggle>
        <TweakRadio label="Organization" value={t.org} options={["asset", "brokerage", "both"]} onChange={v => setTweak("org", v)}></TweakRadio>
        <TweakToggle label="ELD / HOS integration" value={t.hos} onChange={v => setTweak("hos", v)}></TweakToggle>
        <TweakRadio label="Board" value={t.board} options={["quiet", "busy", "high volume"]} onChange={v => setTweak("board", v)}></TweakRadio>
        <TweakSection label="Appearance"></TweakSection>
        <TweakRadio label="Theme" value={t.theme} options={["dark", "light"]} onChange={v => setTweak("theme", v)}></TweakRadio>
        <TweakRadio label="Density" value={t.density} options={["comfortable", "compact"]} onChange={v => setTweak("density", v)}></TweakRadio>
      </TweaksPanel>
    </>
  );
}
ReactDOM.createRoot(document.getElementById("root")).render(<Root></Root>);
