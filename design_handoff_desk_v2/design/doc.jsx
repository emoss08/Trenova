// Document artifact — a written brief the agent authored, with sources, versions, rewrite-on-select and edit mode.
const DOC_SRC = [
  { n: 1, code: "weather.alerts", label: "NWS Des Moines · Severe thunderstorm warning", detail: "Central Iowa · until 9:45 PM" },
  { n: 2, code: "shipments.track", label: "Live GPS on 3 loads", detail: "Last pings 2–6 min ago" },
  { n: 3, code: "shipments.list", label: "Today's board · 42 loads", art: "board" },
  { n: 4, code: "customers.get", label: "Dock hours · Acme, Bluewater, Northline", detail: "From customer records" },
  { n: 5, code: "messages.draft", label: "Delay notice to Acme", art: "email" },
];
const DOC_V2 = [
  { id: "sum", t: "sum", x: ["Three loads on I-80 and I-35 will arrive late tonight. None should miss its delivery window, but ", { b: "Acme and Bluewater need a heads-up before 7 PM" }, "."], alt: "Three loads will be late tonight; none miss their window. Tell Acme and Bluewater before 7 PM." },
  { id: "h1", t: "h", x: "What's happening" },
  { id: "p1", t: "p", x: ["A line of severe storms over central Iowa has slowed traffic on I-80 and I-35 since about 6 PM", { c: 1 }, ". The warning runs until 9:45 PM, and the three loads below are all inside it", { c: 2 }, "."], alt: "Storms over central Iowa have slowed I-80 and I-35 since 6 PM. The warning ends at 9:45 PM and covers all three loads." },
  { id: "h2", t: "h", x: "Affected loads" },
  { id: "loads", t: "loads", c: 3, rows: [["SEED-SHP-002", "Des Moines → Omaha", "Acme", "+2 h", "6:45 PM"], ["SEED-SHP-005", "Cedar Rapids → Madison", "Bluewater", "+1.5 h", "7:20 PM"], ["SEED-SHP-007", "Ames → Minneapolis", "Northline", "+3 h", "8:05 PM"]] },
  { id: "h3", t: "h", x: "What I'd do" },
  { id: "p2", t: "p", x: ["Acme and Bluewater close their docks at 7 PM", { c: 4 }, ", so tell both of them now and ask whether they'll hold a door. Northline receives overnight, so a note in the morning is enough."], alt: "Tell Acme and Bluewater now — their docks close at 7 PM. Northline receives overnight; a morning note is fine." },
  { id: "ol", t: "ol", items: ["Send the drafted delay notice to Acme", "Draft a matching note for Bluewater", "Re-check all three ETAs at 8 PM, after the warning clears"] },
  { id: "note", t: "note", x: "The delay notice to Acme is drafted and waiting in this conversation.", art: "email", c: 5 },
];
const DOC_V1 = DOC_V2.map(b => b.id === "p2" ? { ...b, x: ["Tell all three customers now so their docks can plan around the new times."] } : b.id === "sum" ? { ...b, x: ["Three loads on I-80 and I-35 will arrive late tonight. None should miss its delivery window."] } : b).filter(b => b.id !== "note");
Object.assign(ART2.doc, { docType: "Brief", author: "Dispatch agent", heading: "Storm impact on today's Midwest loads", basis: "from 42 loads and 3 weather alerts" });

const docText = x => typeof x === "string" ? x : x.map(s => typeof s === "string" ? s : s.b || "").join("");
const docWords = bs => bs.reduce((n, b) => n + (b.rows ? b.rows.length * 6 : 0) + (b.items ? b.items.join(" ") : docText(b.x || "")).split(/\s+/).filter(Boolean).length, 0);

function DocCite({ n, ctx }) {
  const s = DOC_SRC.find(x => x.n === n);
  const [open, setOpen] = React.useState(false);
  const t = React.useRef(null);
  const show = () => { clearTimeout(t.current); setOpen(true); };
  const hide = () => { t.current = setTimeout(() => setOpen(false), 120); };
  return (
    <span className="dx-cw" onMouseEnter={show} onMouseLeave={hide} contentEditable={false}>
      <button className="dx-c" onClick={() => s.art ? ctx.open && ctx.open(s.art) : null}>{n}</button>
      {open && <span className="dx-cp" onMouseEnter={show} onMouseLeave={hide}>
        <code>{s.code}</code><b>{s.label}</b>{s.detail && <em>{s.detail}</em>}
        {s.art && <button onClick={() => ctx.open && ctx.open(s.art)}><AI n="a-ext" s={11} />Open artifact</button>}
      </span>}
    </span>
  );
}

function DocRich({ x, ctx }) {
  if (typeof x === "string") return x;
  return x.map((s, i) => typeof s === "string" ? <React.Fragment key={i}>{s}</React.Fragment> : s.c ? <DocCite key={i} n={s.c} ctx={ctx} /> : <b key={i}>{s.b}</b>);
}

function DocMenu({ label, icon, children, align = "right" }) {
  const [open, setOpen] = React.useState(false);
  const root = React.useRef(null);
  React.useEffect(() => { if (!open) return; const off = e => root.current && !root.current.contains(e.target) && setOpen(false); document.addEventListener("mousedown", off); return () => document.removeEventListener("mousedown", off); }, [open]);
  return (
    <span className="dx-m" ref={root}>
      <button className={"dx-tb" + (open ? " on" : "")} onClick={() => setOpen(o => !o)}>{icon}{label}<AI n="a-down" s={10} w={2.4} /></button>
      {open && <div className={"dx-mp " + align} onClick={() => setOpen(false)}>{children}</div>}
    </span>
  );
}

function DocBody({ a, ctx }) {
  const [vers, setVers] = React.useState([{ v: 1, at: "10:24 PM", note: "First draft", blocks: DOC_V1 }, { v: 2, at: "10:26 PM", note: "Added dock hours and the Acme draft", blocks: DOC_V2 }]);
  const [view, setView] = React.useState(null);
  const latest = vers[vers.length - 1];
  const cur = view == null ? latest : vers[view];
  const [blocks, setBlocks] = React.useState(latest.blocks);
  React.useEffect(() => { setBlocks(cur.blocks); }, [cur]);
  const [mode, setMode] = React.useState("read");
  const [dirty, setDirty] = React.useState(false);
  const [sel, setSel] = React.useState(null);
  const [ask, setAsk] = React.useState("");
  const [busy, setBusy] = React.useState(null);
  const [sugg, setSugg] = React.useState(null);
  const [vOpen, setVOpen] = React.useState(false);
  const [toast, setToast] = React.useState(null);
  const paper = React.useRef(null);
  const vRoot = React.useRef(null);
  React.useEffect(() => { if (!vOpen) return; const off = e => vRoot.current && !vRoot.current.contains(e.target) && setVOpen(false); document.addEventListener("mousedown", off); return () => document.removeEventListener("mousedown", off); }, [vOpen]);
  const flash = m => { setToast(m); setTimeout(() => setToast(t => t === m ? null : t), 1600); };
  const words = docWords(blocks);
  const old = view != null && view !== vers.length - 1;
  const heads = blocks.filter(b => b.t === "h");

  const scrollTo = id => {
    const el = paper.current && paper.current.querySelector('[data-blk="' + id + '"]'); if (!el) return;
    let sc = el.parentElement; while (sc && sc !== document.body && !(sc.scrollHeight > sc.clientHeight && /auto|scroll/.test(getComputedStyle(sc).overflowY))) sc = sc.parentElement;
    if (sc) sc.scrollTo({ top: sc.scrollTop + el.getBoundingClientRect().top - sc.getBoundingClientRect().top - 12, behavior: "smooth" });
    el.classList.remove("dx-hit"); void el.offsetWidth; el.classList.add("dx-hit");
  };
  const onUp = () => {
    if (mode !== "read" || old || busy || sugg) return;
    const s = window.getSelection(); if (!s || s.isCollapsed || !s.toString().trim()) { setSel(null); return; }
    const node = s.anchorNode && (s.anchorNode.nodeType === 1 ? s.anchorNode : s.anchorNode.parentElement);
    const blk = node && node.closest("[data-blk]"); if (!blk || !paper.current.contains(blk)) return;
    const b = blocks.find(x => x.id === blk.dataset.blk); if (!b || !(b.t === "p" || b.t === "sum")) { setSel(null); return; }
    const r = s.getRangeAt(0).getBoundingClientRect(), pr = paper.current.getBoundingClientRect();
    setSel({ id: b.id, x: Math.max(8, Math.min(pr.width - 250, r.left - pr.left + r.width / 2 - 125)), y: r.top - pr.top - 44 }); setAsk("");
  };
  React.useEffect(() => { const d = e => { if (!e.target.closest(".dx-sel")) setTimeout(() => { const s = window.getSelection(); if (!s || s.isCollapsed) setSel(null); }, 0); }; document.addEventListener("mousedown", d); return () => document.removeEventListener("mousedown", d); }, []);
  const rewrite = (how) => {
    const b = blocks.find(x => x.id === sel.id); const id = sel.id; setSel(null); window.getSelection().removeAllRanges(); setBusy(id);
    const first = docText(b.x).split(/(?<=\.)\s/)[0];
    const next = how === "shorter" ? (b.alt || first) : how === "plain" ? (b.alt || docText(b.x)).replace(/ — /g, ". ") : (b.alt || first);
    setTimeout(() => { setBusy(null); setSugg({ id, how, next, ask: how === "ask" ? ask : null }); }, 1100);
  };
  const accept = () => {
    const nb = blocks.map(b => b.id === sugg.id ? { ...b, x: sugg.next, alt: null } : b);
    const note = sugg.how === "shorter" ? "Shortened a paragraph" : sugg.how === "plain" ? "Simplified a paragraph" : "Rewrote: “" + (sugg.ask || "paragraph") + "”";
    setVers(v => [...v, { v: v.length + 1, at: "Just now", note, blocks: nb }]); setView(null); setSugg(null); flash("Saved as v" + (vers.length + 1));
  };
  const save = () => {
    const nb = blocks.map(b => { const el = paper.current.querySelector('[data-blk="' + b.id + '"]'); if (!el) return b;
      if (b.t === "ol") return { ...b, items: [...el.querySelectorAll("li")].map(li => li.innerText.trim()).filter(Boolean) };
      if (b.t === "loads") return b;
      return { ...b, x: el.innerText.replace(/\n+/g, " ").trim(), alt: null }; });
    setVers(v => [...v, { v: v.length + 1, at: "Just now", note: "Your edits", blocks: nb, you: true }]); setView(null); setDirty(false); setMode("read"); flash("Saved as v" + (vers.length + 1));
  };
  const discard = () => { setBlocks(b => b.map(x => ({ ...x }))); setDirty(false); setMode("read"); };
  const setM = m => { if (m === mode) return; if (mode === "edit" && dirty) { discard(); return; } setSel(null); setSugg(null); setMode(m); };
  const ed = mode === "edit" && !old;

  const renderBlock = b => {
    const p = { key: b.id + ":" + cur.v + ":" + mode, "data-blk": b.id, contentEditable: ed && b.t !== "loads" ? true : undefined, suppressContentEditableWarning: true, onInput: () => setDirty(true), className: "dx-b dx-" + b.t + (busy === b.id ? " busy" : "") + (sugg && sugg.id === b.id ? " sg" : "") };
    if (sugg && sugg.id === b.id) return (
      <div key={b.id} data-blk={b.id} className="dx-b dx-sgw">
        <div className="dx-sgh"><AI n="a-diff" s={12} />{sugg.how === "shorter" ? "Shorter" : sugg.how === "plain" ? "Plainer" : "“" + sugg.ask + "”"}<span>{docText(b.x).split(/\s+/).length} → {sugg.next.split(/\s+/).length} words</span></div>
        <p className="dx-old">{docText(b.x)}</p><p className="dx-new">{sugg.next}</p>
        <div className="dx-sga"><button className="ax-btn ghost" onClick={() => setSugg(null)}>Keep original</button><button className="ax-btn ghost" onClick={() => { const s = sugg; setSugg(null); setBusy(s.id); setTimeout(() => { setBusy(null); setSugg({ ...s, next: docText(b.x).split(/(?<=\.)\s/).slice(-1)[0] === s.next ? s.next : s.next }); }, 900); }}>Try again</button><button className="ax-btn ink" onClick={accept}><AI n="a-check" s={12} w={2.4} />Accept</button></div>
      </div>);
    if (b.t === "h") return <h3 {...p}>{b.x}</h3>;
    if (b.t === "sum") return <p {...p}><DocRich x={b.x} ctx={ctx} /></p>;
    if (b.t === "p") return <p {...p}><DocRich x={b.x} ctx={ctx} /></p>;
    if (b.t === "ol") return <ol {...p}>{b.items.map((it, i) => <li key={i}>{it}</li>)}</ol>;
    if (b.t === "note") return <div {...p}><AI n="a-mail" s={13} /><span>{b.x}</span>{!ed && <button contentEditable={false} onClick={() => ctx.open && ctx.open(b.art)}>Open draft</button>}</div>;
    if (b.t === "loads") return (
      <div {...p}><table><thead><tr><th>Load</th><th>Lane</th><th>Customer</th><th className="ar">Delay</th><th className="ar">New ETA</th></tr></thead>
        <tbody>{b.rows.map(r => <tr key={r[0]}><td className="ax-id">{r[0]}</td><td>{r[1]}</td><td>{r[2]}</td><td className="ar"><span className="dx-late">{r[3]}</span></td><td className="ar ax-mono">{r[4]}</td></tr>)}</tbody></table>
        <div className="dx-tsrc">Source <DocCite n={b.c} ctx={ctx} /></div></div>);
    return null;
  };

  return (
    <div className={"dx" + (ed ? " editing" : "")}>
      <div className="dx-bar">
        <div className="dx-seg">{["read", "edit"].map(m => <button key={m} className={mode === m ? "on" : ""} disabled={old && m === "edit"} onClick={() => setM(m)}>{m === "read" ? "Read" : "Edit"}</button>)}</div>
        <span className="axv" ref={vRoot}>
          <button className={"axv-b" + (vOpen ? " on" : "") + (old ? " old" : "")} onClick={() => setVOpen(o => !o)}>{"v" + cur.v}<svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round"><path d="M6 9l6 6 6-6"></path></svg></button>
          {vOpen && <div className="axv-pop">{[...vers].reverse().map(v => { const i = vers.indexOf(v); const on = (view == null ? vers.length - 1 : view) === i; return (
            <button key={v.v} className={"axv-r" + (on ? " on" : "")} onClick={() => { setView(i === vers.length - 1 ? null : i); setVOpen(false); setMode("read"); setSugg(null); }}>
              <b>{"v" + v.v}</b><span><em>{v.note}</em><i>{(v.you ? "You · " : "Dispatch · ") + v.at + (i === vers.length - 1 ? " · Latest" : "")}</i></span>{on && <AI n="a-check" s={12} w={2.4} />}
            </button>); })}</div>}
        </span>
        <span style={{ flex: 1 }}></span>
        <DocMenu label="Export" icon={<AI n="a-dl" s={13} />}>
          <button onClick={() => flash("Downloading PDF…")}><span className="dx-fx">PDF</span>Download as PDF</button>
          <button onClick={() => flash("Downloading .docx…")}><span className="dx-fx">DOC</span>Word document</button>
          <button onClick={() => flash("Copied as Markdown")}><span className="dx-fx">MD</span>Copy as Markdown</button>
          <button onClick={() => flash("Copied plain text")}><AI n="a-copy" s={13} />Copy text</button>
        </DocMenu>
      </div>
      {heads.length > 1 && <nav className="dx-toc">{heads.map(h => <button key={h.id} onClick={() => scrollTo(h.id)}>{h.x}</button>)}</nav>}
      {old && <div className="dx-oldbar"><span>Viewing <b>v{cur.v}</b> · {cur.note}</span><button onClick={() => setView(null)}>Back to latest</button><button className="ink" onClick={() => { setVers(v => [...v, { v: v.length + 1, at: "Just now", note: "Restored v" + cur.v, blocks: cur.blocks, you: true }]); setView(null); flash("Restored as v" + (vers.length + 1)); }}>Restore</button></div>}
      <article className="dx-paper" ref={paper} onMouseUp={onUp}>
        <div className="dx-kick"><span>{a.docType || "Document"}</span><i></i><span>{words} words · {Math.max(1, Math.round(words / 220))} min read</span></div>
        <h2 className="dx-h">{a.heading || a.title}</h2>
        <div className="dx-by"><span className="dx-av">D</span><span>{a.author || "Dispatch agent"} · {cur.at === "Just now" ? "edited just now" : "written " + cur.at} · {a.basis || "from this conversation"}</span></div>
        {blocks.map(renderBlock)}
        <section className="dx-srcs">
          <h4>Sources</h4>
          {DOC_SRC.map(s => <div key={s.n} className={s.art ? "go" : ""} onClick={() => s.art && ctx.open && ctx.open(s.art)}><span className="dx-c">{s.n}</span><span><b>{s.label}</b><code>{s.code}</code></span>{s.art && <AI n="a-ext" s={11} />}</div>)}
        </section>
        {sel && <div className="dx-sel" style={{ left: sel.x, top: sel.y }} onMouseDown={e => e.target.tagName !== "INPUT" && e.preventDefault()}>
          <button onClick={() => rewrite("shorter")}>Shorter</button><button onClick={() => rewrite("plain")}>Plainer</button><i></i>
          <input value={ask} onChange={e => setAsk(e.target.value)} placeholder="Ask to change…" onKeyDown={e => { e.stopPropagation(); if (e.key === "Enter" && ask.trim()) rewrite("ask"); if (e.key === "Escape") setSel(null); }} />
        </div>}
      </article>
      {ed && <div className="dx-save"><span className={dirty ? "on" : ""}>{dirty ? "Unsaved changes" : "Click any paragraph to edit"}</span><span style={{ flex: 1 }}></span><button className="ax-btn ghost" onClick={discard}>{dirty ? "Discard" : "Done"}</button>{dirty && <button className="ax-btn ink" onClick={save}>Save as v{vers.length + 1}</button>}</div>}
      {toast && <div className="dx-toast"><AI n="a-check" s={12} w={2.4} />{toast}</div>}
    </div>
  );
}

window.Body_doc = DocBody;
