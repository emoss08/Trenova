// The Formula Studio's data, from the design handoff (design_handoff_formula_studio/design/data.jsx):
// the shipment schema the formula engine reads, the standard library, the sample load and the
// shipments it can be priced on, and three saved copies of the prototype's "Regional per mile +
// fuel" template — a draft, one in review and one active — with their versions, scenarios,
// reviews and usage. Timestamps are relative to the board's anchor.
import { ORG, USER } from "./session.mjs";

const MOCK_ORIGIN =
  process.env.MOCK_API_PUBLIC_URL ?? `http://localhost:${process.env.MOCK_API_PORT ?? 8080}`;

export const DRAFT_TEMPLATE_ID = "fmt_studio_draft";
export const REVIEW_TEMPLATE_ID = "fmt_studio_review";
export const ACTIVE_TEMPLATE_ID = "fmt_studio_active";

/** The shipment schema as the prototype draws it: name, type, category, nullable, description. */
export const VARIABLES = [
  { name: "baseRate", type: "Number", category: "charges", description: "The rate on the lane or shipment: per mile, per stop or per CWT, whatever the method multiplies it by" },
  { name: "freightChargeAmount", type: "Number", category: "charges", description: "Freight charge already entered on the shipment" },
  { name: "otherChargeAmount", type: "Number", category: "charges", description: "Other charges already entered on the shipment" },
  { name: "currentTotalCharge", type: "Number", category: "charges", description: "The shipment's current total charge" },
  { name: "totalDistance", type: "Number", category: "computed", description: "Miles across every move on the shipment" },
  { name: "totalStops", type: "Number", category: "computed", description: "Stops on the shipment, pickups and deliveries included" },
  { name: "totalWeight", type: "Number", category: "computed", description: "Pounds across every commodity" },
  { name: "totalPieces", type: "Number", category: "computed", description: "Pieces or pallets across every commodity" },
  { name: "totalLinearFeet", type: "Number", category: "computed", description: "Linear feet occupied, from commodity dimensions" },
  { name: "totalHours", type: "Number", category: "computed", nullable: true, description: "Hours from first arrival to last departure; empty until stop times exist" },
  { name: "pickupDayOfWeek", type: "Number", category: "computed", description: "Day of the first pickup, 0 = Sunday to 6 = Saturday" },
  { name: "pickupHour", type: "Number", category: "computed", description: "Hour of the first pickup, 0 to 23" },
  { name: "pickupMonth", type: "Number", category: "computed", description: "Month of the first pickup, 1 to 12" },
  { name: "isWeekendPickup", type: "Boolean", category: "computed", description: "True when the first pickup falls on a Saturday or Sunday" },
  { name: "hasHazmat", type: "Boolean", category: "computed", description: "True when any commodity is hazardous" },
  { name: "requiresTemperatureControl", type: "Boolean", category: "computed", description: "True when any commodity needs temperature control" },
  { name: "temperatureDifferential", type: "Number", category: "computed", nullable: true, description: "Maximum minus minimum temperature; empty without temperature control" },
  { name: "weight", type: "Number", category: "shipment", description: "Weight entered on the shipment" },
  { name: "pieces", type: "Number", category: "shipment", description: "Pieces entered on the shipment" },
  { name: "temperatureMin", type: "Number", category: "shipment", nullable: true, description: "Lowest allowed temperature, °F" },
  { name: "temperatureMax", type: "Number", category: "shipment", nullable: true, description: "Highest allowed temperature, °F" },
  { name: "proNumber", type: "String", category: "shipment", description: "PRO tracking number" },
  { name: "status", type: "String", category: "shipment", description: "Current shipment status" },
  { name: "ratingUnit", type: "Number", category: "shipment", description: "Rating unit value" },
  { name: "customer.name", type: "String", category: "customer", description: "Customer name" },
  { name: "customer.code", type: "String", category: "customer", description: "Customer code" },
  { name: "tractorType.name", type: "String", category: "equipment", nullable: true, description: "Tractor type name" },
  { name: "tractorType.code", type: "String", category: "equipment", nullable: true, description: "Tractor type code" },
  { name: "tractorType.costPerMile", type: "Number", category: "equipment", nullable: true, description: "Tractor cost per mile" },
  { name: "trailerType.name", type: "String", category: "equipment", nullable: true, description: "Trailer type name" },
  { name: "trailerType.code", type: "String", category: "equipment", nullable: true, description: "Trailer type code" },
  { name: "trailerType.costPerMile", type: "Number", category: "equipment", nullable: true, description: "Trailer cost per mile" },
  { name: "origin.city", type: "String", category: "location", description: "First pickup city" },
  { name: "origin.state", type: "String", category: "location", description: "First pickup state" },
  { name: "origin.zip", type: "String", category: "location", description: "First pickup postal code" },
  { name: "destination.city", type: "String", category: "location", description: "Last delivery city" },
  { name: "destination.state", type: "String", category: "location", description: "Last delivery state" },
  { name: "destination.zip", type: "String", category: "location", description: "Last delivery postal code" },
];

export const FUNCTIONS = [
  { name: "min", signature: "min(...values)", category: "Math", description: "The smallest of the values", example: "min(totalDistance * 2.1, 900)" },
  { name: "max", signature: "max(...values)", category: "Math", description: "The largest of the values", example: "max(250, baseRate * totalDistance)" },
  { name: "clamp", signature: "clamp(value, min, max)", category: "Math", description: "Holds a value between two bounds", example: "clamp(totalStops - 2, 0, 6)" },
  { name: "pow", signature: "pow(base, exp)", category: "Math", description: "base raised to exp", example: "pow(1.02, pickupMonth)" },
  { name: "abs", signature: "abs(x)", category: "Math", description: "Absolute value", example: "abs(temperatureDifferential)" },
  { name: "ceil", signature: "ceil(x)", category: "Math", description: "Rounds up to the next whole number", example: "ceil(totalWeight / 100)" },
  { name: "floor", signature: "floor(x)", category: "Math", description: "Rounds down to the whole number", example: "floor(totalHours)" },
  { name: "round", signature: "round(x, decimals?)", category: "Rounding", description: "Rounds half up", example: "round(baseRate * totalDistance, 2)" },
  { name: "roundUp", signature: "roundUp(x, decimals?)", category: "Rounding", description: "Always rounds up", example: "roundUp(totalHours, 1)" },
  { name: "roundDown", signature: "roundDown(x, decimals?)", category: "Rounding", description: "Always rounds down", example: "roundDown(totalDistance, 0)" },
  { name: "roundHalfEven", signature: "roundHalfEven(x, decimals?)", category: "Rounding", description: "Banker's rounding", example: "roundHalfEven(x, 2)" },
  { name: "roundTo", signature: "roundTo(x, increment)", category: "Rounding", description: "Rounds to the nearest increment", example: "roundTo(charge, 5)" },
  { name: "sum", signature: "sum(a, b, ...)", category: "Aggregate", description: "Adds the values", example: "sum(linehaul, fuel, stops)" },
  { name: "avg", signature: "avg(a, b, ...)", category: "Aggregate", description: "Mean of the values", example: "avg(a, b)" },
  { name: "coalesce", signature: "coalesce(a, b, ...)", category: "Aggregate", description: "The first value that is not empty", example: "coalesce(totalHours, 0)" },
  { name: "lookup", signature: "lookup(table, key)", category: "Rate tables", description: "Reads a value from a rate table by key", example: 'lookup("zone_rates", destination.state)' },
  { name: "lookupOr", signature: "lookupOr(table, key, fallback)", category: "Rate tables", description: "lookup, with a fallback when no row matches", example: 'lookupOr("zone_rates", destination.state, 2.4)' },
  { name: "lookup2", signature: "lookup2(table, rowKey, colKey)", category: "Rate tables", description: "Reads a two-key rate table", example: 'lookup2("lane_matrix", origin.state, destination.state)' },
  { name: "lookupInterp", signature: "lookupInterp(table, key)", category: "Rate tables", description: "Interpolates between bands of a rate table", example: 'lookupInterp("weight_breaks", totalWeight)' },
  { name: "deficitWeight", signature: "deficitWeight(table, weight)", category: "Rate tables", description: "Rates at the next weight break when that is cheaper", example: 'deficitWeight("ltl_breaks", totalWeight)' },
];

export const TABLES = {
  zone_rates: { TX: 2.35, GA: 2.6, CO: 3.1, LA: 2.45, CA: 3.4 },
};

/** The sample load the preview prices when no shipment is picked: 742 mi, 3 stops. */
export const SAMPLE = {
  baseRate: 2.85, freightChargeAmount: 0, otherChargeAmount: 0, currentTotalCharge: 0,
  totalDistance: 742, totalStops: 3, totalWeight: 38400, totalPieces: 22, totalLinearFeet: 44, totalHours: null,
  pickupDayOfWeek: 2, pickupHour: 8, pickupMonth: 10, isWeekendPickup: false, hasHazmat: false,
  requiresTemperatureControl: false, temperatureDifferential: null, weight: 38400, pieces: 22,
  temperatureMin: null, temperatureMax: null, proNumber: "S-SAMPLE", status: "New", ratingUnit: 1,
  "customer.name": "Acme Foods", "customer.code": "ACME",
  "tractorType.name": "Day cab", "tractorType.code": "DC", "tractorType.costPerMile": 1.42,
  "trailerType.name": "53' dry van", "trailerType.code": "DV53", "trailerType.costPerMile": 0.18,
  "origin.city": "Dallas", "origin.state": "TX", "origin.zip": "75207",
  "destination.city": "Atlanta", "destination.state": "GA", "destination.zip": "30318",
};

/** Fields the shipment itself carries; every other schema value is computed from it. */
export const FIELD_SOURCE = new Set([
  "baseRate", "freightChargeAmount", "otherChargeAmount", "currentTotalCharge", "weight", "pieces",
  "temperatureMin", "temperatureMax", "proNumber", "status", "ratingUnit", "customer.name", "customer.code",
  "origin.city", "origin.state", "origin.zip", "destination.city", "destination.state", "destination.zip",
  "tractorType.name", "tractorType.code", "tractorType.costPerMile", "trailerType.name", "trailerType.code",
  "trailerType.costPerMile",
]);

/**
 * The real-shipment picker's loads: the prototype's three, and four more that sit on the template's
 * edges (a short haul on the minimum, a weekend multi-stop, a reefer with no stop times yet, hazmat).
 */
export const SHIPMENTS = [
  { id: "shp_fs_240811", pro: "S-240811", lane: "Dallas, TX → Atlanta, GA", meta: "781 mi · 2 stops · dry van", customer: "Acme Foods", values: { baseRate: 2.85, totalDistance: 781, totalStops: 2, totalWeight: 41200, totalPieces: 24, totalHours: 14.5, pickupDayOfWeek: 1, isWeekendPickup: false } },
  { id: "shp_fs_240794", pro: "S-240794", lane: "Fresno, CA → Denver, CO", meta: "1,146 mi · 3 stops · reefer", customer: "Harborline Grocers", values: { baseRate: 3.4, totalDistance: 1146, totalStops: 3, totalWeight: 39800, totalHours: 31.5, pickupDayOfWeek: 6, isWeekendPickup: true, requiresTemperatureControl: true, temperatureMin: 34, temperatureMax: 38, temperatureDifferential: 4, "origin.city": "Fresno", "origin.state": "CA", "origin.zip": "93706", "destination.city": "Denver", "destination.state": "CO", "destination.zip": "80216", "trailerType.name": "53' reefer", "trailerType.code": "RF53" } },
  { id: "shp_fs_240772", pro: "S-240772", lane: "Houston, TX → Baton Rouge, LA", meta: "271 mi · 4 stops · hazmat", customer: "Gulf Coast Chemical", values: { baseRate: 3.05, totalDistance: 271, totalStops: 4, totalWeight: 22600, totalHours: 9.25, hasHazmat: true, "origin.city": "Houston", "origin.zip": "77029", "destination.city": "Baton Rouge", "destination.state": "LA", "destination.zip": "70805" } },
  { id: "shp_fs_240758", pro: "S-240758", lane: "Dallas, TX → Fort Worth, TX", meta: "38 mi · 2 stops · dry van", customer: "Crestline Retail", values: { baseRate: 2.85, totalDistance: 38, totalStops: 2, totalWeight: 12400, totalPieces: 9, totalHours: 3.5, pickupDayOfWeek: 3, "destination.city": "Fort Worth", "destination.state": "TX", "destination.zip": "76102" } },
  { id: "shp_fs_240741", pro: "S-240741", lane: "Memphis, TN → Chicago, IL", meta: "532 mi · 5 stops · weekend", customer: "Acme Foods", values: { baseRate: 2.7, totalDistance: 532, totalStops: 5, totalWeight: 30500, totalPieces: 18, totalHours: 16, pickupDayOfWeek: 0, isWeekendPickup: true, "origin.city": "Memphis", "origin.state": "TN", "origin.zip": "38118", "destination.city": "Chicago", "destination.state": "IL", "destination.zip": "60609" } },
  { id: "shp_fs_240733", pro: "S-240733", lane: "Phoenix, AZ → Los Angeles, CA", meta: "372 mi · 2 stops · reefer, no stop times", customer: "Harborline Grocers", values: { baseRate: 3.25, totalDistance: 372, totalStops: 2, totalWeight: 36100, totalHours: null, pickupDayOfWeek: 4, requiresTemperatureControl: true, temperatureMin: 33, temperatureMax: 36, temperatureDifferential: 3, "origin.city": "Phoenix", "origin.state": "AZ", "origin.zip": "85043", "destination.city": "Los Angeles", "destination.state": "CA", "destination.zip": "90021", "trailerType.name": "53' reefer", "trailerType.code": "RF53" } },
  { id: "shp_fs_240719", pro: "S-240719", lane: "Denver, CO → Salt Lake City, UT", meta: "525 mi · 3 stops · hazmat", customer: "Gulf Coast Chemical", values: { baseRate: 3.1, totalDistance: 525, totalStops: 3, totalWeight: 28800, totalHours: 11, hasHazmat: true, pickupDayOfWeek: 2, "origin.city": "Denver", "origin.state": "CO", "origin.zip": "80216", "destination.city": "Salt Lake City", "destination.state": "UT", "destination.zip": "84104" } },
];

/**
 * The shipments a backtest or approval impact re-rates: the picker's seven, then recent freight
 * drawn the way the prototype's backtest draws it (formulatemplateservice/backtest.go rates the
 * template's own recently rated shipments; here every template shares one pool of 128).
 */
export function backtestShipments() {
  let seed = 7;
  const rnd = () => (seed = (seed * 16807) % 2147483647) / 2147483647;
  const rows = SHIPMENTS.map((s) => ({ id: s.id, pro: s.pro, values: s.values }));
  for (let i = rows.length; i < 128; i++) {
    const v = {
      baseRate: +(2.4 + rnd() * 1.2).toFixed(2),
      totalDistance: Math.round(40 + rnd() * 1300),
      totalStops: 2 + Math.floor(rnd() * rnd() * 5),
      hasHazmat: rnd() < 0.08,
      isWeekendPickup: rnd() < 0.16,
      requiresTemperatureControl: rnd() < 0.2,
      totalWeight: Math.round(8000 + rnd() * 36000),
      totalHours: +(4 + rnd() * 30).toFixed(1),
    };
    v.pickupDayOfWeek = v.isWeekendPickup ? 6 : 1 + Math.floor(rnd() * 5);
    rows.push({ id: `shp_fs_bt_${239400 + i * 7}`, pro: `S-${239400 + i * 7}`, values: v });
  }
  return rows;
}

/** standard_templates.yaml, as the prototype lists it. */
export const STANDARDS = [
  { name: "Per Mile", type: "FreightCharge", expression: "baseRate * totalDistance", description: "Multiplies the base rate per mile by total shipment distance.", variables: [] },
  { name: "Per Mile with Minimum", type: "FreightCharge", expression: "max(minimumCharge, baseRate * totalDistance)", description: "Per mile rate with a minimum charge floor.", variables: [{ name: "minimumCharge", type: "Number", defaultValue: 250, required: true, description: "Minimum charge amount" }] },
  { name: "Per Mile with Fuel Surcharge", type: "FreightCharge", expression: "round((baseRate * totalDistance) * (1 + fuelSurchargePercent / 100), 2)", description: "Per mile rate with a percentage fuel surcharge on top.", variables: [{ name: "fuelSurchargePercent", type: "Number", defaultValue: 18, required: true, description: "Fuel surcharge percentage" }] },
  { name: "Per CWT", type: "FreightCharge", expression: "baseRate * ceil(totalWeight / 100)", description: "Base rate per hundredweight, weight rounded up to the next 100 lb.", variables: [] },
  { name: "Per Stop", type: "FreightCharge", expression: "baseRate * totalStops", description: "Base rate per stop times the number of stops.", variables: [] },
  { name: "Per Hour", type: "FreightCharge", expression: "baseRate * totalHours", description: "Base rate per hour of service.", variables: [] },
  { name: "Flat Rate", type: "FreightCharge", expression: "baseRate", description: "The base rate entered on the shipment.", variables: [] },
  { name: "Hazmat Surcharge", type: "AccessorialCharge", expression: "hasHazmat ? hazmatFee : 0", description: "A hazmat fee only when the shipment carries hazardous materials.", variables: [{ name: "hazmatFee", type: "Number", defaultValue: 175, required: true, description: "Hazardous materials handling fee" }] },
  { name: "Additional Stop Fee", type: "AccessorialCharge", expression: "totalStops > baseStops ? (totalStops - baseStops) * additionalStopFee : 0", description: "First stops included, a fee for each one after.", variables: [{ name: "baseStops", type: "Number", defaultValue: 2, required: true, description: "Stops included in the base rate" }, { name: "additionalStopFee", type: "Number", defaultValue: 50, required: true, description: "Fee per additional stop" }] },
];

export const USAGE = [
  { type: "Rate agreement lanes", count: 6 },
  { type: "Rate matrices", count: 2 },
  { type: "Shipments rated this month", count: 214 },
];

const EDIT_EXPRESSION = `let linehaul = baseRate * totalDistance;
let fuel = linehaul * (fuelSurchargePercent / 100);
let stops = max(totalStops - includedStops, 0) * stopFee;
linehaul + fuel + stops + (hasHazmat ? hazmatFee : 0)`;

/** The draft's inputs; hazmatFee is the prototype's one input without a description. */
function editVariables(fuelSurchargePercent) {
  return [
    { name: "fuelSurchargePercent", type: "Number", description: "Fuel surcharge as a percent of linehaul", required: true, defaultValue: fuelSurchargePercent },
    { name: "includedStops", type: "Number", description: "Stops the linehaul already covers", required: true, defaultValue: 2 },
    { name: "stopFee", type: "Number", description: "Fee for each stop past the included ones", required: true, defaultValue: 50 },
    { name: "hazmatFee", type: "Number", description: "", required: false, defaultValue: 175 },
  ];
}

const EDIT_BREAKDOWN = [
  { name: "linehaul", label: "Linehaul", expression: "baseRate * totalDistance" },
  { name: "fuelSurcharge", label: "Fuel surcharge", expression: "baseRate * totalDistance * fuelSurchargePercent / 100" },
  { name: "stopCharges", label: "Additional stops", expression: "max(totalStops - includedStops, 0) * stopFee" },
  { name: "hazmat", label: "Hazmat", expression: "hasHazmat ? hazmatFee : 0" },
];

const DESCRIPTION =
  "Regional dry van lanes. Per-mile linehaul with a percentage fuel surcharge, a fee for each stop past the second, and a hazmat fee.";

/** The template content the engine prices with: everything a version snapshot holds. */
function content({ name, description = DESCRIPTION, expression = EDIT_EXPRESSION, fuel, breakdown = EDIT_BREAKDOWN }) {
  return {
    name,
    description,
    type: "FreightCharge",
    expression,
    schemaId: "shipment",
    variableDefinitions: editVariables(fuel),
    breakdownDefinitions: breakdown.map((line) => ({ ...line })),
    minCharge: "250",
    maxCharge: null,
    roundingMode: "HalfUp",
    roundingPrecision: 2,
  };
}

/** The team the other fixtures use (Triage's), each a full user as the server serves one. */
export const TEAM = [
  { id: "usr_priya", name: "Priya Shah", username: "priya.shah", emailAddress: "priya.shah@trenova.app", jobTitle: "Pricing lead" },
  { id: "usr_marcus", name: "Marcus Lee", username: "marcus.lee", emailAddress: "marcus.lee@trenova.app", jobTitle: "Controller" },
  { id: "usr_dana", name: "Dana Whitfield", username: "dana.whitfield", emailAddress: "dana.whitfield@trenova.app", jobTitle: "Billing" },
];

export function avatarUrl(id) {
  return `${MOCK_ORIGIN}/__mock/avatar/${id}.svg`;
}

export function userRecord(id) {
  const person = id === USER.id ? { ...USER, jobTitle: "Administrator" } : TEAM.find((u) => u.id === id);
  if (!person) return null;
  const picture = id === USER.id ? "" : avatarUrl(id);
  return {
    id: person.id,
    version: 1,
    createdAt: 1700000000,
    updatedAt: 1700000000,
    businessUnitId: USER.businessUnitId,
    currentOrganizationId: ORG.id,
    status: "Active",
    name: person.name,
    username: person.username,
    emailAddress: person.emailAddress,
    profilePicUrl: picture,
    thumbnailUrl: picture,
    timezone: "America/Chicago",
    timeFormat: "12-hour",
    locale: "en",
    isLocked: false,
    mustChangePassword: false,
    jobTitle: person.jobTitle ?? "",
  };
}

const DAY = 86400;

/**
 * Builds one saved template: its row, version snapshots, scenarios and review history. `versions`
 * lists each snapshot's content, status, author and age; the row carries the last one's number.
 */
function savedTemplate(anchor, spec) {
  const versions = spec.versions.map((version, index) => ({
    versionNumber: index + 1,
    status: version.status,
    createdById: version.by,
    createdAt: anchor - version.daysAgo * DAY,
    changeMessage: version.message,
    tags: version.tags ?? [],
    content: version.content,
  }));
  const current = versions.at(-1);
  const approved = [...versions].reverse().find((version) => version.status === "Active");
  return {
    id: spec.id,
    status: spec.status,
    ...structuredClone(current.content),
    submittedById: spec.submittedById ?? null,
    submittedAt: spec.submittedDaysAgo != null ? anchor - spec.submittedDaysAgo * DAY : null,
    approvedById: spec.status === "Active" ? (spec.approvedById ?? null) : null,
    approvedAt: spec.status === "Active" && approved ? approved.createdAt : null,
    reviewComment: spec.reviewComment ?? "",
    metadata: {},
    version: spec.rowVersion ?? versions.length + 2,
    sourceTemplateId: spec.sourceTemplateId ?? null,
    sourceVersionNumber: spec.sourceVersionNumber ?? null,
    currentVersionNumber: current.versionNumber,
    createdAt: versions[0].createdAt,
    updatedAt: current.createdAt + 600,
    versions,
    usages: spec.usages ?? [],
    testCases: spec.testCases.map((tc, index) => ({
      id: `${spec.id.replace("fmt_", "ftc_")}_${index + 1}`,
      templateId: spec.id,
      name: tc.name,
      description: tc.description ?? "",
      variables: tc.variables,
      expectedAmount: tc.expected,
      tolerance: "0.01",
      version: 1,
      createdById: tc.by,
      recordedOnVersion: tc.recordedOnVersion,
      recordedById: tc.by,
      recordedAt: anchor - tc.daysAgo * DAY,
      recordedContent: structuredClone(versions[tc.recordedFrom - 1].content),
      createdAt: anchor - tc.createdDaysAgo * DAY,
      updatedAt: anchor - tc.daysAgo * DAY,
    })),
    reviews: spec.reviews.map((review, index) => ({
      id: `${spec.id.replace("fmt_", "ftr_")}_${index + 1}`,
      templateId: spec.id,
      round: review.round,
      decision: review.decision,
      actorId: review.by,
      comment: review.comment ?? "",
      baseVersionNumber: review.base,
      createdAt: anchor - review.daysAgo * DAY,
      requestedReviewerId: review.reviewerId ?? null,
      reviewerNote: review.reviewerNote ?? "",
    })),
  };
}

/**
 * The prototype's four scenarios. The first three were recorded on `base` and the fourth on
 * `weekend`: each says which version number it was recorded on (`on`), which snapshot holds the
 * content it was computed from (`from`), who recorded it and how long ago.
 */
function scenarios({ weekendExpected, base, weekend }) {
  const stamp = (at) => ({ recordedOnVersion: at.on, recordedFrom: at.from, by: at.by, daysAgo: at.daysAgo, createdDaysAgo: at.createdDaysAgo ?? at.daysAgo });
  return [
    { name: "Dallas → Atlanta, 2 stops", variables: { baseRate: 2.85, totalDistance: 781, totalStops: 2, hasHazmat: false }, expected: "2626.5", ...stamp(base) },
    { name: "Short haul lands on the minimum", description: "A 60-mile load bills the $250 minimum, not the formula's $201.78.", variables: { baseRate: 2.85, totalDistance: 60, totalStops: 2, hasHazmat: false }, expected: "250", ...stamp(base) },
    { name: "Hazmat, 4 stops", variables: { baseRate: 2.85, totalDistance: 500, totalStops: 4, hasHazmat: true }, expected: "1956.5", ...stamp(base) },
    { name: "Weekend reefer, Fresno", variables: { baseRate: 3.1, totalDistance: 520, totalStops: 2, hasHazmat: false }, expected: weekendExpected, ...stamp(weekend) },
  ];
}

/** The three saved templates, built fresh for each scenario. */
export function createFormulaTemplates(anchor) {
  const draftName = "Regional per mile + fuel";
  const draftV1 = content({
    name: draftName,
    expression: `let linehaul = baseRate * totalDistance;
let fuel = linehaul * (fuelSurchargePercent / 100);
let stops = max(totalStops - includedStops, 0) * stopFee;
linehaul + fuel + stops`,
    fuel: 14,
    breakdown: [],
  });
  const draftV3 = content({ name: draftName, fuel: 14, breakdown: [] });
  const draftV4 = content({ name: draftName, fuel: 18 });

  const draft = savedTemplate(anchor, {
    id: DRAFT_TEMPLATE_ID,
    status: "Draft",
    reviewComment: "Approved for the regional lanes.",
    versions: [
      { content: draftV1, status: "Draft", by: "usr_dana", daysAgo: 40, message: "Initial version" },
      { content: draftV3, status: "Draft", by: "usr_dana", daysAgo: 33, message: "" },
      { content: draftV3, status: "Active", by: "usr_priya", daysAgo: 30, message: "Approved", tags: ["Production"] },
      { content: draftV4, status: "Draft", by: USER.id, daysAgo: 1, message: "" },
    ],
    usages: USAGE,
    testCases: scenarios({
      weekendExpected: "1837.68",
      base: { on: 4, from: 4, by: USER.id, daysAgo: 1, createdDaysAgo: 21 },
      weekend: { on: 3, from: 3, by: "usr_priya", daysAgo: 12 },
    }),
    reviews: [
      { round: 1, decision: "Submitted", by: "usr_dana", base: 0, daysAgo: 38, comment: "First cut of the regional per-mile method." },
      { round: 1, decision: "ChangesRequested", by: "usr_marcus", base: 0, daysAgo: 36, comment: "Hazmat loads on these lanes carry a handling fee. Add it before this goes live." },
      { round: 1, decision: "Submitted", by: "usr_dana", base: 0, daysAgo: 33, comment: "Added hazmatFee, $175 by default." },
      { round: 1, decision: "Approved", by: "usr_priya", base: 0, daysAgo: 30, comment: "Approved for the regional lanes." },
    ],
  });

  const reviewName = "Regional per mile + fuel · Southeast";
  const reviewV1 = content({ name: reviewName, fuel: 16, breakdown: [] });
  const reviewV3 = content({ name: reviewName, fuel: 18 });
  const review = savedTemplate(anchor, {
    id: REVIEW_TEMPLATE_ID,
    status: "InReview",
    submittedById: "usr_dana",
    submittedDaysAgo: 2,
    reviewComment: "Fuel to 18% to match the DOE index this quarter, and the charge now itemizes on the invoice.",
    sourceTemplateId: DRAFT_TEMPLATE_ID,
    sourceVersionNumber: 3,
    versions: [
      { content: reviewV1, status: "Draft", by: "usr_dana", daysAgo: 26, message: "Forked from Regional per mile + fuel v3" },
      { content: reviewV1, status: "Active", by: "usr_marcus", daysAgo: 24, message: "Approved", tags: ["Production"] },
      { content: reviewV3, status: "Draft", by: "usr_dana", daysAgo: 2, message: "" },
    ],
    usages: [
      { type: "Rate agreement lanes", count: 3 },
      { type: "Shipments rated this month", count: 88 },
    ],
    testCases: scenarios({
      weekendExpected: "1902.16",
      base: { on: 3, from: 3, by: "usr_dana", daysAgo: 2, createdDaysAgo: 25 },
      weekend: { on: 3, from: 3, by: "usr_dana", daysAgo: 2 },
    }),
    reviews: [
      { round: 1, decision: "Submitted", by: "usr_dana", base: 0, daysAgo: 25, comment: "Southeast copy of the regional method." },
      { round: 1, decision: "Approved", by: "usr_marcus", base: 0, daysAgo: 24, comment: "" },
      {
        round: 2,
        decision: "Submitted",
        by: "usr_dana",
        base: 2,
        daysAgo: 2,
        comment: "Fuel to 18% to match the DOE index this quarter, and the charge now itemizes on the invoice.",
        reviewerId: USER.id,
        reviewerNote:
          "Raises `fuelSurchargePercent` from 16 to 18 and splits the charge into linehaul, fuel, stop and hazmat lines on the invoice. Every scenario was re-run against this version and passes.",
      },
    ],
  });

  const activeName = "Regional per mile + fuel · Mountain West";
  const activeV1 = content({ name: activeName, fuel: 18 });
  const active = savedTemplate(anchor, {
    id: ACTIVE_TEMPLATE_ID,
    status: "Active",
    approvedById: "usr_priya",
    reviewComment: "Good to go.",
    sourceTemplateId: DRAFT_TEMPLATE_ID,
    sourceVersionNumber: 3,
    versions: [
      { content: activeV1, status: "Draft", by: "usr_marcus", daysAgo: 16, message: "Forked from Regional per mile + fuel v3" },
      { content: activeV1, status: "Active", by: "usr_priya", daysAgo: 14, message: "Approved", tags: ["Production", "Stable"] },
    ],
    usages: USAGE,
    testCases: scenarios({
      weekendExpected: "1902.16",
      base: { on: 0, from: 1, by: "usr_marcus", daysAgo: 16 },
      weekend: { on: 0, from: 1, by: "usr_marcus", daysAgo: 16 },
    }),
    reviews: [
      { round: 1, decision: "Submitted", by: "usr_marcus", base: 0, daysAgo: 15, comment: "Mountain West lanes, same method at 18% fuel." },
      { round: 1, decision: "Approved", by: "usr_priya", base: 0, daysAgo: 14, comment: "Good to go." },
    ],
  });

  return [draft, review, active];
}

export { USER, ORG };
