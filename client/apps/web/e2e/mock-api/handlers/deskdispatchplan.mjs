// A dispatch plan on the Desk: the Dispatch desk asked which drivers should take
// six new loads, and the dispatch_plan tool answered. Written from the server's
// contract: the artifact payload toolartifacts/dispatch_plan_artifact.go stores,
// and the board's move fragment (DispatchBoardMoveFields) that dispatchMovesByIds
// reads each load back with. Every time is relative to the anchor.
//
// The `deskDispatchPlan` scenario field picks where the loads stand now: "plan"
// (nothing assigned), "proposed" (an agent proposed the planned drivers),
// "assigned" (the planned drivers are on them, one load taken by someone else),
// or "none" (no driver can take any of them). Assigning from the card moves
// "plan" or "proposed" to "assigned".

const THREAD = "athr_desk_dispatch_plan";
const ARTIFACT = "aart_dispatch_plan";
const SLUG = "drivers-for-6-loads";
const QUESTION =
  "Tell me what workers I should assign to these new shipments, and show it side by side.";
const STAGES = ["plan", "proposed", "assigned", "none"];

function stageOf(state) {
  const stage = state.scenario.deskDispatchPlan ?? "plan";
  return STAGES.includes(stage) ? stage : "plan";
}

const LOADS = [
  {
    move: "sm_dpl_omaha",
    pro: "S26110993231548",
    origin: ["Omaha", "NE"],
    dest: ["Dayton", "OH"],
    pickupIn: 20,
    worker: ["wrk_dpl_adam", "Adam Stephens"],
    tractor: ["trc_dpl_104", "T-104"],
    cost: "$412",
    deadhead: 61.2,
    drive: 9.5,
    nextBest: [
      ["Cynthia Daniels", "$126"],
      ["Miguel Graham", "$199"],
    ],
  },
  {
    move: "sm_dpl_springs",
    pro: "S26110991834141",
    origin: ["Colorado Springs", "CO"],
    dest: ["Cincinnati", "OH"],
    pickupIn: 42,
    worker: ["wrk_dpl_robert", "Robert Brown"],
    tractor: ["trc_dpl_212", "T-212"],
    cost: "$455",
    deadhead: 72,
    drive: 11,
    nextBest: [["Jacob Kowalski", "$247"]],
  },
  {
    move: "sm_dpl_cheyenne",
    pro: "S26110997745533",
    origin: ["Cheyenne", "WY"],
    dest: ["Joliet", "IL"],
    pickupIn: 62,
    worker: ["wrk_dpl_jose", "Jose Daniels"],
    tractor: ["trc_dpl_330", "T-330"],
    cost: "$1,190",
    deadhead: 440,
    drive: null,
    nextBest: [["Ashley Rose", "$212"]],
  },
  {
    move: "sm_dpl_fortworth",
    pro: "S26110989642549",
    origin: ["Fort Worth", "TX"],
    dest: ["Greensboro", "NC"],
    pickupIn: 160,
    worker: ["wrk_dpl_george", "George Thompson"],
    tractor: ["trc_dpl_118", "T-118"],
    cost: "$640",
    deadhead: 188,
    drive: 11,
    nextBest: [
      ["Mary Kowalski", "$15"],
      ["Carlos Rivera", "$80"],
    ],
  },
];

const GAPS = [
  {
    move: "sm_dpl_stockton",
    pro: "S26110985022175",
    origin: ["Stockton", "CA"],
    dest: ["Fresno", "CA"],
    pickupIn: 190,
    reason:
      "The nearest driver, Patricia Ferguson, can't take it: Not enough drive hours for this trip under any rest plan that fits before departure.",
    findings: [
      {
        code: "hos.insufficient_for_trip",
        severity: "Block",
        message:
          "Not enough drive hours for this trip under any rest plan that fits before departure",
      },
    ],
  },
  {
    move: "sm_dpl_fargo",
    pro: "S26110998394820",
    origin: ["Fargo", "ND"],
    dest: ["Los Angeles", "CA"],
    pickupIn: 192,
    reason: "No driver could take this move.",
    findings: [],
  },
];

function lineOf(load, anchor) {
  return {
    shipmentMoveId: load.move,
    proNumber: load.pro,
    lane: `${load.origin.join(", ")} to ${load.dest.join(", ")}`,
    pickupOpens: "",
    pickupAt: anchor + load.pickupIn * 3600,
    workerId: load.worker[0],
    workerName: load.worker[1],
    ...(load.drive === null ? {} : { driveRemainingHours: load.drive }),
    cost: load.cost,
    confidence: load.drive === null ? "0.62" : "0.91",
    rationale: `${load.worker[1]} costs the plan ${load.cost} to cover ${load.pro}`,
    deadheadMiles: load.deadhead,
    ...(load.drive === null
      ? { hours: "No ELD has reported this driver's hours, so they are unknown, not zero." }
      : {}),
    findings: [],
    nextBest: load.nextBest.map(([workerName, extraCost]) => ({ workerName, extraCost })),
  };
}

function payload(state) {
  const stage = stageOf(state);
  const lines = stage === "none" ? [] : LOADS.map((load) => lineOf(load, state.anchor));
  return {
    display: 1,
    tool: "dispatch_plan",
    planningMode: "Immediate",
    cost: stage === "none" ? "" : "$2,697",
    lines,
    uncovered: GAPS.map((gap) => ({
      shipmentMoveId: gap.move,
      proNumber: gap.pro,
      reason: gap.reason,
      findings: gap.findings,
    })),
    assignments:
      stage === "none"
        ? []
        : LOADS.map((load) => ({
            shipmentMoveId: load.move,
            primaryWorkerId: load.worker[0],
            tractorId: load.tractor[0],
            reason: `${load.worker[1]} is the cheapest cover`,
          })),
    notPlannedCount: 1,
  };
}

function title(state) {
  return stageOf(state) === "none" ? "Drivers for 2 loads" : "Drivers for 6 loads";
}

function artifact(state) {
  const at = state.anchor - 6 * 60;
  return {
    id: ARTIFACT,
    threadId: THREAD,
    messageId: "amsg_dpl_a1",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "dispatch_plan",
    status: "Ready",
    title: title(state),
    payload: payload(state),
    sourceToolCallId: "call_dpl",
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: SLUG,
    turn: QUESTION,
    createdAt: at,
    updatedAt: at,
  };
}

/** A load as the board's move fragment reads it, standing where the stage puts it. */
function boardMove(state, load, planned) {
  const stage = stageOf(state);
  const taken = stage === "assigned" && load.move === "sm_dpl_fortworth";
  const assigned = planned && stage === "assigned" && !taken;
  const proposed = planned && stage === "proposed";
  return {
    moveId: load.move,
    shipmentId: `shp_${load.move}`,
    proNumber: load.pro,
    bol: "",
    moveStatus: assigned || taken ? "Assigned" : "New",
    shipmentStatus: assigned || taken ? "Assigned" : "New",
    sequence: 0,
    moveCount: 1,
    loaded: true,
    distance: 700,
    revenue: 2400,
    customerId: "cus_dpl",
    customerName: "Moore Medical Supply Brands",
    serviceTypeId: "",
    urgency: "Soon",
    minutesToPickup: load.pickupIn * 60,
    isCovered: assigned || taken,
    originStopId: `stp_${load.move}_o`,
    originLocationId: "loc_o",
    originName: `${load.origin[0]} Distribution Center`,
    originCity: load.origin[0],
    originState: load.origin[1],
    originLatitude: 0,
    originLongitude: 0,
    originWindowStart: state.anchor + load.pickupIn * 3600,
    originWindowEnd: state.anchor + (load.pickupIn + 2) * 3600,
    originActualArrival: null,
    destinationStopId: `stp_${load.move}_d`,
    destinationLocationId: "loc_d",
    destinationName: `${load.dest[0]} Warehouse`,
    destinationCity: load.dest[0],
    destinationState: load.dest[1],
    destinationLatitude: 0,
    destinationLongitude: 0,
    destinationWindowStart: state.anchor + (load.pickupIn + 30) * 3600,
    destinationWindowEnd: state.anchor + (load.pickupIn + 32) * 3600,
    assignmentId: assigned || taken ? `asn_${load.move}` : null,
    assignedWorkerId: taken ? "wrk_dpl_lee" : assigned ? load.worker[0] : null,
    assignedWorkerName: taken ? "Lee Park" : assigned ? load.worker[1] : null,
    assignedTractorId: assigned ? load.tractor[0] : null,
    assignedTractorCode: assigned ? load.tractor[1] : null,
    assignedTrailerId: null,
    assignedTrailerCode: null,
    assignmentAckStatus: null,
    previousMoveTrailerId: null,
    coverageType: assigned || taken ? "driver" : "unassigned",
    carrierAssignmentId: null,
    assignedCarrierId: null,
    assignedCarrierName: null,
    carrierTotalCost: null,
    liveTender: null,
    pendingProposal: proposed
      ? {
          id: `aprop_${load.move}`,
          workerId: load.worker[0],
          workerName: load.worker[1],
          tractorId: load.tractor[0],
          trailerId: null,
          confidence: 0.9,
          rationale: "",
          createdAt: state.anchor - 300,
          expiresAt: state.anchor + 86400,
          stale: false,
        }
      : null,
    failedProposal: null,
    planPrice: null,
  };
}

function messages(state) {
  const asked = state.anchor - 7 * 60;
  return [
    {
      id: "amsg_dpl_u1",
      threadId: THREAD,
      sequence: 1,
      role: "User",
      content: QUESTION,
      kind: "Message",
      toolFailed: false,
      refused: false,
      createdAt: asked,
    },
    {
      id: "amsg_dpl_a1",
      threadId: THREAD,
      sequence: 2,
      role: "Assistant",
      content: `The optimizer planned the six loads together, so no driver is on two of them: [${title(state)}](artifact:${ARTIFACT}). Four have a driver; Stockton and Fargo have nobody who can take them as things stand. Assign the four from the card, or tell me to propose them.`,
      kind: "Message",
      toolFailed: false,
      refused: false,
      createdAt: asked + 60,
    },
  ];
}

function thread(state) {
  const list = messages(state);
  return {
    id: THREAD,
    businessUnitId: "bu_mock",
    organizationId: "org_mock",
    userId: "usr_mock",
    agentDefinitionId: "agdef_dispatch",
    title: QUESTION,
    status: "Active",
    lastMessageAt: list.at(-1).createdAt,
    lastReadAt: state.anchor,
    origin: "Desk",
    pinned: false,
    canContinue: true,
    version: 2,
    createdAt: list[0].createdAt,
    updatedAt: list.at(-1).createdAt,
  };
}

/** The conversation this file serves, for the rail's list and shelf counts. */
export function deskDispatchPlanThreads(state) {
  return [thread(state)];
}

const route = (rest = "") => new RegExp(`^/api/v1/assistant/threads/${THREAD}${rest}/?$`);

export const DESK_DISPATCH_PLAN_ROUTES = [
  ["GET", route(), thread],
  [
    "GET",
    route("/messages"),
    (state) => {
      const list = messages(state);
      return { results: list, hasMore: false, total: list.length, limit: 50 };
    },
  ],
  [
    "GET",
    route("/artifacts"),
    (state) => ({
      results: [artifact(state)],
      total: 1,
      nextCursor: "",
      counts: { all: 1, pinned: 0, families: { plan: 1 } },
    }),
  ],
  [
    "GET",
    route("/artifacts/by-slug/([a-z0-9-]+)"),
    (state, path) => {
      const slug = path.match(/by-slug\/([a-z0-9-]+)/)?.[1] ?? "";
      return { results: slug === SLUG ? [artifact(state)] : [] };
    },
  ],
  [
    "GET",
    route("/artifacts/(aart_[a-z0-9_]+)"),
    (state, path) => {
      const id = path.match(/artifacts\/(aart_[a-z0-9_]+)/)?.[1] ?? "";
      return { results: id === ARTIFACT ? [artifact(state)] : [] };
    },
  ],
  ["GET", route("/proposals"), () => ({ results: [] })],
  ["GET", route("/plans"), () => ({ results: [] })],
  ["GET", route("/schedules"), () => ({ items: [], total: 0 })],
  ["GET", route("/waits"), () => ({ items: [] })],
  ["GET", route("/queue"), () => ({ items: [] })],
  ["GET", route("/turns/active"), () => ({ turn: null })],
];

const PLANNED = new Set(LOADS.map((load) => load.move));

export const DESK_DISPATCH_PLAN_HANDLERS = {
  DispatchMovesByIds: (state, variables) => {
    const wanted = new Set(variables.moveIds ?? []);
    const moves = [...LOADS, ...GAPS]
      .filter((load) => wanted.has(load.move))
      .map((load) => boardMove(state, load, PLANNED.has(load.move) && stageOf(state) !== "none"));
    return { dispatchMovesByIds: moves };
  },
};

/**
 * The plan's own loads assigned from its card, or null for any other move so the board's
 * handler keeps them.
 */
export function assignDispatchPlanMoves(state, input) {
  if (!input.length || !input.every((item) => PLANNED.has(item.moveId))) {
    return null;
  }
  state.scenario.deskDispatchPlan = "assigned";
  return {
    dispatchAssignMoves: {
      succeeded: input.length,
      failed: 0,
      results: input.map((item) => ({
        moveId: item.moveId,
        success: true,
        assignmentId: `asn_${item.moveId}`,
        error: null,
        findings: [],
      })),
    },
  };
}
