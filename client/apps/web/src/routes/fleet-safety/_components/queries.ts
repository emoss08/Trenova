import { fetchFleetSafety, FLEET_SAFETY_KEY } from "@/lib/graphql/fleet-safety";

export const RANK_LIMIT = 10;

export type FleetSafetyArgs = {
  windowMonths: number;
  fleetCodeId: string | null;
};

export function fleetSafetyQuery(args: FleetSafetyArgs) {
  return {
    queryKey: [FLEET_SAFETY_KEY, args.windowMonths, args.fleetCodeId] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchFleetSafety(
        { windowMonths: args.windowMonths, fleetCodeId: args.fleetCodeId, rankLimit: RANK_LIMIT },
        { signal },
      ),
  };
}
