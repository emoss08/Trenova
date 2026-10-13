const EXT_PREVIEW = [
  { id: "browserbase", n: "Browserbase", vendor: "Browserbase", cat: "Browser automation", m: "Bb", h: 30, c: 0.14, summary: "A real headless browser for agents — log in to carrier and broker portals, click through, and read what's on the page.", caps: ["Opens a browser session an agent can drive", "Signs in to portals with saved credentials", "Records a replay of every session"], tools: ["browser_open", "browser_act", "browser_extract"], notice: "Pages load in Browserbase's cloud under your account. Portal passwords stay in Trenova's vault.", ph: "Browserbase API key", leaves: true, isNew: true, released: 9 },
  { id: "firecrawl", n: "Firecrawl", vendor: "Firecrawl", cat: "Scraping & crawling", m: "Fc", h: 40, c: 0.15, summary: "Turns any website into clean markdown, so agents can read rate sheets, fuel tables and carrier sites without the clutter.", caps: ["Scrapes a page to clean markdown", "Crawls a whole site or section", "Maps every URL on a domain"], tools: ["scrape_page", "crawl_site", "map_site"], notice: "Only the URLs an agent asks for are sent to Firecrawl.", ph: "fc-…", released: 8 },
  { id: "tinyfish", n: "TinyFish", vendor: "TinyFish", cat: "Browser automation", m: "TF", h: 210, c: 0.12, summary: "Web agents that run many sites in parallel and bring back structured data — useful for checking tracking pages or posted rates at scale.", caps: ["Runs a goal across many sites at once", "Returns structured fields, not raw pages", "Handles logins and pagination"], tools: ["run_web_agent", "extract_structured"], notice: "Goals and target sites go to TinyFish under your account. Record details stay in Trenova.", ph: "TinyFish API key", isNew: true, released: 10 },
  { id: "tavily", n: "Tavily", vendor: "Tavily", cat: "Web research", m: "Tv", h: 258, c: 0.12, summary: "A search engine built for agents: ranked results with the page text already extracted.", caps: ["Search tuned for agent questions", "Extracts page text with each result", "News and general modes"], tools: ["tavily_search", "tavily_extract"], notice: "Only the search text is sent, never your records.", ph: "tvly-…", released: 6 },
  { id: "perplexity", n: "Perplexity Sonar", vendor: "Perplexity", cat: "Web research", m: "Px", h: 190, c: 0.1, summary: "Ask a question and get a cited answer from the live web in one call — handy for regulations and news that move fast.", caps: ["Cited answers from the live web", "Recent news and regulations", "Returns every source it used"], tools: ["ask_sonar"], notice: "Only the question is sent, never your records.", ph: "pplx-…", released: 5 },
  { id: "e2b", n: "E2B", vendor: "E2B", cat: "Code execution", m: "E2", h: 145, c: 0.13, summary: "A sandbox where agents run Python to crunch numbers — lane margins, IFTA math, settlement checks — and show their work.", caps: ["Runs Python in an isolated sandbox", "Reads CSV and Excel exports", "Returns charts and tables"], tools: ["run_code", "read_file"], notice: "Only the data an agent passes in is uploaded; the sandbox is wiped after each run.", ph: "e2b_…", released: 7 },
  { id: "reducto", n: "Reducto", vendor: "Reducto", cat: "Documents", m: "Rd", h: 300, c: 0.12, summary: "Reads messy PDFs — scanned BOLs, rate confirmations, lumper receipts — into tables and fields agents can trust.", caps: ["Parses scanned and native PDFs", "Pulls tables with their layout kept", "Confidence on every field"], tools: ["parse_document", "extract_table"], notice: "Documents are sent to Reducto under your account and deleted after parsing.", ph: "Reducto API key", released: 4 },
  { id: "deepgram", n: "Deepgram", vendor: "Deepgram", cat: "Voice", m: "Dg", h: 160, c: 0.12, summary: "Transcribes driver and customer calls so agents can file what was promised against the right load.", caps: ["Transcribes recorded calls", "Speaker labels and timestamps", "Pulls out times, places and load numbers"], tools: ["transcribe_call"], notice: "Call audio goes to Deepgram under your account.", ph: "Deepgram API key", released: 3 },
  { id: "elevenlabs", n: "ElevenLabs", vendor: "ElevenLabs", cat: "Voice", m: "11", h: 0, c: 0, summary: "Gives agents a voice for check calls and appointment confirmations — every call waits for a person to approve.", caps: ["Natural voice for outbound calls", "Reads back appointment details", "Always held for approval"], tools: ["place_call", "speak"], notice: "Calls leave the organization, so every one waits for approval.", ph: "ElevenLabs API key", leaves: true, released: 2 },
];
const EXT_CATS_ORDER = ["Web research", "Browser automation", "Scraping & crawling", "Documents", "Code execution", "Voice"];
const EXT_WS = { id: "websearch", cat: "Web research", featured: true, released: 10, ph: "Exa API key", opts: true };
const DEMO_NAR = [["Searching the web", "Ridgeline Freight MC 884213 insurance", "0.8s"], ["Reading 2 pages", "safer.fmcsa.dot.gov · ridgelinefreight.com", "1.3s"]];
const DEMO_ANS = "Yes. Their liability policy with Great West Casualty runs to Mar 31, 2027, and FMCSA shows the authority as active.".split(" ");
function Demo() {
  const [t, setT] = React.useState(0);
  React.useEffect(() => { const id = setInterval(() => setT(x => x >= 48 ? 0 : x + 1), 140); return () => clearInterval(id); }, []);
  const q = t >= 1, w = t >= 5, n1 = t >= 8, n2 = t >= 15, done = t >= 22, words = done ? Math.min(DEMO_ANS.length, t - 22) : 0, src = t >= 22 + DEMO_ANS.length;
  const ag = AGENTS0.find(x => x.id === "dispatch");
  return (
    <div className="demo">
      <span className="demo-k"><Ic n="chat" s={11}></Ic>How agents use it · Desk</span>
      <div className={"dq" + (q ? " in" : "")}><span className="dq-w"><span className="me sm">DO</span><b>Dana</b><span className="mono">9:41</span></span><p>Is Ridgeline Freight's insurance current?</p></div>
      <div className={"da" + (w ? " in" : "")}>
        <span className="who"><span className={"dm" + (w && !done ? " busy" : "")}></span><b>{ag.n}</b><span className="dtg"><Ic n="search" s={10}></Ic>Web search</span></span>
        <div className="dn">{DEMO_NAR.map(([l, d, el], i) => { const on = i ? n2 : n1, past = i ? done : n2; return <div key={on ? "on" + i : "off" + i} className={"dnl" + (!on ? " hid" : past ? " past" : " cur")}><span className="ck">{past ? <Ic n="check" s={12} w={2.4}></Ic> : <i className="spn"></i>}</span><span className="tx">{past ? l.replace("Searching", "Searched").replace("Reading", "Read") : l}<em>{d}</em></span><span className="el mono">{past ? el : ""}</span></div>; })}</div>
        <p className="dp">{DEMO_ANS.slice(0, words).map((x, i) => <span key={i} className="dw">{x} </span>)}{!done ? null : words < DEMO_ANS.length ? <span className="car"></span> : <span className="wc w"><span className="wf" style={{ "--wh": 230, width: 12, height: 12, fontSize: 7.5 }}>F</span>fmcsa.dot.gov<em>+1</em></span>}</p>
        <div className={"dsx" + (src ? " in" : "")}><button className="ws-b"><span className="ws-st"><span className="wf" style={{ "--wh": 230, width: 16, height: 16, fontSize: 9 }}>F</span><span className="wf" style={{ "--wh": 25, width: 16, height: 16, fontSize: 9 }}>R</span></span>2 sources<Ic n="chevR" s={11}></Ic></button></div>
      </div>
    </div>
  );
}
function stOf(e) { return e.enabled && (e.key || e.nokey) ? "on" : e.enabled ? "setup" : "off"; }
function StateTag({ e }) { const s = stOf(e); return s === "on" ? <span className="tg k"><i></i>On</span> : s === "setup" ? <span className="tg w">Needs a key</span> : e.isNew ? <span className="tg b">New</span> : null; }
function ExtDetail({ e, S, set }) {
  const [k, setK] = React.useState(""); const [testing, setTesting] = React.useState(null);
  const chat = S.agents.filter(a => a.trig === "Chat" && a.tools);
  const agents = e.agents || [];
  const on = stOf(e) === "on", canOn = e.key || e.nokey || k;
  return (
    <div className="pd">
      <div className="pd-g">
        <div className="pd-s"><h4>What it can do</h4>
          <ul className="caps">{e.caps.map(c => <li key={c}><Ic n="check" s={12} w={2.2}></Ic>{c}</li>)}</ul>
          <div className="pv-tk mt8">{e.tools.map(t => <span key={t} className="tk mono"><Ic n="tool" s={10}></Ic>{t}</span>)}</div>
          <p className="ex-n"><Ic n="shield" s={12}></Ic>{e.notice}</p>
        </div>
        <div className="pd-s"><h4>Connection</h4>
          {e.nokey ? <div className="kst"><Ic n="check" s={12} w={2.2}></Ic><span>No key needed — it's a public source</span></div>
            : e.key ? <div className="kst"><Ic n="key" s={12}></Ic><span>{e.oauth ? "Connected to " + e.vendor : e.vendor + " key stored"}</span><button className="lnk" onClick={() => set({ key: false })}>{e.oauth ? "Disconnect" : "Replace"}</button></div>
              : e.oauth ? <button className="btn" onClick={() => set({ key: true }, "Connected to " + e.vendor)}><Ic n="link" s={13}></Ic>{e.ph}</button>
                : <div className="pd-kf"><Ic n="key" s={13}></Ic><input type="password" placeholder={e.ph || "API key"} value={k} onChange={ev => setK(ev.target.value)} autoFocus></input></div>}
          {e.opts && <><h4 className="mt">Search depth</h4><Seg v={e.depth || "auto"} opts={[["auto", "Auto"], ["fast", "Fast"], ["deep", "Deep"]]} onChange={v => set({ depth: v })} cls="sm"></Seg></>}
          <h4 className="mt">Daily request limit</h4>
          <Seg v={e.limit || 500} opts={[[100, "100"], [500, "500"], [1000, "1,000"], [5000, "5,000"]]} onChange={v => set({ limit: v })} cls="sm"></Seg>
        </div>
        <div className="pd-s"><h4>Available to</h4>
          <Seg v={e.avail || "SelectedAgents"} opts={[["SelectedAgents", "Agents you choose"], ["AllAgents", "Every agent"]]} onChange={v => set({ avail: v })} cls="sm"></Seg>
          {(e.avail || "SelectedAgents") === "SelectedAgents" ? <div className="ex-ag">{chat.map(a => { const o = agents.includes(a.id); return <button key={a.id} className={"ex-a" + (o ? " on" : "")} onClick={() => set({ agents: o ? agents.filter(x => x !== a.id) : [...agents, a.id] })}><Tile a={a} s={18}></Tile><span>{a.n}</span>{o && <Ic n="check" s={11} w={2.4}></Ic>}</button>; })}</div>
            : <p className="ad-h">Every agent, including the assistant, gets {e.tools.map((t, i) => <React.Fragment key={t}>{i ? ", " : ""}<span className="mono">{t}</span></React.Fragment>)}.</p>}
          {e.leaves && <p className="ex-n w"><Ic n="warn" s={12}></Ic>Its work leaves the organization, so it never runs past approval.</p>}
        </div>
      </div>
      <div className="ad-bar sh-f">
        {on ? <button className="xa d" onClick={() => set({ enabled: false }, e.n + " is off")}><Ic n="ban" s={13}></Ic>Turn off</button>
          : <button className="btn sm ink" disabled={!canOn} onClick={() => set({ key: true, enabled: true }, e.n + " is on for " + ((e.avail || "SelectedAgents") === "AllAgents" ? "every agent" : agents.length + " agent" + (agents.length === 1 ? "" : "s")))}>Turn on</button>}
        {!e.nokey && <button className="xa" disabled={!e.key || testing === "run"} onClick={() => { setTesting("run"); setTimeout(() => setTesting("ok"), 1100); }}><Ic n="plug" s={13}></Ic>{testing === "run" ? "Testing…" : "Test connection"}</button>}
        {testing === "ok" && <span className="tr t-k"><Ic n="check" s={11} w={2.4}></Ic>Works · 412 ms</span>}
        <span className="sp"></span>
        <a className="xa" href="#" onClick={ev => ev.preventDefault()}><Ic n="ext" s={13}></Ic>{e.vendor} docs</a>
      </div>
    </div>
  );
}
function Magic({ from, to, className, children, onClick, as }) {
  const Tag = as || "button";
  const el = React.useRef(), orb = React.useRef(), st = React.useRef({ x: -200, y: -200, tx: -200, ty: -200, o: 0, to: 0, raf: 0 });
  const loop = () => { const s = st.current; s.x += (s.tx - s.x) * 0.22; s.y += (s.ty - s.y) * 0.22; s.o += (s.to - s.o) * 0.16; if (orb.current) { orb.current.style.transform = `translate(${s.x}px,${s.y}px) translate(-50%,-50%)`; orb.current.style.opacity = s.o; } if (el.current) { el.current.style.setProperty("--mx", s.tx + "px"); el.current.style.setProperty("--my", s.ty + "px"); } if (Math.abs(s.tx - s.x) > 0.5 || Math.abs(s.ty - s.y) > 0.5 || Math.abs(s.to - s.o) > 0.01) s.raf = requestAnimationFrame(loop); else { s.raf = 0; s.o = s.to; if (orb.current) orb.current.style.opacity = s.o; } };
  const kick = () => { if (!st.current.raf) st.current.raf = requestAnimationFrame(loop); };
  const move = e => { const r = el.current.getBoundingClientRect(), s = st.current; s.tx = e.clientX - r.left; s.ty = e.clientY - r.top; if (s.o < 0.02 && s.to === 0) { s.x = s.tx; s.y = s.ty; } kick(); };
  React.useEffect(() => () => cancelAnimationFrame(st.current.raf), []);
  return (
    <Tag ref={el} {...(Tag === "button" ? {} : { role: "button", tabIndex: 0, onKeyDown: ev => { if ((ev.key === "Enter" || ev.key === " ") && ev.target === ev.currentTarget) { ev.preventDefault(); onClick && onClick(); } } })} className={"mg " + (className || "")} style={{ "--gf": from, "--gt": to }} onClick={onClick} onPointerMove={move} onPointerEnter={e => { move(e); st.current.to = 0.9; kick(); }} onPointerLeave={() => { st.current.to = 0; st.current.tx = st.current.tx; kick(); }}>
      <span className="mg-bg"></span>
      <span ref={orb} className="mg-orb" aria-hidden="true"></span>
      <span className="mg-c">{children}</span>
    </Tag>
  );
}
function Tile2({ e, onOpen }) {
  const s = stOf(e), share = Math.min(1, (e.today || 0) / (e.limit || 500));
  return (
    <Magic className={"mkt-t" + (s === "on" ? " on" : "")} onClick={onOpen} from={e.c ? `oklch(0.66 0.17 ${e.h})` : "var(--brand)"} to={e.c ? `oklch(0.62 0.16 ${(e.h + 50) % 360})` : "oklch(0.7 0.12 230)"}>
      <div className="mkt-h"><Mark p={e} s={40}></Mark><div className="mkt-n"><b>{e.n}</b><span>{e.vendor} · {e.cat}</span></div><StateTag e={e}></StateTag></div>
      <p>{e.summary}</p>
      <div className="mkt-f">
        <span className="mono">{e.tools.length} tool{e.tools.length > 1 ? "s" : ""}</span>
        {e.nokey && <span>No key</span>}{e.leaves && <span className="t-w">Held for approval</span>}
        <span className="sp"></span>
        {s === "on" ? <span className="mkt-u"><span className="ubar"><i style={{ width: share * 100 + "%" }}></i></span><span className="mono">{e.today || 0}/{e.limit || 500}</span></span> : <span className="mkt-cta">{s === "setup" ? "Finish setup" : "Set up"}<Ic n="arrowR" s={11}></Ic></span>}
      </div>
    </Magic>
  );
}
function ExtensionsTab({ S, A, searchRef }) {
  const [q, setQ] = React.useState(""); const [cat, setCat] = React.useState("all"); const [sort, setSort] = React.useState("featured"); const [open, setOpen] = React.useState(null);
  const ws = { ...EXT0, ...EXT_WS, ...S.ext };
  const cat0 = S.extCatalog === "preview" ? [ws, ...EXT_PREVIEW.map(e => ({ ...e, ...(S.extOv[e.id] || {}) }))] : [ws];
  const setFor = e => (patch, msg) => e.id === "websearch" ? A.setExt(patch, msg) : A.setExtOv(e.id, patch, msg);
  const cats = EXT_CATS_ORDER.map(c => [c, cat0.filter(e => e.cat === c).length]).filter(x => x[1]);
  const onList = cat0.filter(e => stOf(e) === "on");
  let list = cat0.filter(e => (cat === "all" || (cat === "on" ? stOf(e) === "on" : e.cat === cat)) && (e.n + e.vendor + e.summary + e.tools.join(" ") + e.cat).toLowerCase().includes(q.toLowerCase()));
  list = [...list].sort((a, b) => sort === "name" ? a.n.localeCompare(b.n) : sort === "newest" ? b.released - a.released : (b.featured ? 1 : 0) - (a.featured ? 1 : 0) || b.released - a.released);
  const feat = cat0.find(e => e.featured);
  const showFeat = feat && cat === "all" && !q;
  const oe = open && cat0.find(e => e.id === open);
  return (
    <div className="tabp">
      <div className="mkt-top">
        <div><h2>Give your agents new abilities</h2><p>Each extension runs under your organization's own account with the vendor, and only inside AI features.</p></div>
        <label className="srch lg"><Ic n="search" s={14}></Ic><input ref={searchRef} value={q} onChange={e => setQ(e.target.value)} placeholder="Search extensions, vendors or tools"></input><span className="kbd">/</span></label>
      </div>
      {showFeat && (
        <section className={"spot" + (stOf(feat) === "on" ? " on" : "")}>
          <div className="spot-m">
            <span className="spot-k">Featured</span>
            <div className="mkt-h"><Mark p={feat} s={48}></Mark><div className="mkt-n"><b>{feat.n}</b><span>By {feat.vendor} · {feat.cat}</span></div><StateTag e={feat}></StateTag></div>
            <p className="spot-p">{feat.summary} Agents cite every page they used, and only the question leaves Trenova.</p>
            <ul className="caps two">{feat.caps.map(c => <li key={c}><Ic n="check" s={12} w={2.2}></Ic>{c}</li>)}</ul>
            <div className="spot-a">{stOf(feat) === "on" ? <><button className="btn" onClick={() => setOpen(feat.id)}><Ic n="gear" s={13}></Ic>Manage</button><span className="mkt-u"><span className="ubar"><i style={{ width: Math.min(1, (feat.today || 0) / feat.limit) * 100 + "%" }}></i></span><span className="mono">{feat.today || 0} of {feat.limit} today</span></span></> : <><button className="btn ink lg" onClick={() => setOpen(feat.id)}>Set up {feat.n}<Ic n="arrowR" s={12}></Ic></button><span className="spot-s">About a minute · needs an Exa key</span></>}</div>
          </div>
          <Demo></Demo>
        </section>
      )}
      <div className="mkt">
        <nav className="mkt-c">
          <button className={cat === "all" ? "on" : ""} onClick={() => setCat("all")}><span>All extensions</span><em className="mono">{cat0.length}</em></button>
          <button className={cat === "on" ? "on" : ""} onClick={() => setCat("on")}><span>On</span><em className="mono">{onList.length}</em></button>
          <span className="mkt-ch">Categories</span>
          {cats.map(([c, n]) => <button key={c} className={cat === c ? "on" : ""} onClick={() => setCat(c)}><span>{c}</span><em className="mono">{n}</em></button>)}
        </nav>
        <div className="mkt-b">
          <div className="mkt-bh"><b>{cat === "all" ? (q ? "Results" : "All extensions") : cat === "on" ? "On for your agents" : cat}</b><em className="mono">{list.length}</em><span className="sp"></span><Seg v={sort} opts={[["featured", "Featured"], ["newest", "Newest"], ["name", "Name"]]} onChange={setSort} cls="sm"></Seg></div>
          {list.length ? <div className="mkt-g">{list.map(e => <Tile2 key={e.id} e={e} onOpen={() => setOpen(e.id)}></Tile2>)}</div>
            : <div className="dtx-e"><b>{cat === "on" ? "Nothing is on yet" : "No extensions match"}</b><span>{cat === "on" ? "Set one up and it shows here." : "Try another word or category."}</span></div>}
          <div className="mkt-more"><Ic n="sparkle" s={13}></Ic><span>{S.extCatalog === "preview" ? "New extensions arrive with Trenova updates." : "More extensions are on the way — they arrive with Trenova updates."}</span><button className="lnk" onClick={() => A.toast("Thanks — we'll pass it to the team")}>Request an extension<Ic n="arrowR" s={11}></Ic></button></div>
        </div>
      </div>
      {oe && <Sheet onClose={() => setOpen(null)} head={<><Mark p={oe} s={36}></Mark><div className="sh-t"><b>{oe.n}</b><span>By {oe.vendor} · {oe.cat}</span></div>{stOf(oe) === "on" && <Switch on={true} label="Turn off" onChange={() => setFor(oe)({ enabled: false }, oe.n + " is off")}></Switch>}</>}>
        <p className="sh-d">{oe.summary}</p>
        <ExtDetail e={oe} S={S} set={setFor(oe)}></ExtDetail>
      </Sheet>}
    </div>
  );
}
Object.assign(window, { ExtensionsTab, Magic });
