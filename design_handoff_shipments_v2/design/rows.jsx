const $$ = n => "$" + Math.round(n).toLocaleString();
const ct = c => c.split(",")[0];
const fmtH = h => { const d = h >= 24 ? "Oct 6" : "Oct 5"; const x = h % 24, hh = Math.floor(x), mm = Math.round((x - hh) * 60); return `${d}, ${String(hh).padStart(2, "0")}:${String(mm).padStart(2, "0")}`; };
const BILLING = { "S2610-0366": ["Transferred", "ok"], "S2610-0371": ["Ready to bill", "b"] };
const billOf = s => BILLING[s.id] || (s.st === "done" ? ["Ready to bill", "b"] : null);
const facility = (s, end) => end === "o" ? ({ "Des Moines": "Hy-Vee DC 4", "Omaha": "Cargill Protein Plant", "Kansas City": "AB Brewery KC", "Fort Worth": "Sample Warehouse", "San Antonio": "Sample Plant", "Atlanta": "Publix Lakeland Hub" }[ct(s.o[1])] || ct(s.o[1]) + " Terminal") : ({ "Chicago": "Hy-Vee Chicago Hub", "Minneapolis": "Cargill Cold Store", "Dallas": "Sample Dist. Center", "Houston": "Sample Retail Store" }[ct(s.d[1])] || s.cust.split(" ")[0] + " " + ct(s.d[1]));

const COLS = [
  { id: "lane", label: "Lane", lock: true, sort: s => s.o[1] },
  { id: "status", label: "Status", sort: s => ["delayed", "new", "transit", "assigned", "done"].indexOf(s.st) },
  { id: "tender", label: "Tender", sort: s => s.tender },
  { id: "billing", label: "Billing", sort: s => (billOf(s) || ["~"])[0] },
  { id: "pro", label: "PRO / BOL", sort: s => s.id },
  { id: "order", label: "Order", sort: s => s.order },
  { id: "cust", label: "Customer", sort: s => s.cust },
  { id: "cov", label: "Coverage", sort: s => s.drv ? s.drv.n : "~" },
  { id: "eta", label: "ETA", sort: s => (HOURS[s.id] || [0, 0])[1] },
  { id: "pickup", label: "Pickup appt", sort: s => (HOURS[s.id] || [0])[0] },
  { id: "delivery", label: "Delivery appt", sort: s => (HOURS[s.id] || [0, 0])[1] },
  { id: "rev", label: "Revenue", r: true, sort: s => s.rev },
  { id: "margin", label: "Margin", r: true, sort: s => s.mg },
];
const DEFAULT_HIDDEN = ["pickup", "delivery"];

function Cell({ id, s, onAssign }) {
  const togo = Math.round(s.mi * (1 - s.pct / 100));
  const H = HOURS[s.id] || [0, 0];
  switch (id) {
    case "lane": return <div className="lane"><div className="lane-r">{ct(s.o[1])}<Ic n="arrowR" s={11}></Ic>{ct(s.d[1])}<span className="bar in"><i className={s.st === "delayed" ? "d" : s.st === "done" ? "k" : ""} style={{ width: s.pct + "%" }}></i></span></div><div className="lane-m">{s.st === "transit" || s.st === "delayed" ? togo + " mi left" : s.mi + " mi"}</div></div>;
    case "status": return <span className={"st " + s.st}><i></i>{s.st === "assigned" && window.__ORG === "brokerage" ? "Booked" : ST[s.st]}</span>;
    case "tender": return <span className={"tn" + (s.tender === "Pending" || s.tender === "Sent" ? " p" : "")}>{s.tender === "Sent" ? "Sent · " + s.tenderTo.n.split(" ")[0] : s.tender}</span>;
    case "billing": { const b = billOf(s); return b ? <span className={"bl " + b[1]}>{b[0]}</span> : <span className="dash">—</span>; }
    case "pro": return <div className="two m"><b>{s.id}</b><span>{s.bol}</span></div>;
    case "order": return <span className="mono lk">{s.order}</span>;
    case "cust": return <div className="two"><b className="ul">{s.cust}</b><span>{s.wt.toLocaleString()} lb · {s.equip.replace(" 53′", "")}</span></div>;
    case "cov": { const cv = covOf(s, window.__ORG); if (!cv && s.tenderTo) return <div className="drv"><span className="av sq tnd" style={{ "--h": s.tenderTo.h }}>{s.tenderTo.i}</span><div className="drv-t"><b>{s.tenderTo.n}</b><span className="t-w">Tendered · awaiting</span></div></div>; if (!cv) return <span className="nodrv"><Ic n="warn" s={12}></Ic>Needs coverage</span>;
      if (cv.carrier) return <div className="drv"><span className="av sq" style={{ "--h": cv.carrier.h }}>{cv.carrier.i}</span><div className="drv-t"><b>{cv.carrier.n}</b><span>{cv.driver ? cv.driver.n : cv.carrier.unit}</span></div></div>;
      return <div className="drv"><span className="av" style={{ "--h": s.drv.h }}>{s.drv.i}</span><div className="drv-t"><b>{s.drv.n}</b><span>{s.drv.unit}<span className="hosv">{s.drv.hos !== "—" ? " · " + s.drv.hos + " HOS" : ""}</span></span></div></div>; }
    case "eta": return <div className="two m in"><b>{s.eta}</b><span className={s.late ? "late" : /early|on time/.test(s.delta || "") ? "ok" : ""}>{s.late ? s.delta : s.delta || ""}</span></div>;
    case "pickup": return <span className="mono">{fmtH(H[0])}</span>;
    case "delivery": return <span className="mono">{fmtH(H[1])}</span>;
    case "rev": return <div className="two m in" style={{ alignItems: "flex-end", justifyContent: "flex-end" }}><b>{$$(s.rev)}</b><span>${(s.rev / s.mi).toFixed(2)}/mi</span></div>;
    case "margin": return <span className={"mono mg " + (s.mg < 12 ? "lo" : s.mg > 22 ? "hi" : "")}>{s.mg}%</span>;
  }
  return null;
}

function Menu({ items, onClose, align = "right" }) {
  const ref = React.useRef();
  React.useEffect(() => { const h = e => { if (ref.current && !ref.current.contains(e.target)) onClose(); }; const k = e => e.key === "Escape" && onClose(); setTimeout(() => document.addEventListener("mousedown", h)); document.addEventListener("keydown", k); return () => { document.removeEventListener("mousedown", h); document.removeEventListener("keydown", k); }; }, []);
  return (
    <div className={"mn " + align} ref={ref} onClick={e => e.stopPropagation()}>
      {items.map((it, i) => it === "-" ? <span key={i} className="mn-sep"></span> : <button key={it[1]} className={"mn-i" + (it[2] ? " d" : "")} onClick={() => { it[3] && it[3](); onClose(); }}><Ic n={it[0]} s={13}></Ic><span>{it[1]}</span>{it[4] && <span className="kbd">{it[4]}</span>}</button>)}
    </div>
  );
}

function RowMenu({ s, onAct }) {
  const [open, setOpen] = React.useState(false);
  return (
    <div className="rmw">
      <button className={"ib rm" + (open ? " on" : "")} onClick={e => { e.stopPropagation(); setOpen(!open); }}><Ic n="more" s={14}></Ic></button>
      {open && <Menu onClose={() => setOpen(false)} items={[["edit", "Edit", false, () => onAct("Editing " + s.id), "E"], ["copy", "Duplicate", false, () => onAct(s.id + " duplicated")], ["link", "Copy link", false, () => onAct("Link copied"), MOD + " L"], ["swap", "Transfer ownership", false, () => onAct("Transfer " + s.id)], ["ext", "Open full record", false, () => onAct("Opening " + s.id)], "-", ["ban", "Cancel shipment", true, () => onAct(s.id + " cancelled")]]}></Menu>}
    </div>
  );
}

function ColMenu({ hidden, setHidden, onClose }) {
  const ref = React.useRef();
  React.useEffect(() => { const h = e => { if (ref.current && !ref.current.contains(e.target) && !e.target.closest(".cbtn")) onClose(); }; document.addEventListener("mousedown", h); return () => document.removeEventListener("mousedown", h); }, []);
  return (
    <div className="fm" ref={ref} style={{ width: 220, right: 0, left: "auto" }}>
      <div className="fm-h">Columns</div>
      {COLS.map(c => { const on = !hidden.includes(c.id); return <button key={c.id} className="fm-i" disabled={c.lock} onClick={() => setHidden(h => on ? [...h, c.id] : h.filter(x => x !== c.id))}><span className={"cbx" + (on ? " on" : "")} style={c.lock ? { opacity: .4 } : null}>{on && <Ic n="check" s={10} w={3}></Ic>}</span><span className="fm-l">{c.label}</span>{c.lock && <em>locked</em>}</button>; })}
      <div className="fm-f"><button className="btn ghost sm" onClick={() => setHidden(DEFAULT_HIDDEN)}>Reset</button><span className="sp"></span><button className="btn sm" onClick={onClose}>Done</button></div>
    </div>
  );
}

const MOD = /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent) ? "⌘" : "Ctrl";
const K = ({ k }) => <span className="kg">{k.map(x => <span key={x} className="kbd">{x}</span>)}</span>;

function Expand({ s, ai, onAssign, onNotify, onAct, onAsk, assignedNow, onCollapse }) {
  const H = HOURS[s.id] || [0, 0];
  const moving = s.st === "transit" || s.st === "delayed";
  const done = Math.round(s.mi * s.pct / 100), togo = s.mi - done;
  const lh = s.rev * 0.8, fuel = s.rev * 0.14, acc = s.rev - lh - fuel, cost = s.rev * (1 - s.mg / 100);
  const sug = SUGGEST[s.id];
  const winEnd = String(Math.floor(H[0] % 24) + 2).padStart(2, "0") + ":00";
  const docs = [["Rate con", "ok"], ["BOL", s.pct > 0 ? "ok" : "wait"], ["POD", s.st === "done" ? "ok" : "due"]];
  if (s.id === "S2610-0398") docs.push(["Detention", "warn"]);
  const tone = s.st === "delayed" ? "d" : s.st === "done" ? "k" : "";
  const org = window.__ORG || "asset";
  const cv = covOf(s, org);
  const csug = CARRIERS.filter((c, i) => (hashId(s.id) + i) % 4 === 0).slice(0, 2).map(c => ({ ...c, quote: Math.round(s.mi * c.rate / 10) * 10 }));
  let next;
  if (assignedNow && cv) next = <div className="nx ok"><span className="nx-k">{cv.carrier && !cv.driver ? "Tendered" : "Assigned"}</span><div className="nx-d"><span className={"av" + (cv.carrier ? " sq" : "")} style={{ "--h": (cv.carrier || cv.driver).h }}>{(cv.carrier || cv.driver).i}</span><div className="drv-t"><b>{(cv.carrier || cv.driver).n}</b><span>{cv.carrier ? "Rate con sent · awaiting acceptance" : s.drv.unit + " · dispatch sheet sent"}</span></div></div></div>;
  else if (!s.drv && (sug || csug.length)) next = (
    <div className="nx">
      {org !== "brokerage" && sug && <>
        <span className="nx-k">{ai ? "Suggested drivers" : "Nearest drivers"}</span>
        {sug.map((d, i) => <button key={d.n} className="nx-o" onClick={() => onAssign(s.id, d)}><span className="av" style={{ "--h": d.h }}>{d.i}</span><div className="drv-t"><b>{d.n}</b><span>{d.mi + " mi out"}<span className="hosv">{" · " + d.hos + " HOS"}</span></span></div>{ai ? <em className="fit">{i ? "81" : "96"}%</em> : null}<span className="nx-a">Assign</span></button>)}
      </>}
      {org !== "asset" && <>
        <span className="nx-k" style={org === "both" ? { marginTop: 10 } : null}>{org === "both" ? "Or tender to a carrier" : ai ? "Suggested carriers" : "Carriers on this lane"}</span>
        {csug.map(c => <button key={c.n} className="nx-o" onClick={() => onAssign(s.id, c)}><span className="av sq" style={{ "--h": c.h }}>{c.i}</span><div className="drv-t"><b>{c.n}</b><span>{"$" + c.quote.toLocaleString() + " · $" + c.rate.toFixed(2) + "/mi · " + c.acc + "% accept"}</span></div><span className="nx-a">Tender</span></button>)}
      </>}
    </div>);
  else if (s.late) next = (
    <div className="nx">
      <span className="nx-k d">{ai ? "Drafted update" : "Running late"}</span>
      <p className="nx-p">{ai ? `“${s.risk} has delayed ${s.id}. New ETA ${s.eta}. We'll confirm a new window within the hour.”` : `${s.delta} behind · ${s.risk}. The appointment will be missed.`}</p>
      <div className="nx-b"><button className={"btn sm " + (ai ? "teal" : "")} onClick={() => onNotify(s)}><Ic n="send" s={11}></Ic>{ai ? "Send to " + s.cust.split(" ")[0] : "Notify customer"}</button>{ai && <button className="btn ghost sm">Edit</button>}</div>
    </div>);
  else if (cv) next = (
    <div className="nx">
      <span className="nx-k">Coverage</span>
      {cv.carrier && <div className="nx-d"><span className="av sq" style={{ "--h": cv.carrier.h }}>{cv.carrier.i}</span><div className="drv-t"><b>{cv.carrier.n}</b><span>{cv.carrier.unit + " · " + cv.carrier.acc + "% on-time"}</span></div><button className="ib" title="Call carrier"><Ic n="chat" s={13}></Ic></button></div>}
      {cv.driver && <div className="nx-d"><span className="av" style={{ "--h": cv.driver.h }}>{cv.driver.i}</span><div className="drv-t"><b>{cv.driver.n}</b><span>{cv.driver.unit}</span></div>{!cv.carrier && <button className="ib" title="Message"><Ic n="chat" s={13}></Ic></button>}</div>}
      {!cv.carrier && cv.driver.hos !== "—" && <div className="hos hosv"><span className="bar"><i className={parseFloat(cv.driver.hos) < 4 ? "w" : "k"} style={{ width: Math.min(100, parseFloat(cv.driver.hos) / 11 * 100) + "%" }}></i></span><span className="mono">{cv.driver.hos} of 11:00 drive</span></div>}
    </div>);
  return (
    <div className="x2">
      <div className="x2-main">
        <div className="rt2">
          <div className="rt2-trk">
            <span className={"rt2-n o" + (s.pct > 0 ? " d" : "")}></span>
            <span className="rt2-line"><i className={tone} style={{ width: s.pct + "%" }}></i>{s.late && <b className="rt2-late" style={{ left: s.pct + "%" }}></b>}{moving && <span className={"rt2-tr " + tone} style={{ left: s.pct + "%" }}><Ic n="truck" s={11} w={2}></Ic></span>}</span>
            <span className={"rt2-n" + (s.st === "done" ? " d" : "")}></span>
          </div>
          <div className="rt2-ends">
            <div><span className="sk">Pickup</span><b>{facility(s, "o")}</b><span>{`${s.o[1]} · ${fmtH(H[0])}–${winEnd}`}</span><em className={s.pct > 0 ? "ok" : ""}>{s.pct > 0 ? "Departed 06:12" : "Scheduled"}</em></div>
            <div className="mid">{moving ? <><b>{s.late ? s.risk : "On schedule"}</b><span className="mono">{`${done} mi done · ${togo} to go · ping 3m`}</span></> : <span className="mono">{s.st === "done" ? s.mi + " mi · delivered" : s.mi + " mi · not started"}</span>}</div>
            <div className="r"><span className="sk">Delivery</span><b>{facility(s, "d")}</b><span>{`${s.d[1]} · appt ${fmtH(H[1])}`}</span><em className={s.late ? "d" : s.st === "done" ? "ok" : ""}>{s.st === "done" ? "POD signed " + s.eta : s.late ? "Will miss · " + s.delta : "ETA " + s.eta}</em></div>
          </div>
        </div>
        <div className="x2-row">
          <dl className="mny">
            <div className="hi"><dt>Revenue</dt><dd>{$$(s.rev)}<small>{`$${(s.rev / s.mi).toFixed(2)}/mi`}</small></dd></div>
            <div><dt>Linehaul</dt><dd>{$$(lh)}</dd></div>
            <div><dt>Fuel</dt><dd>{$$(fuel)}</dd></div>
            <div><dt>Accessorials</dt><dd>{$$(acc)}</dd></div>
            <div><dt>Est. cost</dt><dd>{$$(cost)}</dd></div>
            <div><dt>Margin</dt><dd className={"mg " + (s.mg < 12 ? "lo" : s.mg > 22 ? "hi" : "")}>{s.mg + "%"}</dd></div>
          </dl>
          <div className="dcs"><span className="dcs-l">Docs</span>{docs.map(([n, t]) => <span key={n} className={"dc " + t}>{t === "ok" ? <Ic n="check" s={10} w={2.6}></Ic> : t === "warn" ? <Ic n="clock" s={10} w={2.2}></Ic> : <i></i>}{n}</span>)}<button className="dc add"><Ic n="plus" s={10} w={2.2}></Ic>Upload</button></div>
        </div>
      </div>
      <div className="x2-side">
        {next}
        <div className="qa2">
          <span className="nx-k">Quick actions</span>
          <div className="qa2-l">
            <button onClick={() => onAct("Editing " + s.id)}><Ic n="edit" s={13}></Ic><span>Edit</span><K k={["E"]}></K></button>
            <button onClick={() => onAct(s.id + " duplicated")}><Ic n="copy" s={13}></Ic><span>Duplicate</span></button>
            <button onClick={() => onAct("Transfer " + s.id)}><Ic n="swap" s={13}></Ic><span>Transfer ownership</span></button>
            <button onClick={() => onAct("Link copied")}><Ic n="link" s={13}></Ic><span>Copy link</span><K k={[MOD, "L"]}></K></button>
            <button onClick={() => onAct("Comment on " + s.id)}><Ic n="chat" s={13}></Ic><span>Add comment</span></button>
            <button className="d" onClick={() => onAct(s.id + " cancelled")}><Ic n="ban" s={13}></Ic><span>Cancel shipment</span></button>
          </div>
        </div>
      </div>
      <button className="x2-c" onClick={onCollapse}><Ic n="chevD" s={12}></Ic>Collapse<span className="kbd">Esc</span></button>
    </div>
  );
}

function Shortcuts({ onClose }) {
  const groups = [
    ["Shipments table", [["Move between rows", ["J"], ["K"]], ["Expand or collapse row", ["Enter"]], ["Collapse / clear selection", ["Esc"]], ["Select row", ["X"]], ["Edit shipment", ["E"]], ["Copy PRO number", ["Alt", "C"]], ["Copy link", [MOD, "L"]], ["Search the table", ["/"]]]],
    ["General", [["Command palette", [MOD, "K"]], ["Toggle sidebar", [MOD, "B"]], ["Assistant", [MOD, "J"]], ["User settings", [MOD, "Shift", "S"]], ["Keyboard shortcuts", [MOD, "/"]]]],
  ];
  React.useEffect(() => { const k = e => e.key === "Escape" && onClose(); window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k); }, []);
  return (
    <div className="scm" onMouseDown={e => e.target === e.currentTarget && onClose()}>
      <div className="scm-in">
        <div className="scm-h"><b>Keyboard shortcuts</b><span>Available throughout the application.</span><button className="ib" onClick={onClose}><Ic n="x" s={13}></Ic></button></div>
        <div className="scm-b">{groups.map(([g, l]) => <div key={g}><div className="scm-g">{g}</div>{l.map(([lab, ...ks]) => <div key={lab} className="scm-r"><span>{lab}</span><span className="kg">{ks.map((k, i) => <React.Fragment key={i}>{i > 0 && <em>or</em>}{k.map(x => <span key={x} className="kbd">{x}</span>)}</React.Fragment>)}</span></div>)}</div>)}</div>
      </div>
    </div>
  );
}
Object.assign(window, { COLS, DEFAULT_HIDDEN, Cell, RowMenu, ColMenu, Expand, Menu, Shortcuts, MOD });
