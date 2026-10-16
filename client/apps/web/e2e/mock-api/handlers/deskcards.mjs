// The Desk's resource cards: the artifact a conversation holds, and the
// GraphQL each card sends for its record and its sections. The `desk` scenario
// field picks what the cards read: "full" (the fixture), "empty" (every
// section empty) or "hold" (every card request left open, for the skeleton).
import {
  DESK_RECORD_NAMES,
  DESK_THREAD_ID,
  deskRateAgreements,
  deskServiceFailures,
  laneSummary,
  deskCardArtifacts,
  deskDocument,
  deskThread,
  deskTractors,
  deskTrailers,
  deskCarriers,
  deskLocations,
  deskOrders,
  deskWorkers,
} from "../fixtures/desk-cards.mjs";
import { deskEdiMessages } from "../fixtures/desk-edi.mjs";
import { deskEdiInboundFiles } from "../fixtures/desk-edi-files.mjs";
import { DESK_EDI_FILE_TENDERS, deskEdiTransfers } from "../fixtures/desk-edi-transfers.mjs";
import { deskInvoices } from "../fixtures/desk-invoices.mjs";
import { deskCarrierSettlements, deskDriverSettlements } from "../fixtures/desk-settlements.mjs";
import { deskDetentions } from "../fixtures/desk-detention.mjs";
import { deskEdiPartners } from "../fixtures/desk-edi-partners.mjs";
import { hash } from "../state.mjs";
import { TriageError } from "./triage-model.mjs";
import { userSelectOptions } from "./settings.mjs";

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

function carrierFixture(state, id) {
  return deskCarriers(state.anchor)[id] ?? null;
}

function orderFixture(state, id) {
  return deskOrders(state.anchor)[id] ?? null;
}

function failureFixture(state, id) {
  return deskServiceFailures(state.anchor)[id] ?? null;
}

/** The fixture failure recorded on a shipment, whose load's failures the card lists. */
function loadFailures(state, shipmentId) {
  const fixture = Object.values(deskServiceFailures(state.anchor)).find(
    (entry) => entry.failure.shipmentId === shipmentId,
  );
  return fixture ? fixture.loadFailures : [];
}

function agreementFixture(state, id) {
  return deskRateAgreements(state.anchor)[id] ?? null;
}

/** The agreement a rate agreement route names, or the 404 the API answers for one that is gone. */
function agreementRoute(state, path) {
  const id = path.match(/rate-agreements\/([a-z0-9_]+)/)?.[1] ?? "";
  const fixture = agreementFixture(state, id);
  return fixture
    ? { fixture }
    : {
        missing: {
          __status: 404,
          body: { title: "Not Found", status: 404, detail: "Rate agreement not found" },
        },
      };
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
    locationFixture(state, unitId) ??
    carrierFixture(state, unitId) ??
    orderFixture(state, unitId);
  if (!fixture || empty(state)) return [];
  return [...state.shipments]
    .filter((shipment) => hash(shipment.id + unitId) % 4 === 0)
    .sort(
      (a, b) => b.moves[0].stops[0].scheduledWindowStart - a.moves[0].stops[0].scheduledWindowStart,
    )
    .slice(0, fixture.shipments);
}

/** The shipments an invoice bills: the one it names, or a page of the board for a consolidated one. */
function invoiceShipments(state, invoiceId) {
  const fixture = deskInvoices(state.anchor)[invoiceId];
  if (!fixture || empty(state)) return [];
  const count = Math.min(fixture.invoice.shipmentCount, 12);
  return [...state.shipments]
    .filter((shipment) => hash(shipment.id + invoiceId) % 3 === 0)
    .slice(0, count);
}

/** A settlement of either kind, as its card reads it. */
function settlementFixture(state, id) {
  return (
    deskDriverSettlements(state.anchor)[id] ?? deskCarrierSettlements(state.anchor)[id] ?? null
  );
}

/** The shipments a settlement pays, as many as it counts, from the board. */
function settlementShipments(state, settlementId) {
  const fixture = settlementFixture(state, settlementId);
  if (!fixture || empty(state)) return [];
  return [...state.shipments]
    .filter((shipment) => hash(shipment.id + settlementId) % 3 === 0)
    .slice(0, Math.min(fixture.settlement.shipmentCount, 12));
}

/**
 * The settlement with each load's share filled from the shipments the mock
 * answers for it, so the card's loads carry the amount the settlement pays.
 */
function settlementWithShares(state, settlementId, net) {
  const fixture = settlementFixture(state, settlementId);
  if (!fixture) return null;
  const record = fixture.settlement;
  const loads = settlementShipments(state, settlementId);
  const each = loads.length ? Math.round(record[net] / record.shipmentCount) : 0;
  return {
    ...record,
    loadShares: loads.map((shipment, index) => ({
      shipmentId: shipment.id,
      proNumber: shipment.proNumber,
      amountMinor: each + ((hash(shipment.id) % 9) - 4) * 1_250 * (index % 2 ? 1 : -1),
    })),
  };
}

/**
 * The board shipment a detention charge's stop belongs to, picked the same way
 * on every read so the card's shipment and its pro number agree.
 */
function detentionShipment(state, occurrenceId) {
  const shipments = state.shipments;
  return shipments.length ? shipments[hash(occurrenceId) % shipments.length] : null;
}

function scopedShipments(state, input) {
  const shipmentId = input.fieldFilters?.find(
    (filter) => filter.field === "id" && filter.operator === "eq",
  )?.value;
  if (shipmentId) return state.shipments.filter((shipment) => shipment.id === shipmentId);
  if (input.invoiceIds?.length) return invoiceShipments(state, input.invoiceIds[0]);
  if (input.driverSettlementIds?.length) {
    return settlementShipments(state, input.driverSettlementIds[0]);
  }
  if (input.carrierSettlementIds?.length) {
    return settlementShipments(state, input.carrierSettlementIds[0]);
  }
  return null;
}

function recordShipments(state, variables) {
  if (held(state)) return HOLD;
  const input = variables.input ?? {};
  const scoped = scopedShipments(state, input);
  if (scoped) {
    const offset = input.after ? Number(Buffer.from(input.after, "base64url").toString()) : 0;
    const page = scoped.slice(offset, offset + (input.first ?? 20));
    const end = offset + page.length;
    return {
      shipments: {
        edges: page.map((shipment) => ({ node: recordShipmentNode(shipment) })),
        pageInfo: {
          hasNextPage: end < scoped.length,
          endCursor: page.length ? Buffer.from(String(end)).toString("base64url") : null,
        },
        ...(variables.includeTotalCount ? { totalCount: scoped.length } : {}),
      },
    };
  }
  const unitId =
    input.tractorIds?.[0] ??
    input.trailerIds?.[0] ??
    input.workerIds?.[0] ??
    input.stopLocationIds?.[0] ??
    input.carrierIds?.[0] ??
    input.fieldFilters?.find((filter) => filter.field === "orderId")?.value;
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
  template: "DispatchAssistant",
  icon: "",
  accent: "",
  toolNames: [
    "get_tractor",
    "get_trailer",
    "get_worker",
    "get_location",
    "get_carrier",
    "get_order",
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
  DeskRecordFailures: (state, variables) => {
    if (held(state)) return HOLD;
    const rows = empty(state)
      ? []
      : variables.shipmentId
        ? loadFailures(state, variables.shipmentId)
        : (locationFixture(state, variables.locationId)?.failures ?? []);
    return {
      serviceFailures: connectionPage(rows, variables.input, variables.includeTotalCount),
    };
  },
  DeskServiceFailureCard: (state, variables) => {
    if (held(state)) return HOLD;
    return { serviceFailure: failureFixture(state, variables.id)?.failure ?? null };
  },
  DeskCarrierCard: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = carrierFixture(state, variables.id);
    return { carrier: fixture ? fixture.carrier : null };
  },
  DeskCarrierEligibility: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = carrierFixture(state, variables.carrierId);
    return {
      dispatchCarrierAssignmentPreview: fixture?.eligibility ?? {
        blockers: [],
        warnings: [],
        advisories: [],
        findings: [],
      },
    };
  },
  DeskCarrierIntel: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = carrierFixture(state, variables.id);
    if (!fixture) return { carrier: null };
    return { carrier: { id: variables.id, intelligence: empty(state) ? null : fixture.intel } };
  },
  DeskCarrierSettlements: (state, variables) => {
    if (held(state)) return HOLD;
    const carrierId = variables.input?.fieldFilters?.find(
      (filter) => filter.field === "carrierId",
    )?.value;
    const fixture = carrierFixture(state, carrierId);
    const rows = fixture && !empty(state) ? fixture.settlements : [];
    return {
      carrierSettlements: connectionPage(rows, variables.input, variables.includeTotalCount),
    };
  },
  DeskOrderCard: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = orderFixture(state, variables.id);
    return { order: fixture ? fixture.order : null };
  },
  DeskInvoiceCard: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = deskInvoices(state.anchor)[variables.id];
    if (!fixture) return { invoice: null };
    if (!empty(state)) return { invoice: fixture.invoice };
    return {
      invoice: {
        ...fixture.invoice,
        paymentApplications: [],
        creditApplications: [],
        disputes: [],
        relatedInvoices: [],
      },
    };
  },
  DeskDriverSettlementCard: (state, variables) => {
    if (held(state)) return HOLD;
    const record = settlementWithShares(state, variables.id, "netPayMinor");
    if (!record || !deskDriverSettlements(state.anchor)[variables.id]) {
      return { driverSettlement: null };
    }
    return {
      driverSettlement: empty(state)
        ? { ...record, lineSummary: [], loadShares: [], disputes: [], exceptions: [] }
        : record,
    };
  },
  DeskCarrierSettlementCard: (state, variables) => {
    if (held(state)) return HOLD;
    const record = settlementWithShares(state, variables.id, "netPayableMinor");
    if (!record || !deskCarrierSettlements(state.anchor)[variables.id]) {
      return { carrierSettlement: null };
    }
    return {
      carrierSettlement: empty(state)
        ? { ...record, costSummary: [], loadShares: [], moveInvoices: [] }
        : record,
    };
  },
  DeskEdiPartnerCard: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = deskEdiPartners(state.anchor)[variables.id];
    if (!fixture) {
      throw new TriageError("EDIPartner not found", { code: "NOT_FOUND", type: "not-found" });
    }
    if (!empty(state)) return { ediPartnerCard: fixture.card };
    const card = fixture.card;
    return {
      ediPartnerCard: {
        ...card,
        transports: [],
        mappings: { profileId: null, updatedAt: null, counts: [] },
        activity: { ...card.activity, sets: [], series: [], ackStatusCounts: [] },
        attention: {
          ...card.attention,
          overdueAckCount: 0,
          oldestOverdueAt: null,
          overdueAcks: [],
          deadLettered: [],
          quarantinedFiles: [],
        },
        carriers: card.carriers === null ? null : [],
      },
    };
  },
  DeskEdiPartnerMessages: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = deskEdiPartners(state.anchor)[variables.partnerId];
    const rows = fixture && !empty(state) ? fixture.messages : [];
    return { ediMessages: connectionPage(rows, variables.input, variables.includeTotalCount) };
  },
  DeskDetentionCard: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture = deskDetentions(state.anchor)[variables.id];
    if (!fixture) {
      throw new TriageError("Detention occurrence not found", {
        code: "NOT_FOUND",
        type: "not-found",
      });
    }
    const shipment = detentionShipment(state, variables.id);
    const detail = {
      ...fixture,
      occurrence: {
        ...fixture.occurrence,
        shipmentId: shipment?.id ?? "",
        shipmentProNumber: shipment?.proNumber ?? "",
      },
    };
    return {
      detentionOccurrence: empty(state) ? { ...detail, evidence: [], notices: [] } : detail,
    };
  },
  AccountingSyncObjectStates: (state, variables) => {
    if (held(state)) return HOLD;
    const invoices = deskInvoices(state.anchor);
    return {
      accountingSyncObjectStates: (variables.objectIds ?? [])
        .map((id) => invoices[id]?.sync)
        .filter(Boolean),
    };
  },
  DeskRateAgreementPricing: (state, variables) => {
    if (held(state)) return HOLD;
    const agreementId = variables.input?.fieldFilters?.find(
      (filter) => filter.field === "rateAgreementId",
    )?.value;
    const pricing = agreementFixture(state, agreementId)?.pricing;
    const loads = pricing && !empty(state) ? pricing.loads : 0;
    return {
      rateQuotes: {
        edges: loads ? [{ node: { id: "rq_latest", ratedAt: pricing.lastRatedAt } }] : [],
        totalCount: loads,
      },
    };
  },
  SelectOptions: (state, variables) => {
    if (variables.input?.resource === "USER") return userSelectOptions(state, variables.input);
    const ids = variables.input?.ids ?? [];
    const nodes = ids
      .filter((id) => DESK_RECORD_NAMES[id])
      .map((id) => ({
        id,
        label: DESK_RECORD_NAMES[id].label,
        description: null,
        meta: DESK_RECORD_NAMES[id].meta ?? null,
      }));
    return {
      selectOptions: {
        edges: nodes.map((node) => ({ node, cursor: node.id })),
        pageInfo: { hasNextPage: false, endCursor: null },
        totalCount: nodes.length,
      },
    };
  },
  DeskRecordActivity: (state, variables) => {
    if (held(state)) return HOLD;
    const fixture =
      orderFixture(state, variables.resourceId) ??
      agreementFixture(state, variables.resourceId) ??
      failureFixture(state, variables.resourceId) ??
      deskInvoices(state.anchor)[variables.resourceId] ??
      settlementFixture(state, variables.resourceId);
    const rows = fixture && !empty(state) ? fixture.activity : [];
    return {
      auditEntriesByResourceId: connectionPage(rows, variables.input, variables.includeTotalCount),
    };
  },
  DeskRecordShipments: recordShipments,
};

/** The card the conversation holds: the one the screenshot asked for with `deskSlug`. */
function threadSlug(state) {
  return state.scenario.deskSlug ?? "tractor-t-101";
}

/** Where an EDI message came from, as the inspector reports it. */
function provenance({ message }) {
  return {
    messageId: message.id,
    profileId: message.partnerDocumentProfileId,
    templateId: message.templateId,
    templateVersionId: message.templateVersionId,
    generatedAt: message.generatedAt,
    generatedById: message.generatedById,
  };
}

/** The EDI message a route names, or the 404 the API answers for one that is gone. */
function ediRoute(state, path) {
  const id = path.match(/edi\/messages\/([a-z0-9_]+)/)?.[1] ?? "";
  const fixture = deskEdiMessages(state.anchor)[id];
  return fixture
    ? { fixture }
    : {
        missing: {
          __status: 404,
          body: { title: "Not Found", status: 404, detail: "EDI message not found" },
        },
      };
}

/** The EDI inbound file a route names, or the 404 the API answers for one that is gone. */
function ediFileRoute(state, path) {
  const id = path.match(/edi\/inbound-files\/([a-z0-9_]+)/)?.[1] ?? "";
  const fixture = deskEdiInboundFiles(state.anchor)[id];
  return fixture
    ? { fixture }
    : {
        missing: {
          __status: 404,
          body: { title: "Not Found", status: 404, detail: "EDI inbound file not found" },
        },
      };
}

/** The EDI transfer a route names, or the 404 the API answers for one that is gone. */
function ediTransferRoute(state, path) {
  const id = path.match(/edi\/transfers\/([a-z0-9_]+)/)?.[1] ?? "";
  const fixture = deskEdiTransfers(state.anchor)[id];
  return fixture
    ? { fixture }
    : {
        missing: {
          __status: 404,
          body: { title: "Not Found", status: 404, detail: "EDI transfer not found" },
        },
      };
}

/** The inbound tenders one file opened, as the transfers list answers for `inboundFileId`. */
function ediFileTenders(state, query) {
  const transfers = deskEdiTransfers(state.anchor);
  const results = (DESK_EDI_FILE_TENDERS[query?.get("inboundFileId") ?? ""] ?? []).map(
    (id) => transfers[id].transfer,
  );
  return { results, count: results.length, next: null, prev: null };
}

/** The invoice a billing route names, or the 404 the API answers for one that is gone. */
function invoiceRoute(state, path) {
  const id = path.match(/billing\/invoices\/([a-z0-9_]+)/)?.[1] ?? "";
  const fixture = deskInvoices(state.anchor)[id];
  return fixture
    ? { fixture }
    : {
        missing: {
          __status: 404,
          body: { title: "Not Found", status: 404, detail: "Invoice not found" },
        },
      };
}

/** An email attempt as the invoice's send history lists it. */
function emailAttempt(fixture) {
  const { invoice, attempt } = fixture;
  return {
    id: `inea_${invoice.id}`,
    organizationId: "org_mock",
    businessUnitId: "bu_mock",
    version: 1,
    invoiceId: invoice.id,
    emailMessageId: null,
    attemptNumber: 1,
    partNumber: 1,
    totalParts: 1,
    status: attempt.error ? "Failed" : "Sent",
    provider: "resend",
    providerMessageId: null,
    toRecipients: attempt.toRecipients,
    ccRecipients: [],
    bccRecipients: [],
    subject: `Invoice ${invoice.number}`,
    body: null,
    estimatedSize: 182_000,
    warnings: [],
    error: attempt.error,
    sentAt: attempt.sentAt,
    createdById: null,
    createdAt: attempt.sentAt ?? invoice.createdAt,
    updatedAt: attempt.sentAt ?? invoice.createdAt,
    attachments: [],
  };
}

export const DESK_CARD_ROUTES = [
  [
    "POST",
    /^\/api\/v1\/edi\/communication-profiles\/[a-z0-9_]+\/test-connection\/?$/,
    (state, path) => {
      if (held(state)) return HOLD;
      const failing = path.includes("edicp_acme_sftp");
      return {
        success: !failing,
        checks: failing
          ? [
              {
                name: "Host reachable",
                status: "passed",
                message: "sftp.acmefoods.example:22 answered",
              },
              { name: "Host key", status: "passed", message: "Matches the pinned ED25519 key" },
              {
                name: "Sign in",
                status: "failed",
                message: "ssh: unable to authenticate, attempted methods [none publickey]",
              },
            ]
          : [
              {
                name: "Endpoint reachable",
                status: "passed",
                message: "https://edi.tyson.example/as2 answered 200",
              },
              {
                name: "Certificates",
                status: "passed",
                message: "Every certificate is valid for 230 days or more",
              },
            ],
      };
    },
  ],
  [
    "POST",
    /^\/api\/v1\/edi\/communication-profiles\/[a-z0-9_]+\/poll\/?$/,
    (state, path) => {
      if (held(state)) return HOLD;
      const profileId = path.match(/communication-profiles\/([a-z0-9_]+)/)?.[1] ?? "";
      return { profileId, stagedFileIds: [], skippedFiles: 0 };
    },
  ],
  [
    "GET",
    /^\/api\/v1\/billing\/invoices\/[a-z0-9_]+\/email-attempts\/?$/,
    (state, path) => {
      if (held(state)) return HOLD;
      const { fixture, missing } = invoiceRoute(state, path);
      if (missing) return missing;
      const results = fixture.attempt ? [emailAttempt(fixture)] : [];
      return { results, count: results.length, next: null, prev: null };
    },
  ],
  [
    "GET",
    /^\/api\/v1\/billing\/invoices\/[a-z0-9_]+\/send-plan\/?$/,
    (state, path) => {
      if (held(state)) return HOLD;
      const { fixture, missing } = invoiceRoute(state, path);
      if (missing) return missing;
      const plan = fixture.plan ?? { to: [], errors: [] };
      return {
        invoiceId: fixture.invoice.id,
        providerLimitBytes: 40_000_000,
        estimatedBodyBytes: 4_000,
        parts: [],
        warnings: [],
        errors: plan.errors,
        recipients: { to: plan.to, cc: [], bcc: [] },
        fromEmail: "billing@trenova.example",
        subject: `Invoice ${fixture.invoice.number}`,
        body: "",
        invoicePdfDocumentId: null,
      };
    },
  ],
  [
    "GET",
    /^\/api\/v1\/edi\/transfers\/?$/,
    (state, _path, _body, query) => {
      if (held(state)) return HOLD;
      return ediFileTenders(state, query);
    },
  ],
  [
    "GET",
    /^\/api\/v1\/edi\/transfers\/[a-z0-9_]+\/mapping-preview\/?$/,
    (state, path) => {
      if (held(state)) return HOLD;
      const { fixture, missing } = ediTransferRoute(state, path);
      if (missing) return missing;
      if (!fixture.preview) {
        return {
          __status: 400,
          body: {
            title: "Bad Request",
            status: 400,
            detail: "Mapping preview is only available for transfers that have not been decided",
          },
        };
      }
      return fixture.preview;
    },
  ],
  [
    "GET",
    /^\/api\/v1\/edi\/transfers\/[a-z0-9_]+\/?$/,
    (state, path) => {
      if (held(state)) return HOLD;
      const { fixture, missing } = ediTransferRoute(state, path);
      return missing ?? fixture.transfer;
    },
  ],
  [
    "GET",
    /^\/api\/v1\/edi\/inbound-files\/[a-z0-9_]+\/inspect\/?$/,
    (state, path) => {
      if (held(state)) return HOLD;
      const { fixture, missing } = ediFileRoute(state, path);
      if (missing) return missing;
      if (!fixture.inspection) {
        return {
          __status: 400,
          body: {
            title: "Bad Request",
            status: 400,
            detail: "The raw file was purged under the retention policy, so it cannot be inspected",
          },
        };
      }
      return fixture.inspection;
    },
  ],
  [
    "GET",
    /^\/api\/v1\/edi\/inbound-files\/[a-z0-9_]+\/?$/,
    (state, path) => {
      if (held(state)) return HOLD;
      const { fixture, missing } = ediFileRoute(state, path);
      return missing ?? fixture.file;
    },
  ],
  [
    "GET",
    /^\/api\/v1\/edi\/messages\/[a-z0-9_]+\/inspect\/?$/,
    (state, path) => {
      if (held(state)) return HOLD;
      const { fixture, missing } = ediRoute(state, path);
      if (missing) return missing;
      if (!fixture.inspection) {
        return {
          __status: 422,
          body: { title: "Unprocessable", status: 422, detail: "The raw X12 was purged" },
        };
      }
      return {
        message: fixture.message,
        inspection: fixture.inspection,
        provenance: provenance(fixture),
      };
    },
  ],
  [
    "GET",
    /^\/api\/v1\/edi\/messages\/[a-z0-9_]+\/?$/,
    (state, path) => {
      if (held(state)) return HOLD;
      const { fixture, missing } = ediRoute(state, path);
      return missing ?? fixture.message;
    },
  ],
  [
    "GET",
    /^\/api\/v1\/rate-agreements\/[a-z0-9_]+\/lanes\/summary\/?$/,
    (state, path) => {
      if (held(state)) return HOLD;
      const { fixture, missing } = agreementRoute(state, path);
      if (missing) return missing;
      return laneSummary(empty(state) ? [] : fixture.lanes);
    },
  ],
  [
    "GET",
    /^\/api\/v1\/rate-agreements\/[a-z0-9_]+\/lanes\/?$/,
    (state, path, _body, query) => {
      if (held(state)) return HOLD;
      const { fixture, missing } = agreementRoute(state, path);
      if (missing) return missing;
      const lanes = empty(state) ? [] : fixture.lanes;
      const limit = Number(query?.get("limit") ?? 20);
      const offset = Number(query?.get("offset") ?? 0);
      return {
        results: lanes.slice(offset, offset + limit),
        count: lanes.length,
        retired: empty(state) ? 0 : fixture.retired,
        limit,
        offset,
      };
    },
  ],
  [
    "GET",
    /^\/api\/v1\/rate-agreements\/[a-z0-9_]+\/versions\/?$/,
    (state, path) => {
      if (held(state)) return HOLD;
      const { fixture, missing } = agreementRoute(state, path);
      if (missing) return missing;
      return { results: empty(state) ? [] : fixture.versions };
    },
  ],
  [
    "GET",
    /^\/api\/v1\/rate-agreements\/[a-z0-9_]+\/?$/,
    (state, path) => {
      if (held(state)) return HOLD;
      const { fixture, missing } = agreementRoute(state, path);
      return missing ?? fixture.agreement;
    },
  ],
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
