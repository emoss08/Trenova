const NOVA_DID = [["box", "Filed 14 proofs of delivery", 14], ["receipt", "Queued 6 carrier invoices", 6], ["archive", "Ignored 9 newsletters", 9]];
const MIN_PER = { file: 1, desk: 2, you: 4, link: 2, held: 0.5 };
const GROUPS = [["file", "clip", n => n + (n > 1 ? " documents" : " document") + " to file", "File all", "Each one names its PRO. One tap files them all."], ["desk", "nova", n => n + (n > 1 ? " drafts" : " draft") + " Nova can write", "Draft all", "Loads and replies. Nothing is sent or created until you look."], ["you", "user", n => n + " need your call", "Start", "Disputes, exceptions — anything with money on it."], ["link", "link", n => n + " without a record", "Link", "No PRO, PO or known sender to match on."], ["held", "shield", n => n + " held back", "Review", "Looks like phishing. Ignore unless you know them."]];
const NOVA_T = [[460, 3], [430, 5], [395, 6], [350, 4], [300, 5], [250, 3], [170, 2], [80, 1], [35, 1]];

function useCount(to, on, ms = 900) {
  const [v, setV] = React.useState(0);
  const from = React.useRef(0);
  React.useEffect(() => {
    if (!on) return; const a = from.current, t0 = Date.now();
    const iv = setInterval(() => { const k = Math.min(1, (Date.now() - t0) / ms), e = 1 - Math.pow(1 - k, 3); setV(Math.round(a + (to - a) * e)); if (k >= 1) { clearInterval(iv); from.current = to; } }, 30);
    return () => clearInterval(iv);
  }, [to, on]);
  return v;
}
function Ring({ p, n, busy }) {
  const r = 46, c = 2 * Math.PI * r, shown = useCount(n, !busy);
  return (
    <div className={"br-ring" + (p >= 1 ? " full" : "") + (busy ? " busy" : "")}>
      <svg viewBox="0 0 108 108"><defs><linearGradient id="novag" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stopColor="oklch(0.651 0.178 259.1)"></stop><stop offset=".55" stopColor="oklch(0.651 0.191 295.9)"></stop><stop offset="1" stopColor="oklch(0.731 0.166 7.7)"></stop></linearGradient></defs>
        <circle cx="54" cy="54" r={r} className="br-rt"></circle>
        <circle cx="54" cy="54" r={r} className="br-rs" strokeDasharray={c * 0.18 + " " + c}></circle>
        <circle cx="54" cy="54" r={r} className="br-rf" strokeDasharray={c} strokeDashoffset={busy ? c : c * (1 - Math.max(0.02, p))}></circle></svg>
      <div className="br-rc">{p >= 1 ? <Ic n="check" s={30} w={2.4}></Ic> : <><b className="mono">{busy ? "" : shown}</b><span>{busy ? "" : "for you"}</span></>}</div>
    </div>
  );
}

function River({ msgs, need, onOpen }) {
  const [hov, setHov] = React.useState(null);
  const X = min => Math.max(2, Math.min(98, 100 - (Math.min(min, 520) / 520) * 96));
  const nova = []; NOVA_T.forEach(([min, k]) => { for (let i = 0; i < k; i++) nova.push(min - i * 9); });
  const ticks = [[460, "6a"], [340, "8a"], [220, "10a"], [100, "12p"], [0, "now"]];
  const hm = hov && msgs.find(m => m.id === hov);
  return (
    <div className="br-rv">
      <div className="br-ax">{ticks.map(([m, l]) => <span key={l} style={{ left: X(m) + "%" }}>{l}</span>)}</div>
      <div className="br-ln nova"><em>Nova</em>{nova.map((m, i) => <i key={i} style={{ left: X(m) + (i % 3) * 0.4 + "%", top: 8 + (i % 4) * 5 + "px", animationDelay: i * 22 + "ms" }}></i>)}</div>
      <div className="br-ln you"><em>You</em>{msgs.filter(m => isNeed(m) || (m.by === ME && m.byMin === 0)).map((m, i) => { const done = !isNeed(m) || (m.snooze || m.await); const d = dueOf(m); return (
        <button key={m.id} className={"br-dt" + (done ? " ok" : d < 0 ? " over" : "")} style={{ left: X(m.min) + "%", "--h": KIND[m.kind].h == null ? 0 : KIND[m.kind].h, animationDelay: 300 + i * 50 + "ms" }} onMouseEnter={() => setHov(m.id)} onMouseLeave={() => setHov(null)} onClick={() => onOpen(m.id)}>{done && <Ic n="check" s={8} w={3.4}></Ic>}</button>); })}</div>
      <span className="br-now" style={{ left: X(0) + "%" }}></span>
      {hm && <div className={"br-tip" + (X(hm.min) < 18 ? " l" : X(hm.min) > 82 ? " r" : "")} style={{ left: X(hm.min) + "%" }}><b>{senderOf(hm)}</b><span>{hm.subject || "(no subject)"}</span><em className={"mono" + (isNeed(hm) && dueOf(hm) < 0 ? " d" : "")}>{isNeed(hm) ? dueTxt(dueOf(hm)) : "done"}</em></div>}
    </div>
  );
}

function Motes() {
  const ref = React.useRef();
  React.useEffect(() => {
    const cv = ref.current, cx = cv.getContext("2d"), dpr = Math.min(2, window.devicePixelRatio || 1);
    const still = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    let w = 0, h = 0, raf, t0 = performance.now();
    const hues = [[1, 0, 0]];
    const P = Array.from({ length: 46 }, (_, i) => ({ x: Math.random(), y: Math.random(), r: 0.4 + Math.random() * 1.1, s: 0.004 + Math.random() * 0.012, ph: Math.random() * 6.28, c: hues[0], a: 0.25 + Math.random() * 0.55 }));
    const size = () => { const r = cv.getBoundingClientRect(); w = r.width; h = r.height; cv.width = w * dpr; cv.height = h * dpr; cx.setTransform(dpr, 0, 0, dpr, 0, 0); };
    size(); const ro = new ResizeObserver(size); ro.observe(cv);
    const ink = () => cv.closest(".tv.dk") ? "255,255,255" : "20,20,24";
    const draw = now => {
      const k = ink();
      const t = (now - t0) / 1000; cx.clearRect(0, 0, w, h);
      P.forEach(p => {
        const y = ((p.y - t * p.s) % 1 + 1) % 1, x = p.x + Math.sin(t * 0.3 + p.ph) * 0.018;
        const px = x * w, py = y * h, fall = Math.max(0, 1 - Math.hypot((px - w / 2) / (w * 0.5), (py - h * 0.35) / (h * 0.6)));
        const tw = 0.45 + 0.55 * Math.abs(Math.sin(t * (0.8 + p.s * 60) + p.ph));
        cx.beginPath(); cx.arc(px, py, p.r, 0, 6.283);
        cx.fillStyle = "rgba(" + k + "," + (p.a * tw * fall * (k[0] === "2" && k[1] === "0" ? 0.8 : 1)).toFixed(3) + ")";
        cx.shadowColor = "rgba(" + k + ",.6)"; cx.shadowBlur = k[0] === "2" && k[1] === "5" ? p.r * 3 : p.r * 1.5;
        cx.fill();
      });
      if (!still) raf = requestAnimationFrame(draw);
    };
    raf = requestAnimationFrame(draw);
    return () => { cancelAnimationFrame(raf); ro.disconnect(); };
  }, []);
  return <canvas className="br-motes" ref={ref} aria-hidden="true"></canvas>;
}

function FirstDay({ stage, onConnect, readN }) {
  const tally = [["Tender", 4], ["RateConfirmation", 3], ["ProofOfDelivery", 9], ["StatusRequest", 5], ["Invoice", 6], ["Other", 11]];
  const tot = tally.reduce((s, x) => s + x[1], 0), k = Math.min(1, readN / tot);
  return (
    <section className="br fd">
      <div className="fd-in">
        <div className="fd-orb"><span className={"fd-ring" + (stage === 1 ? " go" : "")}></span><span className="fd-n mono">{stage === 1 ? Math.round(readN) : stage === 2 ? "✓" : "0"}</span></div>
        <h2>{stage === 0 ? "Nothing has come in yet" : stage === 1 ? "Reading what was already waiting…" : "Done reading"}</h2>
        <p>{stage === 0 ? "Point a shared address at Triage — dispatch@, pod@, ap@. Nova reads what's already sitting there, so your first briefing isn't empty." : stage === 1 ? "Sorting, matching to shipments, and setting aside anything that needs a person." : "Your first briefing is ready."}</p>
        {stage === 0 && <button className="btn ink lg" onClick={onConnect}><Ic n="plus" s={13}></Ic>Add a source</button>}
        {stage >= 1 && <div className="fd-t">{tally.map(([kk, n], i) => { const got = Math.min(n, Math.max(0, Math.round(readN - tally.slice(0, i).reduce((s, x) => s + x[1], 0)))); return <span key={kk} className={got ? "on" : ""}><Kind k={kk}></Kind><b className="mono">{got}</b></span>; })}</div>}
        <ol className="fd-s">{["Add a source", "Nova reads the backlog", "Your first briefing"].map((t, i) => <li key={t} className={i < stage + (stage === 2 ? 1 : 0) ? "d" : i === stage ? "on" : ""}><i className="mono">{i < stage || stage === 2 ? "✓" : i + 1}</i>{t}</li>)}</ol>
      </div>
    </section>
  );
}

function Briefing({ msgs, rules, onOpen, onLane, api, corr = 0, eod, firstDay, fdStage, fdRead, onConnect }) {
  if (firstDay) return <FirstDay stage={fdStage} readN={fdRead} onConnect={onConnect}></FirstDay>;
  return <BriefingMain msgs={msgs} rules={rules} onOpen={onOpen} onLane={onLane} api={api} corr={corr} eod={eod}></BriefingMain>;
}
function BriefingMain({ msgs, rules, onOpen, onLane, api, corr, eod }) {
  const [handed, setHanded] = React.useState(false);
  const [ready, setReady] = React.useState(false);
  React.useEffect(() => { const t = setTimeout(() => setReady(true), 1000); return () => clearTimeout(t); }, []);
  const [snap, setSnap] = React.useState({});
  const [cur, setCur] = React.useState(0);
  const [fly, setFly] = React.useState(null);
  const [taught, setTaught] = React.useState(false);
  const need = msgs.filter(m => isNeed(m) && !m.snooze && !m.await);
  const base = React.useRef(null); if (base.current == null) base.current = need.length || 1;
  const held = new Set(Object.values(snap).flatMap(s => s.ids));
  const live = {};
  need.forEach(m => { if (held.has(m.id)) return; const a = actsOf(m).find(x => !x.done); if (!a) return; const k = m.status === "Quarantined" ? "held" : a.k === "file" && m.shp ? "file" : a.desk ? "desk" : a.k === "link" ? "link" : "you"; (live[k] = live[k] || []).push([m, a]); });
  const deck = GROUPS.filter(g => live[g[0]] && !snap[g[0]]);
  const doneG = GROUPS.filter(g => snap[g[0]]);
  const left = need.filter(m => !held.has(m.id)).length;
  const p = 1 - left / base.current;
  const mins = Math.round(deck.reduce((s, g) => s + live[g[0]].length * MIN_PER[g[0]], 0));
  const real = need.filter(m => m.status !== "Quarantined"), over = real.filter(m => dueOf(m) < 0).sort((a, b) => dueOf(a) - dueOf(b));
  const did = NOVA_DID.reduce((s, x) => s + x[2], 0);
  const sick = MAILBOXES.filter(b => !b.ok), away = msgs.filter(m => isNeed(m) && (m.snooze || m.await));
  const rc = need.find(m => m.kind === "RateConfirmation"), teach = rc && !rules.some(r => r.kind === "RateConfirmation" && r.from === rc.from[1].split("@")[1]) && { r: { kind: "RateConfirmation", from: rc.from[1].split("@")[1], mb: rc.mb, act: RULE_FOR.attach }, dom: rc.from[1].split("@")[1] };
  const ci = Math.min(cur, Math.max(0, deck.length - 1));
  const toss = (k, after) => { setFly(k); setTimeout(() => { setFly(null); after(); }, 420); };
  const act = (k, its) => {
    const ids = its.map(([m]) => m.id);
    if (k === "file") return toss(k, () => { its.forEach(([m, a]) => api.fileOn(m.id, a.a.n, m.shp)); setSnap(s => ({ ...s, file: { ids, l: "Filed " + its.length + " documents" } })); });
    if (k === "desk") { setSnap(s => ({ ...s, desk: { ids, l: "Drafting…", busy: true } })); its.forEach(([m, a]) => api.draftFor(m, a)); setTimeout(() => setSnap(s => ({ ...s, desk: { ids, l: its.length + " drafts ready", first: ids[0] } })), 1500); return; }
    onOpen(its[0][0].id);
  };
  const lead = over[0];
  return (
    <section className="br">
      <Motes></Motes>
      <div className="br-in">
        <header className="br-top">
          <Ring p={p} n={left} busy={!ready}></Ring>
          <div className="br-tx">
            <div className="br-who"><b>Nova</b><span className="mono">{hhmm(NOW)}</span></div>
            {!ready ? <p className="br-say"><span className="shim">Going through the day…</span></p> : eod ? <p className="br-say say">{"Wrapping up. I handled "}<b>{did}</b>{" and you closed "}<b>{msgs.filter(m => m.by === ME).length + 6}</b>{". "}{left ? <><b>{left}</b>{" carry over to tomorrow — "}<b>Marcus</b>{" is on overnight."}</> : "Nothing carries over."}</p> : left === 0 ? <p className="br-say say">That's everything. <b>You're clear</b> — I'll bring the next one here when it lands.</p> :
            <p className="br-say say">{"I took care of "}<b>{did}</b>{" today. "}<b>{left === 1 ? "One is" : left + " are"}</b>{" yours"}{lead ? <>{" — "}<span className="br-ref d" onClick={() => onOpen(lead.id)}>{senderOf(lead).split(" ")[0]}</span>{" has waited longest."}</> : "."}{mins ? <span className="br-eta">{" About " + mins + " minutes of work."}</span> : null}</p>}
          </div>
        </header>

        <div className={"br-st" + (ready ? " on" : "")} style={{ "--d": "80ms" }}>{ready && <River msgs={msgs} need={need} onOpen={onOpen}></River>}</div>

        <div className={"br-deck-w br-st" + (ready ? " on" : "")} style={{ "--d": "260ms" }}>{ready && <>
          {eod ? <div className="eod">
            <div className="eod-h"><b>Carrying over</b><span className="mono">{left}</span><span className="sp"></span>{handed ? <span className="eod-ok"><Ic n="check" s={12} w={2.4}></Ic>Handed to Marcus · he'll see them first thing</span> : <button className="btn ink sm" onClick={() => { setHanded(true); api.act("Handed " + left + " to Marcus for overnight"); }}><Av n="Marcus Lee" s={18}></Av>Hand off to Marcus</button>}</div>
            {need.filter(m => !held.has(m.id)).sort((a, z) => dueOf(a) - dueOf(z)).map((m, i) => { const a = actsOf(m).find(x => !x.done); return <button key={m.id} className={"eod-r" + (handed ? " h" : "")} style={{ animationDelay: i * 50 + "ms" }} onClick={() => onOpen(m.id)}><i className={"kdot" + (KIND[m.kind].h == null ? " n" : "")} style={{ "--h": KIND[m.kind].h }}></i><b>{senderOf(m)}</b><span>{a ? a.t : m.subject}</span><em className={"mono" + (dueOf(m) < 0 ? " d" : "")}>{dueTxt(dueOf(m))}</em>{handed && <span className="eod-to"><Av n="Marcus Lee" s={16}></Av></span>}</button>; })}
          </div> : deck.length > 0 ? <div className="br-deck">
            {deck.map((g, i) => { const [k, ic, title, btn, why] = g, its = live[k], o = (i - ci + deck.length) % deck.length; if (o > 2) return null; return (
              <div key={k} className={"br-card k-" + k + (o === 0 ? " front" : "") + (fly === k ? " fly" : "") + (k === "desk" && snap.desk && snap.desk.busy ? " working" : "")} style={{ "--o": o, zIndex: 10 - o }} onClick={() => o !== 0 && setCur(i)}>
                {k === "desk" && <><span className="br-aura" aria-hidden="true"></span><span className="br-edge" aria-hidden="true"></span></>}
                <div className="br-ch"><span className={"br-ci k-" + k}>{ic === "nova" ? <span className="dm md"></span> : <Ic n={ic} s={15}></Ic>}</span><span className="br-cn mono">{(o === 0 ? ci + 1 : i + 1) + " / " + deck.length}</span></div>
                <h3>{title(its.length)}</h3>
                <p>{why}</p>
                {o === 0 && <><div className="br-its">{its.slice(0, 4).map(([m, a], j) => <button key={m.id} style={{ animationDelay: (k === "desk" ? 180 + j * 90 : 0) + "ms" }} onClick={e => { e.stopPropagation(); onOpen(m.id); }}><span className="av">{initials(senderOf(m))}</span><span><b>{senderOf(m)}</b><em>{a.t}</em></span><span className={"mono" + (dueOf(m) < 0 ? " d" : "")}>{dueTxt(dueOf(m))}</span></button>)}</div>
                  <div className="br-ca"><button className="ib" title="Previous" onClick={e => { e.stopPropagation(); setCur((ci - 1 + deck.length) % deck.length); }} disabled={deck.length < 2}><Ic n="chevL" s={14}></Ic></button><button className="ib" title="Next" onClick={e => { e.stopPropagation(); setCur((ci + 1) % deck.length); }} disabled={deck.length < 2}><Ic n="chevR" s={14}></Ic></button><span className="sp"></span><span className="br-mn mono">{"~" + Math.max(1, Math.round(its.length * MIN_PER[k])) + " min"}</span><button className={"btn " + (k === "desk" ? "teal" : "ink")} onClick={e => { e.stopPropagation(); act(k, its); }}>{btn}</button></div></>}
              </div>); })}
          </div> : <div className="br-empty"><span className="br-eok"><Ic n="check" s={20} w={2.4}></Ic></span><b>Nothing left in the deck</b><span>Nova is watching {MAILBOXES.filter(b => b.ok).length} sources.</span></div>}
          </>}
          {ready && doneG.length > 0 && <div className="br-done">{doneG.map(([k]) => { const s = snap[k]; return <button key={k} className={"br-dc" + (s.busy ? " busy" : "")} onClick={() => s.first ? onOpen(s.first) : onLane({ t: "lane", v: "waiting" })}>{s.busy ? <span className="dm xs spin"></span> : <Ic n="check" s={11} w={2.6}></Ic>}{s.l}</button>; })}</div>}
        </div>

        <footer className={"br-ft br-st" + (ready ? " on" : "")} style={{ "--d": "460ms" }}>
          <button className="br-pill" onClick={() => onLane({ t: "lane", v: "handled" })}><b className="mono">{did}</b><span>handled by Nova</span><em>Spot-check</em></button>
          {corr > 0 && <button className="br-pill" onClick={() => onLane({ t: "lane", v: "all" })}><Ic n="check" s={12} w={2.2}></Ic><b className="mono">{corr}</b><span>{"corrections this week · reading accuracy up " + Math.min(4, 1 + Math.floor(corr / 2)) + "%"}</span></button>}
          {away.length > 0 && <button className="br-pill" onClick={() => onLane({ t: "lane", v: away[0].await ? "them" : "snoozed" })}><Ic n="clock" s={12}></Ic><b className="mono">{away.length}</b><span>coming back later</span></button>}
          {sick.map(b => <button key={b.id} className="br-pill w" onClick={() => onLane({ t: "mb", v: b.id })}><i className="br-blink"></i><span>{b.name + " stopped receiving mail"}</span><em>Fix</em></button>)}
          {teach && (taught ? <span className="br-pill ok"><Ic n="check" s={11} w={2.6}></Ic><span>Nova files rate cons from {teach.dom} now</span></span> :
            <span className="br-pill t"><span title={teach.dom}>{"You filed 11 of 11 rate cons from " + teach.dom.split(".")[0] + " by hand"}</span><button onClick={() => api.preview(teach.r)}>Preview</button><button className="go" onClick={() => { api.addRule(teach.r); setTaught(true); }}>Hand it to Nova</button></span>)}
        </footer>
      </div>
    </section>
  );
}
Object.assign(window, { Briefing });
