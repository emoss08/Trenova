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
