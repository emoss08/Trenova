import type {
  CarrierIntelDepth,
  CarrierIntelLookupInput,
  CarrierSourcingSearchInput,
  CarrierSourcingSort,
} from "@trenova/graphql/generated/graphql";
import { DAYS_PER_YEAR, profileAuthorityAgeDays } from "@/lib/carrier-intelligence";
import type { CarrierIntelFinding, CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import type {
  CarrierIntelProspectLookup,
  CarrierSourcingResult,
} from "@/lib/graphql/carrier-sourcing";

export const CARRIER_INTEL_CAPABILITY_SEARCH = "Search";
export const CARRIER_INTEL_CAPABILITY_AUTOCOMPLETE = "Autocomplete";

export const SOURCING_PAGE_SIZE = 25;
export const SOURCING_TEXT_MAX_LENGTH = 200;

export type CarrierSourcingAvailability =
  | "unknown"
  | "not-configured"
  | "unsupported"
  | "supported";

export type CarrierIntelProviderCapabilities = {
  configured: boolean;
  provider: string | null;
  capabilities: readonly string[];
};

export function carrierSourcingAvailability(
  provider: CarrierIntelProviderCapabilities | null | undefined,
): CarrierSourcingAvailability {
  if (!provider) {
    return "unknown";
  }
  if (!provider.configured) {
    return "not-configured";
  }
  return provider.capabilities.includes(CARRIER_INTEL_CAPABILITY_SEARCH)
    ? "supported"
    : "unsupported";
}

export function carrierIntelSupportsAutocomplete(
  provider: CarrierIntelProviderCapabilities | null | undefined,
): boolean {
  return (
    !!provider?.configured && provider.capabilities.includes(CARRIER_INTEL_CAPABILITY_AUTOCOMPLETE)
  );
}

export function withExistingCarrier<
  T extends { dotNumber: string; existingCarrierId: string | null },
>(items: readonly T[], dotNumber: string, carrierId: string): T[] {
  return items.map((item) =>
    item.dotNumber === dotNumber ? { ...item, existingCarrierId: carrierId } : item,
  );
}

export type SourcingIntent =
  | { kind: "empty" }
  | { kind: "dot"; dotNumber: string }
  | { kind: "mc"; docketNumber: string }
  | { kind: "ein"; text: string }
  | { kind: "vin"; text: string }
  | { kind: "name"; text: string };

const DOT_PREFIXED = /^(?:US\s*DOT|DOT)\s*(?:#|:|-|NO\.?)?\s*(\d{1,8})$/i;
const MC_PREFIXED = /^MC\s*(?:#|:|-|NO\.?)?\s*(\d{1,8})$/i;
const DIGITS_ONLY = /^\d{1,8}$/;
const EIN_PATTERN = /^(?:\d{9}|\d{2}-\d{7})$/;
const VIN_PATTERN = /^[A-HJ-NPR-Z0-9]{17}$/i;
const HAS_DIGIT = /\d/;

export function detectSourcingIntent(raw: string): SourcingIntent {
  const text = raw.trim().slice(0, SOURCING_TEXT_MAX_LENGTH);
  if (text === "") {
    return { kind: "empty" };
  }
  const dot = DOT_PREFIXED.exec(text);
  if (dot) {
    return { kind: "dot", dotNumber: dot[1] };
  }
  const mc = MC_PREFIXED.exec(text);
  if (mc) {
    return { kind: "mc", docketNumber: mc[1] };
  }
  if (DIGITS_ONLY.test(text)) {
    return { kind: "dot", dotNumber: text };
  }
  if (EIN_PATTERN.test(text)) {
    return { kind: "ein", text };
  }
  if (VIN_PATTERN.test(text) && HAS_DIGIT.test(text)) {
    return { kind: "vin", text: text.toUpperCase() };
  }
  return { kind: "name", text };
}

export function isLookupIntent(
  intent: SourcingIntent,
): intent is Extract<SourcingIntent, { kind: "dot" | "mc" }> {
  return intent.kind === "dot" || intent.kind === "mc";
}

export function isSearchIntent(
  intent: SourcingIntent,
): intent is Extract<SourcingIntent, { kind: "name" | "ein" | "vin" }> {
  return intent.kind === "name" || intent.kind === "ein" || intent.kind === "vin";
}

export function lookupInputForIntent(
  intent: Extract<SourcingIntent, { kind: "dot" | "mc" }>,
  depth: CarrierIntelDepth | null,
): CarrierIntelLookupInput {
  return intent.kind === "dot"
    ? { dotNumber: intent.dotNumber, depth }
    : { docketNumber: intent.docketNumber, depth };
}

export const POWER_UNIT_RANGES = ["1-10", "11-50", "51-250", "251+"] as const;
export type PowerUnitRange = (typeof POWER_UNIT_RANGES)[number];

export const POWER_UNIT_BOUNDS: Record<PowerUnitRange, { min: number; max: number | null }> = {
  "1-10": { min: 1, max: 10 },
  "11-50": { min: 11, max: 50 },
  "51-250": { min: 51, max: 250 },
  "251+": { min: 251, max: null },
};

export const AUTHORITY_AGE_PRESETS = ["under1", "1-3", "3-5", "5+"] as const;
export type AuthorityAgePreset = (typeof AUTHORITY_AGE_PRESETS)[number];

export const AUTHORITY_AGE_BOUNDS: Record<
  AuthorityAgePreset,
  { minDays: number | null; maxDays: number | null }
> = {
  under1: { minDays: null, maxDays: DAYS_PER_YEAR - 1 },
  "1-3": { minDays: DAYS_PER_YEAR, maxDays: 3 * DAYS_PER_YEAR - 1 },
  "3-5": { minDays: 3 * DAYS_PER_YEAR, maxDays: 5 * DAYS_PER_YEAR - 1 },
  "5+": { minDays: 5 * DAYS_PER_YEAR, maxDays: null },
};

export const SOURCING_SORTS = [
  "BestMatch",
  "FleetSizeDesc",
  "AuthorityAgeDesc",
] as const satisfies readonly CarrierSourcingSort[];
export const DEFAULT_SOURCING_SORT: CarrierSourcingSort = "BestMatch";

export const SOURCING_SCREENS = ["hazmat", "hideBlocked", "hideExisting"] as const;
export type SourcingScreen = (typeof SOURCING_SCREENS)[number];

export type SourcingFilters = {
  state: string | null;
  originState: string | null;
  destinationState: string | null;
  powerUnits: PowerUnitRange | null;
  authorityAge: AuthorityAgePreset | null;
  screens: readonly SourcingScreen[];
};

export const EMPTY_SOURCING_FILTERS: SourcingFilters = {
  state: null,
  originState: null,
  destinationState: null,
  powerUnits: null,
  authorityAge: null,
  screens: [],
};

export function countActiveFilters(filters: SourcingFilters): number {
  return (
    (filters.state ? 1 : 0) +
    (filters.originState || filters.destinationState ? 1 : 0) +
    (filters.powerUnits ? 1 : 0) +
    (filters.authorityAge ? 1 : 0) +
    filters.screens.length
  );
}

export function hasLocationFilter(filters: SourcingFilters): boolean {
  return !!(filters.state || filters.originState || filters.destinationState);
}

export function buildSourcingSearchInput(
  text: string,
  filters: SourcingFilters,
  sort: CarrierSourcingSort,
): Omit<CarrierSourcingSearchInput, "offset"> | null {
  const trimmed = text.trim().slice(0, SOURCING_TEXT_MAX_LENGTH);
  if (trimmed === "" && !hasLocationFilter(filters)) {
    return null;
  }
  const units = filters.powerUnits ? POWER_UNIT_BOUNDS[filters.powerUnits] : null;
  const authorityAge = filters.authorityAge ? AUTHORITY_AGE_BOUNDS[filters.authorityAge] : null;
  return {
    text: trimmed === "" ? null : trimmed,
    state: filters.state,
    originState: filters.originState,
    destinationState: filters.destinationState,
    minPowerUnits: units?.min ?? null,
    maxPowerUnits: units?.max ?? null,
    minAuthorityAgeDays: authorityAge?.minDays ?? null,
    maxAuthorityAgeDays: authorityAge?.maxDays ?? null,
    hazmatOnly: filters.screens.includes("hazmat"),
    excludeBlocking: filters.screens.includes("hideBlocked"),
    excludeExistingCarriers: filters.screens.includes("hideExisting"),
    sort,
    limit: SOURCING_PAGE_SIZE,
  };
}

export type SourcingCandidate = {
  dotNumber: string;
  legalName: string | null;
  existingCarrierId: string | null;
  riskLevel: string;
  findings: readonly CarrierIntelFinding[];
  profile: CarrierIntelProfile;
  provider: string | null;
  depth: CarrierIntelDepth | null;
  depthFetchedAt: number | null;
  fetchedDepth: CarrierIntelDepth | null;
  notFound: boolean;
  laneMatches: number | null;
  asOf: number | null;
  fetchedAt: number | null;
  confirmedAt: number | null;
  sourceAsOf: number | null;
};

export function candidateFromSearchResult(
  result: CarrierSourcingResult,
  provider: string | null,
  fetchedAt: number | null,
): SourcingCandidate {
  return {
    dotNumber: result.dotNumber,
    legalName: result.legalName || result.profile.identity?.legalName || null,
    existingCarrierId: result.existingCarrierId,
    riskLevel: result.riskLevel,
    findings: result.findings,
    profile: result.profile,
    provider,
    depth: null,
    depthFetchedAt: null,
    fetchedDepth: null,
    notFound: false,
    laneMatches: result.laneMatches,
    asOf: fetchedAt,
    fetchedAt,
    confirmedAt: null,
    sourceAsOf: null,
  };
}

export function candidateFromLookup(lookup: CarrierIntelProspectLookup): SourcingCandidate {
  const { snapshot } = lookup;
  return {
    dotNumber: snapshot.dotNumber,
    legalName: snapshot.profile.identity?.legalName ?? null,
    existingCarrierId: lookup.existingCarrierId,
    riskLevel: snapshot.riskLevel,
    findings: snapshot.findings,
    profile: snapshot.profile,
    provider: snapshot.provider,
    depth: snapshot.depth,
    depthFetchedAt: snapshot.depthFetchedAt,
    fetchedDepth: snapshot.fetchedDepth,
    notFound: snapshot.notFound,
    laneMatches: null,
    asOf: snapshot.effectiveAsOf,
    fetchedAt: snapshot.fetchedAt,
    confirmedAt: snapshot.confirmedAt,
    sourceAsOf: snapshot.sourceAsOf,
  };
}

export function withImportedCarriers(
  candidate: SourcingCandidate,
  imported: Readonly<Record<string, string>>,
): SourcingCandidate {
  const carrierId = imported[candidate.dotNumber];
  return carrierId && candidate.existingCarrierId !== carrierId
    ? { ...candidate, existingCarrierId: carrierId }
    : candidate;
}

export function dedupeByDotNumber<T extends { dotNumber: string }>(items: readonly T[]): T[] {
  const seen = new Set<string>();
  const unique: T[] = [];
  for (const item of items) {
    if (item.dotNumber !== "" && seen.has(item.dotNumber)) {
      continue;
    }
    seen.add(item.dotNumber);
    unique.push(item);
  }
  return unique;
}

export function candidateAuthorityAgeDays(candidate: SourcingCandidate): number | null {
  return profileAuthorityAgeDays(candidate.profile);
}

export type FindingCounts = { blockers: number; advisories: number; notices: number };

export function countFindings(findings: readonly { action: string }[]): FindingCounts {
  const counts: FindingCounts = { blockers: 0, advisories: 0, notices: 0 };
  for (const finding of findings) {
    if (finding.action === "Block") {
      counts.blockers += 1;
    } else if (finding.action === "Warn") {
      counts.advisories += 1;
    } else if (finding.action === "Notify") {
      counts.notices += 1;
    }
  }
  return counts;
}
