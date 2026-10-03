const { useState, useRef, useEffect, useCallback } = React;

TURNS.tx = {
  dur: "2s", time: "10:42 PM",
  trace: [{ pose: "think", live: "Thinking it through", done: "Considered the request", k: "reason", v: "context", r: "ok", t: "1.2s", ms: 1200 }],
  text: [["Noted. I'll pick that up as soon as the open decision is settled — nothing else is waiting on me right now."]],
};
TURNS.h1 = { dur: "4s", time: "9:58 PM", trace: [{ pose: "lookup", live: "Reading today's loads", done: "Read today's loads", k: "shipments.list", v: "today", r: "42 rows", t: "1.4s", ms: 1400, art: "board" }], text: [["Morning. ", { b: "42 loads" }, " are on the board for today", { r: 1, t: "" }, " — 37 are on schedule, 3 are running late out of the Midwest, and 2 still need a driver."], ["The late ones are all behind the same weather system around Des Moines. I'd expect them to slip about 2–3 hours, not miss their appointments."]] };
TURNS.h2 = { dur: "3s", time: "10:04 PM", trace: [{ pose: "lookup", live: "Checking driver hours", done: "Checked driver hours", k: "drivers.hos", v: "available", r: "6 drivers", t: "1.1s", ms: 1100, art: "shp" }], text: [["Two drivers can take the open loads without going over their hours: ", { b: "Marcus Hill" }, " and ", { b: "Dana Ortiz" }, { r: 1, t: "" }, ". Marcus is 18 miles from the Joliet pickup, Dana is already in Gary."], ["Want me to draft the assignments so you can approve them together?"]] };
TURNS.h3 = { dur: "2s", time: "10:11 PM", trace: [{ pose: "change", live: "Assigning drivers", done: "Assigned drivers", k: "shipments.assign", v: "2 loads", r: "2 updated", t: "0.9s", ms: 900, art: "plan" }], text: [["Done — both loads are assigned", { r: 1, t: "" }, " and the drivers have been notified. I'll keep an eye on the Des Moines delays and tell you if any appointment is at risk."]] };
const HISTORY = () => [
  { type: "u", id: "h1u", text: "Give me a quick rundown of today's board" },
  { type: "a", id: "h1a", turn: "h1", phase: "done", step: 1, open: false },
  { type: "u", id: "h2u", text: "Who can cover the two loads without a driver?" },
  { type: "a", id: "h2a", turn: "h2", phase: "done", step: 1, open: false },
  { type: "u", id: "h3u", text: "Yes, assign Marcus to Joliet and Dana to Gary" },
  { type: "a", id: "h3a", turn: "h3", phase: "done", step: 1, open: false },
  { type: "u", id: "u1", text: "Check and see what shipments are eligible to be transferred" },
  { type: "a", id: "a1", turn: "t1", phase: "done", step: 2, open: false },
  { type: "u", id: "u2", text: "Can you tell me what shipments are in the billing queue then?" },
  { type: "a", id: "a2", turn: "t2", phase: "done", step: 2, open: false },
  { type: "u", id: "u2m", text: "Break this week's billing down by customer" },
  { type: "a", id: "a2m", turn: "md", phase: "done", step: 2, open: false },
  { type: "u", id: "u2f", text: "How is Acme's fuel surcharge worked out?" },
  { type: "a", id: "a2f", turn: "mdf", phase: "done", step: 1, open: false },
  { type: "u", id: "u2b", text: "What's holding up BQ-24101?" },
  { type: "a", id: "a2b", turn: "bq", phase: "done", step: 2, open: false },
];
const ACT0 = [
  { t: "10:30 PM", x: "Searched ready-to-invoice shipments · 0 rows" },
  { t: "10:30 PM", x: "Searched completed shipments · 0 rows" },
  { t: "10:32 PM", x: "Read the billing queue · 15 rows" },
];

const LOCKS = {
  daily: ["lock", "Billing Specialist has reached today's 200-request limit.", "It resets at midnight.", "Ask General Assistant"],
  budget: ["alert", "October's budget for Billing Specialist is used up.", "It resets Nov 1, or an admin can raise it.", "Ask an admin"],
  usage: ["lock", "You've used your AI allowance for this period.", "It refreshes in 4 days · 1,000 of 1,000 messages.", "Request more"],
  disabled: ["lock", "Jordan Pike turned off Billing Specialist on Sep 30.", "You can still read this conversation.", "Start with another agent"],
  noaccess: ["lock", "You no longer have access to Billing Specialist.", "Your role changed on Oct 1.", "Request access"],
  nomodel: ["info", "No AI model is set up for your organization yet.", "An admin can connect one in Agent Control.", "Open Agent Control"],
};
const SCENARIOS = [
  ["none", "Happy path"], ["refused", "Question not accepted"], ["retry", "Retry, then answer"], ["fallback", "Answered by a fallback model"],
  ["cut", "Stopped partway"], ["allfail", "No model could answer"], ["tools", "Tool steps fail"], ["writefail", "Change didn't go through"],
  ["wouldfail", "Approval would be refused"], ["stale", "Proposal out of date"], ["ratelimit", "Sending too fast"], ["context", "Conversation too long"],
  ["budgetnear", "Budget almost used"], ["budget", "Budget used up"], ["daily", "Agent's daily limit"], ["usage", "Your usage limit"],
  ["disabled", "Agent turned off"], ["noaccess", "No access to agent"], ["nomodel", "No model set up"], ["offline", "You're offline"],
];
const SCEN_GROUPS = [
  ["Everything works", ["none"]],
  ["Didn't get an answer", ["refused", "retry", "fallback", "cut", "allfail"]],
  ["A step or change failed", ["tools", "writefail", "wouldfail", "stale"]],
  ["Hit a limit", ["ratelimit", "context", "budgetnear", "budget", "daily", "usage"]],
  ["Can't use this agent", ["disabled", "noaccess", "nomodel"]],
  ["Connection", ["offline"]],
];
TURNS.doc = { dur: "6s", time: "10:44 PM", trace: [
  { pose: "lookup", live: "Reading rate-con-acme-77812.pdf · page 1 of 3", done: "Read the document", k: "documents.read", v: "3 pages", r: "3 pages", t: "1.9s", ms: 1900 },
  { pose: "think", live: "Classifying the document", done: "Classified it as a rate confirmation", k: "documents.classify", v: "type", r: "98%", t: "0.8s", ms: 900 },
  { pose: "lookup", live: "Pulling out the fields", done: "Extracted 12 fields", k: "documents.extract", v: "rate confirmation", r: "12 fields", t: "1.6s", ms: 1700, art: "extract" },
],
  text: [["That's a ", { b: "rate confirmation" }, " from Acme for PO 77812, Chicago to Columbus on Oct 6, ", { b: "$1,359.56" }, " all in", { r: 3, t: "" }, ". Two fields need a look: the delivery time is smudged and the detention terms are handwritten. Want me to draft the shipment once you've checked them?"]] };
TURNS.md = { dur: "5s", time: "10:34 PM", trace: [
  { pose: "lookup", live: "Reading invoices by customer", done: "Read invoices by customer", k: "invoices.by_customer", v: "this week", r: "6 rows", t: "1.6s", ms: 1500 },
  { pose: "think", live: "Comparing to last week", done: "Compared to last week", k: "compare", v: "wk 39 vs 40", r: "6 rows", t: "0.7s", ms: 900 }],
  text: ["## Billing by customer\nThis week's queue totals **$" + money(TOTAL) + "** across six customers, up *12%* on last week. Acme is the only account carrying an unbilled `TONU`.\n\n| Customer | Items | Amount | vs last wk |\n|---|---|---|---|\n| Acme Manufacturing | 3 | $9,835.50 | +18% |\n| Peak Distributing | 3 | $8,685.00 | +4% |\n| Northline Foods | 2 | $7,420.00 | — |\n| Harbor Supply Co. | 3 | $8,210.00 | +22% |\n| Granite Building | 2 | $5,890.00 | -6% |\n| Bluewater Retail | 2 | $6,175.00 | +9% |\n\n### Before posting\n1. Confirm the **Acme TONU** with dispatch — it's not on the rate con.\n2. Harbor's jump is two late PODs from week 39, not new volume.\n   - `SEED-SHP-0121` delivered Sep 28\n   - `SEED-SHP-0130` delivered Sep 29\n3. Everything else matches the rate confirmations.\n\n- [x] Billers assigned\n- [x] Rates matched\n- [ ] Acme TONU confirmed\n\n> Posting can't be undone, so I'll hold until the TONU is settled.\n\nTo pull the same view yourself:\n\n```sql\nselect customer, count(*) as items, sum(amount)\nfrom billing_queue\nwhere queued_at >= date_trunc('week', now())\ngroup by customer\norder by sum(amount) desc;\n```"] };
TURNS.mdf = { dur: "3s", time: "10:35 PM", trace: [
  { pose: "lookup", live: "Reading Acme's contract", done: "Read Acme's contract", k: "contracts.read", v: "Acme Manufacturing", r: "1 doc", t: "1.3s", ms: 1300 }],
  text: ["Fuel surcharge\n==============\nAcme's contract uses a *mileage-based* surcharge tied to the weekly [DOE diesel average][doe], with a base of $p_0 = \\$1.25$ per gallon and an assumed $6.0$ mpg.\n\n$$\nFSC = \\max\\left(0,\\ \\frac{p - p_0}{6.0}\\right) \\times d\n$$\n\nFor PO 77812 this week, with $p = \\$3.79$ and $d = 357$ miles:\n\n$$\nFSC = \\frac{3.79 - 1.25}{6.0} \\times 357 = \\$151.13\n$$\n\nThat matches the rate confirmation to the cent. _Section 4.2_ of the contract also caps it at **18% of linehaul**, which this load doesn't reach.\n\nRates are pulled every Monday from:\nEIA · Weekly Retail On-Highway Diesel\nU.S. average, all types  \nSee the [contract terms][acme] for the full schedule.\n\n[doe]: https://www.eia.gov/petroleum/gasdiesel/ \"EIA diesel prices\"\n[acme]: https://example.com/contracts/acme-2026"] };
TURNS.bq = { dur: "3s", time: "10:38 PM", trace: [
  { pose: "lookup", live: "Opening BQ-24101", done: "Opened BQ-24101", k: "billing.queue.get", v: "BQ-24101", r: "1 item", t: "0.9s", ms: 1000, art: "bqi" },
  { pose: "think", live: "Checking it against the rate con", done: "Checked it against the rate con", k: "rating.compare", v: "Acme 77544", r: "1 mismatch", t: "1.1s", ms: 1100 }],
  text: [["Two things: nobody is assigned as ", { b: "biller" }, ", and there's a ", { b: "$110.00 lumper fee" }, " that isn't on Acme's rate con", { r: 2, t: "" }, ". Acme's contract passes lumpers through with a receipt, and the driver uploaded one, so I'd keep it. Everything else checks out. You can settle both from the item", { a: "bqi" }, "."]] };
TURNS.ex = { dur: "4s", time: "10:42 PM", trace: [{ pose: "lookup", live: "Reading aging buckets", done: "Read aging buckets", k: "invoices.aging", v: "over 60 days", r: "3 rows", t: "1.2s", ms: 1200 }],
  text: [["Three customers are past 60 days: ", { b: "Acme Manufacturing" }, " ($9,800), ", { b: "Harbor Supply Co." }, " ($5,600) and ", { b: "Lakeside Grocers" }, " ($1,850). Acme's balance is two invoices from the Columbus plant, both disputed over detention."]] };
TURNS.exTools = { dur: "5s", time: "10:42 PM", trace: [{ pose: "lookup", live: "Reading the billing queue", done: "Read the billing queue", k: "billing.queue.list", v: "ready", r: "14 rows", t: "1.1s", ms: 1100 }],
  text: [["I found 14 invoices ready to post, but couldn't check credit holds or payment history, so I haven't drafted the posting yet. The details are below."]] };

function DeskApp({ mode, theme: tweakTheme, startView, onReplay, scenario = "none", pageKey = "shipment", noChats = false, noArts = false, memAsk = false }) {
  const hist = noArts ? [] : ART_HISTORY;
  const [cfg, setCfg] = useDeskSettings();
  const [pageSel, setPageSel] = useState(PAGES[pageKey] && cfg.sharePage !== "off" ? pageKey : null);
  const cfgRef = useRef(cfg); cfgRef.current = cfg; window.DESK_CFG = cfg;
  const page = pageSel ? PAGES[pageSel] : null;
  const [settingsOpen, setSettingsOpen] = useState(false);
  const sysDark = typeof matchMedia !== "undefined" && matchMedia("(prefers-color-scheme: dark)").matches;
  const theme = cfg.theme === "system" ? (tweakTheme || (sysDark ? "dark" : "light")) : cfg.theme;
  const [view, setView] = useState(noChats ? "home" : cfg.start === "last" ? "thread" : startView);
  const [items, setItems] = useState(HISTORY);
  const [arts, setArts] = useState(noArts ? [] : ["board", "shp", "plan", "rate", "email", "report", "diff", "doc", "view", "queue", "bqi"]);
  const [newest, setNewest] = useState(null);
  const [sheet, setSheetS] = useState({ open: false, tab: "data", art: "queue" });
  const [input, setInput] = useState("Perfect, let's go ahead and post all of these");
  const [working, setWorking] = useState(false);
  const [status, setStatus] = useState("");
  const [elapsed, setElapsed] = useState(0);
  const [pending, setPending] = useState([]);
  const [resolved, setResolved] = useState([]);
  const [confirming, setConfirming] = useState(null);
  const [assigned, setAssigned] = useState(false);
  const [posted, setPosted] = useState(false);
  const [flash, setFlash] = useState(0);
  const [activity, setActivity] = useState(ACT0);
  const [hot, setHot] = useState(null);
  const [newArt, setNewArt] = useState(false);
  const [ctxParts, setCtxParts] = useState(noChats ? CTX_FRESH : CTX0);
  const [autoCmp, setAutoCmp] = useState(true);
  const [compacting, setCompacting] = useState(false);
  const cmpTimer = useRef(null);
  useEffect(() => () => clearTimeout(cmpTimer.current), []);
  const [toast, setToast] = useState(null);
  const [stage, setStage] = useState(0);
  const [reading, setReading] = useState(null);
  const [undoing, setUndoing] = useState(null);
  const [facts, setFacts] = useState(["PO 77812", "Acme contract 2026", "Billing week Sep 28 – Oct 4"]);
  const up = useUploads();
  const [drag, setDrag] = useState({ on: false, hot: false, count: 0 });
  useEffect(() => {
    let depth = 0;
    const isFiles = e => e.dataTransfer && [...e.dataTransfer.types].includes("Files");
    const enter = e => { if (!isFiles(e)) return; e.preventDefault(); depth++; setDrag(d => ({ ...d, on: true, count: e.dataTransfer.items ? e.dataTransfer.items.length : 1 })); };
    const over = e => { if (!isFiles(e)) return; e.preventDefault(); e.dataTransfer.dropEffect = "copy"; const hot = !!(e.target.closest && e.target.closest(".cmp")); setDrag(d => d.hot === hot && d.on ? d : { ...d, on: true, hot }); };
    const leave = e => { if (!isFiles(e)) return; depth = Math.max(0, depth - 1); if (!depth) setDrag({ on: false, hot: false, count: 0 }); };
    const drop = e => { if (!isFiles(e)) return; e.preventDefault(); depth = 0; setDrag({ on: false, hot: false, count: 0 }); if (cfgRef.current.drop === "composer" && !(e.target.closest && e.target.closest(".cmp"))) return; if (e.dataTransfer.files.length) up.add(e.dataTransfer.files); };
    window.addEventListener("dragenter", enter); window.addEventListener("dragover", over); window.addEventListener("dragleave", leave); window.addEventListener("drop", drop);
    return () => { window.removeEventListener("dragenter", enter); window.removeEventListener("dragover", over); window.removeEventListener("dragleave", leave); window.removeEventListener("drop", drop); };
  }, []);
  useEffect(() => {
    const k = e => { if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "u") { e.preventDefault(); up.add(SAMPLES.slice(0, 1)); } };
    const demo = e => {
      const kind = e.detail; setView(v => v); up.clear();
      if (kind === "sample") up.add(SAMPLES.slice(0, 1));
      if (kind === "fail") up.add([{ ...SAMPLES[1], name: "BOL-778.pdf", failOnce: true }, SAMPLES[0]]);
      if (kind === "large") up.add([{ name: "scan-batch-sept.pdf", size: 61000000 }]);
      if (kind === "type") up.add([{ name: "carrier-packet.zip", size: 8000000 }]);
      if (kind === "locked") up.add([{ name: "locked-invoice.pdf", size: 320000 }]);
      if (kind === "many") up.add([...SAMPLES, { name: "pod-2.jpg", size: 1800000 }, { name: "lumper-receipt.pdf", size: 220000 }, { name: "extra.pdf", size: 120000 }]);
      if (kind === "messy") up.add([...SAMPLES.slice(1), { ...SAMPLES[0], name: "invoice-sept.pdf", failOnce: true }, ...BAD_SAMPLES]);
    };
    window.addEventListener("keydown", k); window.addEventListener("desk:upload-demo", demo);
    return () => { window.removeEventListener("keydown", k); window.removeEventListener("desk:upload-demo", demo); };
  }, []);
  const [cphase, setCphase] = useState(null);
  const [lock, setLock] = useState(LOCKS[scenario] || null);
  const [cool, setCool] = useState(0);
  const [dockCard, setDockCard] = useState(scenario === "budgetnear" ? "meter" : null);
  const [offline] = useState(scenario === "offline");
  const scenUsed = useRef(false);
  const [burst, setBurst] = useState(null);
  const [searching, setSearching] = useState(false);
  useEffect(() => {
    const k = e => { if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "j") { e.preventDefault(); setView("thread"); setSheetS(s => s.open && s.browse ? { ...s, open: false, browse: false } : { ...s, open: true, browse: true }); setNewArt(false); } };
    window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k);
  }, []);
  useEffect(() => {
    const k = e => { if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") { e.preventDefault(); setSearching(v => !v); } };
    window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k);
  }, []);
  const timers = useRef([]);
  const scroller = useRef(null);
  const T = (fn, ms) => { const h = setTimeout(fn, ms); timers.current.push(h); return h; };
  useEffect(() => () => timers.current.forEach(h => { clearTimeout(h); clearInterval(h); }), []);

  const setSheet = p => setSheetS(s => ({ ...s, ...p }));
  const patch = (id, p) => setItems(xs => xs.map(x => x.id === id ? { ...x, ...(typeof p === "function" ? p(x) : p) } : x));
  const log = (x, c) => setActivity(a => [...a, { t: "10:" + (36 + Math.min(a.length - 3, 9)) + " PM", x, c }]);
  const openArt = useCallback(art => { setSheetS(s => ({ ...s, open: true, tab: "data", art })); setNewArt(false); }, []);

  const stick = useRef(true);
  const [away, setAway] = useState(false);
  const [unread, setUnread] = useState(0);
  const seenLen = useRef(0);
  React.useLayoutEffect(() => {
    const el = scroller.current; if (!el) return;
    const pin = () => { if (stick.current) el.scrollTop = el.scrollHeight; };
    const onScroll = () => {
      const d = el.scrollHeight - el.scrollTop - el.clientHeight;
      if (d < 40) { stick.current = true; setUnread(0); }
      setAway(d > 160);
    };
    const up = () => { stick.current = false; };
    const onWheel = e => { if (e.deltaY < 0) up(); };
    const onKey = e => { if (["ArrowUp", "PageUp", "Home"].includes(e.key)) up(); };
    el.addEventListener("scroll", onScroll);
    el.addEventListener("wheel", onWheel, { passive: true });
    el.addEventListener("touchmove", up, { passive: true });
    el.addEventListener("keydown", onKey);
    const ro = new ResizeObserver(pin); ro.observe(el); if (el.firstChild) ro.observe(el.firstChild);
    pin();
    return () => { ro.disconnect(); el.removeEventListener("scroll", onScroll); el.removeEventListener("wheel", onWheel); el.removeEventListener("touchmove", up); el.removeEventListener("keydown", onKey); };
  }, [view]);
  React.useLayoutEffect(() => {
    const added = items.length - seenLen.current; seenLen.current = items.length;
    const last = items[items.length - 1];
    if (last && last.type === "u" && added > 0) stick.current = true;
    else if (!stick.current && added > 0) setUnread(u => u + items.slice(-added).filter(x => x.type === "a").length);
  }, [items.length]);
  const dockRef = useRef(null);
  React.useLayoutEffect(() => {
    const d = dockRef.current; if (!d) return;
    const set = () => d.parentElement && d.parentElement.style.setProperty("--dock-h", d.offsetHeight + "px");
    const ro = new ResizeObserver(set); ro.observe(d); set();
    return () => ro.disconnect();
  }, [view]);
  const jumpLatest = () => { const el = scroller.current; if (!el) return; stick.current = true; setUnread(0); el.scrollTo({ top: el.scrollHeight, behavior: "smooth" }); };
  React.useLayoutEffect(() => { const el = scroller.current; if (el && stick.current) el.scrollTop = el.scrollHeight; });

  const runTurn = (key, after, opts = {}) => {
    const def = TURNS[key];
    const id = "a" + Math.random().toString(36).slice(2, 8);
    setCphase(null);
    setItems(xs => [...xs, { type: "a", id, turn: key, phase: "work", step: 0, shown: 0, open: false, fb: opts.fb, fails: opts.fails }]);
    setWorking(true); setElapsed(0); setStatus({ t: def.trace[0].live + "…", p: def.trace[0].pose });
    const start = performance.now();
    const tick = setInterval(() => setElapsed((performance.now() - start) / 1000), 100);
    timers.current.push(tick);
    let at = 500;
    def.trace.forEach((s, i) => {
      at += s.ms;
      T(() => {
        patch(id, { step: i + 1 });
        log(s.done + " · " + s.r, s.pose === "change" ? "w" : "");
        setStatus(def.trace[i + 1] ? { t: def.trace[i + 1].live + "…", p: def.trace[i + 1].pose } : { t: "Writing the answer…", p: "write" });
        if (s.art) { setArts(a => { if (a.includes(s.art)) return a; setNewArt(cfgRef.current.artNotify !== "off"); setNewest(s.art); return [...a, s.art]; }); if (cfgRef.current.autoOpen === "on") T(() => openArt(s.art), 60); }
      }, at);
    });
    const total = opts.cutAt || tokenize(def.text).length;
    T(() => {
      patch(id, { phase: "write", step: def.trace.length });
      const t0 = performance.now();
      const iv = setInterval(() => {
        const n = Math.min(total, Math.floor((performance.now() - t0) / 30) + 1);
        patch(id, { shown: n });
        if (n >= total) {
          clearInterval(iv);
          if (opts.cutAt) { patch(id, { phase: "done", cut: true, shown: opts.cutAt }); setWorking(false); setStatus(""); clearInterval(tick); after && after(id); return; }
          patch(id, { phase: "settle" });
          T(() => { patch(id, { phase: "done" }); setWorking(false); clearInterval(tick); setCtxParts(p => ({ ...p, conv: p.conv + 3200, tools: p.tools + def.trace.length * 2400 })); after && after(); }, 900);
        }
      }, 32);
      timers.current.push(iv);
    }, at + 300);
  };

  const GUARD = { ms: 900, st: { t: "Checking the question…", p: "check" }, ph: "check" };
  const retrySt = (prov, n, secs) => ({ ms: secs * 1000, st: { t: prov + " is overloaded · retrying", p: "retry", secs, extra: "Attempt " + n + " of 3" }, ph: "retry" });
  const playStatus = (seq, done) => {
    setWorking(true); let at = 0;
    seq.forEach(s => { T(() => { setStatus(s.st); setCphase(s.ph || null); }, at); at += s.ms; });
    T(done, at);
  };
  const endWork = () => { setWorking(false); setStatus(""); setCphase(null); };
  const addErr = (kind, extra = {}) => setItems(xs => [...xs, { type: "err", id: "e" + Date.now() + kind, kind, ...extra }]);
  const retryLast = id => { setItems(xs => xs.filter(x => x.id !== id)); playStatus([GUARD], () => runTurn("ex")); };

  const runScenario = (text) => {
    const sc = scenario;
    if (sc === "refused") return playStatus([GUARD], () => { endWork(); setItems(xs => xs.map((x, i) => i === xs.length - 1 && x.type === "u" ? { ...x, refused: true } : x)); addErr("refused", { text }); });
    if (sc === "retry") return playStatus([GUARD, retrySt("Anthropic", 2, 3)], () => runTurn("ex"));
    if (sc === "fallback") return playStatus([GUARD, retrySt("Anthropic", 2, 2), retrySt("Anthropic", 3, 2)], () => runTurn("ex", null, { fb: true }));
    if (sc === "cut") return playStatus([GUARD], () => runTurn("ex", () => addErr("cut"), { cutAt: 22 }));
    if (sc === "allfail") return playStatus([GUARD, retrySt("Anthropic", 2, 2), { ms: 2200, st: { t: "Trying GPT-5…", p: "think" } }, { ms: 1800, st: { t: "Trying Gemini 2.5 Pro…", p: "think" } }], () => { endWork(); addErr("allfail"); });
    if (sc === "tools") return playStatus([GUARD], () => runTurn("exTools", null, { fails: true }));
    if (sc === "writefail" || sc === "wouldfail" || sc === "stale") return false;
    if (sc === "context") return playStatus([GUARD], () => { endWork(); setDockCard("context"); });
    if (sc === "offline") { setItems(xs => xs.map((x, i) => i === xs.length - 1 ? { ...x, queued: true } : x)); return true; }
    return false;
  };

  const compact = auto => {
    if (working || compacting) return;
    const id = "cmp" + Date.now(), before = ctxTotal(ctxParts), after = ctxParts.sys + 7400 + 9100 + Math.min(ctxParts.files, 4000);
    setCompacting({ auto, before, after });
    cmpTimer.current = setTimeout(() => {
      stick.current = true;
      setItems(xs => [...xs, { type: "cmp", id, auto, before, after, turns: xs.filter(x => x.type === "u" || x.type === "a").length }]);
      setCtxParts(p => ({ sys: p.sys, conv: 7400, tools: 9100, files: Math.min(p.files, 4000) }));
      setCompacting(false);
      log((auto ? "Auto-compacted" : "Compacted") + " the conversation", "");
    }, 2600);
  };
  const cancelCompact = () => { clearTimeout(cmpTimer.current); setCompacting(false); if (compacting && compacting.auto) setAutoCmp(false); };
  useEffect(() => { if (autoCmp && !working && !compacting && ctxTotal(ctxParts) >= CTX_MAX * CTX_AUTO_AT) T(() => compact(true), 600); }, [ctxParts, working]);
  useEffect(() => {
    const h = e => { const key = e.detail; if (!MEM_DEMOS[key]) return; setView("thread"); stick.current = true; setItems(xs => [...xs, { type: "u", id: "u" + Date.now(), text: MEM_DEMOS[key] }]); T(() => playStatus([GUARD], () => runTurn(key)), 350); };
    window.addEventListener("desk:mem", h); return () => window.removeEventListener("desk:mem", h);
  });
  const send = (override) => {
    const ready = up.atts.filter(x => x.status === "ready");
    const text = (typeof override === "string" ? override : input).trim() || (ready.length ? "What's in " + (ready.length > 1 ? "these?" : "this?") : "");
    if (text === "/compact") { setInput(""); compact(false); return; }
    if (!text || compacting || working || lock || cool || up.atts.some(x => x.status === "uploading" || x.status === "scanning")) return;
    if (ready.length) {
      window.dispatchEvent(new Event("desk:sent"));
      setView("thread");
      setItems(xs => [...xs, { type: "u", id: "u" + Date.now(), text, atts: ready, page: page || null }]);
      setInput(""); up.clear();
      T(() => playStatus([GUARD], () => runTurn("doc")), 300);
      return;
    }
    if (scenario === "ratelimit" && !scenUsed.current) { scenUsed.current = true; setCool(18); T(() => setCool(0), 18000); return; }
    window.dispatchEvent(new Event("desk:sent"));
    setView("thread");
    setItems(xs => [...xs, { type: "u", id: "u" + Date.now(), text, page: page || null }]);
    setInput("");
    if (/^(\/schedule\s+)?(every|each)\s/i.test(text)) { const s = schedAdd(text); T(() => setItems(xs => [...xs, { type: "sch", id: "sch" + Date.now(), sid: s.id }]), 500); log("Scheduled · " + s.when, ""); return; }
    if (scenario !== "none" && !scenUsed.current && !["writefail", "wouldfail", "stale"].includes(scenario)) { scenUsed.current = true; if (runScenario(text) !== false) return; }
    if (/\b(BQ-\d+|queue item|holding)\b/i.test(text)) { T(() => playStatus([GUARD], () => runTurn("bq")), 350); return; }
    if (/\b(surcharge|formula|math|calculat\w*)\b/i.test(text)) { T(() => playStatus([GUARD], () => runTurn("mdf")), 350); return; }
    if (/\b(table|compare|breakdown|markdown|summar\w*|by customer)\b/i.test(text)) { T(() => playStatus([GUARD], () => runTurn("md")), 350); return; }
    if (stage === 0) { setStage(1); T(() => playStatus([GUARD], () => runTurn("t3", () => { if (scenario === "wouldfail") { setDockCard("would"); return; } if (scenario === "stale") { setDockCard("stale"); return; } setPending(["d1"]); setArts(x => x.includes("dec") ? x : [...x, "dec"]); setNewest("dec"); log("Drafted Assign biller · 11 items · waiting on you", "w"); })), 350); }
    else T(() => playStatus([GUARD], () => runTurn("tx")), 350);
  };
  const approve = id => {
    if (confirming || undoing) return;
    setPending(p => p.filter(x => x !== id));
    setUndoing({ id, n: 5 });
  };
  const commit = id => {
    setConfirming(id);
    const b = Date.now(); if (cfg.celebrate === "confetti" && cfg.motion !== "reduced") setBurst(b); T(() => setBurst(x => x === b ? null : x), 5400);
    T(() => {
      setConfirming(null);
      setResolved(r => [[id, "approved"], ...r]);
      const d = DECISIONS[id];
      setItems(xs => [...xs, { type: "ev", id: "e" + id, text: "You approved " + d.title.toLowerCase(), sub: d.scope }]);
      log("You approved " + d.title + " · " + d.scope, "g");
      setFlash(f => f + 1);
      if (id === "d1") { setAssigned(true); runTurn("t4", () => { setPending(["d2"]); setArts(x => x.includes("dec2") ? x : [...x, "dec2"]); setNewest("dec2"); log("Drafted Post invoices · 15 drafts · waiting on you", "w"); }); }
      else if (scenario === "writefail") { addErr("writefail"); }
      else { setPosted(true); setItems(xs => [...xs, { type: "ev", id: "eposted", text: "Posted 15 invoices", sub: "$" + money(TOTAL) + " across 6 customers" }]); log("Posted 15 invoices", "g"); }
    }, 1100);
  };
  const dismiss = id => {
    setPending(p => p.filter(x => x !== id));
    setResolved(r => [[id, "dismissed"], ...r]);
    setItems(xs => [...xs, { type: "ev", id: "x" + id, text: "You set aside " + DECISIONS[id].title.toLowerCase(), sub: "it stays in Changes" }]);
  };
  useEffect(() => { if (!undoing) return; if (undoing.n <= 0) { const id = undoing.id; setUndoing(null); commit(id); return; } const h = setTimeout(() => setUndoing(u => u && { ...u, n: u.n - 1 }), 1000); return () => clearTimeout(h); }, [undoing]);
  const undoApprove = () => { const id = undoing.id; setUndoing(null); setPending(p => [id, ...p]); log("Undid approving " + DECISIONS[id].title, ""); };
  const approveNow = () => { const id = undoing.id; setUndoing(null); commit(id); };
  const handoff = ag => { setView("thread"); stick.current = true; setItems(xs => [...xs, { type: "ho", id: "ho" + Date.now(), agent: ag, facts, time: "Just now" }]); log("Handed off to " + ag.name, ""); };
  const openQueueItem = id => { setView("thread"); if (id) bqOpen(id); setSheet({ open: true, art: id ? "bqi" : "queue", browse: false }); };
  const copy = () => { setToast("Link copied — this artifact opens on its own page"); T(() => setToast(null), 1900); };
  const onKey = e => {
    if ((e.metaKey || e.ctrlKey) && e.key === "Enter" && pending[0]) { e.preventDefault(); approve(pending[0]); }
    if (e.key === "Escape" && sheet.open) setSheet({ open: false });
  };

  const openBulk = id => { setArts(x => x.includes(id) ? x : [...x, id]); setNewest(id); openArt(id); };
  const errView = it => {
    if (it.kind === "refused") return <ECard tone="neutral" icon="shield" title="This is outside what Billing Specialist can do" sub="Pay rates and people records go to the Workforce Coordinator. Nothing was looked up or changed."
      actions={<><Btn k="ink">Ask Workforce Coordinator</Btn><button className="ec-btn" onClick={() => setInput(it.text)}>Rephrase</button></>} />;
    if (it.kind === "cut") return <ECard tone="err" compact title="Reply stopped partway" sub="The connection to Anthropic dropped after 22 words. Nothing was changed."
      actions={<><button className="ec-btn ink" onClick={() => retryLast(it.id)}><Ic n="replay" s={12} />Try again</button><Btn>Continue from here</Btn><Btn>Copy what arrived</Btn></>} />;
    if (it.kind === "allfail") return <ECard tone="err" title="No model could answer" sub="Desk tried every model your organization set up. Your message is saved.">
      <div className="ec-provs">{PROV.map(([p, m, s, d]) => <div key={p}><BrandMark id={p} s={13} /><b>{m}</b><span className="ec-pst">{s}</span><em>{d}</em></div>)}</div>
      <div className="ec-acts"><button className="ec-btn ink" onClick={() => retryLast(it.id)}><Ic n="replay" s={12} />Try again</button><Btn>Check provider status</Btn></div>
    </ECard>;
    if (it.kind === "writefail") return <ECard tone="err" title="Posted 73 of 120 invoices · 47 didn't go through" sub="The 47 stayed as drafts. The 73 that posted are final.">
      <BulkList items={FAILED_POST} noun="invoices" onOpenTable={() => openBulk("failpost")} />
      <div className="ec-acts"><Btn k="ink">Ask agent to fix these 47</Btn><Btn>Open in billing</Btn></div>
    </ECard>;
    return null;
  };

  const renderItem = (it, i) => {
    const first = i === 0 ? " first" : "";
    if (it.type === "u") return <div className={"r" + first} key={it.id}><div className="g"></div><div className="c"><div className={"q" + (it.text.length > 90 ? " long" : it.text.length > 48 ? " mid" : "") + (it.refused ? " ec-q muted" : "")}><MentionText text={it.text} />{it.refused && <span className="ec-qtag">Not sent to the agent</span>}</div><MsgAtts atts={it.atts} /><PageSent page={it.page} />{it.queued && <div className="ec-queued" style={{ marginTop: 8 }}><Ic n="undo" s={11} w={2.2} />Waiting to send</div>}</div><div className="m"></div></div>;
    if (it.type === "ev") return null;
    if (it.type === "ho") return <div className={"r ev" + first} key={it.id}><div className="g"></div><div className="c"><HandoffCard it={it} /></div><div className="m"></div></div>;
    if (it.type === "sch") return <div className={"r ev" + first} key={it.id}><div className="g"></div><div className="c"><SchedCard sid={it.sid} /></div><div className="m"></div></div>;
    if (it.type === "cmp") return <div className={"r ev cmpd" + (it.live ? " live" : "") + first} key={it.id}><div className="g"></div><div className="c"><CompactMark it={it} /></div><div className="m"></div></div>;
    if (it.type === "err") return <div className={"r ev" + first} key={it.id}><div className="g"></div><div className="c ec-in">{errView(it)}</div><div className="m"></div></div>;
    if (it.type === "evx") return <div className={"r ev" + first} key={it.id}><div className="g"></div><div className="c"><div className="evl"><span className="tk"><Ic n="check" s={10} w={2.6} /></span><span><b>{it.text}</b> · {it.sub}</span></div></div><div className="m"></div></div>;
    const def = TURNS[it.turn];
    const ph = norm(def, it).phase;
    const showText = ph === "write" || ph === "settle" || ph === "done";
    if (!showText) return null;
    const made = [...new Set(def.trace.slice(0, it.step).map(s => s.art).filter(Boolean))];
    const onRef = r => { const s = def.trace[r - 1]; if (s && s.art) openArt(s.art); else if (def.art) openArt(def.art); };
    const elapsedFor = it.phase === "done" ? 0 : elapsed;
    return (
      <div className={"r a" + first} key={it.id}>
        <div className="g"><span className="time">{def.time}</span></div>
        <div className="c">
          {mode === "narrated" && <Narr def={def} item={it} elapsed={elapsedFor} onToggle={() => patch(it.id, x => ({ open: !x.open }))} onArt={openArt} />}
          {def.recall && <MemRecall ids={def.recall} />}
          {showText && <div className={it.cut ? "ec-cutwrap" : ""}><Prose def={def} itemId={it.id} shown={ph === "write" || it.cut ? it.shown : null} mode={mode} streaming={ph === "write"} onRef={onRef} hot={hot} setHot={setHot} onArtOpen={openArt} activeArt={sheet.open ? sheet.art : null} /></div>}
          {it.fb && it.phase === "done" && <div className="ec-fb" style={{ marginTop: 12 }}><span className="ec-fbm off"><BrandMark id="anthropic" s={11} /></span><Ic n="chevR" s={10} /><span className="ec-fbm"><BrandMark id="openai" s={11} /></span><span>Answered by <b>GPT-5</b> · Claude Sonnet 4.5 didn't respond</span></div>}
          {it.fails && it.phase === "done" && <div style={{ marginTop: 12 }}><StepFails /></div>}
          {def.save && it.phase === "done" && <MemSave mem={def.save} ask={memAsk} />}
          {it.phase === "done" && <MsgActions pinned={!!it.pinned} reading={reading === it.id} chapter={items.filter(x => x.pinned).findIndex(x => x.id === it.id) + 1}
            onPin={() => patch(it.id, x => ({ pinned: !x.pinned }))} onRead={() => setReading(r => r === it.id ? null : it.id)} />}
        </div>
        <div className="m"></div>
      </div>
    );
  };

  return (
    <div className={"dsk" + (drag.on ? " dragging" : "") + (theme === "dark" ? " dk" : "") + " w-" + cfg.width + " t-" + cfg.text + (cfg.ring === "off" || cfg.motion === "reduced" ? " no-ring" : "") + (cfg.motion === "reduced" ? " calm" : "") + (cfg.drop === "composer" ? " drop-cmp" : "") + (cfg.refs === "off" ? " no-refs" : "") + (cfg.celebrate !== "confetti" ? " calm-ok" : "") + (cfg.celebrate === "off" ? " no-ok-fx" : "")} tabIndex={-1} onKeyDown={onKey} style={{ outline: "none" }}>
      <Rail view={view} noChats={noChats} onNew={() => { setView("home"); setSheet({ open: false }); }} onThread={() => setView("thread")} working={working} pending={pending.length} onSearch={() => setSearching(true)} onSettings={() => setSettingsOpen(true)} onWatch={() => { setView("watch"); setSheet({ open: false }); }} onDecisions={() => { setView("decisions"); setSheet({ open: false }); }} onToday={() => { setView("today"); setSheet({ open: false }); }} onMemory={() => { setView("memory"); setSheet({ open: false }); }} />
      <div className="mainc">
      <TopBar view={view} open={sheet.open} setOpen={o => { setSheet({ open: o }); setNewArt(false); }} pending={pending.length > 0} newArt={newArt} nArts={arts.length + hist.length}
        onReplay={onReplay} onHome={() => { setView("home"); setSheet({ open: false }); }} onThread={() => setView("thread")} onAgent={() => { setView("agent"); setSheet({ open: false }); }} onHandoff={handoff} />
      {view === "today" ? <TodayPage onAsk={t => send(t)} onWatch={() => setView("watch")} onDecisions={() => setView("decisions")} onThread={() => setView("thread")} onQueueItem={openQueueItem} /> : view === "memory" ? <MemoryPage memAsk={memAsk} /> : view === "agent" ? <AgentPage onBack={() => setView("thread")} /> : view === "decisions" ? <Decisions onAsk={() => setView("thread")} /> : view === "watch" ? <Watchtower onAsk={() => setView("thread")} /> : view === "home" ? <div className="home-w"><DropOverlay show={drag.on} hot={drag.hot} count={drag.count} /><Home value={input} setValue={setInput} onSend={send} up={up} drop={drag} cfg={cfg} /></div> : (
        <div className={"stage" + (sheet.open ? " open" : "")}>
          <div className={"room" + (working ? " lit" : "")}>
            <DropOverlay show={drag.on} hot={drag.hot} count={drag.count} />
            {offline && <div className="ec-off ec-offtop"><span className="ec-offd"></span>You're offline · messages will send when you reconnect</div>}
            <div className="scroll" ref={scroller} tabIndex={0}><div className="grid flow">{items.map(renderItem)}</div></div>
            {burst && <Confetti key={burst} seed={burst % 1000} />}
            <div className={"jump" + (away || unread ? " show" : "") + (working && (away || unread > 0) ? " live" : "")}>
              <button className="jump-b" onClick={jumpLatest} tabIndex={away || unread ? 0 : -1}>
                {working && (away || unread > 0) ? <><span className="jump-d"></span>Agent is replying</> : unread ? <><span className="jump-n">{unread}</span>New {unread === 1 ? "reply" : "replies"}</> : "Jump to latest"}
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round"><path d="M12 5v14M6 13l6 6 6-6"></path></svg>
              </button>
            </div>
            <div className="dock" ref={dockRef}><div className="grid">
              <div className="g"></div>
              <div className={"c" + (pending.length || confirming || undoing ? " has-dec" : "")}>
                {!pending.length && !confirming && !undoing && !dockCard && <TermsNote />}
                <FactsBar facts={facts} setFacts={setFacts} />
                {dockCard === "meter" && <div className="ec-meter"><div className="ec-mt"><span>Billing Specialist has used <b>92%</b> of October's budget</span><span className="ax-num">$460 / $500</span></div><div className="ec-mb"><i style={{ width: "92%" }}></i></div></div>}
                {dockCard === "context" && <ECard tone="info" icon="info" compact title="This conversation is too long for Claude Sonnet 4.5" sub="Start fresh and Desk carries over a summary and your pinned artifacts."
                  actions={<><button className="ec-btn ink" onClick={() => { setDockCard(null); setView("home"); }}>Continue in a new conversation</button><button className="ec-btn" onClick={() => { setDockCard(null); playStatus([GUARD], () => runTurn("ex")); }}><BrandMark id="gemini" s={11} />Switch to Gemini 2.5 Pro</button></>} />}
                {dockCard === "would" && <div className="dcx ec-would"><span className="dcx-i"><Ic n="alert" s={15} w={2} /></span><span className="dcx-t"><b>Assign biller <span className="dcx-s">would be refused as it stands</span></b><span className="dcx-d">38 of 120 items · 22 locked by a dispute · 11 in a closed period · 5 already have a biller</span></span><button className="bt sm" onClick={() => openBulk("wouldfail")}>Review 38</button><button className="apv-b ec-fix" onClick={() => { setDockCard(null); setPending(["d1"]); }}>Approve 82, skip 38</button></div>}
                {dockCard === "stale" && <div className="dcx ec-stale"><span className="dcx-i"><Ic n="undo" s={15} w={2} /></span><span className="dcx-t"><b>Assign biller <span className="dcx-s">on 11 items</span></b><span className="dcx-d">3 of these items changed after it was drafted · Jordan Pike assigned them</span></span><button className="bt sm" onClick={() => setDockCard(null)}>Not now</button><button className="apv-b" onClick={() => { setDockCard(null); setPending(["d1"]); }}>Redraft with 8 items</button></div>}
                {undoing && <UndoBar id={undoing.id} n={undoing.n} total={5} onUndo={undoApprove} onNow={approveNow} />}
                {!undoing && <DecisionCard pending={pending} confirming={confirming} onApprove={approve} onDismiss={dismiss} onReview={id => setSheet({ open: true, art: id === "d2" ? "dec2" : "dec" })} />}
                <Composer value={input} setValue={setInput} onSend={send} busy={working} status={status} placeholder={pending.length ? "Reply, or ask about this change…" : "Reply to Billing Specialist…"} ctx={{ parts: ctxParts, auto: autoCmp, setAuto: setAutoCmp }} onCompact={compact} compacting={compacting} onCancelCompact={cancelCompact} sendMod={cfg.send === "mod"} phase={cphase} cool={cool} up={up} drop={drag} pageKey={pageKey} pageSel={pageSel} setPageSel={setPageSel} cfg={cfg}
                  off={lock && <><Ic n={lock[0]} s={13} w={2} /><span><b>{lock[1]}</b> {lock[2]}</span><button className="ec-link">{lock[3]}</button></>} />
                <div className="hint">{pending.length ? <span>Press <span className="kbd">⌘↵</span> to approve</span> : <span>Desk can make mistakes. Check important details before you act on them.</span>}</div>
              </div>
              <div className="m"></div>
            </div></div>
          </div>
          <ArtifactPane s={sheet} set={setSheet} arts={arts} history={hist} newest={newest} pending={pending} resolved={resolved}
            assigned={assigned} posted={posted} flash={flash} onApprove={approve} onDismiss={dismiss} onCopy={copy} />
        </div>
      )}
      {settingsOpen && <SettingsDialog s={cfg} set={setCfg} onClose={() => setSettingsOpen(false)} />}
      {searching && <SearchPalette onClose={() => setSearching(false)} onOpenThread={() => setView("thread")}
        onOpenArt={art => { setView("thread"); openArt(art); }} onOpenDecisions={() => { setView("thread"); setSheet({ open: true, art: arts.includes("dec2") ? "dec2" : arts.includes("dec") ? "dec" : arts[arts.length - 1] }); }} />}
      {toast && <div className="toast"><Ic n="check" s={12} w={2.4} />{toast}</div>}
      </div>
    </div>
  );
}

function Replayable(props) {
  const [k, setK] = useState(0);
  return <DeskApp key={k + props.startView + props.mode + props.scenario + props.pageKey + props.noChats + props.noArts + props.memAsk} {...props} onReplay={() => setK(x => x + 1)} />;
}

const PAGE_OPTS = [["shipment", "Shipments"], ["billing", "Billing queue"], ["customer", "Customers"], ["report", "Detention report"], ["none", "Desk only (no page)"]];
const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "theme": "dark",
  "startView": "thread",
  "scenario": "Happy path",
  "page": "Shipments",
  "noChats": false,
  "noArts": false,
  "memAsk": false
}/*EDITMODE-END*/;

function Root() {
  const [t, setTweak] = useTweaks(TWEAK_DEFAULTS);
  return (
    <>
      <div style={{ width: "100vw", height: "100vh" }}><Replayable mode="ledger" theme={t.theme} startView={t.startView} scenario={(SCENARIOS.find(s => s[1] === t.scenario) || SCENARIOS[0])[0]} pageKey={PAGE_OPTS.find(p => p[1] === t.page) ? PAGE_OPTS.find(p => p[1] === t.page)[0] : "shipment"} noChats={t.noChats} noArts={t.noArts} memAsk={t.memAsk} /></div>
      <TweaksPanel>
        <TweakSection label="Appearance" />
        <TweakRadio label="Theme" value={t.theme} options={["dark", "light"]} onChange={v => setTweak("theme", v)} />
        <TweakRadio label="Start on" value={t.startView} options={["thread", "home"]} onChange={v => setTweak("startView", v)} />
        <TweakSection label="Empty states" />
        <TweakToggle label="No conversations" value={t.noChats} onChange={v => setTweak("noChats", v)} />
        <TweakToggle label="No artifacts" value={t.noArts} onChange={v => setTweak("noArts", v)} />
        <TweakSection label="Memory" />
        <TweakRadio label="Saving" value={t.memAsk ? "ask" : "auto"} options={["auto", "ask"]} onChange={v => setTweak("memAsk", v === "ask")} />
        <TweakButton label="Recall memories" secondary onClick={() => window.dispatchEvent(new CustomEvent("desk:mem", { detail: "mrec" }))} />
        <TweakButton label="Save a memory" secondary onClick={() => window.dispatchEvent(new CustomEvent("desk:mem", { detail: "msave" }))} />
        <TweakSection label="Current page" />
        <TweakSelect label="User is on" value={t.page} options={PAGE_OPTS.map(p => p[1])} onChange={v => setTweak("page", v)} />
        <TweakSection label="Uploads" />
        {[["cap-ok", "Scan from Capture"], ["cap-offline", "Capture · computer offline"], ["cap-none", "Capture · not set up"]].map(([k, l]) => (
          <TweakButton key={k} label={l} secondary onClick={() => window.dispatchEvent(new CustomEvent("desk:capture", { detail: k.slice(4) }))} />
        ))}
        {[["sample", "Sample rate confirmation"], ["fail", "Upload fails partway"], ["large", "File too large"], ["type", "Unsupported file type"], ["locked", "Password-protected PDF"], ["many", "More than 5 files"], ["messy", "Every state at once"]].map(([k, l]) => (
          <TweakButton key={k} label={l} secondary onClick={() => window.dispatchEvent(new CustomEvent("desk:upload-demo", { detail: k }))} />
        ))}
        <TweakSection label="Failure scenario" />
        <TweakSelect label="Kind" value={(SCEN_GROUPS.find(g => g[1].includes((SCENARIOS.find(s => s[1] === t.scenario) || SCENARIOS[0])[0])) || SCEN_GROUPS[0])[0]} options={SCEN_GROUPS.map(g => g[0])} onChange={v => setTweak("scenario", SCENARIOS.find(s => s[0] === SCEN_GROUPS.find(g => g[0] === v)[1][0])[1])} />
        <TweakSelect label="On next send" value={t.scenario} options={SCENARIOS.filter(s => (SCEN_GROUPS.find(g => g[1].includes((SCENARIOS.find(x => x[1] === t.scenario) || SCENARIOS[0])[0])) || SCEN_GROUPS[0])[1].includes(s[0])).map(s => s[1])} onChange={v => setTweak("scenario", v)} />
      </TweaksPanel>
    </>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<Root />);
