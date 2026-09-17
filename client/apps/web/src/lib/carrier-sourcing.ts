import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";

export const CARRIER_INTEL_CAPABILITY_SEARCH = "Search";
export const CARRIER_INTEL_CAPABILITY_AUTOCOMPLETE = "Autocomplete";

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

export function cityStateLabel(
  city: string | null | undefined,
  state: string | null | undefined,
): string | null {
  if (city && state) {
    return `${city}, ${state}`;
  }
  return city || state || null;
}

type AuthorityGrantLike = { status: string; ageDays: number | null } | null | undefined;

export function oldestActiveAuthorityAgeDays(
  authority:
    | Pick<NonNullable<CarrierIntelProfile["authority"]>, "common" | "contract" | "broker">
    | null
    | undefined,
): number | null {
  if (!authority) {
    return null;
  }
  let oldest: number | null = null;
  const grants: AuthorityGrantLike[] = [authority.common, authority.contract, authority.broker];
  for (const grant of grants) {
    if (!grant || grant.status !== "Active" || grant.ageDays === null) {
      continue;
    }
    if (oldest === null || grant.ageDays > oldest) {
      oldest = grant.ageDays;
    }
  }
  return oldest;
}

export function withExistingCarrier<
  T extends { dotNumber: string; existingCarrierId: string | null },
>(items: readonly T[], dotNumber: string, carrierId: string): T[] {
  return items.map((item) =>
    item.dotNumber === dotNumber ? { ...item, existingCarrierId: carrierId } : item,
  );
}
