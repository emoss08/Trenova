const ST = { new: "New", assigned: "Assigned", transit: "In transit", delayed: "Delayed", done: "Delivered" };
const D = (n, i, h, unit, hos, mi) => ({ n, i, h, unit, hos, mi });
const SHIPMENTS = [
  { id: "S2610-0412", bol: "BOL-88214", order: "0261000412", o: ["DSM", "Des Moines, IA"], d: ["CHI", "Chicago, IL"], st: "delayed", pct: 48, mi: 334, drv: D("Jaime Park", "JP", 28, "T-211", "2:15"), eta: "18:40", etaD: "Today", delta: "+2h 40m", late: true, cust: "Hy-Vee Distribution", wt: 41200, rev: 2140, mg: 14, tender: "Accepted", equip: "Reefer 53′", reg: ["Midwest", "Midwest"], today: true, risk: "Weather on I-80" },
  { id: "S2610-0398", bol: "BOL-88190", order: "0261000398", o: ["OMA", "Omaha, NE"], d: ["MSP", "Minneapolis, MN"], st: "delayed", pct: 31, mi: 381, drv: D("Kai Whitehorse", "KW", 192, "T-176", "4:30"), eta: "21:10", etaD: "Today", delta: "+1h 55m", late: true, cust: "Cargill Protein", wt: 39800, rev: 1985, mg: 18, tender: "Accepted", equip: "Reefer 53′", reg: ["Midwest", "Midwest"], today: true, risk: "Weather on I-35" },
  { id: "S2610-0421", bol: "BOL-88231", order: "0261000421", o: ["KCK", "Kansas City, KS"], d: ["STL", "St. Louis, MO"], st: "delayed", pct: 66, mi: 248, drv: D("Luis Mendez", "LM", 45, "T-189", "7:48"), eta: "16:20", etaD: "Today", delta: "+45m", late: true, cust: "Anheuser-Busch", wt: 43100, rev: 1320, mg: 22, tender: "Accepted", equip: "Dry van 53′", reg: ["Midwest", "Midwest"], today: true, risk: "Shipper ran late" },
  { id: "S2610-0430", bol: "BOL-88244", order: "0261000430", o: ["FTW", "Fort Worth, TX"], d: ["HOU", "Houston, TX"], st: "new", pct: 0, mi: 263, drv: null, eta: "10/07 17:00", etaD: "Pickup tomorrow 08:00", cust: "Sample Retail Group", wt: 38000, rev: 1920, mg: 24, tender: "Pending", equip: "Dry van 53′", reg: ["South", "South"] },
  { id: "S2610-0431", bol: "BOL-88245", order: "0261000431", o: ["SAT", "San Antonio, TX"], d: ["DAL", "Dallas, TX"], st: "new", pct: 0, mi: 274, drv: null, eta: "10/06 17:00", etaD: "Pickup today 13:00", cust: "Sample Shipper Co", wt: 38000, rev: 2900, mg: 27, tender: "Pending", equip: "Reefer 53′", reg: ["South", "South"], today: true },
  { id: "S2610-0377", bol: "BOL-88162", order: "0261000377", o: ["ATL", "Atlanta, GA"], d: ["CLT", "Charlotte, NC"], st: "transit", pct: 74, mi: 244, drv: D("Dana Okafor", "DO", 320, "T-204", "6:10"), eta: "14:05", etaD: "Today", delta: "on time", cust: "Publix Supermarkets", wt: 36400, rev: 1210, mg: 19, tender: "Accepted", equip: "Reefer 53′", reg: ["South", "Northeast"], today: true },
  { id: "S2610-0385", bol: "BOL-88171", order: "0261000385", o: ["PHX", "Phoenix, AZ"], d: ["LAX", "Los Angeles, CA"], st: "transit", pct: 52, mi: 372, drv: D("Marcus Bell", "MB", 25, "T-158", "5:40"), eta: "17:30", etaD: "Today", delta: "12m early", cust: "Costco Wholesale", wt: 42000, rev: 1760, mg: 21, tender: "Accepted", equip: "Dry van 53′", reg: ["West", "West"], today: true },
  { id: "S2610-0389", bol: "BOL-88176", order: "0261000389", o: ["DEN", "Denver, CO"], d: ["SLC", "Salt Lake City, UT"], st: "transit", pct: 23, mi: 520, drv: D("Priya Raman", "PR", 280, "T-233", "9:20"), eta: "10/06 09:15", etaD: "Tomorrow", delta: "on time", cust: "Kroger Mountain", wt: 34900, rev: 2480, mg: 23, tender: "Accepted", equip: "Reefer 53′", reg: ["West", "West"] },
  { id: "S2610-0402", bol: "BOL-88199", order: "0261000402", o: ["MEM", "Memphis, TN"], d: ["NSH", "Nashville, TN"], st: "transit", pct: 88, mi: 212, drv: D("Tom Avery", "TA", 140, "T-117", "3:05"), eta: "12:50", etaD: "Today", delta: "on time", cust: "FedEx Supply Chain", wt: 28700, rev: 980, mg: 17, tender: "Accepted", equip: "Dry van 53′", reg: ["South", "South"], today: true },
  { id: "S2610-0408", bol: "BOL-88207", order: "0261000408", o: ["NWK", "Newark, NJ"], d: ["BOS", "Boston, MA"], st: "transit", pct: 39, mi: 226, drv: D("Sam Ortiz", "SO", 60, "T-220", "8:00"), eta: "15:15", etaD: "Today", delta: "on time", cust: "Stop & Shop", wt: 40100, rev: 1450, mg: 9, tender: "Accepted", equip: "Reefer 53′", reg: ["Northeast", "Northeast"], today: true },
  { id: "S2610-0415", bol: "BOL-88222", order: "0261000415", o: ["CHI", "Chicago, IL"], d: ["DET", "Detroit, MI"], st: "assigned", pct: 0, mi: 283, drv: D("Rena Cole", "RC", 230, "T-242", "11:00"), eta: "10/06 11:00", etaD: "Pickup 16:00", cust: "Meijer", wt: 37500, rev: 1390, mg: 20, tender: "Accepted", equip: "Dry van 53′", reg: ["Midwest", "Midwest"] },
  { id: "S2610-0418", bol: "BOL-88227", order: "0261000418", o: ["SEA", "Seattle, WA"], d: ["PDX", "Portland, OR"], st: "assigned", pct: 0, mi: 174, drv: D("Ben Takeda", "BT", 160, "T-131", "10:45"), eta: "10/06 08:30", etaD: "Pickup 21:00", cust: "Fred Meyer", wt: 31800, rev: 890, mg: 25, tender: "Accepted", equip: "Reefer 53′", reg: ["West", "West"] },
  { id: "S2610-0366", bol: "BOL-88140", order: "0261000366", o: ["HOU", "Houston, TX"], d: ["NOL", "New Orleans, LA"], st: "done", pct: 100, mi: 348, drv: D("Ana Reyes", "AR", 350, "T-109", "—"), eta: "09:42", etaD: "Delivered", delta: "18m early", cust: "Sysco Gulf Coast", wt: 39000, rev: 1650, mg: 26, tender: "Accepted", equip: "Reefer 53′", reg: ["South", "South"], today: true },
  { id: "S2610-0371", bol: "BOL-88151", order: "0261000371", o: ["IND", "Indianapolis, IN"], d: ["CMH", "Columbus, OH"], st: "done", pct: 100, mi: 176, drv: D("Will Grant", "WG", 95, "T-148", "—"), eta: "08:15", etaD: "Delivered", delta: "on time", cust: "Kroger Central", wt: 35200, rev: 760, mg: 15, tender: "Accepted", equip: "Dry van 53′", reg: ["Midwest", "Midwest"], today: true },
];
const QUIET = SHIPMENTS.filter(s => s.cust.startsWith("Sample"));
const SUGGEST = {
  "S2610-0430": [D("Kai Whitehorse", "KW", 192, "T-176", "9:40", 18), D("Rena Cole", "RC", 230, "T-242", "11:00", 42)],
  "S2610-0431": [D("Marcus Bell", "MB", 25, "T-158", "10:10", 11), D("Ana Reyes", "AR", 350, "T-109", "8:55", 36)],
};
const ATTENTION = [
  { id: "a1", sev: "crit", ic: "cloud", kind: "ETA slip", due: "2h 40m late", hot: true, t: "Three Midwest loads are behind the same storm", s: "Weather on I-80 and I-35. Delivery appointments at Hy-Vee and Cargill will be missed without a reschedule.", a: "Notify customers", seg: "risk" },
  { id: "a2", sev: "warn", ic: "user", kind: "Coverage", due: "pickup 13:00", hot: true, t: "2 loads still need a driver", s: "$4,820 revenue waiting. Dispatch has a strong match for both.", a: "Review matches", seg: "unassigned" },
  { id: "a3", sev: "warn", ic: "clock", kind: "Hours of service", due: "2:15 left", t: "Jaime Park runs out of hours before Chicago", s: "S2610-0412 is 172 mi out. A relay at Davenport keeps the load moving.", a: "Plan relay", open: "S2610-0412" },
  { id: "a4", sev: "info", ic: "hand", kind: "Detention", due: "2h 18m", t: "Kai Whitehorse is waiting at Cargill Omaha", s: "Free time ended at 2h. Detention billable at $75/hr from 11:42.", a: "Add detention", open: "S2610-0398" },
  { id: "a5", sev: "agent", ic: "receipt", kind: "Tender", due: "1h ago", t: "Werner declined S2610-0433", s: "Two backup carriers on this lane have accepted similar loads this week.", a: "Re-tender" },
];
const ACTIVITY = [
  { c: "c", t: <><b>S2610-0412</b> ETA pushed to 18:40</>, at: "2m" },
  { c: "n", t: <><b>Kai Whitehorse</b> arrived at Cargill Omaha</>, at: "14m" },
  { t: <><b>Ana Reyes</b> delivered S2610-0366 · POD signed</>, at: "1h" },
  { t: <><b>Eric Moss</b> created S2610-0431</>, at: "3h" },
  { t: <><b>Eric Moss</b> created S2610-0430</>, at: "3h" },
  { t: <><b>Werner</b> declined tender on S2610-0433</>, at: "4h" },
];
const CUSTOMERS = [["Hy-Vee Distribution", 6420, 4], ["Costco Wholesale", 5280, 3], ["Sample Shipper Co", 2900, 1], ["Publix Supermarkets", 2420, 2], ["Sample Retail Group", 1920, 1]];
const REGIONS = ["West", "Midwest", "South", "Northeast"];
const HOURS = { "S2610-0412": [6.2, 16, 18.67], "S2610-0398": [7.5, 19.25, 21.17], "S2610-0421": [9, 15.6, 16.33], "S2610-0430": [32, 41], "S2610-0431": [13, 41], "S2610-0377": [8, 14.08], "S2610-0385": [9.5, 17.5], "S2610-0389": [11, 33.25], "S2610-0402": [9, 12.83], "S2610-0408": [8.5, 15.25], "S2610-0415": [16, 35], "S2610-0418": [21, 32.5], "S2610-0366": [3.5, 9.7], "S2610-0371": [5, 8.25] };
const NOW = 11.5;

SHIPMENTS.find(s => s.id === "S2610-0398").det = 138;
const HV = (() => {
  let seed = 7; const rnd = () => (seed = (seed * 16807) % 2147483647) / 2147483647;
  const CUST = ["Hy-Vee Distribution", "Cargill Protein", "Costco Wholesale", "Publix Supermarkets", "Kroger Central", "Sysco Gulf Coast", "Meijer", "Anheuser-Busch", "FedEx Supply Chain", "Stop & Shop", "Fred Meyer", "Walmart DC 6094", "Target RDC", "PepsiCo Frito-Lay", "Tyson Foods", "General Mills"];
  const drivers = SHIPMENTS.filter(s => s.drv).map(s => s.drv);
  const fmt = h => { const day = h >= 24 ? "10/06 " : ""; const x = h % 24; return day + String(Math.floor(x)).padStart(2, "0") + ":" + String(Math.round((x % 1) * 60) % 60).padStart(2, "0"); };
  const out = [...SHIPMENTS];
  for (let i = 0; i < 466; i++) {
    const t = SHIPMENTS[i % SHIPMENTS.length], r = rnd();
    const st = r < .4 ? "done" : r < .68 ? "transit" : r < .74 ? "delayed" : r < .87 ? "assigned" : "new";
    let p, dd, x;
    if (st === "done") { dd = 6 + rnd() * (NOW - 6.2); p = Math.max(0, dd - 2 - rnd() * 6); }
    else if (st === "transit" || st === "delayed") { p = 4 + rnd() * (NOW - 4); dd = NOW + 0.3 + rnd() * 12; if (st === "delayed") x = dd + 0.4 + rnd() * 3; }
    else { p = NOW + 0.2 + rnd() * 30; dd = p + 2 + rnd() * 8; }
    const id = "S2610-" + (1000 + i);
    HOURS[id] = x ? [p, dd, x] : [p, dd];
    const pct = st === "done" ? 100 : st === "transit" || st === "delayed" ? Math.round(10 + rnd() * 80) : 0;
    const drv = st === "new" ? null : drivers[Math.floor(rnd() * drivers.length)];
    const rev = Math.round((t.rev * (0.7 + rnd() * 0.7)) / 10) * 10;
    const late = st === "delayed";
    const mins = x ? Math.round((x - dd) * 60) : 0;
    const s = { ...t, det: undefined, id, bol: "BOL-" + (90000 + i), order: "026100" + (1000 + i), st, pct, drv, cust: CUST[Math.floor(rnd() * CUST.length)], rev, mg: Math.round(8 + rnd() * 22), tender: st === "new" ? (rnd() < .5 ? "Pending" : "Accepted") : "Accepted", eta: fmt(x || dd), etaD: st === "new" ? "Pickup " + fmt(p) : t.etaD, late, delta: late ? "+" + Math.floor(mins / 60) + "h " + (mins % 60) + "m" : st === "done" ? (rnd() < .7 ? "on time" : Math.round(rnd() * 30) + "m early") : "on time", risk: late ? ["Weather on I-80", "Shipper ran late", "Traffic near Dallas", "Breakdown, unit swapped"][Math.floor(rnd() * 4)] : undefined, today: dd < 24 };
    if ((st === "transit" || st === "assigned") && rnd() < .07) s.det = 125 + Math.round(rnd() * 150);
    if (!drv && SUGGEST[t.id]) SUGGEST[id] = SUGGEST[t.id];
    if (!drv && !SUGGEST[id]) SUGGEST[id] = SUGGEST["S2610-0431"];
    out.push(s);
  }
  return out;
})();
const POOL = (() => {
  let seed = 31; const rnd = () => (seed = (seed * 16807) % 2147483647) / 2147483647;
  const F = ["Maria", "Dev", "Chris", "Tasha", "Omar", "Jake", "Lena", "Rafael", "Nina", "Hector", "Paige", "Andre", "Sofia", "Grant", "Yusuf", "Carla", "Brett", "Ivy", "Malik", "Rosa", "Tyler", "Kim", "Victor", "June", "Darnell", "Elise", "Sam", "Noor", "Wes", "Alma", "Cole", "Rita", "Jonah", "Mae", "Luis", "Tara", "Felix", "Gwen", "Abe", "Lou"];
  const L = ["Ortega", "Patel", "Nguyen", "Brooks", "Haddad", "Miller", "Kowalski", "Diaz", "Fischer", "Ruiz", "Morgan", "Wallace", "Rossi", "Hughes", "Karim", "Vega", "Larson", "Chen", "Jordan", "Flores", "Reed", "Park", "Santos", "Bell", "Price", "Moreau", "Cruz", "Aziz", "Hale", "Ramos", "Byrne", "Silva", "Stone", "Lin", "Romero", "Shah", "Wolfe", "Grant", "Nash", "Kerr"];
  const C = ["Dallas, TX", "Fort Worth, TX", "Houston, TX", "San Antonio, TX", "Atlanta, GA", "Memphis, TN", "Chicago, IL", "Kansas City, KS", "Phoenix, AZ", "Denver, CO", "Omaha, NE", "Newark, NJ"];
  return F.map((f, i) => { const r = rnd(); const free = r < .45 ? 0 : r < .8 ? +(0.2 + rnd() * 2).toFixed(2) : +(2 + rnd() * 4).toFixed(2); const hos = 3 + rnd() * 8; return { n: f + " " + L[i], i: f[0] + L[i][0], h: Math.round(rnd() * 360), unit: "T-" + (300 + i), hos: Math.floor(hos) + ":" + String(Math.round((hos % 1) * 60) % 60).padStart(2, "0"), hosN: hos, city: C[Math.floor(rnd() * C.length)], free, mi: 4 + Math.round(rnd() * 46) }; });
})();
const CARRIERS = [["Knight-Swift", "KS", 210], ["Schneider", "SN", 25], ["J.B. Hunt", "JB", 45], ["Werner", "WE", 0], ["Heartland Express", "HE", 140], ["Prime Inc.", "PR", 300], ["Covenant", "CV", 260], ["Landstar", "LS", 190], ["Marten", "MT", 160], ["Crete Carrier", "CR", 80], ["Hub Group", "HG", 230], ["U.S. Xpress", "UX", 350], ["Estes", "ES", 110], ["R+L Carriers", "RL", 60], ["Old Dominion", "OD", 120], ["Swift Reefer", "SR", 200]].map(([n, i, h], k) => ({ n, i, h, carrier: true, unit: "MC " + (104000 + k * 3917), hos: "—", acc: 68 + ((k * 37) % 30), rate: +(2.05 + ((k * 13) % 9) / 10).toFixed(2), trucks: (k * 7) % 5, mi: 0 }));
const hashId = id => [...id].reduce((a, c) => (a * 31 + c.charCodeAt(0)) % 9973, 7);
const covOf = (s, org) => { if (!s.drv) return null; if (s.drv.carrier) return { carrier: s.drv }; const h = hashId(s.id); if (org === "brokerage" || (org === "both" && h % 3 === 0)) return { carrier: CARRIERS[h % CARRIERS.length], driver: s.drv }; return { driver: s.drv }; };
Object.assign(window, { CARRIERS, covOf, hashId, POOL, HV, HOURS, NOW,  ST, SHIPMENTS, QUIET, SUGGEST, ATTENTION, ACTIVITY, CUSTOMERS, REGIONS });
