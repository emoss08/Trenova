// Stand-in for whatever page the user was on. Settings opens over it.
function BackdropApp({ s, dark, dim, onOpen }) {
  const [menu, setMenu] = React.useState(false);
  const rows = [["S-48213", "Chicago, IL → Dallas, TX", "In transit", "brand"], ["S-48219", "Joliet, IL → Memphis, TN", "Needs review", "warning"], ["S-48230", "Gary, IN → Atlanta, GA", "Tendered", ""], ["S-48231", "Aurora, IL → Denver, CO", "No carrier", "danger"], ["S-48240", "Elgin, IL → Omaha, NE", "Assigned", ""], ["S-48244", "Naperville, IL → Detroit, MI", "Delivered", "success"]];
  return (
    <div className={"us bk" + (dark ? " dk" : "") + (dim ? " dim" : "")} aria-hidden={dim}>
      <aside className="bk-sb">
        <div className="bk-org"><span className="us-logo">T</span><b>Acme Logistics</b></div>
        {[["truck", "Shipments", true], ["receipt", "Billing"], ["radar", "Carriers"], ["badge", "Workers"], ["file", "Reports"]].map(([i, l, on]) => <span key={l} className={"bk-ni" + (on ? " on" : "")}><UI n={i} s={15} />{l}</span>)}
        <span style={{ flex: 1 }}></span>
        <div className="bk-me-w">
          {menu && <div className="pop bk-menu">
            <div className="bk-mh"><Avatar s={s} size="sm" /><div><b>{s.name}</b><span>{s.email}</span></div></div>
            <button onClick={() => { setMenu(false); onOpen(); }}><UI n="sliders" s={14} />Settings<span className="kbd">⌘⇧S</span></button>
            <button onClick={() => setMenu(false)}><UI n="out" s={14} />Log out</button>
          </div>}
          <button className="bk-me" onClick={() => setMenu(m => !m)}><Avatar s={s} size="xs" /><span><b>{s.name}</b><em>{s.email}</em></span><UI n="chevD" s={13} /></button>
        </div>
      </aside>
      <main className="bk-main">
        <div className="bk-top"><b>Shipments</b><span style={{ flex: 1 }}></span><Button variant="outline" size="sm" onClick={onOpen}><UI n="sliders" s={14} />Settings<span className="kbd">⌘⇧S</span></Button></div>
        <div className="bk-tbl">{rows.map(([id, lane, st, tone]) => <div key={id} className="bk-r"><span className="mono">{id}</span><span>{lane}</span><Badge tone={tone || "neutral"}>{st}</Badge></div>)}</div>
        {!dim && <p className="bk-hint">Settings opens over any page — from the account menu or <span className="kbd">⌘</span><span className="kbd">⇧</span><span className="kbd">S</span> — and closes back to where you were.</p>}
      </main>
    </div>
  );
}
Object.assign(window, { BackdropApp });
