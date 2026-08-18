import { z } from "zod";
import {
  decimalStringSchema,
  optionalStringSchema,
  statusSchema,
  tenantInfoSchema,
} from "./helpers";

/* -------------------------------------------------------------------------- */
/*                                  Geography                                  */
/* -------------------------------------------------------------------------- */

/**
 * How one end of a lane names a place.
 *
 * The order matters: it runs from the widest reading to the narrowest, which is
 * the order the resolver ranks them in. A rule written at a narrower scope beats
 * one written at a wider scope covering the same shipment.
 */
export const rateScopeTypeSchema = z.enum([
  "Any",
  "Country",
  "State",
  "Zone",
  "Radius",
  "CityState",
  "Zip3",
  "Zip5",
  "Location",
]);
export type RateScopeType = z.infer<typeof rateScopeTypeSchema>;

export const rateZoneKindSchema = z.enum(["Custom", "KMA", "Regional", "Metro", "Country"]);
export type RateZoneKind = z.infer<typeof rateZoneKindSchema>;

export const rateZoneMemberSchema = z.object({
  id: optionalStringSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  rateZoneId: optionalStringSchema,
  scopeType: rateScopeTypeSchema,
  scopeValue: z.string().default(""),
  city: z.string().default(""),
  matchKey: optionalStringSchema,
});
export type RateZoneMember = z.infer<typeof rateZoneMemberSchema>;

export const rateZoneSchema = z.object({
  ...tenantInfoSchema.shape,

  code: z
    .string({ error: "Code is required" })
    .min(2, { error: "Code must be at least 2 characters" })
    .max(50, { error: "Code must be less than 50 characters" }),
  name: z
    .string({ error: "Name is required" })
    .min(1, { error: "Name is required" })
    .max(100, { error: "Name must be less than 100 characters" }),
  description: z.string().max(500).default(""),
  kind: rateZoneKindSchema.default("Custom"),
  status: statusSchema.default("Active"),
  members: z.array(rateZoneMemberSchema).default([]),
});
export type RateZone = z.infer<typeof rateZoneSchema>;

/* -------------------------------------------------------------------------- */
/*                                  Matrices                                   */
/* -------------------------------------------------------------------------- */

export const rateMatrixDimensionKindSchema = z.enum([
  "Zone",
  "Zip3",
  "Zip5",
  "State",
  "Country",
  "WeightBreak",
  "Distance",
  "PieceCount",
  "LinearFeet",
  "FreightClass",
  "EquipmentType",
  "ServiceType",
  "Custom",
]);
export type RateMatrixDimensionKind = z.infer<typeof rateMatrixDimensionKindSchema>;

export const rateMatrixMatchModeSchema = z.enum(["Exact", "Range"]);
export type RateMatrixMatchMode = z.infer<typeof rateMatrixMatchModeSchema>;

/**
 * What a cell's number means, which is what tells the engine how to spend it.
 *
 * The same grid of numbers is a per-mile tariff, a hundredweight tariff or a
 * discount table depending on this one field, so it is stated on the matrix
 * rather than inferred from the lane that reads it.
 */
export const rateMatrixValueKindSchema = z.enum([
  "FlatRate",
  "PerMile",
  "PerCwt",
  "PerPiece",
  "PerStop",
  "Percent",
  "Discount",
  "MinimumOnly",
]);
export type RateMatrixValueKind = z.infer<typeof rateMatrixValueKindSchema>;

export const rateMatrixRoundingModeSchema = z.enum(["HalfUp", "HalfEven", "Up", "Down", "None"]);

export const rateMatrixDimensionSchema = z.object({
  id: optionalStringSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  rateMatrixId: optionalStringSchema,
  position: z.number().int().min(0).max(3),
  kind: rateMatrixDimensionKindSchema,
  matchMode: rateMatrixMatchModeSchema,
  label: z.string().default(""),
});
export type RateMatrixDimension = z.infer<typeof rateMatrixDimensionSchema>;

export const rateMatrixSchema = z.object({
  ...tenantInfoSchema.shape,

  code: z
    .string({ error: "Code is required" })
    .min(2, { error: "Code must be at least 2 characters" })
    .max(64, { error: "Code must be less than 64 characters" }),
  name: z.string({ error: "Name is required" }).min(1, { error: "Name is required" }).max(100),
  description: z.string().max(500).default(""),
  status: statusSchema.default("Active"),
  valueKind: rateMatrixValueKindSchema.default("FlatRate"),
  currency: z.string().length(3).default("USD"),
  roundingMode: rateMatrixRoundingModeSchema.default("HalfUp"),
  roundingPrecision: z.number().int().min(0).max(6).default(2),
  dimensions: z.array(rateMatrixDimensionSchema).default([]),
});
export type RateMatrix = z.infer<typeof rateMatrixSchema>;

/**
 * One priced coordinate in the grid.
 *
 * Every axis carries both a key and a band because a matrix mixes match modes:
 * origin zone is looked up by key, weight break by band, and a cell has to say
 * where it sits on each. The axis the matrix declares as Exact reads the key
 * and ignores the band, and the reverse for Range.
 *
 * Bands are half open — `[min, max)` — mirroring
 * `ratematrix.RateMatrixCell.ContainsQuantity`. A tariff written "1000 to 2000"
 * rates two thousand pounds at the next tier up, and closed intervals would let
 * two adjacent bands both claim the boundary.
 */
export const rateMatrixCellSchema = z.object({
  id: optionalStringSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  rateMatrixId: optionalStringSchema,

  d0Key: z.string().max(120).default(""),
  d1Key: z.string().max(120).default(""),
  d2Key: z.string().max(120).default(""),
  d3Key: z.string().max(120).default(""),

  d0Min: decimalStringSchema.nullish(),
  d0Max: decimalStringSchema.nullish(),
  d1Min: decimalStringSchema.nullish(),
  d1Max: decimalStringSchema.nullish(),
  d2Min: decimalStringSchema.nullish(),
  d2Max: decimalStringSchema.nullish(),
  d3Min: decimalStringSchema.nullish(),
  d3Max: decimalStringSchema.nullish(),

  value: decimalStringSchema,
  minCharge: decimalStringSchema.nullish(),
  deficitEligible: z.boolean().default(true),
});
export type RateMatrixCell = z.infer<typeof rateMatrixCellSchema>;

/* -------------------------------------------------------------------------- */
/*                                 Agreements                                  */
/* -------------------------------------------------------------------------- */

export const ratePartyTypeSchema = z.enum(["Customer", "Carrier"]);
export type RatePartyType = z.infer<typeof ratePartyTypeSchema>;

export const rateAgreementTypeSchema = z.enum([
  "Contract",
  "Tariff",
  "Spot",
  "Project",
  "Dedicated",
]);
export type RateAgreementType = z.infer<typeof rateAgreementTypeSchema>;

export const rateAgreementStatusSchema = z.enum([
  "Draft",
  "InReview",
  "Active",
  "Suspended",
  "Expired",
  "Archived",
]);
export type RateAgreementStatus = z.infer<typeof rateAgreementStatusSchema>;

export const rateRoundingModeSchema = z.enum(["HalfUp", "HalfEven", "Up", "Down", "None"]);
export type RateRoundingMode = z.infer<typeof rateRoundingModeSchema>;

/** How a lane arrives at its linehaul. */
export const ratingBasisSchema = z.enum([
  "Flat",
  "PerMile",
  "PerCwt",
  "PerPiece",
  "PerStop",
  "PerPallet",
  "PerLinearFoot",
  "PerHour",
  "Percent",
  "Matrix",
  "Formula",
]);
export type RatingBasis = z.infer<typeof ratingBasisSchema>;

export const ratePercentBasisSchema = z.enum(["Linehaul", "LinehaulPlusAccessorials", "SellTotal"]);
export type RatePercentBasis = z.infer<typeof ratePercentBasisSchema>;

export const rateDirectionSchema = z.enum(["Directional", "Bidirectional"]);
export type RateDirection = z.infer<typeof rateDirectionSchema>;

export const freightClassSourceSchema = z.enum(["Commodity", "Fixed", "Density"]);
export type FreightClassSource = z.infer<typeof freightClassSourceSchema>;

export const rateRuleStatusSchema = z.enum(["Active", "Inactive"]);
export type RateRuleStatus = z.infer<typeof rateRuleStatusSchema>;

export const rateAgreementRuleBreakSchema = z.object({
  id: optionalStringSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  rateAgreementRuleId: optionalStringSchema,
  minQuantity: decimalStringSchema,
  maxQuantity: decimalStringSchema,
  rate: decimalStringSchema,
  minCharge: decimalStringSchema,
  sortOrder: z.number().int().default(0),
});
export type RateAgreementRuleBreak = z.infer<typeof rateAgreementRuleBreakSchema>;

/**
 * One priced lane of an agreement.
 *
 * `laneKey` and `specificityScore` are derived server-side on every write, so
 * they are read-only here — a client that sent its own would only be able to
 * make the stored value disagree with the fields it was derived from.
 */
export const rateAgreementRuleSchema = z
  .object({
    ...tenantInfoSchema.shape,

    rateAgreementId: optionalStringSchema,
    label: z.string().max(150).default(""),
    status: rateRuleStatusSchema.default("Active"),

    originScopeType: rateScopeTypeSchema.default("Any"),
    originScopeValue: z.string().default(""),
    originCity: z.string().default(""),
    destinationScopeType: rateScopeTypeSchema.default("Any"),
    destinationScopeValue: z.string().default(""),
    destinationCity: z.string().default(""),
    direction: rateDirectionSchema.default("Directional"),

    laneKey: optionalStringSchema,
    specificityScore: z.number().int().optional(),

    originRadiusMeters: decimalStringSchema,
    destinationRadiusMeters: decimalStringSchema,
    originLatitude: decimalStringSchema,
    originLongitude: decimalStringSchema,
    destinationLatitude: decimalStringSchema,
    destinationLongitude: decimalStringSchema,

    serviceTypeIds: z.array(z.string()).nullish(),
    shipmentTypeIds: z.array(z.string()).nullish(),
    tractorTypeIds: z.array(z.string()).nullish(),
    trailerTypeIds: z.array(z.string()).nullish(),
    commodityIds: z.array(z.string()).nullish(),
    freightClasses: z.array(z.string()).nullish(),
    serviceModels: z.array(z.string()).nullish(),
    equipmentClasses: z.array(z.string()).nullish(),

    minWeight: decimalStringSchema,
    maxWeight: decimalStringSchema,
    minDistance: decimalStringSchema,
    maxDistance: decimalStringSchema,
    minStops: z.number().int().nullish(),
    maxStops: z.number().int().nullish(),
    daysOfWeek: z.number().int().default(0),
    hazmatOnly: z.boolean().default(false),
    tempControlOnly: z.boolean().default(false),

    ratingBasis: ratingBasisSchema,
    rate: decimalStringSchema,
    rateMatrixId: z.string().nullish(),
    formulaTemplateId: z.string().nullish(),
    percentBasis: ratePercentBasisSchema.nullish(),
    currency: z.string().nullish(),

    freightClassSource: freightClassSourceSchema.default("Commodity"),
    fixedFreightClass: z.string().nullish(),
    densityScaleId: z.string().nullish(),
    discountPercent: decimalStringSchema,
    absoluteMinCharge: decimalStringSchema,
    allowDeficitRating: z.boolean().default(true),

    minCharge: decimalStringSchema,
    maxCharge: decimalStringSchema,
    minBillableDistance: decimalStringSchema,
    roundingMode: rateRoundingModeSchema.nullish(),

    priority: z.number().int().default(0),
    effectiveFrom: z.number().int(),
    effectiveTo: z.number().int().nullish(),
    supersedesRuleId: z.string().nullish(),

    breaks: z.array(rateAgreementRuleBreakSchema).default([]),
  })
  .refine((rule) => rule.ratingBasis !== "Matrix" || Boolean(rule.rateMatrixId), {
    path: ["rateMatrixId"],
    message: "A matrix rated lane needs a rate matrix",
  })
  .refine((rule) => rule.ratingBasis !== "Formula" || Boolean(rule.formulaTemplateId), {
    path: ["formulaTemplateId"],
    message: "A formula rated lane needs a formula template",
  })
  .refine(
    (rule) =>
      rule.ratingBasis === "Matrix" ||
      rule.ratingBasis === "Formula" ||
      rule.rate != null ||
      (rule.breaks?.length ?? 0) > 0,
    {
      path: ["rate"],
      message: "A rate is required unless the lane is banded, matrix or formula rated",
    },
  );
export type RateAgreementRule = z.infer<typeof rateAgreementRuleSchema>;

export const rateAgreementAccessorialSchema = z.object({
  ...tenantInfoSchema.shape,

  rateAgreementId: optionalStringSchema,
  accessorialChargeId: z.string({ error: "Accessorial charge is required" }),
  method: z.enum(["Flat", "PerUnit", "Percentage"]),
  rateUnit: z.enum(["Mile", "Hour", "Day", "Stop"]).nullish(),
  amount: decimalStringSchema,
  waived: z.boolean().default(false),
  autoApply: z.boolean().default(false),
  applyCondition: z.string().default(""),
  freeUnits: z.number().int().nullish(),
  maxAmount: decimalStringSchema,
  serviceTypeIds: z.array(z.string()).nullish(),
  shipmentTypeIds: z.array(z.string()).nullish(),
  formulaTemplateId: z.string().nullish(),
  effectiveFrom: z.number().int().nullish(),
  effectiveTo: z.number().int().nullish(),
});
export type RateAgreementAccessorial = z.infer<typeof rateAgreementAccessorialSchema>;

/**
 * The contract's fuel terms.
 *
 * A waiver and an override describe opposite intentions, which is why the two
 * cannot both be set: leaving both would make the effective terms depend on
 * which the engine happened to read first.
 */
export const rateAgreementFuelBindingSchema = z
  .object({
    id: optionalStringSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,
    rateAgreementId: optionalStringSchema,
    fuelSurchargeProgramId: z.string({ error: "Fuel surcharge program is required" }),
    waived: z.boolean().default(false),
    pegPriceOverride: decimalStringSchema,
    incrementRateOverride: decimalStringSchema,
    capAmount: decimalStringSchema,
  })
  .refine(
    (binding) =>
      !binding.waived ||
      (binding.pegPriceOverride == null &&
        binding.incrementRateOverride == null &&
        binding.capAmount == null),
    {
      path: ["waived"],
      message: "A waived fuel binding cannot also override the program's terms",
    },
  );
export type RateAgreementFuelBinding = z.infer<typeof rateAgreementFuelBindingSchema>;

export const rateAgreementVersionSchema = z.object({
  id: optionalStringSchema,
  rateAgreementId: optionalStringSchema,
  versionNumber: z.number().int(),
  effectiveFrom: z.number().int(),
  effectiveTo: z.number().int().nullish(),
  changeMessage: z.string().default(""),
  createdById: z.string().nullish(),
  createdAt: z.number().int().optional(),
});
export type RateAgreementVersion = z.infer<typeof rateAgreementVersionSchema>;

export const rateAgreementSchema = z
  .object({
    ...tenantInfoSchema.shape,

    partyType: ratePartyTypeSchema.default("Customer"),
    customerId: z.string().nullish(),
    carrierId: z.string().nullish(),

    code: z
      .string({ error: "Code is required" })
      .min(2, { error: "Code must be at least 2 characters" })
      .max(50, { error: "Code must be less than 50 characters" }),
    name: z.string({ error: "Name is required" }).min(1, { error: "Name is required" }).max(150),
    description: z.string().max(2000).default(""),
    agreementType: rateAgreementTypeSchema.default("Contract"),
    status: rateAgreementStatusSchema.default("Draft"),
    contractRef: z.string().max(100).default(""),
    documentId: z.string().nullish(),

    priority: z.number().int().default(0),
    effectiveFrom: z.number().int(),
    effectiveTo: z.number().int().nullish(),
    autoRenew: z.boolean().default(false),
    renewalNoticeDays: z.number().int().default(30),

    currency: z.string().length(3).default("USD"),
    defaultMinCharge: decimalStringSchema,
    defaultMaxCharge: decimalStringSchema,
    roundingMode: rateRoundingModeSchema.default("HalfUp"),
    roundingPrecision: z.number().int().min(0).max(6).default(2),

    billToCustomerId: z.string().nullish(),
    marginFloorPercent: decimalStringSchema,
    maxPayPercentOfSell: decimalStringSchema,

    submittedById: z.string().nullish(),
    submittedAt: z.number().int().nullish(),
    approvedById: z.string().nullish(),
    approvedAt: z.number().int().nullish(),
    reviewComment: z.string().default(""),
    currentVersionNumber: z.number().int().default(1),

    rules: z.array(rateAgreementRuleSchema).default([]),
    accessorials: z.array(rateAgreementAccessorialSchema).default([]),
    fuelBinding: rateAgreementFuelBindingSchema.nullish(),
    versions: z.array(rateAgreementVersionSchema).default([]),
  })
  .refine((agreement) => agreement.partyType !== "Customer" || Boolean(agreement.customerId), {
    path: ["customerId"],
    message: "A customer agreement needs a customer",
  })
  .refine((agreement) => agreement.partyType !== "Carrier" || Boolean(agreement.carrierId), {
    path: ["carrierId"],
    message: "A carrier agreement needs a carrier",
  })
  .refine(
    (agreement) => agreement.effectiveTo == null || agreement.effectiveTo > agreement.effectiveFrom,
    { path: ["effectiveTo"], message: "The end date must fall after the start date" },
  );
export type RateAgreement = z.infer<typeof rateAgreementSchema>;

/* -------------------------------------------------------------------------- */
/*                                   Quotes                                    */
/* -------------------------------------------------------------------------- */

export const rateQuoteOutcomeSchema = z.enum([
  "Rated",
  "FormulaFallback",
  "ManualOverride",
  "NoRateFound",
  "Error",
]);
export type RateQuoteOutcome = z.infer<typeof rateQuoteOutcomeSchema>;

export const rateQuotePurposeSchema = z.enum([
  "Rating",
  "Quote",
  "Shopping",
  "Simulation",
  "WhatIf",
]);
export type RateQuotePurpose = z.infer<typeof rateQuotePurposeSchema>;

/** Why a rate did not apply, or why it lost to another one. */
export const rateRejectReasonSchema = z.string();

export const rateTraceCandidateSchema = z.object({
  agreementId: z.string().default(""),
  agreementCode: z.string().default(""),
  agreementName: z.string().default(""),
  agreementPriority: z.number().int().default(0),
  ruleId: z.string().default(""),
  ruleLabel: z.string().default(""),
  laneKey: z.string().default(""),
  specificityScore: z.number().int().default(0),
  rulePriority: z.number().int().default(0),
  effectiveFrom: z.number().int().default(0),
  effectiveTo: z.number().int().nullish(),
  won: z.boolean().default(false),
  rank: z.number().int().default(0),
  matchedOn: z.array(z.string()).nullish(),
  rejectReason: rateRejectReasonSchema.default(""),
  rejectDetail: z.string().default(""),
});
export type RateTraceCandidate = z.infer<typeof rateTraceCandidateSchema>;

export const rateTraceComponentSchema = z.object({
  sequence: z.number().int().default(0),
  kind: z.string().default(""),
  label: z.string().default(""),
  basis: z.string().default(""),
  quantity: decimalStringSchema,
  rate: decimalStringSchema,
  amount: decimalStringSchema,
  runningTotal: decimalStringSchema,
  source: z.string().default(""),
  sourceId: z.string().default(""),
  sourceName: z.string().default(""),
  detail: z.record(z.string(), z.unknown()).nullish(),
});
export type RateTraceComponent = z.infer<typeof rateTraceComponentSchema>;

export const rateTraceGuardrailSchema = z.object({
  kind: z.string().default(""),
  label: z.string().default(""),
  bound: decimalStringSchema,
  rawAmount: decimalStringSchema,
  amount: decimalStringSchema,
  applied: z.boolean().default(false),
});
export type RateTraceGuardrail = z.infer<typeof rateTraceGuardrailSchema>;

export const rateTraceSchema = z.object({
  engineVersion: z.string().default(""),
  laneKeysTried: z.array(z.string()).nullish(),
  candidateCount: z.number().int().default(0),
  candidates: z.array(rateTraceCandidateSchema).nullish(),
  components: z.array(rateTraceComponentSchema).nullish(),
  guardrails: z.array(rateTraceGuardrailSchema).nullish(),
  tieBreak: z.string().default(""),
  warnings: z.array(z.string()).nullish(),
  error: z.string().default(""),
  totals: z
    .object({
      linehaul: decimalStringSchema,
      total: decimalStringSchema,
    })
    .nullish(),
  inputs: z.record(z.string(), z.unknown()).nullish(),
  fx: z
    .object({
      fromCurrency: z.string().default(""),
      toCurrency: z.string().default(""),
      rate: decimalStringSchema,
      rateDate: z.string().default(""),
    })
    .nullish(),
});
export type RateTrace = z.infer<typeof rateTraceSchema>;

export const rateQuoteSchema = z.object({
  ...tenantInfoSchema.shape,

  shipmentId: z.string().nullish(),
  partyType: ratePartyTypeSchema,
  partyId: z.string().default(""),
  purpose: rateQuotePurposeSchema,
  outcome: rateQuoteOutcomeSchema,
  rateAgreementId: z.string().nullish(),
  rateAgreementRuleId: z.string().nullish(),
  formulaTemplateId: z.string().nullish(),
  specificityScore: z.number().int().default(0),
  currency: z.string().default("USD"),
  billingCurrency: z.string().default("USD"),
  linehaulAmount: decimalStringSchema,
  totalAmount: decimalStringSchema,
  billingAmount: decimalStringSchema,
  foregoneAmount: decimalStringSchema,
  overrideReason: z.string().default(""),
  asOf: z.number().int().default(0),
  ratedAt: z.number().int().default(0),
  ratedById: z.string().nullish(),
  engineVersion: z.string().default(""),
  trace: rateTraceSchema.nullish(),
});
export type RateQuote = z.infer<typeof rateQuoteSchema>;

/** What the engine returned for one rating, saved or hypothetical. */
export const ratedShipmentSchema = z.object({
  amount: decimalStringSchema,
  currency: z.string().default("USD"),
  outcome: rateQuoteOutcomeSchema,
  quote: rateQuoteSchema.nullish(),
  agreementId: z.string().nullish(),
  ruleId: z.string().nullish(),
  formulaTemplateId: z.string().nullish(),
});
export type RatedShipment = z.infer<typeof ratedShipmentSchema>;

/* -------------------------------------------------------------------------- */
/*                                  Shopping                                   */
/* -------------------------------------------------------------------------- */

/**
 * What "best" means for one shopping run.
 *
 * It is a per-run choice rather than a setting, because the answer changes with
 * the load: a cheap carrier is the right pick on a lane with room in it and the
 * wrong pick on one the customer will cancel over.
 */
export const shopStrategySchema = z.enum(["LeastCost", "BestMargin", "GuideRank", "FastestAccept"]);
export type ShopStrategy = z.infer<typeof shopStrategySchema>;

/**
 * What a shipment's two sides came to, and whether that is allowed.
 *
 * The three numbers are never absent: the server carries them as plain decimals
 * rather than nullable ones, and a margin of zero is a real answer that a null
 * would blur into "not measured".
 */
export const marginVerdictSchema = z.object({
  amount: decimalStringSchema.nullish().transform((value) => value ?? 0),
  percent: decimalStringSchema.nullish().transform((value) => value ?? 0),
  /** The buy price as a share of the sell price. */
  payPercent: decimalStringSchema.nullish().transform((value) => value ?? 0),

  floorApplies: z.boolean().default(false),
  belowFloor: z.boolean().default(false),
  ceilingApplies: z.boolean().default(false),
  abovePayCeiling: z.boolean().default(false),

  explanation: z.string().default(""),
});
export type MarginVerdict = z.infer<typeof marginVerdictSchema>;

/** One carrier's answer to "what would you charge to haul this". */
export const shopOptionSchema = z.object({
  carrierId: z.string(),
  carrierName: z.string().default(""),

  rank: z.number().int().default(0),
  /** The routing guide's own rank, zero when the carrier came from no guide. */
  guideRank: z.number().int().default(0),
  offerTtlSeconds: z.number().int().default(0),

  outcome: rateQuoteOutcomeSchema,
  cost: decimalStringSchema.nullish().transform((value) => value ?? 0),
  currency: z.string().default("USD"),

  margin: marginVerdictSchema,

  quote: rateQuoteSchema.nullish(),
  agreementId: z.string().nullish(),
  ruleId: z.string().nullish(),

  /** Why this option ranked where it did, or why it could not be priced. */
  note: z.string().default(""),
});
export type ShopOption = z.infer<typeof shopOptionSchema>;

export const shopResultSchema = z.object({
  strategy: shopStrategySchema,
  options: z.array(shopOptionSchema).default([]),
  routingGuideId: z.string().nullish(),
  sellTotal: decimalStringSchema.nullish(),
  warnings: z.array(z.string()).default([]),
});
export type ShopResult = z.infer<typeof shopResultSchema>;

/* -------------------------------------------------------------------------- */
/*                                 Simulation                                  */
/* -------------------------------------------------------------------------- */

export const rateSimulationStatusSchema = z.enum([
  "Pending",
  "Running",
  "Completed",
  "Failed",
  "Canceled",
]);
export type RateSimulationStatus = z.infer<typeof rateSimulationStatusSchema>;

/**
 * What happened to one of the simulated agreement's rules across a whole run.
 *
 * The two failure states are the point. A rule that never matched anything was
 * written for freight the organization does not move; a rule that matched every
 * time and never won is shadowed by something narrower. Both are invisible in
 * the revenue total.
 */
export const ruleOutcomeSchema = z.enum(["Won", "Lost", "NeverFired"]);
export type RuleOutcome = z.infer<typeof ruleOutcomeSchema>;

export const ruleCoverageSchema = z.object({
  ruleId: z.string(),
  label: z.string().default(""),
  laneKey: z.string().default(""),
  outcome: ruleOutcomeSchema,
  wonCount: z.number().int().default(0),
  lostCount: z.number().int().default(0),
  lostTo: z.string().nullish(),
  lostToLabel: z.string().default(""),
});
export type RuleCoverage = z.infer<typeof ruleCoverageSchema>;

/** What a whole simulation came to. */
export const rateSimulationSummarySchema = z.object({
  shipmentCount: z.number().int().default(0),
  evaluatedCount: z.number().int().default(0),
  changedCount: z.number().int().default(0),
  increasedCount: z.number().int().default(0),
  decreasedCount: z.number().int().default(0),
  errorCount: z.number().int().default(0),

  beforeTotal: decimalStringSchema.nullish().transform((value) => value ?? 0),
  afterTotal: decimalStringSchema.nullish().transform((value) => value ?? 0),
  totalDelta: decimalStringSchema.nullish().transform((value) => value ?? 0),
  totalDeltaPct: decimalStringSchema.nullish().transform((value) => value ?? 0),
  maxIncrease: decimalStringSchema.nullish().transform((value) => value ?? 0),
  maxDecrease: decimalStringSchema.nullish().transform((value) => value ?? 0),
});
export type RateSimulationSummary = z.infer<typeof rateSimulationSummarySchema>;

export const rateSimulationSchema = z.object({
  ...tenantInfoSchema.shape,

  rateAgreementId: z.string({ error: "An agreement to simulate is required" }).min(1),
  name: z
    .string({ error: "Name is required" })
    .min(1, { error: "Name is required" })
    .max(150, { error: "Name must be less than 150 characters" }),
  description: z.string().max(500).default(""),
  status: rateSimulationStatusSchema.default("Pending"),
  partyType: ratePartyTypeSchema.default("Customer"),

  sampleFrom: z.number({ error: "A start date is required" }).int(),
  sampleTo: z.number({ error: "An end date is required" }).int(),
  sampleLimit: z.number().int().min(0).default(0),

  summary: rateSimulationSummarySchema.nullish(),
  ruleCoverage: z.array(ruleCoverageSchema).nullish(),

  error: z.string().default(""),
  startedAt: z.number().nullish(),
  completedAt: z.number().nullish(),
  requestedBy: z.string().nullish(),
  workflowId: z.string().default(""),
});
export type RateSimulation = z.infer<typeof rateSimulationSchema>;

/** One shipment, priced two ways. */
export const rateSimulationResultSchema = z.object({
  id: optionalStringSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  rateSimulationId: optionalStringSchema,

  shipmentId: z.string(),
  proNumber: z.string().default(""),
  customerId: z.string().nullish(),
  laneKey: z.string().default(""),
  equipmentTypeId: z.string().nullish(),

  beforeAmount: decimalStringSchema.nullish().transform((value) => value ?? 0),
  afterAmount: decimalStringSchema.nullish().transform((value) => value ?? 0),
  delta: decimalStringSchema.nullish().transform((value) => value ?? 0),
  deltaPercent: decimalStringSchema.nullish().transform((value) => value ?? 0),

  outcome: rateQuoteOutcomeSchema,
  beforeRuleId: z.string().nullish(),
  afterRuleId: z.string().nullish(),
  error: z.string().default(""),
  createdAt: z.number().nullish(),
});
export type RateSimulationResult = z.infer<typeof rateSimulationResultSchema>;
