import { buildShipments, CARRIERS, createDriverPool } from "./fixtures/board.mjs";

export const SCENARIO_DEFAULTS = {
  // quiet | busy | high (480 rows)
  board: process.env.MOCK_BOARD ?? "high",
  ai: process.env.MOCK_AI !== "0",
  // asset | brokerage | both
  operationType: process.env.MOCK_OPERATION_TYPE ?? "both",
  hos: process.env.MOCK_HOS !== "0",
  maps: process.env.MOCK_MAPS === "1",
  // rows in the generic list pages (hazardous materials), for checking a short and a long list
  listRows: Number(process.env.MOCK_LIST_ROWS ?? 3),
};

const STAGE_RANK = { Late: 1, NeedsCoverage: 2, Moving: 3, Scheduled: 4, Delivered: 5, Canceled: 6 };
const STAGE_OF_STATUS = {
  Delayed: "Late",
  New: "NeedsCoverage",
  PartiallyAssigned: "NeedsCoverage",
  InTransit: "Moving",
  PartiallyCompleted: "Moving",
  Assigned: "Scheduled",
  Completed: "Delivered",
  ReadyToInvoice: "Delivered",
  Invoiced: "Delivered",
  Canceled: "Canceled",
};

export function stageOf(shipment) {
  return STAGE_OF_STATUS[shipment.status] ?? "NeedsCoverage";
}

export function stageRankOf(shipment) {
  return STAGE_RANK[stageOf(shipment)];
}

export const STAGE_RANKS = STAGE_RANK;

/** The board's "now": 11:30 UTC today, the hour the handoff's prototype is drawn at. */
function anchorTime() {
  const now = Math.floor(Date.now() / 1000);
  return now - (now % 86400) + 11.5 * 3600;
}

export function createState(scenario) {
  const anchor = anchorTime();
  const startedAt = Date.now();
  const shipments = buildShipments(scenario.board, anchor);
  if (scenario.operationType === "brokerage") {
    convertToBrokerage(shipments);
  } else if (scenario.operationType === "both") {
    shipments.forEach((shipment, index) => {
      if (index % 3 === 0) convertToBrokerage([shipment]);
    });
  }
  const pool = createDriverPool();
  return {
    scenario,
    anchor,
    now: () => anchor + Math.floor((Date.now() - startedAt) / 1000),
    shipments,
    drivers: scenario.board === "quiet" ? pool.slice(0, 4) : scenario.board === "high" ? pool : pool.slice(0, 9),
    carriers: CARRIERS.map((carrier) => ({ ...carrier })),
    usedDrivers: new Set(),
    decisions: new Map(),
    handledThisShift: scenario.board === "quiet" ? 0 : 7,
    billedDetention: new Set(),
    events: seedEvents(shipments, anchor),
  };
}

function convertToBrokerage(shipments) {
  for (const shipment of shipments) {
    const move = shipment.moves[0];
    if (!move.assignment) continue;
    const carrier = CARRIERS[hash(shipment.id) % CARRIERS.length];
    move.coverageType = "carrier";
    move.carrierAssignment = carrierAssignment(move, carrier, shipment.board.driverName);
    move.assignment = null;
  }
}

export function carrierAssignment(move, carrier, driverName = null) {
  return {
    id: `cas_${move.id}`,
    businessUnitId: "bu_mock",
    organizationId: "org_mock",
    shipmentMoveId: move.id,
    carrierId: carrier.id,
    status: "Confirmed",
    rateMethod: "PerMile",
    baseRate: String(carrier.rate),
    baseAmount: String(Math.round(move.distance * carrier.rate)),
    fuelSurcharge: "0",
    accessorialTotal: "0",
    totalCost: String(Math.round(move.distance * carrier.rate)),
    currencyCode: "USD",
    proNumber: null,
    externalDriverName: driverName,
    externalDriverPhone: null,
    externalTractorNumber: null,
    externalTrailerNumber: null,
    confirmedAt: null,
    canceledAt: null,
    cancellationReason: null,
    version: 1,
    createdAt: 0,
    updatedAt: 0,
    carrier: { id: carrier.id, code: carrier.scac, name: carrier.name, scac: carrier.scac },
    accessorials: [],
  };
}

export function hash(id) {
  return [...id].reduce((acc, ch) => (acc * 31 + ch.charCodeAt(0)) % 9973, 7);
}

function seedEvents(shipments, anchor) {
  const find = (pro) => shipments.find((s) => s.proNumber === pro);
  const rows = [
    ["S2610-0412", "StatusChanged", "danger", "system", "Trenova", "ETA pushed to 18:40", 2],
    ["S2610-0398", "MoveArrived", "brand", "user", "Kai Whitehorse", "Arrived at Cargill Protein Plant", 14],
    ["S2610-0366", "StopCompleted", "success", "user", "Ana Reyes", "Delivered · POD signed", 60],
    ["S2610-0431", "ShipmentCreated", "info", "user", "Eric Moss", "Created shipment", 180],
    ["S2610-0430", "ShipmentCreated", "info", "user", "Eric Moss", "Created shipment", 182],
    ["S2610-0415", "TenderDeclined", "danger", "edi", "Werner", "Declined the tender", 240],
  ];
  return rows
    .map(([pro, type, severity, actorType, actorLabel, summary, minutesAgo], i) => {
      const shipment = find(pro);
      if (!shipment) return null;
      return {
        __typename: "ShipmentLifecycleEvent",
        id: `evt_${i}`,
        organizationId: "org_mock",
        businessUnitId: "bu_mock",
        shipmentId: shipment.id,
        type,
        severity,
        actorType,
        actorId: null,
        actorLabel,
        summary,
        metadata: {},
        occurredAt: anchor - minutesAgo * 60,
        correlationId: null,
        actor: null,
        shipment: { id: shipment.id, proNumber: shipment.proNumber },
        proNumber: shipment.proNumber,
        previousStatus: null,
        newStatus: null,
        reason: null,
      };
    })
    .filter(Boolean);
}
