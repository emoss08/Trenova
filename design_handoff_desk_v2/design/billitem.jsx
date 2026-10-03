// Billing queue items — one artifact ("bqi") that shows whichever queue row is selected.
const BI_PATHS = {
  check: <path d="M5 12.5l4.5 4.5L19 7"></path>,
  warn: <><path d="M12 4l9 15.5H3z"></path><path d="M12 10v4M12 17v.01"></path></>,
  x: <path d="M7 7l10 10M17 7L7 17"></path>,
  file: <><path d="M6 3.5h8l4 4v13H6z"></path><path d="M14 3.5v4h4"></path></>,
  pause: <path d="M9 6v12M15 6v12"></path>,
  truck: <><path d="M3.5 6.5h10v9h-10zM13.5 9.5h4l3 3v3h-7"></path><circle cx="7" cy="17" r="1.6"></circle><circle cx="17" cy="17" r="1.6"></circle></>,
  ext: <path d="M14 4.5h5.5V10M19.5 4.5L11 13M17 14v4.5a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V8a1 1 0 0 1 1-1h4.5"></path>,
  down: <path d="M6 9l6 6 6-6"></path>,
  up: <path d="M6 15l6-6 6 6"></path>,
  undo: <path d="M9 7L4.5 11.5 9 16M5 11.5h9a5 5 0 0 1 0 10h-2"></path>,
};
function BIc({ n, s = 13, w = 1.9 }) {
  return <svg width={s} height={s} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={w} strokeLinecap="round" strokeLinejoin="round">{BI_PATHS[n]}</svg>;
}
const bi$ = n => (n < 0 ? "−$" : "$") + Math.abs(n).toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
const r2 = n => Math.round(n * 100) / 100;

const BI_CUST = {
  "Acme Manufacturing": ["Acme AP · ap@acme.com", 30], "Peak Distributing": ["Peak Payables · ar@peakdist.com", 15],
  "Northline Foods": ["Northline AP · invoices@northline.com", 30], "Harbor Supply Co.": ["Harbor Finance · ap@harborsupply.com", 45],
  "Granite Building": ["Granite AP · billing@granitebuild.com", 30], "Bluewater Retail": ["Bluewater AP · ap@bluewater.com", 21],
};
const BI_LANES = [["Chicago, IL", "Columbus, OH", 357], ["Des Moines, IA", "Omaha, NE", 136], ["Joliet, IL", "Indianapolis, IN", 168], ["Gary, IN", "Detroit, MI", 268], ["Cedar Rapids, IA", "Madison, WI", 196], ["St. Louis, MO", "Memphis, TN", 284], ["Ames, IA", "Minneapolis, MN", 225], ["Milwaukee, WI", "Rockford, IL", 92]];
const BI_DRV = ["Dana Ortiz", "Marcus Hill", "Lena Park", "Tom Reyes", "Ana Cruz", "Will Grant"];
const BI_DAYS = ["Sep 28", "Sep 29", "Sep 30", "Oct 1", "Oct 2"];
const BI_STEPS = ["Ready for review", "Approved", "Posted"];
const BI_HOLDS = ["Waiting on paperwork", "Customer dispute", "Rate question"];
const BI_BILLERS = ["Avery Lane", "Jordan Pike", "Sam Okafor"];

function biBuild(row, i) {
  const [billTo, net] = BI_CUST[row.cust];
  const [from, to, mi] = BI_LANES[i % BI_LANES.length];
  const day = i === 0 ? "Oct 1" : BI_DAYS[(i * 3) % 5];
  const due = new Date(2026, 9, 3 + net).toLocaleDateString("en-US", { month: "short", day: "numeric" });
  const n = row.id.slice(3);
  const issue = i === 0 ? "lump" : i % 4 === 1 ? "pod" : i % 4 === 3 ? "det" : null;
  let charges;
  if (i === 0) charges = [
    { k: "lh", l: "Linehaul", b: "357 mi × $5.45", rc: 1945.65, amt: 1945.65 },
    { k: "fsc", l: "Fuel surcharge", b: "DOE $3.79 · Acme FSC table", rc: 151.13, amt: 151.13 },
    { k: "det", l: "Detention", b: "2 h × $65 after 2 h free", rc: 130, amt: 130 },
    { k: "stop", l: "Stop-off", b: "Acme DC 2, Gary IN", rc: 75, amt: 75 },
    { k: "acc", l: "Accessorial · liftgate", b: "Per rate con", rc: 738.22, amt: 738.22 },
    { k: "lump", l: "Lumper fee", b: "Paid by driver · receipt #4471", rc: null, amt: 110 },
  ];
  else {
    const lh = r2(row.amt * 0.72), fsc = r2(row.amt * 0.14), det = issue === "det" ? 97.5 : 0;
    charges = [
      { k: "lh", l: "Linehaul", b: mi + " mi × $" + (lh / mi).toFixed(2), rc: lh, amt: lh },
      { k: "fsc", l: "Fuel surcharge", b: "DOE $3.79 · contract table", rc: fsc, amt: fsc },
      ...(det ? [{ k: "det", l: "Detention", b: "3.5 h logged × $65 after 2 h free", rc: det, amt: det }] : []),
      { k: "acc", l: "Accessorials", b: "Stop-off, liftgate · per rate con", rc: r2(row.amt - lh - fsc - det), amt: r2(row.amt - lh - fsc - det) },
    ];
  }
  const short = row.cust.split(" ")[0];
  const ISSUES = {
    lump: { k: "charges", x: "$110.00 lumper fee isn't on the rate con", why: "Acme's contract (§5.3) passes lumper costs through when there's a receipt. Billing Specialist found receipt #4471 on the load.", flag: "lump",
      opts: [{ k: "keep", l: "Keep it · bill $110.00", done: "Lumper fee kept · receipt attached" }, { k: "drop", l: "Remove it", done: "Lumper fee removed", drop: "lump" }] },
    pod: { k: "pod", x: "POD isn't signed — the receiver's line is blank", why: short + "'s terms need a signed POD before invoicing. " + BI_DRV[i % 6] + " can re-send it from the driver app.",
      opts: [{ k: "req", l: "Ask the driver for it", done: "Signed POD received from " + BI_DRV[i % 6] }, { k: "ok", l: "Bill without it", done: "Billing without a signed POD" }] },
    det: { k: "charges", x: "Detention doesn't match the ELD", why: "The driver logged 3.5 h at the dock, but the ELD shows 2 h 30 m — that's 0.5 h billable, not 1.5 h.", flag: "det",
      opts: [{ k: "eld", l: "Bill ELD time · $32.50", done: "Detention set to ELD time", set: ["det", 32.5, "0.5 h × $65 · from ELD"] }, { k: "drop", l: "Remove detention", done: "Detention removed", drop: "det" }] },
  };
  return {
    id: row.id, i, draft: "INV-D-" + n, inv: "INV-" + n, shp: row.shp, ref: "PO " + (77544 + i * 13), cust: row.cust, billTo, terms: "Net " + net, due,
    queued: day + ", " + (1 + (i * 5) % 11) + ":" + String((i * 17) % 60).padStart(2, "0") + " PM", age: (i === 0 ? 2 : 1 + (5 - BI_DAYS.indexOf(day))) + " days in queue",
    lane: { from, to, delivered: "Delivered " + day, driver: BI_DRV[i % 6], miles: mi + " mi" },
    charges, issue: issue && ISSUES[issue],
    docs: [["POD", issue === "pod" ? "Unsigned" : "Signed"], ["BOL", "1 page"], ["Rate con", short + " " + (77544 + i * 13)], ...(i === 0 ? [["Lumper receipt", "$110.00"]] : [])],
  };
}

/* ---------- store ---------- */
const BQ = { sel: "BQ-24101", items: {}, st: {}, ctx: {} };
QUEUE.forEach((r, i) => {
  const it = biBuild(r, i);
  BQ.items[r.id] = it;
  BQ.st[r.id] = { biller: r.biller, res: null, st: r.status === "Approved" ? 1 : 0, hold: null,
    log: [{ t: it.queued, x: "Queued when " + r.shp + " delivered", who: "System" }, ...(it.issue ? [{ t: "10:36 PM", x: "Flagged: " + it.issue.x.replace(/ —.*/, ""), who: "Billing Specialist" }] : [])] };
});
const bqEmit = () => window.dispatchEvent(new Event("desk:bq"));
function bqGet(id) {
  const s = BQ.st[id], it = BQ.items[id];
  const biller = s.biller || (BQ.ctx.assigned ? "Avery Lane" : null);
  const st = BQ.ctx.posted ? 2 : s.st;
  const opt = it.issue && s.res ? it.issue.opts.find(o => o.k === s.res) : null;
  const lines = it.charges.map(c => opt && opt.drop === c.k ? { ...c, off: true } : opt && opt.set && opt.set[0] === c.k ? { ...c, amt: opt.set[1], b: opt.set[2], adj: true } : c);
  const total = r2(lines.reduce((t, c) => t + (c.off ? 0 : c.amt), 0));
  const needs = (biller ? 0 : 1) + (it.issue && !s.res ? 1 : 0);
  return { ...s, biller, st, opt, lines, total, needs, ready: st === 0 && !s.hold && needs === 0 };
}
function bqSet(id, patch, logX) {
  const s = BQ.st[id];
  Object.assign(s, typeof patch === "function" ? patch(s) : patch);
  if (logX) s.log = [...s.log, { t: "Now", x: logX, who: "You" }];
  bqEmit();
}
function bqOpen(id) { BQ.sel = id; bqEmit(); }
function bqApproveMany(ids) { ids.forEach(id => { const g = bqGet(id); if (g.ready) bqSet(id, { st: 1 }, "Approved the invoice for " + bi$(g.total)); }); }
function bqUnapprove(ids) { ids.forEach(id => bqSet(id, { st: 0 }, "Undid the approval")); }
function useBQ() { const [, f] = React.useState(0); React.useEffect(() => { const h = () => f(x => x + 1); window.addEventListener("desk:bq", h); return () => window.removeEventListener("desk:bq", h); }, []); }

Object.defineProperty(ART2.bqi, "title", { get: () => BQ.sel + " · " + BQ.items[BQ.sel].cust.split(" ")[0] });
Object.defineProperty(ART2.bqi, "slug", { get: () => BQ.sel.toLowerCase() });
Object.defineProperty(ART2.bqi, "preview", { get: () => { const g = bqGet(BQ.sel); return bi$(g.total) + (g.needs ? " · " + g.needs + " need you" : " · " + BI_STEPS[g.st]); } });

/* ---------- body ---------- */
function BillBody({ a, ctx }) {
  useBQ();
  BQ.ctx = { assigned: ctx.assigned, posted: ctx.posted };
  return <BillItem key={BQ.sel} id={BQ.sel} />;
}

function BillItem({ id }) {
  const d = BQ.items[id], g = bqGet(id);
  const ids = Object.keys(BQ.items), at = ids.indexOf(id);
  const [pick, setPick] = React.useState(false);
  const [holdOpen, setHoldOpen] = React.useState(false);
  const rcTotal = r2(d.charges.reduce((s, c) => s + (c.rc || 0), 0));
  const iss = d.issue;
  const STD = {
    charges: ["Charges match the rate con", "Every line is on the rate con"],
    pod: ["Proof of delivery", "Signed · " + d.lane.delivered.replace("Delivered ", "")],
    terms: ["Bill-to and terms", d.billTo.split(" · ")[0] + " · " + d.terms],
    dup: ["Not a duplicate", "No other invoice for " + d.shp],
  };
  const checks = [
    g.biller ? { k: "biller", s: "ok", l: "Biller", x: g.biller } : { k: "biller", s: "bad", l: "Biller", x: "Nobody is assigned", act: "biller" },
    ...Object.keys(STD).map(k => iss && iss.k === k ? (g.res ? { k, s: "ok", l: STD[k][0], x: g.opt.done } : { k, s: "warn", l: STD[k][0], x: iss.x, act: "issue" }) : { k, s: "ok", l: STD[k][0], x: STD[k][1] }),
  ];
  const blocker = !g.biller ? "Assign a biller first" : iss && !g.res ? "Settle the flagged check first" : null;
  const assign = n => { setPick(false); bqSet(id, { biller: n }, "Assigned " + n + " as biller"); };

  return (
    <div className={"bi" + (g.st === 2 ? " posted" : "") + (g.hold ? " held" : "")}>
      <div className="bi-head">
        <div className="bi-kick">
          <span>{d.id}</span><i></i><span>{g.st === 2 ? d.inv : d.draft}</span><i></i><span>{d.ref}</span>
          <span className="bi-nav">
            <button title="Previous item" disabled={at === 0} onClick={() => bqOpen(ids[at - 1])}><BIc n="up" s={12} w={2.2} /></button>
            <em>{at + 1} of {ids.length}</em>
            <button title="Next item" disabled={at === ids.length - 1} onClick={() => bqOpen(ids[at + 1])}><BIc n="down" s={12} w={2.2} /></button>
          </span>
        </div>
        <div className="bi-hrow">
          <div className="bi-who"><h2>{d.cust}</h2><span>{d.billTo}</span></div>
          <div className="bi-amt"><b key={g.total}>{bi$(g.total)}</b><span>{d.terms} · due {d.due}</span></div>
        </div>
        <div className="bi-meta">
          {g.hold ? <span className="bi-st hold"><BIc n="pause" s={11} w={2.6} />On hold · {g.hold}</span> : <span className={"bi-st s" + g.st}>{g.st > 0 ? <BIc n="check" s={11} w={2.8} /> : <i></i>}{BI_STEPS[g.st]}</span>}
          <span className="bi-q"><span>{"Queued " + d.queued}</span><b aria-hidden="true">·</b><span>{d.age}</span></span>
          {g.hold && <button className="bi-rel" onClick={() => bqSet(id, { hold: null }, "Released the hold")}>Release</button>}
        </div>
      </div>

      {g.st === 0 && (
        <section className="bi-sec">
          <div className="bi-sh"><h3>Before it can be approved</h3><span className={g.needs ? "warn" : "ok"}>{g.needs ? g.needs + " of 5 need you" : "All clear"}</span></div>
          <div className="bi-checks">
            {checks.map(c => (
              <div key={c.k} className={"bi-ck " + c.s}>
                <span className="bi-ci"><BIc n={c.s === "ok" ? "check" : c.s === "warn" ? "warn" : "x"} s={c.s === "warn" ? 12 : 11} w={c.s === "warn" ? 2 : 2.8} /></span>
                <div className="bi-cb">
                  <b>{c.l}</b><span>{c.x}</span>
                  {c.act === "biller" && (pick
                    ? <div className="bi-opts">{BI_BILLERS.map((n, i) => <button key={n} onClick={() => assign(n)}><span className="bi-av">{n.split(" ").map(w => w[0]).join("")}</span>{n}{i === 0 && <em>you</em>}</button>)}</div>
                    : <div className="bi-opts"><button className="pri" onClick={() => assign("Avery Lane")}>Assign to me</button><button onClick={() => setPick(true)}>Someone else…</button></div>)}
                  {c.act === "issue" && <>
                    <p className="bi-why">{iss.why}</p>
                    <div className="bi-opts">{iss.opts.map((o, k) => <button key={o.k} className={k === 0 ? "pri" : ""} onClick={() => bqSet(id, { res: o.k }, o.done)}>{o.l}</button>)}</div>
                  </>}
                </div>
              </div>
            ))}
          </div>
        </section>
      )}

      <section className="bi-sec">
        <div className="bi-sh"><h3>Charges</h3><span>vs rate con {bi$(rcTotal)}</span></div>
        <div className="bi-led">
          <div className="bi-lh"><span>Charge</span><span>Rate con</span><span>Billed</span></div>
          {g.lines.map((c, i) => (
            <div key={c.k} className={"bi-lr" + (iss && iss.flag === c.k && !g.res ? " flag" : "") + (c.off ? " off" : "")} style={{ animationDelay: i * 40 + "ms" }}>
              <span className="bi-ll"><b>{c.l}</b><span>{c.b}</span></span>
              <span className="bi-n mut">{c.rc == null ? "—" : bi$(c.rc)}</span>
              <span className="bi-n">{bi$(c.amt)}</span>
              {(c.off || c.adj) && g.st === 0 && <button className="bi-undo" title="Undo" onClick={() => bqSet(id, { res: null }, "Undid: " + g.opt.done.toLowerCase())}><BIc n="undo" s={12} /></button>}
            </div>
          ))}
          <div className="bi-lt"><span>Total</span><span className="bi-n mut">{bi$(rcTotal)}</span><span className="bi-n">{bi$(g.total)}</span></div>
          {g.total !== rcTotal && <div className="bi-diff">{bi$(Math.abs(g.total - rcTotal))} {g.total > rcTotal ? "over" : "under"} the rate con</div>}
        </div>
      </section>

      <section className="bi-sec">
        <div className="bi-sh"><h3>Shipment</h3><button className="bi-lnk">{d.shp}<BIc n="ext" s={11} /></button></div>
        <div className="bi-lane">
          <span className="bi-lt-i"><BIc n="truck" s={14} /></span>
          <div><b>{d.lane.from} <i>→</i> {d.lane.to}</b><span>{d.lane.delivered} · {d.lane.driver} · {d.lane.miles}</span></div>
        </div>
        <div className="bi-docs">{d.docs.map(([n, s]) => <button key={n} className={s === "Unsigned" && !g.res ? "miss" : ""}><BIc n="file" s={13} /><span><b>{n}</b><em>{s === "Unsigned" && g.res === "req" ? "Signed" : s}</em></span></button>)}</div>
      </section>

      <section className="bi-sec">
        <div className="bi-sh"><h3>Activity</h3></div>
        <ol className="bi-log">{g.log.map((e, i) => <li key={i} className={e.who === "You" ? "you" : ""}><i></i><span>{e.x}</span><em>{e.who} · {e.t}</em></li>)}</ol>
      </section>

      <div className="bi-bar">
        {g.st === 2 ? <div className="bi-done"><BIc n="check" s={13} w={2.6} />Posted as {d.inv} · sent to {d.billTo.split(" · ")[0]}</div> : g.hold ? <span className="bi-bh">Release the hold to continue</span> : <>
          <div className="bi-m">
            <button className={"ax-btn ghost" + (holdOpen ? " on" : "")} onClick={() => setHoldOpen(o => !o)}><BIc n="pause" s={12} w={2.4} />Hold<BIc n="down" s={10} w={2.4} /></button>
            {holdOpen && <div className="bi-mp">{BI_HOLDS.map(r => <button key={r} onClick={() => { setHoldOpen(false); bqSet(id, { hold: r }, "Put on hold · " + r.toLowerCase()); }}>{r}</button>)}</div>}
          </div>
          <span className="bi-bh">{g.st === 0 ? blocker : "Posting sends it to the customer and can't be undone"}</span>
          {g.st === 0 ? <button className="ax-btn ink" disabled={!!blocker} onClick={() => bqSet(id, { st: 1 }, "Approved the invoice for " + bi$(g.total))}>Approve</button>
            : <button className="ax-btn ink" onClick={() => bqSet(id, { st: 2 }, "Posted as " + d.inv + " · emailed to " + d.billTo.split(" · ")[0])}>Post {bi$(g.total)}</button>}
        </>}
      </div>
    </div>
  );
}

window.Body_bill = BillBody;
Object.assign(window, { BQ, bqGet, bqSet, bqOpen, bqApproveMany, bqUnapprove, useBQ, BI_STEPS });
