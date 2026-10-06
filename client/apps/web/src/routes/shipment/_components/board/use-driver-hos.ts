import { queries } from "@/lib/queries";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";
import { useQuery } from "@tanstack/react-query";

const HOS_REFRESH_MS = 60_000;

/**
 * Drive time left for each driver, read once for the page and shared by every
 * cell that shows it. Nothing is fetched when no ELD is connected.
 */
export function useDriverHos(workerId: string | null | undefined): number | null {
  const { hos } = useShipmentCapabilities();
  const { data } = useQuery({
    ...queries.telematics.workerHosStates(),
    enabled: hos,
    staleTime: HOS_REFRESH_MS,
    refetchInterval: HOS_REFRESH_MS,
    select: (states) => new Map(states.map((state) => [state.workerId, state.driveRemainingMs])),
  });
  if (!hos || !workerId) return null;
  return data?.get(workerId) ?? null;
}
