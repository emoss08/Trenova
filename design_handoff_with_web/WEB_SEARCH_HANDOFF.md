# Web search sources — implementation handoff

Scope: ONLY web search. Everything else in desk v2 is already shipped. Match this spec exactly; when this doc and the prototype disagree, the prototype (`trenova/desk-v2/websrc.jsx`, `websrc.css`) wins.

## 1. What the user sees

Three surfaces, in this order during a turn:

**A. Live search block** (only while the active trace step is a web search)
- Rendered in the assistant row's content column (same `.r.a` row, time in the gutter), in place of the prose that hasn't started yet.
- Line 1: globe icon (13px) + step label, e.g. "Searching the web". Label has a left→right shimmer (1.6s linear infinite). Reduced motion: no shimmer, plain `--muted`.
- Line 2: the query in a mono pill (`--sunken` bg, 6px radius, 11.5px IBM Plex Mono, `--subtle`).
- Line 3: one chip per source as it's found — monogram favicon (12px) + domain. Chips pop in (320ms spring, from translateY(4px) scale(.94), opacity 0). In the prototype they stagger across the step duration: `delay = 350 + i * ((step.ms - 600) / sources.length)` ms. **In production, append each chip when the backend actually reports that result** (no fake stagger); keep the same entry animation.
- Disappears when the step finishes (next step / writing begins). No collapse animation.

**B. Inline citation pills** (inside the reply prose)
- Placed immediately after the sentence/clause they support, before the period is fine (the prototype puts them before the period: `up 3¢[eia +1].`).
- Pill: 18px tall, 9px radius, 0 6px padding, 2px horizontal margin, `vertical-align:1px`, `--sunken` bg, `--subtle` text, Geist 500 11px.
- Label = first source's domain with a trailing `.com|.gov|.org` stripped (`eia.gov` → `eia`, `freightwaves.com` → `freightwaves`). If more than one source: append `+N` (N = extra count) in `--faint`.
- Hover: pill inverts (`--fg` bg, `--card` text, `+N` inherits at .7 opacity), and a popover opens above, centered (same animation as the step footnote popover: `fnp` 200ms settle).
- Popover: 280px wide, 4px padding, 12px radius, `--raised`, `--lift-hi` shadow. One entry per cited source, separated by 1px `--b-sub`; each entry: row of [favicon 12px · domain (Plex 11px `--subtle`) · date right-aligned `--faint`], then title (12.5px 500 `--fg`), then snippet (12px `--muted`). Entry hover `--hover` bg, 8px radius.
- Clicking the pill opens the first source in a new tab; clicking an entry in the popover opens that source. (The prototype calls `preventDefault` — **remove that in production**; links are real.)
- While streaming, the pill uses the same pop-in as artifact badges (`badge` keyframe, 520ms spring).
- If none of the cited IDs resolve to a source, render nothing.

**C. Sources footer** (after the reply finishes, phase = done)
- Sits under the prose, 10px top margin, before memory-save / message actions.
- Collapsed button: 26px tall pill, overlapping stack of the first 4 favicons (14px each, -4px overlap, 1.5px `--card` ring), label "N sources", chevron-right 10px. `--subtle` text; hover/open → `--hover` bg, `--fg` text. Chevron rotates 90° when open (200ms ease).
- Expanded (grid-rows 0fr→1fr, 260ms ease): first the query line (search icon 11px + query, Plex 11px `--faint`), then a numbered list. Each row is a link: grid `16px 14px 1fr auto` → [index (Plex 10.5px `--faint`, right-aligned) · favicon 14px · title (12.5px, single line, ellipsis) · "domain · date" (Plex 11px `--faint`, nowrap)]. Row hover `--hover`, 8px radius.
- Only shown when the turn has ≥1 source.

**Favicon** (`Fav`): a monogram tile, not a fetched favicon. First letter of the domain, uppercase, Geist 600, font-size = 0.6 × size, 4px radius. Colour is derived from the domain: `hue = Σ (h*31 + charCode) mod 360`; bg `oklch(0.58 0.12 hue)` (dark theme `oklch(0.52 0.11 hue)`), text `oklch(0.98 0.02 hue)`. Same domain = same colour everywhere.

**Trace / receipt:** web search is a normal trace step like any other tool: `k: "web.search"`, `v` = query, `r` = "6 sources". A follow-up page read is `k: "web.fetch"`, `v` = "eia.gov · ohgo.com · dat.com", `r` = "3 pages". These appear in the existing step footnotes/receipt exactly like other tools — no special styling.

## 2. Data contract

Per assistant turn:

```ts
type WebSource = {
  d: string;   // domain, no scheme/www — "eia.gov"
  t: string;   // page title
  s: string;   // one-line snippet (the cited text, trimmed)
  a: string;   // display age/date — "Today" | "Sep 28"
  url?: string // full URL (production; prototype builds https://{d})
};
type Turn = {
  trace: Step[];         // web.search step has web: true
  sources?: WebSource[]; // ordered; citations are 1-based indexes into this
  text: Segment[][];     // new segment kind: { c: number[] }  → citation pill
};
```

- Dedupe sources by URL; order = order first returned by the search.
- Citation segment `{ c: [1, 4] }` is zero-width in the token stream (like `{ a: artId }`): it doesn't count as a word for streaming, it renders once reached.
- Mapping from Anthropic `web_search` tool output: each `web_search_tool_result` item → a `WebSource` (`url`, `title`, `page_age` → `a`). Each text block's `citations[]` (`web_search_result_location`: `url`, `title`, `cited_text`) → after that text block, emit one `{ c: [...] }` with the indexes of those URLs; use `cited_text` as the source's `s` if it doesn't have one. Other providers: map their equivalent; if a provider gives no citations, show only the sources footer.
- Stream events needed by the UI: step start (query) → one event per result found (append chip) → step end (count) → text deltas with citation markers → turn done (final `sources`).
- Permissions/budget: web search is already a tool in Agent permissions ("Allowed / Ask first / Off") and budget ("Web search · resets Nov 1"). No new UI there.

## 3. Integration points (exact changes made to the prototype)

**thread.jsx — tokenizer**, add after the `seg.a` line:
```js
if (seg && seg.c) { out.push({ p, cite: seg.c, w: "" }); return; }
```
**thread.jsx — Prose**, first line inside the word map:
```js
if (t.cite) return <WebCite key={i} ids={t.cite} sources={def.sources || []} live={!!cls} />;
```
**app.jsx — renderItem**, replace `if (!showText) return null;`:
```js
if (!showText) {
  const cur = ph === "work" && def.trace[it.step];
  if (!cur || !cur.web || !def.sources) return null;
  return <div className={"r a" + first} key={it.id}><div className="g"><span className="time">{def.time}</span></div><div className="c"><WebLive step={cur} sources={def.sources} /></div><div className="m"></div></div>;
}
```
**app.jsx — renderItem**, directly after the `<Prose …/>` wrapper and before the fallback-model line:
```js
{def.sources && it.phase === "done" && <WebSources def={def} />}
```
**icons.jsx** — new icon:
```js
globe: <><circle cx="12" cy="12" r="8"></circle><path d="M4 12h16M12 4c2.4 2.3 3.5 5 3.5 8s-1.1 5.7-3.5 8c-2.4-2.3-3.5-5-3.5-8s1.1-5.7 3.5-8z"></path></>,
```
(24×24 viewBox, stroke icon, same stroke settings as the rest of the set.)

Prototype-only (do NOT ship): the `/\b(web|online|…)\b/` send regex, the `desk:web` event, the "Web search" Tweaks button, and `TURNS.web` demo data.

## 4. Tokens used

Light / dark: `--card` #fff / #0a0a0a · `--raised` #fff / #111 · `--sunken` #f2f2f2 / #1a1a1a · `--hover` #f2f2f2 / #1f1f1f · `--fg` #171717 / #ededed · `--muted` #4d4d4d / #a1a1a1 · `--subtle` #666 / #a1a1a1 · `--faint` #8f8f8f / #707070 · `--b-sub` #ebebeb / #1f1f1f · `--b` #e5e5e5 / #2e2e2e
`--lift-hi` light: `0 0 0 1px rgba(0,0,0,.08),0 4px 8px -4px rgba(0,0,0,.06),0 16px 24px -8px rgba(0,0,0,.08)` · dark: `0 0 0 1px rgba(255,255,255,.12),0 16px 32px -12px rgba(0,0,0,.8)`
Easing: `--spring` cubic-bezier(.34,1.4,.64,1) · `--settle` cubic-bezier(.16,1,.3,1) · `--ease` cubic-bezier(.2,.8,.2,1)
Fonts: Geist (UI), IBM Plex Mono (`--plex`: query, domains, indexes).
Existing keyframes reused: `@keyframes fnp{from{opacity:0;transform:translate(-50%,4px)}to{opacity:1;transform:translate(-50%,0)}}` · `@keyframes badge{from{opacity:0;transform:scale(.85) translateY(4px)}to{opacity:1;transform:none}}`

## 5. Acceptance checklist

- [ ] Live block shows label shimmer, query pill, chips appearing per real result; gone once the step ends.
- [ ] Citation pills appear inline at the right spot during streaming with pop-in; label rule (`.com/.gov/.org` stripped, `+N`) correct.
- [ ] Hover popover lists every cited source with domain, date, title, snippet; links open in a new tab.
- [ ] Sources footer appears only when done and only if sources exist; stack shows max 4 favicons; expands/collapses with the grid-rows transition; query line shown.
- [ ] Monogram colour is deterministic per domain and identical in pill popover, live chips and footer.
- [ ] `web.search` / `web.fetch` appear in the trace/receipt like other tools.
- [ ] Light + dark both correct; reduced motion disables shimmer and chip animation.
- [ ] Unknown citation indexes render nothing; turns without web search render exactly as before.

## 6. Reference source — websrc.jsx (verbatim)

```jsx
TURNS.web = {
  dur: "5s", time: "10:46 PM",
  trace: [
    { pose: "lookup", live: "Searching the web", done: "Searched the web", k: "web.search", v: "DOE diesel average · I-70 Ohio closures", r: "6 sources", t: "2.2s", ms: 2600, web: true },
    { pose: "lookup", live: "Reading 3 pages", done: "Read 3 pages", k: "web.fetch", v: "eia.gov · ohgo.com · dat.com", r: "3 pages", t: "1.4s", ms: 1400 },
  ],
  sources: [
    { d: "eia.gov", t: "Gasoline and Diesel Fuel Update", s: "Weekly U.S. on-highway diesel average, released Mondays.", a: "Sep 28" },
    { d: "ohgo.com", t: "OHGO · Ohio traffic and road conditions", s: "Lane restriction on I-70 eastbound near Zanesville, mile 152–155.", a: "Today" },
    { d: "dat.com", t: "Weekly freight market update", s: "Midwest dry van spot rates steady week over week.", a: "Sep 30" },
    { d: "freightwaves.com", t: "Diesel ticks up for a second week", s: "Retail diesel rose for the second straight week.", a: "Sep 29" },
    { d: "weather.gov", t: "Columbus, OH forecast", s: "Clear through Thursday.", a: "Today" },
    { d: "ttnews.com", t: "Ohio DOT schedules overnight work on I-70", s: "Overnight closures expected through Oct 9.", a: "Sep 27" },
  ],
  text: [
    ["The DOE diesel average is ", { b: "$3.74/gal" }, " this week, up 3¢", { c: [1, 4] }, ". That moves Acme's fuel surcharge from ", { b: "$0.41" }, " to ", { b: "$0.42" }, " per mile starting Monday."],
    ["There's no full closure on I-70, but eastbound is down to one lane near Zanesville overnight through Oct 9", { c: [2, 6] }, ". The two Columbus loads deliver in daylight, so I'd leave them routed as is."],
  ],
};

const webHue = d => { let h = 0; for (let i = 0; i < d.length; i++) h = (h * 31 + d.charCodeAt(i)) % 360; return h; };
function Fav({ d, s = 14 }) {
  return <span className="wf" style={{ width: s, height: s, fontSize: s * 0.6, "--wh": webHue(d) }}>{d[0].toUpperCase()}</span>;
}

function WebCite({ ids, sources, live }) {
  const [open, setOpen] = React.useState(false);
  const list = ids.map(i => sources[i - 1]).filter(Boolean);
  if (!list.length) return null;
  const first = list[0];
  return (
    <span className="wc-w" onMouseEnter={() => setOpen(true)} onMouseLeave={() => setOpen(false)}>
      <a className={"wc" + (live ? " w" : "")} href={"https://" + first.d} target="_blank" rel="noreferrer" onClick={e => e.preventDefault()}>
        {first.d.replace(/\.(com|gov|org)$/, "")}{list.length > 1 && <em>+{list.length - 1}</em>}
      </a>
      {open && (
        <span className="wc-pop"><span className="wc-in">
          {list.map(s => (
            <span key={s.d} className="wc-it">
              <span className="wc-d"><Fav d={s.d} s={12} />{s.d}<i>{s.a}</i></span>
              <b>{s.t}</b>
              <span className="wc-s">{s.s}</span>
            </span>
          ))}
        </span></span>
      )}
    </span>
  );
}

function WebLive({ step, sources }) {
  return (
    <div className="wl">
      <div className="wl-h"><Ic n="globe" s={13} /><span className="wl-l">{step.live}</span></div>
      <div className="wl-q">{step.v}</div>
      <div className="wl-s">
        {sources.map((s, i) => (
          <span key={s.d} className="wl-it" style={{ animationDelay: 350 + i * ((step.ms - 600) / sources.length) + "ms" }}><Fav d={s.d} s={12} />{s.d}</span>
        ))}
      </div>
    </div>
  );
}

function WebSources({ def }) {
  const [open, setOpen] = React.useState(false);
  const src = def.sources;
  const q = (def.trace.find(s => s.web) || {}).v;
  return (
    <div className={"ws" + (open ? " open" : "")}>
      <button className="ws-b" onClick={() => setOpen(o => !o)}>
        <span className="ws-st">{src.slice(0, 4).map(s => <Fav key={s.d} d={s.d} s={14} />)}</span>
        <span>{src.length} sources</span>
        <Ic n="chevR" s={10} />
      </button>
      <div className="ws-x"><div>
        {q && <div className="ws-q"><Ic n="search" s={11} />{q}</div>}
        <ol className="ws-l">
          {src.map((s, i) => (
            <li key={s.d}>
              <a href={"https://" + s.d} target="_blank" rel="noreferrer" onClick={e => e.preventDefault()}>
                <span className="ws-n">{i + 1}</span><Fav d={s.d} s={14} />
                <span className="ws-t">{s.t}</span>
                <span className="ws-d">{s.d} · {s.a}</span>
              </a>
            </li>
          ))}
        </ol>
      </div></div>
    </div>
  );
}

Object.assign(window, { WebCite, WebLive, WebSources, Fav });
```

## 7. Reference source — websrc.css (verbatim)

```css
.wf{display:inline-grid;place-items:center;flex:none;border-radius:4px;font-family:'Geist',sans-serif;font-weight:600;line-height:1;color:oklch(0.98 0.02 var(--wh));background:oklch(0.58 0.12 var(--wh));box-shadow:0 0 0 1.5px var(--card)}
.dk .wf{background:oklch(0.52 0.11 var(--wh))}
.wc-w{position:relative;display:inline}
.wc{display:inline-flex;align-items:center;gap:3px;height:18px;padding:0 6px;margin:0 2px;vertical-align:1px;border-radius:9px;background:var(--sunken);color:var(--subtle);font:500 11px/1 'Geist',sans-serif;text-decoration:none;transition:background 140ms,color 140ms}
.wc em{font-style:normal;color:var(--faint)}
.wc:hover,.wc-w:hover .wc{background:var(--fg);color:var(--card)}
.wc-w:hover .wc em{color:inherit;opacity:.7}
.wc.w{animation:badge 520ms var(--spring) both}
.wc-pop{position:absolute;left:50%;bottom:100%;transform:translateX(-50%);padding-bottom:8px;z-index:6;animation:fnp 200ms var(--settle) both}
.wc-in{display:flex;flex-direction:column;width:280px;padding:4px;border-radius:12px;background:var(--raised);box-shadow:var(--lift-hi);text-align:left;white-space:normal;line-height:1.4}
.wc-it{display:flex;flex-direction:column;gap:3px;padding:8px 10px;border-radius:8px;cursor:pointer}
.wc-it:hover{background:var(--hover)}
.wc-it+.wc-it{border-top:1px solid var(--b-sub)}
.wc-d{display:flex;align-items:center;gap:6px;font:11px var(--plex);color:var(--subtle)}
.wc-d i{margin-left:auto;font-style:normal;color:var(--faint)}
.wc-it b{font-size:12.5px;font-weight:500;color:var(--fg)}
.wc-s{font-size:12px;color:var(--muted)}
.wl{display:flex;flex-direction:column;gap:8px;padding:2px 0 4px}
.wl-h{display:flex;align-items:center;gap:7px;color:var(--muted);font-size:13px}
.wl-l{background:linear-gradient(90deg,var(--muted) 30%,var(--fg) 50%,var(--muted) 70%);background-size:200% 100%;-webkit-background-clip:text;background-clip:text;color:transparent;animation:wlsh 1.6s linear infinite}
@keyframes wlsh{from{background-position:100% 0}to{background-position:-100% 0}}
.wl-q{align-self:flex-start;padding:3px 9px;border-radius:6px;background:var(--sunken);font:11.5px var(--plex);color:var(--subtle)}
.wl-s{display:flex;flex-wrap:wrap;gap:6px}
.wl-it{display:inline-flex;align-items:center;gap:6px;height:24px;padding:0 9px 0 6px;border-radius:12px;box-shadow:0 0 0 1px var(--b);font-size:12px;color:var(--muted);animation:wlin 320ms var(--spring) both}
@keyframes wlin{from{opacity:0;transform:translateY(4px) scale(.94)}}
.ws{margin-top:10px}
.ws-b{display:inline-flex;align-items:center;gap:8px;height:26px;padding:0 10px 0 6px;border-radius:13px;color:var(--subtle);font-size:12px;transition:background 140ms,color 140ms}
.ws-b:hover,.ws.open .ws-b{background:var(--hover);color:var(--fg)}
.ws-b svg{transition:transform 200ms var(--ease)}
.ws.open .ws-b svg{transform:rotate(90deg)}
.ws-st{display:flex}.ws-st .wf+.wf{margin-left:-4px}
.ws-x{display:grid;grid-template-rows:0fr;transition:grid-template-rows 260ms var(--ease)}
.ws.open .ws-x{grid-template-rows:1fr}
.ws-x>div{overflow:hidden;min-height:0}
.ws-q{display:flex;align-items:center;gap:6px;margin:8px 0 2px 6px;font:11px var(--plex);color:var(--faint)}
.ws-l{list-style:none;margin:4px 0 0;padding:0;display:flex;flex-direction:column}
.ws-l a{display:grid;grid-template-columns:16px 14px minmax(0,1fr) auto;align-items:center;gap:8px;padding:6px;border-radius:8px;text-decoration:none;color:var(--fg)}
.ws-l a:hover{background:var(--hover)}
.ws-n{font:10.5px var(--plex);color:var(--faint);text-align:right}
.ws-t{font-size:12.5px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.ws-d{font:11px var(--plex);color:var(--faint);white-space:nowrap}
@media (prefers-reduced-motion:reduce){.wl-l{animation:none;color:var(--muted)}.wl-it{animation:none}}
```
