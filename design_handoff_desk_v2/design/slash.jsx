const SLASH_COMMANDS = [
  { name: "schedule", icon: "receipt", description: "Run a request on a schedule and post the results here", slots: [{ name: "when", hint: "every Monday at 8am" }, { name: "request", hint: "what to ask" }], template: "{when}, {request}" },
  { name: "status", icon: "truck", description: "Where a shipment is and what is holding it up", slots: [{ name: "shipment", hint: "PRO or shipment number" }], template: "What is the status of shipment {shipment} right now, and is anything holding it up?" },
  { name: "quote", icon: "route", description: "What we would charge for a lane", slots: [{ name: "origin", hint: "origin city" }, { name: "destination", hint: "destination city" }], template: "Quote a truckload shipment from {origin} to {destination}. Say which rate applied and what it is made of." },
  { name: "report", icon: "compass", description: "Run a saved report and show the results", slots: [{ name: "report", hint: "report name" }], template: "Run the report {report} and show me the results." },
  { name: "explain", icon: "search", description: "Explain what is on this page", slots: [], template: "Explain what I am looking at on this page: what the filters mean, what the figures say, and what stands out." },
];
const SLASH_SUGGESTIONS = [
  { label: "What's blocking the billing queue?", prompt: "What's blocking the billing queue?" },
  { label: "Which invoices can post today?", prompt: "Which invoices in the billing queue can post today?" },
  { label: "Who's past 60 days?", prompt: "Which customers have invoices more than 60 days past due?" },
];

function slashParse(draft) {
  if (!draft.startsWith("/") || draft.includes("\n")) return null;
  const body = draft.slice(1), sp = body.search(/\s/);
  const name = (sp === -1 ? body : body.slice(0, sp)).toLowerCase();
  const command = SLASH_COMMANDS.find(c => c.name === name);
  if (!command) return null;
  let rest = sp === -1 ? "" : body.slice(sp + 1).trim();
  const args = [];
  command.slots.forEach((s, i) => {
    if (i === command.slots.length - 1) { args.push(rest.trim()); return; }
    const cut = rest.search(/\s/);
    if (cut === -1) { args.push(rest.trim()); rest = ""; } else { args.push(rest.slice(0, cut)); rest = rest.slice(cut + 1).trimStart(); }
  });
  return { command, args, started: sp !== -1, complete: args.length === command.slots.length && args.every(a => a !== "") };
}
const slashFill = (c, args) => c.slots.reduce((p, s, i) => p.split("{" + s.name + "}").join((args[i] || "").trim()), c.template);

function slashEntries(q) {
  const typed = slashParse("/" + q);
  if (typed && /\s/.test(q)) return [{ kind: "command", command: typed.command }];
  const n = q.trim().toLowerCase();
  return [
    ...SLASH_COMMANDS.filter(c => !n || c.name.startsWith(n)).map(c => ({ kind: "command", command: c })),
    ...SLASH_SUGGESTIONS.filter(s => !n || (s.label + " " + s.prompt).toLowerCase().includes(n)).map(s => ({ kind: "question", ...s })),
  ];
}

function useSlash(value, setValue, taRef, onSendText) {
  const [hi, setHi] = React.useState(0);
  const [dismissed, setDismissed] = React.useState(null);
  const active = value.startsWith("/") && !value.includes("\n") && dismissed !== value;
  const q = active ? value.slice(1) : "";
  const parsed = active ? slashParse(value) : null;
  const entries = active ? slashEntries(q) : [];
  React.useEffect(() => { setHi(0); }, [q.split(" ")[0]]);
  const focusEnd = () => requestAnimationFrame(() => { const t = taRef.current; if (t) { t.focus(); t.setSelectionRange(t.value.length, t.value.length); } });
  const choose = e => {
    if (!e) return;
    if (e.kind === "question") { setValue(""); onSendText(e.prompt); return; }
    if (!e.command.slots.length) { setValue(""); onSendText(slashFill(e.command, [])); return; }
    setValue("/" + e.command.name + " "); focusEnd();
  };
  const onKey = ev => {
    if (!active) return false;
    if (parsed && parsed.started) {
      const t = taRef.current;
      const caret = t ? t.selectionStart : value.length;
      const cmdEnd = parsed.command.name.length + 2;
      if (ev.key === "Backspace" && t && t.selectionStart === t.selectionEnd && caret <= cmdEnd && value.slice(cmdEnd).trim() === "") { ev.preventDefault(); setValue(""); return true; }
      if (ev.key === "Enter" && !ev.shiftKey) { ev.preventDefault(); if (parsed.complete) { setValue(""); onSendText(slashFill(parsed.command, parsed.args)); } return true; }
      if (ev.key === "Escape") { ev.preventDefault(); setDismissed(value); return true; }
      return false;
    }
    if (ev.key === "ArrowDown") { ev.preventDefault(); setHi(h => Math.min(entries.length - 1, h + 1)); return true; }
    if (ev.key === "ArrowUp") { ev.preventDefault(); setHi(h => Math.max(0, h - 1)); return true; }
    if ((ev.key === "Enter" && !ev.shiftKey) || ev.key === "Tab") { if (entries[hi]) { ev.preventDefault(); choose(entries[hi]); return true; } }
    if (ev.key === "Escape") { ev.preventDefault(); setDismissed(value); return true; }
    return false;
  };
  return { active, q, parsed, entries, hi, setHi, choose, onKey };
}

function SlashMenu({ sl }) {
  if (!sl.active) return null;
  const p = sl.parsed;
  if (p && p.started) {
    const c = p.command;
    return (
      <div className="sl sl-fill" onMouseDown={e => e.preventDefault()}>
        <div className="sl-fh"><span className="sl-ic"><svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">{AG_ICON[c.icon]}</svg></span><b>{"/" + c.name}</b><span>{c.description}</span></div>
        <div className="sl-slots">{c.slots.map((s, i) => <span key={s.name} className={"sl-slot" + (p.args[i] ? " done" : i === p.args.filter(Boolean).length ? " cur" : "")}>{p.args[i] || s.hint}</span>)}</div>
        <div className="sl-prev"><em>Desk will be asked</em>{slashFill(c, p.args.map((a, i) => a || "…"))}</div>
        <div className="mn-ft"><span><span className="kbd">↵</span>{p.complete ? "ask" : "fill every slot to ask"}</span><span><span className="kbd">Esc</span>type it as plain text</span></div>
      </div>
    );
  }
  let shownQ = false;
  return (
    <div className="sl" onMouseDown={e => e.preventDefault()}>
      <div className="sl-l">
        {sl.entries.some(e => e.kind === "command") && <div className="mn-h">Commands</div>}
        {sl.entries.map((e, i) => {
          const head = e.kind === "question" && !shownQ; if (head) shownQ = true;
          return (
            <React.Fragment key={e.kind + (e.command ? e.command.name : e.label)}>
              {head && <div className="mn-h">Ask Billing Specialist</div>}
              <button className={"sl-r" + (sl.hi === i ? " hi" : "")} onMouseMove={() => sl.hi !== i && sl.setHi(i)} onClick={() => sl.choose(e)}>
                {e.kind === "command" ? <>
                  <span className="sl-ic"><svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">{AG_ICON[e.command.icon]}</svg></span>
                  <span className="sl-t"><b><span className="sl-nm">{"/" + e.command.name}</span>{e.command.slots.map(s => <i key={s.name}>{s.hint}</i>)}</b><em>{e.command.description}</em></span>
                </> : <>
                  <span className="sl-ic q"><Ic n="chat" s={13} /></span>
                  <span className="sl-t"><b>{e.label}</b></span>
                </>}
                {sl.hi === i && <span className="kbd">{e.kind === "command" && e.command.slots.length ? "Tab" : "↵"}</span>}
              </button>
            </React.Fragment>
          );
        })}
        {!sl.entries.length && <div className="mn-empty">{"No command called “/" + sl.q + "”. Press Esc to send it as text."}</div>}
      </div>
    </div>
  );
}

function SlashMirror({ sl }) {
  const p = sl.parsed;
  if (!sl.active || !p || !p.started) return null;
  const hint = p.command.slots.filter((s, i) => !p.args[i]).map(s => s.hint).join("  ");
  return <div className="sl-mirror" aria-hidden="true"><span className="sl-cmd">{"/" + p.command.name}</span><span className="sl-typed">{" " + sl.q.slice(p.command.name.length + 1)}</span>{hint && <span className="sl-ghost">{(sl.q.endsWith(" ") ? "" : " ") + hint}</span>}</div>;
}

Object.assign(window, { useSlash, SlashMenu, SlashMirror, SLASH_COMMANDS });
