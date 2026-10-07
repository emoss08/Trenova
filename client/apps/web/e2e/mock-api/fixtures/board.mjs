// The shipment board's world: shipments, drivers and carriers, generated the
// same way the design handoff's prototype does (design_handoff_shipments_v2/
// design/data.jsx) so the screens can be compared side by side. Hours are
// relative to the anchor, the board's "now".

const HOUR = 3600;

export function createRandom(seed) {
  let state = seed;
  return () => {
    state = (state * 16807) % 2147483647;
    return state / 2147483647;
  };
}

const CITY = {
  DSM: ["Des Moines", "IA", "Hy-Vee DC 4"],
  CHI: ["Chicago", "IL", "Hy-Vee Chicago Hub"],
  OMA: ["Omaha", "NE", "Cargill Protein Plant"],
  MSP: ["Minneapolis", "MN", "Cargill Cold Store"],
  KCK: ["Kansas City", "KS", "AB Brewery KC"],
  STL: ["St. Louis", "MO", "St. Louis Terminal"],
  FTW: ["Fort Worth", "TX", "Sample Warehouse"],
  HOU: ["Houston", "TX", "Sample Retail Store"],
  SAT: ["San Antonio", "TX", "Sample Plant"],
  DAL: ["Dallas", "TX", "Sample Dist. Center"],
  ATL: ["Atlanta", "GA", "Publix Lakeland Hub"],
  CLT: ["Charlotte", "NC", "Publix Charlotte"],
  PHX: ["Phoenix", "AZ", "Phoenix Terminal"],
  LAX: ["Los Angeles", "CA", "Costco Los Angeles"],
  DEN: ["Denver", "CO", "Denver Terminal"],
  SLC: ["Salt Lake City", "UT", "Kroger Salt Lake City"],
  MEM: ["Memphis", "TN", "Memphis Terminal"],
  NSH: ["Nashville", "TN", "FedEx Nashville"],
  NWK: ["Newark", "NJ", "Newark Terminal"],
  BOS: ["Boston", "MA", "Stop Boston"],
  DET: ["Detroit", "MI", "Meijer Detroit"],
  SEA: ["Seattle", "WA", "Seattle Terminal"],
  PDX: ["Portland", "OR", "Fred Portland"],
  NOL: ["New Orleans", "LA", "Sysco New Orleans"],
  IND: ["Indianapolis", "IN", "Indianapolis Terminal"],
  CMH: ["Columbus", "OH", "Kroger Columbus"],
};

const D = (name, unit, hos) => ({ name, unit, hos });

// [pro, origin, dest, status, pct, miles, driver, cust, weight, revenue, margin, tender, reefer, risk, hours]
const BASE = [
  ["S2610-0412", "DSM", "CHI", "delayed", 48, 334, D("Jaime Park", "T-211", 2.25), "Hy-Vee Distribution", 41200, 2140, 14, "Accepted", true, "Weather on I-80", [6.2, 16, 18.67]],
  ["S2610-0398", "OMA", "MSP", "delayed", 31, 381, D("Kai Whitehorse", "T-176", 4.5), "Cargill Protein", 39800, 1985, 18, "Accepted", true, "Weather on I-35", [7.5, 19.25, 21.17]],
  ["S2610-0421", "KCK", "STL", "delayed", 66, 248, D("Luis Mendez", "T-189", 7.8), "Anheuser-Busch", 43100, 1320, 22, "Accepted", false, "Shipper ran late", [9, 15.6, 16.33]],
  ["S2610-0430", "FTW", "HOU", "new", 0, 263, null, "Sample Retail Group", 38000, 1920, 24, null, false, null, [32, 41]],
  ["S2610-0431", "SAT", "DAL", "new", 0, 274, null, "Sample Shipper Co", 38000, 2900, 27, null, true, null, [13, 41]],
  ["S2610-0377", "ATL", "CLT", "transit", 74, 244, D("Dana Okafor", "T-204", 6.17), "Publix Supermarkets", 36400, 1210, 19, "Accepted", true, null, [8, 14.08]],
  ["S2610-0385", "PHX", "LAX", "transit", 52, 372, D("Marcus Bell", "T-158", 5.67), "Costco Wholesale", 42000, 1760, 21, "Accepted", false, null, [9.5, 17.5]],
  ["S2610-0389", "DEN", "SLC", "transit", 23, 520, D("Priya Raman", "T-233", 9.33), "Kroger Mountain", 34900, 2480, 23, "Accepted", true, null, [11, 33.25]],
  ["S2610-0402", "MEM", "NSH", "transit", 88, 212, D("Tom Avery", "T-117", 3.08), "FedEx Supply Chain", 28700, 980, 17, "Accepted", false, null, [9, 12.83]],
  ["S2610-0408", "NWK", "BOS", "transit", 39, 226, D("Sam Ortiz", "T-220", 8), "Stop & Shop", 40100, 1450, 9, "Accepted", true, null, [8.5, 15.25]],
  ["S2610-0415", "CHI", "DET", "assigned", 0, 283, D("Rena Cole", "T-242", 11), "Meijer", 37500, 1390, 20, "Accepted", false, null, [16, 35]],
  ["S2610-0418", "SEA", "PDX", "assigned", 0, 174, D("Ben Takeda", "T-131", 10.75), "Fred Meyer", 31800, 890, 25, "Accepted", true, null, [21, 32.5]],
  ["S2610-0366", "HOU", "NOL", "done", 100, 348, D("Ana Reyes", "T-109", 0), "Sysco Gulf Coast", 39000, 1650, 26, "Accepted", true, null, [3.5, 9.7]],
  ["S2610-0371", "IND", "CMH", "done", 100, 176, D("Will Grant", "T-148", 0), "Kroger Central", 35200, 760, 15, "Accepted", false, null, [5, 8.25]],
];

const HV_CUSTOMERS = [
  "Hy-Vee Distribution", "Cargill Protein", "Costco Wholesale", "Publix Supermarkets", "Kroger Central",
  "Sysco Gulf Coast", "Meijer", "Anheuser-Busch", "FedEx Supply Chain", "Stop & Shop", "Fred Meyer",
  "Walmart DC 6094", "Target RDC", "PepsiCo Frito-Lay", "Tyson Foods", "General Mills",
];

const RISKS = ["Weather on I-80", "Shipper ran late", "Traffic near Dallas", "Breakdown, unit swapped"];

export const CARRIERS = [
  ["Knight-Swift", "KNXS"], ["Schneider", "SNDR"], ["J.B. Hunt", "HJBT"], ["Werner", "WERN"],
  ["Heartland Express", "HTLD"], ["Prime Inc.", "PRIM"], ["Covenant", "CVTI"], ["Landstar", "LSTR"],
  ["Marten", "MRTN"], ["Crete Carrier", "CRTE"], ["Hub Group", "HUBG"], ["U.S. Xpress", "USXP"],
  ["Estes", "EXLA"], ["R+L Carriers", "RLCA"], ["Old Dominion", "ODFL"], ["Swift Reefer", "SWFR"],
].map(([name, scac], k) => ({
  id: `car_${String(k + 1).padStart(4, "0")}`,
  name,
  scac,
  mcNumber: `MC ${104000 + k * 3917}`,
  acceptance: 68 + ((k * 37) % 30),
  rate: Number((2.05 + ((k * 13) % 9) / 10).toFixed(2)),
  trucks: (k * 7) % 5,
}));

const FIRST = ["Maria", "Dev", "Chris", "Tasha", "Omar", "Jake", "Lena", "Rafael", "Nina", "Hector", "Paige", "Andre", "Sofia", "Grant", "Yusuf", "Carla", "Brett", "Ivy", "Malik", "Rosa", "Tyler", "Kim", "Victor", "June", "Darnell", "Elise", "Sam", "Noor", "Wes", "Alma", "Cole", "Rita", "Jonah", "Mae", "Luis", "Tara", "Felix", "Gwen", "Abe", "Lou"];
const LAST = ["Ortega", "Patel", "Nguyen", "Brooks", "Haddad", "Miller", "Kowalski", "Diaz", "Fischer", "Ruiz", "Morgan", "Wallace", "Rossi", "Hughes", "Karim", "Vega", "Larson", "Chen", "Jordan", "Flores", "Reed", "Park", "Santos", "Bell", "Price", "Moreau", "Cruz", "Aziz", "Hale", "Ramos", "Byrne", "Silva", "Stone", "Lin", "Romero", "Shah", "Wolfe", "Grant", "Nash", "Kerr"];
const POOL_CITIES = ["Dallas, TX", "Fort Worth, TX", "Houston, TX", "San Antonio, TX", "Atlanta, GA", "Memphis, TN", "Chicago, IL", "Kansas City, KS", "Phoenix, AZ", "Denver, CO", "Omaha, NE", "Newark, NJ"];

export function createDriverPool() {
  const rnd = createRandom(31);
  return FIRST.map((first, i) => {
    const r = rnd();
    const freeInHours = r < 0.45 ? 0 : r < 0.8 ? Number((0.2 + rnd() * 2).toFixed(2)) : Number((2 + rnd() * 4).toFixed(2));
    const hos = 3 + rnd() * 8;
    return {
      id: `wrk_${String(i + 100).padStart(4, "0")}`,
      tractorId: `trc_${300 + i}`,
      firstName: first,
      lastName: LAST[i],
      unit: `T-${300 + i}`,
      hosHours: hos,
      city: POOL_CITIES[Math.floor(rnd() * POOL_CITIES.length)],
      freeInHours,
      deadheadMiles: 4 + Math.round(rnd() * 46),
    };
  });
}

function location(code) {
  const [city, state, name] = CITY[code];
  return {
    id: `loc_${code}`,
    name,
    code,
    status: "Active",
    locationCategoryId: null,
    stateId: `st_${state}`,
    addressLine1: `100 ${city} Ave`,
    addressLine2: null,
    city,
    postalCode: "00000",
    longitude: null,
    latitude: null,
    state,
  };
}

function customer(name) {
  const id = `cus_${name.replace(/[^A-Za-z0-9]/g, "").slice(0, 12).toLowerCase()}`;
  return {
    id,
    businessUnitId: "bu_mock",
    organizationId: "org_mock",
    stateId: null,
    status: "Active",
    code: name.slice(0, 4).toUpperCase(),
    name,
    addressLine1: null,
    addressLine2: null,
    city: null,
    postalCode: null,
    isGeocoded: false,
    longitude: null,
    latitude: null,
    placeId: null,
    externalId: null,
    allowConsolidation: false,
    exclusiveConsolidation: false,
    consolidationPriority: 0,
    version: 1,
    createdAt: 0,
    updatedAt: 0,
    ediPartner: null,
  };
}

const STATUS = { new: "New", assigned: "Assigned", transit: "InTransit", delayed: "Delayed", done: "Completed" };

/* Dispatchers who own loads; every fifth load has no owner, so "No owner" shows. */
const OWNERS = [
  { id: "usr_dispatch_ava", name: "Ava Lindqvist" },
  { id: "usr_dispatch_marcus", name: "Marcus Bell" },
  { id: "usr_dispatch_priya", name: "Priya Raman" },
];

function ownerOf(index) {
  return index % 5 === 4 ? null : OWNERS[index % OWNERS.length];
}

function ownerUser({ id, name }) {
  const username = name.toLowerCase().replace(/\s+/g, ".");
  return {
    id,
    name,
    username,
    emailAddress: `${username}@mock.trenova.test`,
    timezone: "America/Chicago",
    status: "Active",
    profilePicUrl: null,
    thumbnailUrl: null,
  };
}

function splitName(name) {
  const [firstName, ...rest] = name.split(" ");
  return { firstName, lastName: rest.join(" ") };
}

function buildShipment(raw, index, anchor) {
  const [pro, o, d, st, pct, miles, driver, cust, weight, revenue, margin, tender, reefer, risk, hours] = raw;
  const dayStart = anchor - (anchor % 86400);
  const at = (h) => Math.round(dayStart + h * HOUR);
  const [pickupH, deliveryH, slipH] = hours;
  const id = `shp_${pro.replace("S2610-", "")}`;
  const moveId = `mov_${pro.replace("S2610-", "")}`;
  const status = STATUS[st];
  const moving = st === "transit" || st === "delayed";
  const covered = !!driver;
  const pickup = {
    id: `stp_${id}_p`, businessUnitId: "bu_mock", organizationId: "org_mock", shipmentMoveId: moveId,
    locationId: `loc_${o}`, status: pct > 0 ? "Completed" : "New", type: "Pickup", scheduleType: "Appointment",
    sequence: 0, pieces: 22, weight, scheduledWindowStart: at(pickupH), scheduledWindowEnd: at(pickupH + 2),
    actualArrival: pct > 0 ? at(pickupH - 0.2) : null, actualDeparture: pct > 0 ? at(pickupH + 0.4) : null,
    countLateOverride: null, countDetentionOverride: null, addressLine: null, version: 1, createdAt: 0, updatedAt: 0,
    location: location(o),
  };
  const delivery = {
    id: `stp_${id}_d`, businessUnitId: "bu_mock", organizationId: "org_mock", shipmentMoveId: moveId,
    locationId: `loc_${d}`, status: st === "done" ? "Completed" : moving ? "InTransit" : "New", type: "Delivery",
    scheduleType: "Appointment", sequence: 1, pieces: 22, weight,
    scheduledWindowStart: at(deliveryH), scheduledWindowEnd: at(deliveryH + 1),
    actualArrival: st === "done" ? at(deliveryH - 0.3) : null, actualDeparture: st === "done" ? at(deliveryH + 0.2) : null,
    countLateOverride: null, countDetentionOverride: null, addressLine: null, version: 1, createdAt: 0, updatedAt: 0,
    location: location(d),
  };
  const driverPerson = driver ? { id: `wrk_${index}`, ...splitName(driver.name) } : null;
  return {
    id,
    businessUnitId: "bu_mock",
    organizationId: "org_mock",
    proNumber: pro,
    bol: `BOL-${88000 + index}`,
    orderNumber: `026100${pro.slice(-4)}`,
    orderId: `ord_${index}`,
    orderStatus: "Open",
    status,
    tenderStatus: tender,
    customerId: customer(cust).id,
    customer: customer(cust),
    ownerId: ownerOf(index)?.id ?? null,
    owner: ownerOf(index) ? ownerUser(ownerOf(index)) : null,
    billToCustomer: null,
    totalChargeAmount: String(revenue),
    freightChargeAmount: String(Math.round(revenue * 0.8)),
    otherChargeAmount: String(Math.round(revenue * 0.2)),
    baseRate: String(Math.round(revenue * 0.8)),
    weight,
    pieces: 22,
    temperatureMin: reefer ? 34 : null,
    temperatureMax: reefer ? 38 : null,
    billingTransferStatus: null,
    markedReadyToBillAt: null,
    transferredToBillingAt: null,
    billedAt: null,
    actualDeliveryDate: st === "done" ? at(deliveryH) : null,
    actualShipDate: pct > 0 ? at(pickupH) : null,
    createdAt: Math.round(anchor - (500 - index) * 60),
    updatedAt: Math.round(anchor - index * 30),
    version: 1,
    commodities: [{ id: `com_${index}`, commodity: { id: reefer ? "cmd_reefer" : "cmd_dry", name: reefer ? "Produce" : "Dry goods" }, pieces: 22, weight }],
    additionalCharges: [],
    chargeAllocations: [],
    profitabilityEstimate: {
      shipmentId: id, loadedMiles: miles, deadheadMiles: 0, totalMiles: miles, costPerMile: String(((revenue * (1 - margin / 100)) / miles).toFixed(2)),
      estimatedCost: String(Math.round(revenue * (1 - margin / 100))), profit: String(Math.round((revenue * margin) / 100)),
      marginPercent: String(margin), breakEvenRpm: null, targetMarginPercent: "18", missingDistance: false,
    },
    moves: [
      {
        id: moveId, businessUnitId: "bu_mock", organizationId: "org_mock", shipmentId: id,
        status: st === "done" ? "Completed" : moving ? "InTransit" : covered ? "Assigned" : "New",
        loaded: true, sequence: 0, distance: miles, distanceSource: "Calculated", coverageType: covered ? "driver" : "unassigned",
        version: 1, createdAt: 0, updatedAt: 0, stops: [pickup, delivery],
        assignment: covered
          ? {
              id: `asg_${index}`, businessUnitId: "bu_mock", organizationId: "org_mock", shipmentMoveId: moveId,
              primaryWorkerId: driverPerson.id, tractorId: `trc_${index}`, trailerId: null, secondaryWorkerId: null,
              status: "New", archivedAt: null, version: 1, createdAt: 0, updatedAt: 0,
              tractor: { id: `trc_${index}`, code: driver.unit }, trailer: null,
              primaryWorker: { id: driverPerson.id, ...driverPerson, wholeName: driver.name, profilePicUrl: null },
              secondaryWorker: null,
            }
          : null,
        carrierAssignment: null,
      },
    ],
    board: {
      pct,
      miles,
      pickupAt: at(pickupH),
      deliveryAt: at(deliveryH),
      slippedEta: slipH ? at(slipH) : null,
      hosHours: driver?.hos ?? null,
      reefer,
      margin,
      revenue,
      risk,
      detentionMinutes: null,
      driverName: driver?.name ?? null,
    },
  };
}

/** Builds the board for a scenario: "quiet", "busy" or "high" (the 480-row board). */
export function buildShipments(scenario, anchor) {
  const base = BASE.map((raw, i) => buildShipment(raw, i, anchor));
  base.find((s) => s.proNumber === "S2610-0398").board.detentionMinutes = 138;
  if (scenario === "quiet") {
    return base.filter((s) => s.customer.name.startsWith("Sample"));
  }
  if (scenario !== "high") {
    return base;
  }

  const rnd = createRandom(7);
  const drivers = BASE.filter((raw) => raw[6]).map((raw) => raw[6]);
  const now = 11.5;
  const out = [...base];
  for (let i = 0; i < 466; i++) {
    const template = BASE[i % BASE.length];
    const r = rnd();
    const st = r < 0.4 ? "done" : r < 0.68 ? "transit" : r < 0.74 ? "delayed" : r < 0.87 ? "assigned" : "new";
    let p;
    let dd;
    let x;
    if (st === "done") {
      dd = 6 + rnd() * (now - 6.2);
      p = Math.max(0, dd - 2 - rnd() * 6);
    } else if (st === "transit" || st === "delayed") {
      p = 4 + rnd() * (now - 4);
      dd = now + 0.3 + rnd() * 12;
      if (st === "delayed") x = dd + 0.4 + rnd() * 3;
    } else {
      p = now + 0.2 + rnd() * 30;
      dd = p + 2 + rnd() * 8;
    }
    const pct = st === "done" ? 100 : st === "transit" || st === "delayed" ? Math.round(10 + rnd() * 80) : 0;
    const driver = st === "new" ? null : drivers[Math.floor(rnd() * drivers.length)];
    const revenue = Math.round((template[9] * (0.7 + rnd() * 0.7)) / 10) * 10;
    const tender = st === "new" ? (rnd() < 0.5 ? null : "Tendered") : "Accepted";
    const cust = HV_CUSTOMERS[Math.floor(rnd() * HV_CUSTOMERS.length)];
    const margin = Math.round(8 + rnd() * 22);
    const risk = st === "delayed" ? RISKS[Math.floor(rnd() * RISKS.length)] : null;
    const raw = [
      `S2610-${1000 + i}`, template[1], template[2], st, pct, template[5], driver, cust, template[8], revenue,
      margin, tender, template[12], risk, x ? [p, dd, x] : [p, dd],
    ];
    const shipment = buildShipment(raw, 100 + i, anchor);
    if ((st === "transit" || st === "assigned") && rnd() < 0.07) {
      shipment.board.detentionMinutes = 125 + Math.round(rnd() * 150);
    }
    out.push(shipment);
  }
  return out;
}
