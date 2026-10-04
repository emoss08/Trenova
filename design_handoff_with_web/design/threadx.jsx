// Thread extras: undo window, pinned facts, hand-off, schedule card, step explanations.
function UndoBar({ id, n, total, onUndo, onNow }) {
  const d = DECISIONS[id];
  const what = id === "d1" ? "Assigning you as biller on 11 items" : "Posting 15 invoices";
  return (
    <div className="udo" key={id}>
      <svg className="udo-ring" width="22" height="22" viewBox="0 0 22 22"><circle cx="11" cy="11" r="9" className="bg"></circle><circle cx="11" cy="11" r="9" className="fg" style={{ strokeDashoffset: 56.5 * (1 - n / total) }}></circle><text x="11" y="14.5" textAnchor="middle">{n}</text></svg>
      <span className="udo-t"><b>Approved {d.title.toLowerCase()}</b><span>{what} in {n}s</span></span>
      <button className="bt sm" onClick={onUndo}><Ic n="undo" s={12} w={2} />Undo</button>
      <button className="bt sm ghost" onClick={onNow}>Do it now</button>
    </div>
  );
}

function FactsBar({ facts, setFacts }) {
  const [adding, setAdding] = React.useState(false);
  const [v, setV] = React.useState("");
  const add = () => { const t = v.trim(); if (t && !facts.includes(t)) setFacts(f => [...f, t]); setV(""); setAdding(false); };
  return (
    <div className="fx" title="Agents keep these in mind for the whole conversation, even after it's compacted">
      <span className="fx-l"><Ic n="pin" s={12} />Keeping in mind</span>
      {facts.map(f => <span key={f} className="fx-c">{f}<button title="Unpin" onClick={() => setFacts(xs => xs.filter(x => x !== f))}><Ic n="x" s={10} w={2.2} /></button></span>)}
      {adding ? <input className="fx-in" autoFocus value={v} onChange={e => setV(e.target.value)} onBlur={add} onKeyDown={e => { if (e.key === "Enter") add(); if (e.key === "Escape") { setV(""); setAdding(false); } }} placeholder="e.g. Invoice date is Oct 3" />
        : <button className="fx-add" onClick={() => setAdding(true)}><Ic n="plus" s={11} w={2.2} />{facts.length ? "" : "Pin a fact"}</button>}
    </div>
  );
}

function HandoffMenu({ onPick }) {
  const [open, setOpen] = React.useState(false);
  const root = React.useRef(null);
  React.useEffect(() => { if (!open) return; const off = e => root.current && !root.current.contains(e.target) && setOpen(false); document.addEventListener("mousedown", off); return () => document.removeEventListener("mousedown", off); }, [open]);
  const list = AGENTS.filter(a => ["ag_dispatch", "ag_bex", "ag_detention", "ag_cash", "ag_receivables", "ag_updates", "ag_general"].includes(a.id));
  return (
    <span className="hom" ref={root}>
      <button className={"ib" + (open ? " on" : "")} title="Hand off to another agent" onClick={() => setOpen(o => !o)}><Ic n="handoff" s={14} /></button>
      {open && <div className="hom-p">
        <div className="hom-h"><b>Hand off to</b><span>Carries over a summary, pinned facts and pinned artifacts</span></div>
        {list.map(a => <button key={a.id} onClick={() => { setOpen(false); onPick(a); }}><AgentTile agent={a} size="xs" /><span><b>{a.name}</b><em>{a.desc}</em></span></button>)}
      </div>}
    </span>
  );
}

function HandoffCard({ it }) {
  return (
    <div className="ho">
      <div className="ho-h"><AgentTile agent={it.agent} size="sm" /><span><b>Handed off to {it.agent.name}</b><em>{it.time} · this conversation stays open here</em></span></div>
      <div className="ho-c">
        <span>Carried over</span>
        <div>{["Summary of this conversation", ...(it.facts.length ? [it.facts.length + " pinned " + (it.facts.length === 1 ? "fact" : "facts")] : []), "Billing queue table"].map(x => <span key={x} className="ho-chip"><Ic n="check" s={10} w={2.6} />{x}</span>)}</div>
      </div>
      <button className="ho-go">Open their conversation<Ic n="chevR" s={12} /></button>
    </div>
  );
}

function SchedCard({ sid }) {
  const list = useSched();
  const s = list.find(x => x.id === sid);
  if (!s) return <div className="schc gone"><Ic n="clock" s={13} />Schedule deleted</div>;
  return <div className="schc"><div className="schc-h"><b>Scheduled</b><span>Results will post into this conversation</span></div><SchedRow s={{ ...s, fresh: false }} /></div>;
}

const STEP_WHY = {
  "shipments.search": ["Shipments marked ready to invoice", "Transfers only apply to shipments that are ready to invoice, so it searched that status first.", "Searching every shipment — slower, and most can't transfer."],
  "billing.queue.list": ["The full billing queue, 15 items", "You asked what's in the queue, so it read all of it instead of a filtered slice.", "Showing only items ready for review — would have hidden the approved one."],
  "billing.queue.inspect": ["The biller field on each of the 15 items", "Posting needs a biller, so it checked every item one by one to be certain.", "Trusting the queue summary, which doesn't show billers."],
  "billing.queue.get": ["BQ-24101 and its rate con", "You named the item, so it opened that one directly.", "Searching the queue for Acme items."],
  "rating.compare": ["Six charge lines vs the Acme rate con", "Anything billed that isn't on the rate con gets flagged before posting.", "Skipping the check because the total looked normal."],
  "contracts.read": ["Acme's 2026 contract, section 4", "Fuel surcharge rules live in the contract, not the rate con.", "Using the default company FSC table."],
  "memory.search": ["Saved notes about Acme and AR summaries", "Your team saved how Acme is billed, so it checked memory before reading invoices.", "Using standard net-30 terms."],
  "documents.extract": ["Text and layout of the 3-page PDF", "It was classified as a rate confirmation, so it pulled that form's 12 standard fields.", "Reading it as a generic document — no fields."],
};
function StepWhy({ step }) {
  const w = STEP_WHY[step.k] || [step.v + " → " + step.r, "This tool answers the question directly with the least data.", "Asking you to narrow it down first."];
  return (
    <span className="why">
      <span><em>Saw</em>{w[0]}</span>
      <span><em>Because</em>{w[1]}</span>
      <span><em>Instead of</em>{w[2]}</span>
    </span>
  );
}

Object.assign(window, { UndoBar, FactsBar, HandoffMenu, HandoffCard, SchedCard, StepWhy });
