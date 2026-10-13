// Shipment schema (client/packages/shared/src/types/formula-template.ts SHIPMENT_VARIABLES + AVAILABLE_FUNCTIONS)
const FS_VARS = [
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
const FS_CATS = { custom: "This template's inputs", charges: "Rates & charges", computed: "Computed", shipment: "Shipment", equipment: "Equipment", location: "Origin & destination" };

const FS_FUNCS = [
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

const FS_TABLES = {
  zone_rates: { TX: 2.35, GA: 2.6, CO: 3.1, LA: 2.45, CA: 3.4 },
};

const FS_SAMPLE = {
  baseRate: 2.85, freightChargeAmount: 0, otherChargeAmount: 0, currentTotalCharge: 0,
  totalDistance: 742, totalStops: 3, totalWeight: 38400, totalPieces: 22, totalLinearFeet: 44, totalHours: null,
  pickupDayOfWeek: 2, pickupHour: 8, pickupMonth: 10, isWeekendPickup: false, hasHazmat: false,
  requiresTemperatureControl: false, temperatureDifferential: null, weight: 38400, pieces: 22,
  temperatureMin: null, temperatureMax: null,
  "tractorType.name": "Day cab", "tractorType.code": "DC", "tractorType.costPerMile": 1.42,
  "trailerType.name": "53' dry van", "trailerType.code": "DV53", "trailerType.costPerMile": 0.18,
  "origin.city": "Dallas", "origin.state": "TX", "origin.zip": "75207",
  "destination.city": "Atlanta", "destination.state": "GA", "destination.zip": "30318",
};

const FS_SHIPMENTS = [
  { id: "S-240811", lane: "Dallas, TX → Atlanta, GA", meta: "781 mi · 2 stops · dry van", values: { baseRate: 2.85, totalDistance: 781, totalStops: 2, totalWeight: 41200, totalPieces: 24, totalHours: 14.5, pickupDayOfWeek: 1, isWeekendPickup: false } },
  { id: "S-240794", lane: "Fresno, CA → Denver, CO", meta: "1,146 mi · 3 stops · reefer", values: { baseRate: 3.4, totalDistance: 1146, totalStops: 3, totalWeight: 39800, totalHours: 31.5, pickupDayOfWeek: 6, isWeekendPickup: true, requiresTemperatureControl: true, temperatureMin: 34, temperatureMax: 38, temperatureDifferential: 4, "origin.city": "Fresno", "origin.state": "CA", "origin.zip": "93706", "destination.city": "Denver", "destination.state": "CO", "destination.zip": "80216", "trailerType.name": "53' reefer", "trailerType.code": "RF53" } },
  { id: "S-240772", lane: "Houston, TX → Baton Rouge, LA", meta: "271 mi · 4 stops · hazmat", values: { baseRate: 3.05, totalDistance: 271, totalStops: 4, totalWeight: 22600, totalHours: 9.25, hasHazmat: true, "origin.city": "Houston", "origin.zip": "77029", "destination.city": "Baton Rouge", "destination.state": "LA", "destination.zip": "70805" } },
];
const FS_FIELD_SOURCE = new Set(["baseRate", "freightChargeAmount", "otherChargeAmount", "currentTotalCharge", "weight", "pieces", "temperatureMin", "temperatureMax", "origin.city", "origin.state", "origin.zip", "destination.city", "destination.state", "destination.zip", "tractorType.name", "tractorType.code", "tractorType.costPerMile", "trailerType.name", "trailerType.code", "trailerType.costPerMile"]);

// standard_templates.yaml
const FS_STANDARDS = [
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
const FS_ORG_TEMPLATES = [
  { name: "Regional per mile + fuel", meta: "Active · v4", expression: "linehaul + fuel + stops + hazmat" },
  { name: "Southeast reefer", meta: "Active · v7", expression: "baseRate * totalDistance + (requiresTemperatureControl ? 225 : 0)" },
  { name: "LTL weight breaks", meta: "Draft · v2", expression: 'deficitWeight("ltl_breaks", totalWeight)' },
];

const FS_EDIT_EXPR = `let linehaul = baseRate * totalDistance;
let fuel = linehaul * (fuelSurchargePercent / 100);
let stops = max(totalStops - includedStops, 0) * stopFee;
linehaul + fuel + stops + (hasHazmat ? hazmatFee : 0)`;

let fsSeq = 0;
const fsId = (p = "id") => `${p}_${++fsSeq}`;

const FS_EDIT_TEMPLATE = {
  name: "Regional per mile + fuel",
  type: "FreightCharge",
  description: "Regional dry van lanes. Per-mile linehaul with a percentage fuel surcharge, a fee for each stop past the second, and a hazmat fee.",
  expression: FS_EDIT_EXPR,
  variables: [
    { id: "v1", name: "fuelSurchargePercent", type: "Number", defaultValue: 18, required: true, description: "Fuel surcharge as a percent of linehaul" },
    { id: "v2", name: "includedStops", type: "Number", defaultValue: 2, required: true, description: "Stops the linehaul already covers" },
    { id: "v3", name: "stopFee", type: "Number", defaultValue: 50, required: true, description: "Fee for each stop past the included ones" },
    { id: "v4", name: "hazmatFee", type: "Number", defaultValue: 175, required: false, description: "" },
  ],
  breakdown: [
    { id: "b1", name: "linehaul", label: "Linehaul", expression: "baseRate * totalDistance" },
    { id: "b2", name: "fuelSurcharge", label: "Fuel surcharge", expression: "baseRate * totalDistance * fuelSurchargePercent / 100" },
    { id: "b3", name: "stopCharges", label: "Additional stops", expression: "max(totalStops - includedStops, 0) * stopFee" },
    { id: "b4", name: "hazmat", label: "Hazmat", expression: "hasHazmat ? hazmatFee : 0" },
  ],
  policy: { min: 250, max: null, mode: "HalfUp", precision: 2 },
};
const FS_EMPTY_TEMPLATE = { name: "", type: "FreightCharge", description: "", expression: "", variables: [], breakdown: [], policy: { min: null, max: null, mode: "HalfUp", precision: 2 } };

const FS_EDIT_SCENARIOS = [
  { id: "sc1", name: "Dallas → Atlanta, 2 stops", inputs: { baseRate: 2.85, totalDistance: 781, totalStops: 2, hasHazmat: false }, expected: 2626.5 },
  { id: "sc2", name: "Short haul lands on the minimum", inputs: { baseRate: 2.85, totalDistance: 60, totalStops: 2, hasHazmat: false }, expected: 250 },
  { id: "sc3", name: "Hazmat, 4 stops", inputs: { baseRate: 2.85, totalDistance: 500, totalStops: 4, hasHazmat: true }, expected: 1956.5 },
  { id: "sc4", name: "Weekend reefer, Fresno", inputs: { baseRate: 3.1, totalDistance: 520, totalStops: 2, hasHazmat: false }, expected: 1837.68, recordedOn: { version: 3, by: "Priya Raman", variables: { fuelSurchargePercent: 14 } } },
];

const FS_USAGE = [
  { type: "Rate agreement lanes", count: 6 },
  { type: "Rate matrices", count: 2 },
  { type: "Shipments rated this month", count: 214 },
];

const FS_ROUNDING = [
  { value: "HalfUp", label: "Half up" },
  { value: "HalfEven", label: "Half even" },
  { value: "Up", label: "Up" },
  { value: "Down", label: "Down" },
  { value: "None", label: "None" },
];

Object.assign(window, { FS_VARS, FS_CATS, FS_FUNCS, FS_TABLES, FS_SAMPLE, FS_SHIPMENTS, FS_FIELD_SOURCE, FS_STANDARDS, FS_ORG_TEMPLATES, FS_EDIT_TEMPLATE, FS_EMPTY_TEMPLATE, FS_EDIT_SCENARIOS, FS_USAGE, FS_ROUNDING, fsId });
