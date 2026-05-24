/* Mock data for Trenova Shipments Command Center */

const SHIPMENTS = [
  { id: "SHP-2026-1042", pro: "PRO-984512", bol: "BOL-2026-0042", customer: "Acme Manufacturing", commodity: "Industrial parts", weight: "42,180 lb", origin: "Long Beach, CA", originCode: "TERM-LA", dest: "Chicago, IL", destCode: "DC-CHI", driver: "M. Alvarez", tractor: "T-1184", trailer: "TR-9921", status: "In Transit", progress: 62, eta: "Apr 22, 14:30", etaStatus: "on-time", revenue: 4280, margin: 24, rpm: 2.41, miles: 2014, milesDone: 1248, lat: 39.7, lon: -98.5, hosLeft: "06:42", deliveries: 1, stops: 4, dwell: null, risk: "low", tendered: true, lastEvent: "Crossed AZ/NM border", lastEventAt: "12 min ago" },
  { id: "SHP-2026-1041", pro: "PRO-984501", bol: "BOL-2026-0041", customer: "FreshHaul Foods", commodity: "Refrigerated produce", weight: "38,400 lb", origin: "Phoenix, AZ", originCode: "COLD-PHX", dest: "Customer DC, TX", destCode: "CUST-DEN", driver: "J. Park", tractor: "T-2208", trailer: "REF-441", status: "At Risk", progress: 41, eta: "Apr 22, 22:15", etaStatus: "late", revenue: 3120, margin: 11, rpm: 1.92, miles: 1102, milesDone: 451, lat: 32.7, lon: -106.4, hosLeft: "02:15", deliveries: 1, stops: 2, dwell: 142, risk: "high", tendered: true, lastEvent: "Reefer temp warning -2°F", lastEventAt: "4 min ago" },
  { id: "SHP-2026-1040", pro: "PRO-984488", bol: "BOL-2026-0040", customer: "GlobalTrade Imports", commodity: "Mixed dry goods", weight: "29,600 lb", origin: "Los Angeles, CA", originCode: "TERM-LA", dest: "Chicago, IL", destCode: "DC-CHI", driver: "S. Ndiaye", tractor: "T-1170", trailer: "TR-8810", status: "Detention", progress: 18, eta: "Apr 23, 09:00", etaStatus: "watch", revenue: 3640, margin: 18, rpm: 2.05, miles: 1745, milesDone: 312, lat: 35.0, lon: -114.5, hosLeft: "08:11", deliveries: 1, stops: 3, dwell: 218, risk: "med", tendered: true, lastEvent: "Stop 2 dwell 3h 38m", lastEventAt: "now" },
  { id: "SHP-2026-1039", pro: "PRO-984477", bol: "BOL-2026-0039", customer: "Acme Manufacturing", commodity: "Steel coils", weight: "44,800 lb", origin: "Cleveland, OH", originCode: "WH-CLE", dest: "Detroit, MI", destCode: "MANF-01", driver: "R. O'Sullivan", tractor: "T-3019", trailer: "FB-220", status: "Loading", progress: 8, eta: "Apr 22, 19:00", etaStatus: "on-time", revenue: 1820, margin: 27, rpm: 2.51, miles: 178, milesDone: 14, lat: 41.5, lon: -81.7, hosLeft: "10:55", deliveries: 1, stops: 2, dwell: 32, risk: "low", tendered: true, lastEvent: "Loading at WH-CLE", lastEventAt: "22 min ago" },
  { id: "SHP-2026-1038", pro: "PRO-984465", bol: "BOL-2026-0038", customer: "Range Logistics", commodity: "Auto parts", weight: "31,200 lb", origin: "Atlanta, GA", originCode: "TERM-ATL", dest: "Charlotte, NC", destCode: "CUST-CLT", driver: "A. Romero", tractor: "T-1145", trailer: "TR-7714", status: "Delivered", progress: 100, eta: "Apr 21, 16:42", etaStatus: "delivered", revenue: 1480, margin: 31, rpm: 2.74, miles: 244, milesDone: 244, lat: 35.2, lon: -80.8, hosLeft: "—", deliveries: 1, stops: 2, dwell: 0, risk: "none", tendered: true, lastEvent: "POD captured", lastEventAt: "2h ago" },
  { id: "SHP-2026-1037", pro: "PRO-984451", bol: "BOL-2026-0037", customer: "Peak Distributing", commodity: "Beverages", weight: "39,900 lb", origin: "Denver, CO", originCode: "WH-DEN", dest: "Salt Lake City, UT", destCode: "DC-SLC", driver: "K. Whitehorse", tractor: "T-2240", trailer: "TR-9001", status: "In Transit", progress: 78, eta: "Apr 22, 18:20", etaStatus: "on-time", revenue: 2240, margin: 22, rpm: 2.18, miles: 525, milesDone: 410, lat: 40.5, lon: -110.0, hosLeft: "04:30", deliveries: 1, stops: 2, dwell: 18, risk: "low", tendered: true, lastEvent: "Fuel stop Grand Junction", lastEventAt: "38 min ago" },
  { id: "SHP-2026-1036", pro: "PRO-984440", bol: "BOL-2026-0036", customer: "FreshHaul Foods", commodity: "Frozen", weight: "40,100 lb", origin: "Seattle, WA", originCode: "COLD-SEA", dest: "Portland, OR", destCode: "DC-PDX", driver: "L. Mendez", tractor: "T-1162", trailer: "REF-405", status: "In Transit", progress: 55, eta: "Apr 22, 16:00", etaStatus: "on-time", revenue: 1340, margin: 19, rpm: 2.10, miles: 174, milesDone: 96, lat: 46.7, lon: -122.7, hosLeft: "07:48", deliveries: 1, stops: 2, dwell: 12, risk: "low", tendered: true, lastEvent: "Departed COLD-SEA", lastEventAt: "1h 12m ago" },
  { id: "SHP-2026-1035", pro: "PRO-984431", bol: "BOL-2026-0035", customer: "Acme Manufacturing", commodity: "Industrial parts", weight: "27,300 lb", origin: "Houston, TX", originCode: "TERM-HOU", dest: "New Orleans, LA", destCode: "CUST-NOL", driver: "—", tractor: "—", trailer: "—", status: "Unassigned", progress: 0, eta: "Apr 23, 12:00", etaStatus: "pending", revenue: 1850, margin: 26, rpm: 2.48, miles: 348, milesDone: 0, lat: null, lon: null, hosLeft: "—", deliveries: 1, stops: 2, dwell: 0, risk: "low", tendered: false, lastEvent: "Tendered to fleet", lastEventAt: "8 min ago" },
];

const UNASSIGNED = [
  { id: "SHP-2026-1035", lane: "TERM-HOU → CUST-NOL", customer: "Acme Manufacturing", pickup: "Apr 22, 14:00", revenue: 1850, miles: 348, equip: "53' DRY", priority: "high" },
  { id: "SHP-2026-1034", lane: "DAL → MEM",            customer: "Range Logistics",      pickup: "Apr 22, 17:00", revenue: 1620, miles: 463, equip: "53' DRY", priority: "med" },
  { id: "SHP-2026-1033", lane: "LAX → PHX",            customer: "GlobalTrade Imports",  pickup: "Apr 22, 19:30", revenue: 980,  miles: 372, equip: "53' DRY", priority: "med" },
  { id: "SHP-2026-1032", lane: "SEA → SPK",            customer: "FreshHaul Foods",      pickup: "Apr 23, 06:00", revenue: 1180, miles: 280, equip: "REEFER",  priority: "low" },
];

const DRIVERS = [
  { id: "D-204", name: "M. Alvarez",   tractor: "T-1184", hosLeft: "06:42", lane: "LAX → CHI", lat: 35.5, lon: -101.5, status: "moving", load: "SHP-2026-1042" },
  { id: "D-211", name: "J. Park",       tractor: "T-2208", hosLeft: "02:15", lane: "PHX → DAL", lat: 32.7, lon: -106.4, status: "moving", load: "SHP-2026-1041" },
  { id: "D-198", name: "S. Ndiaye",     tractor: "T-1170", hosLeft: "08:11", lane: "LAX → CHI", lat: 35.0, lon: -114.5, status: "dwell",  load: "SHP-2026-1040" },
  { id: "D-225", name: "R. O'Sullivan", tractor: "T-3019", hosLeft: "10:55", lane: "CLE → DET", lat: 41.5, lon: -81.7,  status: "loading",load: "SHP-2026-1039" },
  { id: "D-176", name: "K. Whitehorse", tractor: "T-2240", hosLeft: "04:30", lane: "DEN → SLC", lat: 40.5, lon: -110.0, status: "moving", load: "SHP-2026-1037" },
  { id: "D-189", name: "L. Mendez",     tractor: "T-1162", hosLeft: "07:48", lane: "SEA → PDX", lat: 46.7, lon: -122.7, status: "moving", load: "SHP-2026-1036" },
];

// 24h sparkline series (relative magnitudes)
const SERIES = {
  active:    [38,42,40,45,48,52,49,54,58,55,60,62,58,55,52,50,48,45,47,49,51,53,55,58],
  ontime:    [94,93,93,92,91,89,88,87,86,86,85,84,82,80,79,78,77,76,75,74,73,72,71,70],
  revenue:   [1.2,1.8,3.1,4.4,5.6,6.8,8.1,9.5,10.9,12.4,14.1,15.8,17.4,19.0,20.8,22.7,24.6,26.5,28.4,30.2,31.9,33.5,35.0,36.4],
  emptyMile: [12,12,11,11,12,13,14,14,15,14,13,12,12,11,11,11,10,10,11,11,11,12,12,12],
  ready:     [4,5,4,3,4,6,8,7,9,11,12,11,10,9,8,7,7,6,7,8,9,10,11,12],
  detention: [0,0,1,1,2,2,3,3,3,4,4,5,5,5,6,6,7,7,7,7,7,7,7,7],
  atrisk:    [3,3,4,5,5,6,5,5,4,4,4,5,6,6,7,7,8,8,9,9,9,9,9,9],
  hos:       [1,1,2,2,2,2,2,3,3,3,3,3,3,3,3,3,3,3,4,4,4,4,4,4],
  unassign:  [12,12,10,9,8,7,8,9,11,12,13,11,10,9,8,7,6,5,5,5,5,5,5,5],
  stops:     [2,5,9,14,20,28,35,42,49,56,62,68,73,78,82,86,89,92,95,97,99,101,103,104],
  tender:    [88,89,90,90,91,91,92,92,93,93,93,94,94,95,95,95,95,94,94,94,94,94,94,94],
  margin:    [18,19,18,17,18,19,20,21,21,22,22,23,23,22,22,21,21,21,22,22,23,23,23,24],
};

const ACTIVITY = [
  { t: "now",      type: "alert",    text: "Reefer temp warning −2°F on SHP-2026-1041", who: "system", sev: "danger" },
  { t: "2 min",    type: "comment",  text: "@ops-night requesting status on Acme order #1042", who: "@m.diaz", sev: "info" },
  { t: "4 min",    type: "geofence", text: "T-1184 entered AZ/NM border zone",          who: "system", sev: "muted" },
  { t: "8 min",    type: "tender",   text: "Acme Manufacturing tendered SHP-2026-1035", who: "edi",    sev: "brand" },
  { t: "12 min",   type: "assign",   text: "Driver S. Ndiaye assigned to SHP-2026-1040", who: "@b.huang", sev: "muted" },
  { t: "22 min",   type: "loading",  text: "Loading started at WH-CLE — SHP-2026-1039", who: "system", sev: "muted" },
  { t: "38 min",   type: "fuel",     text: "Fuel stop Grand Junction — T-2240",         who: "system", sev: "muted" },
  { t: "1h 12m",   type: "depart",   text: "T-1162 departed COLD-SEA",                  who: "system", sev: "muted" },
  { t: "2h",       type: "pod",      text: "POD captured for SHP-2026-1038",            who: "@a.romero", sev: "success" },
  { t: "2h 14m",   type: "rate",     text: "Spot rate confirmed $1,850 — Acme/HOU→NOL", who: "@k.tan", sev: "muted" },
];

const CUSTOMERS = [
  { name: "Acme Manufacturing", revenue: 41200, share: 34, loads: 18, trend: 1.2 },
  { name: "FreshHaul Foods",    revenue: 22800, share: 19, loads: 11, trend: -0.4 },
  { name: "GlobalTrade Imports",revenue: 18400, share: 15, loads: 9,  trend: 0.8 },
  { name: "Range Logistics",    revenue: 14600, share: 12, loads: 8,  trend: 0.3 },
  { name: "Peak Distributing",  revenue: 12100, share: 10, loads: 7,  trend: -0.1 },
  { name: "Other (6)",          revenue: 11900, share: 10, loads: 14, trend: 0.0 },
];

const LANES = [
  // origin region → dest region; counts
  { o: "West",     d: "Midwest", count: 14 },
  { o: "West",     d: "South",   count: 6  },
  { o: "West",     d: "West",    count: 9  },
  { o: "West",     d: "Northeast", count: 3 },
  { o: "Midwest",  d: "Midwest", count: 11 },
  { o: "Midwest",  d: "Northeast", count: 8 },
  { o: "Midwest",  d: "South",   count: 5  },
  { o: "Midwest",  d: "West",    count: 4  },
  { o: "South",    d: "South",   count: 12 },
  { o: "South",    d: "Midwest", count: 7  },
  { o: "South",    d: "Northeast", count: 4 },
  { o: "South",    d: "West",    count: 2  },
  { o: "Northeast",d: "Northeast", count: 6 },
  { o: "Northeast",d: "Midwest", count: 5  },
  { o: "Northeast",d: "South",   count: 3  },
  { o: "Northeast",d: "West",    count: 1  },
];

const PICKUPS_TOMORROW = [
  { time: "06:00", customer: "Acme Manufacturing", lane: "TERM-LA → DC-CHI", driver: "M. Alvarez", status: "scheduled" },
  { time: "07:30", customer: "FreshHaul Foods",    lane: "COLD-SEA → DC-PDX",driver: "L. Mendez",   status: "scheduled" },
  { time: "08:15", customer: "GlobalTrade Imports",lane: "TERM-LA → PHX",    driver: "—",            status: "unassigned" },
  { time: "09:00", customer: "Acme Manufacturing", lane: "TERM-HOU → NOL",   driver: "—",            status: "unassigned" },
  { time: "10:45", customer: "Peak Distributing",  lane: "WH-DEN → DC-SLC",  driver: "K. Whitehorse",status: "scheduled" },
  { time: "12:30", customer: "Range Logistics",    lane: "ATL → CLT",        driver: "A. Romero",   status: "scheduled" },
  { time: "14:00", customer: "Acme Manufacturing", lane: "CLE → DET",        driver: "R. O'Sullivan",status: "scheduled" },
  { time: "16:30", customer: "FreshHaul Foods",    lane: "PHX → DAL",        driver: "—",            status: "unassigned" },
];

const HOS_AT_RISK = [
  { driver: "J. Park",     id: "D-211", left: "02:15", load: "SHP-2026-1041", action: "Force 30m break by 19:45", sev: "danger" },
  { driver: "K. Whitehorse",id:"D-176", left: "04:30", load: "SHP-2026-1037", action: "Reset window opens 04:00",  sev: "warning" },
  { driver: "L. Mendez",   id: "D-189", left: "07:48", load: "SHP-2026-1036", action: "On track",                  sev: "muted" },
];

const SAVED_VIEWS = [
  { id: "all",       label: "All shipments",     count: 142 },
  { id: "transit",   label: "In transit",         count: 58 },
  { id: "atrisk",    label: "At risk",            count: 9  },
  { id: "unassign",  label: "Unassigned",         count: 5  },
  { id: "detention", label: "Detention",          count: 7  },
  { id: "today",     label: "Delivering today",   count: 23 },
];

Object.assign(window, { SHIPMENTS, UNASSIGNED, DRIVERS, SERIES, ACTIVITY, CUSTOMERS, LANES, PICKUPS_TOMORROW, HOS_AT_RISK, SAVED_VIEWS });
