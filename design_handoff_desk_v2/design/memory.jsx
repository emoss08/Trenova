const MEM_PATH = <><path d="M7 4h10a2 2 0 0 1 2 2v14l-7-3.5L5 20V6a2 2 0 0 1 2-2z"></path><path d="M12 8.5v4M10 10.5h4"></path></>;
function MemIc({ s = 13, w = 2 }) { return <svg width={s} height={s} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={w} strokeLinecap="round" strokeLinejoin="round">{MEM_PATH}</svg>; }

const MEMORIES = {
  m1: { text: "Acme Manufacturing is billed net-45, not net-30. Terms were renegotiated in August.", scope: "Billing team", when: "Sep 18", src: "Acme rate review" },
  m2: { text: "Avery wants AR and detention summaries grouped by facility, not by load.", scope: "Just you", when: "Sep 30", src: "Detention by customer" },
};

TURNS.mrec = {
  time: "now", dur: "4.2s",
  recall: ["m1", "m2"],
  trace: [
    { pose: "recall", live: "Checking what Desk remembers about Acme", done: "Recalled 2 memories", k: "memory.search", v: "Acme · AR summary", r: "2 memories", t: "0.6s", ms: 1300 },
    { pose: "lookup", live: "Pulling Acme's open invoices", done: "Pulled Acme's open invoices", k: "invoices.search", v: "customer = Acme · open", r: "9 rows", t: "1.4s", ms: 1400 },
  ],
  text: [
    ["Acme has ", { r: 2, t: "9 open invoices" }, " totalling $64,310. I've grouped them by facility, ", { r: 1, t: "the way you prefer" }, "."],
    ["Using their ", { r: 1, t: "net-45 terms" }, ", only 3 are past due: $18,240, all from the Columbus plant. Under net-30 it would have looked like 7, so this is better than the aging report suggests."],
  ],
};
TURNS.msave = {
  time: "now", dur: "1.9s",
  save: { text: "Columbus, OH loads need a lumper receipt attached before they can be invoiced.", scope: "Billing team" },
  trace: [
    { pose: "remember", live: "Saving this to memory", done: "Saved 1 memory", k: "memory.save", v: "Columbus OH · lumper receipt", r: "saved", t: "0.4s", ms: 1100 },
  ],
  text: [["Got it. Before I draft an invoice for a Columbus, OH load, I'll check that a lumper receipt is attached and flag it if one is missing."]],
};
const MEM_DEMOS = { mrec: "Draft this week's AR summary for Acme", msave: "Remember that Columbus OH loads always need a lumper receipt before we bill" };

function MemRecall({ ids }) {
  const [open, setOpen] = React.useState(false);
  const [gone, setGone] = React.useState({});
  const live = ids.filter(id => !gone[id]).length;
  return (
    <div className={"mrc" + (open ? " open" : "")}>
      <button className="mrc-b" onClick={() => setOpen(o => !o)}>
        <span className="mem-ic"><MemIc s={12} /></span>
        <span>Used {live} {live === 1 ? "memory" : "memories"}</span>
        <span className="mrc-cv"><Ic n="chevR" s={11} w={2} /></span>
      </button>
      {open && (
        <div className="mrc-l">
          {ids.map(id => { const m = MEMORIES[id]; return (
            <div key={id} className={"mrc-i" + (gone[id] ? " gone" : "")}>
              <div className="mrc-t">{m.text}</div>
              <div className="mrc-m">
                {gone[id] ? <><span>Forgotten. Desk won't use this again.</span><button onClick={() => setGone(g => ({ ...g, [id]: false }))}>Undo</button></> : <>
                  <span className="mem-scope">{m.scope}</span><span>Saved {m.when} from “{m.src}”</span>
                  <button onClick={() => setGone(g => ({ ...g, [id]: true }))}>Forget</button>
                </>}
              </div>
            </div>
          ); })}
        </div>
      )}
    </div>
  );
}

function MemSave({ mem, ask }) {
  const [st, setSt] = React.useState(ask ? "ask" : "saved");
  const [text, setText] = React.useState(mem.text);
  const [draft, setDraft] = React.useState(mem.text);
  const [scope, setScope] = React.useState(mem.scope);
  const ta = React.useRef(null);
  React.useEffect(() => { if ((st === "edit" || st === "ask") && ta.current) { ta.current.style.height = "auto"; ta.current.style.height = ta.current.scrollHeight + "px"; } }, [st, draft]);
  const ScopePick = () => <span className="mem-seg">{["Just you", "Billing team"].map(s => <button key={s} className={scope === s ? "on" : ""} onClick={() => setScope(s)}>{s}</button>)}</span>;
  if (st === "removed" || st === "declined") return (
    <div className="mem mem-off"><span className="mem-ic"><MemIc s={12} /></span><span>{st === "removed" ? "Removed from memory" : "Not saved"}</span><button className="mem-l" onClick={() => setSt(st === "removed" ? "saved" : "ask")}>Undo</button></div>
  );
  if (st === "ask" || st === "edit") return (
    <div className={"mem mem-card" + (st === "ask" ? " ask" : "")}>
      <div className="mem-h"><span className="mem-ic"><MemIc s={12} /></span><b>{st === "ask" ? "Remember this for next time?" : "Edit memory"}</b></div>
      <textarea ref={ta} rows={1} value={draft} onChange={e => setDraft(e.target.value)} onKeyDown={e => { e.stopPropagation(); if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); if (draft.trim()) { setText(draft.trim()); setSt("saved"); } } if (e.key === "Escape" && st === "edit") { setDraft(text); setSt("saved"); } }}></textarea>
      <div className="mem-f">
        <span className="mem-fl">Visible to</span><ScopePick />
        <span style={{ flex: 1 }}></span>
        <button className="bt sm" onClick={() => { if (st === "edit") { setDraft(text); setSt("saved"); } else setSt("declined"); }}>{st === "ask" ? "Don't save" : "Cancel"}</button>
        <button className="mem-save" disabled={!draft.trim()} onClick={() => { setText(draft.trim()); setSt("saved"); }}>{st === "ask" ? "Save memory" : "Save"}</button>
      </div>
    </div>
  );
  return (
    <div className="mem mem-done">
      <span className="mem-ic on"><MemIc s={12} /></span>
      <span className="mem-tx"><em>Saved to memory</em>{text}</span>
      <span className="mem-scope">{scope}</span>
      <span className="mem-acts">
        <button className="mem-l" onClick={() => { setDraft(text); setSt("edit"); }}>Edit</button>
        <button className="mem-l" onClick={() => setSt("removed")}>Undo</button>
      </span>
    </div>
  );
}

Object.assign(window, { MemIc, MemRecall, MemSave, MEMORIES, MEM_DEMOS });
