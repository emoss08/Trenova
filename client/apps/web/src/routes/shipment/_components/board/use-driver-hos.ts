import { queries } from "@/lib/queries";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";
import { useQuery } from "@tanstack/react-query";
import { useCallback } from "react";

const HOS_REFRESH_MS = 60_000;

type HosState = { workerId: string; driveRemainingMs: number };

const indexes = new WeakMap<readonly HosState[], Map<string, number>>();

/**
 * Drive time left per driver, indexed once per fetched list however many
 * cells read it. Keyed on the list itself, so a refetch builds one new index
 * and the old one goes with the old list.
 */
function driveRemainingIndex(states: readonly HosState[]): Map<string, number> {
  let index = indexes.get(states);
  if (!index) {
    index = new Map(states.map((state) => [state.workerId, state.driveRemainingMs]));
    indexes.set(states, index);
  }
  return index;
}

/**
 * Drive time left for one driver, read from the page's one HOS query. Each
 * cell selects only its own driver's number, so a refetch redraws the cells
 * whose number changed rather than every cell on the board. Nothing is
 * fetched when no ELD is connected.
 */
export function useDriverHos(workerId: string | null | undefined): number | null {
  const { hos } = useShipmentCapabilities();
  const select = useCallback(
    (states: readonly HosState[]) =>
      workerId ? (driveRemainingIndex(states).get(workerId) ?? null) : null,
    [workerId],
  );
  const { data } = useQuery({
    ...queries.telematics.workerHosStates(),
    enabled: hos,
    staleTime: HOS_REFRESH_MS,
    refetchInterval: HOS_REFRESH_MS,
    select,
  });
  if (!hos || !workerId) return null;
  return data ?? null;
}
