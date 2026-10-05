const PAGES = {
  shipment: { icon: "truck", title: "Shipments", path: "/shipments" },
  billing: { icon: "receipt", title: "Billing queue", path: "/billing/queue" },
  customer: { icon: "headset", title: "Customers", path: "/customers" },
  report: { icon: "compass", title: "Detention report", path: "/reports/detention" },
};

function PageGlyph({ icon, s = 13 }) {
  return <svg width={s} height={s} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">{AG_ICON[icon]}</svg>;
}


const RECENT_PAGES = ["billing", "customer", "report"];
const PAGE_AGO = { billing: "4m", customer: "18m", report: "1h", shipment: "2h" };

function PageChip({ currentKey, sel, setSel, onExplain }) {
  const [open, setOpen] = React.useState(false);
  const [q, setQ] = React.useState("");
  const root = React.useRef(null);
  React.useEffect(() => {
    if (!open) return;
    setQ("");
    const off = e => root.current && !root.current.contains(e.target) && setOpen(false);
    const esc = e => e.key === "Escape" && setOpen(false);
    document.addEventListener("mousedown", off); window.addEventListener("keydown", esc);
    return () => { document.removeEventListener("mousedown", off); window.removeEventListener("keydown", esc); };
  }, [open]);
  const page = sel ? PAGES[sel] : null;
  const keys = [...(PAGES[currentKey] ? [currentKey] : []), ...RECENT_PAGES.filter(k => k !== currentKey)];
  const pick = k => { setSel(k); setOpen(false); };
  const ql = q.trim().toLowerCase();
  const shown = keys.filter(k => !ql || (PAGES[k].title + " " + PAGES[k].path).toLowerCase().includes(ql));
  const SmallRow = ({ k }) => (
    <button className={"pc-o" + (sel === k ? " sel" : "")} onClick={() => pick(k)}>
      {k ? <PageGlyph icon={PAGES[k].icon} s={13} /> : <Ic n="eye" s={13} />}
      <span className="pc-on">{k ? PAGES[k].title : "Don't share a page"}</span>
      {k === currentKey && <span className="pc-here">Current</span>}
      {sel === k && <Ic n="check" s={12} w={2.4} />}
    </button>
  );
  const Row = ({ k }) => (
    <button className={"pc-o2" + (sel === k ? " sel" : "")} onClick={() => pick(k)}>
      <span className="pc-oic">{k ? <PageGlyph icon={PAGES[k].icon} s={14} /> : <Ic n="eye" s={14} />}</span>
      <span className="pc-ot"><b>{k ? PAGES[k].title : "Don't share a page"}</b><em>{!k ? "The agent answers without seeing your screen" : k === currentKey ? "You're on this page" : "Viewed " + PAGE_AGO[k] + " ago"}</em></span>
      {sel === k && <span className="pc-ck"><Ic n="check" s={13} w={2.4} /></span>}
    </button>
  );
  return (
    <span className="pc" ref={root}>
      <button className={"pc-b" + (page ? "" : " off") + (open ? " open" : "")} onClick={() => setOpen(o => !o)} title={page ? "The assistant can see " + page.title : "Not sharing a page"}>
        <span className="pc-ic">{page ? <PageGlyph icon={page.icon} s={12} /> : <Ic n="eye" s={12} />}</span>
        <span className="pc-t">{page ? page.title : "No page"}</span>
      </button>
      {open && (
        <div className="pc-pop p1">
          <div className="pc-h1">
            <span className="pc-hic">{PAGES[currentKey] ? <PageGlyph icon={PAGES[currentKey].icon} s={16} /> : <Ic n="eye" s={16} />}</span>
            <span><b>{PAGES[currentKey] ? PAGES[currentKey].title : "Assistant"}</b><em>{PAGES[currentKey] ? "The page you're on" : "You're not on a page"}</em></span>
          </div>
          {PAGES[currentKey] && <dl className="pc-dl1"><div><dt>Page</dt><dd>{PAGES[currentKey].path}</dd></div></dl>}
          {PAGES[currentKey] && <label className="pc-tg1">
            <span><b>Share this page with the assistant</b><em>{sel ? "Sent with each message so the agent knows what you're looking at" : "The agent won't see what you're looking at"}</em></span>
            <button role="switch" aria-checked={!!sel} className={"pc-sw" + (sel ? " on" : "")} onClick={() => setSel(sel ? null : currentKey)}><i></i></button>
          </label>}
          {sel && onExplain && <button className="pc-ex" onClick={() => { onExplain(); setOpen(false); }}><Ic n="info" s={13} w={2} />Explain what's on this page<span className="kbd">/explain</span></button>}
        </div>
      )}
    </span>
  );
}

function PageSent() {
  return null;
}

Object.assign(window, { PAGES, PageChip, PageSent, PageGlyph });
