const EXAMPLES = [["Chase detention when a truck waits more than two hours at a dock", "detention"], ["Every Friday, email each customer a list of their open invoices", "invoices"], ["Warn drivers 30 days before their medical card expires", "medical"]];
const STARTS = [["chat", "Desk agent", "Answers people in the assistant"], ["calendar", "Scheduled report", "Runs on a timetable and sends a summary"], ["bolt", "Event watcher", "Wakes when something happens"], ["edit", "Start blank", "Set everything up yourself"]];
function draftFrom(desc) {
  const t = desc.toLowerCase();
  if (/detention|dock|wait/.test(t)) return { n: "Detention desk", d: "Starts the detention clock when a truck waits too long and drafts the charge.", ic: "clock", h: 25, trig: "Event", events: ["Detention started"], tools: { find_in_trenova: "AutoExecute", get_shipment: "AutoExecute", recall_memory: "AutoExecute", flag_for_manual_review: "AutoExecute", send_customer_reply: "ActWithApproval", correct_charge_code: "Propose" }, instr: "You are the detention desk for {{organization}}.\n\nWhen a truck has waited more than two hours past its appointment, check the arrival time on the shipment and the customer's detention terms.\nDraft a short note to the customer with the start time and the running total, and propose the detention charge.\nIf the arrival time is missing, flag the shipment for review instead of guessing." };
  if (/invoice|friday|email/.test(t)) return { n: "Open invoice reminders", d: "Every Friday, sends each customer a tidy list of what they still owe.", ic: "dollar", h: 200, trig: "Scheduled", cron: "0 9 * * 5", tools: { find_in_trenova: "AutoExecute", search_shipments: "AutoExecute", get_accounting_sync_status: "AutoExecute", schedule_report_email: "Propose" }, instr: "You are the receivables reminder for {{organization}}.\n\nEvery Friday, list each customer's open invoices with amount, age and reference.\nSkip customers with a dispute open, and never include internal notes.\nPropose one email per customer for a person to approve." };
  if (/medical|expire|qualification/.test(t)) return { n: "Qualification watch", d: "Warns drivers and the safety team before a medical card or license expires.", ic: "shield", h: 160, trig: "Event", events: ["Qualification expiring"], tools: { find_in_trenova: "AutoExecute", read_document: "AutoExecute", send_driver_message: "ActWithApproval", update_qualification: "Propose" }, instr: "You watch driver qualifications for {{organization}}.\n\nThirty days before a medical card or license expires, message the driver with what they need to renew and where to upload it.\nTell the safety manager about anything due within seven days." };
  return { n: "New desk agent", d: desc.slice(0, 90), instr: "You help {{user.name}} at {{organization}}.\n\n" + desc };
}
function CreateIntro({ S, onStart }) {
  const [desc, setDesc] = React.useState(""); const [step, setStep] = React.useState(-1);
  const ai = S.providers.some(p => p.on);
  const STEPS = ["Naming it", "Writing instructions", "Choosing when it runs", "Picking tools and how free each one is", "Setting guardrails"];
  const draft = () => { if (!desc.trim()) return; setStep(0); let i = 0; const t = () => { i++; if (i < STEPS.length) { setStep(i); setTimeout(t, 420); } else setTimeout(() => onStart({ ...agentDraft(null, "Desk agent"), ...draftFrom(desc), mode: "shadow" }, true), 380); }; setTimeout(t, 420); };
  return (
    <div className="ci">
      <div className="ci-in">
        <span className="ci-k"><span className="ci-o"></span>New agent</span>
        <h1>What should it do?</h1>
        <p className="ci-s">Describe the job in a sentence or two. {ai ? "Nova drafts the name, instructions, trigger and tools, and you adjust from there." : "Then pick a starting point below."}</p>
        {step < 0 ? <>
          <div className={"ci-box" + (ai ? "" : " off")}>
            <textarea autoFocus rows={3} value={desc} placeholder="When a truck waits more than two hours at a dock…" onChange={e => setDesc(e.target.value)} onKeyDown={e => { if ((e.metaKey || e.ctrlKey) && e.key === "Enter") { e.preventDefault(); ai && draft(); } }}></textarea>
            <div className="ci-bar">
              <div className="ci-ex">{EXAMPLES.map(([x]) => <button key={x} onClick={() => setDesc(x)}>{x.split(" ").slice(0, 5).join(" ")}…</button>)}</div>
              {ai ? <button className="btn ink" disabled={!desc.trim()} onClick={draft}><Ic n="sparkle" s={13}></Ic>Draft it<span className="kbd">{MOD} ↵</span></button> : <span className="ci-na"><Ic n="plug" s={12}></Ic>Connect a provider to draft with AI</span>}
            </div>
          </div>
          <div className="ci-or"><span>{ai ? "or start from" : "Start from"}</span></div>
          <div className="ci-g">{STARTS.map(([ic, l, s]) => <button key={l} className="ci-t" onClick={() => onStart({ ...agentDraft(null, l), ...(desc.trim() && l !== "Start blank" ? { d: desc.trim().slice(0, 120) } : {}) }, false)}><span className="ci-ti"><Ic n={ic} s={16}></Ic></span><b>{l}</b><span>{s}</span><Ic n="arrowR" s={13}></Ic></button>)}</div>
        </> : <div className="ci-run">
          <p className="ci-q">“{desc}”</p>
          {STEPS.map((x, i) => i <= step && <div key={x} className={"ci-st" + (i === step ? " cur" : "")}><span className="ck">{i < step ? <Ic n="check" s={12} w={2.4}></Ic> : <i className="spn"></i>}</span><span className="tx">{x}</span></div>)}
        </div>}
      </div>
    </div>
  );
}
const WK_DAYS = [["Wed", 3], ["Thu", 4], ["Fri", 5], ["Sat", 6], ["Sun", 0], ["Mon", 1], ["Tue", 2]];
function cronRuns(c) {
  const p = (c || "").trim().split(/\s+/); if (p.length < 5) return null;
  const [, h, dom, , dow] = p; if (dom !== "*") return { none: true };
  const hours = h === "*" ? Array.from({ length: 24 }, (_, i) => i) : h.split(",").map(Number).filter(x => !isNaN(x));
  const days = dow === "*" ? [0, 1, 2, 3, 4, 5, 6] : dow.split(",").flatMap(x => { const [a, b] = x.split("-").map(Number); return b != null ? Array.from({ length: b - a + 1 }, (_, i) => a + i) : [a]; });
  return { hours, days };
}
function WeekStrip({ cron }) {
  const r = cronRuns(cron);
  if (!r) return <div className="wks nil">Not a schedule Trenova can read yet.</div>;
  if (r.none) return <div className="wks nil"><Ic n="calendar" s={13}></Ic>Runs on set days of the month, so nothing in the next seven days.</div>;
  const count = r.days.length * r.hours.length;
  return <div className="wks">
    {WK_DAYS.map(([l, dw], i) => { const on = r.days.includes(dw); return <div key={l} className={"wks-d" + (i === 0 ? " today" : "")}><div className="wks-t">{i === 0 && <span className="wks-now" style={{ top: (13 / 24 * 100) + "%" }}></span>}{on && r.hours.map(h => <i key={h} className={(i === 0 && h < 13 ? "past" : "") + (r.hours.length > 6 ? " tick" : "")} style={{ top: (h / 24 * 100) + "%" }}></i>)}</div><span>{i === 0 ? "Today" : l}</span></div>; })}
    <div className="wks-s"><b className="mono">{count}</b><span>runs in the next 7 days</span></div>
  </div>;
}
function TryPanel({ a, v, S, onTested, onClose }) {
  const ex = (a && TRY[a.id]) || { q: v.n ? "What would you do with SHP-48302 right now?" : "What can you help me with?", calls: Object.keys(v.tools).slice(0, 3), a: (v.d || "I'd answer from what I can read") + " Here's what I found on SHP-48302 and what I'd propose next." };
  const [runs, setRuns] = React.useState([]); const [q, setQ] = React.useState(""); const thread = React.useRef();
  const ai = S.providers.some(p => p.on); const snap = JSON.stringify(v);
  const sugg = [ex.q, "What can't you do?", "Walk me through your last decision"];
  const run = text => {
    const t = (text || q).trim(); if (!t || !ai) return; setQ("");
    const calls = ex.calls.filter(n => TOOLS.find(x => x.n === n)), id = Date.now(), words = (t === ex.q ? ex.a : t.startsWith("What can't") ? "I can't " + (Object.keys(v.tools).some(n => (TOOLS.find(x => x.n === n) || {}).kind === "Action") ? "act on anything outside my tools, and anything above " + TIER_L[v.ceil].toLowerCase() + " waits for a person." : "change records at all. I can read and explain.") : ex.a).split(" ");
    setRuns(rs => [...rs, { id, q: t, calls, words, step: 0, w: 0, snap, v: JSON.parse(snap) }]);
    let i = 0; const st = () => { i++; setRuns(rs => rs.map(r => r.id === id ? { ...r, step: i } : r)); if (i <= calls.length) setTimeout(st, 650); else { let w = 0; const wt = () => { w += 2; setRuns(rs => rs.map(r => r.id === id ? { ...r, w } : r)); if (w < words.length) setTimeout(wt, 45); else onTested(snap); }; wt(); } };
    setTimeout(st, 650);
  };
  React.useEffect(() => { const el = thread.current; el && el.scrollTo({ top: el.scrollHeight, behavior: "smooth" }); }, [runs]);
  return (
    <aside className="tp2">
      <header className="tp2-h"><Ic n="flask" s={13}></Ic><b>Try it</b><span className="tp2-p">Simulation</span><span className="sp"></span>{runs.length > 0 && <button className="lnk" onClick={() => setRuns([])}>Clear</button>}<button className="ib" title="Hide" onClick={onClose}><Ic n="x" s={13}></Ic></button></header>
      <div className="tp2-b" ref={thread}>
        {!runs.length ? <div className="tp2-e"><Tile a={{ ic: v.ic, h: v.h }} s={40}></Tile><b>Ask {v.n || "it"} something</b><p>It runs your unsaved draft against live records. Nothing is written, sent or offered to anyone.</p>
          <div className="tp2-sg">{sugg.map(x => <button key={x} disabled={!ai} onClick={() => run(x)}>{x}<Ic n="arrowR" s={11}></Ic></button>)}</div>
          {!ai && <Callout tone="w">Connect a provider to try agents.</Callout>}</div>
          : runs.map((r, ri) => { const stale = r.snap !== snap && ri === runs.length - 1 && r.w >= r.words.length; return (
            <div key={r.id} className="tp2-r">
              <div className="tp2-q"><span className="me sm">SA</span><p>{r.q}</p></div>
              <div className="tp2-a">
                <div className="tp2-w"><Tile a={{ ic: r.v.ic, h: r.v.h }} s={18}></Tile><b>{r.v.n || "New agent"}</b><span className="mono">{ED_MODE_NOTE && r.v.mode === "live" ? "live draft" : r.v.mode}</span></div>
                <div className="tp2-l">{r.calls.slice(0, r.step).map((n, i) => { const t = TOOLS.find(x => x.n === n), [o, c] = outcomeOf(r.v, n), cur = i === r.step - 1 && r.step <= r.calls.length; return <div key={n} className={"tr-s" + (cur ? " cur" : "")}><span className="ck">{cur ? <i className="spn"></i> : c === "x" ? <Ic n="ban" s={12}></Ic> : <Ic n="check" s={12} w={2.4}></Ic>}</span><span className="tx"><b>{t.l}</b><em className="mono">{t.n}</em></span>{!cur && <span className={"oc " + c}>{o}</span>}</div>; })}</div>
                {r.w > 0 && <p className="tp2-t">{r.words.slice(0, r.w).join(" ")}{r.w < r.words.length && <span className="car"></span>}</p>}
                {r.w >= r.words.length && <span className="tp2-m mono">1.4s · 2.1k tokens · ~$0.004 · nothing written</span>}
                {stale && <div className="tp2-st"><Ic n="refresh" s={12}></Ic><span>You changed the draft since this run</span><button className="btn sm" onClick={() => run(r.q)}>Run again</button></div>}
              </div>
            </div>); })}
      </div>
      <div className="tp2-c"><textarea rows={1} value={q} disabled={!ai} placeholder={ai ? "Ask as " + "Sarah Alvarez" + "…" : "Connect a provider first"} onChange={e => setQ(e.target.value)} onKeyDown={e => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); run(); } }}></textarea><button className="tp2-go" disabled={!q.trim() || !ai} onClick={() => run()} title="Run (Enter)"><Ic n="up" s={14} w={2.2}></Ic></button></div>
    </aside>
  );
}
Object.assign(window, { CreateIntro, WeekStrip, TryPanel });
