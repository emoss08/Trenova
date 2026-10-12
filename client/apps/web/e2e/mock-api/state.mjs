import { createAIState } from "./fixtures/aicontrol.mjs";
import { buildShipments, CARRIERS, createDriverPool } from "./fixtures/board.mjs";
import { createTriageState } from "./handlers/triage-model.mjs";
import { createSettingsState } from "./fixtures/settings.mjs";

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
  // AI control: configured (the prototype's three) | many | none
  aiProviders: process.env.MOCK_AI_PROVIDERS ?? "configured",
  // every agent paused in shadow
  aiPaused: process.env.MOCK_AI_PAUSED === "1",
  // Triage: live (the prototype's queue, a new message every 26s) | firstDay | noMailbox | deskOff
  triage: process.env.MOCK_TRIAGE ?? "live",
  // Triage clock: afternoon (13:39, the prototype's NOW) | endOfDay (17:20, after the 17:00 wrap-up)
  triageTime: process.env.MOCK_TRIAGE_TIME ?? "afternoon",
  // how long after "Wait for reply" the sender answers on the thread (app.jsx waits 6.5s)
  triageReplyAfterMs: Number(process.env.MOCK_TRIAGE_REPLY_MS ?? 6500),
  // Desk resource cards: full | empty | hold (every card request left open, for the skeleton)
  desk: process.env.MOCK_DESK ?? "full",
  // the card the Desk cards conversation holds
  deskSlug: process.env.MOCK_DESK_SLUG ?? "tractor-t-101",
  // the Desk's live conversation: idle | working (a reply under way, steered, follow-ups queued)
  deskReply: process.env.MOCK_DESK_REPLY ?? "idle",
};

const TRIAGE_SCENARIOS = new Set(["live", "firstDay", "noMailbox", "deskOff"]);
const TRIAGE_TIMES = new Set(["afternoon", "endOfDay"]);

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

/** Triage's "now": 13:39 UTC today (the prototype's NOW), or 17:20 for the end of the day. */
function triageAnchorTime(triageTime) {
  const now = Math.floor(Date.now() / 1000);
  return now - (now % 86400) + (triageTime === "endOfDay" ? 17 * 3600 + 20 * 60 : 13 * 3600 + 39 * 60);
}

export function createState(requested) {
  const scenario = {
    ...requested,
    triage: TRIAGE_SCENARIOS.has(requested.triage) ? requested.triage : "live",
    triageTime: TRIAGE_TIMES.has(requested.triageTime) ? requested.triageTime : "afternoon",
  };
  const anchor = anchorTime();
  const triageAnchor = triageAnchorTime(scenario.triageTime);
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
  const triage = createTriageState({ anchor: triageAnchor, scenario });
  return {
    scenario,
    anchor,
    triageAnchor,
    triage,
    dispose: () => triage.dispose(),
    now: () => anchor + Math.floor((Date.now() - startedAt) / 1000),
    ai: createAIState(anchor, scenario),
    settings: createSettingsState(anchor),
    shipments,
    drivers: scenario.board === "quiet" ? pool.slice(0, 4) : scenario.board === "high" ? pool : pool.slice(0, 9),
    carriers: CARRIERS.map((carrier) => ({ ...carrier })),
    usedDrivers: new Set(),
    decisions: new Map(),
    handledThisShift: scenario.board === "quiet" ? 0 : 7,
    billedDetention: new Set(),
    brief: null,
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
