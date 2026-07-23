import { fetchMyLoads, type PortalLoad, type PortalStop } from "@/lib/graphql/driver-portal";
import type { PortalLoadScope } from "@/graphql/generated/graphql";
import { useQueries, useQuery } from "@tanstack/react-query";

export function useMyLoads(scope: PortalLoadScope) {
  return useQuery({
    queryKey: ["dash-loads", scope],
    queryFn: () => fetchMyLoads(scope, 25),
  });
}

export function useLoad(assignmentId: string) {
  const results = useQueries({
    queries: (["Active", "History"] as const).map((scope) => ({
      queryKey: ["dash-loads", scope],
      queryFn: () => fetchMyLoads(scope, 25),
    })),
  });

  const isPending = results.some((result) => result.isPending);
  const load =
    results
      .flatMap((result) => result.data ?? [])
      .find((candidate) => candidate.assignmentId === assignmentId) ?? null;

  return { load, isPending };
}

export function originStop(load: PortalLoad): PortalStop | undefined {
  return load.stops[0];
}

export function destinationStop(load: PortalLoad): PortalStop | undefined {
  return load.stops.length > 1 ? load.stops[load.stops.length - 1] : undefined;
}

export function stopPlace(stop: PortalStop | undefined): string {
  if (!stop) return "—";
  return stop.locationName || stop.addressLine || "—";
}

const usAddressPattern = /,\s*[A-Z]{2}\s+\d{5}(-\d{4})?(\s*,?\s*(USA|United States))?\s*$/i;

export function isLikelyUSAddress(address: string): boolean {
  return usAddressPattern.test(address);
}

export function directionsUrl(stop: PortalStop): string {
  const destination = stop.addressLine || stop.locationName;
  if (isLikelyUSAddress(destination)) {
    return `https://truckmap.com/search/${encodeURIComponent(destination)}`;
  }
  return `https://www.google.com/maps/dir/?api=1&destination=${encodeURIComponent(destination)}`;
}

export function formatMiles(miles?: number | null): string | null {
  if (miles == null || miles <= 0) return null;
  return `${new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 }).format(miles)} mi`;
}

export function formatWeight(weight?: number | null): string | null {
  if (weight == null || weight <= 0) return null;
  return `${new Intl.NumberFormat("en-US").format(weight)} lbs`;
}

export function formatPieces(pieces?: number | null): string | null {
  if (pieces == null || pieces <= 0) return null;
  return `${new Intl.NumberFormat("en-US").format(pieces)} pcs`;
}
