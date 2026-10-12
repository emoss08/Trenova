const { useState, useMemo, useRef, useEffect, useLayoutEffect, useCallback } = React;
const SCOPE_IDS = SCOPES.map((s) => s.id);
const MENTION_RE = /(^|\s)@(\w*)$/;
const ALIASES = { doc: "document", docs: "document", ship: "shipment", load: "shipment", loads: "shipment", driver: "worker", drivers: "worker", cmd: "command" };

function NovaCard({ nova, onCite, onPeek, onDesk, onCopy, onFollow }) {
  const toks = useMemo(() => tokenize(nova.text), [nova.text]);
  const busy = nova.phase !== "done";
  return (
    <div className={"nv" + (busy ? " busy" : "")} onMouseDown={(e) => e.preventDefault()}>
      <div className="nv-h"><NovaMark s={14} busy={busy}></NovaMark><b>Nova</b><span>{nova.phase === "think" ? "Working…" : nova.phase === "stream" ? "Answering" : "Answered from your records"}</span><span className="sp"></span>{nova.subject && <span className="mono" style={{ fontSize: 11.5 }}>{nova.subject.title}</span>}</div>
      <p className="nv-q">{nova.q}</p>
      {nova.phase === "think" || nova.shown === 0 ? (
        <ul className="nv-st">{nova.steps.slice(0, nova.stepAt + 1).map((s, i) => <li key={i}>{i < nova.stepAt || nova.phase !== "think" ? <Ic n="check" s={12}></Ic> : <span className="spin"></span>}{s}</li>)}</ul>
      ) : (
        <p className="nv-t">
          {toks.slice(0, nova.shown).map((t, i) => {
            if (typeof t === "string") return t;
            const r = RECORD_BY_KEY[t.ref];
            if (!r) return null;
            return <span key={i} className="ct" onClick={() => onCite(r)} onMouseEnter={() => onPeek(r)} onMouseLeave={() => onPeek(null)}><i style={{ background: ENT[r.type].dot }}></i>{r.type === "shipment" || r.type === "worker" ? r.id : r.title}</span>;
          })}
          {nova.phase === "stream" && <span className="car"></span>}
        </p>
      )}
      {nova.phase === "done" && <div className="nv-f">
        <button className="pri" onClick={onDesk}><Ic n="desk" s={13}></Ic>Continue in Desk<Keys k={[MOD, "↵"]}></Keys></button>
        <button onClick={onCopy}><Ic n="copy" s={13}></Ic>Copy</button>
        <button onClick={onFollow}><Ic n="enter" s={13}></Ic>Follow up</button>
        <span>Hover a record to peek</span>
      </div>}
    </div>
  );
}

function Palette({ theme, onTheme, onClose, onNavigate, onToggleSidebar, appToast }) {
  const [query, setQuery] = useState("");
  const [scope, setScope] = useState("all");
  const [subject, setSubject] = useState(null);
  const [sel, setSel] = useState("");
  const [actOpen, setActOpen] = useState(false);
  const [actQ, setActQ] = useState("");
  const [actSel, setActSel] = useState(0);
  const [mIdx, setMIdx] = useState(0);
  const [nova, setNova] = useState(null);
  const [toast, setToast] = useState(null);
  const [pinned, setPinned] = useState(INITIAL_PINNED);
  const [recents, setRecents] = useState(INITIAL_RECENTS);
  const [peek, setPeek] = useState(null);
  const inputRef = useRef(null), actRef = useRef(null), lsRef = useRef(null), indRef = useRef(null), toastT = useRef(0);

  const commands = useMemo(() => buildCommands(theme), [theme]);
  const ctx = { pinned, recents, commands };

  // ——— parse the box ———
  let sc = scope, q = query;
  if (!subject && sc === "all" && q.startsWith(">")) { sc = "command"; q = q.slice(1); }
  const mm = !subject ? MENTION_RE.exec(q) : null;
  const mText = mm ? mm[2].toLowerCase() : null;
  if (mm) q = q.replace(MENTION_RE, "");
  const mentionOpts = mm ? SCOPES.filter((s) => s.id !== "all" && (s.id.startsWith(mText) || s.label.toLowerCase().startsWith(mText) || ALIASES[mText] === s.id)) : [];
  const mentionOpen = mentionOpts.length > 0;
  const qt = q.trim();

  const allHits = useMemo(() => (qt && !subject ? search(qt, "all", commands) : []), [qt, subject, commands]);
  const counts = useMemo(() => { const c = {}; allHits.forEach((h) => { const s = scopeOf(h.item); c[s] = (c[s] || 0) + 1; }); c.all = allHits.length; return c; }, [allHits]);
  const question = !subject && isQuestion(qt) ? qt : null;

  const { sections, mOf, empty } = useMemo(() => {
    const mOf = {}; const S = []; let empty = false;
    const ask = (text, subj) => ({ kind: "ask", key: "ask:" + text, q: text, subject: subj || null });
    if (subject) {
      const items = qt ? [ask(qt, subject)] : SUBJECT_QUESTIONS[subject.type].map((x) => ask(x, subject));
      S.push({ id: "ask", head: "Ask Nova about " + subject.title, items });
    } else if (!qt && sc === "all") {
      S.push({ id: "recent", head: "Jump back in", items: recents.map(([k, t]) => ({ ...RECORD_BY_KEY[k], key: "recent:" + k, time: t })).filter((x) => x.kind) });
      S.push({ id: "att", head: "Needs you", items: ATTENTION });
      S.push({ id: "pinned", head: "Pinned", items: pinned.map((id) => ({ ...PAGE_BY_ID[id], pinned: true })) });
      S.push({ id: "quick", head: "Quick actions", items: ["new-shipment", "upload", "nova-new", "insights"].map((id) => commands.find((c) => c.id === id)) });
    } else if (!qt) {
      if (sc === "command") Object.keys(CMD_GROUPS).forEach((g) => S.push({ id: "g:" + g, head: CMD_GROUPS[g], items: commands.filter((c) => c.group === g) }));
      else if (sc === "page") S.push({ id: "pages", head: "All pages", items: PAGES });
      else {
        const rec = recents.map(([k]) => RECORD_BY_KEY[k]).filter((x) => x && x.type === sc);
        S.push({ id: "rec", head: "Recent", items: rec });
        S.push({ id: "all", head: "All " + ENT[sc].plural.toLowerCase(), items: RECORDS.filter((r) => r.type === sc && !rec.includes(r)) });
      }
    } else {
      const hits = sc === "all" ? allHits : allHits.filter((h) => scopeOf(h.item) === sc);
      hits.forEach((h) => (mOf[h.item.key] = h.m));
      if (question) S.push({ id: "nova", head: "Nova", items: [ask(question)] });
      if (sc === "all") {
        const top = hits[0] && hits[0].m.score >= 0.9 ? hits[0].item : null;
        if (top) S.push({ id: "top", head: "Top hit", items: [top] });
        ["shipment", "customer", "worker", "document", "page", "command"].forEach((t) => {
          const list = hits.filter((h) => scopeOf(h.item) === t && h.item !== top).map((h) => h.item);
          const cap = t === "page" ? 4 : 3;
          if (list.length) S.push({ id: "t:" + t, head: SCOPES.find((s) => s.id === t).label, count: list.length + (top && scopeOf(top) === t ? 1 : 0), seeAll: list.length > cap ? t : null, items: list.slice(0, cap) });
        });
      } else if (sc === "command") {
        Object.keys(CMD_GROUPS).forEach((g) => { const items = hits.filter((h) => h.item.group === g).map((h) => h.item); if (items.length) S.push({ id: "g:" + g, head: CMD_GROUPS[g], items }); });
      } else if (hits.length) S.push({ id: "res", head: SCOPES.find((s) => s.id === sc).label, count: hits.length, items: hits.slice(0, 50).map((h) => h.item) });
      empty = hits.length === 0 && !question;
      if (!question && (sc === "all" || ENT[sc])) S.push({ id: "askf", head: empty ? "Try asking instead" : "Nova", items: [ask(qt)] });
      if (empty && sc !== "all") S.push({ id: "widen", head: "Or", items: [{ kind: "more", key: "more:all", label: "Search everything for “" + qt + "”" }] });
    }
    return { sections: S.filter((s) => s.items.length), mOf, empty };
  }, [subject, qt, sc, recents, pinned, commands, allHits, question]);

  const novaHere = nova && (subject ? nova.subject === subject && (!qt || nova.q === qt) : nova.q === question);
  const visible = !novaHere ? sections : !subject ? sections.filter((s) => s.id !== "nova") : qt ? sections.filter((s) => s.id !== "ask") : sections.map((s) => (s.id === "ask" ? { ...s, head: "Ask something else", items: s.items.filter((i) => i.q !== nova.q) } : s)).filter((s) => s.items.length);
  const items = useMemo(() => visible.flatMap((s) => s.items), [visible]);
  const selKey = items.some((i) => i.key === sel) ? sel : items[0] ? items[0].key : "";
  const selItem = items.find((i) => i.key === selKey) || null;
  const baseItem = selItem && selItem.kind === "record" && selItem.key.startsWith("recent:") ? RECORD_BY_KEY[selItem.key.slice(7)] : selItem;
  const actions = useMemo(() => buildActions(baseItem, ctx), [baseItem, pinned, commands]);
  const pvItem = peek || baseItem;
  const pvActions = useMemo(() => (peek ? buildActions(peek, ctx) : actions), [peek, actions]);

  const actList = actions.filter((a) => !actQ || a.label.toLowerCase().includes(actQ.toLowerCase()));

  // ——— selection indicator + keep in view ———
  useLayoutEffect(() => {
    const ls = lsRef.current, ind = indRef.current;
    if (!ls || !ind) return;
    const el = ls.querySelector(`[data-k="${CSS.escape(selKey)}"]`);
    if (!el) { ind.style.opacity = 0; return; }
    ind.style.opacity = 1;
    ind.style.transform = `translateY(${el.offsetTop}px)`;
    ind.style.height = el.offsetHeight + "px";
    if (el.offsetTop < ls.scrollTop + 36) ls.scrollTop = Math.max(0, el.offsetTop - 36);
    else if (el.offsetTop + el.offsetHeight > ls.scrollTop + ls.clientHeight - 8) ls.scrollTop = el.offsetTop + el.offsetHeight - ls.clientHeight + 8;
  });
  useEffect(() => { inputRef.current && inputRef.current.focus(); }, [subject, scope]);
  useEffect(() => { if (actOpen && actRef.current) actRef.current.focus(); }, [actOpen]);

  // ——— Nova ———
  useEffect(() => {
    if (!nova || nova.phase === "done") return;
    const total = tokenize(nova.text).length, think = nova.steps.length * 560;
    const tick = () => setNova((n) => {
      if (!n || n.phase === "done") return n;
      const el = Date.now() - n.t0;
      if (el < think) { const s = Math.floor(el / 560); return s === n.stepAt ? n : { ...n, stepAt: s }; }
      const shown = Math.min(total, Math.floor((el - think) / 15));
      return { ...n, phase: shown >= total ? "done" : "stream", stepAt: n.steps.length, shown };
    });
    const id = setInterval(tick, 30);
    return () => clearInterval(id);
  }, [nova && nova.t0, nova && nova.phase === "done"]);

  const startNova = (it) => {
    const a = novaAnswer(it.q, it.subject, allHits);
    setActOpen(false);
    setNova({ q: it.q, subject: it.subject, steps: a.steps, text: a.text, phase: "think", stepAt: 0, shown: 0, t0: Date.now() });
    if (it.subject && !qt) setQuery("");
  };

  const say = (msg) => { clearTimeout(toastT.current); setToast(msg); toastT.current = setTimeout(() => setToast(null), 1800); };
  const touch = (rec) => setRecents((r) => [[rec.key, "now"], ...r.filter((x) => x[0] !== rec.key)].slice(0, 5));

  const runCommand = (c) => {
    if (c.theme) { onTheme(c.theme); say(c.label.replace("Use ", "").replace(/^./, (x) => x.toUpperCase()) + " on"); return; }
    if (c.id === "sidebar") { onToggleSidebar(); onClose(); return; }
    if (c.id === "copy-page") { say("Link to this page copied"); return; }
    if (c.id === "read-all") { say("All caught up"); return; }
    if (c.id === "logout") { onClose(); appToast("Signed out — this is a prototype, so you're still here"); return; }
    if (c.to) { const p = PAGE_BY_ID[c.to]; onNavigate(c.group === "create" ? c.label : p.title, c.group === "create" ? [...p.trail, p.title] : p.trail); onClose(); return; }
    onClose(); appToast(c.label);
  };
  const run = (intent) => {
    if (!intent) return;
    switch (intent.t) {
      case "nav": if (!intent.rec) return; touch(intent.rec); onNavigate(intent.rec.title, [ENT[intent.rec.type].plural]); onClose(); return;
      case "navp": onNavigate(intent.page.title, intent.page.trail); onClose(); return;
      case "navl": onNavigate(intent.label, intent.crumb || []); onClose(); return;
      case "tab": onClose(); appToast(`Opened ${intent.label} in a new tab`); return;
      case "copy": setActOpen(false); say(`${intent.what} copied`); return;
      case "toast": setActOpen(false); say(intent.msg); return;
      case "pin": setActOpen(false); setPinned((p) => { const has = p.includes(intent.id); say(has ? "Unpinned" : "Pinned to home"); return has ? p.filter((x) => x !== intent.id) : [...p, intent.id]; }); return;
      case "ask": askAbout(intent.rec); return;
      case "cmd": setActOpen(false); runCommand(intent.cmd); return;
      case "nova": if (selItem && selItem.kind === "ask") startNova(selItem); return;
    }
  };
  const askAbout = (rec) => { setActOpen(false); setSubject(rec); setScope("all"); setQuery(""); setSel(""); setNova(null); setPeek(null); };
  const pick = (it) => {
    if (it.kind === "ask") { startNova(it); return; }
    if (it.kind === "more") { setScope("all"); setSel(""); return; }
    const base = it.key.startsWith("recent:") ? RECORD_BY_KEY[it.key.slice(7)] : it;
    const a = buildActions(base, ctx)[0];
    if (a) run(a.intent);
  };
  const toDesk = () => { onNavigate("Desk", [], nova && nova.q); onClose(); };

  const change = (v) => { setQuery(v); setSel(""); setMIdx(0); setPeek(null); setActOpen(false); };
  const pickMention = (opt) => { setScope(opt.id); setQuery(query.replace(MENTION_RE, "$1").replace(/^>/, "").trimStart()); setMIdx(0); setSel(""); };
  const clearAll = () => { if (actOpen) { setActOpen(false); return true; } if (nova) { setNova(null); return true; } if (subject) { setSubject(null); setQuery(""); return true; } if (query) { change(""); return true; } if (scope !== "all") { setScope("all"); return true; } return false; };

  const move = (d) => { if (!items.length) return; const i = items.findIndex((x) => x.key === selKey); setSel(items[(i + d + items.length) % items.length].key); setPeek(null); };
  const onKey = (e) => {
    const mod = e.metaKey || e.ctrlKey, k = e.key.toLowerCase();
    if (mentionOpen) {
      if (e.key === "ArrowDown" || e.key === "ArrowUp") { e.preventDefault(); setMIdx((i) => (i + (e.key === "ArrowDown" ? 1 : -1) + mentionOpts.length) % mentionOpts.length); return; }
      if (e.key === "Enter" || e.key === "Tab") { e.preventDefault(); pickMention(mentionOpts[Math.min(mIdx, mentionOpts.length - 1)]); return; }
    }
    if (e.key === "Escape") { e.preventDefault(); if (mentionOpen) { setQuery(query.replace(MENTION_RE, "$1")); return; } if (!clearAll()) onClose(); return; }
    if (e.key === "ArrowDown") { e.preventDefault(); move(1); return; }
    if (e.key === "ArrowUp") { e.preventDefault(); move(-1); return; }
    if (e.key === "Tab") { e.preventDefault(); if (!subject) { const i = SCOPE_IDS.indexOf(scope); setScope(SCOPE_IDS[(i + (e.shiftKey ? -1 : 1) + SCOPE_IDS.length) % SCOPE_IDS.length]); setSel(""); } return; }
    if (mod && k === "k") { e.preventDefault(); if (actions.length > 1) { setActQ(""); setActSel(0); setActOpen(true); } return; }
    if (mod && e.key === "Enter") { e.preventDefault(); if (novaHere && nova.phase === "done") toDesk(); else { const a = actions.find((x) => x.id === "tab"); a && run(a.intent); } return; }
    if (e.key === "Enter") { e.preventDefault(); selItem && pick(selItem); return; }
    if (e.key === "ArrowRight" && e.currentTarget.selectionStart === query.length && actions.length > 1) { e.preventDefault(); setActQ(""); setActSel(0); setActOpen(true); return; }
    if (mod && k === "i") { e.preventDefault(); baseItem && baseItem.kind === "record" && askAbout(baseItem); return; }
    if (mod && k === "l") { const a = actions.find((x) => x.id === "link"); if (a) { e.preventDefault(); run(a.intent); } return; }
    if (mod && k === "p") { const a = actions.find((x) => x.id === "pin"); if (a) { e.preventDefault(); run(a.intent); } return; }
    if (e.altKey && e.code === "KeyC") { const a = actions.find((x) => x.id === "id"); if (a) { e.preventDefault(); run(a.intent); } return; }
    if (e.key === "Backspace" && query === "") { if (subject) { e.preventDefault(); setSubject(null); setNova(null); } else if (scope !== "all") { e.preventDefault(); setScope("all"); } }
  };
  const onActKey = (e) => {
    const mod = e.metaKey || e.ctrlKey;
    if (e.key === "ArrowDown" || e.key === "ArrowUp") { e.preventDefault(); setActSel((i) => (i + (e.key === "ArrowDown" ? 1 : -1) + actList.length) % Math.max(1, actList.length)); return; }
    if (e.key === "Enter") { e.preventDefault(); const a = actList[actSel]; a && run(a.intent); return; }
    if (e.key === "Escape" || (mod && e.key.toLowerCase() === "k") || ((e.key === "ArrowLeft" || e.key === "Backspace") && actQ === "")) { e.preventDefault(); setActOpen(false); inputRef.current && inputRef.current.focus(); }
  };

  const placeholder = subject ? `Ask Nova about ${subject.title}…` : sc === "command" ? "Run a command…" : sc === "page" ? "Jump to a page…" : ENT[sc] ? { shipment: "PRO, BOL, customer or lane…", customer: "Name, code or city…", worker: "Name, ID or home terminal…", document: "File name, type or record…" }[sc] : "Search records and pages, run a command, or ask Nova…";
  const scopeObj = SCOPES.find((s) => s.id === scope);

  const hints = actOpen ? [[["↑", "↓"], "Choose"], [["↵"], "Run"], [["Esc"], "Back"]] : [
    [["↑", "↓"], "Navigate"],
    [["↵"], selItem ? (selItem.kind === "ask" ? "Ask" : selItem.kind === "command" ? "Run" : "Open") : "Open"],
    ...(novaHere && nova.phase === "done" ? [[[MOD, "↵"], "Continue in Desk"]] : actions.some((a) => a.id === "tab") ? [[[MOD, "↵"], "New tab"]] : []),
    ...(actions.length > 1 ? [[[MOD, "K"], "Actions"]] : []),
    ...(!subject ? [[["Tab"], "Scope"]] : []),
  ];

  let firstSection = true;
  return (
    <div className="ov" onMouseDown={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="pal" role="dialog" aria-label="Command palette" onMouseDown={(e) => { if (!/INPUT/.test(e.target.tagName)) e.preventDefault(); }}>
        <div className="inp">
          <Ic n="search" s={17}></Ic>
          {subject && <span className={"chip t-" + subject.type}><NovaMark s={12}></NovaMark>{subject.title}<button onClick={() => { setSubject(null); setNova(null); }} aria-label="Clear subject"><Ic n="x" s={11} w={2}></Ic></button></span>}
          {!subject && scope !== "all" && <span className="chip t-neutral" style={{ color: "var(--fg)" }}>{ENT[scope] && <span className="dot" style={{ width: 7, height: 7, borderRadius: 2, background: ENT[scope].dot }}></span>}{scopeObj.label}<button onClick={() => setScope("all")} aria-label="Clear scope"><Ic n="x" s={11} w={2}></Ic></button></span>}
          {!subject && scope === "all" && query.startsWith(">") && <span className="chip t-neutral" style={{ color: "var(--fg)", marginRight: -6 }}>Commands</span>}
          <input ref={inputRef} value={query} onChange={(e) => change(e.target.value)} onKeyDown={onKey} placeholder={placeholder} spellCheck={false} autoComplete="off" aria-label="Search"></input>
          {query && <button className="clr" onClick={() => { change(""); inputRef.current.focus(); }} aria-label="Clear"><Ic n="x" s={13}></Ic></button>}
          <span className="kbd">Esc</span>
          {mentionOpen && <div className="mp">{mentionOpts.map((o, i) => <button key={o.id} className={i === Math.min(mIdx, mentionOpts.length - 1) ? "on" : ""} onClick={() => pickMention(o)}>{ENT[o.id] ? <span className={"tl t-" + o.id}><Ic n={ENT[o.id].icon} s={12}></Ic></span> : <span className="tl t-neutral"><Ic n={o.id === "page" ? "page" : "cmd"} s={12}></Ic></span>}{o.label}<em>@{o.id}</em></button>)}</div>}
        </div>
        {!subject ? <div className="sc" role="tablist">{SCOPES.map((s) => {
          const n = qt ? counts[s.id] || 0 : null;
          return <button key={s.id} role="tab" aria-selected={scope === s.id} className={(scope === s.id ? "on" : "") + (qt && !n && s.id !== "all" ? " zero" : "")} onClick={() => { setScope(s.id); setSel(""); }}>{ENT[s.id] && <span className="dot" style={{ background: ENT[s.id].dot }}></span>}{s.label}{qt && <em>{n}</em>}</button>;
        })}</div> : <div style={{ borderBottom: "1px solid var(--b-sub)" }}></div>}
        <div className="bd">
          <div className="ls" ref={lsRef}>
            <div className="ind" ref={indRef}></div>
            {novaHere && <NovaCard nova={nova} onCite={(r) => run({ t: "nav", rec: r })} onPeek={setPeek} onDesk={toDesk} onCopy={() => say("Answer copied")} onFollow={() => { if (subject) setNova(null); change(""); }}></NovaCard>}
            {empty && <div className="emp"><p>Nothing called “{qt}”{sc !== "all" ? " in " + scopeObj.label.toLowerCase() : ""}</p><span>{sc !== "all" ? "Try everything instead, or ask Nova — it reads records the search index doesn't, like stop notes and email." : "Check the spelling, try a PRO or BOL number, or ask Nova."}</span></div>}
            {visible.map((s) => {
              const f = firstSection; firstSection = false;
              return <div key={s.id} role="group" aria-label={s.head}>
                <div className="sh" style={f && !novaHere && !empty ? { paddingTop: 6, height: 30 } : null}>{s.head}{s.count > 1 && <em>{s.count}</em>}{s.seeAll && <button onClick={() => { setScope(s.seeAll); setSel(""); }}>See all {s.count}<Ic n="chevR" s={11}></Ic></button>}</div>
                {s.items.map((it) => <Row key={it.key} item={it} m={mOf[it.key]} on={it.key === selKey} time={it.time} onPick={pick} onHover={(x) => { if (x.key !== selKey) { setSel(x.key); setPeek(null); } }}></Row>)}
              </div>;
            })}
          </div>
          <PreviewPane item={pvItem} actions={pvActions} run={(intent) => run(intent)} onMore={() => { setActQ(""); setActSel(0); setActOpen(true); }} ctx={ctx} sources={novaHere && nova.phase === "done" ? [...new Set(tokenize(nova.text).filter((x) => x.ref).map((x) => x.ref))].map((k) => RECORD_BY_KEY[k]).filter(Boolean) : null}></PreviewPane>
        </div>
        {actOpen && baseItem && <div className="ac" onMouseDown={(e) => { if (e.target.tagName !== "INPUT") e.preventDefault(); }}>
          <div className="ac-h"><Tile item={baseItem}></Tile><b>{baseItem.title || baseItem.label}</b><span className="sp"></span><Keys k={["Esc"]}></Keys></div>
          <input ref={actRef} value={actQ} onChange={(e) => { setActQ(e.target.value); setActSel(0); }} onKeyDown={onActKey} placeholder="Filter actions…"></input>
          <div className="ac-l">{actList.map((a, i) => <button key={a.id} className={(i === actSel ? "on" : "") + (a.danger ? " dng" : "")} onMouseMove={() => setActSel(i)} onClick={() => run(a.intent)}><AIcon n={a.icon} s={14}></AIcon>{a.label}<span className="sp"></span>{a.keys && <Keys k={a.keys}></Keys>}</button>)}{!actList.length && <div style={{ padding: "10px 8px", fontSize: 12.5, color: "var(--subtle)" }}>No matching actions</div>}</div>
        </div>}
        <div className="ft">
          {toast ? <span className="tst"><Ic n="check" s={14} w={2}></Ic>{toast}</span> : hints.map(([k, l], i) => <span key={i} className={i > 2 ? "hide-s" : ""}><Keys k={k}></Keys>{l}</span>)}
          <span className="r hide-s">{subject ? <><Keys k={["⌫"]}></Keys>Stop asking about {subject.title}</> : question ? <><NovaMark s={12}></NovaMark>Nova will answer this</> : <><NovaMark s={12}></NovaMark>End with <span className="kbd">?</span> to ask Nova</>}</span>
        </div>
      </div>
    </div>
  );
}
window.Palette = Palette;
