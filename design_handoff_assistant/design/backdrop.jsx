const QROWS = [
  { id: "40118", cust: "Halvorsen Foods", lane: "Fresno, CA → Reno, NV", st: "Blocked", why: "Missing signed POD", age: 9, amt: "2,860.00" },
  { id: "40102", cust: "Ridgeline Steel", lane: "Gary, IN → Toledo, OH", st: "Blocked", why: "Rate mismatch", age: 7, amt: "1,180.00" },
  { id: "40131", cust: "Cascade Paper", lane: "Longview, WA → Boise, ID", st: "Blocked", why: "Accessorial not approved", age: 5, amt: "3,415.50" },
  { id: "40136", cust: "Blue Mesa Produce", lane: "Yuma, AZ → Phoenix, AZ", st: "Blocked", why: "Missing lumper receipt", age: 3, amt: "940.00" },
  { id: "40139", cust: "Northwind Grocers", lane: "Stockton, CA → Sparks, NV", st: "In review", why: "Detention added", age: 2, amt: "1,725.00" },
  { id: "40141", cust: "Summit Ag Supply", lane: "Salinas, CA → Medford, OR", st: "In review", why: "", age: 1, amt: "2,310.00" },
  { id: "40144", cust: "Ironbridge Fabrication", lane: "Peoria, IL → Davenport, IA", st: "Ready", why: "", age: 1, amt: "1,090.00" },
  { id: "40145", cust: "Halvorsen Foods", lane: "Fresno, CA → Sacramento, CA", st: "Ready", why: "", age: 0, amt: "1,480.00" },
  { id: "40147", cust: "Copperline Beverage", lane: "Modesto, CA → Reno, NV", st: "Ready", why: "", age: 0, amt: "2,045.00" },
  { id: "40149", cust: "Cascade Paper", lane: "Longview, WA → Portland, OR", st: "Ready", why: "", age: 0, amt: "860.00" },
];

function BgApp({ rows, hot, peek, flash, docked }) {
  const blocked = rows.filter(r => r.st === "Blocked").length;
  const nav = [["home", "Overview"], ["truck", "Shipments", "128"], ["radar", "Dispatch"], ["receipt", "Billing queue", String(rows.length), true], ["table", "Invoices"], ["chat", "Customers"]];
  return (
    <div className={"tv-app" + (docked ? " docked" : "")}>
      <aside className="tv-nav">
        <div className="tv-brand"><img src="logo.png" alt=""></img>Trenova</div>
        {nav.map(([ic, l, n, on]) => <div key={l} className={"tv-ni" + (on ? " on" : "")}><Ic n={ic} s={15}></Ic>{l}{n && <span className="n">{n}</span>}</div>)}
        <div className="tv-ng">Settings</div>
        <div className="tv-ni"><Ic n="gear" s={15}></Ic>Workspace</div>
      </aside>
      <main className="tv-main">
        <header className="tv-hd">
          <span className="tv-crumb">Billing<i>/</i><b>Queue</b></span>
          <div className="sp"></div>
          <div className="tv-search"><Ic n="search" s={13}></Ic>Search shipments<span className="kbd">/</span></div>
        </header>
        <div className="tv-bar">
          <div className="tv-tab on">All<span>{rows.length}</span></div>
          <div className="tv-tab">Blocked<span>{blocked}</span></div>
          <div className="tv-tab">In review<span>{rows.filter(r => r.st === "In review").length}</span></div>
          <div className="tv-tab">Ready<span>{rows.filter(r => r.st === "Ready").length}</span></div>
        </div>
        <div className="tv-tw">
          <table className="tv-t">
            <thead><tr><th>PRO</th><th>Customer</th><th>Lane</th><th>Status</th><th>Blocker</th><th className="r">Age</th><th className="r">Amount</th></tr></thead>
            <tbody>
              {rows.map(r => (
                <tr key={r.id} className={(hot === r.id ? "hot" : "") + (flash === r.id ? " flash" : "")}>
                  <td className="m">{r.id}</td>
                  <td>{r.cust}</td>
                  <td className="l">{r.lane}</td>
                  <td><span className={"tv-st " + (r.st === "In review" ? "review" : r.st)}><i></i>{r.st}</span></td>
                  <td className="tv-why">{r.why || "—"}</td>
                  <td className="m r">{r.age}d</td>
                  <td className="m r">${r.amt}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {peek && <div className="tv-peek"><span><Ic n="eye" s={12}></Ic>Shared with the assistant</span></div>}
      </main>
    </div>
  );
}
Object.assign(window, { QROWS, BgApp });
