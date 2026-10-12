// The Desk's resource cards: the artifact a conversation holds, and the
// GraphQL each card sends for its record and its sections. The `desk` scenario
// field picks what the cards read: "full" (the fixture), "empty" (every
// section empty) or "hold" (every card request left open, for the skeleton).
import {
  DESK_THREAD_ID,
  deskCardArtifacts,
  deskDocument,
  deskThread,
  deskTractors,
  deskTrailers,
  deskLocations,
  deskWorkers,
} from "../fixtures/desk-cards.mjs";
import { hash } from "../state.mjs";

const HOLD = { __hold: true };

function tractorFixture(state, id) {
  return deskTractors(state.anchor)[id] ?? null;
}

function held(state) {
  return state.scenario.desk === "hold";
}

function empty(state) {
  return state.scenario.desk === "empty";
}

/** A shipment from the board, drawn as the card's shipment list reads it. */
function recordShipmentNode(shipment) {
  return {
    id: shipment.id,
    proNumber: shipment.proNumber,
    status: shipment.status,
    totalChargeAmount: shipment.totalChargeAmount,
    moves: shipment.moves.map((move) => ({
      sequence: move.sequence,
      stops: move.stops.map((stop) => ({
        sequence: stop.sequence,
        type: stop.type,
        scheduledWindowStart: stop.scheduledWindowStart,
        location: { city: stop.location.city, state: { abbreviation: stop.location.state } },
      })),
    })),
  };
}

function trailerFixture(state, id) {
  return deskTrailers(state.anchor)[id] ?? null;
}

function workerFixture(state, id) {
  return deskWorkers(state.anchor)[id] ?? null;
}

function locationFixture(state, id) {
  return deskLocations(state.anchor)[id] ?? null;
}

/** A page of a fixture's rows, read the way a connection pages them. */
function connectionPage(rows, input, includeTotalCount) {
  const offset = input?.after ? Number(Buffer.from(input.after, "base64url").toString()) : 0;
  const first = input?.first ?? 20;
  const page = rows.slice(offset, offset + first);
  const end = offset + page.length;
  return {
    edges: page.map((node) => ({ node })),
    pageInfo: {
      hasNextPage: end < rows.length,
      endCursor: page.length ? Buffer.from(String(end)).toString("base64url") : null,
    },
    ...(includeTotalCount ? { totalCount: rows.length } : {}),
  };
}

/** The board's shipments a unit or worker moved: a fixed handful each, newest pickup first. */
function unitShipments(state, unitId) {
  const fixture =
    tractorFixture(state, unitId) ??
    trailerFixture(state, unitId) ??
    workerFixture(state, unitId) ??
    locationFixture(state, unitId);
  if (!fixture || empty(state)) return [];
  return [...state.shipments]
    .filter((shipment) => hash(shipment.id + unitId) % 4 === 0)
    .sort(
      (a, b) => b.moves[0].stops[0].scheduledWindowStart - a.moves[0].stops[0].scheduledWindowStart,
    )
    .slice(0, fixture.shipments);
}

function recordShipments(state, variables) {
  if (held(state)) return HOLD;
  const input = variables.input ?? {};
  const unitId =
    input.tractorIds?.[0] ??
    input.trailerIds?.[0] ??
    input.workerIds?.[0] ??
    input.stopLocationIds?.[0];
  const all = unitId ? unitShipments(state, unitId) : [];
  const offset = input.after ? Number(Buffer.from(input.after, "base64url").toString()) : 0;
  const first = input.first ?? 20;
  const page = all.slice(offset, offset + first);
  const end = offset + page.length;
  return {
    shipments: {
      edges: page.map((shipment) => ({ node: recordShipmentNode(shipment) })),
      pageInfo: {
        hasNextPage: end < all.length,
        endCursor: page.length ? Buffer.from(String(end)).toString("base64url") : null,
      },
      ...(variables.includeTotalCount ? { totalCount: all.length } : {}),
    },
  };
}

function tractorInspections(state, variables) {
  if (held(state)) return HOLD;
  const fixture = tractorFixture(state, variables.tractorId);
  const rows = fixture && !empty(state) ? fixture.inspections : [];
  const after = variables.before
    ? rows.filter(
        (row) =>
          row.startedAt < variables.before ||
          (row.startedAt === variables.before && variables.beforeId && row.id < variables.beforeId),
      )
    : rows;
  return { vehicleInspections: after.slice(0, variables.limit ?? after.length) };
}

/** The agent the Desk cards conversation is with, as the person's own agents list it. */
const DISPATCH_AGENT = {
  id: "agd_dispatch",
  name: "Dispatch desk",
  description: "Follows loads, drivers and equipment for the dispatch team.",
  template: "dispatch",
  icon: "",
  accent: "",
  toolNames: [
    "get_tractor",
    "get_trailer",
    "get_worker",
    "get_location",
    "list_tractors",
    "get_shipment",
  ],
  systemKey: "",
  starters: [],
  delegates: [],
};

export const DESK_CARD_HANDLERS = {
  MyAgents: () => ({
    myAgents: {
      edges: [{ node: DISPATCH_AGENT }],
      totalCount: 1,
      pageInfo: { hasNextPage: false, endCursor: null, hasPreviousPage: false, startCursor: null },
    },
  }),
  DeskTractorCard: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = tractorFixture(state, variables.id);
    if (!fixture) return { tractor: null };
    if (!empty(state)) return { tractor: fixture.tractor };
    return {
      tractor: {
        ...fixture.tractor,
        secondaryWorker: null,
        lastKnownLocation: null,
      },
    };
  },
  DeskTractorPosition: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = tractorFixture(state, variables.tractorId);
    return { vehiclePosition: fixture && !empty(state) ? fixture.position : null };
  },
  DeskTractorInspections: tractorInspections,
  DeskTrailerCard: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = trailerFixture(state, variables.id);
    return { trailer: fixture ? fixture.trailer : null };
  },
  DeskWorkerCard: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = workerFixture(state, variables.id);
    return { worker: fixture ? fixture.worker : null };
  },
  DeskWorkerHos: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = workerFixture(state, variables.workerId);
    if (!fixture || empty(state)) return { workerHosState: null, workerHosViolations: [] };
    return {
      workerHosState: fixture.hos.state,
      workerHosViolations: fixture.hos.violations.slice(0, variables.violations),
    };
  },
  DeskWorkerStanding: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = workerFixture(state, variables.workerId);
    return {
      workerOverview:
        fixture && !empty(state) ? fixture.standing : { standing: "Good", concerns: [] },
    };
  },
  DeskWorkerCredentials: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = workerFixture(state, variables.workerId);
    return {
      workerCredentialSummary: { items: fixture && !empty(state) ? fixture.credentials : [] },
    };
  },
  DeskLocationCard: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = locationFixture(state, variables.id);
    return { location: fixture ? fixture.location : null };
  },
  DeskLocationDwell: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = locationFixture(state, variables.locationId);
    return {
      detentionFacilityStats: fixture?.dwell && !empty(state) ? [fixture.dwell] : [],
    };
  },
  DeskLocationPolicies: (state, variables) => {
    if (held(state)) return HOLD;
    const locationId = variables.input?.fieldFilters?.find(
      (filter) => filter.field === "locationId",
    )?.value;
    const fixture = locationFixture(state, locationId);
    const rows = fixture && !empty(state) ? fixture.policies : [];
    const shown = rows.slice(0, variables.input?.first ?? rows.length);
    return {
      detentionPolicies: {
        totalCount: fixture?.policyTotal ?? rows.length,
        edges: shown.map((node) => ({ node })),
      },
    };
  },
  DeskLocationFailures: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = locationFixture(state, variables.locationId);
    const rows = fixture && !empty(state) ? fixture.failures : [];
    return {
      serviceFailures: connectionPage(rows, variables.input, variables.includeTotalCount),
    };
  },
  DeskRecordShipments: recordShipments,
};

/** The card the conversation holds: the one the screenshot asked for with `deskSlug`. */
function threadSlug(state) {
  return state.scenario.deskSlug ?? "tractor-t-101";
}

export const DESK_CARD_ROUTES = [
  [
    "GET",
    /^\/api\/v1\/assistant\/providers\/?$/,
    () => ({
      results: [
        {
          id: "aip_main",
          name: "Claude",
          kind: "Anthropic",
          model: "claude-sonnet-5-5",
          trusted: true,
        },
      ],
    }),
  ],
  [
    "GET",
    new RegExp(`^/api/v1/assistant/threads/${DESK_THREAD_ID}/artifacts/(art_[a-z0-9-]+)/?$`),
    (state, path) => {
      const id = path.match(/artifacts\/(art_[a-z0-9-]+)/)?.[1] ?? "";
      const artifact = Object.values(deskCardArtifacts(state.anchor)).find(
        (entry) => entry.id === id,
      );
      return { results: artifact ? [artifact] : [] };
    },
  ],
  [
    "GET",
    new RegExp(`^/api/v1/assistant/threads/${DESK_THREAD_ID}/?$`),
    (state) => deskThread(state.anchor, threadSlug(state)).thread,
  ],
  [
    "GET",
    new RegExp(`^/api/v1/assistant/threads/${DESK_THREAD_ID}/messages/?$`),
    (state) => {
      const { messages } = deskThread(state.anchor, threadSlug(state));
      return { results: messages, hasMore: false, total: messages.length, limit: 50 };
    },
  ],
  [
    "GET",
    new RegExp(`^/api/v1/assistant/threads/${DESK_THREAD_ID}/artifacts/?$`),
    (state) => {
      const artifact = deskCardArtifacts(state.anchor)[threadSlug(state)];
      return {
        results: artifact ? [artifact] : [],
        total: artifact ? 1 : 0,
        nextCursor: "",
        counts: { all: artifact ? 1 : 0, pinned: 0, families: { record: artifact ? 1 : 0 } },
      };
    },
  ],
  [
    "GET",
    new RegExp(`^/api/v1/assistant/threads/${DESK_THREAD_ID}/artifacts/by-slug/([a-z0-9-]+)/?$`),
    (state, path) => {
      const slug = path.match(/by-slug\/([a-z0-9-]+)/)?.[1] ?? "";
      const artifact = deskCardArtifacts(state.anchor)[slug];
      return { results: artifact ? [artifact] : [] };
    },
  ],
  [
    "GET",
    /^\/api\/v1\/documents\/resource\/(tractor|trailer|worker)\/([a-z0-9_]+)\/?$/,
    (state, path) => {
      if (held(state)) return HOLD;
      const [, kind, id] = path.match(/resource\/(tractor|trailer|worker)\/([a-z0-9_]+)/) ?? [];
      const fixture =
        kind === "tractor"
          ? tractorFixture(state, id)
          : kind === "trailer"
            ? trailerFixture(state, id)
            : workerFixture(state, id);
      if (!fixture || empty(state)) return [];
      return fixture.documents.map((file, index) =>
        deskDocument(kind, id, file, index, state.anchor),
      );
    },
  ],
];
