const useS = React.useState, useE = React.useEffect;

function Countdown({ from = 18, size = 14, tone = "warn" }) {
  const [n, setN] = useS(from);
  useE(() => { const iv = setInterval(() => setN(x => (x <= 1 ? from : x - 1)), 1000); return () => clearInterval(iv); }, [from]);
  const r = size / 2 - 1.5, c = 2 * Math.PI * r;
  return (
    <span className={"ec-cd t-" + tone} style={{ width: size, height: size }}>
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}><circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="currentColor" strokeOpacity=".22" strokeWidth="1.6"></circle><circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeDasharray={c} strokeDashoffset={c * (1 - n / from)} transform={`rotate(-90 ${size / 2} ${size / 2})`} style={{ transition: "stroke-dashoffset 1s linear" }}></circle></svg>
      <b>{n}</b>
    </span>
  );
}

function EComp({ value = "", placeholder = "Reply to Billing Specialist…", status, ring, off, send = "up", model = "auto", note }) {
  return (
    <div className={"cmp ec" + (ring ? " busy r-" + ring : "") + (off ? " off" : "")}>
      <span className="cmp-ring" aria-hidden="true"></span>
      {status && <div className="cmp-st ec-st">{status}</div>}
      {off ? <div className="ec-offmsg">{off}</div> : <textarea readOnly rows={2} value={value} placeholder={placeholder}></textarea>}
      <div className="cmp-b">
        <button className="ib" disabled={!!off}><Ic n="plus" s={16} /></button>
        <span className="apill"><AgentMark s={10} />Billing Specialist</span>
        <span style={{ flex: 1 }}></span>
        <span className={"mp2-b" + (model === "auto" ? " auto" : "")}><span className="mp2-bi"><BrandMark id={model} s={13} /></span><span className="mp2-bl">{model === "auto" ? "Auto" : MODEL_OPTIONS.find(o => o.p === model).model}</span></span>
        <button className="ib" disabled={!!off}><Ic n="mic" s={15} /></button>
        {send === "stop" ? <button className="send stop"><span className="sq"></span></button>
          : send === "count" ? <button className="send" disabled><Countdown from={18} size={18} tone="ink" /></button>
          : <button className="send" disabled={!value || !!off}><Ic n="up" s={15} w={2.2} /></button>}
      </div>
      {note && <div className="ec-cnote">{note}</div>}
    </div>
  );
}

function ECard({ tone = "err", icon = "alert", title, sub, children, actions, compact }) {
  return (
    <div className={"ec-card t-" + tone + (compact ? " sm" : "")}>
      <span className="ec-ic"><Ic n={icon} s={15} w={2} /></span>
      <div className="ec-body">
        {title && <b>{title}</b>}
        {sub && <span className="ec-sub">{sub}</span>}
        {children}
        {actions && <div className="ec-acts">{actions}</div>}
      </div>
    </div>
  );
}

const Btn = ({ k = "ghost", children, disabled, title }) => <button className={"ec-btn " + k} disabled={disabled} title={title}>{children}</button>;
const Q = ({ children, muted, tag }) => <div className={"ec-q" + (muted ? " muted" : "")}>{children}{tag && <span className="ec-qtag">{tag}</span>}</div>;
const R = ({ children, fade }) => <div className={"prose ec-r" + (fade ? " fade" : "")}>{children}</div>;

function Frame({ label, sub, children, dock, top }) {
  return (
    <figure className="ec-fig">
      <figcaption><b>{label}</b><span>{sub}</span></figcaption>
      <div className="ec-frame">
        {top}
        <div className="ec-thread">{children}</div>
        <div className="ec-dock">{dock}</div>
      </div>
    </figure>
  );
}

const PROV = [["anthropic", "Claude Sonnet 4.5", "Overloaded", "Anthropic returned 529 twice"], ["openai", "GPT-5", "Timed out", "No response after 30s"], ["gemini", "Gemini 2.5 Pro", "Not set up", "Not given the assistant task"]];

function StepRow({ st, verb, detail, why, action }) {
  const ic = { done: "check", failed: "x", denied: "lock", budget: "alert", invalid: "info", dup: "undo" }[st];
  return (
    <div className={"ec-step s-" + st}>
      <span className="ec-sic"><Ic n={ic} s={11} w={2.4} /></span>
      <span className="ec-stx"><b>{verb}</b>{detail && <span>{detail}</span>}{why && <em>{why}</em>}</span>
      {action}
    </div>
  );
}


const BULK_CUST = ["Granite Building", "Bluewater Retail", "Acme Manufacturing", "Harbor Supply Co.", "Northline Foods", "Peak Distributing", "Lakeside Grocers"];
function bulkItems(spec, prefix = "BQ-", start = 24100) {
  const out = []; let n = 0;
  spec.forEach(([reason, count]) => { for (let i = 0; i < count; i++) { out.push({ id: prefix + (start + ((n * 37) % 900)), name: BULK_CUST[(n * 3 + i) % 7], reason }); n++; } });
  return out;
}
const FAILED_POST = bulkItems([["Missing bill-to address", 31], ["Customer is on credit hold", 12], ["Accounting period is closed", 4]]);
const WOULD_FAIL = bulkItems([["Locked by an open dispute", 22], ["Belongs to a closed period", 11], ["Already has a biller", 5]], "BQ-", 25100);
const LOOKUP_FAIL = bulkItems([["Shipment not found", 14], ["Shipment is archived", 4]], "SEED-SHP-", 100);

function BulkList({ items, limit = 3, onOpenTable, noun = "items" }) {
  const reasons = [];
  items.forEach(it => { const r = reasons.find(x => x[0] === it.reason); r ? r[1]++ : reasons.push([it.reason, 1]); });
  const [f, setF] = useS(null);
  const [all, setAll] = useS(false);
  const [q, setQ] = useS("");
  let rows = f ? items.filter(i => i.reason === f) : items;
  if (q) rows = rows.filter(i => (i.id + " " + i.name).toLowerCase().includes(q.toLowerCase()));
  const shown = all ? rows : rows.slice(0, limit);
  return (
    <div className="bl">
      {reasons.length > 1 && <div className="bl-rs">{reasons.map(([r, n]) => (
        <button key={r} className={f === r ? "on" : ""} onClick={() => { setF(x => x === r ? null : r); }}><b>{n}</b>{r}</button>))}</div>}
      {all && items.length > 8 && <label className="bl-q"><Ic n="search" s={12} /><input value={q} onChange={e => setQ(e.target.value)} placeholder={"Filter " + rows.length + " " + noun} /></label>}
      <div className={"bl-l" + (all ? " all" : "")}>
        {shown.map((it, k) => <div key={it.id + k}><span className="ax-id">{it.id}</span><span>{it.name}</span>{!f && reasons.length > 1 ? <em>{it.reason}</em> : <em></em>}</div>)}
        {!shown.length && <div className="bl-none">No {noun} match “{q}”</div>}
      </div>
      <div className="bl-f">
        {rows.length > limit && <button className="ec-link" onClick={() => setAll(a => !a)}>{all ? "Show fewer" : "Show all " + rows.length + " " + noun}</button>}
        {onOpenTable && <button className="ec-link" onClick={onOpenTable}>Open as table</button>}
      </div>
    </div>
  );
}

function StepGroup({ st, verb, why, items, action, noun }) {
  const [open, setOpen] = useS(false);
  return (
    <div className={"ec-step s-" + st + " grp"}>
      <span className="ec-sic"><Ic n={st === "failed" ? "x" : st === "denied" ? "lock" : "alert"} s={11} w={2.4} /></span>
      <span className="ec-stx"><b>{verb}</b>{why && <em>{why}</em>}
        <button className="ec-link sm" onClick={() => setOpen(o => !o)}>{open ? "Hide" : "Show " + items.length}</button>
        {open && <div className="ec-sgl"><BulkList items={items} limit={6} noun={noun} /></div>}
      </span>
      {action}
    </div>
  );
}

const STEP_FAILS = [
  ["failed", "x", "18 couldn't be found", "Shipment lookups · 14 not found, 4 archived", () => LOOKUP_FAIL, "shipments"],
  ["denied", "lock", "7 not permitted", "Credit holds · needs Credit Manager", () => bulkItems([["Needs Credit Manager role", 7]], "CUST-", 300), "customers", "Request access"],
  ["budget", "alert", "1 out of budget", "Web search · resets Nov 1"],
  ["invalid", "info", "1 not accepted", "List payments · “last quarter” isn't a date range"],
  ["dup", "undo", "1 skipped", "Repeat of Read the billing queue"],
];
function StepFails() {
  const [open, setOpen] = useS(false);
  const [sel, setSel] = useS(null);
  return (
    <div className={"sf" + (open ? " open" : "")}>
      <button className="sf-h" onClick={() => { setOpen(o => !o); setSel(null); }}>
        <span className="sf-ic"><Ic n="alert" s={12} w={2.2} /></span>
        <span><b>28 of 31 steps didn't go through</b></span>
        <span className="sf-dots">{STEP_FAILS.map(f => <i key={f[0]} className={"s-" + f[0]}></i>)}</span>
        <span className="sf-cv"><Ic n="chevR" s={11} w={2.2} /></span>
      </button>
      {open && <div className="sf-l">
        {STEP_FAILS.map(([st, ic, label, detail, items, noun, act]) => (
          <div key={st} className={"sf-r s-" + st + (sel === st ? " on" : "")}>
            <button className="sf-rh" onClick={() => items && setSel(x => x === st ? null : st)} style={{ cursor: items ? "pointer" : "default" }}>
              <span className="ec-sic"><Ic n={ic} s={10} w={2.4} /></span><b>{label}</b><span>{detail}</span>
              {act && <span className="ec-link sm">{act}</span>}
              {items && <span className="sf-cv"><Ic n="chevR" s={10} w={2.2} /></span>}
            </button>
            {sel === st && items && <div className="sf-items"><BulkList items={items()} limit={5} noun={noun} /></div>}
          </div>
        ))}
      </div>}
    </div>
  );
}

Object.assign(window, { StepFails, BulkList, StepGroup, FAILED_POST, WOULD_FAIL, LOOKUP_FAIL, bulkItems });
Object.assign(window, { Countdown, EComp, ECard, Btn, Q, R, Frame, PROV, StepRow });
