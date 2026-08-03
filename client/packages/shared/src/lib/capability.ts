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

export function isFieldRequired(
  profile: ResolvedModeProfile | null | undefined,
  field: string,
): boolean {
  if (!profile?.rules) return false;

  return Object.values(profile.rules).some(
    (rule) => ruleBlocks(rule) && (rule.fields?.includes(field) ?? false),
  );
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
