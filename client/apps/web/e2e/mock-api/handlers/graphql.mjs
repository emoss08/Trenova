import { AI_HANDLERS } from "./aicontrol.mjs";
import { hash, stageOf, stageRankOf, STAGE_RANKS } from "../state.mjs";

const HOUR = 3600;

const city = (stop) => stop.location.city;
const pickupOf = (s) => s.moves[0].stops[0];
const deliveryOf = (s) => s.moves[0].stops[1];
const isCovered = (s) => !!(s.moves[0].assignment || s.moves[0].carrierAssignment);
const isUncovered = (s) => stageOf(s) === "NeedsCoverage" && !isCovered(s);
const isTendered = (s) => s.tenderStatus === "Tendered";
const projectedArrival = (s) => s.board.slippedEta ?? s.board.deliveryAt;
const startOfDay = (t) => t - (t % 86400);
const money = (n) => String(Math.round(n * 100) / 100);
const name = (d) => `${d.firstName} ${d.lastName}`;
const initials = (text) =>
  text
    .replace(/[-.&+]/g, " ")
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0].toUpperCase())
    .join("");

function quickFilterMatches(state, s, token) {
  const now = state.now();
  const stage = stageOf(s);
  switch (token.filter) {
    case "Late":
      return stage === "Late";
    case "Uncovered":
      return isUncovered(s);
    case "Moving":
      return stage === "Moving";
    case "DeliveringToday":
      return startOfDay(projectedArrival(s)) === startOfDay(now) && stage !== "NeedsCoverage" && stage !== "Canceled";
    case "Reefer":
      return s.board.reefer;
    case "LowMargin":
      return s.board.margin < 15;
    case "DeliveryHour":
      return (
        startOfDay(projectedArrival(s)) === startOfDay(now) &&
        Math.floor((projectedArrival(s) % 86400) / HOUR) === token.hour &&
        stage !== "NeedsCoverage"
      );
    case "PickupWindow": {
      if (!isUncovered(s)) return false;
      const minutes = (s.board.pickupAt - now) / 60;
      return minutes >= token.windowStartMinutes && (token.windowEndMinutes == null || minutes < token.windowEndMinutes);
    }
    case "Detention":
      return !!s.board.detentionMinutes && stage !== "Delivered" && !state.billedDetention.has(s.id);
    case "ReadyToBill":
      return stage === "Delivered" && !s.billingTransferStatus;
    default:
      return true;
  }
}

const FIELD_GETTERS = new Map(Object.entries({
  stageRank: (s) => stageRankOf(s),
  status: (s) => s.status,
  tenderStatus: (s) => s.tenderStatus,
  customerId: (s) => s.customerId,
  equipmentClass: (s) => (s.board.reefer ? "Reefer" : "DryVan"),
  proNumber: (s) => s.proNumber,
  createdAt: (s) => s.createdAt,
  totalChargeAmount: (s) => s.board.revenue,
  "customer.name": (s) => s.customer.name,
  ownerId: (s) => s.ownerId ?? null,
  "owner.name": (s) => s.owner?.name ?? null,
  "shipperStop.scheduledWindowStart": (s) => pickupOf(s).scheduledWindowStart,
  "consigneeStop.scheduledWindowStart": (s) => deliveryOf(s).scheduledWindowStart,
  bol: (s) => s.bol,
  billingTransferStatus: (s) => s.billingTransferStatus ?? "",
}));

function fieldGetter(field) {
  return FIELD_GETTERS.get(field) ?? ((s) => (Object.hasOwn(s, field) ? s[field] : undefined));
}

function fieldFilterMatches(s, filter) {
  const get = FIELD_GETTERS.get(filter.field);
  if (!get) return true;
  const value = get(s);
  const list = Array.isArray(filter.value) ? filter.value : [filter.value];
  switch (filter.operator) {
    case "eq":
      return String(value) === String(filter.value);
    case "ne":
      return String(value) !== String(filter.value);
    case "in":
      return list.map(String).includes(String(value));
    case "notin":
      return value !== null && value !== undefined && !list.map(String).includes(String(value));
    case "contains":
      return String(value ?? "").toLowerCase().includes(String(filter.value).toLowerCase());
    case "lt":
      return value !== null && value !== undefined && value < filter.value;
    case "gte":
      return value !== null && value !== undefined && value >= filter.value;
    case "isnull":
      return value === null || value === undefined;
    case "isnotnull":
      return value !== null && value !== undefined;
    default:
      return true;
  }
}

function scoped(state, input = {}) {
  const query = (input.query ?? "").trim().toLowerCase();
  return state.shipments.filter((s) => {
    if (query) {
      const haystack = [s.proNumber, s.bol, s.customer.name, city(pickupOf(s)), city(deliveryOf(s)), s.board.driverName ?? ""]
        .join(" ")
        .toLowerCase();
      if (!haystack.includes(query)) return false;
    }
    for (const filter of input.fieldFilters ?? []) {
      if (!fieldFilterMatches(s, filter)) return false;
    }
    for (const group of input.filterGroups ?? []) {
      if (group.filters?.length && !group.filters.some((f) => fieldFilterMatches(s, f))) return false;
    }
    for (const token of input.quickFilters ?? []) {
      if (!quickFilterMatches(state, s, token)) return false;
    }
    return true;
  });
}

function sorted(list, sort = []) {
  const terms = sort.length ? sort : [{ field: "createdAt", direction: "desc" }];
  return [...list].sort((a, b) => {
    for (const term of terms) {
      const get = fieldGetter(term.field);
      const x = get(a);
      const y = get(b);
      if (x === y) continue;
      if (x === null || x === undefined) return 1;
      if (y === null || y === undefined) return -1;
      const order = x > y ? 1 : -1;
      return term.direction === "desc" ? -order : order;
    }
    return a.id < b.id ? -1 : 1;
  });
}

function shipmentEta(s) {
  const stage = stageOf(s);
  if (stage === "Delivered" || stage === "Canceled") return null;
  if (stage === "Late") {
    return {
      estimatedArrival: s.board.slippedEta,
      slackMinutes: -Math.round((s.board.slippedEta - s.board.deliveryAt) / 60),
      verdict: "Late",
      reason: s.board.risk,
    };
  }
  return {
    estimatedArrival: s.board.deliveryAt - (stage === "Moving" ? 600 : 0),
    slackMinutes: stage === "Moving" ? 70 : 0,
    verdict: "OnTime",
    reason: null,
  };
}

function toNode(s) {
  const { board: _board, ...node } = s;
  return { ...node, stage: stageOf(s), stageRank: stageRankOf(s), eta: shipmentEta(s) };
}

function shipmentConnection(state, input, includeTotalCount) {
  const all = sorted(scoped(state, input), input.sort);
  const offset = input.after ? Number(Buffer.from(input.after, "base64url").toString()) : 0;
  const first = input.first ?? 20;
  const page = all.slice(offset, offset + first);
  const end = offset + page.length;
  return {
    edges: page.map((s) => ({ node: toNode(s) })),
    totalCount: includeTotalCount === false ? undefined : all.length,
    pageInfo: {
      hasNextPage: end < all.length,
      endCursor: page.length ? Buffer.from(String(end)).toString("base64url") : null,
      hasPreviousPage: offset > 0,
      startCursor: null,
    },
  };
}

function stageSummary(state, input) {
  const list = scoped(state, input);
  return Object.entries(STAGE_RANKS).map(([stage, rank]) => {
    const rows = list.filter((s) => stageOf(s) === stage);
    return { stage, rank, count: rows.length, revenue: money(rows.reduce((t, s) => t + s.board.revenue, 0)) };
  });
}

/* Mirrors the server: groups in the order the board sorts their rows, days
   in the board's zone, and an empty key for no date or no owner. */
const GROUP_KEYS = {
  ShipDate: (s, tz) => dayKey(pickupOf(s).scheduledWindowStart, tz),
  DeliveryDate: (s, tz) => dayKey(deliveryOf(s).scheduledWindowStart, tz),
  Customer: (s) => s.customerId,
  Owner: (s) => s.ownerId ?? "",
};

function dayKey(unix, timezone) {
  if (unix === null || unix === undefined) return "";
  return new Intl.DateTimeFormat("en-CA", { timeZone: timezone, year: "numeric", month: "2-digit", day: "2-digit" })
    .format(new Date(unix * 1000));
}

function boardGroups(state, input, groupBy) {
  const keyOf = GROUP_KEYS[groupBy];
  if (!keyOf) throw new Error(`unknown grouping ${groupBy}`);
  const groups = new Map();
  for (const s of scoped(state, input)) {
    const key = keyOf(s, input.timezone ?? "UTC");
    const label = groupBy === "Customer" ? s.customer.name : groupBy === "Owner" ? (s.owner?.name ?? "") : "";
    const entry = groups.get(key) ?? { key, label, count: 0, total: 0 };
    entry.count += 1;
    entry.total += s.board.revenue;
    groups.set(key, entry);
  }
  const blankLast = (a, b) => (a.key === "") - (b.key === "");
  const byName = (a, b) => blankLast(a, b) || a.label.localeCompare(b.label) || (a.key < b.key ? -1 : 1);
  const byKey = (a, b) => blankLast(a, b) || (a.key < b.key ? -1 : 1);
  return [...groups.values()]
    .sort(groupBy === "Customer" || groupBy === "Owner" ? byName : byKey)
    .map(({ key, label, count, total }) => ({ key, label, count, revenue: money(total) }));
}

function quickFilterCounts(state, input) {
  const list = scoped(state, input);
  return ["Late", "Uncovered", "Moving", "DeliveringToday", "Reefer", "LowMargin"].map((filter) => ({
    filter,
    count: list.filter((s) => quickFilterMatches(state, s, { filter })).length,
  }));
}

function coverageNoun(operationType, count) {
  if (operationType === "asset") return count === 1 ? "a driver" : "drivers";
  if (operationType === "brokerage") return count === 1 ? "a carrier" : "carriers";
  return "coverage";
}

function plural(count, word) {
  return `${count} ${count === 1 ? word : `${word}s`}`;
}

function boardFacts(state) {
  const count = (filter) => state.shipments.filter((s) => quickFilterMatches(state, s, { filter })).length;
  const late = state.shipments.filter((s) => stageOf(s) === "Late");
  const openSuggestions = suggestionItems(state).filter((item) => state.decisions.get(item.key) !== "Done").length;
  return {
    deliveringToday: count("DeliveringToday"),
    moving: count("Moving"),
    late: late.length,
    uncovered: count("Uncovered"),
    detention: count("Detention"),
    readyToBill: count("ReadyToBill"),
    lateReason: late.find((s) => s.board.risk)?.board.risk ?? "",
    openSuggestions,
  };
}

function openIssues(facts) {
  return facts.late + facts.uncovered + facts.openSuggestions;
}

function joinClauses(clauses) {
  const segments = [];
  clauses.forEach((clause, index) => {
    if (index > 0) segments.push({ text: index === clauses.length - 1 ? " and " : ", ", filter: null });
    segments.push(...clause);
  });
  if (clauses.length) segments.push({ text: ". ", filter: null });
  return segments;
}

function briefSegments(state, facts) {
  const loads = (count) => plural(count, "load");
  const verb = (count, one, many) => (count === 1 ? one : many);
  const board = [];
  if (facts.deliveringToday) board.push([{ text: `${loads(facts.deliveringToday)} ${verb(facts.deliveringToday, "delivers", "deliver")} today`, filter: "DeliveringToday" }]);
  if (facts.moving) board.push([{ text: `${loads(facts.moving)} ${verb(facts.moving, "is", "are")} moving on schedule`, filter: "Moving" }]);
  if (facts.late) {
    const clause = [{ text: `${loads(facts.late)} ${verb(facts.late, "is", "are")} late`, filter: "Late" }];
    if (facts.lateReason) clause.push({ text: `, mostly ${facts.lateReason.toLowerCase()}`, filter: null });
    board.push(clause);
  }
  const followUp = [];
  if (facts.uncovered) followUp.push([{ text: `${loads(facts.uncovered)} still ${verb(facts.uncovered, "needs", "need")} ${coverageNoun(state.scenario.operationType, facts.uncovered)}`, filter: "Uncovered" }]);
  if (facts.detention) followUp.push([{ text: `${loads(facts.detention)} ${verb(facts.detention, "is", "are")} accruing detention`, filter: "Detention" }]);
  if (facts.readyToBill) followUp.push([{ text: `${loads(facts.readyToBill)} ${verb(facts.readyToBill, "is", "are")} ready to bill`, filter: "ReadyToBill" }]);
  const segments = [...joinClauses(board), ...joinClauses(followUp)];
  if (!segments.length) return [{ text: "Nothing on the board needs you right now.", filter: null }];
  segments[segments.length - 1] = { text: ".", filter: null };
  return segments;
}

/**
 * The day's brief is stored like the server stores it: written once, read
 * after that, and written again only when everything it flagged is cleared.
 */
function briefing(state) {
  const facts = boardFacts(state);
  const stored = state.brief;
  if (!stored || (stored.openIssues > 0 && openIssues(facts) === 0 && stored.generation < 6)) {
    state.brief = {
      segments: briefSegments(state, facts),
      narrated: state.scenario.ai,
      generatedAt: state.now(),
      generation: (stored?.generation ?? 0) + 1,
      openIssues: openIssues(facts),
    };
  }
  const { openIssues: _open, ...brief } = state.brief;
  return brief;
}

function driverUnits(state) {
  return state.drivers
    .filter((d) => !state.usedDrivers.has(d.id) && d.freeInHours <= 2)
    .sort((a, b) => a.freeInHours - b.freeInHours || a.deadheadMiles - b.deadheadMiles)
    .map((d) => ({
      id: d.id,
      kind: "Driver",
      name: name(d),
      initials: initials(name(d)),
      group: d.freeInHours === 0 ? "ReadyNow" : "WithinTwoHours",
      freeAt: d.freeInHours === 0 ? null : Math.round(state.now() + d.freeInHours * HOUR),
      city: d.city,
      unitLabel: d.unit,
      ring: state.scenario.hos ? { value: Number(d.hosHours.toFixed(2)), max: 11, low: d.hosHours < 4 } : null,
      badgeCount: null,
      ratePerMile: null,
      acceptancePercent: null,
      driveRemainingMs: state.scenario.hos ? Math.round(d.hosHours * HOUR * 1000) : null,
      tractorId: d.tractorId,
    }));
}

const HAZMAT_NAMES = ["Gasoline", "Diesel fuel", "Propane", "Sulfuric acid", "Ammonia, anhydrous", "Lithium ion batteries", "Paint", "Chlorine"];

function hazardousMaterials(state, input) {
  const rows = Array.from({ length: state.scenario.listRows }, (_, i) => ({
    id: `hm_${i}`,
    businessUnitId: "bu_mock",
    organizationId: "org_mock",
    status: "Active",
    code: `HM${String(i + 1).padStart(3, "0")}`,
    name: `${HAZMAT_NAMES[i % HAZMAT_NAMES.length]}${i >= HAZMAT_NAMES.length ? ` ${Math.floor(i / HAZMAT_NAMES.length) + 1}` : ""}`,
    description: "",
    class: "HazardClass3",
    unNumber: `UN${1200 + i}`,
    packingGroup: ["I", "II", "III"][i % 3],
    subsidiaryHazardClass: null,
    ergGuideNumber: "128",
    labelCodes: null,
    specialProvisions: null,
    properShippingName: null,
    handlingInstructions: null,
    emergencyContact: null,
    emergencyContactPhoneNumber: null,
    quantityThreshold: null,
    placardRequired: true,
    isReportableQuantity: false,
    marinePollutant: false,
    inhalationHazard: false,
    version: 1,
    createdAt: state.anchor - i * 86400,
    updatedAt: state.anchor - i * 86400,
  }));
  const offset = input.after ? Number(Buffer.from(input.after, "base64url").toString()) : 0;
  const page = rows.slice(offset, offset + (input.first ?? 10));
  const end = offset + page.length;
  return {
    edges: page.map((node) => ({ node })),
    totalCount: rows.length,
    pageInfo: {
      hasNextPage: end < rows.length,
      endCursor: page.length ? Buffer.from(String(end)).toString("base64url") : null,
      hasPreviousPage: offset > 0,
      startCursor: null,
    },
  };
}

function billingReadiness(state, shipmentId) {
  const shipment = state.shipments.find((s) => s.id === shipmentId);
  const delivered = shipment ? stageOf(shipment) === "Delivered" : false;
  const requirement = (code, name, satisfied) => ({
    documentTypeId: `dt_${code}`,
    documentTypeCode: code,
    documentTypeName: name,
    satisfied,
    documentCount: satisfied ? 1 : 0,
    documentIds: satisfied ? [`doc_${code}_${shipmentId}`] : [],
    ineligibleDocuments: [],
  });
  const requirements = [requirement("BOL", "Bill of lading", true), requirement("POD", "Proof of delivery", delivered)];
  return {
    shipmentId,
    shipmentStatus: shipment?.status ?? "New",
    policy: {
      shipmentBillingRequirementEnforcement: "Warn",
      rateValidationEnforcement: "Warn",
      billingExceptionDisposition: "RouteToBillingQueue",
      notifyOnBillingExceptions: false,
      readyToBillAssignmentMode: "Manual",
      billingQueueTransferMode: "Manual",
    },
    requirements,
    missingRequirements: requirements.filter((r) => !r.satisfied),
    validationFailures: [],
    warnings: [],
    serviceFailureContext: { hasUnresolved: false, unresolvedCount: 0, serviceFailureIds: [] },
    canMarkReadyToInvoice: delivered,
    shouldAutoMarkReadyToInvoice: false,
    shouldAutoTransferToBilling: false,
    shouldAutoApproveBilling: false,
    payers: [],
  };
}

function hosStates(state) {
  if (!state.scenario.hos) return [];
  const rows = new Map();
  for (const shipment of state.shipments) {
    const workerId = shipment.moves[0].assignment?.primaryWorkerId;
    if (workerId && shipment.board.hosHours != null) rows.set(workerId, shipment.board.hosHours);
  }
  for (const driver of state.drivers) rows.set(driver.id, driver.hosHours);
  return [...rows].map(([workerId, hours]) => ({
    workerId,
    workerName: null,
    provider: "Samsara",
    providerDriverId: workerId,
    dutyStatus: "OnDuty",
    driveRemainingMs: Math.round(hours * HOUR * 1000),
    shiftRemainingMs: Math.round((hours + 3) * HOUR * 1000),
    cycleRemainingMs: Math.round(40 * HOUR * 1000),
    cycleTomorrowMs: null,
    breakRemainingMs: Math.round(4 * HOUR * 1000),
    cycleStartedAt: null,
    shiftDrivingViolationMs: 0,
    cycleViolationMs: 0,
    currentVehicleId: null,
    currentTractorId: null,
    rulesetCycle: "USA 70 hour / 8 day",
    rulesetShift: "US Interstate Property",
    rulesetJurisdiction: "US",
    driveLimitMs: 11 * HOUR * 1000,
    shiftLimitMs: 14 * HOUR * 1000,
    cycleLimitMs: 70 * HOUR * 1000,
    breakLimitMs: 8 * HOUR * 1000,
    recordedAt: state.now(),
  }));
}

function carrierUnits(state) {
  return state.carriers
    .filter((c) => c.trucks > 0 || c.acceptance >= 80)
    .sort((a, b) => (b.trucks > 0) - (a.trucks > 0) || b.acceptance - a.acceptance)
    .map((c) => ({
      id: c.id,
      kind: "Carrier",
      name: c.name,
      initials: initials(c.name),
      group: c.trucks > 0 ? "TrucksPosted" : "UsuallyAccept",
      freeAt: null,
      city: null,
      unitLabel: c.mcNumber,
      ring: { value: c.acceptance, max: 100, low: c.acceptance < 80 },
      badgeCount: c.trucks,
      ratePerMile: c.rate.toFixed(2),
      acceptancePercent: c.acceptance,
      driveRemainingMs: null,
      tractorId: null,
    }));
}

function capacity(state, kind) {
  const uncovered = state.shipments.filter(isUncovered);
  const untendered = uncovered.filter((s) => !isTendered(s));
  if (kind === "Driver") {
    const units = driverUnits(state);
    const ready = units.filter((u) => u.group === "ReadyNow").length;
    const within = units.length - ready;
    return {
      kind,
      units,
      drivers: { ready, withinTwoHours: within, short: Math.max(0, untendered.length - units.length), uncovered: untendered.length },
      carriers: null,
    };
  }
  const units = carrierUnits(state);
  const posting = units.filter((u) => u.group === "TrucksPosted");
  const avg = posting.reduce((t, u) => t + Number(u.ratePerMile), 0) / Math.max(1, posting.length);
  return {
    kind,
    units,
    drivers: null,
    carriers: {
      posting: posting.length,
      untendered: untendered.length,
      awaitingAcceptance: uncovered.filter(isTendered).length,
      avgRatePerMile: avg.toFixed(2),
    },
  };
}

function quoteFor(carrier, shipment) {
  return Math.round((shipment.board.miles * carrier.rate) / 10) * 10;
}

function capacityMatches(state, kind, unitId, limit = 2) {
  const open = state.shipments
    .filter((s) => isUncovered(s) && !isTendered(s))
    .sort((a, b) => a.board.pickupAt - b.board.pickupAt);
  if (!open.length) return [];
  const offset = hash(unitId) % open.length;
  const picks = [];
  for (let k = 0; k < Math.min(limit, open.length); k++) picks.push(open[(offset + k * 3) % open.length]);
  const carrier = state.carriers.find((c) => c.id === unitId);
  const driver = state.drivers.find((d) => d.id === unitId);
  return [...new Set(picks)].map((s, k) => {
    const quote = carrier ? quoteFor(carrier, s) : null;
    return {
      shipmentId: s.id,
      moveId: s.moves[0].id,
      proNumber: s.proNumber,
      originCity: city(pickupOf(s)),
      destinationCity: city(deliveryOf(s)),
      pickupAt: s.board.pickupAt,
      revenue: money(s.board.revenue),
      deadheadMiles: driver ? driver.deadheadMiles + k * 9 : null,
      quote: quote == null ? null : money(quote),
      marginPercent: quote == null ? null : Math.round((1 - quote / s.board.revenue) * 100),
      fitPercent: state.scenario.ai ? 96 - k * 15 : null,
    };
  });
}

function coverageSuggestions(state, shipmentId) {
  const shipment = state.shipments.find((s) => s.id === shipmentId);
  if (!shipment) return { drivers: [], carriers: [] };
  const moveId = shipment.moves[0].id;
  const drivers = state.scenario.operationType === "brokerage"
    ? []
    : driverUnits(state)
        .slice()
        .sort((a, b) => {
          const da = state.drivers.find((d) => d.id === a.id).deadheadMiles;
          const db = state.drivers.find((d) => d.id === b.id).deadheadMiles;
          return da - db;
        })
        .slice(0, 2)
        .map((u, k) => ({
          workerId: u.id,
          tractorId: u.tractorId,
          moveId,
          name: u.name,
          initials: u.initials,
          unitLabel: u.unitLabel,
          distanceMiles: state.drivers.find((d) => d.id === u.id).deadheadMiles,
          driveRemainingMs: u.driveRemainingMs,
          fitPercent: state.scenario.ai ? (k ? 81 : 96) : null,
        }));
  const carriers = state.scenario.operationType === "asset"
    ? []
    : state.carriers
        .filter((_c, i) => (hash(shipmentId) + i) % 4 === 0)
        .slice(0, 2)
        .map((c) => ({
          carrierId: c.id,
          moveId,
          name: c.name,
          initials: initials(c.name),
          mcNumber: c.mcNumber,
          quote: money(quoteFor(c, shipment)),
          ratePerMile: c.rate.toFixed(2),
          acceptancePercent: c.acceptance,
          posted: c.trucks > 0,
        }));
  return { drivers, carriers };
}

function suggestionItems(state) {
  const { ai, operationType, hos } = state.scenario;
  const now = state.now();
  const items = [];
  const brokerage = operationType === "brokerage";
  const open = state.shipments.filter((s) => isUncovered(s) && !isTendered(s)).sort((a, b) => a.board.pickupAt - b.board.pickupAt);
  for (const s of open.slice(0, 2)) {
    const lane = `${city(pickupOf(s))} → ${city(deliveryOf(s))}`;
    if (brokerage) {
      const carrier = state.carriers[hash(s.id) % state.carriers.length];
      const quote = quoteFor(carrier, s);
      items.push({
        key: `coverage:${s.id}`, kind: "Tender", tone: "Warning", shipmentId: s.id, proNumber: s.proNumber,
        title: ai ? `Tender ${lane} to ${carrier.name}` : `${s.proNumber} has no carrier`,
        reason: ai ? `${carrier.acceptance}% acceptance on this lane, quoting $${quote.toLocaleString()} ($${carrier.rate.toFixed(2)}/mi).` : `Pickup at ${new Date(s.board.pickupAt * 1000).toISOString().slice(11, 16)}. No carrier has been tendered.`,
        impact: [`$${quote.toLocaleString()} quote`, `${carrier.acceptance}% accept`, `${Math.round((1 - quote / s.board.revenue) * 100)}% margin`],
        primary: { type: "TenderCarrier", label: ai ? "Tender" : "Find carrier", moveId: s.moves[0].id, workerId: null, tractorId: null, carrierId: carrier.id, detentionOccurrenceId: null, message: null },
        manualLabel: "Find carrier", dueAt: s.board.pickupAt, deferred: false,
      });
    } else {
      const driver = state.drivers.filter((d) => !state.usedDrivers.has(d.id)).sort((a, b) => a.deadheadMiles - b.deadheadMiles)[0];
      if (!driver) continue;
      items.push({
        key: `coverage:${s.id}`, kind: "Coverage", tone: "Warning", shipmentId: s.id, proNumber: s.proNumber,
        title: ai ? `Assign ${name(driver)} to ${lane}` : `${s.proNumber} needs a driver`,
        reason: ai ? `${driver.deadheadMiles} mi out with ${Math.floor(driver.hosHours)}h of drive time left. Pickup closes at ${new Date((s.board.pickupAt + 2 * HOUR) * 1000).toISOString().slice(11, 16)}.` : `Pickup at ${new Date(s.board.pickupAt * 1000).toISOString().slice(11, 16)}. Nobody is assigned yet.`,
        impact: [`$${s.board.revenue.toLocaleString()}`, `${driver.deadheadMiles} mi out`, ...(ai ? ["96% fit"] : [])],
        primary: { type: "AssignDriver", label: ai ? "Assign" : "Assign driver", moveId: s.moves[0].id, workerId: driver.id, tractorId: driver.tractorId, carrierId: null, detentionOccurrenceId: null, message: null },
        manualLabel: "Assign driver", dueAt: s.board.pickupAt, deferred: false,
      });
    }
  }
  const late = state.shipments.filter((s) => stageOf(s) === "Late").sort((a, b) => (b.board.slippedEta - b.board.deliveryAt) - (a.board.slippedEta - a.board.deliveryAt));
  if (late.length) {
    const s = late[0];
    const delta = Math.round((s.board.slippedEta - s.board.deliveryAt) / 60);
    const eta = new Date(s.board.slippedEta * 1000).toISOString().slice(11, 16);
    const message = `${s.board.risk ?? "A delay"} has held up ${s.proNumber}. The new ETA is ${eta}; we'll confirm a new delivery window within the hour.`;
    items.push({
      key: `delay:${s.id}`, kind: "DelayNotice", tone: "Danger", shipmentId: s.id, proNumber: s.proNumber,
      title: ai ? `Let ${s.customer.name} know about the delay` : `${s.proNumber} will miss its appointment`,
      reason: ai ? `${Math.floor(delta / 60)}h ${delta % 60}m behind · ${s.board.risk}. I've drafted the notice.` : `${Math.floor(delta / 60)}h ${delta % 60}m behind · ${s.board.risk}.`,
      impact: [`${Math.floor(delta / 60)}h ${delta % 60}m late`, s.customer.name],
      primary: { type: "NotifyCustomer", label: ai ? "Send notice" : "Notify customer", moveId: null, workerId: null, tractorId: null, carrierId: null, detentionOccurrenceId: null, message },
      manualLabel: "Notify customer", dueAt: s.board.deliveryAt, deferred: false,
    });
  }
  if (hos && !brokerage) {
    const tight = state.shipments.find((s) => stageOf(s) !== "Delivered" && s.board.hosHours != null && s.board.hosHours < 4 && s.moves[0].assignment && s.board.pct > 0);
    if (tight) {
      const left = `${Math.floor(tight.board.hosHours)}:${String(Math.round((tight.board.hosHours % 1) * 60)).padStart(2, "0")}`;
      items.push({
        key: `hos:${tight.id}`, kind: "HoursOfService", tone: "Warning", shipmentId: tight.id, proNumber: tight.proNumber,
        title: `${tight.board.driverName} runs out of hours before ${city(deliveryOf(tight))}`,
        reason: `${tight.proNumber} is ${Math.round(tight.board.miles * (1 - tight.board.pct / 100))} mi out with ${left} of drive time left.`,
        impact: [`${left} left`, `${Math.round(tight.board.miles * (1 - tight.board.pct / 100))} mi to go`],
        primary: { type: "Review", label: "Plan relay", moveId: null, workerId: null, tractorId: null, carrierId: null, detentionOccurrenceId: null, message: null },
        manualLabel: "Plan relay", dueAt: now + tight.board.hosHours * HOUR, deferred: false,
      });
    }
  }
  const detained = state.shipments.find((s) => s.board.detentionMinutes && stageOf(s) !== "Delivered" && !state.billedDetention.has(s.id));
  if (detained) {
    items.push({
      key: `detention:${detained.id}`, kind: "Detention", tone: "Accent", shipmentId: detained.id, proNumber: detained.proNumber,
      title: `${detained.board.driverName ?? "The driver"} is waiting at ${pickupOf(detained).location.name}`,
      reason: `Free time ended at 2h. Detention is billable at $75/hr.`,
      impact: [`${Math.floor(detained.board.detentionMinutes / 60)}h ${detained.board.detentionMinutes % 60}m on site`, "$75/hr"],
      primary: { type: "ApproveDetention", label: "Add detention", moveId: null, workerId: null, tractorId: null, carrierId: null, detentionOccurrenceId: `det_${detained.id}`, message: null },
      manualLabel: "Add detention", dueAt: null, deferred: false,
    });
  }
  if (state.scenario.board === "quiet") return items.slice(0, 2);
  return items;
}

function suggestions(state) {
  const items = suggestionItems(state)
    .filter((item) => state.decisions.get(item.key) !== "Done")
    .map((item) => ({ ...item, deferred: state.decisions.get(item.key) === "Later" }))
    .sort((a, b) => Number(a.deferred) - Number(b.deferred));
  return { items, handledThisShift: state.handledThisShift, narrated: state.scenario.ai };
}

function watchlist(state) {
  const now = state.now();
  const dayStart = startOfDay(now);
  const today = state.shipments.filter((s) => startOfDay(projectedArrival(s)) === dayStart && stageOf(s) !== "NeedsCoverage" && stageOf(s) !== "Canceled");
  const buckets = Array.from({ length: 18 }, (_, i) => {
    const hour = 6 + i;
    const rows = today.filter((s) => Math.floor((projectedArrival(s) % 86400) / HOUR) === hour);
    return {
      hour,
      delivered: rows.filter((s) => stageOf(s) === "Delivered").length,
      late: rows.filter((s) => stageOf(s) === "Late").length,
      scheduled: rows.filter((s) => stageOf(s) !== "Delivered" && stageOf(s) !== "Late").length,
    };
  });
  const late = today.filter((s) => stageOf(s) === "Late").sort((a, b) => (b.board.slippedEta - b.board.deliveryAt) - (a.board.slippedEta - a.board.deliveryAt));
  const uncovered = state.shipments.filter((s) => isUncovered(s) && !isTendered(s)).sort((a, b) => a.board.pickupAt - b.board.pickupAt);
  const toMidnight = Math.round((dayStart + 86400 - now) / 60);
  const windowDefs = [["UnderTwoHours", 0, 120], ["TwoToSixHours", 120, 360], ["LaterToday", 360, toMidnight], ["TomorrowOrLater", toMidnight, null]];
  const windows = windowDefs.map(([window, start, end]) => {
    const rows = uncovered.filter((s) => {
      const minutes = (s.board.pickupAt - now) / 60;
      return minutes >= start && (end == null || minutes < end);
    });
    return { window, startMinutes: start, endMinutes: end, count: rows.length, revenue: money(rows.reduce((t, s) => t + s.board.revenue, 0)) };
  });
  const next = uncovered.find((s) => s.board.pickupAt > now) ?? null;
  const detained = state.shipments
    .filter((s) => s.board.detentionMinutes && stageOf(s) !== "Delivered" && !state.billedDetention.has(s.id))
    .sort((a, b) => b.board.detentionMinutes - a.board.detentionMinutes);
  const rate = 75;
  const accrual = (s) => ((s.board.detentionMinutes - 120) / 60) * rate + ((now - state.anchor) / 3600) * rate;
  const billable = state.shipments.filter((s) => stageOf(s) === "Delivered" && !s.billingTransferStatus);
  const byCustomer = new Map();
  for (const s of billable) {
    const entry = byCustomer.get(s.customerId) ?? { customerId: s.customerId, name: s.customer.name, count: 0, total: 0 };
    entry.count += 1;
    entry.total += s.board.revenue;
    byCustomer.set(s.customerId, entry);
  }
  const customers = [...byCustomer.values()].sort((a, b) => b.total - a.total);
  return {
    deliveries: {
      onTime: today.filter((s) => stageOf(s) !== "Late").length,
      total: today.length,
      lateCount: late.length,
      buckets,
      worstLate: late.slice(0, 3).map((s) => ({
        shipmentId: s.id,
        proNumber: s.proNumber,
        deltaMinutes: Math.round((s.board.slippedEta - s.board.deliveryAt) / 60),
        city: city(deliveryOf(s)),
        customerName: s.customer.name,
      })),
    },
    uncovered: {
      count: uncovered.length,
      revenue: money(uncovered.reduce((t, s) => t + s.board.revenue, 0)),
      windows,
      next: next && { shipmentId: next.id, pickupAt: next.board.pickupAt, originCity: city(pickupOf(next)), destinationCity: city(deliveryOf(next)) },
    },
    detention: {
      stopCount: detained.length,
      amount: money(detained.reduce((t, s) => t + accrual(s), 0)),
      ratePerHour: money(detained.length * rate),
      snapshotAt: now,
      top: detained.slice(0, 3).map((s) => ({
        shipmentId: s.id,
        stopId: pickupOf(s).id,
        occurrenceId: `det_${s.id}`,
        facilityName: pickupOf(s).location.name,
        coverageName: s.board.driverName ?? s.moves[0].carrierAssignment?.carrier.name ?? null,
        billableSince: now - (s.board.detentionMinutes - 120) * 60,
        ratePerHour: money(rate),
        amount: money(accrual(s)),
      })),
    },
    billing: {
      count: billable.length,
      total: money(billable.reduce((t, s) => t + s.board.revenue, 0)),
      customers: customers.slice(0, 4).map((c) => ({ ...c, total: money(c.total) })),
      moreCustomers: Math.max(0, customers.length - 4),
    },
  };
}

function pushEvent(state, shipment, type, summary, actorLabel = "Sam Admin") {
  state.events.unshift({
    __typename: "ShipmentLifecycleEvent",
    id: `evt_${state.events.length + 100}`,
    organizationId: "org_mock",
    businessUnitId: "bu_mock",
    shipmentId: shipment.id,
    type,
    severity: "brand",
    actorType: "user",
    actorId: "usr_mock",
    actorLabel,
    summary,
    metadata: {},
    occurredAt: state.now(),
    correlationId: null,
    actor: null,
    shipment: { id: shipment.id, proNumber: shipment.proNumber },
    proNumber: shipment.proNumber,
    previousStatus: null,
    newStatus: null,
    reason: null,
  });
}

function assignMoves(state, input) {
  const results = input.map((item) => {
    const shipment = state.shipments.find((s) => s.moves[0].id === item.moveId);
    const driver = state.drivers.find((d) => d.id === item.primaryWorkerId);
    if (!shipment || !driver) {
      return { moveId: item.moveId, success: false, assignmentId: null, error: "Not found", findings: [] };
    }
    const move = shipment.moves[0];
    move.coverageType = "driver";
    move.status = "Assigned";
    move.assignment = {
      id: `asg_${move.id}`, businessUnitId: "bu_mock", organizationId: "org_mock", shipmentMoveId: move.id,
      primaryWorkerId: driver.id, tractorId: driver.tractorId, trailerId: null, secondaryWorkerId: null,
      status: "New", archivedAt: null, version: 1, createdAt: 0, updatedAt: 0,
      tractor: { id: driver.tractorId, code: driver.unit }, trailer: null,
      primaryWorker: { id: driver.id, firstName: driver.firstName, lastName: driver.lastName, wholeName: name(driver), profilePicUrl: null },
      secondaryWorker: null,
    };
    shipment.status = "Assigned";
    shipment.tenderStatus = "Accepted";
    shipment.board.driverName = name(driver);
    shipment.board.hosHours = driver.hosHours;
    state.usedDrivers.add(driver.id);
    state.decisions.set(`coverage:${shipment.id}`, "Done");
    pushEvent(state, shipment, "DriverAssigned", `Assigned ${name(driver)}`);
    return { moveId: item.moveId, success: true, assignmentId: move.assignment.id, error: null, findings: [] };
  });
  const succeeded = results.filter((r) => r.success).length;
  return { succeeded, failed: results.length - succeeded, results };
}

function tender(state, input) {
  const tendered = [];
  const failed = [];
  for (const item of input.items) {
    const shipment = state.shipments.find((s) => s.id === item.shipmentId);
    if (!shipment || !isUncovered(shipment)) {
      failed.push({ shipmentId: item.shipmentId, message: "Shipment is not waiting for coverage" });
      continue;
    }
    const carrier = state.carriers.find((c) => c.id === item.carrierId) ?? state.carriers[hash(shipment.id) % state.carriers.length];
    shipment.tenderStatus = "Tendered";
    shipment.board.tenderTo = carrier.name;
    state.decisions.set(`coverage:${shipment.id}`, "Done");
    pushEvent(state, shipment, "TenderOffered", `Tendered to ${carrier.name}`);
    tendered.push({ shipmentId: shipment.id, tenderId: `tdr_${shipment.id}`, carrierId: carrier.id, carrierName: carrier.name });
  }
  return { tendered, failed };
}

const EMPTY_CONNECTION = {
  edges: [],
  totalCount: 0,
  pageInfo: { hasNextPage: false, endCursor: null, hasPreviousPage: false, startCursor: null },
};

const SHELL_HANDLERS = {
  MyAgents: () => ({ myAgents: EMPTY_CONNECTION }),
  AttentionSummary: () => ({
    attentionSummary: {
      billingQueue: 0,
      pendingApprovals: 0,
      reconciliationExceptions: 0,
      serviceFailures: 0,
      ediAttention: 0,
      agentDecisions: 0,
    },
  }),
  SidebarPreferences: () => ({ sidebarPreferences: null }),
  NotificationUnreadCount: () => ({ notificationUnreadCount: 0 }),
  DefaultTableConfiguration: () => ({ defaultTableConfiguration: null }),
  TableConfigurationTable: () => ({ tableConfigurations: EMPTY_CONNECTION }),
};

const HANDLERS = {
  ...SHELL_HANDLERS,
  ...AI_HANDLERS,
  ShipmentBoardTable: (state, v) => ({ shipments: shipmentConnection(state, v.input, v.includeTotalCount) }),
  ShipmentDetail: (state, v) => {
    const s = state.shipments.find((x) => x.id === v.id);
    return { shipment: s ? toNode(s) : null };
  },
  HazardousMaterialTable: (state, v) => ({ hazardousMaterials: hazardousMaterials(state, v.input) }),
  TelematicsStatus: (state) => ({
    telematicsStatus: {
      provider: "Samsara",
      enabled: state.scenario.hos,
      configured: state.scenario.hos,
      webhookConfigured: state.scenario.hos,
      lastPolledAt: state.scenario.hos ? state.now() - 60 : null,
      lastSuccessAt: state.scenario.hos ? state.now() - 60 : null,
      failureCount: 0,
      lastError: null,
      mappedTractors: state.scenario.hos ? 18 : 0,
      totalTractors: 24,
      mappedWorkers: state.scenario.hos ? 24 : 0,
    },
  }),
  ShipmentBillingReadiness: (state, v) => ({ shipmentBillingReadiness: billingReadiness(state, v.shipmentId) }),
  WorkerHosStates: (state) => ({ workerHosStates: hosStates(state) }),
  ShipmentBoardCapabilities: (state) => ({
    shipmentBoardCapabilities: {
      ai: state.scenario.ai,
      operationType: state.scenario.operationType,
      hos: state.scenario.hos,
      maps: state.scenario.maps,
    },
  }),
  ShipmentStageSummary: (state, v) => ({ shipmentStageSummary: stageSummary(state, v.input) }),
  ShipmentBoardGroups: (state, v) => ({ shipmentBoardGroups: boardGroups(state, v.input, v.groupBy) }),
  ShipmentQuickFilterCounts: (state, v) => ({ shipmentQuickFilterCounts: quickFilterCounts(state, v.input) }),
  ShipmentBriefing: (state) => ({ shipmentBriefing: briefing(state) }),
  ShipmentCapacity: (state, v) => ({ shipmentCapacity: capacity(state, v.kind) }),
  CapacityUnitMatches: (state, v) => ({ capacityUnitMatches: capacityMatches(state, v.kind, v.unitId, v.limit ?? 2) }),
  ShipmentCoverageSuggestions: (state, v) => ({ shipmentCoverageSuggestions: coverageSuggestions(state, v.shipmentId) }),
  ShipmentSuggestions: (state) => ({ shipmentSuggestions: suggestions(state) }),
  ShipmentWatchlist: (state) => ({ shipmentWatchlist: watchlist(state) }),
  ShipmentEvents: (state, v) => ({
    shipmentEvents: state.events
      .filter((e) => !v.input?.shipmentId || e.shipmentId === v.input.shipmentId)
      .slice(0, v.input?.limit ?? 50),
  }),
  DispatchAssignMoves: (state, v) => ({ dispatchAssignMoves: assignMoves(state, v.input) }),
  TenderShipments: (state, v) => ({ tenderShipments: tender(state, v.input) }),
  DecideShipmentSuggestion: (state, v) => {
    state.decisions.set(v.input.key, v.input.decision);
    if (v.input.decision === "Done") state.handledThisShift += 1;
    return { decideShipmentSuggestion: true };
  },
  UndoShipmentSuggestionDecision: (state, v) => {
    if (state.decisions.get(v.key) === "Done") state.handledThisShift -= 1;
    state.decisions.delete(v.key);
    return { undoShipmentSuggestionDecision: true };
  },
  NotifyShipmentDelay: (state, v) => {
    const s = state.shipments.find((x) => x.id === v.input.shipmentId);
    if (s) {
      state.decisions.set(`delay:${s.id}`, "Done");
      pushEvent(state, s, "CommentPosted", `Sent a delay notice to ${s.customer.name}`);
    }
    return { notifyShipmentDelay: true };
  },
  ApproveDetentionOccurrence: (state, v) => {
    const s = state.shipments.find((x) => `det_${x.id}` === v.occurrenceId);
    if (s) {
      state.billedDetention.add(s.id);
      state.decisions.set(`detention:${s.id}`, "Done");
    }
    const now = state.now();
    return {
      approveDetentionOccurrence: {
        id: v.occurrenceId, organizationId: "org_mock", businessUnitId: "bu_mock",
        shipmentId: s?.id ?? "", shipmentMoveId: s?.moves[0].id ?? "", stopId: s ? pickupOf(s).id : "",
        customerId: s?.customerId ?? "", locationId: s ? pickupOf(s).locationId : "", detentionPolicyId: null,
        stopType: "Pickup", scheduleType: "Appointment", clockStartAt: now - 7200, freeTimeExpiresAt: now - 3600,
        isOpen: true, arrivedLate: false, lateByMinutes: 0, freeMinutesGranted: 120, rawDwellMinutes: 180,
        billableMinutes: 60, roundedMinutes: 60, billableUnits: 1, grossAmount: 75, billableAmount: 75,
        driverPayMinutes: 0, driverPayAmount: 0, netMargin: 75, capApplied: "None", convertedToLayover: false,
        currency: "USD", status: "Approved", notificationStatus: "NotRequired", suppressedByGate: false,
        requiresApproval: false, waivedAmount: 0, collectabilityScore: 80, version: 2, createdAt: now, updatedAt: now,
      },
    };
  },
  BillingTransferCandidateIds: (state) => {
    const ids = state.shipments
      .filter((s) => stageOf(s) === "Delivered" && !s.billingTransferStatus)
      .map((s) => s.id);
    return { shipmentBillingTransferCandidateIds: { ids, totalCount: ids.length, truncated: false } };
  },
  BulkTransferShipmentsToBilling: (state, v) => {
    const ids = new Set(v.input.shipmentIds);
    const results = state.shipments
      .filter((s) => ids.has(s.id))
      .map((s) => {
        s.billingTransferStatus = "ReadyForReview";
        return { shipmentId: s.id, proNumber: s.proNumber, success: true, markedReadyToInvoice: true, failureCode: null, error: null, billingQueueItem: null, billingQueueItems: [] };
      });
    return { bulkTransferShipmentsToBilling: { results, transferredCount: results.length, failedCount: 0 } };
  },
};

export function handleGraphQL(state, body) {
  const handler = Object.hasOwn(HANDLERS, body.operationName) ? HANDLERS[body.operationName] : undefined;
  if (!handler) {
    return { known: false, payload: { data: null, errors: [{ message: `mock-api: no handler for ${body.operationName}`, extensions: { code: "MOCK_UNHANDLED" } }] } };
  }
  try {
    return { known: true, payload: { data: handler(state, body.variables ?? {}) } };
  } catch (error) {
    return { known: true, payload: { data: null, errors: [{ message: String(error?.stack ?? error) }] } };
  }
}
