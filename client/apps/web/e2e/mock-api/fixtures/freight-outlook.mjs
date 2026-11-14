// The freight outlook (`/dispatch/outlook`): a two-day forecast for a regional carrier working
// Ohio and its neighbours. Cleveland and Columbus run short tomorrow morning, Toledo, Pittsburgh
// and Cincinnati have drivers to spare, and each kind of move the page offers is here once.
// Every time is a four-hour window counted from the window that holds `now`.

const SLOT = 4 * 3_600;
const DAY = 24 * 3_600;
const WINDOWS = 12;

const CELL = {
  cleveland: 12987,
  columbus: 12641,
  indianapolis: 11874,
  louisville: 11420,
  toledo: 13052,
  pittsburgh: 12990,
  cincinnati: 12233,
  fortWayne: 12466,
  akron: 12988,
  dayton: 12452,
  detroit: 13311,
  lexington: 11833,
  erie: 13422,
};

/** An area's windows: the same quiet shape everywhere, with the windows that differ named. */
function slotsOf(starts, base, overrides = {}) {
  return starts.map((start, index) => ({
    start,
    end: start + SLOT,
    likelyLow: base.low,
    likelyHigh: base.high,
    booked: base.booked,
    need: base.need,
    drivers: base.drivers,
    short: 0,
    spare: 0,
    ...overrides[index],
  }));
}

function areasOf(starts) {
  const area = (cell, label, latitude, longitude, valueCents, base, overrides, customers = []) => ({
    cell,
    label,
    latitude,
    longitude,
    valueCents,
    slots: slotsOf(starts, base, overrides),
    customers,
  });
  return [
    area(
      CELL.cleveland,
      "Cleveland, OH",
      41.4993,
      -81.6944,
      104_000,
      { low: 2, high: 4, booked: 2, need: 3, drivers: 3 },
      {
        1: { likelyLow: 3, likelyHigh: 6, booked: 3, need: 5, drivers: 3, short: 1.6 },
        2: { likelyLow: 5, likelyHigh: 8, booked: 4, need: 7, drivers: 4, short: 2.8 },
        3: { likelyLow: 4, likelyHigh: 7, booked: 3, need: 6, drivers: 4, short: 2.1 },
        7: { likelyLow: 3, likelyHigh: 6, booked: 2, need: 5, drivers: 4, short: 1.2 },
        8: { likelyLow: 5, likelyHigh: 8, booked: 2, need: 7, drivers: 4, short: 2.4 },
      },
      [
        { customerId: "cus_northstar", name: "Northstar Meats", share: 0.58 },
        { customerId: "cus_lakefront", name: "Lakefront Paper", share: 0.24 },
      ],
    ),
    area(
      CELL.columbus,
      "Columbus, OH",
      39.9612,
      -82.9988,
      86_000,
      { low: 2, high: 4, booked: 2, need: 3, drivers: 3 },
      {
        2: { likelyLow: 4, likelyHigh: 6, booked: 3, need: 5, drivers: 4, short: 1.2 },
        3: { likelyLow: 4, likelyHigh: 7, booked: 3, need: 6, drivers: 4, short: 1.9 },
        8: { likelyLow: 4, likelyHigh: 6, booked: 2, need: 5, drivers: 4, short: 1.1 },
        9: { likelyLow: 4, likelyHigh: 7, booked: 2, need: 6, drivers: 4, short: 1.7 },
      },
      [{ customerId: "cus_buckeye", name: "Buckeye Steel", share: 0.41 }],
    ),
    area(
      CELL.indianapolis,
      "Indianapolis, IN",
      39.7684,
      -86.1581,
      52_000,
      { low: 2, high: 4, booked: 2, need: 3, drivers: 3 },
      {
        3: { likelyLow: 3, likelyHigh: 5, need: 4, drivers: 3, short: 0.9 },
        4: { likelyLow: 3, likelyHigh: 5, need: 4, drivers: 3, short: 1.1 },
      },
      [{ customerId: "cus_crossroads", name: "Crossroads Auto Parts", share: 0.37 }],
    ),
    area(
      CELL.louisville,
      "Louisville, KY",
      38.2527,
      -85.7585,
      44_000,
      { low: 1, high: 3, booked: 1, need: 2, drivers: 2 },
      { 9: { likelyLow: 2, likelyHigh: 4, need: 3, drivers: 2, short: 0.8 } },
    ),
    area(
      CELL.toledo,
      "Toledo, OH",
      41.6528,
      -83.5379,
      9_000,
      { low: 1, high: 3, booked: 1, need: 2, drivers: 3 },
      {
        0: { drivers: 5, spare: 3.1 },
        1: { drivers: 5, spare: 2.7 },
        2: { drivers: 4, spare: 1.9 },
        6: { drivers: 4, spare: 1.6 },
        7: { drivers: 4, spare: 1.8 },
      },
    ),
    area(
      CELL.pittsburgh,
      "Pittsburgh, PA",
      40.4406,
      -79.9959,
      12_000,
      { low: 2, high: 4, booked: 2, need: 3, drivers: 4 },
      {
        1: { drivers: 6, spare: 2.4 },
        2: { drivers: 6, spare: 2.2 },
        3: { drivers: 5, spare: 1.5 },
        8: { drivers: 5, spare: 1.4 },
      },
    ),
    area(
      CELL.cincinnati,
      "Cincinnati, OH",
      39.1031,
      -84.512,
      14_000,
      { low: 2, high: 4, booked: 2, need: 3, drivers: 4 },
      {
        2: { drivers: 5, spare: 1.8 },
        3: { drivers: 5, spare: 1.6 },
        9: { drivers: 5, spare: 1.3 },
      },
    ),
    area(
      CELL.fortWayne,
      "Fort Wayne, IN",
      41.0793,
      -85.1394,
      11_000,
      {
        low: 1,
        high: 2,
        booked: 1,
        need: 2,
        drivers: 2,
      },
      { 4: { drivers: 3, spare: 1.2 } },
    ),
    area(CELL.akron, "Akron, OH", 41.0814, -81.519, 31_000, {
      low: 1,
      high: 3,
      booked: 1,
      need: 2,
      drivers: 2,
    }),
    area(CELL.dayton, "Dayton, OH", 39.7589, -84.1916, 27_000, {
      low: 1,
      high: 3,
      booked: 1,
      need: 2,
      drivers: 2,
    }),
    area(CELL.detroit, "Detroit, MI", 42.3314, -83.0458, 38_000, {
      low: 2,
      high: 4,
      booked: 2,
      need: 3,
      drivers: 3,
    }),
    area(CELL.lexington, "Lexington, KY", 38.0406, -84.5037, 22_000, {
      low: 1,
      high: 2,
      booked: 1,
      need: 2,
      drivers: 2,
    }),
    area(CELL.erie, "Erie, PA", 42.1292, -80.0851, 19_000, {
      low: 1,
      high: 2,
      booked: 1,
      need: 1,
      drivers: 1,
    }),
  ];
}

const NEED = [24, 31, 39, 35, 27, 19, 22, 33, 41, 36, 28, 20];
const DRIVERS = [41, 40, 42, 41, 39, 38, 40, 41, 42, 41, 39, 38];
const UNCOVERED = [0, 2, 5, 4, 1, 0, 0, 1, 4, 3, 1, 0];
const IDLE = [4, 5, 6, 5, 3, 2, 3, 4, 5, 4, 2, 2];

function networkOf(starts) {
  return starts.map((start, index) => ({
    start,
    end: start + SLOT,
    likelyLow: NEED[index] - 4,
    likelyHigh: NEED[index] + 7,
    needLow: NEED[index] - 5,
    needHigh: NEED[index] + 6,
    need: NEED[index],
    drivers: DRIVERS[index],
    short: 0,
    uncovered: UNCOVERED[index],
    idle: IDLE[index],
  }));
}

function emptyMove(starts) {
  return {
    kind: "Send",
    workerId: null,
    workerName: "",
    fromCell: 0,
    fromLabel: "",
    toCell: CELL.cleveland,
    toLabel: "Cleveland, OH",
    slotStart: starts[2],
    slotEnd: starts[2] + SLOT,
    short: 2.8,
    freeAt: null,
    arriveAt: null,
    miles: 0,
    emptyCostCents: 0,
    dropCents: 0,
    stayCents: 0,
    netCents: 0,
    savedCents: 0,
    moveId: null,
    proNumber: "",
    laneMoveIds: [],
    partnerId: null,
    partnerName: "",
    partnerCell: 0,
    partnerLabel: "",
    partnerFreeAt: null,
    partnerMiles: 0,
    customerId: null,
    customerName: "",
    usual: 0,
    byNow: 0,
    tendered: 0,
    loads: 0,
  };
}

function movesOf(starts) {
  const base = emptyMove(starts);
  return [
    {
      ...base,
      kind: "Book",
      workerId: "wrk_marcus_hill",
      workerName: "Marcus Hill",
      fromCell: CELL.pittsburgh,
      fromLabel: "Pittsburgh, PA",
      freeAt: starts[1] + 1_800,
      arriveAt: starts[2] + 900,
      miles: 133,
      emptyCostCents: 0,
      dropCents: 104_000,
      stayCents: 12_000,
      netCents: 92_000,
      savedCents: 26_600,
      moveId: "sm_outlook_book_1",
      proNumber: "S2611-0412",
      laneMoveIds: ["sm_outlook_book_1", "sm_outlook_book_2"],
    },
    {
      ...base,
      kind: "Send",
      workerId: "wrk_ana_ruiz",
      workerName: "Ana Ruiz",
      fromCell: CELL.toledo,
      fromLabel: "Toledo, OH",
      freeAt: starts[1] + 2_700,
      arriveAt: starts[2],
      miles: 112,
      emptyCostCents: 22_400,
      dropCents: 104_000,
      stayCents: 9_000,
      netCents: 72_600,
    },
    {
      ...base,
      kind: "Keep",
      workerId: "wrk_dana_okafor",
      workerName: "Dana Okafor",
      fromCell: CELL.columbus,
      fromLabel: "Columbus, OH",
      toCell: CELL.columbus,
      toLabel: "Columbus, OH",
      slotStart: starts[3],
      slotEnd: starts[3] + SLOT,
      short: 1.9,
      dropCents: 86_000,
      stayCents: 14_000,
      emptyCostCents: 19_600,
      netCents: 52_400,
      moveId: "sm_outlook_keep_1",
      proNumber: "S2611-0397",
      partnerId: "wrk_luis_ortega",
      partnerName: "Luis Ortega",
      partnerCell: CELL.cincinnati,
      partnerLabel: "Cincinnati, OH",
      partnerFreeAt: starts[2] + 3_600,
      partnerMiles: 98,
    },
    {
      ...base,
      kind: "Call",
      toCell: CELL.cleveland,
      toLabel: "Cleveland, OH",
      slotStart: starts[2],
      slotEnd: starts[2] + SLOT,
      customerId: "cus_northstar",
      customerName: "Northstar Meats",
      usual: 6,
      byNow: 4,
      tendered: 2,
      loads: 4,
    },
  ];
}

function paceOf(starts) {
  const pace = (customerId, customerName, cell, label, readings) => ({
    customerId,
    customerName,
    cell,
    label,
    slots: starts.map((start, index) => ({
      start,
      usual: readings[index]?.[0] ?? 0,
      byNow: readings[index]?.[1] ?? 0,
      tendered: readings[index]?.[2] ?? 0,
    })),
  });
  return [
    pace("cus_northstar", "Northstar Meats", CELL.cleveland, "Cleveland, OH", {
      1: [2, 1.6, 1],
      2: [6, 4, 2],
      3: [3, 1.8, 1],
      8: [5, 1.2, 0],
    }),
    pace("cus_buckeye", "Buckeye Steel", CELL.columbus, "Columbus, OH", {
      2: [3, 2.2, 2],
      3: [4, 2.6, 3],
      9: [3, 0.8, 1],
    }),
    pace("cus_lakefront", "Lakefront Paper", CELL.cleveland, "Cleveland, OH", {
      2: [2, 1.4, 3],
      3: [2, 1.2, 3],
    }),
    pace("cus_crossroads", "Crossroads Auto Parts", CELL.indianapolis, "Indianapolis, IN", {
      3: [2, 1.3, 1],
      4: [3, 1.6, 2],
    }),
  ];
}

// The last fourteen days: the range the forecast gave and the loads that came. Two days ran
// busier than their range, so twelve of fourteen held.
const RECORD = [
  [118, 142, 131, 127],
  [124, 150, 137, 141],
  [121, 147, 134, 152],
  [96, 118, 107, 104],
  [58, 76, 67, 71],
  [112, 136, 124, 119],
  [126, 152, 139, 133],
  [122, 148, 135, 146],
  [127, 153, 140, 138],
  [119, 145, 132, 151],
  [94, 116, 105, 109],
  [61, 79, 70, 66],
  [115, 139, 127, 131],
  [128, 154, 141, 136],
];

function recordOf(start) {
  const today = start - (start % DAY);
  return RECORD.map(([low, high, mean, actual], index) => ({
    dayStart: today - (RECORD.length - index) * DAY,
    low,
    high,
    mean,
    actual,
  }));
}

const REACH = [
  ["wrk_ana_ruiz", "Ana Ruiz", "Toledo, OH", 41.6528, -83.5379, 112, 72_600, "OnTime"],
  ["wrk_marcus_hill", "Marcus Hill", "Pittsburgh, PA", 40.4406, -79.9959, 133, 65_400, "OnTime"],
  ["wrk_sam_whitfield", "Sam Whitfield", "Akron, OH", 41.0814, -81.519, 39, 58_200, "OnTime"],
  ["wrk_jo_bianchi", "Jo Bianchi", "Fort Wayne, IN", 41.0793, -85.1394, 208, -6_600, "NotWorth"],
  ["wrk_ty_nakamura", "Ty Nakamura", "Cincinnati, OH", 39.1031, -84.512, 249, 31_200, "TooLate"],
];

/** The outlook as `DispatchFreightOutlook` returns it, with the sends already made. */
export function buildFreightOutlook(now, positionings) {
  const start = now - (now % SLOT);
  const starts = Array.from({ length: WINDOWS }, (_, index) => start + index * SLOT);
  const sent = new Set(
    positionings.filter((row) => row.status === "Active").map((row) => row.workerId),
  );
  return {
    generatedAt: now - 540,
    quality: {
      status: "Ready",
      historyWeeks: 35,
      historyLoads: 22_400,
      weeksTested: 8,
      loadsTested: 9_800,
      intervalCoverage: 0.86,
      meanAbsoluteError: 0.9,
      naiveMeanAbsoluteError: 1.22,
      skill: 0.26,
      fittedAt: start - 5 * 3_600,
      refitting: false,
    },
    slotStarts: starts,
    network: networkOf(starts),
    areas: areasOf(starts),
    areaCount: Object.keys(CELL).length,
    moves: movesOf(starts).filter((move) => move.kind !== "Send" || !sent.has(move.workerId)),
    pace: paceOf(starts),
    record: recordOf(start),
    positionings: positionings.filter((row) => row.status === "Active"),
  };
}

/** One area as `DispatchFreightArea` returns it: who can reach it, its moves and who books there. */
export function buildFreightArea(now, positionings, cell, slotStart) {
  const outlook = buildFreightOutlook(now, positionings);
  const area = outlook.areas.find((row) => row.cell === cell);
  if (!area) return null;
  const worst = area.slots.reduce(
    (best, slot) => (slot.short > best.short ? slot : best),
    area.slots[0],
  );
  const slot = area.slots.find((row) => row.start === slotStart) ?? worst;
  // The same window of the week in each of the last eight weeks, oldest first.
  const recent = [-1, 1, 0, 2, -2, 1, 0, 2].map((drift) => Math.max(0, slot.need + drift));
  return {
    generatedAt: outlook.generatedAt,
    quality: outlook.quality,
    area,
    bounds: {
      south: area.latitude - 0.45,
      west: area.longitude - 0.6,
      north: area.latitude + 0.45,
      east: area.longitude + 0.6,
    },
    slotStart: slot.start,
    slotEnd: slot.end,
    reach: REACH.map(
      ([workerId, workerName, label, latitude, longitude, miles, netCents, reach]) => ({
        workerId,
        workerName,
        label,
        latitude,
        longitude,
        freeAt: slot.start - SLOT + 1_800,
        arriveAt: reach === "TooLate" ? slot.end + 1_800 : slot.start,
        miles,
        netCents,
        reach,
      }),
    ),
    moves: outlook.moves.filter((move) => move.toCell === cell || move.fromCell === cell),
    pace: outlook.pace.filter((row) => row.cell === cell),
    recent,
    recentLow: Math.max(0, slot.need - 2),
    recentHigh: slot.need + 2,
    deliveryLocationIds: ["loc_outlook_dock_1", "loc_outlook_dock_2"],
    positionings: outlook.positionings.filter((row) => row.toCell === cell),
  };
}

/** The driver already on the way when the page opens. */
export function seedPositionings(now) {
  const start = now - (now % SLOT);
  return [
    {
      id: "dpo_outlook_1",
      workerId: "wrk_sam_whitfield",
      workerName: "Sam Whitfield",
      fromLabel: "Akron, OH",
      toCell: CELL.cleveland,
      toLabel: "Cleveland, OH",
      slotStart: start + SLOT,
      freeAt: now - 1_200,
      arriveAt: start + SLOT,
      miles: 39,
      valueCents: 58_200,
      status: "Active",
      createdAt: now - 900,
    },
  ];
}
