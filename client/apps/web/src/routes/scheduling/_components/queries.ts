import {
  fetchRota,
  fetchShiftSwapRequests,
  fetchShiftTemplates,
  ROTA_KEY,
  SHIFT_SWAPS_KEY,
  SHIFT_TEMPLATES_KEY,
} from "@/lib/graphql/scheduling";

export type RotaQueryArgs = {
  weekStart: number;
  weeks: number;
  teamOnly: boolean;
  fleetCodeId: string | null;
};

export function rotaQuery(args: RotaQueryArgs) {
  return {
    queryKey: [ROTA_KEY, args.weekStart, args.weeks, args.teamOnly, args.fleetCodeId] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchRota(
        {
          at: args.weekStart,
          weeks: args.weeks,
          teamOnly: args.teamOnly,
          fleetCodeId: args.fleetCodeId,
        },
        { signal },
      ),
  };
}

export function shiftTemplatesQuery() {
  return {
    queryKey: [SHIFT_TEMPLATES_KEY] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) => fetchShiftTemplates(undefined, { signal }),
  };
}

/** Swaps still going somewhere: the ones the office may have to decide. */
export function openSwapsQuery() {
  return {
    queryKey: [SHIFT_SWAPS_KEY, "open"] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchShiftSwapRequests({ openOnly: true }, { signal }),
  };
}
