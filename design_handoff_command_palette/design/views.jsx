function AIcon({ n, s = 15 }) {
  return n === "nova" ? <NovaMark s={s - 2}></NovaMark> : <Ic n={n} s={s}></Ic>;
}
function initials(name) { return name.split(" ").map((x) => x[0]).slice(0, 2).join(""); }
function Tile({ item, lg }) {
  const c = "tl" + (lg ? " lg" : "");
  const s = lg ? 17 : 15;
  if (item.kind === "record") return <span className={c + " t-" + item.type}>{item.type === "worker" ? initials(item.title) : <Ic n={ENT[item.type].icon} s={s}></Ic>}</span>;
  if (item.kind === "page") return <span className={c + " t-neutral"}><Ic n={item.icon} s={s}></Ic></span>;
  if (item.kind === "command") return <span className={c + (item.group === "create" || item.group === "nova" ? " t-brand" : item.danger ? " t-danger" : " t-neutral")}><Ic n={item.icon} s={s}></Ic></span>;
  if (item.kind === "attention") return <span className={c + " t-" + item.tone}><Ic n="alert" s={s}></Ic></span>;
  return <span className={c + " t-nova"}><NovaMark s={lg ? 18 : 15} busy={item.busy}></NovaMark></span>;
}
function Pill({ tone, children }) { return <span className={"pill " + tone}><i></i>{children}</span>; }
function ShipPill({ s }) { const [l, t] = SHIP_STATUS[s.status]; return <Pill tone={t}>{l}</Pill>; }
function Sep() { return <i>·</i>; }

function Row({ item, m, on, time, onPick, onHover }) {
  const hl = (m && m.hl) || {};
  let title = null, sub = null, end = null;
  if (item.kind === "record") {
    const d = item.d;
    title = <span><HL text={item.title} r={hl.title}></HL></span>;
    if (item.type === "shipment") {
      sub = <><span className="k"><HL text={d.cust} r={hl.cust}></HL></span><Sep></Sep><span><HL text={`${d.o} → ${d.d}`} r={hl.route}></HL></span>{(hl.bol) && <><Sep></Sep><span className="k"><HL text={"BOL " + d.bol} r={hl.bol}></HL></span></>}</>;
      end = <ShipPill s={d}></ShipPill>;
    } else if (item.type === "customer") {
      sub = <><span className="k mono"><HL text={d.id} r={hl.code}></HL></span><Sep></Sep><span><HL text={d.city} r={hl.city}></HL></span><Sep></Sep><span className="k">{money(d.bal)} open</span></>;
      end = d.status !== "Active" && <Pill tone="warn">{d.status}</Pill>;
    } else if (item.type === "worker") {
      sub = <><span className="k mono"><HL text={d.id} r={hl.code}></HL></span><Sep></Sep><span><HL text={d.role} r={hl.role}></HL></span>{d.load && <><Sep></Sep><span className="k mono">{d.load}</span></>}</>;
      end = <Pill tone={d.status[1]}>{d.status[0]}</Pill>;
    } else {
      sub = <><span className="k"><HL text={d.kind} r={hl.dkind}></HL></span><Sep></Sep><span className="mono"><HL text={d.on} r={hl.on}></HL></span><Sep></Sep><span>{d.when}</span></>;
    }
  } else if (item.kind === "page") {
    title = <span><HL text={item.title} r={hl.title}></HL></span>;
    sub = item.trail.length ? <span><HL text={item.trail.join(" › ")} r={hl.trail}></HL></span> : <span>Workspace</span>;
    end = item.pinned && <Ic n="pin" s={13}></Ic>;
  } else if (item.kind === "command") {
    title = <span><HL text={item.label} r={hl.title}></HL></span>;
    sub = <span><HL text={item.desc} r={hl.desc}></HL></span>;
    end = <>{item.checked && <span style={{ color: "var(--brand-fg)", display: "flex" }}><Ic n="check" s={15}></Ic></span>}{item.keys && <Keys k={item.keys}></Keys>}</>;
  } else if (item.kind === "attention") {
    title = <span>{item.label}</span>;
    sub = <span>{item.module}</span>;
    end = <span className={"cnt " + item.tone}>{item.count}</span>;
  } else if (item.kind === "ask") {
    title = <span>{item.subject ? item.q : "Ask Nova"}</span>;
    sub = <span>{item.subject ? "About " + item.subject.title : <>“{item.q}”</>}</span>;
  } else if (item.kind === "more") {
    return <div className={"rw one" + (on ? " on" : "")} data-k={item.key} onClick={() => onPick(item)} onPointerMove={() => onHover(item)}><span className="tl t-neutral"><Ic n="arrowR" s={14}></Ic></span><span className="rt"><b><span>{item.label}</span></b></span><span className="end go"><Keys k={["↵"]}></Keys></span></div>;
  }
  const via = m && m.via;
  return (
    <div className={"rw" + (on ? " on" : "") + (item.danger ? " dng" : "")} data-k={item.key} onClick={() => onPick(item)} onPointerMove={() => onHover(item)}>
      <Tile item={item}></Tile>
      <span className="rt"><b>{title}</b>{sub && <small>{sub}</small>}</span>
      <span className="end">
        {via && <span className="via">{via.label} <b><HL text={via.text} r={via.r}></HL></b></span>}
        {time && <span className="mono" style={{ fontSize: 11.5, color: "var(--faint)" }}>{time}</span>}
        {end}
        {!(item.kind === "command" && item.keys) && <span className="go"><Keys k={["↵"]}></Keys></span>}
      </span>
    </div>
  );
}

// ——— Preview: Desk artifact vocabulary ———
function Fields({ rows }) {
  return <dl className="ar-f">{rows.filter(Boolean).map(([k, val, cls]) => <div key={k}><dt>{k}</dt><dd className={cls || ""}>{val}</dd></div>)}</dl>;
}
function Sec({ label, aside, children }) {
  return <section className="ar-s"><h4>{label}{aside && <span>{aside}</span>}</h4>{children}</section>;
}
function Prov({ kind, bits = [] }) {
  return <div className="ar-prov"><code>{kind}</code>{bits.filter(Boolean).map((b, i) => <span key={i}>{b}</span>)}</div>;
}
function BigPill({ tone, children }) { return <span className="ar-pill"><i className={tone}></i>{children}</span>; }
function Head({ title, mono, sub, pill }) {
  return <div className="ar-h"><div><div className={"ar-big" + (mono ? " mono" : "")}>{title}</div>{sub && <div className="ar-sub">{sub}</div>}</div>{pill}</div>;
}
function Stats({ items }) {
  return <div className="ar-stats">{items.map(([l, val, t]) => <div key={l}><span>{l}</span><b className={t || ""}>{val}</b></div>)}</div>;
}
function RelRow({ rec, run, right }) {
  if (!rec) return null;
  const d = rec.d;
  const sub = rec.type === "shipment" ? `${d.o} → ${d.d}` : rec.type === "worker" ? d.role : rec.type === "customer" ? d.city : d.kind;
  return <button className="ar-rel" onMouseDown={(e) => e.preventDefault()} onClick={() => run({ t: "nav", rec })}>
    <span className="ar-ri"><Ic n={ENT[rec.type].icon} s={13}></Ic></span>
    <span className="ar-rt"><b className={rec.type === "shipment" || rec.type === "document" ? "mono" : ""}>{rec.title}</b><em>{sub}</em></span>
    {right !== undefined ? <span className="ar-rr">{right}</span> : rec.type === "shipment" && <span className="ar-rr">{SHIP_STATUS[d.status][0]}</span>}
    <Ic n="chevR" s={12}></Ic>
  </button>;
}
function Lnk({ icon, label, hint, onClick }) {
  return <button className="ar-rel" onMouseDown={(e) => e.preventDefault()} onClick={onClick}><span className="ar-ri"><AIcon n={icon} s={13}></AIcon></span><span className="ar-rt"><b>{label}</b></span>{hint && <span className="ar-rr">{hint}</span>}<Ic n="chevR" s={12}></Ic></button>;
}
const toneDot = (t) => (t === "warn" ? "warn" : t === "success" ? "ok" : t === "danger" ? "bad" : "");

function ShipmentPv({ r, run }) {
  const d = r.d, pct = Math.round(d.done * 100), [sl, st] = SHIP_STATUS[d.status];
  const drv = d.driver && RECORDS.find((x) => x.title === d.driver);
  return <>
    <Prov kind="shipment" bits={[d.done > 0 && d.done < 1 ? "Live" : null, d.ev[0][1].includes(":") ? "Updated " + d.ev[0][1] : null]}></Prov>
    <Head title={d.id} mono sub={`${d.cust} · BOL ${d.bol}`} pill={<BigPill tone={toneDot(st)}>{sl}</BigPill>}></Head>
    <div className="ar-route">
      <div className="ar-stop"><i></i><b>{d.o}</b><em>{d.pu}</em></div>
      <div className="ar-line"><span className="ar-done" style={{ width: pct + "%" }}></span>{d.done > 0 && d.done < 1 && <span className="ar-trk" style={{ left: pct + "%" }}><Ic n="truck" s={12}></Ic></span>}</div>
      <div className="ar-stop end"><i></i><b>{d.d}</b><em>{d.del}</em></div>
      <div className="ar-rfoot"><span>{d.miles} mi{d.done > 0 && d.done < 1 ? ` · ${Math.round(d.miles * (1 - d.done))} to go` : ""}</span><span className={d.tone === "warn" ? "late" : ""}>{d.tone === "warn" && <i></i>}{d.late}{d.eta !== "—" && !d.eta.startsWith("Delivered") ? " · ETA " + d.eta.replace("Today ", "") : ""}</span></div>
    </div>
    {d.note && <p className="ar-note">{d.note}.</p>}
    <Fields rows={[
      ["Driver", drv ? <a href="#" onClick={(e) => { e.preventDefault(); run({ t: "nav", rec: drv }); }}>{d.driver}</a> : "Unassigned", drv ? "" : "warn"],
      ["Total", <span className="mono">{money(d.total)}</span>],
      ["Tractor", <span className="mono">{d.tractor || "—"}</span>],
      ["Trailer", <span className="mono">{d.trailer || "—"}</span>],
    ]}></Fields>
    <Sec label="Latest"><ul className="ar-tl">{d.ev.map((e, i) => <li key={i}><span>{e[0]}</span><em>{e[1]}</em></li>)}</ul></Sec>
  </>;
}
function CustomerPv({ r, run }) {
  const d = r.d, tot = d.aging.reduce((a, b) => a + b, 0) || 1, past = d.aging[1] + d.aging[2] + d.aging[3];
  const loads = RECORDS.filter((x) => x.type === "shipment" && x.d.code === d.id && x.d.status !== "invoiced");
  const shades = ["var(--b-strong)", "var(--faint)", "var(--muted)", "var(--danger)"];
  return <>
    <Prov kind="customer" bits={[d.terms, d.open + " open " + (d.open === 1 ? "load" : "loads")]}></Prov>
    <Head title={d.name} sub={`${d.id} · ${d.city}`} pill={<BigPill tone={d.status === "Active" ? "ok" : "warn"}>{d.status}</BigPill>}></Head>
    <Stats items={[["Open balance", money(d.bal)], ["Past terms", past ? money(past) : "—", d.aging[3] ? "bad" : ""], ["Credit used", Math.round(d.credit * 100) + "%", d.credit > 0.8 ? "warn" : ""]]}></Stats>
    <Sec label="Aging" aside={"of " + money(d.bal)}>
      <div className="ar-ag">{d.aging.map((x, i) => x > 0 && <i key={i} style={{ flex: x / tot, background: shades[i] }}></i>)}</div>
      <div className="ar-agl">{["Current", "31–60", "61–90", "90+"].map((l, i) => <span key={l}><em>{l}</em><b>{d.aging[i] ? "$" + (d.aging[i] / 1000).toFixed(1) + "k" : "—"}</b></span>)}</div>
    </Sec>
    {loads.length > 0 && <Sec label="Open loads" aside={loads.length}>{loads.map((x) => <RelRow key={x.key} rec={x} run={run}></RelRow>)}</Sec>}
    <Fields rows={[["Contact", d.contact], ["Credit limit", <span className="mono">{d.limit}</span>]]}></Fields>
  </>;
}
function WorkerPv({ r, run }) {
  const d = r.d;
  return <>
    <Prov kind="worker" bits={["ELD · 2 min ago"]}></Prov>
    <Head title={d.name} sub={`${d.id} · ${d.role}`} pill={<BigPill tone={toneDot(d.status[1])}>{d.status[0]}</BigPill>}></Head>
    <div className="ar-hos">{d.hos.map(([l, val, p, w]) => <div key={l}><span>{l}</span><b className={w ? "warn" : ""}>{val.replace(" left", "")}</b><i><u className={w ? "warn" : ""} style={{ width: Math.round(p * 100) + "%" }}></u></i></div>)}</div>
    {d.load && <Sec label="On load"><RelRow rec={recordById(d.load)} run={run}></RelRow></Sec>}
    <Fields rows={[["Home terminal", d.home], ["CDL", "Through " + d.cdl], ["Medical card", (d.medWarn ? "Expires " : "Through ") + d.med, d.medWarn ? "warn" : ""]]}></Fields>
  </>;
}
function DocumentPv({ r, run }) {
  const d = r.d;
  return <>
    <Prov kind="document" bits={[d.kind, d.conf + "% read"]}></Prov>
    <Head title={d.name} mono sub={`${d.kind} · ${d.when}`}></Head>
    <div className="ar-thumb"><span>{d.kind.toLowerCase()} · page 1</span></div>
    <Sec label="Read by Nova" aside={d.conf + "% confidence"}><Fields rows={d.fields}></Fields></Sec>
    {recordById(d.on) && <Sec label="Attached to"><RelRow rec={recordById(d.on)} run={run}></RelRow></Sec>}
    <Fields rows={[["Came in through", d.by]]}></Fields>
  </>;
}
function PagePv({ p, run, ctx }) {
  const rel = (p.related || []).map((id) => ctx.commands.find((c) => c.id === id)).filter(Boolean);
  const recs = p.types ? ctx.recents.map(([k]) => RECORD_BY_KEY[k]).filter((x) => x && x.type === p.types).slice(0, 3) : [];
  return <>
    <Prov kind="page" bits={[p.visited ? "Visited " + p.visited : null]}></Prov>
    <Head title={p.title} sub={p.trail.length ? p.trail.join(" / ") : "Workspace"} pill={ctx.pinned.includes(p.id) && <BigPill>Pinned</BigPill>}></Head>
    {p.glance && <Stats items={p.glance.map(([l, val, t]) => [l, val, t === "warn" ? "warn" : ""])}></Stats>}
    {rel.length > 0 && <Sec label="Start something here">{rel.map((c) => <Lnk key={c.id} icon={c.icon} label={c.label} hint={c.keys && <Keys k={c.keys}></Keys>} onClick={() => run({ t: "cmd", cmd: c })}></Lnk>)}</Sec>}
    {recs.length > 0 && <Sec label="You were just in">{recs.map((x) => <RelRow key={x.key} rec={x} run={run} right={ctx.recents.find((y) => y[0] === x.key)[1] + " ago"}></RelRow>)}</Sec>}
    <Fields rows={[["Address", <span className="mono">{p.href}</span>]]}></Fields>
  </>;
}
function CommandPv({ c }) {
  return <>
    <Prov kind="command" bits={[CMD_GROUPS[c.group]]}></Prov>
    <Head title={c.label} sub={c.desc} pill={c.checked && <BigPill tone="ok">Current</BigPill>}></Head>
    {c.keys && <Fields rows={[["Shortcut", <Keys k={c.keys}></Keys>]]}></Fields>}
  </>;
}
function AttentionPv({ a, run }) {
  return <>
    <Prov kind="queue" bits={[a.module, "Refreshes every minute"]}></Prov>
    <Head title={a.label} sub={a.why}></Head>
    <Stats items={[["Waiting on you", a.count, a.tone === "warn" ? "warn" : ""]]}></Stats>
    <Sec label="Includes">{a.sample.map((id) => <RelRow key={id} rec={recordById(id)} run={run} right={recordById(id).type === "shipment" ? recordById(id).d.late : recordById(id).d.medWarn ? "Med card " + recordById(id).d.med : ""}></RelRow>)}</Sec>
  </>;
}
function AskPv({ a }) {
  return <>
    <Prov kind="nova" bits={["Answers from your records"]}></Prov>
    <Head title="Ask Nova" sub={a.subject ? "About " + a.subject.title : null}></Head>
    <blockquote className="ar-q">{a.q}</blockquote>
    <p className="ar-note plain">The answer streams in above your results, with every record it used linked. Hover one to look at it here; continue in the Desk to keep going.</p>
  </>;
}
function SourcesPv({ refs, run }) {
  return <>
    <Prov kind="nova" bits={[refs.length + " records used"]}></Prov>
    <Head title="What Nova read" sub="Every record linked in the answer"></Head>
    <Sec label="Sources">{refs.map((x) => <RelRow key={x.key} rec={x} run={run}></RelRow>)}</Sec>
  </>;
}
function TipsPv() {
  return <div className="ar-tips">
    <h4>Get around faster</h4>
    <div><Keys k={["@"]}></Keys><span>Narrow to one kind — @shipment, @customer, @worker, @doc</span></div>
    <div><Keys k={[">"]}></Keys><span>Start with &gt; to run a command</span></div>
    <div><Keys k={["?"]}></Keys><span>End with a question mark to ask Nova</span></div>
    <div><Keys k={[MOD, "K"]}></Keys><span>Everything you can do with the selected row</span></div>
  </div>;
}

function PreviewPane({ item, actions, run, onMore, ctx, sources }) {
  let body = sources && sources.length ? <SourcesPv refs={sources} run={run}></SourcesPv> : <TipsPv></TipsPv>;
  if (item) {
    if (item.kind === "record") body = { shipment: <ShipmentPv r={item} run={run}></ShipmentPv>, customer: <CustomerPv r={item} run={run}></CustomerPv>, worker: <WorkerPv r={item} run={run}></WorkerPv>, document: <DocumentPv r={item} run={run}></DocumentPv> }[item.type];
    else if (item.kind === "page") body = <PagePv p={item} run={run} ctx={ctx}></PagePv>;
    else if (item.kind === "command") body = <CommandPv c={item}></CommandPv>;
    else if (item.kind === "attention") body = <AttentionPv a={item} run={run}></AttentionPv>;
    else if (item.kind === "ask") body = <AskPv a={item}></AskPv>;
  }
  const pri = actions[0], sec = actions.find((a) => a.id === "ask"), link = actions.find((a) => a.id === "link"), tab = actions.find((a) => a.id === "tab");
  const md = (e) => e.preventDefault();
  return (
    <aside className="pv ar" aria-label="Preview">
      <div className="ar-body" key={item ? item.key : sources && sources.length ? "src" : "tips"}>{body}</div>
      {pri && <div className="ar-bar">
        <button className="ar-btn ink" onMouseDown={md} onClick={() => run(pri.intent)}>{pri.label}<span className="kbd inv">↵</span></button>
        {sec && <button className="ar-btn" onMouseDown={md} onClick={() => run(sec.intent)}><NovaMark s={12}></NovaMark>Ask</button>}
        <span className="sp"></span>
        {link && <button className="ar-ib" title={"Copy link  " + link.keys.join(" ")} onMouseDown={md} onClick={() => run(link.intent)}><Ic n="link" s={14}></Ic></button>}
        {tab && <button className="ar-ib" title={"Open in new tab  " + tab.keys.join(" ")} onMouseDown={md} onClick={() => run(tab.intent)}><Ic n="upRight" s={14}></Ic></button>}
        {actions.length > 1 && <button className="ar-ib" title={"All actions  " + MOD + " K"} onMouseDown={md} onClick={onMore}><Ic n="more" s={14}></Ic></button>}
      </div>}
    </aside>
  );
}

Object.assign(window, { AIcon, Tile, Pill, Row, PreviewPane });
