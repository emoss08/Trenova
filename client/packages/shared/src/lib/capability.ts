import type {
  EnforcementLevel,
  ResolvedCapabilityRule,
  ResolvedModeProfile,
  ShipmentUIPolicy,
} from "../types/shipment";

export const RULE_KEYS = {
  hazmatSegregation: "hazmat.segregation",
  duplicateBol: "documentation.duplicateBol",
  moveRemoval: "dispatch.moveRemoval",
  maxShipmentWeight: "cargo.maxShipmentWeight",
  temperatureRange: "cargo.temperatureRange",
  dimensionsRequired: "cargo.dimensionsRequired",
  envelopeExceedsEquipment: "cargo.envelopeExceedsEquipment",
  permitRequired: "permit.requiredBeforeDispatch",
  permitExpiry: "permit.expiresBeforeDelivery",
  permitEscorts: "permit.escortsArranged",
  permitLeadTime: "permit.leadTimeInsufficient",
  permitCurfewConflict: "permit.curfewConflict",
} as const;

export type RuleKey = (typeof RULE_KEYS)[keyof typeof RULE_KEYS];

export const RULE_PARAMETERS = {
  maxWeight: "maxWeight",
  requireBothBounds: "requireBothBounds",
  maxOverhangFeet: "maxOverhangFeet",
  attachDerivedCharges: "attachDerivedCharges",
} as const;

export const CAPABILITIES = {
  core: "Core",
  temperatureControl: "TemperatureControl",
  hazmat: "Hazmat",
  dimensionalCargo: "DimensionalCargo",
  permitting: "Permitting",
  compartmentizedCargo: "CompartmentizedCargo",
  meteredQuantity: "MeteredQuantity",
  pieceLevelTracking: "PieceLevelTracking",
  containerizedEquipment: "ContainerizedEquipment",
  crewBased: "CrewBased",
  multiShipmentConsolidation: "MultiShipmentConsolidation",
  appointmentRequired: "AppointmentRequired",
} as const;

export type Capability = (typeof CAPABILITIES)[keyof typeof CAPABILITIES];

export function getProfile(policy: ShipmentUIPolicy | undefined | null) {
  return policy?.profile ?? null;
}

export function hasCapability(
  profile: ResolvedModeProfile | null | undefined,
  capability: Capability,
): boolean {
  return profile?.capabilities?.includes(capability) ?? false;
}

export function getRule(
  profile: ResolvedModeProfile | null | undefined,
  ruleKey: RuleKey,
): ResolvedCapabilityRule | null {
  return profile?.rules?.[ruleKey] ?? null;
}

export function ruleApplies(rule: ResolvedCapabilityRule | null): boolean {
  return !!rule && rule.enabled && rule.enforcement !== "Ignore";
}

export function ruleBlocks(rule: ResolvedCapabilityRule | null): boolean {
  return !!rule && rule.enabled && rule.enforcement === "Block";
}

/**
 * Whether the profile makes a field mandatory.
 *
 * Reads `requiredFields`, not `fields`. A rule targets every field it has
 * something to say about — that is what drives the explainer — but most rules
 * constrain a value that is already there rather than demanding one. The
 * duplicate-BOL rule targets `bol` and forbids reusing it; move removal targets
 * `moves` and forbids deleting one; the weight cap returns early when weight is
 * absent. None of them make their field required.
 *
 * The server computes `requiredFields` from the resolved parameters, so a
 * requirement switched off by a parameter — temperatureMax without
 * requireBothBounds — is already absent here.
 */
export function isFieldRequired(
  profile: ResolvedModeProfile | null | undefined,
  field: string,
): boolean {
  if (!profile?.rules) return false;

  return Object.values(profile.rules).some(
    (rule) => ruleBlocks(rule) && (rule.requiredFields?.includes(field) ?? false),
  );
}

/**
 * Whether a persisted move may be removed.
 *
 * Mirrors `moveRemovalRuleFor` in
 * `services/tms/internal/core/services/shipmentservice/capability_policy.go`:
 * the profile's `dispatch.moveRemoval` rule decides whenever it is present, and
 * the organization's `allowMoveRemovals` flag is a fallback for profiles that
 * carry no such rule.
 *
 * The precedence is the whole point. Reading the flag alone lets the form permit
 * a removal the server will reject on save — the operator deletes a move, submits,
 * and gets a validation failure against a rule the form never showed them.
 *
 * Keep this in step with the Go function; a change to one without the other
 * reopens exactly that divergence.
 */
export function isMoveRemovalAllowed(
  profile: ResolvedModeProfile | null | undefined,
  allowMoveRemovals: boolean | undefined,
): boolean {
  const rule = getRule(profile, RULE_KEYS.moveRemoval);
  if (rule) {
    return !ruleBlocks(rule);
  }

  return allowMoveRemovals === true;
}

export function rulesForField(
  profile: ResolvedModeProfile | null | undefined,
  field: string,
): ResolvedCapabilityRule[] {
  if (!profile?.rules) return [];

  return Object.values(profile.rules).filter(
    (rule) => ruleApplies(rule) && (rule.fields?.includes(field) ?? false),
  );
}

export function isCapabilitySectionVisible(
  profile: ResolvedModeProfile | null | undefined,
  requiredCapability: Capability,
): boolean {
  if (!profile) return true;

  return hasCapability(profile, requiredCapability);
}

export function getNumericParameter(
  rule: ResolvedCapabilityRule | null,
  name: string,
): number | null {
  const value = rule?.parameters?.[name];
  return typeof value === "number" ? value : null;
}

export function getBooleanParameter(
  rule: ResolvedCapabilityRule | null,
  name: string,
): boolean | null {
  const value = rule?.parameters?.[name];
  return typeof value === "boolean" ? value : null;
}

const ENFORCEMENT_LABELS: Record<EnforcementLevel, string> = {
  Ignore: "Not enforced",
  Warn: "Warns and records",
  RequireReview: "Requires review",
  Block: "Blocks saving",
};

export function enforcementLabel(level: EnforcementLevel): string {
  return ENFORCEMENT_LABELS[level];
}

const ENFORCEMENT_TONES: Record<EnforcementLevel, string> = {
  Ignore: "text-muted-foreground",
  Warn: "text-amber-600 dark:text-amber-500",
  RequireReview: "text-blue-600 dark:text-blue-500",
  Block: "text-red-600 dark:text-red-500",
};

export function enforcementTone(level: EnforcementLevel): string {
  return ENFORCEMENT_TONES[level];
}

const MATCH_LABELS: Record<string, string> = {
  organizationDefault: "organization default",
  customer: "customer",
  serviceType: "service type",
  shipmentType: "shipment type",
  equipmentType: "equipment type",
};

export function describeMatch(matchedOn: string[] | null | undefined): string {
  if (!matchedOn?.length) return "no specific scope";

  return matchedOn.map((match) => MATCH_LABELS[match] ?? match).join(", ");
}
